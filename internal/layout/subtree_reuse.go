package layout

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"
	"sort"
	"sync/atomic"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// Subtree reuse is the incremental fast path in front of the element patches.
// A placed subtree is fingerprinted from its DOM content, its used styles,
// and its width basis. On a rebuild with a retained previous display, a clean
// subtree reuses its operations verbatim (keeping their IDs), a moved-only
// sibling is shifted by coordinate translation, and any size, content, or
// style mismatch in a dirty subtree falls back to the existing full path,
// which reports errSubtreeResized or ErrOverrideReflow so the host runs a
// full layout. Correctness comes first: the attempt gives up at the slightest
// doubt and never mutates the retained result on a miss.
//
// The single-element entry is RelayoutSubtreeWithOverrides with no overrides
// (it delegates to RelayoutSubtree, so the call is the ordinary element
// patch). The batch entry is the same function with overrides: the override
// allow list is layout only, so an override always changes layout inputs and
// the attempt falls back to the normal override build.
//
//nolint:gochecknoglobals // process-wide reuse counters back ReuseStatsSnapshot
var (
	reuseHitCount        atomic.Uint64
	reuseMissCount       atomic.Uint64
	reuseTranslatedCount atomic.Uint64
)

// ReuseStats reports the subtree reuse counters: fast-path hits, fallbacks
// to the existing path, and windows moved by coordinate translation.
type ReuseStats struct {
	Hits       uint64
	Misses     uint64
	Translated uint64
}

// ReuseStatsSnapshot returns the current reuse counters.
func ReuseStatsSnapshot() ReuseStats {
	return ReuseStats{
		Hits:       reuseHitCount.Load(),
		Misses:     reuseMissCount.Load(),
		Translated: reuseTranslatedCount.Load(),
	}
}

// ResetReuseStats zeroes the reuse counters. Tests use it to isolate one
// patch from the ones before it.
func ResetReuseStats() {
	reuseHitCount.Store(0)
	reuseMissCount.Store(0)
	reuseTranslatedCount.Store(0)
}

// trySubtreeReuse attempts the paint-only fast path for node. It returns true
// with the patched result on a hit. On any doubt it returns false without
// touching prev, and the caller runs the existing build. An override list is
// always a miss: overrides name layout properties, so the inputs moved.
func trySubtreeReuse(
	ctx context.Context,
	prev *Result,
	node *html.Node,
	overrides []StyleOverride,
	opts Options,
) (*Result, bool) {
	if ctx == nil || prev == nil || prev.styles == nil || node == nil {
		return nil, false
	}

	if ctx.Err() != nil {
		return nil, false
	}

	if len(overrides) > 0 {
		reuseMissCount.Add(1)

		return nil, false
	}

	target, start, oldCount, found := subtreeReuseTarget(prev, node, opts)
	if !found {
		reuseMissCount.Add(1)

		return nil, false
	}

	styles, found := subtreeReuseStyles(ctx, prev, target, node, opts, start, oldCount)
	if !found {
		reuseMissCount.Add(1)

		return nil, false
	}

	newOps, newBox, ok := subtreeReuseBuild(ctx, prev, target, node, opts, styles, oldCount)
	if !ok {
		reuseMissCount.Add(1)

		return nil, false
	}

	reuseHitCount.Add(1)

	return subtreeReuseCommit(prev, target, newBox, newOps, start, oldCount, styles), true
}

// subtreeReuseCommit installs the rebuilt subtree into prev and returns the
// patched result.
func subtreeReuseCommit(
	prev *Result,
	target *relayoutTarget,
	newBox *box,
	newOps []Op,
	start, oldCount int,
	styles map[*html.Node]*ResolvedStyle,
) *Result {
	for index := range newOps {
		newOps[index].ID = prev.Ops[start+index].ID
	}

	delta := len(newOps) - oldCount

	relayoutSpliceOps(prev, start, oldCount, newOps)
	relayoutRebaseTree(newBox, start)
	relayoutSwapBox(target, newBox)
	relayoutShiftRanges(prev.boxTreeRoot(), newBox, start, start+oldCount-1, delta)
	unionChildOpRanges(prev.boxTreeRoot())

	relayoutKeepBoxes(prev)
	relayoutKeepStyles(prev, styles)
	prev.MaxContentX, prev.HasFragmentLinks = censusOps(prev.Ops, prev.Width)

	return relayoutSubtreeResult(prev, newBox, start, len(newOps), styles)
}

