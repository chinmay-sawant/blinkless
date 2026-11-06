package layout

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// simpleFlexInvoiceCSS is the shape the fast path exists for: a column flex
// container of row flex containers, each row holding fixed-width text cells.
const simpleFlexInvoiceCSS = `body{margin:0;font-size:10pt}` +
	`.invoice{display:flex;flex-direction:column;padding:16px}` +
	`.row{display:flex;flex-wrap:nowrap;border-bottom:1px solid #ddd;padding:2px 0}` +
	`.cell{width:50pt;padding:1px 4px;border:1px solid #ccc}`

// simpleFlexInvoiceDoc builds count invoice rows. The cells sit directly
// inside the row with no whitespace text nodes, so every in-flow flex child
// is an element with one text run.
func simpleFlexInvoiceDoc(count int) string {
	var doc strings.Builder

	doc.WriteString(`<div class="invoice">`)

	for r := range count {
		doc.WriteString(`<div class="row">`)

		for c := range 8 {
			fmt.Fprintf(&doc, `<span class="cell">R%dC%d</span>`, r, c)
		}

		doc.WriteString(`</div>`)
	}

	doc.WriteString(`</div>`)

	return doc.String()
}

// simpleFlexRowCase is one direct helper case. wantFast false means the helper
// must refuse the shape and leave the full measure in charge.
type simpleFlexRowCase struct {
	name     string
	css      string
	body     string
	wantFast bool
}

// simpleFlexRowCases covers every accepted ingredient once and the shapes the
// guards must refuse.
//
//nolint:funlen // one table of independent shape fixtures reads better together
func simpleFlexRowCases() []simpleFlexRowCase {
	const outerCSS = `#outer{display:flex;flex-direction:column;width:400pt}`

	cells := func(count int, class string) string {
		var body strings.Builder

		body.WriteString(`<div class="row" id="row">`)

		for i := range count {
			fmt.Fprintf(&body, `<span class="cell %s">cell%d</span>`, class, i)
		}

		body.WriteString(`</div>`)

		return body.String()
	}

	return []simpleFlexRowCase{
		{
			name: "fixed-widths",
			css:  outerCSS + `.row{display:flex}.cell{width:80pt}`,
			body: cells(3, "plain"), wantFast: true,
		},
		{
			name: "padding-and-border",
			css: outerCSS + `.row{display:flex}` +
				`.cell{width:80pt;padding:3pt 5pt;border:1.5pt solid #000}`,
			body: cells(3, "plain"), wantFast: true,
		},
		{
			name: "column-gap",
			css:  outerCSS + `.row{display:flex;gap:10pt}.cell{width:80pt}`,
			body: cells(3, "plain"), wantFast: true,
		},
		{
			name: "vertical-margins",
			css:  outerCSS + `.row{display:flex}.cell{width:80pt;margin:4pt 6pt}`,
			body: cells(3, "plain"), wantFast: true,
		},
		{
			name: "negative-margin",
			css:  outerCSS + `.row{display:flex}.cell{width:80pt;margin-top:-2pt;margin-bottom:3pt}`,
			body: cells(3, "plain"), wantFast: true,
		},
		{
			name: "line-height",
			css:  outerCSS + `.row{display:flex;line-height:30pt}.cell{width:80pt}`,
			body: cells(3, "plain"), wantFast: true,
		},
		{
			name: "mixed-font-and-line-height",
			css: outerCSS + `.row{display:flex}` +
				`.cell{width:80pt}.tall{font-size:20pt;line-height:36pt}`,
			body: cells(2, "tall"), wantFast: true,
		},
		{
			name: "border-box",
			css: outerCSS + `.row{display:flex}` +
				`.cell{box-sizing:border-box;width:100pt;padding:4pt;border:2pt solid #000}`,
			body: cells(3, "plain"), wantFast: true,
		},
		{
			name: "text-transform",
			css:  outerCSS + `.row{display:flex}.cell{width:80pt;text-transform:uppercase}`,
			body: cells(3, "plain"), wantFast: true,
		},
		{
			name: "wrap",
			css:  outerCSS + `.row{display:flex;flex-wrap:wrap}.cell{width:80pt}`,
			body: cells(3, "plain"), wantFast: false,
		},
		{
			name: "column-direction",
			css:  outerCSS + `.row{display:flex;flex-direction:column}.cell{width:80pt}`,
			body: cells(3, "plain"), wantFast: false,
		},
		{
			name: "nested-flex",
			css:  outerCSS + `.row{display:flex}.cell{display:flex;width:80pt}`,
			body: cells(3, "plain"), wantFast: false,
		},
		{
			name: "image",
			css:  outerCSS + `.row{display:flex}.cell{width:80pt}`,
			body: `<div class="row" id="row"><span class="cell">` +
				`<img src="missing.png" alt=""></span></div>`,
			wantFast: false,
		},
		{
			name:     "auto-width",
			css:      outerCSS + `.row{display:flex}.cell{padding:2pt}`,
			body:     cells(3, "plain"),
			wantFast: false,
		},
		{
			name: "text-does-not-fit",
			css:  outerCSS + `.row{display:flex}.cell{width:20pt}`,
			body: `<div class="row" id="row">` +
				`<span class="cell">Supercalifragilisticexpialidocious</span></div>`,
			wantFast: false,
		},
		{
			name: "percentage-height",
			css:  outerCSS + `.row{display:flex;height:50%}.cell{width:80pt}`,
			body: cells(3, "plain"), wantFast: false,
		},
		{
			name: "positioned-child",
			css:  outerCSS + `.row{display:flex}.cell{width:80pt;position:relative}`,
			body: cells(3, "plain"), wantFast: false,
		},
		{
			name: "percentage-child-width",
			css:  outerCSS + `.row{display:flex}.cell{width:25%}`,
			body: cells(3, "plain"), wantFast: false,
		},
		{
			name: "rtl-row",
			css:  outerCSS + `.row{display:flex;direction:rtl}.cell{width:80pt}`,
			body: cells(3, "plain"), wantFast: false,
		},
		{
			name: "pseudo-content",
			css:  outerCSS + `.row{display:flex}.cell{width:80pt}.cell::before{content:"x"}`,
			body: cells(3, "plain"), wantFast: false,
		},
	}
}

