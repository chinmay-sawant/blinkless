package html

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// RemoveAttribute deletes the named attribute from the element with the given
// id and reports whether an attribute was removed.
//
// Matching is case-insensitive on HTML elements and exact on foreign (SVG or
// MathML) elements. A missing element, missing attribute, or empty name changes
// nothing and reports false without bumping MutationVersion or marking dirty.
func (d *Document) RemoveAttribute(elementID, name string) bool {
	if name == "" || d == nil || d.root == nil || elementID == "" {
		return false
	}

	dirty := dirtyFor(d)

	idx := elementIndexFor(d)
	idx.mu.Lock()
	defer idx.mu.Unlock()

	node := idx.findLocked(d.root, elementID)
	if node == nil {
		return false
	}

	foreign := node.Namespace != html.NamespaceHTML
	if !removeNodeAttrLocked(node, name, foreign) {
		return false
	}

	bumpMutationLocked(d.root, node, false)

	removedID := (!foreign && strings.EqualFold(name, "id")) || (foreign && name == "id")
	if removedID {
		invalidateIDLocked(idx, node, elementID, "")
	}

	dirty.Mark(elementID)

	return true
}

// AddClass adds className to the element's class list and reports whether the
// class list changed.
//
// Empty, whitespace-only, or multi-token class names are ignored. If the class
// is already present, nothing changes and false is returned.
func (d *Document) AddClass(elementID, className string) bool {
	if d == nil || d.root == nil || elementID == "" {
		return false
	}

	token, ok := normalizeClassToken(className)
	if !ok {
		return false
	}

	dirty := dirtyFor(d)

	idx := elementIndexFor(d)
	idx.mu.Lock()
	defer idx.mu.Unlock()

	node := idx.findLocked(d.root, elementID)
	if node == nil {
		return false
	}

	tokens := strings.Fields(node.Attribute("class"))
	if slices.Contains(tokens, token) {
		return false
	}

	tokens = append(tokens, token)
	setNodeAttrLocked(node, "class", strings.Join(tokens, " "))

	bumpMutationLocked(d.root, node, false)
	dirty.Mark(elementID)

	return true
}

// RemoveClass removes className from the element's class list and reports
// whether the class list changed.
//
// Empty, whitespace-only, or multi-token class names are ignored. If the class
// is not present, nothing changes and false is returned.
func (d *Document) RemoveClass(elementID, className string) bool {
	if d == nil || d.root == nil || elementID == "" {
		return false
	}

	token, ok := normalizeClassToken(className)
	if !ok {
		return false
	}

	dirty := dirtyFor(d)

	idx := elementIndexFor(d)
	idx.mu.Lock()
	defer idx.mu.Unlock()

	node := idx.findLocked(d.root, elementID)
	if node == nil {
		return false
	}

	tokens := strings.Fields(node.Attribute("class"))
	kept := make([]string, 0, len(tokens))
	removed := false

	for _, existing := range tokens {
		if existing == token {
			removed = true

			continue
		}

		kept = append(kept, existing)
	}

	if !removed {
		return false
	}

	setNodeAttrLocked(node, "class", strings.Join(kept, " "))

	bumpMutationLocked(d.root, node, false)
	dirty.Mark(elementID)

	return true
}

// ToggleClass adds className if absent or removes it if present.
//
// It returns added (true when the class is now present) and ok (true when the
// element existed and className was a valid single class token).
func (d *Document) ToggleClass(elementID, className string) (added, ok bool) {
	if d == nil || d.root == nil || elementID == "" {
		return false, false
	}

	token, valid := normalizeClassToken(className)
	if !valid {
		return false, false
	}

	dirty := dirtyFor(d)

	idx := elementIndexFor(d)
	idx.mu.Lock()
	defer idx.mu.Unlock()

	node := idx.findLocked(d.root, elementID)
	if node == nil {
		return false, false
	}

	tokens := strings.Fields(node.Attribute("class"))
	kept := make([]string, 0, len(tokens))
	found := false

	for _, existing := range tokens {
		if existing == token {
			found = true

			continue
		}

		kept = append(kept, existing)
	}

	if found {
		setNodeAttrLocked(node, "class", strings.Join(kept, " "))
		added = false
	} else {
		tokens = append(tokens, token)
		setNodeAttrLocked(node, "class", strings.Join(tokens, " "))
		added = true
	}

	bumpMutationLocked(d.root, node, false)
	dirty.Mark(elementID)

	return added, true
}

