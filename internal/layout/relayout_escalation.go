package layout

import (
	"context"
	"errors"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// relayoutBuiltTarget is one winning escalation attempt: the target whose
// subtree kept its size, with the styles and the engine that built it and the
// box they produced.
type relayoutBuiltTarget struct {
	target *relayoutTarget
	styles map[*html.Node]*ResolvedStyle
	eng    *engine
	box    *box
}

// relayoutBuildEscalating builds target's subtree, widening the patched range
// to the holder while the build reports a size change. Each attempt builds
// the range from the kept placement, so nothing is committed until an
// ancestor keeps its size; the rest of the page then keeps its boxes and its
// operations, and only the winning range is re-laid out.
//
// A wider range the splice cannot place ends the escalation with the size
// change the host must handle: the original errSubtreeResized is what a full
// layout resolves. Anything else (context, engine) is reported as is, and a
// first attempt that never resized keeps its own error.
//
//nolint:cyclop // the retry loop repeats the same guarded build for a wider target
func relayoutBuildEscalating(
	ctx context.Context, prev *Result, target *relayoutTarget, opts Options,
) (*relayoutBuiltTarget, error) {
	var firstResize error

	for {
		if target == nil || target.box == nil || target.box.node == nil {
			if firstResize != nil {
				return nil, firstResize
			}

			return nil, errElementNotPlaced
		}

		if err := checkRelayoutTarget(prev, target); err != nil {
			return nil, escalatingError(firstResize, err)
		}

		styles, err := relayoutSubtreeStyles(ctx, prev, target, target.box.node, opts)
		if err != nil {
			return nil, escalatingError(firstResize, err)
		}

		eng, err := relayoutEngine(ctx, prev, opts, styles)
		if err != nil {
			return nil, err
		}

		//nolint:contextcheck // the engine keeps the context and reads it at its recursion boundaries
		newBox, err := relayoutBuild(eng, prev, target, target.box.node, opts)
		if err == nil {
			return &relayoutBuiltTarget{target: target, styles: styles, eng: eng, box: newBox}, nil
		}

		if !errors.Is(err, errSubtreeResized) {
			return nil, err
		}

		if firstResize == nil {
			firstResize = err
		}

		// A moved grid refuses instead of widening: the table check compares
		// the whole grid, and a wider range would re-place the table without
		// that check, so the size change ends here.
		if target.box != nil && target.box.kind == boxKindTable {
			return nil, firstResize
		}

		target = relayoutWiderTarget(target)
		if target == nil {
			return nil, firstResize
		}
	}
}

// escalatingError picks the size-change error when a wider range inherits the
// refusal, and the fresh error otherwise.
func escalatingError(firstResize, err error) error {
	if firstResize != nil && isRelayoutStructural(err) {
		return firstResize
	}

	return err
}

// isRelayoutStructural reports whether err refuses a splice range for a
// reason a wider range inherits: the range cannot be placed, so the
// escalation ends with the size change instead of the wider refusal.
func isRelayoutStructural(err error) bool {
	return errors.Is(err, errSubtreeNotSplicable) || errors.Is(err, errElementNotPlaced)
}

// relayoutWiderTarget returns the patch target one level above target: the
// box that holds it becomes the patched box. It returns nil when target is
// already the outermost box, so the escalation ends with the size error.
func relayoutWiderTarget(target *relayoutTarget) *relayoutTarget {
	if target == nil || len(target.ancestors) == 0 {
		return nil
	}

	ancestors := target.ancestors
	grand := ancestors[:len(ancestors)-1]

	var holder *box

	if len(grand) > 0 {
		holder = grand[len(grand)-1]
	}

	return &relayoutTarget{box: ancestors[len(ancestors)-1], parent: holder, ancestors: grand}
}
