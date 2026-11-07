package layout

import "testing"

// TestStyleStoreChunksGrowAndKeepPointers checks that the style store keeps
// its chunks small at first and that the pointers it handed out stay valid
// when a later chunk is added.
func TestStyleStoreChunksGrowAndKeepPointers(t *testing.T) {
	t.Parallel()

	store := &styleStore{}

	const count = 40

	pointers := make([]*ResolvedStyle, 0, count)

	for index := range count {
		style := ResolvedStyle{Color: [3]float64{float64(index), 0, 0}}
		pointers = append(pointers, store.append(style))
	}

	if len(store.chunks) < 2 {
		t.Fatalf("chunks = %d, want more than one for %d styles", len(store.chunks), count)
	}

	for index, pointer := range pointers {
		if pointer == nil {
			t.Fatalf("style %d has no record", index)
		}

		if got := pointer.Color[0]; got != float64(index) {
			t.Fatalf("style %d reads %v, want %v", index, got, float64(index))
		}
	}

	first := cap(store.chunks[0])
	if first != styleStoreFirstChunk {
		t.Fatalf("the first chunk holds %d records, want %d", first, styleStoreFirstChunk)
	}

	for index, chunk := range store.chunks {
		if cap(chunk) > styleStoreChunkSize {
			t.Fatalf("chunk %d holds %d records, want at most %d", index, cap(chunk), styleStoreChunkSize)
		}
	}
}

// TestStyleStoreSharesDuplicateStyles checks that the store still interns an
// exact duplicate, so a page with repeated shapes keeps one record.
func TestStyleStoreSharesDuplicateStyles(t *testing.T) {
	t.Parallel()

	store := &styleStore{}

	first := store.append(ResolvedStyle{Color: [3]float64{7, 0, 0}})
	second := store.append(ResolvedStyle{Color: [3]float64{7, 0, 0}})

	if first != second {
		t.Fatal("the store kept two records for one style")
	}
}

// TestStyleStoreChunkFreeListChannelsBoundedAndRecycled tests that putStyleStoreChunk
// zeroes lingering records and routes chunks to the segregated free channels, and
// that drainStyleStoreChunkPools drains them cleanly.
func TestStyleStoreChunkFreeListChannelsBoundedAndRecycled(t *testing.T) {
	drainStyleStoreChunkPools()
	defer drainStyleStoreChunkPools()

	// Put an 8-cap chunk with non-zero fields
	c8 := make([]ResolvedStyle, 0, styleStoreFirstChunk)
	c8 = append(c8, ResolvedStyle{Color: [3]float64{1, 2, 3}, CustomProps: map[string]string{"--foo": "bar"}})
	putStyleStoreChunk(&c8)

	// Buffer must be cleared
	if c8[:cap(c8)][0].CustomProps != nil {
		t.Fatal("putStyleStoreChunk did not clear CustomProps map pointer")
	}

	got8 := getStyleStoreChunk(styleStoreFirstChunk)
	if got8 == nil || cap(*got8) != styleStoreFirstChunk {
		t.Fatalf("getStyleStoreChunk(%d) = %v, want recycled chunk", styleStoreFirstChunk, got8)
	}

	// 32-cap chunk
	c32 := make([]ResolvedStyle, 0, styleStoreFirstChunk*4)
	c32 = append(c32, ResolvedStyle{FontSize: 24})
	putStyleStoreChunk(&c32)

	got32 := getStyleStoreChunk(styleStoreFirstChunk * 4)
	if got32 == nil || cap(*got32) != styleStoreFirstChunk*4 {
		t.Fatalf("getStyleStoreChunk(%d) = %v, want recycled chunk", styleStoreFirstChunk*4, got32)
	}

	// 64-cap chunk
	c64 := make([]ResolvedStyle, 0, styleStoreChunkSize)
	c64 = append(c64, ResolvedStyle{Width: 100})
	putStyleStoreChunk(&c64)

	got64 := getStyleStoreChunk(styleStoreChunkSize)
	if got64 == nil || cap(*got64) != styleStoreChunkSize {
		t.Fatalf("getStyleStoreChunk(%d) = %v, want recycled chunk", styleStoreChunkSize, got64)
	}

	// Draining leaves channels empty
	putStyleStoreChunk(&c8)
	putStyleStoreChunk(&c32)
	putStyleStoreChunk(&c64)
	drainStyleStoreChunkPools()

	if len(styleStoreChunkFree8) != 0 || len(styleStoreChunkFree32) != 0 || len(styleStoreChunkFree64) != 0 {
		t.Fatal("drainStyleStoreChunkPools left entries in channels")
	}
}

// TestResultReleaseRecyclesStoreChunks verifies that Result.Release returns
// storeChunks to the free lists.
func TestResultReleaseRecyclesStoreChunks(t *testing.T) {
	drainStyleStoreChunkPools()
	defer drainStyleStoreChunkPools()

	c64 := make([]ResolvedStyle, 0, styleStoreChunkSize)
	res := &Result{
		storeChunks: [][]ResolvedStyle{c64},
	}

	res.Release()

	if res.storeChunks != nil {
		t.Fatal("res.storeChunks was not cleared after Release")
	}

	got := getStyleStoreChunk(styleStoreChunkSize)
	if got == nil {
		t.Fatal("Result.Release did not recycle storeChunk to free list")
	}
}

// TestSubtreeStyleStoreRejectedChunksTransferredToResult verifies that when a
// store is rejected from the subtree store pool, its chunks are transferred to
// prev.storeChunks rather than leaked.
func TestSubtreeStyleStoreRejectedChunksTransferredToResult(t *testing.T) {
	prev := &Result{}
	store := &styleStore{}

	// Allocate more chunks than maxSubtreeStyleStoreChunks (which is 4)
	for range maxSubtreeStyleStoreChunks + 1 {
		store.chunks = append(store.chunks, make([]ResolvedStyle, 0, styleStoreChunkSize))
	}

	tag := subtreeStyleDocTag{root: 1, sheet: 2, sheets: 1}
	putSubtreeStyleStore(prev, tag, store)

	if len(prev.storeChunks) != maxSubtreeStyleStoreChunks+1 {
		t.Fatalf("prev.storeChunks len = %d, want %d", len(prev.storeChunks), maxSubtreeStyleStoreChunks+1)
	}

	if len(store.chunks) != 0 {
		t.Fatalf("store.chunks len = %d, want 0 after transfer", len(store.chunks))
	}
}
