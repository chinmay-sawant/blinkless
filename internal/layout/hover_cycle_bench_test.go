package layout

import (
	"fmt"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// The cycle below mirrors the app's real hover frame: one full Layout of a
// small flex/grid fixture, then repeated element text patches with varying
// text plus one hover-scope evaluation per patch, then one more full Layout
// standing in for the parse cycle.

// hoverCycleBenchCSS lays a flex column around a two-column grid. The detail
// target carries a fixed size so a text edit keeps its geometry and the patch
// splices instead of reporting errSubtreeResized. The :hover rule is
// paint-only so the scope evaluation has one real hover selector to resolve.
const hoverCycleBenchCSS = `
	.wrap { display: flex; flex-direction: column; width: 320px; }
	.grid { display: grid; grid-template-columns: 1fr auto; gap: 8px; }
	.card { width: 200px; height: 60px; background-color: #eeeeee; }
	#detail { width: 180px; height: 20px; }
	#detail:hover { background-color: #dddddd; }
	#side { width: 80px; height: 20px; }
`

// hoverCycleBenchHTML holds seven elements: html, body, the flex wrap, the
// grid, the detail target, its side sibling, and a tail card below the grid.
const hoverCycleBenchHTML = `<html><body>` +
	`<div class="wrap">` +
	`<div class="grid">` +
	`<div id="detail">detail-0000</div>` +
	`<div id="side">side</div>` +
	`</div>` +
	`<div id="tail" class="card">tail</div>` +
	`</div>` +
	`</body></html>`

const (
	// hoverCyclePatchCount is the number of text patches per measured cycle.
	hoverCyclePatchCount = 8
	// hoverCycleVariantCount bounds the precomputed text variants. Every
	// variant shares one length so the fixed-size target keeps its geometry.
	hoverCycleVariantCount = 64
)

var hoverCycleSink int //nolint:gochecknoglobals // benchmark sink

// hoverCycleHoverSelectors collects the hover-dependent selectors once, the
// way the shared hover list caches them outside the per-frame walk.
func hoverCycleHoverSelectors(sheets []*css.Stylesheet) []css.Selector {
	var out []css.Selector

	for _, sheet := range sheets {
		if sheet == nil {
			continue
		}

		for _, rule := range sheet.Rules {
			for _, sel := range rule.Selectors {
				if css.SelectorUsesHover(sel) {
					out = append(out, sel)
				}
			}
		}
	}

	return out
}

// hoverCycleChanged mirrors the css.Document.HoverChangedScopes core: it walks
// the tree and reports how many elements match differently under old and next.
// It stays in-package on internal/css so the bench needs no host document.
func hoverCycleChanged(root *html.Node, selectors []css.Selector, old, next css.MatchState) int {
	changed := 0

	var walk func(*html.Node)

	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			for _, sel := range selectors {
				if len(sel.Parts) == 0 {
					continue
				}

				pseudo := sel.Parts[len(sel.Parts)-1].PseudoElement

				var before, after bool

				if pseudo != "" {
					before = old.MatchesPseudo(sel, node, pseudo)
					after = next.MatchesPseudo(sel, node, pseudo)
				} else {
					before = old.Matches(sel, node)
					after = next.Matches(sel, node)
				}

				if before != after {
					changed++

					break
				}
			}
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(root)

	return changed
}

// BenchmarkHoverCycle measures one app cycle per op: a full Layout, then
// hoverCyclePatchCount varying-text element patches each followed by a hover
// scope evaluation, then one more full Layout for the parse-cycle equivalent.
func BenchmarkHoverCycle(b *testing.B) {
	sheet, err := css.Parse(hoverCycleBenchCSS)
	if err != nil {
		b.Fatalf("css.Parse: %v", err)
	}

	root, err := html.Parse(hoverCycleBenchHTML)
	if err != nil {
		b.Fatalf("html.Parse: %v", err)
	}

	sheets := []*css.Stylesheet{sheet}
	selectors := hoverCycleHoverSelectors(sheets)

	opts := Options{ //nolint:exhaustruct // benchmark uses the standard layout inputs
		Width:      testViewport,
		Height:     800,
		Sheets:     sheets,
		Background: true,
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

	texts := make([]string, hoverCycleVariantCount)
	for i := range texts {
		texts[i] = fmt.Sprintf("detail-%04d", i)
	}

	ctx := b.Context()
	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		prev, err := Layout(root, opts)
		if err != nil {
			b.Fatalf("Layout: %v", err)
		}

		for k := range hoverCyclePatchCount {
			textNode.Text = texts[(i*hoverCyclePatchCount+k)%len(texts)]

			if _, err := RelayoutSubtreeWithOverrides(ctx, prev, target, nil, opts); err != nil {
				b.Fatalf("RelayoutSubtreeWithOverrides: %v", err)
			}

			previous := ""
			if k%2 == 1 {
				previous = "detail"
			}

			old := css.MatchState{Hover: previous}
			next := css.MatchState{Hover: "detail"}

			hoverCycleSink += hoverCycleChanged(root, selectors, old, next)
		}

		if _, err := Layout(root, opts); err != nil {
			b.Fatalf("Layout: %v", err)
		}
	}
}
