package layout

import (
	"math"
	"strconv"
	"strings"
)

// Grammar of the six motion path properties, CSS Motion Path 1.

// motionAxis is one coordinate: frac of the reference size plus an offset.
type motionAxis struct{ frac, off float64 }

// at resolves the axis against a reference size.
func (a motionAxis) at(size float64) float64 { return a.frac*size + a.off }

// axisFrac is an axis that is a fraction of the reference size.
func axisFrac(frac float64) motionAxis { return motionAxis{frac, 0} }

// axisOff is an axis that is a fixed offset.
func axisOff(off float64) motionAxis { return motionAxis{0, off} }

// motionPos is a <position>.
type motionPos struct{ x, y motionAxis }

// noMotionPos is the zero position.
func noMotionPos() motionPos { return motionPos{axisFrac(0), axisFrac(0)} }

// motionLen is a <length-percentage>. pt is in points.
type motionLen struct {
	pt  float64
	pct float64
	isP bool
}

// lenPt is a length in points.
func lenPt(pt float64) motionLen { return motionLen{pt, 0, false} }

// lenPct is a percentage.
func lenPct(pct float64) motionLen { return motionLen{0, pct, true} }

// resolve turns the length into points against total.
func (l motionLen) resolve(total float64) float64 {
	if l.isP {
		return total * l.pct / motionPercent
	}

	return l.pt
}

// motionShape is an <offset-path> value other than none. Raw holds the
// function arguments, so a shape is rebuilt against each box size.
type motionShape struct {
	kind string // path, ray, circle, ellipse, inset, polygon
	raw  string
}

// noMotionShape is the empty shape.
func noMotionShape() motionShape { return motionShape{"", ""} }

// motionRotate is an offset-rotate value.
type motionRotate struct {
	auto, reverse bool
	angle         float64
}

// noMotionRotate is the zero rotation.
func noMotionRotate() motionRotate { return motionRotate{false, false, 0} }

// motionSpec is every motion value of one element, longhands over shorthand.
type motionSpec struct {
	hasPath    bool
	shape      motionShape
	distSet    bool
	dist       motionLen
	rotSet     bool
	rot        motionRotate
	anchorSet  bool // false or "auto": use transform-origin
	anchor     motionPos
	posSet     bool // false for normal and auto
	pos        motionPos
	pathCleard bool // an explicit offset-path: none
}

// Numbers the grammar uses by name.
const (
	motionNormFontSize = 12.0
	motionPercent      = 100
	motionHalf         = 0.5
	motionTwoTokens    = 2
	motionThreeTokens  = 3
	motionFourTokens   = 4
	motionNormBox      = 100
)

// motionCoordBox reports whether word is a <coord-box> keyword.
func motionCoordBox(word string) bool {
	switch word {
	case shapeBoxContent, shapeBoxPadding, shapeBoxBorder, shapeBoxMargin,
		geometryFillBox, geometryStrokeBox, geometryViewBox:
		return true
	}

	return false
}

// isMotionLengthToken reports whether tok looks like a length or percentage.
func isMotionLengthToken(tok string) bool {
	if tok == "0" {
		return true
	}

	if tok == "" {
		return false
	}

	last := tok[len(tok)-1]

	return (last >= 'a' && last <= 'z') || (last >= 'A' && last <= 'Z') || last == '%'
}

// parseMotionLen reads one <length-percentage>.
func parseMotionLen(tok string, fsize float64) (motionLen, bool) {
	tok = strings.TrimSpace(tok)
	if !isMotionLengthToken(tok) {
		return lenPt(0), false
	}

	if strings.HasSuffix(tok, "%") {
		pct, err := strconv.ParseFloat(strings.TrimSuffix(tok, "%"), 64)

		return lenPct(pct), err == nil
	}

	if tok == "0" {
		return lenPt(0), true
	}

	pt, ok := plainLength(tok, fsize, 0)

	return lenPt(pt), ok
}

