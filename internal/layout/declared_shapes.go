package layout

import (
	"math"
	"strconv"
	"strings"
)

// shapesClipProp reports whether prop is one of the eleven border clip names.
// They share one grammar.
func shapesClipProp(prop string) bool {
	switch prop {
	case declPropBorderClip, declPropBorderTopClip, declPropBorderRightClip, declPropBorderBottomClip,
		declPropBorderLeftClip, declPropBorderBlockClip, declPropBorderBlockStartClip,
		declPropBorderBlockEndClip, declPropBorderInlineClip, declPropBorderInlineStartClip,
		declPropBorderInlineEndClip:
		return true
	}

	return false
}

// Names the shape grammars share.
const (
	geometryViewBox   = "view-box"
	geometryStrokeBox = "stroke-box"
	geometryFillBox   = "fill-box"
	shapeKindPath     = "path"
	shapeKindXywh     = "xywh"
)

// shapesGeometryBox reports whether word is a geometry-box keyword that
// border-shape accepts.
func shapesGeometryBox(word string) bool {
	switch word {
	case shapeBoxMargin, shapeBoxBorder, shapeBoxPadding, shapeBoxContent,
		geometryFillBox, geometryStrokeBox, geometryViewBox:
		return true
	}

	return false
}

// shapesBasicShape reports whether name is a basic-shape function name that
// border-shape accepts.
func shapesBasicShape(name string) bool {
	switch name {
	case "inset", "circle", "ellipse", "polygon", shapeKindPath, objectViewBoxRectShape, shapeKindXywh, "shape":
		return true
	}

	return false
}

// normalizeShapesDeclared checks and normalizes the value of one draft border
// or shape property. It returns the normalized text, or false when the value
// does not match the property's grammar. An initial value is stored as "",
// except for the border clips, where `none` must stay visible so a longhand
// can override a shorthand.
func normalizeShapesDeclared(prop, value string) (string, bool) {
	text := strings.TrimSpace(value)
	if text == "" {
		return "", false
	}

	if shapesClipProp(prop) {
		return normalizeBorderClip(text)
	}

	switch prop {
	case declPropBorderLimit:
		return normalizeBorderLimit(text)
	case "border-boundary":
		return shapesInitialKeyword(text, "none", "parent", "display")
	case "border-shape":
		return normalizeBorderShape(text)
	case declPropTextFit:
		return normalizeTextFit(text)
	case declPropFloatDefer:
		return normalizeFloatDefer(text)
	case "shape-image-threshold":
		return normalizeShapeImageThreshold(text)
	case "column-rule-visibility-items":
		return shapesInitialKeyword(text, "normal", "all", "around", "between")
	}

	return "", false
}

// shapesInitialKeyword accepts one keyword of the set. The first keyword is
// the initial value and normalizes to "".
func shapesInitialKeyword(text string, set ...string) (string, bool) {
	word := strings.ToLower(text)

	for index, keyword := range set {
		if word != keyword {
			continue
		}

		if index == 0 {
			return "", true
		}

		return word, true
	}

	return "", false
}

// shapesNumberByte reports whether char may sit at index of a number.
func shapesNumberByte(char byte, index int) bool {
	return (char >= '0' && char <= '9') || char == '.' || (index == 0 && (char == '+' || char == '-'))
}

// shapesNumberUnit splits "12.5px" into its number and lowercase unit.
func shapesNumberUnit(token string) (float64, string, bool) {
	token = strings.ToLower(token)
	end := 0

	for end < len(token) && shapesNumberByte(token[end], end) {
		end++
	}

	number, err := strconv.ParseFloat(token[:end], 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, "", false
	}

	return number, token[end:], true
}

// shapesLength parses one non-negative length or percentage. A bare 0 is a
// length. It returns the number and the unit ("" for the bare 0).
func shapesLength(token string) (float64, string, bool) {
	number, unit, ok := shapesNumberUnit(token)
	if !ok || number < 0 {
		return 0, "", false
	}

	if unit == "" {
		return number, "", number == 0
	}

	if shapesLengthUnit(unit) {
		return number, unit, true
	}

	return 0, "", false
}

// shapesFlex parses a `<flex>` token such as 2fr.
func shapesFlex(token string) (float64, bool) {
	number, unit, ok := shapesNumberUnit(token)

	return number, ok && unit == "fr" && number >= 0
}

// normalizeBorderClip accepts `none | [ <length-percentage [0,inf]> | <flex> ]+`.
func normalizeBorderClip(text string) (string, bool) {
	fields := strings.Fields(strings.ToLower(text))
	if len(fields) == 1 && fields[0] == "none" {
		return "none", true
	}

	for _, field := range fields {
		if _, _, ok := shapesLength(field); ok {
			continue
		}

		if _, ok := shapesFlex(field); ok {
			continue
		}

		return "", false
	}

	return strings.Join(fields, " "), true
}

// normalizeBorderLimit accepts
// `all | [ sides | corners ] <length-percentage>? | [ top | right | bottom | left ] <length-percentage>`.
func normalizeBorderLimit(text string) (string, bool) {
	fields := strings.Fields(strings.ToLower(text))

	switch fields[0] {
	case "all":
		if len(fields) == 1 {
			return "", true
		}
	case borderLimitSides, "corners":
		if len(fields) == 1 {
			return fields[0], true
		}

		if _, _, ok := shapesLength(fields[1]); ok && len(fields) == 2 {
			return strings.Join(fields, " "), true
		}
	case "top", "right", "bottom", "left":
		if _, _, ok := shapesLength(fieldAt(fields, 1)); ok && len(fields) == 2 {
			return strings.Join(fields, " "), true
		}
	}

	return "", false
}

