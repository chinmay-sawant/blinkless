package layout_test

import (
	"image"
	"math"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

// elementPatchSource is a two-card page. The .selected class changes only the
// background color, so the element keeps its box size, its operation count,
// and its operation kinds, which is what a spliceable patch needs. The .wide
// class changes the width, so it moves the element's box.
const elementPatchSource = `
<style>
  .card { width: 200px; height: 60px; background-color: #eeeeee; border: 2px solid #333333; }
  .card.selected { background-color: #3366cc; }
  .card.wide { width: 260px; }
</style>
<div id="card" class="card">Alpha</div>
<div id="other" class="card">Beta</div>
`

// TestRelayoutElementMatchesAFullRelayout is the element patch parity gate.
// The patched element's operations must equal the ones a full relayout of the
// same document produces, and every other element must keep its operations
// and its box.
func TestRelayoutElementMatchesAFullRelayout(t *testing.T) {
	t.Parallel()

	tree, styled := elementPatchPage(t)

	current := layRetained(t, styled)

	if !tree.SetAttribute("card", "class", "card selected") {
		t.Fatalf("SetAttribute found no element with id card")
	}

	patch, err := layout.RelayoutElement(t.Context(), styled, current, "card", layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElement: %v", err)
	}

	full := layOf(t, styled)

	assertPatchedElementMatches(t, patch, full)
	assertOtherElementUnchanged(t, current, full)
	assertPatchedFillChanged(t, patch, current, full)
}

// TestRelayoutElementGuardsTheSplice pins the three checks a host makes
// before it copies the patch into its retained display: the element is in
// both displays, its box size did not change, and its operation count and
// kinds still line up.
func TestRelayoutElementGuardsTheSplice(t *testing.T) {
	t.Parallel()

	tree, styled := elementPatchPage(t)

	current := layRetained(t, styled)

	if !tree.SetAttribute("card", "class", "card selected") {
		t.Fatalf("SetAttribute found no element with id card")
	}

	patch, err := layout.RelayoutElement(t.Context(), styled, current, "card", layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElement: %v", err)
	}

	if patch.Old.Empty() || patch.New.Empty() {
		t.Fatalf("bounds old=%v new=%v, want the element in both displays", patch.Old, patch.New)
	}

	if patch.Old.Dx() != patch.New.Dx() || patch.Old.Dy() != patch.New.Dy() {
		t.Fatalf("patched box %v, want the old size %v", patch.New, patch.Old)
	}

	oldOps := opsOfElement(current, "card")

	if len(oldOps) != len(patch.Ops) {
		t.Fatalf("patch carries %d operations, want the %d of the element", len(patch.Ops), len(oldOps))
	}

	for index := range oldOps {
		if oldOps[index].Kind != patch.Ops[index].Kind {
			t.Fatalf("operation %d kind %v, want %v", index, patch.Ops[index].Kind, oldOps[index].Kind)
		}
	}
}

// assertPatchedElementMatches checks that the patch holds the operations, the
// boxes, and the bounds a full relayout of the same document produces for the
// patched element.
func assertPatchedElementMatches(t *testing.T, patch *layout.ElementPatch, full *layout.Display) {
	t.Helper()

	if len(patch.Ops) == 0 {
		t.Fatalf("patch carries no operations")
	}

	compareOpSlices(t, "patched element", patch.Ops, opsOfElement(full, "card"))

	cardAfter := boxByID(t, full, "card")

	if patch.New != rectOf(cardAfter) {
		t.Fatalf("new bounds %v, want %v", patch.New, rectOf(cardAfter))
	}

	if len(patch.Boxes) == 0 {
		t.Fatalf("patch carries no boxes")
	}

	if !sameBox(patch.Boxes[0], cardAfter) {
		t.Fatalf("patched box %+v, want %+v", patch.Boxes[0], cardAfter)
	}
}

// assertOtherElementUnchanged checks that an element the patch did not touch
// keeps its operations and its box across the patch.
func assertOtherElementUnchanged(t *testing.T, current, full *layout.Display) {
	t.Helper()

	compareOpSlices(t, "other element", opsOfElement(current, "other"), opsOfElement(full, "other"))

	otherBefore := boxByID(t, current, "other")
	otherAfter := boxByID(t, full, "other")

	if !sameBox(otherBefore, otherAfter) {
		t.Fatalf("other box %+v, want %+v", otherAfter, otherBefore)
	}

	if otherBefore.ParentIndex != otherAfter.ParentIndex {
		t.Fatalf("other box parent %d, want %d", otherAfter.ParentIndex, otherBefore.ParentIndex)
	}
}

// assertPatchedFillChanged checks that the patch carries the new background
// color and that the color changed.
func assertPatchedFillChanged(t *testing.T, patch *layout.ElementPatch, current, full *layout.Display) {
	t.Helper()

	patchedRGB, found := fillRGB(patch.Ops)
	if !found {
		t.Fatalf("patched element carries no fill operation")
	}

	fullRGB, found := fillRGB(opsOfElement(full, "card"))
	if !found {
		t.Fatalf("full relayout carries no fill operation for the element")
	}

	if patchedRGB != fullRGB {
		t.Fatalf("patched fill %v, want %v", patchedRGB, fullRGB)
	}

	beforeRGB, found := fillRGB(opsOfElement(current, "card"))
	if !found {
		t.Fatalf("first layout carries no fill operation for the element")
	}

	if patchedRGB == beforeRGB {
		t.Fatalf("patched fill %v, want a change from %v", patchedRGB, beforeRGB)
	}
}

// elementPatchPage parses elementPatchSource and applies it at one viewport.
func elementPatchPage(t *testing.T) (*html.Document, *css.Document) {
	t.Helper()

	tree, err := html.Parse([]byte(elementPatchSource))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	return tree, mustApply(t, tree, 400, 300, "", "", "")
}

// opsOfElement returns the operations a display attributes to the element
// with the given id and to that element's descendants, in source order.
func opsOfElement(display *layout.Display, elementID string) []layout.DisplayOp {
	index := boxIndexOf(display, elementID)
	if index < 0 {
		return nil
	}

	ops := make([]layout.DisplayOp, 0, len(display.Ops))

	for opIndex, owner := range display.OpOwners {
		if owner >= 0 && boxIsUnder(display, owner, index) {
			ops = append(ops, display.Ops[opIndex])
		}
	}

	return ops
}

// boxIndexOf returns the index of the box for one element id, or -1.
func boxIndexOf(display *layout.Display, elementID string) int {
	for index, box := range display.Boxes {
		if box.ID == elementID {
			return index
		}
	}

	return -1
}

// boxIsUnder reports whether the box at index is the box at ancestor or one
// of its descendants.
func boxIsUnder(display *layout.Display, index, ancestor int) bool {
	for index >= 0 {
		if index == ancestor {
			return true
		}

		index = display.Boxes[index].ParentIndex
	}

	return false
}

// compareOpSlices compares two operation slices operation by operation. The
// kind, the geometry, the text, and the color must all agree.
func compareOpSlices(t *testing.T, label string, got, want []layout.DisplayOp) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s: %d operations, want %d", label, len(got), len(want))
	}

	for index := range want {
		compareOp(t, index, &got[index], &want[index])

		if got[index].R != want[index].R || got[index].G != want[index].G || got[index].B != want[index].B {
			t.Fatalf("%s operation %d: rgb=(%g,%g,%g), want (%g,%g,%g)",
				label, index, got[index].R, got[index].G, got[index].B, want[index].R, want[index].G, want[index].B)
		}
	}
}

// sameBox reports whether two boxes carry the same identity and geometry.
// The parent index is left out: a patch carries the subtree's boxes on their
// own, so the patched element is the root of that list and its parent index
// is -1 there.
func sameBox(got, want layout.Box) bool {
	return got.ID == want.ID && got.Tag == want.Tag && got.Action == want.Action &&
		got.Text == want.Text &&
		got.X == want.X && got.Y == want.Y && got.W == want.W && got.H == want.H
}

// rectOf returns the border box of one box as a rectangle in CSS pixels. The
// engine rounds the box to whole pixels when it reports the patch bounds.
func rectOf(box layout.Box) image.Rectangle {
	left := int(math.Round(box.X))
	top := int(math.Round(box.Y))

	return image.Rect(left, top, left+int(math.Round(box.W)), top+int(math.Round(box.H)))
}

// fillRGB returns the RGB of the first fill operation in the slice.
func fillRGB(ops []layout.DisplayOp) ([3]float64, bool) {
	for _, op := range ops {
		if op.Kind == layout.DisplayOpFillRect {
			return [3]float64{op.R, op.G, op.B}, true
		}
	}

	return [3]float64{}, false
}
