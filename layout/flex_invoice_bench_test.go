package layout_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	ilayout "github.com/chinmay-sawant/blinkless/internal/layout"
	"github.com/chinmay-sawant/blinkless/layout"
)

// flexInvoiceScaleCSS is the simple invoice shape: a column flex container of
// row flex containers, each row holding fixed-width text cells. Every row is a
// column flex item, so its base height runs through flexItemBaseHeight.
const flexInvoiceScaleCSS = `body{margin:0;font-size:10pt}` +
	`.invoice{display:flex;flex-direction:column;padding:16px}` +
	`.row{display:flex;flex-wrap:nowrap;border-bottom:1px solid #ddd;padding:2px 0}` +
	`.cell{width:50pt;padding:1px 4px;border:1px solid #ccc;background-color:#f8f8f8}`

// flexInvoiceScalePage builds count invoice rows with eight fixed-width cells
// each. The cells sit directly inside the row with no whitespace text nodes,
// so every in-flow flex child is an element with one text run.
func flexInvoiceScalePage(count int) string {
	var page strings.Builder

	page.WriteString(`<style>` + flexInvoiceScaleCSS + `</style><div class="invoice">`)

	for r := range count {
		page.WriteString(`<div class="row">`)

		for c := range 8 {
			fmt.Fprintf(&page, `<span class="cell">R%dC%d</span>`, r, c)
		}

		page.WriteString(`</div>`)
	}

	page.WriteString(`</div>`)

	return page.String()
}

// flexInvoiceDisplay lays out the fixture once with the fast path on or
// bypassed. The caller must not run these concurrently: the bypass is package
// state.
func flexInvoiceDisplay(t *testing.T, rows int, bypass bool) *layout.Display {
	t.Helper()

	ilayout.SetSimpleFlexHeightBypassForTest(bypass)
	t.Cleanup(func() { ilayout.SetSimpleFlexHeightBypassForTest(false) })

	tree, err := html.Parse([]byte(flexInvoiceScalePage(rows)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled, err := css.Apply(t.Context(), tree, css.Options{
		WidthPx:  800,
		HeightPx: 600,
		Media:    "screen",
	})
	if err != nil {
		t.Fatalf("css: %v", err)
	}

	display, err := layout.DisplayList(t.Context(), styled)
	if err != nil {
		t.Fatalf("display: %v", err)
	}

	return display
}

// TestFlexInvoiceHeightDifferential pins the public differential for the
// benchmark fixture: the fast row height and the full noEmit measure produce
// identical operations, boxes, and page height.
//
//nolint:paralleltest // flips the package bypass, so the test must run serially
func TestFlexInvoiceHeightDifferential(t *testing.T) {
	fast := flexInvoiceDisplay(t, 40, false)
	plain := flexInvoiceDisplay(t, 40, true)

	if len(fast.Ops) == 0 {
		t.Fatal("fast layout produced no ops")
	}

	if !reflect.DeepEqual(fast.Ops, plain.Ops) {
		t.Fatalf("ops differ: fast %d, bypass %d", len(fast.Ops), len(plain.Ops))
	}

	if !reflect.DeepEqual(fast.Boxes, plain.Boxes) {
		t.Fatalf("boxes differ: fast %d, bypass %d", len(fast.Boxes), len(plain.Boxes))
	}

	if fast.Height != plain.Height || fast.Width != plain.Width {
		t.Fatalf("result differs: height %d/%d width %d/%d",
			fast.Height, plain.Height, fast.Width, plain.Width)
	}
}

// BenchmarkFlexInvoiceHeight measures the same 1000-row fixture with the fast
// row height on and bypassed, so the two sub-benchmarks differ only in that
// path.
func BenchmarkFlexInvoiceHeight(b *testing.B) {
	tree, err := html.Parse([]byte(flexInvoiceScalePage(1000)))
	if err != nil {
		b.Fatalf("parse: %v", err)
	}

	styled, err := css.Apply(b.Context(), tree, css.Options{
		WidthPx:  800,
		HeightPx: 600,
		Media:    "screen",
	})
	if err != nil {
		b.Fatalf("css: %v", err)
	}

	b.Run("fast", func(b *testing.B) {
		ilayout.SetSimpleFlexHeightBypassForTest(false)
		defer ilayout.SetSimpleFlexHeightBypassForTest(false)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			display, err := layout.DisplayList(b.Context(), styled)
			if err != nil {
				b.Fatal(err)
			}

			if len(display.Ops) <= 5000 {
				b.Fatalf("invoice scale produced %d ops, want > 5000", len(display.Ops))
			}
		}
	})

	b.Run("bypass", func(b *testing.B) {
		ilayout.SetSimpleFlexHeightBypassForTest(true)
		defer ilayout.SetSimpleFlexHeightBypassForTest(false)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			display, err := layout.DisplayList(b.Context(), styled)
			if err != nil {
				b.Fatal(err)
			}

			if len(display.Ops) <= 5000 {
				b.Fatalf("invoice scale produced %d ops, want > 5000", len(display.Ops))
			}
		}
	})
}
