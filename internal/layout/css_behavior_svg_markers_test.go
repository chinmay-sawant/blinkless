package layout

import (
	"fmt"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
	"github.com/chinmay-sawant/blinkless/internal/svg"
)

// SVG marker and CSS d behavior tests. Every test rasterizes the serialized
// SVG payload and compares painted-pixel counts, never stored strings.
// Reference browser for every case: Chrome 143.0.7499.40. Shared helpers
// mustParse, sheet, resolveStyles, decodeNRGBA, and engine come from
// layout_test.go, paint_modern_test.go, and style.go and are never
// redeclared here.

// behaviorSvgMarkersArrowDefs is a reusable arrowhead marker definition.
// markerWidth/markerHeight/refX/refY keep their camelCase through the HTML
// foreign-attribute table (internal/html/foreign.go) into the serializer.
const behaviorSvgMarkersArrowDefs = `<defs><marker id="ah" markerWidth="8" ` +
	`markerHeight="8" refX="4" refY="4" orient="auto">` +
	`<path d="M0 0 L8 4 L0 8 Z" fill="#000000"/></marker></defs>`

// behaviorSvgMarkersPainted counts clearly painted pixels (alpha >= 128) in
// the raster of one inline SVG document. Deterministic for a fixed canvas
// build; tests assert directions and equalities, not exact counts.
func behaviorSvgMarkersPainted(t *testing.T, svgInner string) int {
	t.Helper()

	root := mustParse(t, `<html><body>`+
		`<svg width="64" height="40" viewBox="0 0 64 40" xmlns="http://www.w3.org/2000/svg">`+
		svgInner+`</svg></body></html>`)
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

	pngBytes, _, _, err := svg.Rasterize(eng.serializeInlineSVG(svgNode), 256)
	if err != nil {
		t.Fatalf("rasterize: %v", err)
	}

	img := decodeNRGBA(t, pngBytes)
	painted := 0
	bounds := img.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.NRGBAAt(x, y).A >= 128 {
				painted++
			}
		}
	}

	return painted
}

// TestBehaviorSvgMarkerEndPaintsArrowhead proves CSS marker-end reaches the
// rasterizer: a line with marker-end:url(#ah) paints strictly more than the
// same line without it. Chrome 143.0.7499.40 draws the arrowhead; the pinned
// canvas renderer draws markers at vertices from url(#id) refs.
func TestBehaviorSvgMarkerEndPaintsArrowhead(t *testing.T) {
	t.Parallel()

	line := `<path d="M8 30 H56" fill="none" style="%s"/>`
	plain := behaviorSvgMarkersPainted(t, behaviorSvgMarkersArrowDefs+
		fmt.Sprintf(line, "stroke:#000000;stroke-width:3"))
	marked := behaviorSvgMarkersPainted(t, behaviorSvgMarkersArrowDefs+
		fmt.Sprintf(line, "stroke:#000000;stroke-width:3;marker-end:url(#ah)"))

	if marked <= plain {
		t.Fatalf("marker-end painted = %d, plain = %d, want marker-end strictly greater", marked, plain)
	}
}

// TestBehaviorSvgMarkerShorthandPaintsVertices proves the marker shorthand
// expands onto start, mid, and end: a two-segment polyline with
// marker:url(#ah) paints strictly more than the bare polyline. Chrome
// 143.0.7499.40 treats marker as the shorthand for the three longhands.
func TestBehaviorSvgMarkerShorthandPaintsVertices(t *testing.T) {
	t.Parallel()

	line := `<path d="M8 34 L32 8 L56 34" fill="none" style="%s"/>`
	plain := behaviorSvgMarkersPainted(t, behaviorSvgMarkersArrowDefs+
		fmt.Sprintf(line, "stroke:#000000;stroke-width:3"))
	marked := behaviorSvgMarkersPainted(t, behaviorSvgMarkersArrowDefs+
		fmt.Sprintf(line, "stroke:#000000;stroke-width:3;marker:url(#ah)"))

	if marked <= plain {
		t.Fatalf("marker shorthand painted = %d, plain = %d, want shorthand strictly greater", marked, plain)
	}
}

// TestBehaviorSvgCSSDProvidesPathData proves CSS d supplies path data when
// no d attribute exists: a path with only d:path(...) paints, an empty path
// paints nothing. Chrome 143.0.7499.40 supports d as a geometry property
// (since Chrome 121); the pinned canvas rasterizer reads path data only
// from the d attribute, so layout_svg.go bakes the property into one.
func TestBehaviorSvgCSSDProvidesPathData(t *testing.T) {
	t.Parallel()

	empty := behaviorSvgMarkersPainted(t, `<path fill="#000000"/>`)
	viaCSS := behaviorSvgMarkersPainted(t,
		`<path fill="#000000" style="d:path('M8 8 H56 V32 H8 Z')"/>`)

	if empty != 0 {
		t.Fatalf("empty path painted = %d, want 0", empty)
	}

	if viaCSS <= 0 {
		t.Fatalf("CSS d path painted = %d, want > 0", viaCSS)
	}
}

