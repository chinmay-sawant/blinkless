package layout

import (
	"encoding/binary"
	"math"
	"sync"
	"sync/atomic"

	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
)

// lineHeightOf returns the used line height for a style in style points: the
// declared line-height when one is set, otherwise the normal line height from
// the resolved face's metrics.
func (e *engine) lineHeightOf(style *ResolvedStyle) float64 {
	if style == nil {
		return 0
	}

	if style.LineHeight > 0 {
		return style.LineHeight
	}

	return e.normalLineHeight(style)
}

// normalLineHeight returns line-height:normal in style points. Browsers derive
// the normal line height from the face's hhea ascent, descent, and line gap,
// each rounded to whole CSS pixels at the used size, so the line box is an
// integer number of CSS pixels. The engine's style lengths are points, hence
// the point to pixel round trip.
//
// The result is memoized across layouts: every same-style fragment of one
// text node resolves the same face, size, and zoom, so repeats share one
// entry instead of each redoing the rounding math.
func (e *engine) normalLineHeight(style *ResolvedStyle) float64 {
	face := e.faceFor(style)
	if face == nil || face.UnitsPerEm() <= 0 {
		return defaultLineHeightRatio * style.FontSize
	}

	key := lineHeightKey{
		face:     face,
		sizeBits: math.Float64bits(style.FontSize),
		scaleBit: math.Float64bits(e.scale),
	}
	if last := lastLineHeight.Load(); last != nil && last.key == key {
		sharedLineHeightHits.Add(1)

		return last.height
	}

	if height, ok := lookupLineHeight(face, style.FontSize, e.scale); ok {
		lastLineHeight.Store(&lastLineHeightEntry{key: key, height: height})

		return height
	}

	upem := float64(face.UnitsPerEm())
	// Used CSS pixels for this style (pxToPt converts px -> pt).
	pxSize := style.FontSize * e.scale / pxToPt(1)

	ascent := math.Max(0, math.Round(float64(face.Ascent())/upem*pxSize))
	descent := math.Max(0, math.Round(float64(-face.Descent())/upem*pxSize))
	gap := math.Max(0, math.Round(e.faceLineGapRatio(face)*pxSize))

	linePx := ascent + descent + gap
	if linePx <= 0 {
		return defaultLineHeightRatio * style.FontSize
	}

	height := pxToPt(linePx) / e.scale
	storeLineHeight(face, style.FontSize, e.scale, height)
	lastLineHeight.Store(&lastLineHeightEntry{key: key, height: height})

	return height
}

// faceLineGapRatio returns the face's hhea line gap as a fraction of the em,
// or 0 when the face exposes none. Browsers include the line gap in
// line-height: normal, but the font package does not surface it, so the raw
// SFNT bytes are read here. Results are cached per face.
func (e *engine) faceLineGapRatio(face *pdf.Font) float64 {
	if face == nil {
		return 0
	}

	if e.lineGapByFace == nil {
		e.lineGapByFace = make(map[*pdf.Font]float64)
	}

	if ratio, ok := e.lineGapByFace[face]; ok {
		return ratio
	}

	ratio := hheaLineGapRatio(face.Bytes(), face.UnitsPerEm())
	e.lineGapByFace[face] = ratio

	return ratio
}

// hheaLineGapRatio reads lineGap/unitsPerEm from the hhea table of a raw SFNT
// face. It returns 0 when the bytes do not contain a usable hhea table.
func hheaLineGapRatio(data []byte, upem int16) float64 {
	if upem <= 0 {
		return 0
	}

	tableOff, tableLen, ok := findSFNTTable(data, "hhea")
	if !ok || tableLen < hheaLineGapOffset+2 {
		return 0
	}

	//nolint:gosec // hhea line gap is a signed int16 per spec
	gap := int16(binary.BigEndian.Uint16(data[tableOff+hheaLineGapOffset : tableOff+hheaLineGapOffset+2]))
	if gap <= 0 {
		return 0
	}

	return float64(gap) / float64(upem)
}

// findSFNTTable returns the byte offset and length of an SFNT table, or
// ok=false when the directory or the table bounds are unusable.
func findSFNTTable(data []byte, tag string) (int, int, bool) {
	if len(data) < sfntHeaderSize {
		return 0, 0, false
	}

	numTables := int(binary.BigEndian.Uint16(data[4:6]))
	if numTables <= 0 || sfntHeaderSize+sfntTableRecordSize*numTables > len(data) {
		return 0, 0, false
	}

	for idx := range numTables {
		off := sfntHeaderSize + sfntTableRecordSize*idx
		if string(data[off:off+4]) != tag {
			continue
		}

		tableOff := int(binary.BigEndian.Uint32(data[off+8 : off+12]))
		tableLen := int(binary.BigEndian.Uint32(data[off+12 : off+16]))

		if tableOff < 0 || tableLen < 0 || tableOff+tableLen > len(data) {
			return 0, 0, false
		}

		return tableOff, tableLen, true
	}

	return 0, 0, false
}

