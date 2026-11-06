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
	if boxNode == nil || boxNode.style == nil || len(nodes) != 1 {
		return inlineItem{}, false //nolint:exhaustruct // rejected candidate
	}

	textNode, element, ok := singleTextRunNodes(nodes[0])
	if !ok {
		return inlineItem{}, false //nolint:exhaustruct // rejected candidate
	}

	// Only display:inline elements reach collectInlineSpanPtr. inline-block,
	// inline-flex, and inline-grid build an atomic item instead.
	if element != nil && e.stylePtr(element).Display != cssDisplayInline {
		return inlineItem{}, false //nolint:exhaustruct // rejected candidate
	}

	textStyle := e.stylePtr(textNode)
	if !singleTextRunStyleOK(boxNode.style, textStyle) {
		return inlineItem{}, false //nolint:exhaustruct // rejected candidate
	}

	// Pseudo content adds items on either the block or the inline element.
	// No content rule anywhere proves neither probe can produce one.
	if e.hasPseudoContent() {
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

	if element != nil {
		elementStyle := e.stylePtr(element)

		// collectInlineSpanPtr moves the element's horizontal margins onto
		// its first and last generated item.
		item.marginL += e.scalePt(elementStyle.MarginLeft)
		item.marginR += e.scalePt(elementStyle.MarginRight)

		if element.Name == cssTagA {
			href := strings.TrimSpace(element.Attribute("href"))
			if isLinkHref(href) {
				item.href = href
			}
		}
	}

	return item, true
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
//
//nolint:cyclop,lll // one flat list of independent fallback predicates
func singleTextRunStyleOK(blockStyle, textStyle *ResolvedStyle) bool {
	if blockStyle == nil || textStyle == nil || textStyle.Display == cssDisplayNone {
		return false
	}

	if blockStyle.TextIndent != 0 || blockStyle.LineClamp > 0 || blockStyle.MaxLines > 0 {
		return false
	}

	switch blockStyle.TextWrapStyle {
	case "", textWrapStyleAuto, textWrapStyleStable:
	default:
		return false
	}

	if blockStyle.WhiteSpaceTrim != "" {
		return false
	}

	// shape-inside narrows the line even on an otherwise simple run.
	source := blockStyle.ShapeInside
	if blockStyle.ShapeInsideOutside {
		source = blockStyle.ShapeOutside
	}

	if source != "" && source != shapeOutsideNone && source != shapeInsideAuto {
		return false
	}

	if blockStyle.InitialLetterSize >= 1 || textStyle.InitialLetterSize >= 1 {
		return false
	}

	if isVerticalWritingMode(blockStyle.WritingMode) || isVerticalWritingMode(textStyle.WritingMode) {
		return false
	}

	if blockStyle.Direction == cssDirectionRTL || textStyle.Direction == cssDirectionRTL {
		return false
	}

	if isScopedBidi(blockStyle.UnicodeBidi) || isScopedBidi(textStyle.UnicodeBidi) {
		return false
	}

	if hasWrapBreakPropsPtr(blockStyle) || hasWrapBreakPropsPtr(textStyle) {
		return false
	}

	if textStyle.WhiteSpace != "" && textStyle.WhiteSpace != contentNormal {
		return false
	}

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

	if textStyle.VerticalAlignShift != 0 ||
		(textStyle.VerticalAlign != "" && textStyle.VerticalAlign != "baseline") {
		return false
	}

	return true
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
