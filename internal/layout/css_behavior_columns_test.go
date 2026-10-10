package layout

import "testing"

// This file holds the columns and fragmentation behavior tests: every
// observable property below is asserted through a USED value (a measured
// column width, a used gap offset, an emitted rule op), never through a
// stored style string. The engine lays out in points and 1 CSS pixel =
// 0.75pt (ptPerCSSPx in css_review_02_test.go), so every expectation is
// written in CSS pixels and converted with pxToPt before comparing, which is
// how Chrome reports it. Reference browser for every case: Chrome
// 143.0.7499.40.
//
// Fragmentation properties (break-inside, break-before, orphans) only take
// effect across page breaks. This engine returns a single drawing list: layout
// emits no forced breaks (page-break keywords are parsed onto the box and
// only the IndependentBlocks detector in independent_blocks.go reads them for
// convert), and Paint moves ops that straddle a page boundary wholly to the
// next page without ever splitting a line. Those three tests therefore pin
// the current no-op behavior with identical-geometry comparisons and are
// reported as unobservable, never as failing tests.

// behaviorColumnsAssertSameOps asserts two results paint the same ops in the
// same order: same kind, geometry, and text. Used to pin no-op behavior for
// properties the drawing list cannot observe.
func behaviorColumnsAssertSameOps(t *testing.T, a, b *Result) {
	t.Helper()

	if len(a.Ops) != len(b.Ops) {
		t.Fatalf("op count = %d vs %d, want identical geometry", len(a.Ops), len(b.Ops))
	}

	for i := range a.Ops {
		x, y := a.Ops[i], b.Ops[i]
		if x.Kind != y.Kind || x.Text != y.Text ||
			!near(x.X, y.X) || !near(x.Y, y.Y) ||
			!near(x.W, y.W) || !near(x.H, y.H) {
			t.Fatalf("op %d differs: %+v vs %+v, want identical geometry", i, x, y)
		}
	}
}

// behaviorColumnsRuleOps returns the vertical line ops in a result, which is
// the shape emitColumnRules produces for a column rule (see
// TestColumnRulePaints in multicol_test.go).
func behaviorColumnsRuleOps(res *Result) []Op {
	var out []Op

	for _, op := range res.Ops {
		if op.Kind != OpLine || op.W >= 0.5 || op.H < 4 {
			continue
		}

		out = append(out, op)
	}

	return out
}

// behaviorColumnsChildXs returns the used x of each listed child box.
func behaviorColumnsChildXs(t *testing.T, res *Result, ids ...string) []float64 {
	t.Helper()

	xs := make([]float64, 0, len(ids))

	for _, id := range ids {
		xs = append(xs, boxByID(t, res, id).x)
	}

	return xs
}

// TestBehaviorColumnsShorthandUsedCount is columns: the shorthand sets both
// longhands, so columns:100px 2 on a 320px container with a 10px gap keeps 2
// columns (2*100+10 = 210 fits) stretched to (320-10)/2 = 155px each.
// Reference: Chrome 143.0.7499.40.
func TestBehaviorColumnsShorthandUsedCount(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="mc" style="width:320px;columns:100px 2;column-gap:10px">`+
		`<div id="k1" style="height:20px"></div><div id="k2" style="height:20px"></div>`+
		`</div></body></html>`)

	for _, id := range []string{"k1", "k2"} {
		if w := boxByID(t, res, id).w; !near(w, pxToPt(155)) {
			t.Errorf("#%s used width = %.4fpt (%.2fpx), want 155px", id, w, w/ptPerCSSPx)
		}
	}
}

// TestBehaviorColumnCountUsedWidths is column-count: 3 columns on a 300px
// container with column-gap:0 split the content width evenly, so every child
// measures 100px wide. Reference: Chrome 143.0.7499.40.
func TestBehaviorColumnCountUsedWidths(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="mc" style="width:300px;column-count:3;column-gap:0px">`+
		`<div id="k1" style="height:20px"></div>`+
		`<div id="k2" style="height:20px"></div>`+
		`<div id="k3" style="height:20px"></div>`+
		`</div></body></html>`)

	for _, id := range []string{"k1", "k2", "k3"} {
		if w := boxByID(t, res, id).w; !near(w, pxToPt(100)) {
			t.Errorf("#%s used width = %.4fpt (%.2fpx), want 100px", id, w, w/ptPerCSSPx)
		}
	}
}

