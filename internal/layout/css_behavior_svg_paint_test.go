package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// SVG and paint-adjacent CSS behavior tests. Each test covers one property
// and asserts the emitted display-list ops or their numeric used values,
// never stored strings. Reference browser for every case: Chrome
// 143.0.7499.40. Shared helpers boxByID, near, pxToPt, opsOfKind,
// layoutHTML, layoutHTMLWithImages, and tinyPNG come from layout_test.go and
// css_review_02_test.go and are never redeclared here.

// behaviorSvgAssertSameOps pins no-op behavior: two results paint the same
// ops in the same order with the same kind, geometry, and text.
func behaviorSvgAssertSameOps(t *testing.T, first, second *Result) {
	t.Helper()

	if len(first.Ops) != len(second.Ops) {
		t.Fatalf("op count = %d vs %d, want identical geometry", len(first.Ops), len(second.Ops))
	}

	for i := range first.Ops {
		x, y := first.Ops[i], second.Ops[i]
		if x.Kind != y.Kind || x.Text != y.Text ||
			!near(x.X, y.X) || !near(x.Y, y.Y) ||
			!near(x.W, y.W) || !near(x.H, y.H) {
			t.Fatalf("op %d differs: %+v vs %+v, want identical geometry", i, x, y)
		}
	}
}

// behaviorSvgLayoutSVG lays out one inline SVG holding a single rect with the
// given rect style. The SVG path (layout_svg.go) bakes resolved
// fill/stroke presentation props into attributes before rasterization.
func behaviorSvgLayoutSVG(t *testing.T, rectStyle string) *Result {
	t.Helper()

	return layoutHTML(t, `<html><body style="margin:0">`+
		`<svg width="64" height="28" viewBox="0 0 64 28" xmlns="http://www.w3.org/2000/svg">`+
		`<rect x="2" y="2" width="60" height="24" style="`+rectStyle+`"/>`+
		`</svg></body></html>`)
}

// behaviorSvgSerializeRect returns the serialized SVG XML for one rect style,
// which is the payload the rasterizer consumes.
func behaviorSvgSerializeRect(t *testing.T, rectStyle string) string {
	t.Helper()

	root := mustParse(t, `<html><body>`+
		`<svg width="64" height="28" viewBox="0 0 64 28" xmlns="http://www.w3.org/2000/svg">`+
		`<rect x="2" y="2" width="60" height="24" style="`+rectStyle+`"/>`+
		`</svg></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `body{margin:0}`)}, "print", 500, 800)
	eng := &engine{styles: styles, scale: 1}

	var svgNode *html.Node

	root.Walk(func(node *html.Node) {
		if svgNode == nil && node.Type == html.ElementNode && node.Name == "svg" {
			svgNode = node
		}
	})

	if svgNode == nil {
		t.Fatal("svg missing")
	}

	return string(eng.serializeInlineSVG(svgNode))
}

// behaviorSvgRectUsedStyle returns the resolved used style of the rect node
// for numeric assertions on color channels, widths, and opacities.
func behaviorSvgRectUsedStyle(t *testing.T, rectStyle string) ResolvedStyle {
	t.Helper()

	root := mustParse(t, `<html><body>`+
		`<svg width="64" height="28" viewBox="0 0 64 28" xmlns="http://www.w3.org/2000/svg">`+
		`<rect x="2" y="2" width="60" height="24" style="`+rectStyle+`"/>`+
		`</svg></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `body{margin:0}`)}, "print", 500, 800)

	var rectNode *html.Node

	root.Walk(func(node *html.Node) {
		if rectNode == nil && node.Type == html.ElementNode && node.Name == "rect" {
			rectNode = node
		}
	})

	if rectNode == nil {
		t.Fatal("rect missing")
	}

	sty, ok := styles[rectNode]
	if !ok || sty == nil {
		t.Fatal("no resolved style for rect")
	}

	return *sty
}

