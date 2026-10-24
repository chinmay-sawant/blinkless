package layout

import (
	"errors"
	"reflect"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// overrideFlexCSS is a fixed-size flex row, so widening the first card moves
// the second without resizing the row the patch re-places.
const overrideFlexCSS = `
	.row { display: flex; width: 300px; height: 60px; }
	.card { width: 100px; height: 40px; }
`

// overrideFlexHTML holds the cards inside the row.
const overrideFlexHTML = `<html><body>` +
	`<div id="row" class="row">` +
	`<div id="card" class="card">Card</div>` +
	`<div id="tail" class="card">Tail</div>` +
	`</div>` +
	`</body></html>`

// overrideOptions builds the layout options of the flex fixture.
func overrideOptions(t *testing.T) Options {
	t.Helper()

	parsed, err := css.Parse(overrideFlexCSS)
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

// TestRelayoutSubtreeWithOverridesRestoresTheCascade proves the override does
// not survive the call: the retained style map and the element box point at
// the base resolved style again, while the operations and the box keep the
// overridden geometry the call produced.
//
//nolint:cyclop,funlen // the fixture walks the whole override lifecycle
func TestRelayoutSubtreeWithOverridesRestoresTheCascade(t *testing.T) {
	t.Parallel()

	root := mustParse(t, overrideFlexHTML)
	opts := overrideOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	cardNode := relayoutNodeByID(t, root, "card")

	baseStyle := prev.styles[cardNode]
	if baseStyle == nil {
		t.Fatal("the first layout resolved no style for the card")
	}

	cardBefore := *relayoutBoxByID(t, prev, "card")
	tailBefore := *relayoutBoxByID(t, prev, "tail")

	result, err := RelayoutSubtreeWithOverrides(t.Context(), prev, cardNode,
		[]StyleOverride{{Property: "width", Value: "180px"}}, opts)
	if err != nil {
		t.Fatalf("RelayoutSubtreeWithOverrides: %v", err)
	}

	cardAfter := relayoutBoxByID(t, prev, "card")
	if !near(cardAfter.w, 180*ptPerCSSPx) {
		t.Fatalf("patched card width = %gpt, want %gpt", cardAfter.w, 180*ptPerCSSPx)
	}

	if cardAfter.w <= cardBefore.w {
		t.Fatalf("patched card width = %gpt, want wider than %gpt", cardAfter.w, cardBefore.w)
	}

	tailAfter := relayoutBoxByID(t, prev, "tail")
	if !near(tailAfter.x, cardAfter.x+cardAfter.w) {
		t.Fatalf("tail x = %gpt, want %gpt after the card", tailAfter.x, cardAfter.x+cardAfter.w)
	}

	if tailAfter.x <= tailBefore.x {
		t.Fatalf("tail x = %gpt, want a move right from %gpt", tailAfter.x, tailBefore.x)
	}

	if result.root.node != cardNode {
		t.Fatalf("result root node = %v, want the card node", result.root.node)
	}

	if !near(result.root.w, 180*ptPerCSSPx) {
		t.Fatalf("result card width = %gpt, want %gpt", result.root.w, 180*ptPerCSSPx)
	}

	// The boxes keep the overridden geometry, but the style a later patch
	// inherits from is the cascade's style again.
	if !near(cardAfter.style.Width, 100*ptPerCSSPx) {
		t.Fatalf("patched card style width = %gpt, want the base %gpt",
			cardAfter.style.Width, 100*ptPerCSSPx)
	}

	if !near(baseStyle.Width, 100*ptPerCSSPx) {
		t.Fatalf("base width = %gpt, want %gpt", baseStyle.Width, 100*ptPerCSSPx)
	}

	if !near(prev.styles[cardNode].Width, 100*ptPerCSSPx) {
		t.Fatalf("retained style width = %gpt, want the base %gpt",
			prev.styles[cardNode].Width, 100*ptPerCSSPx)
	}

	if cardAfter.style != prev.styles[cardNode] {
		t.Fatal("the patched box and the retained style map disagree about the base style")
	}

	// A second call starts from the stylesheet's width, not from the width
	// the first call installed.
	result, err = RelayoutSubtreeWithOverrides(t.Context(), prev, cardNode,
		[]StyleOverride{{Property: "width", Value: "100px"}}, opts)
	if err != nil {
		t.Fatalf("second RelayoutSubtreeWithOverrides: %v", err)
	}

	if !near(result.root.w, 100*ptPerCSSPx) {
		t.Fatalf("second card width = %gpt, want %gpt", result.root.w, 100*ptPerCSSPx)
	}

	if !near(relayoutBoxByID(t, prev, "tail").x, tailBefore.x) {
		t.Fatalf("tail x = %gpt, want the base %gpt",
			relayoutBoxByID(t, prev, "tail").x, tailBefore.x)
	}
}

// TestRelayoutSubtreeWithOverridesRefusedLeavesPrevUntouched checks the error
// path: validation runs before the splice, so a refused property leaves the
// retained result exactly as it was.
func TestRelayoutSubtreeWithOverridesRefusedLeavesPrevUntouched(t *testing.T) {
	t.Parallel()

	root := mustParse(t, overrideFlexHTML)
	opts := overrideOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	beforeOps := cloneOps(prev.Ops)
	beforeBoxes := make([]*box, len(prev.boxes))
	copy(beforeBoxes, prev.boxes)

	cardNode := relayoutNodeByID(t, root, "card")

	_, err = RelayoutSubtreeWithOverrides(t.Context(), prev, cardNode,
		[]StyleOverride{{Property: "flibbertigibbet", Value: "1"}}, opts)
	if !errors.Is(err, ErrUnknownOverrideProperty) {
		t.Fatalf("error = %v, want %v", err, ErrUnknownOverrideProperty)
	}

	if !reflect.DeepEqual(prev.Ops, beforeOps) {
		t.Fatal("a refused override changed the operations")
	}

	if len(prev.boxes) != len(beforeBoxes) {
		t.Fatalf("a refused override left %d boxes, want %d", len(prev.boxes), len(beforeBoxes))
	}

	for index := range beforeBoxes {
		if prev.boxes[index] != beforeBoxes[index] {
			t.Fatalf("a refused override replaced box %d", index)
		}
	}
}
