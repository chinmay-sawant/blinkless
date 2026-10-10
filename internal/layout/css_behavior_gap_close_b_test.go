package layout

import (
	"math"
	"testing"
)

// Gap-B behavior closes. Each test covers one named partial property and
// asserts resolved used values or emitted display-list geometry, never stored
// strings. Reference browser for every case: Chrome 143.0.7499.40. Shared
// helpers boxByID, near, pxToPt, ptPerCSSPx, opsOfKind, layoutHTML,
// behaviorSvgAssertSameOps, and behaviorPaintFxXformed come from
// layout_test.go, css_review_02_test.go, css_behavior_svg_paint_test.go, and
// css_behavior_paint_effects_test.go and are never redeclared here.

// TestGapBTransformBoxContentBoxShiftsOrigin is transform-box: content-box
// resolves transform-origin percentages against the content box while
// border-box uses the border box, so the same 90deg rotation about 100% 100%
// bakes a different matrix when 10pt padding insets the content box.
// Reference: Chrome 143.0.7499.40 moves the painted box with the reference
// box.
func TestGapBTransformBoxContentBoxShiftsOrigin(t *testing.T) {
	t.Parallel()

	doc := func(box string) string {
		return `<html><body style="margin:0">` +
			`<div id="box" style="width:40pt;height:20pt;padding:10pt;background:red;` +
			`transform:rotate(90deg);transform-origin:100% 100%;transform-box:` + box + `"></div></body></html>`
	}

	borderRes := layoutHTML(t, doc("border-box"))
	contentRes := layoutHTML(t, doc("content-box"))

	borderBox := boxByID(t, borderRes, "box")
	contentBox := boxByID(t, contentRes, "box")

	if borderBox.style.TransformBox != "border-box" {
		t.Errorf("transform-box used = %q, want %q", borderBox.style.TransformBox, "border-box")
	}

	if contentBox.style.TransformBox != "content-box" {
		t.Errorf("transform-box used = %q, want %q", contentBox.style.TransformBox, "content-box")
	}

	borderMoved := behaviorPaintFxXformed(borderRes)
	contentMoved := behaviorPaintFxXformed(contentRes)

	if len(borderMoved) == 0 || len(contentMoved) == 0 {
		t.Fatalf("want transformed ops for both boxes, got border=%d content=%d",
			len(borderMoved), len(contentMoved))
	}

	b, c := borderMoved[0].Transform(), contentMoved[0].Transform()
	if math.Abs(b.E-c.E) < 1 && math.Abs(b.F-c.F) < 1 {
		t.Errorf("content-box shift too small: border E,F=(%.3f,%.3f) content E,F=(%.3f,%.3f)",
			b.E, b.F, c.E, c.F)
	}
}

// TestGapBTransformBoxFillViewAliasPinsViewportGap is transform-box fill-box
// vs view-box on a CSS layout box: both alias the border box (there is no SVG
// viewport in the 2D box stamper), so the used values store canonically but
// the baked geometry matches. GAP: an SVG viewport reference box needs the
// nearest viewport carried into boxTransformAccum (transform.go). Reported
// with used values, never failing. Reference: Chrome 143.0.7499.40 would
// rotate about the object bounding box vs the nearest SVG viewport.
func TestGapBTransformBoxFillViewAliasPinsViewportGap(t *testing.T) {
	t.Parallel()

	doc := func(box string) string {
		return `<html><body style="margin:0">` +
			`<div id="box" style="width:40pt;height:20pt;background:red;` +
			`transform:rotate(90deg);transform-origin:0 0;transform-box:` + box + `"></div></body></html>`
	}

	fillRes := layoutHTML(t, doc("fill-box"))
	viewRes := layoutHTML(t, doc("view-box"))

	if got := boxByID(t, fillRes, "box").style.TransformBox; got != "fill-box" {
		t.Errorf("transform-box used = %q, want fill-box", got)
	}

	if got := boxByID(t, viewRes, "box").style.TransformBox; got != "view-box" {
		t.Errorf("transform-box used = %q, want view-box", got)
	}

	behaviorSvgAssertSameOps(t, fillRes, viewRes)

	bogus := boxByID(t, layoutHTML(t, doc("bogus")), "box")
	if bogus.style.TransformBox != "view-box" {
		t.Errorf("transform-box:bogus used = %q, want initial view-box", bogus.style.TransformBox)
	}
}

