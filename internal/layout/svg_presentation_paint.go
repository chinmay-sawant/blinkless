package layout

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// Stylesheet values for the SVG paint presentation properties: marker and its
// three longhands, stop-color, stop-opacity, paint-order, and path-length.
// The cascade stores a valid value on ResolvedStyle. prepareSVGStylesheetPaint
// copies it into the bake decls unless the inline style already declares that
// property, and the serializer drops the matching presentation attribute so
// the stylesheet value wins. An invalid value is not stored, so an earlier
// valid value (or the attribute) stays.

const (
	svgPropMarker       = "marker"
	svgPropMarkerStart  = "marker-start"
	svgPropMarkerMid    = "marker-mid"
	svgPropMarkerEnd    = "marker-end"
	svgPropStopColor    = "stop-color"
	svgPropStopOpacity  = "stop-opacity"
	svgPropPaintOrder   = "paint-order"
	svgPropPathLength   = "path-length"
	svgPropVectorEffect = "vector-effect"
	svgPropFloodColor   = "flood-color"
	svgPropFloodOpacity = "flood-opacity"
)

const svgValueNone = "none"

// applySVGPaintProps stores the ten SVG paint properties from the cascade.
func applySVGPaintProps(style *ResolvedStyle, prop, value string) bool {
	if style == nil {
		return false
	}

	trimmed := strings.TrimSpace(value)

	if handled, ok := applySVGMarkerProps(style, prop, trimmed); handled {
		return ok
	}

	return applySVGPaintSwitch(style, prop, trimmed)
}

// applySVGPaintSwitch stores one non-marker SVG paint property.
func applySVGPaintSwitch(style *ResolvedStyle, prop, trimmed string) bool {
	switch prop {
	case svgPropFloodColor:
		return storeSVGColorHex(&style.editSvg().SVGFloodColor, style.Color, trimmed)
	case svgPropFloodOpacity:
		return storeSVGString(&style.editSvg().SVGFloodOpacity, trimmed, svgBakeOpacityValue)
	case svgPropVectorEffect:
		return storeSVGString(&style.editSvg().SVGVectorEffect, trimmed, svgBakeVectorEffect)
	case svgPropStopColor:
		return storeSVGStopColor(style, trimmed)
	case svgPropStopOpacity:
		return storeSVGString(&style.editSvg().SVGStopOpacity, trimmed, svgBakeOpacityValue)
	case svgPropPaintOrder:
		return storeSVGString(&style.editSvg().SVGPaintOrder, trimmed, svgBakePaintOrder)
	case svgPropPathLength:
		return storeSVGString(&style.editSvg().SVGPathLength, trimmed, svgPathLengthValue)
	default:
		return false
	}
}

func applySVGMarkerProps(style *ResolvedStyle, prop, value string) (bool, bool) {
	switch prop {
	case svgPropMarker:
		return true, storeSVGMarker(style, value, true, true, true)
	case svgPropMarkerStart:
		return true, storeSVGMarker(style, value, true, false, false)
	case svgPropMarkerMid:
		return true, storeSVGMarker(style, value, false, true, false)
	case svgPropMarkerEnd:
		return true, storeSVGMarker(style, value, false, false, true)
	default:
		return false, false
	}
}

// storeSVGString stores check's canonical form. A CSS-wide keyword is
// accepted and leaves the field alone; an invalid value is refused.
func storeSVGString(dest *string, value string, check func(string) (string, bool)) bool {
	if cssWideKeyword(value) {
		return true
	}

	canon, ok := check(value)
	if !ok {
		return false
	}

	*dest = canon

	return true
}

// storeSVGMarker stores none or a url(...) reference onto the chosen
// longhands. The shorthand selects all three, so a later longhand or
// shorthand in the cascade replaces it in order.
func storeSVGMarker(style *ResolvedStyle, value string, start, mid, end bool) bool {
	if cssWideKeyword(value) {
		return true
	}

	if strings.EqualFold(value, svgValueNone) {
		value = svgValueNone
	} else if !isSVGMarkerRef(value) {
		return false
	}

	if start {
		style.editSvg().SVGMarkerStart = value
	}

	if mid {
		style.editSvg().SVGMarkerMid = value
	}

	if end {
		style.editSvg().SVGMarkerEnd = value
	}

	return true
}

// storeSVGStopColor stores a color the canvas parser reads (hex, rgb, rgba).
// A named color is converted to hex first, because the parser would turn it
// black.
func storeSVGStopColor(style *ResolvedStyle, value string) bool {
	if cssWideKeyword(value) {
		return true
	}

	if svgBakeStopColorValid(value) {
		style.editSvg().SVGStopColor = value

		return true
	}

	if strings.EqualFold(value, "currentcolor") {
		return false
	}

	color, ok := parseUsedColor(value, style.Color)
	if !ok {
		return false
	}

	style.editSvg().SVGStopColor = cssColorHex(color)

	return true
}

// svgPathLengthValue accepts none, or a number that is at least 0 (a bare
// number or px length), and returns the pathLength attribute text.
func svgPathLengthValue(raw string) (string, bool) {
	text := strings.ToLower(strings.TrimSpace(raw))
	if text == svgValueNone {
		return svgValueNone, true
	}

	f, err := strconv.ParseFloat(strings.TrimSuffix(text, "px"), 64)
	if err != nil || f < 0 {
		return "", false
	}

	return strconv.FormatFloat(f, 'g', -1, 64), true
}