// behaviorSvgIsolationGroups counts blend-group boundary markers whose group
// is isolated. Used to observe isolation:isolate.
func behaviorSvgIsolationGroups(res *Result) (int, int) {
	begins, ends := 0, 0

	for idx := range res.Ops {
		if res.Ops[idx].IsGroupBegin() {
			if group := res.Ops[idx].Group(); group != nil && group.Isolate {
				begins++
			}
		}

		if res.Ops[idx].IsGroupEnd() {
			if group := res.Ops[idx].Group(); group != nil && group.Isolate {
				ends++
			}
		}
	}

	return begins, ends
}

// TestBehaviorStrokeBakesPaint is stroke: the rect used color is red and the
// serialized SVG carries a red stroke attribute for the rasterizer, and the
// SVG emits an image op. Reference: Chrome 143.0.7499.40 strokes the rect
// outline in red.
func TestBehaviorStrokeBakesPaint(t *testing.T) {
	t.Parallel()

	sty := behaviorSvgRectUsedStyle(t, "fill:none;stroke:#ff0000")
	if !sty.StrokeSet {
		t.Fatal("stroke:#ff0000: StrokeSet = false, want true")
	}

	if !near(sty.Stroke[0], 1) || !near(sty.Stroke[1], 0) || !near(sty.Stroke[2], 0) {
		t.Errorf("stroke color = (%.3f, %.3f, %.3f), want (1, 0, 0)",
			sty.Stroke[0], sty.Stroke[1], sty.Stroke[2])
	}

	serialized := behaviorSvgSerializeRect(t, "fill:none;stroke:#ff0000")
	if !strings.Contains(serialized, `stroke="#ff0000"`) {
		t.Errorf("serialized SVG lacks red stroke, got %s", serialized)
	}

	images := opsOfKind(behaviorSvgLayoutSVG(t, "fill:none;stroke:#ff0000"), OpImage)
	if len(images) == 0 {
		t.Error("inline svg with stroke produced no OpImage")
	}
}

// TestBehaviorStrokeOpacityBakesPaint is stroke-opacity: the rect used opacity
// is 0.3 and the serialized SVG carries the attr for the rasterizer.
// Reference: Chrome 143.0.7499.40 fades the outline at 30 percent.
func TestBehaviorStrokeOpacityBakesPaint(t *testing.T) {
	t.Parallel()

	sty := behaviorSvgRectUsedStyle(t, "fill:none;stroke:#ff0000;stroke-opacity:0.3")
	if !near(sty.StrokeOpacity, 0.3) {
		t.Errorf("stroke-opacity used = %.4f, want 0.3", sty.StrokeOpacity)
	}

	serialized := behaviorSvgSerializeRect(t, "fill:none;stroke:#ff0000;stroke-opacity:0.3")
	if !strings.Contains(serialized, `stroke-opacity="0.3"`) {
		t.Errorf("serialized SVG lacks stroke-opacity 0.3, got %s", serialized)
	}

	images := opsOfKind(behaviorSvgLayoutSVG(t, "fill:none;stroke:#ff0000;stroke-opacity:0.3"), OpImage)
	if len(images) == 0 {
		t.Error("inline svg with stroke-opacity produced no OpImage")
	}
}

// TestBehaviorStrokeWidthBakesPaint is stroke-width: the rect used width is
// 3pt and the serialized SVG carries the attr for the rasterizer. Reference:
// Chrome 143.0.7499.40 widens the outline to the declared width.
func TestBehaviorStrokeWidthBakesPaint(t *testing.T) {
	t.Parallel()

	sty := behaviorSvgRectUsedStyle(t, "fill:none;stroke:#000000;stroke-width:3pt")
	if !sty.StrokeWidthSet {
		t.Fatal("stroke-width:3pt: StrokeWidthSet = false, want true")
	}

	if !near(sty.StrokeWidth, 3) {
		t.Errorf("stroke-width used = %.4fpt, want 3pt", sty.StrokeWidth)
	}

	serialized := behaviorSvgSerializeRect(t, "fill:none;stroke:#000000;stroke-width:3pt")
	if !strings.Contains(serialized, `stroke-width="3"`) {
		t.Errorf("serialized SVG lacks stroke-width 3, got %s", serialized)
	}

	images := opsOfKind(behaviorSvgLayoutSVG(t, "fill:none;stroke:#000000;stroke-width:3pt"), OpImage)
	if len(images) == 0 {
		t.Error("inline svg with stroke-width produced no OpImage")
	}
}

