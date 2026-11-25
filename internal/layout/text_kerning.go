package layout

import (
	"strings"

	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
)

// kernTextWidth reports whether the author enabled kerning explicitly.
// OwnFrame paints default runs with kerning, but the default Blinkless
// geometry still uses individual glyph advances. Explicit kern settings
// take precedence over font-kerning, as they do when painting.
func kernTextWidth(style *ResolvedStyle) bool {
	if style == nil || style.fontGroup == nil {
		return false
	}

	if style.inlineTextGroup != nil && style.inlineTextGroup.LetterSpacing != 0 {
		return false
	}

	fg := style.fontGroup
	enabled := fg.FontKerning == fontKerningNormal

	if fg.FontFeatureSettings != "" && strings.Contains(fg.FontFeatureSettings, `"kern"`) {
		for _, part := range splitFontVariationList(fg.FontFeatureSettings) {
			if tag, value, ok := parseFeatureSettingsPart(part); ok && tag == "kern" {
				enabled = value != 0
			}
		}
	}

	return enabled
}

// textKerningAdjustment follows the font and spacing boundaries of replay.
// A glyph cannot kern with a glyph drawn in a separate face or autospace span.
func (e *engine) textKerningAdjustment(
	content string, sty *ResolvedStyle, primary *pdf.Font, size float64, allowFallback bool,
) float64 {
	if !kernTextWidth(sty) {
		return 0
	}

	content = stripSoftHyphens(content)
	settings := fontShapingFeatureSettings(sty)
	lang := fontShapingLanguage(sty)
	auto := autospaceForRun(sty)
	face, start := primary, 0

	var (
		prev  rune
		total float64
	)

	for index, current := range content {
		next := e.kerningFace(sty, primary, current, allowFallback)
		if next != face || auto.gap(prev, current, size) != 0 || current == '\t' || prev == '\t' {
			total += face.KerningAdjustment(content[start:index], size, settings, lang)
			face, start = next, index
		}

		prev = current
	}

	return total + face.KerningAdjustment(content[start:], size, settings, lang)
}

func (e *engine) kerningFace(sty *ResolvedStyle, primary *pdf.Font, current rune, allowFallback bool) *pdf.Font {
	if !allowFallback || isRuneWhitespace(current) || primary.GlyphID(current) != 0 {
		return primary
	}

	if face := e.faceForRuneFallback(sty, current, primary); face != nil {
		return face
	}

	return e.font
}
