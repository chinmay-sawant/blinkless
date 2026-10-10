//nolint:varnamelen // gap-interp observable pixel suite
package layout

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// Color interpolation behavior tests. Each test asserts USED pixel values
// from decoded display-list PNG bytes (never stored style strings) and shows
// the interpolation selector observably changing output. Reference browser
// for every case: Chrome 143.0.7499.40. Shared helpers layoutHTML,
// opsOfKind, and makeTestPNG come from other files in this package and are
// reused here, never redeclared.

// gapInterpDecodePNG decodes one display-list image payload into pixels.
func gapInterpDecodePNG(t *testing.T, payload []byte) image.Image {
	t.Helper()

	img, err := png.Decode(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}

	return img
}

// gapInterpPixelAt reads the row-0 pixel at column x as 8-bit channels.
func gapInterpPixelAt(t *testing.T, img image.Image, x int) color.NRGBA {
	t.Helper()

	converted, ok := color.NRGBAModel.Convert(img.At(x, 0)).(color.NRGBA)
	if !ok {
		t.Fatalf("pixel at %d is %T, want NRGBA", x, img.At(x, 0))
	}

	return converted
}

// gapInterpAbsDiff is the absolute channel distance between two bytes.
func gapInterpAbsDiff(a, b uint8) int {
	if a > b {
		return int(a) - int(b)
	}

	return int(b) - int(a)
}

// gapInterpPaintedMidpoints asserts two results paint identical op geometry,
// then returns the horizontal-center used pixels of their first image ops.
func gapInterpPaintedMidpoints(t *testing.T, first, second *Result) (color.NRGBA, color.NRGBA) {
	t.Helper()

	if len(first.Ops) != len(second.Ops) {
		t.Fatalf("op count = %d vs %d, want identical geometry", len(first.Ops), len(second.Ops))
	}

	for i := range first.Ops {
		a, b := first.Ops[i], second.Ops[i]
		if a.Kind != b.Kind || !near(a.X, b.X) || !near(a.Y, b.Y) || !near(a.W, b.W) || !near(a.H, b.H) {
			t.Fatalf("op %d geometry differs: %+v vs %+v, want identical", i, a, b)
		}
	}

	payload := func(res *Result) []byte {
		for _, op := range opsOfKind(res, OpImage) {
			if len(op.Image) > 0 {
				return op.Image
			}
		}

		t.Fatal("no OpImage with payload")

		return nil
	}

	firstImg := gapInterpDecodePNG(t, payload(first))
	secondImg := gapInterpDecodePNG(t, payload(second))
	midX := firstImg.Bounds().Dx() / 2

	return gapInterpPixelAt(t, firstImg, midX), gapInterpPixelAt(t, secondImg, midX)
}

// TestGapInterpColorInterpolationLinearDiffersFromSRGB is color-interpolation
// on gradients: sampling a black-to-white gradient in linear light renders a
// brighter midpoint than sampling in sRGB gamma. Reference: Chrome
// 143.0.7499.40 interpolates the selected space instead of always gamma.
func TestGapInterpColorInterpolationLinearDiffersFromSRGB(t *testing.T) {
	t.Parallel()

	const gradient = "linear-gradient(to right, #000000, #ffffff)"

	srgbBytes, _, _, ok := renderGradientPNGWithInterp(gradient, 101, 1, [3]float64{0, 0, 0}, "srgb")
	if !ok {
		t.Fatal("srgb gradient render failed")
	}

	linearBytes, _, _, ok := renderGradientPNGWithInterp(gradient, 101, 1, [3]float64{0, 0, 0}, "linearrgb")
	if !ok {
		t.Fatal("linear gradient render failed")
	}

	srgbMid := gapInterpPixelAt(t, gapInterpDecodePNG(t, srgbBytes), 50)
	linearMid := gapInterpPixelAt(t, gapInterpDecodePNG(t, linearBytes), 50)

	if gapInterpAbsDiff(srgbMid.R, linearMid.R) < 30 {
		t.Errorf("gradient midpoint R = %d (srgb) vs %d (linear), want >= 30 apart",
			srgbMid.R, linearMid.R)
	}

	if linearMid.R <= srgbMid.R {
		t.Errorf("linear midpoint R = %d, want brighter than srgb %d", linearMid.R, srgbMid.R)
	}
}

