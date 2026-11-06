package layout

import (
	"fmt"
	"reflect"
	"testing"
)

// relayoutCreepHTML holds one id-bearing target plus filler cards, so a patch
// of the target leaves the rest of the page alone.
const relayoutCreepHTML = `<html><body>` +
	`<div id="target" class="card">v00</div>` +
	`<div class="card">filler-a</div>` +
	`<div class="card">filler-b</div>` +
	`</body></html>`

// relayoutCreepTargetID is the element the creep test patches.
const relayoutCreepTargetID = "target"

// relayoutCreepPatches is the number of distinct text patches the test applies.
const relayoutCreepPatches = 50

// relayoutCreepWarm is the patch after which the retained style map is
// measured as the steady-state baseline.
const relayoutCreepWarm = 10

// TestRelayoutSubtreeRepeatedTextPatchesDoNotGrowStyles proves repeated text
// patches do not grow the retained Result. RelayoutSubtree is the internal
// element patch path behind the public RelayoutElement, so the test drives it
// directly with 50 distinct same-size text values.
func TestRelayoutSubtreeRepeatedTextPatchesDoNotGrowStyles(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutCreepHTML)
	opts := relayoutOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	target := relayoutNodeByID(t, root, relayoutCreepTargetID)
	if len(target.Children) == 0 {
		t.Fatal("the target has no text child to patch")
	}

	warmSize := 0

	for i := range relayoutCreepPatches {
		target.Children[0].Text = fmt.Sprintf("creep-%02d", i)

		if _, err := RelayoutSubtree(t.Context(), prev, target, opts); err != nil {
			t.Fatalf("RelayoutSubtree patch %d: %v", i, err)
		}

		if i == relayoutCreepWarm-1 {
			warmSize = len(prev.styles)
		}
	}

	finalSize := len(prev.styles)

	if warmSize == 0 {
		t.Fatal("the warmup recorded no styles")
	}

	// The bound is exactly zero growth from patch 10 to patch 50. Every patch
	// re-resolves the same subtree nodes and overwrites the same keys, so a
	// correct retain keeps the map size steady; any per-patch leak would add
	// one entry per distinct text and show up as growth here.
	if growth := finalSize - warmSize; growth != 0 {
		t.Fatalf("styles grew by %d from patch %d to patch %d (warm %d, final %d), want 0",
			growth, relayoutCreepWarm, relayoutCreepPatches, warmSize, finalSize)
	}

	fresh, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout (fresh): %v", err)
	}

	if len(prev.Ops) != len(fresh.Ops) {
		t.Fatalf("patched ops = %d, want the fresh %d", len(prev.Ops), len(fresh.Ops))
	}

	if !reflect.DeepEqual(relayoutStripIDs(prev.Ops), relayoutStripIDs(fresh.Ops)) {
		t.Fatalf("patched ops differ from a fresh layout:\n got %+v\nwant %+v", prev.Ops, fresh.Ops)
	}

	relayoutCreepAssertBoxesMatch(t, prev, fresh)
}

// relayoutCreepAssertBoxesMatch proves the patched page kept the boxes a fresh
// layout produces: the same nodes in the same order with the same geometry
// and the same operation ranges.
func relayoutCreepAssertBoxesMatch(t *testing.T, prev, fresh *Result) {
	t.Helper()

	if len(prev.boxes) != len(fresh.boxes) {
		t.Fatalf("patched boxes = %d, want the fresh %d", len(prev.boxes), len(fresh.boxes))
	}

	for i := range prev.boxes {
		got, want := prev.boxes[i], fresh.boxes[i]

		if got.node != want.node {
			t.Fatalf("box %d node = %p, want %p", i, got.node, want.node)
		}

		if got.kind != want.kind {
			t.Fatalf("box %d kind = %v, want %v", i, got.kind, want.kind)
		}

		if !relayoutNear(got.x, want.x) || !relayoutNear(got.y, want.y) ||
			!relayoutNear(got.w, want.w) || !relayoutNear(got.height, want.height) {
			t.Fatalf("box %d geometry = (%g, %g, %gx%g), want (%g, %g, %gx%g)",
				i, got.x, got.y, got.w, got.height, want.x, want.y, want.w, want.height)
		}

		if got.opStart != want.opStart || got.opEnd != want.opEnd {
			t.Fatalf("box %d range = [%d, %d], want [%d, %d]",
				i, got.opStart, got.opEnd, want.opStart, want.opEnd)
		}
	}
}
