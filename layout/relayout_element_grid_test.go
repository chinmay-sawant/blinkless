package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

// gridAbsoluteSource is a two column grid whose card holds an absolutely
// positioned label. The label's containing block is the card, so the card's
// subtree can be placed again on its own.
const gridAbsoluteSource = `
<style>
  #grid { display: grid; grid-template-columns: 200px 200px; gap: 10px; }
  .card { height: 60px; background-color: #eeeeee; border: 2px solid #333333; position: relative; }
  .card.selected { background-color: #3366cc; }
  .label { position: absolute; top: 4px; left: 8px; width: 40px; height: 12px; background: #ff0000; }
</style>
<div id="grid">
  <div id="card" class="card">Alpha<span id="label" class="label"></span></div>
  <div id="other" class="card">Beta</div>
</div>
`

// TestRelayoutElementPatchesAGridItemWithAnAbsoluteChild checks that a grid
// item that holds an absolutely positioned child is spliced, that the child
// keeps its place, and that the rest of the grid is untouched.
func TestRelayoutElementPatchesAGridItemWithAnAbsoluteChild(t *testing.T) {
	t.Parallel()

	tree, err := html.Parse([]byte(gridAbsoluteSource))
	if err != nil {
		t.Fatal(err)
	}

	styled := mustApply(t, tree, 500, 300, "", "", "")
	current := layRetained(t, styled)

	cardBefore := boxOf(t, current, "card")
	labelBefore := boxOf(t, current, "label")

	if !tree.SetAttribute("card", "class", "card selected") {
		t.Fatal("no card")
	}

	patch, err := layout.RelayoutElement(t.Context(), styled, current, "card", layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElement: %v", err)
	}

	if len(patch.Ops) == 0 {
		t.Fatal("the patch carries no operations")
	}

	if patch.Display == nil {
		t.Fatal("the patch carries no display")
	}

	cardAfter := boxOf(t, patch.Display, "card")
	labelAfter := boxOf(t, patch.Display, "label")

	assertBoxUnchanged(t, cardBefore, cardAfter, "the card moved")
	assertBoxUnchanged(t, labelBefore, labelAfter, "the label moved")

	other := boxOf(t, patch.Display, "other")
	if other.W == 0 || other.H == 0 {
		t.Fatalf("the other card is gone: %+v", other)
	}

	full := layOf(t, styled)
	fullCard := boxOf(t, full, "card")
	assertBoxUnchanged(t, fullCard, cardAfter, "the patched card does not match the full relayout")
}

// assertBoxUnchanged fails when the two boxes differ in position or size.
func assertBoxUnchanged(t *testing.T, want, got layout.Box, message string) {
	t.Helper()

	if want.X != got.X || want.Y != got.Y || want.W != got.W || want.H != got.H {
		t.Fatalf("%s: %+v then %+v", message, want, got)
	}
}

// boxOf returns the box with the given id in a display.
func boxOf(t *testing.T, display *layout.Display, elementID string) layout.Box {
	t.Helper()

	for _, box := range display.Boxes {
		if box.ID == elementID {
			return box
		}
	}

	t.Fatalf("no box %q", elementID)

	return layout.Box{}
}
