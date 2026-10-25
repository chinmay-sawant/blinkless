package layout

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// gridMemoNestedDoc is an outer grid of cards each holding an inner grid on
// the general two-pass path (explicit template rows defeat the single-column
// and auto-row fast paths). Line breaks fall on grid-item boundaries, where
// whitespace text is inert, so the source stays under the line limit without
// changing layout.
const gridMemoNestedDoc = `<html><body>
<div class="outer">
<div class="card"><h2>Alpha market</h2><div class="inner"><div>price one</div><div>change up</div>
<div>volume high</div><div>cap large</div></div><p>card one footer text here</p></div>
<div class="card"><h2>Beta market</h2><div class="inner"><div>price two</div><div>change down</div>
<div>volume low</div><div>cap small</div></div><p>card two footer text here</p></div>
<div class="card"><h2>Gamma market</h2><div class="inner"><div>price three</div><div>change flat</div>
<div>volume mid</div><div>cap mid</div></div><p>card three footer text here</p></div>
<div class="card"><h2>Delta market</h2><div class="inner"><div>price four</div><div>change up</div>
<div>volume high</div><div>cap large</div></div><p>card four footer text here</p></div>
<div class="card"><h2>Epsilon market</h2><div class="inner"><div>price five</div><div>change down</div>
<div>volume low</div><div>cap small</div></div><p>card five footer text here</p></div>
<div class="card"><h2>Zeta market</h2><div class="inner"><div>price six</div><div>change flat</div>
<div>volume mid</div><div>cap mid</div></div><p>card six footer text here</p></div>
</div></body></html>`

const gridMemoNestedCSS = `
body{margin:0}
.outer{display:grid;grid-template-columns:repeat(3,1fr);grid-template-rows:auto auto;gap:6pt;width:480pt}
.card{background:#eee}
.inner{display:grid;grid-template-columns:1fr 1fr;grid-template-rows:auto auto;gap:4pt}
`

// gridMemoOpsComparable returns a copy of ops with the font pointers cleared.
// Faces are run-shared caches whose internal shaping state differs between
// Layout calls; geometry, text, and paint payload stay fully compared.
func gridMemoOpsComparable(ops []Op) []Op {
	out := make([]Op, len(ops))
	copy(out, ops)

	for index := range out {
		out[index].Font = nil
	}

	return out
}

// checkGridMemoIdentical lays out src with and without the memo and requires
// byte-identical display lists.
func checkGridMemoIdentical(t *testing.T, src, cssSrc string, images func(string) ([]byte, error)) {
	t.Helper()

	sheets := []*css.Stylesheet{sheet(t, cssSrc)}

	layoutWith := func(bypass bool) *Result {
		root := mustParse(t, src)

		opts := Options{
			Width: testViewport, Height: 800, Sheets: sheets, Background: true,
			Images: images,
		}
		opts.gridMemoBypass = bypass

		res, err := Layout(root, opts)
		if err != nil {
			t.Fatalf("Layout(bypass=%v): %v", bypass, err)
		}

		return res
	}

	cached := layoutWith(false)
	plain := layoutWith(true)

	if !reflect.DeepEqual(gridMemoOpsComparable(cached.Ops), gridMemoOpsComparable(plain.Ops)) {
		t.Fatalf("memoized ops differ: %d vs %d ops", len(cached.Ops), len(plain.Ops))
	}

	if cached.Height != plain.Height || cached.MaxContentX != plain.MaxContentX ||
		cached.HasFragmentLinks != plain.HasFragmentLinks {
		t.Fatalf(
			"memoized result differs: height %.3f/%.3f maxX %.3f/%.3f frag %v/%v",
			cached.Height, plain.Height, cached.MaxContentX, plain.MaxContentX,
			cached.HasFragmentLinks, plain.HasFragmentLinks,
		)
	}
}

// TestGridMemoIdenticalOutput pins the memo's transparency: nested grids on
// the general measure-then-emit path produce identical display lists with and
// without the cache.
func TestGridMemoIdenticalOutput(t *testing.T) {
	t.Parallel()

	checkGridMemoIdentical(t, gridMemoNestedDoc, gridMemoNestedCSS, nil)
}

// TestGridMemoIdenticalOutputResponsiveImage pins the imgMaxW key ambient: a
// display:block max-width:100% image inside a nested grid measures differently
// under the outer measure pass (ambient clamp) and the outer emit pass (cell
// clamp), so the second inner measure must rebuild instead of reusing the
// first height.
func TestGridMemoIdenticalOutputResponsiveImage(t *testing.T) {
	t.Parallel()

	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
		0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xfe, 0xd4, 0xef, 0x00, 0x00,
		0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}

	const doc = `<html><body>
<div class="outer">
<div class="card"><div class="inner"><div><img class="ph" width="700" src="p.png"></div><div>alpha one</div>
<div>alpha two</div><div>alpha three</div></div></div>
<div class="card"><div class="inner"><div><img class="ph" width="700" src="p.png"></div><div>beta one</div>
<div>beta two</div><div>beta three</div></div></div>
</div></body></html>`

	const cssSrc = `
body{margin:0}
.outer{display:grid;grid-template-columns:1fr 1fr;grid-template-rows:auto;gap:6pt;width:480pt;align-items:start}
.inner{display:grid;grid-template-columns:1fr 1fr;grid-template-rows:auto auto;gap:4pt}
.ph{display:block;max-width:100%;height:auto}
`

	images := func(string) ([]byte, error) { return png, nil }
	checkGridMemoIdentical(t, doc, cssSrc, images)
}

