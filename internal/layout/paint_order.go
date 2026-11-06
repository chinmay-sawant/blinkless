package layout

import (
	"slices"
)

const (
	paintLayerChrome = iota
	paintLayerContent
	paintLayerOutline
)

// PaintOrder returns operation indices in the canonical display-list paint
// order. Display-list consumers share this same policy; backend drawing
// remains responsible for interpreting each operation.
func PaintOrder(ops []Op) []int {
	idx := make([]int, len(ops))
	for i := range ops {
		idx[i] = i
	}

	sortPaintIndices(ops, idx)

	return idx
}

// paintOrderSubset returns an ordered copy of an operation subset. All
// adapters use this policy so a band cannot accidentally fall back to source
// order while the paginated body uses paint order.
func paintOrderSubset(ops []Op, idxs []int) []int {
	ordered := append([]int(nil), idxs...)
	sortPaintIndices(ops, ordered)

	return ordered
}

// sortPaintIndices sorts an existing operation subset without changing which
// operations it contains. Pagination tests use this compatibility helper for
// fixed and per-page lists; the comparison itself lives in comparePaintOrder.
func sortPaintIndices(ops []Op, idxs []int) {
	if len(idxs) <= 1 {
		return
	}

	if hasGroups(ops, idxs) {
		ordered := groupedPaintOrder(ops, idxs)
		copy(idxs, ordered)

		return
	}

	alreadySorted := true

	for i := range len(idxs) - 1 {
		if comparePaintOrder(ops, idxs[i], idxs[i+1]) > 0 {
			alreadySorted = false

			break
		}
	}

	if alreadySorted {
		return
	}

	slices.SortStableFunc(idxs, func(a, b int) int {
		return comparePaintOrder(ops, a, b)
	})
}

func hasGroups(ops []Op, idxs []int) bool {
	for _, idx := range idxs {
		if ops[idx].Group() != nil {
			return true
		}
	}

	return false
}

type paintGroupOrder struct {
	group *BlendGroup
	// anchor uses the group's begin marker from the full op list, even when a
	// subset omits that marker. It is the group's slot in its parent's order.
	anchor int
	// begins and ends contain only marker indices selected by the caller.
	begins   []int
	ends     []int
	ops      []int
	children []*paintGroupOrder
}

type paintOrderItem struct {
	op    int
	group *paintGroupOrder
}

type paintOrderBuilder struct {
	anchors map[*BlendGroup]int
	groups  map[*BlendGroup]*paintGroupOrder
	roots   []*paintGroupOrder
}

// groupedPaintOrder keeps every group as one item in its parent's paint
// order. Group membership comes from the operation's group pointer and the
// group's Parent chain. Boundary markers are tokens to emit, not source-index
// ranges: inline paint can belong to a group after its end marker in Ops.
func groupedPaintOrder(ops []Op, idxs []int) []int {
	builder := newPaintOrderBuilder(ops)
	rootOps := builder.collect(ops, idxs)

	items := make([]paintOrderItem, 0, len(rootOps)+len(builder.roots))
	for _, idx := range rootOps {
		items = append(items, paintOrderItem{op: idx, group: nil})
	}

	for _, group := range builder.roots {
		items = append(items, paintOrderItem{op: 0, group: group})
	}

	out := make([]int, 0, len(idxs))
	appendSortedPaintItems(ops, items, &out)

	return out
}

func newPaintOrderBuilder(ops []Op) *paintOrderBuilder {
	builder := &paintOrderBuilder{
		anchors: paintGroupAnchors(ops),
		groups:  make(map[*BlendGroup]*paintGroupOrder),
		roots:   nil,
	}

	return builder
}

func paintGroupAnchors(ops []Op) map[*BlendGroup]int {
	firstMember := make(map[*BlendGroup]int)
	firstBegin := make(map[*BlendGroup]int)

	// Use the full list to keep a group's owner slot when idxs omits its
	// boundary markers, as page and band subsets may do.
	for idx := range ops {
		operation := &ops[idx]
		group := operation.Group()

		for owner := group; owner != nil; owner = owner.Parent {
			if _, ok := firstMember[owner]; !ok {
				firstMember[owner] = idx
			}
		}

		if group != nil && operation.IsGroupBegin() {
			if _, ok := firstBegin[group]; !ok {
				firstBegin[group] = idx
			}
		}
	}

	anchors := make(map[*BlendGroup]int, len(firstMember)+len(firstBegin))
	for group, idx := range firstMember {
		anchors[group] = idx
	}

	for group, idx := range firstBegin {
		anchors[group] = idx
	}

	return anchors
}

