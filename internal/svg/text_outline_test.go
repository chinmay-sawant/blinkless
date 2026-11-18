package svg

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

// inkRows returns the first and last pixel row that holds dark ink.
func inkRows(t *testing.T, svgText string) (int, int) {
	t.Helper()

	data, _, _, err := Rasterize([]byte(svgText), 200)
	if err != nil {
		t.Fatal(err)
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	return darkRows(img)
}

func darkRows(img image.Image) (int, int) {
	first, last := -1, -1
	bounds := img.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			red, green, blue, alpha := img.At(x, y).RGBA()
			if alpha > 0x8000 && red < 0x8000 && green < 0x8000 && blue < 0x8000 {
				if first < 0 {
					first = y
				}

				last = y

				break
			}
		}
	}

	return first, last
}

func textSVG(tspanStyle string) string {
	return `<svg xmlns="http://www.w3.org/2000/svg" width="200" height="140" font-size="40" ` +
		`font-family="Liberation Sans"><text x="10" y="80"><tspan style="` + tspanStyle + `">H</tspan></text></svg>`
}

// TestSVGTextDrawsOutlines: a <text> element now leaves ink, sitting on its
// baseline.
func TestSVGTextDrawsOutlines(t *testing.T) {
	t.Parallel()

	first, last := inkRows(t, textSVG("x:0"))
	if first < 0 {
		t.Fatal("SVG text left no ink")
	}

	if last < 78 || last > 80 {
		t.Errorf("H bottom row = %d, want the baseline at 80", last)
	}

	if first > 56 || first < 48 {
		t.Errorf("H top row = %d, want a cap height of about 29px above the baseline", first)
	}
}

// TestSVGBaselineShift: a length raises by that many user units, a percentage
// is taken of the font size, sub and super move by half of ascent plus descent.
func TestSVGBaselineShift(t *testing.T) {
	t.Parallel()

	_, base := inkRows(t, textSVG("x:0"))

	for style, want := range map[string]int{
		"baseline-shift:10px": 10, "baseline-shift:50%": 20, "baseline-shift:1em": 40,
		"baseline-shift:super": 22, "baseline-shift:sub": -22,
	} {
		_, got := inkRows(t, textSVG(style))
		if diff := base - got; diff < want-1 || diff > want+1 {
			t.Errorf("%s raised the baseline by %d rows, want %d", style, diff, want)
		}
	}
}

// TestSVGTextAnchorMiddle: text-anchor middle centers the run on x.
func TestSVGTextAnchorMiddle(t *testing.T) {
	t.Parallel()

	data, _, _, err := Rasterize([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="200" height="100" font-size="30">`+
		`<text x="100" y="60" text-anchor="middle">HIH</text></svg>`), 200)
	if err != nil {
		t.Fatal(err)
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	minX, maxX := img.Bounds().Max.X, 0

	for y := 0; y < img.Bounds().Max.Y; y++ {
		for x := 0; x < img.Bounds().Max.X; x++ {
			if red, _, _, alpha := img.At(x, 60).RGBA(); alpha > 0x8000 && red < 0x8000 {
				minX, maxX = min(minX, x), max(maxX, x)
			}
		}
	}

	if centre := (minX + maxX) / 2; centre < 97 || centre > 103 {
		t.Errorf("ink spans %d..%d, centre %d, want about 100", minX, maxX, centre)
	}
}

// litPixel rasterizes a rect under one diffuse light and returns the pixel at
// column x of row 60 as r, g, b.
func litPixel(t *testing.T, lightingColor string, x int) [3]int {
	t.Helper()

	data, _, _, err := Rasterize([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="140" height="120"><defs>`+
		`<filter id="f"><feDiffuseLighting in="SourceGraphic" lighting-color="`+lightingColor+`" surfaceScale="5">`+
		`<feDistantLight azimuth="45" elevation="55"/></feDiffuseLighting></filter></defs>`+
		`<rect x="10" y="10" width="100" height="100" fill="#aaa" filter="url(#f)"/></svg>`), 140)
	if err != nil {
		t.Fatal(err)
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	r, g, b, _ := img.At(x, 60).RGBA()

	return [3]int{int(r >> 8), int(g >> 8), int(b >> 8)}
}

func nearRGB(got, want [3]int) bool {
	for i := range got {
		if d := got[i] - want[i]; d < -2 || d > 2 {
			return false
		}
	}

	return true
}

// TestSVGLightingColorShadesTheShape: lighting-color tints a diffuse light.
// The flat interior is lighting-color times sin(elevation) in linear RGB, and
// the outline bands tilt the normal away from or toward the light.
func TestSVGLightingColorShadesTheShape(t *testing.T) {
	t.Parallel()

	if got := litPixel(t, "white", 60); !nearRGB(got, [3]int{234, 234, 234}) {
		t.Errorf("white interior = %v, want about 234", got)
	}

	if got := litPixel(t, "#336699", 60); !nearRGB(got, [3]int{46, 93, 139}) {
		t.Errorf("#336699 interior = %v, want about (46, 93, 139)", got)
	}

	if got := litPixel(t, "white", 10); !nearRGB(got, [3]int{0, 0, 0}) {
		t.Errorf("left outline band = %v, want black (tilted away from the light)", got)
	}

	if got := litPixel(t, "white", 110); !nearRGB(got, [3]int{197, 197, 197}) {
		t.Errorf("right outline band = %v, want about 197", got)
	}
}

func rasterBytes(t *testing.T, svgText string) []byte {
	t.Helper()

	data, _, _, err := Rasterize([]byte(svgText), 300)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

// TestSVGPropertiesWithNoRenderingEffect: text-rendering and
// glyph-orientation-vertical on horizontal text, and marker-side on a marked
// path, leave the picture unchanged.
func TestSVGPropertiesWithNoRenderingEffect(t *testing.T) {
	t.Parallel()

	text := func(style string) string {
		return `<svg xmlns="http://www.w3.org/2000/svg" width="300" height="60" font-size="30"><text x="5" y="40" style="` +
			style + `">Wave Ab12</text></svg>`
	}

	plain := rasterBytes(t, text("x:0"))

	for _, style := range []string{
		"text-rendering:optimizeSpeed", "text-rendering:optimizeLegibility", "text-rendering:geometricPrecision",
		"glyph-orientation-vertical:90", "glyph-orientation-vertical:0",
	} {
		if !bytes.Equal(plain, rasterBytes(t, text(style))) {
			t.Errorf("%s changed the picture of horizontal text", style)
		}
	}

	path := func(style string) string {
		return `<svg xmlns="http://www.w3.org/2000/svg" width="300" height="80"><defs><marker id="m" markerWidth="12" ` +
			`markerHeight="12" refX="6" refY="6" orient="auto"><path d="M0 0 L12 6 L0 12 Z" fill="blue"/></marker></defs>` +
			`<path d="M10 20 L150 20 L190 60" stroke="red" fill="none" stroke-width="3" marker-end="url(#m)" ` +
			`style="` + style + `"/></svg>`
	}

	base := rasterBytes(t, path("x:0"))

	for _, style := range []string{"marker-side:match-self", "marker-side:match-parent"} {
		if !bytes.Equal(base, rasterBytes(t, path(style))) {
			t.Errorf("%s changed the picture of a marked path", style)
		}
	}
}
