package layout

import "testing"

// This file exercises the scroller runtime in scroll_runtime.go: each
// scrollable overflow region carries a settable scroll offset, overflow
// clipping reads the offset-aware port, and snap-target visibility insets that
// port by the target's used scroll margins. Every expectation below is a USED
// value (a carried point offset, a computed clip edge, a trimmed fill rect),
// never a stored style string. The engine lays out in points and 1 CSS pixel
// = 0.75pt (pxToPt in style_values.go), so expectations are written in CSS
// pixels and converted before comparing, which is how Chrome reports them.
// Reference browser for every case: Chrome 143.0.7499.40, which uses
// scroll-margin as the scroll-snap offset of the target's margin edges.
//
// Static paint without a carried offset is unchanged: the zero offset reads as
// the padding box, so the identical-geometry pins in
// css_behavior_overflow2_test.go keep passing. Full snapping stays follow-up
// documented in scroll.go.

// scrollRuntimeDoc builds a scrollable 200x200px container holding a 100x50px
// red snap target carrying the scroll-margin declaration under test.
func scrollRuntimeDoc(extra string) string {
	item := scrollProbeItemStyle
	if extra != "" {
		item += ";" + extra
	}

	return `<html style="margin:0"><body style="margin:0">` +
		`<div id="rt-clip" style="width:200px;height:200px;overflow:auto">` +
		`<div id="rt-item" style="` + item + `"></div></div></body></html>`
}

// TestScrollRuntimeZeroOffsetPreservesStaticClip pins the static path: with no
// carried offset the scroller clip equals the padding box, so static paint is
// unchanged. Reference: Chrome 143.0.7499.40.
func TestScrollRuntimeZeroOffsetPreservesStaticClip(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, scrollRuntimeDoc(""))
	clipBox := boxByID(t, res, "rt-clip")

	eng := &engine{scale: 1}
	got := eng.computeBoxOverflowClip(clipBox, nil)
	want := eng.paddingBoxOf(clipBox)

	if got == nil {
		t.Fatalf("scroller clip is nil, want padding box (%.4f, %.4f, %.4f, %.4f)pt",
			want.x, want.y, want.w, want.h)
	}

	if !near(got.x, want.x) || !near(got.y, want.y) ||
		!near(got.w, want.w) || !near(got.h, want.h) {
		t.Errorf("scroller clip = (%.4f, %.4f, %.4f, %.4f)pt, want padding box (%.4f, %.4f, %.4f, %.4f)pt",
			got.x, got.y, got.w, got.h, want.x, want.y, want.w, want.h)
	}
}

// TestScrollRuntimeOffsetShiftsScrollerClip is the carried-offset half of the
// runtime: setting a test offset on the scroller translates the viewport and
// the overflow clip by exactly the scrolled distance, and a probe fill
// clipped to each rect trims to different used geometry. Reference: Chrome
// 143.0.7499.40.
func TestScrollRuntimeOffsetShiftsScrollerClip(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, scrollRuntimeDoc(""))
	clipBox := boxByID(t, res, "rt-clip")

	setTestScrollOffset(clipBox, ScrollOffset{X: pxToPt(20), Y: pxToPt(10)})
	defer unsetTestScrollOffset(clipBox)

	eng := &engine{scale: 1}
	view := eng.scrollRegionViewport(clipBox)

	if !near(view.At.X, pxToPt(20)) || !near(view.At.Y, pxToPt(10)) {
		t.Fatalf("carried offset = (%.4f, %.4f)pt, want (%.4f, %.4f)pt",
			view.At.X, view.At.Y, pxToPt(20), pxToPt(10))
	}

	resting := eng.paddingBoxOf(clipBox)
	got := eng.computeBoxOverflowClip(clipBox, nil)

	if got == nil {
		t.Fatalf("scrolled clip is nil, want port translated by (20px, 10px)")
	}

	if !near(got.x, resting.x+pxToPt(20)) || !near(got.y, resting.y+pxToPt(10)) ||
		!near(got.w, resting.w) || !near(got.h, resting.h) {
		t.Errorf("scrolled clip = (%.4f, %.4f, %.4f, %.4f)pt, want port (%.4f, %.4f, %.4f, %.4f)pt shifted by (20px, 10px)",
			got.x, got.y, got.w, got.h, resting.x, resting.y, resting.w, resting.h)
	}

	// A fill straddling the static right edge trims to different used widths
	// under each clip, proving the offset observably shifts clipped output.
	probe := Op{
		Kind: OpFillRect,
		X:    resting.x + resting.w - pxToPt(5), Y: resting.y,
		W: pxToPt(20), H: pxToPt(20),
	}
	staticOp := probe
	clipRectOp(&staticOp, resting)

	shiftedOp := probe
	clipRectOp(&shiftedOp, *got)

	if !near(staticOp.W, pxToPt(5)) {
		t.Errorf("static trimmed width = %.4fpt, want 5px", staticOp.W)
	}

	if !near(shiftedOp.W, pxToPt(20)) {
		t.Errorf("scrolled trimmed width = %.4fpt, want full 20px inside the shifted clip", shiftedOp.W)
	}
}

