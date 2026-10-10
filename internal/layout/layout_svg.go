//nolint:all // inline SVG serialize+raster path
package layout

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/html"
	"github.com/chinmay-sawant/blinkless/internal/svg"
)

const cssTagSVG = "svg"

// buildInlineSVG rasterizes an inline <svg> subtree (with cascade presentation
// props baked into attributes). When paint is false, only size/imgRef are
// filled so the inline line placer can emit at the final position (same
// contract as buildImage); painting at (0,0) here left stray SVGs on the
// masthead and hid logo.png.
func (e *engine) buildInlineSVG(node *html.Node, sty ResolvedStyle, posX, posY float64, paint bool) *box {
	boxNode := &box{ //nolint:exhaustruct // intentional zero fields
		node: node, style: e.stylePtr(node), kind: boxKindReplaced, x: posX, y: posY,
	}

	data := e.serializeInlineSVG(node)
	ref := &imageRef{src: "#inline-svg"} //nolint:exhaustruct // synthetic raster
	if png, pw, ph, err := svg.Rasterize(data, 1024); err == nil && len(png) > 0 {
		ref.data, ref.w, ref.h = png, pw, ph
	}
	boxNode.img = ref

	size := e.usedInlineSVGSize(node, sty, ref)
	padL := e.scalePt(sty.PaddingLeft)
	padR := e.scalePt(sty.PaddingRight)
	padT := e.scalePt(sty.PaddingTop)
	padB := e.scalePt(sty.PaddingBottom)
	borderL := e.scalePt(borderPaint(sty.BorderLeft))
	borderR := e.scalePt(borderPaint(sty.BorderRight))
	borderT := e.scalePt(borderPaint(sty.BorderTop))
	borderB := e.scalePt(borderPaint(sty.BorderBottom))
	boxNode.w = size.w + padL + padR + borderL + borderR
	boxNode.height = size.h + padT + padB + borderT + borderB

	if paint && ref.data != nil && !e.noEmit {
		imgX := posX + borderL + padL
		imgY := posY + borderT + padT
		opStart := len(e.ops)
		e.add((Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpImage, X: imgX, Y: imgY, W: size.w, H: size.h,
		}).withImage(ref.data, ref.w, ref.h, ""))
		e.prependChrome(opStart, boxNode, sty, posX, posY, boxNode.w, boxNode.height)
	}

	return boxNode
}

// usedInlineSVGSize prefers width/height attributes (CSS px), then the
// raster intrinsic size, then a small fallback.
func (e *engine) usedInlineSVGSize(node *html.Node, sty ResolvedStyle, ref *imageRef) imageUsedSize {
	// Attribute lengths are CSS px → pt, then zoomed like other style lengths.
	wAttr := e.scalePt(parseSVGLengthPx(node.Attribute("width")))
	hAttr := e.scalePt(parseSVGLengthPx(node.Attribute("height")))
	if sty.Width >= 0 {
		wAttr = e.scalePt(sty.Width)
	} else if sty.WidthPercent >= 0 && e.opts.Width > 0 {
		wAttr = e.opts.Width * sty.WidthPercent / 100
	}
	if sty.Height >= 0 {
		hAttr = e.scalePt(sty.Height)
	}

	intrW, intrH := 0.0, 0.0
	if ref != nil && ref.w > 0 && ref.h > 0 {
		intrW = e.scalePt(pxToPt(float64(ref.w)))
		intrH = e.scalePt(pxToPt(float64(ref.h)))
	}
	if wAttr <= 0 {
		wAttr = intrW
	}
	if hAttr <= 0 {
		hAttr = intrH
	}
	if wAttr <= 0 {
		wAttr = e.scalePt(pxToPt(64))
	}
	if hAttr <= 0 {
		hAttr = e.scalePt(pxToPt(28))
	}

	// Max constraints clamp like usedImageSize does for <img>: a definite
	// width or height keeps its axis, and only an auto axis follows the used
	// ratio (Chrome 143: a 200x200 svg with max-height 100pt stays 200 wide).
	size := imageUsedSize{w: wAttr, h: hAttr}
	cssW := sty.Width >= 0
	if sty.WidthPercent >= 0 && e.imageContainingWidth() > 0 {
		cssW = true
	}
	widthDefinite := cssW || parseSVGLengthPx(node.Attribute("width")) > 0
	heightDefinite := sty.Height >= 0 || parseSVGLengthPx(node.Attribute("height")) > 0
	size = clampImageWidth(size, e.imageMaxWidth(sty, cssW), heightDefinite)
	size = clampImageHeight(e, size, sty, widthDefinite)

	return size
}

