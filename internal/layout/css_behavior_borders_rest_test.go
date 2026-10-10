package layout

import "testing"

// Rest border behavior tests: per-side colors and styles, physical and
// logical corner radii, and the border-image shorthand. Every property below
// is asserted through an emitted op or a used value (a measured border-box
// edge or an op field), never through a stored style string. Lengths are
// written in CSS px and converted with pxToPt (1px = 0.75pt, ptPerCSSPx in
// css_review_02_test.go), which is how Chrome reports them. Helpers boxByID,
// near, pxToPt, layoutHTML, layoutHTMLWithImages, tinyPNG, opsOfKind, and
// redFillOp come from this same package; behaviorBordersHorizontalLines,
// behaviorBordersVerticalLines, behaviorBgImages, and behaviorBgCorner come
// from css_behavior_borders_test.go and css_behavior_backgrounds_test.go.
// Reference browser for every case: Chrome 143.0.7499.40.

// behaviorBordRestRadiusFill lays out one 120px by 60px red box with the
// given radius declaration and returns its background fill op.
func behaviorBordRestRadiusFill(t *testing.T, radiusDecl string) Op {
	t.Helper()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:120px;height:60px;background-color:#f00;`+radiusDecl+`"></div>`+
		`</body></html>`)

	return redFillOp(t, res.Ops)
}

// behaviorBordRestAssertRadiusCorners asserts the four fill corner radii in
// CSS px order top-left, top-right, bottom-right, bottom-left.
func behaviorBordRestAssertRadiusCorners(
	t *testing.T, fill Op, wantTL, wantTR, wantBR, wantBL float64, tag string,
) {
	t.Helper()

	for name, pair := range map[string][2]float64{
		"RadiusTopLeft":     {fill.RadiusTopLeft, wantTL},
		"RadiusTopRight":    {fill.RadiusTopRight, wantTR},
		"RadiusBottomRight": {fill.RadiusBottomRight, wantBR},
		"RadiusBottomLeft":  {fill.RadiusBottomLeft, wantBL},
	} {
		if !near(pair[0], pxToPt(pair[1])) {
			t.Errorf("%s fill %s = %.4fpt (%.2fpx), want %.2fpx",
				tag, name, pair[0], pair[0]/ptPerCSSPx, pair[1])
		}
	}
}

// behaviorBordRestHorizontalCountAt counts horizontal border line ops at y.
func behaviorBordRestHorizontalCountAt(ops []Op, y float64) int {
	count := 0

	for _, paintOp := range behaviorBordersHorizontalLines(ops) {
		if near(paintOp.Y, y) {
			count++
		}
	}

	return count
}

// behaviorBordRestVerticalCountAt counts vertical border line ops at x.
func behaviorBordRestVerticalCountAt(ops []Op, x float64) int {
	count := 0

	for _, paintOp := range behaviorBordersVerticalLines(ops) {
		if near(paintOp.X, x) {
			count++
		}
	}

	return count
}

// TestBehaviorBorderBottomColorPaintsEdge is border-bottom-color: the bottom
// edge stroke carries the declared color. Reference: Chrome 143.0.7499.40.
// behaviorBordRestCheckColor lays out one 200x40px box with style and asserts
// the border edge at the item coordinate carries the wanted color.
// selectLines picks the candidate line ops, edgeAt reads the item edge
// coordinate, lineAt reads the same coordinate off a line op.
func behaviorBordRestCheckColor(
	t *testing.T, style string, wantR, wantG, wantB float64,
	selectLines func([]Op) []Op,
	edgeAt func(*box) float64,
	lineAt func(Op) float64,
	edgeName string,
) {
	t.Helper()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:200px;height:40px;`+style+`"></div>`+
		`</body></html>`)

	item := boxByID(t, res, "item")

	found := false

	for _, paintOp := range selectLines(res.Ops) {
		if !near(lineAt(paintOp), edgeAt(item)) {
			continue
		}

		found = true

		if !near(paintOp.R, wantR) || !near(paintOp.G, wantG) || !near(paintOp.B, wantB) {
			t.Errorf("%s edge color = (%.2f, %.2f, %.2f), want (%.0f, %.0f, %.0f)",
				edgeName, paintOp.R, paintOp.G, paintOp.B, wantR, wantG, wantB)
		}
	}

	if !found {
		t.Fatalf("no %s border op at %.4fpt", edgeName, edgeAt(item))
	}
}