// subtreeReuseTarget resolves the kept box for node and its retained
// operation window, rejecting an override-sensitive or moved target.
func subtreeReuseTarget(prev *Result, node *html.Node, opts Options) (*relayoutTarget, int, int, bool) {
	target := findRelayoutTarget(prev.boxTreeRoot(), node)
	if target == nil || target.box == nil {
		return nil, 0, 0, false
	}

	if checkRelayoutTarget(prev, target) != nil {
		return nil, 0, 0, false
	}

	if target.box.style == nil {
		return nil, 0, 0, false
	}

	if !relayoutNear(opts.Width, prev.Width) {
		return nil, 0, 0, false
	}

	start, oldCount, err := relayoutWindow(prev, target)
	if err != nil {
		return nil, 0, 0, false
	}

	return target, start, oldCount, true
}

// subtreeReuseStyles resolves the subtree styles and requires both the DOM
// content hash and the layout-input hash to match the retained side.
func subtreeReuseStyles(
	ctx context.Context,
	prev *Result,
	target *relayoutTarget,
	node *html.Node,
	opts Options,
	start, oldCount int,
) (map[*html.Node]*ResolvedStyle, bool) {
	if !keptSubtreeMatchesDOM(target.box, node) {
		return nil, false
	}

	if fingerprintContentDOM(node) != fingerprintKeptContent(target.box, prev.Ops, start, oldCount) {
		return nil, false
	}

	styles, err := relayoutSubtreeStyles(ctx, prev, target, node, opts)
	if err != nil {
		return nil, false
	}

	if fingerprintLayoutSubtree(node, prev.styles) != fingerprintLayoutSubtree(node, styles) {
		return nil, false
	}

	return styles, true
}

// subtreeReuseBuild rebuilds the subtree in an isolated engine and requires the
// operation count to match the retained window.
func subtreeReuseBuild(
	ctx context.Context,
	prev *Result,
	target *relayoutTarget,
	node *html.Node,
	opts Options,
	styles map[*html.Node]*ResolvedStyle,
	oldCount int,
) ([]Op, *box, bool) {
	if basis := widthBasisFor(target.box, opts); !(basis > 0) || !finitePositive(basis) {
		return nil, nil, false
	}

	eng, err := relayoutEngine(ctx, prev, opts, styles)
	if err != nil {
		return nil, nil, false
	}

	//nolint:contextcheck // the engine keeps the context and reads it at its recursion boundaries
	newBox, err := relayoutBuild(eng, prev, target, node, opts)
	if err != nil {
		return nil, nil, false
	}

	if len(eng.ops) != oldCount {
		return nil, nil, false
	}

	return eng.ops, newBox, true
}