// TestGapBTransformStyleStoresFlatVsPreserve3D is transform-style: flat and
// preserve-3d store canonically but the 2D engine flattens both, so nested
// rotations paint identical geometry. GAP: 3D planes are a permanent print
// non-goal. Reported with used values, never failing. Reference: Chrome
// 143.0.7499.40 would keep the child in the parent 3D plane under
// preserve-3d.
func TestGapBTransformStyleStoresFlatVsPreserve3D(t *testing.T) {
	t.Parallel()

	doc := func(style string) string {
		return `<html><body style="margin:0">` +
			`<div id="outer" style="width:100px;height:100px;transform:rotate(10deg);` +
			`transform-style:` + style + `">` +
			`<div id="inner" style="width:50px;height:50px;background-color:#00ff00;` +
			`transform:rotate(5deg)"></div></div></body></html>`
	}

	flat := layoutHTML(t, doc("flat"))
	preserved := layoutHTML(t, doc("preserve-3d"))

	if got := boxByID(t, flat, "outer").style.TransformStyle; got != "flat" {
		t.Errorf("transform-style used = %q, want flat", got)
	}

	if got := boxByID(t, preserved, "outer").style.TransformStyle; got != "preserve-3d" {
		t.Errorf("transform-style used = %q, want preserve-3d", got)
	}

	behaviorSvgAssertSameOps(t, flat, preserved)

	bogus := boxByID(t, layoutHTML(t, doc("bogus")), "outer")
	if bogus.style.TransformStyle != "flat" {
		t.Errorf("transform-style:bogus used = %q, want initial flat", bogus.style.TransformStyle)
	}
}

