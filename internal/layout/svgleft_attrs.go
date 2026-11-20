package layout

import (
	"bytes"
	"fmt"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// svgTextContentElement reports whether name takes the text leftovers.
func svgTextContentElement(name string) bool {
	switch name {
	case "text", "tspan", "textPath", "tref":
		return true
	}

	return false
}

// svgMarkedShape reports whether name takes marker-side.
func svgMarkedShape(name string) bool {
	switch name {
	case shapeKindPath, "line", "polyline", "polygon", objectViewBoxRectShape, "circle", "ellipse":
		return true
	}

	return false
}

// writeSVGLeftoverAttrs writes the cascaded value of the five SVG leftovers
// (text-rendering, lighting-color, glyph-orientation-vertical, baseline-shift,
// marker-side) as presentation attributes. It reads the resolved Declared
// store, so a stylesheet rule reaches the XML the same way an inline style
// does. The pinned canvas rasterizer reads none of the five, so the effect is
// in the serialized SVG only. An attribute already written, or a
// text-rendering value the inline-style arm in layout_svg.go bakes, wins.
func (e *engine) writeSVGLeftoverAttrs(
	buf *bytes.Buffer, node *html.Node, written *svgWrittenSet, decls *svgStyleDecls,
) {
	if node == nil || !e.hasStyle(node) {
		return
	}

	declared := e.stylePtr(node).declared()
	name := node.Name

	write := func(attr, value string) {
		if value != "" && !written.has(attr) {
			fmt.Fprintf(buf, ` %s="%s"`, attr, escapeXML(value))
		}
	}

	if svgTextContentElement(name) {
		if _, inline := svgBakeTextRendering(decls.prop("text-rendering")); !inline {
			write("text-rendering", declared.TextRendering)
		}

		write("glyph-orientation-vertical", declared.GlyphOrientationVertical)
		write("baseline-shift", declared.BaselineShift)
	}

	if name == "feDiffuseLighting" || name == "feSpecularLighting" {
		write("lighting-color", declared.LightingColor)
	}

	if svgMarkedShape(name) {
		write("marker-side", declared.MarkerSide)
	}
}
