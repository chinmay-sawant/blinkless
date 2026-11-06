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

	if err := ctx.Err(); err != nil {
		return nil, false
	}

	if len(overrides) > 0 {
		reuseMissCount.Add(1)

		return nil, false
	}

	target := findRelayoutTarget(prev.boxTreeRoot(), node)
	if target == nil || target.box == nil {
		reuseMissCount.Add(1)

		return nil, false
	}

	if err := checkRelayoutTarget(prev, target); err != nil {
		reuseMissCount.Add(1)

		return nil, false
	}

	kept := target.box
	if kept.style == nil {
		reuseMissCount.Add(1)

		return nil, false
	}

	if !relayoutNear(opts.Width, prev.Width) {
		reuseMissCount.Add(1)

		return nil, false
	}

	start, oldCount, err := relayoutWindow(prev, target)
	if err != nil {
		reuseMissCount.Add(1)

		return nil, false
	}

	if !keptSubtreeMatchesDOM(kept, node) {
		reuseMissCount.Add(1)

		return nil, false
	}

	if fingerprintContentDOM(node) != fingerprintKeptContent(kept, prev.Ops, start, oldCount) {
		reuseMissCount.Add(1)

		return nil, false
	}

	styles, err := relayoutSubtreeStyles(ctx, prev, target, node, opts)
	if err != nil {
		reuseMissCount.Add(1)

		return nil, false
	}

	if fingerprintLayoutSubtree(node, prev.styles) != fingerprintLayoutSubtree(node, styles) {
		reuseMissCount.Add(1)

		return nil, false
	}

	if basis := widthBasisFor(kept, opts); !(basis > 0) || !finitePositive(basis) {
		reuseMissCount.Add(1)

		return nil, false
	}

	eng, err := relayoutEngine(ctx, prev, opts, styles)
	if err != nil {
		reuseMissCount.Add(1)

		return nil, false
	}

	//nolint:contextcheck // the engine keeps the context and reads it at its recursion boundaries
	newBox, err := relayoutBuild(eng, prev, target, node, opts)
	if err != nil {
		reuseMissCount.Add(1)

		return nil, false
	}

	if len(eng.ops) != oldCount {
		reuseMissCount.Add(1)

		return nil, false
	}

	for index := range eng.ops {
		eng.ops[index].ID = prev.Ops[start+index].ID
	}

	newOps := eng.ops
	delta := len(newOps) - oldCount

	relayoutSpliceOps(prev, start, oldCount, newOps)
	relayoutRebaseTree(newBox, start)
	relayoutSwapBox(target, newBox)
	relayoutShiftRanges(prev.boxTreeRoot(), newBox, start, start+oldCount-1, delta)
	unionChildOpRanges(prev.boxTreeRoot())

	relayoutKeepBoxes(prev)
	relayoutKeepStyles(prev, styles)
	prev.MaxContentX, prev.HasFragmentLinks = censusOps(prev.Ops, prev.Width)

	reuseHitCount.Add(1)

	return relayoutSubtreeResult(prev, newBox, start, len(newOps), styles), true
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
func translateOpsRange(ops []Op, start, end int, dx, dy float64) bool {
	if dx == 0 && dy == 0 {
		return true
	}

	if start < 0 || end < start || end >= len(ops) {
		return false
	}

	for index := start; index <= end; index++ {
		op := &ops[index]

		if op.Fixed {
			return false
		}

		if op.XformSet || op.HasProjective {
			return false
		}

		if op.opExtra != nil && op.opExtra != emptyExtra {
			extra := op.opExtra
			if extra.HasPath || extra.CornerPath != nil || extra.HasClip || extra.Pristine != nil {
				return false
			}
		}
	}

	for index := start; index <= end; index++ {
		op := &ops[index]
		op.X += dx
		op.Y += dy

		if op.Kind == OpGridRun && op.Grid != nil {
			for seg := range op.Grid.Segs {
				op.Grid.Segs[seg].X += dx
				op.Grid.Segs[seg].Y += dy
			}
		}
	}

	reuseTranslatedCount.Add(1)

	return true
}
