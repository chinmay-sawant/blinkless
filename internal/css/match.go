package css

import (
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
	"unicode/utf8"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// siblingInfo holds the sibling metrics tree-structural pseudo-classes need:
// the neighbouring element siblings and the 1-based position of a node among
// all element siblings and among same-name siblings.
type siblingInfo struct {
	prev          *html.Node
	next          *html.Node
	index         int
	typeIndex     int
	typeLastIndex int
}

// asciiEqualFold reports whether a and b match ASCII case-insensitively
// without allocating. HTML type selectors and the Selectors 4 ASCII i flag
// fold ASCII letters only; non-ASCII bytes compare exactly. A rare non-ASCII
// mismatch falls back to EqualFold so Unicode tags still behave.
func asciiEqualFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		ca := a[i]
		cb := b[i]

		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}

		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}

		if ca != cb {
			if ca >= 0x80 || cb >= 0x80 {
				return strings.EqualFold(a, b)
			}

			return false
		}
	}

	return true
}

// asciiLower returns the ASCII-lowered copy of s, or s itself when already
// lower (no allocation on the fast path). Only ASCII A-Z folds; this matches
// the Selectors 4 ASCII i flag and HTML type-selector rules.
func asciiLower(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			b := make([]byte, len(s))
			copy(b, s)

			for j := i; j < len(s); j++ {
				if b[j] >= 'A' && b[j] <= 'Z' {
					b[j] += 'a' - 'A'
				}
			}

			return string(b)
		}
	}

	return s
}

// sibParentData is one memoized parent: the element children in order plus
// the sibling metrics for every child. Built once per parent, reused for all
// candidates. Immutable after publication, so readers use it lock-free.
type sibParentData struct {
	elems    []*html.Node
	infos    map[*html.Node]siblingInfo
	childLen int
}

var (
	sibMu      sync.RWMutex
	sibParents map[*html.Node]*sibParentData
	sibHits    int64
	sibMisses  int64
)

// siblingTypeKey groups element siblings for of-type indices. HTML names fold
// ASCII case-insensitively; foreign names stay exact.
func siblingTypeKey(n *html.Node) string {
	if n.Namespace == html.NamespaceHTML {
		return "h\x00" + asciiLower(n.Name)
	}

	return "f\x00" + strconv.Itoa(int(n.Namespace)) + "\x00" + n.Name
}

// buildSibEntry scans parent once to collect element children and their
// 1-based indices plus of-type indices.
func buildSibEntry(parent *html.Node) *sibParentData {
	var elems []*html.Node

	for _, child := range parent.Children {
		if child.Type == html.ElementNode {
			elems = append(elems, child)
		}
	}

	infos := make(map[*html.Node]siblingInfo, len(elems))
	totals := make(map[string]int, len(elems))
	keys := make([]string, len(elems))

	for i, elem := range elems {
		k := siblingTypeKey(elem)
		keys[i] = k
		totals[k]++
	}

	seen := make(map[string]int, len(totals))

	for i, elem := range elems {
		k := keys[i]
		seen[k]++

		typeIdx := seen[k]
		total := totals[k]

		var prev, next *html.Node

		if i > 0 {
			prev = elems[i-1]
		}

		if i+1 < len(elems) {
			next = elems[i+1]
		}

		infos[elem] = siblingInfo{
			prev:          prev,
			next:          next,
			index:         i + 1,
			typeIndex:     typeIdx,
			typeLastIndex: total - typeIdx + 1,
		}
	}

	return &sibParentData{elems: elems, infos: infos, childLen: len(parent.Children)}
}

// siblingEntryFor returns the memoized entry for parent, building it once.
// Hits and misses are counted for tests via siblingMemoStats.
func siblingEntryFor(parent *html.Node) *sibParentData {
	if parent == nil {
		return nil
	}

	sibMu.RLock()
	entry := sibParents[parent]
	sibMu.RUnlock()

	curLen := len(parent.Children)

	if entry != nil && entry.childLen == curLen {
		atomic.AddInt64(&sibHits, 1)

		return entry
	}

	newEntry := buildSibEntry(parent)

	sibMu.Lock()

	if existing, ok := sibParents[parent]; ok && existing.childLen == curLen {
		sibMu.Unlock()
		atomic.AddInt64(&sibHits, 1)

		return existing
	}

	if sibParents == nil {
		sibParents = make(map[*html.Node]*sibParentData)
	}

	if len(sibParents) > 16384 {
		sibParents = make(map[*html.Node]*sibParentData)
	}

	sibParents[parent] = newEntry
	sibMu.Unlock()
	atomic.AddInt64(&sibMisses, 1)

	return newEntry
}

