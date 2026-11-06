package layout

import "github.com/chinmay-sawant/blinkless/internal/html"

// PlacedElement is one element border box in canvas points. Y grows downward
// from the top of the layout canvas. Layout fills these boxes before pagination.
type PlacedElement struct {
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

const defaultOpIDsMultiplier = 4

// WalkPlacedOptions tunes what WalkPlacedWithOptions computes per element.
// The zero value is the lean path: geometry and ownership without text.
type WalkPlacedOptions struct {
	// IncludeText fills PlacedElement.Text with the element subtree text.
	// TextContent walks the whole subtree per element, so bulk walks that
	// only need boxes leave it off.
	IncludeText bool
}

// WalkPlaced walks laid-out element boxes in document order, invoking fn for each element.
// A nil result does nothing.
//
//nolint:varnamelen // fn is standard callback identifier
func WalkPlaced(res *Result, fn func(elem *PlacedElement)) {
	WalkPlacedWithOptions(res, WalkPlacedOptions{IncludeText: true}, fn)
}

// WalkPlacedWithOptions walks like WalkPlaced, computing only what opts
// requests. Callers that need boxes, transforms, and operation ownership
// without text use the zero value.
//
//nolint:varnamelen // fn is standard callback identifier
func WalkPlacedWithOptions(res *Result, opts WalkPlacedOptions, fn func(elem *PlacedElement)) {
	if res == nil || res.root == nil {
		return
	}

	allIDs := make([]uint64, 0, len(res.Ops)*defaultOpIDsMultiplier)

	var item PlacedElement

	viewport := svgViewport{x: 0, y: 0, w: 0, h: 0, set: false}

	walkPlacedFunc(res.root, res.Ops, -1, IdentityMatrix(), viewport, fn, &allIDs, 0, &item, opts)
}

//nolint:varnamelen // fn is standard callback identifier
func walkPlacedFunc(
	boxNode *box, ops []Op, parentIndex int, parentAccum Matrix2D, viewport svgViewport,
	fn func(elem *PlacedElement), allIDs *[]uint64, index int, item *PlacedElement, opts WalkPlacedOptions,
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
		elementID := boxNode.node.Attribute("id")
		*item = PlacedElement{
			ID:                  elementID,
			Tag:                 boxNode.node.Name,
			Action:              boxNode.node.Attribute("data-action"),
			ParentIndex:         parentIndex,
			OperationIDs:        appendOperationIDs(ops, boxNode.opStart, boxNode.opEnd, allIDs),
			Animations:          animationRecordsFromStyle(boxNode.style.Animation),
			Transitions:         transitionRecordsFromStyle(boxNode.style.Transition),
			TriggerScope:        boxNode.style.Animation.TriggerScope,
			ViewTransitionName:  boxNode.style.Animation.ViewTransitionName,
			ViewTransitionClass: boxNode.style.Animation.ViewTransitionClass,
			ViewTransitionGroup: boxNode.style.Animation.ViewTransitionGroup,
			ViewTransitionScope: boxNode.style.Animation.ViewTransitionScope,
			Timelines:           timelineDefsFromStyle(boxNode.style, elementID),
			OverflowClip:        elementOverflowClip(boxNode),
			HasOverflowClip:     boxNode.overflowClip != nil,
			Transform:           info,
			X:                   boxNode.x,
			Y:                   boxNode.y,
			W:                   boxNode.w,
			H:                   boxNode.height,
		}

		if opts.IncludeText {
			item.Text = boxNode.node.TextContent()
		}

		childParent = index
		index++

		fn(item)
	}

	// An <svg> box establishes a viewport for its subtree; the box's own
	// transform resolves against the ancestor viewport above, matching the
	// stamp walk.
	childViewport := childViewportForBox(boxNode, viewport)

	for _, child := range boxNode.children {
		index = walkPlacedFunc(child, ops, childParent, childAccum, childViewport, fn, allIDs, index, item, opts)
	}

	return index
}

// placedTransformInfo resolves the transform metadata for one box while
// computing the box bake once: Own is the single boxTransformAccum call and
// World is the parent product of it, which is the same multiplication
// boxTransformAccum performs internally. Boxes without a static transform
// skip the bake entirely and hand the parent matrix through.
func placedTransformInfo(boxNode *box, parentAccum Matrix2D, viewport svgViewport) TransformInfo {
	if boxNode == nil || boxNode.style == nil {
		return TransformInfo{} //nolint:exhaustruct // no box, no metadata
	}

	sty := boxNode.style
	originX, originY := resolveTransformOriginWithBox(sty.TransformOrigin, boxNode, sty.TransformBox, viewport)

	own := IdentityMatrix()
	world := parentAccum

	if sty.HasTransform {
		own = boxTransformAccum(boxNode, IdentityMatrix(), viewport)
		world = parentAccum.Mul(own)
	}

	info := TransformInfo{
		OriginX:            originX,
		OriginY:            originY,
		Box:                sty.TransformBox,
		Parent:             parentAccum,
		Own:                own,
		World:              world,
		HasMatrix:          sty.HasTransform,
		HasTransform3D:     sty.HasTransform3D,
		Perspective:        0,
		PerspectiveOriginX: 0,
		PerspectiveOriginY: 0,
		HasPerspective:     sty.HasPerspective,
		Style:              sty.TransformStyle,
		Backface:           sty.BackfaceVisibility,
	}

	if sty.HasPerspective {
		info.Perspective = sty.PerspectiveDist
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

// elementOverflowClip reads the ancestor overflow box clipOverflowTree
// recorded, in canvas points. Zero with no clip.
func elementOverflowClip(boxNode *box) [4]float64 {
	if boxNode == nil || boxNode.overflowClip == nil {
		return [4]float64{}
	}

	c := boxNode.overflowClip

	return [4]float64{c.x, c.y, c.w, c.h}
}

//nolint:gochecknoglobals // non-nil empty slice for elements without operation ownership
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
