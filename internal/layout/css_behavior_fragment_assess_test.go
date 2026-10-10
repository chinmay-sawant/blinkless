package layout

import "testing"

// Fragmentation assessment: orphans, widows, break-inside, break-after,
// margin-break (plus break-before, which rides the same multicol seam).
//
// Normal flow has no fragment break point: layout emits one continuous
// display list, block siblings stack with margin collapse and no break query
// (layout_flow.go layoutBlockChild), and Paint assigns each op to the page of
// its top edge, moving ops that cross a page boundary wholly to the next page
// (layout.go:181-184) without ever splitting a line. The five properties
// therefore pin as no-ops there (css_behavior_columns_test.go:308-379,
// css_behavior_text_remaining_test.go:269-310,
// css_behavior_margin_break_test.go:68-115,
// css_behavior_hard_gaps_test.go:127-191).
//
// The multicol element-item path IS a real fragmentation point: whole child
// boxes are distributed across column breaks by placeMulticolLine
// (multicol.go:698-750), which previously read only heights
// (advanceMulticolColumn at multicol.go:765-783). Forced breaks
// (break-before/break-after column/page, parsed to always at
// style_properties.go:1680-1681 and 1693-1694) now force the next column via
// multicolForcedColumnBreak (multicol.go:785-806). Everything else below pins
// the remaining gaps with identical-geometry asserts, never failing.
// Reference browser for every case: Chrome 143.0.7499.40.

// fragmentAssessAutoDoc returns a margin-zero 2-column auto-fill multicol
// container with a definite 100px height and three 30px children. Auto fill
// stacks sequentially, so all three fit the first column (90px <= 100px) and
// any column jump by #k3 is caused by the tested declaration alone.
// extra is an additional declaration for #k3 ("" omits it).
func fragmentAssessAutoDoc(extra string) string {
	k3Style := "height:30px"
	if extra != "" {
		k3Style += ";" + extra
	}

	return `<html style="margin:0"><body style="margin:0">` +
		`<div id="mc" style="width:300px;column-count:2;column-gap:20px;height:100px;column-fill:auto">` +
		`<div id="k1" style="height:30px"></div><div id="k2" style="height:30px"></div>` +
		`<div id="k3" style="` + k3Style + `"></div>` +
		`</div></body></html>`
}

// TestFragmentAssessBreakBeforeColumnForcesColumn is break-before in multicol:
// break-before:column on #k3 (parsed to PageBreakBefore always) moves it from
// the first column into the second: +160px in x (140px column + 20px gap on a
// 300px container) and back to the top y. Reference: Chrome 143.0.7499.40
// starts the block in the next column.
func TestFragmentAssessBreakBeforeColumnForcesColumn(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, fragmentAssessAutoDoc(""))
	forced := layoutHTML(t, fragmentAssessAutoDoc("break-before:column"))

	if got := boxByID(t, forced, "k3").style.PageBreakBefore; got != pageBreakAlways {
		t.Fatalf("break-before:column used = %q, want always", got)
	}

	plainK1 := boxByID(t, plain, "k1")
	plainK3 := boxByID(t, plain, "k3")

	if !near(plainK3.x-plainK1.x, 0) {
		t.Fatalf("plain #k3 x offset = %.4fpt (%.2fpx), want 0 (stacked in column 1)",
			plainK3.x-plainK1.x, (plainK3.x-plainK1.x)/ptPerCSSPx)
	}

	forcedK1 := boxByID(t, forced, "k1")
	forcedK3 := boxByID(t, forced, "k3")

	if !near(forcedK3.x-forcedK1.x, pxToPt(160)) {
		t.Errorf("break-before #k3 x offset = %.4fpt (%.2fpx), want 160px (column 2)",
			forcedK3.x-forcedK1.x, (forcedK3.x-forcedK1.x)/ptPerCSSPx)
	}

	if !near(forcedK3.y, forcedK1.y) {
		t.Errorf("break-before #k3 y = %.4fpt, want column-top %.4fpt",
			forcedK3.y, forcedK1.y)
	}
}