// gridMemoCellNode returns the first .cell element under root.
func gridMemoCellNode(t *testing.T, root *html.Node) *html.Node {
	t.Helper()

	var found *html.Node

	root.Walk(func(node *html.Node) {
		if found != nil || node.Type != html.ElementNode || node.Attribute("class") != "cell" {
			return
		}

		found = node
	})

	if found == nil {
		t.Fatal("missing .cell element")
	}

	return found
}

// gridMemoEngine builds an engine over root for direct memo unit tests,
// mirroring TestFlexColumnBaseHeightMemoized.
func gridMemoEngine(t *testing.T, root *html.Node, opts Options) *engine {
	t.Helper()

	styles, containers, err := resolveStylesForLayoutContext(t.Context(), root, opts)
	if err != nil {
		t.Fatalf("resolve styles: %v", err)
	}

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatalf("load faces: %v", err)
	}

	return newEngine(t.Context(), opts, faces, faces.Regular, styles, containers, nil)
}

// TestGridMemoPrefHHit pins the per-run memo: measuring the same item twice at
// the same containing width returns the same height without growing the cache,
// while a new width measures and stores again.
func TestGridMemoPrefHHit(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<div id="g"><div class="cell"><span>alpha beta gamma delta</span></div></div>`)
	opts := Options{
		Width: testViewport, Height: 800, Background: true,
		Sheets: []*css.Stylesheet{sheet(t, `#g{width:400pt}.cell{background:#eee}`)},
	}
	eng := gridMemoEngine(t, root, opts)
	cell := gridMemoCellNode(t, root)

	first := measureGridItemPrefH(eng, cell, 200, 0, 0)
	if first <= 0 {
		t.Fatalf("prefH = %v, want positive", first)
	}

	if cached := len(eng.gridMeasureCache); cached == 0 {
		t.Fatal("first measure left no memoized entry")
	}

	second := measureGridItemPrefH(eng, cell, 200, 0, 0)
	if second != first {
		t.Fatalf("second prefH = %v, want %v", second, first)
	}

	if cached := len(eng.gridMeasureCache); cached != 1 {
		t.Fatalf("second measure grew the memo to %d entries, want 1", cached)
	}

	measureGridItemPrefH(eng, cell, 100, 0, 0)

	if cached := len(eng.gridMeasureCache); cached != 2 {
		t.Fatalf("new width left %d entries, want 2", cached)
	}
}

// TestGridMemoBypassDisablesCache pins the test-only bypass: with bypass set,
// measurements run uncached and store nothing.
func TestGridMemoBypassDisablesCache(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<div id="g"><div class="cell"><span>alpha beta</span></div></div>`)
	opts := Options{
		Width: testViewport, Height: 800, Background: true,
		Sheets: []*css.Stylesheet{sheet(t, `#g{width:400pt}`)},
	}
	opts.gridMemoBypass = true

	eng := gridMemoEngine(t, root, opts)
	cell := gridMemoCellNode(t, root)

	if height := measureGridItemPrefH(eng, cell, 200, 0, 0); height <= 0 {
		t.Fatalf("bypassed prefH = %v, want positive", height)
	}

	if cached := len(eng.gridMeasureCache); cached != 0 {
		t.Fatalf("bypassed measure stored %d entries, want 0", cached)
	}
}

// BenchmarkGridMemoNested compares a trade-like nested-grid document with and
// without the memo.
func BenchmarkGridMemoNested(b *testing.B) {
	gridMemoBenchLayouts(b, gridMemoNestedDoc, gridMemoNestedCSS)
}

// BenchmarkGridMemoMulticol compares a multicol document holding grids with
// and without the memo. Multicol lays children out once to measure and once
// to emit under identical ambients, so the second measure hits the cache.
func BenchmarkGridMemoMulticol(b *testing.B) {
	const cssSrc = `
body{margin:0}
.mc{columns:3;column-gap:12pt;width:600pt}
.hold{break-inside:avoid}
.inner{display:grid;grid-template-columns:1fr 1fr;grid-template-rows:auto auto;gap:4pt}
`

	var doc strings.Builder

	doc.WriteString(`<html><body><div class="mc">`)

	for section := range 12 {
		fmt.Fprintf(&doc, `<div class="hold"><h2>Section %d market overview</h2>`, section)
		doc.WriteString(`<div class="inner"><div>price one two</div><div>change up down</div>`)
		doc.WriteString(`<div>volume high low</div><div>cap large small mid</div></div>`)
		doc.WriteString(`<p>closing summary paragraph with enough text to wrap across lines</p></div>`)
	}

	doc.WriteString(`</div></body></html>`)
	gridMemoBenchLayouts(b, doc.String(), cssSrc)
}

// gridMemoBenchLayouts runs cached/bypass sub-benchmarks over one document.
func gridMemoBenchLayouts(b *testing.B, doc, cssSrc string) {
	b.Helper()

	parsed, err := css.Parse(cssSrc)
	if err != nil {
		b.Fatalf("css.Parse: %v", err)
	}

	sheets := []*css.Stylesheet{parsed}

	bench := func(bypass bool) {
		for range b.N {
			root, err := html.Parse(doc)
			if err != nil {
				b.Fatalf("Parse: %v", err)
			}

			opts := Options{
				Width: testViewport, Height: 800, Sheets: sheets, Background: true,
			}
			opts.gridMemoBypass = bypass

			if _, err := Layout(root, opts); err != nil {
				b.Fatalf("Layout(bypass=%v): %v", bypass, err)
			}
		}
	}

	b.Run("cached", func(b *testing.B) {
		b.ReportAllocs()
		bench(false)
	})

	b.Run("bypass", func(b *testing.B) {
		b.ReportAllocs()
		bench(true)
	})
}
