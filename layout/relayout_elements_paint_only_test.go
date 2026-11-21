package layout_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

// paintOnlyClassSource has two fixed cards, so a class that only changes the
// background keeps every box in place.
const paintOnlyClassSource = `<style>
  .card { width: 200px; height: 60px; background-color: #eeeeee; }
  .card.selected { background-color: #3366cc; }
</style>
<div id="a" class="card">Alpha</div>
<div id="b" class="card">Beta</div>
`

// paintOnlyTextSource nests a fixed block inside a fixed block, so a text edit
// changes the shaped run but not the border boxes.
const paintOnlyTextSource = `<style>
  #card { width: 200px; height: 80px; }
  #label { width: 120px; height: 30px; }
</style>
<div id="card"><div id="label">Alpha</div></div>
`

// paintOnlyPadSource gives a fixed border-box container a padding class: the
// container keeps its border box, but its child moves and narrows.
const paintOnlyPadSource = `<style>
  .padbox { width: 200px; height: 100px; box-sizing: border-box; padding: 0; background-color: #eeeeee; }
  .padbox.padded { padding: 20px; }
  #inner { width: 50px; height: 20px; background-color: #ff0000; }
</style>
<div id="pad" class="padbox"><div id="inner"></div></div>
<div id="tail">Tail</div>
`

// paintOnlyMixedSource holds a class-only card and a padding container, so one
// batch mixes a paint-only id with a geometry-changing one.
const paintOnlyMixedSource = `<style>
  .card { width: 200px; height: 60px; background-color: #eeeeee; }
  .card.selected { background-color: #3366cc; }
  .padbox { width: 200px; height: 100px; box-sizing: border-box; padding: 0; background-color: #eeeeee; }
  .padbox.padded { padding: 20px; }
  #inner { width: 50px; height: 20px; background-color: #ff0000; }
</style>
<div id="a" class="card">Alpha</div>
<div id="pad" class="padbox"><div id="inner"></div></div>
`

// TestRelayoutElementsPaintOnlyClassMatchesFull proves a class-only batch is a
// placement-for-placement replacement for a full relayout: every operation,
// box, owner, order, and page-break slice agrees, and the bounds and fills
// change. The boxes are the previous display's slice, which the paint-only
// path reuses instead of rebuilding.
func TestRelayoutElementsPaintOnlyClassMatchesFull(t *testing.T) {
	t.Parallel()

	tree, err := html.Parse([]byte(paintOnlyClassSource))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")
	display := layRetained(t, styled)

	if !tree.SetAttribute("a", "class", "card selected") {
		t.Fatal("SetAttribute found no element with id a")
	}

	batched, err := layout.RelayoutElements(t.Context(), styled, display, []string{"a"}, layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElements: %v", err)
	}

	full := layOf(t, styled)
	compareDisplays(t, batched, full)

	if !hasFill(batched, 0x33, 0x66, 0xcc) {
		t.Fatal("paint-only batch carries no selected fill")
	}

	if &batched.Boxes[0] != &display.Boxes[0] {
		t.Fatal("paint-only batch rebuilt the display box list")
	}
}

// TestRelayoutElementsPaintOnlyTextRefreshesBoxes proves a text-only batch
// carries the new text in the operations and refreshes the box text of the
// edited element and every ancestor that contains it.
func TestRelayoutElementsPaintOnlyTextRefreshesBoxes(t *testing.T) {
	t.Parallel()

	tree, err := html.Parse([]byte(paintOnlyTextSource))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")
	display := layRetained(t, styled)

	if !tree.SetText("label", "Omega") {
		t.Fatal("SetText found no element with id label")
	}

	batched, err := layout.RelayoutElements(t.Context(), styled, display, []string{"label"}, layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElements: %v", err)
	}

	full := layOf(t, styled)
	compareDisplays(t, batched, full)

	if got := boxByID(t, batched, "label").Text; got != "Omega" {
		t.Fatalf("label box text = %q, want Omega", got)
	}

	if got := boxByID(t, batched, "card").Text; got != "Omega" {
		t.Fatalf("ancestor box text = %q, want Omega", got)
	}

	if !strings.Contains(batched.Boxes[0].Text, "Omega") {
		t.Fatalf("root box text = %q, want it to carry Omega", batched.Boxes[0].Text)
	}

	foundText := false

	for _, op := range batched.Ops {
		if op.Kind == layout.DisplayOpText && strings.Contains(op.Text, "Omega") {
			foundText = true

			break
		}
	}

	if !foundText {
		t.Fatal("paint-only batch carries no Omega text operation")
	}

	if &batched.Boxes[0] != &display.Boxes[0] {
		t.Fatal("paint-only text batch rebuilt the display box list")
	}
}

// TestRelayoutElementsPaddingChangeMatchesFull proves the geometry check
// refuses a child-moving padding change: the batch runs the full splice and
// the result equals a full relayout with a fresh box list.
func TestRelayoutElementsPaddingChangeMatchesFull(t *testing.T) {
	t.Parallel()

	tree, err := html.Parse([]byte(paintOnlyPadSource))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")
	display := layRetained(t, styled)

	innerBefore := boxByID(t, display, "inner")

	if !tree.SetAttribute("pad", "class", "padbox padded") {
		t.Fatal("SetAttribute found no element with id pad")
	}

	batched, err := layout.RelayoutElements(t.Context(), styled, display, []string{"pad"}, layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElements: %v", err)
	}

	full := layOf(t, styled)
	compareDisplays(t, batched, full)

	if boxByID(t, batched, "inner").X == innerBefore.X && boxByID(t, batched, "inner").Y == innerBefore.Y {
		t.Fatal("the padding change did not move the inner box")
	}

	if &batched.Boxes[0] == &display.Boxes[0] {
		t.Fatal("the full splice reused the previous display box list")
	}
}