func (builder *paintOrderBuilder) collect(ops []Op, idxs []int) []int {
	rootOps := make([]int, 0, len(idxs))

	for _, idx := range idxs {
		operation := &ops[idx]
		group := operation.Group()

		if group == nil {
			rootOps = append(rootOps, idx)

			continue
		}

		builder.collectGroupOperation(group, operation.GroupBoundary(), idx)
	}

	return rootOps
}

func (builder *paintOrderBuilder) ensureGroup(group *BlendGroup) *paintGroupOrder {
	if group == nil {
		return nil
	}

	if order, ok := builder.groups[group]; ok {
		return order
	}

	anchor, ok := builder.anchors[group]
	if !ok {
		anchor = -1
	}

	order := &paintGroupOrder{
		group:    group,
		anchor:   anchor,
		begins:   nil,
		ends:     nil,
		ops:      nil,
		children: nil,
	}
	builder.groups[group] = order

	parent := builder.ensureGroup(group.Parent)
	if parent == nil {
		builder.roots = append(builder.roots, order)
	} else {
		parent.children = append(parent.children, order)
	}

	return order
}

func (builder *paintOrderBuilder) collectGroupOperation(
	group *BlendGroup, boundary uint8, opIndex int,
) {
	order := builder.ensureGroup(group)

	switch boundary {
	case groupMarkBegin:
		order.begins = append(order.begins, opIndex)
	case groupMarkEnd:
		order.ends = append(order.ends, opIndex)
	default:
		order.ops = append(order.ops, opIndex)
	}
}

func appendSortedPaintItems(ops []Op, items []paintOrderItem, out *[]int) {
	if len(items) == 0 {
		return
	}

	itemIndex := func(item paintOrderItem) int {
		if item.group == nil {
			return item.op
		}

		return item.group.anchor
	}

	slices.SortStableFunc(items, func(a, b paintOrderItem) int {
		return comparePaintOrder(ops, itemIndex(a), itemIndex(b))
	})

	for _, item := range items {
		if item.group == nil {
			*out = append(*out, item.op)

			continue
		}

		*out = append(*out, item.group.begins...)

		children := make([]paintOrderItem, 0, len(item.group.ops)+len(item.group.children))
		for _, idx := range item.group.ops {
			children = append(children, paintOrderItem{op: idx, group: nil})
		}

		for _, child := range item.group.children {
			children = append(children, paintOrderItem{op: 0, group: child})
		}

		appendSortedPaintItems(ops, children, out)

		*out = append(*out, item.group.ends...)
	}
}

//nolint:cyclop // compares z-index, position, and chrome layers
func comparePaintOrder(ops []Op, left, right int) int {
	leftOp, rightOp := &ops[left], &ops[right]

	leftZ, rightZ := 0, 0
	if leftOp.ZIndexSet {
		leftZ = leftOp.ZIndex
	}

	if rightOp.ZIndexSet {
		rightZ = rightOp.ZIndex
	}

	if leftZ != rightZ {
		if leftZ < rightZ {
			return -1
		}

		return 1
	}

	if leftOp.Positioned != rightOp.Positioned {
		if !leftOp.Positioned {
			return -1
		}

		return 1
	}

	leftLayer, rightLayer := paintLayer(leftOp), paintLayer(rightOp)
	if leftLayer != rightLayer {
		if leftLayer < rightLayer {
			return -1
		}

		return 1
	}

	if left < right {
		return -1
	}

	if left > right {
		return 1
	}

	return 0
}

