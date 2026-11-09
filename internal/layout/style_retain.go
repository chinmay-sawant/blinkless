package layout

import (
	"strings"
	"sync"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// styleDirtySet is the read half of the dirty-set worker's contract: a
// thread-safe set with Mark(id), MarkSubtree(id), Clear(), and
// Snapshot() []string. The style pass only calls Snapshot; subtree expansion
// (a dirty root dirties every descendant) happens by tree walk, so Mark and
// MarkSubtree are equivalent inputs here. html/dirty.go owns the
// implementation; this interface keeps the style pass compiling until it
// lands and states exactly what the pass consumes.
type styleDirtySet interface {
	Snapshot() []string
}

// retainedStyleEntry is one cached element style: the interned resolved
// style plus the ancestor-chain fingerprints that prove reuse is still
// valid. chain is the slash-joined ancestor tokens from the root, so any
// reparenting, wrap, unwrap, or ancestor rename invalidates the entry.
// parentFP is the intern fingerprint of the parent style the entry resolved
// against, so any inherited-value change (color, font, custom properties,
// rem base) forces the child to re-resolve even when the child itself is
// clean.
type retainedStyleEntry struct {
	style    *ResolvedStyle
	chain    string
	parentFP uint64
}

// styleRetainInputs captures every resolveStyles input that can change an
// outcome without touching the tree: the sheet set identity, the media and
// viewport, the stateful pseudo-class ids, and the operator underline
// policy. Zoom is layout scale only and never feeds the cascade. Any input
// change falls back to the full resolve path.
type styleRetainInputs struct {
	sheetsKey          string
	media              string
	viewportW          float64
	viewportH          float64
	focus              string
	hover              string
	active             string
	printLinkUnderline bool
}

// styleRetainer is the per-document retained style map: element id
// (the id attribute) to resolved style plus ancestor-chain fingerprints.
// Entries are immutable after restock; the mutex guards the map and the
// counters only, so concurrent layouts share the retainer race-clean.
// Stored styles point at the interned records of the pass that stocked
// them, which the GC keeps alive exactly like any other shared style.
type styleRetainer struct {
	mu       sync.RWMutex
	entries  map[string]retainedStyleEntry
	inputs   styleRetainInputs
	elements int
	full     uint64
	retained uint64
	reused   uint64
}

// newStyleRetainer returns an empty retainer. The first resolve with it
// always takes the full path and stocks the map.
func newStyleRetainer() *styleRetainer {
	return new(styleRetainer)
}

// retainInputsOf folds the style inputs of one resolve into a comparable key.
// The sheet key is pointer identity plus rule count (sheetsCacheKey), so a
// fresh parse misses and re-resolves while an in-place rule mutation keeps
// the key: the existing contract (fresh parse required, no in-place edits)
// is preserved, not weakened.
func retainInputsOf(opts Options) styleRetainInputs {
	return styleRetainInputs{
		sheetsKey:          sheetsCacheKey(opts.Sheets),
		media:              opts.Media,
		viewportW:          opts.Width,
		viewportH:          opts.Height,
		focus:              opts.State.Focus,
		hover:              opts.State.Hover,
		active:             opts.State.Active,
		printLinkUnderline: opts.PrintLinkUnderline,
	}
}

// retainStyleFP fingerprints one resolved style for the parent check. A nil
// parent (the document root's child) fingerprints as zero, which is also
// what restock stores for it.
func retainStyleFP(style *ResolvedStyle) uint64 {
	if style == nil {
		return 0
	}

	return styleInternFingerprint(style)
}

// retainToken names one ancestor for the chain: tag plus id attribute. The
// id alone would miss a wrap or rename above the element; the tag alone
// would miss a move between same-named ancestors.
func retainToken(node *html.Node) string {
	if node.Type != html.ElementNode {
		return "#"
	}

	return node.Name + "#" + node.Attribute("id")
}

// retainChain returns the slash-joined ancestor tokens from the root down to
// the parent. Siblings share a prefix; any structural edit above the
// element changes its chain and falls back to full resolve.
func retainChain(node *html.Node) string {
	var toks []string

	for parent := node.Parent; parent != nil; parent = parent.Parent {
		toks = append(toks, retainToken(parent))
	}

	for i, j := 0, len(toks)-1; i < j; i, j = i+1, j-1 {
		toks[i], toks[j] = toks[j], toks[i]
	}

	return strings.Join(toks, "/")
}

// retainIndexIDs indexes the document's id-bearing elements and counts all
// elements. dup reports a repeated id attribute: ambiguous keys can never
// reuse, so the caller falls back to full resolve while any duplicate
// exists.
func retainIndexIDs(root *html.Node) (map[string]*html.Node, int, bool) {
	byID := make(map[string]*html.Node)
	total := 0
	dup := false

	var walk func(node *html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			total++

			if id := node.Attribute("id"); id != "" {
				if _, taken := byID[id]; taken {
					dup = true
				} else {
					byID[id] = node
				}
			}
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	if root != nil {
		walk(root)
	}

	return byID, total, dup
}

// lookup returns the retained entry for id. Callers recheck chain and
// parentFP against the live tree; the map alone is never trusted.
func (r *styleRetainer) lookup(id string) (retainedStyleEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.entries[id]

	return entry, ok
}

// check reports whether the incremental path may run: same inputs, same
// element count, same id set, every dirty id known on both sides, and every
// id-bearing node's ancestor chain unchanged. Any doubt returns false and
// the caller takes the existing full resolve path unchanged.
func (r *styleRetainer) check(
	_ *html.Node,
	byID map[string]*html.Node,
	total int,
	inputs styleRetainInputs,
	dirty map[string]struct{},
) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.matchesShape(byID, total, inputs) {
		return false
	}

	for id := range dirty {
		if _, ok := byID[id]; !ok {
			return false
		}

		if _, ok := r.entries[id]; !ok {
			return false
		}
	}

	for id, node := range byID {
		entry, ok := r.entries[id]
		if !ok {
			return false
		}

		if entry.chain != retainChain(node) {
			return false
		}
	}

	return true
}

// matchesShape reports whether the retained map has the same size, element
// count, and inputs as the current pass.
func (r *styleRetainer) matchesShape(byID map[string]*html.Node, total int, inputs styleRetainInputs) bool {
	if len(r.entries) == 0 || r.elements != total || len(r.entries) != len(byID) {
		return false
	}

	return r.inputs == inputs
}

// restock rebuilds the map from a finished resolve. It runs after both the
// full and the incremental paths, so the next layout always reads the
// latest styles, chains, and inputs.
func (r *styleRetainer) restock(
	root *html.Node, styles map[*html.Node]*ResolvedStyle, inputs styleRetainInputs, total int,
) {
	fresh := make(map[string]retainedStyleEntry)

	var walk func(node *html.Node, parent *ResolvedStyle)
	walk = func(node *html.Node, parent *ResolvedStyle) {
		var style *ResolvedStyle

		switch node.Type {
		case html.ElementNode:
			style = stockRetainedEntry(fresh, node, styles[node], parent)
		case html.TextNode:
			style = parent
			if style == nil {
				style = &defaultTextStyle
			}
		case html.CommentNode, html.DoctypeNode, html.NodeUnknown:
			style = &zeroResolvedStyle
		}

		for _, child := range node.Children {
			if child.Type == html.TextNode {
				continue
			}

			walk(child, style)
		}
	}

	if root != nil {
		walk(root, nil)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries = fresh
	r.inputs = inputs
	r.elements = total
}

// stockRetainedEntry records one id-bearing element style and returns the
// style its children inherit.
func stockRetainedEntry(
	fresh map[string]retainedStyleEntry, node *html.Node, style, parent *ResolvedStyle,
) *ResolvedStyle {
	if style == nil {
		return nil
	}

	styleID := node.Attribute("id")
	if styleID == "" {
		return style
	}

	if _, taken := fresh[styleID]; taken {
		return style
	}

	fresh[styleID] = retainedStyleEntry{
		style:    style,
		chain:    retainChain(node),
		parentFP: retainStyleFP(parent),
	}

	return style
}

// noteFullResolve records one fallback to the existing full path.
func (r *styleRetainer) noteFullResolve() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.full++
}

// noteRetainedResolve records one incremental resolve and how many clean
// id-bearing subtrees it reused verbatim.
func (r *styleRetainer) noteRetainedResolve(reused uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.retained++
	r.reused += reused
}

// fullCount reports how many resolves fell back to the full path.
func (r *styleRetainer) fullCount() uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.full
}

// retainedCount reports how many resolves took the incremental path.
func (r *styleRetainer) retainedCount() uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.retained
}

// reusedCount reports how many clean id-bearing subtrees were reused
// verbatim across all incremental resolves.
func (r *styleRetainer) reusedCount() uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.reused
}
