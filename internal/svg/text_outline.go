package svg

import (
	"bytes"
	"html"
	"math"
	"strconv"
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/fonts"
)

// SVG text is drawn as filled outlines. The canvas parser would load a system
// font for every <text>, with no size cap and a panic for a missing family, so
// the rasterizer never hands it text. outlineSVGText replaces each plain
// <text> element with <path> elements built from the bundled faces before the
// parse, and stripSVGTextElements still removes whatever it leaves behind.
//
// Covered: font-family, font-size, font-weight and font-style (own attributes,
// inline style and ancestors), x, y, dx, dy, text-anchor, tspan runs with their
// own position and fill, and baseline-shift. Not covered: textPath, vertical
// writing modes, rotate, textLength, kerning and ligatures, and font settings
// that arrive only through a <style> sheet. Those elements are left for the
// strip pass, so they stay absent exactly as before.

const (
	// maxOutlineGlyphs bounds the glyphs one document may outline.
	maxOutlineGlyphs = 20000
	// maxOutlineFontSize bounds a font size in user units.
	maxOutlineFontSize = 4000
	// defaultOutlineFontSize is the SVG initial font-size.
	defaultOutlineFontSize = 16
	// cssNormalWeight and cssBoldWeight are the CSS font-weight numbers.
	cssNormalWeight = 400
	cssBoldWeight   = 700
)

// textProps are the font settings one element passes down to its children.
type textProps struct {
	families []string
	size     float64
	weight   int
	italic   bool
	anchor   string
	vertical bool
}

// textRun is one stretch of text with the position and paint it carries.
type textRun struct {
	text      string
	props     textProps
	x, y      float64
	hasX      bool
	hasY      bool
	dx, dy    float64
	shift     float64
	fill      string
	chunkHead bool
}

// outlineSVGText rewrites <text> elements into outline paths. It returns the
// input unchanged when there is nothing to rewrite.
//
//nolint:cyclop // one scan loop over the tags of the document
func outlineSVGText(data []byte) []byte {
	if !bytesContainsFold(data, []byte("<text")) {
		return data
	}

	faces, err := fonts.LoadDefaultFaces()
	if err != nil || faces == nil {
		return data
	}

	var out bytes.Buffer

	budget := maxOutlineGlyphs
	changed := false
	stack := []textProps{{
		families: nil, size: defaultOutlineFontSize, weight: cssNormalWeight, italic: false, anchor: "start", vertical: false,
	}}
	flush, pos := 0, 0

	for pos < len(data) {
		rel := bytes.IndexByte(data[pos:], '<')
		if rel < 0 {
			break
		}

		start := pos + rel

		end, kind, local, ok := svgScanMarkup(data, start)
		if !ok {
			break
		}

		pos = end + 1

		switch {
		case kind == svgTagClose:
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		case kind == svgTagOpen && local == "text":
			stop, text := outlineTextElement(data, start, end, stack[len(stack)-1], &budget)
			if stop > 0 && text != "" {
				out.Write(data[flush:start])
				out.WriteString(text)

				flush, pos, changed = stop, stop, true
			}
		case kind == svgTagOpen:
			attrs := emulateAttrMap(emulateParseAttrs(data, start+1+len(local), end))
			stack = append(stack, textPropsFrom(stack[len(stack)-1], attrs))
		}
	}

	if !changed {
		return data
	}

	out.Write(data[flush:])

	return out.Bytes()
}

// textPropsFrom applies one element's font attributes and inline style on top
// of what its parent passes down.
//
//nolint:cyclop // one branch per font property
func textPropsFrom(parent textProps, attrs map[string]string) textProps {
	props := parent
	merged := map[string]string{}

	for key, value := range attrs {
		merged[strings.ToLower(key)] = value
	}

	for key, value := range styleDeclarations(merged["style"]) {
		merged[key] = value
	}

	if value := merged["font-family"]; value != "" {
		props.families = splitFontFamilies(value)
	}

	if value := merged["font-size"]; value != "" {
		if size, ok := parseFontSize(value, parent.size); ok {
			props.size = size
		}
	}

	if value := merged["font-weight"]; value != "" {
		props.weight = parseFontWeight(value, parent.weight)
	}

	if value := merged["font-style"]; value != "" {
		props.italic = value == "italic" || value == "oblique"
	}

	if value := merged["text-anchor"]; value != "" {
		props.anchor = strings.TrimSpace(value)
	}

	if value := merged["writing-mode"]; value != "" {
		mode := strings.TrimSpace(value)
		props.vertical = strings.HasPrefix(mode, "vertical") || strings.HasPrefix(mode, "tb") ||
			strings.HasPrefix(mode, "sideways")
	}

	return props
}

