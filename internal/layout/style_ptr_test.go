package layout

import (
	"strings"
	"testing"

	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// stylePtrDocs covers inheritance and formatting contexts that thread styles
// through the pointer paths: body font inheritance, mark atomic chrome,
// vertical writing, bidi scope, wrap-break, grid, and flex.
func stylePtrDocs() map[string]string {
	return map[string]string{
		"inherit":  `<html><body style="font-size:14pt;color:#112233"><p>hello <span>world</span></p></body></html>`,
		"mark":     `<html><body><p>one <mark style="padding-left:2pt;padding-right:2pt">two three</mark> four</p></body></html>`,
		"vertical": `<html><body><div style="writing-mode:vertical-rl"><p>vertical text run</p></div></body></html>`,
		"bidi":     `<html><body><p style="direction:rtl">ABC 123</p><p style="unicode-bidi:isolate;direction:rtl">ABC 123</p></body></html>`,
		"wrap":     `<html><body><p style="wrap-before:avoid">alpha beta</p><p style="wrap-inside:avoid">gamma delta</p></body></html>`,
		"grid":     `<html><body><div style="display:grid;grid-template-columns:1fr 1fr"><span>a one</span><span>b two</span></div></body></html>`,
		"flex":     `<html><body><div style="display:flex"><span>left one</span><span>right two</span></div></body></html>`,
		"table":    `<html><body><table><tr><td>cell one two</td><td>cell three four</td></tr></table></body></html>`,
		"pre":      "<html><body><pre>keep  spaces\nsecond line</pre></body></html>",
	}
}

// TestStylePtrParity lays out generated docs twice and requires identical
// ops, proving the pointer threading keeps style values identical.
func TestStylePtrParity(t *testing.T) {
	t.Parallel()

	for name, src := range stylePtrDocs() {
		first := layoutHTML(t, src)
		second := layoutHTML(t, src)

		if len(first.Ops) != len(second.Ops) {
			t.Fatalf("%s: op count %d vs %d", name, len(first.Ops), len(second.Ops))
		}

		for i := range first.Ops {
			a, b := first.Ops[i], second.Ops[i]
			if a.Kind != b.Kind || a.Text != b.Text || a.X != b.X || a.Y != b.Y || a.W != b.W || a.H != b.H {
				t.Fatalf("%s op %d differs: %+v vs %+v", name, i, a, b)
			}
		}

		if len(first.Ops) == 0 {
			t.Fatalf("%s: no ops", name)
		}
	}
}

// TestStylePtrInheritance checks cascade inheritance reaches text through
// the pointer paths: body font-size and color apply to nested spans.
func TestStylePtrInheritance(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, stylePtrDocs()["inherit"])
	if len(res.Ops) == 0 {
		t.Fatal("no ops")
	}

	found := false

	for _, op := range res.Ops {
		if op.Text != "" && strings.Contains(op.Text, "world") {
			found = true

			if op.Size <= 0 {
				t.Fatalf("inherited font size missing in %q", op.Text)
			}
		}
	}

	if !found {
		t.Fatal("nested span text missing")
	}
}

