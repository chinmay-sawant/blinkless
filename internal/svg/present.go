package svg

import (
	"bytes"
	"math"
	"strconv"
	"strings"

	"github.com/tdewolff/canvas"
)

// Two SVG presentation attributes the pinned canvas parser reads nothing
// for: paint-order and pathLength. Both are emulated by rewriting the shape
// tag before the parse, so they change raster pixels.
//
// paint-order: a shape with both a fill and a stroke is written as one copy
// per painted part, in the requested order. A fill copy has stroke="none", a
// stroke copy has fill="none", and a markers copy repeats the stroke with a
// zero-length dash pattern so only its markers show. The canvas draws copies
// in document order.
//
// pathLength: stroke-dasharray and stroke-dashoffset are scaled by
// actualLength / pathLength, which is what the attribute means for dashes.
//
// Limits: only the element's own attributes count (paint-order set on a
// group is not read), and only leaf shapes are rewritten. A rect with
// unequal rx and ry, units other than px, and unparsable geometry keep their
// dashes unscaled.

const emulateZeroDash = "0 100000"

// emulatePresentation applies both emulations. It returns the input slice
// when no tag changes.
func emulatePresentation(data []byte) []byte {
	if !bytesContainsFold(data, []byte("paint-order")) && !bytesContainsFold(data, []byte("pathlength")) {
		return data
	}

	var out bytes.Buffer

	flush, pos, tags := 0, 0, 0
	changed := false

	for pos < len(data) {
		tags++
		if tags > maxEmulateTags {
			return data
		}

		rel := bytes.IndexByte(data[pos:], '<')
		if rel < 0 {
			break
		}

		start := pos + rel

		end, ok := emulateTagEnd(data, start)
		if !ok {
			break
		}

		name, attrs, selfClose := emulateParseTag(data, start, end)

		replaced, modified := emulateShapeTag(name, attrs, selfClose)
		if !modified {
			pos = start + 1

			continue
		}

		changed = true

		out.Write(data[flush:start])
		out.WriteString(replaced)

		flush = end + 1
		pos = flush
	}

	if !changed {
		return data
	}

	out.Write(data[flush:])

	return out.Bytes()
}

func emulateIsShape(name string) bool {
	switch name {
	case "path", "rect", "circle", "ellipse", "line", "polyline", "polygon":
		return true
	default:
		return false
	}
}

// emulateAttrMap indexes attrs by lowercase name.
func emulateAttrMap(attrs []emulateAttr) map[string]string {
	attrMap := make(map[string]string, len(attrs))

	for _, attr := range attrs {
		attrMap[strings.ToLower(attr.name)] = attr.value
	}

	return attrMap
}

// emulateShapeTag rewrites one leaf shape tag, or reports no change.
func emulateShapeTag(name string, attrs []emulateAttr, selfClose bool) (string, bool) {
	if !selfClose || !emulateIsShape(name) {
		return "", false
	}

	index := emulateAttrMap(attrs)

	scaled, scaledOK := emulateScaleDashes(name, attrs, index)
	if scaledOK {
		attrs = scaled
		index = emulateAttrMap(attrs)
	}

	order, orderOK := emulateStrokeFirstOrder(index)
	if !orderOK {
		if scaledOK {
			return emulateTagText(name, attrs, nil), true
		}

		return "", false
	}

	return emulateSplitPaintOrder(name, attrs, index, order), true
}

// emulateTagText renders a self-closed tag, with extra attributes appended
// and the named ones replaced or removed.
func emulateTagText(name string, attrs []emulateAttr, set map[string]string, drop ...string) string {
	var out strings.Builder

	out.WriteByte('<')
	out.WriteString(name)

	written := map[string]bool{}

	for _, attr := range attrs {
		low := strings.ToLower(attr.name)
		if emulateContains(drop, low) {
			continue
		}

		value := attr.value
		if replaced, ok := set[low]; ok {
			value = replaced
		}

		written[low] = true

		out.WriteString(` ` + attr.name + `="` + value + `"`)
	}

	for key, value := range set {
		if !written[key] {
			out.WriteString(` ` + key + `="` + value + `"`)
		}
	}

	out.WriteString("/>")

	return out.String()
}

func emulateContains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}

	return false
}

// emulateStrokeFirstOrder returns the effective paint order when the shape has
// a visible fill and stroke and the stroke comes before the fill.
func emulateStrokeFirstOrder(index map[string]string) ([3]string, bool) {
	raw, ok := index["paint-order"]
	if !ok {
		return [3]string{}, false
	}

	fill, stroke := index["fill"], index["stroke"]
	if strings.EqualFold(fill, "none") || stroke == "" || strings.EqualFold(stroke, "none") {
		return [3]string{}, false
	}

	order := effectivePaintOrder(raw)

	fillAt, strokeAt := 0, 0

	for idx, part := range order {
		switch part {
		case "fill":
			fillAt = idx
		case "stroke":
			strokeAt = idx
		}
	}

	return order, strokeAt < fillAt
}

// effectivePaintOrder lists fill, stroke, and markers in paint order: the
// named parts first, the rest in their default order.
func effectivePaintOrder(raw string) [3]string {
	var order [3]string

	count := 0

	for _, tok := range strings.Fields(strings.ToLower(raw)) {
		if (tok == "fill" || tok == "stroke" || tok == "markers") && !emulateContains(order[:count], tok) {
			order[count] = tok
			count++
		}
	}

	for _, tok := range [3]string{"fill", "stroke", "markers"} {
		if !emulateContains(order[:count], tok) {
			order[count] = tok
			count++
		}
	}

	return order
}

