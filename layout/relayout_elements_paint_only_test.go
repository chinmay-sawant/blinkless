package layout_test

import (
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
