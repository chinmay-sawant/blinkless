package layout

import (
	"math"
	"strconv"
	"strings"
)

// applyZoomProps owns the zoom property. A number or percentage at or above
// zero is stored as a factor (0 and 0% become 1, per css-viewport). A negative
// number, a negative percentage, or any other token is ignored so an earlier
// valid factor stays in place. This does not write Options.Zoom.
func applyZoomProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext,
	parent *ResolvedStyle, _ bool,
) bool {
	if prop != "zoom" {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case inheritKeyword:
		if parent != nil {
			style.editGeometry().Zoom = elementZoom(parent)
		}

		return true
	case cssKeywordInitial, cssKeywordUnset, cssKeywordRevert, stringSetRevertLayer:
		style.editGeometry().Zoom = 1

		return true
	}

	factor, ok := parseZoomFactor(value)
	if !ok {
		return true
	}

	style.editGeometry().Zoom = factor

	return true
}

// zoomDeclarationAccepted reports whether value is a legal zoom. Every other
// property keeps the default-accept rule of the cascade gate.
func zoomDeclarationAccepted(prop, value string) bool {
	if prop != "zoom" {
		return true
	}

	_, ok := parseZoomFactor(value)

	return ok
}

// parseZoomFactor accepts one non-negative number or percentage.
// 150% is 1.5. 0 and 0% are 1. Anything else is rejected.
func parseZoomFactor(raw string) (float64, bool) {
	fields := strings.Fields(raw)
	if len(fields) != 1 {
		return 0, false
	}

	token := fields[0]
	if before, percent := strings.CutSuffix(token, "%"); percent {
		return zoomNumber(before, true)
	}

	return zoomNumber(token, false)
}

func zoomNumber(raw string, percentage bool) (float64, bool) {
	number, err := strconv.ParseFloat(raw, 64)
	if err != nil || number < 0 || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, false
	}

	if number == 0 {
		return 1, true
	}

	if percentage {
		return number / oneHundred, true
	}

	return number, true
}

// elementZoom returns the element's own factor. The initial value and a zero
// style are both 1. This is not the document zoom in layout_faces.go.
func elementZoom(style *ResolvedStyle) float64 {
	if style == nil || style.geometry().Zoom <= 0 || math.IsNaN(style.geometry().Zoom) || math.IsInf(style.geometry().Zoom, 0) {
		return 1
	}

	return style.geometry().Zoom
}

// applyElementZoomBorderBox multiplies a block's used border box by the
// element's zoom factor. Call it once width and height are both final.
// An axis that came from auto or a percentage is left alone. The document
// scale (engine.scale) is not changed.
func applyElementZoomBorderBox(boxNode *box, style *ResolvedStyle) {
	factor := elementZoom(style)
	if boxNode == nil || style == nil || factor == 1 {
		return
	}

	if zoomScalesAxis(style.Width, style.WidthPercent) {
		boxNode.w *= factor
	}

	if zoomScalesAxis(style.Height, style.HeightPercent) {
		boxNode.height *= factor
	}
}

// zoomScalesAxis reports whether a used size came from a length. auto,
// a percentage, and an intrinsic keyword are not multiplied.
func zoomScalesAxis(length, percent float64) bool {
	if percent >= 0 || length < 0 || isIntrinsicWidth(length) {
		return false
	}

	return true
}
