package layout

import (
	"bytes"
	"container/list"
	"crypto/sha256"
	"math"
	"strconv"
	"strings"
	"sync"
)

const (
	imageEffectCacheMaxEntries = 256
	imageEffectCacheMaxBytes   = 512 << 10
	imageEffectCacheTotalBytes = 32 << 20
)

type imageFilterCacheKey struct {
	imgHash   [sha256.Size]byte
	imgLen    int
	linear    bool
	filterSig string
}

type clipPathMaskCacheKey struct {
	imgHash  [sha256.Size]byte
	imgLen   int
	relX     uint64
	relY     uint64
	drawnW   uint64
	drawnH   uint64
	refW     uint64
	refH     uint64
	shapeSig string
}

type byteLRUCacheEntry[K comparable] struct {
	key K
	val []byte
}

// byteLRUCache is a bounded, thread-safe LRU cache for immutable encoded image
// payloads. Stored slices are cloned to exact length (cap == len) so they never
// retain oversized encoder buffers and caller appends cannot alias cache memory.
type byteLRUCache[K comparable] struct {
	mu            sync.Mutex
	maxEntries    int
	maxEntryBytes int
	maxTotalBytes int
	totalBytes    int
	items         map[K]*list.Element
	order         list.List
}

func newByteLRUCache[K comparable](maxEntries, maxEntryBytes, maxTotalBytes int) *byteLRUCache[K] {
	return &byteLRUCache[K]{ //nolint:exhaustruct // order starts empty
		maxEntries:    maxEntries,
		maxEntryBytes: maxEntryBytes,
		maxTotalBytes: maxTotalBytes,
		items:         make(map[K]*list.Element, maxEntries),
	}
}

func (c *byteLRUCache[K]) get(key K) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return nil, false
	}

	c.order.MoveToFront(elem)

	//nolint:forcetypeassert // list only stores *byteLRUCacheEntry[K]
	return elem.Value.(*byteLRUCacheEntry[K]).val, true
}

func (c *byteLRUCache[K]) put(key K, out []byte) []byte {
	trimmed := bytes.Clone(out)
	if len(trimmed) == 0 {
		return out
	}

	// Cap slice capacity to length so caller append() allocates instead of
	// overwriting anything adjacent in memory.
	trimmed = trimmed[:len(trimmed):len(trimmed)]
	if len(trimmed) > c.maxEntryBytes || len(trimmed) > c.maxTotalBytes {
		return trimmed
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		//nolint:forcetypeassert // list only stores *byteLRUCacheEntry[K]
		entry := elem.Value.(*byteLRUCacheEntry[K])
		c.totalBytes += len(trimmed) - len(entry.val)
		entry.val = trimmed
		c.order.MoveToFront(elem)
		c.evictLocked()

		return trimmed
	}

	entry := &byteLRUCacheEntry[K]{key: key, val: trimmed}
	c.items[key] = c.order.PushFront(entry)
	c.totalBytes += len(trimmed)
	c.evictLocked()

	return trimmed
}

func (c *byteLRUCache[K]) evictLocked() {
	for len(c.items) > c.maxEntries || c.totalBytes > c.maxTotalBytes {
		back := c.order.Back()
		if back == nil {
			break
		}

		//nolint:forcetypeassert // list only stores *byteLRUCacheEntry[K]
		entry := back.Value.(*byteLRUCacheEntry[K])
		c.order.Remove(back)
		delete(c.items, entry.key)
		c.totalBytes -= len(entry.val)
	}
}

//nolint:gochecknoglobals // bounded process-wide raster effect caches matching gradCache
var (
	imageFilterCache  = newByteLRUCache[imageFilterCacheKey](imageEffectCacheMaxEntries, imageEffectCacheMaxBytes, imageEffectCacheTotalBytes)
	clipPathMaskCache = newByteLRUCache[clipPathMaskCacheKey](imageEffectCacheMaxEntries, imageEffectCacheMaxBytes, imageEffectCacheTotalBytes)
)

func canonicalFloat64Bits(val float64) uint64 {
	if val == 0 {
		return 0
	}

	return math.Float64bits(val)
}

