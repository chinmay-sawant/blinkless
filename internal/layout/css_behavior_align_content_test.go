package layout

import "testing"

// Grid align-content distributes free block-axis space across row tracks.
// Fixture for every case: 200px container, two fixed 50px rows, no gap, so
// tracks use 100px and 100px is free. Reference: Chrome 143.0.7499.40.
// Helpers behaviorFlexGridCheckOffset / behaviorFlexGridCheckUsedSize come
// from css_behavior_flex_grid_test.go; expectations are CSS pixels.

// TestBehaviorAlignContentGridStartUsedOffset is align-content:start: tracks
// pack at the start, first row at y 0, second at y 50.
// Reference: Chrome 143.0.7499.40, rows at y 0 and y 50.
func TestBehaviorAlignContentGridStartUsedOffset(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:200px;height:200px;`+
		`grid-template-columns:200px;grid-template-rows:50px 50px;align-content:start">`+
		`<div id="a" style="height:50px"></div><div id="b" style="height:50px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckOffset(t, res, "a", "g", 0, 0)
	behaviorFlexGridCheckOffset(t, res, "b", "g", 0, 50)
}

// TestBehaviorAlignContentGridCenterUsedOffset is align-content:center: the
// 100px of free space splits evenly, first row at y 50, second at y 100.
// Reference: Chrome 143.0.7499.40, (200-100)/2 = 50px leading offset.
func TestBehaviorAlignContentGridCenterUsedOffset(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:200px;height:200px;`+
		`grid-template-columns:200px;grid-template-rows:50px 50px;align-content:center">`+
		`<div id="a" style="height:50px"></div><div id="b" style="height:50px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckOffset(t, res, "a", "g", 0, 50)
	behaviorFlexGridCheckOffset(t, res, "b", "g", 0, 100)
}

// TestBehaviorAlignContentGridEndUsedOffset is align-content:end: all free
// space goes before the tracks, first row at y 100, second at y 150.
// Reference: Chrome 143.0.7499.40, 100px leading offset.
func TestBehaviorAlignContentGridEndUsedOffset(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:200px;height:200px;`+
		`grid-template-columns:200px;grid-template-rows:50px 50px;align-content:end">`+
		`<div id="a" style="height:50px"></div><div id="b" style="height:50px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckOffset(t, res, "a", "g", 0, 100)
	behaviorFlexGridCheckOffset(t, res, "b", "g", 0, 150)
}

// TestBehaviorAlignContentGridBetweenUsedOffset is align-content:space-between:
// first row pins at y 0 and the 100px of free space sits between the tracks,
// so the second row starts at y 150.
// Reference: Chrome 143.0.7499.40, rows at y 0 and y 150.
func TestBehaviorAlignContentGridBetweenUsedOffset(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:200px;height:200px;`+
		`grid-template-columns:200px;grid-template-rows:50px 50px;align-content:space-between">`+
		`<div id="a" style="height:50px"></div><div id="b" style="height:50px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckOffset(t, res, "a", "g", 0, 0)
	behaviorFlexGridCheckOffset(t, res, "b", "g", 0, 150)
}

// TestBehaviorAlignContentGridAroundUsedOffset is align-content:space-around:
// free/4 = 25px pads each edge and 50px sits between the tracks, so rows
// start at y 25 and y 125.
// Reference: Chrome 143.0.7499.40, rows at y 25 and y 125.
func TestBehaviorAlignContentGridAroundUsedOffset(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:200px;height:200px;`+
		`grid-template-columns:200px;grid-template-rows:50px 50px;align-content:space-around">`+
		`<div id="a" style="height:50px"></div><div id="b" style="height:50px"></div>`+
		`</div></body></html>`)

	behaviorFlexGridCheckOffset(t, res, "a", "g", 0, 25)
	behaviorFlexGridCheckOffset(t, res, "b", "g", 0, 125)
}