var emulateMarkerAttrs = []string{"marker-start", "marker-mid", "marker-end"}

// emulateSplitPaintOrder writes one copy per painted part in order.
func emulateSplitPaintOrder(name string, attrs []emulateAttr, index map[string]string, order [3]string) string {
	var out strings.Builder

	hasMarkers := false

	for _, key := range emulateMarkerAttrs {
		if _, ok := index[key]; ok {
			hasMarkers = true
		}
	}

	base := append([]string{"paint-order", "id"}, emulateMarkerAttrs...)

	for _, part := range order {
		switch part {
		case "fill":
			out.WriteString(emulateTagText(name, attrs, map[string]string{"stroke": "none"}, base...))
		case "stroke":
			out.WriteString(emulateTagText(name, attrs, map[string]string{"fill": "none"}, base...))
		case "markers":
			if hasMarkers {
				out.WriteString(emulateTagText(name, attrs, map[string]string{
					"fill": "none", "stroke-dasharray": emulateZeroDash,
				}, "paint-order", "id"))
			}
		}
	}

	return out.String()
}

// emulateScaleDashes rescales the dash attributes by actual / pathLength.
func emulateScaleDashes(name string, attrs []emulateAttr, index map[string]string) ([]emulateAttr, bool) {
	rawLen, ok := index["pathlength"]
	if !ok {
		return nil, false
	}

	declared, err := strconv.ParseFloat(strings.TrimSpace(rawLen), 64)
	if err != nil || declared <= 0 {
		return nil, false
	}

	dashes, hasDashes := index["stroke-dasharray"]
	if !hasDashes || strings.EqualFold(strings.TrimSpace(dashes), "none") {
		return nil, false
	}

	actual, measured := emulateShapeLength(name, index)
	if !measured || actual <= 0 {
		return nil, false
	}

	factor := actual / declared
	out := make([]emulateAttr, 0, len(attrs))

	for _, attr := range attrs {
		switch strings.ToLower(attr.name) {
		case "pathlength":
			continue
		case "stroke-dasharray", "stroke-dashoffset":
			scaled, ok := emulateScaleList(attr.value, factor)
			if !ok {
				return nil, false
			}

			attr.value = scaled
		}

		out = append(out, attr)
	}

	return out, true
}

func emulateScaleList(value string, factor float64) (string, bool) {
	fields := strings.FieldsFunc(value, func(r rune) bool { return r == ' ' || r == ',' || r == '\t' })
	parts := make([]string, 0, len(fields))

	for _, field := range fields {
		num, err := strconv.ParseFloat(strings.TrimSuffix(field, "px"), 64)
		if err != nil {
			return "", false
		}

		parts = append(parts, strconv.FormatFloat(num*factor, 'g', -1, 64))
	}

	return strings.Join(parts, " "), len(parts) > 0
}

func emulateNum(index map[string]string, key string) float64 {
	num, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(index[key]), "px"), 64)
	if err != nil {
		return 0
	}

	return num
}

// emulateShapeLength measures a shape in user units.
func emulateShapeLength(name string, index map[string]string) (float64, bool) {
	switch name {
	case "path":
		path, err := canvas.ParseSVGPath(index["d"])
		if err != nil {
			return 0, false
		}

		return path.Length(), true
	case "line":
		return math.Hypot(emulateNum(index, "x2")-emulateNum(index, "x1"),
			emulateNum(index, "y2")-emulateNum(index, "y1")), true
	case "circle":
		return 2 * math.Pi * emulateNum(index, "r"), true
	case "ellipse":
		return emulateEllipse(emulateNum(index, "rx"), emulateNum(index, "ry")), true
	case "rect":
		return emulateRectLength(index)
	default:
		return emulatePolyLength(index["points"], name == "polygon")
	}
}

// emulateEllipse is Ramanujan's perimeter approximation.
func emulateEllipse(rx, ry float64) float64 {
	h := math.Pow(rx-ry, 2) / math.Pow(rx+ry, 2)

	return math.Pi * (rx + ry) * (1 + 3*h/(10+math.Sqrt(4-3*h)))
}

func emulateRectLength(index map[string]string) (float64, bool) {
	width, height := emulateNum(index, "width"), emulateNum(index, "height")
	rx, ry := emulateNum(index, "rx"), emulateNum(index, "ry")

	if rx == 0 {
		rx = ry
	}

	if ry == 0 {
		ry = rx
	}

	if rx != ry {
		return 0, false
	}

	rx = math.Min(rx, math.Min(width, height)/2)

	return 2*(width-2*rx) + 2*(height-2*rx) + 2*math.Pi*rx, true
}

func emulatePolyLength(points string, closed bool) (float64, bool) {
	fields := strings.FieldsFunc(points, func(r rune) bool { return r == ' ' || r == ',' || r == '\t' || r == '\n' })
	if len(fields) < 4 || len(fields)%2 != 0 {
		return 0, false
	}

	coords := make([]float64, len(fields))

	for idx, field := range fields {
		num, err := strconv.ParseFloat(field, 64)
		if err != nil {
			return 0, false
		}

		coords[idx] = num
	}

	total := 0.0

	for idx := 2; idx < len(coords); idx += 2 {
		total += math.Hypot(coords[idx]-coords[idx-2], coords[idx+1]-coords[idx-1])
	}

	if closed {
		total += math.Hypot(coords[0]-coords[len(coords)-2], coords[1]-coords[len(coords)-1])
	}

	return total, true
}
