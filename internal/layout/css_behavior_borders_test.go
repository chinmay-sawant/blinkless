package layout

import "testing"

// This file holds the border and paint behavior tests: every property below
// is asserted through an emitted op or a used value (a measured border-box
// edge or an op field), never through a stored style string. The engine lays
// out in points and 1 CSS pixel = 0.75pt (ptPerCSSPx in
// css_review_02_test.go), so every expectation is written in CSS pixels and
// converted with pxToPt before comparing, which is how Chrome reports it.
// Reference browser for every case: Chrome 143.0.7499.40.

// behaviorBordersOutlineOps returns the outline ops in the display list.
func behaviorBordersOutlineOps(ops []Op) []Op {
	var out []Op

	for _, paintOp := range ops {
		if paintOp.isOutline() {
			out = append(out, paintOp)
		}
	}

	return out
}

// behaviorBordersBoxOps returns the ops owned by the box with the given id,
// using its recorded op range.
func behaviorBordersBoxOps(t *testing.T, res *Result, elementID string) []Op {
	t.Helper()

	elementBox := boxByID(t, res, elementID)

	if elementBox.opEnd < elementBox.opStart {
		return nil
	}

	start, end := elementBox.opStart, elementBox.opEnd
	if end >= len(res.Ops) {
		end = len(res.Ops) - 1
	}

	return res.Ops[start : end+1]
}

// behaviorBordersHorizontalLines returns horizontal (W > 0, H == 0) OpLine ops.
func behaviorBordersHorizontalLines(ops []Op) []Op {
	var out []Op

	for _, paintOp := range ops {
		if paintOp.Kind == OpLine && paintOp.H == 0 && paintOp.W > 0 {
			out = append(out, paintOp)
		}
	}

	return out
}

// behaviorBordersVerticalLines returns vertical (H > 0, W == 0) OpLine ops.
func behaviorBordersVerticalLines(ops []Op) []Op {
	var out []Op

	for _, paintOp := range ops {
		if paintOp.Kind == OpLine && paintOp.W == 0 && paintOp.H > 0 {
			out = append(out, paintOp)
		}
	}

	return out
}

// behaviorBordersRunWidthCase lays out one 200x40px box with the given border
// style and asserts the edge stroke width. selectLines picks the candidate
// line ops, edgeOf reads the item edge coordinate, linePos reads the same
// coordinate off a line op, and edgeName names the edge in failures.
func behaviorBordersRunWidthCase(
	t *testing.T, style string, wantPx float64,
	selectLines func([]Op) []Op,
	edgeOf func(*box) float64,
	linePos func(Op) float64,
	edgeName string,
) {
	t.Helper()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:200px;height:40px;`+style+`"></div>`+
		`</body></html>`)

	item := boxByID(t, res, "item")

	var found *Op

	for _, lineOp := range selectLines(res.Ops) {
		if near(linePos(lineOp), edgeOf(item)) {
			candidate := lineOp
			found = &candidate

			break
		}
	}

	if found == nil {
		t.Fatalf("no %s border op at %.4fpt", edgeName, edgeOf(item))
	}

	if !near(found.Width, pxToPt(wantPx)) {
		t.Errorf("%s border stroke width = %.4fpt (%.2fpx), want %.2fpx",
			edgeName, found.Width, found.Width/ptPerCSSPx, wantPx)
	}
}

// TestBehaviorBorderTopWidthEmittedStrokeWidth is border-top-width: the top
// edge stroke width follows the declared width. Reference: Chrome
// 143.0.7499.40, 2px vs 6px top border on a 200px box.
func TestBehaviorBorderTopWidthEmittedStrokeWidth(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name   string
		style  string
		wantPx float64
	}{
		{"thin top edge", "border-top-width:2px;border-top-style:solid;border-top-color:#000", 2},
		{"thick top edge", "border-top-width:6px;border-top-style:solid;border-top-color:#000", 6},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			behaviorBordersRunWidthCase(t, testCase.style, testCase.wantPx,
				behaviorBordersHorizontalLines,
				func(item *box) float64 { return item.y },
				func(lineOp Op) float64 { return lineOp.Y },
				"top")
		})
	}
}

