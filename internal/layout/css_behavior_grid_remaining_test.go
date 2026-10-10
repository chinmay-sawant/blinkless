package layout

import "testing"

// This file covers the remaining grid behavior properties: every property
// below is asserted through a USED value (a measured border-box size or a
// measured offset between two boxes), never through a stored style string.
// The engine lays out in points and 1 CSS pixel = 0.75pt (ptPerCSSPx in
// css_review_02_test.go), so every expectation is written in CSS pixels and
// converted with pxToPt before comparing, which is how Chrome reports it.
// Reference browser for every case: Chrome 143.0.7499.40. The shared helpers
// behaviorFlexGridCheckUsedSize and behaviorFlexGridCheckOffset from
// css_behavior_flex_grid_test.go are reused directly.

// TestBehaviorAlignContentCenterNoOp is align-content: the grid engine does
// not distribute free block space between tracks (grid.go has no AlignContent
// reader), so align-content:center is a no-op and the rows stay packed at the
// start. GAP: Chrome 143.0.7499.40 centers the two 50px rows in the 200px
// container (first row at y 50); the engine keeps the first row at y 0.
func TestBehaviorAlignContentCenterNoOp(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:200px;height:200px;`+
		`grid-template-columns:200px;grid-template-rows:50px 50px;align-content:center">`+
		`<div id="a" style="height:50px"></div><div id="b" style="height:50px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckOffset(t, res, "a", "g", 0, 0)
	behaviorFlexGridCheckOffset(t, res, "b", "g", 0, 50)
}

// TestBehaviorJustifyItemsCenterUsedOffset is justify-items: the 40px items
// center in their 100px tracks with a 30px inline offset.
// Reference: Chrome 143.0.7499.40, tracks 100px + 100px, items at x 30, 130.
func TestBehaviorJustifyItemsCenterUsedOffset(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:200px;grid-template-columns:100px 100px;justify-items:center">`+
		`<div id="a" style="width:40px;height:40px"></div><div id="b" style="width:40px;height:40px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckUsedSize(t, res, "a", 40, 40)
	behaviorFlexGridCheckOffset(t, res, "a", "g", 30, 0)
	behaviorFlexGridCheckOffset(t, res, "b", "g", 130, 0)
}

// TestBehaviorJustifySelfCenterUsedOffset is justify-self: only #b centers in
// its 100px track (30px inline offset) while #a keeps the stretch default.
// Reference: Chrome 143.0.7499.40, second track starts at x 100, item at x 130.
func TestBehaviorJustifySelfCenterUsedOffset(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:200px;grid-template-columns:100px 100px">`+
		`<div id="a" style="height:40px"></div><div id="b" style="width:40px;height:40px;justify-self:center"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckUsedSize(t, res, "a", 100, 40)
	behaviorFlexGridCheckOffset(t, res, "b", "g", 130, 0)
}

// TestBehaviorGridTemplateAreasUsedPlacement is grid-template-areas: the named
// areas place #a in column 1 and #b in column 2 of the 100px + 100px tracks.
// Reference: Chrome 143.0.7499.40, areas 'a b', items at x 0 and x 100.
func TestBehaviorGridTemplateAreasUsedPlacement(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:200px;grid-template-columns:100px 100px;grid-template-areas:'a b'">`+
		`<div id="pa" style="grid-area:a;height:40px"></div><div id="pb" style="grid-area:b;height:40px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckOffset(t, res, "pa", "g", 0, 0)
	behaviorFlexGridCheckOffset(t, res, "pb", "g", 100, 0)
	behaviorFlexGridCheckUsedSize(t, res, "pb", 100, 40)
}

// TestBehaviorGridAutoRowsUsedHeight is grid-auto-rows: with no template rows
// the two implicit rows size to the fixed 50px auto track.
// Reference: Chrome 143.0.7499.40, auto rows 50px, second item at y 50.
func TestBehaviorGridAutoRowsUsedHeight(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:100px;grid-template-columns:100px;grid-auto-rows:50px">`+
		`<div id="a"></div><div id="b"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckUsedSize(t, res, "a", 100, 50)
	behaviorFlexGridCheckOffset(t, res, "b", "g", 0, 50)
	behaviorFlexGridCheckUsedSize(t, res, "b", 100, 50)
}

// TestBehaviorGridColumnSpanUsedWidth is grid-column: span 2 stretches #a over
// both 100px tracks for a used width of 200px.
// Reference: Chrome 143.0.7499.40, two 100px tracks, spanning item 200px wide.
func TestBehaviorGridColumnSpanUsedWidth(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:200px;grid-template-columns:100px 100px;gap:0">`+
		`<div id="a" style="grid-column:span 2;height:40px"></div><div id="b" style="height:40px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckUsedSize(t, res, "a", 200, 40)
	behaviorFlexGridCheckOffset(t, res, "a", "g", 0, 0)
}

// TestBehaviorGridColumnStartUsedOffset is grid-column-start: #a pins to line 2
// so it lays out in the second 100px track at x 100.
// Reference: Chrome 143.0.7499.40, tracks 100px + 100px, pinned item at x 100.
func TestBehaviorGridColumnStartUsedOffset(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:200px;grid-template-columns:100px 100px">`+
		`<div id="a" style="grid-column-start:2;height:40px"></div><div id="b" style="height:40px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckOffset(t, res, "a", "g", 100, 0)
	behaviorFlexGridCheckUsedSize(t, res, "a", 100, 40)
}

// TestBehaviorGridColumnEndUsedWidth is grid-column-end: lines 1 to 3 span both
// 100px tracks, so #a lays out 200px wide at x 0.
// Reference: Chrome 143.0.7499.40, 1 / 3 over 100px + 100px tracks = 200px.
func TestBehaviorGridColumnEndUsedWidth(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:200px;grid-template-columns:100px 100px;gap:0">`+
		`<div id="a" style="grid-column-start:1;grid-column-end:3;height:40px"></div>`+
		`<div id="b" style="height:40px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckUsedSize(t, res, "a", 200, 40)
	behaviorFlexGridCheckOffset(t, res, "a", "g", 0, 0)
}
