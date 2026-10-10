package layout

import "testing"

// Reference behaviors pinned against Chrome 143.0.7499.40. Every assertion
// below reads a USED value (a measured box edge or a painted op bound),
// never a stored style string. Lengths are written in CSS px and converted
// with pxToPt (1px = 0.75pt, ptPerCSSPx in css_review_02_test.go), which is
// how Chrome reports them. Helpers boxByID, near, pxToPt, opsOfKind,
// layoutHTMLWithImages, and tinyPNG come from css_review_02_test.go and
// layout_test.go in this same package.

// TestBehaviorAlignItemsCenterOffset: in a 400x200px flex row with
// align-items:center, a 120x40px item centers on the cross axis, so its used
// y offset from the row is (200-40)/2 = 80px while x stays 0.
// Reference: Chrome 143.0.7499.40.
func TestBehaviorAlignItemsCenterOffset(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="row" style="display:flex;width:400px;height:200px;align-items:center">`+
		`<div id="item" style="width:120px;height:40px"></div>`+
		`</div></body></html>`)

	item := boxByID(t, res, "item")
	row := boxByID(t, res, "row")

	if !near(item.w, pxToPt(120)) || !near(item.height, pxToPt(40)) {
		t.Errorf("item used border box = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want 120px x 40px",
			item.w, item.height, item.w/ptPerCSSPx, item.height/ptPerCSSPx)
	}

	if !near(item.x-row.x, pxToPt(0)) || !near(item.y-row.y, pxToPt(80)) {
		t.Errorf("item used offset from row = (%.4fpt, %.4fpt) = (%.2fpx, %.2fpx), want (0px, 80px)",
			item.x-row.x, item.y-row.y,
			(item.x-row.x)/ptPerCSSPx, (item.y-row.y)/ptPerCSSPx)
	}
}

// TestBehaviorMinWidthClampsUsedWidthReference pins the same CAT-05 min-width
// floor as TestBehaviorMinWidthClampsUsedWidth in
// css_behavior_box_model_test.go:130. It carries a suffixed name because Go
// forbids two tests with one name in package layout; the used values
// asserted are identical. A box declared 50px wide with min-width 120px lays
// out at the 120px floor, while a box already wider than its min-width keeps
// its declared width.
// Reference: Chrome 143.0.7499.40, 400px wrapper.
func TestBehaviorMinWidthClampsUsedWidthReference(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		style  string
		wantPx float64
	}{
		{"clamped up to the floor", "width:50px;min-width:120px;height:30px", 120},
		{"declared width already wins", "width:200px;min-width:120px;height:30px", 200},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
				`<div id="outer" style="width:400px">`+
				`<div id="item" style="`+testCase.style+`"></div>`+
				`</div></body></html>`)

			item := boxByID(t, res, "item")
			if !near(item.w, pxToPt(testCase.wantPx)) || !near(item.height, pxToPt(30)) {
				t.Errorf("item used border box = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want %.2fpx x 30px",
					item.w, item.height, item.w/ptPerCSSPx, item.height/ptPerCSSPx,
					testCase.wantPx)
			}
		})
	}
}

// TestBehaviorBorderImageOutsetPaints: border-image-outset:10px expands the
// painted image bounds 10px beyond every side of the used border box. The
// box is 100x60px content with a 10px border, so its used border box is
// 120x80px and the painted union must be 140x100px centered on it.
// Reference: Chrome 143.0.7499.40. Asserted through painted OpImage bounds
// (used values), never the stored outset string. border-image-outset is
// implemented (borderImageOuterBounds in border_image.go), so this pins real
// behavior, not a gap.
func TestBehaviorBorderImageOutsetPaints(t *testing.T) {
	t.Parallel()

	img := tinyPNG(12, 12)
	res := layoutHTMLWithImages(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="bi" style="width:100px;height:60px;border:10px solid #000;`+
		`border-image-source:url(border.png);border-image-slice:30;border-image-outset:10px"></div>`+
		`</body></html>`, img, "border.png")

	bi := boxByID(t, res, "bi")
	if !near(bi.w, pxToPt(120)) || !near(bi.height, pxToPt(80)) {
		t.Errorf("used border box = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want 120px x 80px",
			bi.w, bi.height, bi.w/ptPerCSSPx, bi.height/ptPerCSSPx)
	}

	var cells []Op
	for _, op := range opsOfKind(res, OpImage) {
		if !op.IsBackground || op.W <= 0 || op.H <= 0 {
			continue
		}
		cells = append(cells, op)
	}

	if len(cells) == 0 {
		t.Fatalf("no painted border-image ops for #bi")
	}

	minX, minY := cells[0].X, cells[0].Y
	maxR, maxB := cells[0].X+cells[0].W, cells[0].Y+cells[0].H
	for _, op := range cells[1:] {
		if op.X < minX {
			minX = op.X
		}
		if op.Y < minY {
			minY = op.Y
		}
		if op.X+op.W > maxR {
			maxR = op.X + op.W
		}
		if op.Y+op.H > maxB {
			maxB = op.Y + op.H
		}
	}

	if !near(minX, bi.x-pxToPt(10)) || !near(minY, bi.y-pxToPt(10)) ||
		!near(maxR, bi.x+bi.w+pxToPt(10)) || !near(maxB, bi.y+bi.height+pxToPt(10)) {
		t.Errorf("border-image painted union = (%.4fpt, %.4fpt)-(%.4fpt, %.4fpt), want the 120x80px border box outset by 10px each side",
			minX, minY, maxR, maxB)
	}
}
