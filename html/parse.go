package html

import (
	"fmt"
	"sync"

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
	root    *html.Node
	stateMu sync.RWMutex
	index   *elementIndex
	dirty   *html.DirtySet
}

// Element is one element found by id.
type Element struct {
	Tag  string
	ID   string
	Text string
}

// MaxParseBytes is the default source size Parse accepts: 64 MiB. A larger
// source returns an error; use ParseWithLimit to choose another bound.
const MaxParseBytes = 64 << 20

// Parse parses UTF-8 HTML into a document the CSS package can style.
// A nil source is an empty document. Sources over MaxParseBytes are rejected;
// ParseWithLimit takes an explicit bound.
func Parse(source []byte) (*Document, error) {
	return ParseWithLimit(source, MaxParseBytes)
}

// ParseWithLimit is Parse with an explicit bound on the source size in bytes
// (after a leading byte order mark is dropped). A maxBytes of zero or less
// means MaxParseBytes.
func ParseWithLimit(source []byte, maxBytes int) (*Document, error) {
	root, err := html.ParseDocumentWithLimit(source, maxBytes)
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
		return Element{}, false
	}

	idx := elementIndexFor(d)

	if element, found, definitive := idx.cachedFind(d.root, elementID); found {
		return element, true
	} else if definitive {
		return Element{}, false
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	node := idx.findLocked(d.root, elementID)
	if node == nil {
		return Element{}, false
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
//
// definitive is true when the id is absent from a complete, current table: a
// miss then needs neither the write lock nor a DOM walk.
func (idx *elementIndex) cachedFind(root *html.Node, elementID string) (element Element, found, definitive bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	node, ok := idx.byID[elementID]
	if !ok && idx.complete && idx.root == root {
		return Element{}, false, true
	}

	if !ok || node == nil || idx.root != root ||
		node.Attribute("id") != elementID || !attachedTo(node, root) {
		return Element{}, false, false
	}

	return Element{
		Tag:  node.Name,
		ID:   elementID,
		Text: node.TextContent(),
	}, true, false
}
