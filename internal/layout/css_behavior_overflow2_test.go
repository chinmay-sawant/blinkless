package layout

import "testing"

// This file holds overflow-axis behavior tests: the logical overflow axes,
// the overflow-clip-margin family, the scroll-margin family, and the
// box-shadow split longhands. Every observable property below is asserted
// through a USED value (a clipped op, a trimmed fill rect, an emitted shadow
// fill), never through a stored style string. The engine lays out in points
// and 1 CSS pixel = 0.75pt (ptPerCSSPx in css_review_02_test.go), so every
// expectation is written in CSS pixels and converted with pxToPt before
// comparing, which is how Chrome reports it. Reference browser for every
// case: Chrome 143.0.7499.40.
//
// Properties with no drawing-list consumer (the scroll-margin family: snap
// offsets with no scroll pass) pin the current no-op behavior with
// identical-geometry comparisons and are reported as unobservable, never as
// failing tests.

// behaviorOver2FillRects returns the live fill-rect ops painted in the given
// color. Deactivated ops have kind opKindNoop, so they never match.
func behaviorOver2FillRects(ops []Op, wantR, wantB float64) []Op {
	var out []Op

	for _, paintOp := range ops {
		if paintOp.Kind != OpFillRect {
			continue
		}

		if near(paintOp.R, wantR) && near(paintOp.G, 0) && near(paintOp.B, wantB) {
			out = append(out, paintOp)
		}
	}

	return out
}

// behaviorOver2FindFill returns the first live fill-rect op at the given rect
// in the given color, or nil.
func behaviorOver2FindFill(ops []Op, wantX, wantY, wantW, wantH, wantR, wantB float64) *Op {
	for _, paintOp := range behaviorOver2FillRects(ops, wantR, wantB) {
		if near(paintOp.X, wantX) && near(paintOp.Y, wantY) &&
			near(paintOp.W, wantW) && near(paintOp.H, wantH) {
			candidate := paintOp

			return &candidate
		}
	}

	return nil
}

// TestBehaviorOverflowBlockClipsTallChild is overflow-block: on the default
// horizontal-tb writing mode block maps to the Y axis (see
// setOverflowLogical in style_overflow_logical.go), so overflow-block:hidden
// on a 200x50px box clips its tall child exactly like overflow-y:hidden.
// Reference: Chrome 143.0.7499.40.
func TestBehaviorOverflowBlockClipsTallChild(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="clip" style="width:200px;height:50px;overflow-block:hidden">`+
		`<div id="tall" style="font-size:16px">`+
		`<p id="l1" style="margin:0">oblk line one</p>`+
		`<p id="l2" style="margin:0">oblk line two</p>`+
		`<p id="l3" style="margin:0">oblk line three</p>`+
		`<p id="l4" style="margin:0">oblk line four</p>`+
		`<p id="l5" style="margin:0">oblk line five</p>`+
		`<p id="l6" style="margin:0">oblk line six</p>`+
		`</div></div></body></html>`)

	behaviorClipAssertDeactivation(t, res, "clip", "oblk")
}

// TestBehaviorOverflowInlineClipsWideChild is overflow-inline: on the default
// horizontal-tb writing mode inline maps to the X axis (see setOverflowLogical
// in style_overflow_logical.go), so overflow-inline:hidden on a 100px box
// trims its 300px child fill to the padding box on the x axis only.
// Reference: Chrome 143.0.7499.40.
func TestBehaviorOverflowInlineClipsWideChild(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="clip" style="width:100px;height:50px;overflow-inline:hidden">`+
		`<div id="wide" style="width:300px;height:20px;background-color:#ff0000"></div>`+
		`</div></body></html>`)

	for _, w := range behaviorInteractFillWidths(res) {
		if w > pxToPt(100)+0.01 {
			t.Errorf("inline-hidden fill width = %.4fpt (%.2fpx), want at most 100px",
				w, w/ptPerCSSPx)
		}
	}

	if len(behaviorInteractFillWidths(res)) == 0 {
		t.Errorf("no live fills, want the 300px child trimmed to 100px")
	}
}