// prepareSVGStylesheetPaint copies stored stylesheet paint values into decls
// when the inline style does not declare them, and returns the presentation
// attributes the serializer must omit. none on a marker or path-length
// clears the attribute without writing a replacement.
func (e *engine) prepareSVGStylesheetPaint(node *html.Node, decls *svgStyleDecls, skip *svgGeomSkip) {
	if node == nil || decls == nil || !e.hasStyle(node) {
		return
	}

	style := e.stylePtr(node)

	inlineMarker := decls.prop(svgPropMarker) != ""

	for _, marker := range [3]struct{ prop, value string }{
		{svgPropMarkerStart, style.svg().SVGMarkerStart},
		{svgPropMarkerMid, style.svg().SVGMarkerMid},
		{svgPropMarkerEnd, style.svg().SVGMarkerEnd},
	} {
		if inlineMarker {
			continue
		}

		skip.note(decls, marker.prop, marker.value)
	}

	skip.note(decls, svgPropPaintOrder, style.svg().SVGPaintOrder)
	skip.note(decls, svgPropVectorEffect, style.svg().SVGVectorEffect)

	if node.Name == "feFlood" {
		skip.note(decls, svgPropFloodColor, style.svg().SVGFloodColor)
		skip.note(decls, svgPropFloodOpacity, style.svg().SVGFloodOpacity)
	}

	if node.Name == "stop" {
		skip.note(decls, svgPropStopColor, style.svg().SVGStopColor)
		skip.note(decls, svgPropStopOpacity, style.svg().SVGStopOpacity)
	}

	if style.svg().SVGPathLength != "" && decls.prop(svgPropPathLength) == "" {
		decls.add(svgPropPathLength, style.svg().SVGPathLength)
		skip.add("pathLength")
		skip.add("pathlength")
	}
}

// writeSVGPathLengthAttr bakes path-length as the pathLength attribute on
// shapes. The pinned canvas rasterizer has no pathLength support, so this is
// a serialization-only bake. none writes nothing.
func writeSVGPathLengthAttr(out *bytes.Buffer, decls *svgStyleDecls, written *svgWrittenSet) {
	if written.has("pathLength") || written.has("pathlength") {
		return
	}

	v, ok := svgPathLengthValue(decls.prop(svgPropPathLength))
	if !ok || v == svgValueNone {
		return
	}

	out.WriteString(` pathLength="` + v + `"`)
}

// The pinned canvas applies marker-start, marker-mid, and marker-end for one
// shape in map order, and each application draws a layer. The second marker
// then takes that new layer as "the path", so it lands at the wrong place or
// not at all. A shape with two or more markers is therefore written once per
// marker: the first copy keeps the paint and one marker, the others are
// copies of the same geometry that paint nothing that carry one marker each.

// svgMarkersForSplit is the marker count from which a shape is split.
const svgMarkersForSplit = 2

var (
	svgMarkerAttrRE = regexp.MustCompile(` marker-(?:start|mid|end)="[^"]*"`)
	svgCloneStripRE = regexp.MustCompile(` (?:id|fill|stroke|stroke-dasharray)="[^"]*"`)
	svgStrokeAttrRE = regexp.MustCompile(` stroke="([^"]*)"`)
)

// splitSVGMarkerShape rewrites the opening tag that starts at tagStart. It
// returns false and leaves the buffer alone when fewer than two markers were
// written. On true the buffer holds the first copy plus the clones, each
// self-closed, and the caller must not write a closing tag.
func splitSVGMarkerShape(buf *bytes.Buffer, tagStart int) bool {
	open := string(buf.Bytes()[tagStart:])

	markers := svgMarkerAttrRE.FindAllString(open, -1)
	if len(markers) < svgMarkersForSplit {
		return false
	}

	base := svgMarkerAttrRE.ReplaceAllString(open, "")

	buf.Truncate(tagStart)
	buf.WriteString(base)
	buf.WriteString(markers[0])
	buf.WriteString("/>")

	// The copy keeps the stroke paint and width, because the canvas skips a
	// shape with an invisible stroke and then misplaces the marker. A
	// zero-length dash pattern leaves nothing painted.
	stroke := "#000000"
	if m := svgStrokeAttrRE.FindStringSubmatch(base); m != nil && !strings.EqualFold(m[1], "none") {
		stroke = m[1]
	}

	ghost := svgCloneStripRE.ReplaceAllString(base, "")

	for _, marker := range markers[1:] {
		buf.WriteString(ghost)
		buf.WriteString(` fill="none" stroke="` + stroke + `" stroke-dasharray="0 100000"`)
		buf.WriteString(marker)
		buf.WriteString("/>")
	}

	return true
}

// storeSVGColorHex stores a color as hex text. Hex, rgb(), and rgba() pass
// through; a named color is converted so the canvas parser never sees a name.
func storeSVGColorHex(dest *string, current [3]float64, value string) bool {
	if cssWideKeyword(value) {
		return true
	}

	if svgBakeStopColorValid(value) {
		*dest = value

		return true
	}

	color, ok := parseUsedColor(value, current)
	if !ok {
		return false
	}

	*dest = cssColorHex(color)

	return true
}
