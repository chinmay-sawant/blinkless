package layout

import (
	"crypto/sha256"
	"math"
	"strconv"
	"strings"
	"sync"
)

const (
	imageEffectCacheMaxEntries = 256
	imageEffectCacheMaxBytes   = 512 << 10
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

//nolint:gochecknoglobals // bounded process-wide raster effect caches matching gradCache
var (
	imageFilterCacheMu sync.RWMutex
	imageFilterCache   = make(map[imageFilterCacheKey][]byte, imageEffectCacheMaxEntries)

	clipPathMaskCacheMu sync.RWMutex
	clipPathMaskCache   = make(map[clipPathMaskCacheKey][]byte, imageEffectCacheMaxEntries)
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
		if flt.kind != filterBlur && flt.kind != filterGrayscale && flt.kind != filterInvert {
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

func lookupImageFilterCache(key imageFilterCacheKey) ([]byte, bool) {
	imageFilterCacheMu.RLock()
	cached, ok := imageFilterCache[key]
	imageFilterCacheMu.RUnlock()

	return cached, ok
}

func storeImageFilterCache(key imageFilterCacheKey, out []byte) {
	if len(out) == 0 || len(out) > imageEffectCacheMaxBytes {
		return
	}

	imageFilterCacheMu.Lock()
	if len(imageFilterCache) >= imageEffectCacheMaxEntries {
		clear(imageFilterCache)
	}

	imageFilterCache[key] = out
	imageFilterCacheMu.Unlock()
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

func lookupClipPathMaskCache(key clipPathMaskCacheKey) ([]byte, bool) {
	clipPathMaskCacheMu.RLock()
	cached, ok := clipPathMaskCache[key]
	clipPathMaskCacheMu.RUnlock()

	return cached, ok
}

func storeClipPathMaskCache(key clipPathMaskCacheKey, out []byte) {
	if len(out) == 0 || len(out) > imageEffectCacheMaxBytes {
		return
	}

	clipPathMaskCacheMu.Lock()
	if len(clipPathMaskCache) >= imageEffectCacheMaxEntries {
		clear(clipPathMaskCache)
	}

	clipPathMaskCache[key] = out
	clipPathMaskCacheMu.Unlock()
}