// resetSiblingMemoForTest clears the sibling cache and counters.
func resetSiblingMemoForTest() {
	sibMu.Lock()
	sibParents = nil
	sibMu.Unlock()
	atomic.StoreInt64(&sibHits, 0)
	atomic.StoreInt64(&sibMisses, 0)
}

// siblingMemoStatsForTest returns cache hits and misses.
func siblingMemoStatsForTest() (hits, misses int64) {
	return atomic.LoadInt64(&sibHits), atomic.LoadInt64(&sibMisses)
}

// getSiblingInfo returns node's sibling metrics from the memoized parent
// entry. The parent's children are scanned once; later candidates reuse the
// stored indices, so a flat list costs O(N) to build plus O(1) per lookup.
func getSiblingInfo(node *html.Node) siblingInfo {
	var info siblingInfo

	if node == nil || node.Parent == nil {
		// No parent means no sibling edges; only the 1-based indices
		// carry meaning here.
		info.index = 1
		info.typeIndex = 1
		info.typeLastIndex = 1

		return info
	}

	entry := siblingEntryFor(node.Parent)
	if entry == nil {
		return info
	}

	if cached, ok := entry.infos[node]; ok {
		return cached
	}

	// node is not one of its parent's element children: all-zero
	// metrics, matching the former cache-miss result.
	return info
}

// maxHasVisits is the number of element inspections one selector match may
// spend inside :has(). Nested relative walks share this budget. When it
// runs out, :has() fails closed instead of scanning the rest of the tree.
const maxHasVisits = 1 << 20

// hasBudget is shared across a match and the :has() walks it starts.
type hasBudget struct {
	left int
}

// MatchState carries the runtime state stateful pseudo-classes need: the ids
// of the focused, hovered, and pressed elements. An empty id never matches.
// budget is set by Matches so every :has() walk in that match shares one cap.
type MatchState struct {
	Focus  string
	Hover  string
	Active string
	budget *hasBudget
}

// Match reports whether the selector matches the element node with no state.
// Matching runs right to left: the last part must match n, earlier parts must
// match ancestors/siblings per their combinators.
func Match(s Selector, n *html.Node) bool {
	return MatchState{Focus: "", Hover: "", Active: "", budget: nil}.Matches(s, n)
}

// selectorContainsHas reports whether sel needs a :has() visit budget: it
// holds :has() itself or nests :has() inside :not()/:is()/:where().
func selectorContainsHas(sel Selector) bool {
	if sel.hasValid {
		return sel.has
	}

	for _, part := range sel.Parts {
		if partContainsHas(part) {
			return true
		}
	}

	return false
}

// partContainsHas reports whether one compound needs a :has() budget.
func partContainsHas(part SelectorPart) bool {
	for _, pseudo := range part.Pseudos {
		if len(pseudo.Has) > 0 {
			return true
		}

		for _, s := range pseudo.Not {
			if selectorContainsHas(s) {
				return true
			}
		}

		for _, s := range pseudo.Is {
			if selectorContainsHas(s) {
				return true
			}
		}
	}

	return false
}

// ensureHasBudget returns st with a visit budget, allocating on the first
// :has() descent. A nil budget means unlimited (no :has() on the path).
func (st MatchState) ensureHasBudget() MatchState {
	if st.budget == nil {
		st.budget = &hasBudget{left: maxHasVisits}
	}

	return st
}

// Matches is Match with pseudo-class state.
func (st MatchState) Matches(s Selector, n *html.Node) bool {
	// Keep budget nil when the selector needs no :has(): nil means unlimited
	// and avoids one allocation per match. Allocate only when :has() may run.
	if st.budget == nil && selectorContainsHas(s) {
		st.budget = &hasBudget{left: maxHasVisits}
	}

	return leftmostMatch(st, s, n) != nil
}

// chargeHasVisit spends one :has() inspection. False means the budget is
// gone and the walk must stop.
func chargeHasVisit(state MatchState) bool {
	if state.budget == nil {
		return true
	}

	if state.budget.left <= 0 {
		return false
	}

	state.budget.left--

	return true
}

