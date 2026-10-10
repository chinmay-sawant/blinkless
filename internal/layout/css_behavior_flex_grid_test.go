package layout

import "testing"

// This file holds the flex and grid behavior tests: every property below is
// asserted through a USED value (a measured border-box size or a measured
// offset between two boxes), never through a stored style string. The engine
// lays out in points and 1 CSS pixel = 0.75pt (ptPerCSSPx in
// css_review_02_test.go), so every expectation is written in CSS pixels and
// converted with pxToPt before comparing, which is how Chrome reports it.
// Reference browser for every case: Chrome 143.0.7499.40.

// behaviorFlexGridCheckUsedSize asserts the used border-box size of one
// element. wantWidthPx / wantHeightPx are CSS pixels.
func behaviorFlexGridCheckUsedSize(t *testing.T, res *Result, elementID string, wantWidthPx, wantHeightPx float64) {
	t.Helper()

	elementBox := boxByID(t, res, elementID)

	if !near(elementBox.w, pxToPt(wantWidthPx)) || !near(elementBox.height, pxToPt(wantHeightPx)) {
		t.Errorf("#%s used border box = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want %.2fpx x %.2fpx",
			elementID, elementBox.w, elementBox.height,
			elementBox.w/ptPerCSSPx, elementBox.height/ptPerCSSPx,
			wantWidthPx, wantHeightPx)
	}
}

// behaviorFlexGridCheckOffset asserts the used position of #innerID relative
// to #refID. wantDxPx / wantDyPx are CSS pixels.
func behaviorFlexGridCheckOffset(t *testing.T, res *Result, innerID, refID string, wantDxPx, wantDyPx float64) {
	t.Helper()

	inner := boxByID(t, res, innerID)
	ref := boxByID(t, res, refID)

	if !near(inner.x-ref.x, pxToPt(wantDxPx)) || !near(inner.y-ref.y, pxToPt(wantDyPx)) {
		t.Errorf("#%s used offset from #%s = (%.4fpt, %.4fpt) = (%.2fpx, %.2fpx), want (%.2fpx, %.2fpx)",
			innerID, refID, inner.x-ref.x, inner.y-ref.y,
			(inner.x-ref.x)/ptPerCSSPx, (inner.y-ref.y)/ptPerCSSPx,
			wantDxPx, wantDyPx)
	}
}

// TestBehaviorFlexDirectionRowUsedOrder is flex-direction: the default row
// axis lays items out left to right in one line.
// Reference: Chrome 143.0.7499.40, 300px row, 60px + 50px items at x 0, 60.
func TestBehaviorFlexDirectionRowUsedOrder(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="row" style="display:flex;width:300px">`+
		`<div id="a" style="width:60px;height:40px"></div>`+
		`<div id="b" style="width:50px;height:40px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckUsedSize(t, res, "a", 60, 40)
	behaviorFlexGridCheckUsedSize(t, res, "b", 50, 40)
	behaviorFlexGridCheckOffset(t, res, "a", "row", 0, 0)
	behaviorFlexGridCheckOffset(t, res, "b", "row", 60, 0)
}

// TestBehaviorFlexDirectionColumnUsedStack is flex-direction: column stacks
// items top to bottom in one column.
// Reference: Chrome 143.0.7499.40, 200px column, two 40px items at y 0, 40.
func TestBehaviorFlexDirectionColumnUsedStack(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="col" style="display:flex;flex-direction:column;width:200px">`+
		`<div id="a" style="height:40px"></div>`+
		`<div id="b" style="height:40px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckUsedSize(t, res, "a", 200, 40)
	behaviorFlexGridCheckUsedSize(t, res, "b", 200, 40)
	behaviorFlexGridCheckOffset(t, res, "a", "col", 0, 0)
	behaviorFlexGridCheckOffset(t, res, "b", "col", 0, 40)
}

// TestBehaviorFlexWrapWrapUsedPosition is flex-wrap: with a 120px container
// and three 50px items the third item wraps to a second line.
// Reference: Chrome 143.0.7499.40, items at (0,0), (50,0), (0,30).
func TestBehaviorFlexWrapWrapUsedPosition(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="row" style="display:flex;flex-wrap:wrap;width:120px">`+
		`<div id="a" style="width:50px;height:30px"></div>`+
		`<div id="b" style="width:50px;height:30px"></div>`+
		`<div id="c" style="width:50px;height:30px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckOffset(t, res, "a", "row", 0, 0)
	behaviorFlexGridCheckOffset(t, res, "b", "row", 50, 0)
	behaviorFlexGridCheckOffset(t, res, "c", "row", 0, 30)
}