const (
	// sfntHeaderSize is the offset table size of an SFNT file (version + numTables + 3 fields).
	sfntHeaderSize = 12
	// sfntTableRecordSize is one SFNT table directory record (tag + checksum + offset + length).
	sfntTableRecordSize = 16
	// hheaLineGapOffset is the byte offset of lineGap inside the hhea table.
	hheaLineGapOffset = 8
	// maxLineHeightCacheEntries bounds the cross-layout normal line-height
	// cache. One entry per (face, size, zoom); a huge document with many
	// sizes stops storing once full.
	maxLineHeightCacheEntries = 1024
)

// lineHeightKey identifies one memoized line-height:normal result: the face
// plus the raw used size and the engine zoom. pxSize derives from size and
// zoom, and the face carries the metrics, so the key is complete.
type lineHeightKey struct {
	face     *pdf.Font
	sizeBits uint64
	scaleBit uint64
}

type lineHeightStore struct {
	mu      sync.RWMutex
	entries map[lineHeightKey]float64
}

//nolint:gochecknoglobals,exhaustruct // cross-layout line-height cache, guarded by RWMutex.
var sharedLineHeights = &lineHeightStore{
	entries: make(map[lineHeightKey]float64, maxLineHeightCacheEntries),
}

// lastLineHeight is the most recent result. A table repeats one text style
// across thousands of cells, so the atomic read serves almost every call
// without taking the shared cache's RWMutex.
type lastLineHeightEntry struct {
	key    lineHeightKey
	height float64
}

//nolint:gochecknoglobals // one-entry read-mostly cache, replaced atomically
var lastLineHeight atomic.Pointer[lastLineHeightEntry]

//nolint:gochecknoglobals // cross-layout line-height hit counter for tests, updated atomically.
var sharedLineHeightHits atomic.Uint64

//nolint:gochecknoglobals // cross-layout line-height miss counter for tests, updated atomically.
var sharedLineHeightMisses atomic.Uint64

// lookupLineHeight returns the memoized line-height:normal for face, size,
// and scale.
func lookupLineHeight(face *pdf.Font, size, scale float64) (float64, bool) {
	if face == nil {
		return 0, false
	}

	key := lineHeightKey{
		face:     face,
		sizeBits: math.Float64bits(size),
		scaleBit: math.Float64bits(scale),
	}

	sharedLineHeights.mu.RLock()
	height, found := sharedLineHeights.entries[key]
	sharedLineHeights.mu.RUnlock()

	if found {
		sharedLineHeightHits.Add(1)
	} else {
		sharedLineHeightMisses.Add(1)
	}

	return height, found
}

// storeLineHeight records height unless the cache is full. A full cache
// evicts one arbitrary entry to stay bounded.
func storeLineHeight(face *pdf.Font, size, scale, height float64) {
	if face == nil || math.IsNaN(height) || math.IsInf(height, 0) {
		return
	}

	key := lineHeightKey{
		face:     face,
		sizeBits: math.Float64bits(size),
		scaleBit: math.Float64bits(scale),
	}

	sharedLineHeights.mu.Lock()
	defer sharedLineHeights.mu.Unlock()

	if sharedLineHeights.entries == nil {
		sharedLineHeights.entries = make(map[lineHeightKey]float64, maxLineHeightCacheEntries)
	}

	if _, ok := sharedLineHeights.entries[key]; ok {
		sharedLineHeights.entries[key] = height

		return
	}

	if len(sharedLineHeights.entries) >= maxLineHeightCacheEntries {
		for evict := range sharedLineHeights.entries {
			delete(sharedLineHeights.entries, evict)

			break
		}
	}

	sharedLineHeights.entries[key] = height
}

// clearLineHeightCache drops every cross-layout line-height entry. Tests use
// it to isolate cases; layouts never call it.
func clearLineHeightCache() {
	sharedLineHeights.mu.Lock()
	sharedLineHeights.entries = make(map[lineHeightKey]float64, maxLineHeightCacheEntries)
	sharedLineHeights.mu.Unlock()
}

// resetLineHeightCounters zeroes the cross-layout line-height counters.
func resetLineHeightCounters() {
	sharedLineHeightHits.Store(0)
	sharedLineHeightMisses.Store(0)
}
