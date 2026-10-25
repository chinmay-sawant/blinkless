package layout

import (
	"math"

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
)

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

	return textWidthKey{ //nolint:exhaustruct // zero tab flag is a valid key member
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

	w, ok := e.textWidthCache[key]

	return w, ok
}

// storeTextWidth records w under key unless the cache is full.
func (e *engine) storeTextWidth(key textWidthKey, w float64) {
	if e.textWidthCache == nil {
		e.textWidthCache = make(map[textWidthKey]float64, 256)
	}

	if len(e.textWidthCache) >= maxTextWidthCacheEntries {
		return
	}

	e.textWidthCache[key] = w
}

// runeWidthCacheKey builds the measureRuneFace key for sty and r.
func runeWidthCacheKey(sty *ResolvedStyle, scale float64, r rune) runeWidthKey {
	optical, variations, sizeBits := variationCacheBits(sty)

	return runeWidthKey{ //nolint:exhaustruct // zero tab flag is a valid key member
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
		r:             r,
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

// storeRuneWidth records w under key unless the cache is full.
func (e *engine) storeRuneWidth(key runeWidthKey, w float64) {
	if e.runeWidthCache == nil {
		e.runeWidthCache = make(map[runeWidthKey]float64, 256)
	}

	if len(e.runeWidthCache) >= maxRuneWidthCacheEntries {
		return
	}

	e.runeWidthCache[key] = w
}