// MatchPseudo reports whether s selects the ::before or ::after pseudo-element
// of n (pe is "before" or "after") with no state.
func MatchPseudo(sel Selector, count *html.Node, pseudo string) bool {
	return MatchState{Focus: "", Hover: "", Active: "", budget: nil}.MatchesPseudo(sel, count, pseudo)
}

// MatchesPseudo is MatchPseudo with pseudo-class state.
func (st MatchState) MatchesPseudo(sel Selector, count *html.Node, pseudo string) bool {
	if count == nil || pseudo == "" || len(sel.Parts) == 0 {
		return false
	}

	if sel.Parts[len(sel.Parts)-1].PseudoElement != pseudo {
		return false
	}

	if st.budget == nil && selectorContainsHas(sel) {
		st.budget = &hasBudget{left: maxHasVisits}
	}

	return matchPseudoWalk(st, sel, count)
}

// matchPseudoWalk mirrors leftmostMatch with the final part's PseudoElement
// treated as cleared, without copying the parts slice (the host element must
// match the pseudo's compound, not the pseudo itself).
func matchPseudoWalk(state MatchState, sel Selector, node *html.Node) bool {
	if node == nil || node.Type != html.ElementNode || len(sel.Parts) == 0 {
		return false
	}

	last := sel.Parts[len(sel.Parts)-1]
	last.PseudoElement = ""

	if !matchPart(state, last, node) {
		return false
	}

	cur := node

	const prevPartOffset = 2 // walk left: last part is host, start at len-2

	for i := len(sel.Parts) - prevPartOffset; i >= 0; i-- {
		next := leftmostStep(state, sel.Parts[i+1].Combinator, sel.Parts[i], cur)
		if next == nil {
			return false
		}

		cur = next
	}

	return true
}

// leftmostMatch walks the selector combinator chain right-to-left (same walk
// as Match) and returns the element that matched the leftmost compound, or nil
// if the selector does not match node.
func leftmostMatch(state MatchState, sel Selector, node *html.Node) *html.Node {
	if node == nil || node.Type != html.ElementNode || len(sel.Parts) == 0 {
		return nil
	}

	if !matchPart(state, sel.Parts[len(sel.Parts)-1], node) {
		return nil
	}

	cur := node
	// Combinator is stored on the right-hand part of each pair (how that
	// part attaches to the previous). Walk left using Parts[i+1].Combinator.
	const prevPartOffset = 2 // walk left: last part is host, start at len-2

	for i := len(sel.Parts) - prevPartOffset; i >= 0; i-- {
		next := leftmostStep(state, sel.Parts[i+1].Combinator, sel.Parts[i], cur)
		if next == nil {
			return nil
		}

		cur = next
	}

	return cur
}

// leftmostStep advances cur one step left through the combinator chain: it
// returns the element that must match part, or nil when none exists.
func leftmostStep(state MatchState, combinator string, part SelectorPart, cur *html.Node) *html.Node {
	switch combinator {
	case ">":
		return matchLeftChild(state, part, cur)
	case "+":
		return matchLeftAdjacent(state, part, cur)
	case "~":
		return matchLeftSibling(state, part, cur)
	default: // descendant
		return matchLeftAncestor(state, part, cur)
	}
}

// matchLeftChild returns cur's element parent when it matches part, else nil.
func matchLeftChild(st MatchState, part SelectorPart, cur *html.Node) *html.Node {
	cur = cur.Parent
	if cur == nil || cur.Type != html.ElementNode || !matchPart(st, part, cur) {
		return nil
	}

	return cur
}

// matchLeftAdjacent returns cur's previous element sibling when it matches
// part, else nil.
func matchLeftAdjacent(st MatchState, part SelectorPart, cur *html.Node) *html.Node {
	prev := previousElementSibling(cur)
	if prev == nil || !matchPart(st, part, prev) {
		return nil
	}

	return prev
}