// behaviorOver2BottomDoc builds a 200x50px overflow:clip container followed
// by a 50px spacer and a 200x20px red target, so the target sits exactly at
// the clip bottom edge (y 50..70px) and is deactivated without a clip margin.
// extra holds the overflow-clip-margin declaration under test.
func behaviorOver2BottomDoc(extra string) string {
	style := "width:200px;height:50px;overflow:clip"
	if extra != "" {
		style += ";" + extra
	}

	return `<html style="margin:0"><body style="margin:0">` +
		`<div id="clip" style="` + style + `">` +
		`<div style="height:50px"></div>` +
		`<div id="target" style="width:200px;height:20px;background-color:#ff0000"></div>` +
		`</div></body></html>`
}

// behaviorOver2TopDoc builds a 200x50px overflow:clip container whose red
// 200x20px target is pulled to y -30..-10px by a negative top margin, so it
// is deactivated without a clip margin. extra holds the declaration under
// test.
func behaviorOver2TopDoc(extra string) string {
	style := "width:200px;height:50px;overflow:clip"
	if extra != "" {
		style += ";" + extra
	}

	return `<html style="margin:0"><body style="margin:0">` +
		`<div id="clip" style="` + style + `">` +
		`<div id="target" style="width:200px;height:20px;margin-top:-30px;background-color:#ff0000"></div>` +
		`</div></body></html>`
}

// behaviorOver2LeftDoc builds a 100x50px overflow:clip container whose red
// 200x20px target is pulled to x -30..170px by a negative left margin, so
// its fill is trimmed to 100px without a clip margin. extra holds the
// declaration under test.
func behaviorOver2LeftDoc(extra string) string {
	style := "width:100px;height:50px;overflow:clip"
	if extra != "" {
		style += ";" + extra
	}

	return `<html style="margin:0"><body style="margin:0">` +
		`<div id="clip" style="` + style + `">` +
		`<div id="target" style="width:200px;height:20px;margin-left:-30px;background-color:#ff0000"></div>` +
		`</div></body></html>`
}

// behaviorOver2RightDoc builds a 100x50px overflow:clip container with a red
// 300x20px target, so its fill is trimmed to 100px without a clip margin.
// extra holds the declaration under test.
func behaviorOver2RightDoc(extra string) string {
	style := "width:100px;height:50px;overflow:clip"
	if extra != "" {
		style += ";" + extra
	}

	return `<html style="margin:0"><body style="margin:0">` +
		`<div id="clip" style="` + style + `">` +
		`<div id="target" style="width:300px;height:20px;background-color:#ff0000"></div>` +
		`</div></body></html>`
}

// TestBehaviorOverflowClipMarginExpandsClip is overflow-clip-margin: the
// 40px shorthand inflates the padding-box clip on all sides (see
// paddingBoxRect in overflow_clip.go), so the below-edge target that is
// deactivated with plain overflow:clip paints its full 200x20px fill.
// Reference: Chrome 143.0.7499.40.
func TestBehaviorOverflowClipMarginExpandsClip(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, behaviorOver2BottomDoc(""))
	clip := boxByID(t, plain, "clip")

	if found := behaviorOver2FindFill(plain.Ops,
		clip.x, clip.y+pxToPt(50), pxToPt(200), pxToPt(20), 1, 0); found != nil {
		t.Fatalf("plain overflow:clip keeps the below-edge fill %+v, want it deactivated", *found)
	}

	margined := layoutHTML(t, behaviorOver2BottomDoc("overflow-clip-margin:40px"))
	mclip := boxByID(t, margined, "clip")

	if found := behaviorOver2FindFill(margined.Ops,
		mclip.x, mclip.y+pxToPt(50), pxToPt(200), pxToPt(20), 1, 0); found == nil {
		t.Errorf("no 200x20px red fill at the below-edge target with overflow-clip-margin:40px")
	}
}