// TestRelayoutElementsMixedBatchMatchesFull proves one geometry-changing id
// clears the batch paint-only flag: an earlier paint-only window is followed by
// a full splice, and the result equals a full relayout.
func TestRelayoutElementsMixedBatchMatchesFull(t *testing.T) {
	t.Parallel()

	tree, err := html.Parse([]byte(paintOnlyMixedSource))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")
	display := layRetained(t, styled)

	if !tree.SetAttribute("a", "class", "card selected") {
		t.Fatal("SetAttribute found no element with id a")
	}

	if !tree.SetAttribute("pad", "class", "padbox padded") {
		t.Fatal("SetAttribute found no element with id pad")
	}

	batched, err := layout.RelayoutElements(t.Context(), styled, display,
		[]string{"a", "pad"}, layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElements: %v", err)
	}

	full := layOf(t, styled)
	compareDisplays(t, batched, full)

	if !hasFill(batched, 0x33, 0x66, 0xcc) {
		t.Fatal("mixed batch carries no selected fill")
	}

	if &batched.Boxes[0] == &display.Boxes[0] {
		t.Fatal("the mixed batch reused the previous display box list")
	}
}

const sameBoxesTooltipSource = `<!doctype html><html><head><style>
body { margin: 0; }
.cell { position: relative; width: 120px; height: 40px; background: #cccccc; }
.cell .tip {
	display: none; position: absolute; top: 0; left: 0;
	width: 80px; height: 20px; background: #111111; color: #ffffff;
}
.cell.open .tip { display: block; }
</style></head><body>` +
	`<div id="a" class="cell open"><span class="tip">Alpha tip</span></div>` +
	`<div id="b" class="cell"><span class="tip">Beta tip</span></div>` +
	`<div id="c" class="cell"><span class="tip">Gamma tip</span></div>` +
	`</body></html>`

// TestRelayoutElementsSameBoxesTooltipParity verifies that the SameBoxes
// incremental box-list path produces exact Display parity with a full layout
// when child boxes appear/disappear (tooltip toggle) in both forward and
// reverse batch order, while leaving prev.Boxes untouched for transition
// capture.
func TestRelayoutElementsSameBoxesTooltipParity(t *testing.T) {
	t.Parallel()

	for _, order := range [][]string{{"a", "b"}, {"b", "a"}} {
		tree, err := html.Parse([]byte(sameBoxesTooltipSource))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}

		styled := mustApply(t, tree, 400, 300, "", "", "")
		display := layRetained(t, styled)
		prevCount := len(display.Boxes)
		prevFirstID := display.Boxes[0].ID

		if !tree.SetAttribute("a", "class", "cell") {
			t.Fatal("SetAttribute a failed")
		}

		if !tree.SetAttribute("b", "class", "cell open") {
			t.Fatal("SetAttribute b failed")
		}

		batched, err := layout.RelayoutElements(t.Context(), styled, display, order, layout.Options{})
		if err != nil {
			t.Fatalf("RelayoutElements(%v): %v", order, err)
		}

		if len(display.Boxes) != prevCount || display.Boxes[0].ID != prevFirstID {
			t.Fatalf("order %v: prev.Boxes was mutated in place", order)
		}

		if &batched.Boxes[0] == &display.Boxes[0] {
			t.Fatalf("order %v: SameBoxes must not alias prev.Boxes", order)
		}

		full := layOf(t, styled)
		compareDisplaysIgnoringOpIDs(t, batched, full)
	}
}

func compareDisplaysIgnoringOpIDs(t *testing.T, got, want *layout.Display) {
	t.Helper()

	if len(got.Boxes) != len(want.Boxes) {
		t.Fatalf("boxes %d, want %d", len(got.Boxes), len(want.Boxes))
	}

	gotBoxOps := make(map[uint64]struct{}, len(got.Ops))
	for idx := range got.Ops {
		gotBoxOps[got.Ops[idx].ID] = struct{}{}
	}

	for idx := range want.Boxes {
		compareBoxIgnoringOpIDs(t, idx, got.Boxes[idx], want.Boxes[idx], gotBoxOps)
	}

	if len(got.Ops) != len(want.Ops) {
		t.Fatalf("ops %d, want %d", len(got.Ops), len(want.Ops))
	}

	for idx := range want.Ops {
		compareOp(t, idx, &got.Ops[idx], &want.Ops[idx])
	}

	if !reflect.DeepEqual(got.OpOwners, want.OpOwners) {
		t.Fatalf("op owners differ: %v, want %v", got.OpOwners, want.OpOwners)
	}

	if !reflect.DeepEqual(got.Order, want.Order) {
		t.Fatalf("paint order differs: %v, want %v", got.Order, want.Order)
	}
}

func compareBoxIgnoringOpIDs(
	t *testing.T, idx int, gotBox, wantBox layout.Box, gotBoxOps map[uint64]struct{},
) {
	t.Helper()

	if len(gotBox.OperationIDs) != len(wantBox.OperationIDs) {
		t.Fatalf("box %d (%s) opID count = %d, want %d",
			idx, gotBox.ID, len(gotBox.OperationIDs), len(wantBox.OperationIDs))
	}

	for _, opID := range gotBox.OperationIDs {
		if _, ok := gotBoxOps[opID]; !ok {
			t.Fatalf("box %d (%s) references unknown opID %d", idx, gotBox.ID, opID)
		}
	}

	gotBox.OperationIDs = wantBox.OperationIDs
	if !reflect.DeepEqual(gotBox, wantBox) {
		t.Fatalf("box %d: %+v, want %+v", idx, gotBox, wantBox)
	}
}
