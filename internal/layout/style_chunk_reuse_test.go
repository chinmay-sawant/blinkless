package layout

import (
	"reflect"
	"testing"
)

// chunkReuseSubtreeStyles is the synthetic subtree one resolution pass
// interns: two repeated declaration shapes (red, blue) plus one distinct
// shape (green), in document order.
func chunkReuseSubtreeStyles() []ResolvedStyle {
	red := ResolvedStyle{Color: [3]float64{1, 0, 0}, FontSize: 12, Width: 100}
	blue := ResolvedStyle{Color: [3]float64{0, 0, 1}, FontSize: 12, Width: 100}
	green := ResolvedStyle{Color: [3]float64{0, 1, 0}, FontSize: 14, Width: 200}

	return []ResolvedStyle{red, blue, red, green, blue}
}

// resolveChunkSubtree appends one subtree walk in document order and returns
// the interned record of each position. Repeated shapes share one record.
func resolveChunkSubtree(store *styleStore, styles []ResolvedStyle) []*ResolvedStyle {
	out := make([]*ResolvedStyle, 0, len(styles))

	for _, style := range styles {
		out = append(out, store.append(style))
	}

	return out
}

// reuseSpareChunks returns a store that reuses the chunk capacities of src
// with empty contents and a clean intern map: the reused-spare path a reset
// hands the next resolution pass.
func reuseSpareChunks(src *styleStore) *styleStore {
	out := &styleStore{}

	for _, chunk := range src.chunks {
		out.chunks = append(out.chunks, make([]ResolvedStyle, 0, cap(chunk)))
	}

	return out
}

// assertChunkSubtreeIdentity requires the repeated-shape positions to share
// one interned record while distinct shapes keep their own.
func assertChunkSubtreeIdentity(t *testing.T, ptrs []*ResolvedStyle) {
	t.Helper()

	if len(ptrs) != 5 {
		t.Fatalf("subtree records = %d, want 5", len(ptrs))
	}

	if ptrs[0] != ptrs[2] {
		t.Fatal("repeated red styles resolved to different records")
	}

	if ptrs[1] != ptrs[4] {
		t.Fatal("repeated blue styles resolved to different records")
	}

	if ptrs[0] == ptrs[1] || ptrs[0] == ptrs[3] || ptrs[1] == ptrs[3] {
		t.Fatal("distinct styles shared one interned record")
	}
}

// TestStyleChunkReuseKeepsInternedIdentity proves chunk reuse never changes
// resolution: the same subtree resolved through a fresh store and through a
// reused-spare store interns identical styles to identical pointers and keeps
// the same values position by position.
func TestStyleChunkReuseKeepsInternedIdentity(t *testing.T) {
	t.Parallel()

	subtree := chunkReuseSubtreeStyles()

	fresh := &styleStore{}
	freshPtrs := resolveChunkSubtree(fresh, subtree)
	assertChunkSubtreeIdentity(t, freshPtrs)

	reused := reuseSpareChunks(fresh)
	reusedPtrs := resolveChunkSubtree(reused, subtree)
	assertChunkSubtreeIdentity(t, reusedPtrs)

	for index := range freshPtrs {
		if !reflect.DeepEqual(*freshPtrs[index], *reusedPtrs[index]) {
			t.Fatalf("position %d differs between the fresh and reused stores", index)
		}
	}
}

// TestStyleChunkResetThenReuseDropsStaleRecord proves a reset-then-reuse
// cycle never serves a stale record: after the reset keeps the spare backing
// and drops the intern map, a mutated input resolves to the new values and
// re-interns on repeat.
func TestStyleChunkResetThenReuseDropsStaleRecord(t *testing.T) {
	t.Parallel()

	before := ResolvedStyle{Color: [3]float64{1, 0, 0}, FontSize: 12, Width: 100}
	after := ResolvedStyle{Color: [3]float64{0, 0, 1}, FontSize: 12, Width: 200}

	store := &styleStore{}
	first := store.append(before)
	snapshot := *first

	for index, chunk := range store.chunks {
		clear(chunk)
		store.chunks[index] = chunk[:0]
	}

	store.intern = nil

	got := store.append(after)

	if !reflect.DeepEqual(*got, after) {
		t.Fatalf("reused chunk color = %v, want %v", got.Color, after.Color)
	}

	if reflect.DeepEqual(*got, snapshot) {
		t.Fatal("mutated style resolved to the stale record")
	}

	if again := store.append(after); again != got {
		t.Fatal("reset store did not intern the repeated style")
	}
}
