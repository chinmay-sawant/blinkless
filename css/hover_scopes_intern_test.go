package css

import (
	"slices"
	"strconv"
	"testing"

	pubhtml "github.com/chinmay-sawant/blinkless/html"
	ihtml "github.com/chinmay-sawant/blinkless/internal/html"
)

// Regression tests for hover-memo string interning.
//
// The per-entry memo keys every element-local outcome by (selector, tag,
// attrs, hover/focus/active state). The tag/attrs half is a pure function of
// the node, so one node evaluated under many hover ids must share a single
// canonical signature: key count grows with distinct states, never with
// repeated evaluations, and distinct attribute maps must never alias the
// same key. The helpers below observe that sharing through len(entry.memo)
// and the distinct (tag, attrs) pairs it holds; if hoverListEntry gains a
// dedicated intern map, repoint entrySigPoolSize at len(entry.<pool>).

func internTestStyled(t *testing.T, body string) *Document {
	t.Helper()

	doc, err := pubhtml.Parse([]byte(`<style>.u:hover{color:red}</style>` + body))
	if err != nil {
		t.Fatal(err)
	}

	styled, err := Apply(t.Context(), doc, Options{WidthPx: 320, HeightPx: 200, Hover: "fixed-next"})
	if err != nil {
		t.Fatal(err)
	}

	return styled
}