// TestBehaviorOverflowClipMarginTopKeepsAboveChild is
// overflow-clip-margin-top: 40px inflates only the top edge of the clip, so
// the above-edge target that is deactivated with plain overflow:clip paints
// its full 200x20px fill. Reference: Chrome 143.0.7499.40.
func TestBehaviorOverflowClipMarginTopKeepsAboveChild(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, behaviorOver2TopDoc(""))
	clip := boxByID(t, plain, "clip")

	if found := behaviorOver2FindFill(plain.Ops,
		clip.x, clip.y-pxToPt(30), pxToPt(200), pxToPt(20), 1, 0); found != nil {
		t.Fatalf("plain overflow:clip keeps the above-edge fill %+v, want it deactivated", *found)
	}

	margined := layoutHTML(t, behaviorOver2TopDoc("overflow-clip-margin-top:40px"))
	mclip := boxByID(t, margined, "clip")

	if found := behaviorOver2FindFill(margined.Ops,
		mclip.x, mclip.y-pxToPt(30), pxToPt(200), pxToPt(20), 1, 0); found == nil {
		t.Errorf("no 200x20px red fill at the above-edge target with overflow-clip-margin-top:40px")
	}
}

// TestBehaviorOverflowClipMarginBottomKeepsBelowChild is
// overflow-clip-margin-bottom: 40px inflates only the bottom edge of the
// clip, so the below-edge target paints its full 200x20px fill. Reference:
// Chrome 143.0.7499.40.
func TestBehaviorOverflowClipMarginBottomKeepsBelowChild(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, behaviorOver2BottomDoc(""))
	clip := boxByID(t, plain, "clip")

	if found := behaviorOver2FindFill(plain.Ops,
		clip.x, clip.y+pxToPt(50), pxToPt(200), pxToPt(20), 1, 0); found != nil {
		t.Fatalf("plain overflow:clip keeps the below-edge fill %+v, want it deactivated", *found)
	}

	margined := layoutHTML(t, behaviorOver2BottomDoc("overflow-clip-margin-bottom:40px"))
	mclip := boxByID(t, margined, "clip")

	if found := behaviorOver2FindFill(margined.Ops,
		mclip.x, mclip.y+pxToPt(50), pxToPt(200), pxToPt(20), 1, 0); found == nil {
		t.Errorf("no 200x20px red fill at the below-edge target with overflow-clip-margin-bottom:40px")
	}
}

// TestBehaviorOverflowClipMarginLeftKeepsLeftChild is
// overflow-clip-margin-left: 40px inflates only the left edge of the clip,
// so the left-overflowing target fill grows from the trimmed 100px to
// 130px. Reference: Chrome 143.0.7499.40.
func TestBehaviorOverflowClipMarginLeftKeepsLeftChild(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, behaviorOver2LeftDoc(""))
	clip := boxByID(t, plain, "clip")

	if found := behaviorOver2FindFill(plain.Ops,
		clip.x, clip.y, pxToPt(100), pxToPt(20), 1, 0); found == nil {
		t.Fatalf("no trimmed 100px fill for the left-overflowing target with plain overflow:clip")
	}

	margined := layoutHTML(t, behaviorOver2LeftDoc("overflow-clip-margin-left:40px"))
	mclip := boxByID(t, margined, "clip")

	if found := behaviorOver2FindFill(margined.Ops,
		mclip.x-pxToPt(30), mclip.y, pxToPt(130), pxToPt(20), 1, 0); found == nil {
		t.Errorf("no 130px red fill for the left-overflowing target with overflow-clip-margin-left:40px")
	}
}

