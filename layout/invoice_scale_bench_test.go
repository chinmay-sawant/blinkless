package layout_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

// invoiceScalePage builds a deterministic 1000-row by 8-cell invoice table
// from plain divs and spans with classes and no external assets.
func invoiceScalePage() string {
	var page strings.Builder

	page.WriteString(`<style>
		body { margin: 0; font-size: 10pt }
		.invoice { padding: 16px }
		.header { padding: 8px 2px; border-bottom: 2px solid #222 }
		.title { color: #111111 }
		.table { padding: 8px 0 }
		.row { border-bottom: 1px solid #ddd; padding: 2px 0 }
		.cell { color: #223344; padding: 1px 4px; border: 1px solid #ccc; background-color: #f8f8f8 }
		.total { color: #445566; text-align: right }
	</style><div class="invoice"><div class="header"><span class="title">` +
		`Invoice INV-2026-001</span></div><div class="table">`)

	for r := range 1000 {
		page.WriteString(`<div class="row">`)
		for c := range 8 {
			fmt.Fprintf(&page, `<span class="cell">R%dC%d</span>`, r, c)
		}
		page.WriteString(`</div>`)
	}

	page.WriteString(`</div><div class="total">Total $8000.00</div></div>`)

	return page.String()
}

// BenchmarkInvoiceScale measures one full document build for a 1000-row
// invoice: html parse plus css cascade plus layout to DisplayList.
func BenchmarkInvoiceScale(b *testing.B) {
	src := []byte(invoiceScalePage())

	b.ResetTimer()

	for b.Loop() {
		tree, err := html.Parse(src)
		if err != nil {
			b.Fatal(err)
		}

		styled, err := css.Apply(b.Context(), tree, css.Options{WidthPx: 800, HeightPx: 600, Media: "screen"})
		if err != nil {
			b.Fatal(err)
		}

		disp, err := layout.DisplayList(b.Context(), styled)
		if err != nil {
			b.Fatal(err)
		}

		if len(disp.Ops) <= 5000 {
			b.Fatalf("invoice scale produced %d ops, want > 5000", len(disp.Ops))
		}
	}
}