// TestFragmentAssessBreakAfterColumnForcesColumn is break-after in multicol:
// break-after:column on #k2 (parsed to PageBreakAfter always) pushes #k3 into
// the second column with the same geometry as the break-before case.
// Reference: Chrome 143.0.7499.40.
func TestFragmentAssessBreakAfterColumnForcesColumn(t *testing.T) {
	t.Parallel()

	doc := func(extra string) string {
		k2Style := "height:30px"
		if extra != "" {
			k2Style += ";" + extra
		}

		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="mc" style="width:300px;column-count:2;column-gap:20px;height:100px;column-fill:auto">` +
			`<div id="k1" style="height:30px"></div>` +
			`<div id="k2" style="` + k2Style + `"></div>` +
			`<div id="k3" style="height:30px"></div>` +
			`</div></body></html>`
	}

	plain := layoutHTML(t, doc(""))
	forced := layoutHTML(t, doc("break-after:column"))

	if got := boxByID(t, forced, "k2").style.PageBreakAfter; got != pageBreakAlways {
		t.Fatalf("break-after:column used = %q, want always", got)
	}

	plainK1 := boxByID(t, plain, "k1")
	if !near(boxByID(t, plain, "k3").x-plainK1.x, 0) {
		t.Fatalf("plain #k3 stacks in column 1, want x offset 0")
	}

	forcedK1 := boxByID(t, forced, "k1")
	forcedK3 := boxByID(t, forced, "k3")

	if !near(forcedK3.x-forcedK1.x, pxToPt(160)) {
		t.Errorf("break-after #k3 x offset = %.4fpt (%.2fpx), want 160px (column 2)",
			forcedK3.x-forcedK1.x, (forcedK3.x-forcedK1.x)/ptPerCSSPx)
	}

	if !near(forcedK3.y, forcedK1.y) {
		t.Errorf("break-after #k3 y = %.4fpt, want column-top %.4fpt",
			forcedK3.y, forcedK1.y)
	}
}

// TestFragmentAssessBreakBeforeFirstChildNoColumnEffect pins the forced-break
// guard: break-before on the first item of a line targets an empty column,
// which is already satisfied, so the document paints identical geometry to
// the unstyled one (multicol.go:797-799). Never failing.
func TestFragmentAssessBreakBeforeFirstChildNoColumnEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, fragmentAssessAutoDoc(""))
	first := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="mc" style="width:300px;column-count:2;column-gap:20px;height:100px;column-fill:auto">`+
		`<div id="k1" style="height:30px;break-before:column"></div>`+
		`<div id="k2" style="height:30px"></div>`+
		`<div id="k3" style="height:30px"></div>`+
		`</div></body></html>`)

	behaviorColumnsAssertSameOps(t, plain, first)
}

// TestFragmentAssessBreakAvoidMulticolNoEffect pins break avoid in multicol:
// with 60px children in a 100px auto-fill column, #k2 overflows the first
// column and advances in both documents. Chrome 143.0.7499.40 would honor
// break-before:avoid by keeping #k2 in column 1 overflowing; this engine has
// no keep-together pass (multicolForcedColumnBreak only fires on always at
// multicol.go:801-805), so avoid paints identical geometry to plain. Never
// failing.
func TestFragmentAssessBreakAvoidMulticolNoEffect(t *testing.T) {
	t.Parallel()

	doc := func(extra string) string {
		k2Style := "height:60px"
		if extra != "" {
			k2Style += ";" + extra
		}

		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="mc" style="width:300px;column-count:2;column-gap:20px;height:100px;column-fill:auto">` +
			`<div id="k1" style="height:60px"></div>` +
			`<div id="k2" style="` + k2Style + `"></div>` +
			`</div></body></html>`
	}

	plain := layoutHTML(t, doc(""))
	avoid := layoutHTML(t, doc("break-before:avoid"))

	if got := boxByID(t, avoid, "k2").style.PageBreakBefore; got != "avoid" {
		t.Fatalf("break-before:avoid used = %q, want avoid", got)
	}

	behaviorColumnsAssertSameOps(t, plain, avoid)

	for name, res := range map[string]*Result{"plain": plain, "avoid": avoid} {
		k1 := boxByID(t, res, "k1")
		if gap := boxByID(t, res, "k2").x - k1.x; !near(gap, pxToPt(160)) {
			t.Errorf("%s #k2 x offset = %.4fpt (%.2fpx), want 160px (advanced despite avoid)",
				name, gap, gap/ptPerCSSPx)
		}
	}
}

// TestFragmentAssessBreakInsideAvoidMulticolNoEffect pins break-inside in
// multicol: items are always assigned whole to one column
// (placeMulticolLine builds each item once at multicol.go:736-742 and only
// advances on heights at multicol.go:765-783), so break-inside:avoid matches
// the default auto behavior exactly. An avoid box can no more split than an
// auto box can. Never failing. Reference: Chrome 143.0.7499.40.
func TestFragmentAssessBreakInsideAvoidMulticolNoEffect(t *testing.T) {
	t.Parallel()

	doc := func(extra string) string {
		k1Style := "height:40px"
		if extra != "" {
			k1Style += ";" + extra
		}

		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="mc" style="width:300px;column-count:2;column-gap:20px">` +
			`<div id="k1" style="` + k1Style + `"></div>` +
			`<div id="k2" style="height:40px"></div>` +
			`</div></body></html>`
	}

	plain := layoutHTML(t, doc(""))
	avoid := layoutHTML(t, doc("break-inside:avoid"))

	if got := boxByID(t, avoid, "k1").style.PageBreakInside; got != "avoid" {
		t.Fatalf("break-inside:avoid used = %q, want avoid", got)
	}

	behaviorColumnsAssertSameOps(t, plain, avoid)
}

