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

// Named values used by the geometry and filter-region math below.
const (
	emulateTwo           = 2   // a square's exponent, a half size, or a diameter factor
	emulateRamanujan     = 4   // constant term in Ramanujan's ellipse perimeter formula
	emulateViewBoxFields = 4   // min-x, min-y, width, height in a viewBox
	emulatePercent       = 100 // divides a percentage into a fraction
	emulateDefaultSpan   = 1.2 // default filter region width and height, as a fraction of the box
	emulateLineTag       = "line"
	emulateEllipseTag    = "ellipse"
	emulateCircleTag     = "circle"
	emulatePathTag       = "path"
	// emulateRamanujanOffset is added to the root term in Ramanujan's perimeter formula.
	emulateRamanujanOffset = 10
	emulateFillPart        = "fill"
	emulateStrokePart      = "stroke"
)

// emulateMayRewrite reports whether data mentions a feature the emulations read.
func emulateMayRewrite(data []byte) bool {
	return bytesContainsFold(data, []byte("paint-order")) || bytesContainsFold(data, []byte("pathlength")) ||
		bytesContainsFold(data, []byte("vector-effect")) || bytesContainsFold(data, []byte("feflood")) ||
		bytesContainsFold(data, []byte("fediffuselighting"))
}

// emulatePresentation applies both emulations. It returns the input slice
// when no tag changes.
func emulatePresentation(data []byte) []byte {
	if !emulateMayRewrite(data) {
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
	case emulatePathTag, svgRectTag, emulateCircleTag, emulateEllipseTag, emulateLineTag, "polyline", "polygon":
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

	if flat, ok := emulateFloodShape(name, index, floods, rootScale); ok {
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
		case emulateFillPart:
			fillAt = idx
		case emulateStrokePart:
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
		if (tok == emulateFillPart || tok == emulateStrokePart || tok == "markers") && !emulateContains(order[:count], tok) {
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

// emulateMarkerAttrNames lists the three marker presentation attributes.
func emulateMarkerAttrNames() []string {
	return []string{"marker-start", "marker-mid", "marker-end"}
}

// emulateSplitPaintOrder writes one copy per painted part in order.
func emulateSplitPaintOrder(name string, attrs []emulateAttr, index map[string]string, order [3]string) string {
	var out strings.Builder

	hasMarkers := false

	for _, key := range emulateMarkerAttrNames() {
		if _, ok := index[key]; ok {
			hasMarkers = true
		}
	}

	base := append([]string{"paint-order", "id"}, emulateMarkerAttrNames()...)

	for _, part := range order {
		switch part {
		case emulateFillPart:
			out.WriteString(emulateTagText(name, attrs, map[string]string{"stroke": "none"}, base...))
		case emulateStrokePart:
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

	if !emulateHasDashArray(index) {
		return nil, false
	}

	actual, measured := emulateShapeLength(name, index)
	if !measured || actual <= 0 {
		return nil, false
	}

	factor := actual / declared

	return emulateScaleAttrs(attrs, factor)
}

// emulateScaleAttrs multiplies the dash attributes by factor and drops
// pathLength. ok is false when a dash value does not parse.
func emulateScaleAttrs(attrs []emulateAttr, factor float64) ([]emulateAttr, bool) {
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

// emulateHasDashArray reports whether the shape sets a stroke-dasharray other than none.
func emulateHasDashArray(index map[string]string) bool {
	dashes, ok := index["stroke-dasharray"]

	return ok && !strings.EqualFold(strings.TrimSpace(dashes), "none")
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
	case emulatePathTag:
		path, err := canvas.ParseSVGPath(index["d"])
		if err != nil {
			return 0, false
		}

		return path.Length(), true
	case emulateLineTag:
		return math.Hypot(emulateNum(index, "x2")-emulateNum(index, "x1"),
			emulateNum(index, "y2")-emulateNum(index, "y1")), true
	case emulateCircleTag:
		return 2 * math.Pi * emulateNum(index, "r"), true
	case emulateEllipseTag:
		return emulateEllipse(emulateNum(index, "rx"), emulateNum(index, "ry")), true
	case svgRectTag:
		return emulateRectLength(index)
	default:
		return emulatePolyLength(index["points"], name == "polygon")
	}
}

// emulateEllipse is Ramanujan's perimeter approximation.
func emulateEllipse(rx, ry float64) float64 {
	h := math.Pow(rx-ry, emulateTwo) / math.Pow(rx+ry, emulateTwo)

	return math.Pi * (rx + ry) * (1 + 3*h/(emulateRamanujanOffset+math.Sqrt(emulateRamanujan-3*h)))
}

func emulateRectLength(index map[string]string) (float64, bool) {
	width, height := emulateNum(index, "width"), emulateNum(index, "height")
	corner, cornerY := emulateNum(index, "rx"), emulateNum(index, "ry")

	if corner == 0 {
		corner = cornerY
	}

	if cornerY == 0 {
		cornerY = corner
	}

	if corner != cornerY {
		return 0, false
	}

	corner = math.Min(corner, math.Min(width, height)/emulateTwo)

	return 2*(width-2*corner) + 2*(height-2*corner) + 2*math.Pi*corner, true
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
	if len(box) != emulateViewBoxFields {
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
	light          *emulateLighting
}

// emulateMayCollectFloods reports whether data holds a filter the collector reads.
func emulateMayCollectFloods(data []byte) bool {
	return bytesContainsFold(data, []byte("feflood")) || bytesContainsFold(data, []byte("fediffuselighting"))
}

// emulateFilterScan is the filter emulateCollectFloods is reading. Its tags
// fill it in order, and its closing tag stores the result.
type emulateFilterScan struct {
	open     bool
	id       string
	region   emulateFlood
	prims    []string
	flood    map[string]string
	filterAt map[string]string
	diffuse  map[string]string
	distant  map[string]string
}

// begin starts a filter from its opening tag attributes.
func (s *emulateFilterScan) begin(attrs []emulateAttr) {
	index := emulateAttrMap(attrs)
	s.open, s.id, s.prims, s.flood = true, index["id"], nil, nil
	s.filterAt, s.diffuse, s.distant = index, nil, nil
	s.region = emulateFlood{
		color: "", opacity: "", light: nil,
		x: emulateFraction(index["x"], -0.1), y: emulateFraction(index["y"], -0.1),
		w: emulateFraction(index["width"], emulateDefaultSpan), h: emulateFraction(index["height"], emulateDefaultSpan),
	}
}

// primitive records one fe* primitive inside the open filter.
func (s *emulateFilterScan) primitive(name string, attrs []emulateAttr) {
	s.prims = append(s.prims, name)

	switch name {
	case "feflood":
		s.flood = emulateAttrMap(attrs)
	case "fediffuselighting":
		s.diffuse = emulateAttrMap(attrs)
	case "fedistantlight":
		s.distant = emulateAttrMap(attrs)
	}
}

// element handles one opening tag while a filter may be open.
func (s *emulateFilterScan) element(name string, attrs []emulateAttr, selfClose bool) {
	switch {
	case name == "filter" && !selfClose:
		s.begin(attrs)
	case s.open && strings.HasPrefix(name, "fe"):
		s.primitive(name, attrs)
	}
}

// close stores the filter's flood when it reduces to one, then ends the filter.
func (s *emulateFilterScan) close(floods map[string]emulateFlood) {
	if s.id != "" {
		if flood, ok := s.reduced(); ok {
			floods[s.id] = flood
		}
	}

	s.open = false
}

// reduced returns the flood a filter reduces to: one feFlood, or a lighting chain.
func (s *emulateFilterScan) reduced() (emulateFlood, bool) {
	region := s.region

	if len(s.prims) == 1 && s.prims[0] == "feflood" {
		region.color, region.opacity = s.flood["flood-color"], s.flood["flood-opacity"]

		return region, true
	}

	light, ok := emulateLightingFilter(s.prims, s.filterAt, s.diffuse, s.distant)
	if !ok {
		return emulateFlood{color: "", opacity: "", x: 0, y: 0, w: 0, h: 0, light: nil}, false
	}

	region.light = &light

	return region, true
}

// emulateCollectFloods finds each filter whose only primitive is one feFlood.
// A filter with any other primitive is left to the canvas, which ignores it.
func emulateCollectFloods(data []byte) map[string]emulateFlood {
	if !emulateMayCollectFloods(data) {
		return nil
	}

	floods := map[string]emulateFlood{}

	var scan emulateFilterScan

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
			if scan.open && strings.EqualFold(strings.TrimSpace(string(data[start+2:end])), "filter") {
				scan.close(floods)
			}

			continue
		}

		name, attrs, selfClose := emulateParseTag(data, start, end)
		scan.element(name, attrs, selfClose)
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

		return num / emulatePercent
	}

	num, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}

	return num
}

// emulateFloodShape replaces a rect, circle, or ellipse that uses a
// single-feFlood filter with one flat rect covering the filter region.
func emulateFloodShape(
	name string, index map[string]string, floods map[string]emulateFlood, rootScale float64,
) (string, bool) {
	ref := strings.TrimSpace(index["filter"])
	if len(floods) == 0 || !strings.HasPrefix(ref, "url(#") || !strings.HasSuffix(ref, ")") {
		return "", false
	}

	flood, found := floods[strings.TrimSuffix(strings.TrimPrefix(ref, "url(#"), ")")]
	if !found {
		return "", false
	}

	boxX, boxY, boxW, boxH, boxOK := emulateFloodBox(name, index)
	if !boxOK {
		return "", false
	}

	if flood.light != nil {
		unit := 1.0
		if rootScale > 0 {
			unit = 1 / rootScale
		}

		return emulateLitShape(name, index, *flood.light,
			boxX+flood.x*boxW, boxY+flood.y*boxH, flood.w*boxW, flood.h*boxH, boxX, boxY, boxW, boxH, unit), true
	}

	return emulateFloodRect(flood, index, boxX, boxY, boxW, boxH), true
}

// emulateFloodBox returns the user-space box a rect, circle, or ellipse covers.
// ok is false for any other element and for an empty box.
func emulateFloodBox(name string, index map[string]string) (float64, float64, float64, float64, bool) {
	var boxX, boxY, boxW, boxH float64

	switch name {
	case svgRectTag:
		boxX, boxY, boxW, boxH = emulateNum(index, "x"), emulateNum(index, "y"),
			emulateNum(index, "width"), emulateNum(index, "height")
	case emulateCircleTag:
		radius := emulateNum(index, "r")
		boxX, boxY, boxW, boxH = emulateNum(index, "cx")-radius, emulateNum(index, "cy")-radius,
			emulateTwo*radius, emulateTwo*radius
	case emulateEllipseTag:
		radX, radY := emulateNum(index, "rx"), emulateNum(index, "ry")
		boxX, boxY, boxW, boxH = emulateNum(index, "cx")-radX, emulateNum(index, "cy")-radY,
			emulateTwo*radX, emulateTwo*radY
	default:
		return 0, 0, 0, 0, false
	}

	return boxX, boxY, boxW, boxH, boxW > 0 && boxH > 0
}

// emulateFloodRect writes the flat rect that replaces a single-feFlood shape.
func emulateFloodRect(flood emulateFlood, index map[string]string, boxX, boxY, boxW, boxH float64) string {
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

	return emulateTagText(svgRectTag, attrs, nil)
}

func num(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}
