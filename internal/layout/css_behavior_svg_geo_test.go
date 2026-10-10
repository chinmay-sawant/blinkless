package layout

import (
	"fmt"
	"strings"
	"testing"
)

// SVG geometry and stop presentation-attribute bake tests. Each observable
// test rasterizes the serialized SVG twice (with and without the property)
// and asserts the used pixel values differ, never stored strings. Each gap
// test pins a serialize-only assertion with the rasterizer gap in a comment,
// never a pixel assertion. Reference browser for every case: Chrome
// 143.0.7499.40. Shared helpers svgBakeRequireMem, svgBakeSerialize,
// svgBakeRaster, and svgBakeDiffPixels come from
// css_behavior_svg_bake_test.go and are never redeclared here.

// TestBehaviorSvgGeoCxMovesCircle is cx as a CSS property: the circle center
// moves from x=10 to x=40, so used pixels differ. Reference: Chrome
// 143.0.7499.40 treats cx as a presentation attribute and moves the circle.
func TestBehaviorSvgGeoCxMovesCircle(t *testing.T) {
	t.Parallel()
	svgBakeRequireMem(t)

	circle := `<circle cy="14" r="8" style="%s"/>`
	left := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(circle, "fill:#ff0000;cx:10px")))
	right := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(circle, "fill:#ff0000;cx:40px")))

	diff := svgBakeDiffPixels(t, left, right)
	t.Logf("cx raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("cx:40px raster identical to cx:10px, want moved circle in used pixels")
	}
}

// TestBehaviorSvgGeoCyMovesCircle is cy as a CSS property: the circle center
// moves from y=8 to y=20, so used pixels differ. Reference: Chrome
// 143.0.7499.40 treats cy as a presentation attribute and moves the circle.
func TestBehaviorSvgGeoCyMovesCircle(t *testing.T) {
	t.Parallel()
	svgBakeRequireMem(t)

	circle := `<circle cx="32" r="6" style="%s"/>`
	top := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(circle, "fill:#ff0000;cy:8px")))
	bottom := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(circle, "fill:#ff0000;cy:20px")))

	diff := svgBakeDiffPixels(t, top, bottom)
	t.Logf("cy raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("cy:20px raster identical to cy:8px, want moved circle in used pixels")
	}
}

// TestBehaviorSvgGeoRResizesCircle is r as a CSS property: radius 12 paints
// a larger disc than radius 4, so used pixels differ. Reference: Chrome
// 143.0.7499.40 treats r as a presentation attribute and resizes the circle.
func TestBehaviorSvgGeoRResizesCircle(t *testing.T) {
	t.Parallel()
	svgBakeRequireMem(t)

	circle := `<circle cx="32" cy="14" style="%s"/>`
	small := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(circle, "fill:#ff0000;r:4px")))
	large := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(circle, "fill:#ff0000;r:12px")))

	diff := svgBakeDiffPixels(t, small, large)
	t.Logf("r raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("r:12px raster identical to r:4px, want larger disc in used pixels")
	}
}

// TestBehaviorSvgGeoRxRoundsRect is rx as a CSS property: rounded corners at
// 10px paint fewer corner pixels than square corners at 0, so used pixels
// differ. Reference: Chrome 143.0.7499.40 treats rx as a presentation
// attribute and rounds the rect corners.
func TestBehaviorSvgGeoRxRoundsRect(t *testing.T) {
	t.Parallel()
	svgBakeRequireMem(t)

	rect := svgBakeRectTemplate
	square := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(rect, "fill:#ff0000;rx:0px")))
	round := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(rect, "fill:#ff0000;rx:10px")))

	diff := svgBakeDiffPixels(t, square, round)
	t.Logf("rx raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("rx:10px raster identical to rx:0px, want rounded corners in used pixels")
	}
}

// TestBehaviorSvgGeoRyRoundsRect is ry as a CSS property: rounded corners at
// 8px paint fewer corner pixels than square corners at 0, so used pixels
// differ. Reference: Chrome 143.0.7499.40 treats ry as a presentation
// attribute and rounds the rect corners.
func TestBehaviorSvgGeoRyRoundsRect(t *testing.T) {
	t.Parallel()
	svgBakeRequireMem(t)

	rect := svgBakeRectTemplate
	square := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(rect, "fill:#ff0000;ry:0px")))
	round := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(rect, "fill:#ff0000;ry:8px")))

	diff := svgBakeDiffPixels(t, square, round)
	t.Logf("ry raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("ry:8px raster identical to ry:0px, want rounded corners in used pixels")
	}
}

