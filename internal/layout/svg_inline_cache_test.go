package layout

import (
	"bytes"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// findInlineSVGNode returns the first inline svg element of the fixture,
// failing the test when none is present.
func findInlineSVGNode(t *testing.T, root *html.Node) *html.Node {
	t.Helper()

	var svgNode *html.Node

	root.Walk(func(node *html.Node) {
		if svgNode == nil && node.Type == html.ElementNode && node.Name == cssTagSVG {
			svgNode = node
		}
	})

	if svgNode == nil {
		t.Fatal("no inline svg in fixture")
	}

	return svgNode
}

// TestBuildInlineSVGCachesDuplicateRaster pins the per-run raster cache for
// inline <svg>: building the same icon twice rasterizes once and reuses the
// stored payload instead of re-encoding.
func TestBuildInlineSVGCachesDuplicateRaster(t *testing.T) {
	t.Parallel()

	root := mustParse(t, svgPaintInlineIconSource(
		`<rect x="2" y="2" width="60" height="24" style="fill:#00ff00"/>`,
	))
	svgNode := findInlineSVGNode(t, root)

	opts := Options{Width: testViewport, Height: 800}

	styles, containers, err := resolveStylesForLayoutContext(t.Context(), root, opts)
	if err != nil {
		t.Fatalf("resolve styles: %v", err)
	}

	eng := newEngine(t.Context(), opts, nil, nil, styles, containers, nil)
	sty := eng.stylePtr(svgNode)

	if sty == nil {
		t.Fatal("inline svg has no resolved style")
	}

	first := eng.buildInlineSVG(svgNode, *sty, 0, 0, false)
	second := eng.buildInlineSVG(svgNode, *sty, 0, 0, false)

	if len(first.img.data) == 0 || len(second.img.data) == 0 {
		t.Fatal("inline svg built no raster")
	}

	if !bytes.Equal(first.img.data, second.img.data) {
		t.Fatal("duplicate icon re-encoded instead of reusing the cached raster")
	}

	// A hit returns the stored slice, so the payload is the same bytes, not a
	// second encode that happens to match.
	if &first.img.data[0] != &second.img.data[0] {
		t.Fatal("cache hit copied instead of sharing the stored raster")
	}

	if eng.svgInlineCache.hits != 1 || eng.svgInlineCache.misses != 1 {
		t.Fatalf("cache = %d hits, %d misses, want 1 and 1",
			eng.svgInlineCache.hits, eng.svgInlineCache.misses)
	}
}
