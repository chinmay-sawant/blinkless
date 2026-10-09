// tree rules; same conventions as html.go
//
//nolint:all // SVG/MathML namespace, adjustment tables, and foreign-content
package html

import "strings"

// Namespace identifies the DOM namespace an element belongs to. The zero
// value is the HTML namespace, so zero-constructed nodes keep working.
type Namespace uint8

const (
	// NamespaceHTML is the namespace of ordinary HTML elements.
	NamespaceHTML Namespace = iota
	// NamespaceSVG is the SVG namespace.
	NamespaceSVG
	// NamespaceMathML is the MathML namespace.
	NamespaceMathML
)

// Attr is one attribute with its namespace. Namespace is empty for ordinary
// attributes and "xlink", "xml", or "xmlns" for attributes adjusted under the
// standard's foreign-attribute rules.
type Attr struct {
	Namespace string
	Name      string
	Value     string
}

// svgElementNames maps lowercased SVG tag names to their adjusted DOM names
// (the standard's "adjust SVG tag names" table).
var svgElementNames = map[string]string{
	"altglyph":            "altGlyph",
	"altglyphdef":         "altGlyphDef",
	"altglyphitem":        "altGlyphItem",
	"animatecolor":        "animateColor",
	"animatemotion":       "animateMotion",
	"animatetransform":    "animateTransform",
	"clippath":            "clipPath",
	"feblend":             "feBlend",
	"fecolormatrix":       "feColorMatrix",
	"fecomponenttransfer": "feComponentTransfer",
	"fecomposite":         "feComposite",
	"feconvolvematrix":    "feConvolveMatrix",
	"fediffuselighting":   "feDiffuseLighting",
	"fedisplacementmap":   "feDisplacementMap",
	"fedistantlight":      "feDistantLight",
	"fedropshadow":        "feDropShadow",
	"feflood":             "feFlood",
	"fefunca":             "feFuncA",
	"fefuncb":             "feFuncB",
	"fefuncg":             "feFuncG",
	"fefuncr":             "feFuncR",
	"fegaussianblur":      "feGaussianBlur",
	"feimage":             "feImage",
	"femerge":             "feMerge",
	"femergenode":         "feMergeNode",
	"femorphology":        "feMorphology",
	"feoffset":            "feOffset",
	"fepointlight":        "fePointLight",
	"fespecularlighting":  "feSpecularLighting",
	"fespotlight":         "feSpotLight",
	"fetile":              "feTile",
	"feturbulence":        "feTurbulence",
	"foreignobject":       "foreignObject",
	"glyphref":            "glyphRef",
	"lineargradient":      "linearGradient",
	"radialgradient":      "radialGradient",
	"textpath":            "textPath",
}

// svgAttributeNames maps lowercased SVG attribute names to their adjusted
// names (the standard's "adjust SVG attributes" table).
var svgAttributeNames = map[string]string{
	"attributename":       "attributeName",
	"attributetype":       "attributeType",
	"basefrequency":       "baseFrequency",
	"baseprofile":         "baseProfile",
	"calcmode":            "calcMode",
	"clippathunits":       "clipPathUnits",
	"diffuseconstant":     "diffuseConstant",
	"edgemode":            "edgeMode",
	"filterunits":         "filterUnits",
	"glyphref":            "glyphRef",
	"gradienttransform":   "gradientTransform",
	"gradientunits":       "gradientUnits",
	"kernelmatrix":        "kernelMatrix",
	"kernelunitlength":    "kernelUnitLength",
	"keypoints":           "keyPoints",
	"keysplines":          "keySplines",
	"keytimes":            "keyTimes",
	"lengthadjust":        "lengthAdjust",
	"limitingconeangle":   "limitingConeAngle",
	"markerheight":        "markerHeight",
	"markerunits":         "markerUnits",
	"markerwidth":         "markerWidth",
	"maskcontentunits":    "maskContentUnits",
	"maskunits":           "maskUnits",
	"numoctaves":          "numOctaves",
	"pathlength":          "pathLength",
	"patterncontentunits": "patternContentUnits",
	"patterntransform":    "patternTransform",
	"patternunits":        "patternUnits",
	"pointsatx":           "pointsAtX",
	"pointsaty":           "pointsAtY",
	"pointsatz":           "pointsAtZ",
	"preservealpha":       "preserveAlpha",
	"preserveaspectratio": "preserveAspectRatio",
	"primitiveunits":      "primitiveUnits",
	"refx":                "refX",
	"refy":                "refY",
	"repeatcount":         "repeatCount",
	"repeatdur":           "repeatDur",
	"requiredextensions":  "requiredExtensions",
	"requiredfeatures":    "requiredFeatures",
	"specularconstant":    "specularConstant",
	"specularexponent":    "specularExponent",
	"spreadmethod":        "spreadMethod",
	"startoffset":         "startOffset",
	"stddeviation":        "stdDeviation",
	"stitchtiles":         "stitchTiles",
	"surfacescale":        "surfaceScale",
	"systemlanguage":      "systemLanguage",
	"tablevalues":         "tableValues",
	"targetx":             "targetX",
	"targety":             "targetY",
	"textlength":          "textLength",
	"viewbox":             "viewBox",
	"viewtarget":          "viewTarget",
	"xchannelselector":    "xChannelSelector",
	"ychannelselector":    "yChannelSelector",
	"zoomandpan":          "zoomAndPan",
}