func parseSVGLengthPx(raw string) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	raw = strings.TrimSuffix(raw, "px")
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		return 0
	}

	return pxToPt(v)
}

// serializeInlineSVG emits SVG XML for canvas rasterization, baking resolved
// fill/stroke presentation props onto each element as attributes.
func (e *engine) serializeInlineSVG(node *html.Node) []byte {
	var b strings.Builder
	e.writeSVGNode(&b, node)

	return []byte(b.String())
}

func (e *engine) writeSVGNode(b *strings.Builder, node *html.Node) {
	if node == nil {
		return
	}
	switch node.Type {
	case html.TextNode:
		b.WriteString(escapeXML(node.Text))
	case html.ElementNode:
		b.WriteByte('<')
		b.WriteString(node.Name)
		written := map[string]bool{}
		// CSS d overrides the d presentation attribute (CSS wins over
		// presentation attributes in Chrome 143.0.7499.40), so the attribute
		// is suppressed here and the baked CSS value is emitted by
		// writeSVGExtraPresentationAttrs below. Applies to path only: d is
		// not a valid property on other elements.
		suppressDAttr := false
		if node.Name == "path" {
			_, suppressDAttr = svgCSSPathData(node.Attribute("style"))
		}
		for k, v := range node.Attrs {
			if k == "style" {
				continue
			}
			if k == "d" && suppressDAttr {
				continue
			}
			b.WriteByte(' ')
			b.WriteString(k)
			b.WriteString(`="`)
			b.WriteString(escapeXML(v))
			b.WriteByte('"')
			written[k] = true
		}
		if node.Name == cssTagSVG && !written["xmlns"] {
			b.WriteString(` xmlns="http://www.w3.org/2000/svg"`)
		}
		e.writeSVGPresentationAttrs(b, node, written)
		writeSVGExtraPresentationAttrs(b, node, written)
		if len(node.Children) == 0 {
			b.WriteString("/>")
			return
		}
		b.WriteByte('>')
		for _, c := range node.Children {
			e.writeSVGNode(b, c)
		}
		b.WriteString("</")
		b.WriteString(node.Name)
		b.WriteByte('>')
	}
}

