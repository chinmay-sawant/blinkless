package layout

import "testing"

// break-inside:avoid in balanced multicol: gap pin.
//
// Chrome 143.0.7499.40 keeps a break-inside:avoid box whole in one column:
// when the box would span the balanced column break it is pushed whole to
// the next column. This engine has no such push at this seam, so the avoid
// box stays where the height-only balancer puts it and paints identical
// geometry to the unstyled document. Never failing; pins the gap until the
// seam can join the avoid flag with the straddle test.
//
// Why the seam cannot do it today (all in internal/layout/multicol.go):
//   - advanceMulticolColumn (multicol.go:767-783) is a package-level free
//     function: it sees column heights, the item height, maxColH and the
//     balance target, but it has no engine receiver, and *html.Node carries
//     no style (internal/html/html.go:32-49). The cascade is only reachable
//     through engine.stylePtr (internal/layout/layout.go:1293), so the free
//     function cannot test PageBreakInside.
//   - multicolForcedColumnBreak (multicol.go:794-806) does receive the item
//     style (break-inside:avoid lands on PageBreakInside via
//     internal/layout/style_properties.go:1706-1718) but it receives no item
//     height, target, or maxColH, so it cannot test "would straddle".
//   - Joining the two needs the call at multicol.go:720-721 inside
//     placeMulticolLine (multicol.go:698-750), which is owned by a sibling
//     agent this wave and out of bounds for this change.
//
// There is also no "auto splitting" baseline for element items to contrast
// against: placeMulticolLine builds each item whole exactly once
// (multicol.go:736-742). Only the lone anonymous text strip fragments, by
// height bands (multicol.go:467-518), and its synthetic style carries no
// break-inside. So an avoid box can no more split than an auto box can; the
// missing piece is only which column a straddler lands in.
func TestBreakInsideAvoidBalancedStraddleGap(t *testing.T) {
	t.Parallel()

	doc := func(extra string) string {
		k2Style := "height:40px"
		if extra != "" {
			k2Style += ";" + extra
		}

		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="mc" style="width:300px;column-count:2;column-gap:20px">` +
			`<div id="k1" style="height:40px"></div>` +
			`<div id="k2" style="` + k2Style + `"></div>` +
			`<div id="k3" style="height:40px"></div>` +
			`</div></body></html>`
	}

	plain := layoutHTML(t, doc(""))
	avoid := layoutHTML(t, doc("break-inside:avoid"))

	// Used box geometry of the avoid item: whole in column 1 under #k1.
	// Total 120px over 2 columns balances at 60px, so #k2 spans 40..80px and
	// straddles the break. Chrome 143.0.7499.40 would push it whole to column
	// 2 (x offset 160px, top-aligned with #k1); the engine keeps it in column
	// 1 because the midpoint rule (multicol.go:779) does not fire.
	avoidK1 := boxByID(t, avoid, "k1")
	avoidK2 := boxByID(t, avoid, "k2")
	avoidK3 := boxByID(t, avoid, "k3")

	if !near(avoidK2.x-avoidK1.x, 0) {
		t.Errorf("avoid #k2 x offset = %.4fpt (%.2fpx), want 0 (kept in column 1)",
			avoidK2.x-avoidK1.x, (avoidK2.x-avoidK1.x)/ptPerCSSPx)
	}

	if !near(avoidK2.y-avoidK1.y, pxToPt(40)) {
		t.Errorf("avoid #k2 y offset = %.4fpt (%.2fpx), want 40px (stacked under #k1)",
			avoidK2.y-avoidK1.y, (avoidK2.y-avoidK1.y)/ptPerCSSPx)
	}

	if !near(avoidK2.height, pxToPt(40)) {
		t.Errorf("avoid #k2 height = %.4fpt (%.2fpx), want 40px (kept whole in one column)",
			avoidK2.height, avoidK2.height/ptPerCSSPx)
	}

	// Locks the distribution the balancer chose: columns hold [#k1 #k2] and
	// [#k3], so #k3 sits in column 2 (140px column + 20px gap) at column top.
	if !near(avoidK3.x-avoidK1.x, pxToPt(160)) {
		t.Errorf("avoid #k3 x offset = %.4fpt (%.2fpx), want 160px (column 2)",
			avoidK3.x-avoidK1.x, (avoidK3.x-avoidK1.x)/ptPerCSSPx)
	}

	if !near(avoidK3.y, avoidK1.y) {
		t.Errorf("avoid #k3 y = %.4fpt, want column-top %.4fpt",
			avoidK3.y, avoidK1.y)
	}

	// No observable avoid effect anywhere in the drawing list.
	behaviorColumnsAssertSameOps(t, plain, avoid)
}
