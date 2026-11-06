package layout

import (
	"sync"
	"sync/atomic"
)

// Box and per-box slice pooling for full-document layouts.
//
// Allocation sites found (all in internal/layout unless noted):
//
//   - layout_chunks.go allocBox appends one box value per element into a
//     64-entry chunk. A full document needs about 9k boxes, so about 141
//     chunks plus one pointer per box in the flattened list.
//   - layout_context.go finalizeResult makes boxes with cap len(boxChunks)*64
//     and layout.go flattenBoxes appends one *box per element into it.
//   - Per-box children slices ([]*box) grow by append in build paths such as
//     layout_flow.go, multicol.go, flex_logical_axes.go, flex_columns.go,
//     grid.go, grid_single.go, grid_auto_row.go, grid_masonry.go,
//     inline_paint.go, layout_svg.go, and layout_chrome.go.
//   - Per-table slices in layout_tables.go: tableBox.children, cellData
//     ([][]*box), cellData rows ([]*box), rowTops and rowHeights ([]float64),
//     plus colW, colMin, colPct, colAbs work slices.
//   - Box graph copies allocate one struct plus fresh children and row slices
//     per box in layout.go cloneBoxGraph and relayout_subtree.go rebaseBox.
//   - overflow_clip.go computeBoxOverflowClip assigns one *clipRect per box
//     that clips descendants.
//   - placed.go WalkPlaced makes one allIDs ([]uint64) backing per walk and
//     hands each element a subslice as OperationIDs; PlacedElements makes one
//     []PlacedElement per call. The public wrapper in layout/layout.go
//     boxesFrom makes one []Box per call plus one Animations, Transitions,
//     and Timelines slice per box that declares them.
//
// Adopt contract for the future call-site change (not wired yet):
//
//   - The layout that owns the boxes rents; the same layout run returns them.
//     The natural pair is finalizeResult (or Result.Release) renting the
//     flattened boxes slice and the per-box children, row, float, and ID
//     slices, and returning them when the Result is released. allocBox keeps
//     owning box structs in chunks; a second step can route rebuildBox,
//     cloneBoxGraph, and rebaseBox through rentBox and returnBox.
//   - Rents have exact capacity: rentBoxPtrs, rentFloats, rentUint64s, and
//     rentBoxRows return len n with cap exactly n. A caller that appends may
//     use the result sliced to zero length and must not append past n.
//   - Returned values must not be retained. Return clears the backing so a
//     retained alias observes zeros and the garbage collector can free the
//     elements. Callers must drop the slice header and, for structs, the
//     pointer after return.
//   - Only the pool may be shared across goroutines. A rented slice or box is
//     owned by the renting goroutine until it is returned. Do not return the
//     same value twice and do not use a value after returning it.
//   - Oversized values bypass the pool: rents above maxBoxAllocSliceCap
//     allocate directly and returns with larger capacity are dropped, so one
//     huge table cannot pin memory for later small documents.

const (
	// maxBoxAllocSliceCap bounds one pooled backing. Larger rents allocate
	// directly and larger returns are dropped.
	maxBoxAllocSliceCap = 16384
	// maxBoxAllocRetained bounds retained slices per typed pool.
	maxBoxAllocRetained = 2048
	// maxBoxAllocBuckets bounds distinct capacities per typed pool.
	maxBoxAllocBuckets = 64
	// maxBoxAllocBoxes bounds retained *box structs.
	maxBoxAllocBoxes = 16384
	// maxBoxAllocClips bounds retained *clipRect structs.
	maxBoxAllocClips = 4096
)

// exactSlicePool retains slices keyed by capacity so a rent for n reuses a
// backing with cap n and nothing else. Each capacity has its own sync.Pool;
// the outer pool only tracks how many slices are retained and drops returns
// once any bound is hit.
type exactSlicePool[T any] struct {
	mu       sync.Mutex
	buckets  map[int]*sync.Pool
	retained atomic.Int64
}

