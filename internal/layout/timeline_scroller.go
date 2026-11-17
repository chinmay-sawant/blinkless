package layout

import "github.com/chinmay-sawant/blinkless/internal/html"

// placedScroller is the nearest ancestor scroll container found while walking
// the placed boxes. set is false when there is none, so the page scrolls.
type placedScroller struct {
	set bool
	id  string
}

// isScrollContainerBox reports whether the box is an element whose overflow
// makes it a scroll container: hidden, scroll, or auto on either axis. The
// root and body elements are not, because their overflow belongs to the
// viewport, which is the page scroller.
func isScrollContainerBox(boxNode *box) bool {
	if boxNode == nil || boxNode.style == nil || boxNode.node == nil || boxNode.node.Type != html.ElementNode {
		return false
	}

	switch boxNode.node.Name {
	case "html", "body":
		return false
	}

	return scrollContainerOverflow(boxNode.style.OverflowX) || scrollContainerOverflow(boxNode.style.OverflowY)
}

func scrollContainerOverflow(overflow string) bool {
	switch overflow {
	case overflowHidden, overflowScroll, overflowAuto:
		return true
	}

	return false
}

// fillTimelineScrollers names the scroller of each timeline an element
// exports. A scroll timeline reads the element's own scroll, so its scroller
// is the element when it is a scroll container, and the page otherwise. A view
// timeline reads the nearest ancestor scroll container, or the page when there
// is none. A scroller with no id attribute cannot be named, so the definition
// says so with ScrollerAnonymous. Pointer timelines have no scroller.
func fillTimelineScrollers(defs []TimelineDef, ownScroller bool, ownID string, nearest placedScroller) {
	for idx := range defs {
		switch defs[idx].Kind {
		case timelineKindScroll:
			if ownScroller {
				defs[idx].ScrollerID, defs[idx].ScrollerAnonymous = ownID, ownID == ""
			}
		case timelineKindView:
			if nearest.set {
				defs[idx].ScrollerID, defs[idx].ScrollerAnonymous = nearest.id, nearest.id == ""
			}
		}
	}
}

// placedTimelineDefs builds the timelines one element exports and names the
// scroller of each: nearest is the closest scroll container above the element.
func placedTimelineDefs(boxNode *box, elementID string, nearest placedScroller) []TimelineDef {
	defs := timelineDefsFromStyle(boxNode.style, elementID)
	fillTimelineScrollers(defs, isScrollContainerBox(boxNode), elementID, nearest)

	return defs
}

// scrollerBelow is the nearest scroll container for the children of boxNode:
// the box itself when it is a scroll container, and nearest otherwise.
func scrollerBelow(boxNode *box, nearest placedScroller) placedScroller {
	if isScrollContainerBox(boxNode) {
		return placedScroller{set: true, id: boxNode.node.Attribute("id")}
	}

	return nearest
}
