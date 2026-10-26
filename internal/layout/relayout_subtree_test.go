package layout

import (
	"errors"
	"reflect"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// relayoutFixtureCSS gives every card a size and a chrome, and a selected
// class that only changes the background color, so a selected card keeps its
// size and its operation count. The wide class changes the width, so a wide
// card no longer fits where it stood.
const relayoutFixtureCSS = `
	.card { width: 200px; height: 60px; background-color: #eeeeee; border: 2px solid #333333; }
	.card.selected { background-color: #3366cc; }
	.card.wide { width: 260px; }
	.tinted { background-color: #3366cc; }
`

// relayoutFixtureHTML holds two cards in a row, so a patch of the first card
// leaves the second card and the page around them alone.
const relayoutFixtureHTML = `<html><body>` +
	`<div id="first" class="card">Alpha</div>` +
	`<div id="second" class="card">Beta</div>` +
	`</body></html>`

// relayoutFloatHTML holds a float inside the element a patch would touch.
const relayoutFloatHTML = `<html><body>` +
	`<div id="host"><div id="floater" style="float:left">floated</div></div>` +
	`</body></html>`

// relayoutNestedHTML holds a card inside a plain block, so the patch reaches
// an element that is not a child of the body.
const relayoutNestedHTML = `<html><body>` +
	`<div id="outer"><div id="first" class="card">Alpha</div></div>` +
	`<div id="second" class="card">Beta</div>` +
	`</body></html>`

// relayoutEmptyHTML holds an element that paints nothing next to a card that
// paints something.
const relayoutEmptyHTML = `<html><body>` +
	`<div id="empty"></div>` +
	`<div id="second" class="card">Beta</div>` +
	`</body></html>`

// selectedBlue is the background the selected class paints.
var selectedBlue = [3]float64{51.0 / 255, 102.0 / 255, 204.0 / 255} //nolint:gochecknoglobals // a test constant

// TestRelayoutSubtreeKeepsTheRestOfThePage proves the splice is local: the
// patched element's operations equal the ones a full relayout of the same
// document produces, and every other operation and box stays byte identical.
func TestRelayoutSubtreeKeepsTheRestOfThePage(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutFixtureHTML)
	opts := relayoutOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	before := cloneOps(prev.Ops)

	firstBox := relayoutBoxByID(t, prev, "first")
	secondBefore := *relayoutBoxByID(t, prev, "second")

	oldStart, oldEnd := firstBox.opStart, firstBox.opEnd

	first := relayoutNodeByID(t, root, "first")
	first.Attrs["class"] = "card selected"

	patch, err := RelayoutSubtree(t.Context(), prev, first, opts)
	if err != nil {
		t.Fatalf("RelayoutSubtree: %v", err)
	}

	relayoutAssertMatchesFullRelayout(t, root, opts, patch, "first")

	// The patch really restyled the element: the background is the selected
	// blue, not the grey the first layout painted.
	fill, ok := relayoutFirstFill(patch.Ops)
	if !ok {
		t.Fatal("the patch carries no fill operation")
	}

	if fill != selectedBlue {
		t.Fatalf("patched background = %v, want the selected blue", fill)
	}

	relayoutAssertRestUnchanged(t, prev, before, oldStart, oldEnd, len(patch.Ops))

	// The box of the untouched element keeps its geometry and its operation
	// range.
	secondAfter := relayoutBoxByID(t, prev, "second")
	if !reflect.DeepEqual(*secondAfter, secondBefore) {
		t.Fatalf("the second card changed:\n got %+v\nwant %+v", *secondAfter, secondBefore)
	}
}

// TestRelayoutSubtreePatchesText proves a text change reaches the display
// list: the element keeps its size, so the splice places it, and its text
// operation carries the new text.
func TestRelayoutSubtreePatchesText(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutFixtureHTML)
	opts := relayoutOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	first := relayoutNodeByID(t, root, "first")
	first.Children[0].Text = "Omega"

	patch, err := RelayoutSubtree(t.Context(), prev, first, opts)
	if err != nil {
		t.Fatalf("RelayoutSubtree: %v", err)
	}

	relayoutAssertMatchesFullRelayout(t, root, opts, patch, "first")

	if got := relayoutPatchText(patch.Ops); got != "Omega" {
		t.Fatalf("patched text = %q, want %q", got, "Omega")
	}
}

// TestRelayoutSubtreePatchesANestedElement proves the splice reaches an
// element deeper in the tree: the element is placed inside the block that
// holds it, and the page around it keeps its operations.
func TestRelayoutSubtreePatchesANestedElement(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutNestedHTML)
	opts := relayoutOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	before := cloneOps(prev.Ops)

	firstBox := relayoutBoxByID(t, prev, "first")
	oldStart, oldEnd := firstBox.opStart, firstBox.opEnd

	first := relayoutNodeByID(t, root, "first")
	first.Attrs["class"] = "card selected"

	patch, err := RelayoutSubtree(t.Context(), prev, first, opts)
	if err != nil {
		t.Fatalf("RelayoutSubtree: %v", err)
	}

	relayoutAssertMatchesFullRelayout(t, root, opts, patch, "first")
	relayoutAssertRestUnchanged(t, prev, before, oldStart, oldEnd, len(patch.Ops))
}

