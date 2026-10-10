package layout

import "testing"

// Remaining position and replaced-content behavior tests: every property
// below is asserted through a USED value (a used box offset or a painted
// image op rect), never through a stored style string. Lengths are written
// in CSS px and converted with pxToPt (1px = 0.75pt, ptPerCSSPx in
// css_review_02_test.go), which is how Chrome reports them. Reference
// browser for every case: Chrome 143.0.7499.40. Helpers boxByID, near,
// pxToPt, opsOfKind, layoutHTMLWithImages, and tinyPNG come from
// css_review_02_test.go and layout_test.go in this same package.

// TestBehaviorRightShiftsRelativeBox is right: right:30px on a relative box
// moves its used x left 30px against the same box with right:auto at x 0.
// Reference: Chrome 143.0.7499.40, 50px block, right 30px lands its border
// left at -30px.
func TestBehaviorRightShiftsRelativeBox(t *testing.T) {
	t.Parallel()

	with := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="r1" style="position:relative;right:30px;width:50px;height:50px"></div>`+
		`</body></html>`)
	without := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="r1" style="position:relative;width:50px;height:50px"></div>`+
		`</body></html>`)

	got := boxByID(t, with, "r1")
	base := boxByID(t, without, "r1")

	if !near(base.x, 0) {
		t.Fatalf("auto-right base x = %.4fpt, want 0", base.x)
	}

	if !near(got.x-base.x, pxToPt(-30)) {
		t.Errorf("right delta = %.4fpt (%.2fpx), want -30px",
			got.x-base.x, (got.x-base.x)/ptPerCSSPx)
	}
}

// TestBehaviorBottomShiftsRelativeBox is bottom: bottom:20px on a relative
// box moves its used y up 20px against the same box with bottom:auto at
// y 0. Reference: Chrome 143.0.7499.40, 50px block, bottom 20px lands its
// border top at -20px.
func TestBehaviorBottomShiftsRelativeBox(t *testing.T) {
	t.Parallel()

	with := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="b1" style="position:relative;bottom:20px;width:50px;height:50px"></div>`+
		`</body></html>`)
	without := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="b1" style="position:relative;width:50px;height:50px"></div>`+
		`</body></html>`)

	got := boxByID(t, with, "b1")
	base := boxByID(t, without, "b1")

	if !near(base.y, 0) {
		t.Fatalf("auto-bottom base y = %.4fpt, want 0", base.y)
	}

	if !near(got.y-base.y, pxToPt(-20)) {
		t.Errorf("bottom delta = %.4fpt (%.2fpx), want -20px",
			got.y-base.y, (got.y-base.y)/ptPerCSSPx)
	}
}

// TestBehaviorInsetBlockStartAnchorsAbsoluteBox is inset-block-start: in
// horizontal-tb it maps to top, so inset-block-start:16px on an absolute box
// pins its used top 16px inside the relative parent's padding box.
// Reference: Chrome 143.0.7499.40, 200px relative parent, 50x40px absolute
// child lands at y 16px.
func TestBehaviorInsetBlockStartAnchorsAbsoluteBox(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="par" style="position:relative;width:200px;height:200px">`+
		`<div id="kid" style="position:absolute;inset-block-start:16px;width:50px;height:40px"></div>`+
		`</div></body></html>`)

	parent := boxByID(t, res, "par")
	kid := boxByID(t, res, "kid")

	if !near(kid.y-parent.y, pxToPt(16)) {
		t.Errorf("kid top offset = %.4fpt (%.2fpx), want 16px",
			kid.y-parent.y, (kid.y-parent.y)/ptPerCSSPx)
	}

	if !near(kid.w, pxToPt(50)) || !near(kid.height, pxToPt(40)) {
		t.Errorf("kid size = %.4fpt x %.4fpt, want 50px x 40px", kid.w, kid.height)
	}
}

// TestBehaviorInsetBlockEndAnchorsAbsoluteBox is inset-block-end: in
// horizontal-tb it maps to bottom, so inset-block-end:16px on a 40px tall
// absolute box inside a 200px relative parent pins its used top at
// 200-40-16 = 144px. Reference: Chrome 143.0.7499.40.
func TestBehaviorInsetBlockEndAnchorsAbsoluteBox(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="par" style="position:relative;width:200px;height:200px">`+
		`<div id="kid" style="position:absolute;inset-block-end:16px;width:50px;height:40px"></div>`+
		`</div></body></html>`)

	parent := boxByID(t, res, "par")
	kid := boxByID(t, res, "kid")

	if !near(kid.y-parent.y, pxToPt(144)) {
		t.Errorf("kid top offset = %.4fpt (%.2fpx), want 144px (200-40-16)",
			kid.y-parent.y, (kid.y-parent.y)/ptPerCSSPx)
	}

	if !near(kid.height, pxToPt(40)) {
		t.Errorf("kid height = %.4fpt (%.2fpx), want 40px",
			kid.height, kid.height/ptPerCSSPx)
	}
}