// matchLeftSibling returns the nearest previous element sibling of cur that
// matches part, else nil. The memoized parent entry supplies the element-only
// list and cur's position, so the walk visits element siblings only and never
// rescans the parent's full children per candidate.
func matchLeftSibling(state MatchState, part SelectorPart, cur *html.Node) *html.Node {
	if cur == nil || cur.Parent == nil {
		return nil
	}

	entry := siblingEntryFor(cur.Parent)
	if entry == nil {
		return nil
	}

	info, ok := entry.infos[cur]
	if !ok {
		return nil
	}

	idx := info.index - 1

	for i := idx - 1; i >= 0; i-- {
		cand := entry.elems[i]
		if matchPart(state, part, cand) {
			return cand
		}
	}

	return nil
}

// matchLeftAncestor returns the nearest element ancestor of cur that matches
// part, else nil.
func matchLeftAncestor(st MatchState, part SelectorPart, cur *html.Node) *html.Node {
	cur = cur.Parent
	for cur != nil && (cur.Type != html.ElementNode || !matchPart(st, part, cur)) {
		cur = cur.Parent
	}

	return cur
}

// SelectorIsElementLocal reports whether matching sel against an element depends
// only on the element's own tag, id, classes, and attributes (plus the
// pass-constant match state), never on ancestors, siblings, descendants, or
// sibling position. The layout cascade caches outcomes of element-local
// selectors per element signature and evaluates every other selector fresh
// for each element.
//
// A selector is element-local when it holds a single compound (no combinator
// to cross) whose pseudo-classes are element-bound: :link and :visited read
// the tag and href, :checked reads the tag and attributes, and :hover,
// :active, :focus, and :focus-visible compare the element id against the pass
// state. Every other pseudo-class (tree-structural, :root, :has(), :not(),
// :is(), :where(), and unknown names) makes the selector contextual. The
// final compound's pseudo-element is ignored: the selector index buckets
// pseudo rules separately, so it is constant within one lookup.
func SelectorIsElementLocal(sel Selector) bool {
	if len(sel.Parts) != 1 {
		return false
	}

	for _, pseudo := range sel.Parts[0].Pseudos {
		if len(pseudo.Has) > 0 || len(pseudo.Not) > 0 || len(pseudo.Is) > 0 {
			return false
		}

		switch pseudo.Name {
		case pseudoClassLink, pseudoClassVisited, pseudoClassChecked,
			pseudoClassFocus, pseudoClassFocusVisible, pseudoClassHover, pseudoClassActive:
		default:
			return false
		}
	}

	return true
}

// matchPart matches one compound against an element.
func matchPart(state MatchState, part SelectorPart, node *html.Node) bool {
	if node.Type != html.ElementNode {
		return false
	}
	// ::before/::after never match the host element (declarations apply to
	// generated pseudo boxes via MatchPseudo).
	if part.PseudoElement != "" {
		return false
	}

	if part.Tag != "*" && !tagNameMatches(part.Tag, node) {
		return false
	}

	// "id" is ASCII lower-case, so the direct map read equals
	// node.Attribute("id"): Attribute tries the exact key first and its
	// lower-case fallback is the same key. One lookup, no case scan.
	if part.ID != "" && node.Attrs["id"] != part.ID {
		return false
	}

	if !hasClasses(part, node) {
		return false
	}

	if !matchAttrs(part, node) {
		return false
	}

	return matchPseudos(state, part.Pseudos, node)
}

// tagNameMatches applies the CSS type-selector case rules: HTML element names
// match ASCII case-insensitively, foreign (SVG/MathML) element names match
// exactly, since their adjusted names are case-sensitive.
func tagNameMatches(selectorName string, node *html.Node) bool {
	if node.Namespace == html.NamespaceHTML {
		return asciiEqualFold(selectorName, node.Name)
	}

	return selectorName == node.Name
}

// sameElementType reports whether two element siblings share a namespace and
// a type name under the same case rules as tagNameMatches.
func sameElementType(left, right *html.Node) bool {
	if left.Namespace != right.Namespace {
		return false
	}

	if left.Namespace == html.NamespaceHTML {
		return asciiEqualFold(left.Name, right.Name)
	}

	return left.Name == right.Name
}

// matchPseudos reports whether every pseudo-class of the part matches node.
func matchPseudos(st MatchState, pseudos []PseudoClass, node *html.Node) bool {
	for _, pseudo := range pseudos {
		if !matchPseudo(st, pseudo, node) {
			return false
		}
	}

	return true
}