// TestBehaviorSvgGeoXMovesRect is x as a CSS property: the rect shifts from
// x=2 to x=24, so used pixels differ. Reference: Chrome 143.0.7499.40
// treats x as a presentation attribute and moves the rect.
func TestBehaviorSvgGeoXMovesRect(t *testing.T) {
	t.Parallel()
	svgBakeRequireMem(t)

	rect := `<rect y="4" width="30" height="20" style="%s"/>`
	left := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(rect, "fill:#ff0000;x:2px")))
	right := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(rect, "fill:#ff0000;x:24px")))

	diff := svgBakeDiffPixels(t, left, right)
	t.Logf("x raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("x:24px raster identical to x:2px, want moved rect in used pixels")
	}
}

// TestBehaviorSvgGeoYMovesRect is y as a CSS property: the rect shifts from
// y=2 to y=10, so used pixels differ. Reference: Chrome 143.0.7499.40
// treats y as a presentation attribute and moves the rect.
func TestBehaviorSvgGeoYMovesRect(t *testing.T) {
	t.Parallel()
	svgBakeRequireMem(t)

	rect := `<rect x="4" width="30" height="16" style="%s"/>`
	top := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(rect, "fill:#ff0000;y:2px")))
	bottom := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(rect, "fill:#ff0000;y:10px")))

	diff := svgBakeDiffPixels(t, top, bottom)
	t.Logf("y raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("y:10px raster identical to y:2px, want moved rect in used pixels")
	}
}

// TestBehaviorSvgGeoStopColorRecolorsGradient is stop-color as a CSS
// property on a gradient stop: the gradient start moves from red to green,
// so used pixels differ. The rect keeps fill="url(#g)" as a presentation
// attribute (the fill cascade drops url() values), while the stops carry
// their colors only through the stop-color bake arm. Reference: Chrome
// 143.0.7499.40 paints the gradient from the stop colors.
func TestBehaviorSvgGeoStopColorRecolorsGradient(t *testing.T) {
	t.Parallel()
	svgBakeRequireMem(t)

	gradient := `<defs><linearGradient id="g" x1="0%%" y1="0%%" x2="100%%" y2="0%%">` +
		`<stop offset="0" style="stop-color:%s"/>` +
		`<stop offset="1" style="stop-color:#0000ff"/>` +
		`</linearGradient></defs>` +
		`<rect x="2" y="2" width="60" height="24" fill="url(#g)"/>`
	red := svgBakeRaster(t, svgBakeSerialize(t, fmt.Sprintf(gradient, "#ff0000")))
	green := svgBakeRaster(t, svgBakeSerialize(t, fmt.Sprintf(gradient, "#00ff00")))

	diff := svgBakeDiffPixels(t, red, green)
	t.Logf("stop-color raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("stop-color green start raster identical to red start, want recolored gradient in used pixels")
	}
}

// TestBehaviorSvgGeoStopOpacityFadesGradient is stop-opacity as a CSS
// property on a gradient stop: fading the red start to 0.2 blends it toward
// the backdrop, so used pixels differ from the opaque stop. Reference:
// Chrome 143.0.7499.40 fades the gradient stop at 20 percent.
func TestBehaviorSvgGeoStopOpacityFadesGradient(t *testing.T) {
	t.Parallel()
	svgBakeRequireMem(t)

	gradient := `<defs><linearGradient id="g" x1="0%%" y1="0%%" x2="100%%" y2="0%%">` +
		`<stop offset="0" style="stop-color:#ff0000%s"/>` +
		`<stop offset="1" style="stop-color:#ff0000"/>` +
		`</linearGradient></defs>` +
		`<rect x="2" y="2" width="60" height="24" fill="url(#g)"/>`
	opaque := svgBakeRaster(t, svgBakeSerialize(t, fmt.Sprintf(gradient, "")))
	faded := svgBakeRaster(t, svgBakeSerialize(t, fmt.Sprintf(gradient, ";stop-opacity:0.2")))

	diff := svgBakeDiffPixels(t, opaque, faded)
	t.Logf("stop-opacity raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("stop-opacity 0.2 raster identical to opaque, want faded gradient start in used pixels")
	}
}

