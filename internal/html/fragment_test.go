package html

import "testing"

// TestParseFragmentContexts pins the internal context-aware fragment entry
// point: select ignores, RCDATA/RAWTEXT/PLAINTEXT substitution, foreign
// context elements, template mode, and the html/table/frameset seeds.
//
//nolint:cyclop // sequential context pins; splitting would hide the context list
func TestParseFragmentContexts(t *testing.T) {
	t.Parallel()

	// A select context ignores input and select start tags.
	root := parseFragment("select", NamespaceHTML, nil, `<input><select><option>a`)
	if len(root.Children) != 1 || root.Children[0].Name != "option" {
		t.Fatalf("select context children = %s, want [option]", treeString(root))
	}

	// RCDATA decodes, RAWTEXT and the script substitution do not.
	root = parseFragment("textarea", NamespaceHTML, nil, `a&amp;b`)
	if got := root.TextContent(); got != "a&b" {
		t.Errorf("textarea context text = %q, want %q", got, "a&b")
	}

	root = parseFragment("style", NamespaceHTML, nil, `a&amp;b`)
	if got := root.TextContent(); got != "a&amp;b" {
		t.Errorf("style context text = %q, want %q", got, "a&amp;b")
	}

	root = parseFragment("script", NamespaceHTML, nil, `<div>`)
	if got := root.TextContent(); got != "<div>" {
		t.Errorf("script context text = %q, want %q", got, "<div>")
	}

	// A foreign context element seeds the foreign-content rules.
	root = parseFragment("path", NamespaceSVG, nil, `<circle/>`)
	if len(root.Children) != 1 {
		t.Fatalf("svg context children = %s, want one circle", treeString(root))
	}

	circle := root.Children[0]
	if circle.Namespace != NamespaceSVG || circle.Name != "circle" {
		t.Errorf("svg context child = {%v %s}, want SVG circle", circle.Namespace, circle.Name)
	}

	// A template context pushes the in-template mode; content goes to the
	// fragment, not to a template element.
	root = parseFragment("template", NamespaceHTML, nil, `<div>x</div>`)
	if len(root.Children) != 1 || root.Children[0].Name != "div" {
		t.Fatalf("template context children = %s, want [div]", treeString(root))
	}

	// An html context builds head and body through the mode chain.
	root = parseFragment("html", NamespaceHTML, nil, `text`)
	if len(root.Children) != 2 || root.Children[0].Name != "head" || root.Children[1].Name != "body" {
		t.Fatalf("html context children = %s, want [head body]", treeString(root))
	}

	// A table context wraps rows in tbody.
	root = parseFragment("table", NamespaceHTML, nil, `<tr><td>x`)
	if len(root.Children) != 1 || root.Children[0].Name != "tbody" {
		t.Fatalf("table context children = %s, want [tbody]", treeString(root))
	}

	// A frameset context inserts frame children.
	root = parseFragment("frameset", NamespaceHTML, nil, `<frame>`)
	if len(root.Children) != 1 || root.Children[0].Name != "frame" {
		t.Fatalf("frameset context children = %s, want [frame]", treeString(root))
	}
}

// TestParseFragmentContextAttributes pins the annotation-xml integration
// point: the context encoding attribute decides whether child markup is HTML
// or MathML. The pinned corpus context lines carry no attributes, so this is
// the only coverage of the attribute path.
func TestParseFragmentContextAttributes(t *testing.T) {
	t.Parallel()

	root := parseFragment("annotation-xml", NamespaceMathML,
		[]string{"encoding", "text/html"}, `<circle/>`)
	if len(root.Children) != 1 {
		t.Fatalf("annotation-xml children = %s, want [circle]", treeString(root))
	}

	if got := root.Children[0].Namespace; got != NamespaceHTML {
		t.Errorf("text/html annotation-xml child namespace = %v, want HTML", got)
	}

	root = parseFragment("annotation-xml", NamespaceMathML, nil, `<circle/>`)
	if len(root.Children) != 1 {
		t.Fatalf("annotation-xml children = %s, want [circle]", treeString(root))
	}

	if got := root.Children[0].Namespace; got != NamespaceMathML {
		t.Errorf("bare annotation-xml child namespace = %v, want MathML", got)
	}
}

// TestParseFragmentForeignRawText pins the foreign raw-text repair: an SVG
// title is not an RCDATA element, so its swallowed content is re-tokenized in
// the Data state and following markup is processed normally.
func TestParseFragmentForeignRawText(t *testing.T) {
	t.Parallel()

	root := parseFragment("svg", NamespaceSVG, nil, `<title><tr>`)
	if len(root.Children) != 1 || root.Children[0].Name != "title" {
		t.Fatalf("svg title fragment = %s, want [title]", treeString(root))
	}

	title := root.Children[0]
	if title.Namespace != NamespaceSVG {
		t.Errorf("title namespace = %v, want SVG", title.Namespace)
	}

	// SVG title is an HTML integration point: the tr start tag is ignored by
	// the HTML table rules instead of becoming a child of title.
	if len(title.Children) != 0 {
		t.Errorf("svg title children = %s, want none", treeString(title))
	}
}
