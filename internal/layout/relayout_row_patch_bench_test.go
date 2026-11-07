package layout

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// The benchmark below covers the engine path a table-row visibility toggle
// takes: RelayoutSubtreeWithOverrides with no overrides, which fronts the
// subtree-reuse fingerprint check and delegates to RelayoutSubtree for the
// ordinary element patch. A row has no box of its own, so the patch escalates
// to the owning table and rebuilds that subtree before splicing it back. The
// fixture is a small inventory-style table: two rows by three cells with
// percent column widths, one span inside a td, and one nested div.

// relayoutRowPatchBenchCSS fixes the table width so the percent columns have
// a basis, and hides only the paint of the toggled row: visibility keeps the
// row's geometry, so every patch splices instead of reporting
// errSubtreeResized.
const relayoutRowPatchBenchCSS = `
	table.grid { width: 480px; table-layout: fixed; border-collapse: collapse; }
	td { padding: 4px; border: 1px solid #999999; }
	td.c-name { width: 40%; }
	td.c-qty { width: 35%; }
	td.c-price { width: 25%; }
	tr.hidden { visibility: hidden; }
	.box { width: 60px; height: 18px; background-color: #eeeeee; }
`

// relayoutRowPatchBenchHTML holds a two-by-three table: the target row carries
// a span in its first cell and a nested div in its third, and the second row
// holds plain text cells beside it.
const relayoutRowPatchBenchHTML = `<html><body>` +
	`<table class="grid">` +
	`<tr id="target">` +
	`<td class="c-name"><span>Widget A</span></td>` +
	`<td class="c-qty">12 pcs</td>` +
	`<td class="c-price"><div class="box">9.99</div></td>` +
	`</tr>` +
	`<tr>` +
	`<td class="c-name">Widget B</td>` +
	`<td class="c-qty">7 pcs</td>` +
	`<td class="c-price">4.50</td>` +
	`</tr>` +
	`</table>` +
	`</body></html>`

// BenchmarkRelayoutRowPatchVisibility lays the table out once, then patches
// the target row once per op, alternating its class between hidden and
// visible. Alternating defeats the content-fingerprint no-op fast path on
// every iteration, so each op measures the miss path: the table subtree
// rebuild plus the operation splice. No full Layout runs inside the loop.
func BenchmarkRelayoutRowPatchVisibility(b *testing.B) {
	sheet, err := css.Parse(relayoutRowPatchBenchCSS)
	if err != nil {
		b.Fatalf("css.Parse: %v", err)
	}

	root, err := html.Parse(relayoutRowPatchBenchHTML)
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

	row := root.FindFirst(func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Attribute("id") == "target"
	})
	if row == nil {
		b.Fatal("no element with id target")
	}

	ctx := b.Context()
	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		if i%2 == 0 {
			row.Attrs["class"] = "hidden"
		} else {
			row.Attrs["class"] = ""
		}

		if _, err := RelayoutSubtreeWithOverrides(ctx, prev, row, nil, opts); err != nil {
			b.Fatalf("RelayoutSubtreeWithOverrides: %v", err)
		}
	}
}
