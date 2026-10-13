package svg_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/svg"
)

func TestRasterizeMaskUsesSourceAlphaForAlphaMode(t *testing.T) {
	t.Parallel()

	source := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="8" height="2" viewBox="0 0 8 2">
  <defs><mask id="alpha" mask-type="alpha">
    <rect width="4" height="2" fill="black" fill-opacity="0.5"/>
    <rect x="4" width="4" height="2" fill="white"/>
  </mask></defs>
</svg>`)

	pngBytes, width, height, err := svg.RasterizeMask(source, "alpha", 8, 2, "match-source")
	if err != nil {
		t.Fatalf("RasterizeMask: %v", err)
	}

	if width != 8 || height != 2 {
		t.Fatalf("logical size = %dx%d, want 8x2", width, height)
	}

	mask := decodeMaskPNG(t, pngBytes)
	if got := maskAlphaAtLogicalPoint(t, mask, width, 2); got < 120 || got > 136 {
		t.Errorf("half-opacity black alpha = %d, want about 128", got)
	}

	if got := maskAlphaAtLogicalPoint(t, mask, width, 6); got != 255 {
		t.Errorf("opaque white alpha = %d, want 255", got)
	}
}

func TestRasterizeMaskUsesSourceLuminanceForMatchSource(t *testing.T) {
	t.Parallel()

	source := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="12" height="2" viewBox="0 0 12 2">
	<defs><mask id="luminance" mask-type="alpha" style="mask-type: luminance">
    <rect width="4" height="2" fill="black"/>
    <rect x="4" width="4" height="2" fill="#808080"/>
    <rect x="8" width="4" height="2" fill="white" fill-opacity="0.5"/>
  </mask></defs>
</svg>`)

	pngBytes, width, height, err := svg.RasterizeMask(source, "luminance", 12, 2, "match-source")
	if err != nil {
		t.Fatalf("RasterizeMask: %v", err)
	}

	if width != 12 || height != 2 {
		t.Fatalf("logical size = %dx%d, want 12x2", width, height)
	}

	mask := decodeMaskPNG(t, pngBytes)
	if got := maskAlphaAtLogicalPoint(t, mask, width, 2); got != 0 {
		t.Errorf("black luminance alpha = %d, want 0", got)
	}

	if got := maskAlphaAtLogicalPoint(t, mask, width, 6); got < 120 || got > 136 {
		t.Errorf("gray luminance alpha = %d, want about 128", got)
	}

	if got := maskAlphaAtLogicalPoint(t, mask, width, 10); got < 120 || got > 136 {
		t.Errorf("half-opacity white luminance alpha = %d, want about 128", got)
	}
}

func TestRasterizeMaskUsesLinearRGBForLuminance(t *testing.T) {
	t.Parallel()

	source := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="4" height="2" viewBox="0 0 4 2">
  <defs><mask id="linear" mask-type="luminance" color-interpolation="linearRGB">
    <rect width="4" height="2" fill="#808080"/>
  </mask></defs>
</svg>`)

	pngBytes, width, height, err := svg.RasterizeMask(source, "linear", 4, 2, "match-source")
	if err != nil {
		t.Fatalf("RasterizeMask: %v", err)
	}

	if width != 4 || height != 2 {
		t.Fatalf("logical size = %dx%d, want 4x2", width, height)
	}

	mask := decodeMaskPNG(t, pngBytes)
	if got := maskAlphaAtLogicalPoint(t, mask, width, 2); got < 53 || got > 57 {
		t.Errorf("linearRGB gray luminance alpha = %d, want about 55", got)
	}
}

func TestRasterizeMaskExplicitModeOverridesSourceMaskType(t *testing.T) {
	t.Parallel()

	source := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="4" height="2" viewBox="0 0 4 2">
  <defs><mask id="source" mask-type="luminance"><rect width="4" height="2" fill="black"/></mask></defs>
</svg>`)

	pngBytes, width, _, err := svg.RasterizeMask(source, "source", 4, 2, "alpha")
	if err != nil {
		t.Fatalf("RasterizeMask: %v", err)
	}

	mask := decodeMaskPNG(t, pngBytes)
	if got := maskAlphaAtLogicalPoint(t, mask, width, 2); got != 255 {
		t.Errorf("alpha override = %d, want 255", got)
	}
}

func decodeMaskPNG(t *testing.T, data []byte) image.Image {
	t.Helper()

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}

	return img
}

func maskAlphaAtLogicalPoint(t *testing.T, img image.Image, logicalW, logicalX int) uint8 {
	t.Helper()

	bounds := img.Bounds()
	pixelX := bounds.Min.X + (2*logicalX+1)*bounds.Dx()/(2*logicalW)
	pixelY := bounds.Min.Y + bounds.Dy()/2

	return maskAlphaAtPixel(t, img.At(pixelX, pixelY))
}

func maskAlphaAtPixel(t *testing.T, pixel color.Color) uint8 {
	t.Helper()

	switch converted := color.NRGBAModel.Convert(pixel).(type) {
	case color.NRGBA:
		return converted.A
	default:
		t.Fatalf("pixel type = %T, want color.NRGBA", pixel)
	}

	return 0
}