// TestBehaviorOverflowClipMarginRightKeepsWideChild is
// overflow-clip-margin-right: 40px inflates only the right edge of the clip,
// so the 300px target fill grows from the trimmed 100px to 140px.
// Reference: Chrome 143.0.7499.40.
func TestBehaviorOverflowClipMarginRightKeepsWideChild(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, behaviorOver2RightDoc(""))
	clip := boxByID(t, plain, "clip")

	if found := behaviorOver2FindFill(plain.Ops,
		clip.x, clip.y, pxToPt(100), pxToPt(20), 1, 0); found == nil {
		t.Fatalf("no trimmed 100px fill for the wide target with plain overflow:clip")
	}

	margined := layoutHTML(t, behaviorOver2RightDoc("overflow-clip-margin-right:40px"))
	mclip := boxByID(t, margined, "clip")

	if found := behaviorOver2FindFill(margined.Ops,
		mclip.x, mclip.y, pxToPt(140), pxToPt(20), 1, 0); found == nil {
		t.Errorf("no 140px red fill for the wide target with overflow-clip-margin-right:40px")
	}
}

// TestBehaviorOverflowClipMarginInlineKeepsWideChild is
// overflow-clip-margin-inline: 40px inflates the left and right edges for
// horizontal-tb (see applyAdvancedProps in style_advanced_props.go), so the
// 300px target fill grows from the trimmed 100px to 140px. Reference:
// Chrome 143.0.7499.40.
func TestBehaviorOverflowClipMarginInlineKeepsWideChild(t *testing.T) {
	t.Parallel()

	margined := layoutHTML(t, behaviorOver2RightDoc("overflow-clip-margin-inline:40px"))
	mclip := boxByID(t, margined, "clip")

	if found := behaviorOver2FindFill(margined.Ops,
		mclip.x, mclip.y, pxToPt(140), pxToPt(20), 1, 0); found == nil {
		t.Errorf("no 140px red fill for the wide target with overflow-clip-margin-inline:40px")
	}
}

// TestBehaviorOverflowClipMarginBlockKeepsBelowChild is
// overflow-clip-margin-block: 40px inflates the top and bottom edges for
// horizontal-tb (see applyAdvancedProps in style_advanced_props.go), so the
// below-edge target paints its full 200x20px fill. Reference: Chrome
// 143.0.7499.40.
func TestBehaviorOverflowClipMarginBlockKeepsBelowChild(t *testing.T) {
	t.Parallel()

	margined := layoutHTML(t, behaviorOver2BottomDoc("overflow-clip-margin-block:40px"))
	mclip := boxByID(t, margined, "clip")

	if found := behaviorOver2FindFill(margined.Ops,
		mclip.x, mclip.y+pxToPt(50), pxToPt(200), pxToPt(20), 1, 0); found == nil {
		t.Errorf("no 200x20px red fill at the below-edge target with overflow-clip-margin-block:40px")
	}
}

// TestBehaviorOverflowClipMarginInlineStartKeepsLeftChild is
// overflow-clip-margin-inline-start: 40px inflates the left edge in LTR
// horizontal-tb (see applyAdvancedProps in style_advanced_props.go), so the
// left-overflowing target fill grows from the trimmed 100px to 130px.
// Reference: Chrome 143.0.7499.40.
func TestBehaviorOverflowClipMarginInlineStartKeepsLeftChild(t *testing.T) {
	t.Parallel()

	margined := layoutHTML(t, behaviorOver2LeftDoc("overflow-clip-margin-inline-start:40px"))
	mclip := boxByID(t, margined, "clip")

	if found := behaviorOver2FindFill(margined.Ops,
		mclip.x-pxToPt(30), mclip.y, pxToPt(130), pxToPt(20), 1, 0); found == nil {
		t.Errorf("no 130px red fill for the left-overflowing target with overflow-clip-margin-inline-start:40px")
	}
}

// TestBehaviorOverflowClipMarginInlineEndKeepsWideChild is
// overflow-clip-margin-inline-end: 40px inflates the right edge in LTR
// horizontal-tb (see applyAdvancedProps in style_advanced_props.go), so the
// 300px target fill grows from the trimmed 100px to 140px. Reference:
// Chrome 143.0.7499.40.
func TestBehaviorOverflowClipMarginInlineEndKeepsWideChild(t *testing.T) {
	t.Parallel()

	margined := layoutHTML(t, behaviorOver2RightDoc("overflow-clip-margin-inline-end:40px"))
	mclip := boxByID(t, margined, "clip")

	if found := behaviorOver2FindFill(margined.Ops,
		mclip.x, mclip.y, pxToPt(140), pxToPt(20), 1, 0); found == nil {
		t.Errorf("no 140px red fill for the wide target with overflow-clip-margin-inline-end:40px")
	}
}