// TestScrollRuntimeMarginsShiftVisibleClip makes each scroll-margin property
// observable: with the same carried offset, the used visibility rect of the
// target is the scrolled port inset by that property's used margins, and a
// probe fill covering the port trims to different used geometry per property.
// Reference: Chrome 143.0.7499.40.
func TestScrollRuntimeMarginsShiftVisibleClip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                     string
		decl                     string
		top, right, bottom, left float64
	}{
		{name: "shorthand", decl: "scroll-margin:40px", top: 40, right: 40, bottom: 40, left: 40},
		{name: "top", decl: "scroll-margin-top:40px", top: 40},
		{name: "right", decl: "scroll-margin-right:40px", right: 40},
		{name: "bottom", decl: "scroll-margin-bottom:40px", bottom: 40},
		{name: "left", decl: "scroll-margin-left:40px", left: 40},
	}

	for _, marginCase := range cases {
		t.Run(marginCase.name, func(t *testing.T) {
			t.Parallel()

			scrollRuntimeMarginCase(t, marginCase.decl, scrollProbeWidths{
				top: marginCase.top, right: marginCase.right,
				bottom: marginCase.bottom, left: marginCase.left,
			})
		})
	}
}

// scrollProbeWidths carries the per-side pixel insets one scroll-margin
// property is expected to produce.
type scrollProbeWidths struct {
	top, right, bottom, left float64
}

// scrollRuntimeMarginCase asserts one scroll-margin declaration shifts the
// used visible rect and the clipped probe fill. Split out of the table test so
// each declaration has its own readable body.
func scrollRuntimeMarginCase(t *testing.T, decl string, want scrollProbeWidths) {
	t.Helper()

	res := layoutHTML(t, scrollRuntimeDoc(decl))
	clipBox := boxByID(t, res, "rt-clip")
	itemBox := boxByID(t, res, "rt-item")

	setTestScrollOffset(clipBox, ScrollOffset{X: pxToPt(20), Y: pxToPt(10)})
	defer unsetTestScrollOffset(clipBox)

	eng := &engine{scale: 1}
	view := eng.scrollRegionViewport(clipBox)
	margins := scrollMarginsOf(itemBox.style)

	assertScrollMargins(t, margins, want)
	visible := scrollVisibleRect(view, margins)
	assertVisibleRectInset(t, view, visible, want)
	assertMarginedClipShift(t, view, visible, want, decl)
}

// assertScrollMargins checks the used scroll margins equal the declared insets.
func assertScrollMargins(t *testing.T, got ScrollMargins, want scrollProbeWidths) {
	t.Helper()

	if !near(got.Top, pxToPt(want.top)) || !near(got.Right, pxToPt(want.right)) ||
		!near(got.Bottom, pxToPt(want.bottom)) || !near(got.Left, pxToPt(want.left)) {
		t.Fatalf("used margins = (%.4f, %.4f, %.4f, %.4f)pt, want (%gpx, %gpx, %gpx, %gpx)",
			got.Top, got.Right, got.Bottom, got.Left,
			want.top, want.right, want.bottom, want.left)
	}
}

