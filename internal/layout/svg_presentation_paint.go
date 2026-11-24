package layout

import (
	"bytes"
	"fmt"
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

	if po := style.svg().SVGPaintOrder; po != "" {
		inherited := decls.prop(svgPropPaintOrder) == "" &&
			node.Attribute(svgPropPaintOrder) != "" &&
			node.Parent != nil && e.hasStyle(node.Parent) &&
			e.stylePtr(node.Parent).svg().SVGPaintOrder == po
		if !inherited {
			skip.note(decls, svgPropPaintOrder, po)
		}
	}
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

// prepareSVGInheritedPresentation records presentation attributes that must be
// suppressed during sorted XML attribute emission because CSS overrides them
// on this element, or because stroke-linejoin must be emitted after
// stroke-miterlimit for the pinned canvas parser.
func (e *engine) prepareSVGInheritedPresentation(node *html.Node, decls *svgStyleDecls, skip *svgGeomSkip) {
	if node == nil || decls == nil || !e.hasStyle(node) {
		return
	}

	st := e.stylePtr(node)

	var parent *ResolvedStyle
	if node.Parent != nil && e.hasStyle(node.Parent) {
		parent = e.stylePtr(node.Parent)
	}

	cssMiter := decls.prop("stroke-miterlimit") != "" ||
		(st.svg().StrokeMiterLimit >= 1 && (parent == nil || parent.svg().StrokeMiterLimit != st.svg().StrokeMiterLimit))
	if cssMiter {
		skip.add("stroke-miterlimit")
	}

	cssJoin := decls.prop("stroke-linejoin") != "" ||
		(st.svg().StrokeLineJoin != "" && (parent == nil || parent.svg().StrokeLineJoin != st.svg().StrokeLineJoin))
	if cssJoin {
		skip.add("stroke-linejoin")
		skip.add("stroke-linejoin-css")
	} else if st.svg().StrokeMiterLimit >= 1 || node.Attribute("stroke-miterlimit") != "" {
		// Defer authored stroke-linejoin so stroke-miterlimit is emitted first.
		skip.add("stroke-linejoin")
	}

	cssFillRule := decls.prop("fill-rule") != "" ||
		(st.svg().FillRule != "" && parent != nil && parent.svg().FillRule != st.svg().FillRule)
	if cssFillRule {
		skip.add("fill-rule")
	}

	cssInterp := decls.prop("color-interpolation") != "" ||
		(st.svg().ColorInterpolation != "" && parent != nil && parent.svg().ColorInterpolation != st.svg().ColorInterpolation)
	if cssInterp {
		skip.add("color-interpolation")
	}

	cssInterpFilters := decls.prop("color-interpolation-filters") != "" ||
		(st.svg().ColorInterpolationFilters != "" && parent != nil && parent.svg().ColorInterpolationFilters != st.svg().ColorInterpolationFilters)
	if cssInterpFilters {
		skip.add("color-interpolation-filters")
	}
}

// effectiveSVGStrokeLineJoin returns the stroke-linejoin attribute value to
// emit after stroke-miterlimit.
func effectiveSVGStrokeLineJoin(
	node *html.Node, st *ResolvedStyle, decls *svgStyleDecls, geomSkip *svgGeomSkip, miterWritten bool,
) string {
	if v := strings.ToLower(strings.TrimSpace(decls.prop("stroke-linejoin"))); v != "" {
		return v
	}

	attr := strings.ToLower(strings.TrimSpace(node.Attribute("stroke-linejoin")))
	if attr != "" && !geomSkip.skips("stroke-linejoin-css") {
		return attr
	}

	if st.svg().StrokeLineJoin != "" {
		return st.svg().StrokeLineJoin
	}

	if attr != "" {
		return attr
	}

	if miterWritten {
		return "miter"
	}

	return ""
}

// svgGradientUsesObjectBoundingBox reports whether node is an SVG gradient
// whose coordinates are in objectBoundingBox units (the SVG default).
func svgGradientUsesObjectBoundingBox(node *html.Node) bool {
	if node == nil || (node.Name != "linearGradient" && node.Name != "radialGradient") {
		return false
	}

	return !strings.EqualFold(strings.TrimSpace(node.Attribute("gradientUnits")), "userspaceonuse")
}

// normalizeSVGGradientCoord converts unitless objectBoundingBox gradient
// coordinates (e.g. x1="0" x2="1") into percentages ("0%", "100%") because the
// pinned canvas parser only scales coordinates by the target bounding box when
// they carry a "%" suffix.
func normalizeSVGGradientCoord(tag, attr, value string) string {
	switch tag {
	case "linearGradient":
		if attr != "x1" && attr != "y1" && attr != "x2" && attr != "y2" {
			return value
		}
	case "radialGradient":
		if attr != "cx" && attr != "cy" && attr != "r" && attr != "fx" && attr != "fy" && attr != "fr" {
			return value
		}
	default:
		return value
	}

	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.HasSuffix(trimmed, "%") {
		return value
	}

	f, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return value
	}

	return strconv.FormatFloat(f*100, 'g', -1, 64) + "%"
}