func (e *engine) writeSVGPresentationAttrs(b *strings.Builder, node *html.Node, written map[string]bool) {
	if !e.hasStyle(node) {
		return
	}

	st := e.stylePtr(node)
	if st.FillSet && !written["fill"] {
		if st.FillOpacity == 0 && st.Fill == [3]float64{} {
			b.WriteString(` fill="none"`)
		} else {
			fmt.Fprintf(b, ` fill="%s"`, cssColorHex(st.Fill))
		}
	}
	if st.FillOpacity >= 0 && st.FillOpacity < 1 && !written["fill-opacity"] {
		fmt.Fprintf(b, ` fill-opacity="%g"`, st.FillOpacity)
	}
	if st.StrokeSet && !written["stroke"] {
		fmt.Fprintf(b, ` stroke="%s"`, cssColorHex(st.Stroke))
	}
	if st.StrokeWidthSet && st.StrokeWidth > 0 && !written["stroke-width"] {
		fmt.Fprintf(b, ` stroke-width="%g"`, st.StrokeWidth)
	}
	if st.StrokeOpacity >= 0 && st.StrokeOpacity < 1 && !written["stroke-opacity"] {
		fmt.Fprintf(b, ` stroke-opacity="%g"`, st.StrokeOpacity)
	}
	if len(st.StrokeDashArray) > 0 && !written["stroke-dasharray"] {
		parts := make([]string, 0, len(st.StrokeDashArray))
		for _, d := range st.StrokeDashArray {
			parts = append(parts, strconv.FormatFloat(d, 'g', -1, 64))
		}
		b.WriteString(` stroke-dasharray="`)
		b.WriteString(escapeXML(strings.Join(parts, " ")))
		b.WriteByte('"')
	}
	if st.StrokeDashOffset != 0 && !written["stroke-dashoffset"] {
		fmt.Fprintf(b, ` stroke-dashoffset="%g"`, st.StrokeDashOffset)
	}
	if st.StrokeLineCap != "" && !written["stroke-linecap"] {
		switch st.StrokeLineCap {
		case "butt", "round", "square":
			fmt.Fprintf(b, ` stroke-linecap="%s"`, st.StrokeLineCap)
		}
	}
	// stroke-miterlimit is baked before stroke-linejoin on purpose: the
	// pinned canvas parser keeps the limit in its state and only reads it
	// when a later linejoin value builds the miter joiner, so the reverse
	// order would leave a bare miterlimit with no observable effect.
	if st.StrokeMiterLimit >= 1 && !written["stroke-miterlimit"] {
		fmt.Fprintf(b, ` stroke-miterlimit="%g"`, st.StrokeMiterLimit)
	}
	if st.StrokeLineJoin != "" && !written["stroke-linejoin"] {
		switch st.StrokeLineJoin {
		case "miter", "miter-clip", "round", "bevel", "arcs":
			fmt.Fprintf(b, ` stroke-linejoin="%s"`, st.StrokeLineJoin)
		}
	}
	// fill-rule and shape-rendering have no ResolvedStyle field, so they are
	// recovered from the element's inline style attribute here. A matching
	// presentation attribute already passed through above (written map) and
	// wins by the no-duplicate guard. Stylesheet rules for these two need a
	// stored field owned elsewhere and stay unbaked.
	if !written["fill-rule"] {
		if rule := svgInlineStyleProp(node.Attribute("style"), "fill-rule"); rule != "" {
			switch strings.ToLower(rule) {
			case "nonzero", "evenodd":
				fmt.Fprintf(b, ` fill-rule="%s"`, strings.ToLower(rule))
			}
		}
	}
	if !written["shape-rendering"] {
		if hint := svgInlineStyleProp(node.Attribute("style"), "shape-rendering"); hint != "" {
			switch strings.ToLower(hint) {
			case "auto":
				b.WriteString(` shape-rendering="auto"`)
			case "optimizespeed":
				b.WriteString(` shape-rendering="optimizeSpeed"`)
			case "crispedges":
				b.WriteString(` shape-rendering="crispEdges"`)
			case "geometricprecision":
				b.WriteString(` shape-rendering="geometricPrecision"`)
			}
		}
	}
	// SVG geometry and stop presentation properties below are recovered from
	// the element's inline style attribute, like fill-rule above: they have
	// no ResolvedStyle field, and the canvas rasterizer only reads them as
	// XML attributes (drawShape and parseDefs in the pinned canvas svg.go),
	// never from CSS style text (its setAttribute switch has no case for
	// them, and the style attribute itself is skipped above). A matching
	// presentation attribute already passed through above (written map) and
	// wins by the no-duplicate guard. Stylesheet rules for these need a
	// stored field owned elsewhere and stay unbaked.
	styleAttr := node.Attribute("style")
	if !written["cx"] {
		if v, ok := svgBakeLengthValue(svgInlineStyleProp(styleAttr, "cx")); ok {
			fmt.Fprintf(b, ` cx="%s"`, escapeXML(v))
		}
	}
	if !written["cy"] {
		if v, ok := svgBakeLengthValue(svgInlineStyleProp(styleAttr, "cy")); ok {
			fmt.Fprintf(b, ` cy="%s"`, escapeXML(v))
		}
	}
	if !written["r"] {
		if v, ok := svgBakeNonNegativeLength(svgInlineStyleProp(styleAttr, "r")); ok {
			fmt.Fprintf(b, ` r="%s"`, escapeXML(v))
		}
	}
	if !written["rx"] {
		if v, ok := svgBakeNonNegativeLength(svgInlineStyleProp(styleAttr, "rx")); ok {
			fmt.Fprintf(b, ` rx="%s"`, escapeXML(v))
		}
	}
	if !written["ry"] {
		if v, ok := svgBakeNonNegativeLength(svgInlineStyleProp(styleAttr, "ry")); ok {
			fmt.Fprintf(b, ` ry="%s"`, escapeXML(v))
		}
	}
	if !written["x"] {
		if v, ok := svgBakeLengthValue(svgInlineStyleProp(styleAttr, "x")); ok {
			fmt.Fprintf(b, ` x="%s"`, escapeXML(v))
		}
	}
	if !written["y"] {
		if v, ok := svgBakeLengthValue(svgInlineStyleProp(styleAttr, "y")); ok {
			fmt.Fprintf(b, ` y="%s"`, escapeXML(v))
		}
	}
	// stop-color and stop-opacity only affect gradient stops: canvas
	// parseDefs reads them from the stop tag attributes, so baking them on
	// any other element would be noise that the rasterizer ignores.
	if node.Name == "stop" {
		if !written["stop-color"] {
			if v := strings.TrimSpace(svgInlineStyleProp(styleAttr, "stop-color")); svgBakeStopColorValid(v) {
				fmt.Fprintf(b, ` stop-color="%s"`, escapeXML(v))
			}
		}
		if !written["stop-opacity"] {
			if v, ok := svgBakeOpacityValue(svgInlineStyleProp(styleAttr, "stop-opacity")); ok {
				fmt.Fprintf(b, ` stop-opacity="%s"`, escapeXML(v))
			}
		}
	}
	// paint-order, vector-effect, and text-rendering are baked for
	// serialization parity with Chrome 143, but the pinned canvas
	// setAttribute has no case for any of them, so the raster cannot
	// observe them. Each gap is pinned in css_behavior_svg_geo_test.go with
	// a serialize-only assertion, never a pixel assertion.
	if !written["paint-order"] {
		if v, ok := svgBakePaintOrder(svgInlineStyleProp(styleAttr, "paint-order")); ok {
			fmt.Fprintf(b, ` paint-order="%s"`, escapeXML(v))
		}
	}
	if !written["vector-effect"] {
		if v, ok := svgBakeVectorEffect(svgInlineStyleProp(styleAttr, "vector-effect")); ok {
			fmt.Fprintf(b, ` vector-effect="%s"`, escapeXML(v))
		}
	}
	if !written["text-rendering"] {
		if v, ok := svgBakeTextRendering(svgInlineStyleProp(styleAttr, "text-rendering")); ok {
			fmt.Fprintf(b, ` text-rendering="%s"`, escapeXML(v))
		}
	}
}

