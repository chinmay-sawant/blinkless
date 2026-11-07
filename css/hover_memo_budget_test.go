package css

import (
	"strconv"
	"strings"
	"testing"
)

// Regression tests for the hover memo byte budget.
//
// The per-entry memo is keyed by (selector, tag, attrs, state), so flooding
// it with distinct fat attribute strings grows both the key count and the
// retained string bytes. The clear-on-full path must bound both: it drops the
// memo together with the intern pool, so retained memory stays under the
// count cap times the fattest key even when the flood inserts more states and
// more bytes than either bound allows. The tests below observe that through
// len(entry.memo) and the retained intern bytes; if the memo gains a
// dedicated byte cap, repoint budgetEntryInternBytes at that accounting.

func budgetFatValue(i, size int) string {
	return strings.Repeat("v", size) + "-" + strconv.Itoa(i)
}

// budgetEntryInternBytes sums the retained backing bytes of the entry intern
// pool: the canonical tag/attrs strings every memo key shares.
func budgetEntryInternBytes(entry *hoverListEntry) int {
	entry.mu.RLock()
	defer entry.mu.RUnlock()

	total := 0
	for _, s := range entry.intern {
		total += len(s)
	}

	return total
}

func TestHoverMemoByteBudgetFloodStaysBoundedAndCorrect(t *testing.T) {
	resetHoverCache()

	styled := internTestStyled(t, `.u:hover{color:red}`, `<div id="panel"><button class="u" id="a">A</button></div>`)
	entry := cachedHoverList(styled.sheets)
	idx, pseudo := internTestLocalSelector(t, entry)
	node := internTestNodeByID(t, styled.root, "a")

	const fatSize = 2048
	const extra = 1024
	total := maxHoverMemoPerEntry + extra

	inserted := 0
	maxAttrs := 0
	for i := range total {
		node.Attrs["data-fat"] = budgetFatValue(i, fatSize)
		tag := hoverTagName(node)
		attrs := hoverAttrKey(node.Attrs)
		inserted += len(attrs)
		maxAttrs = max(maxAttrs, len(attrs))

		old := styled.state
		old.Hover = "flood-" + strconv.Itoa(i)
		hoverMemoMatch(entry, idx, node, tag, attrs, old, pseudo)
	}

	// (1) The flood inserts more states than the count cap and more attr
	// bytes than the retained budget, but the memo stays bounded by the
	// count cap: clear-on-full fired and took the intern pool with it.
	budget := maxHoverMemoPerEntry * (maxAttrs + len(hoverTagName(node)) + 64)
	if inserted <= budget {
		t.Fatalf("flood inserted %d attr bytes under budget %d: raise extra or fatSize", inserted, budget)
	}
	if got := memoLen(entry); got > maxHoverMemoPerEntry {
		t.Fatalf("memo holds %d keys after fat flood, want <= %d", got, maxHoverMemoPerEntry)
	} else if got >= total {
		t.Fatalf("memo holds %d keys for %d distinct fat states: clear-on-full never dropped", got, total)
	}
	if got := budgetEntryInternBytes(entry); got > budget {
		t.Fatalf("intern pool retains %d bytes after fat flood, want <= %d", got, budget)
	}

	// (2) Post-clear evaluations still equal uncached Matches: for states the
	// flood evicted, for the most recent state, and for a never-seen one.
	for _, i := range []int{0, 1, total - 1} {
		node.Attrs["data-fat"] = budgetFatValue(i, fatSize)
		tag := hoverTagName(node)
		attrs := hoverAttrKey(node.Attrs)

		st := styled.state
		st.Hover = "flood-" + strconv.Itoa(i)
		if got, want := hoverMemoMatch(entry, idx, node, tag, attrs, st, pseudo),
			st.Matches(entry.selectors[idx], node); got != want {
			t.Fatalf("evicted flood-%d memoized %v after clear, want fresh %v", i, got, want)
		}
	}

	node.Attrs["data-fat"] = budgetFatValue(total, fatSize)
	freshTag := hoverTagName(node)
	freshAttrs := hoverAttrKey(node.Attrs)
	fresh := styled.state
	fresh.Hover = "never-seen-budget"
	if got, want := hoverMemoMatch(entry, idx, node, freshTag, freshAttrs, fresh, pseudo),
		fresh.Matches(entry.selectors[idx], node); got != want {
		t.Fatalf("fresh hover memoized %v after clear, want fresh %v", got, want)
	}
}

func TestHoverMemoFatNodeSharesOneCanonicalSignature(t *testing.T) {
	resetHoverCache()

	styled := internTestStyled(t, `.u:hover{color:red}`, `<div id="panel"><button class="u" id="a">A</button></div>`)
	entry := cachedHoverList(styled.sheets)
	idx, pseudo := internTestLocalSelector(t, entry)
	node := internTestNodeByID(t, styled.root, "a")

	// One fixed fat signature evaluated under many hover ids: the intern
	// pool must keep a single canonical backing, not one copy per state.
	node.Attrs["data-blob"] = strings.Repeat("b", 2048)
	tag := hoverTagName(node)
	attrs := hoverAttrKey(node.Attrs)

	const ids = 64
	first := make([]bool, ids)
	for i := range ids {
		st := styled.state
		st.Hover = "pinned-" + strconv.Itoa(i)
		first[i] = hoverMemoMatch(entry, idx, node, tag, attrs, st, pseudo)
		if want := st.Matches(entry.selectors[idx], node); first[i] != want {
			t.Fatalf("pinned-%d memoized %v, want fresh %v", i, first[i], want)
		}
	}

	// (3) Many hover ids, one canonical signature: key count grows with
	// distinct states while the signature pool stays at one entry and the
	// retained bytes stay near one fat backing.
	if got, want := memoLen(entry), ids; got != want {
		t.Fatalf("memo holds %d keys for %d hover ids, want %d", got, ids, want)
	}
	if got := entrySigPoolSize(entry); got != 1 {
		t.Fatalf("fat node retains %d canonical signatures under %d hover ids, want 1", got, ids)
	}
	if got, want := budgetEntryInternBytes(entry), len(tag)+len(attrs)+64; got > want {
		t.Fatalf("intern pool retains %d bytes for one fat signature, want <= %d", got, want)
	}

	for i := range ids {
		st := styled.state
		st.Hover = "pinned-" + strconv.Itoa(i)
		if got := hoverMemoMatch(entry, idx, node, tag, attrs, st, pseudo); got != first[i] {
			t.Fatalf("pinned-%d re-evaluated to %v, want %v", i, got, first[i])
		}
	}
	if got := memoLen(entry); got != ids {
		t.Fatalf("repeat evaluations grew memo to %d keys, want %d (no growth)", got, ids)
	}
}