// TestBehaviorOverflowClipMarginBlockStartKeepsAboveChild is
// overflow-clip-margin-block-start: 40px inflates the top edge for
// horizontal-tb (see applyAdvancedProps in style_advanced_props.go), so the
// above-edge target paints its full 200x20px fill. Reference: Chrome
// 143.0.7499.40.
func TestBehaviorOverflowClipMarginBlockStartKeepsAboveChild(t *testing.T) {
	t.Parallel()

	margined := layoutHTML(t, behaviorOver2TopDoc("overflow-clip-margin-block-start:40px"))
	mclip := boxByID(t, margined, "clip")

	if found := behaviorOver2FindFill(margined.Ops,
		mclip.x, mclip.y-pxToPt(30), pxToPt(200), pxToPt(20), 1, 0); found == nil {
		t.Errorf("no 200x20px red fill at the above-edge target with overflow-clip-margin-block-start:40px")
	}
}

// TestBehaviorOverflowClipMarginBlockEndKeepsBelowChild is
// overflow-clip-margin-block-end: 40px inflates the bottom edge for
// horizontal-tb (see applyAdvancedProps in style_advanced_props.go), so the
// below-edge target paints its full 200x20px fill. Reference: Chrome
// 143.0.7499.40.
func TestBehaviorOverflowClipMarginBlockEndKeepsBelowChild(t *testing.T) {
	t.Parallel()

	margined := layoutHTML(t, behaviorOver2BottomDoc("overflow-clip-margin-block-end:40px"))
	mclip := boxByID(t, margined, "clip")

	if found := behaviorOver2FindFill(margined.Ops,
		mclip.x, mclip.y+pxToPt(50), pxToPt(200), pxToPt(20), 1, 0); found == nil {
		t.Errorf("no 200x20px red fill at the below-edge target with overflow-clip-margin-block-end:40px")
	}
}

// behaviorOver2ScrollDoc builds a 100x50px red box carrying the scroll-margin
// declaration under test.
func behaviorOver2ScrollDoc(extra string) string {
	style := scrollProbeItemStyle
	if extra != "" {
		style += ";" + extra
	}

	return `<html style="margin:0"><body style="margin:0">` +
		`<div id="item" style="` + style + `"></div></body></html>`
}

// TestBehaviorScrollMarginNoPaintEffect documents the current behavior of
// scroll-margin: Chrome 143.0.7499.40 uses it as a scroll-snap offset, but
// this engine has no scroll pass and keeps no ResolvedStyle field for it
// (style_paint_props.go routes it to applyLeftoversProps, which has no case
// for it in style_leftovers.go), so the styled document paints
// byte-identical geometry to the unstyled one. Unobservable in the drawing
// list: reported as a no-op, never failing.
func TestBehaviorScrollMarginNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, behaviorOver2ScrollDoc(""))
	styled := layoutHTML(t, behaviorOver2ScrollDoc("scroll-margin:40px"))

	behaviorColumnsAssertSameOps(t, plain, styled)
}

// TestBehaviorScrollMarginTopNoPaintEffect documents the current behavior of
// scroll-margin-top: same gap as scroll-margin (no scroll pass, no stored
// field), so scroll-margin-top:40px paints identical geometry to the
// unstyled document. Unobservable in the drawing list: reported as a no-op,
// never failing. Reference: Chrome 143.0.7499.40.
func TestBehaviorScrollMarginTopNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, behaviorOver2ScrollDoc(""))
	styled := layoutHTML(t, behaviorOver2ScrollDoc("scroll-margin-top:40px"))

	behaviorColumnsAssertSameOps(t, plain, styled)
}

