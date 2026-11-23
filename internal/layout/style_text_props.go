package layout

import (
	"slices"
	"strconv"
	"strings"
)

// applyTextPropsWave3 handles CSS text and text-decoration properties.
func applyTextPropsWave3(
	style *ResolvedStyle, prop, value string, fsize float64, _ *ResolvedStyle, _ bool,
) bool {
	switch prop {
	case "text-align-last":
		setTextAlignLast(style, value)
	case "text-align-all":
		setTextAlignValue(style, value)
		setTextAlignLast(style, value)
	case tabSizeProperty:
		setTabSize(style, value, fsize)
	case propTextWrap:
		setTextWrap(style, value)
	case "text-wrap-mode":
		setTextWrapMode(style, value)
	case propTextWrapStyle:
		style.editInlineText().TextWrapStyle = strings.ToLower(strings.TrimSpace(value))
	case "white-space-collapse":
		setWhiteSpaceCollapse(style, value)
	case "white-space-trim":
		style.editInlineText().WhiteSpaceTrim = strings.ToLower(strings.TrimSpace(value))
	case propHyphens:
		setHyphens(style, value)
	case "hyphenate-character":
		style.editInlineText().HyphenateCharacter = strings.Trim(strings.TrimSpace(value), `"'`)
	case "text-justify":
		style.editInlineText().TextJustify = strings.ToLower(strings.TrimSpace(value))
	case "line-break":
		style.editInlineText().LineBreak = strings.ToLower(strings.TrimSpace(value))

	case "text-decoration-line":
		setTextDecorationLine(style, value)
	case propTextDecorationColor:
		if c, ok := parseUsedColor(value, style.Color); ok {
			style.editInlineText().TextDecorationColor = c
			style.editInlineText().TextDecorationColorSet = true
		}
	case "text-decoration-style":
		setTextDecorationStyle(style, value)
	case "text-decoration-thickness":
		setTextDecorationThickness(style, value, fsize)
	case "text-underline-offset":
		if off, ok := plainLength(value, fsize, 0); ok {
			style.editInlineText().TextUnderlineOffset = off
		}
	case "text-underline-position":
		style.editInlineText().TextUnderlinePosition = strings.ToLower(strings.TrimSpace(value))
	case textShadowProperty:
		applyTextShadow(style, value, fsize)
	case textEmphasisProperty:
		setTextEmphasis(style, value)
	case textEmphasisStyleProperty:
		setTextEmphasisStyle(style, value)
	case textEmphasisColorProperty:
		setTextEmphasisColor(style, value)
	case textEmphasisPositionProperty:
		setTextEmphasisPosition(style, value)
	case textEmphasisSkipProperty:
		ensureEmphasisMap(style)
		style.CustomProps["__emph_skip"] = strings.ToLower(strings.TrimSpace(value))

	default:
		return false
	}

	return true
}

// whiteSpaceTrimHas reports whether a white-space-trim value contains tok as
// a whole token (none | discard-before || discard-after || discard-inner).
func whiteSpaceTrimHas(trim, tok string) bool {
	return slices.Contains(strings.Fields(strings.ToLower(trim)), tok)
}

// textJustifyMode normalizes a text-justify value to its justification
// method (auto | none | inter-word | inter-character). The no-compress
// modifier only forbids shrinking, which this engine never does, so it is
// stripped. ruby has no dedicated handler and falls back to auto.
func textJustifyMode(style *ResolvedStyle) string {
	if style == nil {
		return "auto"
	}

	cleaned := strings.ReplaceAll(strings.ToLower(style.inlineText().TextJustify), "no-compress", " ")
	fields := strings.Fields(cleaned)

	for _, field := range fields {
		switch field {
		case "none":
			return "none"
		case "inter-character":
			return "inter-character"
		case "inter-word":
			return "inter-word"
		}
	}

	return "auto"
}

func ensureEmphasisMap(style *ResolvedStyle) {
	if style.CustomProps == nil {
		style.CustomProps = make(map[string]string)
	}
}

func setTextEmphasis(style *ResolvedStyle, value string) {
	ensureEmphasisMap(style)
	val := strings.TrimSpace(value)
	low := strings.ToLower(val)
	if low == "" || low == "none" {
		style.CustomProps["__emph_style"] = "none"
		return
	}
	parts := strings.Fields(val)
	for _, tok := range parts {
		if c, ok := parseUsedColor(tok, style.Color); ok {
			style.CustomProps["__emph_color"] = tok
			_ = c
			continue
		}
		tl := strings.ToLower(tok)
		if tl != "" {
			style.CustomProps["__emph_style"] = tl
		}
	}
	if style.CustomProps["__emph_style"] == "" {
		style.CustomProps["__emph_style"] = "filled"
	}
}

func setTextEmphasisStyle(style *ResolvedStyle, value string) {
	ensureEmphasisMap(style)
	style.CustomProps["__emph_style"] = strings.ToLower(strings.TrimSpace(value))
}

