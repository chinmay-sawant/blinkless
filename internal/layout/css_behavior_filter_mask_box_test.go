package layout

import (
	"math"
	"testing"
)

func TestBehaviorFilterCreatesBlendGroup(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div style="width:60px;height:30px;background:#f00;filter:blur(2px)">hello</div>`+
		`<div style="width:60px;height:30px;background:#0f0;filter:none">plain</div>`+
		`<p><span style="filter:blur(3px)">inline</span></p>`+
		`</body></html>`)

	var blockFound, inlineFound bool
	for i := range res.Ops {
		op := &res.Ops[i]
		g := op.Group()
		if g == nil {
			continue
		}
		switch g.Filter {
		case "blur(2px)":
			if !g.Isolate {
				t.Fatalf("block filter group Isolate = false, want true")
			}
			blockFound = true
		case "blur(3px)":
			if !g.Isolate {
				t.Fatalf("inline filter group Isolate = false, want true")
			}
			inlineFound = true
		case "none":
			t.Fatalf("filter:none should not create a blend group")
		}
	}

	if !blockFound {
		t.Fatal("expected blend group with Filter=blur(2px) for block element")
	}
	if !inlineFound {
		t.Fatal("expected blend group with Filter=blur(3px) for inline element")
	}
}

func TestBehaviorMaskDescriptorCarriesBorderAndPaddingInsets(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div style="box-sizing:border-box;width:100px;height:80px;`+
		`border:8px solid #000;padding:12px 16px;background:#369;`+
		`mask-image:linear-gradient(#000,transparent);`+
		`-webkit-mask-origin:content-box;-webkit-mask-clip:padding-box"></div>`+
		`</body></html>`)

	group := onlyMaskGroup(t, res)
	desc := group.Mask
	if desc == nil || len(desc.Layers) != 1 {
		t.Fatalf("expected 1 mask layer, got %+v", desc)
	}

	layer := desc.Layers[0]
	if layer.Origin != "content-box" || layer.Clip != "padding-box" {
		t.Fatalf("origin/clip = %q/%q, want content-box/padding-box", layer.Origin, layer.Clip)
	}

	// 1 CSS px = 0.75 pt at 96 DPI.
	wantBorder := [4]float64{6, 6, 6, 6}
	wantPadding := [4]float64{9, 12, 9, 12}
	for i := range 4 {
		if math.Abs(desc.BorderInsets[i]-wantBorder[i]) > 0.05 {
			t.Fatalf("BorderInsets[%d] = %v, want %v", i, desc.BorderInsets[i], wantBorder[i])
		}
		if math.Abs(desc.PaddingInsets[i]-wantPadding[i]) > 0.05 {
			t.Fatalf("PaddingInsets[%d] = %v, want %v", i, desc.PaddingInsets[i], wantPadding[i])
		}
	}

	// Content-box width = 100 - 16 - 32 = 52px; height = 80 - 16 - 24 = 40px.
	// Border-box would be 100x80, so the gradient raster must be smaller than border-box.
	if layer.PixelWidth <= 0 || layer.PixelHeight <= 0 {
		t.Fatalf("gradient mask raster missing: %dx%d", layer.PixelWidth, layer.PixelHeight)
	}
	if layer.PixelWidth >= 80 || layer.PixelHeight >= 60 {
		t.Fatalf("content-box gradient mask size = %dx%d, expected smaller than border-box (100x80)",
			layer.PixelWidth, layer.PixelHeight)
	}
}
