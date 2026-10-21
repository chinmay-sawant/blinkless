package html

import (
	"testing"

	ihtml "github.com/chinmay-sawant/blinkless/internal/html"
)

func TestSetAttributeUpdatesTheTree(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte(`<div id="box" class="old">Hello</div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if !doc.SetAttribute("box", "class", "new") {
		t.Fatal("SetAttribute reported no element")
	}

	node := doc.elementByID("box")
	if node == nil {
		t.Fatal("box not found")
	}

	if got := node.Attribute("class"); got != "new" {
		t.Fatalf("class = %q, want %q", got, "new")
	}

	if len(node.AttrList) != 2 || node.AttrList[1].Value != "new" {
		t.Fatalf("AttrList = %+v, want the class value updated", node.AttrList)
	}
}

func TestSetAttributeAddsAMissingAttribute(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte(`<div id="box">Hello</div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if !doc.SetAttribute("box", "data-role", "banner") {
		t.Fatal("SetAttribute reported no element")
	}

	node := doc.elementByID("box")
	if node == nil {
		t.Fatal("box not found")
	}

	if got := node.Attribute("data-role"); got != "banner" {
		t.Fatalf("data-role = %q, want %q", got, "banner")
	}
}

func TestSetAttributeOnMissingElement(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte(`<div id="box">Hello</div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if doc.SetAttribute("other", "class", "new") {
		t.Fatal("SetAttribute found an element that is not there")
	}

	if doc.SetAttribute("box", "", "new") {
		t.Fatal("SetAttribute accepted an empty name")
	}
}

func TestSetTextReplacesTheContent(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte(`<div id="box"><span>Hello</span></div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if !doc.SetText("box", "Bye") {
		t.Fatal("SetText reported no element")
	}

	node := doc.elementByID("box")
	if node == nil {
		t.Fatal("box not found")
	}

	if got := node.TextContent(); got != "Bye" {
		t.Fatalf("text = %q, want %q", got, "Bye")
	}

	if len(node.Children) != 1 || node.Children[0].Type != ihtml.TextNode {
		t.Fatalf("children = %+v, want one text node", node.Children)
	}

	if node.Children[0].Parent != node {
		t.Fatal("the text node has no parent link")
	}
}

func TestSetTextOnMissingElement(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte(`<div id="box">Hello</div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if doc.SetText("other", "Bye") {
		t.Fatal("SetText found an element that is not there")
	}
}