// hasClasses reports whether the element carries every class of the part.
func hasClasses(part SelectorPart, node *html.Node) bool {
	if len(part.Classes) == 0 {
		return true
	}

	// Fetch once: the old per-class node.Attribute("class") repeated the
	// map lookup and the upper-case scan for every class. "class" is lower-
	// case, so the direct read equals Attribute (missing key reads as "").
	classVal := node.Attrs["class"]

	for _, c := range part.Classes {
		if !hasClassToken(classVal, c) {
			return false
		}
	}

	return true
}

// attrCacheKey identifies one cached lowered attribute value: the element
// plus the attribute name. The original value is stored alongside so a
// mutated attribute misses instead of returning stale text.
type attrCacheKey struct {
	node *html.Node
	name string
}

// attrCacheVal is the cached original and its ASCII-lowered form.
type attrCacheVal struct {
	orig    string
	lowered string
}

var (
	attrCacheMu     sync.RWMutex
	attrCache       map[attrCacheKey]attrCacheVal
	attrCacheHits   int64
	attrCacheMisses int64
)

// getCachedLoweredAttr returns the ASCII-lowered attribute value, reusing the
// per-element cache. The fast path returns the original string when already
// lower, so no allocation happens then; an upper-case value allocates once
// per element and reuses the copy for later candidates.
func getCachedLoweredAttr(node *html.Node, name, val string) string {
	key := attrCacheKey{node: node, name: name}

	attrCacheMu.RLock()
	if attrCache != nil {
		if cached, ok := attrCache[key]; ok && cached.orig == val {
			lowered := cached.lowered
			attrCacheMu.RUnlock()
			atomic.AddInt64(&attrCacheHits, 1)

			return lowered
		}
	}
	attrCacheMu.RUnlock()

	lowered := asciiLower(val)

	attrCacheMu.Lock()

	if attrCache == nil {
		attrCache = make(map[attrCacheKey]attrCacheVal)
	}

	if len(attrCache) > 16384 {
		attrCache = make(map[attrCacheKey]attrCacheVal)
	}

	attrCache[key] = attrCacheVal{orig: val, lowered: lowered}
	attrCacheMu.Unlock()
	atomic.AddInt64(&attrCacheMisses, 1)

	return lowered
}

// resetAttrMemoForTest clears the i-flag lowered-value cache and counters.
func resetAttrMemoForTest() {
	attrCacheMu.Lock()
	attrCache = nil
	attrCacheMu.Unlock()
	atomic.StoreInt64(&attrCacheHits, 0)
	atomic.StoreInt64(&attrCacheMisses, 0)
}

// attrMemoStatsForTest returns cache hits and misses.
func attrMemoStatsForTest() (hits, misses int64) {
	return atomic.LoadInt64(&attrCacheHits), atomic.LoadInt64(&attrCacheMisses)
}

// resetMatchMemoForTest clears sibling and attribute caches plus counters.
func resetMatchMemoForTest() {
	resetSiblingMemoForTest()
	resetAttrMemoForTest()
}

// matchAttrs matches every attribute selector of the part against node.
func matchAttrs(part SelectorPart, node *html.Node) bool {
	for _, arg := range part.Attrs {
		val, found := "", false
		if node.Attrs != nil {
			val, found = node.Attrs[arg.Name]
		}

		if arg.Op == "" {
			if !found {
				return false
			}

			continue
		}

		if !found {
			return false
		}

		if !attrValueMatches(node, arg, val) {
			return false
		}
	}

	return true
}

// attrValueMatches evaluates one attribute operator against a value.
// ignoreCase is the Selectors 4 ASCII i flag; the selector side is pre-lowered
// at parse time (AttrSelector.valueLower) and the element side reuses the
// per-element lowered cache, so no per-check ToLower allocation happens.
func attrValueMatches(node *html.Node, arg AttrSelector, val string) bool {
	want := attrWantValue(arg)

	if arg.IgnoreCase {
		lowered := getCachedLoweredAttr(node, arg.Name, val)

		switch arg.Op {
		case "=":
			return lowered == want
		case "~=":
			return containsWord(lowered, want)
		case "*=", "^=", "$=", "|=":
			if want == "" {
				return false
			}

			return matchSubstringOp(arg.Op, lowered, want)
		}

		return false
	}

	switch arg.Op {
	case "=":
		return val == want
	case "~=":
		return containsWord(val, want)
	case "*=", "^=", "$=", "|=":
		if want == "" {
			return false
		}

		return matchSubstringOp(arg.Op, val, want)
	}

	return false
}

