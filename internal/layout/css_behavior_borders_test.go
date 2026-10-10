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

	for _, op := range ops {
		if op.isOutline() {
			out = append(out, op)
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

	for _, op := range ops {
		if op.Kind == OpLine && op.H == 0 && op.W > 0 {
			out = append(out, op)
		}
	}

	return out
}

// behaviorBordersVerticalLines returns vertical (H > 0, W == 0) OpLine ops.
func behaviorBordersVerticalLines(ops []Op) []Op {
	var out []Op

	for _, op := range ops {
		if op.Kind == OpLine && op.W == 0 && op.H > 0 {
			out = append(out, op)
		}
	}

	return out
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

			res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
				`<div id="item" style="width:200px;height:40px;`+testCase.style+`"></div>`+
				`</body></html>`)

			item := boxByID(t, res, "item")
			var top *Op

			for _, op := range behaviorBordersHorizontalLines(res.Ops) {
				if near(op.Y, item.y) {
					candidate := op
					top = &candidate

					break
				}
			}

			if top == nil {
				t.Fatalf("no top border op at y %.4fpt", item.y)
			}

			if !near(top.Width, pxToPt(testCase.wantPx)) {
				t.Errorf("top border stroke width = %.4fpt (%.2fpx), want %.2fpx",
					top.Width, top.Width/ptPerCSSPx, testCase.wantPx)
			}
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

			res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
				`<div id="item" style="width:200px;height:40px;`+testCase.style+`"></div>`+
				`</body></html>`)

			item := boxByID(t, res, "item")
			var left *Op

			for _, op := range behaviorBordersVerticalLines(res.Ops) {
				if near(op.X, item.x) {
					candidate := op
					left = &candidate

					break
				}
			}

			if left == nil {
				t.Fatalf("no left border op at x %.4fpt", item.x)
			}

			if !near(left.Width, pxToPt(testCase.wantPx)) {
				t.Errorf("left border stroke width = %.4fpt (%.2fpx), want %.2fpx",
					left.Width, left.Width/ptPerCSSPx, testCase.wantPx)
			}
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
		n := 0

		for _, op := range behaviorBordersHorizontalLines(res.Ops) {
			if near(op.Y, y) {
				n++
			}
		}

		return n
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

	for _, stroke := range zeroStrokes {
		if !outlineStrokeMatchesRect(stroke, zeroBox.x-zeroInflate, zeroBox.y-zeroInflate,
			zeroBox.w+2*zeroInflate, zeroBox.height+2*zeroInflate) {
			t.Errorf("zero-offset outline side at (%.2f, %.2f) misses the 1px inflated rect",
				stroke.X, stroke.Y)
		}
	}

	for _, stroke := range wideStrokes {
		if !outlineStrokeMatchesRect(stroke, wideBox.x-wideInflate, wideBox.y-wideInflate,
			wideBox.w+2*wideInflate, wideBox.height+2*wideInflate) {
			t.Errorf("8px-offset outline side at (%.2f, %.2f) misses the 9px inflated rect",
				stroke.X, stroke.Y)
		}
	}

	// The wider offset must sit further out than the zero offset.
	zeroMinX, wideMinX := zeroStrokes[0].X, wideStrokes[0].X
	for _, stroke := range zeroStrokes {
		if stroke.X < zeroMinX {
			zeroMinX = stroke.X
		}
	}

	for _, stroke := range wideStrokes {
		if stroke.X < wideMinX {
			wideMinX = stroke.X
		}
	}

	if !near(wideMinX, wideBox.x-wideInflate) || !near(zeroMinX, zeroBox.x-zeroInflate) {
		t.Errorf("outline left edge = %.4fpt / %.4fpt, want %.4fpt / %.4fpt (8px further out)",
			zeroMinX, wideMinX, zeroBox.x-zeroInflate, wideBox.x-wideInflate)
	}
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

	for _, op := range halfOps {
		if op.Kind != OpFillRect && op.Kind != OpText {
			continue
		}

		if !near(op.Opacity(), 0.5) {
			t.Errorf("half-opacity op kind %v Opacity() = %.3f, want 0.5", op.Kind, op.Opacity())
		}
	}

	fullOps := behaviorBordersBoxOps(t, full, "item")
	if len(fullOps) == 0 {
		t.Fatal("full-opacity box owns no ops")
	}

	for _, op := range fullOps {
		if op.Kind != OpFillRect && op.Kind != OpText {
			continue
		}

		if !near(op.Opacity(), 1) {
			t.Errorf("default op kind %v Opacity() = %.3f, want 1", op.Kind, op.Opacity())
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

	found := false

	for _, op := range res.Ops {
		if op.Kind != OpFillRect {
			continue
		}

		if near(op.X, wantX) && near(op.Y, wantY) && near(op.W, item.w) && near(op.H, item.height) &&
			near(op.R, 1) && near(op.G, 0) && near(op.B, 0) {
			found = true

			break
		}
	}

	if !found {
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

	found := false

	for _, op := range res.Ops {
		if op.Kind != OpFillRect {
			continue
		}

		if near(op.X, item.x) && near(op.Y, item.y) && near(op.W, item.w) && near(op.H, item.height) &&
			near(op.R, 1) && near(op.G, 0) && near(op.B, 0) {
			found = true

			break
		}
	}

	if !found {
		t.Errorf("no red background fill at box (%.4fpt, %.4fpt) %.4fpt x %.4fpt",
			item.x, item.y, item.w, item.height)
	}
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

	var leftImg, rightImg *Op

	for _, op := range left.Ops {
		if op.Kind == OpImage && op.IsBackground {
			candidate := op
			leftImg = &candidate

			break
		}
	}

	for _, op := range right.Ops {
		if op.Kind == OpImage && op.IsBackground {
			candidate := op
			rightImg = &candidate

			break
		}
	}

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

			res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
				`<div id="item" style="width:200px;height:40px;`+testCase.style+`"></div>`+
				`</body></html>`)

			item := boxByID(t, res, "item")

			found := false

			for _, op := range behaviorBordersHorizontalLines(res.Ops) {
				if !near(op.Y, item.y) {
					continue
				}

				found = true

				if !near(op.R, testCase.wantR) || !near(op.G, testCase.wantG) || !near(op.B, testCase.wantB) {
					t.Errorf("top edge color = (%.2f, %.2f, %.2f), want (%.0f, %.0f, %.0f)",
						op.R, op.G, op.B, testCase.wantR, testCase.wantG, testCase.wantB)
				}
			}

			if !found {
				t.Fatalf("no top border op at y %.4fpt", item.y)
			}
		})
	}
}
