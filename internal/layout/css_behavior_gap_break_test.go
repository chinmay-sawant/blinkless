package layout

import "testing"

// The fixtures here mirror the css-gaps-1 geometry probes run against
// Chrome 155 (chrome-headless-shell): a 3x2 grid, columns at [0,50],
// [70,120], [140,190] and rows at [0,50], [70,120], all in CSS px. The
// column gap centers sit at x=60 and x=130; the row gap center sits at y=60.
// Every expectation below records what Chrome painted.

// gapRulePlainGrid is a 3x2 grid with every track filled. decls is the
// container's gap and rule styles.
func gapRulePlainGrid(decls string) string {
	return `<html style="margin:0"><body style="margin:0">` +
		`<div style="display:grid;width:190px;grid-template-columns:50px 50px 50px;` +
		`grid-template-rows:50px 50px;gap:20px;` + decls + `">` +
		`<div style="grid-column:1/2;grid-row:1/2"></div>` +
		`<div style="grid-column:2/3;grid-row:1/2"></div>` +
		`<div style="grid-column:3/4;grid-row:1/2"></div>` +
		`<div style="grid-column:1/2;grid-row:2/3"></div>` +
		`<div style="grid-column:2/3;grid-row:2/3"></div>` +
		`<div style="grid-column:3/4;grid-row:2/3"></div>` +
		`</div></body></html>`
}

// gapRuleSpanGrid spans column 1 across both rows: a child covering the row
// gap, the css-gaps-1 spanning item case.
func gapRuleSpanGrid(decls string) string {
	return `<html style="margin:0"><body style="margin:0">` +
		`<div style="display:grid;width:190px;grid-template-columns:50px 50px 50px;` +
		`grid-template-rows:50px 50px;gap:20px;` + decls + `">` +
		`<div style="grid-column:1/2;grid-row:1/3"></div>` +
		`<div style="grid-column:2/3;grid-row:1/2"></div>` +
		`<div style="grid-column:3/4;grid-row:1/2"></div>` +
		`<div style="grid-column:2/3;grid-row:2/3"></div>` +
		`<div style="grid-column:3/4;grid-row:2/3"></div>` +
		`</div></body></html>`
}

// gapRuleFlankedGrid spans columns 2-3 in both rows: children on opposing
// sides of the row gap that flank the column 2/3 junction.
func gapRuleFlankedGrid(decls string) string {
	return `<html style="margin:0"><body style="margin:0">` +
		`<div style="display:grid;width:190px;grid-template-columns:50px 50px 50px;` +
		`grid-template-rows:50px 50px;gap:20px;` + decls + `">` +
		`<div style="grid-column:1/2;grid-row:1/3"></div>` +
		`<div style="grid-column:2/4;grid-row:1/2"></div>` +
		`<div style="grid-column:2/4;grid-row:2/3"></div>` +
		`</div></body></html>`
}

// checkRowRuleSegments compares the red 4px horizontal rule segments as
// (x, width) pairs in CSS px, in painting order.
func checkRowRuleSegments(t *testing.T, res *Result, want ...[2]float64) {
	t.Helper()

	lines := rowRuleRedStrokes(res, pxToPt(4))
	if len(lines) != len(want) {
		t.Fatalf("row rule segments = %d, want %d", len(lines), len(want))
	}

	for index, paintOp := range lines {
		got := [2]float64{paintOp.X / ptPerCSSPx, paintOp.W / ptPerCSSPx}
		if !near(got[0], want[index][0]) || !near(got[1], want[index][1]) {
			t.Fatalf("row rule segment %d = %v, want %v", index, got, want[index])
		}
	}
}

// checkColumnRuleSegments compares the blue 4px vertical rule segments as
// (x, y, height) triples in CSS px, in painting order.
func checkColumnRuleSegments(t *testing.T, res *Result, want ...[3]float64) {
	t.Helper()

	lines := columnRuleBlueLines(res)

	if len(lines) != len(want) {
		t.Fatalf("column rule segments = %d, want %d", len(lines), len(want))
	}

	for index, paintOp := range lines {
		got := [3]float64{paintOp.X / ptPerCSSPx, paintOp.Y / ptPerCSSPx, paintOp.H / ptPerCSSPx}
		if !near(got[0], want[index][0]) || !near(got[1], want[index][1]) ||
			!near(got[2], want[index][2]) {
			t.Fatalf("column rule segment %d = %v, want %v", index, got, want[index])
		}
	}
}

// columnRuleBlueLines is every blue vertical OpLine in painting order: the
// 4px column rule paints under 0.5pt wide with a positive height.
func columnRuleBlueLines(res *Result) []Op {
	lines := make([]Op, 0)

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpLine && paintOp.W < 0.5 && paintOp.H > 0 && near(paintOp.B, 1) {
			lines = append(lines, paintOp)
		}
	}

	return lines
}

// TestBehaviorRowRuleBreakIntersectionSplitsAtColumnGaps: in a fully
// populated grid, intersection breaks the row rule at both column-gap
// junctions, while the default normal keeps one continuous rule.
func TestBehaviorRowRuleBreakIntersectionSplitsAtColumnGaps(t *testing.T) {
	t.Parallel()

	split := layoutHTML(t, gapRulePlainGrid(
		"row-rule:4px solid #c00;row-rule-break:intersection"))
	checkRowRuleSegments(t, split,
		[2]float64{0, 50}, [2]float64{70, 50}, [2]float64{140, 50})

	joined := layoutHTML(t, gapRulePlainGrid("row-rule:4px solid #c00"))
	checkRowRuleSegments(t, joined, [2]float64{0, 190})
}

