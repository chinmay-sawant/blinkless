package layout

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
	"github.com/chinmay-sawant/blinkless/internal/svg"
)

// SVG presentation-attribute bake tests. Each bake test rasterizes the
// serialized SVG twice (with and without the property) and asserts the used
// pixel values differ, never stored strings. Reference browser for every
// case: Chrome 143.0.7499.40. Shared helpers mustParse, sheet, near,
// pxToPt, and opsOfKind come from layout_test.go and css_review_02_test.go
// and are never redeclared here.

// svgBakeMemAvailableMB reads MemAvailable from /proc/meminfo. A read error
// yields a large value so non-Linux runners proceed instead of skipping.
func svgBakeMemAvailableMB() float64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 1 << 30
	}

	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 3 && fields[0] == "MemAvailable:" && fields[2] == "kB" {
			if kb, err := strconv.ParseFloat(fields[1], 64); err == nil {
				return kb / 1024
			}
		}
	}

	return 1 << 30
}

// svgBakeRequireMem skips the test when under 1200 MB available, sleeping
// 15s and retrying twice before giving up.
func svgBakeRequireMem(t *testing.T) {
	t.Helper()

	for attempt := range 3 {
		if svgBakeMemAvailableMB() >= 1200 {
			return
		}

		if attempt < 2 {
			time.Sleep(15 * time.Second)
		}
	}

	t.Skip("skip: under 1200 MB MemAvailable after two retries")
}