// TestBehaviorFillBakesPaint is fill: the rect used color is red and the
// serialized SVG carries a red fill attribute for the rasterizer, and the SVG
// emits an image op. Reference: Chrome 143.0.7499.40 fills the rect in red.
func TestBehaviorFillBakesPaint(t *testing.T) {
	t.Parallel()

	sty := behaviorSvgRectUsedStyle(t, "fill:#ff0000")
	if !sty.FillSet {
		t.Fatal("fill:#ff0000: FillSet = false, want true")
	}

	if !near(sty.Fill[0], 1) || !near(sty.Fill[1], 0) || !near(sty.Fill[2], 0) {
		t.Errorf("fill color = (%.3f, %.3f, %.3f), want (1, 0, 0)",
			sty.Fill[0], sty.Fill[1], sty.Fill[2])
	}

	serialized := behaviorSvgSerializeRect(t, "fill:#ff0000")
	if !strings.Contains(serialized, `fill="#ff0000"`) {
		t.Errorf("serialized SVG lacks red fill, got %s", serialized)
	}

	images := opsOfKind(behaviorSvgLayoutSVG(t, "fill:#ff0000"), OpImage)
	if len(images) == 0 {
		t.Error("inline svg with fill produced no OpImage")
	}
}

// TestBehaviorFillOpacityBakesPaint is fill-opacity: the rect used opacity is
// 0.4 and the serialized SVG carries the attr for the rasterizer. Reference:
// Chrome 143.0.7499.40 fades the fill at 40 percent.
func TestBehaviorFillOpacityBakesPaint(t *testing.T) {
	t.Parallel()

	sty := behaviorSvgRectUsedStyle(t, "fill:#00ff00;fill-opacity:0.4")
	if !near(sty.FillOpacity, 0.4) {
		t.Errorf("fill-opacity used = %.4f, want 0.4", sty.FillOpacity)
	}

	serialized := behaviorSvgSerializeRect(t, "fill:#00ff00;fill-opacity:0.4")
	if !strings.Contains(serialized, `fill-opacity="0.4"`) {
		t.Errorf("serialized SVG lacks fill-opacity 0.4, got %s", serialized)
	}

	images := opsOfKind(behaviorSvgLayoutSVG(t, "fill:#00ff00;fill-opacity:0.4"), OpImage)
	if len(images) == 0 {
		t.Error("inline svg with fill-opacity produced no OpImage")
	}
}

// TestBehaviorFillRuleNoPaintEffect documents the current behavior of
// fill-rule: the declaration is dropped and no paint pass reads it, so the
// styled SVG paints identical geometry to the default. GAP: style_paint_props
// forwards fill-rule to applyLeftoversProps which has no fill-rule case and
// returns false, and the rasterizer never sees the rule. Unobservable in the
// drawing list: reported as a no-op, never failing. Reference: Chrome
// 143.0.7499.40 would apply evenodd winding to holed paths.
func TestBehaviorFillRuleNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := behaviorSvgLayoutSVG(t, "fill:#ff0000")
	ruled := behaviorSvgLayoutSVG(t, "fill:#ff0000;fill-rule:evenodd")

	behaviorSvgAssertSameOps(t, plain, ruled)
}