// RemoveElement detaches the element with the given id from its parent and
// reports whether an element was removed.
//
// Removing the document root itself is refused. Detached descendant ids are
// cleared from the id index, the parent (if id-bearing) is marked dirty, and
// the structural dirty flag is set.
func (d *Document) RemoveElement(elementID string) bool {
	if d == nil || d.root == nil || elementID == "" {
		return false
	}

	dirty := dirtyFor(d)

	idx := elementIndexFor(d)
	idx.mu.Lock()
	defer idx.mu.Unlock()

	node := idx.findLocked(d.root, elementID)
	if node == nil || node == d.root || node.Parent == nil {
		return false
	}

	parent := node.Parent
	pos := slices.Index(parent.Children, node)
	if pos < 0 {
		return false
	}

	copy(parent.Children[pos:], parent.Children[pos+1:])
	parent.Children[len(parent.Children)-1] = nil
	parent.Children = parent.Children[:len(parent.Children)-1]

	parent.ChildVersion++
	d.root.MutationVersion++

	forgetSubtreesLocked(idx, node)

	if parentID := parent.Attribute("id"); parentID != "" {
		dirty.Mark(parentID)
	}

	dirty.MarkStructural()

	return true
}

// SetInnerHTML replaces the children of the element with parentID by parsing
// fragment as HTML and attaching the resulting nodes. It reports whether the
// parent element existed.
func (d *Document) SetInnerHTML(parentID, fragment string) (bool, error) {
	if d == nil || d.root == nil || parentID == "" {
		return false, nil
	}

	dirty := dirtyFor(d)

	idx := elementIndexFor(d)
	idx.mu.Lock()
	defer idx.mu.Unlock()

	parent := idx.findLocked(d.root, parentID)
	if parent == nil {
		return false, nil
	}

	children, err := parseFragmentChildren(parent, fragment)
	if err != nil {
		return false, err
	}

	if len(parent.Children) == 0 && len(children) == 0 {
		return true, nil
	}

	if len(parent.Children) > 0 {
		dropChildrenLocked(idx, parent)
	} else {
		parent.ChildVersion++
	}

	if len(children) > 0 {
		for _, child := range children {
			child.Parent = parent
		}

		parent.Children = children
		invalidateInsertedIDsLocked(idx, children...)
	}

	d.root.MutationVersion++
	dirty.Mark(parentID)
	dirty.MarkStructural()

	return true, nil
}

// AppendHTML parses fragment as HTML and appends the resulting child nodes to
// the element with parentID. It reports whether the parent element existed.
func (d *Document) AppendHTML(parentID, fragment string) (bool, error) {
	if d == nil || d.root == nil || parentID == "" {
		return false, nil
	}

	dirty := dirtyFor(d)

	idx := elementIndexFor(d)
	idx.mu.Lock()
	defer idx.mu.Unlock()

	parent := idx.findLocked(d.root, parentID)
	if parent == nil {
		return false, nil
	}

	children, err := parseFragmentChildren(parent, fragment)
	if err != nil {
		return false, err
	}

	if len(children) == 0 {
		return true, nil
	}

	for _, child := range children {
		child.Parent = parent
	}

	parent.Children = append(parent.Children, children...)
	parent.ChildVersion++
	d.root.MutationVersion++

	invalidateInsertedIDsLocked(idx, children...)

	dirty.Mark(parentID)
	dirty.MarkStructural()

	return true, nil
}