func setTextEmphasisColor(style *ResolvedStyle, value string) {
	ensureEmphasisMap(style)
	if c, ok := parseUsedColor(value, style.Color); ok {
		_ = c
		style.CustomProps["__emph_color"] = strings.TrimSpace(value)
	} else {
		style.CustomProps["__emph_color"] = strings.TrimSpace(value)
	}
}

func setTextEmphasisPosition(style *ResolvedStyle, value string) {
	ensureEmphasisMap(style)
	style.CustomProps["__emph_position"] = strings.ToLower(strings.TrimSpace(value))
}

func setTextAlignLast(style *ResolvedStyle, value string) {
	val := strings.ToLower(strings.TrimSpace(value))
	switch val {
	case "auto", floatLeft, floatRight, fxCenter, cssTextAlignJustify, "start", "end":
		style.editInlineText().TextAlignLast = val
	}
}

func setTabSize(style *ResolvedStyle, value string, fsize float64) {
	val := strings.TrimSpace(value)
	if n, err := strconv.ParseFloat(val, 64); err == nil && n >= 0 {
		style.editInlineText().TabSize = n
		if style.CustomProps != nil {
			delete(style.CustomProps, "__tab_size_is_length")
		}
		return
	}
	if length, ok := plainLength(val, fsize, 0); ok && length >= 0 {
		style.editInlineText().TabSize = length
		ensureEmphasisMap(style)
		style.CustomProps["__tab_size_is_length"] = "1"
	}
}

func setTextWrap(style *ResolvedStyle, value string) {
	style.editInlineText().TextWrap = strings.TrimSpace(value)
	for _, tok := range strings.Fields(value) {
		tok = strings.ToLower(tok)
		if tok == "wrap" || tok == "nowrap" {
			setTextWrapMode(style, tok)
		} else {
			style.editInlineText().TextWrapStyle = tok
		}
	}
}

func setTextWrapMode(style *ResolvedStyle, value string) {
	val := strings.ToLower(strings.TrimSpace(value))
	style.editInlineText().TextWrapMode = val
	if val == "nowrap" {
		style.WhiteSpace = cssWhiteSpaceNowrap
	} else if val == "wrap" && style.WhiteSpace == cssWhiteSpaceNowrap {
		style.WhiteSpace = contentNormal
	}
}

func setWhiteSpaceCollapse(style *ResolvedStyle, value string) {
	val := strings.ToLower(strings.TrimSpace(value))
	style.editInlineText().WhiteSpaceCollapse = val
	switch val {
	case "collapse":
		style.WhiteSpace = contentNormal
	case "preserve":
		style.WhiteSpace = cssWhiteSpacePre
	case "preserve-breaks":
		style.WhiteSpace = cssWhiteSpacePreLine
	case "preserve-spaces", "break-spaces":
		style.WhiteSpace = cssWhiteSpacePreWrap
	}
}

func setHyphens(style *ResolvedStyle, value string) {
	val := strings.ToLower(strings.TrimSpace(value))
	switch val {
	case "none", "manual", "auto":
		style.editInlineText().Hyphens = val
	}
}

func setTextDecorationLine(style *ResolvedStyle, value string) {
	val := strings.ToLower(strings.TrimSpace(value))
	style.editInlineText().TextDecorationLine = val
	switch {
	case strings.Contains(val, "underline"):
		style.editInlineText().TextDecoration = cssTextDecorationUnderline
	case strings.Contains(val, "line-through"):
		style.editInlineText().TextDecoration = cssTextDecorationLineThrough
	case val == "none":
		style.editInlineText().TextDecoration = cssDisplayNone
	}
}

func setTextDecorationStyle(style *ResolvedStyle, value string) {
	val := strings.ToLower(strings.TrimSpace(value))
	switch val {
	case solidKeyword, "double", "dotted", "dashed", "wavy":
		style.editInlineText().TextDecorationStyle = val
	}
}

func setTextDecorationThickness(style *ResolvedStyle, value string, fsize float64) {
	val := strings.TrimSpace(value)
	if val == "auto" || val == "from-font" {
		style.editInlineText().TextDecorationThickness = 0
		return
	}
	if th, ok := plainLength(val, fsize, 0); ok && th > 0 {
		style.editInlineText().TextDecorationThickness = th
	}
}