// parseMotionAngle reads one <angle> in degrees.
func parseMotionAngle(tok string) (float64, bool) {
	tok = strings.ToLower(strings.TrimSpace(tok))
	if tok == "0" {
		return 0, true
	}

	if !strings.HasSuffix(tok, "deg") && !strings.HasSuffix(tok, "rad") &&
		!strings.HasSuffix(tok, "grad") && !strings.HasSuffix(tok, "turn") {
		return 0, false
	}

	rad, ok := parseCSSAngle(tok)

	return rad * 180 / math.Pi, ok
}

// lenAxis turns a length into an axis.
func lenAxis(length motionLen) motionAxis {
	if length.isP {
		return axisFrac(length.pct / motionPercent)
	}

	return axisOff(length.pt)
}

// keywordFrac maps a position keyword to its fraction of the box.
func keywordFrac(tok string) (float64, bool) {
	switch tok {
	case "left", "top":
		return 0, true
	case "center":
		return motionHalf, true
	case "right", "bottom":
		return 1, true
	}

	return 0, false
}

// parseMotionPosition reads a CSS <position> of one to four tokens.
func parseMotionPosition(toks []string, fsize float64) (motionPos, bool) {
	for i := range toks {
		toks[i] = strings.ToLower(toks[i])
	}

	center := axisFrac(motionHalf)

	switch len(toks) {
	case 1:
		return parsePos1(toks[0], fsize, center)
	case motionTwoTokens:
		return parsePos2(toks, fsize)
	case motionThreeTokens, motionFourTokens:
		return parsePosEdges(toks, fsize)
	}

	return noMotionPos(), false
}

// parsePos1 reads the one token form of a position.
func parsePos1(tok string, fsize float64, center motionAxis) (motionPos, bool) {
	switch tok {
	case "top", "bottom":
		frac, _ := keywordFrac(tok)

		return motionPos{center, axisFrac(frac)}, true
	case "left", "right", "center":
		frac, _ := keywordFrac(tok)

		return motionPos{axisFrac(frac), center}, true
	}

	length, ok := parseMotionLen(tok, fsize)

	return motionPos{lenAxis(length), center}, ok
}

// parsePos2 reads the two token form of a position.
func parsePos2(toks []string, fsize float64) (motionPos, bool) {
	first, second := toks[0], toks[1]
	if first == "top" || first == "bottom" || second == "left" || second == "right" {
		first, second = second, first
	}

	if first == "top" || first == "bottom" || second == "left" || second == "right" {
		return noMotionPos(), false
	}

	xAxis, okX := axisOf(first, fsize)
	yAxis, okY := axisOf(second, fsize)

	return motionPos{xAxis, yAxis}, okX && okY
}

// axisOf reads one keyword or length as an axis.
func axisOf(tok string, fsize float64) (motionAxis, bool) {
	if frac, ok := keywordFrac(tok); ok {
		return axisFrac(frac), true
	}

	length, ok := parseMotionLen(tok, fsize)

	return lenAxis(length), ok
}

// posEdgeState tracks which axes of an edge position are already set.
type posEdgeState struct {
	pos          motionPos
	seenX, seenY bool
}

// place stores the axis of keyword kw. It returns false when the axis is
// already set or a center keyword carries an offset.
func (st *posEdgeState) place(kw string, axis motionAxis, hasOffset bool) bool {
	switch kw {
	case "left", "right":
		if st.seenX {
			return false
		}

		st.pos.x, st.seenX = axis, true
	case "top", "bottom":
		if st.seenY {
			return false
		}

		st.pos.y, st.seenY = axis, true
	default: // center
		if hasOffset {
			return false
		}

		switch {
		case !st.seenX:
			st.pos.x, st.seenX = axis, true
		case !st.seenY:
			st.pos.y, st.seenY = axis, true
		default:
			return false
		}
	}

	return true
}

