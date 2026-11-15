package layout

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/svg"
)

// link-parameters against testdata/css/v012/link-parameters. swatch.svg paints
// a rect with fill="env(--color, #0000ff)". The pinned canvas rasterizer does
// not evaluate env(), so internal/svg rewrites it before the raster call.

func linkParamsLayout(t *testing.T) *Result {
	t.Helper()

	dir := filepath.Join("..", "..", "testdata", "css", "v012", "link-parameters")

	page, err := os.ReadFile(filepath.Join(dir, "page.html"))
	if err != nil {
		t.Fatalf("read page: %v", err)
	}

	swatch := linkParamsSwatch(t)

	root := mustParse(t, string(page))

	res, err := Layout(root, Options{
		Width: testViewport, Height: 800, Background: true,
		Sheets: []*css.Stylesheet{sheet(t, svgGeomStyleText(t, string(page)))},
		Images: func(string) ([]byte, error) { return swatch, nil },
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	return res
}

func linkParamsSwatch(t *testing.T) []byte {
	t.Helper()

	swatch, err := os.ReadFile(filepath.Join("..", "..", "testdata", "css", "v012", "link-parameters", "swatch.svg"))
	if err != nil {
		t.Fatalf("read svg: %v", err)
	}

	return swatch
}

func linkParamsImage(t *testing.T, payload []byte) image.Image {
	t.Helper()

	img, err := png.Decode(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	return img
}

func TestBehaviorLinkParametersRecolorsSvg(t *testing.T) {
	t.Parallel()

	res := linkParamsLayout(t)
	imgs := opsOfKind(res, OpImage)

	if len(imgs) != 2 {
		t.Fatalf("got %d image ops, want 2", len(imgs))
	}

	red, _, _ := imgs[0].ImageBytes()
	plain, _, _ := imgs[1].ImageBytes()

	if bytes.Equal(red, plain) {
		t.Fatal("param(--color: #ff0000) left the image bytes equal to link-parameters: none")
	}

	r, g, b, _ := linkParamsImage(t, red).At(8, 8).RGBA()
	if r>>8 < 200 || g>>8 > 60 || b>>8 > 60 {
		t.Errorf("param image pixel = (%d,%d,%d), want red", r>>8, g>>8, b>>8)
	}

	// none passes nothing, so the bytes match the file rasterized as stored.
	raw := linkParamsSwatch(t)

	want, _, _, err := svg.Rasterize(raw, svgRasterMaxDim)
	if err != nil {
		t.Fatalf("rasterize raw svg: %v", err)
	}

	if !bytes.Equal(plain, want) {
		t.Error("link-parameters: none changed the image bytes, want the stored file's raster")
	}
}

func TestBehaviorLinkParametersLeavesAnchorURI(t *testing.T) {
	t.Parallel()

	res := linkParamsLayout(t)

	var found bool

	for _, op := range opsOfKind(res, OpLinkURI) {
		if op.URI == "https://example.com/a" {
			found = true
		}
	}

	if !found {
		t.Error("anchor lost its authored href under link-parameters")
	}
}

func TestBehaviorLinkParametersParsing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in, want string
		ok       bool
	}{
		{"param(--a: #f00)", "--a\t#f00", true},
		{"param(--a, red), param(--b: 1px)", "--a\tred\n--b\t1px", true},
		{"param(--a: red), param(bad), param(--c)", "--a\tred", true},
		{"param(a: red)", "", false},
		{"bogus", "", false},
	}

	for _, tc := range cases {
		got, ok := parseLinkParameters(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Errorf("parseLinkParameters(%q) = %q, %v; want %q, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