// svgInlineStyleProp returns the trimmed value of one CSS property from a raw
// inline style attribute, or "" when absent. It splits on the first colon so
// values holding URLs keep working.
func svgInlineStyleProp(styleAttr, prop string) string {
	for _, decl := range strings.Split(styleAttr, ";") {
		name, value, ok := strings.Cut(decl, ":")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(name), prop) {
			return strings.TrimSpace(value)
		}
	}

	return ""
}

// splitBakeUnit splits a validated-looking CSS length into its numeric text
// and lowercase unit ("" for plain numbers, "%" for percentages).
func splitBakeUnit(v string) (string, string) {
	if strings.HasSuffix(v, "%") {
		return v[:len(v)-1], "%"
	}

	i := len(v)
	for i > 0 {
		c := v[i-1]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			i--
		} else {
			break
		}
	}

	return v[:i], strings.ToLower(v[i:])
}

// svgBakeLengthValue validates a CSS geometry value for the cx/cy/x/y bake
// arms. Only units the canvas rasterizer resolves (plain numbers, px, pt,
// pc, cm, mm, q, in, %) pass: font and viewport units stay unbaked so a
// value the rasterizer would misread never changes pixels.
func svgBakeLengthValue(raw string) (string, bool) {
	v := strings.TrimSpace(raw)
	if v == "" || strings.ContainsAny(v, " \t\n") {
		return "", false
	}

	num, unit := splitBakeUnit(v)

	if _, err := strconv.ParseFloat(num, 64); err != nil {
		return "", false
	}

	switch unit {
	case "", "px", "pt", "pc", "cm", "mm", "q", "in", "%":
		return v, true
	}

	return "", false
}

