package layout

import (
	"reflect"
	"strings"
	"testing"
)

// paintOnlyPadCSS gives a fixed border-box container a padding class. The
// container keeps its border box, but its child moves and narrows, so the
// geometry check refuses the paint-only splice.
const paintOnlyPadCSS = relayoutFixtureCSS + `
	.padbox { width: 200px; height: 100px; box-sizing: border-box; padding: 0; background-color: #eeeeee; }
	.padbox.padded { padding: 20px; }
`

// paintOnlyPadHTML holds a padded container beside a plain card, so a batch
// can mix a paint-only id with a geometry-changing one.
const paintOnlyPadHTML = `<html><body>` +
	`<div id="first" class="card">Alpha</div>` +
	`<div id="pad" class="padbox"><div id="inner">Inner</div></div>` +
	`</body></html>`

// TestBatchPaintOnlyClassMatchesFullRelayout proves a class-only batch takes
// the paint-only path: the operation window is overwritten, the flat box list
// is not rebuilt, and the operations, owners, order, and box text match a full
// relayout of the same edited document.
func TestBatchPaintOnlyClassMatchesFullRelayout(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutFixtureHTML)
	opts := relayoutOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	boxesBacking := &prev.boxes[0]
	boxCount := len(prev.boxes)

	ids := []string{"first", "second"}

	for _, id := range ids {
		relayoutNodeByID(t, root, id).Attrs["class"] = relayoutSelectedClass
	}

	out, err := RelayoutSubtrees(t.Context(), prev, ids, opts,
		OperationOwners(prev), PaintOrder(prev.Ops))
	if err != nil {
		t.Fatalf("RelayoutSubtrees: %v", err)
	}

	if !out.PaintOnly {
		t.Fatal("a class-only batch did not take the paint-only path")
	}

	for _, window := range out.Windows {
		if !window.PaintOnly || window.OldCount != window.NewCount {
			t.Fatalf("window %+v is not a same-length paint-only window", window)
		}
	}

	if &prev.boxes[0] != boxesBacking || len(prev.boxes) != boxCount {
		t.Fatal("a paint-only batch rebuilt the flat box list")
	}

	if len(out.Texts) != len(ids) {
		t.Fatalf("text refresh entries = %d, want %d", len(out.Texts), len(ids))
	}

	for _, id := range ids {
		boxNode := relayoutBoxByID(t, prev, id)
		fill, ok := relayoutFirstFill(prev.Ops[boxNode.opStart : boxNode.opEnd+1])
		if !ok {
			t.Fatalf("patched box %s carries no fill", id)
		}

		if fill != selectedBlue {
			t.Fatalf("patched box %s fill = %v, want the selected blue", id, fill)
		}
	}

	full, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout (full relayout): %v", err)
	}

	if !reflect.DeepEqual(relayoutStripIDs(prev.Ops), relayoutStripIDs(full.Ops)) {
		t.Fatal("paint-only operations differ from a full relayout")
	}

	if !reflect.DeepEqual(out.Owners, OperationOwners(full)) {
		t.Fatal("paint-only owners differ from a full relayout")
	}

	if !reflect.DeepEqual(out.Order, PaintOrder(full.Ops)) {
		t.Fatal("paint-only paint order differs from a full relayout")
	}
}

// TestBatchPaintOnlyRefreshesBoxText proves the text refresh a paint-only batch
// reports: the edited element's text and every element ancestor's text come
// from the committed tree, nearest first.
func TestBatchPaintOnlyRefreshesBoxText(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutFixtureHTML)
	opts := relayoutOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	relayoutNodeByID(t, root, "first").Children[0].Text = "Omega"

	out, err := RelayoutSubtrees(t.Context(), prev, []string{"first"}, opts,
		OperationOwners(prev), PaintOrder(prev.Ops))
	if err != nil {
		t.Fatalf("RelayoutSubtrees: %v", err)
	}

	if !out.PaintOnly {
		t.Fatal("a same-size text edit did not take the paint-only path")
	}

	if len(out.Texts) != 1 || out.Texts[0].ID != "first" {
		t.Fatalf("text refresh entries = %+v, want one for first", out.Texts)
	}

	chain := out.Texts[0].Texts
	if len(chain) < 2 || chain[0] != "Omega" {
		t.Fatalf("text chain = %v, want the element text then its ancestors", chain)
	}

	for _, ancestorText := range chain[1:] {
		if !strings.Contains(ancestorText, "Omega") {
			t.Fatalf("ancestor text %q does not carry the edited text", ancestorText)
		}
	}

	boxNode := relayoutBoxByID(t, prev, "first")
	if got := relayoutPatchText(prev.Ops[boxNode.opStart : boxNode.opEnd+1]); got != "Omega" {
		t.Fatalf("patched text = %q, want Omega", got)
	}
}

