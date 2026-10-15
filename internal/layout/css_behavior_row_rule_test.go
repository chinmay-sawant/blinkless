package layout

import "testing"

// rowRuleTwoRowGrid is a 100px grid with two 50px rows and no borders.
// decls is the container's gap and row-rule style.
func rowRuleTwoRowGrid(decls string) string {
	return `<html style="margin:0"><body style="margin:0">` +
		`<div id="g" style="display:grid;width:100px;grid-template-columns:100px;` +
		`grid-template-rows:50px 50px;` + decls + `">` +
		`<div id="a" style="height:50px"></div>` +
		`<div id="b" style="height:50px"></div>` +
		`</div></body></html>`
}

// rowRuleHorizontalLines is every OpLine that runs across (H == 0, W > 0).
func rowRuleHorizontalLines(res *Result) []Op {
	lines := make([]Op, 0)

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpLine || paintOp.H != 0 || paintOp.W <= 0 {
			continue
		}

		lines = append(lines, paintOp)
	}

	return lines
}

// rowRuleRed is the #c00 used color.
func rowRuleRed(paintOp Op) bool {
	return near(paintOp.R, 0.8) && near(paintOp.G, 0) && near(paintOp.B, 0)
}

// rowRuleGapCenterY is the middle of the gap between #a and #b. Child b
// stays about 70px below child a, so the rule did not move the rows.
func rowRuleGapCenterY(t *testing.T, res *Result) float64 {
	t.Helper()

	firstRow := boxByID(t, res, "a")
	secondRow := boxByID(t, res, "b")

	if !near(secondRow.y-firstRow.y, pxToPt(70)) {
		t.Fatalf("child b is %.2fpx below a, want 70px", (secondRow.y-firstRow.y)/ptPerCSSPx)
	}

	gapTop := firstRow.y + firstRow.height

	return gapTop + (secondRow.y-gapTop)/2
}

func rowRuleRedStrokes(res *Result, widthPt float64) []Op {
	var matched []Op

	for _, paintOp := range rowRuleHorizontalLines(res) {
		if rowRuleRed(paintOp) && near(paintOp.Width, widthPt) {
			matched = append(matched, paintOp)
		}
	}

	return matched
}

// TestBehaviorRowRulePaintedBetweenRows is row-rule on a two-row grid: one
// horizontal red 3px line centered in the 20px gap. Child b stays 70px below a.
func TestBehaviorRowRulePaintedBetweenRows(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, rowRuleTwoRowGrid("row-gap:20px;row-rule:3px solid #c00"))
	centerY := rowRuleGapCenterY(t, res)
	matched := rowRuleRedStrokes(res, pxToPt(3))

	if len(matched) != 1 {
		t.Fatalf("red 3px horizontal lines = %d, want 1", len(matched))
	}

	if matched[0].H != 0 || matched[0].W <= 0 {
		t.Fatalf("rule geometry H=%.3f W=%.3f, want H == 0 and W > 0", matched[0].H, matched[0].W)
	}

	if !near(matched[0].Y, centerY) {
		t.Fatalf("rule y=%.3f, want gap center %.3f", matched[0].Y, centerY)
	}
}

// TestBehaviorRowRuleDashedSegments is row-rule: dashed. More than one
// horizontal segment shares the gap-center y.
func TestBehaviorRowRuleDashedSegments(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, rowRuleTwoRowGrid("row-gap:20px;row-rule:3px dashed #c00"))
	centerY := rowRuleGapCenterY(t, res)

	var segments []Op

	for _, paintOp := range rowRuleHorizontalLines(res) {
		if near(paintOp.Y, centerY) {
			segments = append(segments, paintOp)
		}
	}

	if len(segments) < 2 {
		t.Fatalf("dashed segments at gap center = %d, want more than one", len(segments))
	}
}

// TestBehaviorRowRuleNonePaintsNothing is row-rule-style: none. A width and
// a color are set, and the fixture has no borders, so no horizontal line is painted.
func TestBehaviorRowRuleNonePaintsNothing(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, rowRuleTwoRowGrid(
		"row-gap:20px;row-rule-width:3px;row-rule-style:none;row-rule-color:#c00"))

	if lines := rowRuleHorizontalLines(res); len(lines) != 0 {
		t.Fatalf("horizontal lines = %d, want 0", len(lines))
	}
}

// TestBehaviorRowRuleZeroGapPaintsNothing is row-gap: 0. A solid red rule
// paints no horizontal line.
func TestBehaviorRowRuleZeroGapPaintsNothing(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, rowRuleTwoRowGrid("row-gap:0;row-rule:3px solid #c00"))

	if lines := rowRuleHorizontalLines(res); len(lines) != 0 {
		t.Fatalf("horizontal lines = %d, want 0", len(lines))
	}
}

