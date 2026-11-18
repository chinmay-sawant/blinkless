package layout

import "testing"

// break-inside in balanced multicol. Width 300px, column-count 2, column-gap
// 20px, three 40px boxes. Total height 120px balances at 60px, so #k2 runs
// from 40px to 80px and crosses the column break.
//
// break-inside:auto (and a missing declaration) leaves #k2 in column 1.
// break-inside:avoid and break-inside:avoid-column move that box whole to
// column 2: x offset 160px (140px column plus 20px gap), same y as #k1,
// height 40px. #k3 then sits under #k2 in column 2. A box that already fits
// one column is covered by TestFragmentAssessBreakInsideAvoidMulticolNoEffect.
func TestBreakInsideAvoidBalancedStraddleGap(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, breakInsideStraddleDoc(""))
	auto := layoutHTML(t, breakInsideStraddleDoc("break-inside:auto"))
	avoid := layoutHTML(t, breakInsideStraddleDoc("break-inside:avoid"))

	assertBreakInsideStaysInColumn1(t, "auto omitted", plain)
	assertBreakInsideStaysInColumn1(t, "auto", auto)
	behaviorColumnsAssertSameOps(t, plain, auto)

	assertBreakInsidePushedToColumn2(t, avoid)

	if got := boxByID(t, avoid, "k2").style.PageBreakInside; got != avoidKeyword {
		t.Fatalf("break-inside:avoid used = %q, want avoid", got)
	}

	_, keep := PageBreaks(avoid, 0)
	k2 := boxByID(t, avoid, "k2")
	if len(keep) != 1 || !near(keep[0][0], k2.y) || !near(keep[0][1], k2.y+k2.height) {
		t.Fatalf("avoid keep = %v, want one range [%v %v]", keep, k2.y, k2.y+k2.height)
	}

	assertBreakInsideOpsDiffer(t, plain, avoid)
}

