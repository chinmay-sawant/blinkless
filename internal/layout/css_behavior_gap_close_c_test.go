package layout

import (
	"image"
	"strings"
	"testing"
)

// Gap-C behavior closes and pins. Each test asserts resolved used values
// (raster pixels, baked matrices, line breaks), never stored strings.
// Reference browser for every case: Chrome 143.0.7499.40. Shared helpers
// layoutHTML, boxByID, behaviorPaintFxXformed, svgBakeSerialize,
// svgBakeRaster, svgBakeDiffPixels, svgBakeRequireMem, and near come from
// layout_test.go, css_behavior_paint_effects_test.go,
// css_behavior_svg_paint_test.go, and css_behavior_svg_bake_test.go and are
// never redeclared here.

// gapCHoledPathD is one path with two same-winding nested rect subpaths:
// an outer 44x24 rect holding an inner 20x12 hole. Under nonzero winding
// the hole fills; under evenodd it stays empty.
const gapCHoledPathD = "M10 10 H54 V34 H10 Z M22 16 H42 V28 H22 Z"

// TestGapCFillRuleEvenOddHolesRaster is fill-rule: evenodd on a holed path
// rasterizes the hole transparent while nonzero fills it, so used pixels
// differ. The pinned canvas parser has no fill-rule case in its style
// switch, so internal/svg emulates evenodd by reversing the inner subpath
// winding before ParseSVG (raster.go). Reference: Chrome 143.0.7499.40
// applies evenodd winding to holed paths.
func TestGapCFillRuleEvenOddHolesRaster(t *testing.T) {
	t.Parallel()
	svgBakeRequireMem(t)

	path := func(rule string) string {
		return `<path d="` + gapCHoledPathD + `" style="fill:#ff0000;fill-rule:` + rule + `"/>`
	}

	serialized := svgBakeSerialize(t, path("evenodd"))
	if !strings.Contains(serialized, `fill-rule="evenodd"`) {
		t.Fatalf("serialized SVG lacks baked fill-rule, got %s", serialized)
	}

	nonzero := svgBakeRaster(t, svgBakeSerialize(t, path("nonzero")))
	evenodd := svgBakeRaster(t, serialized)

	diff := svgBakeDiffPixels(t, nonzero, evenodd)
	t.Logf("fill-rule raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("fill-rule:evenodd raster identical to nonzero, want holed pixels in used output")
	}

	// The inner hole spans x 22..42, y 16..28 of the 64x28 viewBox, so its
	// center sits at (32/64, 22/28) of either raster bounds.
	holePixel := func(img image.Image) [4]uint32 {
		bounds := img.Bounds()
		x := bounds.Min.X + bounds.Dx()*32/64
		y := bounds.Min.Y + bounds.Dy()*22/28
		r, g, bl, a := img.At(x, y).RGBA()

		return [4]uint32{r, g, bl, a}
	}

	nonzeroHole := holePixel(nonzero)
	evenoddHole := holePixel(evenodd)

	if nonzeroHole[0] < 0x8000 || nonzeroHole[3] == 0 {
		t.Errorf("nonzero hole center = %04x alpha %04x, want filled red", nonzeroHole[0], nonzeroHole[3])
	}

	if evenoddHole[3] != 0 {
		t.Errorf("evenodd hole center alpha = %04x, want transparent hole", evenoddHole[3])
	}
}

// TestGapCShapeRenderingCrispSnapsRaster is shape-rendering: crispEdges on a
// half-pixel rect snaps its edges to whole pixels while auto keeps the
// fractional position, so used pixels differ. The pinned canvas parser has
// no shape-rendering case, so internal/svg rounds crisp rect/circle/ellipse
// geometry to whole pixels before ParseSVG (raster.go). Reference: Chrome
// 143.0.7499.40 snaps edges with crispEdges.
func TestGapCShapeRenderingCrispSnapsRaster(t *testing.T) {
	t.Parallel()
	svgBakeRequireMem(t)

	rect := func(extra string) string {
		return `<rect x="2.5" y="2.5" width="60" height="24" style="fill:#ff0000` + extra + `"/>`
	}

	serialized := svgBakeSerialize(t, rect(";shape-rendering:crispEdges"))
	if !strings.Contains(serialized, `shape-rendering="crispEdges"`) {
		t.Fatalf("serialized SVG lacks baked shape-rendering, got %s", serialized)
	}

	auto := svgBakeRaster(t, svgBakeSerialize(t, rect("")))
	crisp := svgBakeRaster(t, serialized)

	diff := svgBakeDiffPixels(t, auto, crisp)
	t.Logf("shape-rendering raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("shape-rendering:crispEdges raster identical to auto, want snapped edges in used pixels")
	}
}

