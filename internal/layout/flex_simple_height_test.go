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

// simpleFlexStackCSS is the invoice-app shape: flex rows sit in a plain block,
// not in a column flex item, so each row resolves its own line cross during the
// emit pass instead of through the column base height.
const simpleFlexStackCSS = `body{margin:0;font-size:10pt}` +
	`.stack{width:400pt;padding:8pt}` +
	`.row{display:flex;flex-wrap:nowrap;padding:2pt 0;border-bottom:1px solid #ddd}` +
	`.cell{padding:1pt 4pt;border:1px solid #ccc}`

// simpleFlexStackCase is one block-parent row document for the line-cross
// differential. wantFast false means the helper must refuse the row and keep
// the per-item measure.
type simpleFlexStackCase struct {
	name     string
	css      string
	body     string
	wantFast bool
}

// simpleFlexStackCases covers every accepted ingredient once and the shapes the
// line-cross guards must refuse.
//

func simpleFlexStackCases() []simpleFlexStackCase {
	const rowOpen = `<div class="stack" id="stack"><div class="row" id="row">`

	row := func(inner string) string {
		return rowOpen + inner + `</div></div>`
	}

	cells := func(count int) string {
		var body strings.Builder

		for i := range count {
			fmt.Fprintf(&body, `<span class="cell plain">cell%d</span>`, i)
		}

		return body.String()
	}

	return []simpleFlexStackCase{
		{
			name: "fixed-width-cells",
			css:  `.row{display:flex}.cell{width:80pt}`,
			body: row(cells(3)), wantFast: true,
		},
		{
			name: "padding",
			css:  `.row{display:flex}.cell{width:80pt;padding:3pt 5pt}`,
			body: row(cells(3)), wantFast: true,
		},
		{
			name: "borders",
			css:  `.row{display:flex}.cell{width:80pt;border:1.5pt solid #000}`,
			body: row(cells(3)), wantFast: true,
		},
		{
			name: "gap",
			css:  `.row{display:flex;gap:10pt}.cell{width:80pt}`,
			body: row(cells(3)), wantFast: true,
		},
		{
			name: "margins",
			css:  `.row{display:flex}.cell{width:80pt;margin:4pt 6pt}`,
			body: row(cells(3)), wantFast: true,
		},
		{
			name: "mixed-font-sizes",
			css:  `.row{display:flex}.cell{width:80pt}.tall{font-size:20pt}`,
			body: row(cells(2) + `<span class="cell tall">tall</span>`), wantFast: true,
		},
		{
			name: "row-line-height",
			css:  `.row{display:flex;line-height:30pt}.cell{width:80pt}`,
			body: row(cells(3)), wantFast: true,
		},
		{
			name: "border-box",
			css:  `.row{display:flex}.cell{box-sizing:border-box;width:90pt;padding:4pt}`,
			body: row(cells(3)), wantFast: true,
		},
		{
			name: "align-start",
			css:  `.row{display:flex;align-items:flex-start}.cell{width:80pt}`,
			body: row(cells(3)), wantFast: true,
		},
		{
			name: "wrapping-row",
			css:  `.row{display:flex;flex-wrap:wrap}.cell{width:80pt}`,
			body: row(cells(3)), wantFast: false,
		},
		{
			name: "column-direction",
			css:  `.row{display:flex;flex-direction:column}.cell{width:80pt}`,
			body: row(cells(3)), wantFast: false,
		},
		{
			name: "nested-flex-row",
			css:  `.row{display:flex}.cell{display:flex;width:80pt}`,
			body: row(cells(3)), wantFast: false,
		},
		{
			name:     "image",
			css:      `.row{display:flex}.cell{width:80pt}`,
			body:     row(`<span class="cell"><img src="missing.png" alt=""></span>`),
			wantFast: false,
		},
		{
			name: "auto-width",
			css:  `.row{display:flex}.cell{padding:2pt}`,
			body: row(cells(3)), wantFast: false,
		},
		{
			name:     "text-does-not-fit",
			css:      `.row{display:flex}.cell{width:20pt}`,
			body:     row(`<span class="cell">Supercalifragilisticexpialidocious</span>`),
			wantFast: false,
		},
		{
			name: "positioned-child",
			css:  `.row{display:flex}.cell{width:80pt;position:relative}`,
			body: row(cells(3)), wantFast: false,
		},
		{
			name: "percentage-height-row",
			css:  `.row{display:flex;height:50%}.cell{width:80pt}`,
			body: row(cells(3)), wantFast: false,
		},
	}
}

