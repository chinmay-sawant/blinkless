package layout

const (
	vsTextPresentation     = '\uFE0E'
	vsEmojiPresentation    = '\uFE0F'
	fontVariantEmojiKW     = "emoji"
	fontVariantTextKW      = "text"
	fontVariantUnicode     = "unicode"
	emojiVariationCapacity = 4
	// liteEmojiFill is the solid fallback when emoji presentation is requested
	// and the face has no color glyph. Distinct from typical CSS color:#111 so
	// text vs emoji A/B demos change paint without a COLR emoji font.
	liteEmojiFillR = 1.0
	liteEmojiFillG = 0.78
	liteEmojiFillB = 0.12
)

// applyEmojiVariationSelectors inserts U+FE0E (text) or U+FE0F (emoji) after
// emoji-candidate codepoints according to font-variant-emoji. Existing
// variation selectors are replaced. Callers that paint text must drop
// selectors the face cannot cmap (bundled Liberation maps them to .notdef).
func applyEmojiVariationSelectors(text, variant string) string {
	if text == "" {
		return text
	}

	out := make([]rune, 0, len(text)+emojiVariationCapacity)

	for _, candidate := range text {
		if candidate == vsTextPresentation || candidate == vsEmojiPresentation {
			continue
		}

		out = append(out, candidate)

		if !isEmojiCandidate(candidate) {
			continue
		}

		vs := vsTextPresentation
		if wantsEmojiPresentation(candidate, variant) {
			vs = vsEmojiPresentation
		}

		out = append(out, vs)
	}

	return string(out)
}

func wantsEmojiPresentation(candidate rune, variant string) bool {
	if !isEmojiCandidate(candidate) {
		return false
	}

	switch variant {
	case fontVariantEmojiKW:
		return true
	case fontVariantTextKW:
		return false
	case fontVariantUnicode, fontVariantNormal:
		return emojiDefaultIsEmoji(candidate)
	default:
		return emojiDefaultIsEmoji(candidate)
	}
}

//nolint:cyclop // Unicode emoji ranges are independent, spec-defined cases
func isEmojiCandidate(candidate rune) bool {
	switch {
	case candidate >= 0x1F000 && candidate <= 0x1FAFF:
		return true
	case candidate >= 0x2600 && candidate <= 0x27BF:
		return true
	case candidate >= 0x2300 && candidate <= 0x23FF:
		return true
	case candidate >= 0x2B50 && candidate <= 0x2B55:
		return true
	case candidate == 0x00A9 || candidate == 0x00AE || candidate == 0x2122:
		return true
	default:
		return false
	}
}

func emojiDefaultIsEmoji(r rune) bool {
	return r >= 0x1F000 && r <= 0x1FAFF
}

func isVariationSelector(r rune) bool {
	return r == vsTextPresentation || r == vsEmojiPresentation
}

// emojiPresentationAdvance returns a 1em minimum advance for emoji
// presentation of an emoji-candidate rune, else 0 (caller keeps glyph width).
func emojiPresentationAdvance(sty *ResolvedStyle, r rune, size float64) float64 {
	if sty == nil || isVariationSelector(r) {
		return 0
	}

	if !wantsEmojiPresentation(r, sty.FontVariantEmoji) {
		return 0
	}

	return size
}

// fontVariantEmojiFill applies the lite emoji presentation fill when every
// non-space, non-VS rune in text is an emoji candidate under emoji
// presentation. Mixed runs keep CSS color. A text-variant run can never
// qualify, so it exits before the per-rune scan.
func fontVariantEmojiFill(sty *ResolvedStyle, text string) (float64, float64, float64, bool) {
	if sty == nil || text == "" {
		return 0, 0, 0, false
	}

	if sty.FontVariantEmoji == fontVariantTextKW {
		return 0, 0, 0, false
	}

	variant := sty.FontVariantEmoji
	hasEmoji := false

	for _, r := range text {
		if r == ' ' || r == '\t' || isVariationSelector(r) {
			continue
		}

		if !wantsEmojiPresentation(r, variant) {
			return 0, 0, 0, false
		}

		hasEmoji = true
	}

	if !hasEmoji {
		return 0, 0, 0, false
	}

	return liteEmojiFillR, liteEmojiFillG, liteEmojiFillB, true
}

// emojiAdvanceActive reports whether emoji presentation can raise an advance
// for sty: font-variant-emoji:text never presents emoji, every other value
// may for at least one codepoint. Callers hoist this per run and skip the
// per-rune presentation check entirely when it is false.
func emojiAdvanceActive(sty *ResolvedStyle) bool {
	if sty == nil {
		return false
	}

	return sty.FontVariantEmoji != fontVariantTextKW
}

// emojiRunState is the per-run hoisted form of the emoji presentation
// decision: one variant read and one active check serve the whole run.
type emojiRunState struct {
	variant string
	size    float64
	active  bool
}

// emojiRunFor hoists the emoji presentation decision for sty at size.
func emojiRunFor(sty *ResolvedStyle, size float64) emojiRunState {
	if sty == nil {
		return emojiRunState{}
	}

	return emojiRunState{
		variant: sty.FontVariantEmoji,
		size:    size,
		active:  emojiAdvanceActive(sty),
	}
}

// advance returns the emoji presentation minimum for r, or 0 when the caller
// keeps the glyph width. It matches emojiPresentationAdvance exactly.
func (s emojiRunState) advance(r rune) float64 {
	if !s.active || isVariationSelector(r) {
		return 0
	}

	if !wantsEmojiPresentation(r, s.variant) {
		return 0
	}

	return s.size
}

// textHasVariantSelector reports whether text holds a variation selector.
// The cached string width skips variation selectors while the per-word
// paint loop advances them, so such runs keep the loop.
func textHasVariantSelector(text string) bool {
	for _, r := range text {
		if isVariationSelector(r) {
			return true
		}
	}

	return false
}

// textHasEmojiCandidate reports whether text holds an emoji candidate. The
// cached string width folds the emoji presentation minimum while the
// per-word paint loop does not, so such runs keep the loop when emoji
// presentation is active.
func textHasEmojiCandidate(text string) bool {
	for _, r := range text {
		if isEmojiCandidate(r) {
			return true
		}
	}

	return false
}
