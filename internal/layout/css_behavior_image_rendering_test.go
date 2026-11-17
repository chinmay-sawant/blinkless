package layout

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// image-rendering is carried on the OpImage extra. The engine does not
// resample, so the drawing-list read is the observable result. It inherits.

func imageRenderingOps(t *testing.T, bodyHTML, sheetText string) []Op {
	t.Helper()

	swatch := linkParamsSwatch(t)
	root := mustParse(t, `<html><body style="margin:0">`+bodyHTML+`</body></html>`)

	res, err := Layout(root, Options{
		Width: testViewport, Height: 800, Background: true,
		Sheets: []*css.Stylesheet{sheet(t, sheetText)},
		Images: func(string) ([]byte, error) { return swatch, nil },
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	return opsOfKind(res, OpImage)
}

func TestBehaviorSvgPresentImageRendering(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                                "auto",
		"image-rendering:auto":            "auto",
		"image-rendering:optimizeSpeed":   "optimizeSpeed",
		"image-rendering:optimizespeed":   "optimizeSpeed",
		"image-rendering:optimizeQuality": "optimizeQuality",
	}

	for decl, want := range cases {
		ops := imageRenderingOps(t, `<img src="a.svg" width="64" height="40" alt="">`, `img{`+decl+`}`)
		if len(ops) != 1 {
			t.Fatalf("%q: got %d image ops, want 1", decl, len(ops))
		}

		if got := ops[0].ImageRendering(); got != want {
			t.Errorf("%q: image op keyword = %q, want %q", decl, got, want)
		}
	}

	// A token outside the catalog syntax is not stored.
	for _, bad := range []string{"pixelated", "crisp-edges", "bogus"} {
		ops := imageRenderingOps(t, `<img src="a.svg" width="64" height="40" alt="">`, `img{image-rendering:`+bad+`}`)
		if got := ops[0].ImageRendering(); got != "auto" {
			t.Errorf("%s: image op keyword = %q, want auto", bad, got)
		}
	}

	// The property inherits from the parent element.
	ops := imageRenderingOps(t,
		`<div style="image-rendering:optimizeSpeed"><img src="a.svg" width="64" height="40" alt=""></div>`, ``)
	if got := ops[0].ImageRendering(); got != "optimizeSpeed" {
		t.Errorf("inherited image op keyword = %q, want optimizeSpeed", got)
	}

	// An inline svg carries it too.
	root := mustParse(t, `<html><body style="margin:0"><svg width="64" height="40" viewBox="0 0 64 40" `+
		`xmlns="http://www.w3.org/2000/svg" style="image-rendering:optimizeSpeed">`+
		`<rect width="64" height="40" fill="#f00"/></svg></body></html>`)

	res, err := Layout(root, Options{Width: testViewport, Height: 800, Background: true})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	svgOps := opsOfKind(res, OpImage)
	if len(svgOps) != 1 || svgOps[0].ImageRendering() != "optimizeSpeed" {
		t.Errorf("inline svg image op keyword not carried: %d ops", len(svgOps))
	}
}
