package layout

import (
	"fmt"
	"strings"
	"testing"
)

// Raster-observable cases for fill-rule and shape-rendering. Each test
// serializes an inline SVG twice (with and without the property), rasterizes
// both payloads, and asserts the used pixels differ, never stored strings.
// The pinned canvas parser (tdewolff/canvas svg.go setAttribute) has no case
// for either attribute, so raw baked output would rasterize identically; the
// in-tree bridge in internal/svg/raster.go (emulateDroppedPresentation)
// rewrites the geometry before parsing so the difference is observable. No
// vendoring or fork of the external module. Reference browser for every
// case: Chrome 143.0.7499.40. Shared helpers svgBakeRequireMem,
// svgBakeSerialize, svgBakeRaster, and svgBakeDiffPixels come from
// css_behavior_svg_bake_test.go and are never redeclared here.

// TestBehaviorSvgRasterFillRuleEvenoddCarvesHole is fill-rule:evenodd on a
// holed path of two nested same-winding squares. The default nonzero winding
// fills the inner square, while evenodd carves it into a hole, so used
// pixels differ. Reference: Chrome 143.0.7499.40 applies evenodd winding to
// holed paths.
func TestBehaviorSvgRasterFillRuleEvenoddCarvesHole(t *testing.T) {
	t.Parallel()
	svgBakeRequireMem(t)

	holed := `<path d="M4 4 L60 4 L60 24 L4 24 Z M20 8 L44 8 L44 20 L20 20 Z" style="%s"/>`
	filled := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(holed, "fill:#ff0000")))
	punched := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(holed, "fill:#ff0000;fill-rule:evenodd")))

	serialized := svgBakeSerialize(t, fmt.Sprintf(holed, "fill:#ff0000;fill-rule:evenodd"))
	if !strings.Contains(serialized, `fill-rule="evenodd"`) {
		t.Errorf("serialized SVG lacks baked fill-rule, got %s", serialized)
	}

	diff := svgBakeDiffPixels(t, filled, punched)
	t.Logf("fill-rule raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("fill-rule:evenodd raster identical to nonzero, want carved hole in used pixels")
	}
}

// TestBehaviorSvgRasterShapeRenderingCrispEdgesSnapsRect is
// shape-rendering:crispEdges on a fractional rect. The crisp hint rounds
// x/y/width/height to whole pixels while the default leaves the fractional
// edges antialiased, so used pixels differ. Reference: Chrome 143.0.7499.40
// snaps crispEdges geometry to device pixels.
func TestBehaviorSvgRasterShapeRenderingCrispEdgesSnapsRect(t *testing.T) {
	t.Parallel()
	svgBakeRequireMem(t)

	rect := `<rect x="2.5" y="2.5" width="59.5" height="23.5" style="%s"/>`
	soft := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(rect, "fill:#ff0000")))
	crisp := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(rect, "fill:#ff0000;shape-rendering:crispEdges")))

	serialized := svgBakeSerialize(t, fmt.Sprintf(rect, "fill:#ff0000;shape-rendering:crispEdges"))
	if !strings.Contains(serialized, `shape-rendering="crispEdges"`) {
		t.Errorf("serialized SVG lacks baked shape-rendering, got %s", serialized)
	}

	diff := svgBakeDiffPixels(t, soft, crisp)
	t.Logf("shape-rendering raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("shape-rendering:crispEdges raster identical to default, want snapped edges in used pixels")
	}
}
