package markup_test

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/markup"
)

func TestParseDivAndScript(t *testing.T) {
	t.Parallel()

	root, err := markup.Parse([]byte(
		`<div id="user" data-action="focus">Ada</div><script>alert(1)</script>`,
	))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	div := find(root, func(node *markup.Node) bool {
		return node.Type == markup.TypeElement && node.Name == "div"
	})
	if div == nil {
		t.Fatal("div not found")
	}

	if div.Attrs["id"] != "user" || div.Attrs["data-action"] != "focus" {
		t.Fatalf("attrs = %#v", div.Attrs)
	}

	if text := nodeText(div); text != "Ada" {
		t.Fatalf("div text = %q", text)
	}

	script := find(root, func(node *markup.Node) bool {
		return node.Type == markup.TypeElement && node.Name == "script"
	})
	if script == nil {
		t.Fatal("script not found")
	}

	if text := nodeText(script); text != "alert(1)" {
		t.Fatalf("script text = %q", text)
	}
}

func TestParseCopyCarriesDocumentMode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		src  string
		want markup.DocumentMode
	}{
		{name: "standards", src: `<!DOCTYPE html><p>x`, want: markup.NoQuirks},
		{
			name: "limited quirks",
			src: `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" ` +
				`"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd"><p>x`,
			want: markup.LimitedQuirks,
		},
		{name: "quirks", src: `<p>x`, want: markup.Quirks},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root, err := markup.Parse([]byte(testCase.src))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			if root.Mode != testCase.want {
				t.Fatalf("root.Mode = %s, want %s", root.Mode, testCase.want)
			}
		})
	}
}

func TestParseCopyCarriesDoctype(t *testing.T) {
	t.Parallel()

	const src = `<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.01//EN" ` +
		`"http://www.w3.org/TR/html4/strict.dtd"><p>x</p>`

	root, err := markup.Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	doctype := find(root, func(node *markup.Node) bool { return node.Type == markup.TypeDoctype })
	if doctype == nil {
		t.Fatal("doctype node not found")
	}

	if doctype.Text == "" {
		t.Error("doctype raw text is empty")
	}

	want := markup.Doctype{
		Name:        "html",
		PublicID:    "-//W3C//DTD HTML 4.01//EN",
		HasPublicID: true,
		SystemID:    "http://www.w3.org/TR/html4/strict.dtd",
		HasSystemID: true,
	}
	if doctype.Doctype != want {
		t.Fatalf("doctype = %+v, want %+v", doctype.Doctype, want)
	}
}

func TestParseCopyCarriesForeignFields(t *testing.T) {
	t.Parallel()

	const src = `<svg viewBox="0 0 10 10"><path xlink:href="#p"/></svg>`

	root, err := markup.Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	svg := find(root, func(node *markup.Node) bool { return node.Name == "svg" })
	if svg == nil {
		t.Fatal("svg node not found")
	}

	if svg.Namespace != markup.NamespaceSVG {
		t.Errorf("svg namespace = %v, want SVG", svg.Namespace)
	}

	if got := svg.Attrs["viewBox"]; got != "0 0 10 10" {
		t.Errorf("svg viewBox = %q, want adjusted key viewBox", got)
	}

	path := find(root, func(node *markup.Node) bool { return node.Name == "path" })
	if path == nil {
		t.Fatal("path node not found")
	}

	if path.Namespace != markup.NamespaceSVG {
		t.Errorf("path namespace = %v, want SVG", path.Namespace)
	}

	if len(path.AttrList) != 1 {
		t.Fatalf("path AttrList = %+v, want one xlink entry", path.AttrList)
	}

	wantAttr := markup.Attr{Namespace: "xlink", Name: "href", Value: "#p"}
	if path.AttrList[0] != wantAttr {
		t.Errorf("path AttrList[0] = %+v, want %+v", path.AttrList[0], wantAttr)
	}
}

func TestParseCopyCarriesMathMLNamespace(t *testing.T) {
	t.Parallel()

	root, err := markup.Parse([]byte(`<math><mi>x</mi></math>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	math := find(root, func(node *markup.Node) bool { return node.Name == "math" })
	if math == nil {
		t.Fatal("math node not found")
	}

	if math.Namespace != markup.NamespaceMathML {
		t.Errorf("math namespace = %v, want MathML", math.Namespace)
	}

	miNode := find(root, func(node *markup.Node) bool { return node.Name == "mi" })
	if miNode == nil {
		t.Fatal("mi node not found")
	}

	if miNode.Namespace != markup.NamespaceMathML {
		t.Errorf("mi namespace = %v, want MathML", miNode.Namespace)
	}
}

func TestParseCopyCarriesTemplateContents(t *testing.T) {
	t.Parallel()

	root, err := markup.Parse([]byte(
		`<div><template><style>.in-template { color: #f00 }</style></template></div>`,
	))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	template := find(root, func(node *markup.Node) bool { return node.Name == "template" })
	if template == nil {
		t.Fatal("template node not found")
	}

	if len(template.Children) != 0 {
		t.Fatalf("template Children = %d, want content kept out of Children", len(template.Children))
	}

	if len(template.Contents) != 1 {
		t.Fatalf("template Contents = %d nodes, want 1", len(template.Contents))
	}

	style := template.Contents[0]
	if style.Type != markup.TypeElement || style.Name != "style" {
		t.Fatalf("template content = %+v, want the style element", style)
	}

	if got := nodeText(style); got != ".in-template { color: #f00 }" {
		t.Fatalf("template style text = %q", got)
	}
}

func find(node *markup.Node, match func(*markup.Node) bool) *markup.Node {
	if node == nil {
		return nil
	}

	if match(node) {
		return node
	}

	for _, child := range node.Children {
		if found := find(child, match); found != nil {
			return found
		}
	}

	return nil
}

func nodeText(node *markup.Node) string {
	if node == nil {
		return ""
	}

	text := node.Text

	for _, child := range node.Children {
		text += nodeText(child)
	}

	return text
}
