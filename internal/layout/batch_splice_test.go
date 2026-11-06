package layout

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// batchTableHTML builds a generated table document: plain block cards the
// folded owners path covers, divs inside table cells the rows-track fallback
// covers, an unpatched absolute element that keeps the page
// ordering-relevant, and a relative z-index card the order fallback covers.
func batchTableHTML(rows int) string {
	var page strings.Builder

	page.WriteString(`<html><body>`)
	page.WriteString(`<div id="a" class="card">Alpha</div>`)
	page.WriteString(`<div id="b" class="card">Beta</div>`)
	page.WriteString(`<div id="abs" style="position:absolute;top:0;left:0;width:40px;height:10px">Abs</div>`)
	page.WriteString(`<div id="zin" class="card" style="position:relative;z-index:1">Zed</div>`)
	page.WriteString(`<table>`)

	for i := range rows {
		fmt.Fprintf(&page, `<tr><td><div id="row-%d" class="cell">Row %d</div></td></tr>`, i, i)
	}

	page.WriteString(`</table></body></html>`)

	return page.String()
}

// batchTableCSS sizes the cards and the cells, and gives every selected card
// the same paint-only restyle, so every patched element keeps its place.
func batchTableCSS() string {
	return relayoutFixtureCSS + `
		.cell { width: 200px; height: 20px; background-color: #eeeeee; }
		.cell.selected { background-color: #3366cc; }
	`
}

// editBatchAttr sets one attribute on one element of every edited root, so
// the batch and the sequential runs patch the same change.
func editBatchAttr(t *testing.T, roots []*html.Node, id, attr, value string) {
	t.Helper()

	for _, root := range roots {
		relayoutNodeByID(t, root, id).Attrs[attr] = value
	}
}

// layoutBatchPrev lays out one root for a batch run.
func layoutBatchPrev(t *testing.T, root *html.Node, opts Options) *Result {
	t.Helper()

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	return prev
}

// TestBatchSpliceMatchesSequentialSplices proves a batched multi-id run is
// identical to sequential splices: the operations (golden, identities
// included), the owners, and the paint order all agree, and the maintained
// owners and order equal a fresh full derivation from the patched result.
func TestBatchSpliceMatchesSequentialSplices(t *testing.T) {
	t.Parallel()

	const rows = 30

	rootBatch := mustParse(t, batchTableHTML(rows))
	rootSeq := mustParse(t, batchTableHTML(rows))
	opts := relayoutOptionsForCSS(t, batchTableCSS())

	batchPrev := layoutBatchPrev(t, rootBatch, opts)
	seqPrev := layoutBatchPrev(t, rootSeq, opts)

	ids := []string{"a", "b", "row-3", "row-17", "zin"}

	roots := []*html.Node{rootBatch, rootSeq}
	editBatchAttr(t, roots, "a", "class", relayoutSelectedClass)
	editBatchAttr(t, roots, "b", "class", relayoutSelectedClass)
	editBatchAttr(t, roots, "row-3", "class", "cell selected")
	editBatchAttr(t, roots, "row-17", "class", "cell selected")
	editBatchAttr(t, roots, "zin", "style", "position:relative;z-index:9")

	batchOwners := OperationOwners(batchPrev)
	batchOrder := PaintOrder(batchPrev.Ops)

	out, err := RelayoutSubtrees(t.Context(), batchPrev, ids, opts, batchOwners, batchOrder)
	if err != nil {
		t.Fatalf("RelayoutSubtrees: %v", err)
	}

	for _, id := range ids {
		node := relayoutNodeByID(t, rootSeq, id)

		if _, err := RelayoutSubtree(t.Context(), seqPrev, node, opts); err != nil {
			t.Fatalf("RelayoutSubtree %s: %v", id, err)
		}
	}

	seqOwners := OperationOwners(seqPrev)
	seqOrder := PaintOrder(seqPrev.Ops)

	if !reflect.DeepEqual(batchPrev.Ops, seqPrev.Ops) {
		t.Fatal("batched operations differ from the sequential splices")
	}

	if !reflect.DeepEqual(out.Owners, seqOwners) {
		t.Fatal("batched owners differ from the sequential splices")
	}

	if !reflect.DeepEqual(out.Order, seqOrder) {
		t.Fatal("batched paint order differs from the sequential splices")
	}

	if !reflect.DeepEqual(out.Owners, OperationOwners(batchPrev)) {
		t.Fatal("incremental owners differ from a full rebuild")
	}

	if !reflect.DeepEqual(out.Order, PaintOrder(batchPrev.Ops)) {
		t.Fatal("incremental paint order differs from a full rebuild")
	}

	if len(out.Windows) != len(ids) {
		t.Fatalf("windows = %d, want one per id (%d)", len(out.Windows), len(ids))
	}

	for _, window := range out.Windows {
		if window.NewCount <= 0 || !window.HasNewDamage {
			t.Fatalf("window %+v carries no damage, want the painted window", window)
		}
	}
}