// fieldAt returns fields[index] or "" past the end.
func fieldAt(fields []string, index int) string {
	if index < len(fields) {
		return fields[index]
	}

	return ""
}

// topSplitter collects fields split at top-level white space.
type topSplitter struct {
	out   []string
	depth int
	start int
}

// field opens or closes a field at byte offset index, depending on whether
// the rune is white space.
func (ts *topSplitter) field(text string, index int, space bool) {
	switch {
	case !space && ts.start < 0:
		ts.start = index
	case space && ts.depth == 0 && ts.start >= 0:
		ts.out = append(ts.out, text[ts.start:index])
		ts.start = -1
	}
}

// step takes one rune. It returns false when a ')' has no opener.
func (ts *topSplitter) step(text string, index int, char rune) bool {
	switch char {
	case '(':
		ts.depth++
	case ')':
		ts.depth--
		if ts.depth < 0 {
			return false
		}
	}

	ts.field(text, index, char == ' ' || char == '\t' || char == '\n')

	return true
}

// shapesSplitTop splits text at top-level spaces, keeping a parenthesized
// group whole. It returns false for unbalanced parentheses.
func shapesSplitTop(text string) ([]string, bool) {
	splitter := topSplitter{out: nil, depth: 0, start: -1}

	for index, char := range text {
		if !splitter.step(text, index, char) {
			return nil, false
		}
	}

	if splitter.depth != 0 {
		return nil, false
	}

	if splitter.start >= 0 {
		splitter.out = append(splitter.out, text[splitter.start:])
	}

	return splitter.out, true
}

// shapesShapeToken normalizes one basic-shape function token such as
// Circle(1px). ok is false when the token is not a known function.
func shapesShapeToken(token string) (string, bool) {
	open := strings.IndexByte(token, '(')
	if open > 0 && strings.HasSuffix(token, ")") && shapesBasicShape(strings.ToLower(token[:open])) {
		return strings.ToLower(token[:open]) + token[open:], true
	}

	return "", false
}

// normalizeBorderShape accepts `none | [ <basic-shape> <geometry-box>? ]{1,2}`.
func normalizeBorderShape(text string) (string, bool) {
	if strings.EqualFold(text, "none") {
		return "", true
	}

	tokens, ok := shapesSplitTop(text)
	if !ok || len(tokens) == 0 {
		return "", false
	}

	shapes := 0
	prevShape := false
	out := make([]string, 0, len(tokens))

	for _, token := range tokens {
		if shape, isShape := shapesShapeToken(token); isShape {
			shapes++
			prevShape = true

			out = append(out, shape)

			continue
		}

		box := strings.ToLower(token)
		if !shapesGeometryBox(box) || !prevShape {
			return "", false
		}

		prevShape = false

		out = append(out, box)
	}

	if shapes < 1 || shapes > 2 {
		return "", false
	}

	return strings.Join(out, " "), true
}

// textFitTail checks the optional mode and percentage that follow the first
// keyword of text-fit.
func textFitTail(rest []string) bool {
	if len(rest) > 0 && (rest[0] == "consistent" || rest[0] == "per-line" || rest[0] == "per-line-all") {
		rest = rest[1:]
	}

	switch len(rest) {
	case 0:
		return true
	case 1:
		number, unit, ok := shapesNumberUnit(rest[0])

		return ok && unit == "%" && number >= 0
	}

	return false
}

// normalizeTextFit accepts
// `[ none | grow | shrink ] [ consistent | per-line | per-line-all ]? <percentage>?`.
func normalizeTextFit(text string) (string, bool) {
	fields := strings.Fields(strings.ToLower(text))
	if fields[0] != "none" && fields[0] != "grow" && fields[0] != "shrink" {
		return "", false
	}

	if !textFitTail(fields[1:]) {
		return "", false
	}

	if len(fields) == 1 && fields[0] == "none" {
		return "", true
	}

	return strings.Join(fields, " "), true
}

// normalizeFloatDefer accepts `<integer> | last | none`.
func normalizeFloatDefer(text string) (string, bool) {
	word := strings.ToLower(text)

	switch word {
	case "none":
		return "", true
	case "last":
		return word, true
	}

	number, err := strconv.Atoi(strings.TrimPrefix(word, "+"))
	if err != nil {
		return "", false
	}

	return strconv.Itoa(number), true
}

// normalizeShapeImageThreshold accepts an opacity value: a number or a
// percentage, clamped to 0..1. Zero is the initial value and stores as "".
func normalizeShapeImageThreshold(text string) (string, bool) {
	number, unit, ok := shapesNumberUnit(text)
	if !ok || (unit != "" && unit != "%") {
		return "", false
	}

	if unit == "%" {
		number /= 100
	}

	number = math.Min(math.Max(number, 0), 1)
	if number == 0 {
		return "", true
	}

	return strconv.FormatFloat(number, 'g', 4, 64), true
}

// borderLimitSides is the keyword of border-limit that limits the four sides.
const borderLimitSides = "sides"

// shapesLengthUnit reports whether unit is one of the length-percentage units
// the draft properties accept.
func shapesLengthUnit(unit string) bool {
	switch unit {
	case "px", "pt", "pc", "in", "cm", "mm", "q", remUnit, "em", "%":
		return true
	}

	return false
}
