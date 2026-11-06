package layout

// Container scratch golden and memo tests.
//
// These tests pin the pooled-scratch and memo migration in flex.go,
// grid_tracks.go, grid_placement.go, and multicol.go: generated flex, grid,
// table, and multicol documents lay out byte-identical display lists with and
// without the memos (golden op counts and key rects), while the per-run
// memos prove they actually hit.
//
// None of the tests in this file use t.Parallel: the bypass switches are
// process-wide atomics shared with parallel tests in other files. Atomic
// access keeps the sharing race-clean; staying off the parallel lane keeps
// the bypass windows deterministic.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

const containerScratchFlexDoc = `<html><body><div class="row">` +
	`<div class="chip"><span>alpha one</span></div>` +
	`<div class="chip"><span>beta two three</span></div>` +
	`<div class="chip"><span>gamma</span></div>` +
	`<div class="chip"><span>delta four five six</span></div>` +
	`<div class="chip"><span>epsilon seven</span></div>` +
	`<div class="chip"><span>zeta eight nine ten</span></div>` +
	`<div class="chip"><span>eta eleven</span></div>` +
	`<div class="chip"><span>theta twelve thirteen</span></div>` +
	`<div class="chip"><span>iota</span></div>` +
	`<div class="chip"><span>kappa fourteen fifteen</span></div>` +
	`<div class="chip"><span>lambda sixteen</span></div>` +
	`<div class="chip"><span>mu seventeen eighteen nineteen</span></div>` +
	`</div></body></html>`

const containerScratchFlexCSS = `body{margin:0}` +
	`.row{display:flex;flex-wrap:wrap;gap:4pt;width:460pt}` +
	`.chip{background:#eee}`

const containerScratchGridDoc = `<html><body>
<div class="outer">
<div class="card"><h2>Alpha market</h2><div class="inner"><div>price one</div><div>change up</div>
<div>volume high</div><div>cap large</div></div><p>card one footer text here</p></div>
<div class="card"><h2>Beta market</h2><div class="inner"><div>price two</div><div>change down</div>
<div>volume low</div><div>cap small</div></div><p>card two footer text here</p></div>
<div class="card"><h2>Gamma market</h2><div class="inner"><div>price three</div><div>change flat</div>
<div>volume mid</div><div>cap mid</div></div><p>card three footer text here</p></div>
<div class="card special"><h2>Delta market</h2><p>spanning card body text</p></div>
</div></body></html>`

const containerScratchGridCSS = `body{margin:0}` +
	`.outer{display:grid;grid-template-columns:repeat(3,1fr);grid-template-rows:auto auto;gap:6pt;width:480pt}` +
	`.card{background:#eee}` +
	`.special{grid-column:span 2}` +
	`.inner{display:grid;grid-template-columns:1fr 1fr;grid-template-rows:auto auto;gap:4pt}`

const containerScratchMulticolDoc = `<html><body><div class="col"><div class="row">` +
	`<div class="chip"><span>alpha one two</span></div>` +
	`<div class="chip"><span>beta three</span></div>` +
	`<div class="chip"><span>gamma four five</span></div>` +
	`<div class="chip"><span>delta six</span></div>` +
	`</div><div class="mc">` +
	`<div class="hold"><h2>Section one</h2><p>closing summary paragraph with enough text to wrap</p></div>` +
	`<div class="hold"><h2>Section two</h2><p>another summary paragraph with enough text to wrap</p></div>` +
	`<div class="hold"><h2>Section three</h2><p>third summary paragraph with enough text to wrap</p></div>` +
	`<div class="hold"><h2>Section four</h2><p>fourth summary paragraph with enough text to wrap</p></div>` +
	`<div class="hold"><h2>Section five</h2><p>fifth summary paragraph with enough text to wrap</p></div>` +
	`<div class="hold"><h2>Section six</h2><p>sixth summary paragraph with enough text to wrap</p></div>` +
	`</div></div></body></html>`

const containerScratchMulticolCSS = `body{margin:0}` +
	`.col{display:flex;flex-direction:column;width:460pt}` +
	`.row{display:flex;flex-wrap:wrap;gap:4pt}` +
	`.chip{background:#eee}` +
	`.mc{columns:3;column-gap:12pt}` +
	`.hold{break-inside:avoid}`

const containerScratchTableDoc = `<html><body>
<table>
<tr><td>On-time delivery</td><td>96.4 %</td><td>above target</td><td>note one</td></tr>
<tr><td>First-pass yield</td><td colspan="2">98.1 % combined</td><td>note two</td></tr>
<tr><td>Scrap rate</td><td>1.2 %</td><td>below target</td><td>note three</td></tr>
<tr><td>Rework hours</td><td>34.0</td><td>flat</td><td>note four</td></tr>
</table></body></html>`

