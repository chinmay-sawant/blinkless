package layout

import "testing"

// Widows control in the multicol whole-box distribution loop
// (placeMulticolLine at multicol.go:709-766 with helpers in
// fragment_widows.go). Each child box counts as round(height / line-height)
// lines; when keeping the current item in this column would leave fewer than
// the widows minimum lines for the remaining columns, the break moves one
// item earlier so the next column keeps at least widows lines together.
// Reference browser for every case: Chrome 143.0.7499.40.

// widowsBuildDoc returns a margin-zero 2-column auto-fill multicol container
// with a definite 100px height and five 30px children. Each child sets
// line-height:30px so it counts as exactly one line. Auto fill packs 90px
// (3 items) into the first 100px column by height alone, leaving 1 line for
// the next column. extra holds an additional declaration for #mc.
func widowsBuildDoc(extra string) string {
	mcStyle := "width:300px;column-count:2;column-gap:20px;height:100px;column-fill:auto"
	if extra != "" {
		mcStyle += ";" + extra
	}

	kid := func(id string) string {
		return `<div id="` + id + `" style="height:30px;line-height:30px;margin:0"></div>`
	}

	return `<html style="margin:0"><body style="margin:0">` +
		`<div id="mc" style="` + mcStyle + `">` +
		kid("k1") + kid("k2") + kid("k3") + kid("k4") + kid("k5") +
		`</div></body></html>`
}

// widowsBuildColCounts buckets child ids by used x into first/second column
// counts. Each child is one line, so counts are line counts.
func widowsBuildColCounts(t *testing.T, res *Result, ids ...string) (int, int) {
	t.Helper()

	positions := behaviorColumnsChildXs(t, res, ids...)
	first := positions[0]
	col0, col1 := 0, 0

	for _, xPos := range positions {
		switch {
		case near(xPos, first):
			col0++
		case near(xPos-first, pxToPt(160)):
			col1++
		default:
			t.Fatalf("child x offset = %.4fpt (%.2fpx), want 0 or 160px", xPos-first, (xPos-first)/ptPerCSSPx)
		}
	}

	return col0, col1
}

// widowsBuildAssertCounts asserts the used column line distribution.
func widowsBuildAssertCounts(t *testing.T, res *Result, ids []string, want0, want1 int) (int, int) {
	t.Helper()

	got0, got1 := widowsBuildColCounts(t, res, ids...)
	if got0 != want0 || got1 != want1 {
		t.Errorf("columns = %d-%d lines, want %d-%d", got0, got1, want0, want1)
	}

	return got0, got1
}

// TestBehaviorWidowsKeepsMinLinesInNextColumn is widows in multicol: five
// one-line items in a 100px auto-fill column pack 3-2 by height alone, so the
// next column holds 2 lines and satisfies the initial widows 2. With
// widows:3 the same heights would leave only 2 lines on top of the next
// column, so the break moves one item earlier to 2-3 and the next column
// keeps 3 lines together. Reference: Chrome 143.0.7499.40 keeps the widows
// minimum lines together at a column break.
func TestBehaviorWidowsKeepsMinLinesInNextColumn(t *testing.T) {
	t.Parallel()

	ids := []string{"k1", "k2", "k3", "k4", "k5"}

	plain := layoutHTML(t, widowsBuildDoc(""))
	if got := boxByID(t, plain, "mc").style.Widows; got != 2 {
		t.Fatalf("plain widows used = %d, want initial 2", got)
	}

	_, plainNext := widowsBuildAssertCounts(t, plain, ids, 3, 2)
	if plainNext < 2 {
		t.Errorf("plain next column = %d lines, want >= initial widows 2", plainNext)
	}

	styled := layoutHTML(t, widowsBuildDoc("widows:3"))
	if got := boxByID(t, styled, "mc").style.Widows; got != 3 {
		t.Fatalf("widows used = %d, want 3", got)
	}

	_, styledNext := widowsBuildAssertCounts(t, styled, ids, 2, 3)
	if styledNext < 3 {
		t.Errorf("widows:3 next column = %d lines, want >= 3", styledNext)
	}

	widowsBuildAssertBreakMovesEarlier(t, plain, styled)
}

// widowsBuildAssertBreakMovesEarlier asserts the used break position: #k3
// stays in column 1 by default and moves to column 2 under widows:3.
func widowsBuildAssertBreakMovesEarlier(t *testing.T, plain, styled *Result) {
	t.Helper()

	plainK1 := boxByID(t, plain, "k1")
	if !near(boxByID(t, plain, "k3").x-plainK1.x, 0) {
		t.Errorf("plain #k3 stays in column 1, want x offset 0")
	}

	styledK1 := boxByID(t, styled, "k1")
	styledK3 := boxByID(t, styled, "k3")

	if !near(styledK3.x-styledK1.x, pxToPt(160)) {
		t.Errorf("widows:3 #k3 x offset = %.4fpt (%.2fpx), want 160px (column 2)",
			styledK3.x-styledK1.x, (styledK3.x-styledK1.x)/ptPerCSSPx)
	}
}
