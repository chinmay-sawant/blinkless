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

	if cardAfter.X != cardBefore.X || cardAfter.Y != cardBefore.Y ||
		cardAfter.W != cardBefore.W || cardAfter.H != cardBefore.H {
		t.Fatalf("the card moved: %+v then %+v", cardBefore, cardAfter)
	}

	if labelAfter.X != labelBefore.X || labelAfter.Y != labelBefore.Y {
		t.Fatalf("the label moved: %+v then %+v", labelBefore, labelAfter)
	}

	other := boxOf(t, patch.Display, "other")
	if other.W == 0 || other.H == 0 {
		t.Fatalf("the other card is gone: %+v", other)
	}

	full := layOf(t, styled)
	fullCard := boxOf(t, full, "card")
	if fullCard.X != cardAfter.X || fullCard.Y != cardAfter.Y ||
		fullCard.W != cardAfter.W || fullCard.H != cardAfter.H {
		t.Fatalf("the patched card %+v does not match the full relayout %+v", cardAfter, fullCard)
	}
}

// boxOf returns the box with the given id in a display.
func boxOf(t *testing.T, display *layout.Display, id string) layout.Box {
	t.Helper()

	for _, box := range display.Boxes {
		if box.ID == id {
			return box
		}
	}

	t.Fatalf("no box %q", id)

	return layout.Box{}
}