const containerScratchTableCSS = `body{margin:0}` +
	`table{width:460pt;border-collapse:collapse}` +
	`td{border:1px solid #000;padding:4px;font-size:10pt}`

// setContainerBypasses flips every container-scratch memo at once. The
// caller restores the default with defer.
func setContainerBypasses(t *testing.T, bypass bool) {
	t.Helper()

	flexRowMemoBypass.Store(bypass)
	trackIntrBypass.Store(bypass)
	multicolMemoBypass.Store(bypass)

	t.Cleanup(func() {
		flexRowMemoBypass.Store(false)
		trackIntrBypass.Store(false)
		multicolMemoBypass.Store(false)
	})
}

// layoutContainerDoc lays out a generated document with the container
// memos enabled or fully bypassed.
func layoutContainerDoc(t *testing.T, doc, cssSrc string, bypass bool) *Result {
	t.Helper()

	sheets := []*css.Stylesheet{sheet(t, cssSrc)}
	root := mustParse(t, doc)
	opts := Options{
		Width: testViewport, Height: 800, Sheets: sheets, Background: true,
	}
	opts.gridMemoBypass = bypass
	opts.flexMinBypass = bypass

	res, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout(bypass=%v): %v", bypass, err)
	}

	return res
}

// checkContainerIdentical lays out one generated document both ways and
// requires identical op counts, identical ops, and identical key rects.
func checkContainerIdentical(t *testing.T, name, doc, cssSrc string) {
	t.Helper()

	setContainerBypasses(t, false)
	cached := layoutContainerDoc(t, doc, cssSrc, false)

	setContainerBypasses(t, true)
	plain := layoutContainerDoc(t, doc, cssSrc, true)

	setContainerBypasses(t, false)

	if len(cached.Ops) == 0 {
		t.Fatalf("%s: cached layout produced no ops", name)
	}

	if len(cached.Ops) != len(plain.Ops) {
		t.Fatalf("%s: op count changed %d -> %d", name, len(plain.Ops), len(cached.Ops))
	}

	t.Logf("%s: golden op count %d", name, len(cached.Ops))

	if !reflect.DeepEqual(gridMemoOpsComparable(cached.Ops), gridMemoOpsComparable(plain.Ops)) {
		t.Fatalf("%s: memoized ops differ", name)
	}

	if cached.Height != plain.Height || cached.MaxContentX != plain.MaxContentX ||
		cached.HasFragmentLinks != plain.HasFragmentLinks {
		t.Fatalf(
			"%s: result differs: height %.3f/%.3f maxX %.3f/%.3f frag %v/%v",
			name, cached.Height, plain.Height, cached.MaxContentX, plain.MaxContentX,
			cached.HasFragmentLinks, plain.HasFragmentLinks,
		)
	}
}

// TestContainerScratchIdenticalOutput pins the whole migration: flex row
// memo, grid bitset plus intrinsics memo plus pooled defs, and the multicol
// height memo change no layout result on generated documents. The table doc
// exercises the untouched table path as a regression guard.
func TestMemoContainerScratchIdenticalOutput(t *testing.T) {
	checkContainerIdentical(t, "flex", containerScratchFlexDoc, containerScratchFlexCSS)
	checkContainerIdentical(t, "grid", containerScratchGridDoc, containerScratchGridCSS)
	checkContainerIdentical(t, "multicol", containerScratchMulticolDoc, containerScratchMulticolCSS)
	checkContainerIdentical(t, "table", containerScratchTableDoc, containerScratchTableCSS)
}

// TestFlexRowBaseMemoHit pins the row-path base-width memo: measuring the
// same row items twice returns identical widths without growing the shared
// per-run map, and the hit counter moves.
func TestFlexRowBaseMemoHit(t *testing.T) {
	root := mustParse(t, `<div id="outer"><div class="chip"><span>alpha beta</span></div>`+
		`<div class="chip"><span>gamma delta epsilon</span></div></div>`)
	opts := Options{
		Width: testViewport, Height: 800, Background: true,
		Sheets: []*css.Stylesheet{
			sheet(t, `#outer{display:flex;flex-wrap:wrap;gap:4pt;width:460pt}`),
		},
	}
	eng := gridMemoEngine(t, root, opts)
	outer := findOuterByID(t, root)
	kids := eng.flexChildren(outer, eng.stylePtr(outer))

	if len(kids) != 2 {
		t.Fatalf("flex children = %d, want 2", len(kids))
	}

	before := len(eng.flexBaseH)
	hits := flexRowBaseHits.Load()
	first := eng.flexRowItems(kids, 460, -1)

	if len(eng.flexBaseH) == before {
		t.Fatal("first flexRowItems left no memoized base widths")
	}

	second := eng.flexRowItems(kids, 460, -1)

	if len(eng.flexBaseH) != before+len(kids) {
		t.Fatalf("second flexRowItems grew the memo to %d entries", len(eng.flexBaseH)-before)
	}

	if flexRowBaseHits.Load() <= hits {
		t.Fatal("second flexRowItems recorded no row base memo hit")
	}

	for idx := range first {
		if first[idx].baseW != second[idx].baseW {
			t.Fatalf("item %d baseW changed %v -> %v", idx, first[idx].baseW, second[idx].baseW)
		}

		if first[idx].baseW <= 0 {
			t.Fatalf("item %d baseW = %v, want positive", idx, first[idx].baseW)
		}
	}
}

