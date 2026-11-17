package layout

import (
	"strings"
)

// normalizeSvgDeclared checks and normalizes the value of one SVG leftover:
// `text-rendering`, `lighting-color`, `glyph-orientation-vertical`,
// `baseline-shift`, `marker-side`. text-rendering and glyph-orientation-vertical
// inherit, so their initial keyword stays visible in the stored text.
func normalizeSvgDeclared(prop, value string) (string, bool) {
	text := strings.TrimSpace(value)
	if text == "" {
		return "", false
	}

	switch prop {
	case declPropTextRendering:
		return svgBakeTextRendering(text)
	case declPropLightingColor:
		return svgLightingColor(text)
	case declPropGlyphOrientationVertical:
		return svgGlyphOrientation(text)
	case declPropBaselineShift:
		return svgBaselineShift(text)
	case declPropMarkerSide:
		return svgMarkerSide(text)
	}

	return "", false
}

// svgLightingColor accepts any color the engine can read.
func svgLightingColor(text string) (string, bool) {
	if _, ok := parseUsedColor(text, [3]float64{}); ok {
		return strings.ToLower(text), true
	}

	return "", false
}

// svgGlyphOrientation accepts auto, 0deg or 90deg (a bare 0 or 90 too).
func svgGlyphOrientation(text string) (string, bool) {
	switch strings.ToLower(text) {
	case "auto":
		return "auto", true
	case "0", "0deg":
		return "0deg", true
	case "90", "90deg":
		return "90deg", true
	}

	return "", false
}

// svgBaselineShiftWord reports whether word is a keyword of baseline-shift.
func svgBaselineShiftWord(word string) bool {
	switch word {
	case verticalAlignSub, verticalAlignSuper, "top", "center", "bottom":
		return true
	}

	return false
}

// svgBaselineShift accepts a keyword, a bare 0 or a length-percentage.
func svgBaselineShift(text string) (string, bool) {
	word := strings.ToLower(text)
	if svgBaselineShiftWord(word) {
		return word, true
	}

	number, unit, ok := shapesNumberUnit(word)
	if !ok || strings.ContainsAny(word, " \t") {
		return "", false
	}

	if unit == "" && number == 0 {
		return "0", true
	}

	if unit != "" && shapesLengthUnit(unit) {
		return word, true
	}

	return "", false
}

// svgMarkerSide accepts match-self or match-parent.
func svgMarkerSide(text string) (string, bool) {
	switch strings.ToLower(text) {
	case "match-self":
		return "match-self", true
	case "match-parent":
		return "match-parent", true
	}

	return "", false
}