// TestBehaviorBreakInsideAvoidColumnPushesStraddle is the same column move
// for break-inside:avoid-column. The stored string is avoid-column, not
// avoid, so PageBreaks does not add a page keep. avoid-page still stores
// avoid. avoid-region is dropped. A later bad token does not replace an
// earlier avoid.
func TestBehaviorBreakInsideAvoidColumnPushesStraddle(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, breakInsideStraddleDoc(""))
	column := layoutHTML(t, breakInsideStraddleDoc("break-inside:avoid-column"))

	assertBreakInsidePushedToColumn2(t, column)
	assertBreakInsideOpsDiffer(t, plain, column)

	k2 := boxByID(t, column, "k2")
	if k2.style.PageBreakInside != avoidColumnValue {
		t.Fatalf("PageBreakInside = %q, want avoid-column", k2.style.PageBreakInside)
	}

	if _, keep := PageBreaks(column, 0); len(keep) != 0 {
		t.Fatalf("avoid-column keep = %v, want none", keep)
	}

	stored := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="kept" style="break-inside:avoid;break-inside:nope"></div>`+
		`<div id="region" style="break-inside:avoid-region"></div>`+
		`<div id="regionAfter" style="break-inside:avoid;break-inside:avoid-region"></div>`+
		`<div id="page" style="break-inside:avoid-page"></div>`+
		`<div id="cleared" style="break-inside:avoid;break-inside:auto"></div>`+
		`</body></html>`)

	if got := boxByID(t, stored, "kept").style.PageBreakInside; got != avoidKeyword {
		t.Fatalf("bad token replaced avoid with %q", got)
	}

	if got := boxByID(t, stored, "region").style.PageBreakInside; got != "" {
		t.Fatalf("avoid-region stored %q, want empty", got)
	}

	if got := boxByID(t, stored, "regionAfter").style.PageBreakInside; got != avoidKeyword {
		t.Fatalf("avoid-region replaced avoid with %q", got)
	}

	if got := boxByID(t, stored, "page").style.PageBreakInside; got != avoidKeyword {
		t.Fatalf("avoid-page stored %q, want avoid", got)
	}

	if got := boxByID(t, stored, "cleared").style.PageBreakInside; got != "" {
		t.Fatalf("auto stored %q, want empty", got)
	}

	var direct ResolvedStyle
	if !applyBreakInsideProps(&direct, avoidKeyword) {
		t.Fatal("apply avoid returned false")
	}

	if applyBreakInsideProps(&direct, "not-a-token") {
		t.Fatal("bad token was accepted")
	}

	if direct.PageBreakInside != avoidKeyword {
		t.Fatalf("apply bad token left %q, want avoid", direct.PageBreakInside)
	}

	if applyBreakInsideProps(&direct, "avoid-region") {
		t.Fatal("avoid-region was accepted")
	}

	if direct.PageBreakInside != avoidKeyword {
		t.Fatalf("apply avoid-region left %q, want avoid", direct.PageBreakInside)
	}
}

func breakInsideStraddleDoc(extra string) string {
	k2Style := "height:40px;background:#c2410c"
	if extra != "" {
		k2Style += ";" + extra
	}

	return `<html style="margin:0"><body style="margin:0">` +
		`<div id="mc" style="width:300px;column-count:2;column-gap:20px">` +
		`<div id="k1" style="height:40px;background:#d6d3d1"></div>` +
		`<div id="k2" style="` + k2Style + `"></div>` +
		`<div id="k3" style="height:40px;background:#a8a29e"></div>` +
		`</div></body></html>`
}

func assertBreakInsideStaysInColumn1(t *testing.T, name string, res *Result) {
	t.Helper()

	k1 := boxByID(t, res, "k1")
	k2 := boxByID(t, res, "k2")
	k3 := boxByID(t, res, "k3")

	if !near(k2.x-k1.x, 0) {
		t.Errorf("%s #k2 x offset = %.2fpx, want 0 (column 1)", name, (k2.x-k1.x)/ptPerCSSPx)
	}

	if !near(k2.y-k1.y, pxToPt(40)) {
		t.Errorf("%s #k2 y offset = %.2fpx, want 40px", name, (k2.y-k1.y)/ptPerCSSPx)
	}

	if !near(k2.height, pxToPt(40)) {
		t.Errorf("%s #k2 height = %.2fpx, want 40px", name, k2.height/ptPerCSSPx)
	}

	if !near(k3.x-k1.x, pxToPt(160)) {
		t.Errorf("%s #k3 x offset = %.2fpx, want 160px (column 2)", name, (k3.x-k1.x)/ptPerCSSPx)
	}

	if !near(k3.y, k1.y) {
		t.Errorf("%s #k3 y = %.4fpt, want column-top %.4fpt", name, k3.y, k1.y)
	}
}

func assertBreakInsidePushedToColumn2(t *testing.T, res *Result) {
	t.Helper()

	k1 := boxByID(t, res, "k1")
	k2 := boxByID(t, res, "k2")
	k3 := boxByID(t, res, "k3")

	if !near(k2.x-k1.x, pxToPt(160)) {
		t.Errorf("#k2 x offset = %.2fpx, want 160px (column 2)", (k2.x-k1.x)/ptPerCSSPx)
	}

	if !near(k2.y, k1.y) {
		t.Errorf("#k2 y = %.4fpt, want same as #k1 %.4fpt", k2.y, k1.y)
	}

	if !near(k2.height, pxToPt(40)) {
		t.Errorf("#k2 height = %.2fpx, want 40px", k2.height/ptPerCSSPx)
	}

	// #k2 took the top of column 2, so #k3 stacks under it there.
	if !near(k3.x-k1.x, pxToPt(160)) {
		t.Errorf("#k3 x offset = %.2fpx, want 160px (column 2, under #k2)", (k3.x-k1.x)/ptPerCSSPx)
	}

	if !near(k3.y-k1.y, pxToPt(40)) {
		t.Errorf("#k3 y offset = %.2fpx, want 40px under #k2", (k3.y-k1.y)/ptPerCSSPx)
	}

	if !near(k3.height, pxToPt(40)) {
		t.Errorf("#k3 height = %.2fpx, want 40px", k3.height/ptPerCSSPx)
	}
}

func assertBreakInsideOpsDiffer(t *testing.T, first, second *Result) {
	t.Helper()

	if len(first.Ops) != len(second.Ops) {
		return
	}

	for i := range first.Ops {
		x, y := first.Ops[i], second.Ops[i]
		if x.Kind != y.Kind || x.Text != y.Text ||
			!near(x.X, y.X) || !near(x.Y, y.Y) ||
			!near(x.W, y.W) || !near(x.H, y.H) {
			return
		}
	}

	t.Fatal("ops match the auto document, want the straddling box in column 2")
}
