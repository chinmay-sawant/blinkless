package layout

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// maskBorderPartialTestPNG builds a 60x60 source image with asymmetric color
// bands on both horizontal and vertical edges so slice, width, outset, repeat,
// round, space, and luminance mode each produce observable raster or layer
// differences.
func maskBorderPartialTestPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 60, 60))

	for y := range 60 {
		for x := range 60 {
			switch {
			case x < 15 && y < 15:
				img.Set(x, y, color.RGBA{R: 220, G: 40, B: 40, A: 255})
			case y < 15 && x == 18:
				img.Set(x, y, color.RGBA{R: 240, G: 220, B: 40, A: 255})
			case x < 15 && y == 18:
				img.Set(x, y, color.RGBA{R: 40, G: 220, B: 80, A: 255})
			case x >= 15 && x < 45 && y >= 15 && y < 45:
				// Transparent center.
			default:
				img.Set(x, y, color.RGBA{R: 40, G: 80, B: 220, A: 255})
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode partial test source: %v", err)
	}

	return buf.Bytes()
}

// TestBehaviorMaskBorderPartialAllNineProperties verifies that all 9
// mask-border-* and -webkit-mask-box-image-* properties populate
// group.Mask.Layers and produce distinct rasters (or distinct Mode for
// mask-border-mode) compared to a control element.
func TestBehaviorMaskBorderPartialAllNineProperties(t *testing.T) {
	t.Parallel()

	srcPNG := maskBorderPartialTestPNG(t)
	images := map[string][]byte{"frame.png": srcPNG}

	baseStyle := `width:160px;height:120px;background:#0f0;` +
		`mask-border-source:url(frame.png);mask-border-slice:15;mask-border-width:16px;`

	layoutOne := func(extra string) *BlendGroup {
		t.Helper()
		doc := `<html><body style="margin:0"><div style="` + baseStyle + extra + `"></div></body></html>`
		res := layoutMaskBorderHTML(t, doc, images)
		group := firstMaskBorderGroup(t, res)
		if group == nil || group.Mask == nil {
			t.Fatalf("style %q: missing group or group.Mask", extra)
		}
		if len(group.Mask.Layers) != 1 {
			t.Fatalf("style %q: got %d layers, want 1", extra, len(group.Mask.Layers))
		}
		layer := group.Mask.Layers[0]
		if layer.Source != "url(frame.png)" || layer.Repeat != "no-repeat" ||
			layer.Size != "100% 100%" || layer.Position != "0% 0%" ||
			layer.Origin != "border-box" || layer.Clip != "border-box" ||
			len(layer.ImageBytes) == 0 || layer.PixelWidth <= 0 || layer.PixelHeight <= 0 {
			t.Fatalf("style %q: unexpected MaskLayer metadata: %+v", extra, layer)
		}
		return group
	}

	control := layoutOne("")
	controlBytes := control.Mask.Layers[0].ImageBytes

	// 1. mask-border-mode: luminance wires Mode onto MaskLayer.
	t.Run("mask-border-mode", func(t *testing.T) {
		t.Parallel()
		got := layoutOne("mask-border-mode:luminance;")
		if got.Mask.Layers[0].Mode != maskModeLuminance {
			t.Fatalf("layer mode = %q, want %q", got.Mask.Layers[0].Mode, maskModeLuminance)
		}
		if control.Mask.Layers[0].Mode != maskModeAlpha {
			t.Fatalf("control layer mode = %q, want %q", control.Mask.Layers[0].Mode, maskModeAlpha)
		}
	})

	rasterCases := []struct {
		name  string
		extra string
	}{
		{name: "mask-border-outset", extra: "mask-border-outset:12px;"},
		{name: "webkit-mask-box-image-outset", extra: "-webkit-mask-box-image-outset:2px;"},
		{name: "mask-border-repeat", extra: "mask-border-repeat:repeat;"},
		{name: "webkit-mask-box-image-repeat", extra: "-webkit-mask-box-image-repeat:round;"},
		{name: "mask-border-slice", extra: "mask-border-slice:40%;"},
		{name: "webkit-mask-box-image-slice", extra: "-webkit-mask-box-image-slice:10;"},
		{name: "mask-border-width", extra: "mask-border-width:12px;"},
		{name: "webkit-mask-box-image-width", extra: "-webkit-mask-box-image-width:10px;"},
	}

	for _, tc := range rasterCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := layoutOne(tc.extra)
			if bytes.Equal(got.Mask.Layers[0].ImageBytes, controlBytes) {
				t.Fatalf("%s (%q) produced identical raster bytes to control", tc.name, tc.extra)
			}
			if !bytes.Equal(got.Mask.Layers[0].ImageBytes, got.Mask.RasterBytes()) {
				t.Fatalf("%s: layer ImageBytes diverged from Mask.Raster.ImageBytes", tc.name)
			}
		})
	}
}