// TestBehaviorBorderLeftWidthEmittedStrokeWidth is border-left-width: the left
// edge stroke width follows the declared width. Reference: Chrome
// 143.0.7499.40, 3px vs 7px left border on a 200px box.
func TestBehaviorBorderLeftWidthEmittedStrokeWidth(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name   string
		style  string
		wantPx float64
	}{
		{"thin left edge", "border-left-width:3px;border-left-style:solid;border-left-color:#000", 3},
		{"thick left edge", "border-left-width:7px;border-left-style:solid;border-left-color:#000", 7},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			behaviorBordersRunWidthCase(t, testCase.style, testCase.wantPx,
				behaviorBordersVerticalLines,
				func(item *box) float64 { return item.x },
				func(lineOp Op) float64 { return lineOp.X },
				"left")
		})
	}
}

// TestBehaviorBorderTopStyleDashedExpandsSegments is border-top-style: solid
// paints one segment, dashed expands into several, none paints nothing.
// Reference: Chrome 143.0.7499.40, 200px top edge.
func TestBehaviorBorderTopStyleDashedExpandsSegments(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="solid" style="width:200px;height:20px;border-top:3px solid #000"></div>`+
		`<div id="dashed" style="width:200px;height:20px;border-top:3px dashed #000"></div>`+
		`<div id="none" style="width:200px;height:20px;border-top:3px none #000"></div>`+
		`</body></html>`)

	solid := boxByID(t, res, "solid")
	dashed := boxByID(t, res, "dashed")
	none := boxByID(t, res, "none")

	countAt := func(y float64) int {
		segCount := 0

		for _, paintOp := range behaviorBordersHorizontalLines(res.Ops) {
			if near(paintOp.Y, y) {
				segCount++
			}
		}

		return segCount
	}

	if got := countAt(solid.y); got != 1 {
		t.Errorf("solid top edge segments = %d, want 1", got)
	}

	if got := countAt(dashed.y); got <= 1 {
		t.Errorf("dashed top edge segments = %d, want more than 1", got)
	}

	if got := countAt(none.y); got != 0 {
		t.Errorf("none top edge segments = %d, want 0", got)
	}
}

// TestBehaviorBorderTopLeftRadiusPaintsCorner is border-top-left-radius: only
// the top-left corner of the background fill carries the radius. Reference:
// Chrome 143.0.7499.40, 120px by 60px box, 16px corner.
func TestBehaviorBorderTopLeftRadiusPaintsCorner(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:120px;height:60px;background-color:#f00;border-top-left-radius:16px"></div>`+
		`</body></html>`)

	fill := redFillOp(t, res.Ops)
	if !near(fill.RadiusTopLeft, pxToPt(16)) {
		t.Errorf("fill RadiusTopLeft = %.4fpt (%.2fpx), want 16px",
			fill.RadiusTopLeft, fill.RadiusTopLeft/ptPerCSSPx)
	}

	for name, got := range map[string]float64{
		"RadiusTopRight": fill.RadiusTopRight, "RadiusBottomRight": fill.RadiusBottomRight,
		"RadiusBottomLeft": fill.RadiusBottomLeft,
	} {
		if !near(got, 0) {
			t.Errorf("fill %s = %.4fpt, want 0 (only top-left rounds)", name, got)
		}
	}
}

// TestBehaviorOutlineShorthandPaintsOutsideBorderBox is outline: the shorthand
// strokes four sides outside the border box without changing layout size.
// Reference: Chrome 143.0.7499.40, 80px by 40px box, 3px outline, 5px offset.
// This uses a different width and offset than TestOutlinePaintsInflatedRect.
func TestBehaviorOutlineShorthandPaintsOutsideBorderBox(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:80px;height:40px;outline:3px solid red;outline-offset:5px"></div>`+
		`</body></html>`)

	item := boxByID(t, res, "item")
	if !near(item.w, pxToPt(80)) || !near(item.height, pxToPt(40)) {
		t.Fatalf("outline changed the layout box: %.4fpt x %.4fpt, want 80px x 40px",
			item.w/ptPerCSSPx, item.height/ptPerCSSPx)
	}

	strokes := behaviorBordersOutlineOps(res.Ops)
	if len(strokes) != 4 {
		t.Fatalf("outline ops = %d, want 4 sides", len(strokes))
	}

	const inflatePx = 6.5 // outline-offset 5px + outline-width 3px / 2

	outX := item.x - pxToPt(inflatePx)
	outY := item.y - pxToPt(inflatePx)
	outW := item.w + pxToPt(2*inflatePx)
	outH := item.height + pxToPt(2*inflatePx)

	for _, stroke := range strokes {
		if stroke.Kind != OpLine || !near(stroke.Width, pxToPt(3)) {
			t.Fatalf("outline side = kind %v width %.4fpt, want OpLine width 3px",
				stroke.Kind, stroke.Width)
		}

		if !outlineStrokeMatchesRect(stroke, outX, outY, outW, outH) {
			t.Fatalf("outline side at (%.2f, %.2f) %.2f x %.2f is outside the inflated rect (%.2f, %.2f) %.2f x %.2f",
				stroke.X, stroke.Y, stroke.W, stroke.H, outX, outY, outW, outH)
		}
	}
}

