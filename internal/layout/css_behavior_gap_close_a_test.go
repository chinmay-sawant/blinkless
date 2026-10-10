package layout

import (
	"testing"
)

// Gap-close A: precise pins for six partial properties whose consumers live
// in forbidden files or a frozen bake region, so no honest paint change fits
// the ownership rules. Each test asserts USED drawing-list geometry (op kind
// plus X/Y/W/H and text), never stored style strings. Reference browser for
// every case: Chrome 143.0.7499.40. Shared helpers layoutHTML,
// behaviorText2AssertSameOps, behaviorSvgLayoutSVG, and
// behaviorSvgAssertSameOps come from other files in this package and are
// reused here, never redeclared.

// TestGapCloseATextAnchorMiddlePinsSvgBakeGap pins text-anchor:middle as a
// no-op for HTML and unbaked for SVG. The declaration is forwarded by
// internal/layout/style_paint_props.go:89-96 to applyLeftoversProps in
// internal/layout/style_leftovers.go:11-41 which has no text-anchor case and
// returns false, so the value is dropped before paint. Baking it for SVG
// text would need the frozen bake arms in
// internal/layout/layout_svg.go:169-260 which this task must not edit.
// Chrome 143.0.7499.40 middle-anchors SVG text only, so HTML stays identical
// and this pin records identical SVG rect ops and moves on.
func TestGapCloseATextAnchorMiddlePinsSvgBakeGap(t *testing.T) {
	t.Parallel()

	plain := behaviorSvgLayoutSVG(t, "fill:#ff0000")
	anchored := behaviorSvgLayoutSVG(t, "fill:#ff0000;text-anchor:middle")

	behaviorSvgAssertSameOps(t, plain, anchored)
}

// TestGapCloseAAlignmentBaselineMiddlePinsBaselineGap pins
// alignment-baseline:middle as a no-op. The declaration is forwarded by
// internal/layout/style_paint_props.go:89-96 to applyLeftoversProps in
// internal/layout/style_leftovers.go:11-41 which has no alignment-baseline
// case and returns false. No baseline model reads it: the inline baseline
// comes from internal/layout/inline.go:1186-1197 (firstBaseline) and the
// paint shift reads only vertical-align at
// internal/layout/inline_vertical_align.go:49-71 and
// internal/layout/inline_paint.go:170-173. Chrome 143.0.7499.40 would shift
// the text baseline, but that needs the forbidden paint path, so this pin
// records identical ops and moves on.
func TestGapCloseAAlignmentBaselineMiddlePinsBaselineGap(t *testing.T) {
	t.Parallel()

	plain := behaviorSvgLayoutSVG(t, "fill:#ff0000")
	shifted := behaviorSvgLayoutSVG(t, "fill:#ff0000;alignment-baseline:middle")

	behaviorSvgAssertSameOps(t, plain, shifted)
}

// TestGapCloseAColorInterpolationSRGBPinsRasterGap pins color-interpolation
// as a no-op. The declaration is forwarded by
// internal/layout/style_paint_props.go:89-96 to applyLeftoversProps in
// internal/layout/style_leftovers.go:11-41 which has no color-interpolation
// case and returns false. No interpolation reader exists: CSS gradients in
// internal/layout/gradient.go:366-393 sample stops in sRGB gamma directly,
// filter.go has no color-space read, and the SVG rasterizer in
// internal/svg/raster.go delegates to the external canvas library. Chrome
// 143.0.7499.40 would interpolate SVG gradients in sRGB instead, but that
// needs the frozen bake arms in internal/layout/layout_svg.go:169-260, so
// this pin records identical ops and moves on.
func TestGapCloseAColorInterpolationSRGBPinsRasterGap(t *testing.T) {
	t.Parallel()

	plain := behaviorSvgLayoutSVG(t, "fill:#ff0000")
	tuned := behaviorSvgLayoutSVG(t, "fill:#ff0000;color-interpolation:sRGB")

	behaviorSvgAssertSameOps(t, plain, tuned)
}

// TestGapCloseAColorInterpolationFiltersSRGBPinsRasterGap pins
// color-interpolation-filters as a no-op on the same path as
// color-interpolation: forwarded by
// internal/layout/style_paint_props.go:89-96 to applyLeftoversProps in
// internal/layout/style_leftovers.go:11-41 with no matching case, and no
// filter pass reads a color space. Chrome 143.0.7499.40 would change the
// filter color space, but that needs the frozen bake arms in
// internal/layout/layout_svg.go:169-260, so this pin records identical ops
// and moves on.
func TestGapCloseAColorInterpolationFiltersSRGBPinsRasterGap(t *testing.T) {
	t.Parallel()

	plain := behaviorSvgLayoutSVG(t, "fill:#ff0000")
	tuned := behaviorSvgLayoutSVG(t, "fill:#ff0000;color-interpolation-filters:sRGB")

	behaviorSvgAssertSameOps(t, plain, tuned)
}
