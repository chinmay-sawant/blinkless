package html

import "testing"

//
//nolint:cyclop // sequential template-content pins; splitting would hide the list
func TestParseTemplateContentsAreSeparate(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<body><template><div>hi</div><p>x</p></template></body>`)
	body := firstElement(root, "body")
	tmpl := body.FirstChild("template")

	if tmpl == nil {
		t.Fatalf("no template element in tree:\n%s", treeString(root))
	}

	if len(tmpl.Children) != 0 {
		t.Fatalf("template Children = %d, want 0 (content lives in Contents)", len(tmpl.Children))
	}

	if len(tmpl.Contents) != 2 {
		t.Fatalf("template Contents = %d, want 2:\n%s", len(tmpl.Contents), treeString(tmpl))
	}

	if tmpl.Contents[0].Name != "div" || tmpl.Contents[1].Name != "p" {
		t.Fatalf("template contents = [%s %s], want [div p]", tmpl.Contents[0].Name, tmpl.Contents[1].Name)
	}

	// DOM semantics: template contents are a separate fragment, so text,
	// walking, and search all skip them.
	if got := tmpl.TextContent(); got != "" {
		t.Errorf("template TextContent = %q, want empty", got)
	}

	if got := body.TextContent(); got != "" {
		t.Errorf("body TextContent = %q, want empty (template content is inert)", got)
	}

	visited := 0

	tmpl.Walk(func(node *Node) {
		if node.Name == "div" || node.Name == "p" {
			visited++
		}
	})

	if visited != 0 {
		t.Errorf("Walk visited %d template-content nodes, want 0", visited)
	}

	if found := root.FindFirst(func(node *Node) bool { return node.Name == "div" }); found != nil {
		t.Errorf("FindFirst found a template-content div, want nil")
	}

	if got := root.TextContentOf("div"); got != "" {
		t.Errorf("TextContentOf(div) = %q, want empty", got)
	}
}

func TestParseSelectStates(t *testing.T) {
	t.Parallel()

	// hr generates implied end tags, closing option and optgroup before it.
	root := mustParse(t, `<select><optgroup><option>a<option>b<hr>`)
	sel := firstElement(root, "select")
	assertChildren(t, sel, "optgroup", "hr")
	assertChildren(t, sel.FirstChild("optgroup"), "option", "option")

	// Arbitrary content inside select parses with the in-body rules.
	root = mustParse(t, `<select><div>x</div><option>y`)
	sel = firstElement(root, "select")
	assertChildren(t, sel, "div", "option")

	if got := sel.FirstChild("div").TextContent(); got != "x" {
		t.Errorf("select div text = %q, want %q", got, "x")
	}

	// input closes the select and lands outside it.
	root = mustParse(t, `<select><option>a<input>`)
	body := firstElement(root, "body")
	assertChildren(t, body, "select", "input")

	// A nested select start tag pops the outer select and is ignored.
	root = mustParse(t, `<select><button><div><select></select>`)
	sel = firstElement(root, "select")
	assertChildren(t, sel, "button")
	assertChildren(t, sel.FirstChild("button"), "div")

	// A formatting element blocked by select scope is removed from the
	// active formatting list, not closed.
	root = mustParse(t, `<font><select><option>a</option></font></select>`)
	font := firstElement(root, "font")
	assertChildren(t, font, "select")
	assertChildren(t, font.FirstChild("select"), "option")
}

func TestParseSelectedContentMirror(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<select><button><selectedcontent></button><option>X`)
	sel := firstElement(root, "select")
	selectedContent := sel.FirstChild("button").FirstChild("selectedcontent")

	if got := selectedContent.TextContent(); got != "X" {
		t.Errorf("selectedcontent text = %q, want %q", got, "X")
	}

	if got := sel.FirstChild("option").TextContent(); got != "X" {
		t.Errorf("option text = %q, want %q", got, "X")
	}

	// The last option with the selected attribute wins.
	root = mustParse(t, `<select><button><selectedcontent></button><option>X<option selected>Y`)
	sel = firstElement(root, "select")
	selectedContent = sel.FirstChild("button").FirstChild("selectedcontent")

	if got := selectedContent.TextContent(); got != "Y" {
		t.Errorf("selectedcontent text with selected= = %q, want %q", got, "Y")
	}

	// The clone mirrors reconstructed formatting children.
	root = mustParse(t, `<select><button><selectedcontent></button><option>x<i>i<b>ib</i>b`)
	sel = firstElement(root, "select")
	selectedContent = sel.FirstChild("button").FirstChild("selectedcontent")
	option := sel.FirstChild("option")

	if len(selectedContent.Children) != len(option.Children) {
		t.Fatalf("selectedcontent children = %d, want %d (same as option)",
			len(selectedContent.Children), len(option.Children))
	}

	if selectedContent.Children[0].Type != TextNode || selectedContent.Children[0].Text != "x" {
		t.Errorf("selectedcontent first child = %+v, want text %q", selectedContent.Children[0], "x")
	}

	if selectedContent.Children[2].Name != "b" {
		t.Errorf("selectedcontent last child = %q, want b", selectedContent.Children[2].Name)
	}
}

func TestParseTemplateInsertionModes(t *testing.T) {
	t.Parallel()

	// A template inside a table keeps its content separate from the table.
	root := mustParse(t, `<table><template><tr><td>cell</template></table>`)
	table := firstElement(root, "table")
	tmpl := table.FirstChild("template")

	if tmpl == nil || len(tmpl.Children) != 0 {
		t.Fatalf("table template = %+v, want element with empty Children", tmpl)
	}

	if len(tmpl.Contents) != 1 || tmpl.Contents[0].Name != "tr" {
		t.Fatalf("table template contents = %s, want [tr]", treeString(tmpl))
	}

	assertChildren(t, tmpl.Contents[0], "td")

	if got := tmpl.Contents[0].FirstChild("td").TextContent(); got != "cell" {
		t.Errorf("cell text = %q, want %q", got, "cell")
	}

	// EOF inside an open template still closes it and builds body.
	root = mustParse(t, `<template><div>`)
	html := firstElement(root, "html")
	assertChildren(t, html, "head", "body")

	head := html.FirstChild("head")
	tmpl = head.FirstChild("template")

	if tmpl == nil || len(tmpl.Contents) != 1 || tmpl.Contents[0].Name != "div" {
		t.Fatalf("EOF template = %+v, want one div in Contents", tmpl)
	}

	// Framesets nest and frames never take content.
	root = mustParse(t, `<frameset><frame><frameset><frame></frameset><noframes></noframes></frameset>`)
	html = firstElement(root, "html")
	assertChildren(t, html, "head", "frameset")

	outer := html.FirstChild("frameset")
	assertChildren(t, outer, "frame", "frameset", "noframes")
	assertChildren(t, outer.FirstChild("frameset"), "frame")
}