// styleDeclarations splits an inline style attribute into lower-case pairs.
func styleDeclarations(style string) map[string]string {
	out := map[string]string{}

	for _, part := range strings.Split(style, ";") {
		name, value, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}

		out[strings.ToLower(strings.TrimSpace(name))] = strings.TrimSpace(value)
	}

	return out
}

func splitFontFamilies(value string) []string {
	var out []string

	for _, family := range strings.Split(value, ",") {
		family = strings.Trim(strings.TrimSpace(family), `"'`)
		if family != "" {
			out = append(out, family)
		}
	}

	return out
}

// parseFontSize reads a length in user units. A percentage and em are relative
// to the parent size.
func parseFontSize(value string, parent float64) (float64, bool) {
	value = strings.ToLower(strings.TrimSpace(value))

	number, unit := splitNumberUnit(value)
	if math.IsNaN(number) || number < 0 {
		return 0, false
	}

	switch unit {
	case "", "px":
		return number, true
	case "pt":
		return number * 4 / 3, true //nolint:mnd // 96/72
	case "em":
		return number * parent, true
	case "%":
		return number / 100 * parent, true
	}

	return 0, false
}

// isNumberByte reports whether char can be part of a decimal number.
func isNumberByte(char byte) bool {
	return char == '.' || char == '-' || char == '+' || char >= '0' && char <= '9'
}

func splitNumberUnit(value string) (float64, string) {
	end := 0
	for end < len(value) && isNumberByte(value[end]) {
		end++
	}

	number, err := strconv.ParseFloat(value[:end], 64)
	if err != nil {
		return math.NaN(), ""
	}

	return number, strings.TrimSpace(value[end:])
}

func parseFontWeight(value string, parent int) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "bold", "bolder":
		return cssBoldWeight
	case "normal", "lighter":
		return cssNormalWeight
	}

	weight, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return parent
	}

	return weight
}

// outlineTextElement rewrites the <text> element opening at start. It returns
// the index just past the closing tag and the replacement markup, or 0 and ""
// when the element must stay for the strip pass.
func outlineTextElement(
	data []byte, start, end int, parent textProps, budget *int,
) (int, string) {
	attrs := emulateAttrMap(emulateParseAttrs(data, start+len("<text"), end))
	props := textPropsFrom(parent, attrs)

	if props.vertical {
		return 0, ""
	}

	runs, stop, ok := collectTextRuns(data, end+1, props, attrs)
	if !ok {
		return 0, ""
	}

	var body strings.Builder

	if !layoutTextRuns(&body, runs, budget) {
		return 0, ""
	}

	return stop, `<g` + groupAttributes(attrs) + `>` + body.String() + `</g>`
}

// groupAttributes keeps the paint and identity attributes of a <text> element
// on the group that replaces it. Position and font attributes are consumed by
// the layout and are dropped.
func groupAttributes(attrs map[string]string) string {
	var out strings.Builder

	for _, key := range sortedKeys(attrs) {
		switch strings.ToLower(key) {
		case "x", "y", "dx", "dy", "rotate", "textlength", "lengthadjust", "text-anchor", "font-family",
			"font-size", "font-weight", "font-style", "style", "xml:space", "baseline-shift", "writing-mode":
			continue
		}

		out.WriteString(` ` + key + `="` + html.EscapeString(attrs[key]) + `"`)
	}

	if style := paintStyle(attrs["style"]); style != "" {
		out.WriteString(` style="` + html.EscapeString(style) + `"`)
	}

	return out.String()
}

