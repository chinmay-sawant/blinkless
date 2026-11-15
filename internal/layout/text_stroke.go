package layout

import "strings"

// -webkit-text-stroke draws an outline of the given width around each glyph.
// -webkit-text-stroke-width is a <line-width> (thin, medium, thick, or a
// length of at least 0) and starts at 0, which paints no stroke.
// -webkit-text-stroke-color is a <color> and starts at currentcolor, the
// fill color of the run. The shorthand takes a width and a color in either
// order and resets the one it leaves out. All three inherit.
//
// The drawing list carries the stroke on the OpText extra (Op.TextStroke).
// A host paints the outline; the engine does not rasterize glyph edges.

const (
	textStrokeProp      = "-webkit-text-stroke"
	textStrokeWidthProp = "-webkit-text-stroke-width"
	textStrokeColorProp = "-webkit-text-stroke-color"
)

// applyTextStrokeProps owns the three text-stroke properties. Registered in
// styleGroups (style_cascade.go). A value that fails its grammar leaves the
// previous value.
func applyTextStrokeProps(
	style *ResolvedStyle, prop, value string, fsize float64, _ *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	switch prop {
	case textStrokeWidthProp:
		if width, ok := parseTextStrokeWidth(value, fsize); ok {
			style.TextStrokeWidth = width
		}
	case textStrokeColorProp:
		setTextStrokeColor(style, value)
	case textStrokeProp:
		applyTextStrokeShorthand(style, value, fsize)
	default:
		return false
	}

	return true
}

// parseTextStrokeWidth reads a <line-width>. A negative length is invalid.
func parseTextStrokeWidth(value string, fsize float64) (float64, bool) {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "-") {
		return 0, false
	}

	return parseOutlineWidth(trimmed, fsize)
}

// setTextStrokeColor stores a color. currentcolor clears the stored color so
// the stroke follows the fill.
func setTextStrokeColor(style *ResolvedStyle, value string) {
	if isCurrentColor(strings.TrimSpace(value)) {
		style.TextStrokeColorSet = false

		return
	}

	if color, ok := parseUsedColor(value, style.Color); ok {
		style.TextStrokeColor = color
		style.TextStrokeColorSet = true
	}
}

// applyTextStrokeShorthand sets both longhands from "<width> || <color>". A
// value with a token that is neither, or with a repeated kind, is refused.
func applyTextStrokeShorthand(style *ResolvedStyle, value string, fsize float64) {
	width, haveWidth := 0.0, false
	color, haveColor := "", false

	for _, token := range splitOutsideParens(value) {
		if parsed, ok := parseTextStrokeWidth(token, fsize); ok && !haveWidth {
			width, haveWidth = parsed, true

			continue
		}

		if _, ok := parseUsedColor(token, style.Color); ok && !haveColor {
			color, haveColor = token, true

			continue
		}

		return
	}

	if !haveWidth && !haveColor {
		return
	}

	style.TextStrokeWidth = width
	style.TextStrokeColorSet = false

	if haveColor {
		setTextStrokeColor(style, color)
	}
}

// splitOutsideParens splits on whitespace that is not inside parentheses.
func splitOutsideParens(value string) []string {
	var parts []string

	depth, start := 0, -1

	for pos, char := range value {
		switch {
		case char == '(':
			depth++
		case char == ')' && depth > 0:
			depth--
		}

		isSpace := (char == ' ' || char == '\t' || char == '\n') && depth == 0

		switch {
		case isSpace && start >= 0:
			parts = append(parts, value[start:pos])
			start = -1
		case !isSpace && start < 0:
			start = pos
		}
	}

	if start >= 0 {
		parts = append(parts, value[start:])
	}

	return parts
}

// inheritTextStroke copies the parent's stroke unless the element declares
// any of the three properties.
func inheritTextStroke(dst, parent *ResolvedStyle, raw map[string]string) {
	for _, name := range [3]string{textStrokeProp, textStrokeWidthProp, textStrokeColorProp} {
		if _, ok := raw[name]; ok {
			return
		}
	}

	dst.TextStrokeWidth = parent.TextStrokeWidth
	dst.TextStrokeColor = parent.TextStrokeColor
	dst.TextStrokeColorSet = parent.TextStrokeColorSet
}

// withTextStroke records the stroke on a text op. A run with no stroke width
// pays nothing. An unset color follows the run's fill.
func (op Op) withTextStroke(style *ResolvedStyle) Op {
	if style == nil || style.TextStrokeWidth <= 0 {
		return op
	}

	extra := op.detachedExtraCopy()
	extra.TextStrokeWidth = style.TextStrokeWidth
	extra.TextStrokeColor = [3]float64{op.R, op.G, op.B}

	if style.TextStrokeColorSet {
		extra.TextStrokeColor = style.TextStrokeColor
	}

	op.opExtra = extra

	return op
}

// TextStroke returns the glyph outline of a text op: its width in points and
// its color. ok is false when the run has no stroke.
func (op Op) TextStroke() (width float64, color [3]float64, ok bool) {
	if op.opExtra == nil || op.opExtra.TextStrokeWidth <= 0 {
		return 0, [3]float64{}, false
	}

	return op.opExtra.TextStrokeWidth, op.opExtra.TextStrokeColor, true
}