// TestFlexRowMinMemoHit pins the row-path auto-minimum memo the same way:
// one floor per item and size, with hits on repeat.
func TestFlexRowMinMemoHit(t *testing.T) {
	root := mustParse(t, `<div id="outer"><div class="chip"><span>alpha beta gamma delta</span></div></div>`)
	opts := Options{
		Width: testViewport, Height: 800, Background: true,
		Sheets: []*css.Stylesheet{
			sheet(t, `#outer{display:flex;flex-wrap:wrap;width:460pt}.chip{overflow:visible}`),
		},
	}
	eng := gridMemoEngine(t, root, opts)
	outer := findOuterByID(t, root)
	kids := eng.flexChildren(outer, eng.stylePtr(outer))
	items := eng.flexRowItems(kids, 460, -1)

	if len(items) != 1 {
		t.Fatalf("flex items = %d, want 1", len(items))
	}

	// flexRowItems already ran the hypothetical clamp, so exactly one floor
	// entry exists; repeats at the same size must hit without growing the map.
	base := len(eng.flexMinH)
	if base != 1 {
		t.Fatalf("flexRowItems left %d memo entries, want 1", base)
	}

	hits := flexRowMinHits.Load()
	first := eng.flexMinMainSize(items[0], 460)

	if first <= 0 {
		t.Fatalf("auto-min = %v, want positive", first)
	}

	second := eng.flexMinMainSize(items[0], 460)
	if second != first {
		t.Fatalf("second auto-min = %v, want %v", second, first)
	}

	if len(eng.flexMinH) != base {
		t.Fatalf("repeat auto-min grew the memo to %d entries, want %d", len(eng.flexMinH), base)
	}

	if flexRowMinHits.Load() <= hits {
		t.Fatal("repeat flexMinMainSize recorded no row min memo hit")
	}

	eng.flexMinMainSize(items[0], 300)
	if len(eng.flexMinH) != base+1 {
		t.Fatalf("new main size left %d entries, want %d", len(eng.flexMinH), base+1)
	}
}

// TestTrackKidMemoHit pins the grid intrinsics memo: one contribution per
// child and axis, with hits on repeat.
func TestMemoTrackKidHit(t *testing.T) {
	root := mustParse(t, `<div id="g"><div class="cell"><span>alpha beta gamma</span></div></div>`)
	opts := Options{
		Width: testViewport, Height: 800, Background: true,
		Sheets: []*css.Stylesheet{sheet(t, `#g{width:400pt}`)},
	}
	eng := gridMemoEngine(t, root, opts)
	cell := gridMemoCellNode(t, root)
	hits := trackIntrHits.Load()

	firstMin, firstMax := eng.trackKidContribution(cell, true)
	if firstMin <= 0 || firstMax <= 0 {
		t.Fatalf("contribution = (%v, %v), want positive", firstMin, firstMax)
	}

	secondMin, secondMax := eng.trackKidContribution(cell, true)
	if secondMin != firstMin || secondMax != firstMax {
		t.Fatalf(
			"second contribution = (%v, %v), want (%v, %v)",
			secondMin, secondMax, firstMin, firstMax,
		)
	}

	if trackIntrHits.Load() <= hits {
		t.Fatal("second contribution recorded no intrinsics memo hit")
	}

	thirdMin, _ := eng.trackKidContribution(cell, false)
	if thirdMin <= 0 {
		t.Fatalf("row-axis contribution = %v, want positive", thirdMin)
	}
}