func internTestNodeByID(t *testing.T, root *ihtml.Node, id string) *ihtml.Node {
	t.Helper()

	var found *ihtml.Node

	var walk func(*ihtml.Node)
	walk = func(node *ihtml.Node) {
		if found != nil {
			return
		}
		if node.Type == ihtml.ElementNode && node.Attribute("id") == id {
			found = node

			return
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(root)
	if found == nil {
		t.Fatalf("no element with id %q", id)
	}

	return found
}

func internTestLocalSelector(t *testing.T, entry *hoverListEntry) (int, string) {
	t.Helper()

	for i, sel := range entry.selectors {
		if !entry.local[i] {
			continue
		}
		pseudo := ""
		if len(sel.Parts) > 0 {
			pseudo = sel.Parts[len(sel.Parts)-1].PseudoElement
		}

		return i, pseudo
	}

	t.Fatal("no element-local hover selector cached")

	return 0, ""
}

// memoLen reads the entry memo size: one slot per distinct
// (selector, signature, state) outcome.
func memoLen(entry *hoverListEntry) int {
	entry.mu.RLock()
	defer entry.mu.RUnlock()

	return len(entry.memo)
}

// entrySigPoolSize counts the distinct canonical (tag, attrs) signature
// pairs retained by the entry: the strings an intern pool must deduplicate.
func entrySigPoolSize(entry *hoverListEntry) int {
	entry.mu.RLock()
	defer entry.mu.RUnlock()

	type sig struct{ tag, attrs string }
	sigs := make(map[sig]struct{}, len(entry.memo))
	for key := range entry.memo {
		sigs[sig{key.tag, key.attrs}] = struct{}{}
	}

	return len(sigs)
}

//nolint:paralleltest // resets the process-wide hover cache
func TestHoverMemoSignatureSharedAcrossHoverIDs(t *testing.T) {
	resetHoverCache()

	styled := internTestStyled(t, `<div id="panel"><button class="u" id="a">A</button></div>`)
	entry := cachedHoverList(styled.sheets)
	idx, pseudo := internTestLocalSelector(t, entry)
	node := internTestNodeByID(t, styled.root, "a")
	tag := hoverTagName(node)
	attrs := hoverAttrKey(node.Attrs)

	const ids = 64
	base := memoLen(entry)

	next := styled.state
	next.Hover = "fixed-next"

	first := make([]bool, ids)
	for i := range ids {
		old := styled.state
		old.Hover = "hover-id-" + strconv.Itoa(i)
		first[i] = hoverMemoMatch(entry, idx, node, tag, attrs, old, pseudo)
	}
	nextHit := hoverMemoMatch(entry, idx, node, tag, attrs, next, pseudo)

	if got, want := memoLen(entry)-base, ids+1; got != want {
		t.Fatalf("memo grew by %d keys for %d hover ids, want %d", got, ids, want)
	}
	if got := entrySigPoolSize(entry); got != 1 {
		t.Fatalf("entry retains %d canonical signatures for one node, want 1", got)
	}

	for i := range ids {
		old := styled.state
		old.Hover = "hover-id-" + strconv.Itoa(i)
		if got := hoverMemoMatch(entry, idx, node, tag, attrs, old, pseudo); got != first[i] {
			t.Fatalf("hover id %d re-evaluated to %v, want %v", i, got, first[i])
		}
	}
	if got := hoverMemoMatch(entry, idx, node, tag, attrs, next, pseudo); got != nextHit {
		t.Fatalf("steady hover re-evaluated to %v, want %v", got, nextHit)
	}

	if got, want := memoLen(entry)-base, ids+1; got != want {
		t.Fatalf("repeat evaluations grew memo to %d new keys, want %d (no growth)", got, want)
	}
}

func TestHoverAttrKeyNeverAliasesDistinctMaps(t *testing.T) {
	t.Parallel()

	maps := []map[string]string{
		{"id": "a"},
		{"id": "b"},
		{"a": "bc"},
		{"ab": "c"},
		{"a": "1", "b": "22"},
		{"a": "12", "b": "2"},
		{"x": "p;q"},
		{"x": "p", "q": ""},
		{"a": "x=y"},
		{"a": "x", "y": ""},
		{"class": "u", "id": "a"},
		{"class": "u"},
	}

	seen := make(map[string]int, len(maps))
	for i, attrs := range maps {
		key := hoverAttrKey(attrs)
		if first, dup := seen[key]; dup {
			t.Fatalf("attr maps %v and %v share hover key %q", maps[first], attrs, key)
		}
		seen[key] = i
	}

	// Same entries in a different insertion order encode identically: the
	// names are sorted, so sharing here is correct, not aliasing.
	first := hoverAttrKey(map[string]string{"b": "22", "a": "1"})
	second := hoverAttrKey(map[string]string{"a": "1", "b": "22"})
	if first != second {
		t.Fatalf("same attr map encoded as %q then %q", first, second)
	}

	if got := hoverAttrKey(nil); got != hoverAttrKey(map[string]string{}) {
		t.Fatalf("nil and empty attr maps differ: %q", got)
	}
}

//nolint:paralleltest // resets the process-wide hover cache
func TestHoverMemoDistinctAttrsStayDistinct(t *testing.T) {
	resetHoverCache()

	styled := internTestStyled(t,
		`<div id="panel"><button class="u" id="a">A</button><button class="u" id="b">B</button></div>`)
	entry := cachedHoverList(styled.sheets)
	idx, pseudo := internTestLocalSelector(t, entry)
	base := memoLen(entry)

	st := styled.state
	st.Hover = "someone-else"
	for _, id := range []string{"a", "b"} {
		node := internTestNodeByID(t, styled.root, id)
		want := st.Matches(entry.selectors[idx], node)
		if got := hoverMemoMatch(entry, idx, node, hoverTagName(node), hoverAttrKey(node.Attrs), st, pseudo); got != want {
			t.Fatalf("button %q memoized %v, want fresh %v", id, got, want)
		}
	}

	if got, want := memoLen(entry)-base, 2; got != want {
		t.Fatalf("two distinct attr strings share memo keys: grew by %d, want %d", got, want)
	}
}

//nolint:paralleltest // resets the process-wide hover cache
func TestHoverMemoCapClearStaysBoundedAndCorrect(t *testing.T) {
	resetHoverCache()

	styled := internTestStyled(t, `<div id="panel"><button class="u" id="a">A</button></div>`)
	entry := cachedHoverList(styled.sheets)
	idx, pseudo := internTestLocalSelector(t, entry)
	node := internTestNodeByID(t, styled.root, "a")
	tag := hoverTagName(node)
	attrs := hoverAttrKey(node.Attrs)

	next := styled.state
	if got, want := hoverMemoMatch(entry, idx, node, tag, attrs, next, pseudo),
		next.Matches(entry.selectors[idx], node); got != want {
		t.Fatalf("steady hover memoized %v, want fresh %v", got, want)
	}

	const extra = 256
	inserted := 0
	for i := range maxHoverMemoPerEntry + extra {
		old := styled.state
		old.Hover = "flood-" + strconv.Itoa(i)
		hoverMemoMatch(entry, idx, node, tag, attrs, old, pseudo)
		inserted++
	}

	if got := memoLen(entry); got > maxHoverMemoPerEntry {
		t.Fatalf("memo holds %d keys after flooding, want <= %d", got, maxHoverMemoPerEntry)
	} else if got >= inserted {
		t.Fatalf("memo holds %d keys for %d distinct states: clear-on-full never dropped", got, inserted)
	}

	for _, hover := range []string{"flood-0", "flood-1", "never-seen"} {
		st := styled.state
		st.Hover = hover
		want := st.Matches(entry.selectors[idx], node)
		if got := hoverMemoMatch(entry, idx, node, tag, attrs, st, pseudo); got != want {
			t.Fatalf("hover %q memoized %v after clear, want fresh %v", hover, got, want)
		}
	}

	for _, previous := range []string{"flood-0", "a", ""} {
		got, ok := styled.HoverChangedScopes(previous)
		if !ok {
			t.Fatalf("previous %q: HoverChangedScopes returned false", previous)
		}
		want, _ := hoverCacheReference(t, styled, previous)
		if !slices.Equal(got, want) {
			t.Fatalf("previous %q: scopes = %v, want %v", previous, got, want)
		}
	}
}