// removeNodeAttrLocked deletes matching attribute entries from both node.Attrs
// and node.AttrList, reporting whether anything was removed.
func removeNodeAttrLocked(node *html.Node, name string, foreign bool) bool {
	removed := false

	if node.Attrs != nil {
		if foreign {
			if _, ok := node.Attrs[name]; ok {
				delete(node.Attrs, name)
				removed = true
			}
		} else {
			for k := range node.Attrs {
				if strings.EqualFold(k, name) {
					delete(node.Attrs, k)
					removed = true
				}
			}
		}
	}

	if len(node.AttrList) > 0 {
		dst := 0

		for i := range node.AttrList {
			key := node.AttrList[i].Key()
			match := key == name
			if !foreign {
				match = strings.EqualFold(key, name)
			}

			if match {
				removed = true

				continue
			}

			node.AttrList[dst] = node.AttrList[i]
			dst++
		}

		if dst < len(node.AttrList) {
			clear(node.AttrList[dst:])
			node.AttrList = node.AttrList[:dst]
		}
	}

	return removed
}

// normalizeClassToken trims surrounding whitespace and rejects empty or
// multi-token class names.
func normalizeClassToken(className string) (string, bool) {
	token := strings.TrimSpace(className)
	if token == "" || strings.IndexFunc(token, unicode.IsSpace) >= 0 {
		return "", false
	}

	return token, true
}

// invalidateInsertedIDsLocked clears cached resolutions for any ids carried by
// newly inserted subtrees and marks the index incomplete so the next lookup
// resolves in document order. The caller holds idx.mu.
func invalidateInsertedIDsLocked(idx *elementIndex, roots ...*html.Node) {
	idx.complete = false

	for _, root := range roots {
		root.Walk(func(inner *html.Node) {
			if inner.Type != html.ElementNode {
				return
			}

			if id := inner.Attribute("id"); id != "" {
				delete(idx.byID, id)
			}
		})
	}
}

// parseFragmentChildren parses fragment in the context of parent and returns
// the detached child nodes without synthetic html/head/body wrappers.
func parseFragmentChildren(parent *html.Node, fragment string) ([]*html.Node, error) {
	if fragment == "" {
		return nil, nil
	}

	if len(fragment) > MaxParseBytes {
		return nil, fmt.Errorf("html: fragment exceeds %d byte limit", MaxParseBytes)
	}

	wrapped, path := wrapFragmentForContext(parent, fragment)

	parsed, err := html.ParseWithLimit(wrapped, len(wrapped))
	if err != nil {
		return nil, fmt.Errorf("html: parse fragment: %w", err)
	}

	container := parsed
	for _, tag := range path {
		container = container.FirstChild(tag)
		if container == nil {
			return nil, nil
		}
	}

	if len(container.Children) == 0 {
		return nil, nil
	}

	children := make([]*html.Node, len(container.Children))
	copy(children, container.Children)
	container.Children = nil

	for _, child := range children {
		child.Parent = nil
	}

	return children, nil
}

// wrapFragmentForContext wraps fragment in the minimal container needed for the
// HTML parser to parse it in parent's insertion mode, and returns the tag path
// from #document to that container.
func wrapFragmentForContext(parent *html.Node, fragment string) (wrapped string, path []string) {
	if parent != nil {
		switch parent.Namespace {
		case html.NamespaceSVG:
			return "<body><svg>" + fragment + "</svg></body>", []string{"html", "body", "svg"}
		case html.NamespaceMathML:
			return "<body><math>" + fragment + "</math></body>", []string{"html", "body", "math"}
		case html.NamespaceHTML:
			switch parent.Name {
			case "table":
				return "<body><table>" + fragment + "</table></body>", []string{"html", "body", "table"}
			case "tbody", "thead", "tfoot":
				return "<body><table><" + parent.Name + ">" + fragment + "</" + parent.Name + "></table></body>",
					[]string{"html", "body", "table", parent.Name}
			case "tr":
				return "<body><table><tbody><tr>" + fragment + "</tr></tbody></table></body>",
					[]string{"html", "body", "table", "tbody", "tr"}
			case "select":
				return "<body><select>" + fragment + "</select></body>", []string{"html", "body", "select"}
			}
		}
	}

	return "<body>" + fragment + "</body>", []string{"html", "body"}
}