// TestBehaviorScrollMarginBottomNoPaintEffect documents the current behavior
// of scroll-margin-bottom: same gap as scroll-margin (no scroll pass, no
// stored field), so the styled document paints identical geometry to the
// unstyled one. Unobservable in the drawing list: reported as a no-op, never
// failing. Reference: Chrome 143.0.7499.40.
func TestBehaviorScrollMarginBottomNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, behaviorOver2ScrollDoc(""))
	styled := layoutHTML(t, behaviorOver2ScrollDoc("scroll-margin-bottom:40px"))

	behaviorColumnsAssertSameOps(t, plain, styled)
}

// TestBehaviorScrollMarginLeftNoPaintEffect documents the current behavior of
// scroll-margin-left: same gap as scroll-margin (no scroll pass, no stored
// field), so the styled document paints identical geometry to the unstyled
// one. Unobservable in the drawing list: reported as a no-op, never failing.
// Reference: Chrome 143.0.7499.40.
func TestBehaviorScrollMarginLeftNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, behaviorOver2ScrollDoc(""))
	styled := layoutHTML(t, behaviorOver2ScrollDoc("scroll-margin-left:40px"))

	behaviorColumnsAssertSameOps(t, plain, styled)
}

// TestBehaviorScrollMarginRightNoPaintEffect documents the current behavior
// of scroll-margin-right: same gap as scroll-margin (no scroll pass, no
// stored field), so the styled document paints identical geometry to the
// unstyled one. Unobservable in the drawing list: reported as a no-op, never
// failing. Reference: Chrome 143.0.7499.40.
func TestBehaviorScrollMarginRightNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, behaviorOver2ScrollDoc(""))
	styled := layoutHTML(t, behaviorOver2ScrollDoc("scroll-margin-right:40px"))

	behaviorColumnsAssertSameOps(t, plain, styled)
}

// behaviorOver2ShadowDoc builds a 100x50px box carrying longhand-only
// box-shadow declarations (no shorthand Raw, so paint uses the structured
// BoxShadow fields per appendBoxShadow in box_shadow.go). extra holds the
// declarations under test.
func behaviorOver2ShadowDoc(extra string) string {
	style := "width:100px;height:50px"
	if extra != "" {
		style += ";" + extra
	}

	return `<html style="margin:0"><body style="margin:0">` +
		`<div id="item" style="` + style + `"></div></body></html>`
}

// TestBehaviorBoxShadowBlurExpandsLayers is box-shadow-blur: with longhand
// authoring the blur radius selects the stacked expanding fills in
// appendOuterBoxShadow (box_shadow.go), so blur:8px emits exactly
// boxShadowBlurSteps more red fills than the sharp shadow. Reference: Chrome
// 143.0.7499.40, 100px by 50px box, 8px by 8px red shadow.
func TestBehaviorBoxShadowBlurExpandsLayers(t *testing.T) {
	t.Parallel()

	sharp := layoutHTML(t, behaviorOver2ShadowDoc("box-shadow-offset:8px 8px;box-shadow-color:red"))
	blurred := layoutHTML(t, behaviorOver2ShadowDoc("box-shadow-offset:8px 8px;box-shadow-color:red;box-shadow-blur:8px"))

	sharpFills := behaviorOver2FillRects(sharp.Ops, 1, 0)
	blurFills := behaviorOver2FillRects(blurred.Ops, 1, 0)

	if len(sharpFills) != 1 {
		t.Fatalf("sharp shadow fills = %d, want 1 core fill", len(sharpFills))
	}

	if len(blurFills) != len(sharpFills)+boxShadowBlurSteps {
		t.Errorf("blurred shadow fills = %d, want %d (1 core + %d blur rings)",
			len(blurFills), len(sharpFills)+boxShadowBlurSteps, boxShadowBlurSteps)
	}
}

