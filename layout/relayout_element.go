package layout

import (
	"context"
	"errors"
	"fmt"
	"image"
	"math"

	"github.com/chinmay-sawant/blinkless/css"
	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
	ihtml "github.com/chinmay-sawant/blinkless/internal/html"
	ilayout "github.com/chinmay-sawant/blinkless/internal/layout"
	"github.com/chinmay-sawant/blinkless/internal/pubstate"
)

var (
	// ErrNilDisplay means RelayoutElement was given no previous display
	// list to patch.
	ErrNilDisplay = errors.New("layout: nil display")

	// ErrNoElement means RelayoutElement found no element with the given id.
	ErrNoElement = errors.New("layout: no element with that id")

	// ErrNoRetainedState means the display carries no retained layout
	// state, so it cannot be patched. The display was laid out without
	// Options.RetainState, so the engine kept no box tree and no resolved
	// styles to re-place the element inside. Lay the page out with
	// Options.RetainState set and patch that display.
	ErrNoRetainedState = errors.New("layout: display carries no retained layout state")
)

// ElementPatch is the result of patching one element of a laid out page.
type ElementPatch struct {
	// Ops are the operations the patched element's subtree emits. A host
	// replays them over the region it repaints.
	Ops []DisplayOp

	// OpOwners maps each operation index in Ops to the nearest element box
	// in Boxes that owns it, the rule Display.OpOwners uses.
	OpOwners []int

	// Boxes are the element border boxes of the patched subtree, in document
	// order, in CSS pixels.
	Boxes []Box

	// Old is the element's border box before the patch, in CSS pixels.
	Old image.Rectangle

	// New is the element's border box after the patch, in CSS pixels.
	New image.Rectangle

	// Display is the patched page: the operations, the paint order, the
	// owners and the boxes of the whole page after the patch. A host that
	// keeps its own operation list adopts this one, because a patch that
	// changes the operation count moves every later operation index and
	// with it the paint order and the owners.
	Display *Display
}

// RelayoutElement lays out one element's subtree inside prev and returns the
// operations, boxes and old and new bounds for that element only.
//
// doc is the styled document that holds the element and prev is the display
// list of that document before the change. The element is found by id in
// doc's tree, so a host that edited the tree first (html.SetText,
// html.SetAttribute) gets a patch of the edited element.
//
// prev must carry the retained layout state. The host asks for it when it
// lays the page out: layout.DisplayListOptions with Options.RetainState set.
// A display laid out without that state cannot be patched, and
// RelayoutElement returns ErrNoRetainedState for it.
//
// The patch holds the element's own operations and boxes, not the whole
// page: Ops carries the operations of the element's subtree, Boxes the
// element boxes of that subtree, and OpOwners maps each operation to the
// box in Boxes that owns it. Old is where the element sat in prev, New is
// where it sits after the patch. A host repaints the union of the two
// rectangles.
func RelayoutElement(
	ctx context.Context, doc *css.Document, prev *Display, elementID string, options Options,
) (*ElementPatch, error) {
	if err := checkRelayoutInput(ctx, doc, prev, elementID); err != nil {
		return nil, err
	}

	styled, ok := pubstate.StyledOf(doc)
	if !ok {
		return nil, ErrNilDocument
	}

	node := elementByID(styled.Root, elementID)
	if node == nil {
		return nil, ErrNoElement
	}

	if prev.result == nil {
		return nil, ErrNoRetainedState
	}

	res, err := subtreeResult(ctx, styled, prev, node, options)
	if err != nil {
		return nil, err
	}

	boxes := boxesFrom(ilayout.PlacedElements(res))

	return &ElementPatch{
		Ops:      res.Ops,
		OpOwners: ilayout.OperationOwners(res),
		Boxes:    boxes,
		Old:      elementBounds(prev.Boxes, elementID),
		New:      elementBounds(boxes, elementID),
		Display:  patchedDisplay(styled, prev),
	}, nil
}

// patchedDisplay builds the display list of the patched page. The engine
// patched the placement the previous display was built from, so the display
// carries that placement and the operations, the paint order, the owners and
// the boxes that go with them.
func patchedDisplay(styled pubstate.Styled, prev *Display) *Display {
	canvasPt := float64(prev.Height) / ptToPx

	return newDisplay(prev.result, styled.Sheets, canvasPt,
		prev.ForcedBreaks, prev.KeepTogether, prev.zoom, true)
}

// checkRelayoutInput rejects the inputs RelayoutElement cannot work with.
func checkRelayoutInput(ctx context.Context, doc *css.Document, prev *Display, elementID string) error {
	if ctx == nil {
		return ErrNilContext
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("layout: context: %w", err)
	}

	if doc == nil {
		return ErrNilDocument
	}

	if prev == nil {
		return ErrNilDisplay
	}

	if elementID == "" {
		return ErrNoElement
	}

	return nil
}

// subtreeResult asks the engine to place the element's subtree inside the
// layout the previous display list was built from.
func subtreeResult(
	ctx context.Context,
	styled pubstate.Styled,
	prev *Display,
	node *ihtml.Node,
	options Options,
) (*ilayout.Result, error) {
	font, err := pdf.DefaultFont()
	if err != nil {
		return nil, fmt.Errorf("layout: default font: %w", err)
	}

	widthPt, heightPt := displayViewportPt(styled)
	layoutOpts := displayLayoutOptions(styled, font, options.Images, widthPt, heightPt)
	layoutOpts.Zoom = prev.zoom

	res, err := ilayout.RelayoutSubtree(ctx, prev.result, node, layoutOpts)
	if err != nil {
		return nil, fmt.Errorf("layout: relayout element: %w", err)
	}

	return res, nil
}

// elementByID returns the element node with the given id in the tree, or nil.
func elementByID(root *ihtml.Node, elementID string) *ihtml.Node {
	if root == nil || elementID == "" {
		return nil
	}

	return root.FindFirst(func(candidate *ihtml.Node) bool {
		return candidate.Type == ihtml.ElementNode && candidate.Attribute("id") == elementID
	})
}

// elementBounds returns the border box of the element with the given id, in
// CSS pixels. A missing element gives the empty rectangle.
func elementBounds(boxes []Box, elementID string) image.Rectangle {
	for i := range boxes {
		if boxes[i].ID == elementID {
			return boxBounds(boxes[i])
		}
	}

	return image.Rectangle{} //nolint:exhaustruct // the zero rectangle marks a missing element
}

// boxBounds converts one element box to a rectangle in CSS pixels.
func boxBounds(box Box) image.Rectangle {
	left := int(math.Round(box.X))
	top := int(math.Round(box.Y))

	return image.Rect(left, top, left+int(math.Round(box.W)), top+int(math.Round(box.H)))
}