// TestBehaviorOutlineWidthSetsStrokeWidth is outline-width: the outline stroke
// width follows the longhand. Reference: Chrome 143.0.7499.40, 1px vs 4px.
func TestBehaviorOutlineWidthSetsStrokeWidth(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name   string
		style  string
		wantPx float64
	}{
		{"thin outline", "outline-style:solid;outline-color:red;outline-width:1px", 1},
		{"thick outline", "outline-style:solid;outline-color:red;outline-width:4px", 4},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
				`<div id="item" style="width:80px;height:40px;`+testCase.style+`"></div>`+
				`</body></html>`)

			strokes := behaviorBordersOutlineOps(res.Ops)
			if len(strokes) != 4 {
				t.Fatalf("outline ops = %d, want 4 sides", len(strokes))
			}

			for _, stroke := range strokes {
				if !near(stroke.Width, pxToPt(testCase.wantPx)) {
					t.Errorf("outline stroke width = %.4fpt (%.2fpx), want %.2fpx",
						stroke.Width, stroke.Width/ptPerCSSPx, testCase.wantPx)
				}
			}
		})
	}
}

// TestBehaviorOutlineStyleDashedExpandsSegments is outline-style: solid paints
// four sides, dashed expands each side into dash segments. Reference: Chrome
// 143.0.7499.40, 80px by 40px box, 3px outline.
func TestBehaviorOutlineStyleDashedExpandsSegments(t *testing.T) {
	t.Parallel()

	solid := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:80px;height:40px;outline-width:3px;outline-style:solid;outline-color:red"></div>`+
		`</body></html>`)
	dashed := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:80px;height:40px;outline-width:3px;outline-style:dashed;outline-color:red"></div>`+
		`</body></html>`)

	if got := len(behaviorBordersOutlineOps(solid.Ops)); got != 4 {
		t.Errorf("solid outline ops = %d, want 4 sides", got)
	}

	if got := len(behaviorBordersOutlineOps(dashed.Ops)); got <= 4 {
		t.Errorf("dashed outline ops = %d, want more than 4 dash segments", got)
	}
}

// TestBehaviorOutlineColorPaintsStrokeColor is outline-color: the outline
// stroke carries the declared color. Reference: Chrome 143.0.7499.40.
func TestBehaviorOutlineColorPaintsStrokeColor(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name  string
		style string
		wantR float64
		wantG float64
		wantB float64
	}{
		{"red outline", "outline:3px solid red", 1, 0, 0},
		{"blue outline", "outline:3px solid blue", 0, 0, 1},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
				`<div id="item" style="width:80px;height:40px;`+testCase.style+`"></div>`+
				`</body></html>`)

			strokes := behaviorBordersOutlineOps(res.Ops)
			if len(strokes) != 4 {
				t.Fatalf("outline ops = %d, want 4 sides", len(strokes))
			}

			for _, stroke := range strokes {
				if !near(stroke.R, testCase.wantR) || !near(stroke.G, testCase.wantG) || !near(stroke.B, testCase.wantB) {
					t.Errorf("outline color = (%.2f, %.2f, %.2f), want (%.0f, %.0f, %.0f)",
						stroke.R, stroke.G, stroke.B, testCase.wantR, testCase.wantG, testCase.wantB)
				}
			}
		})
	}
}