// TestBehaviorBoxShadowColorPaintsLayer is box-shadow-color: with longhand
// authoring the layer paints in the declared color, so a blue declaration
// emits the offset fill in blue and no red fill remains. Reference: Chrome
// 143.0.7499.40, 100px by 50px box, 8px by 8px shadow.
func TestBehaviorBoxShadowColorPaintsLayer(t *testing.T) {
	t.Parallel()

	red := layoutHTML(t, behaviorOver2ShadowDoc("box-shadow-offset:8px 8px;box-shadow-color:red"))
	blue := layoutHTML(t, behaviorOver2ShadowDoc("box-shadow-offset:8px 8px;box-shadow-color:blue"))

	redItem := boxByID(t, red, "item")
	blueItem := boxByID(t, blue, "item")

	if found := behaviorOver2FindFill(red.Ops,
		redItem.x+pxToPt(8), redItem.y+pxToPt(8), redItem.w, redItem.height, 1, 0); found == nil {
		t.Errorf("no red shadow fill at (+8px, +8px) for box-shadow-color:red")
	}

	if found := behaviorOver2FindFill(blue.Ops,
		blueItem.x+pxToPt(8), blueItem.y+pxToPt(8), blueItem.w, blueItem.height, 0, 1); found == nil {
		t.Errorf("no blue shadow fill at (+8px, +8px) for box-shadow-color:blue")
	}

	if fills := behaviorOver2FillRects(blue.Ops, 1, 0); len(fills) != 0 {
		t.Errorf("red fills with box-shadow-color:blue = %d, want 0", len(fills))
	}
}

// TestBehaviorBoxShadowInsetPaintsInnerRim is box-shadow-inset: inset paints
// inner rims inside the border box (see appendInsetBoxShadow in
// box_shadow.go) instead of the outset offset fill, so box-shadow-inset:inset
// removes the (+8px, +8px) fill and keeps red fills inside the item box.
// Reference: Chrome 143.0.7499.40, 100px by 50px box.
func TestBehaviorBoxShadowInsetPaintsInnerRim(t *testing.T) {
	t.Parallel()

	outset := layoutHTML(t,
		behaviorOver2ShadowDoc("box-shadow-offset:8px 8px;box-shadow-color:red;box-shadow-inset:outset"))
	inset := layoutHTML(t,
		behaviorOver2ShadowDoc("box-shadow-offset:8px 8px;box-shadow-color:red;box-shadow-inset:inset"))

	outItem := boxByID(t, outset, "item")
	inItem := boxByID(t, inset, "item")

	if found := behaviorOver2FindFill(outset.Ops,
		outItem.x+pxToPt(8), outItem.y+pxToPt(8), outItem.w, outItem.height, 1, 0); found == nil {
		t.Errorf("no outset shadow fill at (+8px, +8px) for box-shadow-inset:outset")
	}

	if found := behaviorOver2FindFill(inset.Ops,
		inItem.x+pxToPt(8), inItem.y+pxToPt(8), inItem.w, inItem.height, 1, 0); found != nil {
		t.Errorf("outset fill %+v survives box-shadow-inset:inset, want inner rims only", *found)
	}

	inside := false

	for _, paintOp := range behaviorOver2FillRects(inset.Ops, 1, 0) {
		if paintOp.X >= inItem.x-0.01 && paintOp.Y >= inItem.y-0.01 &&
			paintOp.X+paintOp.W <= inItem.x+inItem.w+0.01 &&
			paintOp.Y+paintOp.H <= inItem.y+inItem.height+0.01 {
			inside = true

			break
		}
	}

	if !inside {
		t.Errorf("no red inset rim inside the item box for box-shadow-inset:inset")
	}
}

