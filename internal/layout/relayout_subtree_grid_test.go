package layout

import (
	"errors"
	"reflect"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// relayoutGridCSS lays the cards out in a grid with two columns. The card
// class gives a card its height and its background, the selected class
// changes only the background color, and the tall class changes the height.
const relayoutGridCSS = `
	.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
	.card { height: 60px; background-color: #eeeeee; }
	.card.selected { background-color: #3366cc; }
	.card.tall { height: 120px; }
`

// relayoutGridHTML holds two cards in a two-column grid, so a patch of the
// second column leaves the first column and the grid around them alone.
const relayoutGridHTML = `<html><body>` +
	`<div class="grid">` +
	`<div id="left" class="card">Alpha</div>` +
	`<div id="right" class="card">Beta</div>` +
	`</div>` +
	`</body></html>`

// relayoutSelectedClass is the class of a card that only changes its
// background color.
const relayoutSelectedClass = "card selected"

// relayoutGridOptions builds the layout options of the grid fixture.
func relayoutGridOptions(t *testing.T) Options {
	t.Helper()

	parsed, err := css.Parse(relayoutGridCSS)
	if err != nil {
		t.Fatalf("css.Parse: %v", err)
	}

	return Options{
		Width:      testViewport,
		Height:     800,
		Sheets:     []*css.Stylesheet{parsed},
		Background: true,
	}
}

// TestRelayoutSubtreePatchesAGridItem proves the splice places a grid item
// in its own track: the item keeps the x and the width it had, the patch
// reaches the display list, and the first column keeps its operations and
// its box.
func TestRelayoutSubtreePatchesAGridItem(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutGridHTML)
	opts := relayoutGridOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	leftBefore := *relayoutBoxByID(t, prev, "left")
	rightBefore := *relayoutBoxByID(t, prev, "right")
	before := cloneOps(prev.Ops)

	leftStart, leftEnd := leftBefore.opStart, leftBefore.opEnd

	right := relayoutNodeByID(t, root, "right")
	right.Attrs["class"] = relayoutSelectedClass

	patch, err := RelayoutSubtree(t.Context(), prev, right, opts)
	if err != nil {
		t.Fatalf("RelayoutSubtree: %v", err)
	}

	// The patched item keeps the place and the size it had in its track.
	patched := patch.rootBox
	if patched.x != rightBefore.x || patched.w != rightBefore.w {
		t.Fatalf("patched box = %gx%g at (%g, %g), want %gx%g at (%g, %g)",
			patched.w, patched.height, patched.x, patched.y,
			rightBefore.w, rightBefore.height, rightBefore.x, rightBefore.y)
	}

	// The patch really restyled the item: the background is the selected blue.
	fill, ok := relayoutFirstFill(patch.Ops)
	if !ok {
		t.Fatal("the patch carries no fill operation")
	}

	if fill != selectedBlue {
		t.Fatalf("patched background = %v, want the selected blue", fill)
	}

	// The patched operations are the ones a full relayout of the grid gives
	// for that item: the item sits in its track, not in the whole grid.
	relayoutAssertMatchesFullRelayout(t, root, opts, patch, "right")

	// The first column keeps its operations and its box.
	for index := leftStart; index <= leftEnd; index++ {
		if !reflect.DeepEqual(prev.Ops[index], before[index]) {
			t.Fatalf("operation %d of the first column changed", index)
		}
	}

	leftAfter := *relayoutBoxByID(t, prev, "left")
	if !reflect.DeepEqual(leftAfter, leftBefore) {
		t.Fatalf("the first column box changed:\n got %+v\nwant %+v", leftAfter, leftBefore)
	}
}

// TestRelayoutSubtreeRejectsAGridItemHeightChange proves the size guard
// keeps its power on a grid item: a class that changes the height of the
// item is refused, because the row below it would be stale.
func TestRelayoutSubtreeRejectsAGridItemHeightChange(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutGridHTML)
	opts := relayoutGridOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	right := relayoutNodeByID(t, root, "right")
	right.Attrs["class"] = "card tall"

	_, err = RelayoutSubtree(t.Context(), prev, right, opts)
	if !errors.Is(err, errSubtreeResized) {
		t.Fatalf("error = %v, want %v", err, errSubtreeResized)
	}
}
