package layout

import (
	"math"
	"slices"

	ilayout "github.com/chinmay-sawant/blinkless/internal/layout"
)

// StickyConstraint describes viewport scrolling in CSS pixels. Insets and Set
// use top, right, bottom, left order. Bounds is the containing position box
// [x,y,width,height], including the effective margins. Viewport is false for
// nested scrolling, nested sticky boxes, or transformed/fixed ancestors.
type StickyConstraint struct {
	Insets   [4]float64
	Set      [4]bool
	Bounds   [4]float64
	Viewport bool
}

func stickyConstraintFrom(source *ilayout.StickyConstraint) *StickyConstraint {
	if source == nil {
		return nil
	}

	result := &StickyConstraint{
		Insets: [4]float64{}, Set: source.Set, Bounds: [4]float64{}, Viewport: source.Viewport,
	}

	for i := range result.Insets {
		result.Insets[i] = source.Insets[i] * ptToPx
		result.Bounds[i] = source.Bounds[i] * ptToPx
	}

	return result
}

// HasViewportSticky reports whether StickyView needs to follow viewport scroll.
// It does not claim nested overflow scrolling support.
func HasViewportSticky(display *Display) bool {
	if display == nil {
		return false
	}

	for _, box := range display.Boxes {
		if constraint := box.Sticky; constraint != nil && constraint.Viewport {
			for _, set := range constraint.Set {
				if set {
					return true
				}
			}
		}
	}

	return false
}

// StickyView derives paint and hit geometry for one viewport scroll position.
// Scroll and viewport dimensions are in CSS pixels before host zoom. The input
// display remains the authoritative layout and is never mutated. The result is
// read-only, borrows immutable layout payloads, and must not be used for element
// patches or retained motion writes. No offset returns the original display.
// Call again from the original after scrolling, resizing, or changing paint.
//
// The viewport contract supports ordinary horizontal-writing sticky boxes and
// their 2D painted descendants, with containing-block release and ancestor clips.
// Nested scroll containers, nested sticky elements, and transformed or fixed
// ancestors retain their normal layout geometry. A fixed descendant is excluded
// from its sticky ancestor's translation.
func StickyView(display *Display, scrollX, scrollY, viewportW, viewportH float64) *Display {
	if !HasViewportSticky(display) || !validStickyViewport(scrollX, scrollY, viewportW, viewportH) {
		return display
	}

	var view *Display

	for index, box := range display.Boxes {
		constraint := box.Sticky

		if constraint == nil || !constraint.Viewport {
			continue
		}

		deltaX := stickyAxis(
			box.X, box.W, scrollX, viewportW, constraint.Insets[3], constraint.Insets[1],
			constraint.Set[3], constraint.Set[1], constraint.Bounds[0], constraint.Bounds[2],
		)
		deltaY := stickyAxis(
			box.Y, box.H, scrollY, viewportH, constraint.Insets[0], constraint.Insets[2],
			constraint.Set[0], constraint.Set[2], constraint.Bounds[1], constraint.Bounds[3],
		)

		if deltaX == 0 && deltaY == 0 {
			continue
		}

		if view == nil {
			snapshot := *display
			view = &snapshot
			view.result = nil
			view.Ops = slices.Clone(display.Ops)
			view.Boxes = slices.Clone(display.Boxes)
		}

		translateSticky(view, display, index, deltaX, deltaY)
	}

	if view == nil {
		return display
	}

	return view
}

func validStickyViewport(scrollX, scrollY, width, height float64) bool {
	return width > 0 && height > 0 && finiteScroll(scrollX, scrollY, width, height)
}

func finiteScroll(values ...float64) bool {
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}

	return true
}

func stickyAxis(
	normal, size, scroll, viewport, before, after float64,
	beforeSet, afterSet bool, bound, boundSize float64,
) float64 {
	if !beforeSet && !afterSet {
		return 0
	}

	start, end := 0.0, 0.0

	if beforeSet {
		start = before
	}

	if afterSet {
		end = after
	}

	// The end inset shrinks when the sticky box exceeds its view rectangle.
	if viewport-start-end < size {
		end = viewport - start - size
	}

	position := normal

	if afterSet {
		position = math.Min(position, scroll+viewport-end-size)
	}

	if beforeSet {
		position = math.Max(position, scroll+start)
	}

	low, high := bound, bound+boundSize-size

	if high < low {
		high = low
	}

	position = math.Max(low, math.Min(position, high))

	return position - normal
}

func translateSticky(view, source *Display, root int, deltaX, deltaY float64) {
	ids := make(map[uint64]bool, len(source.Boxes[root].OperationIDs))

	for _, id := range source.Boxes[root].OperationIDs {
		ids[id] = true
	}

	translation := Matrix2D{A: 1, B: 0, C: 0, D: 1, E: deltaX * source.PointsPerPixel, F: deltaY * source.PointsPerPixel}
	translateStickyOps(view, source, ids, translation)
	translateStickyBoxes(view, source, root, deltaX, deltaY)
}

func translateStickyOps(view, source *Display, ids map[uint64]bool, translation Matrix2D) {
	for index, operation := range source.Ops {
		if !ids[operation.ID] || operation.Fixed {
			continue
		}

		matrix := IdentityMatrix()

		if operation.XformSet {
			matrix = operation.XformValue()
		}

		translated := operation.Clone()
		translated.SetXform(translation.Mul(matrix))
		view.Ops[index] = translated
	}
}

func translateStickyBoxes(view, source *Display, root int, deltaX, deltaY float64) {
	for index := root; index <= boxSubtreeEnd(source.Boxes, root); index++ {
		fixed := false

		for parent := index; parent >= root; parent = source.Boxes[parent].ParentIndex {
			if source.Boxes[parent].Fixed {
				fixed = true

				break
			}

			if parent == root {
				break
			}
		}

		if fixed {
			continue
		}

		view.Boxes[index].X += deltaX
		view.Boxes[index].Y += deltaY
	}
}
