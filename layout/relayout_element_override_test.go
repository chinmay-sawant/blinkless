package layout_test

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

// overrideFlexSource is a fixed-size flex row: widening the first card must
// move the second card without resizing the row, which keeps the patch inside
// the containing block.
const overrideFlexSource = `
<style>
  .row { display: flex; width: 300px; height: 60px; }
  .card { width: 100px; height: 40px; }
</style>
<div id="row" class="row">
  <div id="card" class="card">Card</div>
  <div id="tail" class="card">Tail</div>
</div>
`

// overrideTextSource puts text on the element and inside a child element, so
// an inherited font-size reaches a run one level down.
const overrideTextSource = `
<div id="card" style="width:200px;height:40px;font-size:12px">Base <span id="child">Child</span></div>
`

// overrideFlowSource is a block card whose height follows its text, inside a
// body whose height follows the card.
const overrideFlowSource = `
<div id="card" style="width:200px;font-size:12px">` +
	`one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen</div>
`

// overrideAbsoluteSource puts an absolutely positioned card in a relative
// wrapper, so the card's containing block is the wrapper and an inset override
// must re-place it against that box.
const overrideAbsoluteSource = `
<div id="wrap" style="position:relative;width:300px;height:80px">
  <div id="card" style="position:absolute;left:10px;top:10px;width:100px;height:40px">Card</div>
</div>
`

// overrideBorderSource gives the cards a content-box border, so a logical
// border width changes the outer width the flex row places.
const overrideBorderSource = `
<style>
  .row { display: flex; width: 300px; height: 80px; }
  .card { width: 100px; height: 40px; border: 0 solid #000; box-sizing: content-box; }
</style>
<div id="row" class="row">
  <div id="card" class="card">Card</div>
  <div id="tail" class="card">Tail</div>
</div>
`

// cssPixelTolerance absorbs the sub-pixel rounding of a laid-out box.
const cssPixelTolerance = 0.6

// TestRelayoutElementWithOverridesChangesWidthAndReflowsSibling pinpoints the
// geometry path: the override widens the card, the flex row places the next
// card after the new width, and the returned patch reports both bounds.
func TestRelayoutElementWithOverridesChangesWidthAndReflowsSibling(t *testing.T) {
	t.Parallel()

	_, styled := overridePage(t, overrideFlexSource)

	display := layRetained(t, styled)
	cardBefore := boxByID(t, display, "card")
	tailBefore := boxByID(t, display, "tail")

	patch, err := layout.RelayoutElementWithOverrides(t.Context(), styled, display, "card",
		[]layout.Override{{ElementID: "card", Property: "width", Value: "180px"}},
		layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElementWithOverrides: %v", err)
	}

	cardAfter := patchBoxByID(t, patch, "card")
	tailAfter := boxByID(t, patch.Display, "tail")

	assertCSSPixels(t, "patch old width", float64(patch.Old.Dx()), cardBefore.W)
	assertCSSPixels(t, "patch new width", float64(patch.New.Dx()), 180)
	assertCSSPixels(t, "patched card width", cardAfter.W, 180)

	if tailAfter.X <= tailBefore.X {
		t.Fatalf("tail x = %g, want a move right from %g", tailAfter.X, tailBefore.X)
	}

	assertCSSPixels(t, "tail placed after the card", tailAfter.X, cardAfter.X+cardAfter.W)
}

// TestRelayoutElementWithOverridesInheritsFontSize proves the override is part
// of the element's used style, not a box-wide restyle: a child element that
// declares no font size resolves its text run against the overridden size.
func TestRelayoutElementWithOverridesInheritsFontSize(t *testing.T) {
	t.Parallel()

	_, styled := overridePage(t, overrideTextSource)

	display := layRetained(t, styled)

	baseRun, found := textRunContaining(display.Ops, "Child")
	if !found {
		t.Fatal("base layout has no Child text run")
	}

	patch, err := layout.RelayoutElementWithOverrides(t.Context(), styled, display, "card",
		[]layout.Override{{Property: "font-size", Value: "24px"}},
		layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElementWithOverrides: %v", err)
	}

	childRun, ok := textRunContaining(patch.Ops, "Child")
	if !ok {
		t.Fatal("patched layout has no Child text run")
	}

	if childRun.Size <= baseRun.Size {
		t.Fatalf("child text size = %gpt, want larger than the base %gpt", childRun.Size, baseRun.Size)
	}

	assertPoints(t, "child text size", childRun.Size, 18)
}