// edgeOffsetAfter reads the optional offset that follows the keyword at
// toks[idx]. used reports whether it took the next token; ok is false for a
// bad offset.
func edgeOffsetAfter(toks []string, idx int, fsize float64) (*motionLen, bool, bool) {
	if idx+1 >= len(toks) {
		return nil, false, true
	}

	if _, isKw := keywordFrac(toks[idx+1]); isKw {
		return nil, false, true
	}

	length, good := parseMotionLen(toks[idx+1], fsize)
	if !good {
		return nil, false, false
	}

	return &length, true, true
}

// parsePosEdges reads the three and four token forms: edge keyword with an
// optional offset, such as "right 10px bottom 20%".
func parsePosEdges(toks []string, fsize float64) (motionPos, bool) {
	state := posEdgeState{pos: motionPos{axisFrac(motionHalf), axisFrac(motionHalf)}, seenX: false, seenY: false}

	for idx := 0; idx < len(toks); idx++ {
		keyword := toks[idx]
		if _, ok := keywordFrac(keyword); !ok {
			return noMotionPos(), false
		}

		off, used, ok := edgeOffsetAfter(toks, idx, fsize)
		if !ok {
			return noMotionPos(), false
		}

		if used {
			idx++
		}

		if !state.place(keyword, edgeAxis(keyword, off), off != nil) {
			return noMotionPos(), false
		}
	}

	return state.pos, true
}

// edgeAxis builds the axis of an edge keyword with an optional offset.
func edgeAxis(keyword string, off *motionLen) motionAxis {
	frac, _ := keywordFrac(keyword)
	axis := axisFrac(frac)

	if off == nil {
		return axis
	}

	sign := 1.0
	if keyword == "right" || keyword == "bottom" {
		sign = -1
	}

	if off.isP {
		axis.frac += sign * off.pct / motionPercent
	} else {
		axis.off = sign * off.pt
	}

	return axis
}

// parseMotionRotate reads [ auto | reverse ] || <angle>.
func parseMotionRotate(toks []string) (motionRotate, bool) {
	rot := noMotionRotate()

	haveKind, haveAngle := false, false

	for _, tok := range toks {
		switch low := strings.ToLower(tok); {
		case (low == "auto" || low == "reverse") && !haveKind:
			rot.auto, rot.reverse, haveKind = true, low == "reverse", true
		default:
			deg, ok := parseMotionAngle(low)
			if !ok || haveAngle {
				return noMotionRotate(), false
			}

			rot.angle, haveAngle = deg, true
		}
	}

	return rot, haveKind || haveAngle
}

// parseMotionShape reads one <offset-path> function.
func parseMotionShape(tok string) (motionShape, bool) {
	for _, kind := range []string{shapeKindPath, "ray", "circle", "ellipse", "inset", "polygon"} {
		if raw, ok := funcArgs(tok, kind); ok {
			shape := motionShape{kind, raw}
			_, built := shape.build(motionNormBox, motionNormBox, motionNormFontSize)

			return shape, built
		}
	}

	return noMotionShape(), false
}

// pathTokenState tracks what parseMotionPathTokens has seen.
type pathTokenState struct {
	shape              motionShape
	haveShape, haveBox bool
}

// take reads one token. It returns false when the token breaks the grammar.
func (st *pathTokenState) take(tok string) bool {
	switch {
	case motionCoordBox(strings.ToLower(tok)) && !st.haveBox:
		st.haveBox = true
	case !st.haveShape:
		shape, good := parseMotionShape(tok)
		if !good {
			return false
		}

		st.shape, st.haveShape = shape, true
	default:
		return false
	}

	return true
}

// parseMotionPathTokens reads none | <offset-path> || <coord-box>. It returns
// the shape, whether the value is none, and whether it parsed.
func parseMotionPathTokens(toks []string) (motionShape, bool, bool) {
	if len(toks) == 1 && strings.EqualFold(toks[0], fragNone) {
		return noMotionShape(), true, true
	}

	var state pathTokenState

	for _, tok := range toks {
		if !state.take(tok) {
			return noMotionShape(), false, false
		}
	}

	if !state.haveShape && !state.haveBox {
		return noMotionShape(), false, false
	}

	return state.shape, !state.haveShape, true
}