// attrWantValue resolves the comparison value, using the parse-time lowered
// cache when available and ASCII-lowering on the fly for hand-built selectors.
func attrWantValue(arg AttrSelector) string {
	if !arg.IgnoreCase {
		return arg.Value
	}

	if arg.valueLower != "" || arg.Value == "" {
		return arg.valueLower
	}

	// Hand-built selector without the parse-time cache.
	return asciiLower(arg.Value)
}

// matchSubstringOp evaluates one substring attribute operator against val.
// want is non-empty; the caller rejects the empty case (which matches
// nothing) before dispatching here.
func matchSubstringOp(op, val, want string) bool {
	switch op {
	case "*=":
		return strings.Contains(val, want)
	case "^=":
		return strings.HasPrefix(val, want)
	case "$=":
		return strings.HasSuffix(val, want)
	}
	// |= : exact match or value followed by a hyphen (HTML lang / BCP47-style).
	return val == want || strings.HasPrefix(val, want+"-")
}

// containsWord reports whether want (a single space-free word) is one of the
// space-separated words of val. The space guard states the CSS ~= rule
// directly; the token walk is shared with class matching (hasClassToken).
func containsWord(val, want string) bool {
	if want == "" || strings.Contains(want, " ") {
		return false
	}

	return hasClassToken(val, want)
}

func matchPseudo(state MatchState, pseudo PseudoClass, node *html.Node) bool {
	// State, link, and checked cases first: hover evaluation hits them on
	// every node, so they precede the tree/:has/:not/:is cases (names are
	// disjoint, order changes no result).
	switch pseudo.Name {
	case pseudoClassFocus, pseudoClassFocusVisible, pseudoClassHover, pseudoClassActive:
		return matchStatePseudo(state, pseudo.Name, node)
	case pseudoClassLink, pseudoClassVisited:
		// Print has no link history, so both match any anchor with an href.
		return isLinkAnchor(node)
	case pseudoClassChecked:
		return matchChecked(node)
	case pseudoClassHas:
		return matchAnyRelative(state, pseudo.Has, node)
	case condKindNot:
		return matchNone(state, pseudo.Not, node)
	case pseudoClassIs, pseudoClassWhere:
		return matchAnySelector(state, pseudo.Is, node)
	case firstChildPseudo, lastChildPseudo, nthChildPseudo, "root",
		firstOfTypePseudo, lastOfTypePseudo, nthOfTypePseudo, nthLastOfTypePseudo:
		return matchTreePseudo(pseudo, node)
	default:
		// Unknown pseudo-classes never match (kept on the compound so
		// selectors do not degrade to the host).
		return false
	}
}

// matchStatePseudo evaluates the runtime-state pseudo-classes against the
// focused, hovered, and pressed element ids.
func matchStatePseudo(state MatchState, name string, node *html.Node) bool {
	switch name {
	case pseudoClassFocus, pseudoClassFocusVisible:
		return matchStateID(state.Focus, node)
	case pseudoClassHover:
		return matchStateID(state.Hover, node)
	case pseudoClassActive:
		return matchStateID(state.Active, node)
	default:
		return false
	}
}

// matchStateID reports whether node carries the given runtime-state id.
func matchStateID(id string, node *html.Node) bool {
	// Direct read: "id" is lower-case, identical to Attribute("id").
	return id != "" && node.Attrs["id"] == id
}

// matchChecked reports whether node is a checked checkbox or radio, or a
// selected option. Tag and type compare ASCII case-insensitively without
// allocating lowered copies.
func matchChecked(node *html.Node) bool {
	if asciiEqualFold(node.Name, "input") {
		// Direct read: "type" is lower-case, identical to Attribute.
		t := node.Attrs["type"]
		if !asciiEqualFold(t, "checkbox") && !asciiEqualFold(t, "radio") {
			return false
		}

		_, ok := node.Attrs["checked"]

		return ok
	}

	if asciiEqualFold(node.Name, "option") {
		_, ok := node.Attrs["selected"]

		return ok
	}

	return false
}