// TestBatchSpliceIncrementalOwnersOrderEqualFullRebuild proves the
// incremental owners and order equal a full rebuild on an ordering-relevant
// page: a positioned sibling and a z-indexed sibling stay unpatched while
// neutral windows merge, and a z-index change falls back to the full sort.
func TestBatchSpliceIncrementalOwnersOrderEqualFullRebuild(t *testing.T) {
	t.Parallel()

	const doc = `<html><body>` +
		`<div id="first" class="card">Alpha</div>` +
		`<div id="second" class="card">Beta</div>` +
		`<div id="side" class="card" style="position:absolute;top:0;left:0">Side</div>` +
		`<div id="top" class="card" style="position:relative;z-index:4">Top</div>` +
		`</body></html>`

	root := mustParse(t, doc)
	opts := relayoutOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	relayoutNodeByID(t, root, "first").Attrs["class"] = relayoutSelectedClass
	relayoutNodeByID(t, root, "second").Attrs["class"] = relayoutSelectedClass

	out, err := RelayoutSubtrees(t.Context(), prev, []string{"first", "second"},
		opts, OperationOwners(prev), PaintOrder(prev.Ops))
	if err != nil {
		t.Fatalf("RelayoutSubtrees: %v", err)
	}

	if !reflect.DeepEqual(out.Owners, OperationOwners(prev)) {
		t.Fatal("incremental owners differ from a full rebuild with neutral windows")
	}

	if !reflect.DeepEqual(out.Order, PaintOrder(prev.Ops)) {
		t.Fatal("incremental paint order differs from a full rebuild with neutral windows")
	}

	relayoutNodeByID(t, root, "top").Attrs["style"] = "position:relative;z-index:9"

	out, err = RelayoutSubtrees(t.Context(), prev, []string{"top"},
		opts, out.Owners, out.Order)
	if err != nil {
		t.Fatalf("RelayoutSubtrees: %v", err)
	}

	if !reflect.DeepEqual(out.Owners, OperationOwners(prev)) {
		t.Fatal("incremental owners differ from a full rebuild after a z-index change")
	}

	if !reflect.DeepEqual(out.Order, PaintOrder(prev.Ops)) {
		t.Fatal("incremental paint order differs from a full rebuild after a z-index change")
	}
}

// TestBatchSpliceRefusalMatchesSingleError proves a refused middle id aborts
// the batch with the same error the single call returns, and the committed
// prefix matches the sequential prefix exactly.
func TestBatchSpliceRefusalMatchesSingleError(t *testing.T) {
	t.Parallel()

	const doc = `<html><body>` +
		`<div id="a" class="card">Alpha</div>` +
		`<div id="b" class="card">Beta</div>` +
		`<div id="c" class="card">Gamma</div>` +
		`</body></html>`

	rootBatch := mustParse(t, doc)
	rootSeq := mustParse(t, doc)
	opts := relayoutOptions(t)

	batchPrev := layoutBatchPrev(t, rootBatch, opts)
	seqPrev := layoutBatchPrev(t, rootSeq, opts)

	roots := []*html.Node{rootBatch, rootSeq}
	editBatchAttr(t, roots, "a", "class", relayoutSelectedClass)
	editBatchAttr(t, roots, "b", "class", "card wide")
	editBatchAttr(t, roots, "c", "class", relayoutSelectedClass)

	nodeA := relayoutNodeByID(t, rootSeq, "a")

	if _, err := RelayoutSubtree(t.Context(), seqPrev, nodeA, opts); err != nil {
		t.Fatalf("RelayoutSubtree a: %v", err)
	}

	nodeB := relayoutNodeByID(t, rootSeq, "b")
	_, singleErr := RelayoutSubtree(t.Context(), seqPrev, nodeB, opts)

	if singleErr == nil {
		t.Fatal("RelayoutSubtree succeeded on the wide card, want a refusal")
	}

	_, batchErr := RelayoutSubtrees(t.Context(), batchPrev,
		[]string{"a", "b", "c"}, opts, OperationOwners(batchPrev), PaintOrder(batchPrev.Ops))
	if batchErr == nil {
		t.Fatal("RelayoutSubtrees succeeded, want the refusal of the middle id")
	}

	// The batch wraps each splice failure the way the public element patch
	// wraps the engine error, so the driver reports the single-call error.
	if batchErr.Error() != relayoutSpliceError(singleErr).Error() {
		t.Fatalf("batch error = %v, want the wrapped single-call error %v", batchErr, singleErr)
	}

	if !reflect.DeepEqual(batchPrev.Ops, seqPrev.Ops) {
		t.Fatal("committed batch prefix differs from the sequential prefix")
	}

	if batchPrev.MaxContentX != seqPrev.MaxContentX ||
		batchPrev.HasFragmentLinks != seqPrev.HasFragmentLinks {
		t.Fatal("committed batch census differs from the sequential prefix")
	}
}

