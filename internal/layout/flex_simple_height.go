package layout

import (
	"math"
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// simpleFlexBaseHeight runs the simple row height fast path and stores the
// result under the same column base-height key the full measure uses. It
// reports false when the test bypass is on or the shape is not simple, so the
// caller keeps the unchanged noEmit path.
func (e *engine) simpleFlexBaseHeight(
	key flexBaseKey, node *html.Node, style *ResolvedStyle, contentW float64,
) (float64, bool) {
	if simpleFlexHeightBypass.Load() {
		return 0, false
	}

	height, ok := e.simpleFlexRowHeight(node, style, contentW)
	if !ok {
		height, ok = e.simpleFlexStackHeight(node, style, contentW)
	}
	if !ok {
		return 0, false
	}

	if e.err == nil {
		if e.flexBaseH == nil {
			e.flexBaseH = make(map[flexBaseKey]float64)
		}

		e.flexBaseH[key] = height
	}

	simpleFlexHeightHits.Add(1)

	return height, true
}

// simpleFlexRowHeight computes the border-box height of a simple flex row
// without building the row subtree. It is the narrow invoice shape: a
// display:flex row, flex-direction:row, flex-wrap:nowrap container whose
// in-flow children each have a definite width and exactly one text run, and
// whose height is one line plus the child's vertical padding, border, and
// margins.
//
// Every condition that can add a line, wrap the text, change a used width,
// change the line metrics, or paint generated content returns false. The
// caller then runs the full noEmit build unchanged.
//
//nolint:cyclop,funlen // one flat guard walk over the row and every child shape
func (e *engine) simpleFlexRowHeight(
	node *html.Node, style *ResolvedStyle, contentW float64,
) (float64, bool) {
	if e == nil || e.err != nil || node == nil || node.Type != html.ElementNode || style == nil {
		return 0, false
	}

	if !e.simpleFlexRowStyleOK(style, contentW) {
		return 0, false
	}

	rowBoxW := resolveUsedWidth(*style, contentW, e)
	if rowBoxW <= 0 {
		return 0, false
	}

	_, rowContentW := e.contentBox(0, rowBoxW, style)
	if rowContentW <= 0 {
		return 0, false
	}

	if len(node.Children) == 0 {
		return 0, false
	}

	_, colGap := e.styleGaps(style)
	if colGap < 0 {
		colGap = 0
	}

	totalW := 0.0
	rowContentH := 0.0
	count := 0

	for _, child := range node.Children {
		if child.Type != html.ElementNode {
			return 0, false
		}

		outerH, outerW, ok := e.simpleFlexCellMetrics(child)
		if !ok {
			return 0, false
		}

		if count > 0 {
			totalW += colGap
		}

		totalW += outerW
		if totalW > rowContentW+layoutEpsilon {
			return 0, false
		}

		if outerH > rowContentH {
			rowContentH = outerH
		}

		count++
	}

	if count == 0 || rowContentH <= 0 {
		return 0, false
	}

	rowH := e.scalePt(style.PaddingTop) + e.scalePt(style.BorderTop.Width) +
		rowContentH +
		e.scalePt(style.PaddingBottom) + e.scalePt(style.BorderBottom.Width)

	if rowH <= 0 {
		return 0, false
	}

	return rowH, true
}

// simpleFlexStackHeight computes the height of a block that stacks simple flex
// rows. It is the invoice grid shape: a block container whose children are all
// one-line flex rows. The rows still build once for placement; only the
// measure build of the whole stack disappears.
func (e *engine) simpleFlexStackHeight(
	node *html.Node, style *ResolvedStyle, contentW float64,
) (float64, bool) {
	if e == nil || e.err != nil || node == nil || node.Type != html.ElementNode || style == nil {
		return 0, false
	}

	if style.Display != "" && style.Display != displayBlock {
		return 0, false
	}

	if style.Height >= 0 || style.HeightPercent >= 0 ||
		style.MinHeight > 0 || style.MinHeightPercent >= 0 ||
		style.MaxHeight >= 0 || style.MaxHeightPercent >= 0 {
		return 0, false
	}

	if style.Float != "" && style.Float != cssDisplayNone {
		return 0, false
	}

	if style.Position != "" && style.Position != positionStatic {
		return 0, false
	}

	if isVerticalWritingMode(style.WritingMode) || style.Direction == cssDirectionRTL {
		return 0, false
	}

	if style.AspectRatio > 0 || style.BorderImageSource != "" ||
		containsSize(*style) || isMulticol(*style) ||
		style.ContentVisibility == contentVisibilityHidden ||
		style.ContentVisibility == contentVisibilityAuto {
		return 0, false
	}

	if e.hasPseudoContent() || contentW < 0 {
		return 0, false
	}

	rowBoxW, _ := resolveBlockWidth(e, node, style, contentW)
	if rowBoxW <= 0 {
		return 0, false
	}

	_, rowContentW := e.contentBox(0, rowBoxW, style)
	if rowContentW <= 0 || len(node.Children) == 0 {
		return 0, false
	}

	contentH := 0.0
	count := 0

	for _, child := range node.Children {
		if child.Type != html.ElementNode {
			return 0, false
		}

		childStyle := e.stylePtr(child)
		if childStyle == nil || childStyle.Display == cssDisplayNone {
			continue
		}

		if childStyle.Position != "" && childStyle.Position != positionStatic {
			return 0, false
		}

		// Block margin collapsing would need the adjoining margins. Rows in
		// this shape carry no vertical margins, so refuse any that do.
		if childStyle.MarginTopAuto || childStyle.MarginBottomAuto ||
			e.scalePt(childStyle.MarginTop) != 0 || e.scalePt(childStyle.MarginBottom) != 0 {
			return 0, false
		}

		height, ok := e.simpleFlexRowHeight(child, childStyle, rowContentW)
		if !ok {
			return 0, false
		}

		contentH += height
		count++
	}

	if count == 0 || contentH <= 0 {
		return 0, false
	}

	boxH := e.scalePt(style.PaddingTop) + e.scalePt(style.BorderTop.Width) +
		contentH +
		e.scalePt(style.PaddingBottom) + e.scalePt(style.BorderBottom.Width)
	if boxH <= 0 {
		return 0, false
	}

	return boxH, true
}

// simpleFlexRowStyleOK lists the container conditions the simple row height
// depends on. A definite height or a height constraint is refused because the
// full path would override the one-line content height.
//
//nolint:cyclop // one flat list of independent fallback predicates
func (e *engine) simpleFlexRowStyleOK(style *ResolvedStyle, contentW float64) bool {
	if style.Display != displayFlex {
		return false
	}

	if style.FlexDirection != "" && style.FlexDirection != fxRow {
		return false
	}

	if style.FlexWrap != "" && style.FlexWrap != cssWhiteSpaceNowrap {
		return false
	}

	if isVerticalWritingMode(style.WritingMode) || style.Direction == cssDirectionRTL {
		return false
	}

	if style.Float != "" && style.Float != cssDisplayNone {
		return false
	}

	if style.Position != "" && style.Position != positionStatic {
		return false
	}

	if style.AspectRatio > 0 || style.IsWebkitBox || isIntrinsicWidth(style.Width) {
		return false
	}

	if isMulticol(*style) {
		return false
	}

	// Percent heights without a definite containing height fall through
	// flexSpecifiedBaseHeight and reach here. Refuse every definite or
	// constrained height: the full path applies them after the content flow.
	if style.Height >= 0 || style.HeightPercent >= 0 {
		return false
	}

	if style.MinHeight > 0 || style.MinHeightPercent >= 0 ||
		style.MaxHeight >= 0 || style.MaxHeightPercent >= 0 {
		return false
	}

	// Border-image replaces the border layout width on some sides. The simple
	// path reads the plain border widths, so refuse the image case.
	if style.BorderImageSource != "" {
		return false
	}

	if containsSize(*style) || style.ContentVisibility == contentVisibilityHidden {
		return false
	}

	if e.hasPseudoContent() {
		return false
	}

	return contentW >= 0 && !math.IsInf(contentW, 0) && !math.IsNaN(contentW)
}

// simpleFlexCellMetrics returns one child's outer height and outer width for
// the simple row. The child must be a plain block or inline box with one text
// child, a definite width, an auto height, and no property that can change the
// line or the used width.
//
//nolint:cyclop,funlen,gocognit,gocyclo // one flat list of independent fallback predicates
func (e *engine) simpleFlexCellMetrics(child *html.Node) (float64, float64, bool) {
	childStyle := e.stylePtr(child)
	if childStyle == nil || childStyle.Display == cssDisplayNone {
		return 0, 0, false
	}

	switch childStyle.Display {
	case cssDisplayInline, displayBlock:
	default:
		return 0, 0, false
	}

	if isSimpleFlexReplacedElement(child) {
		return 0, 0, false
	}

	if childStyle.Position != "" && childStyle.Position != positionStatic {
		return 0, 0, false
	}

	if childStyle.Float != "" && childStyle.Float != cssDisplayNone {
		return 0, 0, false
	}

	// A definite width is the whole point of the fast path. Percentage and
	// intrinsic widths resolve against a containing block the measure build
	// owns, and grow or a width clamp can change the used width afterwards.
	if childStyle.Width < 0 || childStyle.WidthPercent >= 0 {
		return 0, 0, false
	}

	if childStyle.FlexBasis >= 0 || childStyle.FlexBasisPercent >= 0 || childStyle.FlexGrow > 0 {
		return 0, 0, false
	}

	if childStyle.MinWidth > 0 || childStyle.MinWidthPercent >= 0 ||
		childStyle.MaxWidth >= 0 || childStyle.MaxWidthPercent >= 0 {
		return 0, 0, false
	}

	if childStyle.MarginTopAuto || childStyle.MarginRightAuto ||
		childStyle.MarginBottomAuto || childStyle.MarginLeftAuto {
		return 0, 0, false
	}

	// The height comes from the text line only. A definite or constrained
	// height would replace that line-derived size in the full build.
	if childStyle.Height >= 0 || childStyle.HeightPercent >= 0 {
		return 0, 0, false
	}

	if childStyle.MinHeight > 0 || childStyle.MinHeightPercent >= 0 ||
		childStyle.MaxHeight >= 0 || childStyle.MaxHeightPercent >= 0 {
		return 0, 0, false
	}

	if childStyle.AspectRatio > 0 || childStyle.BorderImageSource != "" {
		return 0, 0, false
	}

	if containsSize(*childStyle) || childStyle.ContentVisibility == contentVisibilityHidden {
		return 0, 0, false
	}

	if isVerticalWritingMode(childStyle.WritingMode) || childStyle.Direction == cssDirectionRTL {
		return 0, 0, false
	}

	// Text-box trim drops half-leading on the first and last line. That
	// changes the line height even when the text fits.
	if childStyle.TextBoxTrim != "" && childStyle.TextBoxTrim != cssDisplayNone {
		return 0, 0, false
	}

	if len(child.Children) != 1 || child.Children[0].Type != html.TextNode {
		return 0, 0, false
	}

	textNode := child.Children[0]
	textStyle := e.stylePtr(textNode)

	if !singleTextRunStyleOK(childStyle, textStyle) {
		return 0, 0, false
	}

	text := simpleFlexText(textNode.Text, textStyle)
	if text == "" {
		return 0, 0, false
	}

	textW := e.measureTextFace(transformInlineText(text, textStyle.TextTransform), textStyle)

	padH := e.scalePt(childStyle.PaddingLeft) + e.scalePt(childStyle.PaddingRight) +
		e.scalePt(childStyle.BorderLeft.Width) + e.scalePt(childStyle.BorderRight.Width)
	childW := e.flexBoxSized(childStyle, e.scalePt(childStyle.Width), padH)
	innerW := childW - padH

	if innerW <= 0 || textW > innerW+layoutEpsilon {
		return 0, 0, false
	}

	lineH, ok := e.simpleFlexLineHeight(childStyle, textStyle, text)
	if !ok {
		return 0, 0, false
	}

	padV := e.scalePt(childStyle.PaddingTop) + e.scalePt(childStyle.PaddingBottom) +
		e.scalePt(childStyle.BorderTop.Width) + e.scalePt(childStyle.BorderBottom.Width)
	outerH := padV + lineH +
		e.scalePt(childStyle.MarginTop) + e.scalePt(childStyle.MarginBottom)
	outerW := childW + e.scalePt(childStyle.MarginLeft) + e.scalePt(childStyle.MarginRight)

	return outerH, outerW, true
}

// simpleFlexText returns the collapsed text the full inline collector would
// measure for one child. Unlike the single-text-run fast path, the simple row
// height accepts several words as long as the caller proves the whole run fits
// the child content width, so only breaks that could change the emitted text
// are refused here.
func simpleFlexText(raw string, style *ResolvedStyle) string {
	if raw == "" || style == nil {
		return ""
	}

	if strings.ContainsAny(raw, "\n\r\t") || strings.ContainsRune(raw, softHyphenRune) {
		return ""
	}

	text := strings.TrimLeft(collapseWS(raw), " ")
	if text == "" {
		return ""
	}

	return text
}

// simpleFlexLineHeight mirrors lineMetrics for one text run with no text-box
// trim: the line is the larger of the block strut and the run metrics on each
// side of the baseline. The run metrics include the half-leading from the
// run's own line-height, so a child with a larger font or line-height than the
// row still matches the full build.
func (e *engine) simpleFlexLineHeight(
	blockStyle, textStyle *ResolvedStyle, text string,
) (float64, bool) {
	lineHeight := e.lineHeightOf(textStyle) * e.scale
	if lineHeight <= 0 {
		return 0, false
	}

	strutAscent, strutDescent := e.strutMetrics(blockStyle, false, false)

	ascent, descent := e.inlineFontMetrics(text, textStyle)
	extra := (lineHeight - ascent - descent) / inlineHalfDivisor
	itemAscent, itemDescent := e.adjustTextBoxMetrics(
		textStyle, e.faceFor(textStyle), textStyle.FontSize*e.scale,
		ascent, descent, extra, false, false,
	)

	lineH := max(strutAscent, itemAscent) + max(strutDescent, itemDescent)
	if lineH <= 0 {
		return 0, false
	}

	return lineH, true
}

// isSimpleFlexReplacedElement reports elements whose box is replaced or atomic
// and therefore never a plain text run.
func isSimpleFlexReplacedElement(node *html.Node) bool {
	switch node.Name {
	case cssTagImg, cssTagSVG, "video", "canvas", "iframe", "embed", "object",
		htmlInput, "select", "textarea", "button", htmlMeter, "progress", "hr", "br":
		return true
	default:
		return false
	}
}

// SetSimpleFlexHeightBypassForTest turns the simple flex row height fast path
// off or on. The public layout benchmark uses it to measure one fixture with
// and without the fast path. Production code never calls it.
func SetSimpleFlexHeightBypassForTest(bypass bool) {
	simpleFlexHeightBypass.Store(bypass)
}
