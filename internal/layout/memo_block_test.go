package layout

import (
	"math"
	"sync"
	"testing"
)

// TestMemoBlockKeyStability pins the hit path: storing one measurement and
// looking it up with identical inputs returns the same height without
// growing the memo.
func TestMemoBlockKeyStability(t *testing.T) {
	t.Parallel()

	memo := newBlockIntrinsicMemo()
	memo.Store(11, 22, 200, 42.5)

	height, hit := memo.Lookup(11, 22, 200)
	if !hit {
		t.Fatal("Lookup with identical inputs missed, want hit")
	}

	if height != 42.5 {
		t.Fatalf("Lookup height = %v, want 42.5", height)
	}

	if memo.Len() != 1 {
		t.Fatalf("memo holds %d entries, want 1", memo.Len())
	}

	again, hit := memo.Lookup(11, 22, 200)
	if !hit || again != height {
		t.Fatalf("second Lookup = (%v, %v), want (%v, true)", again, hit, height)
	}

	memo.Store(11, 22, 200, 42.5)

	if memo.Len() != 1 {
		t.Fatalf("re-store grew the memo to %d entries, want 1", memo.Len())
	}
}

// TestMemoBlockInvalidation pins the miss rules: changing any one key
// component (style, content, or width) misses while the stored triple still
// hits.
func TestMemoBlockInvalidation(t *testing.T) {
	t.Parallel()

	memo := newBlockIntrinsicMemo()
	memo.Store(1, 2, 100, 10)

	if _, hit := memo.Lookup(1, 2, 100); !hit {
		t.Fatal("Lookup of stored triple missed, want hit")
	}

	cases := []struct {
		name      string
		styleFP   uint64
		contentFP uint64
		width     float64
	}{
		{name: "style", styleFP: 9, contentFP: 2, width: 100},
		{name: "content", styleFP: 1, contentFP: 3, width: 100},
		{name: "width", styleFP: 1, contentFP: 2, width: 101},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, hit := memo.Lookup(tc.styleFP, tc.contentFP, tc.width); hit {
				t.Fatalf("Lookup after %s change hit, want miss", tc.name)
			}
		})
	}
}

// TestMemoBlockUncacheableWidths pins the width gate: NaN, infinite, and
// negative widths never store and always miss, mirroring gridMeasureBits.
func TestMemoBlockUncacheableWidths(t *testing.T) {
	t.Parallel()

	memo := newBlockIntrinsicMemo()
	widths := []float64{math.NaN(), math.Inf(1), math.Inf(-1), -1}

	for _, width := range widths {
		memo.Store(1, 2, width, 10)

		if _, hit := memo.Lookup(1, 2, width); hit {
			t.Fatalf("Lookup at width %v hit, want miss", width)
		}
	}

	if memo.Len() != 0 {
		t.Fatalf("uncacheable stores left %d entries, want 0", memo.Len())
	}
}

// TestMemoBlockBoundedEviction pins the bound: past the cap the oldest key
// leaves, the newest stays, and re-storing a known key never evicts.
func TestMemoBlockBoundedEviction(t *testing.T) {
	t.Parallel()

	memo := newBlockIntrinsicMemoWithCap(4)

	for content := uint64(0); content < 4; content++ {
		memo.Store(1, content, 100, float64(content)+1)
	}

	if memo.Len() != 4 {
		t.Fatalf("memo holds %d entries, want 4", memo.Len())
	}

	memo.Store(1, 4, 100, 5)

	if memo.Len() != 4 {
		t.Fatalf("memo holds %d entries after eviction, want 4", memo.Len())
	}

	if _, hit := memo.Lookup(1, 0, 100); hit {
		t.Fatal("oldest key still hits after eviction, want miss")
	}

	if height, hit := memo.Lookup(1, 4, 100); !hit || height != 5 {
		t.Fatalf("newest Lookup = (%v, %v), want (5, true)", height, hit)
	}

	memo.Store(1, 1, 100, 11)

	if memo.Len() != 4 {
		t.Fatalf("re-store grew the memo to %d entries, want 4", memo.Len())
	}

	if height, hit := memo.Lookup(1, 1, 100); !hit || height != 11 {
		t.Fatalf("re-stored Lookup = (%v, %v), want (11, true)", height, hit)
	}
}