// rent returns a zeroed slice with len n and cap n. Sizes at or below zero
// return nil. Sizes above maxBoxAllocSliceCap allocate directly and bypass
// the pool so one outlier cannot pin a huge backing.
func (p *exactSlicePool[T]) rent(n int) []T {
	if n <= 0 {
		return nil
	}

	if n > maxBoxAllocSliceCap {
		return make([]T, n)
	}

	p.mu.Lock()

	var bucket *sync.Pool
	if p.buckets != nil {
		bucket = p.buckets[n]
	}

	p.mu.Unlock()

	if bucket != nil {
		if v, ok := bucket.Get().([]T); ok && v != nil {
			p.retained.Add(-1)
			// Buckets only hold backings stored with cap n, but a pooled
			// value may have any length up to n. Re-slice before use.
			v = v[:cap(v)]
			if len(v) != n {
				// Defensive: a foreign backing must never surface with the
				// wrong length. Drop it and allocate the exact size.
				if cap(v) == n {
					v = v[:n]
					clear(v)

					return v
				}

				p.put(cap(v), v[:0])

				return make([]T, n)
			}

			v = v[:n]
			clear(v)

			return v
		}
	}

	return make([]T, n)
}

// put stores s for bucket c. The caller must have cleared s and sliced it to
// zero length already.
func (p *exactSlicePool[T]) put(c int, s []T) {
	if c <= 0 || c > maxBoxAllocSliceCap {
		return
	}

	if p.retained.Load() >= maxBoxAllocRetained {
		return
	}

	p.mu.Lock()

	if p.buckets == nil {
		p.buckets = make(map[int]*sync.Pool)
	}

	bucket, ok := p.buckets[c]
	if !ok {
		if len(p.buckets) >= maxBoxAllocBuckets {
			p.mu.Unlock()

			return
		}
		bucket = &sync.Pool{}
		p.buckets[c] = bucket
	}

	p.mu.Unlock()

	bucket.Put(s)
	p.retained.Add(1)
}

// classes counts retained slices. Tests use it to prove boundedness.
func (p *exactSlicePool[T]) classes() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.buckets)
}

func (p *exactSlicePool[T]) count() int {
	return int(p.retained.Load())
}

// returnSlice clears the full backing and retains the zero-length slice when
// the bounds allow. Values with capacity outside (0, maxBoxAllocSliceCap]
// are dropped.
func (p *exactSlicePool[T]) returnSlice(s []T) {
	if s == nil {
		return
	}

	c := cap(s)
	if c <= 0 || c > maxBoxAllocSliceCap {
		return
	}

	// Clear the whole backing, not just the live length, so a slice returned
	// with len < cap cannot keep appended pointers alive in the pool.
	full := s[:cap(s)]
	clear(full)
	p.put(c, full[:0])
}

//nolint:gochecknoglobals // package-level pools are the point of this file
var (
	boxPtrSlicePool exactSlicePool[*box]
	floatSlicePool  exactSlicePool[float64]
	uint64SlicePool exactSlicePool[uint64]
	boxRowSlicePool exactSlicePool[[]*box]
	opIDSlicePool   exactSlicePool[uint64]
	placedSlicePool exactSlicePool[PlacedElement]
)

//nolint:gochecknoglobals // package-level struct pools match the slice pools
var (
	boxStructPool = sync.Pool{
		New: func() any { return &box{} },
	}
	clipRectPool = sync.Pool{
		New: func() any { return &clipRect{} },
	}
)

//nolint:gochecknoglobals // retained counters bound the struct pools
var (
	retainedBoxes atomic.Int64
	retainedClips atomic.Int64
)

// rentBox returns a zeroed box struct. The caller owns it until returnBox.
func rentBox() *box {
	if v, ok := boxStructPool.Get().(*box); ok && v != nil {
		retainedBoxes.Add(-1)
		*v = box{}

		return v
	}

	return &box{}
}

// returnBox zeroes b and hands it back when the pool has room. A retained
// alias observes the zero value after this call, so callers must drop b.
func returnBox(b *box) {
	if b == nil {
		return
	}

	*b = box{}

	if retainedBoxes.Load() >= maxBoxAllocBoxes {
		return
	}

	boxStructPool.Put(b)
	retainedBoxes.Add(1)
}

