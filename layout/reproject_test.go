package layout_test

import (
	"errors"
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

// restampPage is a card under a perspective stage.
const restampPage = `<html><body style="margin:0">` +
	`<div id="stage" style="width:300px;height:200px;perspective:500px;perspective-origin:30% 20%">` +
	`<div id="card" style="width:100px;height:60px;background-color:#ff0000;transform:rotateY(20deg)">card</div>` +
	`</div></body></html>`

// restampPageAt returns the same page with the card's transform replaced, the
// declaration the override has to reproduce.
func restampPageAt(decl string) string {
	return `<html><body style="margin:0">` +
		`<div id="stage" style="width:300px;height:200px;perspective:500px;perspective-origin:30% 20%">` +
		`<div id="card" style="width:100px;height:60px;background-color:#ff0000;` + decl + `">card</div>` +
		`</div></body></html>`
}

// TestRestampTransformsMatchesFreshDisplay is the public acceptance gate: an
// override produces the operations a fresh layout with that declaration
// produces, and the patch carries adoptable damage.
func TestRestampTransformsMatchesFreshDisplay(t *testing.T) {
	t.Parallel()

	styled := applyAt(t, restampPage, 500, 800, "", "", "")
	display := layRetained(t, styled)

	patch, err := layout.RestampTransforms(t.Context(), styled, display, []layout.TransformOverride{{
		ElementID:    "card",
		Transform:    "rotateY(45deg)",
		HasTransform: true,
	}}, layout.Options{})
	if err != nil {
		t.Fatalf("RestampTransforms: %v", err)
	}

	if patch.Display == nil {
		t.Fatal("patch display = nil")
	}

	if patch.Old.Empty() || patch.New.Empty() {
		t.Fatalf("damage old=%v new=%v, want both non-empty", patch.Old, patch.New)
	}

	if len(patch.Ops) == 0 || len(patch.OpOwners) != len(patch.Ops) {
		t.Fatalf("patch ops=%d owners=%d, want one owner per op", len(patch.Ops), len(patch.OpOwners))
	}

	if len(patch.Boxes) == 0 {
		t.Fatal("patch boxes = 0, want the patched elements")
	}

	fresh := displayAt(t, restampPageAt("transform:rotateY(45deg)"), 500, 800, "", "", "")

	restampCompareDisplays(t, patch.Display, fresh)
}

// restampCompareDisplays compares the painted geometry of two displays.
func restampCompareDisplays(t *testing.T, got, want *layout.Display) {
	t.Helper()

	if len(got.Ops) != len(want.Ops) {
		t.Fatalf("patched ops = %d, want %d", len(got.Ops), len(want.Ops))
	}

	for idx := range want.Ops {
		gotOp, wantOp := &got.Ops[idx], &want.Ops[idx]

		if gotOp.Kind != wantOp.Kind || gotOp.X != wantOp.X || gotOp.Y != wantOp.Y ||
			gotOp.W != wantOp.W || gotOp.H != wantOp.H {
			t.Fatalf("op %d: kind=%v geom=(%g,%g,%g,%g), want kind=%v geom=(%g,%g,%g,%g)",
				idx, gotOp.Kind, gotOp.X, gotOp.Y, gotOp.W, gotOp.H,
				wantOp.Kind, wantOp.X, wantOp.Y, wantOp.W, wantOp.H)
		}
	}
}

// TestRestampTransformsWithNoOverrideIsANoOp pins the empty pass: the display
// hands itself back and nothing is damaged.
func TestRestampTransformsWithNoOverrideIsANoOp(t *testing.T) {
	t.Parallel()

	styled := applyAt(t, restampPage, 500, 800, "", "", "")
	display := layRetained(t, styled)

	patch, err := layout.RestampTransforms(t.Context(), styled, display, nil, layout.Options{})
	if err != nil {
		t.Fatalf("RestampTransforms: %v", err)
	}

	if patch.Display != display {
		t.Fatal("empty patch did not hand the same display back")
	}

	if !patch.Old.Empty() || !patch.New.Empty() {
		t.Fatalf("damage old=%v new=%v, want no damage", patch.Old, patch.New)
	}
}

// TestRestampTransformsWithoutRetainedState pins the retained-state guard.
func TestRestampTransformsWithoutRetainedState(t *testing.T) {
	t.Parallel()

	styled := applyAt(t, restampPage, 500, 800, "", "", "")
	display := layOf(t, styled)

	_, err := layout.RestampTransforms(t.Context(), styled, display, []layout.TransformOverride{{
		ElementID:    "card",
		Transform:    "rotateY(45deg)",
		HasTransform: true,
	}}, layout.Options{})

	if !errors.Is(err, layout.ErrNoRetainedState) {
		t.Fatalf("error = %v, want %v", err, layout.ErrNoRetainedState)
	}
}

// TestRestampTransformsRejectsMalformedTransform pins the parser error.
func TestRestampTransformsRejectsMalformedTransform(t *testing.T) {
	t.Parallel()

	styled := applyAt(t, restampPage, 500, 800, "", "", "")
	display := layRetained(t, styled)

	_, err := layout.RestampTransforms(t.Context(), styled, display, []layout.TransformOverride{{
		ElementID:    "card",
		Transform:    "rotateY(45deg",
		HasTransform: true,
	}}, layout.Options{})

	if !errors.Is(err, layout.ErrInvalidTransformOverride) {
		t.Fatalf("error = %v, want %v", err, layout.ErrInvalidTransformOverride)
	}
}
