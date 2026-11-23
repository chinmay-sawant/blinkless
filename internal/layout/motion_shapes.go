package layout

import (
	"math"
	"strings"
)

const pxToPtScale = 0.75

// Sizes the shape grammar uses by name.
const (
	// rayEpsilon is the smallest direction component a ray edge test trusts.
	rayEpsilon = 1e-9
	// roundEllipseRadii is the radius count of an ellipse.
	roundEllipseRadii = 2
	// insetMaxValues is the most values inset() takes before round.
	insetMaxValues = 4
)

// The four sides of an inset() value, in CSS order.
const (
	insetTop = iota
	insetRight
	insetBottom
	insetLeft
)

// build turns the shape into a path for a box of width by height points. A
// ray with no "at" starts at the box's top-left corner.
func (s motionShape) build(width, height, fsize float64) (motionPath, bool) {
	return s.buildAt(width, height, fsize, 0, 0)
}

// asPath turns a concrete path builder result into a motionPath. It keeps a
// failed build a true nil interface instead of a typed nil pointer.
func asPath[T motionPath](path T, ok bool) (motionPath, bool) {
	if !ok {
		return nil, false
	}

	return path, true
}

// buildAt is build with the start of a ray that names no position.
func (s motionShape) buildAt(width, height, fsize, startX, startY float64) (motionPath, bool) {
	switch s.kind {
	case shapeKindPath:
		return asPath(buildPathShape(s.raw))
	case "ray":
		return asPath(buildRay(s.raw, width, height, fsize, startX, startY))
	case "circle", "ellipse":
		return asPath(buildRound(s.kind, s.raw, width, height, fsize))
	case "inset":
		return asPath(buildInset(s.raw, width, height, fsize))
	case "polygon":
		return asPath(buildPolygon(s.raw, width, height, fsize))
	}

	return nil, false
}

// buildPathShape builds the polyline of path("...").
func buildPathShape(raw string) (*motionPoly, bool) {
	body := raw

	if head, rest, ok := strings.Cut(raw, ","); ok {
		switch strings.ToLower(strings.TrimSpace(head)) {
		case "nonzero", "evenodd":
			body = rest
		}
	}

	body = strings.TrimSpace(body)
	if len(body) < fragQuoteLen || (body[0] != '\'' && body[0] != '"') || body[len(body)-1] != body[0] {
		return nil, false
	}

	return parseMotionPathData(body[1:len(body)-1], pxToPtScale)
}

// rayTail reads the size keywords and the optional "at <position>" after the
// angle of ray(). It returns the ray origin.
func rayTail(tail []string, width, height, fsize, startX, startY float64) (float64, float64, bool) {
	for idx, tok := range tail {
		switch strings.ToLower(tok) {
		case "closest-side", "closest-corner", "farthest-side", "farthest-corner", "sides", "contain":
		case "at":
			pos, good := parseMotionPosition(lowerAll(tail[idx+1:]), fsize)
			if !good {
				return 0, 0, false
			}

			return pos.x.at(width), pos.y.at(height), true
		default:
			return 0, 0, false
		}
	}

	return startX, startY, true
}

// buildRay builds the ray of ray(<angle> ...).
func buildRay(raw string, width, height, fsize, startX, startY float64) (*motionRay, bool) {
	toks := splitTop(raw)
	if len(toks) == 0 {
		return nil, false
	}

	angle, angleOK := parseMotionAngle(toks[0])
	if !angleOK {
		return nil, false
	}

	originX, originY, tailOK := rayTail(toks[1:], width, height, fsize, startX, startY)
	if !tailOK {
		return nil, false
	}

	return &motionRay{
		x: originX, y: originY, angle: angle, span: rayBoxSpan(originX, originY, angle, width, height),
	}, true
}

// lowerAll lower-cases every string of in.
func lowerAll(in []string) []string {
	out := make([]string, len(in))
	for idx, text := range in {
		out[idx] = strings.ToLower(text)
	}

	return out
}

// rayBoxSpan is the distance from (fromX, fromY) to the box edge along the ray.
func rayBoxSpan(fromX, fromY, angle, width, height float64) float64 {
	rad := angle * math.Pi / degPerHalfTurn
	dirX, dirY := math.Sin(rad), -math.Cos(rad)
	span := math.Inf(1)

	if dirX > rayEpsilon {
		span = math.Min(span, (width-fromX)/dirX)
	} else if dirX < -rayEpsilon {
		span = math.Min(span, -fromX/dirX)
	}

	if dirY > rayEpsilon {
		span = math.Min(span, (height-fromY)/dirY)
	} else if dirY < -rayEpsilon {
		span = math.Min(span, -fromY/dirY)
	}

	if math.IsInf(span, 1) || span < 0 {
		return math.Max(width, height)
	}

	return span
}

// splitAt splits toks at the keyword "at". It returns the tokens before, the
// tokens after, and whether the keyword was there.
func splitAt(toks []string) ([]string, []string, bool) {
	for idx, tok := range toks {
		if strings.EqualFold(tok, "at") {
			return toks[:idx], toks[idx+1:], true
		}
	}

	return toks, nil, false
}

