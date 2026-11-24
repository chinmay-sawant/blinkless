package layout

import (
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// Stylesheet geometry for the eight SVG properties whose inline style already
// bakes a presentation attribute. The cascade stores a valid value here.
// prepareSVGStylesheetGeometry copies it into the bake decls when the inline
// style does not already declare that property, and the serializer drops the
// presentation attribute so the stylesheet value wins. d then follows the
// existing path() rule: the attribute is replaced and one d is written.

const (
	svgPropCx = "cx"
	svgPropCy = "cy"
	svgPropR  = "r"
	svgPropRx = "rx"
	svgPropRy = "ry"
	svgPropX  = "x"
	svgPropY  = "y"
	svgPropD  = "d"
)

// svgGeomSkip names presentation attributes a stylesheet geometry value
// replaces. The serializer omits those attributes so the bake writes one.
type svgGeomSkip struct {
	names [20]string
	count int
}

func (skip *svgGeomSkip) add(name string) {
	if skip.count >= len(skip.names) {
		return
	}

	skip.names[skip.count] = name
	skip.count++
}

func (skip *svgGeomSkip) skips(name string) bool {
	for i := range skip.count {
		if skip.names[i] == name {
			return true
		}
	}

	return false
}

// applySVGGeometryProps stores cx, cy, r, rx, ry, x, y, and d from the cascade.
// An invalid value is not stored, so the presentation attribute stays.
func applySVGGeometryProps(style *ResolvedStyle, prop, value string) bool {
	if style == nil {
		return false
	}

	switch prop {
	case svgPropCx, svgPropCy, svgPropX, svgPropY, svgPropR, svgPropRx, svgPropRy:
		return storeSVGGeometryLength(svgGeomLengthField(style, prop), value, svgGeomRadiusProp(prop))
	case svgPropD:
		return storeSVGGeometryPath(&style.editSvg().SVGD, value)
	default:
		return false
	}
}

func svgGeomRadiusProp(prop string) bool {
	switch prop {
	case svgPropR, svgPropRx, svgPropRy:
		return true
	default:
		return false
	}
}

func svgGeomLengthField(style *ResolvedStyle, prop string) *string {
	switch prop {
	case svgPropCx:
		return &style.editSvg().SVGCx
	case svgPropCy:
		return &style.editSvg().SVGCy
	case svgPropX:
		return &style.editSvg().SVGX
	case svgPropY:
		return &style.editSvg().SVGY
	default:
		return svgGeomRadiusField(style, prop)
	}
}

func svgGeomRadiusField(style *ResolvedStyle, prop string) *string {
	switch prop {
	case svgPropR:
		return &style.editSvg().SVGR
	case svgPropRx:
		return &style.editSvg().SVGRx
	case svgPropRy:
		return &style.editSvg().SVGRy
	default:
		return nil
	}
}

func storeSVGGeometryLength(dest *string, value string, nonNegative bool) bool {
	if dest == nil {
		return false
	}

	trimmed := strings.TrimSpace(value)
	if cssWideKeyword(trimmed) {
		return true
	}

	baked, ok := svgGeometryLength(trimmed, nonNegative)
	if !ok {
		return false
	}

	*dest = baked

	return true
}

func svgGeometryLength(value string, nonNegative bool) (string, bool) {
	if nonNegative {
		return svgBakeNonNegativeLength(value)
	}

	return svgBakeLengthValue(value)
}

func storeSVGGeometryPath(dest *string, value string) bool {
	if dest == nil {
		return false
	}

	trimmed := strings.TrimSpace(value)
	if cssWideKeyword(trimmed) {
		return true
	}

	if _, ok := svgGeometryPathData(trimmed); !ok {
		return false
	}

	*dest = trimmed

	return true
}

// svgGeometryPathData returns the path data inside path("...") or path('...').
// none and any other string are not path data, matching the inline d bake.
func svgGeometryPathData(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "none") {
		return "", false
	}

	if !strings.HasPrefix(strings.ToLower(raw), "path(") || !strings.HasSuffix(raw, ")") {
		return "", false
	}

	inner := strings.TrimSpace(raw[len("path(") : len(raw)-1])
	inner = strings.Trim(inner, `"'`)

	if strings.TrimSpace(inner) == "" {
		return "", false
	}

	return inner, true
}

// prepareSVGStylesheetGeometry copies stored stylesheet geometry into decls
// when the inline style does not already declare that property. The returned
// set is the presentation attributes the serializer must omit. d on a path is
// included so the existing path() bake replaces the attribute.
func (e *engine) prepareSVGStylesheetGeometry(node *html.Node, decls *svgStyleDecls) svgGeomSkip {
	var skip svgGeomSkip

	if node == nil || decls == nil || !e.hasStyle(node) {
		return skip
	}

	style := e.stylePtr(node)
	skip.note(decls, svgPropCx, style.svg().SVGCx)
	skip.note(decls, svgPropCy, style.svg().SVGCy)
	skip.note(decls, svgPropR, style.svg().SVGR)
	skip.note(decls, svgPropRx, style.svg().SVGRx)
	skip.note(decls, svgPropRy, style.svg().SVGRy)
	skip.note(decls, svgPropX, style.svg().SVGX)
	skip.note(decls, svgPropY, style.svg().SVGY)

	if node.Name == "path" {
		skip.note(decls, svgPropD, style.svg().SVGD)
	}

	return skip
}

func (skip *svgGeomSkip) note(decls *svgStyleDecls, prop, value string) {
	if value == "" {
		return
	}

	if decls.prop(prop) != "" {
		skip.add(prop)

		return
	}

	decls.add(prop, value)
	skip.add(prop)
}

func (d *svgStyleDecls) add(name, value string) {
	if d.n < len(d.names) {
		d.names[d.n] = name
		d.values[d.n] = value
		d.n++

		return
	}

	d.spillN = append(d.spillN, name)
	d.spillV = append(d.spillV, value)
}
