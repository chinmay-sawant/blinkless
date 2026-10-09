package markup

import (
	"fmt"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// Type classifies a parsed node.
type Type int

const (
	// TypeElement is a tag. Name is lowercase.
	TypeElement Type = iota + 1
	// TypeText is character data.
	TypeText
	// TypeComment is an HTML comment.
	TypeComment
	// TypeDoctype is the doctype declaration.
	TypeDoctype
)

// Namespace identifies the DOM namespace an element belongs to. The zero
// value is the HTML namespace.
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
// attributes and "xlink", "xml", or "xmlns" for adjusted foreign attributes.
type Attr struct {
	Namespace string
	Name      string
	Value     string
}

// Node is one parsed HTML node. Children are in document order.
// There is no parent pointer. Attribute keys are lowercased for HTML elements
// and adjusted (for example viewBox) for foreign elements.
type Node struct {
	Type      Type
	Name      string
	Namespace Namespace
	Attrs     map[string]string
	AttrList  []Attr
	Text      string
	Children  []*Node
}

// Parse parses UTF-8 HTML and returns an owned copy of the tree.
// A nil source is parsed as an empty document.
func Parse(source []byte) (*Node, error) {
	root, err := html.ParseDocument(source)
	if err != nil {
		return nil, fmt.Errorf("markup: parse: %w", err)
	}

	return copyNode(root), nil
}

func copyNode(node *html.Node) *Node {
	if node == nil {
		return nil
	}

	kind, ok := copyType(node.Type)
	if !ok {
		return nil
	}

	children := make([]*Node, 0, len(node.Children))

	for _, child := range node.Children {
		copied := copyNode(child)
		if copied != nil {
			children = append(children, copied)
		}
	}

	return &Node{
		Type:      kind,
		Name:      node.Name,
		Namespace: copyNamespace(node.Namespace),
		Attrs:     copyAttrs(node.Attrs),
		AttrList:  copyAttrList(node.AttrList),
		Text:      node.Text,
		Children:  children,
	}
}

func copyNamespace(ns html.Namespace) Namespace {
	switch ns {
	case html.NamespaceSVG:
		return NamespaceSVG
	case html.NamespaceMathML:
		return NamespaceMathML
	case html.NamespaceHTML:
		return NamespaceHTML
	default:
		return NamespaceHTML
	}
}

func copyAttrList(attrs []html.Attr) []Attr {
	if len(attrs) == 0 {
		return nil
	}

	out := make([]Attr, 0, len(attrs))
	for _, attr := range attrs {
		out = append(out, Attr{Namespace: attr.Namespace, Name: attr.Name, Value: attr.Value})
	}

	return out
}

func copyType(kind html.NodeType) (Type, bool) {
	switch kind {
	case html.ElementNode:
		return TypeElement, true
	case html.TextNode:
		return TypeText, true
	case html.CommentNode:
		return TypeComment, true
	case html.DoctypeNode:
		return TypeDoctype, true
	case html.NodeUnknown:
		return 0, false
	default:
		return 0, false
	}
}

func copyAttrs(attrs map[string]string) map[string]string {
	if len(attrs) == 0 {
		return nil
	}

	out := make(map[string]string, len(attrs))

	for key, value := range attrs {
		out[key] = value
	}

	return out
}
