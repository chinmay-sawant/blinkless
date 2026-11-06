package layout

import (
	"math"
	"sync"
	"sync/atomic"

	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
)

const (
	// maxTextWidthCacheEntries bounds the per-Layout measureTextFace cache.
	// Thousands of repeated price labels share a few dozen distinct strings;
	// a huge document with all-distinct text stops storing once full.
	maxTextWidthCacheEntries = 8192
	// maxRuneWidthCacheEntries bounds the per-Layout measureRuneFace cache.
	maxRuneWidthCacheEntries = 4096
	// maxCachedTextLen caps cached strings by bytes. Short mono labels hit;
	// long paragraphs compute uncached so the key map cannot pin them.
	maxCachedTextLen = 128
	// initialWidthCacheEntries sizes a fresh per-Layout string width cache.
	// Thousands of repeated price labels share a few dozen distinct strings,
	// so a small initial map covers the steady state; the bounds above cap
	// growth for documents with all-distinct text.
	initialWidthCacheEntries = 256
)

// sharedTextWidthKey is the cross-layout measureTextFace key: the per-layout
// textWidthKey plus the engine zoom factor. sizeBits already carries the
// scaled used size, but letter and word spacing scale with e.scale too, so
// two layouts with the same scaled size and different zooms must not share.
type sharedTextWidthKey struct {
	base      textWidthKey
	scaleBits uint64
}

// sharedRuneWidthKey is the cross-layout measureRuneFace key: the per-layout
// runeWidthKey plus the engine zoom factor for the same spacing reason.
type sharedRuneWidthKey struct {
	base      runeWidthKey
	scaleBits uint64
}

type sharedTextWidthStore struct {
	mu      sync.RWMutex
	entries map[sharedTextWidthKey]float64
}

type sharedRuneWidthStore struct {
	mu      sync.RWMutex
	entries map[sharedRuneWidthKey]float64
}

//nolint:gochecknoglobals,exhaustruct // cross-layout string width cache, guarded by RWMutex.
var sharedTextWidths = &sharedTextWidthStore{
	entries: make(map[sharedTextWidthKey]float64, initialWidthCacheEntries),
}

//nolint:gochecknoglobals,exhaustruct // cross-layout rune width cache, guarded by RWMutex.
var sharedRuneWidths = &sharedRuneWidthStore{
	entries: make(map[sharedRuneWidthKey]float64, initialWidthCacheEntries),
}

//nolint:gochecknoglobals // test hook to prove bounded eviction without filling 8k entries.
var maxSharedTextWidthEntries = maxTextWidthCacheEntries

//nolint:gochecknoglobals // test hook to prove bounded eviction without filling 4k entries.
var maxSharedRuneWidthEntries = maxRuneWidthCacheEntries

//nolint:gochecknoglobals // cross-layout hit counters for tests, updated atomically.
var sharedTextHits atomic.Uint64

//nolint:gochecknoglobals // cross-layout miss counter for tests, updated atomically.
var sharedTextMisses atomic.Uint64

//nolint:gochecknoglobals // cross-layout rune hit counter for tests, updated atomically.
var sharedRuneHits atomic.Uint64

//nolint:gochecknoglobals // cross-layout rune miss counter for tests, updated atomically.
var sharedRuneMisses atomic.Uint64

// scaledAdvance returns the advance for r at size through the shared sized
// face, so repeated runes across layouts hit one points cache instead of
// recomputing from font units.
func scaledAdvance(face *pdf.Font, size float64, r rune) float64 {
	return pdf.SharedScaledFace(face, size).Advance(r)
}

// isValidSharedScale reports whether scale can key the shared cache: finite
// and positive, matching the layout zoom contract.
func isValidSharedScale(scale float64) bool {
	if math.IsNaN(scale) || math.IsInf(scale, 0) {
		return false
	}

	return scale > 0
}

// sharedTextBound returns the effective string cache bound.
func sharedTextBound() int {
	if maxSharedTextWidthEntries <= 0 {
		return maxTextWidthCacheEntries
	}

	return maxSharedTextWidthEntries
}