// TestBatchSpliceDamageIsOpWindowOnly proves damage comes from the spliced
// operation windows alone: a paint-only patch reports the old and the new
// window bounds inside the element box, and an element that paints nothing
// reports no damage.
func TestBatchSpliceDamageIsOpWindowOnly(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutFixtureHTML)
	opts := relayoutOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	relayoutNodeByID(t, root, "first").Attrs["class"] = relayoutSelectedClass

	out, err := RelayoutSubtrees(t.Context(), prev, []string{"first"},
		opts, OperationOwners(prev), PaintOrder(prev.Ops))
	if err != nil {
		t.Fatalf("RelayoutSubtrees: %v", err)
	}

	if len(out.Windows) != 1 {
		t.Fatalf("windows = %d, want 1", len(out.Windows))
	}

	window := out.Windows[0]

	if !window.HasOldDamage || !window.HasNewDamage {
		t.Fatalf("window %+v carries no damage, want both window bounds", window)
	}

	if window.OldDamage != window.NewDamage {
		t.Fatalf("old damage %v != new damage %v for a paint-only patch", window.OldDamage, window.NewDamage)
	}

	box := relayoutBoxByID(t, prev, "first")
	assertDamageInBox(t, window.NewDamage, box.x, box.y, box.w, box.height)

	emptyRoot := mustParse(t, relayoutEmptyHTML)

	emptyPrev, err := Layout(emptyRoot, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	emptyOut, err := RelayoutSubtrees(t.Context(), emptyPrev, []string{"empty"},
		opts, OperationOwners(emptyPrev), PaintOrder(emptyPrev.Ops))
	if err != nil {
		t.Fatalf("RelayoutSubtrees: %v", err)
	}

	emptyWindow := emptyOut.Windows[0]

	if emptyWindow.NewCount != 0 || emptyWindow.HasNewDamage {
		t.Fatalf("empty window %+v paints damage, want none", emptyWindow)
	}
}

// assertDamageInBox proves one op-window damage rect sits inside an element
// box, so damage never escapes the patched element.
func assertDamageInBox(t *testing.T, damage [4]float64, x, y, w, h float64) {
	t.Helper()

	const epsilon = 1e-6

	if damage[0] < x-epsilon || damage[1] < y-epsilon ||
		damage[0]+damage[2] > x+w+epsilon || damage[1]+damage[3] > y+h+epsilon {
		t.Fatalf("damage %v escapes the element box (%g, %g, %g, %g)",
			damage, x, y, w, h)
	}
}

