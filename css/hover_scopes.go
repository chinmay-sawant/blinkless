package css

import (
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	icss "github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

const (
	maxHoverListEntries  = 4
	maxHoverMemoPerEntry = 1024
	// hoverMemoInitialCap sizes the memo and intern maps when a cap trips and
	// they are rebuilt empty.
	hoverMemoInitialCap = 64
	// hoverAttrEntryOverhead covers the two length prefixes and the three
	// separators hoverAttrKey writes for one attribute.
	hoverAttrEntryOverhead = 8
	// maxHoverMemoBytes caps retained memo key strings at ~512KB. Keys carry
	// full attribute strings (style attrs run to kilobytes), so the 1024-entry
	// count cap alone retains ~1MB worst case. The byte cap trips first for
	// large keys while small keys still fill the count cap.
	maxHoverMemoBytes = 512 * 1024
)

//nolint:gochecknoglobals // bounded hover caches: sheets and signatures only, never documents.
var (
	hoverListMu     sync.Mutex
	hoverLists      []*hoverListEntry
	hoverMemoHits   atomic.Uint64
	hoverMemoMisses atomic.Uint64
	hoverListHits   atomic.Uint64
)

// hoverListEntry is one cached hover selector set. sheets is a copy of the
// document sheet pointers so pointer identity stays valid while cached.
// selectors shares its selector values read-only across calls and goroutines.
// local precomputes SelectorIsElementLocal per selector. memo maps an
// element-local outcome key to its match result. memoBytes tracks the
// approximate retained key bytes (sum of len(tag)+len(attrs) at insert,
// ignoring map overhead and sharing), guarded by mu alongside memo, and
// trips a clear at maxHoverMemoBytes alongside the count cap. intern pools
// canonical tag and attrs strings so repeated evaluations of the same node
// share one backing string instead of duplicating it per hover id; it is
// guarded by mu alongside memo and is cleared together with memo on either
// cap tripping.
type hoverListEntry struct {
	sheets    []*icss.Stylesheet
	selectors []icss.Selector
	local     []bool
	mu        sync.RWMutex
	memo      map[hoverMemoKey]bool
	memoBytes int
	intern    map[string]string
}

// hoverMemoKey identifies one element-local match outcome. sel is the index
// in the cached selector list, tag and attrs are the element signature,
// pseudo is the ::before/::after pass, and focus/hover/active is the match
// state for that single evaluation.
type hoverMemoKey struct {
	sel    int
	tag    string
	attrs  string
	pseudo string
	focus  string
	hover  string
	active string
}

// HoverChangedScopes returns the IDs of subtrees containing elements whose
// hover-dependent selector matches changed from previous to d's hover state.
// It includes inherited changes by keeping each matched element's subtree.
// Rules inside conditional groups are included conservatively.
//
// A changed element without an ID uses its nearest ID-bearing ancestor. False
// means no such scope exists, or d is nil; the caller must lay out the page.
// An empty result with true means no selector match changed.
func (d *Document) HoverChangedScopes(previous string) ([]string, bool) {
	if d == nil || d.root == nil {
		return nil, false
	}

	if previous == d.state.Hover {
		return nil, true
	}

	old := d.state
	old.Hover = previous

	entry := cachedHoverList(d.sheets)
	if len(entry.selectors) == 0 {
		return nil, true
	}

	scopes := make(map[*html.Node]bool)

	if !walkHoverChanged(d.root, entry, old, d.state, scopes) {
		return nil, false
	}

	return outerHoverScopes(d.root, scopes), true
}

// walkHoverChanged records every hover-changed element's nearest id-bearing
// ancestor in scopes. False means one changed element had no such ancestor, so
// the caller must lay out the page.
func walkHoverChanged(
	node *html.Node, entry *hoverListEntry, old, next icss.MatchState, scopes map[*html.Node]bool,
) bool {
	if node.Type == html.ElementNode && hoverMatchChangedCached(node, entry, old, next) {
		scope := hoverIDScope(node)
		if scope == nil {
			return false
		}

		scopes[scope] = true
	}

	for _, child := range node.Children {
		if !walkHoverChanged(child, entry, old, next, scopes) {
			return false
		}
	}

	return true
}

// cachedHoverList returns the shared hover selector entry for sheets,
// building it once per distinct sheet set. The entry is read-only except for
// its memo map, which has its own lock.
func cachedHoverList(sheets []*icss.Stylesheet) *hoverListEntry {
	hoverListMu.Lock()
	defer hoverListMu.Unlock()

	for _, entry := range hoverLists {
		if sameSheets(entry.sheets, sheets) {
			hoverListHits.Add(1)

			return entry
		}
	}

	selectors := hoverSelectors(sheets)
	local := make([]bool, len(selectors))

	for i, sel := range selectors {
		local[i] = icss.SelectorIsElementLocal(sel)
	}

	entry := &hoverListEntry{
		sheets:    append([]*icss.Stylesheet(nil), sheets...),
		selectors: selectors,
		local:     local,
		mu:        sync.RWMutex{},
		memo:      make(map[hoverMemoKey]bool, hoverMemoInitialCap),
		memoBytes: 0,
		intern:    make(map[string]string, hoverMemoInitialCap),
	}

	hoverLists = append(hoverLists, entry)
	if len(hoverLists) > maxHoverListEntries {
		copy(hoverLists, hoverLists[1:])
		hoverLists[len(hoverLists)-1] = nil
		hoverLists = hoverLists[:len(hoverLists)-1]
	}

	return entry
}

// hoverMatchChangedCached reports whether any cached hover selector matches
// node differently under old and next. Element-local selectors resolve through
// the per-entry memo; every other selector evaluates fresh.
func hoverMatchChangedCached(node *html.Node, entry *hoverListEntry, old, next icss.MatchState) bool {
	var tag, attrs string

	sigReady := false

	isHoverEndpoint := hoverIsEndpoint(node, old, next)

	for idx, sel := range entry.selectors {
		pseudo := hoverPseudoOf(sel)

		if entry.local[idx] {
			if !isHoverEndpoint {
				continue
			}

			if !sigReady {
				tag = hoverTagName(node)
				attrs = hoverAttrKey(node.Attrs)
				sigReady = true
			}

			if hoverMemoMatch(entry, idx, node, tag, attrs, old, pseudo) !=
				hoverMemoMatch(entry, idx, node, tag, attrs, next, pseudo) {
				return true
			}

			continue
		}

		if hoverSelectorMatchChanged(sel, node, old, next, pseudo) {
			return true
		}
	}

	return false
}

// hoverIsEndpoint reports whether node is the element whose hover state
// changed between old and next.
func hoverIsEndpoint(node *html.Node, old, next icss.MatchState) bool {
	if node.Attrs == nil {
		return false
	}

	elementID := node.Attrs["id"]

	return elementID != "" && (elementID == old.Hover || elementID == next.Hover)
}

// hoverPseudoOf returns the selector's final pseudo-element, empty when it
// targets no pseudo-element.
func hoverPseudoOf(sel icss.Selector) string {
	if len(sel.Parts) == 0 {
		return ""
	}

	return sel.Parts[len(sel.Parts)-1].PseudoElement
}

// hoverSelectorMatchChanged reports whether one selector matches node
// differently under old and next.
func hoverSelectorMatchChanged(
	sel icss.Selector, node *html.Node, old, next icss.MatchState, pseudo string,
) bool {
	if pseudo != "" {
		return old.MatchesPseudo(sel, node, pseudo) != next.MatchesPseudo(sel, node, pseudo)
	}

	return old.Matches(sel, node) != next.Matches(sel, node)
}

// hoverMemoMatch resolves one element-local selector outcome through the
// entry memo, evaluating fresh on a miss. The selector values are never
// mutated, so sharing them across goroutines is safe.
func hoverMemoMatch(
	entry *hoverListEntry, idx int, node *html.Node, tag, attrs string, state icss.MatchState, pseudo string,
) bool {
	key := hoverMemoKey{
		sel:    idx,
		tag:    tag,
		attrs:  attrs,
		pseudo: pseudo,
		focus:  state.Focus,
		hover:  state.Hover,
		active: state.Active,
	}

	entry.mu.RLock()
	got, ok := entry.memo[key]
	entry.mu.RUnlock()

	if ok {
		hoverMemoHits.Add(1)

		return got
	}

	sel := entry.selectors[idx]

	var fresh bool

	if pseudo != "" {
		fresh = state.MatchesPseudo(sel, node, pseudo)
	} else {
		fresh = state.Matches(sel, node)
	}

	hoverMemoMisses.Add(1)

	entry.mu.Lock()
	if len(entry.memo) >= maxHoverMemoPerEntry || entry.memoBytes >= maxHoverMemoBytes {
		entry.memo = make(map[hoverMemoKey]bool, hoverMemoInitialCap)
		entry.intern = make(map[string]string, hoverMemoInitialCap)
		entry.memoBytes = 0
	}

	key.sel = idx
	key.tag = internHoverString(entry, tag)
	key.attrs = internHoverString(entry, attrs)

	if _, dup := entry.memo[key]; !dup {
		entry.memoBytes += len(key.tag) + len(key.attrs)
	}

	entry.memo[key] = fresh
	entry.mu.Unlock()

	return fresh
}

// internHoverString returns the canonical pooled copy of value, storing value
// on first sight. The caller must hold entry.mu for writing. Empty strings
// have no backing to share and bypass the pool. Canonicalization preserves
// exact string equality: equal contents map to one shared backing, and unequal
// contents never compare equal.
func internHoverString(entry *hoverListEntry, value string) string {
	if value == "" {
		return value
	}

	if canonical, ok := entry.intern[value]; ok {
		return canonical
	}

	if entry.intern == nil {
		entry.intern = make(map[string]string, hoverMemoInitialCap)
	}

	entry.intern[value] = value

	return value
}

// hoverTagName builds the namespace-aware tag half of an element signature.
// HTML names match ASCII case-insensitively, so they are lowered; foreign
// names keep their case.
func hoverTagName(node *html.Node) string {
	if node.Namespace != html.NamespaceHTML {
		return "f\x00" + strconv.Itoa(int(node.Namespace)) + "\x00" + node.Name
	}

	return "h\x00" + strings.ToLower(node.Name)
}

// hoverAttrKey encodes the full attribute map with length-prefixed sorted
// names so distinct maps never share a key.
func hoverAttrKey(attrs map[string]string) string {
	if len(attrs) == 0 {
		return ""
	}

	names := make([]string, 0, len(attrs))
	for name := range attrs {
		names = append(names, name)
	}

	slices.Sort(names)

	total := 0
	for _, name := range names {
		total += len(name) + len(attrs[name]) + hoverAttrEntryOverhead
	}

	var out strings.Builder

	out.Grow(total)

	for _, name := range names {
		value := attrs[name]
		out.WriteString(strconv.Itoa(len(name)))
		out.WriteByte(':')
		out.WriteString(name)
		out.WriteByte('=')
		out.WriteString(strconv.Itoa(len(value)))
		out.WriteByte(':')
		out.WriteString(value)
		out.WriteByte(';')
	}

	return out.String()
}

// resetHoverCache clears the shared hover caches and stats. Tests use it to
// isolate hit-rate assertions; it is not part of the public API.
func resetHoverCache() {
	hoverListMu.Lock()
	hoverLists = nil
	hoverListMu.Unlock()
	hoverMemoHits.Store(0)
	hoverMemoMisses.Store(0)
	hoverListHits.Store(0)
}

func hoverIDScope(node *html.Node) *html.Node {
	for node != nil && node.Attribute("id") == "" {
		node = node.Parent
	}

	return node
}

func hoverSelectors(sheets []*icss.Stylesheet) []icss.Selector {
	var selectors []icss.Selector

	for _, sheet := range sheets {
		if sheet == nil {
			continue
		}

		for _, rule := range sheet.Rules {
			for _, selector := range rule.Selectors {
				if icss.SelectorUsesHover(selector) {
					selectors = append(selectors, selector)
				}
			}
		}
	}

	return selectors
}

func hoverMatchChanged(node *html.Node, selectors []icss.Selector, old, next icss.MatchState) bool {
	for _, selector := range selectors {
		pseudo := selector.Parts[len(selector.Parts)-1].PseudoElement
		if pseudo != "" {
			if old.MatchesPseudo(selector, node, pseudo) != next.MatchesPseudo(selector, node, pseudo) {
				return true
			}
		} else if old.Matches(selector, node) != next.Matches(selector, node) {
			return true
		}
	}

	return false
}

// Emit tree order and drop scopes already covered by an ancestor's patch.
func outerHoverScopes(root *html.Node, scopes map[*html.Node]bool) []string {
	var ids []string

	var walk func(*html.Node)

	walk = func(node *html.Node) {
		if scopes[node] {
			ids = append(ids, node.Attribute("id"))

			return
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(root)

	return ids
}