// fingerprintLayoutSubtree hashes the layout inputs of root's subtree: every
// element's used style with the paint-only fields cleared, in document order.
// A class change that only repaints keeps the same hash; a width, font, or
// spacing change does not.
func fingerprintLayoutSubtree(root *html.Node, styles map[*html.Node]*ResolvedStyle) uint64 {
	hash := fnv.New64a()

	var digest [8]byte

	var walk func(node *html.Node)
	walk = func(node *html.Node) {
		if node == nil {
			return
		}

		if node.Type == html.ElementNode {
			binary.LittleEndian.PutUint64(digest[:], hashLayoutStyle(styles[node]))
			_, _ = hash.Write(digest[:])
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(root)

	return hash.Sum64()
}

// hashLayoutStyle hashes one used style with the paint-only fields cleared.
// The whole struct is hashed so a future layout field is covered without
// updating an allow list; only BGColor and Color may differ on a hit.
func hashLayoutStyle(style *ResolvedStyle) uint64 {
	if style == nil {
		return 0
	}

	clone := *style
	clone.BGColor = [4]float64{}
	clone.Color = [3]float64{}

	hash := fnv.New64a()
	_, _ = fmt.Fprintf(hash, "%#v", clone)

	return hash.Sum64()
}

// fingerprintContentDOM hashes the DOM shape and text of root's subtree,
// excluding the class attribute, which the style hash carries. Structure and
// text are hashed as separate legs so the kept-side reconstruction below can
// use the same canonical form: element names and non-class attributes in
// document order, then the concatenated text in document order.
func fingerprintContentDOM(root *html.Node) uint64 {
	hash := fnv.New64a()

	var walk func(node *html.Node)
	walk = func(node *html.Node) {
		if node == nil {
			return
		}

		if node.Type == html.ElementNode {
			writeContentElement(hash, node.Type, node.Name, node.Attrs)
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(root)

	_, _ = hash.Write([]byte{0xff})
	_, _ = hash.Write([]byte(subtreeText(root)))

	return hash.Sum64()
}

// fingerprintKeptContent rebuilds the same canonical hash from the retained
// side: the kept box tree carries the pre-edit element shape (class excluded,
// so an in-place class edit does not pollute it) and the retained operations
// carry the pre-edit text. A paint-only edit hashes equal; any content or
// shape edit does not.
func fingerprintKeptContent(kept *box, ops []Op, start, count int) uint64 {
	hash := fnv.New64a()

	var walk func(node *box)
	walk = func(node *box) {
		if node == nil {
			return
		}

		if node.node != nil && node.node.Type == html.ElementNode {
			writeContentElement(hash, node.node.Type, node.node.Name, node.node.Attrs)
		}

		for _, child := range node.children {
			walk(child)
		}

		for _, row := range node.rows {
			for _, cell := range row {
				walk(cell)
			}
		}
	}

	walk(kept)

	_, _ = hash.Write([]byte{0xff})
	_, _ = hash.Write([]byte(opsText(ops, start, count)))

	return hash.Sum64()
}

// writeContentElement writes one element's content identity: its type, its
// name, and its non-class attributes in sorted order. Class is excluded
// because the style hash owns it; a class-only edit must keep the content
// hash stable.
func writeContentElement(hash hash.Hash, nodeType html.NodeType, name string, attrs map[string]string) {
	_, _ = hash.Write([]byte{byte(nodeType)})
	_, _ = hash.Write([]byte(name))
	_, _ = hash.Write([]byte{0})

	keys := make([]string, 0, len(attrs))

	for key := range attrs {
		if key == "class" {
			continue
		}

		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		_, _ = hash.Write([]byte(key))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(attrs[key]))
		_, _ = hash.Write([]byte{0})
	}
}

// widthBasisFor returns the available width the element is laid out in,
// mirroring relayoutAvailableWidth without an engine. The scale comes from
// the options zoom, which is the only engine input the formula reads besides
// the kept style.
func widthBasisFor(kept *box, opts Options) float64 {
	if kept == nil || kept.style == nil {
		return 0
	}

	scale := opts.Zoom
	if scale == 0 {
		scale = 1
	}

	marginL := kept.style.MarginLeft * scale
	marginR := kept.style.MarginRight * scale

	if kept.style.WidthPercent <= 0 {
		return kept.w + marginL + marginR
	}

	chrome := (kept.style.PaddingLeft + kept.style.PaddingRight +
		kept.style.BorderLeft.Width + kept.style.BorderRight.Width) * scale

	contentW := kept.w - chrome
	if kept.style.BoxSizing != borderBox {
		contentW = kept.w
	}

	if contentW < 0 {
		contentW = 0
	}

	return contentW * oneHundred / kept.style.WidthPercent
}

// keptSubtreeMatchesDOM reports whether the kept box subtree and the DOM
// subtree describe the same elements. A new element has no kept box and a
// removed element leaves a kept box whose node no longer sits under root;
// both are structural changes the fast path refuses.
func keptSubtreeMatchesDOM(kept *box, root *html.Node) bool {
	if kept == nil || root == nil {
		return false
	}

	keptNodes := make(map[*html.Node]bool)
	collectKeptNodes(kept, keptNodes)

	var walk func(node *html.Node) bool
	walk = func(node *html.Node) bool {
		if node.Type == html.ElementNode && !keptNodes[node] {
			return false
		}

		for _, child := range node.Children {
			if !walk(child) {
				return false
			}
		}

		return true
	}

	if !walk(root) {
		return false
	}

	for node := range keptNodes {
		if !isDOMDescendant(node, root) {
			return false
		}
	}

	return true
}

// collectKeptNodes gathers every element node a kept box subtree owns.
func collectKeptNodes(kept *box, out map[*html.Node]bool) {
	if kept == nil {
		return
	}

	if kept.node != nil && kept.node.Type == html.ElementNode {
		out[kept.node] = true
	}

	for _, child := range kept.children {
		collectKeptNodes(child, out)
	}

	for _, row := range kept.rows {
		for _, cell := range row {
			collectKeptNodes(cell, out)
		}
	}
}

// isDOMDescendant reports whether node is root or sits under it.
func isDOMDescendant(node, root *html.Node) bool {
	for current := node; current != nil; current = current.Parent {
		if current == root {
			return true
		}
	}

	return false
}

// subtreeText concatenates the text of a DOM subtree.
func subtreeText(root *html.Node) string {
	if root == nil {
		return ""
	}

	if root.Type == html.TextNode {
		return root.Text
	}

	out := ""
	for _, child := range root.Children {
		out += subtreeText(child)
	}

	return out
}

// opsText concatenates the text operations of a retained window.
func opsText(ops []Op, start, count int) string {
	if count <= 0 || start < 0 || start+count > len(ops) {
		return ""
	}

	out := ""

	for _, op := range ops[start : start+count] {
		if op.Kind == OpText {
			out += op.Text
		}
	}

	return out
}

// translateOpsRange shifts every operation in the inclusive window
// [start, end] by (dx, dy), including grid run segments. It refuses windows
// that carry paths, baked transforms, projective maps, clip payloads, or
// fixed positioning, where a plain shift would lie. On success it records a
// translation for the counters and returns true; on refusal it changes
// nothing and returns false.
func translateOpsRange(ops []Op, start, end int, deltaX, deltaY float64) bool {
	if deltaX == 0 && deltaY == 0 {
		return true
	}

	if start < 0 || end < start || end >= len(ops) {
		return false
	}

	for index := start; index <= end; index++ {
		if !translatableOp(&ops[index]) {
			return false
		}
	}

	for index := start; index <= end; index++ {
		operation := &ops[index]
		operation.X += deltaX
		operation.Y += deltaY

		translateGridSegs(operation, deltaX, deltaY)
	}

	reuseTranslatedCount.Add(1)

	return true
}

// translatableOp reports whether a plain coordinate shift is honest.
func translatableOp(operation *Op) bool {
	if operation.Fixed || operation.XformSet || operation.HasProjective {
		return false
	}

	if operation.opExtra == nil || operation.opExtra == emptyExtra {
		return true
	}

	extra := operation.opExtra

	return !extra.HasPath && extra.CornerPath == nil && !extra.HasClip && extra.Pristine == nil
}

// translateGridSegs shifts the segments of a grid run alongside the operation.
func translateGridSegs(operation *Op, deltaX, deltaY float64) {
	if operation.Kind != OpGridRun || operation.Grid == nil {
		return
	}

	for seg := range operation.Grid.Segs {
		operation.Grid.Segs[seg].X += deltaX
		operation.Grid.Segs[seg].Y += deltaY
	}
}
