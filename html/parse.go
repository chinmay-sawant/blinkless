package html

import (
	"fmt"

	"github.com/chinmay-sawant/blinkless/internal/html"
	"github.com/chinmay-sawant/blinkless/internal/pubstate"
)

// The initializer runs before any caller can call Parse. Parse must not write
// package state, because layout can read pubstate.Root from another goroutine.
var _ = pubstate.RegisterRoot(func(doc any) *html.Node {
	parsed, ok := doc.(*Document)
	if !ok || parsed == nil {
		return nil
	}

	return parsed.root
})

// Document is one parsed HTML tree.
// The CSS package reads it. Callers do not see the engine node.
type Document struct {
	root *html.Node
}

// Element is one element found by id.
type Element struct {
	Tag  string
	ID   string
	Text string
}

// Parse parses UTF-8 HTML into a document the CSS package can style.
// A nil source is an empty document.
func Parse(source []byte) (*Document, error) {
	root, err := html.ParseDocument(source)
	if err != nil {
		return nil, fmt.Errorf("html: parse: %w", err)
	}

	if root == nil {
		return nil, errEmpty
	}

	return &Document{root: root}, nil
}

// Find returns the element with the given id.
// The text is the element's descendant text.
//
// The lookup goes through the per-document id index under its lock, so a
// Find running beside SetAttribute or SetText sees one state of the tree.
// A cached hit is validated (same root, id still matches, still attached)
// while holding the read lock; a miss or stale hit falls back to one
// authoritative walk under the write lock, which also heals the table.
// The walk stays in document order with template contents excluded, so the
// result matches a direct tree search.
func (d *Document) Find(elementID string) (Element, bool) {
	if d == nil || d.root == nil || elementID == "" {
		return Element{}, false //nolint:exhaustruct // no element was found
	}

	idx := elementIndexFor(d)

	if element, found := idx.cachedFind(d.root, elementID); found {
		return element, true
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	node := idx.findLocked(d.root, elementID)
	if node == nil {
		return Element{}, false //nolint:exhaustruct // no element was found
	}

	return Element{
		Tag:  node.Name,
		ID:   elementID,
		Text: node.TextContent(),
	}, true
}

// cachedFind returns the validated cached element for elementID while holding
// the read lock. A miss, a stale entry, or a detached node falls through to
// the authoritative walk in Find.
func (idx *elementIndex) cachedFind(root *html.Node, elementID string) (Element, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	node, found := idx.byID[elementID]
	if !found || node == nil || idx.root != root ||
		node.Attribute("id") != elementID || !attachedTo(node, root) {
		return Element{}, false //nolint:exhaustruct // no element was found
	}

	return Element{
		Tag:  node.Name,
		ID:   elementID,
		Text: node.TextContent(),
	}, true
}
