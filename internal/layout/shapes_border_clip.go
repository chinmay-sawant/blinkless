package layout

import (
	"strings"
)

// borderSideTop and its three siblings name the physical edges.
const (
	borderSideTop = iota
	borderSideRight
	borderSideBottom
	borderSideLeft
)

// shapesSpan is one painted stretch of a border edge, measured in points from
// the start of the edge: the left end of a horizontal edge, the top end of a
// vertical one.
type shapesSpan struct{ from, to float64 }

// borderClipActive reports whether the box declares any border clip or limit.
func (d *declaredStyle) borderClipActive() bool {
	for _, value := range []string{
		d.BorderClip, d.BorderTopClip, d.BorderRightClip, d.BorderBottomClip, d.BorderLeftClip,
		d.BorderBlockClip, d.BorderBlockStartClip, d.BorderBlockEndClip, d.BorderInlineClip,
		d.BorderInlineStartClip, d.BorderInlineEndClip, d.BorderLimit,
	} {
		if value != "" {
			return true
		}
	}

	return false
}

// borderClipValue picks the clip list that applies to one physical edge. A
// physical longhand beats a logical longhand, which beats the axis shorthand,
// which beats border-clip. The cascade keeps no source order between these
// names, so this fixed precedence stands in for it. Writing mode is not read:
// block start is the top and inline start is the left.
func (d *declaredStyle) borderClipValue(side int) string {
	var physical, logical, axis string

	switch side {
	case borderSideTop:
		physical, logical, axis = d.BorderTopClip, d.BorderBlockStartClip, d.BorderBlockClip
	case borderSideBottom:
		physical, logical, axis = d.BorderBottomClip, d.BorderBlockEndClip, d.BorderBlockClip
	case borderSideLeft:
		physical, logical, axis = d.BorderLeftClip, d.BorderInlineStartClip, d.BorderInlineClip
	default:
		physical, logical, axis = d.BorderRightClip, d.BorderInlineEndClip, d.BorderInlineClip
	}

	for _, value := range []string{physical, logical, axis, d.BorderClip} {
		if value != "" {
			return value
		}
	}

	return ""
}

// shapesLengthPt resolves a length token to points. A percentage resolves
// against edge. A flex token is zero.
func shapesLengthPt(token string, edge, fontPt float64) float64 {
	number, unit, ok := shapesLength(token)
	if !ok {
		return 0
	}

	switch unit {
	case "%":
		return edge * number / 100 //nolint:mnd // percent
	case "em":
		return number * fontPt
	case remUnit:
		return number * pxToPt(16) //nolint:mnd // default root size
	}

	return shapesAbsolutePt(number, unit)
}

// shapesAbsolutePt converts a number of an absolute unit to points. A unit it
// does not know (a bare 0) is zero.
func shapesAbsolutePt(number float64, unit string) float64 {
	switch unit {
	case "px":
		return pxToPt(number)
	case "pt":
		return number
	case "pc":
		return number * 12 //nolint:mnd // points per pica
	case "in":
		return number * 72 //nolint:mnd // points per inch
	case "cm":
		return number * 72 / 2.54 //nolint:mnd // points per cm
	case "mm":
		return number * 72 / 25.4 //nolint:mnd // points per mm
	case "q":
		return number * 72 / 101.6 //nolint:mnd // points per quarter mm
	}

	return 0
}

// borderClipSpans turns one clip list into the painted spans of an edge that
// is edge points long. Parts alternate painted, skipped, painted. Fixed parts
// take their length, flex parts share what is left, and a list with no flex
// part gets an implicit trailing 1fr so the rest of the edge is painted.
func borderClipSpans(list string, edge, fontPt float64) []shapesSpan {
	if list == "none" {
		return []shapesSpan{{0, edge}}
	}

	fields := strings.Fields(list)
	sizes := make([]float64, len(fields), len(fields)+1)
	flex := make([]float64, len(fields), len(fields)+1)
	fixed, flexSum := 0.0, 0.0

	for index, field := range fields {
		if weight, ok := shapesFlex(field); ok {
			flex[index] = weight
			flexSum += weight

			continue
		}

		sizes[index] = shapesLengthPt(field, edge, fontPt)
		fixed += sizes[index]
	}

	if flexSum == 0 {
		sizes = append(sizes, 0)
		flex = append(flex, 1)
		flexSum = 1
	}

	free := max(edge-fixed, 0)

	var (
		spans  []shapesSpan
		cursor float64
	)

	for index := range sizes {
		length := sizes[index]
		if flex[index] > 0 {
			length = free * flex[index] / flexSum
		}

		if index%2 == 0 && length > 0 {
			spans = append(spans, shapesSpan{cursor, min(cursor+length, edge)})
		}

		cursor += length
		if cursor >= edge {
			break
		}
	}

	return spans
}