// TestRelayoutElementWithOverridesAppliesAListAtOnce covers the grouped call:
// one relayout carries a geometry property and an inherited text property, and
// both land in the same patch.
func TestRelayoutElementWithOverridesAppliesAListAtOnce(t *testing.T) {
	t.Parallel()

	_, styled := overridePage(t, overrideTextSource)

	display := layRetained(t, styled)

	patch, err := layout.RelayoutElementWithOverrides(t.Context(), styled, display, "card",
		[]layout.Override{
			{Property: "width", Value: "180px"},
			{Property: "font-size", Value: "24px"},
		},
		layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElementWithOverrides: %v", err)
	}

	assertCSSPixels(t, "patched card width", patchBoxByID(t, patch, "card").W, 180)

	childRun, found := textRunContaining(patch.Ops, "Child")
	if !found {
		t.Fatal("patched layout has no Child text run")
	}

	assertPoints(t, "child text size", childRun.Size, 18)
}

// TestRelayoutElementWithOverridesDoesNotPersist calls the override twice with
// two widths. The second call has to start from the stylesheet's width, not
// from the width the first call installed.
func TestRelayoutElementWithOverridesDoesNotPersist(t *testing.T) {
	t.Parallel()

	tree, styled := overridePage(t, overrideFlexSource)

	display := layRetained(t, styled)
	tailBefore := boxByID(t, display, "tail")

	_, err := layout.RelayoutElementWithOverrides(t.Context(), styled, display, "card",
		[]layout.Override{{Property: "width", Value: "180px"}},
		layout.Options{})
	if err != nil {
		t.Fatalf("first override: %v", err)
	}

	patch, err := layout.RelayoutElementWithOverrides(t.Context(), styled, display, "card",
		[]layout.Override{{Property: "width", Value: "100px"}},
		layout.Options{})
	if err != nil {
		t.Fatalf("second override: %v", err)
	}

	assertCSSPixels(t, "second patch width", float64(patch.New.Dx()), 100)
	assertCSSPixels(t, "second patched card width", patchBoxByID(t, patch, "card").W, 100)
	assertCSSPixels(t, "tail back at its base x", boxByID(t, patch.Display, "tail").X, tailBefore.X)

	// The parsed tree is unchanged, so a fresh cascade and layout of the same
	// document shows the authored width.
	fresh := layRetained(t, mustApply(t, tree, 400, 300, "", "", ""))

	assertCSSPixels(t, "fresh layout card width", boxByID(t, fresh, "card").W, 100)
}

// TestRelayoutElementWithOverridesWithoutOverridesEqualsRelayoutElement pins
// the delegation: an empty list is the ordinary element patch, not a weaker
// containing-block patch.
func TestRelayoutElementWithOverridesWithoutOverridesEqualsRelayoutElement(t *testing.T) {
	t.Parallel()

	_, styled := overridePage(t, overrideFlexSource)

	current := layRetained(t, styled)
	plain := layRetained(t, styled)

	withOverrides, err := layout.RelayoutElementWithOverrides(t.Context(), styled, current, "card",
		nil, layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElementWithOverrides: %v", err)
	}

	relaid, err := layout.RelayoutElement(t.Context(), styled, plain, "card", layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElement: %v", err)
	}

	compareOpSlices(t, "override path", withOverrides.Ops, relaid.Ops)

	if len(withOverrides.Boxes) != len(relaid.Boxes) {
		t.Fatalf("override path has %d boxes, want %d",
			len(withOverrides.Boxes), len(relaid.Boxes))
	}

	for index := range relaid.Boxes {
		if !sameBox(withOverrides.Boxes[index], relaid.Boxes[index]) {
			t.Fatalf("box %d = %+v, want %+v",
				index, withOverrides.Boxes[index], relaid.Boxes[index])
		}
	}

	if withOverrides.Old != relaid.Old || withOverrides.New != relaid.New {
		t.Fatalf("bounds old=%v new=%v, want old=%v new=%v",
			withOverrides.Old, withOverrides.New, relaid.Old, relaid.New)
	}
}