// TestBehaviorColorInterpolationNoPaintEffect documents the current behavior
// of color-interpolation: the declaration is dropped and no paint pass reads
// it. GAP: style_paint_props forwards it to applyLeftoversProps which has no
// case and returns false. Unobservable in the drawing list: reported as a
// no-op, never failing. Reference: Chrome 143.0.7499.40 would interpolate
// gradients in sRGB instead.
func TestBehaviorColorInterpolationNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := behaviorSvgLayoutSVG(t, "fill:#ff0000")
	tuned := behaviorSvgLayoutSVG(t, "fill:#ff0000;color-interpolation:sRGB")

	behaviorSvgAssertSameOps(t, plain, tuned)
}

// TestBehaviorColorInterpolationFiltersNoPaintEffect documents the current
// behavior of color-interpolation-filters: the declaration is dropped and no
// paint pass reads it. GAP: same forwarding path as color-interpolation with
// no case in applyLeftoversProps. Unobservable in the drawing list: reported
// as a no-op, never failing. Reference: Chrome 143.0.7499.40 would change the
// filter color space.
func TestBehaviorColorInterpolationFiltersNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := behaviorSvgLayoutSVG(t, "fill:#ff0000")
	tuned := behaviorSvgLayoutSVG(t, "fill:#ff0000;color-interpolation-filters:sRGB")

	behaviorSvgAssertSameOps(t, plain, tuned)
}

// TestBehaviorIsolationOpensGroup is isolation: isolate opens an isolated
// blend group around the element while auto opens none. Reference: Chrome
// 143.0.7499.40 isolates the element backdrop per CSS Compositing.
func TestBehaviorIsolationOpensGroup(t *testing.T) {
	t.Parallel()

	page := func(mode string) *Result {
		return layoutHTML(t, `<html style="margin:0"><body style="margin:0;background:white">`+
			`<div id="box" style="isolation:`+mode+`;background:red;`+
			`width:40pt;height:20pt"><p id="in" style="margin:0">x</p></div></body></html>`)
	}

	begins, ends := behaviorSvgIsolationGroups(page("isolate"))
	if begins == 0 || ends == 0 {
		t.Fatalf("isolation:isolate: begins=%d ends=%d, want both >0", begins, ends)
	}

	plainBegins, plainEnds := behaviorSvgIsolationGroups(page("auto"))
	if plainBegins != 0 || plainEnds != 0 {
		t.Errorf("isolation:auto: begins=%d ends=%d, want 0/0", plainBegins, plainEnds)
	}
}

// TestBehaviorShapeMarginExpandsExclusion is shape-margin: with the same
// circle() contour a larger margin widens the exclusion interval, so text
// wraps further from the float. Reference: Chrome 143.0.7499.40 grows the
// shape contour by the margin.
func TestBehaviorShapeMarginExpandsExclusion(t *testing.T) {
	t.Parallel()

	doc := func(margin string) *Result {
		return layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
			`<div id="fl" style="float:left;width:100px;height:100px;shape-outside:circle();shape-margin:`+margin+`"></div>`+
			`<p id="p" style="margin:0;font-size:12pt">wrap text here</p></body></html>`)
	}

	small := boxByID(t, doc("0px"), "fl")
	large := boxByID(t, doc("10px"), "fl")

	if !near(small.style.ShapeMargin, 0) {
		t.Errorf("shape-margin:0px used = %.4fpt, want 0", small.style.ShapeMargin)
	}

	if !near(large.style.ShapeMargin, pxToPt(10)) {
		t.Errorf("shape-margin:10px used = %.4fpt (%.2fpx), want 10px",
			large.style.ShapeMargin, large.style.ShapeMargin/ptPerCSSPx)
	}

	smallExcl := buildShapeExclusion(*small.style, small, 0, 0, 1)
	largeExcl := buildShapeExclusion(*large.style, large, 0, 0, 1)

	if smallExcl == nil || largeExcl == nil {
		t.Fatalf("exclusions = %v / %v, want both non-nil for circle()", smallExcl != nil, largeExcl != nil)
	}

	midY := small.y + small.height/2
	smallLeft, smallRight, okSmall := smallExcl.intervalAt(midY)
	largeLeft, largeRight, okLarge := largeExcl.intervalAt(midY)

	if !okSmall || !okLarge {
		t.Fatalf("mid-line intervals ok = %v / %v, want true/true", okSmall, okLarge)
	}

	if !(largeRight-largeLeft > smallRight-smallLeft+0.5) {
		t.Errorf("margin widths = %.3fpt vs %.3fpt, want large wider by the margin",
			smallRight-smallLeft, largeRight-largeLeft)
	}
}