// paintLayer orders ops within a z-index band: ordinary chrome under content,
// outlines above descendant content.
func paintLayer(paintOp *Op) int {
	if paintOp.IsBackground {
		return paintLayerChrome
	}

	if paintOp.isOutline() {
		return paintLayerOutline
	}

	switch paintOp.Kind {
	case OpFillRect, OpStrokeRect, OpLine, OpGridRun, OpPath:
		return paintLayerChrome
	case OpText, OpImage, OpLinkURI, OpBullet, OpUnknown, opKindNoop:
		return paintLayerContent
	}

	return paintLayerContent
}

// countOrderingRelevant counts the operations that can move under paint
// order: explicitly z-indexed or positioned operations, non-content layers,
// and blend group members. The batch tracks both counts incrementally and
// only re-sorts globally when the spliced window or the page needs it.
func countOrderingRelevant(ops []Op) (relevant, grouped int) {
	for index := range ops {
		op := &ops[index]

		if op.Group() != nil {
			relevant++
			grouped++

			continue
		}

		if op.ZIndexSet || op.Positioned || paintLayer(op) != paintLayerContent {
			relevant++
		}
	}

	return relevant, grouped
}

// splicePaintOrder maintains a paint order over one committed op window. The
// survivors keep their relative order with removed indices dropped and later
// ones shifted; when neither the removed nor the new window carries an
// ordering-relevant operation and no operation anywhere belongs to a blend
// group, the new indices merge into place and the global sort is skipped,
// provided the new run is itself sorted. A window that kept its length and its
// ordering-neutral keys leaves the maintained order untouched: the old window
// was neutral too, so both sorts rank the same indices by index alone and the
// ops outside the window are unchanged. Any other change falls back to the
// full paint order.
func splicePaintOrder(ops []Op, order []int, start, oldCount, newCount int, neutral bool) []int {
	if !neutral || !windowSorted(ops, start, newCount) {
		return PaintOrder(ops)
	}

	if oldCount == newCount && len(order) == len(ops) && windowNeutral(ops, start, newCount) {
		return order
	}

	end := start + oldCount - 1
	shift := newCount - oldCount

	kept := make([]int, 0, len(order)-oldCount+newCount)

	for _, index := range order {
		switch {
		case index >= start && index <= end:
			continue
		case index > end:
			kept = append(kept, index+shift)
		default:
			kept = append(kept, index)
		}
	}

	return mergePaintOrder(ops, kept, start, newCount)
}

// windowNeutral reports whether one index run carries no ordering key: no
// blend group, no explicit z-index, no positioning, and the content layer
// only. Such ops sort by index alone, so a window of them keeps every one of
// its indices in the place a neutral window before it held. It reports false
// for a range outside the op slice.
func windowNeutral(ops []Op, start, count int) bool {
	if start < 0 || count < 0 || start+count > len(ops) {
		return false
	}

	for index := start; index < start+count; index++ {
		op := &ops[index]

		if op.Group() != nil || op.ZIndexSet || op.Positioned || paintLayer(op) != paintLayerContent {
			return false
		}
	}

	return true
}

// mergePaintOrder merges the new window indices into a sorted survivor
// order. A neutral window shares one rank, so its indices are sorted, and
// the survivors stay sorted through the remap; the linear merge is the full
// sort.
func mergePaintOrder(ops []Op, kept []int, start, newCount int) []int {
	merged := make([]int, 0, len(kept)+newCount)
	pos, next := 0, start

	for pos < len(kept) && next < start+newCount {
		if comparePaintOrder(ops, kept[pos], next) <= 0 {
			merged = append(merged, kept[pos])
			pos++
		} else {
			merged = append(merged, next)
			next++
		}
	}

	merged = append(merged, kept[pos:]...)

	for ; next < start+newCount; next++ {
		merged = append(merged, next)
	}

	return merged
}

// windowSorted reports whether one index run is already sorted under the
// paint order. Adjacent pairs decide it: the comparison is a total order,
// so a run with every neighbor in order is sorted.
func windowSorted(ops []Op, start, newCount int) bool {
	for next := start + 1; next < start+newCount; next++ {
		if next <= 0 || next >= len(ops) || next-1 < 0 || next-1 >= len(ops) {
			return false
		}

		if comparePaintOrder(ops, next-1, next) > 0 {
			return false
		}
	}

	return true
}
