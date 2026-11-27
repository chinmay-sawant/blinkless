package layout

import (
	"context"
	"maps"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// tryAbsTranslate checks whether an out-of-flow position:absolute block box
// only changed its left/top offsets (same size, same paint, same DOM subtree,
// and staying clear of any ancestor clip boundary). When true, it translates
// the retained box subtree and its operation slice in place, skipping full
// subtree style resolution and layout.
func tryAbsTranslate(
	ctx context.Context, prev *Result, thread *boxThread, target *relayoutTarget,
	node *html.Node, opts Options, out *SubtreeBatchResult,
	elementIDs []string, currentID string,
) bool {
	if !absTranslateTargetOK(prev, target, node, opts) {
		return false
	}

	if absBatchTouchesDescendant(thread, prev, node, elementIDs, currentID) {
		return false
	}

	start, count, err := relayoutGroupWindow(prev, target)
	if err != nil || count == 0 {
		return false
	}

	if absSubtreeHasTables(target.box) {
		return false
	}

	if absOpsNeedFullRelayout(prev.Ops, start, count) {
		return false
	}

	fresh, ok := resolveSingleElementStyle(ctx, prev, target, node, opts)
	if !ok || fresh == nil {
		return false
	}

	kept := prev.styles[node]
	if kept == nil {
		kept = target.box.style
	}

	dx, dy, ok := absTranslateDelta(kept, fresh, opts)
	if !ok {
		return false
	}

	if !absTranslateClearOfClips(target, dx, dy) {
		return false
	}

	if !absTranslateSameSubtree(prev, node, prev.Ops, start, count) {
		return false
	}

	commitAbsTranslate(prev, thread, target, node, fresh, start, count, dx, dy, out)

	return true
}

func absBatchTouchesDescendant(
	thread *boxThread, prev *Result, root *html.Node, elementIDs []string, currentID string,
) bool {
	if len(elementIDs) <= 1 {
		return false
	}

	for _, id := range elementIDs {
		if id == currentID {
			continue
		}

		other := batchNodeForID(thread, prev, id)
		for cur := other; cur != nil; cur = cur.Parent {
			if cur == root {
				return true
			}
		}
	}

	return false
}

func absTranslateSameSubtree(prev *Result, root *html.Node, ops []Op, start, count int) bool {
	var check func(*html.Node) bool
	check = func(n *html.Node) bool {
		if n == nil {
			return true
		}
		if n.Type == html.ElementNode {
			if prev.styles[n] == nil {
				return false
			}
		}
		for _, c := range n.Children {
			if !check(c) {
				return false
			}
		}
		return true
	}

	if !check(root) {
		return false
	}

	return stripASCIIWhitespace(subtreeText(root)) == stripASCIIWhitespace(opsText(ops, start, count))
}

func stripASCIIWhitespace(s string) string {
	var buf []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			if buf == nil {
				buf = make([]byte, 0, len(s))
				buf = append(buf, s[:i]...)
			}
			continue
		}
		if buf != nil {
			buf = append(buf, c)
		}
	}
	if buf == nil {
		return s
	}
	return string(buf)
}

func absTranslateTargetOK(prev *Result, target *relayoutTarget, node *html.Node, opts Options) bool {
	if prev == nil || prev.styles == nil || target == nil || target.box == nil || node == nil {
		return false
	}

	if node.Type != html.ElementNode || target.box.node != node || target.box.kind != boxKindBlock {
		return false
	}

	if target.box.style == nil || target.box.style.Position != positionAbsolute {
		return false
	}

	if target.parent != nil && target.parent.style != nil {
		ov := target.parent.style.Overflow
		if ov == "auto" || ov == "scroll" {
			return false
		}
	}

	return relayoutNear(opts.Width, prev.Width) && checkRelayoutTarget(prev, target) == nil
}

func absSubtreeHasTables(b *box) bool {
	if b == nil {
		return false
	}

	if b.kind == boxKindTable || len(b.rows) > 0 {
		return true
	}

	for _, c := range b.children {
		if absSubtreeHasTables(c) {
			return true
		}
	}

	return false
}

// absOpsNeedFullRelayout reports whether any op in the window carries geometry
// that shiftOpX/shiftOpY does not translate (vector paths, corner paths, baked
// transforms, or projective transforms).
func absOpsNeedFullRelayout(ops []Op, start, count int) bool {
	end := min(start+count, len(ops))
	for k := start; k < end; k++ {
		op := &ops[k]
		if op.XformSet {
			return true
		}
		if op.opExtra != nil && (op.opExtra.HasPath || op.opExtra.CornerPath != nil || op.opExtra.HasProjective) {
			return true
		}
	}

	return false
}

// resolveSingleElementStyle resolves only node's own style (without walking
// its descendants), reusing the document's pooled styleStore and memo.
func resolveSingleElementStyle(
	ctx context.Context, prev *Result, target *relayoutTarget, node *html.Node, opts Options,
) (*ResolvedStyle, bool) {
	styleCtx := &styleContext{
		ctx:                ctx,
		sheets:             opts.Sheets,
		properties:         registeredProperties(opts.Sheets),
		media:              opts.Media,
		viewportW:          opts.Width,
		viewportH:          opts.Height,
		printLinkUnderline: opts.PrintLinkUnderline,
		state:              opts.State,
	}
	defer styleCtx.releaseMatchCache()

	if remBase := relayoutRemBase(prev); remBase > 0 {
		styleCtx.remBase = remBase
	}

	tag := subtreeStyleDocTagOf(prev, opts)
	store := rentSubtreeStyleStore(tag)
	styleCtx.memo = store.memo

	parent := relayoutParentStyle(target)
	fresh := resolveElementStyleMemo(node, styleCtx, parent, store)
	store.memo = styleCtx.memo

	putSubtreeStyleStore(prev, tag, store)

	if ctx.Err() != nil || fresh == nil {
		return nil, false
	}

	return fresh, true
}

