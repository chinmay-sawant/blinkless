package layout_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

// svgDisplayImage returns the first non-empty image payload of a display.
func svgDisplayImage(t *testing.T, display *layout.Display) []byte {
	t.Helper()

	for index := range display.Ops {
		if display.Ops[index].Kind != layout.DisplayOpImage {
			continue
		}

		if data, _, _ := display.Ops[index].ImageBytes(); len(data) > 0 {
			return data
		}
	}

	t.Fatal("display has no image operation")

	return nil
}

func svgPaintDisplay(t *testing.T) (*layout.Display, *css.Document) {
	t.Helper()

	tree, err := html.Parse([]byte(`<div>` +
		`<svg id="icon" style="display:block" width="64" height="28" viewBox="0 0 64 28" ` +
		`xmlns="http://www.w3.org/2000/svg">` +
		`<rect x="2" y="2" width="60" height="24"/>` +
		`</svg></div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")

	return layRetained(t, styled), styled
}

// TestRestyleInlineSVGPatchesTheImage pins the block-level path: the SVG is
// laid out as its own box and the patch replaces the image payload.
//
//nolint:cyclop // the fixture checks the boxes, bounds, and payload together
func TestRestyleInlineSVGPatchesTheImage(t *testing.T) {
	t.Parallel()

	display, styled := svgPaintDisplay(t)
	oldData := svgDisplayImage(t, display)

	patch, err := layout.RestyleInlineSVG(t.Context(), styled, display, []layout.SVGPaintOverride{{
		ElementID: "icon",
		Fill:      "#ff0000",
		HasFill:   true,
	}}, layout.Options{})
	if err != nil {
		t.Fatalf("restyle inline svg: %v", err)
	}

	if len(patch.Ops) == 0 {
		t.Fatal("patch has no operations")
	}

	if len(patch.Boxes) == 0 || patch.Boxes[0].ID != "icon" {
		t.Fatalf("patch boxes = %+v, want the icon box", patch.Boxes)
	}

	if patch.Old.Empty() || patch.New.Empty() {
		t.Fatalf("patch bounds old = %v new = %v, want the image rectangle", patch.Old, patch.New)
	}

	if patch.Old != patch.New {
		t.Fatalf("patch bounds old = %v new = %v, want a paint-only change", patch.Old, patch.New)
	}

	if patch.Display == nil || patch.Display == display {
		t.Fatal("patch display is not a new patched page")
	}

	newData := svgDisplayImage(t, patch.Display)
	if bytes.Equal(oldData, newData) {
		t.Fatal("restyled display kept the old image payload")
	}
}

func TestRestyleInlineSVGWithoutOverridesDoesNothing(t *testing.T) {
	t.Parallel()

	display, styled := svgPaintDisplay(t)
	before := svgDisplayImage(t, display)

	patch, err := layout.RestyleInlineSVG(t.Context(), styled, display, nil, layout.Options{})
	if err != nil {
		t.Fatalf("restyle inline svg: %v", err)
	}

	if len(patch.Ops) != 0 || len(patch.Boxes) != 0 || len(patch.OpOwners) != 0 {
		t.Fatalf("patch = %d ops, %d boxes, %d owners, want an empty patch",
			len(patch.Ops), len(patch.Boxes), len(patch.OpOwners))
	}

	if patch.Display != display {
		t.Fatal("empty override returned a different display")
	}

	if after := svgDisplayImage(t, display); !bytes.Equal(before, after) {
		t.Fatal("empty override changed the raster")
	}
}

func TestRestyleInlineSVGRequiresRetainedState(t *testing.T) {
	t.Parallel()

	tree, err := html.Parse([]byte(`<svg id="icon" style="display:block" width="64" height="28">` +
		`<rect x="2" y="2" width="60" height="24"/></svg>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")
	display := layOf(t, styled)

	_, err = layout.RestyleInlineSVG(t.Context(), styled, display, []layout.SVGPaintOverride{{
		ElementID: "icon",
		Fill:      "#ff0000",
		HasFill:   true,
	}}, layout.Options{})
	if !errors.Is(err, layout.ErrNoRetainedState) {
		t.Fatalf("error = %v, want %v", err, layout.ErrNoRetainedState)
	}
}

func TestRestyleInlineSVGRejectsAMissingPart(t *testing.T) {
	t.Parallel()

	display, styled := svgPaintDisplay(t)

	_, err := layout.RestyleInlineSVG(t.Context(), styled, display, []layout.SVGPaintOverride{{
		ElementID: "icon",
		PartID:    "missing",
		Fill:      "#ff0000",
		HasFill:   true,
	}}, layout.Options{})
	if !errors.Is(err, layout.ErrNoSVGPart) {
		t.Fatalf("error = %v, want %v", err, layout.ErrNoSVGPart)
	}
}

// svgInlineDisplay lays out a paragraph that holds one inline-level <svg>, so
// the inline collector flattens it into a replaced item instead of a box of its
// own. The line keeps the SVG box and attributes the raster op to it, which is
// what makes the icon addressable.
func svgInlineDisplay(t *testing.T) (*layout.Display, *css.Document) {
	t.Helper()

	tree, err := html.Parse([]byte(`<p>icon ` +
		`<svg id="icon" width="64" height="28" viewBox="0 0 64 28" xmlns="http://www.w3.org/2000/svg">` +
		`<rect x="2" y="2" width="60" height="24"/>` +
		`</svg> tail</p>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")

	return layRetained(t, styled), styled
}

// TestRestyleInlineSVGInlineIcon restyles an inline-level <svg> inside a
// paragraph, the case that used to return ErrSVGNotLaidOut. A second call with
// the same paint reuses the cached raster and leaves the payload identical.
//
//nolint:cyclop // one fixture proves discovery, the patch, and the cached repeat
func TestRestyleInlineSVGInlineIcon(t *testing.T) {
	t.Parallel()

	display, styled := svgInlineDisplay(t)
	oldData := svgDisplayImage(t, display)

	override := layout.SVGPaintOverride{ElementID: "icon", Fill: "#ff0000", HasFill: true}

	patch, err := layout.RestyleInlineSVG(t.Context(), styled, display,
		[]layout.SVGPaintOverride{override}, layout.Options{})
	if err != nil {
		t.Fatalf("restyle inline svg: %v", err)
	}

	if len(patch.Boxes) == 0 || patch.Boxes[0].ID != "icon" {
		t.Fatalf("patch boxes = %+v, want the icon box", patch.Boxes)
	}

	if patch.Old.Empty() || patch.Old != patch.New {
		t.Fatalf("patch bounds old = %v new = %v, want the image rectangle", patch.Old, patch.New)
	}

	if patch.Display == nil || patch.Display == display {
		t.Fatal("patch display is not a new patched page")
	}

	newData := svgDisplayImage(t, patch.Display)
	if bytes.Equal(oldData, newData) {
		t.Fatal("restyled inline icon kept the old image payload")
	}

	repeat, err := layout.RestyleInlineSVG(t.Context(), styled, patch.Display,
		[]layout.SVGPaintOverride{override}, layout.Options{})
	if err != nil {
		t.Fatalf("restyle inline svg (repeat): %v", err)
	}

	if after := svgDisplayImage(t, repeat.Display); !bytes.Equal(newData, after) {
		t.Fatal("repeat restyle changed the cached raster payload")
	}
}

// TestRestyleInlineSVGInlineWithoutOverridesDoesNothing pins the inline no-op:
// an empty override list keeps the display and the raster.
func TestRestyleInlineSVGInlineWithoutOverridesDoesNothing(t *testing.T) {
	t.Parallel()

	display, styled := svgInlineDisplay(t)
	before := svgDisplayImage(t, display)

	patch, err := layout.RestyleInlineSVG(t.Context(), styled, display, nil, layout.Options{})
	if err != nil {
		t.Fatalf("restyle inline svg: %v", err)
	}

	if len(patch.Ops) != 0 || len(patch.Boxes) != 0 || len(patch.OpOwners) != 0 {
		t.Fatalf("patch = %d ops, %d boxes, %d owners, want an empty patch",
			len(patch.Ops), len(patch.Boxes), len(patch.OpOwners))
	}

	if patch.Display != display {
		t.Fatal("empty override returned a different display")
	}

	if after := svgDisplayImage(t, display); !bytes.Equal(before, after) {
		t.Fatal("empty override changed the inline raster")
	}
}

// TestRestyleInlineSVGRejectsAHiddenSVG keeps the error path: an SVG that
// produced no raster has no retained box to restyle.
func TestRestyleInlineSVGRejectsAHiddenSVG(t *testing.T) {
	t.Parallel()

	tree, err := html.Parse([]byte(`<div>` +
		`<svg id="icon" style="display:none" width="64" height="28" viewBox="0 0 64 28">` +
		`<rect x="2" y="2" width="60" height="24"/></svg></div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")
	display := layRetained(t, styled)

	_, err = layout.RestyleInlineSVG(t.Context(), styled, display, []layout.SVGPaintOverride{{
		ElementID: "icon",
		Fill:      "#ff0000",
		HasFill:   true,
	}}, layout.Options{})
	if !errors.Is(err, layout.ErrSVGNotLaidOut) {
		t.Fatalf("error = %v, want %v", err, layout.ErrSVGNotLaidOut)
	}
}