// TestBehaviorFlexGrowWeightedUsedWidths is flex-grow: with zero bases the
// 300px of free space splits 1:2, so the items lay out at 100px and 200px.
// Reference: Chrome 143.0.7499.40, grow 1 vs 2 on a 300px row.
func TestBehaviorFlexGrowWeightedUsedWidths(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="row" style="display:flex;width:300px">`+
		`<div id="a" style="flex-grow:1;flex-basis:0;height:40px"></div>`+
		`<div id="b" style="flex-grow:2;flex-basis:0;height:40px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckUsedSize(t, res, "a", 100, 40)
	behaviorFlexGridCheckUsedSize(t, res, "b", 200, 40)
	behaviorFlexGridCheckOffset(t, res, "b", "row", 100, 0)
}

// TestBehaviorFlexShrinkNoShrinkUsedWidth is flex-shrink: in a 150px row with
// two 100px items the shrink:0 item keeps 100px and the shrink:1 item absorbs
// the 50px overflow, laying out at 50px.
// Reference: Chrome 143.0.7499.40, items at widths 100px and 50px.
func TestBehaviorFlexShrinkNoShrinkUsedWidth(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="row" style="display:flex;width:150px">`+
		`<div id="a" style="width:100px;flex-shrink:0;height:40px"></div>`+
		`<div id="b" style="width:100px;flex-shrink:1;height:40px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckUsedSize(t, res, "a", 100, 40)
	behaviorFlexGridCheckUsedSize(t, res, "b", 50, 40)
	behaviorFlexGridCheckOffset(t, res, "b", "row", 100, 0)
}

// TestBehaviorFlexBasisUsedWidth is flex-basis: the item's main size comes
// from its 120px basis, not from content.
// Reference: Chrome 143.0.7499.40, 400px row, basis item at 120px wide.
func TestBehaviorFlexBasisUsedWidth(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="row" style="display:flex;width:400px">`+
		`<div id="a" style="flex-basis:120px;flex-grow:0;height:40px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckUsedSize(t, res, "a", 120, 40)
	behaviorFlexGridCheckOffset(t, res, "a", "row", 0, 0)
}

// TestBehaviorJustifyContentCenterUsedOffset is justify-content: a lone 100px
// item centers in a 300px row at x 100.
// Reference: Chrome 143.0.7499.40, (300-100)/2 = 100px offset.
func TestBehaviorJustifyContentCenterUsedOffset(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="row" style="display:flex;width:300px;justify-content:center">`+
		`<div id="a" style="width:100px;height:40px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckOffset(t, res, "a", "row", 100, 0)
}

// TestBehaviorAlignSelfEndUsedOffset is align-self: flex-end drops the 30px
// item to the bottom of the 100px row, a 70px cross offset.
// Reference: Chrome 143.0.7499.40, 100px row height, item at y 70.
func TestBehaviorAlignSelfEndUsedOffset(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="row" style="display:flex;width:200px;height:100px">`+
		`<div id="a" style="width:50px;height:30px;align-self:flex-end"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckOffset(t, res, "a", "row", 0, 70)
}

// TestBehaviorOrderReorderUsedPosition is order: #b (order 1) paints before
// #a (order 2) despite DOM order, so #b sits at x 0 and #a at x 50.
// Reference: Chrome 143.0.7499.40, 50px item first, 60px item at x 50.
func TestBehaviorOrderReorderUsedPosition(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="row" style="display:flex;width:300px">`+
		`<div id="a" style="width:60px;height:40px;order:2"></div>`+
		`<div id="b" style="width:50px;height:40px;order:1"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckOffset(t, res, "b", "row", 0, 0)
	behaviorFlexGridCheckOffset(t, res, "a", "row", 50, 0)
}