// rentClipRect returns a zeroed clip rect. The caller owns it until return.
func rentClipRect() *clipRect {
	if v, ok := clipRectPool.Get().(*clipRect); ok && v != nil {
		retainedClips.Add(-1)
		*v = clipRect{}

		return v
	}

	return &clipRect{}
}

// returnClipRect zeroes c and hands it back when the pool has room.
func returnClipRect(c *clipRect) {
	if c == nil {
		return
	}

	*c = clipRect{}

	if retainedClips.Load() >= maxBoxAllocClips {
		return
	}

	clipRectPool.Put(c)
	retainedClips.Add(1)
}

// rentBoxPtrs returns len n with cap exactly n for children lists, flattened
// box lists, and table cell rows.
func rentBoxPtrs(n int) []*box {
	return boxPtrSlicePool.rent(n)
}

// returnBoxPtrs clears s and retains its backing when the bounds allow.
// Callers must not retain s after this call.
func returnBoxPtrs(s []*box) {
	boxPtrSlicePool.returnSlice(s)
}

// rentFloats returns len n with cap exactly n for rowTops, rowHeights, and
// table measure work slices.
func rentFloats(n int) []float64 {
	return floatSlicePool.rent(n)
}

// returnFloats clears s and retains its backing when the bounds allow.
// Callers must not retain s after this call.
func returnFloats(s []float64) {
	floatSlicePool.returnSlice(s)
}

// rentUint64s returns len n with cap exactly n for operation ID lists.
func rentUint64s(n int) []uint64 {
	return uint64SlicePool.rent(n)
}

// returnUint64s clears s and retains its backing when the bounds allow.
// Callers must not retain s after this call.
func returnUint64s(s []uint64) {
	uint64SlicePool.returnSlice(s)
}

// rentBoxRows returns len n with cap exactly n for the rows table on a table
// box. Each row remains nil until the caller fills it.
func rentBoxRows(n int) [][]*box {
	return boxRowSlicePool.rent(n)
}

// returnBoxRows clears s and retains its backing when the bounds allow.
// It does not return the per-row slices; the caller must return each row
// with returnBoxPtrs first. Callers must not retain s after this call.
func returnBoxRows(s [][]*box) {
	boxRowSlicePool.returnSlice(s)
}

// rentOpIDs returns len n with cap exactly n for one element operation range.
// It has its own pool so table ID traffic cannot evict box child backings.
func rentOpIDs(n int) []uint64 {
	return opIDSlicePool.rent(n)
}

// returnOpIDs clears s and retains its backing when the bounds allow.
func returnOpIDs(s []uint64) {
	opIDSlicePool.returnSlice(s)
}

// rentPlaced returns len n with cap exactly n for PlacedElements results.
func rentPlaced(n int) []PlacedElement {
	return placedSlicePool.rent(n)
}

// returnPlaced clears s and retains its backing when the bounds allow.
// Clearing drops per-element OperationIDs references, so callers that keep
// those subslices alive must copy them before returning the outer slice.
func returnPlaced(s []PlacedElement) {
	placedSlicePool.returnSlice(s)
}

// boxAllocStats reports retained pool sizes for tests and future metrics.
type boxAllocStats struct {
	BoxPtrSlices int
	FloatSlices  int
	Uint64Slices int
	BoxRows      int
	OpIDs        int
	Placed       int
	Boxes        int
	Clips        int
	BoxClasses   int
}

// boxAllocRetained returns the current retained counts.
func boxAllocRetained() boxAllocStats {
	return boxAllocStats{
		BoxPtrSlices: boxPtrSlicePool.count(),
		FloatSlices:  floatSlicePool.count(),
		Uint64Slices: uint64SlicePool.count(),
		BoxRows:      boxRowSlicePool.count(),
		OpIDs:        opIDSlicePool.count(),
		Placed:       placedSlicePool.count(),
		Boxes:        int(retainedBoxes.Load()),
		Clips:        int(retainedClips.Load()),
		BoxClasses:   boxPtrSlicePool.classes(),
	}
}