// parseMotionPositionValue reads normal | auto | <position>. It returns the
// position, whether one is set, and whether the value parsed.
func parseMotionPositionValue(toks []string, fsize float64) (motionPos, bool, bool) {
	if len(toks) == 1 {
		switch strings.ToLower(toks[0]) {
		case "normal", "auto":
			return noMotionPos(), false, true
		}
	}

	pos, good := parseMotionPosition(toks, fsize)

	return pos, good, good
}

// offsetPathIndex returns the index of the first token that starts the path
// part of the offset shorthand, or -1.
func offsetPathIndex(toks []string) int {
	for idx, tok := range toks {
		low := strings.ToLower(tok)
		if low == fragNone || motionCoordBox(low) || strings.Contains(tok, "(") {
			return idx
		}
	}

	return -1
}

// parseOffsetShorthand reads the offset shorthand into spec.
func parseOffsetShorthand(value string, fsize float64) (motionSpec, bool) {
	var spec motionSpec

	head, anchor, hasAnchor := strings.Cut(value, "/")
	if hasAnchor {
		pos, set, ok := parseAnchorValue(splitTop(anchor), fsize)
		if !ok {
			return spec, false
		}

		spec.anchor, spec.anchorSet = pos, set
	}

	toks := splitTop(head)
	if len(toks) == 0 {
		return spec, false
	}

	pathAt := offsetPathIndex(toks)
	posToks := toks

	if pathAt >= 0 {
		posToks = toks[:pathAt]
	}

	if len(posToks) > 0 {
		pos, set, ok := parseMotionPositionValue(append([]string(nil), posToks...), fsize)
		if !ok {
			return spec, false
		}

		spec.pos, spec.posSet = pos, set
	}

	if pathAt < 0 {
		return spec, len(posToks) > 0
	}

	return parseShorthandPath(spec, toks[pathAt:], fsize)
}

// parseShorthandPath reads the path, distance and rotate tail of the offset
// shorthand.
func parseShorthandPath(spec motionSpec, toks []string, fsize float64) (motionSpec, bool) {
	count := 1
	if len(toks) > 1 && motionCoordBox(strings.ToLower(toks[1])) && !motionCoordBox(strings.ToLower(toks[0])) {
		count = 2
	}

	shape, none, ok := parseMotionPathTokens(toks[:count])
	if !ok {
		return spec, false
	}

	spec.hasPath, spec.shape, spec.pathCleard = !none, shape, none

	rotToks := make([]string, 0, len(toks)-count)

	for _, tok := range toks[count:] {
		low := strings.ToLower(tok)
		if length, isLen := parseMotionLen(low, fsize); isLen && !spec.distSet {
			spec.dist, spec.distSet = length, true

			continue
		}

		rotToks = append(rotToks, low)
	}

	if len(rotToks) > 0 {
		rot, good := parseMotionRotate(rotToks)
		if !good {
			return spec, false
		}

		spec.rot, spec.rotSet = rot, true
	}

	return spec, true
}

// parseAnchorValue reads auto | <position>. It returns the position, whether
// one is set, and whether the value parsed.
func parseAnchorValue(toks []string, fsize float64) (motionPos, bool, bool) {
	if len(toks) == 1 && strings.EqualFold(toks[0], "auto") {
		return noMotionPos(), false, true
	}

	pos, good := parseMotionPosition(append([]string(nil), toks...), fsize)

	return pos, good, good
}

// normalizeMotionText collapses white space and lowercases bare keywords. A
// token holding a function keeps its case, since path data is case sensitive.
func normalizeMotionText(value string) string {
	toks := splitTop(strings.TrimSpace(value))
	for i, tok := range toks {
		if !strings.ContainsAny(tok, "('\"") {
			toks[i] = strings.ToLower(tok)
		}
	}

	return strings.Join(toks, " ")
}