// TestBehaviorInsetInlineStartAnchorsAbsoluteBox is inset-inline-start: in
// ltr horizontal-tb it maps to left, so inset-inline-start:16px on an
// absolute box pins its used left 16px inside the relative parent's padding
// box. Reference: Chrome 143.0.7499.40, 200px relative parent, 50x40px
// absolute child lands at x 16px.
func TestBehaviorInsetInlineStartAnchorsAbsoluteBox(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="par" style="position:relative;width:200px;height:200px">`+
		`<div id="kid" style="position:absolute;inset-inline-start:16px;width:50px;height:40px"></div>`+
		`</div></body></html>`)

	parent := boxByID(t, res, "par")
	kid := boxByID(t, res, "kid")

	if !near(kid.x-parent.x, pxToPt(16)) {
		t.Errorf("kid left offset = %.4fpt (%.2fpx), want 16px",
			kid.x-parent.x, (kid.x-parent.x)/ptPerCSSPx)
	}

	if !near(kid.w, pxToPt(50)) || !near(kid.height, pxToPt(40)) {
		t.Errorf("kid size = %.4fpt x %.4fpt, want 50px x 40px", kid.w, kid.height)
	}
}

// TestBehaviorInsetInlineEndAnchorsAbsoluteBox is inset-inline-end: in ltr
// horizontal-tb it maps to right, so inset-inline-end:16px on a 50px wide
// absolute box inside a 200px relative parent pins its used left at
// 200-50-16 = 134px. Reference: Chrome 143.0.7499.40.
func TestBehaviorInsetInlineEndAnchorsAbsoluteBox(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="par" style="position:relative;width:200px;height:200px">`+
		`<div id="kid" style="position:absolute;inset-inline-end:16px;width:50px;height:40px"></div>`+
		`</div></body></html>`)

	parent := boxByID(t, res, "par")
	kid := boxByID(t, res, "kid")

	if !near(kid.x-parent.x, pxToPt(134)) {
		t.Errorf("kid left offset = %.4fpt (%.2fpx), want 134px (200-50-16)",
			kid.x-parent.x, (kid.x-parent.x)/ptPerCSSPx)
	}

	if !near(kid.w, pxToPt(50)) {
		t.Errorf("kid width = %.4fpt (%.2fpx), want 50px",
			kid.w, kid.w/ptPerCSSPx)
	}
}

// TestBehaviorObjectFitContainKeepsRatio is object-fit: a 40x20px source in
// a 100x100px img box paints stretched to 100x100px under the default fill
// but letterboxes to 100x50px under contain, centered so its used top sits
// 25px below the box top. Reference: Chrome 143.0.7499.40.
func TestBehaviorObjectFitContainKeepsRatio(t *testing.T) {
	t.Parallel()

	src := tinyPNG(40, 20)
	fillRes := layoutHTMLWithImages(t, `<html style="margin:0"><body style="margin:0">`+
		`<img src="x.png" style="display:block;width:100px;height:100px;object-fit:fill">`+
		`</body></html>`, src, "")
	containRes := layoutHTMLWithImages(t, `<html style="margin:0"><body style="margin:0">`+
		`<img src="x.png" style="display:block;width:100px;height:100px;object-fit:contain">`+
		`</body></html>`, src, "")

	fill := behaviorPosRemSingleImage(t, fillRes)
	contain := behaviorPosRemSingleImage(t, containRes)

	if !near(fill.W, pxToPt(100)) || !near(fill.H, pxToPt(100)) {
		t.Errorf("fill used paint = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want 100px x 100px",
			fill.W, fill.H, fill.W/ptPerCSSPx, fill.H/ptPerCSSPx)
	}

	if !near(contain.W, pxToPt(100)) || !near(contain.H, pxToPt(50)) {
		t.Errorf("contain used paint = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want 100px x 50px",
			contain.W, contain.H, contain.W/ptPerCSSPx, contain.H/ptPerCSSPx)
	}

	if !near(contain.Y-fill.Y, pxToPt(25)) {
		t.Errorf("contain top inset = %.4fpt (%.2fpx), want 25px ((100-50)/2)",
			contain.Y-fill.Y, (contain.Y-fill.Y)/ptPerCSSPx)
	}
}

// TestBehaviorObjectPositionAlignsNoneImage is object-position: with
// object-fit:none a 40x20px source keeps its intrinsic used size and paints
// at the box origin under left top but at the box far corner under right
// bottom, a 60px x shift and an 80px y shift in a 100x100px box. Reference:
// Chrome 143.0.7499.40.
func TestBehaviorObjectPositionAlignsNoneImage(t *testing.T) {
	t.Parallel()

	src := tinyPNG(40, 20)
	startRes := layoutHTMLWithImages(t, `<html style="margin:0"><body style="margin:0">`+
		`<img src="x.png" style="display:block;width:100px;height:100px;object-fit:none;object-position:left top">`+
		`</body></html>`, src, "")
	endRes := layoutHTMLWithImages(t, `<html style="margin:0"><body style="margin:0">`+
		`<img src="x.png" style="display:block;width:100px;height:100px;object-fit:none;object-position:right bottom">`+
		`</body></html>`, src, "")

	start := behaviorPosRemSingleImage(t, startRes)
	end := behaviorPosRemSingleImage(t, endRes)

	for name, op := range map[string]Op{"left top": start, "right bottom": end} {
		if !near(op.W, pxToPt(40)) || !near(op.H, pxToPt(20)) {
			t.Errorf("%s used paint = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want 40px x 20px intrinsic",
				name, op.W, op.H, op.W/ptPerCSSPx, op.H/ptPerCSSPx)
		}
	}

	if !near(end.X-start.X, pxToPt(60)) || !near(end.Y-start.Y, pxToPt(80)) {
		t.Errorf("right bottom shift = (%.4fpt, %.4fpt) = (%.2fpx, %.2fpx), want (60px, 80px)",
			end.X-start.X, end.Y-start.Y,
			(end.X-start.X)/ptPerCSSPx, (end.Y-start.Y)/ptPerCSSPx)
	}
}

// behaviorPosRemSingleImage returns the only image paint op in res, failing
// when the layout did not emit exactly one.
func behaviorPosRemSingleImage(t *testing.T, res *Result) Op {
	t.Helper()

	images := opsOfKind(res, OpImage)
	if len(images) != 1 {
		t.Fatalf("OpImage count = %d, want 1", len(images))
	}

	return images[0]
}
