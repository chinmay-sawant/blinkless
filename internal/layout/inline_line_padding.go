// line-padding is a length added at both ends of every line. It is not
// text-indent, which moves only the first line. A token that is not a
// length leaves the stored value unchanged. Zero is the initial length.
package layout

import (
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// Values used by the line-padding length resolver.
const (
	linePaddingRemPx = 16 // px in one rem, for a length given in rem
	linePaddingSides = 2  // a line gets its padding at both ends
)

// applyLinePaddingProps stores line-padding. Registered on styleGroups.
// A bad token returns without writing, so the previous value stays.
func applyLinePaddingProps(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext,
	parent *ResolvedStyle, hasParent bool,
) bool {
	if prop != "line-padding" {
		return false
	}

	viewportW := 0.0
	if ctx != nil {
		viewportW = ctx.viewportW
	}

	pt, ok := linePaddingSpecified(value, fsize, viewportW, parent, hasParent)
	if ok {
		style.editInlineText().LinePadding = pt
	}

	return true
}

// linePaddingSpecified resolves one line-padding token to points.
// Unrecognized tokens report false so the caller keeps the previous value.
func linePaddingSpecified(
	value string, fsize, viewportW float64, parent *ResolvedStyle, hasParent bool,
) (float64, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case cssKeywordInitial, cssKeywordRevert, stringSetRevertLayer:
		return 0, true
	case inheritKeyword, cssKeywordUnset:
		if hasParent && parent != nil {
			return parent.inlineText().LinePadding, true
		}

		return 0, true
	default:
		return linePaddingLength(value, fsize, viewportW)
	}
}

// linePaddingLength parses a CSS length into points. Percentages are not
// part of the grammar, so they are rejected.
func linePaddingLength(value string, fsize, viewportW float64) (float64, bool) {
	if pt, ok := vminVmaxPt(value, viewportW, viewportW); ok {
		return pt, true
	}

	val, unit, ok := css.ParseLength(value)
	if !ok || unit == "%" {
		return 0, false
	}

	if unit == remUnit {
		return pxToPt(linePaddingRemPx) * val, true
	}

	return lengthToPt(val, unit, fsize)
}

// linePaddingLengthAccepted is the cascade gate. A rejected token never
// replaces an earlier length or the inherited value.
func linePaddingLengthAccepted(value string) bool {
	_, ok := linePaddingLength(value, 0, 0)

	return ok
}

// linePaddingInset shifts the start of one line by the block's line-padding
// and takes that length off both ends of the available width. Justification
// sees the shortened width, so the extra gap is not a justify gap. Zero
// leaves the line unchanged.
func (e *engine) linePaddingInset(boxNode *box, lineX, lineW float64) (float64, float64) {
	if boxNode == nil || boxNode.style == nil {
		return lineX, lineW
	}

	pad := e.scalePt(boxNode.style.inlineText().LinePadding)
	if pad == 0 {
		return lineX, lineW
	}

	lineX += pad
	lineW -= pad * linePaddingSides

	if lineW < 0 {
		lineW = 0
	}

	return lineX, lineW
}
