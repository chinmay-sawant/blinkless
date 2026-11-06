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
	if w, ok := e.fetchRuneWidth(key, e.scale); ok {
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
	if em := emojiRunFor(sty, size).advance(curRune); em > advance {
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
	// paint is the post-transform paint string (text-transform applied,
	// soft hyphens stripped) when it differs from text. primaryFaceRun
	// stores it so paint reuses the collect-time string instead of
	// transforming again; empty means paint derives text the old way.
	paint string
}

// textCoverageKey identifies one face-coverage bit: the face that must
// cover every non-whitespace rune plus the source text. Coverage is pure
// for a fixed face and text, so metrics (inlineFontMetrics) and paint
// (primaryFaceRun, splitTextByFace) share one bit instead of each scanning
// GlyphIDs per item.
type textCoverageKey struct {
	face *pdf.Font
	text string
}

type textCoverageStore struct {
	mu      sync.RWMutex
	entries map[textCoverageKey]bool
}

//nolint:gochecknoglobals,exhaustruct // cross-layout coverage cache, guarded by RWMutex.
var sharedTextCoverage = &textCoverageStore{
	entries: make(map[textCoverageKey]bool, initialWidthCacheEntries),
}

//nolint:gochecknoglobals // cross-layout coverage hit counter for tests, updated atomically.
var sharedCoverageHits atomic.Uint64

//nolint:gochecknoglobals // cross-layout coverage miss counter for tests, updated atomically.
var sharedCoverageMisses atomic.Uint64

// lookupTextCoverage returns the cached all-primary bit for face and text.
// Long strings bypass the cache but still report correctly.
func lookupTextCoverage(face *pdf.Font, text string) (bool, bool) {
	if face == nil || len(text) > maxCachedTextLen {
		return false, false
	}

	sharedTextCoverage.mu.RLock()
	covered, ok := sharedTextCoverage.entries[textCoverageKey{face: face, text: text}]
	sharedTextCoverage.mu.RUnlock()

	if ok {
		sharedCoverageHits.Add(1)
	} else {
		sharedCoverageMisses.Add(1)
	}

	return covered, ok
}

// storeTextCoverage records the all-primary bit unless the text is too long
// or the shared cache is bypassed. A full cache evicts one arbitrary entry
// so the size stays bounded across layouts.
func storeTextCoverage(face *pdf.Font, text string, covered bool) {
	if face == nil || len(text) > maxCachedTextLen {
		return
	}

	key := textCoverageKey{face: face, text: text}

	sharedTextCoverage.mu.Lock()
	defer sharedTextCoverage.mu.Unlock()

	if sharedTextCoverage.entries == nil {
		sharedTextCoverage.entries = make(map[textCoverageKey]bool, initialWidthCacheEntries)
	}

	if _, ok := sharedTextCoverage.entries[key]; ok {
		sharedTextCoverage.entries[key] = covered

		return
	}

	if len(sharedTextCoverage.entries) >= sharedTextBound() {
		for evict := range sharedTextCoverage.entries {
			delete(sharedTextCoverage.entries, evict)

			break
		}
	}

	sharedTextCoverage.entries[key] = covered
}

// clearCoverageCache drops every cross-layout coverage entry. Tests use it
// to isolate cases; layouts never call it.
func clearCoverageCache() {
	sharedTextCoverage.mu.Lock()
	sharedTextCoverage.entries = make(map[textCoverageKey]bool, initialWidthCacheEntries)
	sharedTextCoverage.mu.Unlock()
}

// resetCoverageCounters zeroes the cross-layout coverage hit and miss counters.
func resetCoverageCounters() {
	sharedCoverageHits.Store(0)
	sharedCoverageMisses.Store(0)
}

// fetchTextWidth is the single combined string-width lookup: the per-layout
// map first (no locking), then the shared cache, filling the per-layout map
// on a shared hit so a repeat word pays one map read. A cold per-layout map
// skips straight to the shared cache.
func (e *engine) fetchTextWidth(key textWidthKey, scale float64) (float64, bool) {
	if len(e.textWidthCache) > 0 {
		if width, ok := e.textWidthCache[key]; ok {
			return width, true
		}
	}

	if width, ok := lookupSharedTextWidth(key, scale); ok {
		e.storeTextWidth(key, width)

		return width, true
	}

	return 0, false
}

// fetchRuneWidth is the single combined single-rune lookup with the same
// local-first, shared-second, fill-local-on-hit shape as fetchTextWidth.
func (e *engine) fetchRuneWidth(key runeWidthKey, scale float64) (float64, bool) {
	if len(e.runeWidthCache) > 0 {
		if width, ok := e.runeWidthCache[key]; ok {
			return width, true
		}
	}

	if width, ok := lookupSharedRuneWidth(key, scale); ok {
		e.storeRuneWidth(key, width)

		return width, true
	}

	return 0, false
}

// runeKeyPrefix is the style-fixed half of a runeWidthKey, precomputed once
// per multi-rune walk so the loop pays no repeat style hashing. The per-rune
// key is prefix.key(runeValue).
type runeKeyPrefix struct {
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
}

// makeRuneKeyPrefix precomputes the style-fixed cache key half for sty at
// scale. It mirrors runeWidthCacheKey field for field.
func makeRuneKeyPrefix(sty *ResolvedStyle, scale float64) runeKeyPrefix {
	optical, variations, sizeBits := variationCacheBits(sty)

	return runeKeyPrefix{
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
	}
}

// key completes the prefix with one rune.
func (p runeKeyPrefix) key(runeValue rune) runeWidthKey {
	return runeWidthKey{
		famHash:       p.famHash,
		weight:        p.weight,
		italic:        p.italic,
		optical:       p.optical,
		variations:    p.variations,
		opticalSize:   p.opticalSize,
		sizeBits:      p.sizeBits,
		lsBits:        p.lsBits,
		wsBits:        p.wsBits,
		tabBits:       p.tabBits,
		tabIsLength:   p.tabIsLength,
		emoji:         p.emoji,
		widthScaleBit: p.widthScaleBit,
		r:             runeValue,
	}
}

// primeRuneBatch resolves every rune of text into the per-layout rune cache
// with one shared-cache read lock for the whole run: per-layout hits stay,
// shared hits fill the per-layout map, and leftovers measure uncached. After
// it returns, an in-order walk pays only per-layout map reads. Values match
// sequential measureRuneFace calls exactly.
func (e *engine) primeRuneBatch(sty *ResolvedStyle, prefix runeKeyPrefix, text string) {
	var missKeys []runeWidthKey

	var missRunes []rune

	for _, runic := range text {
		if runic == softHyphenRune || isVariationSelector(runic) {
			continue
		}

		key := prefix.key(runic)
		if _, ok := e.lookupRuneWidth(key); ok {
			continue
		}

		missKeys = append(missKeys, key)
		missRunes = append(missRunes, runic)
	}

	if len(missKeys) == 0 {
		return
	}

	if !isValidSharedScale(e.scale) {
		e.measureRuneLeftovers(sty, missKeys, missRunes)

		return
	}

	scaleBits := math.Float64bits(e.scale)

	var pendingKeys []runeWidthKey

	var pendingRunes []rune

	sharedRuneWidths.mu.RLock()

	for i, key := range missKeys {
		width, ok := sharedRuneWidths.entries[sharedRuneWidthKey{base: key, scaleBits: scaleBits}]
		if !ok {
			pendingKeys = append(pendingKeys, key)
			pendingRunes = append(pendingRunes, missRunes[i])

			continue
		}

		sharedRuneHits.Add(1)
		e.storeRuneWidth(key, width)
	}

	sharedRuneWidths.mu.RUnlock()
	sharedRuneMisses.Add(uint64(len(pendingKeys)))
	e.measureRuneLeftovers(sty, pendingKeys, pendingRunes)
}

// measureRuneLeftovers measures runes that missed both caches and stores
// them in both, mirroring the measureRuneFace tail. A duplicate rune that
// an earlier leftover already stored is skipped via the per-layout map.
func (e *engine) measureRuneLeftovers(sty *ResolvedStyle, keys []runeWidthKey, runes []rune) {
	for i, key := range keys {
		if _, ok := e.lookupRuneWidth(key); ok {
			continue
		}

		width := e.measureRuneFaceUncached(runes[i], sty)
		e.storeRuneWidth(key, width)
		storeSharedRuneWidth(key, e.scale, width)
	}
}

// eachRuneAdvance walks text in order, calling fn per rune with the same
// advance measureRuneFace would return. Returning false from fn stops the
// walk. The batch prime above keeps the whole run to one shared-cache read
// lock however many runes the text holds.
func (e *engine) eachRuneAdvance(text string, sty *ResolvedStyle, fn func(rune, float64) bool) {
	if text == "" || sty == nil {
		return
	}

	prefix := makeRuneKeyPrefix(sty, e.scale)
	e.primeRuneBatch(sty, prefix, text)

	for _, runic := range text {
		if runic == softHyphenRune || isVariationSelector(runic) {
			if !fn(runic, 0) {
				return
			}

			continue
		}

		if width, ok := e.lookupRuneWidth(prefix.key(runic)); ok {
			if !fn(runic, width) {
				return
			}

			continue
		}

		// Full caches skip stores; fall back to the single path so the
		// value is still exact.
		if !fn(runic, e.measureRuneFace(runic, sty)) {
			return
		}
	}
}
