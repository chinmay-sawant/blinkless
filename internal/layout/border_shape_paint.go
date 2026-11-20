package layout

import (
	"math"
	"strings"
)

// borderShapeSegments is how many straight segments trace a full ellipse.
const borderShapeSegments = 72

// borderShapeTurn is one full turn in radians and borderShapeStart is the
// quarter turn back that puts the first point at the top.
const (
	borderShapeTurn  = 2 * math.Pi
	borderShapeStart = math.Pi / 2
)

// borderShapeToken returns the one basic shape of a border-shape value, or
// false for two shapes or none.
func borderShapeToken(raw string) (string, bool) {
	tokens, split := shapesSplitTop(raw)
	if !split {
		return "", false
	}

	found := ""
	count := 0

	for _, token := range tokens {
		if _, isShape := shapesShapeToken(token); isShape {
			count++

			if found == "" {
				found = token
			}
		}
	}

	return found, count == 1
}

// borderShapeOutline reads the box's border-shape and returns the outline of
// its first basic shape, resolved against the border box. ok is false for
// `none`, for a shape this tree does not trace (polygon, path, xywh, rect,
// shape), and for two shapes: the two-shape band is not painted, so the box
// keeps its ordinary border. The geometry box is read as the border box.
func borderShapeOutline(eng *engine, sty ResolvedStyle, posX, posY, width, height float64) ([]DisplayPoint, bool) {
	raw := strings.TrimSpace(sty.declared().BorderShape)
	if raw == "" || !(width > 0 && height > 0) {
		return nil, false
	}

	token, single := borderShapeToken(raw)
	if !single {
		return nil, false
	}

	name, args, parsed := splitShapeFunction(token)
	if !parsed {
		return nil, false
	}

	var shape *shapeExclusion

	switch name {
	case listStyleCircle:
		shape = resolveCircleExclusion(args, posX, posY, width, height, 0, sty.FontSize, eng.scale)
	case shapeEllipseKeyword:
		shape = resolveEllipseExclusion(args, posX, posY, width, height, 0, sty.FontSize, eng.scale)
	case insetKeyword:
		shape = resolveInsetExclusion(args, posX, posY, width, height, 0, sty.FontSize, eng.scale)
	}

	if shape == nil {
		return nil, false
	}

	return shapeOutlinePoints(shape), true
}

// shapeOutlinePoints traces a resolved shape clockwise from its top.
func shapeOutlinePoints(shape *shapeExclusion) []DisplayPoint {
	if shape.kind == shapeKindInset {
		return []DisplayPoint{
			{X: shape.x0, Y: shape.y0}, {X: shape.x1, Y: shape.y0},
			{X: shape.x1, Y: shape.y1}, {X: shape.x0, Y: shape.y1},
		}
	}

	points := make([]DisplayPoint, 0, borderShapeSegments)

	for step := range borderShapeSegments {
		angle := borderShapeTurn*float64(step)/borderShapeSegments - borderShapeStart

		points = append(points, DisplayPoint{
			X: shape.cx + shape.rx*math.Cos(angle),
			Y: shape.cy + shape.ry*math.Sin(angle),
		})
	}

	return points
}

// borderShapePaint replaces the background fill and the border stroke with
// path ops that follow the border-shape outline. The stroke needs four equal
// sides, as the corner-shape stroke does. A box whose border-shape is none,
// untraced, or two shapes returns no ops and no handled flags.
func borderShapePaint(
	eng *engine, boxNode *box, sty ResolvedStyle, posX, posY, width, height float64,
) ([]Op, []Op, bool, bool) {
	if eng == nil || sty.declared().BorderShape == "" {
		return nil, nil, false, false
	}

	points, ok := borderShapeOutline(eng, sty, posX, posY, width, height)
	if !ok {
		return nil, nil, false, false
	}

	commands := cornerShapePathCommands(points)

	var fillOps []Op

	fillHandled := false

	if sty.BGColor[3] > 0 && eng.backgroundPaintEnabled(&sty) {
		usedBG := usedBGForPaintActive(sty, eng.forcedColorsForPaint())
		fillOp := Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpPath, X: posX, Y: posY, W: width, H: height,
			R: usedBG[0], G: usedBG[1], B: usedBG[2], Alpha: usedBG[3],
		}
		fillOp.setCornerPath(commands, false)

		fillOps, fillHandled = []Op{fillOp}, true
	}

	_ = boxNode

	if sty.BorderImageSource != "" {
		return fillOps, nil, fillHandled, false
	}

	base, uniform := cornerShapePaintUniformBorder(sty)
	if !uniform {
		return fillOps, nil, fillHandled, false
	}

	borderOp := Op{ //nolint:exhaustruct // intentional zero fields
		Kind: OpPath, X: posX, Y: posY, W: width, H: height,
		R: base.Color[0], G: base.Color[1], B: base.Color[2],
		Width: eng.scalePt(borderPaint(base)),
	}
	borderOp.setCornerPath(commands, true)

	return fillOps, []Op{borderOp}, fillHandled, true
}