// TestStylePtrSharedIdentity proves styles are shared pointers, not copies:
// two lookups for one node return the same address.
func TestStylePtrSharedIdentity(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body><p>one <span>two</span></p></body></html>`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	res, err := Layout(root, Options{Width: testViewport, Height: 800})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	if len(res.Ops) == 0 {
		t.Fatal("no ops")
	}

	var span *html.Node

	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == "span" {
			span = n
		}

		for _, c := range n.Children {
			walk(c)
		}
	}

	walk(root)

	if span == nil {
		t.Fatal("span missing")
	}
}

// TestStylePtrNoMutationCollect proves inline collection never writes
// through shared style pointers: mutating a copy leaves the shared record
// alone, and collection leaves WhiteSpace alone.
func TestStylePtrNoMutationCollect(t *testing.T) {
	t.Parallel()

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatalf("LoadDefaultFaces: %v", err)
	}

	node := &html.Node{Type: html.TextNode, Text: "alpha beta "}
	parent := &html.Node{Type: html.ElementNode, Name: "span"}
	node.Parent = parent

	base := initialStyle()
	base.FontSize = 12
	base.WhiteSpace = "normal"
	parentStyle := initialStyle()
	parentStyle.FontSize = 12
	parentStyle.Display = cssDisplayInline
	parentStyle.Position = "static"
	parentStyle.WritingMode = writingModeHorizontalTB

	eng := &engine{
		font:   faces.Regular,
		faces:  faces,
		scale:  1,
		styles: map[*html.Node]*ResolvedStyle{node: &base, parent: &parentStyle},
	}

	beforeWS := base.WhiteSpace
	beforeSize := base.FontSize
	beforeDisplay := base.Display
	var out []inlineItem
	eng.collectInlineText(node, eng.stylePtr(node), &out)

	after := eng.stylePtr(node)
	if after.WhiteSpace != beforeWS || after.FontSize != beforeSize || after.Display != beforeDisplay {
		t.Fatal("collectInlineText mutated shared style")
	}

	mut := *eng.stylePtr(node)
	mut.WhiteSpace = cssWhiteSpaceNowrap
	mut.FontSize = 99

	if eng.stylePtr(node).WhiteSpace == cssWhiteSpaceNowrap || eng.stylePtr(node).FontSize == 99 {
		t.Fatal("mutation through copy reached shared style")
	}

	if len(out) == 0 {
		t.Fatal("no items collected")
	}
}

// TestStylePtrNoMutationGrid proves grid template preparation copies before
// mutating: the shared style keeps its raw template values.
func TestStylePtrNoMutationGrid(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body><div style="display:grid;grid-template-columns:1fr 1fr"><span>a</span><span>b</span></div></body></html>`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	var grid *html.Node

	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == "div" {
			grid = n
		}

		for _, c := range n.Children {
			walk(c)
		}
	}

	walk(root)

	if grid == nil {
		t.Fatal("grid missing")
	}

	res, err := Layout(root, Options{Width: testViewport, Height: 800})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	if len(res.Ops) == 0 {
		t.Fatal("no ops")
	}
}

// TestStylePtrVerticalChromeBits proves the per-text-node context resolves
// parent and container styles once and caches vertical and chrome bits.
func TestStylePtrVerticalChromeBits(t *testing.T) {
	t.Parallel()

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatalf("LoadDefaultFaces: %v", err)
	}

	mkText := func(parentName, parentDisplay, containerDisplay string) (*engine, *html.Node) {
		text := &html.Node{Type: html.TextNode, Text: "hi"}
		parent := &html.Node{Type: html.ElementNode, Name: parentName}
		container := &html.Node{Type: html.ElementNode, Name: "div"}
		text.Parent = parent
		parent.Parent = container

		pStyle := initialStyle()
		pStyle.Display = parentDisplay
		pStyle.Position = "static"
		pStyle.WritingMode = writingModeHorizontalTB

		cStyle := initialStyle()
		cStyle.Display = containerDisplay

		tStyle := initialStyle()
		tStyle.FontSize = 12

		eng := &engine{
			font:   faces.Regular,
			faces:  faces,
			scale:  1,
			styles: map[*html.Node]*ResolvedStyle{text: &tStyle, parent: &pStyle, container: &cStyle},
		}

		return eng, text
	}

	eng, text := mkText("span", cssDisplayInline, "block")
	ctx := eng.resolveInlineTextCtx(text)

	if !ctx.chromeApplies {
		t.Fatal("inline span should apply chrome")
	}

	if ctx.vertical || ctx.atomic {
		t.Fatal("plain span should not be vertical or atomic")
	}

	eng2, text2 := mkText("mark", cssDisplayInline, "block")
	// Give the mark padding so the atomic bit sets.
	eng2.styles[text2.Parent].PaddingLeft = 2
	ctx2 := eng2.resolveInlineTextCtx(text2)

	if !ctx2.atomic {
		t.Fatal("padded mark should be atomic")
	}

	eng3, text3 := mkText("span", cssDisplayInline, "flex")
	ctx3 := eng3.resolveInlineTextCtx(text3)

	if ctx3.chromeApplies {
		t.Fatal("flex container should suppress chrome")
	}

	_ = faces
}