// TestRelayoutElementWithOverridesUnknownProperty checks the sentinel and the
// atomicity: a refused property leaves the display exactly as it was.
func TestRelayoutElementWithOverridesUnknownProperty(t *testing.T) {
	t.Parallel()

	_, styled := overridePage(t, overrideFlexSource)

	display := layRetained(t, styled)
	beforeOps := slices.Clone(display.Ops)
	beforeBoxes := slices.Clone(display.Boxes)

	for _, property := range []string{"flibbertigibbet", "transform", "--accent"} {
		_, err := layout.RelayoutElementWithOverrides(t.Context(), styled, display, "card",
			[]layout.Override{{Property: property, Value: "1px"}},
			layout.Options{})
		if !errors.Is(err, layout.ErrUnknownOverrideProperty) {
			t.Fatalf("property %q error = %v, want %v",
				property, err, layout.ErrUnknownOverrideProperty)
		}
	}

	assertDisplayUnchanged(t, display, beforeOps, beforeBoxes)
}

// TestRelayoutElementWithOverridesMalformedValue checks that an invalid value
// is a clean error rather than a silent fallback to the base value.
func TestRelayoutElementWithOverridesMalformedValue(t *testing.T) {
	t.Parallel()

	_, styled := overridePage(t, overrideFlexSource)

	display := layRetained(t, styled)
	beforeOps := slices.Clone(display.Ops)
	beforeBoxes := slices.Clone(display.Boxes)

	for _, value := range []string{"", "wide", "12parsecs", "inherit", "var(--width)"} {
		_, err := layout.RelayoutElementWithOverrides(t.Context(), styled, display, "card",
			[]layout.Override{{Property: "width", Value: value}},
			layout.Options{})
		if !errors.Is(err, layout.ErrInvalidOverrideValue) {
			t.Fatalf("value %q error = %v, want %v",
				value, err, layout.ErrInvalidOverrideValue)
		}
	}

	assertDisplayUnchanged(t, display, beforeOps, beforeBoxes)
}

// TestRelayoutElementWithOverridesRequiresRetainedState pins the same state
// requirement RelayoutElement has.
func TestRelayoutElementWithOverridesRequiresRetainedState(t *testing.T) {
	t.Parallel()

	_, styled := overridePage(t, overrideFlexSource)

	display := layOf(t, styled)

	_, err := layout.RelayoutElementWithOverrides(t.Context(), styled, display, "card",
		[]layout.Override{{Property: "width", Value: "180px"}},
		layout.Options{})
	if !errors.Is(err, layout.ErrNoRetainedState) {
		t.Fatalf("error = %v, want %v", err, layout.ErrNoRetainedState)
	}
}

// TestRelayoutElementWithOverridesOtherElementID rejects an override that
// names an element other than the relayout target instead of ignoring it.
func TestRelayoutElementWithOverridesOtherElementID(t *testing.T) {
	t.Parallel()

	_, styled := overridePage(t, overrideFlexSource)

	display := layRetained(t, styled)

	_, err := layout.RelayoutElementWithOverrides(t.Context(), styled, display, "card",
		[]layout.Override{{ElementID: "tail", Property: "width", Value: "180px"}},
		layout.Options{})
	if !errors.Is(err, layout.ErrNoElement) {
		t.Fatalf("error = %v, want %v", err, layout.ErrNoElement)
	}
}

// TestRelayoutElementWithOverridesRefusesAReflowThatEscapes checks the
// documented limit: a change that resizes the containing block is refused
// instead of leaving the page around it stale.
func TestRelayoutElementWithOverridesRefusesAReflowThatEscapes(t *testing.T) {
	t.Parallel()

	_, styled := overridePage(t, overrideFlowSource)

	display := layRetained(t, styled)
	beforeOps := slices.Clone(display.Ops)
	beforeBoxes := slices.Clone(display.Boxes)

	_, err := layout.RelayoutElementWithOverrides(t.Context(), styled, display, "card",
		[]layout.Override{{Property: "width", Value: "40px"}},
		layout.Options{})
	if !errors.Is(err, layout.ErrOverrideReflow) {
		t.Fatalf("error = %v, want %v", err, layout.ErrOverrideReflow)
	}

	assertDisplayUnchanged(t, display, beforeOps, beforeBoxes)
}