// TestRelayoutSubtreePatchesAnElementWithoutOperations proves the splice
// places the operations of an element that painted nothing: the operations
// land in front of the next sibling that has some.
func TestRelayoutSubtreePatchesAnElementWithoutOperations(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutEmptyHTML)
	opts := relayoutOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	empty := relayoutBoxByID(t, prev, "empty")
	if empty.opEnd >= empty.opStart {
		t.Fatalf("the empty element owns the operations [%d, %d], want none",
			empty.opStart, empty.opEnd)
	}

	secondBefore := *relayoutBoxByID(t, prev, "second")

	relayoutNodeByID(t, root, "empty").Attrs["class"] = "tinted"

	patch, err := RelayoutSubtree(t.Context(), prev, relayoutNodeByID(t, root, "empty"), opts)
	if err != nil {
		t.Fatalf("RelayoutSubtree: %v", err)
	}

	if len(patch.Ops) == 0 {
		t.Fatal("the patch carries no operations")
	}

	fill, ok := relayoutFirstFill(patch.Ops)
	if !ok {
		t.Fatal("the patch carries no fill operation")
	}

	if fill != selectedBlue {
		t.Fatalf("patched background = %v, want the selected blue", fill)
	}

	relayoutAssertMatchesFullRelayout(t, root, opts, patch, "empty")

	secondAfter := relayoutBoxByID(t, prev, "second")
	if !relayoutSameGeometry(secondAfter, &secondBefore) {
		t.Fatalf("the second card moved:\n got %+v\nwant %+v", *secondAfter, secondBefore)
	}
}

// relayoutSameGeometry reports whether two boxes sit at the same place with
// the same size. The operation range is left out: it moves when the patch
// changes the number of operations on the page.
func relayoutSameGeometry(got, want *box) bool {
	return got.x == want.x && got.y == want.y && got.w == want.w && got.height == want.height
}

// TestRelayoutSubtreeRejectsASizeChange proves the splice refuses an element
// that no longer fits where it stood: the boxes after it would be stale.
func TestRelayoutSubtreeRejectsASizeChange(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutFixtureHTML)
	opts := relayoutOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	first := relayoutNodeByID(t, root, "first")
	first.Attrs["class"] = "card wide"

	_, err = RelayoutSubtree(t.Context(), prev, first, opts)
	if !errors.Is(err, errSubtreeResized) {
		t.Fatalf("error = %v, want %v", err, errSubtreeResized)
	}
}

// TestRelayoutSubtreeRejectsAFloat proves the splice refuses a subtree that
// leaves the flow: a float is placed against the state of the block that
// holds it, and the kept result does not carry that state.
func TestRelayoutSubtreeRejectsAFloat(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutFloatHTML)
	opts := relayoutOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	_, err = RelayoutSubtree(t.Context(), prev, relayoutNodeByID(t, root, "host"), opts)
	if !errors.Is(err, errSubtreeNotSplicable) {
		t.Fatalf("error = %v, want %v", err, errSubtreeNotSplicable)
	}
}

// TestRelayoutSubtreeWithoutAPage proves the splice reports the inputs it
// cannot work with instead of guessing.
func TestRelayoutSubtreeWithoutAPage(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutFixtureHTML)
	opts := relayoutOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	first := relayoutNodeByID(t, root, "first")

	_, err = RelayoutSubtree(t.Context(), nil, first, opts)
	if !errors.Is(err, errNoPreviousLayout) {
		t.Fatalf("error = %v, want %v", err, errNoPreviousLayout)
	}

	_, err = RelayoutSubtree(t.Context(), prev, nil, opts)
	if !errors.Is(err, errNilElementNode) {
		t.Fatalf("error = %v, want %v", err, errNilElementNode)
	}

	unknown := &html.Node{Type: html.ElementNode, Name: "div"}

	_, err = RelayoutSubtree(t.Context(), prev, unknown, opts)
	if !errors.Is(err, errElementNotPlaced) {
		t.Fatalf("error = %v, want %v", err, errElementNotPlaced)
	}
}

// relayoutAssertMatchesFullRelayout proves the operations of the patch are
// the ones a full relayout of the same document produces for that element.
func relayoutAssertMatchesFullRelayout(
	t *testing.T, root *html.Node, opts Options, patch *Result, elementID string,
) {
	t.Helper()

	full, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout (full relayout): %v", err)
	}

	fullBox := relayoutBoxByID(t, full, elementID)
	wantOps := full.Ops[fullBox.opStart : fullBox.opEnd+1]

	if len(patch.Ops) != len(wantOps) {
		t.Fatalf("patched operations = %d, want the %d of a full relayout", len(patch.Ops), len(wantOps))
	}

	if !reflect.DeepEqual(relayoutStripIDs(patch.Ops), relayoutStripIDs(wantOps)) {
		t.Fatalf("patched operations differ from a full relayout:\n got %+v\nwant %+v", patch.Ops, wantOps)
	}
}

