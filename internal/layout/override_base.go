package layout

import (
	"maps"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// A layout animation relays out the same containing block every frame with a
// new override value. The override is applied after the cascade, so the base
// styles of that block never change between frames, yet the override pass
// used to run the whole cascade over the block's subtree each time, with a
// fresh selector match cache that never hit. The cost grew with the number of
// siblings in the block.
//
// overrideBase keeps the base styles the last override pass resolved and
// hands them back to the next one when every input that could change the
// cascade is the same:
//
//   - the style inputs (sheets, media, viewport, hover, focus, active, link
//     underline policy) and the rem base are equal;
//   - the target and the parent style it inherits from are the same;
//   - the document has no attribute or text edits since the last pass;
//   - every node of the subtree is present and still resolves to the same
//     style pointer in the retained style map, so any patch that restyled
//     part of the block since (hover scope, paint restyle, SVG restyle)
//     misses instead of reusing a stale style.
//
// The reuse takes the map out of the cache. The pass puts the base styles
// back before it commits, and a pass that fails leaves the cache empty, so
// an error can never leave a half-overridden map behind. The cache lives and
// dies with the Result and is never copied into a clone.
// styleRetainInputs captures every resolveStyles input that can change an
// outcome without touching the tree: the sheet set identity, the media and
// viewport, the stateful pseudo-class ids, and the operator underline
// policy. Zoom is layout scale only and never feeds the cascade.
type styleRetainInputs struct {
	sheetsKey          string
	media              string
	viewportW          float64
	viewportH          float64
	focus              string
	hover              string
	active             string
	printLinkUnderline bool
}

// retainInputsOf folds the style inputs of one resolve into a comparable key.
func retainInputsOf(opts Options) styleRetainInputs {
	return styleRetainInputs{
		sheetsKey:          sheetsCacheKey(opts.Sheets),
		media:              opts.Media,
		viewportW:          opts.Width,
		viewportH:          opts.Height,
		focus:              opts.State.Focus,
		hover:              opts.State.Hover,
		active:             opts.State.Active,
		printLinkUnderline: opts.PrintLinkUnderline,
	}
}

type overrideBase struct {
	inputs          styleRetainInputs
	remBase         float64
	mutationVersion uint64
	root            *html.Node
	parent          *ResolvedStyle
	styles          map[*html.Node]*ResolvedStyle
}

// takeOverrideBase returns the cached base styles for root's subtree and
// empties the cache, or nil when the cache does not prove the inputs equal.
func takeOverrideBase(
	prev *Result, root *html.Node, parent *ResolvedStyle, opts Options,
) map[*html.Node]*ResolvedStyle {
	cache := prev.overrideBase
	prev.overrideBase = nil

	if cache == nil || cache.root != root || cache.parent != parent ||
		cache.inputs != retainInputsOf(opts) || cache.remBase != relayoutRemBase(prev) {
		return nil
	}

	if cache.mutationVersion != overrideMutationVersion(prev) || !subtreeStylesCurrent(root, cache.styles, prev.styles) {
		return nil
	}

	return cache.styles
}

func overrideMutationVersion(prev *Result) uint64 {
	if root := prev.documentRootNode(); root != nil {
		return root.MutationVersion
	}

	return 0
}

// subtreeStylesCurrent reports whether every node under root, root included,
// has a cached style that is the style the retained map still holds.
func subtreeStylesCurrent(root *html.Node, cached, retained map[*html.Node]*ResolvedStyle) bool {
	style, ok := cached[root]
	if !ok || retained[root] != style {
		return false
	}

	for _, child := range root.Children {
		if !subtreeStylesCurrent(child, cached, retained) {
			return false
		}
	}

	return true
}

// keepOverrideBase stores the base styles of a pass that committed. styles
// must hold the base styles again, after the pass restored the overridden
// ones, and the retained map must already carry them. The cache keeps its own
// copy because the result the pass returns holds the pass map.
func keepOverrideBase(
	prev *Result, root *html.Node, parent *ResolvedStyle, opts Options, styles map[*html.Node]*ResolvedStyle,
) {
	prev.overrideBase = &overrideBase{
		inputs:          retainInputsOf(opts),
		remBase:         relayoutRemBase(prev),
		mutationVersion: overrideMutationVersion(prev),
		root:            root,
		parent:          parent,
		styles:          maps.Clone(styles),
	}
}