// TestGapBFillRuleStoresAndInherits is fill-rule: evenodd stores canonically,
// defaults to nonzero, and inherits from the parent. The external canvas
// rasterizer (tdewolff/canvas via internal/svg) has no fill-rule case even
// though its core Style carries a FillRule field, so serialized SVG cannot
// observe it and clip masks keep using ClipRule. GAP pinned with used values,
// never failing. Reference: Chrome 143.0.7499.40 would apply evenodd winding
// to holed paths.
func TestGapBFillRuleStoresAndInherits(t *testing.T) {
	t.Parallel()

	plain := boxByID(t, layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:10pt;height:10pt"></div></body></html>`), "a")
	if plain.style.FillRule != "nonzero" {
		t.Errorf("fill-rule initial used = %q, want nonzero", plain.style.FillRule)
	}

	ruled := boxByID(t, layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:10pt;height:10pt;fill-rule:evenodd"></div></body></html>`), "a")
	if ruled.style.FillRule != "evenodd" {
		t.Errorf("fill-rule used = %q, want evenodd", ruled.style.FillRule)
	}

	child := boxByID(t, layoutHTML(t, `<html><body style="margin:0">`+
		`<div style="fill-rule:evenodd"><div id="a" style="width:10pt;height:10pt"></div></div></body></html>`), "a")
	if child.style.FillRule != "evenodd" {
		t.Errorf("fill-rule inherited used = %q, want evenodd", child.style.FillRule)
	}

	bogus := boxByID(t, layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:10pt;height:10pt;fill-rule:bogus"></div></body></html>`), "a")
	if bogus.style.FillRule != "nonzero" {
		t.Errorf("fill-rule:bogus used = %q, want initial nonzero", bogus.style.FillRule)
	}
}

// TestGapBShapeRenderingStoresHint is shape-rendering: crispEdges stores
// canonically (lowercased) and defaults to auto. The external canvas
// rasterizer has no shape-rendering case, so the hint bakes into serialized
// SVG but cannot change raster pixels. GAP pinned with used values, never
// failing. Reference: Chrome 143.0.7499.40 would snap edges with crispEdges.
func TestGapBShapeRenderingStoresHint(t *testing.T) {
	t.Parallel()

	plain := boxByID(t, layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:10pt;height:10pt"></div></body></html>`), "a")
	if plain.style.ShapeRendering != "auto" {
		t.Errorf("shape-rendering initial used = %q, want auto", plain.style.ShapeRendering)
	}

	crisp := boxByID(t, layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:10pt;height:10pt;shape-rendering:crispEdges"></div></body></html>`), "a")
	if crisp.style.ShapeRendering != "crispedges" {
		t.Errorf("shape-rendering used = %q, want crispedges", crisp.style.ShapeRendering)
	}

	child := boxByID(t, layoutHTML(t, `<html><body style="margin:0">`+
		`<div style="shape-rendering:crispEdges"><div id="a" style="width:10pt;height:10pt"></div></div></body></html>`), "a")
	if child.style.ShapeRendering != "crispedges" {
		t.Errorf("shape-rendering inherited used = %q, want crispedges", child.style.ShapeRendering)
	}

	bogus := boxByID(t, layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:10pt;height:10pt;shape-rendering:bogus"></div></body></html>`), "a")
	if bogus.style.ShapeRendering != "auto" {
		t.Errorf("shape-rendering:bogus used = %q, want initial auto", bogus.style.ShapeRendering)
	}
}

// TestGapBBackfaceVisibilityStoresHidden is backface-visibility: hidden
// stores canonically but the 2D engine has no 3D back face to hide, so it
// paints identical geometry to visible under the same rotation. GAP pinned
// with used values, never failing. Reference: Chrome 143.0.7499.40 would hide
// the back face in 3D.
func TestGapBBackfaceVisibilityStoresHidden(t *testing.T) {
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

	behaviorSvgAssertSameOps(t, front, hidden)

	bogus := boxByID(t, layoutHTML(t, doc("bogus")), "a")
	if bogus.style.BackfaceVisibility != "visible" {
		t.Errorf("backface-visibility:bogus used = %q, want initial visible", bogus.style.BackfaceVisibility)
	}
}

// TestGapBDominantBaselineHangingRaisesText is dominant-baseline: hanging
// stores canonically and raises the 16px run above alphabetic by the hanging
// ratio (about 5px in Chrome 143.0.7499.40), while auto keeps the baseline.
// Reference: Chrome 143.0.7499.40 aligns the hanging run about 5px above
// alphabetic at 16px.
func TestGapBDominantBaselineHangingRaisesText(t *testing.T) {
	t.Parallel()

	doc := func(decl string) string {
		return `<html><body style="margin:0">` +
			`<p style="margin:0;font-size:16px;` + decl + `">Ag</p></body></html>`
	}

	plainRes := layoutHTML(t, doc(""))
	hangRes := layoutHTML(t, doc("dominant-baseline:hanging"))

	plainTexts := opsOfKind(plainRes, OpText)
	hangTexts := opsOfKind(hangRes, OpText)

	if len(plainTexts) == 0 || len(hangTexts) == 0 {
		t.Fatalf("want text ops, got plain=%d hang=%d", len(plainTexts), len(hangTexts))
	}

	plainBoxCheck := boxByID(t, layoutHTML(t, `<html><body style="margin:0">`+
		`<p id="p" style="margin:0;font-size:16px">Ag</p></body></html>`), "p")
	_ = plainBoxCheck

	hangStyle := boxByID(t, layoutHTML(t, `<html><body style="margin:0">`+
		`<p id="p" style="margin:0;font-size:16px;dominant-baseline:hanging">Ag</p></body></html>`), "p")
	if hangStyle.style.DominantBaseline != "hanging" {
		t.Errorf("dominant-baseline used = %q, want hanging", hangStyle.style.DominantBaseline)
	}

	defStyle := boxByID(t, layoutHTML(t, `<html><body style="margin:0">`+
		`<p id="p" style="margin:0;font-size:16px">Ag</p></body></html>`), "p")
	if defStyle.style.DominantBaseline != "auto" {
		t.Errorf("dominant-baseline initial used = %q, want auto", defStyle.style.DominantBaseline)
	}

	lift := plainTexts[0].Y - hangTexts[0].Y
	want := pxToPt(5)

	if lift < 2 || lift > 6 {
		t.Errorf("hanging lift = %.4fpt (%.2fpx), want about 5px (%.4fpt)",
			lift, lift/ptPerCSSPx, want)
	}

	if !near(lift, want) {
		t.Errorf("hanging lift = %.4fpt, want %.4fpt (5px at 16px)", lift, want)
	}
}