// TestRelayoutElementWithOverridesExpandsALogicalBorderWidth covers the
// logical border family: the inline width expands to the two physical sides,
// so the outer box grows by both borders and the next card moves.
func TestRelayoutElementWithOverridesExpandsALogicalBorderWidth(t *testing.T) {
	t.Parallel()

	_, styled := overridePage(t, overrideBorderSource)

	display := layRetained(t, styled)
	tailBefore := boxByID(t, display, "tail")

	patch, err := layout.RelayoutElementWithOverrides(t.Context(), styled, display, "card",
		[]layout.Override{{Property: "border-inline-width", Value: "10px"}},
		layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElementWithOverrides: %v", err)
	}

	cardAfter := patchBoxByID(t, patch, "card")
	assertCSSPixels(t, "card outer width", cardAfter.W, 120)
	assertCSSPixels(t, "tail after the bordered card", boxByID(t, patch.Display, "tail").X,
		cardAfter.X+cardAfter.W)

	if boxByID(t, patch.Display, "tail").X <= tailBefore.X {
		t.Fatalf("tail x = %g, want a move right from %g",
			boxByID(t, patch.Display, "tail").X, tailBefore.X)
	}
}

// TestRelayoutElementWithOverridesMovesAnAbsoluteCard covers the inset family
// and the containing-block target: the card is out of flow, so the patch
// re-places the relative wrapper it is positioned against.
func TestRelayoutElementWithOverridesMovesAnAbsoluteCard(t *testing.T) {
	t.Parallel()

	_, styled := overridePage(t, overrideAbsoluteSource)

	display := layRetained(t, styled)
	cardBefore := boxByID(t, display, "card")
	wrapBefore := boxByID(t, display, "wrap")

	patch, err := layout.RelayoutElementWithOverrides(t.Context(), styled, display, "card",
		[]layout.Override{{Property: "left", Value: "40px"}},
		layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElementWithOverrides: %v", err)
	}

	cardAfter := patchBoxByID(t, patch, "card")
	if cardAfter.X <= cardBefore.X {
		t.Fatalf("card x = %g, want a move right from %g", cardAfter.X, cardBefore.X)
	}

	assertCSSPixels(t, "absolute card offset", cardAfter.X-wrapBefore.X, 40)

	wrapAfter := boxByID(t, patch.Display, "wrap")
	assertCSSPixels(t, "wrapper width", wrapAfter.W, wrapBefore.W)
	assertCSSPixels(t, "wrapper height", wrapAfter.H, wrapBefore.H)
}

// overridePage parses source and applies it at one viewport.
func overridePage(t *testing.T, source string) (*html.Document, *css.Document) {
	t.Helper()

	tree, err := html.Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	return tree, mustApply(t, tree, 400, 300, "", "", "")
}

// patchBoxByID returns the box with the given id from a patch.
//
//nolint:unparam // the helper keeps the same shape as boxByID
func patchBoxByID(t *testing.T, patch *layout.ElementPatch, elementID string) layout.Box {
	t.Helper()

	for index := range patch.Boxes {
		if patch.Boxes[index].ID == elementID {
			return patch.Boxes[index]
		}
	}

	t.Fatalf("patch has no box for id %q", elementID)

	return layout.Box{}
}

// textRunContaining returns the first text operation whose text contains want.
func textRunContaining(ops []layout.DisplayOp, want string) (layout.DisplayOp, bool) {
	for index := range ops {
		if ops[index].Kind == layout.DisplayOpText && strings.Contains(ops[index].Text, want) {
			return ops[index], true
		}
	}

	return layout.DisplayOp{}, false
}

// assertCSSPixels fails when two CSS-pixel values differ by more than the
// layout's sub-pixel rounding.
func assertCSSPixels(t *testing.T, label string, got, want float64) {
	t.Helper()

	if math.Abs(got-want) > cssPixelTolerance {
		t.Fatalf("%s = %gpx, want %gpx", label, got, want)
	}
}

// assertPoints fails when two layout-point values differ by more than the
// layout's sub-pixel rounding.
func assertPoints(t *testing.T, label string, got, want float64) {
	t.Helper()

	if math.Abs(got-want) > 0.1 {
		t.Fatalf("%s = %gpt, want %gpt", label, got, want)
	}
}

// assertDisplayUnchanged checks that a refused call left the operations and
// the boxes of the display exactly as they were.
func assertDisplayUnchanged(
	t *testing.T,
	display *layout.Display,
	ops []layout.DisplayOp,
	boxes []layout.Box,
) {
	t.Helper()

	if !reflect.DeepEqual(display.Ops, ops) {
		t.Fatal("a refused call changed the display operations")
	}

	if !reflect.DeepEqual(display.Boxes, boxes) {
		t.Fatal("a refused call changed the display boxes")
	}
}