func TestBehaviorBorderBottomColorPaintsEdge(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name  string
		style string
		wantR float64
		wantG float64
		wantB float64
	}{
		{"red bottom edge", "border-bottom-width:4px;border-bottom-style:solid;border-bottom-color:red", 1, 0, 0},
		{"blue bottom edge", "border-bottom-width:4px;border-bottom-style:solid;border-bottom-color:blue", 0, 0, 1},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			behaviorBordRestCheckColor(t, testCase.style, testCase.wantR, testCase.wantG, testCase.wantB,
				behaviorBordersHorizontalLines,
				func(item *box) float64 { return item.y + item.height },
				func(lineOp Op) float64 { return lineOp.Y },
				"bottom")
		})
	}
}

// TestBehaviorBorderBottomStyleDashedExpandsSegments is border-bottom-style:
// solid paints one segment, dashed expands into several, none paints
// nothing. Reference: Chrome 143.0.7499.40, 200px bottom edge.
func TestBehaviorBorderBottomStyleDashedExpandsSegments(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="solid" style="width:200px;height:20px;border-bottom:3px solid #000"></div>`+
		`<div id="dashed" style="width:200px;height:20px;border-bottom:3px dashed #000"></div>`+
		`<div id="none" style="width:200px;height:20px;border-bottom:3px none #000"></div>`+
		`</body></html>`)

	solid := boxByID(t, res, "solid")
	dashed := boxByID(t, res, "dashed")
	none := boxByID(t, res, "none")

	if got := behaviorBordRestHorizontalCountAt(res.Ops, solid.y+solid.height); got != 1 {
		t.Errorf("solid bottom edge segments = %d, want 1", got)
	}

	if got := behaviorBordRestHorizontalCountAt(res.Ops, dashed.y+dashed.height); got <= 1 {
		t.Errorf("dashed bottom edge segments = %d, want more than 1", got)
	}

	if got := behaviorBordRestHorizontalCountAt(res.Ops, none.y+none.height); got != 0 {
		t.Errorf("none bottom edge segments = %d, want 0", got)
	}
}

// TestBehaviorBorderBottomRadiusRoundsBottomCorners is border-bottom-radius:
// both bottom corners of the background fill carry the radius while the top
// corners stay square. Reference: Chrome 143.0.7499.40, 120px by 60px box,
// 16px radius. Note: border-bottom-radius is not a standard CSS property, so
// Chrome drops the declaration; this pins the engine extension which rounds
// both bottom corners.
func TestBehaviorBorderBottomRadiusRoundsBottomCorners(t *testing.T) {
	t.Parallel()

	fill := behaviorBordRestRadiusFill(t, "border-bottom-radius:16px")
	behaviorBordRestAssertRadiusCorners(t, fill, 0, 0, 16, 16, "border-bottom-radius")
}

// TestBehaviorBorderBottomLeftRadiusPaintsCorner is
// border-bottom-left-radius: only the bottom-left corner of the background
// fill carries the radius. Reference: Chrome 143.0.7499.40, 120px by 60px
// box, 16px corner.
func TestBehaviorBorderBottomLeftRadiusPaintsCorner(t *testing.T) {
	t.Parallel()

	fill := behaviorBordRestRadiusFill(t, "border-bottom-left-radius:16px")
	behaviorBordRestAssertRadiusCorners(t, fill, 0, 0, 0, 16, "border-bottom-left-radius")
}

// TestBehaviorBorderBottomRightRadiusPaintsCorner is
// border-bottom-right-radius: only the bottom-right corner of the background
// fill carries the radius. Reference: Chrome 143.0.7499.40, 120px by 60px
// box, 16px corner.
func TestBehaviorBorderBottomRightRadiusPaintsCorner(t *testing.T) {
	t.Parallel()

	fill := behaviorBordRestRadiusFill(t, "border-bottom-right-radius:16px")
	behaviorBordRestAssertRadiusCorners(t, fill, 0, 0, 16, 0, "border-bottom-right-radius")
}

// TestBehaviorBorderTopColorPaintsEdge is border-top-color: the top edge
// stroke carries the declared color. Reference: Chrome 143.0.7499.40. This
// uses the per-side longhand where TestBehaviorBorderColorPaintsTopEdge used
// the same colors through its own declarations.
func TestBehaviorBorderTopColorPaintsEdge(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name  string
		style string
		wantR float64
		wantG float64
		wantB float64
	}{
		{"red top edge", "border-top-width:4px;border-top-style:solid;border-top-color:red", 1, 0, 0},
		{"blue top edge", "border-top-width:4px;border-top-style:solid;border-top-color:blue", 0, 0, 1},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			behaviorBordRestCheckColor(t, testCase.style, testCase.wantR, testCase.wantG, testCase.wantB,
				behaviorBordersHorizontalLines,
				func(item *box) float64 { return item.y },
				func(lineOp Op) float64 { return lineOp.Y },
				"top")
		})
	}
}

// TestBehaviorBorderTopRadiusRoundsTopCorners is border-top-radius: both top
// corners of the background fill carry the radius while the bottom corners
// stay square. Reference: Chrome 143.0.7499.40, 120px by 60px box, 16px
// radius. Note: border-top-radius is not a standard CSS property, so Chrome
// drops the declaration; this pins the engine extension which rounds both
// top corners.
func TestBehaviorBorderTopRadiusRoundsTopCorners(t *testing.T) {
	t.Parallel()

	fill := behaviorBordRestRadiusFill(t, "border-top-radius:16px")
	behaviorBordRestAssertRadiusCorners(t, fill, 16, 16, 0, 0, "border-top-radius")
}

// TestBehaviorBorderTopRightRadiusPaintsCorner is border-top-right-radius:
// only the top-right corner of the background fill carries the radius.
// Reference: Chrome 143.0.7499.40, 120px by 60px box, 16px corner.
func TestBehaviorBorderTopRightRadiusPaintsCorner(t *testing.T) {
	t.Parallel()

	fill := behaviorBordRestRadiusFill(t, "border-top-right-radius:16px")
	behaviorBordRestAssertRadiusCorners(t, fill, 0, 16, 0, 0, "border-top-right-radius")
}

// TestBehaviorBorderLeftColorPaintsEdge is border-left-color: the left edge
// stroke carries the declared color. Reference: Chrome 143.0.7499.40.
func TestBehaviorBorderLeftColorPaintsEdge(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name  string
		style string
		wantR float64
		wantG float64
		wantB float64
	}{
		{"red left edge", "border-left-width:4px;border-left-style:solid;border-left-color:red", 1, 0, 0},
		{"blue left edge", "border-left-width:4px;border-left-style:solid;border-left-color:blue", 0, 0, 1},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			behaviorBordRestCheckColor(t, testCase.style, testCase.wantR, testCase.wantG, testCase.wantB,
				behaviorBordersVerticalLines,
				func(item *box) float64 { return item.x },
				func(lineOp Op) float64 { return lineOp.X },
				"left")
		})
	}
}

// TestBehaviorBorderLeftStyleDashedExpandsSegments is border-left-style:
// solid paints one segment, dashed expands into several, none paints
// nothing. Reference: Chrome 143.0.7499.40, 40px left edge. Each style gets
// its own document so the stacked boxes share no edge coordinate.
func TestBehaviorBorderLeftStyleDashedExpandsSegments(t *testing.T) {
	t.Parallel()

	countFor := func(t *testing.T, borderStyle string) int {
		t.Helper()

		res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
			`<div id="item" style="width:200px;height:40px;border-left:3px `+borderStyle+` #000"></div>`+
			`</body></html>`)

		item := boxByID(t, res, "item")

		return behaviorBordRestVerticalCountAt(res.Ops, item.x)
	}

	if got := countFor(t, "solid"); got != 1 {
		t.Errorf("solid left edge segments = %d, want 1", got)
	}

	if got := countFor(t, "dashed"); got <= 1 {
		t.Errorf("dashed left edge segments = %d, want more than 1", got)
	}

	if got := countFor(t, "none"); got != 0 {
		t.Errorf("none left edge segments = %d, want 0", got)
	}
}

// TestBehaviorBorderLeftRadiusRoundsLeftCorners is border-left-radius: both
// left corners of the background fill carry the radius while the right
// corners stay square. Reference: Chrome 143.0.7499.40, 120px by 60px box,
// 16px radius. Note: border-left-radius is not a standard CSS property, so
// Chrome drops the declaration; this pins the engine extension which rounds
// both left corners.
func TestBehaviorBorderLeftRadiusRoundsLeftCorners(t *testing.T) {
	t.Parallel()

	fill := behaviorBordRestRadiusFill(t, "border-left-radius:16px")
	behaviorBordRestAssertRadiusCorners(t, fill, 16, 0, 0, 16, "border-left-radius")
}

// TestBehaviorBorderRightColorPaintsEdge is border-right-color: the right
// edge stroke carries the declared color. Reference: Chrome 143.0.7499.40.
func TestBehaviorBorderRightColorPaintsEdge(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name  string
		style string
		wantR float64
		wantG float64
		wantB float64
	}{
		{"red right edge", "border-right-width:4px;border-right-style:solid;border-right-color:red", 1, 0, 0},
		{"blue right edge", "border-right-width:4px;border-right-style:solid;border-right-color:blue", 0, 0, 1},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			behaviorBordRestCheckColor(t, testCase.style, testCase.wantR, testCase.wantG, testCase.wantB,
				behaviorBordersVerticalLines,
				func(item *box) float64 { return item.x + item.w },
				func(lineOp Op) float64 { return lineOp.X },
				"right")
		})
	}
}

// TestBehaviorBorderRightStyleDashedExpandsSegments is border-right-style:
// solid paints one segment, dashed expands into several, none paints
// nothing. Reference: Chrome 143.0.7499.40, 40px right edge. Each style gets
// its own document so the stacked boxes share no edge coordinate.
func TestBehaviorBorderRightStyleDashedExpandsSegments(t *testing.T) {
	t.Parallel()

	countFor := func(t *testing.T, borderStyle string) int {
		t.Helper()

		res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
			`<div id="item" style="width:200px;height:40px;border-right:3px `+borderStyle+` #000"></div>`+
			`</body></html>`)

		item := boxByID(t, res, "item")

		return behaviorBordRestVerticalCountAt(res.Ops, item.x+item.w)
	}

	if got := countFor(t, "solid"); got != 1 {
		t.Errorf("solid right edge segments = %d, want 1", got)
	}

	if got := countFor(t, "dashed"); got <= 1 {
		t.Errorf("dashed right edge segments = %d, want more than 1", got)
	}

	if got := countFor(t, "none"); got != 0 {
		t.Errorf("none right edge segments = %d, want 0", got)
	}
}

// TestBehaviorBorderRightRadiusRoundsRightCorners is border-right-radius:
// both right corners of the background fill carry the radius while the left
// corners stay square. Reference: Chrome 143.0.7499.40, 120px by 60px box,
// 16px radius. Note: border-right-radius is not a standard CSS property, so
// Chrome drops the declaration; this pins the engine extension which rounds
// both right corners.
func TestBehaviorBorderRightRadiusRoundsRightCorners(t *testing.T) {
	t.Parallel()

	fill := behaviorBordRestRadiusFill(t, "border-right-radius:16px")
	behaviorBordRestAssertRadiusCorners(t, fill, 0, 16, 16, 0, "border-right-radius")
}

// TestBehaviorBorderStartStartRadiusMapsTopLeft is border-start-start-radius:
// in the default left-to-right horizontal flow it rounds only the top-left
// corner of the background fill. Reference: Chrome 143.0.7499.40, 120px by
// 60px box, 16px corner.
func TestBehaviorBorderStartStartRadiusMapsTopLeft(t *testing.T) {
	t.Parallel()

	fill := behaviorBordRestRadiusFill(t, "border-start-start-radius:16px")
	behaviorBordRestAssertRadiusCorners(t, fill, 16, 0, 0, 0, "border-start-start-radius")
}

// TestBehaviorBorderStartEndRadiusMapsTopRight is border-start-end-radius: in
// the default left-to-right horizontal flow it rounds only the top-right
// corner of the background fill. Reference: Chrome 143.0.7499.40, 120px by
// 60px box, 16px corner.
func TestBehaviorBorderStartEndRadiusMapsTopRight(t *testing.T) {
	t.Parallel()

	fill := behaviorBordRestRadiusFill(t, "border-start-end-radius:16px")
	behaviorBordRestAssertRadiusCorners(t, fill, 0, 16, 0, 0, "border-start-end-radius")
}

// TestBehaviorBorderEndStartRadiusMapsBottomLeft is border-end-start-radius:
// in the default left-to-right horizontal flow it rounds only the
// bottom-left corner of the background fill. Reference: Chrome
// 143.0.7499.40, 120px by 60px box, 16px corner.
func TestBehaviorBorderEndStartRadiusMapsBottomLeft(t *testing.T) {
	t.Parallel()

	fill := behaviorBordRestRadiusFill(t, "border-end-start-radius:16px")
	behaviorBordRestAssertRadiusCorners(t, fill, 0, 0, 0, 16, "border-end-start-radius")
}

// TestBehaviorBorderEndEndRadiusMapsBottomRight is border-end-end-radius: in
// the default left-to-right horizontal flow it rounds only the bottom-right
// corner of the background fill. Reference: Chrome 143.0.7499.40, 120px by
// 60px box, 16px corner.
func TestBehaviorBorderEndEndRadiusMapsBottomRight(t *testing.T) {
	t.Parallel()

	fill := behaviorBordRestRadiusFill(t, "border-end-end-radius:16px")
	behaviorBordRestAssertRadiusCorners(t, fill, 0, 0, 16, 0, "border-end-end-radius")
}

// TestBehaviorBorderBlockEndRadiusRoundsBottomCorners is
// border-block-end-radius: both bottom corners of the background fill carry
// the radius while the top corners stay square. Reference: Chrome
// 143.0.7499.40, 120px by 60px box, 16px radius. Note: this side radius is
// not a standard CSS property, so Chrome drops the declaration; this pins
// the engine extension which rounds both bottom corners.
func TestBehaviorBorderBlockEndRadiusRoundsBottomCorners(t *testing.T) {
	t.Parallel()

	fill := behaviorBordRestRadiusFill(t, "border-block-end-radius:16px")
	behaviorBordRestAssertRadiusCorners(t, fill, 0, 0, 16, 16, "border-block-end-radius")
}

// TestBehaviorBorderBlockStartRadiusRoundsTopCorners is
// border-block-start-radius: both top corners of the background fill carry
// the radius while the bottom corners stay square. Reference: Chrome
// 143.0.7499.40, 120px by 60px box, 16px radius. Note: this side radius is
// not a standard CSS property, so Chrome drops the declaration; this pins
// the engine extension which rounds both top corners.
func TestBehaviorBorderBlockStartRadiusRoundsTopCorners(t *testing.T) {
	t.Parallel()

	fill := behaviorBordRestRadiusFill(t, "border-block-start-radius:16px")
	behaviorBordRestAssertRadiusCorners(t, fill, 16, 16, 0, 0, "border-block-start-radius")
}

// TestBehaviorBorderInlineEndRadiusRoundsRightCorners is
// border-inline-end-radius: both right corners of the background fill carry
// the radius while the left corners stay square. Reference: Chrome
// 143.0.7499.40, 120px by 60px box, 16px radius. Note: this side radius is
// not a standard CSS property, so Chrome drops the declaration; this pins
// the engine extension which rounds both right corners.
func TestBehaviorBorderInlineEndRadiusRoundsRightCorners(t *testing.T) {
	t.Parallel()

	fill := behaviorBordRestRadiusFill(t, "border-inline-end-radius:16px")
	behaviorBordRestAssertRadiusCorners(t, fill, 0, 16, 16, 0, "border-inline-end-radius")
}

// TestBehaviorBorderInlineStartRadiusRoundsLeftCorners is
// border-inline-start-radius: both left corners of the background fill carry
// the radius while the right corners stay square. Reference: Chrome
// 143.0.7499.40, 120px by 60px box, 16px radius. Note: this side radius is
// not a standard CSS property, so Chrome drops the declaration; this pins
// the engine extension which rounds both left corners.
func TestBehaviorBorderInlineStartRadiusRoundsLeftCorners(t *testing.T) {
	t.Parallel()

	fill := behaviorBordRestRadiusFill(t, "border-inline-start-radius:16px")
	behaviorBordRestAssertRadiusCorners(t, fill, 16, 0, 0, 16, "border-inline-start-radius")
}

// TestBehaviorBorderImageShorthandPaintsFrame is border-image: the shorthand
// with a 2px slice and a 10px width paints the eight-piece frame whose
// top-left corner carries a 2px by 2px payload at 10px painted size.
// Reference: Chrome 143.0.7499.40, 100px by 60px box with a 10px border and
// an 8px by 8px image.
func TestBehaviorBorderImageShorthandPaintsFrame(t *testing.T) {
	t.Parallel()

	img := tinyPNG(8, 8)
	res := layoutHTMLWithImages(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="bi" style="width:100px;height:60px;border:10px solid #000;`+
		`border-image:url(border.png) 2 / 10px stretch"></div>`+
		`</body></html>`, img, "border.png")

	cells := behaviorBgImages(opsOfKind(res, OpImage))
	if len(cells) != 8 {
		t.Fatalf("border-image cells = %d, want 8", len(cells))
	}

	corner := behaviorBgCorner(cells)
	if corner == nil {
		t.Fatal("no top-left corner cell in the shorthand frame")
	}

	if corner.ImgW != 2 || corner.ImgH != 2 {
		t.Errorf("shorthand corner payload = %dx%d, want 2x2", corner.ImgW, corner.ImgH)
	}

	if !near(corner.W, pxToPt(10)) || !near(corner.H, pxToPt(10)) {
		t.Errorf("shorthand corner painted size = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want 10px x 10px",
			corner.W, corner.H, corner.W/ptPerCSSPx, corner.H/ptPerCSSPx)
	}
}