// svgBakeNonNegativeLength validates a CSS geometry value for the r/rx/ry
// bake arms. A negative radius is invalid per SVG2 and is dropped, matching
// Chrome 143 which ignores the declaration.
func svgBakeNonNegativeLength(raw string) (string, bool) {
	v, ok := svgBakeLengthValue(raw)
	if !ok {
		return "", false
	}

	num, _ := splitBakeUnit(v)

	f, err := strconv.ParseFloat(num, 64)
	if err != nil || f < 0 {
		return "", false
	}

	return v, true
}

// svgBakeStopColorValid reports whether a stop-color value survives the
// canvas rasterizer: it parses hex, rgb(), and rgba() but degrades anything
// else to black, so named colors and currentcolor stay unbaked.
func svgBakeStopColorValid(raw string) bool {
	v := strings.TrimSpace(raw)
	if v == "" {
		return false
	}

	if strings.HasPrefix(v, "#") {
		hex := v[1:]
		if len(hex) != 3 && len(hex) != 6 {
			return false
		}

		for i := 0; i < len(hex); i++ {
			c := hex[i]
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
				return false
			}
		}

		return true
	}

	lower := strings.ToLower(v)

	return strings.HasPrefix(lower, "rgb(") && strings.HasSuffix(lower, ")") ||
		strings.HasPrefix(lower, "rgba(") && strings.HasSuffix(lower, ")")
}

// svgBakeOpacityValue validates a stop-opacity value: a plain number in
// 0..1 or a percentage in 0%..100%, matching what the canvas parseNumber
// resolves for the stop tag.
func svgBakeOpacityValue(raw string) (string, bool) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "", false
	}

	if strings.HasSuffix(v, "%") {
		f, err := strconv.ParseFloat(strings.TrimSpace(v[:len(v)-1]), 64)
		if err != nil || f < 0 || f > 100 {
			return "", false
		}

		return v, true
	}

	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < 0 || f > 1 {
		return "", false
	}

	return v, true
}

// svgBakePaintOrder validates a paint-order value: normal alone, or one or
// more of fill/stroke/markers in any order without repeats.
func svgBakePaintOrder(raw string) (string, bool) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" {
		return "", false
	}

	if v == "normal" {
		return "normal", true
	}

	seen := map[string]bool{}

	var out []string

	for _, tok := range strings.Fields(v) {
		if tok != "fill" && tok != "stroke" && tok != "markers" {
			return "", false
		}

		if seen[tok] {
			return "", false
		}

		seen[tok] = true
		out = append(out, tok)
	}

	if len(out) == 0 {
		return "", false
	}

	return strings.Join(out, " "), true
}

