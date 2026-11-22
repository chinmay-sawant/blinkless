package layout

import "math"

// continue (CSS Overflow 4) says what happens to content that does not fit
// a fragment. normal keeps it. collapse hides the lines that sit past
// max-lines or past max-height. discard drops lines that sit past a definite
// height or max-height. -webkit-legacy acts like collapse only for a
// vertical -webkit-box. A token outside those four leaves the previous value.
// The property does not inherit.

const (
	continueProp         = "continue"
	continueNormal       = "normal"
	continueDiscard      = "discard"
	continueCollapse     = "collapse"
	continueWebkitLegacy = "-webkit-legacy"
)

// applyContinueProps owns continue. Registered in styleGroups
// (style_cascade.go). A bad token returns true without writing, so the
// previous value stays.
func applyContinueProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	if prop != continueProp {
		return false
	}

	switch normalized := normalizeCSSValue(value); normalized {
	case continueNormal, continueDiscard, continueCollapse, continueWebkitLegacy:
		style.editInlineText().Continue = normalized
		// A vertical -webkit-box forces single-line white space through its
		// display mapping. continue clamps lines, so wrapping must come back,
		// the same way line-clamp does it.
		if normalized == continueWebkitLegacy && style.WhiteSpace == cssWhiteSpaceNowrap {
			style.WhiteSpace = "normal"
		}
	}

	return true
}

// continueActive reports whether the box's continue value clamps lines.
func continueActive(style *ResolvedStyle) bool {
	if style == nil {
		return false
	}

	switch style.inlineText().Continue {
	case continueCollapse, continueDiscard:
		return true
	case continueWebkitLegacy:
		return style.flex().IsWebkitBox && style.flex().FlexDirection == fxCol
	default:
		return false
	}
}

// continueLineLimit returns how many lines the box keeps under continue, or
// 0 for no limit. collapse honors max-height; discard honors height first
// and then max-height. The limit is the whole number of lines that fit in the
// content height (at least one). A box with no definite height has no limit.
func (e *engine) continueLineLimit(boxNode *box) int {
	if boxNode == nil || boxNode.style == nil {
		return 0
	}

	if boxNode.style.inlineText().ContinueLines > 0 {
		return boxNode.style.inlineText().ContinueLines
	}

	return e.continueLimitFor(boxNode.style)
}

// continueEpsilon keeps 40/20 from landing on 1.9999999 after scaling.
const continueEpsilon = 1e-6

// continueHeightCap returns the content height the lines must fit, or -1
// when the style has no definite cap.
func continueHeightCap(style *ResolvedStyle) float64 {
	limit := -1.0
	if style.inlineText().Continue == continueDiscard && style.Height >= 0 && style.HeightPercent < 0 {
		limit = style.Height
	}

	if style.geometry().MaxHeight >= 0 && style.geometry().MaxHeightPercent < 0 && (limit < 0 || style.geometry().MaxHeight < limit) {
		limit = style.geometry().MaxHeight
	}

	if limit >= 0 && style.geometry().BoxSizing == "border-box" {
		limit -= style.PaddingTop + style.PaddingBottom
	}

	return limit
}

// continueLimitFor computes the line limit from one style's own height.
func (e *engine) continueLimitFor(style *ResolvedStyle) int {
	if !continueActive(style) {
		return 0
	}

	limit := continueHeightCap(style)
	if limit < 0 {
		return 0
	}

	lineH := e.lineHeightOf(style)
	if lineH <= 0 {
		return 0
	}

	n := int(math.Floor(limit/lineH + continueEpsilon))
	if n < 1 {
		n = 1
	}

	return n
}
