package layout

import "testing"

// TestSingleColumnFastPathUsesTrackWidth checks that a one-column grid builds
// its item at the resolved track width, not at the container content width.
func TestSingleColumnFastPathUsesTrackWidth(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:100px;grid-template-columns:40px">`+
		`<div id="item" style="height:20px;background:#c00"></div>`+
		`</div></body></html>`)

	item := boxByID(t, res, "item")
	if !near(item.w, pxToPt(40)) {
		t.Fatalf("item width = %.2fpx, want 40px", item.w/ptPerCSSPx)
	}
}

// TestSingleColumnFastPathRejectsExtraColumn checks that a grid with a second
// track falls back to the general path and keeps the first track width.
func TestSingleColumnFastPathRejectsExtraColumn(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:100px;grid-template-columns:40px 60px">`+
		`<div id="item" style="height:20px;background:#c00"></div>`+
		`</div></body></html>`)

	item := boxByID(t, res, "item")
	if !near(item.w, pxToPt(40)) {
		t.Fatalf("item width = %.2fpx, want 40px", item.w/ptPerCSSPx)
	}
}

// TestSingleColumnFastPathPaintsRowRules checks that the one-pass path still
// paints row-rule lines in the gaps of an auto-row single-column grid.
func TestSingleColumnFastPathPaintsRowRules(t *testing.T) {
	t.Parallel()

	source := `<html style="margin:0"><body style="margin:0">` +
		`<div id="g" style="display:grid;width:100px;grid-template-columns:100px;` +
		`row-gap:20px;row-rule:3px solid #c00">` +
		`<div id="a" style="height:50px"></div>` +
		`<div id="b" style="height:50px"></div>` +
		`</div></body></html>`

	res := layoutHTML(t, source)
	centerY := rowRuleGapCenterY(t, res)
	matched := rowRuleRedStrokes(res, pxToPt(3))

	if len(matched) != 1 {
		t.Fatalf("red 3px horizontal lines = %d, want 1", len(matched))
	}

	if !near(matched[0].Y, centerY) {
		t.Fatalf("rule y = %.2fpx, want gap center %.2fpx", matched[0].Y/ptPerCSSPx, centerY/ptPerCSSPx)
	}
}