// TestBehaviorOutlineOffsetInflatesRect is outline-offset: a larger offset
// pushes the outline rect further from the border box. Reference: Chrome
// 143.0.7499.40, 2px outline, 0px vs 8px offset.
func TestBehaviorOutlineOffsetInflatesRect(t *testing.T) {
	t.Parallel()

	zero := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:80px;height:40px;outline:2px solid red;outline-offset:0px"></div>`+
		`</body></html>`)
	wide := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:80px;height:40px;outline:2px solid red;outline-offset:8px"></div>`+
		`</body></html>`)

	zeroBox := boxByID(t, zero, "item")
	wideBox := boxByID(t, wide, "item")

	zeroStrokes := behaviorBordersOutlineOps(zero.Ops)
	wideStrokes := behaviorBordersOutlineOps(wide.Ops)

	if len(zeroStrokes) != 4 || len(wideStrokes) != 4 {
		t.Fatalf("outline ops = %d / %d, want 4 sides each", len(zeroStrokes), len(wideStrokes))
	}

	// inflate = offset + width/2: 0px + 1px = 1px vs 8px + 1px = 9px.
	zeroInflate := pxToPt(1)
	wideInflate := pxToPt(9)

	behaviorBordersAssertStrokesMatchRect(t, zeroStrokes, zeroBox, zeroInflate, "zero-offset", "1px")
	behaviorBordersAssertStrokesMatchRect(t, wideStrokes, wideBox, wideInflate, "8px-offset", "9px")

	// The wider offset must sit further out than the zero offset.
	zeroMinX := behaviorBordersMinStrokeX(zeroStrokes)
	wideMinX := behaviorBordersMinStrokeX(wideStrokes)

	if !near(wideMinX, wideBox.x-wideInflate) || !near(zeroMinX, zeroBox.x-zeroInflate) {
		t.Errorf("outline left edge = %.4fpt / %.4fpt, want %.4fpt / %.4fpt (8px further out)",
			zeroMinX, wideMinX, zeroBox.x-zeroInflate, wideBox.x-wideInflate)
	}
}

// behaviorBordersAssertStrokesMatchRect asserts every outline stroke lies on
// the box rect inflated by inflate on each side.
func behaviorBordersAssertStrokesMatchRect(
	t *testing.T, strokes []Op, item *box, inflate float64, tag, wantDesc string,
) {
	t.Helper()

	for _, stroke := range strokes {
		if !outlineStrokeMatchesRect(stroke, item.x-inflate, item.y-inflate,
			item.w+2*inflate, item.height+2*inflate) {
			t.Errorf("%s outline side at (%.2f, %.2f) misses the %s inflated rect",
				tag, stroke.X, stroke.Y, wantDesc)
		}
	}
}

// behaviorBordersMinStrokeX returns the smallest stroke X edge.
func behaviorBordersMinStrokeX(strokes []Op) float64 {
	minX := strokes[0].X

	for _, stroke := range strokes {
		if stroke.X < minX {
			minX = stroke.X
		}
	}

	return minX
}

// TestBehaviorOpacityFoldsIntoPaintOps is opacity: descendant paint ops report
// the element opacity through Opacity(). Reference: Chrome 143.0.7499.40,
// 0.5 vs 1.
func TestBehaviorOpacityFoldsIntoPaintOps(t *testing.T) {
	t.Parallel()

	half := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:100px;height:50px;background-color:#f00;opacity:0.5">x</div>`+
		`</body></html>`)
	full := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:100px;height:50px;background-color:#f00">x</div>`+
		`</body></html>`)

	halfOps := behaviorBordersBoxOps(t, half, "item")
	if len(halfOps) == 0 {
		t.Fatal("half-opacity box owns no ops")
	}

	behaviorBordersAssertFillTextOpacity(t, halfOps, 0.5, "half-opacity")

	fullOps := behaviorBordersBoxOps(t, full, "item")
	if len(fullOps) == 0 {
		t.Fatal("full-opacity box owns no ops")
	}

	behaviorBordersAssertFillTextOpacity(t, fullOps, 1, "default")
}

// behaviorBordersAssertFillTextOpacity asserts every fill and text op in ops
// reports the given opacity.
func behaviorBordersAssertFillTextOpacity(t *testing.T, ops []Op, wantOpacity float64, tag string) {
	t.Helper()

	for _, paintOp := range ops {
		if paintOp.Kind != OpFillRect && paintOp.Kind != OpText {
			continue
		}

		if !near(paintOp.Opacity(), wantOpacity) {
			t.Errorf("%s op kind %v Opacity() = %.3f, want %.3f", tag, paintOp.Kind, paintOp.Opacity(), wantOpacity)
		}
	}
}

// TestBehaviorBoxShadowPaintsOffsetFill is box-shadow: an outset shadow paints
// a fill offset from the border box. Reference: Chrome 143.0.7499.40, 100px
// by 50px box, 8px by 8px red shadow.
func TestBehaviorBoxShadowPaintsOffsetFill(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:100px;height:50px;box-shadow:8px 8px 0 red"></div>`+
		`</body></html>`)

	item := boxByID(t, res, "item")
	wantX, wantY := item.x+pxToPt(8), item.y+pxToPt(8)

	if behaviorBordersFindFill(res.Ops, wantX, wantY, item.w, item.height, 1, 0, 0) == nil {
		t.Errorf("no red shadow fill at (+8px, +8px) %.4fpt,%.4fpt sized %.4fpt x %.4fpt",
			wantX, wantY, item.w, item.height)
	}
}

