package layout

import "testing"

// TestFlexPlaceContentDistributes: place-content expands to align-content plus
// justify-content. In a single-line flex row the justify half must match the
// Chrome 143.0.7499.40 measurements already recorded by
// TestFlexDistributedJustifyKeepsGap (internal/layout/css_review_02_test.go:40-78);
// a wrapped row with a definite height centers its lines through the align
// half (internal/layout/flex.go:513-549). Grid content distribution is not a
// consumer today, so no grid case is asserted here.
func TestFlexPlaceContentDistributes(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="row" style="display:flex;width:300pt;gap:15pt;place-content:center space-between">`+
		`<div id="a" style="width:37.5pt;height:10pt"></div>`+
		`<div id="b" style="width:37.5pt;height:10pt"></div>`+
		`<div id="c" style="width:37.5pt;height:10pt"></div>`+
		`</div></body></html>`)

	for elementID, wantPx := range map[string]float64{"a": 0, "b": 175, "c": 350} {
		itemBox := boxByID(t, res, elementID)
		if !near(itemBox.x, pxToPt(wantPx)) {
			t.Errorf("place-content space-between %s.x = %.4fpt (%.2fpx), want %.2fpx",
				elementID, itemBox.x, itemBox.x/ptPerCSSPx, wantPx)
		}
	}

	wrapped := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="wrap" style="display:flex;flex-wrap:wrap;width:100pt;height:100pt;place-content:center">`+
		`<div id="w1" style="width:40pt;height:10pt"></div>`+
		`<div id="w2" style="width:40pt;height:10pt"></div>`+
		`<div id="w3" style="width:40pt;height:10pt"></div>`+
		`</div></body></html>`)

	wrap := boxByID(t, wrapped, "wrap")
	firstLine := boxByID(t, wrapped, "w1")
	lastLine := boxByID(t, wrapped, "w3")

	// Two 10pt lines in a 100pt container: align-content:center shifts the
	// first line down by (100 - 20) / 2 = 40pt.
	if !near(firstLine.y-wrap.y, 40) {
		t.Errorf("wrapped line 1 y offset = %.2f, want 40 (centered)", firstLine.y-wrap.y)
	}

	if !near(lastLine.y-firstLine.y, 10) {
		t.Errorf("wrapped line gap = %.2f, want one 10pt line", lastLine.y-firstLine.y)
	}
}