// assertVisibleRectInset checks the scrolled port is inset by the used margins.
func assertVisibleRectInset(t *testing.T, view ScrollViewport, visible clipRect, want scrollProbeWidths) {
	t.Helper()

	if !near(visible.x, view.Port.x+pxToPt(20)+pxToPt(want.left)) ||
		!near(visible.y, view.Port.y+pxToPt(10)+pxToPt(want.top)) ||
		!near(visible.w, view.Port.w-pxToPt(want.left)-pxToPt(want.right)) ||
		!near(visible.h, view.Port.h-pxToPt(want.top)-pxToPt(want.bottom)) {
		t.Errorf("visible rect = (%.4f, %.4f, %.4f, %.4f)pt, want scrolled port inset (%gpx, %gpx, %gpx, %gpx)",
			visible.x, visible.y, visible.w, visible.h,
			want.top, want.right, want.bottom, want.left)
	}
}

// assertMarginedClipShift clips one port-covering probe to the unmargined
// translated port and to the margined visible rect. The trimmed used geometry
// differs by exactly the margin totals, which is the observable effect.
func assertMarginedClipShift(t *testing.T, view ScrollViewport, visible clipRect, want scrollProbeWidths, decl string) {
	t.Helper()

	probe := Op{
		Kind: OpFillRect,
		X:    view.Port.x, Y: view.Port.y,
		W: view.Port.w + pxToPt(100), H: view.Port.h + pxToPt(100),
	}
	plainOp := probe
	clipRectOp(&plainOp, scrollOffsetAwarePort(view))

	marginedOp := probe
	clipRectOp(&marginedOp, visible)

	if !near(plainOp.W, view.Port.w) || !near(plainOp.H, view.Port.h) {
		t.Errorf("unmargined trimmed fill = (%.4f, %.4f)pt, want port (%.4f, %.4f)pt",
			plainOp.W, plainOp.H, view.Port.w, view.Port.h)
	}

	if !near(marginedOp.W, visible.w) || !near(marginedOp.H, visible.h) {
		t.Errorf("margined trimmed fill = (%.4f, %.4f)pt, want visible (%.4f, %.4f)pt",
			marginedOp.W, marginedOp.H, visible.w, visible.h)
	}

	if near(marginedOp.W, plainOp.W) && near(marginedOp.H, plainOp.H) &&
		want.top+want.right+want.bottom+want.left > 0 {
		t.Errorf("margined fill matches unmargined fill, want %s to shift clipped output", decl)
	}
}

// TestScrollRuntimeSnapRectOutsetsTarget pins the snap half of the runtime:
// the target border box outset by its used scroll margins is the snap area
// whose margin edges align to the snapport, independent of the carried offset.
// Reference: Chrome 143.0.7499.40.
func TestScrollRuntimeSnapRectOutsetsTarget(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, scrollRuntimeDoc("scroll-margin:40px"))
	clipBox := boxByID(t, res, "rt-clip")
	itemBox := boxByID(t, res, "rt-item")

	setTestScrollOffset(clipBox, ScrollOffset{X: pxToPt(20), Y: pxToPt(10)})
	defer unsetTestScrollOffset(clipBox)

	margins := scrollMarginsOf(itemBox.style)
	target := clipRect{x: itemBox.x, y: itemBox.y, w: itemBox.w, h: itemBox.height}
	snapped := scrollSnapRect(target, margins)

	if !near(snapped.x, target.x-pxToPt(40)) || !near(snapped.y, target.y-pxToPt(40)) ||
		!near(snapped.w, target.w+pxToPt(80)) || !near(snapped.h, target.h+pxToPt(80)) {
		t.Errorf("snap rect = (%.4f, %.4f, %.4f, %.4f)pt, want target outset 40px each side",
			snapped.x, snapped.y, snapped.w, snapped.h)
	}
}
