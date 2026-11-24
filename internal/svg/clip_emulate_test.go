package svg_test

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/svg"
)

func decodePNG(t *testing.T, data []byte) image.Image {
	t.Helper()

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}

	return img
}

func pixelAlpha(img image.Image, logicalX, logicalY int) uint8 {
	const logicalW, logicalH = 120, 80
	b := img.Bounds()
	px := b.Min.X + logicalX*b.Dx()/logicalW
	py := b.Min.Y + logicalY*b.Dy()/logicalH
	_, _, _, a := img.At(px, py).RGBA()

	return uint8((a >> 8) & 0xff)
}

func TestFillRuleEvenOddSelfCrossingStar(t *testing.T) {
	t.Parallel()

	star := `M60 5 L95 75 L10 30 L110 30 L25 75 Z`
	nzSVG := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="120" height="80" viewBox="0 0 120 80">` +
		`<path d="` + star + `" fill="#c0392b" fill-rule="nonzero"/></svg>`)
	eoSVG := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="120" height="80" viewBox="0 0 120 80">` +
		`<path d="` + star + `" fill="#c0392b" fill-rule="evenodd"/></svg>`)

	nzPNG, _, _, err := svg.Rasterize(nzSVG, 256)
	if err != nil {
		t.Fatalf("nonzero rasterize: %v", err)
	}

	eoPNG, _, _, err := svg.Rasterize(eoSVG, 256)
	if err != nil {
		t.Fatalf("evenodd rasterize: %v", err)
	}

	nzImg := decodePNG(t, nzPNG)
	eoImg := decodePNG(t, eoPNG)

	// Center pentagon (60, 45) is filled under nonzero and hollow under evenodd.
	if a := pixelAlpha(nzImg, 60, 45); a < 200 {
		t.Fatalf("nonzero star center alpha = %d, want filled", a)
	}

	if a := pixelAlpha(eoImg, 60, 45); a > 20 {
		t.Fatalf("evenodd star center alpha = %d, want hollow", a)
	}

	// Top arm of the star (60, 18) must stay filled under evenodd.
	if a := pixelAlpha(eoImg, 60, 18); a < 200 {
		t.Fatalf("evenodd star arm alpha = %d, want filled", a)
	}
}

func TestFillRuleEvenOddPolygonStar(t *testing.T) {
	t.Parallel()

	eoSVG := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="120" height="80" viewBox="0 0 120 80">` +
		`<polygon points="60,5 95,75 10,30 110,30 25,75" fill="#c0392b" fill-rule="evenodd"/></svg>`)

	eoPNG, _, _, err := svg.Rasterize(eoSVG, 256)
	if err != nil {
		t.Fatalf("evenodd polygon rasterize: %v", err)
	}

	eoImg := decodePNG(t, eoPNG)

	if a := pixelAlpha(eoImg, 60, 45); a > 20 {
		t.Fatalf("evenodd polygon star center alpha = %d, want hollow", a)
	}

	if a := pixelAlpha(eoImg, 60, 18); a < 200 {
		t.Fatalf("evenodd polygon star arm alpha = %d, want filled", a)
	}
}

func TestClipPathEvenOddStar(t *testing.T) {
	t.Parallel()

	star := `M60 5 L95 75 L10 30 L110 30 L25 75 Z`
	mk := func(rule string) []byte {
		return []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="120" height="80" viewBox="0 0 120 80">` +
			`<defs><clipPath id="s"><path d="` + star + `" clip-rule="` + rule + `"/></clipPath></defs>` +
			`<rect x="0" y="0" width="120" height="80" fill="#c0392b" clip-path="url(#s)"/></svg>`)
	}

	nzPNG, _, _, err := svg.Rasterize(mk("nonzero"), 256)
	if err != nil {
		t.Fatalf("nonzero clipPath rasterize: %v", err)
	}

	eoPNG, _, _, err := svg.Rasterize(mk("evenodd"), 256)
	if err != nil {
		t.Fatalf("evenodd clipPath rasterize: %v", err)
	}

	nzImg := decodePNG(t, nzPNG)
	eoImg := decodePNG(t, eoPNG)

	// Corner (5, 5) outside star must be clipped away.
	if a := pixelAlpha(eoImg, 5, 5); a > 20 {
		t.Fatalf("evenodd clip corner alpha = %d, want clipped (0)", a)
	}

	// Top arm (60, 18) inside star must be painted.
	if a := pixelAlpha(eoImg, 60, 18); a < 200 {
		t.Fatalf("evenodd clip arm alpha = %d, want filled", a)
	}

	// Center pentagon (60, 45) is filled under nonzero and hollow under evenodd.
	if a := pixelAlpha(nzImg, 60, 45); a < 200 {
		t.Fatalf("nonzero clip center alpha = %d, want filled", a)
	}

	if a := pixelAlpha(eoImg, 60, 45); a > 20 {
		t.Fatalf("evenodd clip center alpha = %d, want hollow", a)
	}
}

func TestLightingColorInterpolationFilters(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		color string
	}{
		{"default-white", ""},
		{"colored-red", ` lighting-color="#c0392b"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mk := func(interp string) []byte {
				return []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="60" height="40" viewBox="0 0 60 40">` +
					`<defs><filter id="f" x="0" y="0" width="1" height="1" style="color-interpolation-filters:` + interp + `">` +
					`<feDiffuseLighting in="SourceGraphic"` + tc.color + `>` +
					`<feDistantLight azimuth="45" elevation="45"/>` +
					`</feDiffuseLighting></filter></defs>` +
					`<rect x="5" y="5" width="50" height="30" fill="#3b82f6" filter="url(#f)"/></svg>`)
			}

			srgbPNG, _, _, err := svg.Rasterize(mk("sRGB"), 128)
			if err != nil {
				t.Fatalf("sRGB rasterize: %v", err)
			}

			linPNG, _, _, err := svg.Rasterize(mk("linearRGB"), 128)
			if err != nil {
				t.Fatalf("linearRGB rasterize: %v", err)
			}

			srgbImg := decodePNG(t, srgbPNG)
			linImg := decodePNG(t, linPNG)

			b := srgbImg.Bounds()
			cx, cy := b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2
			sr, sg, sb, _ := srgbImg.At(cx, cy).RGBA()
			lr, lg, lb, _ := linImg.At(cx, cy).RGBA()

			if sr == lr && sg == lg && sb == lb {
				t.Fatalf("%s: sRGB and linearRGB produced identical center pixel (%d,%d,%d)",
					tc.name, sr>>8, sg>>8, sb>>8)
			}
		})
	}
}