// svgBakeVectorEffect validates a vector-effect value. Only the values with
// an observable SVG meaning are baked; the rasterizer ignores them (pinned
// gap), but an invalid token must not reach the serialized output.
func svgBakeVectorEffect(raw string) (string, bool) {
	switch v := strings.ToLower(strings.TrimSpace(raw)); v {
	case "none", "non-scaling-stroke":
		return v, true
	}

	return "", false
}

// svgBakeTextRendering validates a text-rendering value and returns its
// canonical camelCase attribute form.
func svgBakeTextRendering(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "auto":
		return "auto", true
	case "optimizespeed":
		return "optimizeSpeed", true
	case "optimizelegibility":
		return "optimizeLegibility", true
	case "geometricprecision":
		return "geometricPrecision", true
	}

	return "", false
}

func cssColorHex(c [3]float64) string {
	r := int(c[0]*255 + 0.5)
	g := int(c[1]*255 + 0.5)
	bl := int(c[2]*255 + 0.5)
	if r < 0 {
		r = 0
	}
	if g < 0 {
		g = 0
	}
	if bl < 0 {
		bl = 0
	}
	if r > 255 {
		r = 255
	}
	if g > 255 {
		g = 255
	}
	if bl > 255 {
		bl = 255
	}

	var buf [7]byte

	out := append(buf[:0], '#')
	out = appendHexByte(out, byte(r))
	out = appendHexByte(out, byte(g))
	out = appendHexByte(out, byte(bl))

	return string(out)
}

// appendHexByte appends b as a zero-padded 2-digit lowercase hex number.
func appendHexByte(dst []byte, b byte) []byte {
	const hexDigits = "0123456789abcdef"

	return append(dst, hexDigits[b>>4], hexDigits[b&0x0F])
}

// xmlEscaper escapes the five predefined XML entities. It is package level
// because escapeXML runs once per serialized SVG node.
//
//nolint:gochecknoglobals // immutable escaper, safe to share
var xmlEscaper = strings.NewReplacer(
	`&`, "&amp;",
	`<`, "&lt;",
	`>`, "&gt;",
	`"`, "&quot;",
	`'`, "&apos;",
)

func escapeXML(s string) string {
	return xmlEscaper.Replace(s)
}

// collectInlineSVGItem flattens an inline <svg> into one replaced inline item.
// paint=false so emitInlineImage places the bitmap at the line position.
func (e *engine) collectInlineSVGItem(node *html.Node, sty ResolvedStyle, out *[]inlineItem) {
	svgBox := e.buildInlineSVG(node, sty, 0, 0, false)
	*out = append(*out, inlineItem{ //nolint:exhaustruct // intentional zero fields
		img: true, w: svgBox.w, h: svgBox.height, style: e.stylePtr(node),
		imgRef:  svgBox.img,
		marginL: e.scalePt(sty.MarginLeft), marginR: e.scalePt(sty.MarginRight),
		marginT: e.scalePt(sty.MarginTop), marginB: e.scalePt(sty.MarginBottom),
		// Replaced elements rest their bottom margin edge on the baseline.
		marginBaseline: true,
	})
}

// writeSVGExtraPresentationAttrs bakes CSS-only SVG properties that have no
// ResolvedStyle field into presentation attributes. It runs after
// writeSVGPresentationAttrs so the written map keeps authored attributes
// winning over same-name CSS (matching the fill-rule precedent above),
// except CSS d which replaces the d attribute suppressed in writeSVGNode.
// Kept as free functions in this section so the geometry+paint bake arms
// owned elsewhere stay untouched. Reference browser for every branch:
// Chrome 143.0.7499.40.
func writeSVGExtraPresentationAttrs(b *strings.Builder, node *html.Node, written map[string]bool) {
	if node == nil || node.Type != html.ElementNode {
		return
	}

	styleAttr := node.Attribute("style")
	if strings.TrimSpace(styleAttr) == "" {
		return
	}

	writeSVGMarkerAttrs(b, styleAttr, written)
	writeSVGTextAnchorAttr(b, node, styleAttr, written)

	// CSS d (SVG2 geometry property, Chrome 121+) sets path data. The pinned
	// canvas rasterizer reads path data only from the d attribute, so the
	// property value is baked as one. Restricted to path: d is invalid
	// elsewhere and canvas ignores it there too.
	if node.Name == "path" {
		if d, ok := svgCSSPathData(styleAttr); ok && !written["d"] {
			b.WriteString(` d="`)
			b.WriteString(escapeXML(d))
			b.WriteByte('"')
		}
	}
}