// TestGridOccupationBitset pins the bitset placement: spans mark and block,
// including a 70-column grid that crosses the first 64-bit word.
func TestGridOccupationBitset(t *testing.T) {
	occ := newGridOccupation(70)
	defer occ.release()

	if !occ.freeAt(0, 0, 2, 70) {
		t.Fatal("empty 2x70 span reported occupied")
	}

	occ.mark(0, 0, 2, 70)

	if occ.freeAt(0, 69, 1, 1) {
		t.Fatal("occupied tail cell reported free")
	}

	if occ.freeAt(1, 0, 1, 1) {
		t.Fatal("occupied second row reported free")
	}

	if !occ.freeAt(2, 0, 1, 70) {
		t.Fatal("free third row reported occupied")
	}

	if occ.freeAt(0, 70, 1, 1) {
		t.Fatal("out-of-range column reported free")
	}
}

// TestMulticolHeightMemoHit pins the multicol child-height memo: measuring
// the same children twice at one column width returns identical heights with
// hits on the second pass.
func TestMulticolHeightMemoHit(t *testing.T) {
	root := mustParse(t, `<div id="outer"><div><span>alpha beta gamma delta epsilon</span></div>`+
		`<div><span>zeta eta theta iota kappa lambda</span></div></div>`)
	opts := Options{
		Width: testViewport, Height: 800, Background: true,
		Sheets: []*css.Stylesheet{sheet(t, `#outer{columns:3;column-gap:12pt;width:460pt}`)},
	}
	eng := gridMemoEngine(t, root, opts)
	outer := findOuterByID(t, root)
	kids := multicolKids(outer, eng)
	hits := multicolMemoHits.Load()

	first := eng.measureMulticolItems(kids, 140)
	if len(first) != 2 {
		t.Fatalf("multicol items = %d, want 2", len(first))
	}

	second := eng.measureMulticolItems(kids, 140)
	for idx := range first {
		if first[idx].h != second[idx].h {
			t.Fatalf("item %d height changed %v -> %v", idx, first[idx].h, second[idx].h)
		}

		if first[idx].h <= 0 {
			t.Fatalf("item %d height = %v, want positive", idx, first[idx].h)
		}
	}

	if multicolMemoHits.Load() <= hits {
		t.Fatal("second measureMulticolItems recorded no multicol memo hit")
	}
}

// TestContainerMemoCountersFullLayout proves every new memo fires on a real
// document: a column flex holds a row flex, a nested grid, and a multicol,
// so one Layout run hits the row base, track intrinsics, and multicol
// height memos.
func TestMemoContainerCountersFullLayout(t *testing.T) {
	var doc strings.Builder

	doc.WriteString(`<html><body><div class="col"><div class="row">`)
	for _, text := range []string{"alpha one", "beta two three", "gamma", "delta four five"} {
		doc.WriteString(`<div class="chip"><span>` + text + `</span></div>`)
	}

	doc.WriteString(`</div><div class="mc">`)
	for range 6 {
		doc.WriteString(`<div class="hold"><h2>Section market overview</h2>`)
		doc.WriteString(`<p>closing summary paragraph with enough text to wrap across lines</p></div>`)
	}

	doc.WriteString(`</div><div class="outer">`)
	doc.WriteString(`<div class="card"><div class="inner"><div>price one</div><div>change up</div>`)
	doc.WriteString(`<div>volume high</div><div>cap large</div></div></div>`)
	doc.WriteString(`<div class="card"><div class="inner"><div>price two</div><div>change down</div>`)
	doc.WriteString(`<div>volume low</div><div>cap small</div></div></div>`)
	doc.WriteString(`</div></div></body></html>`)

	const cssSrc = `body{margin:0}` +
		`.col{display:flex;flex-direction:column;width:460pt}` +
		`.row{display:flex;flex-wrap:wrap;gap:4pt}` +
		`.mc{columns:3;column-gap:12pt}` +
		`.hold{break-inside:avoid}` +
		`.outer{display:grid;grid-template-columns:repeat(2,1fr);grid-template-rows:auto;gap:6pt}` +
		`.inner{display:grid;grid-template-columns:1fr 1fr;grid-template-rows:auto auto;gap:4pt}`

	flexBase := flexRowBaseHits.Load()
	flexMin := flexRowMinHits.Load()
	intr := trackIntrHits.Load()
	multi := multicolMemoHits.Load()

	setContainerBypasses(t, false)

	if res := layoutContainerDoc(t, doc.String(), cssSrc, false); len(res.Ops) == 0 {
		t.Fatal("combined layout produced no ops")
	}

	if flexRowBaseHits.Load() <= flexBase {
		t.Error("combined layout recorded no flex row base memo hit")
	}

	if flexRowMinHits.Load() <= flexMin {
		t.Error("combined layout recorded no flex row min memo hit")
	}

	if trackIntrHits.Load() <= intr {
		t.Error("combined layout recorded no track intrinsics memo hit")
	}

	if multicolMemoHits.Load() <= multi {
		t.Error("combined layout recorded no multicol height memo hit")
	}
}