// TestComposeMaskBorderVerticalEdgeRoundAndSpace pins that vertical edges
// (row == 1) tile along Y for both round and space repeat modes.
func TestComposeMaskBorderVerticalEdgeRoundAndSpace(t *testing.T) {
	t.Parallel()

	srcPNG := maskBorderPartialTestPNG(t)

	compose := func(repeat string) image.Image {
		t.Helper()
		frame := &MaskBorder{
			Source:      "url(frame.png)",
			ImageBytes:  srcPNG,
			PixelWidth:  60,
			PixelHeight: 60,
			Slice:       [4]float64{15, 15, 15, 15},
			Width:       [4]float64{12, 12, 12, 12},
			Repeat:      repeat,
			Mode:        maskModeAlpha,
		}
		raster := composeMaskBorderRaster(frame, 80, 140)
		if raster == nil {
			t.Fatalf("composeMaskBorderRaster(%q) returned nil", repeat)
		}
		img, _, err := image.Decode(bytes.NewReader(raster.ImageBytes))
		if err != nil {
			t.Fatalf("decode %q raster: %v", repeat, err)
		}
		return img
	}

	// Count green marker rows in the left vertical edge (x=3, y in [12, 128)).
	// The source left edge slice is 30px tall with a green marker at offset y=3.
	// The destination vertical edge is 116px tall (140 - 12 - 12).
	// - stretch: 1 scaled band of ~4 rows
	// - round: round(116/30) = 4 tiles along Y -> 4 distinct green marker rows
	// - space: floor((116-30)/30)+1 = 3 tiles along Y with gaps -> 3 distinct green rows + transparent gaps
	countGreenRuns := func(img image.Image) int {
		runs := 0
		inRun := false
		for y := 12; y < 128; y++ {
			r, g, _, _ := img.At(3, y).RGBA()
			isGreen := g > 0xc000 && r < 0x6000
			if isGreen && !inRun {
				runs++
			}
			inRun = isGreen
		}
		return runs
	}

	stretchImg := compose("stretch")
	roundImg := compose("stretch round")
	spaceImg := compose("stretch space")

	if got := countGreenRuns(stretchImg); got != 1 {
		t.Errorf("stretch vertical green runs = %d, want 1", got)
	}

	if got := countGreenRuns(roundImg); got != 4 {
		t.Errorf("round vertical green runs = %d, want 4", got)
	}

	if got := countGreenRuns(spaceImg); got != 3 {
		t.Errorf("space vertical green runs = %d, want 3", got)
	}

	// Verify space leaves transparent gaps between vertical tiles on the left edge.
	// With 3 tiles of 30px in 116px, room=86, step=43: tiles at [12..42), [55..85), [98..128).
	// y=48 is right in the gap between tile 0 and tile 1.
	if a := alphaAt(spaceImg, 3, 48); a != 0 {
		t.Errorf("space vertical gap alpha at (3, 48) = %d, want 0", a)
	}
}

// TestComposeMaskBorderAppendsToExistingMask verifies that when an element
// already has a mask-image layer, composeMaskBorderOnGroup appends the
// mask-border layer instead of dropping it.
func TestComposeMaskBorderAppendsToExistingMask(t *testing.T) {
	t.Parallel()

	srcPNG := maskBorderPartialTestPNG(t)
	doc := `<html><body style="margin:0">` +
		`<div style="width:100px;height:60px;mask-image:url(mask.png);` +
		`mask-border-source:url(frame.png);mask-border-slice:15;mask-border-width:10px"></div>` +
		`</body></html>`

	res := layoutMaskBorderHTML(t, doc, map[string][]byte{
		"mask.png":  srcPNG,
		"frame.png": srcPNG,
	})

	group := firstMaskBorderGroup(t, res)
	if group == nil || group.Mask == nil {
		t.Fatal("missing group or group.Mask")
	}

	if len(group.Mask.Layers) != 2 {
		t.Fatalf("got %d mask layers, want 2", len(group.Mask.Layers))
	}

	if group.Mask.Layers[0].Source != "url(mask.png)" {
		t.Errorf("layer[0].Source = %q, want url(mask.png)", group.Mask.Layers[0].Source)
	}

	if group.Mask.Layers[1].Source != "url(frame.png)" {
		t.Errorf("layer[1].Source = %q, want url(frame.png)", group.Mask.Layers[1].Source)
	}

	if !bytes.Equal(group.Mask.Layers[1].ImageBytes, group.Mask.RasterBytes()) {
		t.Error("group.Mask.Raster was not updated to the composed mask-border raster")
	}
}
