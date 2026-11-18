package layout

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// Stylesheet SVG geometry. The fixture's style element moves a circle, rect,
// ellipse, and path off the presentation attributes. Reference browser:
// Chrome 143.0.7499.40 lets a stylesheet cx, cy, r, rx, ry, x, y, or d
// replace the attribute. layoutHTML, sheet, opsOfKind, and svgBakeDiffPixels
// come from other files in this package.

func TestBehaviorSvgStylesheetGeometryMovesShape(t *testing.T) {
	t.Parallel()

	movedSrc := svgGeomReadFixture(t)
	twinSrc := strings.NewReplacer(
		"cx: 40px", "cx: 10px",
		"cy: 20px", "cy: 8px",
		"r: 12px", "r: 4px",
		"x: 24px", "x: 2px",
		"y: 10px", "y: 2px",
		"rx: 18px", "rx: 4px",
		"ry: 8px", "ry: 2px",
		"path('M8 8 H56 V32 H8 Z')", "path('M8 8 h2 v2 h-2 Z')",
	).Replace(movedSrc)

	moved := svgGeomImage(t, layoutHTML(t, movedSrc, svgGeomStylesheet(t, movedSrc)))
	twin := svgGeomImage(t, layoutHTML(t, twinSrc, svgGeomStylesheet(t, twinSrc)))
	diff := svgBakeDiffPixels(t, moved, twin)
	t.Logf("stylesheet geometry raster diff pixels = %d", diff)

	if diff == 0 {
		t.Error("stylesheet geometry raster identical to attribute values, want moved shapes")
	}

	serialized := svgGeomSerialize(t, movedSrc)
	svgGeomAssertBaked(t, serialized, `cx="40px"`, `cx="10"`)
	svgGeomAssertBaked(t, serialized, `cy="20px"`, `cy="8"`)
	svgGeomAssertBaked(t, serialized, `r="12px"`, `r="4"`)
	svgGeomAssertBaked(t, serialized, `x="24px"`, `x="2"`)
	svgGeomAssertBaked(t, serialized, `y="10px"`, `y="2"`)
	svgGeomAssertBaked(t, serialized, `rx="18px"`, `rx="4"`)
	svgGeomAssertBaked(t, serialized, `ry="8px"`, `ry="2"`)
	svgGeomAssertBaked(t, serialized, `d="M8 8 H56 V32 H8 Z"`, `h2 v2 h-2`)
}

func svgGeomReadFixture(t *testing.T) string {
	t.Helper()

	body, err := os.ReadFile("../../testdata/css/v012/svg-geometry/page.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	return string(body)
}

func svgGeomStylesheet(t *testing.T, src string) *css.Stylesheet {
	t.Helper()

	return sheet(t, svgGeomStyleText(t, src))
}

func svgGeomStyleText(t *testing.T, src string) string {
	t.Helper()

	root := mustParse(t, src)

	var text string

	root.Walk(func(node *html.Node) {
		if node.Type != html.ElementNode || node.Name != "style" {
			return
		}

		for _, child := range node.Children {
			if child.Type == html.TextNode {
				text += child.Text
			}
		}
	})

	if strings.TrimSpace(text) == "" {
		t.Fatal("fixture style element is empty")
	}

	return text
}

func svgGeomImage(t *testing.T, res *Result) image.Image {
	t.Helper()

	for _, op := range opsOfKind(res, OpImage) {
		payload, _, _ := op.ImageBytes()
		if len(payload) == 0 {
			continue
		}

		img, err := png.Decode(bytes.NewReader(payload))
		if err != nil {
			t.Fatalf("decode svg image: %v", err)
		}

		return img
	}

	t.Fatal("layout produced no svg image")

	return nil
}

func svgGeomSerialize(t *testing.T, src string) string {
	t.Helper()

	root := mustParse(t, src)
	styles := resolveStyles(root, []*css.Stylesheet{svgGeomStylesheet(t, src)}, "print", testViewport, 800)
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

func svgGeomAssertBaked(t *testing.T, serialized, want, absent string) {
	t.Helper()

	if !strings.Contains(serialized, want) {
		t.Errorf("serialized SVG lacks %s, got %s", want, serialized)
	}

	if strings.Contains(serialized, absent) {
		t.Errorf("presentation attribute %s won over the stylesheet, got %s", absent, serialized)
	}
}
