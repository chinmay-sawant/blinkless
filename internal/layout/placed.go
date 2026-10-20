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
	X                   float64
	Y                   float64
	W                   float64
	H                   float64
}

// PlacedElements walks the laid-out element boxes in document order.
// A nil result returns nil. Paint and pagination are not required.
func PlacedElements(res *Result) []PlacedElement {
	if res == nil || res.root == nil {
		return nil
	}

	out := make([]PlacedElement, 0, len(res.boxes))
	walkPlaced(res.root, res.Ops, -1, &out)

	return out
}

func walkPlaced(boxNode *box, ops []Op, parentIndex int, out *[]PlacedElement) {
	if boxNode == nil {
		return
	}

	childParent := parentIndex

	if boxNode.node != nil && boxNode.node.Type == html.ElementNode {
		index := len(*out)
		*out = append(*out, PlacedElement{
			ID:                  boxNode.node.Attribute("id"),
			Tag:                 boxNode.node.Name,
			Action:              boxNode.node.Attribute("data-action"),
			Text:                boxNode.node.TextContent(),
			ParentIndex:         parentIndex,
			OperationIDs:        operationIDs(ops, boxNode.opStart, boxNode.opEnd),
			Animations:          animationRecordsFromStyle(boxNode.style.Animation),
			Transitions:         transitionRecordsFromStyle(boxNode.style.Transition),
			TriggerScope:        boxNode.style.Animation.TriggerScope,
			ViewTransitionName:  boxNode.style.Animation.ViewTransitionName,
			ViewTransitionClass: boxNode.style.Animation.ViewTransitionClass,
			ViewTransitionGroup: boxNode.style.Animation.ViewTransitionGroup,
			ViewTransitionScope: boxNode.style.Animation.ViewTransitionScope,
			X:                   boxNode.x,
			Y:                   boxNode.y,
			W:                   boxNode.w,
			H:                   boxNode.height,
		})
		childParent = index
	}

	for _, child := range boxNode.children {
		walkPlaced(child, ops, childParent, out)
	}
}

func operationIDs(ops []Op, start, end int) []uint64 {
	if start < 0 || end < start {
		return make([]uint64, 0)
	}

	ids := make([]uint64, 0, end-start+1)

	for index := start; index <= end && index < len(ops); index++ {
		if ops[index].ID != 0 {
			ids = append(ids, ops[index].ID)
		}
	}

	return ids
}
