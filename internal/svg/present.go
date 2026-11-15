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
	if !bytesContainsFold(data, []byte("paint-order")) && !bytesContainsFold(data, []byte("pathlength")) &&
		!bytesContainsFold(data, []byte("vector-effect")) && !bytesContainsFold(data, []byte("feflood")) {
		return data
	}

	var out bytes.Buffer

	flush, pos, tags := 0, 0, 0
	changed := false
	rootScale := 0.0
	floods := emulateCollectFloods(data)

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

		if name == "svg" && rootScale == 0 {
			rootScale = emulateRootScale(emulateAttrMap(attrs))
		}

		replaced, modified := emulateShapeTag(name, attrs, selfClose, rootScale, floods)
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
func emulateShapeTag(
	name string, attrs []emulateAttr, selfClose bool, rootScale float64, floods map[string]emulateFlood,
) (string, bool) {
	if !selfClose || !emulateIsShape(name) {
		return "", false
	}

	index := emulateAttrMap(attrs)

	if flat, ok := emulateFloodShape(name, index, floods); ok {
		return flat, true
	}

	scaled, scaledOK := emulateScaleDashes(name, attrs, index)
	if scaledOK {
		attrs = scaled
		index = emulateAttrMap(attrs)
	}

	if fixed, ok := emulateNonScalingStroke(attrs, index, rootScale); ok {
		attrs = fixed
		index = emulateAttrMap(attrs)
		scaledOK = true
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

// emulateRootScale is how many device units one user unit covers at the root
// svg: the width over the viewBox width, averaged with the height ratio. It
// is 1 when either is missing or not a plain number, and for a missing size.
func emulateRootScale(index map[string]string) float64 {
	box := strings.FieldsFunc(index["viewbox"], func(r rune) bool { return r == ' ' || r == ',' })
	if len(box) != 4 {
		return 1
	}

	boxW, errW := strconv.ParseFloat(box[2], 64)
	boxH, errH := strconv.ParseFloat(box[3], 64)

	if errW != nil || errH != nil || boxW <= 0 || boxH <= 0 {
		return 1
	}

	scaleX, scaleY := 1.0, 1.0

	if width := emulateNum(index, "width"); width > 0 {
		scaleX = width / boxW
	}

	if height := emulateNum(index, "height"); height > 0 {
		scaleY = height / boxH
	}

	return math.Sqrt(scaleX * scaleY)
}

// emulateNonScalingStroke divides the stroke width and dashes by the root
// scale for vector-effect: non-scaling-stroke, so the stroke keeps its size
// in device units instead of growing with the viewBox. Chrome 143 ships only
// none and non-scaling-stroke, so only that token acts.
func emulateNonScalingStroke(attrs []emulateAttr, index map[string]string, scale float64) ([]emulateAttr, bool) {
	if !strings.EqualFold(strings.TrimSpace(index["vector-effect"]), "non-scaling-stroke") ||
		scale <= 0 || math.Abs(scale-1) < 1e-9 {
		return nil, false
	}

	out := make([]emulateAttr, 0, len(attrs)+1)
	hadWidth := false

	for _, attr := range attrs {
		switch strings.ToLower(attr.name) {
		case "stroke-width":
			hadWidth = true

			width, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(attr.value), "px"), 64)
			if err != nil {
				return nil, false
			}

			attr.value = strconv.FormatFloat(width/scale, 'g', -1, 64)
		case "stroke-dasharray", "stroke-dashoffset":
			scaled, ok := emulateScaleList(attr.value, 1/scale)
			if !ok {
				return nil, false
			}

			attr.value = scaled
		}

		out = append(out, attr)
	}

	if !hadWidth {
		out = append(out, emulateAttr{name: "stroke-width", value: strconv.FormatFloat(1/scale, 'g', -1, 64)})
	}

	return out, true
}

// emulateFlood is a filter that holds exactly one feFlood.
type emulateFlood struct {
	color, opacity string
	x, y, w, h     float64 // region, as fractions of the bounding box
}

