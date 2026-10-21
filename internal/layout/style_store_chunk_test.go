package layout

import "testing"

// TestStyleStoreChunksGrowAndKeepPointers checks that the style store keeps
// its chunks small at first and that the pointers it handed out stay valid
// when a later chunk is added.
func TestStyleStoreChunksGrowAndKeepPointers(t *testing.T) {
	store := &styleStore{}

	const count = 40

	pointers := make([]*ResolvedStyle, 0, count)
	for i := range count {
		style := ResolvedStyle{Color: [3]float64{float64(i), 0, 0}}
		pointers = append(pointers, store.append(style))
	}

	if len(store.chunks) < 2 {
		t.Fatalf("chunks = %d, want more than one for %d styles", len(store.chunks), count)
	}

	for i, ptr := range pointers {
		if ptr == nil {
			t.Fatalf("style %d has no record", i)
		}

		if got := ptr.Color[0]; got != float64(i) {
			t.Fatalf("style %d reads %v, want %v", i, got, float64(i))
		}
	}

	first := cap(store.chunks[0])
	if first != styleStoreFirstChunk {
		t.Fatalf("the first chunk holds %d records, want %d", first, styleStoreFirstChunk)
	}

	for i, chunk := range store.chunks {
		if cap(chunk) > styleStoreChunkSize {
			t.Fatalf("chunk %d holds %d records, want at most %d", i, cap(chunk), styleStoreChunkSize)
		}
	}
}

// TestStyleStoreSharesDuplicateStyles checks that the store still interns an
// exact duplicate, so a page with repeated shapes keeps one record.
func TestStyleStoreSharesDuplicateStyles(t *testing.T) {
	store := &styleStore{}

	first := store.append(ResolvedStyle{Color: [3]float64{7, 0, 0}})
	second := store.append(ResolvedStyle{Color: [3]float64{7, 0, 0}})

	if first != second {
		t.Fatal("the store kept two records for one style")
	}
}
