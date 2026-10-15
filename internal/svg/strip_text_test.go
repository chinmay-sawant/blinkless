package svg

import (
	"bytes"
	"errors"
	"testing"
)

// TestRasterizeMissingFontFamilyDoesNotPanic proves a missing text family
// cannot reach the system font loader. text, tspan, and textPath are removed
// before canvas.ParseSVG, so glyphs disappear. The rect still rasterizes.
func TestRasterizeMissingFontFamilyDoesNotPanic(t *testing.T) {
	t.Parallel()

	src := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="40" height="20" viewBox="0 0 40 20">` +
		`<rect x="0" y="0" width="40" height="20" fill="#0645ad"/>` +
		`<text font-family="DefinitelyMissing" x="2" y="12">Hi</text>` +
		`<tspan font-family="DefinitelyMissing">x</tspan>` +
		`<textPath href="#p" font-family="DefinitelyMissing">y</textPath>` +
		`</svg>`)

	pngBytes, width, height, err := Rasterize(src, 64)
	if errors.Is(err, errCanvasPanic) {
		t.Fatalf("raster panicked: %v", err)
	}

	if err != nil {
		t.Fatalf("geometry with a missing text family: %v", err)
	}

	if width != 40 || height != 20 || len(pngBytes) < 50 || pngBytes[0] != 0x89 {
		t.Fatalf("geometry raster = %dx%d len %d", width, height, len(pngBytes))
	}
}

// TestRasterizeTextOnlyMissingFamilyDoesNotPanic covers an SVG whose only
// painted element is text. Glyphs disappear. A clean error or an empty
// raster is acceptable. A panic is not.
func TestRasterizeTextOnlyMissingFamilyDoesNotPanic(t *testing.T) {
	t.Parallel()

	src := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10">` +
		`<text font-family="DefinitelyMissing">Hi</text></svg>`)

	pngBytes, width, height, err := Rasterize(src, 32)
	if errors.Is(err, errCanvasPanic) {
		t.Fatalf("raster panicked: %v", err)
	}

	if err != nil && (pngBytes != nil || width != 0 || height != 0) {
		t.Fatalf("error left a partial raster %dx%d err=%v", width, height, err)
	}
}

func TestStripSVGTextKeepsFollowingGeometry(t *testing.T) {
	t.Parallel()

	src := []byte(`<svg><text><![CDATA[</text>]]>` +
		`<tspan font-family="DefinitelyMissing">x</tspan></text>` +
		`<rect width="1" height="1"/></svg>`)
	got := stripSVGTextElements(src)

	if bytes.Contains(bytes.ToLower(got), []byte("definitelymissing")) {
		t.Fatalf("text survived a CDATA end tag: %s", got)
	}

	if !bytes.Contains(got, []byte("<rect")) {
		t.Fatalf("rect dropped: %s", got)
	}
}

func TestStripSVGTextLeavesGeometryUntouched(t *testing.T) {
	t.Parallel()

	src := []byte(`<svg><rect width="4" height="4"/></svg>`)
	if !bytes.Equal(stripSVGTextElements(src), src) {
		t.Fatal("geometry-only SVG was rewritten")
	}
}
