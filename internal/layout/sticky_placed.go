package layout

import "math"

// StickyConstraint retains geometry in canvas points. Viewport is false when
// a nested scroll container, transform, fixed ancestor, or sticky ancestor
// requires a scroll space the viewport-only host contract does not supply.
type StickyConstraint struct {
	Insets   [4]float64
	Set      [4]bool
	Bounds   [4]float64
	Viewport bool
}

type placedScrollContext struct {
	parent  *box
	blocked bool
	scale   float64
}

func childScrollContext(parent *box, context placedScrollContext) placedScrollContext {
	blocked := context.blocked

	if parent.style != nil {
		blocked = blocked || blocksViewportSticky(parent)
	}

	return placedScrollContext{parent: parent, blocked: blocked, scale: context.scale}
}

func blocksViewportSticky(parent *box) bool {
	style := parent.style
	if parent.sticky || style.Position == positionFixed ||
		style.HasTransform || style.HasTransform3D || style.HasPerspective {
		return true
	}

	if stickyRoot(parent) {
		return false
	}

	return stickyScrollContainer(style.OverflowX) || stickyScrollContainer(style.OverflowY) ||
		stickyScrollContainer(style.Overflow)
}

func stickyRoot(parent *box) bool {
	return parent.node != nil && (parent.node.Name == htmlRootName || parent.node.Name == "body")
}

func stickyScrollContainer(overflow string) bool {
	return overflow == contentVisibilityHidden || overflow == "auto" || overflow == timelineKindScroll
}

func placedSticky(node *box, context placedScrollContext) *StickyConstraint {
	if !node.sticky {
		return nil
	}

	scale := context.scale
	if scale == 0 {
		scale = 1
	}

	constraint := &StickyConstraint{
		Insets:   [4]float64{node.stickyTop, node.stickyRight, node.stickyBottom, node.stickyLeft},
		Set:      [4]bool{node.stickyTopSet, node.stickyRightSet, node.stickyBottomSet, node.stickyLeftSet},
		Bounds:   [4]float64{node.x, node.y, node.w, node.height},
		Viewport: !context.blocked,
	}

	if node.style.HasTransform || node.style.HasTransform3D || node.style.HasPerspective {
		constraint.Viewport = false
	}

	parent := context.parent
	if parent == nil || parent.style == nil {
		constraint.Viewport = false

		return constraint
	}

	style := parent.style
	left := (style.BorderLeft.Width + style.PaddingLeft) * scale
	top := (style.BorderTop.Width + style.PaddingTop) * scale
	right := (style.BorderRight.Width + style.PaddingRight) * scale
	bottom := (style.BorderBottom.Width + style.PaddingBottom) * scale
	originX, originY := parent.x+left, parent.y+top
	endX, endY := parent.x+parent.w-right, parent.y+parent.height-bottom
	// The position box uses margins only as far as the containing block has room.
	originX += math.Min(node.style.MarginLeft*scale, node.x-originX)
	originY += math.Min(node.style.MarginTop*scale, node.y-originY)
	endX -= math.Min(node.style.MarginRight*scale, endX-node.x-node.w)
	endY -= math.Min(node.style.MarginBottom*scale, endY-node.y-node.height)
	constraint.Bounds = [4]float64{originX, originY, math.Max(0, endX-originX), math.Max(0, endY-originY)}

	return constraint
}