// TestBehaviorSvgCSSDOverridesDAttribute proves CSS d beats the d
// presentation attribute: the override raster matches the CSS-only raster
// exactly and differs from the attribute-only raster. Chrome
// 143.0.7499.40 gives CSS declarations precedence over presentation
// attributes.
func TestBehaviorSvgCSSDOverridesDAttribute(t *testing.T) {
	t.Parallel()

	attrOnly := behaviorSvgMarkersPainted(t,
		`<path d="M8 8 h2 v2 h-2 Z" fill="#000000"/>`)
	cssOnly := behaviorSvgMarkersPainted(t,
		`<path fill="#000000" style="d:path('M8 8 H56 V32 H8 Z')"/>`)
	overridden := behaviorSvgMarkersPainted(t,
		`<path d="M8 8 h2 v2 h-2 Z" fill="#000000" style="d:path('M8 8 H56 V32 H8 Z')"/>`)

	if overridden != cssOnly {
		t.Fatalf("override painted = %d, CSS-only = %d, want identical", overridden, cssOnly)
	}

	if overridden == attrOnly {
		t.Fatalf("override painted = %d equals attribute-only, want the CSS shape", overridden)
	}
}

// TestBehaviorSvgFillStrokeColorInert pins the fill-color/stroke-color gap:
// both are CSS Fill and Stroke Level 3 draft longhands with no Chrome
// 143.0.7499.40 implementation and no ResolvedStyle field, so they must not
// change the raster. Wiring them as fill/stroke aliases would diverge from
// Chrome, which ignores them. Full support needs a cascade field plus a
// paint-layer model owned outside layout_svg.go.
func TestBehaviorSvgFillStrokeColorInert(t *testing.T) {
	t.Parallel()

	fillBase := behaviorSvgMarkersPainted(t,
		`<rect x="8" y="8" width="48" height="24" style="fill:#ff0000"/>`)
	fillWithLonghand := behaviorSvgMarkersPainted(t,
		`<rect x="8" y="8" width="48" height="24" style="fill:#ff0000;fill-color:#0000ff"/>`)

	if fillWithLonghand != fillBase {
		t.Fatalf("fill-color changed paint %d vs %d, want identical (Chrome ignores it)", fillWithLonghand, fillBase)
	}

	strokeBase := behaviorSvgMarkersPainted(t,
		`<path d="M8 20 H56" fill="none" style="stroke:#ff0000;stroke-width:3"/>`)
	strokeWithLonghand := behaviorSvgMarkersPainted(t,
		`<path d="M8 20 H56" fill="none" style="stroke:#ff0000;stroke-width:3;stroke-color:#0000ff"/>`)

	if strokeWithLonghand != strokeBase {
		t.Fatalf("stroke-color changed paint %d vs %d, want identical (Chrome ignores it)",
			strokeWithLonghand, strokeBase)
	}
}

// TestBehaviorSvgDraftPaintLonghandsInert pins the remaining gap properties:
// each is either a Fill and Stroke Level 3 draft longhand with no Chrome
// 143.0.7499.40 implementation (fill-image, fill-origin, fill-position,
// fill-repeat, fill-size, stroke-align, stroke-image, stroke-origin,
// stroke-position, stroke-repeat, stroke-size, stroke-break, stroke-boundary,
// stroke-limit, stroke-dash-corner, stroke-dash-justify, stroke-dashadjust,
// stroke-dashcorner) or a presentation attribute the pinned canvas renderer
// does not implement (path-length: canvas has no pathLength support, so
// baking the attribute would be a half build with no observable effect).
// Full support needs a paint-server layer subsystem (fill/stroke-image
// family), a stroking model with inside/outside alignment (stroke-align),
// dash-corner/justify shaping, and renderer pathLength scaling. Until then
// every one of these must leave the raster unchanged.
func TestBehaviorSvgDraftPaintLonghandsInert(t *testing.T) {
	t.Parallel()

	rectBase := behaviorSvgMarkersPainted(t,
		`<rect x="8" y="8" width="48" height="24" style="fill:#0645ad"/>`)
	dashBase := behaviorSvgMarkersPainted(t,
		`<path d="M4 20 H60" fill="none" style="stroke:#000000;stroke-width:3;stroke-dasharray:6 3"/>`)

	rectProps := []string{
		"fill-image:url(x.png)",
		"fill-origin:fill-box",
		"fill-position:center",
		"fill-repeat:repeat",
		"fill-size:cover",
		"stroke-image:url(x.png)",
		"stroke-origin:stroke-box",
		"stroke-position:center",
		"stroke-repeat:repeat",
		"stroke-size:cover",
		"stroke-align:inside",
		"stroke-break:bounding-box",
		"stroke-boundary:fill-box",
		"stroke-limit:10",
	}
	lineProps := []string{
		"path-length:100",
		"stroke-dash-corner:round",
		"stroke-dash-justify:stretch",
		"stroke-dashadjust:round",
		"stroke-dashcorner:round",
	}

	for _, prop := range rectProps {
		got := behaviorSvgMarkersPainted(t,
			`<rect x="8" y="8" width="48" height="24" style="fill:#0645ad;`+prop+`"/>`)
		if got != rectBase {
			t.Errorf("%s changed paint %d vs %d, want identical (gap pin, no renderer support)", prop, got, rectBase)
		}
	}

	for _, prop := range lineProps {
		got := behaviorSvgMarkersPainted(t,
			`<path d="M4 20 H60" fill="none" style="stroke:#000000;stroke-width:3;stroke-dasharray:6 3;`+
				prop+`"/>`)
		if got != dashBase {
			t.Errorf("%s changed paint %d vs %d, want identical (gap pin, no renderer support)", prop, got, dashBase)
		}
	}
}
