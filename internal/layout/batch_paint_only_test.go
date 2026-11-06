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