// TestBehaviorBoxShadowOffsetMovesLayer is box-shadow-offset: with longhand
// authoring the offsets position the shadow fill, so 16px offsets land the
// 100x50px red fill 8px further down and right than 8px offsets. Reference:
// Chrome 143.0.7499.40, 100px by 50px box.
func TestBehaviorBoxShadowOffsetMovesLayer(t *testing.T) {
	t.Parallel()

	near2 := layoutHTML(t, behaviorOver2ShadowDoc("box-shadow-offset:8px 8px;box-shadow-color:red"))
	far := layoutHTML(t, behaviorOver2ShadowDoc("box-shadow-offset:16px 16px;box-shadow-color:red"))

	nearItem := boxByID(t, near2, "item")
	farItem := boxByID(t, far, "item")

	if found := behaviorOver2FindFill(near2.Ops,
		nearItem.x+pxToPt(8), nearItem.y+pxToPt(8), nearItem.w, nearItem.height, 1, 0); found == nil {
		t.Errorf("no red shadow fill at (+8px, +8px) for box-shadow-offset:8px 8px")
	}

	if found := behaviorOver2FindFill(far.Ops,
		farItem.x+pxToPt(16), farItem.y+pxToPt(16), farItem.w, farItem.height, 1, 0); found == nil {
		t.Errorf("no red shadow fill at (+16px, +16px) for box-shadow-offset:16px 16px")
	}
}

// TestBehaviorBoxShadowPositionTogglesInset is box-shadow-position: inset
// selects the inner-rim path while outset keeps the offset fill (see
// ApplyBoxShadowPosition in box_shadow.go). Reference: Chrome 143.0.7499.40,
// 100px by 50px box, 8px by 8px shadow.
func TestBehaviorBoxShadowPositionTogglesInset(t *testing.T) {
	t.Parallel()

	outset := layoutHTML(t,
		behaviorOver2ShadowDoc("box-shadow-offset:8px 8px;box-shadow-color:red;box-shadow-position:outset"))
	inset := layoutHTML(t,
		behaviorOver2ShadowDoc("box-shadow-offset:8px 8px;box-shadow-color:red;box-shadow-position:inset"))

	outItem := boxByID(t, outset, "item")
	inItem := boxByID(t, inset, "item")

	if found := behaviorOver2FindFill(outset.Ops,
		outItem.x+pxToPt(8), outItem.y+pxToPt(8), outItem.w, outItem.height, 1, 0); found == nil {
		t.Errorf("no outset shadow fill at (+8px, +8px) for box-shadow-position:outset")
	}

	if found := behaviorOver2FindFill(inset.Ops,
		inItem.x+pxToPt(8), inItem.y+pxToPt(8), inItem.w, inItem.height, 1, 0); found != nil {
		t.Errorf("outset fill %+v survives box-shadow-position:inset, want inner rims only", *found)
	}

	if fills := behaviorOver2FillRects(inset.Ops, 1, 0); len(fills) == 0 {
		t.Errorf("no red inset fills for box-shadow-position:inset")
	}
}

// TestBehaviorBoxShadowSpreadGrowsLayer is box-shadow-spread: with longhand
// authoring the spread radius grows the shadow on every side (see
// appendOuterBoxShadow in box_shadow.go), so spread:10px on an 8px-offset
// 100x50px shadow paints a 120x70px fill at (-2px, -2px) from the item
// origin. Reference: Chrome 143.0.7499.40.
func TestBehaviorBoxShadowSpreadGrowsLayer(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, behaviorOver2ShadowDoc("box-shadow-offset:8px 8px;box-shadow-color:red"))
	spread := layoutHTML(t,
		behaviorOver2ShadowDoc("box-shadow-offset:8px 8px;box-shadow-color:red;box-shadow-spread:10px"))

	plainItem := boxByID(t, plain, "item")
	spreadItem := boxByID(t, spread, "item")

	if found := behaviorOver2FindFill(plain.Ops,
		plainItem.x+pxToPt(8), plainItem.y+pxToPt(8), pxToPt(100), pxToPt(50), 1, 0); found == nil {
		t.Errorf("no 100x50px shadow fill at (+8px, +8px) without box-shadow-spread")
	}

	if found := behaviorOver2FindFill(spread.Ops,
		spreadItem.x-pxToPt(2), spreadItem.y-pxToPt(2), pxToPt(120), pxToPt(70), 1, 0); found == nil {
		t.Errorf("no 120x70px shadow fill at (-2px, -2px) for box-shadow-spread:10px")
	}
}
