package layout

import (
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// layoutSingleTextRun is the conservative one-line fast path for an inline
// formatting context whose content is exactly one text run. It measures the
// text once, builds the same inlineItem the generic collector would build,
// and hands that item to emitLine unchanged. Every condition that can add an
// item, split a line, narrow the line, or reorder the run falls back to
// layoutInlineFloatsGeneric.
//
// It reports false whenever the generic path might produce different output.
func (e *engine) layoutSingleTextRun(
	boxNode *box, nodes []*html.Node, contentW, contentX, lineY float64,
	floats *floatState,
) (float64, bool) {
	// A live float narrows or reorders the line, so the generic per-line
	// exclusion path stays in charge.
	if floats != nil && (floats.hasLeft || floats.hasRight) {
		return 0, false
	}

	item, ok := e.singleTextRunItem(boxNode, nodes)
	if !ok {
		return 0, false
	}

	// The generic packer keeps one line only when the whole margin box fits.
	// Otherwise it may soft-hyphen split, break the token, or push the line
	// below a float, any of which can change the item sequence.
	if item.marginL+item.w+item.marginR > contentW+inlineFitEpsilon {
		return 0, false
	}

	line := []inlineItem{item}

	return e.emitLine(boxNode, line, 0, 1, contentW, contentX, lineY, true, true), true
}

// singleTextRunItem resolves the one text run a block's inline content must
// consist of, or reports false. The returned item is built by the same
// helpers the collector uses, so its text, style, width, height, chrome,
// margins, and href match the generic items field for field.
func (e *engine) singleTextRunItem(boxNode *box, nodes []*html.Node) (inlineItem, bool) {
	textNode, element, textStyle, ok := e.singleTextRunCandidate(boxNode, nodes)
	if !ok {
		return inlineItem{}, false //nolint:exhaustruct // rejected candidate
	}

	text := singleTextRunText(textNode.Text, textStyle)
	if text == "" {
		return inlineItem{}, false //nolint:exhaustruct // rejected candidate
	}

	ctx := e.resolveInlineTextCtx(textNode)
	if ctx.vertical || ctx.atomic {
		return inlineItem{}, false //nolint:exhaustruct // rejected candidate
	}

	item := e.textItem(text, textStyle)
	if ctx.chromeApplies {
		e.enableInlineChrome(&item)
	}

	e.applySingleTextRunElement(&item, element)

	return item, true
}

// singleTextRunCandidate resolves the text node, optional inline element, and
// text style of a one-run block, or reports false when any fallback condition
// applies.
func (e *engine) singleTextRunCandidate(
	boxNode *box, nodes []*html.Node,
) (*html.Node, *html.Node, *ResolvedStyle, bool) {
	if boxNode == nil || boxNode.style == nil || len(nodes) != 1 {
		return nil, nil, nil, false
	}

	textNode, element, ok := singleTextRunNodes(nodes[0])
	if !ok {
		return nil, nil, nil, false
	}

	// Only display:inline elements reach collectInlineSpanPtr. inline-block,
	// inline-flex, and inline-grid build an atomic item instead.
	if element != nil && e.stylePtr(element).Display != cssDisplayInline {
		return nil, nil, nil, false
	}

	textStyle := e.stylePtr(textNode)
	if !singleTextRunStyleOK(boxNode.style, textStyle) {
		return nil, nil, nil, false
	}

	// Pseudo content adds items on either the block or the inline element.
	// No content rule anywhere proves neither probe can produce one.
	if e.hasPseudoContent() {
		return nil, nil, nil, false
	}

	return textNode, element, textStyle, true
}

// applySingleTextRunElement moves the inline element's horizontal margins onto
// the item and copies an anchor href, matching collectInlineSpanPtr.
func (e *engine) applySingleTextRunElement(item *inlineItem, element *html.Node) {
	if element == nil {
		return
	}

	elementStyle := e.stylePtr(element)

	// collectInlineSpanPtr moves the element's horizontal margins onto its
	// first and last generated item.
	item.marginL += e.scalePt(elementStyle.MarginLeft)
	item.marginR += e.scalePt(elementStyle.MarginRight)

	if element.Name != cssTagA {
		return
	}

	href := strings.TrimSpace(element.Attribute("href"))
	if isLinkHref(href) {
		item.href = href
	}
}

// singleTextRunNodes unwraps the sole node of an inline run to its text node.
// A single element is accepted only when it is a plain inline span whose only
// child is text. Replaced, atomic, and block-like elements are rejected.
func singleTextRunNodes(node *html.Node) (*html.Node, *html.Node, bool) {
	if node == nil {
		return nil, nil, false
	}

	switch node.Type {
	case html.TextNode:
		return node, nil, true
	case html.ElementNode:
		switch node.Name {
		case cssTagBR, cssTagImg, cssTagSVG, rubyElementName:
			return nil, nil, false
		}

		if len(node.Children) != 1 || node.Children[0].Type != html.TextNode {
			return nil, nil, false
		}

		return node.Children[0], node, true
	case html.CommentNode, html.DoctypeNode, html.NodeUnknown:
		return nil, nil, false
	default:
		return nil, nil, false
	}
}

// singleTextRunStyleOK rejects every block or text style whose inline layout
// can differ from a single measured run: indentation and clamps, balancing
// and pretty wrapping, floats/shapes, vertical writing, RTL/bidi scopes,
// initial letters, wrap-break hooks, non-default white space, hidden paint,
// ellipsis, shadows, emphasis, decorations, and vertical alignment.
func singleTextRunStyleOK(blockStyle, textStyle *ResolvedStyle) bool {
	if blockStyle == nil || textStyle == nil || textStyle.Display == cssDisplayNone {
		return false
	}

	return singleTextRunBlockGeometryOK(blockStyle, textStyle) &&
		singleTextRunWrapOK(blockStyle, textStyle) &&
		singleTextRunWritingOK(blockStyle, textStyle) &&
		singleTextRunPaintOK(textStyle) &&
		singleTextRunAlignmentOK(textStyle)
}

// singleTextRunBlockGeometryOK rejects indentation, clamps, and shape or
// initial-letter declarations that narrow or reshape the measured run.
func singleTextRunBlockGeometryOK(blockStyle, textStyle *ResolvedStyle) bool {
	if blockStyle.TextIndent != 0 || blockStyle.LineClamp > 0 || blockStyle.MaxLines > 0 {
		return false
	}

	if blockStyle.InitialLetterSize >= 1 || textStyle.InitialLetterSize >= 1 {
		return false
	}

	return singleTextRunShapeOK(blockStyle)
}

// singleTextRunShapeOK rejects a shape that narrows or reshapes the line even
// on an otherwise simple run.
func singleTextRunShapeOK(blockStyle *ResolvedStyle) bool {
	source := blockStyle.ShapeInside
	if blockStyle.ShapeInsideOutside {
		source = blockStyle.ShapeOutside
	}

	return source == "" || source == shapeOutsideNone || source == shapeInsideAuto
}

// singleTextRunWrapOK rejects wrapping, white-space, and break-hook
// declarations that make the collector split or rewrite the run.
func singleTextRunWrapOK(blockStyle, textStyle *ResolvedStyle) bool {
	wrapStyle := blockStyle.TextWrapStyle
	if wrapStyle != "" && wrapStyle != textWrapStyleAuto && wrapStyle != textWrapStyleStable {
		return false
	}

	if blockStyle.WhiteSpaceTrim != "" {
		return false
	}

	if hasWrapBreakPropsPtr(blockStyle) || hasWrapBreakPropsPtr(textStyle) {
		return false
	}

	return textStyle.WhiteSpace == "" || textStyle.WhiteSpace == contentNormal
}

// singleTextRunWritingOK rejects vertical writing, RTL, and scoped bidi.
func singleTextRunWritingOK(blockStyle, textStyle *ResolvedStyle) bool {
	if isVerticalWritingMode(blockStyle.WritingMode) || isVerticalWritingMode(textStyle.WritingMode) {
		return false
	}

	if blockStyle.Direction == cssDirectionRTL || textStyle.Direction == cssDirectionRTL {
		return false
	}

	return !isScopedBidi(blockStyle.UnicodeBidi) && !isScopedBidi(textStyle.UnicodeBidi)
}

// singleTextRunPaintOK rejects hidden paint, ellipsis, shadows, emphasis, and
// decorations that add or hide operations around the run.
func singleTextRunPaintOK(textStyle *ResolvedStyle) bool {
	if hidesPaint(textStyle) {
		return false
	}

	if textStyle.TextOverflow == textOverflowEllipsis || textStyle.TextShadowSet {
		return false
	}

	if hasEmphasisProps(textStyle) ||
		hasUnderline(textStyle) || hasLineThrough(textStyle) || hasOverline(textStyle) {
		return false
	}

	return true
}

// singleTextRunAlignmentOK rejects a non-baseline vertical alignment.
func singleTextRunAlignmentOK(textStyle *ResolvedStyle) bool {
	return textStyle.VerticalAlignShift == 0 &&
		(textStyle.VerticalAlign == "" || textStyle.VerticalAlign == baselineName)
}

// singleTextRunText returns the one word the collector would emit for raw, or
// "" when the node does not collect to exactly one item. Normal white-space
// collapses the text and splits on ASCII spaces, so any internal separator
// means several word items. A soft hyphen is a break opportunity that can
// change the emitted text even when the item fits.
func singleTextRunText(raw string, style *ResolvedStyle) string {
	if raw == "" || style == nil {
		return ""
	}

	text := strings.TrimLeft(collapseWS(raw), " ")
	if text == "" || strings.IndexByte(text, ' ') >= 0 || strings.ContainsRune(text, softHyphenRune) {
		return ""
	}

	return text
}
