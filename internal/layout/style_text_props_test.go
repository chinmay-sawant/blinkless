//nolint:lll // targeted unit tests for Phase 80
package layout

import (
	"testing"
)

func TestTextPropsWave3(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       t.Context(),
		err:       nil,
		work:      0,
		sheets:    nil,
		viewportW: 800,
	}

	s := initialStyle()
	raw := map[string]string{
		"text-align-last":      "center",
		"tab-size":             "4",
		"text-wrap-mode":       "nowrap",
		"text-wrap-style":      "balance",
		"white-space-collapse": "preserve",
		"white-space-trim":     "discard-inner",
		"hyphens":              "manual",
		"hyphenate-character":  "~",
		"text-justify":         "inter-word",
		"line-break":           "strict",
	}
	applyRestProps(&s, raw, ctx, nil)

	if s.inlineText().TextAlignLast != "center" {
		t.Errorf("expected TextAlignLast 'center', got %q", s.inlineText().TextAlignLast)
	}
	if s.inlineText().TabSize != 4 {
		t.Errorf("expected TabSize 4, got %v", s.inlineText().TabSize)
	}
	if s.inlineText().TextWrapMode != "nowrap" || s.inlineText().TextWrapStyle != "balance" {
		t.Errorf("expected text-wrap mode/style nowrap/balance, got mode=%q, style=%q", s.inlineText().TextWrapMode, s.inlineText().TextWrapStyle)
	}
	if s.inlineText().WhiteSpaceCollapse != "preserve" || s.inlineText().WhiteSpaceTrim != "discard-inner" {
		t.Errorf("expected white-space collapse/trim preserve/discard-inner, got %q, %q", s.inlineText().WhiteSpaceCollapse, s.inlineText().WhiteSpaceTrim)
	}
	if s.inlineText().Hyphens != "manual" || s.inlineText().HyphenateCharacter != "~" {
		t.Errorf("expected hyphens manual/~, got %q, %q", s.inlineText().Hyphens, s.inlineText().HyphenateCharacter)
	}
	if s.inlineText().TextJustify != "inter-word" || s.inlineText().LineBreak != "strict" {
		t.Errorf("expected text-justify inter-word / line-break strict, got %q, %q", s.inlineText().TextJustify, s.inlineText().LineBreak)
	}
}

func TestTextDecorationPropsWave3(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       t.Context(),
		err:       nil,
		work:      0,
		sheets:    nil,
		viewportW: 800,
	}

	s := initialStyle()
	raw := map[string]string{
		"text-decoration-line":      "underline",
		"text-decoration-color":     "red",
		"text-decoration-style":     "dashed",
		"text-decoration-thickness": "2pt",
		"text-underline-offset":     "3pt",
		"text-underline-position":   "under",
		"text-shadow":               "1pt 2pt 3pt blue",
	}
	applyRestProps(&s, raw, ctx, nil)

	if s.inlineText().TextDecorationLine != "underline" || s.inlineText().TextDecoration != cssTextDecorationUnderline {
		t.Errorf("mismatch on decoration line: line=%q, dec=%q", s.inlineText().TextDecorationLine, s.inlineText().TextDecoration)
	}
	if s.inlineText().TextDecorationColor != [3]float64{1, 0, 0} || !s.inlineText().TextDecorationColorSet {
		t.Errorf("mismatch on decoration color: color=%v, set=%v", s.inlineText().TextDecorationColor, s.inlineText().TextDecorationColorSet)
	}
	if s.inlineText().TextDecorationStyle != "dashed" || s.inlineText().TextDecorationThickness != 2 {
		t.Errorf("mismatch on decoration style/thickness: style=%q, th=%v", s.inlineText().TextDecorationStyle, s.inlineText().TextDecorationThickness)
	}
	if s.inlineText().TextUnderlineOffset != 3 || s.inlineText().TextUnderlinePosition != "under" {
		t.Errorf("mismatch on underline offset/pos: off=%v, pos=%q", s.inlineText().TextUnderlineOffset, s.inlineText().TextUnderlinePosition)
	}
	if !s.inlineText().TextShadowSet || s.inlineText().TextShadowX != 1 || s.inlineText().TextShadowY != 2 || s.inlineText().TextShadowBlur != 3 ||
		s.inlineText().TextShadowColor != [3]float64{0, 0, 1} {
		t.Errorf("mismatch on text shadow: set=%v, x=%v, y=%v, blur=%v, color=%v",
			s.inlineText().TextShadowSet, s.inlineText().TextShadowX, s.inlineText().TextShadowY, s.inlineText().TextShadowBlur, s.inlineText().TextShadowColor)
	}
}

func TestTextEmphasisPropsReachWave3(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       t.Context(),
		err:       nil,
		work:      0,
		sheets:    nil,
		viewportW: 800,
	}

	s := initialStyle()
	raw := map[string]string{
		"text-emphasis":          "filled dot",
		"text-emphasis-style":    "open sesame",
		"text-emphasis-color":    "#c00",
		"text-emphasis-position": "under",
		"text-emphasis-skip":     "spaces",
	}
	applyRestProps(&s, raw, ctx, nil)

	if s.CustomProps["__emph_style"] != "open sesame" {
		t.Errorf("expected __emph_style open sesame, got %q", s.CustomProps["__emph_style"])
	}
	if s.CustomProps["__emph_color"] != "#c00" {
		t.Errorf("expected __emph_color #c00, got %q", s.CustomProps["__emph_color"])
	}
	if s.CustomProps["__emph_position"] != "under" {
		t.Errorf("expected __emph_position under, got %q", s.CustomProps["__emph_position"])
	}
	if s.CustomProps["__emph_skip"] != "spaces" {
		t.Errorf("expected __emph_skip spaces, got %q", s.CustomProps["__emph_skip"])
	}
}
