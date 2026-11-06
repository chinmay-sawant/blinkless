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
// partial patch is adopted when a later id fails. Adopt the returned
// display only when the error is nil and treat prev as stale, the same
// rule RelayoutElement documents.
//
// The per-element subtree placements stay sequential. Each splice mutates
// the shared placement the next splice reads, so the ids cannot run in
// parallel. The win of the single call is the single shared styled doc and
// the single display build: the cascade is resolved once per subtree from
// the same document, the owners and order move incrementally, and the
// display materializes once, with no per-call host round trip.
func RelayoutElements(
	ctx context.Context, doc *css.Document, prev *Display, elementIDs []string, options Options,
) (*Display, error) {
	if len(elementIDs) == 0 {
		return prev, nil
	}

	if ctx == nil {
		return nil, ErrNilContext
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("layout: context: %w", err)
	}

	if doc == nil {
		return nil, ErrNilDocument
	}

	if prev == nil {
		return nil, ErrNilDisplay
	}

	styled, ok := pubstate.StyledOf(doc)
	if !ok {
		return nil, ErrNilDocument
	}

	if prev.result == nil {
		return nil, ErrNoRetainedState
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
		if errors.Is(err, ilayout.ErrBatchUnknownID) {
			return nil, ErrNoElement
		}

		return nil, err
	}

	// The batch ran one engine pass per id, the count the per-element
	// options build used to report. The font skips count the patched
	// subtrees that shape no text.
	atomic.AddInt64(&relayoutEngineCalls, int64(len(elementIDs)))
	atomic.AddInt64(&relayoutFontSkips, int64(batch.Textless))

	// One display build materializes the patched placement; the owners and
	// the order it derives equal the maintained ones by construction, so the
	// adopted display carries the incremental slices.
	display := patchedDisplayOn(styled, prev, clone)
	display.OpOwners = batch.Owners
	display.Order = batch.Order

	return display, nil
}