// paintStyle keeps the paint declarations of an inline style.
func paintStyle(style string) string {
	var out []string

	for name, value := range styleDeclarations(style) {
		switch name {
		case "fill", "stroke", "stroke-width", "opacity", "fill-opacity", "stroke-opacity", "fill-rule":
			out = append(out, name+":"+value)
		}
	}

	return strings.Join(out, ";")
}

func sortedKeys(attrs map[string]string) []string {
	keys := make([]string, 0, len(attrs))
	for key := range attrs {
		keys = append(keys, key)
	}

	for index := 1; index < len(keys); index++ {
		for j := index; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}

	return keys
}

// collectTextRuns reads the content of a <text> element up to its closing tag.
// Plain text and <tspan> children become runs. Any other child element makes
// the whole text element stay for the strip pass.
//
//nolint:cyclop,gocognit,funlen // one loop over the children of a text element
func collectTextRuns(data []byte, pos int, base textProps, textAttrs map[string]string) ([]textRun, int, bool) {
	type frame struct {
		props textProps
		shift float64
		fill  string
	}

	stack := []frame{{props: base, shift: 0, fill: ""}}

	var runs []textRun

	first := true
	pending := textRun{} //nolint:exhaustruct // positions of the next run

	if value, ok := textAttrs["x"]; ok {
		pending.x, pending.hasX = firstNumber(value)
	}

	if value, ok := textAttrs["y"]; ok {
		pending.y, pending.hasY = firstNumber(value)
	}

	if value, ok := textAttrs["dx"]; ok {
		pending.dx, _ = firstNumber(value)
	}

	if value, ok := textAttrs["dy"]; ok {
		pending.dy, _ = firstNumber(value)
	}

	pending.chunkHead = true
	pending.fill = ""

	for pos < len(data) {
		rel := bytes.IndexByte(data[pos:], '<')
		textEnd := len(data)

		if rel >= 0 {
			textEnd = pos + rel
		}

		if raw := string(data[pos:textEnd]); strings.TrimSpace(raw) != "" || len(raw) > 0 {
			run := pending
			run.text = html.UnescapeString(raw)
			run.props = stack[len(stack)-1].props
			run.shift = stack[len(stack)-1].shift
			run.fill = stack[len(stack)-1].fill

			if first || run.text != "" {
				runs = append(runs, run)
			}

			pending, first = textRun{}, false //nolint:exhaustruct // next run starts fresh
		}

		if rel < 0 {
			return nil, 0, false
		}

		end, kind, local, ok := svgScanMarkup(data, textEnd)
		if !ok {
			return nil, 0, false
		}

		pos = end + 1

		switch {
		case kind == svgTagClose && local == "text":
			return runs, pos, true
		case kind == svgTagClose && local == "tspan":
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		case local == "tspan" && (kind == svgTagOpen || kind == svgTagSelfClose):
			attrs := emulateAttrMap(emulateParseAttrs(data, textEnd+len("<tspan"), end))
			top := stack[len(stack)-1]

			next := frame{props: textPropsFrom(top.props, attrs), shift: top.shift, fill: top.fill}
			if shift, hasShift := baselineShift(attrs, top.props); hasShift {
				next.shift += shift
			}

			if fill := paintFill(attrs); fill != "" {
				next.fill = fill
			}

			pending = tspanPosition(pending, attrs)

			if kind == svgTagOpen {
				stack = append(stack, next)
			}
		default:
			return nil, 0, false
		}
	}

	return nil, 0, false
}

// tspanPosition folds the x, y, dx and dy of a <tspan> into the next run.
func tspanPosition(pending textRun, attrs map[string]string) textRun {
	if value, ok := attrs["x"]; ok {
		pending.x, pending.hasX = firstNumber(value)
		pending.chunkHead = pending.hasX
	}

	if value, ok := attrs["y"]; ok {
		pending.y, pending.hasY = firstNumber(value)
	}

	if value, ok := attrs["dx"]; ok {
		number, _ := firstNumber(value)
		pending.dx += number
	}

	if value, ok := attrs["dy"]; ok {
		number, _ := firstNumber(value)
		pending.dy += number
	}

	return pending
}