// TestMemoBlockDefaultBound pins the production cap: the default constructor
// uses blockIntrinsicMemoCap and still evicts past it.
func TestMemoBlockDefaultBound(t *testing.T) {
	t.Parallel()

	if blockIntrinsicMemoCap <= 0 {
		t.Fatalf("blockIntrinsicMemoCap = %d, want positive", blockIntrinsicMemoCap)
	}

	memo := newBlockIntrinsicMemo()

	for content := uint64(0); content < uint64(blockIntrinsicMemoCap)+1; content++ {
		memo.Store(7, content, 120, float64(content)+1)
	}

	if memo.Len() != blockIntrinsicMemoCap {
		t.Fatalf("memo holds %d entries, want cap %d", memo.Len(), blockIntrinsicMemoCap)
	}

	if _, hit := memo.Lookup(7, 0, 120); hit {
		t.Fatal("oldest key still hits past the default cap, want miss")
	}
}

// TestMemoBlockBypassDisablesCache pins the differential-test switch: a
// bypassed memo stores nothing and always misses.
func TestMemoBlockBypassDisablesCache(t *testing.T) {
	t.Parallel()

	memo := newBlockIntrinsicMemo()
	memo.bypass = true
	memo.Store(1, 2, 100, 10)

	if _, hit := memo.Lookup(1, 2, 100); hit {
		t.Fatal("bypassed Lookup hit, want miss")
	}

	if memo.Len() != 0 {
		t.Fatalf("bypassed store left %d entries, want 0", memo.Len())
	}
}

// TestMemoBlockConcurrent exercises the memo from goroutines with disjoint
// key sets. Run with -race: any unsynchronized map access fails the run.
func TestMemoBlockConcurrent(t *testing.T) {
	t.Parallel()

	memo := newBlockIntrinsicMemo()

	const workers = 8
	const perWorker = 25

	var wg sync.WaitGroup

	for worker := uint64(0); worker < workers; worker++ {
		wg.Add(1)

		go func(styleFP uint64) {
			defer wg.Done()

			for i := uint64(0); i < perWorker; i++ {
				contentFP := styleFP*perWorker + i
				memo.Store(styleFP, contentFP, 160, float64(contentFP)+0.5)

				if height, hit := memo.Lookup(styleFP, contentFP, 160); !hit || height != float64(contentFP)+0.5 {
					t.Errorf("worker %d Lookup(%d) = (%v, %v), want miss-free hit", styleFP, contentFP, height, hit)
					return
				}
			}
		}(worker)
	}

	wg.Wait()

	if memo.Len() != workers*perWorker {
		t.Fatalf("memo holds %d entries, want %d", memo.Len(), workers*perWorker)
	}

	for worker := uint64(0); worker < workers; worker++ {
		for i := uint64(0); i < perWorker; i++ {
			contentFP := worker*perWorker + i
			if height, hit := memo.Lookup(worker, contentFP, 160); !hit || height != float64(contentFP)+0.5 {
				t.Fatalf("final Lookup(%d, %d) = (%v, %v), want hit", worker, contentFP, height, hit)
			}
		}
	}
}

// TestMemoBlockClear pins explicit invalidation: Clear drops every entry.
func TestMemoBlockClear(t *testing.T) {
	t.Parallel()

	memo := newBlockIntrinsicMemo()
	memo.Store(1, 2, 100, 10)
	memo.Clear()

	if memo.Len() != 0 {
		t.Fatalf("memo holds %d entries after Clear, want 0", memo.Len())
	}

	if _, hit := memo.Lookup(1, 2, 100); hit {
		t.Fatal("Lookup after Clear hit, want miss")
	}
}