// TestBehaviorColumnWidthAutoCount is column-width: with auto count the used
// count derives from the width (CSS Multicol 3.3), so column-width:100px on a
// 320px container with a 10px gap gives floor((320+10)/110) = 3 columns of
// (320-20)/3 = 100px. Six equal children balance two per column, proving the
// count through three distinct used x buckets. Reference: Chrome
// 143.0.7499.40.
func TestBehaviorColumnWidthAutoCount(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="mc" style="width:320px;column-width:100px;column-gap:10px">`+
		`<div id="k1" style="height:10px"></div><div id="k2" style="height:10px"></div>`+
		`<div id="k3" style="height:10px"></div><div id="k4" style="height:10px"></div>`+
		`<div id="k5" style="height:10px"></div><div id="k6" style="height:10px"></div>`+
		`</div></body></html>`)

	ids := []string{"k1", "k2", "k3", "k4", "k5", "k6"}

	for _, id := range ids {
		if w := boxByID(t, res, id).w; !near(w, pxToPt(100)) {
			t.Errorf("#%s used width = %.4fpt (%.2fpx), want 100px", id, w, w/ptPerCSSPx)
		}
	}

	var buckets []float64

	for _, x := range behaviorColumnsChildXs(t, res, ids...) {
		seen := false

		for _, b := range buckets {
			if near(x, b) {
				seen = true

				break
			}
		}

		if !seen {
			buckets = append(buckets, x)
		}
	}

	if len(buckets) != 3 {
		t.Errorf("distinct column x buckets = %d (%v), want 3", len(buckets), buckets)
	}
}

// TestBehaviorColumnGapUsedSpacing is column-gap: on a 300px container with 2
// columns and a 30px gap each column is (300-30)/2 = 135px, so the second
// balanced child starts 135+30 = 165px right of the first. Reference: Chrome
// 143.0.7499.40.
func TestBehaviorColumnGapUsedSpacing(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="mc" style="width:300px;column-count:2;column-gap:30px">`+
		`<div id="k1" style="height:40px"></div><div id="k2" style="height:40px"></div>`+
		`</div></body></html>`)

	xs := behaviorColumnsChildXs(t, res, "k1", "k2")

	if !near(xs[1]-xs[0], pxToPt(165)) {
		t.Errorf("column offset = %.4fpt (%.2fpx), want 165px", xs[1]-xs[0], (xs[1]-xs[0])/ptPerCSSPx)
	}
}

// TestBehaviorColumnRulePaintedBetweenColumns is column-rule: the shorthand
// paints one vertical rule centered in the gap. On a 300px container with 2
// columns and a 20px gap each column is 140px, so a 3px solid red rule sits at
// content-left + 150px and spans the multicol line height. Reference: Chrome
// 143.0.7499.40.
func TestBehaviorColumnRulePaintedBetweenColumns(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="mc" style="width:300px;column-count:2;column-gap:20px;`+
		`column-rule:3px solid #c00;font-size:10pt">`+
		`Multi-column sample text repeated. Multi-column sample text repeated. `+
		`Multi-column sample text repeated.</div></body></html>`)

	mc := boxByID(t, res, "mc")

	var found *Op

	for _, op := range behaviorColumnsRuleOps(res) {
		if near(op.R, 0.8) && near(op.G, 0) && near(op.B, 0) && near(op.Width, pxToPt(3)) {
			found = &op

			break
		}
	}

	if found == nil {
		t.Fatal("missing 3px solid red column-rule op between columns")
	}

	if !near(found.X, mc.x+pxToPt(150)) {
		t.Errorf("rule x = %.4fpt (%.2fpx from container %.2fpx), want gap middle 150px",
			found.X, (found.X-mc.x)/ptPerCSSPx, mc.x/ptPerCSSPx)
	}
}

// TestBehaviorColumnRuleWidthUsedThickness is column-rule-width: with a solid
// style a 6px rule emits a line op whose stroke width is 6px, in the text
// color when no rule color is set (currentColor). Reference: Chrome
// 143.0.7499.40.
func TestBehaviorColumnRuleWidthUsedThickness(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="mc" style="width:300px;column-count:2;column-gap:20px;`+
		`column-rule-style:solid;column-rule-width:6px;font-size:10pt">`+
		`Multi-column sample text repeated. Multi-column sample text repeated. `+
		`Multi-column sample text repeated.</div></body></html>`)

	rules := behaviorColumnsRuleOps(res)

	if len(rules) == 0 {
		t.Fatal("missing column-rule op for 6px solid rule")
	}

	if !near(rules[0].Width, pxToPt(6)) {
		t.Errorf("rule stroke width = %.4fpt (%.2fpx), want 6px", rules[0].Width, rules[0].Width/ptPerCSSPx)
	}

	if !near(rules[0].R, 0) || !near(rules[0].G, 0) || !near(rules[0].B, 0) {
		t.Errorf("rule color = (%.2f, %.2f, %.2f), want currentColor black",
			rules[0].R, rules[0].G, rules[0].B)
	}
}

