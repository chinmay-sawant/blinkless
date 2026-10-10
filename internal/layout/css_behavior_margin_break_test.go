package layout

import "testing"

// This file holds the margin-break behavior tests: every observable property
// below is asserted through a USED value (a measured box gap, full op
// geometry), never through a stored style string. The engine lays out in
// points and 1 CSS pixel = 0.75pt (ptPerCSSPx in css_review_02_test.go), so
// every expectation is written in CSS pixels and converted with pxToPt before
// comparing, which is how Chrome reports it. Reference browser for every
// case: Chrome 143.0.7499.40.
//
// margin-break (CSS Fragmentation 4) picks keep or discard for the margins
// that touch a fragment break. This engine has no fragment break point in
// normal flow: layout emits one continuous display list with no page
// splitter, block siblings collapse adjoining margins to their max
// (layout_flow.go:698 via collapseMargins at layout_flow.go:1429), and Paint
// only assigns whole ops to the page of their top edge (layout.go:181-184)
// without splitting a box or touching its margins. The value parses and
// stores (internal/layout/style_properties.go:1658) but no layout or paint
// pass reads it: the only break logic beside BreakInside
// (internal/layout/independent_blocks.go:112) is the IndependentBlocks
// detector, which has no production caller (only its own test file consumes
// it). These two tests therefore pin the current no-op behavior with
// identical-geometry comparisons plus a measured collapsed gap, and are
// reported as unobservable, never as failing tests.

// marginBreakDoc returns a margin-zero document with two 100px blocks whose
// adjoining margins collapse to max(30px, 20px) = 30px. breakDecl is an extra
// declaration for #b ("" omits it).
func marginBreakDoc(breakDecl string) string {
	bStyle := "height:100px;margin-top:20px"
	if breakDecl != "" {
		bStyle += ";" + breakDecl
	}

	return `<html style="margin:0"><body style="margin:0">` +
		`<div id="a" style="height:100px;margin-bottom:30px">aaa</div>` +
		`<div id="b" style="` + bStyle + `">bbb</div></body></html>`
}

// marginBreakTallDoc is marginBreakDoc with a 750px spacer first, so #b starts
// past the 800pt layout height: the spot where Chrome paginates and keep or
// discard would change the used gap.
func marginBreakTallDoc(breakDecl string) string {
	bStyle := "height:100px;margin-top:20px"
	if breakDecl != "" {
		bStyle += ";" + breakDecl
	}

	return `<html style="margin:0"><body style="margin:0">` +
		`<div id="pad" style="height:750px">pad</div>` +
		`<div id="a" style="height:100px;margin-bottom:30px">aaa</div>` +
		`<div id="b" style="` + bStyle + `">bbb</div></body></html>`
}

// marginBreakUsedGap measures the used gap between the bottom of #a and the
// top of #b: the collapsed adjoining margin in points.
func marginBreakUsedGap(t *testing.T, res *Result) float64 {
	t.Helper()

	a := boxByID(t, res, "a")
	b := boxByID(t, res, "b")

	return b.y - (a.y + a.height)
}

// TestBehaviorMarginBreakKeepNoFragmentEffect documents the current behavior
// of margin-break: keep paints identical geometry to auto and to the
// unstyled document, and the used adjoining margin stays collapsed at 30px.
// Unobservable in the drawing list: reported as a no-op, never failing.
// Reference: Chrome 143.0.7499.40 keeps the margin here too when nothing
// fragments; at a real page break Chrome would keep the full margin while
// this engine has no forced-break pass to observe that in.
func TestBehaviorMarginBreakKeepNoFragmentEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, marginBreakDoc(""))
	auto := layoutHTML(t, marginBreakDoc("margin-break:auto"))
	keep := layoutHTML(t, marginBreakDoc("margin-break:keep"))

	behaviorColumnsAssertSameOps(t, plain, auto)
	behaviorColumnsAssertSameOps(t, plain, keep)

	for name, res := range map[string]*Result{"plain": plain, "auto": auto, "keep": keep} {
		if gap := marginBreakUsedGap(t, res); !near(gap, pxToPt(30)) {
			t.Errorf("%s used adjoining margin = %.4fpt (%.2fpx), want 30px collapsed",
				name, gap, gap/ptPerCSSPx)
		}
	}
}

// TestBehaviorMarginBreakDiscardNoFragmentEffect documents the current
// behavior of margin-break: discard paints identical geometry to auto and to
// the unstyled document even when #b starts past the 800pt layout height,
// where Chrome would fragment and drop the adjoining margin. The used gap
// stays collapsed at 30px. Unobservable in the drawing list: reported as a
// no-op, never failing. Reference: Chrome 143.0.7499.40.
func TestBehaviorMarginBreakDiscardNoFragmentEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, marginBreakTallDoc(""))
	auto := layoutHTML(t, marginBreakTallDoc("margin-break:auto"))
	discard := layoutHTML(t, marginBreakTallDoc("margin-break:discard"))

	behaviorColumnsAssertSameOps(t, plain, auto)
	behaviorColumnsAssertSameOps(t, plain, discard)

	for name, res := range map[string]*Result{"plain": plain, "auto": auto, "discard": discard} {
		if gap := marginBreakUsedGap(t, res); !near(gap, pxToPt(30)) {
			t.Errorf("%s used adjoining margin = %.4fpt (%.2fpx), want 30px collapsed",
				name, gap, gap/ptPerCSSPx)
		}
	}
}
