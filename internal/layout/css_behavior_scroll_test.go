package layout

import "testing"

// This file exercises the scroll-offset model in scroll.go: the scroll-margin
// shorthand and its four longhands store real used values (points), each
// scrollable region carries a scroll offset through its viewport, and
// scroll-margin insets the snap/visibility rect. Every expectation below is a
// USED value (a stored point length or a computed rect edge), never a stored
// style string. The engine lays out in points and 1 CSS pixel = 0.75pt
// (pxToPt in style_values.go), so expectations are written in CSS pixels and
// converted before comparing, which is how Chrome reports them. Reference
// browser for every case: Chrome 143.0.7499.40, which uses scroll-margin as
// the scroll-snap offset of the target's margin edges.
//
// Static paint is deliberately unchanged: with the zero offset this engine
// always carries, overflow clipping still uses the padding box, so the
// identical-geometry pins in css_behavior_overflow2_test.go keep passing.
// Full snapping (choosing a snap offset from scroll-snap-align/type) is
// follow-up documented in scroll.go.

// scrollProbeItemStyle is the 100x50px red scroll probe box shared by the
// scroll, scroll-runtime, and overflow behavior fixtures.
const scrollProbeItemStyle = "width:100px;height:50px;background-color:#ff0000"

// scrollSnapDoc builds a scrollable 200x200px container holding a 100x50px red
// snap target carrying the scroll-margin declaration under test. The tall
// container keeps the shorthand margins under test from covering the whole
// port (covering clamps to empty, pinned by TestScrollMarginCoveringPortReadsEmpty).
func scrollSnapDoc(extra string) string {
	item := scrollProbeItemStyle
	if extra != "" {
		item += ";" + extra
	}

	return `<html style="margin:0"><body style="margin:0">` +
		`<div id="clip" style="width:200px;height:200px;overflow:auto">` +
		`<div id="item" style="` + item + `"></div></div></body></html>`
}

// scrollViewportOf returns the test viewport for the #clip region paired with
// the given scroll offset, plus the used scroll margins of #item.
func scrollViewportOf(t *testing.T, res *Result, offset ScrollOffset) (ScrollViewport, ScrollMargins) {
	t.Helper()

	clip := boxByID(t, res, "clip")
	item := boxByID(t, res, "item")

	view := ScrollViewport{At: offset}
	view.Port = clipRect{x: clip.x, y: clip.y, w: clip.w, h: clip.height}

	return view, scrollMarginsOf(item.style)
}

// scrollAssertMargins requires the used scroll margins to equal the four
// expected point values.
func scrollAssertMargins(t *testing.T, margins ScrollMargins, top, right, bottom, left float64) {
	t.Helper()

	if !near(margins.Top, top) || !near(margins.Right, right) ||
		!near(margins.Bottom, bottom) || !near(margins.Left, left) {
		t.Errorf(
			"scroll margins = (%.4f, %.4f, %.4f, %.4f)pt, want (%.4f, %.4f, %.4f, %.4f)pt",
			margins.Top, margins.Right, margins.Bottom, margins.Left, top, right, bottom, left,
		)
	}
}

// TestScrollMarginShorthandInsetsVisibleRect is scroll-margin: the shorthand
// stores 40px on all four sides (30pt each), so the visibility rect of the
// target is the scrolled port inset by 30pt on every side, and the snap rect
// is the target border box outset by 30pt on every side. Reference: Chrome
// 143.0.7499.40.
func TestScrollMarginShorthandInsetsVisibleRect(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, scrollSnapDoc("scroll-margin:40px"))
	view, margins := scrollViewportOf(t, res, ScrollOffset{X: pxToPt(20), Y: pxToPt(10)})

	scrollAssertMargins(t, margins, pxToPt(40), pxToPt(40), pxToPt(40), pxToPt(40))

	visible := scrollVisibleRect(view, margins)

	if !near(visible.x, view.Port.x+pxToPt(20)+pxToPt(40)) ||
		!near(visible.y, view.Port.y+pxToPt(10)+pxToPt(40)) ||
		!near(visible.w, view.Port.w-pxToPt(80)) ||
		!near(visible.h, view.Port.h-pxToPt(80)) {
		t.Errorf(
			"visible rect = (%.4f, %.4f, %.4f, %.4f)pt, want port inset 40px each side at offset (20px, 10px)",
			visible.x, visible.y, visible.w, visible.h,
		)
	}

	item := boxByID(t, res, "item")
	target := clipRect{x: item.x, y: item.y, w: item.w, h: item.height}
	snapped := scrollSnapRect(target, margins)

	if !near(snapped.x, target.x-pxToPt(40)) || !near(snapped.y, target.y-pxToPt(40)) ||
		!near(snapped.w, target.w+pxToPt(80)) || !near(snapped.h, target.h+pxToPt(80)) {
		t.Errorf(
			"snap rect = (%.4f, %.4f, %.4f, %.4f)pt, want target outset 40px each side",
			snapped.x, snapped.y, snapped.w, snapped.h,
		)
	}
}