// TestBehaviorBackgroundColorPaintsFill is background-color: the border box
// paints a fill in the declared color. Reference: Chrome 143.0.7499.40,
// 100px by 50px red box.
func TestBehaviorBackgroundColorPaintsFill(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:100px;height:50px;background-color:#f00"></div>`+
		`</body></html>`)

	item := boxByID(t, res, "item")

	found := behaviorBordersFindFill(res.Ops, item.x, item.y, item.w, item.height, 1, 0, 0)

	if found == nil {
		t.Errorf("no red background fill at box (%.4fpt, %.4fpt) %.4fpt x %.4fpt",
			item.x, item.y, item.w, item.height)
	}
}

// behaviorBordersFindFill returns the first fill op at the given rect with
// the given color, or nil.
func behaviorBordersFindFill(ops []Op, wantX, wantY, wantW, wantH, wantR, wantG, wantB float64) *Op {
	for _, paintOp := range ops {
		if paintOp.Kind != OpFillRect {
			continue
		}

		if near(paintOp.X, wantX) && near(paintOp.Y, wantY) && near(paintOp.W, wantW) && near(paintOp.H, wantH) &&
			near(paintOp.R, wantR) && near(paintOp.G, wantG) && near(paintOp.B, wantB) {
			candidate := paintOp

			return &candidate
		}
	}

	return nil
}

// TestBehaviorBackgroundPositionMovesImage is background-position: right
// bottom slides a no-repeat background image to the far corner while left top
// keeps it at the origin. Reference: Chrome 143.0.7499.40, 80px by 40px box,
// 10px by 10px image.
func TestBehaviorBackgroundPositionMovesImage(t *testing.T) {
	t.Parallel()

	png := tinyPNG(10, 10)
	htmlSrc := func(pos string) string {
		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="item" style="width:80px;height:40px;background-image:url(bg.png);` +
			`background-repeat:no-repeat;background-position:` + pos + `"></div>` +
			`</body></html>`
	}

	left := layoutHTMLWithImages(t, htmlSrc("left top"), png, "")
	right := layoutHTMLWithImages(t, htmlSrc("right bottom"), png, "")

	leftBox := boxByID(t, left, "item")
	rightBox := boxByID(t, right, "item")

	leftImg := behaviorBordersFindBackgroundImage(left.Ops)
	rightImg := behaviorBordersFindBackgroundImage(right.Ops)

	if leftImg == nil || rightImg == nil {
		t.Fatalf("background images = left %v right %v, want one each", leftImg != nil, rightImg != nil)
	}

	if !near(leftImg.X, leftBox.x) || !near(leftImg.Y, leftBox.y) {
		t.Errorf("left top image at (%.4fpt, %.4fpt), want box origin (%.4fpt, %.4fpt)",
			leftImg.X, leftImg.Y, leftBox.x, leftBox.y)
	}

	wantX := rightBox.x + rightBox.w - rightImg.W
	wantY := rightBox.y + rightBox.height - rightImg.H

	if rightBox.w <= rightImg.W || rightBox.height <= rightImg.H {
		t.Fatalf("test image %.4fpt x %.4fpt fills the %.4fpt x %.4fpt box, pick a smaller image",
			rightImg.W, rightImg.H, rightBox.w, rightBox.height)
	}

	if !near(rightImg.X, wantX) || !near(rightImg.Y, wantY) {
		t.Errorf("right bottom image at (%.4fpt, %.4fpt), want (%.4fpt, %.4fpt)",
			rightImg.X, rightImg.Y, wantX, wantY)
	}
}

// behaviorBordersFindBackgroundImage returns the first background image op,
// or nil.
func behaviorBordersFindBackgroundImage(ops []Op) *Op {
	for _, paintOp := range ops {
		if paintOp.Kind == OpImage && paintOp.IsBackground {
			candidate := paintOp

			return &candidate
		}
	}

	return nil
}

// TestBehaviorBorderColorPaintsTopEdge is border-color: the top edge stroke
// carries the declared color. Reference: Chrome 143.0.7499.40.
func TestBehaviorBorderColorPaintsTopEdge(t *testing.T) {
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