// foreignAttr is one entry of the standard's "adjust foreign attributes"
// table: the namespace prefix and the local name.
type foreignAttr struct {
	prefix string
	local  string
}

// foreignAttributeNames maps lowercased qualified attribute names to their
// namespace prefix and local name.
var foreignAttributeNames = map[string]foreignAttr{
	"xlink:actuate": {"xlink", "actuate"},
	"xlink:arcrole": {"xlink", "arcrole"},
	"xlink:href":    {"xlink", "href"},
	"xlink:role":    {"xlink", "role"},
	"xlink:show":    {"xlink", "show"},
	"xlink:title":   {"xlink", "title"},
	"xlink:type":    {"xlink", "type"},
	"xml:lang":      {"xml", "lang"},
	"xml:space":     {"xml", "space"},
	"xmlns":         {"xmlns", "xmlns"},
	"xmlns:xlink":   {"xmlns", "xlink"},
}

// adjustForeignElementName applies the namespace-specific element-name
// adjustment for a foreign element.
func adjustForeignElementName(ns Namespace, name string) string {
	if ns == NamespaceSVG {
		if adjusted, ok := svgElementNames[name]; ok {
			return adjusted
		}
	}

	return name
}

// adjustAttributeName resolves one lowercased token attribute into the map
// key and structured Attr stored on the node. HTML attributes keep their
// name; SVG and MathML attributes are case-adjusted; foreign attributes gain
// a namespace prefix.
func adjustAttributeName(ns Namespace, name, value string) (string, Attr) {
	switch ns {
	case NamespaceSVG:
		if adjusted, ok := svgAttributeNames[name]; ok {
			name = adjusted
		}
	case NamespaceMathML:
		if name == "definitionurl" {
			name = "definitionURL"
		}
	case NamespaceHTML:
	}

	if ns == NamespaceHTML {
		return name, Attr{Name: name, Value: value}
	}

	if foreign, ok := foreignAttributeNames[name]; ok {
		return foreign.prefix + ":" + foreign.local, Attr{
			Namespace: foreign.prefix,
			Name:      foreign.local,
			Value:     value,
		}
	}

	return name, Attr{Name: name, Value: value}
}

// isMathMLTextIntegrationPoint reports whether n is a MathML text integration
// point: mi, mo, mn, ms, or mtext.
func isMathMLTextIntegrationPoint(n *Node) bool {
	if n == nil || n.Namespace != NamespaceMathML {
		return false
	}

	switch n.Name {
	case "mi", "mo", "mn", "ms", "mtext":
		return true
	}

	return false
}

// isHTMLIntegrationPoint reports whether n is an HTML integration point:
// SVG foreignObject, desc, or title, or MathML annotation-xml with a text/html
// or application/xhtml+xml encoding attribute.
func isHTMLIntegrationPoint(n *Node) bool {
	if n == nil {
		return false
	}

	switch n.Namespace {
	case NamespaceSVG:
		switch n.Name {
		case "foreignObject", "desc", "title":
			return true
		}
	case NamespaceMathML:
		if n.Name == "annotation-xml" {
			encoding := strings.ToLower(strings.TrimSpace(n.Attribute("encoding")))

			return encoding == "text/html" || encoding == "application/xhtml+xml"
		}
	case NamespaceHTML:
	}

	return false
}

// foreignBreakoutTags are the start tags that pop foreign content and are
// reprocessed under the HTML rules.
var foreignBreakoutTags = map[string]bool{
	"b": true, "big": true, "blockquote": true, "body": true, "br": true,
	"center": true, "code": true, "dd": true, "div": true, "dl": true,
	"dt": true, "em": true, "embed": true, "h1": true, "h2": true, "h3": true,
	"h4": true, "h5": true, "h6": true, "head": true, "hr": true, "i": true,
	"img": true, "li": true, "listing": true, "menu": true, "meta": true,
	"nobr": true, "ol": true, "p": true, "pre": true, "ruby": true, "s": true,
	"small": true, "span": true, "strong": true, "strike": true, "sub": true,
	"sup": true, "table": true, "tt": true, "u": true, "ul": true, "var": true,
}

// isForeignBreakout reports whether a start tag under foreign content must be
// reprocessed with the HTML rules. font is a breakout only with a color, face,
// or size attribute.
func isForeignBreakout(tokItem *token) bool {
	if foreignBreakoutTags[tokItem.data] {
		return true
	}

	if tokItem.data == "font" {
		for i := 0; i+1 < len(tokItem.attrs); i += 2 {
			switch tokItem.attrs[i] {
			case "color", "face", "size":
				return true
			}
		}
	}

	return false
}