// TestBatchPaintOnlyPaddingFallsBack proves a padding change that moves a child
// fails the geometry check and runs the full splice: the result equals a full
// relayout and the batch reports no paint-only window.
func TestBatchPaintOnlyPaddingFallsBack(t *testing.T) {
	t.Parallel()

	root := mustParse(t, paintOnlyPadHTML)
	opts := relayoutOptionsForCSS(t, paintOnlyPadCSS)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	boxesBacking := &prev.boxes[0]

	relayoutNodeByID(t, root, "pad").Attrs["class"] = "padbox padded"

	out, err := RelayoutSubtrees(t.Context(), prev, []string{"pad"}, opts,
		OperationOwners(prev), PaintOrder(prev.Ops))
	if err != nil {
		t.Fatalf("RelayoutSubtrees: %v", err)
	}

	if out.PaintOnly || out.Windows[0].PaintOnly {
		t.Fatal("a child-moving padding change took the paint-only path")
	}

	if &prev.boxes[0] == boxesBacking {
		t.Fatal("the full splice did not rebuild the flat box list")
	}

	full, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout (full relayout): %v", err)
	}

	if !reflect.DeepEqual(relayoutStripIDs(prev.Ops), relayoutStripIDs(full.Ops)) {
		t.Fatal("full-splice operations differ from a full relayout")
	}

	if !reflect.DeepEqual(out.Owners, OperationOwners(full)) {
		t.Fatal("full-splice owners differ from a full relayout")
	}

	if !reflect.DeepEqual(out.Order, PaintOrder(full.Ops)) {
		t.Fatal("full-splice paint order differs from a full relayout")
	}
}

// TestBatchPaintOnlyMixedFallsBackAsWhole proves one geometry-changing id
// clears the batch flag: the first id commits its paint-only window, the
// second runs the full splice, and the batch result matches a full relayout.
func TestBatchPaintOnlyMixedFallsBackAsWhole(t *testing.T) {
	t.Parallel()

	root := mustParse(t, paintOnlyPadHTML)
	opts := relayoutOptionsForCSS(t, paintOnlyPadCSS)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	relayoutNodeByID(t, root, "first").Attrs["class"] = relayoutSelectedClass
	relayoutNodeByID(t, root, "pad").Attrs["class"] = "padbox padded"

	ids := []string{"first", "pad"}

	out, err := RelayoutSubtrees(t.Context(), prev, ids, opts,
		OperationOwners(prev), PaintOrder(prev.Ops))
	if err != nil {
		t.Fatalf("RelayoutSubtrees: %v", err)
	}

	if out.PaintOnly {
		t.Fatal("a mixed batch kept the paint-only flag")
	}

	if len(out.Windows) != len(ids) {
		t.Fatalf("windows = %d, want %d", len(out.Windows), len(ids))
	}

	if !out.Windows[0].PaintOnly || out.Windows[1].PaintOnly {
		t.Fatalf("windows = %+v, want the first paint-only and the second full", out.Windows)
	}

	if len(out.Texts) != 0 {
		t.Fatalf("mixed batch carries %d text refreshes, want none", len(out.Texts))
	}

	full, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout (full relayout): %v", err)
	}

	if !reflect.DeepEqual(relayoutStripIDs(prev.Ops), relayoutStripIDs(full.Ops)) {
		t.Fatal("mixed-batch operations differ from a full relayout")
	}

	if !reflect.DeepEqual(out.Owners, OperationOwners(full)) {
		t.Fatal("mixed-batch owners differ from a full relayout")
	}

	if !reflect.DeepEqual(out.Order, PaintOrder(full.Ops)) {
		t.Fatal("mixed-batch paint order differs from a full relayout")
	}
}

const sameBoxesTooltipCSS = `
	body { margin: 0; }
	.cell { position: relative; width: 120px; height: 40px; background: #cccccc; }
	.cell .tip {
		display: none; position: absolute; top: 0; left: 0;
		width: 80px; height: 20px; background: #111111; color: #ffffff;
	}
	.cell.open .tip { display: block; }
`