// svgUsesLinearInterpolation reports whether node resolves to
// color-interpolation: linearRGB via CSS or its presentation attribute.
func (e *engine) svgUsesLinearInterpolation(node *html.Node, decls *svgStyleDecls) bool {
	if v := decls.prop("color-interpolation"); v != "" {
		return gradientUsesLinearLight(v)
	}

	if e.hasStyle(node) && e.stylePtr(node).svg().ColorInterpolation == "linearrgb" {
		return true
	}

	if v := node.Attribute("color-interpolation"); v != "" {
		return gradientUsesLinearLight(v)
	}

	return false
}

type svgResolvedStop struct {
	offset  float64
	color   [3]float64
	opacity float64
}

// svgLinearRGBSubdivisions is the number of sub-intervals emitted per stop pair
// when approximating linearRGB gradient interpolation in sRGB for the canvas
// rasterizer.
const svgLinearRGBSubdivisions = 8

// writeSVGLinearRGBStops expands a gradient's <stop> children by interpolating
// intermediate stops in linear light and encoding them back to sRGB. It returns
// false when the gradient has fewer than two parsable stops or non-stop element
// children so the caller falls back to normal child serialization.
func (e *engine) writeSVGLinearRGBStops(b *bytes.Buffer, grad *html.Node) bool {
	var stops []svgResolvedStop

	for _, child := range grad.Children {
		if child.Type != html.ElementNode {
			continue
		}

		if child.Name != "stop" {
			return false
		}

		stop, ok := e.resolveSVGGradientStop(child)
		if !ok {
			return false
		}

		stops = append(stops, stop)
	}

	if len(stops) < 2 {
		return false
	}

	writeStop := func(off float64, c [3]float64, op float64) {
		fmt.Fprintf(b, `<stop offset="%g" stop-color="%s"`, off, cssColorHex(c))
		if op < 1 {
			fmt.Fprintf(b, ` stop-opacity="%g"`, op)
		}
		b.WriteString(`/>`)
	}

	for i := 0; i < len(stops)-1; i++ {
		s0, s1 := stops[i], stops[i+1]
		writeStop(s0.offset, s0.color, s0.opacity)

		if s1.offset <= s0.offset || (s0.color == s1.color && s0.opacity == s1.opacity) {
			continue
		}

		r0, g0, b0 := srgbChannelToLinear(s0.color[0]), srgbChannelToLinear(s0.color[1]), srgbChannelToLinear(s0.color[2])
		r1, g1, b1 := srgbChannelToLinear(s1.color[0]), srgbChannelToLinear(s1.color[1]), srgbChannelToLinear(s1.color[2])

		for step := 1; step < svgLinearRGBSubdivisions; step++ {
			t := float64(step) / svgLinearRGBSubdivisions
			off := s0.offset + t*(s1.offset-s0.offset)
			c := [3]float64{
				linearChannelToSRGB(r0 + t*(r1-r0)),
				linearChannelToSRGB(g0 + t*(g1-g0)),
				linearChannelToSRGB(b0 + t*(b1-b0)),
			}
			op := s0.opacity + t*(s1.opacity-s0.opacity)
			writeStop(off, c, op)
		}
	}

	last := stops[len(stops)-1]
	writeStop(last.offset, last.color, last.opacity)

	return true
}

func (e *engine) resolveSVGGradientStop(node *html.Node) (svgResolvedStop, bool) {
	offset, ok := parseSVGStopFraction(node.Attribute("offset"), 0)
	if !ok {
		return svgResolvedStop{}, false
	}

	decls := parseSVGStyleDecls(node.Attribute("style"))

	var current [3]float64
	var st *ResolvedStyle
	if e.hasStyle(node) {
		st = e.stylePtr(node)
		current = st.Color
	}

	rawColor := ""
	if st != nil && st.svg().SVGStopColor != "" {
		rawColor = st.svg().SVGStopColor
	} else if v := strings.TrimSpace(decls.prop("stop-color")); v != "" {
		rawColor = v
	} else {
		rawColor = strings.TrimSpace(node.Attribute("stop-color"))
	}

	color := [3]float64{0, 0, 0}
	if rawColor != "" {
		parsed, parsedOK := parseUsedColor(rawColor, current)
		if !parsedOK {
			return svgResolvedStop{}, false
		}

		color = parsed
	}

	rawOpacity := ""
	if st != nil && st.svg().SVGStopOpacity != "" {
		rawOpacity = st.svg().SVGStopOpacity
	} else if v := strings.TrimSpace(decls.prop("stop-opacity")); v != "" {
		rawOpacity = v
	} else {
		rawOpacity = strings.TrimSpace(node.Attribute("stop-opacity"))
	}

	opacity, ok := parseSVGStopFraction(rawOpacity, 1)
	if !ok {
		return svgResolvedStop{}, false
	}

	return svgResolvedStop{offset: offset, color: color, opacity: opacity}, true
}

func parseSVGStopFraction(raw string, fallback float64) (float64, bool) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return fallback, true
	}

	if strings.HasSuffix(v, "%") {
		f, err := strconv.ParseFloat(strings.TrimSpace(v[:len(v)-1]), 64)
		if err != nil {
			return 0, false
		}

		return clamp01(f / 100), true
	}

	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, false
	}

	return clamp01(f), true
}