// absTranslateDelta reports the pixel delta (dx, dy) when fresh differs from
// kept only in positioning Left/Top (with both RightAuto and BottomAuto).
func absTranslateDelta(kept, fresh *ResolvedStyle, opts Options) (float64, float64, bool) {
	if kept == nil || fresh == nil {
		return 0, 0, false
	}

	if kept.Position != positionAbsolute || fresh.Position != positionAbsolute {
		return 0, 0, false
	}

	kp, fp := kept.positioning(), fresh.positioning()
	if kp.LeftAuto || kp.TopAuto || !kp.RightAuto || !kp.BottomAuto {
		return 0, 0, false
	}

	if fp.LeftAuto || fp.TopAuto || !fp.RightAuto || !fp.BottomAuto {
		return 0, 0, false
	}

	// Compare every ResolvedStyle field with Left/Top normalized to kept.
	probe := *fresh
	probePos := *fp
	probePos.Left = kp.Left
	probePos.Top = kp.Top
	probe.positioningGroup = &probePos

	if !styleInternEqual(kept, &probe) {
		return 0, 0, false
	}

	scale := zoomScale(opts.Zoom)

	return (fp.Left - kp.Left) * scale, (fp.Top - kp.Top) * scale, true
}

// absTranslateClearOfClips verifies that neither the old nor the translated
// bounding box crosses any clipping ancestor's bounds. If it stays completely
// inside every clipping ancestor, existing op clips remain valid.
func absTranslateClearOfClips(target *relayoutTarget, dx, dy float64) bool {
	oldX0, oldY0 := target.box.x, target.box.y
	oldX1, oldY1 := oldX0+target.box.w, oldY0+target.box.height
	newX0, newY0 := oldX0+dx, oldY0+dy
	newX1, newY1 := oldX1+dx, oldY1+dy

	for _, anc := range target.ancestors {
		if anc == nil || anc.style == nil {
			continue
		}

		ov := anc.style.Overflow
		if ov != "hidden" && ov != "clip" && ov != "scroll" && ov != "auto" {
			continue
		}

		ax0, ay0 := anc.x, anc.y
		ax1, ay1 := anc.x+anc.w, anc.y+anc.height

		if oldX0 < ax0 || oldY0 < ay0 || oldX1 > ax1 || oldY1 > ay1 {
			return false
		}

		if newX0 < ax0 || newY0 < ay0 || newX1 > ax1 || newY1 > ay1 {
			return false
		}
	}

	return true
}

func cloneBoxSubtreeShifted(b *box, dx, dy float64) *box {
	if b == nil {
		return nil
	}

	cp := *b
	cp.x += dx
	cp.y += dy

	if len(b.children) > 0 {
		cp.children = make([]*box, len(b.children))
		for i, c := range b.children {
			cp.children[i] = cloneBoxSubtreeShifted(c, dx, dy)
		}
	}

	return &cp
}

func commitAbsTranslate(
	prev *Result, thread *boxThread, target *relayoutTarget, node *html.Node,
	fresh *ResolvedStyle, start, count int, dx, dy float64, out *SubtreeBatchResult,
) {
	out.PaintOnly = false

	oldDamage, hasOld := spliceDamage(prev.Ops, start, count)

	if dx != 0 || dy != 0 {
		for k := start; k < start+count && k < len(prev.Ops); k++ {
			if prev.Ops[k].Grid != nil {
				g := *prev.Ops[k].Grid
				g.Segs = append([]GridSeg(nil), g.Segs...)
				prev.Ops[k].Grid = &g
			}
			shiftOpX(&prev.Ops[k], dx)
			shiftOpY(&prev.Ops[k], dy)
		}
	}

	newBox := cloneBoxSubtreeShifted(target.box, dx, dy)
	newBox.style = restyledBoxStyle(newBox.style, prev.styles[node], fresh)
	if prev.stylesShared {
		prev.styles = maps.Clone(prev.styles)
		prev.stylesShared = false
	}
	prev.styles[node] = fresh

	for _, child := range node.Children {
		if _, shared := prev.styles[child]; shared && child.Type == html.TextNode {
			prev.styles[child] = fresh
		}
	}

	if prev.boxesShared {
		pathCopySwapBox(prev, target, newBox, thread)
	} else {
		if target.parent != nil {
			for i, c := range target.parent.children {
				if c == target.box {
					target.parent.children[i] = newBox

					break
				}
			}
		}
		thread.replace(target.box, newBox, target.parent)
	}
	prev.invalidateBoxIndex()

	if out.SameBoxes {
		cnt := countElementBoxes(newBox)
		out.boxPatches = append(out.boxPatches, subtreeBoxPatch{
			baseIdx:   -1,
			oldElem:   cnt,
			newElem:   cnt,
			box:       newBox,
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
		PaintOnly:    false,
	})
}