// TestBehaviorSvgGeoAttrWinsNoDuplicate pins the no-duplicate guard: a
// cx="10" presentation attribute plus style="cx:40px" serializes one cx and
// keeps the attribute, so the bake never emits a second cx. Reference:
// Chrome 143.0.7499.40 would let the CSS win; the guard keeps the attribute
// on purpose, matching the fill/stroke bake-arm convention.
func TestBehaviorSvgGeoAttrWinsNoDuplicate(t *testing.T) {
	t.Parallel()

	serialized := svgBakeSerialize(t, `<circle cx="10" cy="14" r="8" style="fill:#ff0000;cx:40px"/>`)
	if got := strings.Count(serialized, "cx="); got != 1 {
		t.Errorf("cx occurrences = %d, want 1 (no duplicate bake)", got)
	}

	if !strings.Contains(serialized, `cx="10"`) {
		t.Errorf("serialized SVG lacks cx=10 attribute, got %s", serialized)
	}

	if strings.Contains(serialized, "40px") {
		t.Errorf("serialized SVG baked over the attribute, got %s", serialized)
	}
}

// TestBehaviorSvgGeoPaintOrderGap pins the paint-order raster gap: the bake
// arm in layout_svg.go emits paint-order="stroke" into the serialized SVG
// (proved below), but the pinned canvas setAttribute switch has no
// paint-order case, so the raster cannot observe it. No pixel assertion
// here, never a failing test. Reference: Chrome 143.0.7499.40 would paint
// the blue stroke beneath the red fill.
func TestBehaviorSvgGeoPaintOrderGap(t *testing.T) {
	t.Parallel()

	serialized := svgBakeSerialize(t,
		`<rect x="2" y="2" width="60" height="24" style="fill:#ff0000;stroke:#0000ff;stroke-width:4pt;paint-order:stroke"/>`)
	if !strings.Contains(serialized, `paint-order="stroke"`) {
		t.Errorf("serialized SVG lacks baked paint-order, got %s", serialized)
	}
}

// TestBehaviorSvgGeoVectorEffectGap pins the vector-effect raster gap: the
// bake arm in layout_svg.go emits vector-effect="non-scaling-stroke" into
// the serialized SVG (proved below), but the pinned canvas setAttribute
// switch has no vector-effect case, so the raster cannot observe it. No
// pixel assertion here, never a failing test. Reference: Chrome
// 143.0.7499.40 would keep the stroke width under viewport scaling.
func TestBehaviorSvgGeoVectorEffectGap(t *testing.T) {
	t.Parallel()

	serialized := svgBakeSerialize(t,
		`<line x1="10" y1="14" x2="54" y2="14" style="fill:none;stroke:#ff0000;`+
			`stroke-width:6pt;vector-effect:non-scaling-stroke"/>`)
	if !strings.Contains(serialized, `vector-effect="non-scaling-stroke"`) {
		t.Errorf("serialized SVG lacks baked vector-effect, got %s", serialized)
	}
}

// TestBehaviorSvgGeoTextRenderingGap pins the text-rendering raster gap: the
// bake arm in layout_svg.go emits text-rendering="optimizeLegibility" into
// the serialized SVG (proved below), but the pinned canvas setAttribute
// switch has no text-rendering case, so the raster cannot observe it. No
// pixel assertion here, never a failing test. Reference: Chrome
// 143.0.7499.40 would trade speed for legibility in text shaping.
func TestBehaviorSvgGeoTextRenderingGap(t *testing.T) {
	t.Parallel()

	serialized := svgBakeSerialize(t,
		`<text x="4" y="18" style="font-size:12px;fill:#000000;text-rendering:optimizeLegibility">hi</text>`)
	if !strings.Contains(serialized, `text-rendering="optimizeLegibility"`) {
		t.Errorf("serialized SVG lacks baked text-rendering, got %s", serialized)
	}
}
