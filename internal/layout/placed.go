package layout

import (
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// PlacedElement is one element border box in canvas points. Y grows downward
// from the top of the layout canvas. Layout fills these boxes before pagination.
type PlacedElement struct {
	Sticky              *StickyConstraint
	Fixed               bool
	ID                  string
	Tag                 string
	Action              string
	Text                string
	ParentIndex         int
	OperationIDs        []uint64
	Animations          []AnimationRecord
	Transitions         []TransitionRecord
	TriggerScope        string
	ViewTransitionName  string
	ViewTransitionClass string
	ViewTransitionGroup string
	ViewTransitionScope string
	// Timelines holds the named scroll and view timelines this element
	// declares. Scroll comes before view; a nil slice means no declaration.
	Timelines []TimelineDef
	// Triggers holds the scope, trigger, interest delay, and interactivity
	// values of this element.
	Triggers PlacedTriggers
	// Declared holds the stored-only properties this element declares, by
	// property name, as computed text. Nil when it declares none.
	Declared map[string]string
	// OverflowClip is the accumulated ancestor overflow padding box in
	// canvas points, empty with HasOverflowClip false when no ancestor
	// clips. A host painting this element through a transform intersects
	// the paint with it; geometry-baked clipping alone moves with the
	// matrix and cannot hold transformed paint inside the box.
	OverflowClip    [4]float64
	HasOverflowClip bool
	// Transform carries the resolved transform metadata for this element. It is
	// the zero value when the element declares no transform.
	Transform TransformInfo
	X         float64
	Y         float64
	W         float64
	H         float64
}

// WalkPlacedOptions tunes what WalkPlacedWithOptions computes per element.
// The zero value is the lean path: geometry and ownership without text.
type WalkPlacedOptions struct {
	// IncludeText fills PlacedElement.Text with the element subtree text.
	// TextContent walks the whole subtree per element, so bulk walks that
	// only need boxes leave it off.
	IncludeText bool
	// LayoutScale is the placement scale, defaulting to one.
	LayoutScale float64
}

// WalkPlaced walks laid-out element boxes in document order, invoking fn for each element.
// A nil result does nothing.
func WalkPlaced(res *Result, fn func(elem *PlacedElement)) {
	WalkPlacedWithOptions(res, WalkPlacedOptions{IncludeText: true, LayoutScale: 0}, fn)
}

// WalkPlacedWithOptions walks like WalkPlaced, computing only what opts
// requests. Callers that need boxes, transforms, and operation ownership
// without text use the zero value.
func WalkPlacedWithOptions(res *Result, opts WalkPlacedOptions, fn func(elem *PlacedElement)) {
	if res == nil || res.root == nil {
		return
	}

	capIDs := estimatePlacedOpIDs(res)
	if capIDs < len(res.Ops) {
		capIDs = len(res.Ops)
	}

	allIDs := make([]uint64, 0, capIDs)

	var item PlacedElement

	viewport := svgViewport{x: 0, y: 0, w: 0, h: 0, set: false}

	scroll := placedScrollContext{parent: nil, blocked: false, scale: opts.LayoutScale}
	page := placedScroller{set: false, id: ""}

	walkPlacedFunc(res.root, res.Ops, -1, IdentityMatrix(), viewport, fn, &allIDs, 0, &item, opts, scroll, page)
}

func estimatePlacedOpIDs(res *Result) int {
	if len(res.boxes) > 0 {
		return estimateBoxOpIDs(res.boxes, len(res.Ops))
	}

	if res.root != nil {
		return countSubtreeOpIDs(res.root, len(res.Ops))
	}

	return len(res.Ops)
}

// estimateBoxOpIDs sums the op-ID counts of the top-level boxes.
func estimateBoxOpIDs(boxes []*box, nOps int) int {
	total := 0

	for _, boxNode := range boxes {
		if boxNode.node != nil && boxNode.node.Type == html.ElementNode &&
			boxNode.opEnd >= boxNode.opStart && boxNode.opStart >= 0 {
			end := boxNode.opEnd
			if end >= nOps {
				end = nOps - 1
			}

			if end >= boxNode.opStart {
				total += end - boxNode.opStart + 1
			}
		}
	}

	return total
}

func countSubtreeOpIDs(boxNode *box, nOps int) int {
	if boxNode == nil {
		return 0
	}

	total := 0

	if boxNode.node != nil && boxNode.node.Type == html.ElementNode &&
		boxNode.opEnd >= boxNode.opStart && boxNode.opStart >= 0 {
		end := boxNode.opEnd
		if end >= nOps {
			end = nOps - 1
		}

		if end >= boxNode.opStart {
			total += end - boxNode.opStart + 1
		}
	}

	for _, child := range boxNode.children {
		total += countSubtreeOpIDs(child, nOps)
	}

	return total
}

func walkPlacedFunc(
	boxNode *box, ops []Op, parentIndex int, parentAccum Matrix2D, viewport svgViewport,
	fn func(elem *PlacedElement), allIDs *[]uint64, index int, item *PlacedElement,
	opts WalkPlacedOptions, scroll placedScrollContext, scroller placedScroller,
) int {
	if boxNode == nil {
		return index
	}

	childParent := parentIndex
	childAccum := parentAccum
	info := placedTransformInfo(boxNode, parentAccum, viewport)

	if info.HasMatrix {
		childAccum = info.World
	}

	if boxNode.node != nil && boxNode.node.Type == html.ElementNode {
		fillPlacedElement(item, boxNode, info, parentIndex, ops, allIDs, opts, scroll, scroller)

		childParent = index
		index++

		fn(item)
	}

	// An <svg> box establishes a viewport for its subtree; the box's own
	// transform resolves against the ancestor viewport above, matching the
	// stamp walk.
	childViewport := childViewportForBox(boxNode, viewport)
	childScroll := childScrollContext(boxNode, scroll)

	for _, child := range boxNode.children {
		index = walkPlacedFunc(
			child, ops, childParent, childAccum, childViewport, fn, allIDs, index, item, opts,
			childScroll, scrollerBelow(boxNode, scroller),
		)
	}

	return index
}

func defaultPlacedTriggers() PlacedTriggers {
	return PlacedTriggers{
		TimelineScope:    TimelineScopeDef{None: true, All: false, Names: nil},
		TimelineTriggers: nil,
		EventTriggers:    nil,
		InterestDelay:    InterestDelayDef{StartNormal: true, EndNormal: true, StartMs: 0, EndMs: 0},
		Interactivity:    overflowAuto,
	}
}

func fillPlacedMeta(
	item *PlacedElement, boxNode *box, sty *ResolvedStyle, elementID string,
	scroll placedScrollContext, scroller placedScroller,
) {
	if boxNode.sticky {
		item.Sticky = placedSticky(boxNode, scroll)
	}

	if sty.motion().Animation.Set {
		item.Animations = animationRecordsFromStyle(sty.motion().Animation)
	}

	if sty.motion().Transition.Set {
		item.Transitions = transitionRecordsFromStyle(sty.motion().Transition)
	}

	if sty.motion().ScrollTimelineName != "" || sty.motion().ViewTimelineName != "" || sty.Triggers != nil {
		item.Timelines = placedTimelineDefs(boxNode, elementID, scroller)
	}

	if sty.Triggers != nil {
		item.Triggers = placedTriggersFromStyle(sty)
	} else {
		item.Triggers = defaultPlacedTriggers()
	}
}

func fillPlacedElement(
	item *PlacedElement, boxNode *box, info TransformInfo, parentIndex int, ops []Op,
	allIDs *[]uint64, opts WalkPlacedOptions, scroll placedScrollContext, scroller placedScroller,
) {
	sty := boxNode.style
	elementID := boxNode.node.AttrExact("id")

	*item = PlacedElement{
		Sticky:              nil,
		Fixed:               sty.Position == positionFixed,
		ID:                  elementID,
		Tag:                 boxNode.node.Name,
		Action:              boxNode.node.AttrExact("data-action"),
		ParentIndex:         parentIndex,
		OperationIDs:        appendOperationIDs(ops, boxNode.opStart, boxNode.opEnd, allIDs),
		Animations:          nil,
		Transitions:         nil,
		TriggerScope:        sty.motion().Animation.TriggerScope,
		ViewTransitionName:  sty.motion().Animation.ViewTransitionName,
		ViewTransitionClass: sty.motion().Animation.ViewTransitionClass,
		ViewTransitionGroup: sty.motion().Animation.ViewTransitionGroup,
		ViewTransitionScope: sty.motion().Animation.ViewTransitionScope,
		Timelines:           nil,
		Triggers:            defaultPlacedTriggers(),
		OverflowClip:        elementOverflowClip(boxNode),
		HasOverflowClip:     boxNode.overflowClip != nil,
		Transform:           info,
		X:                   boxNode.x,
		Y:                   boxNode.y,
		W:                   boxNode.w,
		H:                   boxNode.height,
		Text:                "",
		Declared:            nil,
	}

	fillPlacedMeta(item, boxNode, sty, elementID, scroll, scroller)

	if sty.Declared != nil {
		item.Declared = declaredValues(sty)
	}

	if opts.IncludeText {
		item.Text = boxNode.node.TextContent()
	}
}

// placedTransformInfo resolves the transform metadata for one box while
// computing the box bake once: Own is the single boxTransformAccum call and
// World is the parent product of it, which is the same multiplication
// boxTransformAccum performs internally. Boxes without a static transform
// skip the bake entirely and hand the parent matrix through.
func placedTransformInfo(boxNode *box, parentAccum Matrix2D, viewport svgViewport) TransformInfo {
	if boxNode == nil || boxNode.style == nil {
		return TransformInfo{}
	}

	sty := boxNode.style
	originX, originY := resolveTransformOriginWithBox(sty.effects().TransformOrigin, boxNode, sty.effects().TransformBox, viewport)

	own := IdentityMatrix()
	world := parentAccum

	if sty.effects().HasTransform {
		own = boxTransformAccum(boxNode, IdentityMatrix(), viewport)
		world = parentAccum.Mul(own)
	}

	info := TransformInfo{
		OriginX:            originX,
		OriginY:            originY,
		Box:                sty.effects().TransformBox,
		Parent:             parentAccum,
		Own:                own,
		World:              world,
		HasMatrix:          sty.effects().HasTransform,
		HasTransform3D:     sty.effects().HasTransform3D,
		Perspective:        0,
		PerspectiveOriginX: 0,
		PerspectiveOriginY: 0,
		HasPerspective:     sty.effects().HasPerspective,
		Style:              sty.effects().TransformStyle,
		Backface:           sty.effects().BackfaceVisibility,
	}

	if sty.effects().HasPerspective {
		info.Perspective = sty.effects().PerspectiveDist
		info.PerspectiveOriginX, info.PerspectiveOriginY = perspectiveCenter(boxNode)
	}

	return info
}

// PlacedElements walks the laid-out element boxes in document order.
// A nil result returns nil. Paint and pagination are not required.
func PlacedElements(res *Result) []PlacedElement {
	if res == nil || res.root == nil {
		return nil
	}

	out := make([]PlacedElement, 0, len(res.boxes))
	WalkPlaced(res, func(elem *PlacedElement) {
		out = append(out, *elem)
	})

	return out
}

const patchAncScratchCap = 16

// finalizeSubtreeBoxPatches resolves any deferred PaintOnly base index in a
// mixed batch and validates that patches can be applied in batch order.
func finalizeSubtreeBoxPatches(root *box, patches []subtreeBoxPatch) bool {
	if root == nil || len(patches) == 0 {
		return false
	}

	hasCountChange := patchesHaveCountChange(patches)

	for idx := range patches {
		if patches[idx].oldElem <= 0 || patches[idx].newElem <= 0 {
			return false
		}

		if patches[idx].baseIdx >= 0 {
			continue
		}

		if hasCountChange {
			return false
		}

		foundIdx, ok := findPlacedElementIndex(root, patches[idx].box)
		if !ok {
			return false
		}

		patches[idx].baseIdx = foundIdx
	}

	return true
}

func patchesHaveCountChange(patches []subtreeBoxPatch) bool {
	for idx := range patches {
		if patches[idx].oldElem != patches[idx].newElem {
			return true
		}
	}

	return false
}

func findPlacedElementIndex(root, target *box) (int, bool) {
	count := 0

	var walk func(node *box) bool
	walk = func(node *box) bool {
		if node == nil {
			return false
		}

		if node == target {
			return true
		}

		if node.node != nil && node.node.Type == html.ElementNode {
			count++
		}

		for _, child := range node.children {
			if walk(child) {
				return true
			}
		}

		return false
	}

	if walk(root) {
		return count, true
	}

	return -1, false
}

// PrepareSubtreeBoxSplices applies each patch's element-count delta in batch
// order through spliceFn and returns the final element box count, updating each
// patch's baseIdx to its final post-batch index. It returns false if any two
// patches overlap.
func PrepareSubtreeBoxSplices(
	batch *SubtreeBatchResult, initialCount int,
	spliceFn func(baseIdx, oldElem, newElem int),
) (int, bool) {
	if batch == nil || !batch.SameBoxes || len(batch.boxPatches) == 0 {
		return 0, false
	}

	curCount := initialCount

	for idx := range batch.boxPatches {
		patch := &batch.boxPatches[idx]
		if patch.baseIdx < 0 || patch.baseIdx+patch.oldElem > curCount {
			return 0, false
		}

		delta := patch.newElem - patch.oldElem
		if !shiftPriorSubtreePatches(batch.boxPatches[:idx], patch.baseIdx, patch.oldElem, delta) {
			return 0, false
		}

		if spliceFn != nil && delta != 0 {
			spliceFn(patch.baseIdx, patch.oldElem, patch.newElem)
		}

		curCount += delta
	}

	return curCount, true
}

func shiftPriorSubtreePatches(prior []subtreeBoxPatch, baseIdx, oldElem, delta int) bool {
	for prevIdx := range prior {
		prev := &prior[prevIdx]
		kStart := prev.baseIdx
		kEnd := prev.baseIdx + prev.newElem

		if kStart < baseIdx+oldElem && baseIdx < kEnd {
			return false
		}

		if kStart >= baseIdx+oldElem {
			prev.baseIdx += delta
		}
	}

	return true
}

// ApplySubtreeBoxPatches walks only the patched subtrees of a batch, invoking
// fn for each element box inside a patched subtree and ancFn for each unique
// ancestor box up to the document root.
func ApplySubtreeBoxPatches(
	res *Result, batch *SubtreeBatchResult, opts WalkPlacedOptions,
	boxCount int,
	parentOf func(boxIdx int) int,
	textOf func(boxIdx int) string,
	fn func(boxIdx int, elem *PlacedElement),
	ancFn func(boxIdx int, opIDs []uint64, text string, updateText bool),
) bool {
	if res == nil || res.root == nil || batch == nil || !batch.SameBoxes || len(batch.boxPatches) == 0 {
		return false
	}

	capIDs, valid := estimateSubtreePatchOpIDs(res, batch.boxPatches, boxCount)
	if !valid {
		return false
	}

	allIDs := make([]uint64, 0, capIDs)

	var item PlacedElement

	ancBoxes := make(map[int]*box, patchAncScratchCap)
	ancNeedText := make(map[int]*html.Node, patchAncScratchCap)
	ancOrder := make([]int, 0, patchAncScratchCap)

	var singleOldText, singleNewText string
	singleTextSplice := len(batch.boxPatches) == 1

	for idx := range batch.boxPatches {
		patch := &batch.boxPatches[idx]
		oldText, newText, rootTextChanged, ok := walkPatchedSubtreeBoxes(
			res, patch, opts, &allIDs, &item, parentOf, textOf, fn,
		)

		if !ok {
			return false
		}

		if singleTextSplice && rootTextChanged {
			singleOldText = oldText
			singleNewText = newText
		}

		if !collectPatchedAncestors(
			patch, parentOf(patch.baseIdx), boxCount, rootTextChanged,
			parentOf, ancBoxes, ancNeedText, &ancOrder,
		) {
			return false
		}
	}

	emitPatchedAncestors(res.Ops, ancOrder, ancBoxes, ancNeedText, &allIDs, singleOldText, singleNewText, textOf, ancFn)

	return true
}

func emitPatchedAncestors(
	ops []Op, ancOrder []int, ancBoxes map[int]*box, ancNeedText map[int]*html.Node,
	allIDs *[]uint64, oldSubtreeText, newSubtreeText string,
	textOf func(boxIdx int) string,
	ancFn func(boxIdx int, opIDs []uint64, text string, updateText bool),
) {
	for _, ancIdx := range ancOrder {
		anc := ancBoxes[ancIdx]
		opIDs := appendOperationIDs(ops, anc.opStart, anc.opEnd, allIDs)

		if node, need := ancNeedText[ancIdx]; need {
			var text string
			if spliced, ok := spliceUniqueText(textOf(ancIdx), oldSubtreeText, newSubtreeText); ok {
				text = spliced
			} else {
				text = node.TextContent()
			}
			ancFn(ancIdx, opIDs, text, true)
		} else {
			ancFn(ancIdx, opIDs, "", false)
		}
	}
}

// SpliceOrNodeText updates an ancestor's cached text by splicing a unique
// child-text replacement when possible, falling back to node.TextContent().
func SpliceOrNodeText(ancOld, oldSub, newSub string, node *html.Node) string {
	if spliced, ok := spliceUniqueText(ancOld, oldSub, newSub); ok {
		return spliced
	}
	if node == nil {
		return ""
	}

	return node.TextContent()
}

func spliceUniqueText(ancOld, oldSub, newSub string) (string, bool) {
	if oldSub == "" || len(ancOld) < len(oldSub) {
		return "", false
	}
	idx := strings.Index(ancOld, oldSub)
	if idx < 0 {
		return "", false
	}
	if strings.Contains(ancOld[idx+1:], oldSub) {
		return "", false
	}
	var b strings.Builder
	b.Grow(len(ancOld) - len(oldSub) + len(newSub))
	b.WriteString(ancOld[:idx])
	b.WriteString(newSub)
	b.WriteString(ancOld[idx+len(oldSub):])

	return b.String(), true
}

func estimateSubtreePatchOpIDs(res *Result, patches []subtreeBoxPatch, boxCount int) (int, bool) {
	capIDs := 0
	seenAnc := make(map[*box]struct{}, patchAncScratchCap)

	for idx := range patches {
		patch := &patches[idx]
		if patch.baseIdx < 0 || patch.baseIdx+patch.newElem > boxCount {
			return 0, false
		}

		capIDs += countSubtreeOpIDs(patch.box, len(res.Ops))

		for _, anc := range patch.ancestors {
			if anc == nil || anc.opEnd < anc.opStart || anc.opStart < 0 {
				continue
			}

			if _, seen := seenAnc[anc]; !seen {
				seenAnc[anc] = struct{}{}
				capIDs += anc.opEnd - anc.opStart + 1
			}
		}
	}

	return capIDs, true
}

func ancestorPlacedContext(
	ancestors []*box, layoutScale float64,
) (Matrix2D, svgViewport, placedScrollContext, placedScroller) {
	parentAccum := IdentityMatrix()
	viewport := svgViewport{x: 0, y: 0, w: 0, h: 0, set: false}
	scroll := placedScrollContext{parent: nil, blocked: false, scale: layoutScale}
	scroller := placedScroller{set: false, id: ""}

	for _, anc := range ancestors {
		if anc == nil {
			continue
		}

		info := placedTransformInfo(anc, parentAccum, viewport)
		if info.HasMatrix {
			parentAccum = info.World
		}

		viewport = childViewportForBox(anc, viewport)
		scroll = childScrollContext(anc, scroll)
		scroller = scrollerBelow(anc, scroller)
	}

	return parentAccum, viewport, scroll, scroller
}

func walkPatchedSubtreeBoxes(
	res *Result, patch *subtreeBoxPatch, opts WalkPlacedOptions,
	allIDs *[]uint64, item *PlacedElement,
	parentOf func(boxIdx int) int,
	textOf func(boxIdx int) string,
	fn func(boxIdx int, elem *PlacedElement),
) (string, string, bool, bool) {
	parentAccum, viewport, scroll, scroller := ancestorPlacedContext(patch.ancestors, opts.LayoutScale)
	parentIdx := parentOf(patch.baseIdx)
	curIdx := patch.baseIdx
	var oldText, newText string
	rootTextChanged := false

	endIdx := walkPlacedFunc(
		patch.box, res.Ops, parentIdx, parentAccum, viewport,
		func(elem *PlacedElement) {
			if curIdx == patch.baseIdx && opts.IncludeText {
				oldText = textOf(curIdx)
				newText = elem.Text
				rootTextChanged = oldText != newText
			}

			fn(curIdx, elem)

			curIdx++
		},
		allIDs, patch.baseIdx, item, opts, scroll, scroller,
	)

	return oldText, newText, rootTextChanged, endIdx == patch.baseIdx+patch.newElem
}

func collectPatchedAncestors(
	patch *subtreeBoxPatch, currAncIdx, boxCount int, rootTextChanged bool,
	parentOf func(boxIdx int) int,
	ancBoxes map[int]*box, ancNeedText map[int]*html.Node, ancOrder *[]int,
) bool {
	for idx := len(patch.ancestors) - 1; idx >= 0; idx-- {
		anc := patch.ancestors[idx]
		if anc == nil || anc.node == nil || anc.node.Type != html.ElementNode {
			continue
		}

		if currAncIdx < 0 || currAncIdx >= boxCount {
			return false
		}

		if _, seen := ancBoxes[currAncIdx]; !seen {
			ancBoxes[currAncIdx] = anc

			*ancOrder = append(*ancOrder, currAncIdx)
		}

		if rootTextChanged {
			ancNeedText[currAncIdx] = anc.node
		}

		currAncIdx = parentOf(currAncIdx)
	}

	return true
}

// elementOverflowClip reads the ancestor overflow box clipOverflowTree
// recorded, in canvas points. Zero with no clip.
func elementOverflowClip(boxNode *box) [4]float64 {
	if boxNode == nil || boxNode.overflowClip == nil {
		return [4]float64{}
	}

	c := boxNode.overflowClip

	return [4]float64{c.x, c.y, c.w, c.h}
}

var emptyOperationIDs = []uint64{}

// appendOperationIDs collects the non-zero operation IDs in range in one
// pass, handing the caller a subslice of the shared backing.
func appendOperationIDs(ops []Op, start, end int, allIDs *[]uint64) []uint64 {
	if start < 0 || end < start || start >= len(ops) {
		return emptyOperationIDs
	}

	if end >= len(ops) {
		end = len(ops) - 1
	}

	if allIDs == nil {
		return collectOperationIDs(ops, start, end)
	}

	startIdx := len(*allIDs)

	for index := start; index <= end; index++ {
		if ops[index].ID != 0 {
			*allIDs = append(*allIDs, ops[index].ID)
		}
	}

	if len(*allIDs) == startIdx {
		return emptyOperationIDs
	}

	return (*allIDs)[startIdx:]
}

// collectOperationIDs copies the non-zero operation IDs in range into a fresh
// slice.
func collectOperationIDs(ops []Op, start, end int) []uint64 {
	count := 0

	for index := start; index <= end; index++ {
		if ops[index].ID != 0 {
			count++
		}
	}

	if count == 0 {
		return emptyOperationIDs
	}

	res := make([]uint64, 0, count)

	for index := start; index <= end; index++ {
		if ops[index].ID != 0 {
			res = append(res, ops[index].ID)
		}
	}

	return res
}