// matchTreePseudo handles tree-structural pseudo-classes.
func matchTreePseudo(pseudo PseudoClass, node *html.Node) bool {
	switch pseudo.Name {
	case firstChildPseudo:
		return getSiblingInfo(node).prev == nil
	case lastChildPseudo:
		return getSiblingInfo(node).next == nil
	case nthChildPseudo:
		return matchNth(pseudo.nth, getSiblingInfo(node).index)
	case "root":
		return isRootElement(node)
	default:
		return matchOfTypePseudo(pseudo, node)
	}
}

// matchAnyRelative reports whether any relative selector applies to node.
// The first :has() descent allocates the shared visit budget; nil stays
// unlimited until then.
func matchAnyRelative(st MatchState, rels []RelativeSelector, node *html.Node) bool {
	st = st.ensureHasBudget()

	for _, rs := range rels {
		if matchRelative(st, rs, node) {
			return true
		}
	}

	return false
}

// matchNone reports whether no selector of the list matches node.
// Focus, hover, and active stay unset on the inner match. The visit budget
// is the caller's, or one budget shared by every argument when the caller
// has none, so each :not() argument does not start a fresh cap.
func matchNone(state MatchState, sels []Selector, node *html.Node) bool {
	inner := sharedBudgetState(state)

	for _, sel := range sels {
		if inner.Matches(sel, node) {
			return false
		}
	}

	return true
}

// matchAnySelector reports whether any selector of the list matches node.
// Focus, hover, and active stay unset. Arguments share one visit budget,
// the same rule as matchNone.
func matchAnySelector(state MatchState, sels []Selector, node *html.Node) bool {
	inner := sharedBudgetState(state)

	for _, sel := range sels {
		if inner.Matches(sel, node) {
			return true
		}
	}

	return false
}

// sharedBudgetState is a match state with no focus, hover, or active id.
// It keeps the caller's visit budget, or nil when the caller has none: the
// first :has() descent allocates one budget shared by the whole list, and a
// :not()/:is()/:where() without :has() allocates nothing.
func sharedBudgetState(state MatchState) MatchState {
	return MatchState{Focus: "", Hover: "", Active: "", budget: state.budget}
}

// isRootElement reports whether node is the document element (html in HTML).
// html.Parse wraps the tree in a synthetic ElementNode named "#document" -
// that must not match, and must not block <html> from matching.
func isRootElement(node *html.Node) bool {
	if node.Type != html.ElementNode || node.Name == "#document" || strings.HasPrefix(node.Name, "#") {
		return false
	}

	for p := node.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Name != "#document" && !strings.HasPrefix(p.Name, "#") {
			return false
		}
	}

	return true
}

// isLinkAnchor reports whether n is an <a> element with a non-empty href
// (any scheme, including "#" fragments and relative paths).
func isLinkAnchor(n *html.Node) bool {
	if n == nil || n.Type != html.ElementNode || !asciiEqualFold(n.Name, "a") {
		return false
	}

	href := strings.TrimSpace(n.Attrs["href"])

	return href != ""
}

// previousElementSibling returns count's nearest element sibling before it
// from the memoized parent entry (O(1) after the one per-parent build).
func previousElementSibling(count *html.Node) *html.Node {
	return getSiblingInfo(count).prev
}

// nextElementSibling returns count's nearest element sibling after it
// from the memoized parent entry (O(1) after the one per-parent build).
func nextElementSibling(count *html.Node) *html.Node {
	return getSiblingInfo(count).next
}

// nthKind discriminates the pre-parsed :nth-child() argument forms.
type nthKind int

const (
	nthInvalid nthKind = iota // unparseable, or not an nth-* pseudo
	nthOdd
	nthEven
	nthInt
	nthAnB
)

// nthForm is a :nth-child() argument parsed at selector-parse time so that
// matching is pure integer arithmetic (see matchNth).
type nthForm struct {
	kind nthKind `exhaustruct:"optional"` // nthInt: exact index; nthAnB: coefficient
	a    int     `exhaustruct:"optional"` // nthAnB: the constant
	b    int     `exhaustruct:"optional"`
}

// parseNthArg pre-parses a :nth-child() argument into the form matchNth
// evaluates. The argument is already lower-cased and trimmed at parse time;
// normalizing again here keeps the acceptance rules identical to the former
// string-based parser, including the never-match fallback.
func parseNthArg(arg string) nthForm {
	arg = strings.TrimSpace(strings.ToLower(arg))
	if arg == "" {
		return nthForm{}
	}

	if arg == "odd" {
		return nthForm{kind: nthOdd}
	}

	if arg == "even" {
		return nthForm{kind: nthEven}
	}
	// plain integer
	if n, err := strconv.Atoi(arg); err == nil {
		return nthForm{kind: nthInt, a: n}
	}
	// an+b / n+b / -n+b / an
	if !strings.Contains(arg, "n") {
		return nthForm{}
	}

	a, b, ok := parseAnPlusB(arg)
	if !ok {
		return nthForm{}
	}

	return nthForm{kind: nthAnB, a: a, b: b}
}