// TestBehaviorColumnSpanAllFullWidth is column-span:all: the spanner lays out
// across the full 300px content width while ordinary children measure the
// 140px column width. Reference: Chrome 143.0.7499.40.
func TestBehaviorColumnSpanAllFullWidth(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="mc" style="width:300px;column-count:2;column-gap:20px;font-size:10pt">`+
		`<div id="before" style="height:20px"></div>`+
		`<div id="span" style="column-span:all;height:20px"></div>`+
		`<div id="after" style="height:20px"></div>`+
		`</div></body></html>`)

	if w := boxByID(t, res, "span").w; !near(w, pxToPt(300)) {
		t.Errorf("spanner used width = %.4fpt (%.2fpx), want full 300px", w, w/ptPerCSSPx)
	}

	for _, id := range []string{"before", "after"} {
		if w := boxByID(t, res, id).w; !near(w, pxToPt(140)) {
			t.Errorf("#%s used width = %.4fpt (%.2fpx), want column 140px", id, w, w/ptPerCSSPx)
		}
	}
}

// TestBehaviorColumnFillAutoStacksFirstColumn is column-fill: with a definite
// 100px height and two 50px children, balance splits them across the two
// columns (160px x offset on a 300px container with a 20px gap) while auto
// fills the first column top to bottom (same x, 50px y offset). Reference:
// Chrome 143.0.7499.40.
func TestBehaviorColumnFillAutoStacksFirstColumn(t *testing.T) {
	t.Parallel()

	doc := func(fill string) string {
		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="mc" style="width:300px;column-count:2;column-gap:20px;height:100px;` +
			`column-fill:` + fill + `">` +
			`<div id="k1" style="height:50px"></div><div id="k2" style="height:50px"></div>` +
			`</div></body></html>`
	}

	balanced := layoutHTML(t, doc("balance"))
	bxs := behaviorColumnsChildXs(t, balanced, "k1", "k2")

	if !near(bxs[1]-bxs[0], pxToPt(160)) {
		t.Errorf("balanced column offset = %.4fpt (%.2fpx), want 160px",
			bxs[1]-bxs[0], (bxs[1]-bxs[0])/ptPerCSSPx)
	}

	auto := layoutHTML(t, doc("auto"))
	axs := behaviorColumnsChildXs(t, auto, "k1", "k2")

	if !near(axs[1]-axs[0], 0) {
		t.Errorf("auto second child x offset = %.4fpt (%.2fpx), want 0 (stacked)",
			axs[1]-axs[0], (axs[1]-axs[0])/ptPerCSSPx)
	}

	k1 := boxByID(t, auto, "k1")
	k2 := boxByID(t, auto, "k2")

	if !near(k2.y-k1.y, pxToPt(50)) {
		t.Errorf("auto second child y offset = %.4fpt (%.2fpx), want 50px",
			k2.y-k1.y, (k2.y-k1.y)/ptPerCSSPx)
	}
}

// TestBehaviorBreakInsideAvoidColumnNoPageEffect documents the current
// behavior of break-inside: the column value does not map to any page break
// (see TestMulticolParseProps) and the page value lands on the box but no
// layout or paint pass reads it (paint_geom.go keeps the keywords for later
// passes; only the IndependentBlocks detector reads them for convert), so
// both variants paint byte-identical geometry to the unstyled document.
// Unobservable in the drawing list: reported as a no-op, never failing.
// Reference: Chrome 143.0.7499.40 would keep the block together at a page
// break; this engine has no forced-break pass to observe that in.
func TestBehaviorBreakInsideAvoidColumnNoPageEffect(t *testing.T) {
	t.Parallel()

	base := `<html style="margin:0"><body style="margin:0">` +
		`<div id="a" style="height:100px"></div>` +
		`<div id="b" style="height:100px"></div></body></html>`

	plain := layoutHTML(t, base)

	avoidColumn := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="a" style="height:100px"></div>`+
		`<div id="b" style="height:100px;break-inside:avoid-column"></div></body></html>`)
	behaviorColumnsAssertSameOps(t, plain, avoidColumn)

	avoidPage := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="a" style="height:100px"></div>`+
		`<div id="b" style="height:100px;break-inside:avoid"></div></body></html>`)
	behaviorColumnsAssertSameOps(t, plain, avoidPage)
}