// emulateCollectFloods finds each filter whose only primitive is one feFlood.
// A filter with any other primitive is left to the canvas, which ignores it.
func emulateCollectFloods(data []byte) map[string]emulateFlood {
	if !bytesContainsFold(data, []byte("feflood")) {
		return nil
	}

	floods := map[string]emulateFlood{}

	var (
		inFilter bool
		id       string
		region   emulateFlood
		prims    []string
		flood    map[string]string
	)

	pos, tags := 0, 0

	for pos < len(data) && tags < maxEmulateTags {
		tags++

		rel := bytes.IndexByte(data[pos:], '<')
		if rel < 0 {
			break
		}

		start := pos + rel

		end, ok := emulateTagEnd(data, start)
		if !ok {
			break
		}

		pos = start + 1

		if bytes.HasPrefix(data[start:], []byte("</")) {
			if inFilter && strings.EqualFold(strings.TrimSpace(string(data[start+2:end])), "filter") {
				if len(prims) == 1 && prims[0] == "feflood" && id != "" {
					region.color, region.opacity = flood["flood-color"], flood["flood-opacity"]
					floods[id] = region
				}

				inFilter = false
			}

			continue
		}

		name, attrs, selfClose := emulateParseTag(data, start, end)

		switch {
		case name == "filter" && !selfClose:
			index := emulateAttrMap(attrs)
			inFilter, id, prims, flood = true, index["id"], nil, nil
			region = emulateFlood{
				x: emulateFraction(index["x"], -0.1), y: emulateFraction(index["y"], -0.1),
				w: emulateFraction(index["width"], 1.2), h: emulateFraction(index["height"], 1.2),
			}
		case inFilter && strings.HasPrefix(name, "fe"):
			prims = append(prims, name)
			if name == "feflood" {
				flood = emulateAttrMap(attrs)
			}
		}
	}

	return floods
}

// emulateFraction reads a number or a percentage as a fraction of the box.
func emulateFraction(raw string, fallback float64) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}

	if strings.HasSuffix(raw, "%") {
		num, err := strconv.ParseFloat(strings.TrimSuffix(raw, "%"), 64)
		if err != nil {
			return fallback
		}

		return num / 100
	}

	num, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}

	return num
}

// emulateFloodShape replaces a rect, circle, or ellipse that uses a
// single-feFlood filter with one flat rect covering the filter region.
func emulateFloodShape(name string, index map[string]string, floods map[string]emulateFlood) (string, bool) {
	ref := strings.TrimSpace(index["filter"])
	if len(floods) == 0 || !strings.HasPrefix(ref, "url(#") || !strings.HasSuffix(ref, ")") {
		return "", false
	}

	flood, ok := floods[strings.TrimSuffix(strings.TrimPrefix(ref, "url(#"), ")")]
	if !ok {
		return "", false
	}

	var boxX, boxY, boxW, boxH float64

	switch name {
	case "rect":
		boxX, boxY, boxW, boxH = emulateNum(index, "x"), emulateNum(index, "y"),
			emulateNum(index, "width"), emulateNum(index, "height")
	case "circle":
		radius := emulateNum(index, "r")
		boxX, boxY, boxW, boxH = emulateNum(index, "cx")-radius, emulateNum(index, "cy")-radius, 2*radius, 2*radius
	case "ellipse":
		radX, radY := emulateNum(index, "rx"), emulateNum(index, "ry")
		boxX, boxY, boxW, boxH = emulateNum(index, "cx")-radX, emulateNum(index, "cy")-radY, 2*radX, 2*radY
	default:
		return "", false
	}

	if boxW <= 0 || boxH <= 0 {
		return "", false
	}

	color := flood.color
	if color == "" {
		color = "#000000"
	}

	attrs := []emulateAttr{
		{"x", num(boxX + flood.x*boxW)}, {"y", num(boxY + flood.y*boxH)},
		{"width", num(flood.w * boxW)}, {"height", num(flood.h * boxH)},
		{"fill", color},
	}

	if flood.opacity != "" {
		attrs = append(attrs, emulateAttr{"fill-opacity", flood.opacity})
	}

	if transform, has := index["transform"]; has {
		attrs = append(attrs, emulateAttr{"transform", transform})
	}

	return emulateTagText("rect", attrs, nil), true
}

func num(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}
