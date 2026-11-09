package layout

import (
	"testing"
	"unsafe"
)

// TestStyleChunkFreeListBounded checks that the free list retains at most its
// limit, reuses cleared chunks, and drops chunks larger than the default.
func TestStyleChunkFreeListBounded(t *testing.T) {
	t.Parallel()

	const limit = 4

	free := newStyleChunkFreeList(limit)

	// Many more returns than the bound: the extra chunks must be dropped.
	for range limit * 512 {
		chunk := make([]ResolvedStyle, 0, defaultStyleChunkSize)
		free.put(&chunk)
	}

	if got := free.retained(); got != limit {
		t.Fatalf("retained %d chunks, want %d", got, limit)
	}

	for range limit {
		chunk := free.get()
		if chunk == nil {
			t.Fatal("get returned a nil chunk")
		}

		if len(*chunk) != 0 {
			t.Fatalf("reused chunk has len %d, want 0", len(*chunk))
		}

		if cap(*chunk) != defaultStyleChunkSize {
			t.Fatalf("reused chunk has cap %d, want %d", cap(*chunk), defaultStyleChunkSize)
		}
	}

	if got := free.retained(); got != 0 {
		t.Fatalf("retained %d chunks after draining, want 0", got)
	}

	// An oversized chunk never enters the free list.
	big := make([]ResolvedStyle, 0, defaultStyleChunkSize+1)
	free.put(&big)

	if got := free.retained(); got != 0 {
		t.Fatalf("retained %d chunks after an oversized put, want 0", got)
	}

	if got := cap(*free.get()); got != defaultStyleChunkSize {
		t.Fatalf("new chunk has cap %d, want %d", got, defaultStyleChunkSize)
	}
}

// TestStyleChunkFreeListClearsReusedChunk checks that put zeroes the live
// records, so a retained chunk cannot keep strings, slices, or style payloads
// alive through the pooled backing.
func TestStyleChunkFreeListClearsReusedChunk(t *testing.T) {
	t.Parallel()

	free := newStyleChunkFreeList(1)

	chunk := make([]ResolvedStyle, 0, defaultStyleChunkSize)
	chunk = append(chunk, ResolvedStyle{Color: [3]float64{1, 2, 3}})
	backing := chunk

	free.put(&chunk)

	if got := backing[0].Color; got != [3]float64{} {
		t.Fatalf("returned record keeps color %v, want the zero value", got)
	}

	reused := free.get()
	if reused != &chunk {
		t.Fatal("get did not reuse the retained chunk")
	}

	if len(*reused) != 0 {
		t.Fatalf("reused chunk has len %d, want 0", len(*reused))
	}
}

// TestStyleChunkPoolMemoryBound keeps the free list under its fixed memory cap.
func TestStyleChunkPoolMemoryBound(t *testing.T) {
	t.Parallel()

	const maxRetainedBytes = 4 << 20

	chunkBytes := int(unsafe.Sizeof(ResolvedStyle{})) * defaultStyleChunkSize
	if got := chunkBytes * cap(styleChunkPool.free); got > maxRetainedBytes {
		t.Fatalf("style chunk pool retains up to %d bytes, want at most %d", got, maxRetainedBytes)
	}

	if defaultStyleChunkSize >= defaultBoxChunkSize {
		t.Fatalf("style chunk holds %d entries, want fewer than the box chunk's %d",
			defaultStyleChunkSize, defaultBoxChunkSize)
	}
}