// simpleFlexEngineForCase builds one engine over the case's #outer tree and
// returns the row node and the column container's content width.
func simpleFlexEngineForCase(t *testing.T, testCase simpleFlexRowCase) (*engine, *html.Node, float64) {
	t.Helper()

	root := mustParse(t, `<div id="outer">`+testCase.body+`</div>`)
	opts := Options{
		Width: testViewport, Height: 800, Background: true,
		Sheets: []*css.Stylesheet{sheet(t, testCase.css)},
	}

	eng, outer := flexMinTestEngine(t, root, opts)
	row := findElementByID(t, root, "row")

	outerStyle := eng.stylePtr(outer)
	outerW := resolveUsedWidth(*outerStyle, eng.opts.Width, eng)
	_, contentW := eng.contentBox(0, outerW, outerStyle)

	if contentW <= 0 {
		t.Fatalf("column content width = %v, want positive", contentW)
	}

	return eng, row, contentW
}

// TestSimpleFlexRowHeightMatchesFullMeasure compares the fast height with the
// full noEmit subtree measure for every accepted shape, and pins the shapes
// that must fall back.
//
//nolint:paralleltest // flips the package bypass, so the cases must run serially
func TestSimpleFlexRowHeightMatchesFullMeasure(t *testing.T) {
	t.Cleanup(func() {
		simpleFlexHeightBypass.Store(false)
	})

	//nolint:paralleltest // flips the package bypass, so the cases must run serially
	for _, testCase := range simpleFlexRowCases() {
		t.Run(testCase.name, func(t *testing.T) {
			eng, row, contentW := simpleFlexEngineForCase(t, testCase)
			rowStyle := eng.stylePtr(row)

			simpleFlexHeightBypass.Store(true)

			oldH := eng.flexItemBaseHeight(row, rowStyle, contentW, -1)

			simpleFlexHeightBypass.Store(false)

			fastH, accepted := eng.simpleFlexRowHeight(row, rowStyle, contentW)

			if accepted != testCase.wantFast {
				t.Fatalf("fast path accepted = %v, want %v (fastH=%v oldH=%v)",
					accepted, testCase.wantFast, fastH, oldH)
			}

			if !accepted {
				return
			}

			if math.Abs(oldH-fastH) > 1e-9 {
				t.Fatalf("fast height %v differs from full measure %v", fastH, oldH)
			}
		})
	}
}

// layoutSimpleFlexCase lays out one case with the fast path on or bypassed.
// The caller must not run these concurrently: the bypass is package state.
func layoutSimpleFlexCase(t *testing.T, testCase simpleFlexRowCase, bypass bool) *Result {
	t.Helper()

	simpleFlexHeightBypass.Store(bypass)
	t.Cleanup(func() { simpleFlexHeightBypass.Store(false) })

	return layoutHTML(t, `<div id="outer">`+testCase.body+`</div>`, sheet(t, testCase.css))
}

// simpleFlexRowHeights returns the laid-out row heights in document order.
func simpleFlexRowHeights(res *Result) []float64 {
	var heights []float64

	for _, boxNode := range res.boxes {
		if boxNode.node != nil && boxNode.node.Attribute("class") == "row" {
			heights = append(heights, boxNode.height)
		}
	}

	return heights
}