// TestGapInterpColorInterpolationPropertyChangesGradientPaint wires the
// color-interpolation declaration to gradient paint: two documents that
// differ only in the property emit the same op geometry but different used
// gradient pixels. Reference: Chrome 143.0.7499.40.
func TestGapInterpColorInterpolationPropertyChangesGradientPaint(t *testing.T) {
	t.Parallel()

	doc := func(interp string) *Result {
		return layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
			`<div id="box" style="margin:0;width:101px;height:10px;`+
			`background-image:linear-gradient(to right, #000000, #ffffff);`+
			`color-interpolation:`+interp+`"></div></body></html>`)
	}

	srgbMid, linearMid := gapInterpPaintedMidpoints(t, doc("sRGB"), doc("linearRGB"))

	if gapInterpAbsDiff(srgbMid.R, linearMid.R) < 30 {
		t.Errorf("painted midpoint R = %d (sRGB) vs %d (linearRGB), want >= 30 apart",
			srgbMid.R, linearMid.R)
	}
}

// TestGapInterpColorInterpolationFiltersLinearDiffersFromSRGB wires
// color-interpolation-filters to filter primitives: grayscale(1) over solid
// red computes luminance in linear light instead of gamma, and invert(1)
// over mid-gray inverts the linear value instead of the gamma value, so both
// used pixels differ. Reference: Chrome 143.0.7499.40 runs filter primitives
// in the selected space.
func TestGapInterpColorInterpolationFiltersLinearDiffersFromSRGB(t *testing.T) {
	t.Parallel()

	t.Run("grayscale", func(t *testing.T) {
		t.Parallel()

		raw := makeTestPNG(4, 4, color.RGBA{255, 0, 0, 255})
		filters := parseFilterList("grayscale(1)", [3]float64{0, 0, 0}, 12)

		srgbOut := gapInterpPixelAt(t, gapInterpDecodePNG(t,
			applyImageFilterToImageWithInterp(raw, filters, "srgb")), 0)
		linearOut := gapInterpPixelAt(t, gapInterpDecodePNG(t,
			applyImageFilterToImageWithInterp(raw, filters, "linearrgb")), 0)

		if gapInterpAbsDiff(srgbOut.R, linearOut.R) < 30 {
			t.Errorf("grayscale R = %d (srgb) vs %d (linear), want >= 30 apart",
				srgbOut.R, linearOut.R)
		}
	})

	t.Run("invert", func(t *testing.T) {
		t.Parallel()

		raw := makeTestPNG(4, 4, color.RGBA{128, 128, 128, 255})
		filters := parseFilterList("invert(1)", [3]float64{0, 0, 0}, 12)

		srgbOut := gapInterpPixelAt(t, gapInterpDecodePNG(t,
			applyImageFilterToImageWithInterp(raw, filters, "srgb")), 0)
		linearOut := gapInterpPixelAt(t, gapInterpDecodePNG(t,
			applyImageFilterToImageWithInterp(raw, filters, "linearrgb")), 0)

		if gapInterpAbsDiff(srgbOut.R, linearOut.R) < 50 {
			t.Errorf("invert R = %d (srgb) vs %d (linear), want >= 50 apart",
				srgbOut.R, linearOut.R)
		}
	})
}

// TestGapInterpColorInterpolationFiltersPropertyChangesFilteredPaint wires
// the color-interpolation-filters declaration to filtered paint: two
// documents that differ only in the property emit the same op geometry but
// different used filtered pixels. Reference: Chrome 143.0.7499.40.
func TestGapInterpColorInterpolationFiltersPropertyChangesFilteredPaint(t *testing.T) {
	t.Parallel()

	doc := func(interp string) *Result {
		return layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
			`<div id="box" style="margin:0;width:40px;height:10px;`+
			`background-image:linear-gradient(to right, #ff0000, #ff0000);`+
			`filter:grayscale(1);`+
			`color-interpolation-filters:`+interp+`"></div></body></html>`)
	}

	srgbMid, linearMid := gapInterpPaintedMidpoints(t, doc("sRGB"), doc("linearRGB"))

	if gapInterpAbsDiff(srgbMid.R, linearMid.R) < 30 {
		t.Errorf("filtered R = %d (sRGB) vs %d (linearRGB), want >= 30 apart",
			srgbMid.R, linearMid.R)
	}
}