const sameBoxesTooltipHTML = `<html><body>` +
	`<div id="a" class="cell open"><span class="tip">Alpha tip</span></div>` +
	`<div id="b" class="cell"><span class="tip">Beta tip</span></div>` +
	`<div id="c" class="cell"><span class="tip">Gamma tip</span></div>` +
	`</body></html>`

// TestBatchSameBoxesCountChangeMatchesFullLayout proves a batch that toggles
// display:none <-> display:block children keeps SameBoxes true and produces
// exact parity with a full layout in both forward and reverse batch order.
func TestBatchSameBoxesCountChangeMatchesFullLayout(t *testing.T) {
	t.Parallel()

	for _, order := range [][]string{{"a", "b"}, {"b", "a"}} {
		root := mustParse(t, sameBoxesTooltipHTML)
		opts := relayoutOptionsForCSS(t, sameBoxesTooltipCSS)

		prev, err := Layout(root, opts)
		if err != nil {
			t.Fatalf("Layout: %v", err)
		}

		relayoutNodeByID(t, root, "a").Attrs["class"] = "cell"
		relayoutNodeByID(t, root, "b").Attrs["class"] = "cell open"

		clone := CloneResult(prev)
		out, err := RelayoutSubtrees(t.Context(), clone, order, opts,
			OperationOwners(prev), PaintOrder(prev.Ops))
		if err != nil {
			t.Fatalf("RelayoutSubtrees(%v): %v", order, err)
		}

		if out.PaintOnly {
			t.Fatalf("order %v: expected PaintOnly=false when child box appears/disappears", order)
		}

		if !out.SameBoxes {
			t.Fatalf("order %v: expected SameBoxes=true for tooltip display toggle", order)
		}

		full, err := Layout(root, opts)
		if err != nil {
			t.Fatalf("Layout (full): %v", err)
		}

		if !reflect.DeepEqual(relayoutStripIDs(clone.Ops), relayoutStripIDs(full.Ops)) {
			t.Fatalf("order %v: ops differ from full layout", order)
		}

		if !reflect.DeepEqual(out.Owners, OperationOwners(full)) {
			t.Fatalf("order %v: owners differ from full layout", order)
		}

		if !reflect.DeepEqual(out.Order, PaintOrder(full.Ops)) {
			t.Fatalf("order %v: paint order differs from full layout", order)
		}
	}
}

// TestOpSameExtraAndMixExtraStamp verifies pointer-receiver extra comparison
// and hashing across nil, bound emptyExtra, and non-empty rare payloads.
func TestOpSameExtraAndMixExtraStamp(t *testing.T) {
	t.Parallel()

	var a, b Op
	if a.HasExtra() || a.HasBoundExtra() {
		t.Fatal("zero Op must report HasExtra=false and HasBoundExtra=false")
	}

	if eq, _ := a.SameExtra(&b); !eq {
		t.Fatal("two zero Ops must compare SameExtra=true")
	}

	b.BindEmptyExtra()
	if b.HasExtra() || !b.HasBoundExtra() {
		t.Fatal("bound emptyExtra must report HasExtra=false and HasBoundExtra=true")
	}

	if eq, _ := a.SameExtra(&b); eq {
		t.Fatal("nil extra and bound emptyExtra must differ (IdentityMatrix vs zero Xform)")
	}

	var c Op
	c.BindEmptyExtra()
	if eq, _ := b.SameExtra(&c); !eq {
		t.Fatal("two bound emptyExtra Ops must compare SameExtra=true")
	}

	c.SetURI("https://example.com")
	if !c.HasExtra() {
		t.Fatal("Op with URI must report HasExtra=true")
	}

	if eq, _ := b.SameExtra(&c); eq {
		t.Fatal("emptyExtra and URI extra must compare SameExtra=false")
	}

	const basis, prime uint64 = 14695981039346656037, 1099511628211
	h1, ok1 := b.MixExtraStamp(basis, prime)
	h2, ok2 := c.MixExtraStamp(basis, prime)
	if !ok1 || !ok2 || h1 == h2 {
		t.Fatalf("MixExtraStamp mismatch: ok=(%v,%v) h=(%d,%d)", ok1, ok2, h1, h2)
	}
}
