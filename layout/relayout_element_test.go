package layout_test

import (
	"errors"
	"testing"

	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

func TestRelayoutElementPatchesTheEditedElement(t *testing.T) {
	t.Parallel()

	tree, err := html.Parse([]byte(`<div id="box" style="width:120px;height:40px">Hello</div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")
	display := layRetained(t, styled)

	if !tree.SetText("box", "Hello world") {
		t.Fatal("SetText reported no element")
	}

	patch, err := layout.RelayoutElement(t.Context(), styled, display, "box", layout.Options{})
	if err != nil {
		t.Fatalf("relayout element: %v", err)
	}

	if len(patch.Ops) == 0 {
		t.Fatal("patch has no operations")
	}

	if got := patchText(patch); got != "Hello world" {
		t.Fatalf("patch text = %q, want %q", got, "Hello world")
	}

	if len(patch.Boxes) == 0 {
		t.Fatal("patch has no boxes")
	}

	if patch.Old.Empty() {
		t.Fatalf("patch old bounds = %v, want the element's old box", patch.Old)
	}

	if patch.New.Empty() {
		t.Fatalf("patch new bounds = %v, want the element's new box", patch.New)
	}
}

func TestRelayoutElementCoversOnlyThatElement(t *testing.T) {
	t.Parallel()

	source := `<div id="one" style="width:120px;height:40px">One</div>` +
		`<div id="two" style="width:120px;height:40px">Two</div>`

	tree, err := html.Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")
	display := layRetained(t, styled)

	if !tree.SetText("two", "Two again") {
		t.Fatal("SetText reported no element")
	}

	patch, err := layout.RelayoutElement(t.Context(), styled, display, "two", layout.Options{})
	if err != nil {
		t.Fatalf("relayout element: %v", err)
	}

	if len(patch.Ops) >= len(display.Ops) {
		t.Fatalf("patch ops = %d, want fewer than the %d of the whole page",
			len(patch.Ops), len(display.Ops))
	}

	if got := patchText(patch); got != "Two again" {
		t.Fatalf("patch text = %q, want %q", got, "Two again")
	}
}

func TestRelayoutElementWithoutAnElement(t *testing.T) {
	t.Parallel()

	tree, err := html.Parse([]byte(`<div id="box">Hello</div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")

	// The display carries no retained state, and the element is missing
	// too. The missing element is the answer: the id check does not depend
	// on the retained state.
	display := layOf(t, styled)

	_, err = layout.RelayoutElement(t.Context(), styled, display, "other", layout.Options{})
	if !errors.Is(err, layout.ErrNoElement) {
		t.Fatalf("error = %v, want %v", err, layout.ErrNoElement)
	}
}

func TestRelayoutElementWithoutADisplay(t *testing.T) {
	t.Parallel()

	tree, err := html.Parse([]byte(`<div id="box">Hello</div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")

	_, err = layout.RelayoutElement(t.Context(), styled, nil, "box", layout.Options{})
	if !errors.Is(err, layout.ErrNilDisplay) {
		t.Fatalf("error = %v, want %v", err, layout.ErrNilDisplay)
	}
}

// patchText returns the text of the patch's first text operation.
func patchText(patch *layout.ElementPatch) string {
	for i := range patch.Ops {
		if patch.Ops[i].Kind == layout.DisplayOpText {
			return patch.Ops[i].Text
		}
	}

	return ""
}