// TestSimpleFlexRowHeightLayoutParity pins full-layout parity: the fast path
// and the forced old measure produce identical ops, page height, and row box
// heights for accepted and rejected shapes alike.
//
//nolint:paralleltest // flips the package bypass, so the cases must run serially
func TestSimpleFlexRowHeightLayoutParity(t *testing.T) {
	t.Cleanup(func() {
		simpleFlexHeightBypass.Store(false)
		simpleFlexHeightHits.Store(0)
	})

	//nolint:paralleltest // flips the package bypass, so the cases must run serially
	for _, testCase := range simpleFlexRowCases() {
		t.Run(testCase.name, func(t *testing.T) {
			simpleFlexHeightHits.Store(0)

			fast := layoutSimpleFlexCase(t, testCase, false)
			fastHits := simpleFlexHeightHits.Load()

			plain := layoutSimpleFlexCase(t, testCase, true)

			if testCase.wantFast && fastHits == 0 {
				t.Fatal("accepted shape never reached the fast path in full layout")
			}

			if !testCase.wantFast && fastHits != 0 {
				t.Fatalf("rejected shape reached the fast path %d times", fastHits)
			}

			if !opsDeeplyEqual(fast.Ops, plain.Ops) {
				t.Fatalf("ops differ: fast %d, bypass %d", len(fast.Ops), len(plain.Ops))
			}

			if fast.Height != plain.Height || fast.MaxContentX != plain.MaxContentX {
				t.Fatalf("result differs: height %.4f/%.4f maxX %.4f/%.4f",
					fast.Height, plain.Height, fast.MaxContentX, plain.MaxContentX)
			}

			fastRows := simpleFlexRowHeights(fast)
			plainRows := simpleFlexRowHeights(plain)

			if !reflect.DeepEqual(fastRows, plainRows) {
				t.Fatalf("row heights differ: fast %v, bypass %v", fastRows, plainRows)
			}
		})
	}
}

// TestSimpleFlexInvoiceFixtureHitsFastPath proves the benchmark fixture shape
// is accepted: every invoice row must take the fast height.
//
//nolint:paralleltest // flips the package bypass, so the test must run serially
func TestSimpleFlexInvoiceFixtureHitsFastPath(t *testing.T) {
	t.Cleanup(func() {
		simpleFlexHeightBypass.Store(false)
		simpleFlexHeightHits.Store(0)
	})

	const rows = 24

	simpleFlexHeightBypass.Store(false)

	simpleFlexHeightHits.Store(0)

	res := layoutHTML(t, simpleFlexInvoiceDoc(rows), sheet(t, simpleFlexInvoiceCSS))
	if res.Height <= 0 {
		t.Fatalf("invoice height = %v, want positive", res.Height)
	}

	if hits := simpleFlexHeightHits.Load(); hits < rows {
		t.Fatalf("fast path hits = %d, want at least %d invoice rows", hits, rows)
	}
}

// TestSimpleFlexInvoiceFixtureParity is the full-document differential for the
// benchmark fixture: 24 rows of fixed-width cells lay out identically with the
// fast path on and with it bypassed.
//
//nolint:paralleltest // flips the package bypass, so the test must run serially
func TestSimpleFlexInvoiceFixtureParity(t *testing.T) {
	t.Cleanup(func() {
		simpleFlexHeightBypass.Store(false)
		simpleFlexHeightHits.Store(0)
	})

	doc := simpleFlexInvoiceDoc(24)

	simpleFlexHeightBypass.Store(false)

	fast := layoutHTML(t, doc, sheet(t, simpleFlexInvoiceCSS))

	simpleFlexHeightBypass.Store(true)

	plain := layoutHTML(t, doc, sheet(t, simpleFlexInvoiceCSS))

	simpleFlexHeightBypass.Store(false)

	if !opsDeeplyEqual(fast.Ops, plain.Ops) {
		t.Fatalf("invoice ops differ: fast %d, bypass %d", len(fast.Ops), len(plain.Ops))
	}

	if fast.Height != plain.Height {
		t.Fatalf("invoice height differs: fast %.4f, bypass %.4f", fast.Height, plain.Height)
	}

	fastRows := simpleFlexRowHeights(fast)
	plainRows := simpleFlexRowHeights(plain)

	if len(fastRows) != 24 {
		t.Fatalf("row boxes = %d, want 24", len(fastRows))
	}

	if !reflect.DeepEqual(fastRows, plainRows) {
		t.Fatalf("invoice row heights differ: fast %v, bypass %v", fastRows, plainRows)
	}
}