// TestScrollMarginTopShiftsVisibleRect is scroll-margin-top: only the top
// side stores 40px, so the visibility rect moves its top edge down 30pt and
// loses 30pt of height while the other three edges stay on the scrolled port.
// Reference: Chrome 143.0.7499.40.
func TestScrollMarginTopShiftsVisibleRect(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, scrollSnapDoc("scroll-margin-top:40px"))
	view, margins := scrollViewportOf(t, res, ScrollOffset{X: pxToPt(20), Y: pxToPt(10)})

	scrollAssertMargins(t, margins, pxToPt(40), 0, 0, 0)

	visible := scrollVisibleRect(view, margins)

	if !near(visible.x, view.Port.x+pxToPt(20)) || !near(visible.w, view.Port.w) ||
		!near(visible.y, view.Port.y+pxToPt(10)+pxToPt(40)) ||
		!near(visible.h, view.Port.h-pxToPt(40)) {
		t.Errorf(
			"visible rect = (%.4f, %.4f, %.4f, %.4f)pt, want only the top edge shifted 40px",
			visible.x, visible.y, visible.w, visible.h,
		)
	}
}

// TestScrollMarginBottomShiftsVisibleRect is scroll-margin-bottom: only the
// bottom side stores 40px, so the visibility rect keeps the scrolled port's
// top edge and loses 30pt of height. Reference: Chrome 143.0.7499.40.
func TestScrollMarginBottomShiftsVisibleRect(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, scrollSnapDoc("scroll-margin-bottom:40px"))
	view, margins := scrollViewportOf(t, res, ScrollOffset{X: pxToPt(20), Y: pxToPt(10)})

	scrollAssertMargins(t, margins, 0, 0, pxToPt(40), 0)

	visible := scrollVisibleRect(view, margins)

	if !near(visible.x, view.Port.x+pxToPt(20)) || !near(visible.w, view.Port.w) ||
		!near(visible.y, view.Port.y+pxToPt(10)) ||
		!near(visible.h, view.Port.h-pxToPt(40)) {
		t.Errorf(
			"visible rect = (%.4f, %.4f, %.4f, %.4f)pt, want only the bottom edge shifted 40px",
			visible.x, visible.y, visible.w, visible.h,
		)
	}
}

// TestScrollMarginLeftShiftsVisibleRect is scroll-margin-left: only the left
// side stores 40px, so the visibility rect moves its left edge right 30pt and
// loses 30pt of width while the vertical edges stay on the scrolled port.
// Reference: Chrome 143.0.7499.40.
func TestScrollMarginLeftShiftsVisibleRect(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, scrollSnapDoc("scroll-margin-left:40px"))
	view, margins := scrollViewportOf(t, res, ScrollOffset{X: pxToPt(20), Y: pxToPt(10)})

	scrollAssertMargins(t, margins, 0, 0, 0, pxToPt(40))

	visible := scrollVisibleRect(view, margins)

	if !near(visible.x, view.Port.x+pxToPt(20)+pxToPt(40)) || !near(visible.w, view.Port.w-pxToPt(40)) ||
		!near(visible.y, view.Port.y+pxToPt(10)) || !near(visible.h, view.Port.h) {
		t.Errorf(
			"visible rect = (%.4f, %.4f, %.4f, %.4f)pt, want only the left edge shifted 40px",
			visible.x, visible.y, visible.w, visible.h,
		)
	}
}

// TestScrollMarginRightShiftsVisibleRect is scroll-margin-right: only the
// right side stores 40px, so the visibility rect keeps the scrolled port's
// left edge and loses 30pt of width. Reference: Chrome 143.0.7499.40.
func TestScrollMarginRightShiftsVisibleRect(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, scrollSnapDoc("scroll-margin-right:40px"))
	view, margins := scrollViewportOf(t, res, ScrollOffset{X: pxToPt(20), Y: pxToPt(10)})

	scrollAssertMargins(t, margins, 0, pxToPt(40), 0, 0)

	visible := scrollVisibleRect(view, margins)

	if !near(visible.x, view.Port.x+pxToPt(20)) || !near(visible.w, view.Port.w-pxToPt(40)) ||
		!near(visible.y, view.Port.y+pxToPt(10)) || !near(visible.h, view.Port.h) {
		t.Errorf(
			"visible rect = (%.4f, %.4f, %.4f, %.4f)pt, want only the right edge shifted 40px",
			visible.x, visible.y, visible.w, visible.h,
		)
	}
}