// roundRefs returns, for a circle or ellipse centered at (ctrX, ctrY) in a box,
// the reference sizes of percentages and the closest-side and farthest-side
// radii. A circle has one of each; an ellipse has two.
func roundRefs(kind string, ctrX, ctrY, width, height float64) ([]float64, []float64, []float64) {
	if kind == "ellipse" {
		return []float64{width, height},
			[]float64{math.Min(ctrX, width-ctrX), math.Min(ctrY, height-ctrY)},
			[]float64{math.Max(ctrX, width-ctrX), math.Max(ctrY, height-ctrY)}
	}

	return []float64{math.Sqrt((width*width + height*height) * motionHalf)},
		[]float64{math.Min(math.Min(ctrX, width-ctrX), math.Min(ctrY, height-ctrY))},
		[]float64{math.Max(math.Max(ctrX, width-ctrX), math.Max(ctrY, height-ctrY))}
}

// roundRadii resolves the radius tokens of a circle or ellipse.
func roundRadii(kind string, radii []string, ctrX, ctrY, width, height, fsize float64) ([]float64, bool) {
	refs, closest, farthest := roundRefs(kind, ctrX, ctrY, width, height)

	resolved := make([]float64, len(refs))
	copy(resolved, closest)

	for idx, tok := range radii {
		switch strings.ToLower(tok) {
		case "closest-side":
			resolved[idx] = closest[idx]
		case "farthest-side":
			resolved[idx] = farthest[idx]
		default:
			length, ok := parseMotionLen(tok, fsize)
			if !ok || (!length.isP && length.pt < 0) || (length.isP && length.pct < 0) {
				return nil, false
			}

			resolved[idx] = length.resolve(refs[idx])
		}
	}

	return resolved, true
}

// buildRound builds circle(...) or ellipse(...).
func buildRound(kind, raw string, width, height, fsize float64) (*motionPoly, bool) {
	radii, posToks, hasAt := splitAt(splitTop(raw))

	ctrX, ctrY := width*motionHalf, height*motionHalf

	if hasAt {
		pos, ok := parseMotionPosition(lowerAll(posToks), fsize)
		if !ok {
			return nil, false
		}

		ctrX, ctrY = pos.x.at(width), pos.y.at(height)
	}

	maxRadii := 1
	if kind == "ellipse" {
		maxRadii = roundEllipseRadii
	}

	if len(radii) > maxRadii {
		return nil, false
	}

	resolved, ok := roundRadii(kind, radii, ctrX, ctrY, width, height, fsize)
	if !ok {
		return nil, false
	}

	if kind == "circle" {
		return ellipsePoly(ctrX, ctrY, resolved[0], resolved[0]), true
	}

	return ellipsePoly(ctrX, ctrY, resolved[0], resolved[1]), true
}

// insetValueIndex maps a side to the index of the value that sets it, by the
// CSS one to four value rule.
func insetValueIndex(count, side int) int {
	switch side {
	case insetRight:
		return min(1, count-1)
	case insetBottom:
		if count > insetBottom {
			return insetBottom
		}
	case insetLeft:
		if count > insetLeft {
			return insetLeft
		}

		return min(1, count-1)
	}

	return 0
}

// buildInset builds inset(...); a round clause is ignored.
func buildInset(raw string, width, height, fsize float64) (*motionPoly, bool) {
	toks := splitTop(raw)

	for idx, tok := range toks {
		if strings.EqualFold(tok, "round") {
			toks = toks[:idx]

			break
		}
	}

	if len(toks) == 0 || len(toks) > insetMaxValues {
		return nil, false
	}

	for _, tok := range toks {
		if _, ok := parseMotionLen(tok, fsize); !ok {
			return nil, false
		}
	}

	edge := func(side int, size float64) float64 {
		return resolveEdge(toks[insetValueIndex(len(toks), side)], size, fsize)
	}

	top, right, bottom, left := edge(insetTop, height), edge(insetRight, width),
		edge(insetBottom, height), edge(insetLeft, width)

	pts := [][2]float64{
		{left, top}, {width - right, top}, {width - right, height - bottom}, {left, height - bottom},
	}

	return polyFromPoints(pts, true), true
}

// resolveEdge resolves one inset value against size.
func resolveEdge(tok string, size, fsize float64) float64 {
	length, _ := parseMotionLen(tok, fsize)

	return length.resolve(size)
}

// buildPolygon builds polygon(...).
func buildPolygon(raw string, width, height, fsize float64) (*motionPoly, bool) {
	parts := strings.Split(raw, ",")

	if len(parts) > 0 {
		switch strings.ToLower(strings.TrimSpace(parts[0])) {
		case "nonzero", "evenodd":
			parts = parts[1:]
		}
	}

	if len(parts) == 0 {
		return nil, false
	}

	pts := make([][2]float64, 0, len(parts))

	for _, part := range parts {
		coords := strings.Fields(part)
		if len(coords) != motionTwoTokens {
			return nil, false
		}

		xLen, okX := parseMotionLen(coords[0], fsize)
		yLen, okY := parseMotionLen(coords[1], fsize)

		if !okX || !okY {
			return nil, false
		}

		pts = append(pts, [2]float64{xLen.resolve(width), yLen.resolve(height)})
	}

	return polyFromPoints(pts, true), true
}