func makeImageFilterCacheKey(imgBytes []byte, filters []parsedFilter, linear bool) imageFilterCacheKey {
	var sig strings.Builder

	for _, flt := range filters {
		if !filterAppliesToImage(flt.kind) {
			continue
		}

		sig.WriteString(strconv.Itoa(int(flt.kind)))
		sig.WriteByte(':')
		sig.WriteString(strconv.FormatUint(canonicalFloat64Bits(flt.val), 16))
		sig.WriteByte(';')
	}

	return imageFilterCacheKey{
		imgHash:   sha256.Sum256(imgBytes),
		imgLen:    len(imgBytes),
		linear:    linear,
		filterSig: sig.String(),
	}
}

// lookupImageFilterCache returns a read-only cached PNG slice (cap == len).
func lookupImageFilterCache(key imageFilterCacheKey) ([]byte, bool) {
	return imageFilterCache.get(key)
}

// storeImageFilterCache clones out to exact length, stores it when within cache
// limits, and returns the immutable read-only slice.
func storeImageFilterCache(key imageFilterCacheKey, out []byte) []byte {
	return imageFilterCache.put(key, out)
}

func appendClipPathLengthSig(buf *strings.Builder, length clipPathLength) {
	buf.WriteString(strconv.FormatUint(canonicalFloat64Bits(length.value), 16))

	if length.percent {
		buf.WriteByte('%')
	}

	if length.keyword != "" {
		buf.WriteByte('#')
		buf.WriteString(length.keyword)
	}

	buf.WriteByte(',')
}

func clipPathShapeSig(shape clipPathShape) string {
	var buf strings.Builder

	buf.WriteString(strconv.Itoa(int(shape.kind)))
	buf.WriteByte(':')
	buf.WriteString(shape.fillRule)
	buf.WriteByte(':')

	switch shape.kind {
	case clipPathInset:
		appendClipPathLengthSig(&buf, shape.top)
		appendClipPathLengthSig(&buf, shape.right)
		appendClipPathLengthSig(&buf, shape.bottom)
		appendClipPathLengthSig(&buf, shape.left)
	case clipPathCircle:
		appendClipPathLengthSig(&buf, shape.radius)
		appendClipPathLengthSig(&buf, shape.centerX)
		appendClipPathLengthSig(&buf, shape.centerY)
	case clipPathEllipse:
		appendClipPathLengthSig(&buf, shape.radiusX)
		appendClipPathLengthSig(&buf, shape.radiusY)
		appendClipPathLengthSig(&buf, shape.centerX)
		appendClipPathLengthSig(&buf, shape.centerY)
	case clipPathPolygon:
		for _, pt := range shape.points {
			appendClipPathLengthSig(&buf, pt.x)
			appendClipPathLengthSig(&buf, pt.y)
		}
	}

	return buf.String()
}

func makeClipPathMaskCacheKey(
	data []byte, shape clipPathShape,
	drawnX, drawnY, drawnW, drawnH, refX, refY, refW, refH float64,
) clipPathMaskCacheKey {
	return clipPathMaskCacheKey{
		imgHash:  sha256.Sum256(data),
		imgLen:   len(data),
		relX:     canonicalFloat64Bits(drawnX - refX),
		relY:     canonicalFloat64Bits(drawnY - refY),
		drawnW:   canonicalFloat64Bits(drawnW),
		drawnH:   canonicalFloat64Bits(drawnH),
		refW:     canonicalFloat64Bits(refW),
		refH:     canonicalFloat64Bits(refH),
		shapeSig: clipPathShapeSig(shape),
	}
}

// lookupClipPathMaskCache returns a read-only cached PNG slice (cap == len).
func lookupClipPathMaskCache(key clipPathMaskCacheKey) ([]byte, bool) {
	return clipPathMaskCache.get(key)
}

// storeClipPathMaskCache clones out to exact length, stores it when within
// cache limits, and returns the immutable read-only slice.
func storeClipPathMaskCache(key clipPathMaskCacheKey, out []byte) []byte {
	return clipPathMaskCache.put(key, out)
}
