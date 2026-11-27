package layout

import "github.com/chinmay-sawant/blinkless/internal/html"

// The state carried by Result lets a later layout run re-lay out one subtree
// of a document that was already laid out: the resolved styles say how each
// node was styled, and the box tree says where each element ended up.

// boxForID returns the laid-out box of the element with the given id, or nil
// when no element carries that id. The walk follows the box tree this layout
// run produced, so the result is the box that was placed on the canvas, not a
// fresh build. Table cell boxes are searched too: a cell is held in the
// owning table's row list, not in the children chain.
func (r *Result) boxForID(elementID string) *box {
	if r == nil || elementID == "" {
		return nil
	}

	r.ensureBoxIndex()

	return r.idBoxes[elementID]
}

// invalidateBoxIndex clears the lazily built id and parent maps after an
// in-place box-tree mutation so the next lookup reflects the updated tree.
func (r *Result) invalidateBoxIndex() {
	if r == nil {
		return
	}

	r.idBoxes = nil
	r.boxParents = nil
	r.boxOrder = nil
}

// ensureBoxIndex walks the box tree once to populate idBoxes (document-order
// first match per id, including table cells), boxParents (children-chain
// parent links matching findBoxAncestors), and boxOrder (children-chain
// document order index).
func (r *Result) ensureBoxIndex() {
	if r == nil || r.idBoxes != nil {
		return
	}

	root := r.boxTreeRoot()
	if root == nil {
		return
	}

	capHint := max(len(r.boxes), 32)
	idBoxes := make(map[string]*box, 32)
	boxParents := make(map[*box]*box, capHint)
	boxOrder := make(map[*box]int, capHint)
	order := 0

	var walk func(cur *box, inChildrenChain bool)
	walk = func(cur *box, inChildrenChain bool) {
		if cur == nil {
			return
		}

		if inChildrenChain {
			if _, exists := boxOrder[cur]; !exists {
				boxOrder[cur] = order
				order++
			}
		}

		if cur.node != nil && cur.node.Type == html.ElementNode {
			if id := cur.node.Attribute("id"); id != "" {
				if _, exists := idBoxes[id]; !exists {
					idBoxes[id] = cur
				}
			}
		}

		for _, child := range cur.children {
			if child == nil {
				continue
			}

			if inChildrenChain {
				if _, exists := boxParents[child]; !exists {
					boxParents[child] = cur
				}
			}

			walk(child, inChildrenChain)
		}

		for _, row := range cur.rows {
			for _, cell := range row {
				walk(cell, false)
			}
		}
	}

	walk(root, true)

	r.idBoxes = idBoxes
	r.boxParents = boxParents
	r.boxOrder = boxOrder
}

// boxAncestorsInto appends the ancestor chain from r.boxTreeRoot() down to
// (but not including) target into path using the cached parent map.
func (r *Result) boxAncestorsInto(target *box, path *[]*box) bool {
	if r == nil || target == nil {
		return false
	}

	root := r.boxTreeRoot()
	if root == nil {
		return false
	}

	*path = (*path)[:0]
	if target == root {
		return true
	}

	r.ensureBoxIndex()

	cur := target
	for cur != root {
		parent, ok := r.boxParents[cur]
		if !ok {
			*path = (*path)[:0]

			return false
		}

		*path = append(*path, parent)
		cur = parent
	}

	// Reverse in place so the slice runs root -> ... -> parent(target).
	p := *path
	for left, right := 0, len(p)-1; left < right; left, right = left+1, right-1 {
		p[left], p[right] = p[right], p[left]
	}

	return true
}

// nodeForID returns the html node of the element with the given id, or nil
// when no element carries that id. The box tree is searched first, so the node
// is the one that was laid out. An element that produced no box (display:none)
// is still found in the document tree behind the root box.
func (r *Result) nodeForID(elementID string) *html.Node {
	if r == nil || elementID == "" {
		return nil
	}

	if found := r.boxForID(elementID); found != nil && found.node != nil {
		return found.node
	}

	return findNodeByID(r.documentRootNode(), elementID)
}

// boxTreeRoot returns the root of the box tree. rootBox is the state field
// kept for the subtree lookups; root is the older alias for the same box.
func (r *Result) boxTreeRoot() *box {
	if r == nil {
		return nil
	}

	if r.rootBox != nil {
		return r.rootBox
	}

	return r.root
}

// documentRootNode returns the html node the layout run started from.
func (r *Result) documentRootNode() *html.Node {
	root := r.boxTreeRoot()
	if root == nil {
		return nil
	}

	return root.node
}

// findNodeByID walks the document tree in document order.
func findNodeByID(root *html.Node, elementID string) *html.Node {
	if root == nil {
		return nil
	}

	var found *html.Node

	root.Walk(func(node *html.Node) {
		if found != nil {
			return
		}

		if node.Type == html.ElementNode && node.Attribute("id") == elementID {
			found = node
		}
	})

	return found
}
