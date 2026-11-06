package layout

import (
	"sync"
	"testing"
)

func TestBoxAllocExactCapacity(t *testing.T) {
	t.Parallel()

	sizes := []int{0, 1, 7, 64, 1024}
	for _, n := range sizes {
		if got := rentBoxPtrs(n); n == 0 {
			if got != nil {
				t.Fatalf("rentBoxPtrs(0) = len %d, want nil", len(got))
			}
		} else {
			if len(got) != n || cap(got) != n {
				t.Fatalf("rentBoxPtrs(%d) = len %d cap %d, want exact", n, len(got), cap(got))
			}
			returnBoxPtrs(got)
		}

		if got := rentFloats(n); n == 0 {
			if got != nil {
				t.Fatalf("rentFloats(0) = len %d, want nil", len(got))
			}
		} else {
			if len(got) != n || cap(got) != n {
				t.Fatalf("rentFloats(%d) = len %d cap %d, want exact", n, len(got), cap(got))
			}
			returnFloats(got)
		}

		if got := rentUint64s(n); n == 0 {
			if got != nil {
				t.Fatalf("rentUint64s(0) = len %d, want nil", len(got))
			}
		} else {
			if len(got) != n || cap(got) != n {
				t.Fatalf("rentUint64s(%d) = len %d cap %d, want exact", n, len(got), cap(got))
			}
			returnUint64s(got)
		}

		if got := rentBoxRows(n); n == 0 {
			if got != nil {
				t.Fatalf("rentBoxRows(0) = len %d, want nil", len(got))
			}
		} else {
			if len(got) != n || cap(got) != n {
				t.Fatalf("rentBoxRows(%d) = len %d cap %d, want exact", n, len(got), cap(got))
			}
			returnBoxRows(got)
		}
	}

	// Oversized rents bypass the pool but keep exact shape.
	huge := maxBoxAllocSliceCap + 8
	if got := rentFloats(huge); len(got) != huge || cap(got) != huge {
		t.Fatalf("rentFloats(huge) = len %d cap %d, want %d", len(got), cap(got), huge)
	}
}

func TestBoxAllocRentReturnZeroed(t *testing.T) {
	t.Parallel()

	b := rentBox()
	b.x = 3
	b.y = 4
	b.w = 5
	b.height = 6
	b.opStart = 1
	b.opEnd = 2
	b.kind = 1
	b.children = []*box{{x: 1}}
	b.rows = [][]*box{{{x: 2}}}
	b.rowTops = []float64{1}
	b.rowHeights = []float64{2}
	returnBox(b)

	got := rentBox()
	if got.x != 0 || got.y != 0 || got.w != 0 || got.height != 0 {
		t.Fatalf("reused box kept geometry %+v", got)
	}

	if got.opStart != 0 || got.opEnd != 0 || got.kind != 0 {
		t.Fatalf("reused box kept range or kind %+v", got)
	}

	if got.children != nil || got.rows != nil || got.rowTops != nil || got.rowHeights != nil {
		t.Fatalf("reused box kept slices %+v", got)
	}
	returnBox(got)

	const n = 17
	f := rentFloats(n)
	for i := range f {
		f[i] = float64(i + 1)
	}

	returnFloats(f)

	f2 := rentFloats(n)
	if len(f2) != n || cap(f2) != n {
		t.Fatalf("reused floats len %d cap %d, want %d", len(f2), cap(f2), n)
	}

	for i, v := range f2 {
		if v != 0 {
			t.Fatalf("reused floats[%d] = %v, want 0", i, v)
		}
	}

	returnFloats(f2)

	u := rentUint64s(n)
	for i := range u {
		u[i] = uint64(i + 1)
	}

	returnUint64s(u)

	u2 := rentUint64s(n)
	for i, v := range u2 {
		if v != 0 {
			t.Fatalf("reused uint64s[%d] = %v, want 0", i, v)
		}
	}

	returnUint64s(u2)
}