// svgBakeSerialize returns the serialized SVG XML for one inline SVG body,
// which is the payload the rasterizer consumes.
func svgBakeSerialize(t *testing.T, inner string) string {
	t.Helper()

	root := mustParse(t, `<html><body>`+
		`<svg width="64" height="28" viewBox="0 0 64 28" xmlns="http://www.w3.org/2000/svg">`+
		inner+
		`</svg></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `body{margin:0}`)}, "print", 500, 800)
	eng := &engine{styles: styles, scale: 1}

	var svgNode *html.Node

	root.Walk(func(node *html.Node) {
		if svgNode == nil && node.Type == html.ElementNode && node.Name == "svg" {
			svgNode = node
		}
	})

	if svgNode == nil {
		t.Fatal("svg missing")
	}

	return string(eng.serializeInlineSVG(svgNode))
}

// svgBakeRaster rasterizes serialized SVG XML into a decoded image.
func svgBakeRaster(t *testing.T, svgXML string) image.Image {
	t.Helper()

	pngBytes, _, _, err := svg.Rasterize([]byte(svgXML), 1024)
	if err != nil {
		t.Fatalf("rasterize: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}

	return img
}

// svgBakeDiffPixels counts pixels whose RGBA values differ between two
// rasters of equal bounds.
func svgBakeDiffPixels(t *testing.T, first, second image.Image) int {
	t.Helper()

	if !first.Bounds().Eq(second.Bounds()) {
		t.Fatalf("bounds = %v vs %v, want equal", first.Bounds(), second.Bounds())
	}

	diff := 0
	bounds := first.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if first.At(x, y) != second.At(x, y) {
				diff++
			}
		}
	}

	return diff
}

// TestBehaviorSvgBakeDasharrayDashesStroke is stroke-dasharray: the dashed
// rect rasterizes with gaps while the solid one does not, so used pixels
// differ. Reference: Chrome 143.0.7499.40 dashes the outline.
func TestBehaviorSvgBakeDasharrayDashesStroke(t *testing.T) {
	t.Parallel()
	svgBakeRequireMem(t)

	rect := `<rect x="2" y="2" width="60" height="24" style="%s"/>`
	plain := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(rect, "fill:none;stroke:#ff0000;stroke-width:3pt")))
	dashed := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(rect, "fill:none;stroke:#ff0000;stroke-width:3pt;stroke-dasharray:5pt 2pt")))

	diff := svgBakeDiffPixels(t, plain, dashed)
	t.Logf("dasharray raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("stroke-dasharray raster identical to solid, want dashed gaps in used pixels")
	}
}

// TestBehaviorSvgBakeDashoffsetShiftsPhase is stroke-dashoffset: shifting the
// dash phase moves the gaps along the outline, so used pixels differ.
// Reference: Chrome 143.0.7499.40 shifts the dash phase.
func TestBehaviorSvgBakeDashoffsetShiftsPhase(t *testing.T) {
	t.Parallel()
	svgBakeRequireMem(t)

	rect := `<rect x="2" y="2" width="60" height="24" style="%s"/>`
	base := "fill:none;stroke:#ff0000;stroke-width:3pt;stroke-dasharray:5pt 2pt"
	plain := svgBakeRaster(t, svgBakeSerialize(t, fmt.Sprintf(rect, base)))
	shifted := svgBakeRaster(t, svgBakeSerialize(t, fmt.Sprintf(rect, base+";stroke-dashoffset:3pt")))

	diff := svgBakeDiffPixels(t, plain, shifted)
	t.Logf("dashoffset raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("stroke-dashoffset raster identical to unshifted, want moved gaps in used pixels")
	}
}

// TestBehaviorSvgBakeLinecapRoundsEnds is stroke-linecap: round caps extend
// past the open line ends while butt caps stop flush, so used pixels differ.
// Reference: Chrome 143.0.7499.40 rounds the open path ends.
func TestBehaviorSvgBakeLinecapRoundsEnds(t *testing.T) {
	t.Parallel()
	svgBakeRequireMem(t)

	line := `<line x1="10" y1="14" x2="54" y2="14" style="%s"/>`
	butt := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(line, "fill:none;stroke:#ff0000;stroke-width:6pt;stroke-linecap:butt")))
	round := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(line, "fill:none;stroke:#ff0000;stroke-width:6pt;stroke-linecap:round")))

	diff := svgBakeDiffPixels(t, butt, round)
	t.Logf("linecap raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("stroke-linecap:round raster identical to butt, want extended ends in used pixels")
	}
}

// TestBehaviorSvgBakeLinejoinRoundsCorners is stroke-linejoin: round joins
// curve the rect stroke corners while miter joins spike them, so used pixels
// differ. Reference: Chrome 143.0.7499.40 rounds the stroke corners.
func TestBehaviorSvgBakeLinejoinRoundsCorners(t *testing.T) {
	t.Parallel()
	svgBakeRequireMem(t)

	rect := `<rect x="4" y="4" width="56" height="20" style="%s"/>`
	miter := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(rect, "fill:none;stroke:#ff0000;stroke-width:6pt;stroke-linejoin:miter")))
	round := svgBakeRaster(t,
		svgBakeSerialize(t, fmt.Sprintf(rect, "fill:none;stroke:#ff0000;stroke-width:6pt;stroke-linejoin:round")))

	diff := svgBakeDiffPixels(t, miter, round)
	t.Logf("linejoin raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("stroke-linejoin:round raster identical to miter, want curved corners in used pixels")
	}
}

// TestBehaviorSvgBakeMiterlimitClipsSpike is stroke-miterlimit: at limit 1 a
// right-angle corner (miter ratio 1.414) falls back to bevel while limit 10
// keeps the miter spike, so used pixels differ. Reference: Chrome
// 143.0.7499.40 clips the miter join at the limit.
func TestBehaviorSvgBakeMiterlimitClipsSpike(t *testing.T) {
	t.Parallel()
	svgBakeRequireMem(t)

	rect := `<rect x="4" y="4" width="56" height="20" style="%s"/>`
	base := "fill:none;stroke:#ff0000;stroke-width:6pt;stroke-linejoin:miter"
	clipped := svgBakeRaster(t, svgBakeSerialize(t, fmt.Sprintf(rect, base+";stroke-miterlimit:1")))
	spiked := svgBakeRaster(t, svgBakeSerialize(t, fmt.Sprintf(rect, base+";stroke-miterlimit:10")))

	diff := svgBakeDiffPixels(t, clipped, spiked)
	t.Logf("miterlimit raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("stroke-miterlimit:1 raster identical to 10, want bevel fallback in used pixels")
	}
}

// TestBehaviorSvgBakeFillRuleGap pins the fill-rule raster gap: the bake arm
// in layout_svg.go emits fill-rule="evenodd" into the serialized SVG (proved
// below), but the pinned canvas parser has no fill-rule case in its style
// switch even though its core Style carries a FillRule field, so the raster
// cannot observe it. No pixel assertion here, never a failing test.
// Reference: Chrome 143.0.7499.40 would apply evenodd winding to holed paths.
func TestBehaviorSvgBakeFillRuleGap(t *testing.T) {
	t.Parallel()

	serialized := svgBakeSerialize(t, `<rect x="2" y="2" width="60" height="24" style="fill:#ff0000;fill-rule:evenodd"/>`)
	if !strings.Contains(serialized, `fill-rule="evenodd"`) {
		t.Errorf("serialized SVG lacks baked fill-rule, got %s", serialized)
	}
}

// TestBehaviorSvgBakeShapeRenderingGap pins the shape-rendering raster gap:
// the bake arm in layout_svg.go emits the hint into the serialized SVG
// (proved below), but the pinned canvas parser has no shape-rendering case,
// so the raster cannot observe it. No pixel assertion here, never a failing
// test. Reference: Chrome 143.0.7499.40 would snap edges with crispEdges.
func TestBehaviorSvgBakeShapeRenderingGap(t *testing.T) {
	t.Parallel()

	serialized := svgBakeSerialize(t,
		`<rect x="2" y="2" width="60" height="24" style="fill:#ff0000;shape-rendering:crispEdges"/>`)
	if !strings.Contains(serialized, `shape-rendering="crispEdges"`) {
		t.Errorf("serialized SVG lacks baked shape-rendering, got %s", serialized)
	}
}