// borderLimitSpans turns border-limit into painted spans for one edge, or
// nil when the limit leaves the edge whole. `<side> <length>` keeps that much
// of every edge along the side's axis, measured from the named end. `sides
// <length>` keeps a centered stretch of that length. `corners <length>` keeps
// that much at each end. A missing length keeps the whole edge.
func borderLimitSpans(limit string, side int, edge, fontPt float64) []shapesSpan {
	fields := strings.Fields(limit)
	if len(fields) == 0 {
		return nil
	}

	horizontal := side == borderSideTop || side == borderSideBottom
	length := edge

	if len(fields) == 2 { //nolint:mnd // keyword plus length
		length = shapesLengthPt(fields[1], edge, fontPt)
		length = min(length, edge)
	}

	if borderLimitWrongAxis(fields[0], horizontal) {
		return nil
	}

	switch fields[0] {
	case "left", "top":
		return []shapesSpan{{0, length}}
	case "right", "bottom":
		return []shapesSpan{{edge - length, edge}}
	case borderLimitSides:
		return []shapesSpan{{(edge - length) / 2, (edge + length) / 2}}
	case "corners":
		if length*2 >= edge {
			return nil
		}

		return []shapesSpan{{0, length}, {edge - length, edge}}
	}

	return nil
}

// borderLimitWrongAxis reports whether the limit keyword names an axis the
// edge does not run along: left and right limit the horizontal edges, top and
// bottom the vertical ones.
func borderLimitWrongAxis(keyword string, horizontal bool) bool {
	switch keyword {
	case "left", "right":
		return !horizontal
	case "top", "bottom":
		return horizontal
	}

	return false
}

// shapesIntersect returns the stretches both lists paint.
func shapesIntersect(left, right []shapesSpan) []shapesSpan {
	var out []shapesSpan

	for _, a := range left {
		for _, b := range right {
			from, to := max(a.from, b.from), min(a.to, b.to)
			if to > from {
				out = append(out, shapesSpan{from, to})
			}
		}
	}

	return out
}

// borderEdgeSpans computes, for each edge of a box, the painted spans and
// whether the clip or limit cut that edge at all.
func borderEdgeSpans(sty *ResolvedStyle, wid, height float64) ([4][]shapesSpan, [4]bool) {
	decl := &sty.Declared

	var spans [4][]shapesSpan

	var cut [4]bool

	for side := range spans {
		edge := wid
		if side == borderSideRight || side == borderSideLeft {
			edge = height
		}

		spans[side] = []shapesSpan{{0, edge}}

		if list := decl.borderClipValue(side); list != "" && list != "none" {
			spans[side] = borderClipSpans(list, edge, sty.FontSize)
			cut[side] = true
		}

		if limit := borderLimitSpans(decl.BorderLimit, side, edge, sty.FontSize); limit != nil {
			spans[side] = shapesIntersect(spans[side], limit)
			cut[side] = true
		}
	}

	return spans, cut
}

// clipBorderOps cuts the border line ops appended since start according to
// the box's border clip and limit declarations. Ops outside that set, and
// boxes with no such declaration, pass through untouched.
func (e *engine) clipBorderOps(ops []Op, start int, sty *ResolvedStyle, posX, posY, wid, height float64) []Op {
	if !sty.Declared.borderClipActive() || start >= len(ops) {
		return ops
	}

	spans, cut := borderEdgeSpans(sty, wid, height)

	out := make([]Op, 0, len(ops)+4) //nolint:mnd // a few extra parts
	out = append(out, ops[:start]...)

	for index := start; index < len(ops); index++ {
		line := ops[index]

		side, ok := borderOpSide(line, posX, posY, wid, height)
		if !ok || !cut[side] {
			out = append(out, line)

			continue
		}

		out = appendClippedLine(out, line, side, spans[side], posX, posY)
	}

	return out
}

// borderEps is how far a line may sit from an edge and still belong to it.
const borderEps = 1e-6

// nearCoord reports whether two coordinates agree within borderEps.
func nearCoord(first, second float64) bool {
	return first-second < borderEps && second-first < borderEps
}

// borderOpSide names the edge a border line op belongs to.
func borderOpSide(line Op, posX, posY, wid, height float64) (int, bool) {
	if line.Kind != OpLine {
		return 0, false
	}

	switch {
	case line.H == 0 && line.W > 0:
		if nearCoord(line.Y, posY) {
			return borderSideTop, true
		}

		if nearCoord(line.Y, posY+height) {
			return borderSideBottom, true
		}
	case line.W == 0 && line.H > 0:
		if nearCoord(line.X, posX) {
			return borderSideLeft, true
		}

		if nearCoord(line.X, posX+wid) {
			return borderSideRight, true
		}
	}

	return 0, false
}

// appendClippedLine appends the parts of one border line op that fall inside
// the painted spans of its edge.
func appendClippedLine(out []Op, line Op, side int, spans []shapesSpan, posX, posY float64) []Op {
	horizontal := side == borderSideTop || side == borderSideBottom

	from, end := line.Y, line.Y+line.H
	origin := posY

	if horizontal {
		from, end = line.X, line.X+line.W
		origin = posX
	}

	for _, span := range spans {
		low, high := max(from, origin+span.from), min(end, origin+span.to)
		if high-low <= 0 {
			continue
		}

		part := line
		if horizontal {
			part.X, part.W = low, high-low
		} else {
			part.Y, part.H = low, high-low
		}

		out = append(out, part)
	}

	return out
}
