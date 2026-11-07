package layout

import (
	"fmt"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// The benchmarks below isolate splice GRANULARITY, not absolute speed: each op
// toggles a background-color class on ONE middle row through the
// single-element subtree path (RelayoutSubtreeWithOverrides with no
// overrides, which fronts the subtree-reuse fingerprint check and delegates
// to RelayoutSubtree for the ordinary element patch). A row has no box of its
// own, so today's patch escalates to the owning table and rebuilds that
// subtree before splicing it back; the sibling
// BenchmarkRelayoutRowPatchVisibility bench measures that whole-table path on
// a two-row fixture. Here the per-row content is fixed (span-in-td, nested
// percent div, two plain text cells) and only the row count differs between
// the two benches, so the pair reads out granularity directly.
//
// Scaling expectation: a row-granular patch costs ~flat per-op across table
// sizes (the 120-row ns/op and B/op match the 60-row numbers, since only one
// row rebuilds), while a whole-table patch costs ~linear (the 120-row ns/op
// and B/op roughly double the 60-row numbers, since every row rebuilds).
// Compare the two benches' ns/op for time scaling and B/op (with
// ReportAllocs) for allocation scaling; allocs/op should track B/op.

// relayoutRowSpliceBenchCSS fixes the table width so the percent columns have
// a basis. The tint rule paints only the toggled row's cells: background
// color keeps the row's geometry, so every patch splices instead of reporting
// errSubtreeResized.
const relayoutRowSpliceBenchCSS = `
	table.grid { width: 640px; table-layout: fixed; border-collapse: collapse; }
	td { padding: 4px; border: 1px solid #999999; }
	td.c-a { width: 30%; }
	td.c-b { width: 30%; }
	td.c-c { width: 20%; }
	td.c-d { width: 20%; }
	tr.tint td { background-color: #fff3bf; }
	.box { width: 50%; height: 18px; background-color: #eeeeee; }
`

// relayoutRowSpliceBenchRowsSmall and relayoutRowSpliceBenchRowsLarge are the
// two table sizes. Per-row content is identical between them; only the row
// count differs, so any per-op cost delta across the pair is the table-size
// term.
const (
	relayoutRowSpliceBenchRowsSmall = 60
	relayoutRowSpliceBenchRowsLarge = 120
)

// relayoutRowSpliceBenchHTML builds a rows-by-four table: every row carries a
// span in its first cell, a nested percent-width div in its second, and plain
// text in its third and fourth. The middle row carries id="target" so the
// bench loop can toggle exactly one row; its cells keep the same shape as
// every other row.
func relayoutRowSpliceBenchHTML(rows int) string {
	var b strings.Builder

	b.WriteString(`<html><body><table class="grid">`)

	middle := rows / 2

	for i := range rows {
		if i == middle {
			fmt.Fprintf(&b, `<tr id="target">`)
		} else {
			b.WriteString(`<tr>`)
		}

		fmt.Fprintf(&b, `<td class="c-a"><span>Item %04d</span></td>`, i)
		fmt.Fprintf(&b, `<td class="c-b"><div class="box">%d pcs</div></td>`, i)
		fmt.Fprintf(&b, `<td class="c-c">%d pcs</td>`, i)
		fmt.Fprintf(&b, `<td class="c-d">9.99</td>`)

		b.WriteString(`</tr>`)
	}

	b.WriteString(`</table></body></html>`)

	return b.String()
}

// relayoutRowSpliceBenchSetup parses the fixture CSS, lays one N-row table
// out once, and hands back the middle target row plus the retained result and
// the layout options the patch loop reuses. Nothing else runs inside the timed
// loop: each op only flips the target row's class.
func relayoutRowSpliceBenchSetup(b *testing.B, rows int) (*html.Node, *Result, Options) {
	b.Helper()

	sheet, err := css.Parse(relayoutRowSpliceBenchCSS)
	if err != nil {
		b.Fatalf("css.Parse: %v", err)
	}

	root, err := html.Parse(relayoutRowSpliceBenchHTML(rows))
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

	return row, prev, opts
}

// relayoutRowSpliceBenchToggle patches the target row once per op,
// alternating its class between tinted and plain. Alternating defeats the
// content-fingerprint no-op fast path on every iteration, so each op measures
// the miss path: the subtree rebuild plus the operation splice. No full
// Layout runs inside the loop.
func relayoutRowSpliceBenchToggle(b *testing.B, rows int) {
	b.Helper()

	row, prev, opts := relayoutRowSpliceBenchSetup(b, rows)

	ctx := b.Context()
	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		if i%2 == 0 {
			row.Attrs["class"] = "tint"
		} else {
			row.Attrs["class"] = ""
		}

		if _, err := RelayoutSubtreeWithOverrides(ctx, prev, row, nil, opts); err != nil {
			b.Fatalf("RelayoutSubtreeWithOverrides: %v", err)
		}
	}
}

// BenchmarkRelayoutRowSplice60 toggles one middle row of a 60-row table per
// op. Pair with BenchmarkRelayoutRowSplice120: flat ns/op and B/op across the
// pair means row-granular cost; ~2x on the 120-row table means whole-table
// cost.
func BenchmarkRelayoutRowSplice60(b *testing.B) {
	relayoutRowSpliceBenchToggle(b, relayoutRowSpliceBenchRowsSmall)
}

// BenchmarkRelayoutRowSplice120 toggles one middle row of a 120-row table per
// op, with the same per-row content as BenchmarkRelayoutRowSplice60. Only the
// row count doubles, so any per-op cost growth over the 60-row bench is the
// table-size term.
func BenchmarkRelayoutRowSplice120(b *testing.B) {
	relayoutRowSpliceBenchToggle(b, relayoutRowSpliceBenchRowsLarge)
}