// TestGapCTransformBoxFillViewShareBorderOrigin pins the transform-box
// viewport gap: fill-box and view-box on a CSS layout box bake identical
// matrices because both fall back to the border box. Per CSS Transforms 2,
// fill-box uses the object bounding box (the border box for CSS boxes) and
// view-box uses the nearest SVG viewport, which a plain HTML div has none
// of, so it falls back to the border box too; Chrome 143.0.7499.40 paints
// them identically. Divergence would need the nearest SVG viewport carried
// into boxTransformAccum (transform.go), which the 2D box stamper does not
// carry. Reported with used matrices, never failing.
func TestGapCTransformBoxFillViewShareBorderOrigin(t *testing.T) {
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

	fillMoved := behaviorPaintFxXformed(fillRes)
	viewMoved := behaviorPaintFxXformed(viewRes)

	if len(fillMoved) == 0 || len(viewMoved) == 0 {
		t.Fatalf("want transformed ops for both boxes, got fill=%d view=%d",
			len(fillMoved), len(viewMoved))
	}

	fill, view := fillMoved[0].Transform(), viewMoved[0].Transform()

	if !near(fill.E, view.E) || !near(fill.F, view.F) {
		t.Errorf("fill-box E,F=(%.3f,%.3f) vs view-box E,F=(%.3f,%.3f), want identical border-box origin",
			fill.E, fill.F, view.E, view.F)
	}
}

// TestGapCPrettyAvoidsOrphanTail is text-wrap-style: pretty at helper
// level: four 40pt words in a 130pt box wrap 3+1 greedily, and pretty
// re-breaks them 2+2 at a narrower width so the last line keeps two words.
// Reference: Chrome 143.0.7499.40 pulls another word down instead of
// leaving a one-word orphan.
func TestGapCPrettyAvoidsOrphanTail(t *testing.T) {
	t.Parallel()

	items := []inlineItem{
		{text: "a", w: 40},
		{text: "b", w: 40},
		{text: "c", w: 40},
		{text: "d", w: 40},
	}

	greedy, packOK := prettyPackLines(items, 130)
	if !packOK || len(greedy) != 2 {
		t.Fatalf("prettyPackLines(130) lines = %+v, %v; want 2 lines", greedy, packOK)
	}

	if greedy[1].words != 1 {
		t.Fatalf("greedy last line words = %d, want 1 (orphan fixture)", greedy[1].words)
	}

	got := prettyLineWidth(items, 130, 1)
	if got <= 0 || got >= 130 {
		t.Fatalf("prettyLineWidth(130) = %.2f, want a narrower width in (0, 130)", got)
	}

	fixed, fixOK := prettyPackLines(items, got)
	if !fixOK || len(fixed) != len(greedy) {
		t.Fatalf("prettyPackLines(%.2f) lines = %+v, %v; want %d lines", got, fixed, fixOK, len(greedy))
	}

	if fixed[len(fixed)-1].words < prettyMinLastWords {
		t.Errorf("pretty last line words = %d, want at least %d", fixed[len(fixed)-1].words, prettyMinLastWords)
	}
}

// TestGapCPrettyKeepsGoodBreaks pins the pretty no-op sides: an already
// even tail, a single line, an unfittable width, and a forced break all
// return 0 so greedy stands.
func TestGapCPrettyKeepsGoodBreaks(t *testing.T) {
	t.Parallel()

	items := []inlineItem{
		{text: "a", w: 40},
		{text: "b", w: 40},
		{text: "c", w: 40},
		{text: "d", w: 40},
	}

	if got := prettyLineWidth(items, 90, 0.5); got != 0 {
		t.Errorf("prettyLineWidth(90, even 2+2) = %.2f, want 0", got)
	}

	if got := prettyLineWidth(items, 200, 0.5); got != 0 {
		t.Errorf("prettyLineWidth(200, single line) = %.2f, want 0", got)
	}

	if got := prettyLineWidth(items, 30, 0.5); got != 0 {
		t.Errorf("prettyLineWidth(30, unfittable) = %.2f, want 0", got)
	}

	broken := []inlineItem{{text: "a", w: 40}, {forceBreak: true}, {text: "b", w: 40}}
	if got := prettyLineWidth(broken, 130, 0.5); got != 0 {
		t.Errorf("prettyLineWidth with forced break = %.2f, want 0", got)
	}
}

// TestGapCPrettyApplyGates mirrors the balance gates: floats, line clamp,
// indents, and non-pretty styles opt out.
func TestGapCPrettyApplyGates(t *testing.T) {
	t.Parallel()

	pretty := &ResolvedStyle{TextWrapStyle: "pretty"}
	balance := &ResolvedStyle{TextWrapStyle: "balance"}

	if !prettyCanApply(nil, 0, pretty) {
		t.Error("prettyCanApply(nil, 0, pretty) = false, want true")
	}

	if prettyCanApply(&floatState{hasLeft: true}, 0, pretty) {
		t.Error("prettyCanApply with active float = true, want false")
	}

	if prettyCanApply(nil, 2, pretty) {
		t.Error("prettyCanApply with line clamp = true, want false")
	}

	indented := &ResolvedStyle{TextWrapStyle: "pretty", TextIndent: 12}
	if prettyCanApply(nil, 0, indented) {
		t.Error("prettyCanApply with indent = true, want false")
	}

	if prettyCanApply(nil, 0, balance) {
		t.Error("prettyCanApply(balance style) = true, want false")
	}

	if prettyCanApply(nil, 0, nil) {
		t.Error("prettyCanApply(nil style) = true, want false")
	}
}

// TestGapCPrettyWiringPinsCascadeGap pins the pretty end-to-end wiring gap:
// the cascade acceptance table still rejects pretty, so a pretty paragraph
// lays out exactly like normal greedy text. The prettyLineWidth helper above
// proves the break logic; reaching it from CSS needs the acceptance flip
// (style_value_accept.go) plus a pack-loop hook beside balanceCanApply
// (inline.go, forbidden here). Reported with used line breaks, never
// failing. Reference: Chrome 143.0.7499.40 would re-break the tail.
