package layout

import (
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// Root margin accounting. Chrome places the body's border edge at the
// collapsed top margin shared by body and its first in-flow block child, and
// lets the last child's collapsed bottom margin extend the html element. The
// engine keeps in-flow positions stable by including those margins inside the
// body's content flow, so the root boxes are adjusted after the body flow to
// report the browser's geometry. Children do not move.

// marginCollapsesThrough reports whether a box's margin can collapse through
// one edge: the edge has no padding or border, and the box is not a BFC root
// or a layout-containment box.
func (e *engine) marginCollapsesThrough(style ResolvedStyle, top bool) bool {
	if establishesBFC(style) || containsLayout(style) {
		return false
	}

	if top {
		return style.PaddingTop == 0 && style.BorderTop.Width == 0
	}

	return style.PaddingBottom == 0 && style.BorderBottom.Width == 0
}

// applyRootBoxMargins applies the root html/body margin-collapse reporting
// adjustments: a collapsed first-child top margin offsets the body box and a
// collapsed last-child bottom margin extends the html height (Chrome root
// geometry).
func (e *engine) applyRootBoxMargins(
	node *html.Node, boxNode *box, style ResolvedStyle, curY float64,
) float64 {
	switch node.Name {
	case htmlBodyName:
		return e.applyRootBodyMargins(boxNode, style, curY)
	case htmlRootName:
		return e.extendRootHTMLHeight(boxNode, curY)
	}

	return curY
}

// inFlowBlockChild classifies one child for first/last in-flow block lookup:
// a block child is returned, a skipped child (display:none, out-of-flow,
// float, non-element) yields (nil, true) to keep scanning, and inline content
// yields (nil, false) because it stops margin collapse.
func (e *engine) inFlowBlockChild(child *html.Node) (*html.Node, bool) {
	if child.Type == html.TextNode {
		if strings.TrimSpace(child.Text) != "" {
			return nil, false
		}

		return nil, true
	}

	if child.Type != html.ElementNode {
		return nil, true
	}

	childStyle := e.stylePtr(child)
	if childStyle.Display == cssDisplayNone || isOutOfFlowNode(child, childStyle) ||
		isFlowFloat(child, childStyle) {
		return nil, true
	}

	if e.isInlineChild(child) {
		return nil, false
	}

	return child, true
}

// firstInFlowBlockChild returns the first in-flow block-level element child of
// parent. It returns nil when the first in-flow content is inline, which
// prevents parent/first-child margin collapse.
func (e *engine) firstInFlowBlockChild(parent *html.Node) *html.Node {
	if parent == nil {
		return nil
	}

	for _, child := range parent.Children {
		block, keepScanning := e.inFlowBlockChild(child)
		if !keepScanning {
			return nil
		}

		if block != nil {
			return block
		}
	}

	return nil
}

// lastInFlowBlockChild returns the last in-flow block-level element child of
// parent, or nil when the last in-flow content is inline.
func (e *engine) lastInFlowBlockChild(parent *html.Node) *html.Node {
	if parent == nil {
		return nil
	}

	for idx := len(parent.Children) - 1; idx >= 0; idx-- {
		block, keepScanning := e.inFlowBlockChild(parent.Children[idx])
		if !keepScanning {
			return nil
		}

		if block != nil {
			return block
		}
	}

	return nil
}

// collapsedTopThrough returns the top margin that collapses through node's
// top edge: the max of its own top margin and, when the edge has no border or
// padding, its first in-flow block child's chain.
func (e *engine) collapsedTopThrough(node *html.Node, style ResolvedStyle) float64 {
	margin := e.scalePt(style.MarginTop)
	if !e.marginCollapsesThrough(style, true) {
		return margin
	}

	child := e.firstInFlowBlockChild(node)
	if child == nil {
		return margin
	}

	if childMargin := e.collapsedTopThrough(child, *e.stylePtr(child)); childMargin > margin {
		margin = childMargin
	}

	return margin
}

// collapsedBottomThrough returns the bottom margin that collapses through
// node's bottom edge: the max of its own bottom margin and, when the edge has
// no border or padding, its last in-flow block child's chain.
func (e *engine) collapsedBottomThrough(node *html.Node, style ResolvedStyle) float64 {
	margin := e.scalePt(style.MarginBottom)
	if !e.marginCollapsesThrough(style, false) {
		return margin
	}

	child := e.lastInFlowBlockChild(node)
	if child == nil {
		return margin
	}

	if childMargin := e.collapsedBottomThrough(child, *e.stylePtr(child)); childMargin > margin {
		margin = childMargin
	}

	return margin
}

// applyRootBodyMargins shifts the body box to the collapsed top margin that
// escapes it and removes that margin from its content height. In-flow child
// positions are unchanged. The shift is limited to bodies with no top margin
// of their own: with a nonzero body margin the engine stacks the two margins
// and moving the body without moving its content would break that alignment,
// so those documents keep their previous (pre-C4) root geometry.
func (e *engine) applyRootBodyMargins(boxNode *box, style ResolvedStyle, curY float64) float64 {
	if style.MarginTop != 0 || !e.marginCollapsesThrough(style, true) {
		return curY
	}

	child := e.firstInFlowBlockChild(boxNode.node)
	if child == nil {
		return curY
	}

	childTop := e.collapsedTopThrough(child, *e.stylePtr(child))
	if childTop <= 0 {
		return curY
	}

	curY -= childTop
	boxNode.y += childTop

	return curY
}

// extendRootHTMLHeight lets the collapsed bottom margin that escapes the body
// extend the html height, matching Chrome's root box.
func (e *engine) extendRootHTMLHeight(boxNode *box, curY float64) float64 {
	body := e.rootBodyBox(boxNode)
	if body == nil || body.style == nil || !e.marginCollapsesThrough(*body.style, false) {
		return curY
	}

	// Chrome's body bottom edge is the engine's body bottom minus the top
	// margin part Chrome reports outside the body. The escaped bottom margin
	// then extends the html height.
	topCorr := e.rootBodyTopCorrection(body)
	escaped := e.collapsedBottomThrough(body.node, *body.style)
	bodyBottom := body.y + body.height - topCorr - boxNode.y + escaped

	if adjusted := curY - topCorr; bodyBottom > adjusted {
		return bodyBottom
	}

	return curY - topCorr
}

// rootBodyTopCorrection returns the collapsed top margin that Chrome reports
// outside the body but the engine keeps inside the body's content height. It
// is zero when the body's own top margin is zero (applyRootBodyMargins
// already moved the box) or when no first-child margin collapses.
func (e *engine) rootBodyTopCorrection(body *box) float64 {
	if body == nil || body.style == nil || body.node == nil || body.style.MarginTop == 0 {
		return 0
	}

	child := e.firstInFlowBlockChild(body.node)
	if child == nil || !e.marginCollapsesThrough(*body.style, true) {
		return 0
	}

	childTop := e.collapsedTopThrough(child, *e.stylePtr(child))
	bodyMargin := e.scalePt(body.style.MarginTop)

	if bodyMargin < childTop {
		return bodyMargin
	}

	return childTop
}

// rootBodyBox returns the body element box that is a direct child of an html
// box, or nil when the tree has no such box.
func (e *engine) rootBodyBox(htmlBox *box) *box {
	if htmlBox == nil || htmlBox.node == nil || htmlBox.node.Name != htmlRootName {
		return nil
	}

	for _, child := range htmlBox.children {
		if child.node != nil && child.node.Name == htmlBodyName {
			return child
		}
	}

	return nil
}