// TestBehaviorGridTemplateColumnsUsedTracks is grid-template-columns: the two
// fixed tracks lay out at 100px and 200px with #b starting at x 100.
// Reference: Chrome 143.0.7499.40, 300px grid, tracks 100px + 200px.
func TestBehaviorGridTemplateColumnsUsedTracks(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:300px;grid-template-columns:100px 200px">`+
		`<div id="a" style="height:40px"></div><div id="b" style="height:40px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckUsedSize(t, res, "a", 100, 40)
	behaviorFlexGridCheckUsedSize(t, res, "b", 200, 40)
	behaviorFlexGridCheckOffset(t, res, "b", "g", 100, 0)
}

// TestBehaviorGridTemplateRowsUsedTracks is grid-template-rows: the rows lay
// out at 50px and 70px, so #b starts 50px below #a.
// Reference: Chrome 143.0.7499.40, rows 50px + 70px.
func TestBehaviorGridTemplateRowsUsedTracks(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:200px;grid-template-columns:200px;grid-template-rows:50px 70px">`+
		`<div id="a" style="height:50px"></div><div id="b" style="height:70px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckOffset(t, res, "a", "g", 0, 0)
	behaviorFlexGridCheckOffset(t, res, "b", "g", 0, 50)
	behaviorFlexGridCheckUsedSize(t, res, "b", 200, 70)
}

// TestBehaviorGridAutoFlowColumnUsedPlacement is grid-auto-flow: column fills
// the first column top to bottom, so #b shares #a's x and sits 20px below.
// Reference: Chrome 143.0.7499.40, 2x2 grid of 40x20px cells, column fill.
func TestBehaviorGridAutoFlowColumnUsedPlacement(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:80px;grid-template-columns:40px 40px;grid-template-rows:20px 20px;grid-auto-flow:column;gap:0">`+
		`<div id="a" style="width:40px;height:20px"></div><div id="b" style="width:40px;height:20px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckOffset(t, res, "a", "g", 0, 0)
	behaviorFlexGridCheckOffset(t, res, "b", "g", 0, 20)
}

// TestBehaviorGridAutoColumnsImplicitUsedWidth is grid-auto-columns: #b is
// placed in implicit column 2, which sizes to the 60px auto track at x 100.
// Reference: Chrome 143.0.7499.40, 100px explicit track + 60px implicit track.
func TestBehaviorGridAutoColumnsImplicitUsedWidth(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:300px;grid-template-columns:100px;grid-auto-columns:60px">`+
		`<div id="a" style="height:40px"></div><div id="b" style="grid-column:2;height:40px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckUsedSize(t, res, "b", 60, 40)
	behaviorFlexGridCheckOffset(t, res, "b", "g", 100, 0)
}

// TestBehaviorGapGridUsedSpacing is gap: the 20px gap pushes the second 100px
// column to x 120.
// Reference: Chrome 143.0.7499.40, 100px tracks with 20px gap.
func TestBehaviorGapGridUsedSpacing(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:220px;grid-template-columns:100px 100px;gap:20px">`+
		`<div id="a" style="height:40px"></div><div id="b" style="height:40px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckOffset(t, res, "b", "g", 120, 0)
}

// TestBehaviorRowGapGridUsedSpacing is row-gap: the 20px row gap puts the
// second 50px row 70px below the first.
// Reference: Chrome 143.0.7499.40, 50px rows with 20px row gap.
func TestBehaviorRowGapGridUsedSpacing(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:100px;grid-template-columns:100px;grid-template-rows:50px 50px;row-gap:20px">`+
		`<div id="a" style="height:50px"></div><div id="b" style="height:50px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckOffset(t, res, "b", "g", 0, 70)
}

// TestBehaviorColumnGapFlexUsedSpacing is column-gap: the 20px gap in a flex
// row puts the second 50px item at x 70.
// Reference: Chrome 143.0.7499.40, 50px items with 20px column gap.
func TestBehaviorColumnGapFlexUsedSpacing(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="row" style="display:flex;width:300px;column-gap:20px">`+
		`<div id="a" style="width:50px;height:40px"></div>`+
		`<div id="b" style="width:50px;height:40px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckOffset(t, res, "a", "row", 0, 0)
	behaviorFlexGridCheckOffset(t, res, "b", "row", 70, 0)
}

// TestBehaviorGridRowSpanUsedHeight is grid-row: span 2 stretches #a over both
// 50px rows for a used height of 100px.
// Reference: Chrome 143.0.7499.40, two 50px rows, item spans both.
func TestBehaviorGridRowSpanUsedHeight(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:100px;grid-template-columns:100px;grid-template-rows:50px 50px;gap:0">`+
		`<div id="a" style="grid-row:span 2"></div><div id="b" style="height:50px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckUsedSize(t, res, "a", 100, 100)
}