// TestBehaviorBreakBeforeColumnNoPageEffect documents the current behavior of
// break-before: the column value parses to page-break-before:always (see
// TestMulticolParseProps) but layout emits no forced break and Paint only
// relocates ops that straddle a page boundary, so the styled document paints
// identical geometry to the unstyled one. Unobservable in the drawing list:
// reported as a no-op, never failing. Reference: Chrome 143.0.7499.40 would
// start the block on a new page.
func TestBehaviorBreakBeforeColumnNoPageEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="a" style="height:100px"></div>`+
		`<div id="b" style="height:100px"></div></body></html>`)

	forced := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="a" style="height:100px"></div>`+
		`<div id="b" style="height:100px;break-before:column"></div></body></html>`)

	behaviorColumnsAssertSameOps(t, plain, forced)
}

// TestBehaviorOrphansNoFragmentationEffect documents the current behavior of
// orphans: the value parses (initial 2, integer >= 1) but no layout pass
// reads it, and Paint moves whole ops across page boundaries without ever
// splitting a line, so orphans:5 paints identical text positions to the
// default. Unobservable in the drawing list: reported as a no-op, never
// failing. Reference: Chrome 143.0.7499.40 would keep 5 lines together at a
// page break.
func TestBehaviorOrphansNoFragmentationEffect(t *testing.T) {
	t.Parallel()

	para := `First line of text here. Second line of text here. ` +
		`Third line of text here. Fourth line here. Fifth line of text here. ` +
		`Sixth line of text here.`

	def := layoutHTML(t, `<html style="margin:0"><body style="margin:0;width:200px"><p id="p" `+
		`style="font-size:16px">`+para+`</p></body></html>`)

	five := layoutHTML(t, `<html style="margin:0"><body style="margin:0;width:200px"><p id="p" `+
		`style="font-size:16px;orphans:5">`+para+`</p></body></html>`)

	behaviorColumnsAssertSameOps(t, def, five)
}

// TestBehaviorOverflowHiddenClipsTallChild is overflow: a 200x50px hidden
// container clips its 150px child to the padding box, so paint past the
// bottom edge is deactivated and every surviving text op starts above the
// clip bottom. Reference: Chrome 143.0.7499.40.
func TestBehaviorOverflowHiddenClipsTallChild(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="clip" style="width:200px;height:50px;overflow:hidden">`+
		`<div id="tall" style="font-size:16px">`+
		`<p id="l1" style="margin:0">clip line one</p>`+
		`<p id="l2" style="margin:0">clip line two</p>`+
		`<p id="l3" style="margin:0">clip line three</p>`+
		`<p id="l4" style="margin:0">clip line four</p>`+
		`<p id="l5" style="margin:0">clip line five</p>`+
		`<p id="l6" style="margin:0">clip line six</p>`+
		`</div></div></body></html>`)

	clip := boxByID(t, res, "clip")

	if !near(clip.w, pxToPt(200)) || !near(clip.height, pxToPt(50)) {
		t.Fatalf("clip used box = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want 200px x 50px",
			clip.w, clip.height, clip.w/ptPerCSSPx, clip.height/ptPerCSSPx)
	}

	clipBottom := clip.y + clip.height
	dead, live := 0, 0

	for _, op := range res.Ops {
		if len(op.Text) < 4 || op.Text[:4] != "clip" {
			continue
		}

		if op.Kind == opKindNoop {
			dead++

			continue
		}

		if op.Kind != OpText {
			continue
		}

		live++

		top := op.Y - op.Size
		if op.H > 0 {
			top = op.Y - op.H
		}

		if top >= clipBottom-0.01 {
			t.Errorf("text %q starts at %.4fpt, at or past clip bottom %.4fpt",
				op.Text, top, clipBottom)
		}
	}

	if dead == 0 {
		t.Errorf("no clipped text ops deactivated, want overflow lines past 50px removed")
	}

	if live == 0 {
		t.Errorf("no live text ops, want the first lines visible inside the 50px box")
	}
}