// sharedRuneBound returns the effective rune cache bound.
func sharedRuneBound() int {
	if maxSharedRuneWidthEntries <= 0 {
		return maxRuneWidthCacheEntries
	}

	return maxSharedRuneWidthEntries
}

// lookupSharedTextWidth returns the cross-layout string width for base and
// scale. An invalid scale bypasses the shared cache so the per-layout
// scratch stays the only cache.
func lookupSharedTextWidth(base textWidthKey, scale float64) (float64, bool) {
	if !isValidSharedScale(scale) {
		return 0, false
	}

	key := sharedTextWidthKey{base: base, scaleBits: math.Float64bits(scale)}

	sharedTextWidths.mu.RLock()
	width, ok := sharedTextWidths.entries[key]
	sharedTextWidths.mu.RUnlock()

	if ok {
		sharedTextHits.Add(1)
	} else {
		sharedTextMisses.Add(1)
	}

	return width, ok
}

// storeSharedTextWidth records width under base and scale unless the shared
// cache is bypassed or full. A full cache evicts one arbitrary entry to make
// room so the size stays bounded across layouts.
func storeSharedTextWidth(base textWidthKey, scale float64, width float64) {
	if !isValidSharedScale(scale) {
		return
	}

	if math.IsNaN(width) || math.IsInf(width, 0) {
		return
	}

	key := sharedTextWidthKey{base: base, scaleBits: math.Float64bits(scale)}

	sharedTextWidths.mu.Lock()
	defer sharedTextWidths.mu.Unlock()

	if sharedTextWidths.entries == nil {
		sharedTextWidths.entries = make(map[sharedTextWidthKey]float64, initialWidthCacheEntries)
	}

	if _, ok := sharedTextWidths.entries[key]; ok {
		sharedTextWidths.entries[key] = width

		return
	}

	if len(sharedTextWidths.entries) >= sharedTextBound() {
		for evict := range sharedTextWidths.entries {
			delete(sharedTextWidths.entries, evict)

			break
		}
	}

	sharedTextWidths.entries[key] = width
}

// lookupSharedRuneWidth returns the cross-layout rune width for base and
// scale, bypassing on invalid scale like the string path.
func lookupSharedRuneWidth(base runeWidthKey, scale float64) (float64, bool) {
	if !isValidSharedScale(scale) {
		return 0, false
	}

	key := sharedRuneWidthKey{base: base, scaleBits: math.Float64bits(scale)}

	sharedRuneWidths.mu.RLock()
	width, ok := sharedRuneWidths.entries[key]
	sharedRuneWidths.mu.RUnlock()

	if ok {
		sharedRuneHits.Add(1)
	} else {
		sharedRuneMisses.Add(1)
	}

	return width, ok
}

// storeSharedRuneWidth records a rune width with the same bounded eviction
// as the string path.
func storeSharedRuneWidth(base runeWidthKey, scale float64, width float64) {
	if !isValidSharedScale(scale) {
		return
	}

	if math.IsNaN(width) || math.IsInf(width, 0) {
		return
	}

	key := sharedRuneWidthKey{base: base, scaleBits: math.Float64bits(scale)}

	sharedRuneWidths.mu.Lock()
	defer sharedRuneWidths.mu.Unlock()

	if sharedRuneWidths.entries == nil {
		sharedRuneWidths.entries = make(map[sharedRuneWidthKey]float64, initialWidthCacheEntries)
	}

	if _, ok := sharedRuneWidths.entries[key]; ok {
		sharedRuneWidths.entries[key] = width

		return
	}

	if len(sharedRuneWidths.entries) >= sharedRuneBound() {
		for evict := range sharedRuneWidths.entries {
			delete(sharedRuneWidths.entries, evict)

			break
		}
	}

	sharedRuneWidths.entries[key] = width
}

