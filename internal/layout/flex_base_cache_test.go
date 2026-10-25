package layout

import (
	"context"
	"reflect"
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

// flexMinTestEngine builds an engine over root for direct auto-minimum memo
// tests, mirroring TestFlexColumnBaseHeightMemoized.
func flexMinTestEngine(t *testing.T, root *html.Node, opts Options) (*engine, *html.Node) {
	t.Helper()

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

	return eng, outer
}

// TestFlexColumnAutoMinMemoized pins the per-run memo for column flex
// auto-minimum floors: measuring the same item twice under the same sizes
// returns the identical floor without re-flowing the subtree, while a new
// containing height measures and stores again.
func TestFlexColumnAutoMinMemoized(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<div id="outer"><div class="cell"><span>alpha beta gamma</span></div></div>`)
	opts := Options{
		Width: testViewport, Height: 800, Background: true,
		Sheets: []*css.Stylesheet{sheet(t, `#outer{display:flex;flex-direction:column;width:400pt}.cell{overflow:visible}`)},
	}
	eng, outer := flexMinTestEngine(t, root, opts)

	var cell *html.Node

	outer.Walk(func(node *html.Node) {
		if cell == nil && node.Type == html.ElementNode && node.Attribute("class") == "cell" {
			cell = node
		}
	})

	if cell == nil {
		t.Fatal("missing .cell element")
	}

	first := eng.flexMinCrossMainSize(cell, 20, 300)
	if first <= 0 {
		t.Fatalf("auto-min = %v, want positive", first)
	}

	if cached := len(eng.flexMinH); cached == 0 {
		t.Fatal("first auto-min left no memoized entry")
	}

	second := eng.flexMinCrossMainSize(cell, 20, 300)
	if second != first {
		t.Fatalf("second auto-min = %v, want %v", second, first)
	}

	if cached := len(eng.flexMinH); cached != 1 {
		t.Fatalf("second auto-min grew the memo to %d entries, want 1", cached)
	}

	eng.flexMinCrossMainSize(cell, 20, 500)

	if cached := len(eng.flexMinH); cached != 2 {
		t.Fatalf("new main size left %d entries, want 2", cached)
	}
}

// TestFlexColumnAutoMinIdenticalOutput pins the memo's transparency: a
// nested column-flex document lays out byte-identical display lists with and
// without the auto-minimum cache.
func TestFlexColumnAutoMinIdenticalOutput(t *testing.T) {
	t.Parallel()

	const doc = `<html><body><div class="col"><div class="item"><span>alpha beta gamma delta</span></div>` +
		`<div class="item"><div class="inner"><span>epsilon zeta eta theta</span></div></div></div></body></html>`
	const cssSrc = `body{margin:0}` +
		`.col{display:flex;flex-direction:column;width:400pt;height:300pt}` +
		`.item{flex:1 1 auto;overflow:visible}` +
		`.inner{display:flex;flex-direction:column}`

	sheets := []*css.Stylesheet{sheet(t, cssSrc)}

	layoutWith := func(bypass bool) *Result {
		root := mustParse(t, doc)
		opts := Options{
			Width: testViewport, Height: 800, Sheets: sheets, Background: true,
		}
		opts.flexMinBypass = bypass

		res, err := Layout(root, opts)
		if err != nil {
			t.Fatalf("Layout(bypass=%v): %v", bypass, err)
		}

		return res
	}

	cached := layoutWith(false)
	plain := layoutWith(true)

	if !opsDeeplyEqual(gridMemoOpsComparable(cached.Ops), gridMemoOpsComparable(plain.Ops)) {
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

// opsDeeplyEqual reports whether two display lists are identical. Font
// pointers are run-shared caches whose internal shaping state differs between
// Layout calls, so they are cleared before comparing; opExtra payloads are
// compared by value instead of by pointer for the same reason. Geometry,
// text, and paint payloads stay fully compared.
func opsDeeplyEqual(a, b []Op) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		x, y := a[i], b[i]
		x.Font, y.Font = nil, nil

		var xe, ye any
		if x.opExtra != nil {
			xe = *x.opExtra
		}

		if y.opExtra != nil {
			ye = *y.opExtra
		}

		x.opExtra, y.opExtra = nil, nil

		if x != y || !reflect.DeepEqual(xe, ye) {
			return false
		}
	}

	return true
}
