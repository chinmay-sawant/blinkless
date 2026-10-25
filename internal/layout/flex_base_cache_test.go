package layout

import (
	"context"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// TestFlexColumnBaseHeightMemoized pins the per-run memo for column flex
// base heights: measuring the same items twice under the same containing
// size returns identical heights without re-laying-out the subtrees.
func TestFlexColumnBaseHeightMemoized(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<div id="outer"><div class="cell"><span>alpha</span></div><div class="cell"><span>beta</span></div></div>`)
	opts := Options{
		Width: testViewport, Height: 800, Background: true,
		Sheets: []*css.Stylesheet{sheet(t, `#outer{display:flex;flex-direction:column}.cell{display:flex;flex-direction:column}`)},
	}

	styles, containers, err := resolveStylesForLayoutContext(context.Background(), root, opts)
	if err != nil {
		t.Fatalf("resolve styles: %v", err)
	}

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatalf("load faces: %v", err)
	}

	eng := newEngine(context.Background(), opts, faces, faces.Regular, styles, containers, nil)

	var outer *html.Node

	root.Walk(func(node *html.Node) {
		if outer == nil && node.Type == html.ElementNode && node.Attribute("id") == "outer" {
			outer = node
		}
	})

	if outer == nil {
		t.Fatal("missing #outer element")
	}

	kids := eng.flexChildren(outer, eng.stylePtr(outer))
	if len(kids) != 2 {
		t.Fatalf("flex children = %d, want 2", len(kids))
	}

	first := eng.flexColumnItems(kids, testViewport, -1)
	cached := len(eng.flexBaseH)

	if cached == 0 {
		t.Fatal("flexColumnItems left no memoized base heights")
	}

	second := eng.flexColumnItems(kids, testViewport, -1)
	if len(eng.flexBaseH) != cached {
		t.Fatalf("second flexColumnItems grew the memo %d -> %d", cached, len(eng.flexBaseH))
	}

	for idx := range first {
		if first[idx].baseH != second[idx].baseH {
			t.Fatalf("item %d baseH changed %v -> %v", idx, first[idx].baseH, second[idx].baseH)
		}

		if first[idx].baseH <= 0 {
			t.Fatalf("item %d baseH = %v, want positive", idx, first[idx].baseH)
		}
	}
}
