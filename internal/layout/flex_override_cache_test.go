package layout

import (
	"reflect"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// flexOverrideTestEngine builds one engine over the #outer fixture shared by
// the flex memo tests, so the override helpers can run without a full layout.
func flexOverrideTestEngine(t *testing.T) (*engine, *html.Node) {
	t.Helper()

	root := mustParse(t, `<div id="outer"><div class="cell"><span>alpha</span></div></div>`)
	opts := Options{
		Width: testViewport, Height: 800, Background: true,
		Sheets: []*css.Stylesheet{sheet(t, `#outer{display:flex;width:400pt}`)},
	}

	return flexMinTestEngine(t, root, opts)
}

// TestFlexStyleOverrideCacheReusesEntries pins the per-run override cache:
// identical forced sizes return the same record, the record matches the old
// allocStyle-plus-force sequence, and each distinct force step keeps its own
// entry.
func TestFlexStyleOverrideCacheReusesEntries(t *testing.T) {
	eng, outer := flexOverrideTestEngine(t)
	base := eng.stylePtr(outer)

	if !eng.flexBaseStyleStable(outer, base) {
		t.Fatal("interned cascade style reported unstable")
	}

	if eng.flexBaseStyleStable(outer, eng.allocStyle(base)) {
		t.Fatal("transient style-chunk override reported stable")
	}

	row := eng.flexRowStyleOverride(base, true, true, 40, 120)
	rowAgain := eng.flexRowStyleOverride(base, true, true, 40, 120)

	if rowAgain != row {
		t.Fatal("second row override allocated a new record instead of hitting the cache")
	}

	if row == base {
		t.Fatal("cached row override aliases the interned base style")
	}

	if got := len(eng.flexStyleOverrides); got != 1 {
		t.Fatalf("row overrides cached = %d, want 1", got)
	}

	wantRow := eng.allocStyle(base)
	eng.forceFlexItemMainSize(wantRow, 120)
	eng.forceFlexItemCrossSize(wantRow, 40)

	if !reflect.DeepEqual(*row, *wantRow) {
		t.Fatal("cached row override differs from the uncached force sequence")
	}

	if eng.flexRowStyleOverride(base, true, false, 40, 120) == row {
		t.Fatal("forceStretch=false reused the stretched override")
	}

	column := eng.flexColumnStyleOverride(base, true, 120, 40, true)
	columnAgain := eng.flexColumnStyleOverride(base, true, 120, 40, true)

	if columnAgain != column {
		t.Fatal("second column override allocated a new record instead of hitting the cache")
	}

	if column == row {
		t.Fatal("column override reused a row key with the same forced lengths")
	}

	wantColumn := eng.allocStyle(base)
	eng.forceFlexItemCrossSize(wantColumn, 120)
	eng.forceFlexItemCrossWidth(wantColumn, 40)

	if !reflect.DeepEqual(*column, *wantColumn) {
		t.Fatal("cached column override differs from the uncached force sequence")
	}

	if eng.flexColumnStyleOverride(base, true, 120, 40, false) == column {
		t.Fatal("column override without a forced cross width reused the one with it")
	}

	if got := len(eng.flexStyleOverrides); got != 4 {
		t.Fatalf("overrides cached = %d, want 4", got)
	}

	// A base in a reusable style chunk must not become a key: resetStyles can
	// later put a different record at the same address.
	unstable := eng.allocStyle(base)
	before := len(eng.flexStyleOverrides)

	if eng.flexRowStyleOverride(unstable, false, true, 10, 20) ==
		eng.flexRowStyleOverride(unstable, false, true, 10, 20) {
		t.Fatal("unstable base style reused a cached override")
	}

	if got := len(eng.flexStyleOverrides); got != before {
		t.Fatalf("unstable base style grew the cache to %d entries", got)
	}
}

// TestFlexStyleOverrideCacheBounded pins the 256-entry bound: a document with
// more distinct forced sizes stops growing the map and falls back to a fresh
// copy without changing the computed override.
func TestFlexStyleOverrideCacheBounded(t *testing.T) {
	eng, outer := flexOverrideTestEngine(t)
	base := eng.stylePtr(outer)

	for i := range flexOverrideMaxEntries + 16 {
		eng.flexRowStyleOverride(base, true, false, 0, float64(i)+1)
	}

	if got := len(eng.flexStyleOverrides); got != flexOverrideMaxEntries {
		t.Fatalf("cache entries = %d, want bound %d", got, flexOverrideMaxEntries)
	}

	first := eng.flexRowStyleOverride(base, true, false, 0, 10000)
	second := eng.flexRowStyleOverride(base, true, false, 0, 10000)

	if first == second {
		t.Fatal("past the bound a repeated key reused a stored override")
	}

	if !reflect.DeepEqual(*first, *second) {
		t.Fatal("past the bound two allocations disagree")
	}

	if got := len(eng.flexStyleOverrides); got != flexOverrideMaxEntries {
		t.Fatalf("past the bound cache grew to %d", got)
	}
}

// flexOverrideTestCSS exercises repeated row items, column items with and
// without a measured cross width, and a vertical writing-mode column.
const flexOverrideTestCSS = `body{margin:0}` +
	`.row{display:flex;width:400pt;align-items:stretch;gap:4pt}` +
	`.cell{flex:1 1 auto;padding:2pt;border:0.5pt solid #000}` +
	`.col{display:flex;flex-direction:column;width:400pt;height:180pt}` +
	`.colitem{overflow:visible}` +
	`.colitem.wide{width:150pt}` +
	`.vert{display:flex;flex-direction:column;writing-mode:vertical-rl;width:400pt;height:120pt}` +
	`.vertitem{overflow:visible}` +
	`.aspect{width:120pt;aspect-ratio:2/1}` +
	`.aspectflex{display:flex;width:120pt;aspect-ratio:2/1}` +
	`.anonrow{display:flex;width:400pt;align-items:stretch}`

// flexOverrideDoc builds a document that drives every cached flex path:
// repeated row items, column items with and without a measured cross width,
// the logical-axis (vertical writing-mode) path, a nested flex container
// rebuilt by ancestor measure passes, and boxes that declare aspect-ratio.
func flexOverrideDoc(rows int) string {
	var doc strings.Builder

	doc.WriteString(`<html><body>`)
	doc.WriteString(`<div class="aspect"></div>`)
	doc.WriteString(`<div class="aspectflex"><div class="cell">ratio</div></div>`)

	for range 8 {
		doc.WriteString(`<div class="anonrow">direct text one and two</div>`)
	}

	for range rows {
		doc.WriteString(`<div class="row">`)

		for range 4 {
			doc.WriteString(`<div class="cell"><span>alpha</span></div>`)
		}

		doc.WriteString(`</div>`)
	}

	doc.WriteString(`<div class="col">`)
	doc.WriteString(`<div class="colitem"><div class="row"><div class="cell">nested</div>` +
		`<div class="cell">nested</div></div></div>`)
	doc.WriteString(`<div class="colitem"><div class="anonrow">anon one anon two</div></div>`)
	doc.WriteString(`<div class="colitem wide">beta beta</div>`)
	doc.WriteString(`<div class="colitem">gamma</div>`)
	doc.WriteString(`</div>`)
	doc.WriteString(`<div class="vert">`)
	doc.WriteString(`<div class="vertitem">delta</div>`)
	doc.WriteString(`<div class="vertitem">epsilon epsilon</div>`)
	doc.WriteString(`</div>`)
	doc.WriteString(`</body></html>`)

	return doc.String()
}

// layoutFlexOverrideDoc lays out doc with the override cache enabled or
// bypassed. The caller must not run the two calls concurrently.
func layoutFlexOverrideDoc(t *testing.T, doc string, bypass bool) *Result {
	t.Helper()

	flexOverrideBypass.Store(bypass)
	t.Cleanup(func() { flexOverrideBypass.Store(false) })

	res, err := Layout(mustParse(t, doc), Options{
		Width: testViewport, Height: 800, Background: true,
		Sheets: []*css.Stylesheet{sheet(t, flexOverrideTestCSS)},
	})
	if err != nil {
		t.Fatalf("Layout(bypass=%v): %v", bypass, err)
	}

	return res
}

// TestFlexStyleOverrideCacheIdenticalOutput pins the cache's transparency: a
// document covering every replaced copy site lays out byte-identical display
// lists with and without the override cache.
func TestFlexStyleOverrideCacheIdenticalOutput(t *testing.T) {
	doc := flexOverrideDoc(120)

	cached := layoutFlexOverrideDoc(t, doc, false)
	plain := layoutFlexOverrideDoc(t, doc, true)

	if len(cached.Ops) == 0 {
		t.Fatal("cached layout produced no ops")
	}

	if !opsDeeplyEqual(cached.Ops, plain.Ops) {
		t.Fatalf("cached ops differ from uncached (%d vs %d ops)", len(cached.Ops), len(plain.Ops))
	}

	if cached.Height != plain.Height || cached.MaxContentX != plain.MaxContentX ||
		cached.HasFragmentLinks != plain.HasFragmentLinks {
		t.Fatalf(
			"cached result differs: height %.3f/%.3f maxX %.3f/%.3f frag %v/%v",
			cached.Height, plain.Height, cached.MaxContentX, plain.MaxContentX,
			cached.HasFragmentLinks, plain.HasFragmentLinks,
		)
	}
}

// TestFlexStyleOverrideCacheHitsRepeatedRows pins the win the cache exists
// for: identical invoice-style rows share one override record, so repeated
// items hit the cache far more often than they allocate.
func TestFlexStyleOverrideCacheHitsRepeatedRows(t *testing.T) {
	flexOverrideHits.Store(0)
	flexOverrideMisses.Store(0)

	layoutFlexOverrideDoc(t, flexOverrideDoc(200), false)

	hits := flexOverrideHits.Load()
	misses := flexOverrideMisses.Load()

	if hits < 100 {
		t.Fatalf("cache hits = %d, want repeated rows to hit the cache", hits)
	}

	if misses >= hits {
		t.Fatalf("cache hits %d do not dominate misses %d", hits, misses)
	}
}

// TestFlexAspectRatioHeightFromWidth keeps the guard on the aspect-ratio
// height fast path honest: a ratio box still derives its height from its
// used width, for both block and flex containers, and a ratio-free box keeps
// its ordinary auto height.
func TestFlexAspectRatioHeightFromWidth(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="block" style="width:200px;aspect-ratio:2/1"></div>`+
		`<div id="auto" style="width:200px"></div>`+
		`<div id="flex" style="display:flex;width:200px;aspect-ratio:2/1"><div>x</div></div>`+
		`</body></html>`)

	behaviorBoxModelCheckUsedSize(t, res, "block", 200, 100)
	behaviorBoxModelCheckUsedSize(t, res, "auto", 200, 0)
	behaviorBoxModelCheckUsedSize(t, res, "flex", 200, 100)
}