// TestBehaviorRowRuleWidthUsesFirstItem is row-rule-width: 3px 6px. The
// stroke is the first item, 3px, not 6px.
func TestBehaviorRowRuleWidthUsesFirstItem(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, rowRuleTwoRowGrid(
		"row-gap:20px;row-rule-width:3px 6px;row-rule-style:solid;row-rule-color:#c00"))
	matched := rowRuleRedStrokes(res, pxToPt(3))

	if len(matched) == 0 {
		t.Fatal("missing horizontal red rule at 3px")
	}

	if near(matched[0].Width, pxToPt(6)) {
		t.Fatalf("stroke=%.3f, painter used 6px instead of the first item", matched[0].Width)
	}
}

// TestBehaviorRowRuleWidthCurrentColor is a 6px solid rule with no rule
// color. The stroke is 6px and the color is currentColor black.
func TestBehaviorRowRuleWidthCurrentColor(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, rowRuleTwoRowGrid(
		"row-gap:20px;row-rule-width:6px;row-rule-style:solid"))
	lines := rowRuleHorizontalLines(res)

	if len(lines) == 0 {
		t.Fatal("missing horizontal rule")
	}

	if !near(lines[0].Width, pxToPt(6)) {
		t.Fatalf("stroke=%.3f, want %.3f", lines[0].Width, pxToPt(6))
	}

	if !near(lines[0].R, 0) || !near(lines[0].G, 0) || !near(lines[0].B, 0) {
		t.Fatalf("color=(%.2f, %.2f, %.2f), want black", lines[0].R, lines[0].G, lines[0].B)
	}
}

// TestBehaviorRowRuleFlexWrapPaintsGap is row-rule on a wrapping flex row.
// Two 100px items in a 100px container wrap, so one horizontal red rule is
// painted. The nowrap twin in a 250px container paints no horizontal rule.
func TestBehaviorRowRuleFlexWrapPaintsGap(t *testing.T) {
	t.Parallel()

	wrap := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="row" style="display:flex;flex-wrap:wrap;width:100px;row-gap:20px;row-rule:3px solid #c00">`+
		`<div id="a" style="width:100px;height:40px"></div>`+
		`<div id="b" style="width:100px;height:40px"></div>`+
		`</div></body></html>`)
	matched := rowRuleRedStrokes(wrap, pxToPt(3))

	if len(matched) != 1 {
		t.Fatalf("wrap red 3px horizontal lines = %d, want 1", len(matched))
	}

	nowrap := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="row" style="display:flex;flex-wrap:nowrap;width:250px;row-gap:20px;row-rule:3px solid #c00">`+
		`<div id="a" style="width:100px;height:40px"></div>`+
		`<div id="b" style="width:100px;height:40px"></div>`+
		`</div></body></html>`)

	if lines := rowRuleHorizontalLines(nowrap); len(lines) != 0 {
		t.Fatalf("nowrap horizontal lines = %d, want 0", len(lines))
	}
}