// layoutSimpleFlexStackDoc lays one block-parent document out with the
// line-cross fast path on or bypassed. The caller must not run these
// concurrently: the bypass is package state.
func layoutSimpleFlexStackDoc(t *testing.T, testCase simpleFlexStackCase, bypass bool) *Result {
	t.Helper()

	simpleFlexCrossBypass.Store(bypass)
	t.Cleanup(func() { simpleFlexCrossBypass.Store(false) })

	return layoutHTML(t, testCase.body, sheet(t, simpleFlexStackCSS+testCase.css))
}

// simpleFlexRect is one laid-out box's geometry, compared in document order.
type simpleFlexRect struct {
	x, y, w, h float64
}

// simpleFlexBoxRects returns every box's geometry in document order.
func simpleFlexBoxRects(res *Result) []simpleFlexRect {
	rects := make([]simpleFlexRect, 0, len(res.boxes))

	for _, boxNode := range res.boxes {
		rects = append(rects, simpleFlexRect{x: boxNode.x, y: boxNode.y, w: boxNode.w, h: boxNode.height})
	}

	return rects
}

// compareSimpleFlexLayout pins the full differential: identical ops, page
// height, content width, and every box position.
func compareSimpleFlexLayout(t *testing.T, fast, plain *Result) {
	t.Helper()

	if !opsDeeplyEqual(fast.Ops, plain.Ops) {
		t.Fatalf("ops differ: fast %d, bypass %d", len(fast.Ops), len(plain.Ops))
	}

	if fast.Height != plain.Height || fast.MaxContentX != plain.MaxContentX {
		t.Fatalf("result differs: height %.4f/%.4f maxX %.4f/%.4f",
			fast.Height, plain.Height, fast.MaxContentX, plain.MaxContentX)
	}

	fastRects := simpleFlexBoxRects(fast)
	plainRects := simpleFlexBoxRects(plain)

	if len(fastRects) != len(plainRects) {
		t.Fatalf("box count differs: fast %d, bypass %d", len(fastRects), len(plainRects))
	}

	for idx := range fastRects {
		if fastRects[idx] != plainRects[idx] {
			t.Fatalf("box %d differs: fast %+v, bypass %+v", idx, fastRects[idx], plainRects[idx])
		}
	}
}

// TestSimpleFlexRowContentCrossParity pins the line-cross fast path: a block
// parent gives every row an indefinite cross size, so the fast path derives
// the line height from the cell metrics and the bypass measures each item. Both
// must produce identical ops, heights, and boxes for accepted and refused
// shapes alike.
//
//nolint:paralleltest // flips the package bypass, so the cases must run serially
func TestSimpleFlexRowContentCrossParity(t *testing.T) {
	t.Cleanup(func() {
		simpleFlexCrossBypass.Store(false)
		simpleFlexCrossHits.Store(0)
	})

	//nolint:paralleltest // flips the package bypass, so the cases must run serially
	for _, testCase := range simpleFlexStackCases() {
		t.Run(testCase.name, func(t *testing.T) {
			simpleFlexCrossHits.Store(0)

			fast := layoutSimpleFlexStackDoc(t, testCase, false)
			fastHits := simpleFlexCrossHits.Load()

			plain := layoutSimpleFlexStackDoc(t, testCase, true)

			if testCase.wantFast && fastHits == 0 {
				t.Fatal("accepted row never reached the cross fast path in full layout")
			}

			if !testCase.wantFast && fastHits != 0 {
				t.Fatalf("rejected row reached the cross fast path %d times", fastHits)
			}

			compareSimpleFlexLayout(t, fast, plain)
		})
	}
}

// simpleFlexMinCase is one direct min-content helper case. wantFast false means
// the helper must refuse and leave measureCellMinMax in charge.
type simpleFlexMinCase struct {
	name     string
	css      string
	body     string
	wantFast bool
}

// simpleFlexMinCases covers the fixed-width single-run shapes the min-content
// fast path accepts and every wrap-affecting shape it must refuse.
//

