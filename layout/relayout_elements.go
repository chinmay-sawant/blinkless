package layout

import (
	"context"

	"github.com/chinmay-sawant/blinkless/css"
	ilayout "github.com/chinmay-sawant/blinkless/internal/layout"
)

// RelayoutElements lays out several elements inside prev in one call and
// returns the patched page.
//
// doc is the shared styled document that holds every id, and prev is the
// retained display of that document before the change. Each id is re-laid
// out against that one styled document, threading the display exactly as
// sequential RelayoutElement calls would: the display the first id returns
// is the input of the second id, and so on. An empty id list returns prev
// unchanged with a nil error.
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
// parallel. The win of the single call is the single shared styled doc:
// the cascade is resolved once per subtree from the same document, with no
// re-application of the stylesheets and no per-call host round trip, only
// the subtree work each id would do on its own.
func RelayoutElements(
	ctx context.Context, doc *css.Document, prev *Display, elementIDs []string, options Options,
) (*Display, error) {
	if len(elementIDs) == 0 {
		return prev, nil
	}

	if prev == nil {
		return nil, ErrNilDisplay
	}

	if prev.result == nil {
		return nil, ErrNoRetainedState
	}

	clone := ilayout.CloneResult(prev.result)
	if clone == nil {
		return nil, ErrNoRetainedState
	}

	current := &Display{
		Ops:            prev.Ops,
		OpOwners:       prev.OpOwners,
		Order:          prev.Order,
		Boxes:          prev.Boxes,
		Keyframes:      prev.Keyframes,
		Width:          prev.Width,
		Height:         prev.Height,
		PointsPerPixel: prev.PointsPerPixel,
		PixelPerPoint:  prev.PixelPerPoint,
		ForcedBreaks:   prev.ForcedBreaks,
		KeepTogether:   prev.KeepTogether,
		result:         clone,
		zoom:           prev.zoom,
	}

	for _, elementID := range elementIDs {
		patch, err := RelayoutElement(ctx, doc, current, elementID, options)
		if err != nil {
			return nil, err
		}

		current = patch.Display
	}

	return current, nil
}