// TestBehaviorRowRuleSpanDropsCoveredPair: the child spanning both rows
// over column 1 makes the pair covering it discontiguous, so normal starts
// the rule at the column 1/2 junction exit and intersection also splits at
// the unflanked column 2/3 junction.
func TestBehaviorRowRuleSpanDropsCoveredPair(t *testing.T) {
	t.Parallel()

	normal := layoutHTML(t, gapRuleSpanGrid("row-rule:4px solid #c00"))
	checkRowRuleSegments(t, normal, [2]float64{70, 120})

	intersection := layoutHTML(t, gapRuleSpanGrid(
		"row-rule:4px solid #c00;row-rule-break:intersection"))
	checkRowRuleSegments(t, intersection, [2]float64{70, 50}, [2]float64{140, 50})
}

// TestBehaviorRowRuleBreakNonePaintsThroughJunctions: none paints one
// segment from bound to bound, ignoring the spanning child and the
// junctions; cap insets still shorten its ends.
func TestBehaviorRowRuleBreakNonePaintsThroughJunctions(t *testing.T) {
	t.Parallel()

	full := layoutHTML(t, gapRuleSpanGrid("row-rule:4px solid #c00;row-rule-break:none"))
	checkRowRuleSegments(t, full, [2]float64{0, 190})

	inset := layoutHTML(t, gapRuleSpanGrid(
		"row-rule:4px solid #c00;row-rule-break:none;row-rule-inset-cap:10px"))
	checkRowRuleSegments(t, inset, [2]float64{10, 170})
}

// TestBehaviorRowRuleJunctionAndCapInsets: the rule start sits at the
// column 1/2 junction exit, so junction insets move it and cap insets do
// not; the end at the content edge reads the cap insets.
func TestBehaviorRowRuleJunctionAndCapInsets(t *testing.T) {
	t.Parallel()

	junction := layoutHTML(t, gapRuleSpanGrid(
		"row-rule:4px solid #c00;row-rule-inset-junction:10px"))
	checkRowRuleSegments(t, junction, [2]float64{80, 110})

	capInset := layoutHTML(t, gapRuleSpanGrid(
		"row-rule:4px solid #c00;row-rule-inset-cap:10px"))
	checkRowRuleSegments(t, capInset, [2]float64{70, 110})
}

// TestBehaviorRowRuleJunctionOverlapJoin: overlap-join extends the junction
// start backward by half the crossing gap plus half the used crossing rule
// width: 12px with a 4px column rule, 11.5px with the initial medium 3px
// width when no column rule is set.
func TestBehaviorRowRuleJunctionOverlapJoin(t *testing.T) {
	t.Parallel()

	withColumn := layoutHTML(t, gapRuleSpanGrid(
		"row-rule:4px solid #c00;row-rule-inset-junction:overlap-join;column-rule:4px solid #00f"))
	checkRowRuleSegments(t, withColumn, [2]float64{58, 132})

	withoutColumn := layoutHTML(t, gapRuleSpanGrid(
		"row-rule:4px solid #c00;row-rule-inset-junction:overlap-join"))
	checkRowRuleSegments(t, withoutColumn, [2]float64{58.5, 131.5})
}

// TestBehaviorRowRuleIntersectionFlankedMerge: children spanning columns
// 2-3 in both rows flank the column 2/3 junction, so intersection merges
// through it. The rule still starts at the column 1/2 junction exit.
func TestBehaviorRowRuleIntersectionFlankedMerge(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, gapRuleFlankedGrid(
		"row-rule:4px solid #c00;row-rule-break:intersection"))
	checkRowRuleSegments(t, res, [2]float64{70, 120})
}

// TestBehaviorColumnRuleBreakIntersectionSplitsAtRowGaps: the spanning
// child covers the row gap on the left of the row-gap junction but not on
// the right, so intersection splits both column rules at the gap.
func TestBehaviorColumnRuleBreakIntersectionSplitsAtRowGaps(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, gapRuleSpanGrid(
		"column-rule:4px solid #00f;column-rule-break:intersection"))
	checkColumnRuleSegments(t, res,
		[3]float64{60, 0, 50}, [3]float64{60, 70, 50},
		[3]float64{130, 0, 50}, [3]float64{130, 70, 50})
}

// TestBehaviorColumnRuleJunctionAndCapInsets: junction insets move the
// split segments away from the row-gap junction; cap insets shorten the
// continuous rule at the content edges instead.
func TestBehaviorColumnRuleJunctionAndCapInsets(t *testing.T) {
	t.Parallel()

	junction := layoutHTML(t, gapRuleSpanGrid(
		"column-rule:4px solid #00f;column-rule-break:intersection;"+
			"column-rule-inset-junction:10px"))
	checkColumnRuleSegments(t, junction,
		[3]float64{60, 0, 40}, [3]float64{60, 80, 40},
		[3]float64{130, 0, 40}, [3]float64{130, 80, 40})

	capInset := layoutHTML(t, gapRuleSpanGrid(
		"column-rule:4px solid #00f;column-rule-inset-cap:10px"))
	checkColumnRuleSegments(t, capInset,
		[3]float64{60, 10, 100}, [3]float64{130, 10, 100})
}