func simpleFlexMinCases() []simpleFlexMinCase {
	const rowCSS = `.row{display:flex;width:300pt}`

	cell := func(inner string) string {
		return `<div class="row"><span class="cell" id="cell">` + inner + `</span></div>`
	}

	return []simpleFlexMinCase{
		{
			name: "fixed-width",
			css:  rowCSS + `.cell{width:100pt}`,
			body: cell("alpha beta gamma"), wantFast: true,
		},
		{
			name: "padding-and-border",
			css:  rowCSS + `.cell{width:100pt;padding:3pt 5pt;border:1.5pt solid #000}`,
			body: cell("alpha beta gamma"), wantFast: true,
		},
		{
			name: "block-display",
			css:  rowCSS + `.cell{display:block;width:100pt;padding:2pt}`,
			body: cell("alpha beta gamma"), wantFast: true,
		},
		{
			name: "letter-spacing",
			css:  rowCSS + `.cell{width:100pt;letter-spacing:1pt}`,
			body: cell("alpha beta gamma"), wantFast: true,
		},
		{
			name: "text-transform",
			css:  rowCSS + `.cell{width:100pt;text-transform:uppercase}`,
			body: cell("alpha beta gamma"), wantFast: true,
		},
		{
			name: "whitespace-heavy",
			css:  rowCSS + `.cell{width:100pt}`,
			body: cell("  alpha\tbeta\n gamma  "), wantFast: true,
		},
		{
			name: "single-long-word",
			css:  rowCSS + `.cell{width:20pt}`,
			body: cell("Supercalifragilisticexpialidocious"), wantFast: true,
		},
		{
			name: "auto-width",
			css:  rowCSS + `.cell{padding:2pt}`,
			body: cell("alpha beta gamma"), wantFast: false,
		},
		{
			name: "percentage-width",
			css:  rowCSS + `.cell{width:50%}`,
			body: cell("alpha beta gamma"), wantFast: false,
		},
		{
			name: "inline-block",
			css:  rowCSS + `.cell{display:inline-block;width:100pt}`,
			body: cell("alpha beta gamma"), wantFast: false,
		},
		{
			name: "nowrap",
			css:  rowCSS + `.cell{width:100pt;white-space:nowrap}`,
			body: cell("alpha beta gamma"), wantFast: false,
		},
		{
			name: "break-all",
			css:  rowCSS + `.cell{width:100pt;word-break:break-all}`,
			body: cell("alpha beta gamma"), wantFast: false,
		},
		{
			name: "overflow-wrap-anywhere",
			css:  rowCSS + `.cell{width:100pt;overflow-wrap:anywhere}`,
			body: cell("alpha beta gamma"), wantFast: false,
		},
		{
			name: "break-word",
			css:  rowCSS + `.cell{width:100pt;overflow-wrap:break-word}`,
			body: cell("alpha beta gamma"), wantFast: false,
		},
		{
			name: "keep-all",
			css:  rowCSS + `.cell{width:100pt;word-break:keep-all}`,
			body: cell("alpha beta gamma"), wantFast: false,
		},
		{
			name: "rtl",
			css:  rowCSS + `.cell{width:100pt;direction:rtl}`,
			body: cell("alpha beta gamma"), wantFast: false,
		},
		{
			name:     "nested-element",
			css:      rowCSS + `.cell{width:100pt}`,
			body:     `<div class="row"><span class="cell" id="cell">alpha <b>beta</b></span></div>`,
			wantFast: false,
		},
		{
			name: "image-child",
			css:  rowCSS + `.cell{width:100pt}`,
			body: `<div class="row"><span class="cell" id="cell">` +
				`<img src="missing.png" alt=""></span></div>`,
			wantFast: false,
		},
		{
			name: "size-containment",
			css:  rowCSS + `.cell{width:100pt;contain:size}`,
			body: cell("alpha beta gamma"), wantFast: false,
		},
	}
}

// simpleFlexMinHelperCase builds one engine over the case's #outer tree and
// returns the #cell node.
func simpleFlexMinHelperCase(t *testing.T, testCase simpleFlexMinCase) (*engine, *html.Node) {
	t.Helper()

	root := mustParse(t, `<div id="outer">`+testCase.body+`</div>`)
	opts := Options{
		Width: testViewport, Height: 800, Background: true,
		Sheets: []*css.Stylesheet{sheet(t, testCase.css)},
	}

	eng, _ := flexMinTestEngine(t, root, opts)
	cell := findElementByID(t, root, "cell")

	return eng, cell
}

