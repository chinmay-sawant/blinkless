package layout

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/chinmay-sawant/blinkless/css"
	ilayout "github.com/chinmay-sawant/blinkless/internal/layout"
	"github.com/chinmay-sawant/blinkless/internal/pubstate"
)

// RelayoutElements lays out several elements inside prev in one call and
// returns the patched page.
//
// doc is the shared styled document that holds every id, and prev is the
// retained display of that document before the change. Every id is re-laid
// out result-only against the one placement, threading it exactly as
// sequential RelayoutElement calls would: the placement the first id leaves
// is the input of the second id, and so on. The owners and the paint order
// are maintained incrementally across the splices, and one display is built
// at the end, so a batch of hover or selection repaints pays one display
// build instead of one per id. An empty id list returns prev unchanged with
// a nil error.
//
// A missing or refused id aborts the whole batch with the same error the
// matching RelayoutElement call would return, and prev is left untouched:
// the batch works on a private copy of the retained placement, so no
// partial patch is adopted when a later id fails. A resizing id widens to
// the nearest size-stable ancestor like RelayoutElement. Adopt the returned
// display only when the error is nil and treat prev as stale, the same
// rule RelayoutElement documents.
//
// The per-element subtree placements stay sequential. Each splice mutates
// the shared placement the next splice reads, so the ids cannot run in
// parallel. The win of the single call is the single shared styled doc and
// the single display build: the cascade is resolved once per subtree from
// the same document, the owners and order move incrementally, and the
// display materializes once, with no per-call host round trip.
//
// An id whose only change is the background color of boxes the retained
// placement already holds, such as a hover that moves a card's fill, skips
// layout entirely. The batch resolves the new styles, proves that every
// changed element differs from its kept style in the background color alone,
// that each changed element is a plain block-level box with one solid fill, and
// that the DOM and text below it are unchanged, then rewrites that fill's
// color in the batch's private copy. Any doubt, such as a second change, an
// image, a table, a replaced box, or a child that paints the same color, sends
// the id through the ordinary relayout, so the result is the one the ordinary
// path gives.
func RelayoutElements(
	ctx context.Context, doc *css.Document, prev *Display, elementIDs []string, options Options,
) (*Display, error) {
	if len(elementIDs) == 0 {
		return prev, nil
	}

	styled, err := relayoutElementsReady(ctx, doc, prev)
	if err != nil {
		return nil, err
	}

	clone := ilayout.CloneResult(prev.result)
	if clone == nil {
		return nil, ErrNoRetainedState
	}

	baseOpts := batchBaseOptions(styled, prev, options)
	owners := append([]int(nil), prev.OpOwners...)
	order := append([]int(nil), prev.Order...)

	batch, err := ilayout.RelayoutSubtrees(ctx, clone, elementIDs, baseOpts, owners, order)
	if err != nil {
		clone.Release()

		if errors.Is(err, ilayout.ErrBatchUnknownID) {
			return nil, ErrNoElement
		}

		return nil, fmt.Errorf("layout: relayout subtrees: %w", err)
	}

	// The batch ran one engine pass per id, the count the per-element
	// options build used to report, except for the ids the paint restyle
	// patched without the engine. The font skips count the patched
	// subtrees that shape no text.
	atomic.AddInt64(&relayoutEngineCalls, int64(len(elementIDs)-batch.Restyled))
	atomic.AddInt64(&relayoutFontSkips, int64(batch.Textless))

	// One display build materializes the patched placement; the owners and
	// the order the batch maintained are installed on it, so the build skips
	// both derivations. A paint-only batch carries the previous display's
	// boxes, breaks, keyframes, and size, and the text of the patched elements
	// and their ancestors is refreshed in place.
	display := patchedBatchDisplay(styled, prev, prev.result, clone, batch)
	display.OpOwners = batch.Owners
	display.Order = batch.Order

	if batch.PaintOnly {
		applyPaintOnlyBoxTexts(display.Boxes, batch.Texts)
	}

	return display, nil
}

// relayoutElementsReady validates the batch inputs and returns the styled
// document the batch re-resolves from.
func relayoutElementsReady(ctx context.Context, doc *css.Document, prev *Display) (pubstate.Styled, error) {
	var none pubstate.Styled

	if ctx == nil {
		return none, ErrNilContext
	}

	if err := ctx.Err(); err != nil {
		return none, fmt.Errorf("layout: context: %w", err)
	}

	if doc == nil {
		return none, ErrNilDocument
	}

	if prev == nil {
		return none, ErrNilDisplay
	}

	styled, ok := pubstate.StyledOf(doc)
	if !ok {
		return none, ErrNilDocument
	}

	if prev.result == nil {
		return none, ErrNoRetainedState
	}

	return styled, nil
}

// applyPaintOnlyBoxTexts refreshes the boxes a paint-only batch patched.
// Display.Box.Text is the element's subtree text, so a text edit changes the
// box of the patched element and of every ancestor that contains it. The
// boxes are the previous display's slice, which the API declares stale once
// the patched display is adopted, so they are updated in place: a fresh []Box
// would rebuild the whole list the paint-only path avoids.
func applyPaintOnlyBoxTexts(boxes []Box, texts []ilayout.PaintOnlyText) {
	for index := range texts {
		boxIndex := boxIndexByID(boxes, texts[index].ID)
		if boxIndex < 0 || len(texts[index].Texts) == 0 {
			continue
		}

		chain := texts[index].Texts
		boxes[boxIndex].Text = chain[0]

		parent := boxes[boxIndex].ParentIndex
		for step := 1; step < len(chain); step++ {
			if parent < 0 || parent >= len(boxes) {
				break
			}

			boxes[parent].Text = chain[step]
			parent = boxes[parent].ParentIndex
		}
	}
}
