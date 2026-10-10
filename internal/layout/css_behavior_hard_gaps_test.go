package layout

import "testing"

// Hard-gap pins for backface-visibility and the break family.
//
// Reference browser for every case: Chrome 143.0.7499.40.
//
// (1) backface-visibility: the declaration stores canonically
// (internal/layout/style_leftovers.go:150-155, initial visible at
// internal/layout/style.go:664, not inherited per
// internal/layout/style_cascade.go:502) and stampExclusiveTransformOps
// (internal/layout/transform.go) culls 3D-rotated back faces: a box with
// HasTransform3D whose flattened facing is negative zeroes its exclusive
// op geometry under hidden. 2D content never culls: Chrome keeps 2D
// scaleX(-1) visible under hidden because the property only applies to
// 3D-rotated faces. Pinned below as stored plus 3D cull behavior.
//
// (2) break-before (representative of break-before/break-after/break-inside
// plus orphans/widows/margin-break): the declaration parses onto the box
// (internal/layout/style_properties.go:1680-1681 sets PageBreakBefore to
// always; orphans/widows parse at style_properties.go:1747-1762 with initial
// 2 at style.go:599-600; margin-break stores at style_properties.go:1658-1666)
// but normal flow never reads it. Block siblings stack with margin collapse
// in internal/layout/layout_flow.go:679-734 (layoutBlockChild) with no break
// query; the sole PageBreakBefore reader is the IndependentBlocks detector
// (internal/layout/independent_blocks.go:108-109), which has no production
// caller. Layout emits one continuous display list with no page splitter
// (see css_behavior_margin_break_test.go:14-18), and the Paint-time page
// assignment described at internal/layout/layout.go:181-184 has no Paint
// function left in the tree (deleted per plans/v0.0.1/phase-wise-checklist.md).
// Multicol alone borrows Options.Height as a page height
// (internal/layout/multicol.go:360-367). A minimal break-before:page emission
// that rounds the child past the next Options.Height boundary is geometry
// only: with no pagination pass and no Pages/Locations consumer it would
// render as a blank gap, not a new page, and it would break the existing
// no-op pins. What is missing for a real fix: a page content height source
// (Options.Height at layout.go:98-99 is the viewport for percent heights,
// and the @page consumer is gone), a forced-break layout pass plus a
// whole-op splitter (orphans/widows need line splitting, margin-break needs
// margin recompute at the break), and break-after/before collapsing, avoid,
// first-child, and nested flex/grid/table handling (flex strips break props
// at flex.go:223-225). Verdict: needs the full splitter. Pinned below as
// stored-but-identical, never failing.

