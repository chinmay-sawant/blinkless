//nolint:varnamelen // color-interp observable pixel suite
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

// colorInterpDecodePNG decodes one display-list image payload into pixels.
func colorInterpDecodePNG(t *testing.T, payload []byte) image.Image {
	t.Helper()

	img, err := png.Decode(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}

	return img
}

// colorInterpPixelAt reads the row-0 pixel at column x as 8-bit channels.
func colorInterpPixelAt(t *testing.T, img image.Image, x int) color.NRGBA {
	t.Helper()

	converted, ok := color.NRGBAModel.Convert(img.At(x, 0)).(color.NRGBA)
	if !ok {
		t.Fatalf("pixel at %d is %T, want NRGBA", x, img.At(x, 0))
	}

	return converted
}

// colorInterpAbsDiff is the absolute channel distance between two bytes.
func colorInterpAbsDiff(a, b uint8) int {
	if a > b {
		return int(a) - int(b)
	}

	return int(b) - int(a)
}

// colorInterpPaintedMidpoints asserts two results paint identical op geometry,
// then returns the horizontal-center used pixels of their first image ops.
func colorInterpPaintedMidpoints(t *testing.T, first, second *Result) (color.NRGBA, color.NRGBA) {
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

	firstImg := colorInterpDecodePNG(t, payload(first))
	secondImg := colorInterpDecodePNG(t, payload(second))
	midX := firstImg.Bounds().Dx() / 2

	return colorInterpPixelAt(t, firstImg, midX), colorInterpPixelAt(t, secondImg, midX)
}

// TestColorInterpolationColorInterpolationLinearDiffersFromSRGB is color-interpolation
// on gradients: sampling a black-to-white gradient in linear light renders a
// brighter midpoint than sampling in sRGB gamma. Reference: Chrome
// 143.0.7499.40 interpolates the selected space instead of always gamma.
func TestColorInterpolationColorInterpolationLinearDiffersFromSRGB(t *testing.T) {
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

	srgbMid := colorInterpPixelAt(t, colorInterpDecodePNG(t, srgbBytes), 50)
	linearMid := colorInterpPixelAt(t, colorInterpDecodePNG(t, linearBytes), 50)

	if colorInterpAbsDiff(srgbMid.R, linearMid.R) < 30 {
		t.Errorf("gradient midpoint R = %d (srgb) vs %d (linear), want >= 30 apart",
			srgbMid.R, linearMid.R)
	}

	if linearMid.R <= srgbMid.R {
		t.Errorf("linear midpoint R = %d, want brighter than srgb %d", linearMid.R, srgbMid.R)
	}
}

// TestColorInterpolationColorInterpolationPropertyChangesGradientPaint wires the
// color-interpolation declaration to gradient paint: two documents that
// differ only in the property emit the same op geometry but different used
// gradient pixels. Reference: Chrome 143.0.7499.40.
func TestColorInterpolationColorInterpolationPropertyChangesGradientPaint(t *testing.T) {
	t.Parallel()

	doc := func(interp string) *Result {
		return layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
			`<div id="box" style="margin:0;width:101px;height:10px;`+
			`background-image:linear-gradient(to right, #000000, #ffffff);`+
			`color-interpolation:`+interp+`"></div></body></html>`)
	}

	srgbMid, linearMid := colorInterpPaintedMidpoints(t, doc("sRGB"), doc("linearRGB"))

	if colorInterpAbsDiff(srgbMid.R, linearMid.R) < 30 {
		t.Errorf("painted midpoint R = %d (sRGB) vs %d (linearRGB), want >= 30 apart",
			srgbMid.R, linearMid.R)
	}
}

// TestColorInterpolationColorInterpolationFiltersLinearDiffersFromSRGB wires
// color-interpolation-filters to filter primitives: grayscale(1) over solid
// red computes luminance in linear light instead of gamma, and invert(1)
// over mid-gray inverts the linear value instead of the gamma value, so both
// used pixels differ. Reference: Chrome 143.0.7499.40 runs filter primitives
// in the selected space.
func TestColorInterpolationColorInterpolationFiltersLinearDiffersFromSRGB(t *testing.T) {
	t.Parallel()

	t.Run("grayscale", func(t *testing.T) {
		t.Parallel()

		raw := makeTestPNG(4, 4, color.RGBA{255, 0, 0, 255})
		filters := parseFilterList("grayscale(1)", [3]float64{0, 0, 0}, 12)

		srgbOut := colorInterpPixelAt(t, colorInterpDecodePNG(t,
			applyImageFilterToImageWithInterp(raw, filters, "srgb")), 0)
		linearOut := colorInterpPixelAt(t, colorInterpDecodePNG(t,
			applyImageFilterToImageWithInterp(raw, filters, "linearrgb")), 0)

		if colorInterpAbsDiff(srgbOut.R, linearOut.R) < 30 {
			t.Errorf("grayscale R = %d (srgb) vs %d (linear), want >= 30 apart",
				srgbOut.R, linearOut.R)
		}
	})

	t.Run("invert", func(t *testing.T) {
		t.Parallel()

		raw := makeTestPNG(4, 4, color.RGBA{128, 128, 128, 255})
		filters := parseFilterList("invert(1)", [3]float64{0, 0, 0}, 12)

		srgbOut := colorInterpPixelAt(t, colorInterpDecodePNG(t,
			applyImageFilterToImageWithInterp(raw, filters, "srgb")), 0)
		linearOut := colorInterpPixelAt(t, colorInterpDecodePNG(t,
			applyImageFilterToImageWithInterp(raw, filters, "linearrgb")), 0)

		if colorInterpAbsDiff(srgbOut.R, linearOut.R) < 50 {
			t.Errorf("invert R = %d (srgb) vs %d (linear), want >= 50 apart",
				srgbOut.R, linearOut.R)
		}
	})
}

// TestColorInterpolationColorInterpolationFiltersPropertyChangesFilteredPaint wires
// the color-interpolation-filters declaration to filtered paint: two
// documents that differ only in the property emit the same op geometry but
// different used filtered pixels. Reference: Chrome 143.0.7499.40.
func TestColorInterpolationColorInterpolationFiltersPropertyChangesFilteredPaint(t *testing.T) {
	t.Parallel()

	doc := func(interp string) *Result {
		return layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
			`<div id="box" style="margin:0;width:40px;height:10px;`+
			`background-image:linear-gradient(to right, #ff0000, #ff0000);`+
			`filter:grayscale(1);`+
			`color-interpolation-filters:`+interp+`"></div></body></html>`)
	}

	srgbMid, linearMid := colorInterpPaintedMidpoints(t, doc("sRGB"), doc("linearRGB"))

	if colorInterpAbsDiff(srgbMid.R, linearMid.R) < 30 {
		t.Errorf("filtered R = %d (sRGB) vs %d (linearRGB), want >= 30 apart",
			srgbMid.R, linearMid.R)
	}
}