// TestScrollMarginPairShorthandExpands is the two-token scroll-margin form:
// the first length covers the vertical sides and the second the horizontal
// sides, following the margin shorthand order. Reference: Chrome
// 143.0.7499.40.
func TestScrollMarginPairShorthandExpands(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, scrollSnapDoc("scroll-margin:40px 20px"))
	_, margins := scrollViewportOf(t, res, ScrollOffset{})

	scrollAssertMargins(t, margins, pxToPt(40), pxToPt(20), pxToPt(40), pxToPt(20))
}

// TestScrollRegionCarriesOffset pins the carried-offset half of the model: at
// the zero offset the engine lays out, the visibility rect of a margin-free
// target is the port itself, and a nonzero offset translates the rect by
// exactly the scrolled distance. Reference: Chrome 143.0.7499.40.
func TestScrollRegionCarriesOffset(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, scrollSnapDoc(""))
	view, margins := scrollViewportOf(t, res, ScrollOffset{})

	scrollAssertMargins(t, margins, 0, 0, 0, 0)

	resting := scrollVisibleRect(view, margins)

	if !near(resting.x, view.Port.x) || !near(resting.y, view.Port.y) ||
		!near(resting.w, view.Port.w) || !near(resting.h, view.Port.h) {
		t.Errorf(
			"visible rect at zero offset = (%.4f, %.4f, %.4f, %.4f)pt, want the port (%.4f, %.4f, %.4f, %.4f)pt",
			resting.x, resting.y, resting.w, resting.h, view.Port.x, view.Port.y, view.Port.w, view.Port.h,
		)
	}

	scrolled, _ := scrollViewportOf(t, res, ScrollOffset{X: pxToPt(20), Y: pxToPt(10)})
	moved := scrollVisibleRect(scrolled, margins)

	if !near(moved.x, view.Port.x+pxToPt(20)) || !near(moved.y, view.Port.y+pxToPt(10)) ||
		!near(moved.w, view.Port.w) || !near(moved.h, view.Port.h) {
		t.Errorf(
			"visible rect at offset (20px, 10px) = (%.4f, %.4f, %.4f, %.4f)pt, want the port translated",
			moved.x, moved.y, moved.w, moved.h,
		)
	}
}

// TestScrollMarginCoveringPortReadsEmpty pins the degenerate edge: a target
// whose scroll margins cover the whole port reads as an empty visibility
// rect (fully out of view), matching the empty-intersection convention in
// intersectClip. Reference: Chrome 143.0.7499.40.
func TestScrollMarginCoveringPortReadsEmpty(t *testing.T) {
	t.Parallel()

	view := ScrollViewport{Port: clipRect{x: 0, y: 0, w: pxToPt(200), h: pxToPt(200)}}
	margins := ScrollMargins{
		Top:    pxToPt(100),
		Right:  pxToPt(100),
		Bottom: pxToPt(100),
		Left:   pxToPt(100),
	}

	visible := scrollVisibleRect(view, margins)

	if !visible.empty() {
		t.Errorf(
			"visible rect = (%.4f, %.4f, %.4f, %.4f)pt, want empty when margins cover the port",
			visible.x, visible.y, visible.w, visible.h,
		)
	}
}

// TestScrollMarginRejectsNegative pins the invalid-value half of the model:
// negative scroll margins are rejected at parse time, so the used value stays
// zero and the visibility rect is unshifted; a bogus shorthand token voids
// the whole declaration. Reference: Chrome 143.0.7499.40.
func TestScrollMarginRejectsNegative(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, scrollSnapDoc("scroll-margin-top:-10px"))
	view, margins := scrollViewportOf(t, res, ScrollOffset{X: pxToPt(20), Y: pxToPt(10)})

	scrollAssertMargins(t, margins, 0, 0, 0, 0)

	visible := scrollVisibleRect(view, margins)

	if !near(visible.x, view.Port.x+pxToPt(20)) || !near(visible.y, view.Port.y+pxToPt(10)) ||
		!near(visible.w, view.Port.w) || !near(visible.h, view.Port.h) {
		t.Errorf(
			"visible rect with rejected margin = (%.4f, %.4f, %.4f, %.4f)pt, want the unshifted scrolled port",
			visible.x, visible.y, visible.w, visible.h,
		)
	}

	bogus := layoutHTML(t, scrollSnapDoc("scroll-margin:40px bogus"))
	_, bogusMargins := scrollViewportOf(t, bogus, ScrollOffset{})

	scrollAssertMargins(t, bogusMargins, 0, 0, 0, 0)
}