// TestBehaviorRowRuleMulticolSkipped is row-rule on a multicol container.
// Column-wrap row boxes are not painted in this wave.
func TestBehaviorRowRuleMulticolSkipped(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="mc" style="column-count:2;column-gap:20px;width:300px;row-rule:3px solid #c00">`+
		`Multi-column sample text repeated. Multi-column sample text repeated.`+
		`</div></body></html>`)

	if matched := rowRuleRedStrokes(res, pxToPt(3)); len(matched) != 0 {
		t.Fatalf("multicol horizontal red 3px lines = %d, want 0", len(matched))
	}
}

//nolint:wsl // the line list is the assertion subject
func TestBehaviorRowRuleColorBlue(t *testing.T) {
	t.Parallel()

	res := rowRuleGrid(t, "row-gap:20px;row-rule-style:solid;row-rule-width:3px;row-rule-color:#00f")
	lines := rowRuleHorizontalLines(res)
	if len(lines) == 0 {
		t.Fatal("missing blue row rule")
	}

	if !near(lines[0].R, 0) || !near(lines[0].G, 0) || !near(lines[0].B, 1) {
		t.Errorf("row rule color = (%g,%g,%g), want blue", lines[0].R, lines[0].G, lines[0].B)
	}
}

//nolint:wsl,lll // the two visibility fixtures are one comparison
func TestBehaviorRowRuleVisibilityAround(t *testing.T) {
	t.Parallel()

	between := rowRuleHorizontalLines(rowRuleGrid(t, "row-gap:20px;row-rule:3px solid #c00;row-rule-visibility-items:between"))
	around := rowRuleHorizontalLines(rowRuleGrid(t, "row-gap:20px;row-rule:3px solid #c00;row-rule-visibility-items:around"))
	if len(between) != 1 {
		t.Fatalf("between lines = %d, want 1", len(between))
	}

	if len(around) != 3 {
		t.Fatalf("around lines = %d, want the gap plus the two content edges", len(around))
	}
}

//nolint:wsl // the line list is the assertion subject
func TestBehaviorRowRuleInsetCap(t *testing.T) {
	t.Parallel()

	res := rowRuleGrid(t, "row-gap:20px;row-rule:3px solid #c00;row-rule-inset-cap:10px")
	lines := rowRuleRedStrokes(res, pxToPt(3))
	if len(lines) != 1 {
		t.Fatalf("inset lines = %d, want 1", len(lines))
	}

	if !near(lines[0].W, pxToPt(80)) {
		t.Errorf("inset line width = %.4fpt, want 80px", lines[0].W)
	}
}

func TestBehaviorRowRuleBreakNoneStillPaints(t *testing.T) {
	t.Parallel()

	res := rowRuleGrid(t, "row-gap:20px;row-rule:3px solid #c00;row-rule-break:none")
	if len(rowRuleRedStrokes(res, pxToPt(3))) != 1 {
		t.Fatal("row-rule-break:none painted no line")
	}

	if got := boxByID(t, res, "g").style.RowRuleBreak; got != "none" {
		t.Fatalf("RowRuleBreak = %q, want none", got)
	}
}

//nolint:cyclop,wsl // grid and multicol are checked in one fixture
func TestBehaviorRulePaintsBothAxes(t *testing.T) {
	t.Parallel()

	grid := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:220px;grid-template-columns:100px 100px;grid-template-rows:50px 50px;`+
		`gap:20px;rule:3px solid #c00">`+
		`<div style="height:50px"></div><div style="height:50px"></div>`+
		`<div style="height:50px"></div><div style="height:50px"></div></div></body></html>`)
	if len(rowRuleRedStrokes(grid, pxToPt(3))) == 0 {
		t.Fatal("rule shorthand painted no horizontal row line")
	}

	sawColumn := false
	for _, op := range grid.Ops {
		if op.Kind == OpLine && op.W < 0.5 && op.H > 4 && near(op.Width, pxToPt(3)) {
			sawColumn = true
		}
	}

	if !sawColumn {
		t.Fatal("rule shorthand painted no vertical column line in the grid")
	}

	multicol := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div style="width:300px;column-count:2;column-gap:20px;rule:3px solid #c00">`+
		`Multi-column sample text repeated. Multi-column sample text repeated.`+
		`</div></body></html>`)
	sawMulti := false
	for _, op := range multicol.Ops {
		if op.Kind == OpLine && op.W < 0.5 && op.H > 4 && near(op.Width, pxToPt(3)) && near(op.R, 0.8) {
			sawMulti = true
		}
	}

	if !sawMulti {
		t.Fatal("rule shorthand painted no multicol column rule")
	}
}

//nolint:wsl // the used style follows the line check
func TestBehaviorRuleLonghandsBothAxes(t *testing.T) {
	t.Parallel()

	res := rowRuleGrid(t, "row-gap:20px;rule-style:solid;rule-width:6px;rule-color:#00f")
	lines := rowRuleHorizontalLines(res)
	if len(lines) == 0 || !near(lines[0].Width, pxToPt(6)) || !near(lines[0].B, 1) {
		t.Fatalf("row line from rule longhands = %+v", lines)
	}

	sty := boxByID(t, res, "g").style
	if !near(sty.ColumnRuleWidth, pxToPt(6)) || sty.ColumnRuleStyle != solidKeyword || !sty.ColumnRuleColorSet {
		t.Fatalf("column rule from rule longhands width %.3f style %q set %v",
			sty.ColumnRuleWidth, sty.ColumnRuleStyle, sty.ColumnRuleColorSet)
	}
}

//nolint:wsl // the two overlap orders are one comparison
func TestBehaviorRuleOverlapOrdersLines(t *testing.T) {
	t.Parallel()

	doc := func(overlap string) *Result {
		return layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
			`<div style="display:grid;width:220px;grid-template-columns:100px 100px;grid-template-rows:50px 50px;`+
			`gap:20px;row-rule:3px solid #c00;column-rule:3px solid #00f;rule-overlap:`+overlap+`">`+
			`<div style="height:50px"></div><div style="height:50px"></div>`+
			`<div style="height:50px"></div><div style="height:50px"></div></div></body></html>`)
	}

	rowFirst := doc("column-over-row")
	columnFirst := doc("row-over-column")
	if rowRuleIndex(rowFirst) > columnRuleIndex(rowFirst) {
		t.Error("column-over-row painted the row line after the column line")
	}

	if rowRuleIndex(columnFirst) < columnRuleIndex(columnFirst) {
		t.Error("row-over-column painted the column line after the row line")
	}
}

func rowRuleGrid(t *testing.T, decl string) *Result {
	t.Helper()

	return layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:100px;grid-template-columns:100px;grid-template-rows:50px 50px;`+decl+`">`+
		`<div id="a" style="height:50px"></div><div id="b" style="height:50px"></div>`+
		`</div></body></html>`)
}

func rowRuleIndex(res *Result) int {
	for idx, op := range res.Ops {
		if op.Kind == OpLine && op.H == 0 && op.W > 0 && near(op.R, 0.8) {
			return idx
		}
	}

	return -1
}

func columnRuleIndex(res *Result) int {
	for idx, op := range res.Ops {
		if op.Kind == OpLine && op.W < 0.5 && op.H > 4 && near(op.B, 1) {
			return idx
		}
	}

	return -1
}