// matchNth implements :nth-child(an+b) / odd / even for 1-based index,
// evaluating the pre-parsed argument form.
func matchNth(nth nthForm, index int) bool {
	switch nth.kind {
	case nthOdd:
		return index%2 == 1
	case nthEven:
		return index%2 == 0
	case nthInt:
		return index == nth.a
	case nthAnB:
		if nth.a == 0 {
			return index == nth.b
		}

		if (index-nth.b)%nth.a != 0 {
			return false
		}

		k := (index - nth.b) / nth.a

		return k >= 0
	case nthInvalid:
		return false
	}

	return false
}

// parseAnPlusB parses the "an+b" form of a nth-child argument: "an", "n+b",
// "-n+b", or "b". Returns ok=false for anything unrecognized.
func parseAnPlusB(arg string) (int, int, bool) {
	parts := strings.SplitN(arg, "n", anPlusBSplitParts)
	asVal := strings.TrimSpace(parts[0])

	specA := 1
	if asVal == "-" {
		specA = -1
	} else if asVal != "" && asVal != "+" {
		parsed, err := strconv.Atoi(asVal)
		if err != nil {
			return 0, 0, false
		}

		specA = parsed
	}

	if len(parts) != anPlusBSplitParts || strings.TrimSpace(parts[1]) == "" {
		return specA, 0, true
	}

	specB, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, false
	}

	return specA, specB, true
}

// hasClassToken reports whether want is one whitespace-separated class token.
// The ASCII path covers HTML/CSS's normal class syntax without allocating a
// token slice or map. Non-ASCII whitespace falls back to the same Unicode
// whitespace behavior previously provided by strings.Fields.
func hasClassToken(value, want string) bool {
	if want == "" {
		return false
	}

	for start := 0; start < len(value); {
		for start < len(value) && isClassSpace(value[start]) {
			start++
		}

		end := start
		for end < len(value) && !isClassSpace(value[end]) {
			if value[end] >= nonASCIIStart {
				return hasUnicodeClassToken(value, want)
			}

			end++
		}

		if start < end && value[start:end] == want {
			return true
		}

		start = end
	}

	return false
}

func hasUnicodeClassToken(value, want string) bool {
	for start := 0; start < len(value); {
		for start < len(value) {
			runeValue, size := utf8.DecodeRuneInString(value[start:])
			if !unicode.IsSpace(runeValue) {
				break
			}

			start += size
		}

		end := start
		for end < len(value) {
			runeValue, size := utf8.DecodeRuneInString(value[end:])
			if unicode.IsSpace(runeValue) {
				break
			}

			end += size
		}

		if start < end && value[start:end] == want {
			return true
		}

		start = end
	}

	return false
}

func isClassSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\n' || value == '\v' || value == '\f' || value == '\r'
}

func isNthArgPseudo(name string) bool {
	switch name {
	case nthChildPseudo, nthOfTypePseudo, nthLastOfTypePseudo:
		return true
	default:
		return false
	}
}

// matchOfTypePseudo handles :first-of-type, :last-of-type, :nth-of-type(),
// and :nth-last-of-type(). Index is 1-based among element siblings with the
// same tag (HTML names compare case-insensitively, foreign names exactly).
func matchOfTypePseudo(pseudo PseudoClass, node *html.Node) bool {
	info := getSiblingInfo(node)

	switch pseudo.Name {
	case firstOfTypePseudo:
		return matchNth(nthForm{kind: nthInt, a: 1}, info.typeIndex)
	case lastOfTypePseudo:
		return matchNth(nthForm{kind: nthInt, a: 1}, info.typeLastIndex)
	case nthOfTypePseudo:
		return matchNth(pseudo.nth, info.typeIndex)
	case nthLastOfTypePseudo:
		return matchNth(pseudo.nth, info.typeLastIndex)
	default:
		return false
	}
}