// TestBehaviorShapeOutsideNarrowsWrap is shape-outside: a circle() contour
// resolves to a shape exclusion while none keeps rectangular exclusion.
// Reference: Chrome 143.0.7499.40 wraps text around the circle instead of the
// margin box.
func TestBehaviorShapeOutsideNarrowsWrap(t *testing.T) {
	t.Parallel()

	doc := func(shape string) *Result {
		return layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
			`<div id="fl" style="float:left;width:100px;height:100px;shape-outside:`+shape+`"></div>`+
			`<p id="p" style="margin:0;font-size:12pt">wrap text here</p></body></html>`)
	}

	noneBox := boxByID(t, doc("none"), "fl")
	circleBox := boxByID(t, doc("circle()"), "fl")

	if excl := buildShapeExclusion(*noneBox.style, noneBox, 0, 0, 1); excl != nil {
		t.Errorf("shape-outside:none exclusion = %+v, want nil (rectangular wrap)", excl)
	}

	circleExcl := buildShapeExclusion(*circleBox.style, circleBox, 0, 0, 1)
	if circleExcl == nil {
		t.Fatal("shape-outside:circle() exclusion = nil, want a circle contour")
	}

	midY := circleBox.y + circleBox.height/2
	left, right, ok := circleExcl.intervalAt(midY)

	if !ok {
		t.Fatal("circle mid-line interval missing, want one crossing")
	}

	if !(right-left > 0 && right-left <= circleBox.w+0.01) {
		t.Errorf("circle width = %.4fpt for box w %.4fpt, want (0, w]", right-left, circleBox.w)
	}
}

// TestBehaviorShapeRenderingNoPaintEffect documents the current behavior of
// shape-rendering: the declaration is dropped and no paint pass reads it.
// GAP: style_paint_props forwards it to applyLeftoversProps which has no case
// and returns false, and the canvas rasterizer never sees the hint.
// Unobservable in the drawing list: reported as a no-op, never failing.
// Reference: Chrome 143.0.7499.40 would snap edges with crispEdges.
func TestBehaviorShapeRenderingNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := behaviorSvgLayoutSVG(t, "fill:#ff0000")
	crisp := behaviorSvgLayoutSVG(t, "fill:#ff0000;shape-rendering:crispEdges")

	behaviorSvgAssertSameOps(t, plain, crisp)
}

// TestBehaviorImageOrientationSwapsAxes is image-orientation: a 90deg turn
// swaps the intrinsic axes, so a 20x10px image paints 7.5pt wide and 15pt
// tall while none keeps 15pt by 7.5pt. Reference: Chrome 143.0.7499.40 swaps
// the axes for a quarter turn.
func TestBehaviorImageOrientationSwapsAxes(t *testing.T) {
	t.Parallel()

	png := tinyPNG(20, 20/2)

	doc := func(orient string) *Result {
		return layoutHTMLWithImages(t, `<html style="margin:0"><body style="margin:0">`+
			`<img id="pic" src="x.png" style="image-orientation:`+orient+`">`+
			`</body></html>`, png, "")
	}

	plainImgs := opsOfKind(doc("none"), OpImage)
	turnedImgs := opsOfKind(doc("90deg"), OpImage)

	if len(plainImgs) != 1 || len(turnedImgs) != 1 {
		t.Fatalf("img ops = %d / %d, want 1 each", len(plainImgs), len(turnedImgs))
	}

	if !near(plainImgs[0].W, pxToPt(20)) || !near(plainImgs[0].H, pxToPt(10)) {
		t.Errorf("none image = %.3fpt x %.3fpt, want 20px x 10px",
			plainImgs[0].W, plainImgs[0].H)
	}

	if !near(turnedImgs[0].W, pxToPt(10)) || !near(turnedImgs[0].H, pxToPt(20)) {
		t.Errorf("90deg image = %.3fpt x %.3fpt, want 10px x 20px (axes swapped)",
			turnedImgs[0].W, turnedImgs[0].H)
	}
}