// TestHardGapBackfaceVisibilityStoresButPaintsIdentical pins the
// backface-visibility behavior: hidden stores canonically and paints
// identical geometry to visible under the same 2D rotation, while a 3D
// rotateY(180deg) with hidden zeroes its fill geometry and visible paints.
func TestHardGapBackfaceVisibilityStoresButPaintsIdentical(t *testing.T) {
	t.Parallel()

	doc := func(vis string) string {
		return `<html><body style="margin:0">` +
			`<div id="a" style="width:100px;height:50px;background-color:#ff0000;transform:rotate(10deg);` +
			`backface-visibility:` + vis + `"></div></body></html>`
	}

	front := layoutHTML(t, doc("visible"))
	hidden := layoutHTML(t, doc("hidden"))

	if got := boxByID(t, front, "a").style.BackfaceVisibility; got != "visible" {
		t.Errorf("backface-visibility used = %q, want visible", got)
	}

	if got := boxByID(t, hidden, "a").style.BackfaceVisibility; got != "hidden" {
		t.Errorf("backface-visibility used = %q, want hidden", got)
	}

	behaviorColumnsAssertSameOps(t, front, hidden)

	bogus := boxByID(t, layoutHTML(t, doc("bogus")), "a")
	if bogus.style.BackfaceVisibility != "visible" {
		t.Errorf("backface-visibility:bogus used = %q, want initial visible", bogus.style.BackfaceVisibility)
	}

	flat := boxByID(t, layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:100px;height:50px;transform:rotateY(180deg);`+
		`backface-visibility:hidden"></div></body></html>`), "a")
	if !flat.style.HasTransform3D {
		t.Errorf("rotateY(180deg) HasTransform3D = false, want true (3D parses onto the side channel)")
	}

	hardGapAssertBackfaceCull(t)
}

// hardGapAssertBackfaceCull pins the 3D truth: rotateY(180deg) turns the back
// face to the viewer, so hidden zeroes the fill geometry while visible
// paints. Reference: Chrome 143.0.7499.40.
func hardGapAssertBackfaceCull(t *testing.T) {
	t.Helper()

	threeDoc := func(vis string) string {
		return `<html><body style="margin:0">` +
			`<div id="a" style="width:100px;height:50px;background-color:#ff0000;transform:rotateY(180deg);` +
			`backface-visibility:` + vis + `"></div></body></html>`
	}

	visibleFills := opsOfKind(layoutHTML(t, threeDoc("visible")), OpFillRect)
	hiddenFills := opsOfKind(layoutHTML(t, threeDoc("hidden")), OpFillRect)

	if len(visibleFills) == 0 || len(hiddenFills) == 0 {
		t.Fatalf("want fill ops, got visible=%d hidden=%d", len(visibleFills), len(hiddenFills))
	}

	sawPaint := false

	for _, op := range visibleFills {
		if op.W > 1 && op.H > 1 {
			sawPaint = true
		}
	}

	if !sawPaint {
		t.Errorf("visible rotateY(180deg) paints no live fill, want at least one")
	}

	for _, op := range hiddenFills {
		if op.W != 0 || op.H != 0 {
			t.Errorf("hidden rotateY(180deg) fill = %.4fpt x %.4fpt, want 0 x 0 (culled)",
				op.W, op.H)
		}
	}
}

// TestHardGapBreakBeforePageNoForcedBreak pins the break-before gap: the
// page value parses to PageBreakBefore always yet the block keeps its
// in-flow position below its sibling instead of jumping past the 800pt page
// boundary. Reference: Chrome 143.0.7499.40 would start the block on a new
// page.
func TestHardGapBreakBeforePageNoForcedBreak(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="a" style="height:100px"></div>`+
		`<div id="b" style="height:100px"></div></body></html>`)

	forced := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="a" style="height:100px"></div>`+
		`<div id="b" style="height:100px;break-before:page"></div></body></html>`)

	if got := boxByID(t, forced, "b").style.PageBreakBefore; got != pageBreakAlways {
		t.Errorf("break-before:page used = %q, want always", got)
	}

	behaviorColumnsAssertSameOps(t, plain, forced)

	plainB := boxByID(t, plain, "b")
	forcedB := boxByID(t, forced, "b")

	if !near(forcedB.y, plainB.y) {
		t.Errorf("break-before:page #b y = %.4fpt, want in-flow %.4fpt (no forced break)",
			forcedB.y, plainB.y)
	}

	if forcedB.y >= 800 {
		t.Errorf("break-before:page #b y = %.4fpt, want below the 800pt page boundary (no page push)",
			forcedB.y)
	}
}

// TestHardGapBreakFamilyNoFragmentEffect pins the rest of the break family
// as one representative gap: break-after:page, orphans:5, widows:5, and
// margin-break:discard each paint identical geometry to the unstyled
// document because the engine has no fragment break point to observe them
// at. Reference: Chrome 143.0.7499.40 would break after the block, keep 5
// lines together, and drop the adjoining margin at a real page break.
func TestHardGapBreakFamilyNoFragmentEffect(t *testing.T) {
	t.Parallel()

	blocks := func(extra string) string {
		bStyle := "height:100px"
		if extra != "" {
			bStyle += ";" + extra
		}

		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="a" style="height:100px">aaa</div>` +
			`<div id="b" style="` + bStyle + `">bbb</div></body></html>`
	}

	plain := layoutHTML(t, blocks(""))

	after := layoutHTML(t, blocks("break-after:page"))
	if got := boxByID(t, after, "b").style.PageBreakAfter; got != pageBreakAlways {
		t.Errorf("break-after:page used = %q, want always", got)
	}

	behaviorColumnsAssertSameOps(t, plain, after)

	para := `First line of text here. Second line of text here. ` +
		`Third line of text here. Fourth line here. Fifth line of text here. ` +
		`Sixth line of text here.`
	textDoc := func(decl string) string {
		style := "font-size:16px"
		if decl != "" {
			style += ";" + decl
		}

		return `<html style="margin:0"><body style="margin:0;width:200px"><p id="p" ` +
			`style="` + style + `">` + para + `</p></body></html>`
	}

	defText := layoutHTML(t, textDoc(""))

	orphans := layoutHTML(t, textDoc("orphans:5"))
	if got := boxByID(t, orphans, "p").style.Orphans; got != 5 {
		t.Errorf("orphans used = %d, want 5", got)
	}

	behaviorColumnsAssertSameOps(t, defText, orphans)

	widows := layoutHTML(t, textDoc("widows:5"))
	if got := boxByID(t, widows, "p").style.Widows; got != 5 {
		t.Errorf("widows used = %d, want 5", got)
	}

	behaviorColumnsAssertSameOps(t, defText, widows)

	discard := layoutHTML(t, blocks("margin-break:discard"))
	if got := boxByID(t, discard, "b").style.MarginBreak; got != "discard" {
		t.Errorf("margin-break used = %q, want discard", got)
	}

	behaviorColumnsAssertSameOps(t, plain, discard)
}