func applyTextShadow(style *ResolvedStyle, value string, fsize float64) {
	val := strings.TrimSpace(value)
	if val == "" || strings.EqualFold(val, cssDisplayNone) {
		inlineText := style.editInlineText()
		inlineText.TextShadowSet = false
		inlineText.TextShadowX, inlineText.TextShadowY, inlineText.TextShadowBlur = 0, 0, 0
		if style.CustomProps != nil {
			delete(style.CustomProps, "__text_shadow_extra")
			delete(style.CustomProps, "__text_shadow_raw")
		}
		return
	}

	shadows := splitCommaRespectParens(val)
	if len(shadows) == 0 {
		style.editInlineText().TextShadowSet = false
		return
	}
	// First shadow goes into the primary fields for backward compat.
	first := shadows[0]
	parts := strings.Fields(first)
	if len(parts) >= 2 {
		if x, ok := plainLength(parts[0], fsize, 0); ok {
			style.editInlineText().TextShadowX = x
		}
		if y, ok := plainLength(parts[1], fsize, 0); ok {
			style.editInlineText().TextShadowY = y
		}
		style.editInlineText().TextShadowSet = true
	}
	if len(parts) >= 3 {
		if b, ok := plainLength(parts[2], fsize, 0); ok && b >= 0 {
			style.editInlineText().TextShadowBlur = b
		} else {
			style.editInlineText().TextShadowBlur = 0
		}
	} else {
		style.editInlineText().TextShadowBlur = 0
	}
	if len(parts) >= 4 {
		if c, ok := parseUsedColor(strings.Join(parts[3:], " "), style.Color); ok {
			style.editInlineText().TextShadowColor = c
		} else {
			style.editInlineText().TextShadowColor = style.Color
		}
	} else {
		style.editInlineText().TextShadowColor = style.Color
	}
	// Store remaining shadows encoded for paint.
	if len(shadows) > 1 {
		ensureEmphasisMap(style)
		var extra []string
		for _, sh := range shadows[1:] {
			p := parseSingleShadow(sh, fsize, style.Color)
			extra = append(extra, shadowEncode(p))
		}
		style.CustomProps["__text_shadow_extra"] = strings.Join(extra, "|")
		style.CustomProps["__text_shadow_raw"] = val
	} else if style.CustomProps != nil {
		delete(style.CustomProps, "__text_shadow_extra")
		delete(style.CustomProps, "__text_shadow_raw")
	}
}

type shadowSpec struct {
	x, y, blur float64
	color      [3]float64
}

func parseSingleShadow(input string, fsize float64, fallback [3]float64) shadowSpec {
	var shadow shadowSpec
	shadow.color = fallback
	parts := strings.Fields(input)
	if len(parts) >= 1 {
		if x, ok := plainLength(parts[0], fsize, 0); ok {
			shadow.x = x
		}
	}
	if len(parts) >= 2 {
		if y, ok := plainLength(parts[1], fsize, 0); ok {
			shadow.y = y
		}
	}
	if len(parts) >= 3 {
		applyShadowBlurColor(&shadow, parts, fsize, fallback)
	}
	return shadow
}

func applyShadowBlurColor(shadow *shadowSpec, parts []string, fsize float64, fallback [3]float64) {
	blur, ok := plainLength(parts[2], fsize, 0)
	if !ok || blur < 0 {
		// Third token is actually color when blur absent.
		if c, ok := parseUsedColor(strings.Join(parts[2:], " "), fallback); ok {
			shadow.color = c
		}

		return
	}

	shadow.blur = blur

	if len(parts) < 4 {
		return
	}

	if c, ok := parseUsedColor(strings.Join(parts[3:], " "), fallback); ok {
		shadow.color = c
	}
}

func shadowEncode(shadow shadowSpec) string {
	return strings.Join([]string{
		strconv.FormatFloat(shadow.x, 'f', -1, 64),
		strconv.FormatFloat(shadow.y, 'f', -1, 64),
		strconv.FormatFloat(shadow.blur, 'f', -1, 64),
		strconv.FormatFloat(shadow.color[0], 'f', -1, 64),
		strconv.FormatFloat(shadow.color[1], 'f', -1, 64),
		strconv.FormatFloat(shadow.color[2], 'f', -1, 64),
	}, ",")
}

func shadowDecode(input string) (shadowSpec, bool) {
	empty := shadowSpec{x: 0, y: 0, blur: 0, color: [3]float64{}}
	parts := strings.Split(input, ",")
	if len(parts) != 6 {
		return empty, false
	}
	var shadow shadowSpec
	var err error
	if shadow.x, err = strconv.ParseFloat(parts[0], 64); err != nil {
		return empty, false
	}
	if shadow.y, err = strconv.ParseFloat(parts[1], 64); err != nil {
		return empty, false
	}
	if shadow.blur, err = strconv.ParseFloat(parts[2], 64); err != nil {
		return empty, false
	}
	if shadow.color[0], err = strconv.ParseFloat(parts[3], 64); err != nil {
		return empty, false
	}
	if shadow.color[1], err = strconv.ParseFloat(parts[4], 64); err != nil {
		return empty, false
	}
	if shadow.color[2], err = strconv.ParseFloat(parts[5], 64); err != nil {
		return empty, false
	}
	return shadow, true
}

func splitCommaRespectParens(input string) []string {
	var out []string
	depth := 0
	start := 0
	for idx, r := range input {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				part := strings.TrimSpace(input[start:idx])
				if part != "" {
					out = append(out, part)
				}
				start = idx + 1
			}
		}
	}
	part := strings.TrimSpace(input[start:])
	if part != "" {
		out = append(out, part)
	}
	return out
}