// TestBehaviorImageResolutionScalesIntrinsic is image-resolution: an explicit
// 48dpi doubles the intrinsic size against the 96dpi default, so a 20x10px
// image paints 30pt by 15pt instead of 15pt by 7.5pt. Reference: Chrome
// 143.0.7499.40 scales the intrinsic pixels by 96/dpi.
func TestBehaviorImageResolutionScalesIntrinsic(t *testing.T) {
	t.Parallel()

	png := tinyPNG(20, 10)

	doc := func(res string) *Result {
		style := ""
		if res != "" {
			style = ` style="image-resolution:` + res + `"`
		}

		return layoutHTMLWithImages(t, `<html style="margin:0"><body style="margin:0">`+
			`<img id="pic" src="x.png"`+style+`>`+
			`</body></html>`, png, "")
	}

	defImgs := opsOfKind(doc(""), OpImage)
	halfImgs := opsOfKind(doc("48dpi"), OpImage)

	if len(defImgs) != 1 || len(halfImgs) != 1 {
		t.Fatalf("img ops = %d / %d, want 1 each", len(defImgs), len(halfImgs))
	}

	if !near(defImgs[0].W, pxToPt(20)) || !near(defImgs[0].H, pxToPt(10)) {
		t.Errorf("default image = %.3fpt x %.3fpt, want 20px x 10px",
			defImgs[0].W, defImgs[0].H)
	}

	if !near(halfImgs[0].W, pxToPt(40)) || !near(halfImgs[0].H, pxToPt(20)) {
		t.Errorf("48dpi image = %.3fpt x %.3fpt, want 40px x 20px (doubled)",
			halfImgs[0].W, halfImgs[0].H)
	}
}

// TestBehaviorBackfaceVisibilityNoPaintEffect documents the current behavior
// of backface-visibility: the declaration is dropped and no paint pass reads
// it, so hidden paints identical geometry to visible under the same
// rotation. GAP: applyTransformGroup forwards it to applyLeftoversProps which
// has no backface case and returns false. Unobservable in the drawing list:
// reported as a no-op, never failing. Reference: Chrome 143.0.7499.40 would
// hide the back face in 3D.
func TestBehaviorBackfaceVisibilityNoPaintEffect(t *testing.T) {
	t.Parallel()

	front := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="a" style="width:100px;height:50px;background-color:#ff0000;transform:rotate(10deg);`+
		`backface-visibility:visible"></div>`+
		`</body></html>`)
	hidden := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="a" style="width:100px;height:50px;background-color:#ff0000;transform:rotate(10deg);`+
		`backface-visibility:hidden"></div>`+
		`</body></html>`)

	behaviorSvgAssertSameOps(t, front, hidden)
}

// TestBehaviorAlignmentBaselineNoPaintEffect documents the current behavior of
// alignment-baseline: the declaration is dropped and no paint pass reads it,
// so the styled SVG paints identical geometry to the default. GAP:
// style_paint_props forwards it to applyLeftoversProps which has no case and
// returns false. Unobservable in the drawing list: reported as a no-op, never
// failing. Reference: Chrome 143.0.7499.40 would shift the text baseline.
func TestBehaviorAlignmentBaselineNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := behaviorSvgLayoutSVG(t, "fill:#ff0000")
	shifted := behaviorSvgLayoutSVG(t, "fill:#ff0000;alignment-baseline:middle")

	behaviorSvgAssertSameOps(t, plain, shifted)
}
