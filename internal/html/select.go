// Selected-content mirroring for the customizable select parsing model. The
// pinned html5lib corpus (html5lib-tests issue #180) expects an option's
// content to be cloned into the select's selectedcontent element even when no
// explicit </option> appears, with selectedness resolved as: the last option
// carrying the selected attribute wins, otherwise the first option. Because
// this engine builds the whole tree before returning, the mirroring runs as
// one post-pass over the finished tree.
//
//nolint:all // post-pass over the custom Node model; same conventions as tree.go
package html

// applySelectedContent fills every selectedcontent element with a clone of
// the selected option's children.
func applySelectedContent(root *Node) {
	root.Walk(func(node *Node) {
		if node.Type == ElementNode && node.Namespace == NamespaceHTML && node.Name == "select" {
			mirrorSelectedContent(node)
		}
	})
}

// mirrorSelectedContent finds the first selectedcontent descendant and the
// selected option of one select element, then replaces the selectedcontent
// children with a deep clone of the option's children.
func mirrorSelectedContent(selectNode *Node) {
	var (
		selectedContent *Node
		options         []*Node
	)

	selectNode.Walk(func(node *Node) {
		if node.Type != ElementNode || node.Namespace != NamespaceHTML {
			return
		}

		switch node.Name {
		case "selectedcontent":
			if selectedContent == nil {
				selectedContent = node
			}
		case "option":
			options = append(options, node)
		}
	})

	if selectedContent == nil {
		return
	}

	selected := (*Node)(nil)

	for _, option := range options {
		if _, ok := option.Attrs["selected"]; ok {
			selected = option
		}
	}

	if selected == nil && len(options) > 0 {
		selected = options[0]
	}

	selectedContent.Children = nil

	if selected == nil {
		return
	}

	for _, child := range selected.Children {
		selectedContent.Children = append(selectedContent.Children, cloneSubtree(child))
	}
}

// cloneSubtree deep-copies a node and all its rendered children, preserving
// attributes, namespace, and text. Template contents are copied too so a
// cloned template stays complete.
func cloneSubtree(node *Node) *Node {
	clone := &Node{ //nolint:exhaustruct
		Type:      node.Type,
		Name:      node.Name,
		Namespace: node.Namespace,
		Text:      node.Text,
		Doctype:   node.Doctype,
	}

	if len(node.Attrs) > 0 {
		clone.Attrs = make(map[string]string, len(node.Attrs))

		for key, value := range node.Attrs {
			clone.Attrs[key] = value
		}
	}

	if len(node.AttrList) > 0 {
		clone.AttrList = append([]Attr(nil), node.AttrList...)
	}

	for _, child := range node.Children {
		childClone := cloneSubtree(child)
		childClone.Parent = clone
		clone.Children = append(clone.Children, childClone)
	}

	for _, child := range node.Contents {
		childClone := cloneSubtree(child)
		childClone.Parent = clone
		clone.Contents = append(clone.Contents, childClone)
	}

	return clone
}

// --- select and template state helpers ---

// selectInScope reports whether a select element is in the standard's select
// scope (the default scope list already ends at select).
func (b *treeBuilder) selectInScope() bool {
	return b.hasInScope("select", defaultScopeStops)
}

// popUntilSelectPopped pops elements until a select element has been popped.
func (b *treeBuilder) popUntilSelectPopped() {
	for len(b.stack) > 1 {
		top := b.top()
		b.stack = b.stack[:len(b.stack)-1]

		if top.Namespace == NamespaceHTML && top.Name == "select" {
			return
		}
	}
}

// isTemplateElement reports whether node is an HTML template element. Foreign
// elements named "template" (SVG) are not template elements.
