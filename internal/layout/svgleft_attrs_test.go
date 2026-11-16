package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// svgLeftSerialize serializes one svg that holds inner, under a stylesheet.
func svgLeftSerialize(t *testing.T, sheetText, inner string) string {
	t.Helper()

	root := mustParse(t, `<html><body><svg width="64" height="28" viewBox="0 0 64 28" xmlns="http://www.w3.org/2000/svg">`+
		inner+`</svg></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `body{margin:0}`+sheetText)}, "print", 500, 800)
	eng := &engine{styles: styles, scale: 1}

	var svgNode *html.Node

	root.Walk(func(node *html.Node) {
		if svgNode == nil && node.Type == html.ElementNode && node.Name == "svg" {
			svgNode = node
		}
	})

	if svgNode == nil {
		t.Fatal("svg missing")
	}

	return string(eng.serializeInlineSVG(svgNode))
}

// TestSvgLeftoversBakeFromStylesheet: each of the five names reaches the
// serialized SVG from a stylesheet rule, and the same markup without the rule
// does not carry the attribute. The canvas ignores all five, so there is no
// pixel claim.
func TestSvgLeftoversBakeFromStylesheet(t *testing.T) {
	t.Parallel()

	cases := []struct{ name, sheet, inner, attr string }{
		{
			"text-rendering", `text{text-rendering:optimizeSpeed}`,
			`<text x="2" y="12">hi</text>`, `text-rendering="optimizeSpeed"`,
		},
		{
			"glyph-orientation-vertical", `text{glyph-orientation-vertical:90}`,
			`<text x="2" y="12">hi</text>`, `glyph-orientation-vertical="90deg"`,
		},
		{
			"baseline-shift", `tspan{baseline-shift:super}`,
			`<text x="2" y="12">a<tspan>b</tspan></text>`, `baseline-shift="super"`,
		},
		{
			"lighting-color", `feDiffuseLighting{lighting-color:#336699}`,
			`<filter id="f"><feDiffuseLighting/></filter><rect width="4" height="4" filter="url(#f)"/>`,
			`lighting-color="#336699"`,
		},
		{
			"marker-side", `path{marker-side:match-parent}`,
			`<path d="M0 0L10 10"/>`, `marker-side="match-parent"`,
		},
	}

	for _, c := range cases {
		with := svgLeftSerialize(t, c.sheet, c.inner)
		without := svgLeftSerialize(t, "", c.inner)

		if !strings.Contains(with, c.attr) {
			t.Errorf("%s: serialized SVG lacks %s, got %s", c.name, c.attr, with)
		}

		if strings.Contains(without, c.name+"=") {
			t.Errorf("%s: attribute present without the rule: %s", c.name, without)
		}
	}
}

// TestSvgLeftoversInlineNoDuplicate: an inline text-rendering is written once.
func TestSvgLeftoversInlineNoDuplicate(t *testing.T) {
	t.Parallel()

	out := svgLeftSerialize(t, "", `<text x="2" y="12" style="text-rendering:optimizeLegibility">hi</text>`)
	if strings.Count(out, "text-rendering=") != 1 {
		t.Errorf("want one text-rendering attribute, got %s", out)
	}
}
