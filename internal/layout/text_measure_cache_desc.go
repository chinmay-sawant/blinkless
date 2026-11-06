package layout

import (
	"math"
	"sync"

	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
)

// maxInternedMeasureStyles bounds the process-wide measurement-style intern
// table. A document has a handful of computed text styles, so the table stays
// small; a pathological cascade that exceeds the bound falls back to the
// per-layout cache instead of growing without limit.
const maxInternedMeasureStyles = 4096

// textMeasureDesc is the exact identity of one measurement style: the
// resolved primary face, every field that selects fallback faces, the scaled
// used size, and every field that changes a run's advance. Two styles share a
// cached width only when their descriptors are exactly equal.
//
// sizeBits is the scaled used size (measureTextFace) or the scaled raw font
// size (measureRuneFace). It lives here, not in textWidthKey, so a style ID
// names the complete measurement identity.
type textMeasureDesc struct {
	face        *pdf.Font
	famHash     uint64
	weight      int
	italic      bool
	optical     string
	variations  string
	emoji       string
	opticalSize uint64
	sizeBits    uint64
	lsBits      uint64
	wsBits      uint64
	tabBits     uint64
	tabIsLength bool
	autospace   bool
}

// measureStyleTable interns descriptors to stable IDs. IDs start at 1; 0 is
// reserved for "not interned" so callers bypass the shared cache. The table
// and its counter are process-wide and never reset: a width cached under one
// ID can never be read under another ID even across layouts and tests.
type measureStyleTable struct {
	mu   sync.Mutex
	ids  map[textMeasureDesc]uint64
	next uint64
}

//nolint:gochecknoglobals,exhaustruct // process-wide style IDs, guarded by mu.
var measureStyles = &measureStyleTable{
	ids:  make(map[textMeasureDesc]uint64, initialWidthCacheEntries),
	next: 1,
}

// measureStyleRef is the engine-local intern lookup: one resolved style at one
// scaled size. The global table and its string hashing run once per entry.
type measureStyleRef struct {
	style    *ResolvedStyle
	sizeBits uint64
}

// measureStyleID returns the interned ID for sty at sizeBits, interning the
// full descriptor on first use. It returns 0 when the intern table is full or
// the inputs are nil; callers then keep only the per-layout cache.
func (e *engine) measureStyleID(sty *ResolvedStyle, primary *pdf.Font, sizeBits uint64) uint64 {
	if e == nil || sty == nil {
		return 0
	}

	ref := measureStyleRef{style: sty, sizeBits: sizeBits}
	if e.measureStyleIDs != nil {
		if id, ok := e.measureStyleIDs[ref]; ok {
			return id
		}
	}

	id := internMeasureStyle(measureStyleDescOf(sty, primary, sizeBits))
	if e.measureStyleIDs == nil {
		e.measureStyleIDs = make(map[measureStyleRef]uint64, initialWidthCacheEntries)
	}

	// Cache the zero too: a full table stays full, so retrying would only
	// repeat the lock and descriptor hash on every text run.
	e.measureStyleIDs[ref] = id

	return id
}

// cachedMeasureStyleID returns the engine-local ID for sty at sizeBits without
// building a descriptor or resolving a face. ok is true even when the cached
// ID is 0 (the intern table was full), so callers can stay on the fast path.
func (e *engine) cachedMeasureStyleID(sty *ResolvedStyle, sizeBits uint64) (uint64, bool) {
	if e == nil || sty == nil || e.measureStyleIDs == nil {
		return 0, false
	}

	id, ok := e.measureStyleIDs[measureStyleRef{style: sty, sizeBits: sizeBits}]

	return id, ok
}

// measureStyleDescOf builds the full descriptor for one style at one size.
func measureStyleDescOf(sty *ResolvedStyle, primary *pdf.Font, sizeBits uint64) textMeasureDesc {
	optical, variations, opticalSize := variationCacheBits(sty)

	return textMeasureDesc{
		face:        primary,
		famHash:     sty.famHash,
		weight:      sty.FontWeight,
		italic:      sty.FontItalic,
		optical:     optical,
		variations:  variations,
		emoji:       sty.FontVariantEmoji,
		opticalSize: opticalSize,
		sizeBits:    sizeBits,
		lsBits:      math.Float64bits(sty.LetterSpacing),
		wsBits:      math.Float64bits(sty.WordSpacing),
		tabBits:     math.Float64bits(sty.TabSize),
		tabIsLength: isTabSizeLength(sty),
		autospace:   autospaceIdeographAlpha(sty.TextAutospace),
	}
}

// internMeasureStyle returns the stable ID for desc, assigning the next one on
// first use. It returns 0 when the table is full so callers bypass the shared
// cache rather than reusing an ID that names different measurements.
func internMeasureStyle(desc textMeasureDesc) uint64 {
	return measureStyles.intern(desc, maxInternedMeasureStyles)
}

// intern is the bounded insert used by the process-wide table. Existing
// descriptors keep their ID even after the table fills.
func (t *measureStyleTable) intern(desc textMeasureDesc, bound int) uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	if id, ok := t.ids[desc]; ok {
		return id
	}

	if bound <= 0 || len(t.ids) >= bound {
		return 0
	}

	id := t.next
	t.next++
	t.ids[desc] = id

	return id
}

// internedMeasureStyleCount returns the live intern table size.
func internedMeasureStyleCount() int {
	measureStyles.mu.Lock()
	defer measureStyles.mu.Unlock()

	return len(measureStyles.ids)
}
