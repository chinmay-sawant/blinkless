package layout

import (
	"fmt"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// The benchmarks below cover the engine path public layout.RelayoutElement
// calls for a single-element text patch: RelayoutSubtreeWithOverrides with no
// overrides, which fronts the subtree-reuse fingerprint check and delegates to
// RelayoutSubtree for the ordinary element patch. The fixture is a small
// flex/grid page with an id-bearing detail line, the shape a hover detail-line
// patch edits.

// relayoutPatchBenchCSS lays a flex column around a two-column grid. The
// detail target carries a fixed size so a text edit keeps its geometry and
// the splice succeeds instead of reporting errSubtreeResized.
const relayoutPatchBenchCSS = `
	.wrap { display: flex; flex-direction: column; width: 320px; }
	.grid { display: grid; grid-template-columns: 1fr auto; gap: 8px; }
	.card { width: 200px; height: 60px; background-color: #eeeeee; }
	#detail { width: 180px; height: 20px; }
	#side { width: 80px; height: 20px; }
`

// relayoutPatchBenchHTML holds seven elements: html, body, the flex wrap, the
// grid, the detail target, its side sibling, and a tail card below the grid.
const relayoutPatchBenchHTML = `<html><body>` +
	`<div class="wrap">` +
	`<div class="grid">` +
	`<div id="detail">detail-0000</div>` +
	`<div id="side">side</div>` +
	`</div>` +
	`<div id="tail" class="card">tail</div>` +
	`</div>` +
	`</body></html>`

// relayoutPatchBenchSetup parses the fixture, lays it out once, and hands back
// the id-bearing target element, its text child, the retained result, and the
// layout options the patch loop reuses.
func relayoutPatchBenchSetup(b *testing.B) (*html.Node, *html.Node, *Result, Options) {
	b.Helper()

	sheet, err := css.Parse(relayoutPatchBenchCSS)
	if err != nil {
		b.Fatalf("css.Parse: %v", err)
	}

	root, err := html.Parse(relayoutPatchBenchHTML)
	if err != nil {
		b.Fatalf("html.Parse: %v", err)
	}

	opts := Options{ //nolint:exhaustruct // benchmark uses the standard layout inputs
		Width:      testViewport,
		Height:     800,
		Sheets:     []*css.Stylesheet{sheet},
		Background: true,
	}

	prev, err := Layout(root, opts)
	if err != nil {
		b.Fatalf("Layout: %v", err)
	}

	target := root.FindFirst(func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Attribute("id") == "detail"
	})
	if target == nil {
		b.Fatal("no element with id detail")
	}

	var textNode *html.Node

	for _, child := range target.Children {
		if child.Type == html.TextNode {
			textNode = child

			break
		}
	}

	if textNode == nil {
		b.Fatal("detail element holds no text node")
	}

	return target, textNode, prev, opts
}

// BenchmarkRelayoutPatchRepeat patches the detail line with one value every
// iteration. After the first patch the retained operations already carry that
// text, so every later call meets the content-fingerprint equality in the
// reuse check and exercises the hit path.
func BenchmarkRelayoutPatchRepeat(b *testing.B) {
	target, textNode, prev, opts := relayoutPatchBenchSetup(b)
	ctx := b.Context()
	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		textNode.Text = "detail-0000"

		if _, err := RelayoutSubtreeWithOverrides(ctx, prev, target, nil, opts); err != nil {
			b.Fatalf("RelayoutSubtreeWithOverrides: %v", err)
		}
	}
}

// BenchmarkRelayoutPatchVarying patches the detail line with a new value every
// iteration. Every text differs from the retained operations, so each call
// defeats the content-fingerprint equality and exercises the miss path that
// falls back to the full element build. The variants share one length so the
// fixed-size target keeps its geometry and every patch succeeds.
func BenchmarkRelayoutPatchVarying(b *testing.B) {
	target, textNode, prev, opts := relayoutPatchBenchSetup(b)

	const variantCount = 1024

	texts := make([]string, variantCount)
	for i := range texts {
		texts[i] = fmt.Sprintf("detail-%04d", i)
	}

	ctx := b.Context()
	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		textNode.Text = texts[i%len(texts)]

		if _, err := RelayoutSubtreeWithOverrides(ctx, prev, target, nil, opts); err != nil {
			b.Fatalf("RelayoutSubtreeWithOverrides: %v", err)
		}
	}
}