func TestBoxPoolBoundedness(t *testing.T) {
	t.Parallel()

	// Local pool keeps this test from filling the shared pools.
	var p exactSlicePool[float64]

	const n = 13
	for range maxBoxAllocRetained + 512 {
		s := make([]float64, n)
		for i := range s {
			s[i] = 1
		}
		p.returnSlice(s)
	}

	if got := p.count(); got > maxBoxAllocRetained {
		t.Fatalf("retained %d slices, want at most %d", got, maxBoxAllocRetained)
	}

	before := p.count()
	p.returnSlice(make([]float64, maxBoxAllocSliceCap+16))
	if got := p.count(); got != before {
		t.Fatalf("oversized return changed retained %d to %d", before, got)
	}

	// Distinct capacities are bounded too.
	var q exactSlicePool[float64]
	for i := 1; i <= maxBoxAllocBuckets+16; i++ {
		q.returnSlice(make([]float64, i))
	}

	if got := q.classes(); got > maxBoxAllocBuckets {
		t.Fatalf("retained %d classes, want at most %d", got, maxBoxAllocBuckets)
	}

	// Struct pools drop beyond their caps without growing forever.
	var rp exactSlicePool[*box]
	for range maxBoxAllocRetained + 64 {
		rp.returnSlice(make([]*box, 3))
	}

	if got := rp.count(); got > maxBoxAllocRetained {
		t.Fatalf("retained %d box slices, want at most %d", got, maxBoxAllocRetained)
	}
}

func TestBoxAllocConcurrent(t *testing.T) {
	t.Parallel()

	const workers = 8
	const iters = 200

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				b := rentBox()
				b.x = float64(i)
				b.w = float64(i)
				returnBox(b)

				c := rentClipRect()
				c.w = float64(i)
				returnClipRect(c)

				bp := rentBoxPtrs(9)
				if len(bp) != 9 || cap(bp) != 9 {
					t.Errorf("rentBoxPtrs = len %d cap %d, want 9", len(bp), cap(bp))
				}
				returnBoxPtrs(bp)

				f := rentFloats(11)
				if len(f) != 11 || cap(f) != 11 {
					t.Errorf("rentFloats = len %d cap %d, want 11", len(f), cap(f))
				}
				returnFloats(f)

				u := rentUint64s(5)
				if len(u) != 5 || cap(u) != 5 {
					t.Errorf("rentUint64s = len %d cap %d, want 5", len(u), cap(u))
				}
				returnUint64s(u)

				r := rentBoxRows(4)
				if len(r) != 4 || cap(r) != 4 {
					t.Errorf("rentBoxRows = len %d cap %d, want 4", len(r), cap(r))
				}
				returnBoxRows(r)
			}
		}()
	}
	wg.Wait()
}

func TestBoxPoolNoRetainedReferences(t *testing.T) {
	t.Parallel()

	const n = 6
	s := rentBoxPtrs(n)
	for i := range s {
		s[i] = &box{x: float64(i + 1)}
	}

	alias := s
	returnBoxPtrs(s)

	for i, v := range alias {
		if v != nil {
			t.Fatalf("returned box slice alias[%d] still points to %+v", i, v)
		}
	}

	// A fresh rent over the same capacity must also be zeroed.
	next := rentBoxPtrs(n)
	for i, v := range next {
		if v != nil {
			t.Fatalf("reused box slice [%d] = %+v, want nil", i, v)
		}
	}

	returnBoxPtrs(next)

	b := rentBox()
	inner := rentBoxPtrs(2)
	inner[0] = &box{x: 1}
	b.children = inner
	aliasBox := b
	returnBox(b)

	if aliasBox.children != nil {
		t.Fatalf("returned box alias kept children %v", aliasBox.children)
	}

	if aliasBox.x != 0 {
		t.Fatalf("returned box alias kept x=%v", aliasBox.x)
	}
	returnBoxPtrs(inner)

	f := rentFloats(4)
	aliasFloat := f
	f[0] = 42
	returnFloats(f)

	if aliasFloat[0] != 0 {
		t.Fatalf("returned float alias kept %v", aliasFloat[0])
	}
}