func paintFill(attrs map[string]string) string {
	if value := styleDeclarations(attrs["style"])["fill"]; value != "" {
		return value
	}

	return attrs["fill"]
}

// firstNumber reads the first value of a coordinate list.
func firstNumber(value string) (float64, bool) {
	fields := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' })
	if len(fields) == 0 {
		return 0, false
	}

	number, _ := splitNumberUnit(strings.ToLower(fields[0]))

	return number, !math.IsNaN(number)
}

// baselineShift reads a tspan's baseline-shift and returns how far it raises
// the run, in user units, positive up. sub and super move by half of the
// parent font's ascent plus descent. A length is used as is and a percentage is
// taken of the parent font size.
//
//nolint:cyclop // keywords, then one branch per unit
func baselineShift(attrs map[string]string, parent textProps) (float64, bool) {
	value := strings.ToLower(strings.TrimSpace(styleDeclarations(attrs["style"])["baseline-shift"]))
	if value == "" {
		value = strings.ToLower(strings.TrimSpace(attrs["baseline-shift"]))
	}

	switch value {
	case "", "baseline":
		return 0, false
	case "sub", "super":
		face := faceFor(parent)
		if face == nil || face.UnitsPerEm() <= 0 {
			return 0, false
		}

		height := (float64(face.Ascent()) + math.Abs(float64(face.Descent()))) /
			float64(face.UnitsPerEm()) * parent.size / 2 //nolint:mnd // half

		if value == "sub" {
			return -height, true
		}

		return height, true
	}

	number, unit := splitNumberUnit(value)
	if math.IsNaN(number) {
		return 0, false
	}

	switch unit {
	case "", "px":
		return number, true
	case "pt":
		return number * 4 / 3, true //nolint:mnd // 96/72
	case "em":
		return number * parent.size, true
	case "%":
		return number / 100 * parent.size, true
	}

	return 0, false
}

// faceFor resolves the bundled face a set of font properties selects.
func faceFor(props textProps) *fonts.Font {
	faces, err := fonts.LoadDefaultFaces()
	if err != nil || faces == nil {
		return nil
	}

	if face := faces.ResolveFamily(props.families, props.weight, props.italic); face != nil {
		return face
	}

	return faces.Resolve(props.weight, props.italic)
}

// placedGlyph is one glyph with its pen position, before text-anchor.
type placedGlyph struct {
	run  int
	x, y float64
}

// layoutTextRuns lays the runs out left to right, applies text-anchor to each
// chunk, and writes one <path> per run. It returns false when the budget or a
// font is missing, so the caller leaves the element for the strip pass.
//
//nolint:cyclop,funlen // one pass over runs, chunks and glyphs
func layoutTextRuns(out *strings.Builder, runs []textRun, budget *int) bool {
	if len(runs) == 0 {
		return true
	}

	paths := make([]strings.Builder, len(runs))
	glyphs := make([][]placedGlyph, len(runs))
	chunkStart := make([]float64, 0, len(runs))
	chunkWidth := make([]float64, 0, len(runs))
	chunkOf := make([]int, len(runs))

	var penX, penY float64

	for index, run := range runs {
		if run.hasX {
			penX = run.x
		}

		if run.hasY {
			penY = run.y
		}

		penX += run.dx
		penY += run.dy

		if index == 0 || run.chunkHead {
			chunkStart = append(chunkStart, penX)
			chunkWidth = append(chunkWidth, 0)
		}

		chunkOf[index] = len(chunkStart) - 1

		face := faceFor(run.props)
		if face == nil || run.props.size <= 0 || run.props.size > maxOutlineFontSize {
			return false
		}

		scale := run.props.size / float64(face.UnitsPerEm())

		for _, char := range run.text {
			if *budget--; *budget < 0 {
				return false
			}

			glyphs[index] = append(glyphs[index], placedGlyph{run: index, x: penX, y: penY - run.shift})
			penX += face.Advance(char) * scale
		}

		chunkWidth[len(chunkWidth)-1] = penX - chunkStart[len(chunkStart)-1]
	}

	for index, run := range runs {
		face := faceFor(run.props)
		scale := run.props.size / float64(face.UnitsPerEm())
		shift := anchorShift(run.props.anchor, chunkWidth[chunkOf[index]])

		for glyphIndex, char := range []rune(run.text) {
			placed := glyphs[index][glyphIndex]
			writeGlyphPath(&paths[index], face.GlyphContours(char), placed.x+shift, placed.y, scale)
		}
	}

	for index := range runs {
		if paths[index].Len() == 0 {
			continue
		}

		out.WriteString(`<path d="` + strings.TrimSpace(paths[index].String()) + `"`)

		if runs[index].fill != "" {
			out.WriteString(` fill="` + html.EscapeString(runs[index].fill) + `"`)
		}

		out.WriteString(`/>`)
	}

	return true
}

