package layout

import (
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// box-decoration-break for a wrapping inline on one canvas. Column and page
// fragments are not sliced here. slice (the initial value, including an
// empty field) keeps one background for the whole inline element. clone
// paints the background, padding, and border on each wrapped fragment.
// applyAdvancedProps rejects any other token and leaves the previous value.

const boxDecorationBreakClone = "clone"

// decorBreakState holds per-engine box-decoration-break state for active
// inline formatting contexts. Stored directly on engine so push/pop allocate
// nothing and need no global synchronization.
type decorBreakState struct {
	ids    map[*html.Node]int
	next   int
	openID int
	fill   int
	open   bool
	depth  int
}

func inlineDecorationClone(style *ResolvedStyle) bool {
	return style != nil && style.borders().BoxDecorationBreak == boxDecorationBreakClone
}

// pushInlineDecorationBreak opens one inline formatting context. A nested
// context (an inline-block measured while the outer run is still being
// collected) shares the id map so element ids are not reused.
func (e *engine) pushInlineDecorationBreak() {
	state := &e.decorBreak
	state.depth++
	state.open = false
}

// popInlineDecorationBreak closes the context opened by push. The outermost
// pop clears the id map while keeping its backing for later inline runs.
func (e *engine) popInlineDecorationBreak() {
	state := &e.decorBreak
	if state.depth <= 0 {
		return
	}

	state.depth--
	state.open = false

	if state.depth == 0 {
		if len(state.ids) > 0 {
			clear(state.ids)
		}

		state.next = 0
		state.openID = 0
		state.fill = 0
	}
}

// inlineDecorID returns a stable id for one inline element, or 0 when no
// formatting context is open. Callers then paint each fragment on its own.
func (e *engine) inlineDecorID(node *html.Node) int {
	if node == nil {
		return 0
	}

	state := &e.decorBreak
	if state.depth == 0 {
		return 0
	}

	if id, ok := state.ids[node]; ok {
		return id
	}

	state.next++
	if state.ids == nil {
		state.ids = map[*html.Node]int{}
	}

	state.ids[node] = state.next

	return state.next
}

// markInlineChrome turns on inline chrome for the items just collected from
// one text node and stamps them with their parent element's decoration id.
func (e *engine) markInlineChrome(node *html.Node, out *[]inlineItem, start int) {
	if out == nil || start >= len(*out) {
		return
	}

	var parent *html.Node
	if node != nil {
		parent = node.Parent
	}

	id := e.inlineDecorID(parent)

	for idx := start; idx < len(*out); idx++ {
		e.enableInlineChrome(&(*out)[idx])
		(*out)[idx].decorID = id
	}
}

// paintInlineDecorationBreak paints one inline fragment's box. clone repeats
// the full chrome. slice emits the background on the first fragment and
// grows that one fill across later fragments of the same element.
func (e *engine) paintInlineDecorationBreak(
	item *inlineItem, leftX, baseline, ascent, descent, contentWidth float64,
) {
	if item == nil || item.style == nil {
		return
	}

	if inlineDecorationClone(item.style) || item.decorID == 0 {
		e.paintInlineChrome(item, leftX, baseline, ascent, descent, contentWidth)

		return
	}

	boxX, boxY, boxW, boxH := e.inlineDecorationBox(item.style, leftX, baseline, ascent, descent, contentWidth)
	if boxW <= 0 || boxH <= 0 {
		return
	}

	if e.expandInlineDecorationFill(item.decorID, boxX, boxY, boxW, boxH) {
		e.paintInlineDecorationBorder(item.style, boxX, boxY, boxW, boxH)

		return
	}

	e.paintFirstInlineFragment(item, leftX, baseline, ascent, descent, contentWidth)
}

// paintFirstInlineFragment paints the first slice fragment of an element and
// records its background fill as the one later fragments grow.
func (e *engine) paintFirstInlineFragment(
	item *inlineItem, leftX, baseline, ascent, descent, contentWidth float64,
) {
	before := len(e.ops)
	e.paintInlineChrome(item, leftX, baseline, ascent, descent, contentWidth)

	if !e.noEmit && before < len(e.ops) && e.ops[before].Kind == OpFillRect {
		e.openInlineDecorationFill(item.decorID, before)
	}
}

// inlineDecorationBox is the border box paintInlineChrome uses for one fragment.
func (e *engine) inlineDecorationBox(
	style *ResolvedStyle, leftX, baseline, ascent, descent, contentWidth float64,
) (float64, float64, float64, float64) {
	top := e.inlineChromeTop(style)
	bottom := e.inlineChromeBottom(style)
	lineHeight := e.lineHeightOf(style) * e.scale
	extra := (lineHeight - ascent - descent) / two
	boxY := baseline - ascent - extra - top
	boxH := ascent + extra + top + descent + extra + bottom
	boxW := contentWidth + e.inlineChromeLeft(style) + e.inlineChromeRight(style)

	return leftX, boxY, boxW, boxH
}

// expandInlineDecorationFill grows the open slice background to cover box.
// It reports false when this fragment starts a different element.
func (e *engine) expandInlineDecorationFill(id int, boxX, boxY, boxW, boxH float64) bool {
	state := &e.decorBreak
	if !state.open || state.openID != id {
		return false
	}

	if state.fill < 0 || state.fill >= len(e.ops) || e.ops[state.fill].Kind != OpFillRect {
		return false
	}

	growDecorFill(&e.ops[state.fill], boxX, boxY, boxW, boxH)

	return true
}

func (e *engine) openInlineDecorationFill(elemID, fill int) {
	state := &e.decorBreak
	if state.depth == 0 {
		return
	}

	state.open = true
	state.openID = elemID
	state.fill = fill
}

func growDecorFill(paintOp *Op, boxX, boxY, boxW, boxH float64) {
	right := paintOp.X + paintOp.W
	bottom := paintOp.Y + paintOp.H

	if boxX < paintOp.X {
		paintOp.X = boxX
	}

	if boxY < paintOp.Y {
		paintOp.Y = boxY
	}

	if boxX+boxW > right {
		right = boxX + boxW
	}

	if boxY+boxH > bottom {
		bottom = boxY + boxH
	}

	paintOp.W = right - paintOp.X
	paintOp.H = bottom - paintOp.Y
}

// paintInlineDecorationBorder paints the border of one slice fragment.
// The background stays the single fill opened on the first fragment.
func (e *engine) paintInlineDecorationBorder(style *ResolvedStyle, boxX, boxY, boxW, boxH float64) {
	if style == nil || !inlineHasBorder(style) {
		return
	}

	radii, radiiY := usedBorderRadiiXY(style, boxW, boxH)
	if hasRoundedRadii(radii) && uniformRoundedBorder(style) {
		side := style.borders().BorderTop
		stroke := Op{
			Kind: OpStrokeRect, X: boxX, Y: boxY, W: boxW, H: boxH,
			R: side.Color[0], G: side.Color[1], B: side.Color[2],
			Width: e.scalePt(borderPaint(side)), Radius: uniformRadius(radii),
			RadiusTopLeft: radii[0], RadiusTopRight: radii[1],
			RadiusBottomRight: radii[2], RadiusBottomLeft: radii[3],
		}
		stampOneOpRadiiY(&stroke, radiiY)
		e.add(stroke)

		return
	}

	for _, paintOp := range e.borderOps(style, boxX, boxY, boxW, boxH) {
		e.add(paintOp)
	}
}