// writeSVGTextAnchorAttr bakes text-anchor from inline style into a
// presentation attribute on text containers. Chrome 143.0.7499.40 anchors
// SVG text only (start, middle, end); HTML text is unaffected. The pinned
// canvas rasterizer reads the anchor only as an XML attribute (its
// setAttribute switch has the case, but the style attribute itself is
// skipped above), so without this bake the declaration never moves pixels.
// An authored text-anchor attribute wins by the no-duplicate guard, and
// stylesheet rules stay unbaked like the fill-rule precedent above.
func writeSVGTextAnchorAttr(b *strings.Builder, node *html.Node, styleAttr string, written map[string]bool) {
	if written["text-anchor"] {
		return
	}
	switch node.Name {
	case "text", "tspan", "textPath":
		// Baked below.
	default:
		return
	}
	if anchor, ok := svgBakeTextAnchor(svgInlineStyleProp(styleAttr, "text-anchor")); ok {
		b.WriteString(` text-anchor="`)
		b.WriteString(anchor)
		b.WriteByte('"')
	}
}

// svgBakeTextAnchor validates a text-anchor value. Only the three SVG
// anchors reach the canvas text renderer; anything else stays unbaked.
func svgBakeTextAnchor(raw string) (string, bool) {
	switch v := strings.ToLower(strings.TrimSpace(raw)); v {
	case "start", "middle", "end":
		return v, true
	}
	return "", false
}

// writeSVGMarkerAttrs bakes the marker shorthand and the marker-start,
// marker-mid, and marker-end longhands from inline style into presentation
// attributes. Chrome maps these SVG presentation attributes to CSS, so
// style="marker-end:url(#arrow)" renders markers; the pinned canvas renderer
// draws markers at path vertices from url(#id) refs. The marker shorthand
// expands onto every unset longhand because canvas has no shorthand case.
// A lone "none" stays unbaked: it matches the no-marker initial and there is
// nothing to emit.
func writeSVGMarkerAttrs(b *strings.Builder, styleAttr string, written map[string]bool) {
	shorthand := svgInlineStyleProp(styleAttr, "marker")

	for _, prop := range []string{"marker-start", "marker-mid", "marker-end"} {
		if written[prop] {
			continue
		}

		val := svgInlineStyleProp(styleAttr, prop)
		if val == "" {
			val = shorthand
		}

		if !isSVGMarkerRef(val) {
			continue
		}

		b.WriteByte(' ')
		b.WriteString(prop)
		b.WriteString(`="`)
		b.WriteString(escapeXML(strings.TrimSpace(val)))
		b.WriteByte('"')
	}
}

// isSVGMarkerRef reports whether a marker property value points at a marker
// definition. Only url(#id) refs reach the canvas marker renderer; none and
// anything unparsable mean no marker.
func isSVGMarkerRef(val string) bool {
	v := strings.TrimSpace(val)
	if v == "" || strings.EqualFold(v, "none") {
		return false
	}

	return strings.HasPrefix(strings.ToLower(v), "url(") && strings.HasSuffix(v, ")")
}

// svgCSSPathData returns the path data carried by the CSS d property, which
// must have the form path("...") (double or single quotes). "none" and any
// other shape mean no override.
func svgCSSPathData(styleAttr string) (string, bool) {
	raw := strings.TrimSpace(svgInlineStyleProp(styleAttr, "d"))
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