// TestBatchSpliceCensusMatchesFullLayout proves the windowed census equals a
// full layout census, including the rescan paths: dropping the background
// that carried the page maximum and dropping the only fragment link.
func TestBatchSpliceCensusMatchesFullLayout(t *testing.T) {
	t.Parallel()

	const doc = `<html><body>` +
		`<div id="hero" style="width:700px;height:20px;background:#123456"></div>` +
		`<div id="golink"><a id="fraglink" href="#frag">frag</a></div>` +
		`<div id="plain" class="card">Plain</div>` +
		`</body></html>`

	root := mustParse(t, doc)
	opts := relayoutOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	if !prev.HasFragmentLinks {
		t.Fatal("fixture carries no fragment link")
	}

	heroMax := prev.MaxContentX

	relayoutNodeByID(t, root, "hero").Attrs["style"] = "width:700px;height:20px"
	relayoutNodeByID(t, root, "fraglink").Attrs["href"] = "https://example.com"

	out, err := RelayoutSubtrees(t.Context(), prev, []string{"hero", "golink"},
		opts, OperationOwners(prev), PaintOrder(prev.Ops))
	if err != nil {
		t.Fatalf("RelayoutSubtrees: %v", err)
	}

	full, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout (full relayout): %v", err)
	}

	if prev.MaxContentX != full.MaxContentX {
		t.Fatalf("census max = %g, want the full relayout %g", prev.MaxContentX, full.MaxContentX)
	}

	if prev.HasFragmentLinks != full.HasFragmentLinks {
		t.Fatalf("fragment links = %v, want the full relayout %v",
			prev.HasFragmentLinks, full.HasFragmentLinks)
	}

	if prev.HasFragmentLinks {
		t.Fatal("fragment link survived the patch")
	}

	if prev.MaxContentX >= heroMax {
		t.Fatalf("census max = %g, want less than the removed %g", prev.MaxContentX, heroMax)
	}

	if maxID := maxOpIDInWindow(prev.Ops); prev.nextOpID != maxID {
		t.Fatalf("high-water mark = %d, want the maximum operation identity %d", prev.nextOpID, maxID)
	}

	if clone := CloneResult(prev); clone.nextOpID != prev.nextOpID {
		t.Fatal("CloneResult dropped the high-water mark")
	}

	if !reflect.DeepEqual(out.Owners, OperationOwners(prev)) {
		t.Fatal("incremental owners differ from a full rebuild after the census paths")
	}
}

// TestSplicePaintOrderMatchesFullSort proves the incremental order equals
// the full sort: a neutral merge with ordering-relevant ops elsewhere, a
// non-neutral fallback, and a blend-group fallback.
func TestSplicePaintOrderMatchesFullSort(t *testing.T) {
	t.Parallel()

	neutral := []Op{
		{Kind: OpFillRect},
		{Kind: OpFillRect},
		{Kind: OpText},
		{Kind: OpText},
		{Kind: OpFillRect},
		{Kind: OpText, Positioned: true},
	}
	order := PaintOrder(neutral)

	changed := append([]Op(nil), neutral...)
	changed[2].R, changed[2].G, changed[2].B = 0.2, 0.3, 0.4
	changed[3].R, changed[3].G, changed[3].B = 0.5, 0.6, 0.7

	if got, want := splicePaintOrder(changed, order, 2, 2, 2, true), PaintOrder(changed); !reflect.DeepEqual(got, want) {
		t.Fatalf("neutral merge = %v, want the full sort %v", got, want)
	}

	moved := append([]Op(nil), neutral...)
	moved[2].ZIndex, moved[2].ZIndexSet = 5, true

	if got, want := splicePaintOrder(moved, order, 2, 2, 2, false), PaintOrder(moved); !reflect.DeepEqual(got, want) {
		t.Fatalf("non-neutral fallback = %v, want the full sort %v", got, want)
	}

	group := &BlendGroup{ID: 7, Isolate: true}
	grouped := []Op{
		{Kind: OpFillRect},
		{Kind: OpUnknown},
		{Kind: OpText},
		{Kind: OpUnknown},
		{Kind: OpText},
	}
	grouped[1].setGroupMark(group, groupMarkBegin)
	grouped[2].setBlendGroup(group)
	grouped[3].setGroupMark(group, groupMarkEnd)

	groupedOrder := PaintOrder(grouped)
	changedGrouped := append([]Op(nil), grouped...)
	changedGrouped[2].R = 0.9

	if got, want := splicePaintOrder(changedGrouped, groupedOrder, 2, 1, 1, false), PaintOrder(changedGrouped); !reflect.DeepEqual(got, want) {
		t.Fatalf("grouped fallback = %v, want the full sort %v", got, want)
	}

	if relevant, groupedCount := countOrderingRelevant(grouped); relevant == 0 || groupedCount == 0 {
		t.Fatalf("grouped counts = (%d, %d), want both positive", relevant, groupedCount)
	}

	if relevant, groupedCount := countOrderingRelevant(neutral); relevant != 4 || groupedCount != 0 {
		t.Fatalf("neutral counts = (%d, %d), want (4, 0)", relevant, groupedCount)
	}
}
