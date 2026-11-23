package layout

import (
	"context"
	"math"
	"sync/atomic"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// Paint restyle is the patch that skips layout. A hover or a class edit that
// only moves the background color of boxes the engine already placed has
// nothing to re-lay out: every box keeps its geometry, and the one fill the
// background emitted keeps its place in the operation list. The batch then
// rewrites that fill's color in its private copy and keeps the retained
// styles in step, instead of running the subtree through the engine and
// splicing the result back.
//
// The attempt is a proof, not a guess. It resolves the new subtree styles
// the way the slow path does, then requires all of these:
//
//   - the subtree holds the same elements, in the same order, and the same
//     text as the kept one;
//   - every changed element differs from its kept style in BGColor and in no
//     other field (the compare covers the whole struct, so a layout field
//     added later is covered without touching an allow list);
//   - each changed element is a plain block-level box whose background is one
//     solid, unclipped, unshaped fill, and the kept window holds exactly one
//     operation of the old color, that fill over the border box;
//   - the subtree uses no image, table, or replaced box, whose output the
//     page-level image resolver or the table passes could change.
//
// Any doubt returns false before prev changes, and the caller runs the
// ordinary build, which is always correct. Tests compare every case against
// the forced slow path and against a full layout.

// Counters for the paint restyle. They are process-wide and only meant for
// tests and diagnostics. TestConfig.PaintRestyleDisabled turns the fast path
// off per run.
var (
	paintRestyleHits   atomic.Uint64
	paintRestyleMisses atomic.Uint64
)

const (
	htmlButton   = "button"
	htmlSelect   = "select"
	htmlTextarea = "textarea"
	htmlProgress = "progress"
	// fillTolerance is the slack, in points, when a fill is matched to a box.
	fillTolerance = 1e-6
)

// restyleChange is one element whose background color moved: the kept box,
// the new used color, the new style, and the one fill to rewrite.
type restyleChange struct {
	node    *html.Node
	box     *box
	newRGBA [4]float64
	next    *ResolvedStyle
	opIndex int
}

// tryPaintRestyle takes the paint-only fast path for one batch id. It returns
// the new subtree styles whenever it resolved them, so a declined attempt does
// not make the slow path resolve them twice, and whether it committed.
func tryPaintRestyle(
	ctx context.Context, prev *Result, target *relayoutTarget, node *html.Node,
	opts Options, out *SubtreeBatchResult,
) (map[*html.Node]*ResolvedStyle, bool) {
	if testConfigFrom(ctx).paintRestyleDisabled() {
		return nil, false
	}

	start, count, found := paintRestyleWindow(prev, target, node, opts)
	if !found {
		paintRestyleMisses.Add(1)

		return nil, false
	}

	styles, err := relayoutSubtreeStyles(ctx, prev, target, node, opts)
	if err != nil {
		paintRestyleMisses.Add(1)

		return nil, false
	}

	changes, planned := planPaintRestyle(prev, target, opts, styles, start, count)
	if !planned {
		paintRestyleMisses.Add(1)

		return styles, false
	}

	commitPaintRestyle(prev, target, changes, start, count, out)
	paintRestyleHits.Add(1)

	return styles, true
}

// paintRestyleWindow checks the inputs that do not need the new styles: the
// target is an element box the retained state can patch, the viewport is the
// one the kept layout used, the DOM below it is the DOM the kept boxes and
// text describe, and the kept window holds operations.
func paintRestyleWindow(prev *Result, target *relayoutTarget, node *html.Node, opts Options) (int, int, bool) {
	if !paintRestyleTargetOK(prev, target, node, opts) {
		return 0, 0, false
	}

	start, count, err := relayoutWindow(prev, target)
	if err != nil || count == 0 {
		return 0, 0, false
	}

	if !paintRestyleSameContent(target.box, node, prev.Ops, start, count) {
		return 0, 0, false
	}

	return start, count, true
}

// paintRestyleTargetOK checks the target element and the kept state around it.
func paintRestyleTargetOK(prev *Result, target *relayoutTarget, node *html.Node, opts Options) bool {
	if prev == nil || prev.styles == nil || target == nil || target.box == nil || node == nil {
		return false
	}

	if node.Type != html.ElementNode || target.box.node != node || target.box.style == nil {
		return false
	}

	return relayoutNear(opts.Width, prev.Width) && checkRelayoutTarget(prev, target) == nil
}

// paintRestyleSameContent reports whether the DOM below node is the content
// the kept boxes and operations describe: the same elements in the same
// order, the same text, and no language attribute the shaper could read. The
// kept boxes hold the DOM's own node pointers, so a pointer compare of the two
// preorder lists catches an added, removed, or moved element. The boxes the
// engine invents (an anonymous flex item around loose text) carry nodes that
// are not in the DOM and are skipped.
func paintRestyleSameContent(kept *box, root *html.Node, ops []Op, start, count int) bool {
	elems := domElements(root)

	for _, elem := range elems {
		if elem.Attribute("lang") != "" || elem.Attribute("xml:lang") != "" {
			return false
		}
	}

	inDOM := make(map[*html.Node]bool, len(elems))
	for _, elem := range elems {
		inDOM[elem] = true
	}

	next := 0

	if !matchKeptElements(kept, elems, inDOM, &next) || next != len(elems) {
		return false
	}

	return subtreeText(root) == opsText(ops, start, count)
}

// domElements lists the element nodes of root's subtree in document order.
func domElements(root *html.Node) []*html.Node {
	var elems []*html.Node

	var walk func(node *html.Node)
	walk = func(node *html.Node) {
		if node == nil {
			return
		}

		if node.Type == html.ElementNode {
			elems = append(elems, node)
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(root)

	return elems
}

// matchKeptElements walks the kept boxes in order and checks that the boxes
// whose node is a DOM element visit the DOM list one for one.
func matchKeptElements(boxNode *box, elems []*html.Node, inDOM map[*html.Node]bool, next *int) bool {
	if boxNode == nil {
		return true
	}

	if boxNode.node != nil && inDOM[boxNode.node] {
		if *next >= len(elems) || elems[*next] != boxNode.node {
			return false
		}

		*next++
	}

	for _, child := range boxNode.children {
		if !matchKeptElements(child, elems, inDOM, next) {
			return false
		}
	}

	return true
}

// planPaintRestyle compares the new styles with the kept ones and returns one
// change per element whose background color moved. It returns false when any
// element differs in another field, when nothing changed (the slow path then
// decides), or when a change cannot be proven to be one solid fill.
func planPaintRestyle(
	prev *Result, target *relayoutTarget, opts Options,
	styles map[*html.Node]*ResolvedStyle, start, count int,
) ([]restyleChange, bool) {
	if !paintRestyleBoxesPlain(target.box) {
		return nil, false
	}

	ancestors := make(map[*html.Node]bool, len(target.ancestors))
	for _, ancestor := range target.ancestors {
		ancestors[ancestor.node] = true
	}

	forced := forcedColorsActive() || opts.ForcedColorsActive

	changes, planned := collectBackgroundChanges(prev, target, styles, ancestors, forced, start, count)
	if !planned || len(changes) == 0 || !paintRestyleImageSafe(opts, styles, ancestors) {
		return nil, false
	}

	return changes, true
}

// collectBackgroundChanges classifies every element of the new styles and
// returns the proven background changes. It returns false at the first element
// that differs in another way or whose change cannot be proven.
func collectBackgroundChanges(
	prev *Result, target *relayoutTarget, styles map[*html.Node]*ResolvedStyle, ancestors map[*html.Node]bool,
	forced bool, start, count int,
) ([]restyleChange, bool) {
	changes := make([]restyleChange, 0, 1)
	taken := make(map[int]bool)

	for node, next := range styles {
		if node == nil || next == nil || node.Type != html.ElementNode || ancestors[node] {
			continue
		}

		kept := prev.styles[node]
		if kept == nil {
			return nil, false
		}

		change, moved, planned := planElementChange(prev, target, node, kept, next, forced, start, count, taken)
		if !planned {
			return nil, false
		}

		if moved {
			taken[change.opIndex] = true

			changes = append(changes, change)
		}
	}

	return changes, true
}

// planElementChange classifies one element: unchanged, a background color
// move it can prove, or something else (planned false).
func planElementChange(
	prev *Result, target *relayoutTarget, node *html.Node, kept, next *ResolvedStyle,
	forced bool, start, count int, taken map[int]bool,
) (restyleChange, bool, bool) {
	var none restyleChange

	if kept == next {
		return none, false, true
	}

	colorMoved, otherwiseSame := paintOnlyBGDelta(kept, next)
	if !otherwiseSame {
		return none, false, false
	}

	if !colorMoved {
		return none, false, true
	}

	change, planned := planBackgroundChange(prev, target, node, kept, next, forced, start, count, taken)

	return change, true, planned
}

// paintOnlyBGDelta compares two used styles of one element. The first result
// reports that BGColor differs; the second is false when they differ in any
// field but BGColor. The generated comparator covers every ResolvedStyle field
// without copying either struct or calling reflect.DeepEqual.
func paintOnlyBGDelta(kept, next *ResolvedStyle) (bool, bool) {
	return kept.BGColor != next.BGColor, styleInternEqualExceptBG(kept, next)
}

// planBackgroundChange proves that one element's background is a single solid
// fill and finds that fill in the kept window.
func planBackgroundChange(
	prev *Result, target *relayoutTarget, node *html.Node, kept, next *ResolvedStyle,
	forced bool, start, count int, taken map[int]bool,
) (restyleChange, bool) {
	var none restyleChange

	if !paintRestyleNodeOK(node) || !solidBackground(kept) || !solidBackground(next) {
		return none, false
	}

	// A flex item keeps an adjusted copy of its cascade style (the container
	// writes the used sizes into it), so the box style is the kept one or a
	// copy whose background is still the kept background.
	boxNode := findBoxByNode(target.box, node)
	if boxNode == nil || boxNode.kind != boxKindBlock || boxNode.style == nil ||
		boxNode.style.BGColor != kept.BGColor {
		return none, false
	}

	index, found := findBackgroundFill(prev.Ops, start, count, boxNode, usedBGForPaintActive(kept, forced))
	if !found || taken[index] {
		return none, false
	}

	return restyleChange{
		node: node, box: boxNode, newRGBA: usedBGForPaintActive(next, forced), next: next, opIndex: index,
	}, true
}

// paintRestyleNodeOK rejects the elements whose background reaches beyond
// their own box: the root and body feed the canvas fill.
func paintRestyleNodeOK(node *html.Node) bool {
	return node.Name != htmlRootName && node.Name != htmlBodyName && node.Name != "head"
}

// solidBackground reports whether style paints its background as exactly one
// plain color fill: a visible color, no image layers, no blend, no clip to a
// smaller box or to text, no corner or border shape, and a block-level
// display the chrome pass paints the ordinary way.
func solidBackground(style *ResolvedStyle) bool {
	if style.BGColor[3] <= 0 || style.background().BackgroundImage != "" {
		return false
	}

	if mode := style.background().BackgroundBlendMode; mode != "" && mode != blendNormal {
		return false
	}

	if clip := style.background().BackgroundClip; clip != "" && !asciiFoldEqual(trimSpaceASCII(clip), "border-box") {
		return false
	}

	if !plainCorners(style) {
		return false
	}

	switch style.Display {
	case displayBlock, displayFlex, displayGrid, displayFlowRoot:
		return true
	}

	return false
}

// plainCorners reports whether every corner keeps the initial round shape and
// no border shape is declared, so the fill paints as an ordinary rounded rect.
func plainCorners(style *ResolvedStyle) bool {
	for _, keyword := range [...]string{
		style.borders().CornerShapeTopLeftKeyword, style.borders().CornerShapeTopRightKeyword,
		style.borders().CornerShapeBottomRightKeyword, style.borders().CornerShapeBottomLeftKeyword,
	} {
		if keyword != "" && keyword != "round" {
			return false
		}
	}

	return style.Declared == nil || style.Declared.BorderShape == ""
}

// findBackgroundFill returns the index of the one fill operation that is the
// box's background: a rectangle fill with the used color and the border box.
// It returns false unless the window holds exactly one operation of that
// color, so a child that paints the same color, or any other consumer of the
// color the engine may add later, sends the attempt to the slow path.
func findBackgroundFill(ops []Op, start, count int, boxNode *box, rgba [4]float64) (int, bool) {
	found := -1

	for index := start; index < start+count && index < len(ops); index++ {
		op := &ops[index]
		if op.R != rgba[0] || op.G != rgba[1] || op.B != rgba[2] || op.Alpha != rgba[3] {
			continue
		}

		if found >= 0 || op.Kind != OpFillRect || !fillMatchesBox(op, boxNode) {
			return 0, false
		}

		found = index
	}

	return found, found >= 0
}

// fillMatchesBox reports whether a fill covers exactly the box's border box.
func fillMatchesBox(op *Op, boxNode *box) bool {
	return math.Abs(op.X-boxNode.x) <= fillTolerance && math.Abs(op.Y-boxNode.y) <= fillTolerance &&
		math.Abs(op.W-boxNode.w) <= fillTolerance && math.Abs(op.H-boxNode.height) <= fillTolerance
}

// paintRestyleBoxesPlain reports whether every box of the subtree is a plain
// block box. Tables, cells, and replaced boxes carry output the restyle does
// not model, so they keep the full build.
func paintRestyleBoxesPlain(root *box) bool {
	if root == nil {
		return true
	}

	if root.kind != boxKindBlock || len(root.rows) > 0 {
		return false
	}

	for _, child := range root.children {
		if !paintRestyleBoxesPlain(child) {
			return false
		}
	}

	return true
}

// paintRestyleImageSafe reports whether the subtree can skip the image
// resolver: with a resolver installed, no element of the subtree names an
// image, so its output cannot depend on what the resolver returns now.
func paintRestyleImageSafe(opts Options, styles map[*html.Node]*ResolvedStyle, ancestors map[*html.Node]bool) bool {
	if opts.Images == nil {
		return true
	}

	for node, style := range styles {
		if style == nil || ancestors[node] {
			continue
		}

		if node.Type == html.ElementNode && imageTag(node.Name) || styleNamesImage(style) {
			return false
		}
	}

	return true
}

// styleNamesImage reports whether any image-valued property of style is set.
func styleNamesImage(style *ResolvedStyle) bool {
	for _, value := range [...]string{
		style.background().BackgroundImage, style.borders().BorderImageSource, style.content().ListStyleImage, style.effects().MaskImage, style.effects().MaskBorderSource,
	} {
		if value != "" && value != "none" {
			return true
		}
	}

	return false
}

// imageTag reports whether an element name draws from the image resolver or
// from a widget the restyle does not model.
func imageTag(name string) bool {
	switch name {
	case "img", "svg", "picture", "video", "canvas", "object", "embed", "iframe", htmlInput, htmlButton,
		htmlSelect, htmlTextarea, htmlMeter, htmlProgress:
		return true
	}

	return false
}

// commitPaintRestyle applies a proven plan to prev, the batch's private
// clone: it rewrites each fill's color, points each changed box and the
// retained style map at the new style, and records the unchanged window the
// way a paint-only splice does.
func commitPaintRestyle(
	prev *Result, target *relayoutTarget, changes []restyleChange, start, count int, out *SubtreeBatchResult,
) {
	oldDamage, hasOld := spliceDamage(prev.Ops, start, count)

	for index := range changes {
		change := &changes[index]
		op := &prev.Ops[change.opIndex]
		op.R, op.G, op.B, op.Alpha = change.newRGBA[0], change.newRGBA[1], change.newRGBA[2], change.newRGBA[3]

		change.box.style = restyledBoxStyle(change.box.style, prev.styles[change.node], change.next)
		prev.styles[change.node] = change.next

		// Text children share their element's style record.
		for _, child := range change.node.Children {
			if _, shared := prev.styles[child]; shared && child.Type == html.TextNode {
				prev.styles[child] = change.next
			}
		}
	}

	if out.SameBoxes {
		elements := countElementBoxes(target.box)
		out.boxPatches = append(out.boxPatches, subtreeBoxPatch{
			baseIdx:   -1,
			oldElem:   elements,
			newElem:   elements,
			box:       target.box,
			ancestors: target.ancestors,
		})
	}

	newDamage, hasNew := spliceDamage(prev.Ops, start, count)
	out.Windows = append(out.Windows, SpliceWindow{
		Start:        start,
		OldCount:     count,
		NewCount:     count,
		OldDamage:    oldDamage,
		NewDamage:    newDamage,
		HasOldDamage: hasOld,
		HasNewDamage: hasNew,
		PaintOnly:    true,
	})
}

// restyledBoxStyle returns the style a patched box keeps. A box that holds
// the cascade record takes the new record. A box that holds an adjusted copy
// (a flex item) gets a fresh copy with the new background, because the old
// copy may be shared through the flex override cache and must not change.
func restyledBoxStyle(boxStyle, kept, next *ResolvedStyle) *ResolvedStyle {
	if boxStyle == kept {
		return next
	}

	adjusted := *boxStyle
	adjusted.BGColor = next.BGColor

	return &adjusted
}

// PaintRestyleStats reports how many batch ids took the paint restyle fast
// path and how many it declined. The counters are process-wide and only
// meant for tests and diagnostics.
type PaintRestyleStats struct {
	Hits   uint64
	Misses uint64
}

// PaintRestyleStatsSnapshot returns the current paint restyle counters.
func PaintRestyleStatsSnapshot() PaintRestyleStats {
	return PaintRestyleStats{Hits: paintRestyleHits.Load(), Misses: paintRestyleMisses.Load()}
}

// ResetPaintRestyleStats zeroes the paint restyle counters.
func ResetPaintRestyleStats() {
	paintRestyleHits.Store(0)
	paintRestyleMisses.Store(0)
}