// anchorShift moves a chunk for text-anchor: middle centers it on its start
// and end ends it there.
func anchorShift(anchor string, width float64) float64 {
	switch anchor {
	case "middle":
		return -width / 2 //nolint:mnd // half
	case "end":
		return -width
	}

	return 0
}

// writeGlyphPath appends the outline of one glyph as path data. Font units run
// y-up from the baseline; the SVG y axis runs down.
func writeGlyphPath(out *strings.Builder, contours [][]fonts.GlyphPoint, originX, baseline, scale float64) {
	point := func(pt fonts.GlyphPoint) (float64, float64) {
		return originX + pt.X*scale, baseline - pt.Y*scale
	}

	for _, contour := range contours {
		writeContour(out, contour, point)
	}
}

// writeContour writes one TrueType contour: on-curve points are corners, and
// two off-curve points in a row imply an on-curve point at their midpoint.
func writeContour(out *strings.Builder, contour []fonts.GlyphPoint, point func(fonts.GlyphPoint) (float64, float64)) {
	if len(contour) < 2 { //nolint:mnd // a contour needs two points
		return
	}

	pts := rotateToOnCurve(contour)
	startX, startY := point(pts[0])

	out.WriteString("M" + fmtNum(startX) + " " + fmtNum(startY))

	for index := 1; index <= len(pts); index++ {
		cur := pts[index%len(pts)]

		if cur.On {
			x, y := point(cur)
			out.WriteString("L" + fmtNum(x) + " " + fmtNum(y))

			continue
		}

		next := pts[(index+1)%len(pts)]
		ctrlX, ctrlY := point(cur)

		if next.On {
			endX, endY := point(next)
			out.WriteString("Q" + fmtNum(ctrlX) + " " + fmtNum(ctrlY) + " " + fmtNum(endX) + " " + fmtNum(endY))

			index++

			continue
		}

		nextX, nextY := point(next)
		midX, midY := (ctrlX+nextX)/2, (ctrlY+nextY)/2 //nolint:mnd // midpoint
		out.WriteString("Q" + fmtNum(ctrlX) + " " + fmtNum(ctrlY) + " " + fmtNum(midX) + " " + fmtNum(midY))
	}

	out.WriteString("Z")
}

// rotateToOnCurve returns the contour as a closed loop that starts on an
// on-curve point, inserting the implied midpoint when every point is off-curve.
func rotateToOnCurve(contour []fonts.GlyphPoint) []fonts.GlyphPoint {
	for index, pt := range contour {
		if pt.On {
			rotated := make([]fonts.GlyphPoint, 0, len(contour))
			rotated = append(rotated, contour[index:]...)

			return append(rotated, contour[:index]...)
		}
	}

	first, last := contour[0], contour[len(contour)-1]
	mid := fonts.GlyphPoint{X: (first.X + last.X) / 2, Y: (first.Y + last.Y) / 2, On: true} //nolint:mnd // midpoint

	return append([]fonts.GlyphPoint{mid}, contour...)
}

func fmtNum(value float64) string {
	return strconv.FormatFloat(value, 'f', 3, 64)
}
