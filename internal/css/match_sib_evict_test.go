package css

import (
	"testing"
	"weak"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// evictProbeParent builds a synthetic parent with four span element children,
// wiring Parent links the way the parser would so the sibling memo treats it
// like a real row. It returns the parent and its children in order.
func evictProbeParent() (*html.Node, []*html.Node) {
	const childCount = 4

	parent := &html.Node{Type: html.ElementNode, Namespace: html.NamespaceHTML, Name: "div"}
	kids := make([]*html.Node, childCount)
	for i := range kids {
		kids[i] = &html.Node{Type: html.ElementNode, Namespace: html.NamespaceHTML, Name: "span", Parent: parent}
	}
	parent.Children = kids

	return parent, kids
}

// evictProbeMemoSize returns the current number of memoized sibling entries.
func evictProbeMemoSize() int {
	sibMu.RLock()
	defer sibMu.RUnlock()

	return len(sibParents)
}

// evictProbeParse parses one selector or fails the test.
func evictProbeParse(t *testing.T, src string) Selector {
	t.Helper()

	sel, ok := parseSelector(src)
	if !ok {
		t.Fatalf("parseSelector(%q) failed", src)
	}

	return sel
}

// TestSiblingEntryEvictionBounded inserts past the soft cap through the public
// matching path and pins the memo size: without eviction it would hold one
// entry per distinct parent, with it the map stays at or under the soft cap.
//
//nolint:paralleltest // shares the process-wide sibling memo and its counters
func TestSiblingEntryEvictionBounded(t *testing.T) {
	resetMatchMemoForTest()

	firstSel := evictProbeParse(t, "span:first-child")

	total := sibParentsSoftCap + 256
	for i := 0; i < total; i++ {
		_, kids := evictProbeParent()
		if !Match(firstSel, kids[0]) {
			t.Fatalf("parent %d: span:first-child should match the first child", i)
		}
	}

	size := evictProbeMemoSize()
	if size > sibParentsSoftCap {
		t.Fatalf("sibling memo size = %d after %d parents, want <= soft cap %d", size, total, sibParentsSoftCap)
	}

	if size >= total {
		t.Fatalf("sibling memo size = %d after %d parents: soft-cap eviction never fired", size, total)
	}
}

// TestSiblingEntryEvictedParentRebuilds pins rebuild-on-demand: a parent whose
// entry the cap evicted still matches correctly, the rematch re-populates the
// memo (miss counter moves), and the map stays bounded afterwards.
//
//nolint:paralleltest // shares the process-wide sibling memo and its counters
func TestSiblingEntryEvictedParentRebuilds(t *testing.T) {
	resetMatchMemoForTest()

	firstSel := evictProbeParse(t, "span:first-child")
	secondSel := evictProbeParse(t, "span:nth-child(2)")
	lastSel := evictProbeParse(t, "span:last-child")

	victim, vkids := evictProbeParent()
	if !Match(firstSel, vkids[0]) {
		t.Fatal("setup: span:first-child should match the victim's first child")
	}

	for i := 0; i < sibParentsSoftCap+512; i++ {
		_, kids := evictProbeParent()
		if !Match(firstSel, kids[0]) {
			t.Fatalf("parent %d: span:first-child should match the first child", i)
		}
	}

	// Quarter-evicts pick victims in map order, so the victim may or may not
	// have been dropped already. Evict it explicitly — the same delete the
	// cap path performs — so the rebuild assertions below are deterministic.
	sibMu.Lock()
	delete(sibParents, weak.Make(victim))
	sibMu.Unlock()

	_, missesBefore := siblingMemoStatsForTest()

	if !Match(firstSel, vkids[0]) {
		t.Error("after eviction: span:first-child should still match the victim's first child")
	}

	if Match(firstSel, vkids[1]) {
		t.Error("after eviction: span:first-child should not match the victim's second child")
	}

	if !Match(secondSel, vkids[1]) {
		t.Error("after eviction: span:nth-child(2) should still match the victim's second child")
	}

	if !Match(lastSel, vkids[3]) {
		t.Error("after eviction: span:last-child should still match the victim's last child")
	}

	if Match(lastSel, vkids[0]) {
		t.Error("after eviction: span:last-child should not match the victim's first child")
	}

	if info := getSiblingInfo(vkids[1]); info.index != 2 {
		t.Errorf("after eviction: rebuilt index of second child = %d, want 2", info.index)
	}

	_, missesAfter := siblingMemoStatsForTest()
	if missesAfter <= missesBefore {
		t.Errorf("victim rematch did not rebuild an entry (misses %d -> %d)", missesBefore, missesAfter)
	}

	if size := evictProbeMemoSize(); size > sibParentsSoftCap {
		t.Errorf("sibling memo size = %d after victim rebuild, want <= soft cap %d", size, sibParentsSoftCap)
	}
}

// TestSiblingEntryDroppedTreesBounded pins that entries for dropped trees do
// not accumulate without bound: waves of distinct synthetic trees whose only
// remaining reference is the memo itself still leave the map bounded, and a
// witness from the live wave keeps matching.
//
//nolint:paralleltest // shares the process-wide sibling memo and its counters
func TestSiblingEntryDroppedTreesBounded(t *testing.T) {
	resetMatchMemoForTest()

	nthSel := evictProbeParse(t, "span:nth-child(3)")

	const batches = 4
	const perBatch = 800

	total := batches * perBatch
	if total <= sibParentsSoftCap {
		t.Fatalf("test setup: %d trees do not pass soft cap %d", total, sibParentsSoftCap)
	}

	var witness *html.Node

	for b := 0; b < batches; b++ {
		// Each parent goes out of scope at the end of its iteration: past
		// batches are dropped trees pinned only by the memo, modelling
		// documents the caller has discarded.
		for i := 0; i < perBatch; i++ {
			_, kids := evictProbeParent()
			if !Match(nthSel, kids[2]) {
				t.Fatalf("batch %d parent %d: span:nth-child(3) should match the third child", b, i)
			}

			if b == batches-1 && i == 0 {
				witness = kids[2]
			}
		}
	}

	size := evictProbeMemoSize()
	if size > sibParentsSoftCap {
		t.Fatalf("sibling memo size = %d after %d dropped-tree inserts, want <= soft cap %d", size, total, sibParentsSoftCap)
	}

	if size >= total {
		t.Fatalf("sibling memo size = %d after %d dropped-tree inserts: stale entries never flushed", size, total)
	}

	if witness == nil {
		t.Fatal("test setup: no witness kept from the live batch")
	}

	if !Match(nthSel, witness) {
		t.Error("witness from the live batch should still match span:nth-child(3)")
	}

	if info := getSiblingInfo(witness); info.index != 3 {
		t.Errorf("witness rebuilt index = %d, want 3", info.index)
	}
}
