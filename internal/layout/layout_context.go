package layout

import (
	"context"

	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// rasterContext is the engine context for SVG work. Public Layout stores
// context.Background when the caller did not supply one.
func (e *engine) rasterContext() context.Context {
	if e != nil && e.ctx != nil {
		return e.ctx
	}

	return context.Background()
}

// newEngine constructs the layout engine state (extracted from LayoutContext
// for clarity).
func newEngine(
	ctx context.Context, opts Options, faces *pdf.FaceSet, font *pdf.Font,
	styles map[*html.Node]*ResolvedStyle, containers map[*html.Node]sizeContainer, ops []Op,
) *engine {
	return &engine{ //nolint:exhaustruct // intentional zero fields
		opts:       opts,
		ctx:        ctx,
		font:       font,
		faces:      faces,
		registry:   opts.Registry,
		styles:     styles,
		scale:      zoomScale(opts.Zoom),
		containers: containers,
		ops:        ops,
	}
}

// finalizeResult builds the display list and Result from the constructed
// engine (extracted from LayoutContext for clarity).
func finalizeResult(eng *engine, root *html.Node, opts Options) (*Result, error) {
	eng.seedInitialContainingBlock()
	boxNode := eng.build(root, opts.Width, 0, 0)

	if eng.err != nil {
		return nil, eng.err
	}
	// Merge deferred background/border ops before stamping sticky/fixed flags
	// and CSS transforms (those passes need final op indices).
	eng.finalizeChrome(boxNode)

	boxes := make([]*box, 0)
	flattenBoxes(boxNode, &boxes)

	res := &Result{ //nolint:exhaustruct // intentional zero fields
		Ops:            eng.ops,
		Width:          opts.Width,
		Height:         opts.Height,
		pageSnapHeight: eng.pageSnapHeight,
		root:           boxNode,
		boxes:          boxes,
	}
	if boxNode != nil {
		res.Height = boxNode.y + boxNode.height
	}

	if res.Height < eng.height {
		res.Height = eng.height
	}
	// Paint-time CSS transforms/opacity: stamp composed CTMs after geometry
	// is final so transform-origin % resolves against the border box.
	// Skip the full tree walk when no element had transform/opacity.
	if eng.needsXformStamp {
		propagatePerspective(boxNode)
		stampBoxTransforms(boxNode, IdentityMatrix(), res.Ops, opts.Media)
	}

	res.MaxContentX, res.HasFragmentLinks = censusOps(res.Ops, opts.Width)

	return res, nil
}
