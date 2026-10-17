//nolint:varnamelen // raster pixel probes use x/y coordinates
package layout

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// maskBorderTestPNG builds a 60x60 source with an opaque frame band, a
// transparent center, a red top-left corner cell, and a red marker inside the
// top edge slice, so the frame cells, the fill rule, and the repeat tiling are
// each observable.
func maskBorderTestPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 60, 60))

	for y := range 60 {
		for x := range 60 {
			if c, ok := maskBorderTestPixel(x, y); ok {
				img.Set(x, y, c)
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode test source: %v", err)
	}

	return buf.Bytes()
}

// maskBorderTestPixel colors one source pixel: an opaque blue frame band, a
// red top-left corner cell, a red marker three pixels into the top edge
// slice, and a transparent center. ok is false for the center.
func maskBorderTestPixel(x, y int) (color.RGBA, bool) {
	if x >= 15 && x < 45 && y >= 15 && y < 45 {
		return color.RGBA{}, false
	}

	if x < 15 && y < 15 {
		return color.RGBA{R: 220, G: 40, B: 40, A: 255}, true
	}

	// The marker repeats with the repeat mode and scales away with stretch.
	if y < 15 && x == 18 {
		return color.RGBA{R: 220, G: 40, B: 40, A: 255}, true
	}

	return color.RGBA{R: 40, G: 80, B: 220, A: 255}, true
}

// layoutMaskBorderHTML lays out one document whose images resolve from the
// given map, so the mask-border resolver has a payload.
func layoutMaskBorderHTML(t *testing.T, src string, images map[string][]byte) *Result {
	t.Helper()

	root := mustParse(t, src)
	provider := func(imageSrc string) ([]byte, error) {
		if payload, ok := images[imageSrc]; ok {
			return payload, nil
		}

		return nil, errTestImageMissing
	}

	res, err := Layout(root, Options{
		Width: testViewport, Height: 800, Images: provider, Background: true,
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	return res
}

// TestComposeMaskBorderRasterFrame pins the composer contract: a sliced frame
// raster is produced at the box size, the frame edges are opaque, and the
// center stays transparent without the fill flag.
func TestComposeMaskBorderRasterFrame(t *testing.T) {
	t.Parallel()

	source := maskBorderTestPNG(t)
	frame := &MaskBorder{
		Source:      "url(frame.png)",
		ImageBytes:  source,
		PixelWidth:  60,
		PixelHeight: 60,
		Slice:       [4]float64{15, 15, 15, 15},
		SliceFill:   false,
		Width:       [4]float64{12, 12, 12, 12},
		Repeat:      "stretch",
		Mode:        maskModeAlpha,
	}

	raster := composeMaskBorderRaster(frame, 120, 90)
	if raster == nil {
		t.Fatal("composeMaskBorderRaster returned nil")
	}

	if raster.PixelWidth != 120 || raster.PixelHeight != 90 {
		t.Fatalf("raster size = %dx%d, want 120x90", raster.PixelWidth, raster.PixelHeight)
	}

	img, _, err := image.Decode(bytes.NewReader(raster.ImageBytes))
	if err != nil {
		t.Fatalf("decode raster: %v", err)
	}

	if alphaAt(img, 1, 1) == 0 {
		t.Error("frame corner is transparent, want opaque")
	}

	if alphaAt(img, 60, 45) != 0 {
		t.Errorf("center alpha = %d, want 0 without the fill flag", alphaAt(img, 60, 45))
	}
}

// TestComposeMaskBorderRasterFill pins the fill flag: with fill the center
// copies the source's middle cell (transparent here), and without it the
// center stays fully transparent; the two rasters must agree at the center
// and differ only where the middle slice carries pixels. The observable pin
// is that a fill raster of an opaque-center source becomes opaque.
func TestComposeMaskBorderRasterFill(t *testing.T) {
	t.Parallel()

	// Opaque center source: the fill flag must carry it into the center cell.
	img := image.NewRGBA(image.Rect(0, 0, 60, 60))

	for y := range 60 {
		for x := range 60 {
			img.Set(x, y, color.RGBA{R: 90, G: 160, B: 90, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode test source: %v", err)
	}

	frame := &MaskBorder{
		Source:      "url(frame.png)",
		ImageBytes:  buf.Bytes(),
		PixelWidth:  60,
		PixelHeight: 60,
		Slice:       [4]float64{15, 15, 15, 15},
		SliceFill:   true,
		Width:       [4]float64{12, 12, 12, 12},
		Repeat:      "stretch",
		Mode:        maskModeAlpha,
	}

	raster := composeMaskBorderRaster(frame, 120, 90)
	if raster == nil {
		t.Fatal("composeMaskBorderRaster returned nil")
	}

	decoded, _, err := image.Decode(bytes.NewReader(raster.ImageBytes))
	if err != nil {
		t.Fatalf("decode raster: %v", err)
	}

	if alphaAt(decoded, 60, 45) == 0 {
		t.Error("center alpha = 0, want opaque with the fill flag and an opaque center slice")
	}

	noFill := *frame
	noFill.SliceFill = false

	rasterNoFill := composeMaskBorderRaster(&noFill, 120, 90)
	if rasterNoFill == nil {
		t.Fatal("composeMaskBorderRaster returned nil without fill")
	}

	decodedNoFill, _, err := image.Decode(bytes.NewReader(rasterNoFill.ImageBytes))
	if err != nil {
		t.Fatalf("decode no-fill raster: %v", err)
	}

	if alphaAt(decodedNoFill, 60, 45) != 0 {
		t.Errorf("center alpha = %d without fill, want 0", alphaAt(decodedNoFill, 60, 45))
	}
}

// TestComposeMaskBorderRasterRepeat pins the repeat mode: the edge tiles
// instead of stretching, so repeat and stretch produce different rasters.
func TestComposeMaskBorderRasterRepeat(t *testing.T) {
	t.Parallel()

	source := maskBorderTestPNG(t)
	stretch := &MaskBorder{
		ImageBytes:  source,
		PixelWidth:  60,
		PixelHeight: 60,
		Slice:       [4]float64{15, 15, 15, 15},
		Width:       [4]float64{12, 12, 12, 12},
		Repeat:      "stretch",
	}
	repeat := &MaskBorder{
		ImageBytes:  source,
		PixelWidth:  60,
		PixelHeight: 60,
		Slice:       [4]float64{15, 15, 15, 15},
		Width:       [4]float64{12, 12, 12, 12},
		Repeat:      "repeat",
	}

	stretchImg := decodeMaskBorderRaster(t, stretch)
	repeatImg := decodeMaskBorderRaster(t, repeat)

	// The top edge slice's red marker repeats with the repeat mode and scales
	// away with stretch. The middle slice is 30px wide (60-15-15), so repeat
	// copies it 1:1 and the marker recurs every 30px; stretch spreads the
	// slice across the 186px edge and widens the marker into a band.
	if !maskBorderRedAt(repeatImg, 12+3) {
		t.Error("repeat: red marker missing at its 1:1 position in the first top-edge tile")
	}

	if !maskBorderRedAt(repeatImg, 12+30+3) {
		t.Error("repeat: red marker missing in the second top-edge tile")
	}

	if maskBorderRedAt(repeatImg, 12+3+1) {
		t.Error("repeat: marker column is wider than one pixel, want a 1:1 copy")
	}

	if got := countMaskBorderRed(repeatImg); got != 6 {
		t.Errorf("repeat: %d red columns in the top edge, want 6 (one per 30px tile)", got)
	}

	if got := countMaskBorderRed(stretchImg); got < 5 || got > 7 {
		t.Errorf("stretch: %d red columns in the top edge, want the scaled band (5-7)", got)
	}
}

// decodeMaskBorderRaster composes the frame for one descriptor at 200x90 and
// decodes the raster.
func decodeMaskBorderRaster(t *testing.T, frame *MaskBorder) image.Image {
	t.Helper()

	raster := composeMaskBorderRaster(frame, 200, 90)
	if raster == nil {
		t.Fatal("composer returned nil for a repeat fixture")
	}

	img, _, err := image.Decode(bytes.NewReader(raster.ImageBytes))
	if err != nil {
		t.Fatalf("decode raster: %v", err)
	}

	return img
}

// maskBorderRedAt reports whether the top edge pixel at x carries the red
// marker color.
func maskBorderRedAt(img image.Image, x int) bool {
	r, g, b, _ := img.At(x, 3).RGBA()
	_ = g
	_ = b

	return r > 0xc000
}

// countMaskBorderRed counts the red marker columns in the top edge band.
func countMaskBorderRed(img image.Image) int {
	red := 0

	for x := 12; x < 188; x++ {
		if maskBorderRedAt(img, x) {
			red++
		}
	}

	return red
}

// TestComposeMaskBorderRasterOnGroup pins the popZ hook contract: a built box
// with a mask-border source ends up with a ready-to-paint group raster.
func TestComposeMaskBorderRasterOnGroup(t *testing.T) {
	t.Parallel()

	doc := `<html><body style="margin:0">` +
		`<div id="f" style="width:120px;height:60px;mask-border-source:url(frame.png);` +
		`mask-border-slice:15;mask-border-width:12px"></div>` +
		`</body></html>`

	res := layoutMaskBorderHTML(t, doc, map[string][]byte{"frame.png": maskBorderTestPNG(t)})
	group := firstMaskBorderGroup(t, res)

	if group == nil {
		t.Fatal("no group carries a MaskBorder")
	}

	if group.Mask == nil || len(group.Mask.RasterBytes()) == 0 {
		t.Fatal("group has no ready-to-paint frame raster")
	}

	// The composer works in device pixels: a CSS pixel is ptPerCSSPx points,
	// so the 120x60 CSS box lands at 90x45 points.
	wantW := int(120 * ptPerCSSPx)
	wantH := int(60 * ptPerCSSPx)

	if group.Mask.Raster.PixelWidth != wantW || group.Mask.Raster.PixelHeight != wantH {
		t.Errorf("raster size = %dx%d, want %dx%d",
			group.Mask.Raster.PixelWidth, group.Mask.Raster.PixelHeight, wantW, wantH)
	}
}

// firstMaskBorderGroup returns the first op group carrying a MaskBorder.
func firstMaskBorderGroup(t *testing.T, res *Result) *BlendGroup {
	t.Helper()

	for _, op := range res.Ops {
		if group := op.Group(); group != nil && group.MaskBorder != nil {
			return group
		}
	}

	return nil
}

// alphaAt returns the alpha channel of one raster pixel.
func alphaAt(img image.Image, x, y int) uint32 {
	r, g, b, a := img.At(x, y).RGBA()
	_ = r
	_ = g
	_ = b

	return a
}

// errTestImageMissing reports a fixture image the test map does not carry.
var errTestImageMissing = errors.New("test image missing from fixture map")