// TestSimpleFlexMinContentWidthMatchesCellMeasure pins the min-content fast
// path to the full cell walk: accepted shapes return the identical min-content
// width and every refused shape keeps measureCellMinMax.
//
//nolint:paralleltest // flips the package bypass, so the cases must run serially
func TestSimpleFlexMinContentWidthMatchesCellMeasure(t *testing.T) {
	t.Cleanup(func() {
		simpleFlexMinBypass.Store(false)
		simpleFlexMinHits.Store(0)
	})

	//nolint:paralleltest // flips the package bypass, so the cases must run serially
	for _, testCase := range simpleFlexMinCases() {
		t.Run(testCase.name, func(t *testing.T) {
			eng, cell := simpleFlexMinHelperCase(t, testCase)
			style := eng.stylePtr(cell)

			simpleFlexMinHits.Store(0)

			fast, ok := eng.simpleFlexMinContentWidth(cell, style)

			if testCase.wantFast && simpleFlexMinHits.Load() == 0 {
				t.Fatal("accepted shape never reached the min-content fast path")
			}

			if ok != testCase.wantFast {
				t.Fatalf("fast path accepted = %v, want %v", ok, testCase.wantFast)
			}

			if !ok {
				return
			}

			want, _ := eng.measureCellMinMax(cell, style)
			if math.Abs(fast-want) > 1e-9 {
				t.Fatalf("fast min-content %v differs from cell measure %v", fast, want)
			}
		})
	}
}

// simpleFlexMinDocCSS shrinks two fixed-width cells well below their declared
// width, so the used widths land on the auto-minimum floor the min-content
// suggestion feeds. Each case supplies its own cell width.
const simpleFlexMinDocCSS = `body{margin:0;font-size:10pt}` +
	`#row{display:flex;flex-wrap:nowrap;width:40pt}`

// simpleFlexMinDoc is the forced-shrink fixture: the longest word is wider than
// the fair share, so the clamp and its min-content floor decide the used widths.
const simpleFlexMinDoc = `<div id="row">` +
	`<span class="cell" id="cell0">alpha beta</span>` +
	`<span class="cell" id="cell1">gamma delta</span></div>`

// layoutSimpleFlexMinDoc lays the shrink fixture out with the min-content fast
// path on or bypassed. The caller must not run these concurrently.
func layoutSimpleFlexMinDoc(t *testing.T, cssSrc, body string, bypass bool) *Result {
	t.Helper()

	simpleFlexMinBypass.Store(bypass)
	t.Cleanup(func() { simpleFlexMinBypass.Store(false) })

	return layoutHTML(t, body, sheet(t, simpleFlexMinDocCSS+cssSrc))
}

// TestSimpleFlexMinContentLayoutParity pins the min-content fast path at the
// document level: forcing the cells below their declared width exercises the
// auto-minimum floor, and the bypassed cell walk must produce identical ops,
// heights, and boxes.
//
//nolint:paralleltest // flips the package bypass, so the cases must run serially
func TestSimpleFlexMinContentLayoutParity(t *testing.T) {
	t.Cleanup(func() {
		simpleFlexMinBypass.Store(false)
		simpleFlexMinHits.Store(0)
	})

	cases := []struct {
		name     string
		css      string
		body     string
		wantFast bool
	}{
		{name: "forced-shrink", css: `.cell{width:100pt}`, body: simpleFlexMinDoc, wantFast: true},
		{
			name: "padded-text-transform",
			css:  `.cell{width:100pt;padding:2pt 3pt;border:1pt solid #000;text-transform:uppercase}`,
			body: simpleFlexMinDoc, wantFast: true,
		},
		{
			name: "nowrap-fallback",
			css:  `.cell{width:100pt;white-space:nowrap}`,
			body: simpleFlexMinDoc, wantFast: false,
		},
		{
			name: "auto-width-fallback",
			css:  `.cell{padding:2pt}`,
			body: simpleFlexMinDoc, wantFast: false,
		},
	}

	//nolint:paralleltest // flips the package bypass, so the cases must run serially
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			simpleFlexMinHits.Store(0)

			fast := layoutSimpleFlexMinDoc(t, testCase.css, testCase.body, false)
			fastHits := simpleFlexMinHits.Load()

			plain := layoutSimpleFlexMinDoc(t, testCase.css, testCase.body, true)

			if testCase.wantFast && fastHits == 0 {
				t.Fatal("accepted shape never reached the min-content fast path in full layout")
			}

			if !testCase.wantFast && fastHits != 0 {
				t.Fatalf("rejected shape reached the min-content fast path %d times", fastHits)
			}

			compareSimpleFlexLayout(t, fast, plain)
		})
	}
}