// TestFragmentAssessOrphansWidowsAnonStripNoEffect pins orphans/widows in
// multicol: a lone anonymous text strip IS fragmented across columns, but by
// pure height bands (bandH splits totalH evenly at multicol.go:467, ops are
// assigned by relY/bandH at multicol.go:497) with mid-line cuts explicitly
// kept (multicol.go:516-518). No pass counts lines and nothing reads
// Orphans/Widows (zero references in multicol.go), even though the values
// reach the strip style (anonymousFlexItemStyle copies the parent at
// flex.go:175 and resets only box props at flex.go:176-226, leaving
// Orphans/Widows intact). Honoring them needs line-aware band snapping plus
// min-line enforcement per column: not a small honest edit. Never failing.
// Reference: Chrome 143.0.7499.40 would keep 5 lines together.
func TestFragmentAssessOrphansWidowsAnonStripNoEffect(t *testing.T) {
	t.Parallel()

	text := `Multi-column sample text repeated. Multi-column sample text repeated. ` +
		`Multi-column sample text repeated. Multi-column sample text repeated. ` +
		`Multi-column sample text repeated. Multi-column sample text repeated.`
	doc := func(extra string) string {
		mcStyle := "width:300px;column-count:2;column-gap:20px;font-size:10pt"
		if extra != "" {
			mcStyle += ";" + extra
		}

		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="mc" style="` + mcStyle + `">` + text + `</div></body></html>`
	}

	plain := layoutHTML(t, doc(""))
	fragment := layoutHTML(t, doc("orphans:5;widows:5"))

	if got := boxByID(t, fragment, "mc").style.Orphans; got != 5 {
		t.Fatalf("orphans used = %d, want 5", got)
	}

	if got := boxByID(t, fragment, "mc").style.Widows; got != 5 {
		t.Fatalf("widows used = %d, want 5", got)
	}

	behaviorColumnsAssertSameOps(t, plain, fragment)
}

// TestFragmentAssessMarginBreakMulticolNoEffect pins margin-break in multicol:
// column breaks stack whole child boxes (colHeights accumulation at
// multicol.go:742) with margins baked into the measured heights
// (measureMulticolChildHeight at multicol.go:808-826 builds the full box).
// Nothing recomputes margins at the column boundary and nothing reads
// MarginBreak in multicol.go, so discard paints identical geometry to plain
// even though the value stores. Needs margin recompute at the break: not a
// small honest edit. Never failing. Reference: Chrome 143.0.7499.40 would
// drop the adjoining margin at a real column break.
func TestFragmentAssessMarginBreakMulticolNoEffect(t *testing.T) {
	t.Parallel()

	doc := func(extra string) string {
		k2Style := "height:30px;margin-top:10px"
		if extra != "" {
			k2Style += ";" + extra
		}

		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="mc" style="width:300px;column-count:2;column-gap:20px;height:100px;column-fill:auto">` +
			`<div id="k1" style="height:30px"></div>` +
			`<div id="k2" style="` + k2Style + `"></div>` +
			`<div id="k3" style="height:30px"></div>` +
			`</div></body></html>`
	}

	plain := layoutHTML(t, doc(""))
	discard := layoutHTML(t, doc("margin-break:discard"))

	if got := boxByID(t, discard, "k2").style.MarginBreak; got != "discard" {
		t.Fatalf("margin-break used = %q, want discard", got)
	}

	behaviorColumnsAssertSameOps(t, plain, discard)

	if !near(boxByID(t, discard, "k3").y, boxByID(t, plain, "k3").y) {
		t.Errorf("#k3 y differs with margin-break:discard, want identical (no margin recompute)")
	}
}