// clearSharedMeasureCaches drops every cross-layout entry. Tests use it to
// isolate cases; layouts never call it.
func clearSharedMeasureCaches() {
	sharedTextWidths.mu.Lock()
	sharedTextWidths.entries = make(map[sharedTextWidthKey]float64, initialWidthCacheEntries)
	sharedTextWidths.mu.Unlock()

	sharedRuneWidths.mu.Lock()
	sharedRuneWidths.entries = make(map[sharedRuneWidthKey]float64, initialWidthCacheEntries)
	sharedRuneWidths.mu.Unlock()
}

// resetSharedMeasureCounters zeroes the cross-layout hit and miss counters.
func resetSharedMeasureCounters() {
	sharedTextHits.Store(0)
	sharedTextMisses.Store(0)
	sharedRuneHits.Store(0)
	sharedRuneMisses.Store(0)
}

// sharedTextCacheLen returns the live cross-layout string entry count.
func sharedTextCacheLen() int {
	sharedTextWidths.mu.RLock()
	defer sharedTextWidths.mu.RUnlock()

	return len(sharedTextWidths.entries)
}

// sharedRuneCacheLen returns the live cross-layout rune entry count.
func sharedRuneCacheLen() int {
	sharedRuneWidths.mu.RLock()
	defer sharedRuneWidths.mu.RUnlock()

	return len(sharedRuneWidths.entries)
}

// textWidthKey identifies one measureTextFace result. sizeBits is the scaled
// used size; lsBits/wsBits are the scaled spacing already added per rune;
// tabBits/tabIsLength select the tab-stop path; autospace and emoji select
// the per-rune gap and emoji-minimum paths; the family/weight/italic/optical
// fields select the fallback face for glyphs missing from the primary face.
type textWidthKey struct {
	face        *pdf.Font
	famHash     uint64
	weight      int
	italic      bool
	optical     string
	variations  string
	opticalSize uint64
	sizeBits    uint64
	lsBits      uint64
	wsBits      uint64
	tabBits     uint64
	tabIsLength bool
	autospace   bool
	emoji       string
	text        string
}

// runeWidthKey identifies one measureRuneFace result. Size here is the raw
// scaled FontSize (measureRuneFace does not apply font-size-adjust);
// widthScaleBits carries fontWidthScale; other fields mirror textWidthKey.
type runeWidthKey struct {
	famHash       uint64
	weight        int
	italic        bool
	optical       string
	variations    string
	opticalSize   uint64
	sizeBits      uint64
	lsBits        uint64
	wsBits        uint64
	tabBits       uint64
	tabIsLength   bool
	emoji         string
	widthScaleBit uint64
	r             rune
}

// textWidthCacheKey builds the measureTextFace key for sty and text. primary
// must be the already-resolved e.faceFor(sty) face and size the already
// scaled used size.
func textWidthCacheKey(sty *ResolvedStyle, primary *pdf.Font, size float64, text string) textWidthKey {
	optical, variations, sizeBits := variationCacheBits(sty)

	return textWidthKey{
		face:        primary,
		famHash:     sty.famHash,
		weight:      sty.FontWeight,
		italic:      sty.FontItalic,
		optical:     optical,
		variations:  variations,
		opticalSize: sizeBits,
		sizeBits:    math.Float64bits(size),
		lsBits:      math.Float64bits(sty.LetterSpacing),
		wsBits:      math.Float64bits(sty.WordSpacing),
		tabBits:     math.Float64bits(sty.TabSize),
		tabIsLength: isTabSizeLength(sty),
		autospace:   autospaceIdeographAlpha(sty.TextAutospace),
		emoji:       sty.FontVariantEmoji,
		text:        text,
	}
}

// lookupTextWidth returns the cached string width for key.
func (e *engine) lookupTextWidth(key textWidthKey) (float64, bool) {
	if len(e.textWidthCache) == 0 {
		return 0, false
	}

	width, ok := e.textWidthCache[key]

	return width, ok
}

// storeTextWidth records width under key unless the cache is full.
func (e *engine) storeTextWidth(key textWidthKey, width float64) {
	if e.textWidthCache == nil {
		e.textWidthCache = make(map[textWidthKey]float64, initialWidthCacheEntries)
	}

	if len(e.textWidthCache) >= maxTextWidthCacheEntries {
		return
	}

	e.textWidthCache[key] = width
}