// relayoutAssertRestUnchanged proves the operations outside the patched
// window are the ones the first layout produced.
func relayoutAssertRestUnchanged(
	t *testing.T, prev *Result, before []Op, oldStart, oldEnd, newCount int,
) {
	t.Helper()

	newEnd := oldStart + newCount - 1

	for index := range oldStart {
		if !reflect.DeepEqual(prev.Ops[index], before[index]) {
			t.Fatalf("operation %d before the patch changed", index)
		}
	}

	for index := newEnd + 1; index < len(prev.Ops); index++ {
		oldIndex := oldEnd + 1 + (index - (newEnd + 1))
		if oldIndex >= len(before) {
			t.Fatalf("operation %d has no counterpart in the first layout", index)
		}

		if !reflect.DeepEqual(prev.Ops[index], before[oldIndex]) {
			t.Fatalf("operation %d after the patch changed", index)
		}
	}

	if len(prev.Ops) != len(before) {
		t.Fatalf("patched page carries %d operations, want the %d of the first layout",
			len(prev.Ops), len(before))
	}
}

// relayoutOptions builds the layout options of the fixtures.
func relayoutOptions(t *testing.T) Options {
	t.Helper()

	return relayoutOptionsForCSS(t, relayoutFixtureCSS)
}

// relayoutOptionsForCSS parses cssText and builds layout options around it.
func relayoutOptionsForCSS(t *testing.T, cssText string) Options {
	t.Helper()

	parsed, err := css.Parse(cssText)
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

// relayoutNodeByID returns the element node with the given id.
func relayoutNodeByID(t *testing.T, root *html.Node, elementID string) *html.Node {
	t.Helper()

	found := root.FindFirst(func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Attribute("id") == elementID
	})
	if found == nil {
		t.Fatalf("no element with id %q", elementID)
	}

	return found
}

// relayoutBoxByID returns the box of the element with the given id.
func relayoutBoxByID(t *testing.T, res *Result, elementID string) *box {
	t.Helper()

	found := res.boxForID(elementID)
	if found == nil {
		t.Fatalf("no box for the element with id %q", elementID)
	}

	return found
}

// relayoutStripIDs copies the operations and clears their identity, so two
// results can be compared: the identity of an operation is unique per result.
func relayoutStripIDs(ops []Op) []Op {
	out := make([]Op, len(ops))
	copy(out, ops)

	for index := range out {
		out[index].ID = 0
	}

	return out
}

// relayoutPatchText returns the text of the first text operation.
func relayoutPatchText(ops []Op) string {
	for index := range ops {
		if ops[index].Kind == OpText {
			return ops[index].Text
		}
	}

	return ""
}

// relayoutFirstFill returns the color of the first fill operation.
func relayoutFirstFill(ops []Op) ([3]float64, bool) {
	for index := range ops {
		if ops[index].Kind == OpFillRect {
			return [3]float64{ops[index].R, ops[index].G, ops[index].B}, true
		}
	}

	return [3]float64{}, false
}

// TestRelayoutContainingBlockInsideStartsAtParent checks that an out-of-flow
// box never counts as its own containing block. A fixed box only accepts an
// ancestor inside the subtree that establishes one.
func TestRelayoutContainingBlockInsideStartsAtParent(t *testing.T) {
	t.Parallel()

	document := &html.Node{Type: html.ElementNode, Name: "html"}
	body := &html.Node{Type: html.ElementNode, Name: "body", Parent: document}
	card := &html.Node{Type: html.ElementNode, Name: "div", Parent: body}
	boxNode := &html.Node{Type: html.ElementNode, Name: "span", Parent: card}

	styles := map[*html.Node]*ResolvedStyle{
		card:     {Position: "static"},
		body:     {Position: positionRelative},
		document: {Position: "static"},
		boxNode:  {Position: positionAbsolute},
	}

	if relayoutContainingBlockInside(boxNode, card, styles) {
		t.Fatal("a relative body above the subtree accepted an absolute box")
	}

	styles[card] = &ResolvedStyle{Position: positionRelative}

	if !relayoutContainingBlockInside(boxNode, card, styles) {
		t.Fatal("a relative card inside the subtree was not accepted")
	}

	styles[boxNode] = &ResolvedStyle{Position: positionFixed}
	styles[card] = &ResolvedStyle{Position: positionRelative}

	if relayoutContainingBlockInside(boxNode, card, styles) {
		t.Fatal("a relative card accepted a fixed box")
	}

	styles[card] = &ResolvedStyle{Position: "static", HasTransform: true}

	if !relayoutContainingBlockInside(boxNode, card, styles) {
		t.Fatal("a transformed card did not accept a fixed box")
	}

	styles[card] = &ResolvedStyle{Position: "static"}
	styles[body] = &ResolvedStyle{Position: positionRelative, HasTransform: true}

	if relayoutContainingBlockInside(boxNode, card, styles) {
		t.Fatal("a transformed body above the subtree accepted a fixed box")
	}
}
