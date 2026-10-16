package layout

import "fmt"

// DisplayPoint is one point in canvas coordinates.
type DisplayPoint struct {
	X, Y float64
}

// DisplayPath is a straight-edged path emitted when a display-list shape no
// longer fits the axis-aligned rectangle fields.
type DisplayPath struct {
	Points []DisplayPoint
	Closed bool
}

// GlyphBasis describes the local direction and scale of a projected text run.
// Its values use the display list's y-down canvas coordinates.
type GlyphBasis struct {
	A, B, C, D float64
}

// FinalizeDisplayOps bakes supported 3D projection into display geometry.
// Consumers receive paths and final text positions instead of CSS projection
// matrices for those operations.
func FinalizeDisplayOps(ops []Op) error {
	for idx := range ops {
		op := &ops[idx]
		projective, ok := op.ProjectiveTransformValue()
		if !ok {
			continue
		}

		switch op.Kind {
		case OpFillRect, OpStrokeRect:
			points := projectedRectPath(*op, projective)
			op.setDisplayPath(points, true)
			setProjectedBounds(op, points)
		case OpLine:
			finalizeProjectedLine(op, projective)
		case OpText, OpBullet:
			finalizeProjectedText(op, projective)
		case OpLinkURI:
			finalizeProjectedLink(op, projective)
		case OpGridRun:
			finalizeProjectedGrid(op, projective)
		case OpUnknown, OpKindNoop:
			// Group boundary and deactivated operations carry no geometry.
		case OpImage:
			return fmt.Errorf("layout: perspective image cannot be emitted as vector geometry")
		default:
			return fmt.Errorf("layout: cannot finalize perspective operation kind %d", op.Kind)
		}

		// The projective matrix already includes the operation's affine
		// transforms. Do not let a downstream painter apply them a second time.
		op.XformSet = false
		op.clearProjectiveTransform()
	}

	return nil
}

func projectedRectPath(op Op, projective ProjectiveTransform) []DisplayPoint {
	points := []DisplayPoint{
		{X: op.X, Y: op.Y},
		{X: op.X + op.W, Y: op.Y},
		{X: op.X + op.W, Y: op.Y + op.H},
		{X: op.X, Y: op.Y + op.H},
	}
	for idx := range points {
		points[idx].X, points[idx].Y = projective.Apply(points[idx].X, points[idx].Y)
	}

	return points
}

func setProjectedBounds(op *Op, points []DisplayPoint) {
	if len(points) == 0 {
		return
	}

	left, right := points[0].X, points[0].X
	top, bottom := points[0].Y, points[0].Y
	for _, point := range points[1:] {
		left = min(left, point.X)
		right = max(right, point.X)
		top = min(top, point.Y)
		bottom = max(bottom, point.Y)
	}
	op.X, op.Y = left, top
	op.W, op.H = right-left, bottom-top
}

func finalizeProjectedLine(op *Op, projective ProjectiveTransform) {
	lineX, lineY, lineW, lineH, width := op.PaintLineGeometry()
	startX, startY := projective.Apply(lineX, lineY)
	endX, endY := projective.Apply(lineX+lineW, lineY+lineH)
	op.X, op.Y = startX, startY
	op.W, op.H = endX-startX, endY-startY
	op.Width = width
	op.LineInset = 0
}

func finalizeProjectedText(op *Op, projective ProjectiveTransform) {
	oldX, oldY := op.X, op.Y
	op.X, op.Y = projective.Apply(oldX, oldY)
	op.setGlyphBasis(projective.Jacobian(oldX, oldY))
}

func finalizeProjectedLink(op *Op, projective ProjectiveTransform) {
	points := projectedRectPath(*op, projective)
	setProjectedBounds(op, points)
}

func finalizeProjectedGrid(op *Op, projective ProjectiveTransform) {
	if op.Grid == nil {
		return
	}

	var bounds []DisplayPoint
	for idx := range op.Grid.Segs {
		line := op.Grid.asLine(op, idx)
		lineX, lineY, lineW, lineH, width := line.PaintLineGeometry()
		startX, startY := projective.Apply(lineX, lineY)
		endX, endY := projective.Apply(lineX+lineW, lineY+lineH)
		segment := &op.Grid.Segs[idx]
		segment.X, segment.Y = startX, startY
		segment.W, segment.H = endX-startX, endY-startY
		segment.Width = width
		segment.LineInset = 0
		bounds = append(bounds, DisplayPoint{X: startX, Y: startY}, DisplayPoint{X: endX, Y: endY})
	}
	setProjectedBounds(op, bounds)
}