// runeWidthCacheKey builds the measureRuneFace key for sty and runeValue.
func runeWidthCacheKey(sty *ResolvedStyle, scale float64, runeValue rune) runeWidthKey {
	optical, variations, sizeBits := variationCacheBits(sty)

	return runeWidthKey{
		famHash:       sty.famHash,
		weight:        sty.FontWeight,
		italic:        sty.FontItalic,
		optical:       optical,
		variations:    variations,
		opticalSize:   sizeBits,
		sizeBits:      math.Float64bits(sty.FontSize * scale),
		lsBits:        math.Float64bits(sty.LetterSpacing),
		wsBits:        math.Float64bits(sty.WordSpacing),
		tabBits:       math.Float64bits(sty.TabSize),
		tabIsLength:   isTabSizeLength(sty),
		emoji:         sty.FontVariantEmoji,
		widthScaleBit: math.Float64bits(fontWidthScale(sty)),
		r:             runeValue,
	}
}

// lookupRuneWidth returns the cached single-rune width for key.
func (e *engine) lookupRuneWidth(key runeWidthKey) (float64, bool) {
	if len(e.runeWidthCache) == 0 {
		return 0, false
	}

	w, ok := e.runeWidthCache[key]

	return w, ok
}

// storeRuneWidth records width under key unless the cache is full.
func (e *engine) storeRuneWidth(key runeWidthKey, width float64) {
	if e.runeWidthCache == nil {
		e.runeWidthCache = make(map[runeWidthKey]float64, initialWidthCacheEntries)
	}

	if len(e.runeWidthCache) >= maxRuneWidthCacheEntries {
		return
	}

	e.runeWidthCache[key] = width
}

// measureRuneFace measures a single rune with the same face selection as
// measureTextFace, without allocating string(r). The per-layout map serves
// repeats inside one run; the shared cache serves repeats across layouts.
func (e *engine) measureRuneFace(curRune rune, sty *ResolvedStyle) float64 {
	if sty == nil {
		return 0
	}

	if curRune == softHyphenRune || isVariationSelector(curRune) {
		return 0
	}

	key := runeWidthCacheKey(sty, e.scale, curRune)
	if w, ok := e.lookupRuneWidth(key); ok {
		return w
	}

	if w, ok := lookupSharedRuneWidth(key, e.scale); ok {
		e.storeRuneWidth(key, w)

		return w
	}

	w := e.measureRuneFaceUncached(curRune, sty)
	e.storeRuneWidth(key, w)
	storeSharedRuneWidth(key, e.scale, w)

	return w
}

// measureRuneFaceUncached measures one rune without consulting the rune cache.
func (e *engine) measureRuneFaceUncached(curRune rune, sty *ResolvedStyle) float64 {
	if curRune == '\t' {
		if isTabSizeLength(sty) {
			return e.scalePt(sty.TabSize)
		}

		tabSize := sty.TabSize
		if tabSize <= 0 {
			tabSize = 8
		}

		// \t expands to TabSize * space width per CSS tab-size.
		spaceW := e.measureRuneFace(' ', sty)
		if spaceW <= 0 {
			spaceW = sty.FontSize * e.scale * tabSpaceRatio
		}

		return spaceW * tabSize
	}

	size := sty.FontSize * e.scale
	face := e.faceForRune(sty, curRune)

	if face == nil {
		face = e.font
	}

	advance := scaledAdvance(face, size, curRune)
	if em := emojiPresentationAdvance(sty, curRune, size); em > advance {
		advance = em
	}

	if sty.LetterSpacing != 0 {
		advance += sty.LetterSpacing * e.scale
	}

	if curRune == ' ' && sty.WordSpacing != 0 {
		advance += sty.WordSpacing * e.scale
	}

	return advance * fontWidthScale(sty)
}

type faceRun struct {
	text string
	face *pdf.Font
	w    float64
}
