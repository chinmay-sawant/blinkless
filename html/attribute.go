package html

import (
	"strings"
	"sync"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// SetAttribute sets an attribute on the element with the given id and
// reports whether the element exists.
//
// The name is matched the way the parser stores attributes: an attribute the
// element already carries keeps its stored spelling, a new one is stored
// lowercased (kept as given on an SVG or MathML element, where names such as
// viewBox are case-sensitive). An empty name changes nothing and reports false.
//
// The change lands on the parsed tree, so the next style cascade and the
// next layout see it.
func (d *Document) SetAttribute(elementID, name, value string) bool {
	if name == "" || d == nil || d.root == nil || elementID == "" {
		return false
	}

	// Resolve the dirty set before taking the index lock so the registry
	// lock never nests inside it; marking below only takes the set lock.
	dirty := dirtyFor(d)

	idx := elementIndexFor(d)
	idx.mu.Lock()
	defer idx.mu.Unlock()

	node := idx.findLocked(d.root, elementID)
	if node == nil {
		return false
	}

	setNodeAttrLocked(node, name, value)

	bumpMutationLocked(d.root, node, false)

	// The element's identity changed. Drop the cached resolutions that may
	// now point at the wrong node; the next lookup re-resolves by walk.
	// Identity edits are rare, so lazy repair keeps the common path O(1).
	renamedID := strings.EqualFold(name, "id")
	if renamedID {
		invalidateIDLocked(idx, node, elementID, value)
	}

	// The write landed: the target is dirty, and an id rename dirties the
	// new id too so both resolutions re-resolve. Mark is a no-op for "".
	dirty.Mark(elementID)

	if renamedID {
		dirty.Mark(value)
	}

	return true
}

// bumpMutationLocked advances MutationVersion and, when the edit cannot affect
// collected stylesheets, also advances NonSheetMutations so css.Relayout can
// reuse its collected sheets without walking the DOM.
func bumpMutationLocked(root, target *html.Node, structural bool) {
	root.MutationVersion++
	if !structural && !isSheetElement(target) {
		root.NonSheetMutations++
	}
}

// isSheetElement reports whether node is a <style> or <link> element (or sits
// inside a <style> element) whose edits can change collected stylesheets.
func isSheetElement(node *html.Node) bool {
	for cur := node; cur != nil; cur = cur.Parent {
		if cur.Type == html.ElementNode &&
			(strings.EqualFold(cur.Name, "style") || strings.EqualFold(cur.Name, "link")) {
			return true
		}
	}

	return false
}

// invalidateIDLocked drops the cached resolutions that may now point at the
// wrong node after an id attribute write. The caller holds idx.mu.
func invalidateIDLocked(idx *elementIndex, node *html.Node, elementID, value string) {
	idx.complete = false

	if node.Attribute("id") != elementID {
		delete(idx.byID, elementID)
	}

	if value != elementID {
		delete(idx.byID, value)
	}
}

// SetText replaces the element's text content and reports whether the
// element exists.
//
// The text becomes one text node and every child of the element is dropped,
// the way the DOM textContent setter works. An empty text leaves the element
// without children.
//
// The change lands on the parsed tree, so the next style cascade and the
// next layout see it.
func (d *Document) SetText(elementID, text string) bool {
	if d == nil || d.root == nil || elementID == "" {
		return false
	}

	// Resolve the dirty set before taking the index lock so the registry
	// lock never nests inside it; marking below only takes the set lock.
	dirty := dirtyFor(d)

	idx := elementIndexFor(d)
	idx.mu.Lock()
	defer idx.mu.Unlock()

	node := idx.findLocked(d.root, elementID)
	if node == nil {
		return false
	}

	structural := setTextLocked(idx, node, text)
	bumpMutationLocked(d.root, node, structural)

	// The write landed: the target and the chain up to the nearest block are
	// dirty, because new text can resize the line and shift siblings. Only a
	// changed child shape sets the structural flag.
	markTextChain(dirty, node, elementID)

	if structural {
		dirty.MarkStructural()
	}

	return true
}

// setTextLocked replaces node's text content and reports whether the child
// shape changed. The caller holds idx.mu and marks the dirty chain.
func setTextLocked(idx *elementIndex, node *html.Node, text string) bool {
	// The common patch shape is an element that already holds one text
	// node. Reuse that node instead of dropping and rebuilding it.
	if len(node.Children) == 1 && node.Children[0].Type == html.TextNode {
		return setSingleTextChild(node, text)
	}

	if len(node.Children) == 0 {
		if text == "" {
			return false
		}

		node.Children = textChild(node, text)
		node.ChildVersion++

		return true
	}

	dropChildrenLocked(idx, node)

	if text == "" {
		return true
	}

	node.Children = textChild(node, text)
	node.ChildVersion++

	return true
}

// setSingleTextChild rewrites the element's only text node, dropping it when
// text is empty. It reports whether the child shape changed.
func setSingleTextChild(node *html.Node, text string) bool {
	if text == "" {
		node.Children[0].Parent = nil
		node.Children = nil
		node.ChildVersion++

		return true
	}

	node.Children[0].Text = text

	return false
}

// textChild returns the single text node holding text under parent.
func textChild(parent *html.Node, text string) []*html.Node {
	return []*html.Node{{
		Type:   html.TextNode,
		Text:   text,
		Parent: parent,
	}}
}

// dropChildrenLocked detaches every child of node and forgets the ids that
// pointed inside the dropped subtrees. The caller holds idx.mu.
func dropChildrenLocked(idx *elementIndex, node *html.Node) {
	old := node.Children
	node.Children = nil
	node.ChildVersion++

	forgetSubtreesLocked(idx, old...)
}

// forgetSubtreesLocked detaches roots and forgets any cached ids that pointed
// inside the dropped subtrees. The caller holds idx.mu.
func forgetSubtreesLocked(idx *elementIndex, roots ...*html.Node) {
	idx.complete = false

	dropped := make(map[*html.Node]bool)

	var droppedIDs []string

	for _, root := range roots {
		root.Walk(func(inner *html.Node) {
			if inner.Type != html.ElementNode {
				return
			}

			dropped[inner] = true

			if id := inner.Attribute("id"); id != "" {
				droppedIDs = append(droppedIDs, id)
			}
		})

		root.Parent = nil
	}

	// Forget only the ids that pointed inside the dropped subtree. An id
	// whose cached node survived (a duplicate later in the document) stays
	// valid, and a dropped id re-resolves by walk on next lookup.
	for _, id := range droppedIDs {
		if dropped[idx.byID[id]] {
			delete(idx.byID, id)
		}
	}
}

// setNodeAttrLocked writes name=value to both node.Attrs and node.AttrList,
// preserving existing attribute spelling and foreign case sensitivity.
func setNodeAttrLocked(node *html.Node, name, value string) {
	if node.EnsureAttrs() == nil {
		node.Attrs = make(map[string]string, 1)
	}

	key := attributeKey(node.Attrs, name, node.Namespace != html.NamespaceHTML)
	node.Attrs[key] = value
	setAttrListValue(node, key, value)
}

// elementByID returns the element node with the given id, or nil.
func (d *Document) elementByID(elementID string) *html.Node {
	if d == nil || d.root == nil || elementID == "" {
		return nil
	}

	idx := elementIndexFor(d)

	idx.mu.RLock()

	node, ok := idx.byID[elementID]
	valid := ok && idx.root == d.root &&
		node.Attribute("id") == elementID &&
		attachedTo(node, d.root)
	definitiveMiss := !ok && idx.complete && idx.root == d.root

	idx.mu.RUnlock()

	if valid {
		return node
	}

	if definitiveMiss {
		return nil
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	return idx.findLocked(d.root, elementID)
}

// elementIndex is the per-document element-by-id lookup table. It mirrors
// what FindFirst would return: document order, first element wins on
// duplicate ids, template contents excluded. Every patch flows through
// SetAttribute or SetText, so those two keep the table in step and lookups
// only rebuild it when the hit fails validation.
type elementIndex struct {
	mu   sync.RWMutex
	root *html.Node
	byID map[string]*html.Node
	// complete reports that byID holds the first element of every id in the
	// tree as of the last build, so an absent id is a definitive miss. An id
	// rename or a dropped subtree clears it; the next miss rebuilds once.
	complete bool
}

// dirtyFor returns the dirty set for doc, creating it on first use. A nil
// document gets a detached set so marking helpers never need a nil branch.
func dirtyFor(doc *Document) *html.DirtySet {
	if doc == nil {
		return html.NewDirtySet()
	}

	doc.stateMu.RLock()
	set := doc.dirty
	doc.stateMu.RUnlock()

	if set != nil {
		return set
	}

	doc.stateMu.Lock()
	defer doc.stateMu.Unlock()

	if doc.dirty == nil {
		doc.dirty = html.NewDirtySet()
	}

	return doc.dirty
}

// DirtyIDs returns the dirty element ids in sorted order. The set grows on
// every successful SetAttribute or SetText and shrinks only on ClearDirty,
// so the ids stay available until the style and layout runs adopt them.
func (d *Document) DirtyIDs() []string {
	if d == nil {
		return nil
	}

	return dirtyFor(d).Snapshot()
}

// DirtyStructural reports whether a structural change (dropped or added
// children from SetText) was recorded since the last ClearDirty.
func (d *Document) DirtyStructural() bool {
	if d == nil {
		return false
	}

	return dirtyFor(d).Structural()
}

// ClearDirty drops every dirty id and the structural flag after the style and
// layout runs adopt them. The next edit starts a new run.
func (d *Document) ClearDirty() {
	if d == nil {
		return
	}

	d.stateMu.RLock()
	set := d.dirty
	d.stateMu.RUnlock()

	if set == nil {
		return
	}

	set.Clear()
}

// tagDiv is the block-level div tag name.
const tagDiv = "div"

// isBlockElement reports whether name is a default block-level HTML element.
// The cascade can promote or demote any element through display, and that
// resolution belongs to the style run. This is only the structural heuristic
// for parent-chain marking: a doubtful tag counts as inline so the walk marks
// more ancestors, never fewer.
func isBlockElement(name string) bool {
	switch name {
	case "address", "article", "aside", "blockquote", "body", "dd", "details",
		"dialog", tagDiv, "dl", "dt", "fieldset", "figcaption", "figure",
		"footer", "form", "h1", "h2", "h3", "h4", "h5", "h6", "header",
		"hgroup", "hr", "html", "li", "main", "nav", "ol", "p", "pre",
		"section", "table", "ul":
		return true
	default:
		return false
	}
}

// markTextChain marks the edited element and each id-bearing ancestor up to
// and including the nearest block element. New text can resize the line and
// shift siblings, so the first block boundary contains the damage; anything
// wider arrives through the structural flag and the size-change fallback.
func markTextChain(set *html.DirtySet, node *html.Node, elementID string) {
	if set == nil || node == nil {
		return
	}

	set.Mark(elementID)

	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if parent.Type != html.ElementNode {
			continue
		}

		if id := parent.Attribute("id"); id != "" {
			set.Mark(id)
		}

		if isBlockElement(parent.Name) {
			return
		}
	}
}

// elementIndexFor returns the lookup table for doc, building it on first use.
func elementIndexFor(doc *Document) *elementIndex {
	doc.stateMu.RLock()
	idx := doc.index
	doc.stateMu.RUnlock()

	if idx != nil {
		return idx
	}

	doc.stateMu.Lock()
	defer doc.stateMu.Unlock()

	if doc.index != nil {
		return doc.index
	}

	idx = new(elementIndex)
	idx.byID = map[string]*html.Node{}
	idx.build(doc.root)
	doc.index = idx

	return idx
}

// build fills the table from root in document order, keeping the first
// element for each id so duplicates resolve the way FindFirst does.
func (idx *elementIndex) build(root *html.Node) {
	byID := make(map[string]*html.Node)

	root.Walk(func(node *html.Node) {
		if node.Type != html.ElementNode {
			return
		}

		nodeID := node.Attribute("id")
		if nodeID == "" {
			return
		}

		if _, ok := byID[nodeID]; !ok {
			byID[nodeID] = node
		}
	})

	idx.root = root
	idx.byID = byID
	idx.complete = true
}

// findLocked resolves elementID through the table. An incomplete table or a
// hit that fails validation rebuilds the table first (one walk), after which
// the table is authoritative. The caller holds idx.mu for writing.
func (idx *elementIndex) findLocked(root *html.Node, elementID string) *html.Node {
	if idx.root != root || !idx.complete {
		idx.build(root)
	} else if node, ok := idx.byID[elementID]; ok &&
		(node.Attribute("id") != elementID || !attachedTo(node, root)) {
		idx.build(root)
	}

	// The table is complete here, so it answers authoritatively: a miss needs
	// no further walk.
	if node, ok := idx.byID[elementID]; ok &&
		node.Attribute("id") == elementID && attachedTo(node, root) {
		return node
	}

	return nil
}

// attachedTo reports whether node sits under root.
func attachedTo(node, root *html.Node) bool {
	for p := node; p != nil; p = p.Parent {
		if p == root {
			return true
		}
	}

	return false
}

// attributeKey returns the map key for name: the key the element already
// stores for that attribute, or the lowercased name for a new attribute. A
// foreign (SVG or MathML) element keeps the caller's spelling for a new
// attribute, because names such as viewBox are case-sensitive there.
func attributeKey(attrs map[string]string, name string, foreign bool) string {
	for key := range attrs {
		if strings.EqualFold(key, name) {
			return key
		}
	}

	if foreign {
		return name
	}

	return strings.ToLower(name)
}

// setAttrListValue records the value in the ordered attribute list, so the
// list and the map carry the same value.
func setAttrListValue(node *html.Node, key, value string) {
	for i := range node.AttrList {
		if strings.EqualFold(node.AttrList[i].Name, key) {
			node.AttrList[i].Value = value

			return
		}
	}

	node.AttrList = append(node.AttrList, html.Attr{
		Namespace: "",
		Name:      key,
		Value:     value,
	})
}
