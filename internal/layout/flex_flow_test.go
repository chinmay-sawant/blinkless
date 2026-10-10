package layout

import "testing"

// TestFlexFlowDirectionWrapLayout: the flex-flow shorthand must drive real
// layout, not only resolved fields. flex-flow: row-reverse wrap lays the
// first item at the main-end (right) edge and wraps the overflow onto a
// second line. Browser reference: test/chrome/cases/case-09-legacy-flex-flow-orientations.html,
// case-10-legacy-flex-flow.html, and case-21-wpt-flow-row-wrap.html
// (Chrome 143.0.7499.40); testdata/golden/fixture-61-implemented-props-b.html
// row 55.
func TestFlexFlowDirectionWrapLayout(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="row" style="display:flex;flex-flow:row-reverse wrap;width:100pt">`+
		`<div id="a" style="width:40pt;height:10pt"></div>`+
		`<div id="b" style="width:40pt;height:10pt"></div>`+
		`<div id="c" style="width:40pt;height:10pt"></div>`+
		`</div></body></html>`)

	row := boxByID(t, res, "row")
	aBox := boxByID(t, res, "a")
	bBox := boxByID(t, res, "b")
	cBox := boxByID(t, res, "c")

	if !near(aBox.x-row.x, 60) || !near(bBox.x-row.x, 20) {
		t.Errorf("row-reverse first line: a.x-row.x = %.2f, b.x-row.x = %.2f, want 60 and 20",
			aBox.x-row.x, bBox.x-row.x)
	}

	if !near(aBox.y, bBox.y) {
		t.Errorf("a.y = %.2f, b.y = %.2f, want both on the first line", aBox.y, bBox.y)
	}

	if !near(cBox.y-aBox.y, 10) {
		t.Errorf("third item y - first line y = %.2f, want a second 10pt line", cBox.y-aBox.y)
	}

	if !near(cBox.x-row.x, 60) {
		t.Errorf("wrapped row-reverse item x-row.x = %.2f, want 60 (right edge)", cBox.x-row.x)
	}
}
