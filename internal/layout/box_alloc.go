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

// sliceHolder boxes one backing so a sync.Pool stores a pointer-like value. A
// slice header stored directly would box a fresh header on every Put.
type sliceHolder[T any] struct {
	slice []T
}

// holderPool recycles sliceHolder values with their backings. take hands the
// backing to the caller and parks the empty holder, and put reuses a parked
// holder, so a rent/return cycle allocates nothing once warm.
type holderPool[T any] struct {
	stored  sync.Pool // *sliceHolder[T] with a backing
	holders sync.Pool // *sliceHolder[T] without a backing
}

// take returns a stored backing with cap exactly size.
func (p *holderPool[T]) take(size int) ([]T, bool) {
	holder, ok := p.takeHolder()
	if !ok || cap(holder.slice) != size {
		return nil, false
	}

	slice := holder.slice
	holder.slice = nil
	p.holders.Put(holder)

	return slice, true
}

// takeAtLeast returns a stored backing with cap at least size.
func (p *holderPool[T]) takeAtLeast(size int) ([]T, bool) {
	holder, ok := p.takeHolder()
	if !ok || cap(holder.slice) < size {
		return nil, false
	}

	slice := holder.slice
	holder.slice = nil
	p.holders.Put(holder)

	return slice, true
}

func (p *holderPool[T]) takeHolder() (*sliceHolder[T], bool) {
	raw := p.stored.Get()
	if raw == nil {
		return nil, false
	}

	holder, ok := raw.(*sliceHolder[T])
	if !ok || holder == nil {
		return nil, false
	}

	return holder, true
}

func (p *holderPool[T]) put(slice []T) {
	holder := p.holder()
	holder.slice = slice
	p.stored.Put(holder)
}

func (p *holderPool[T]) holder() *sliceHolder[T] {
	raw := p.holders.Get()
	if raw == nil {
		return new(sliceHolder[T])
	}

	holder, ok := raw.(*sliceHolder[T])
	if !ok || holder == nil {
		return new(sliceHolder[T])
	}

	return holder
}

// exactSlicePool retains slices keyed by capacity so a rent for size reuses a
// backing with cap size and nothing else. Each capacity has its own holder
// pool; the outer pool only tracks how many slices are retained and drops
// returns once any bound is hit.
type exactSlicePool[T any] struct {
	mu       sync.Mutex
	buckets  map[int]*holderPool[T]
	retained atomic.Int64
}

// rent returns a zeroed slice with len size and cap size. Sizes at or below
// zero return nil. Sizes above maxBoxAllocSliceCap allocate directly and
// bypass the pool so one outlier cannot pin a huge backing.
func (p *exactSlicePool[T]) rent(size int) []T {
	if size <= 0 {
		return nil
	}

	if size > maxBoxAllocSliceCap {
		return make([]T, size)
	}

	if slice, ok := p.take(size); ok {
		slice = slice[:size]
		clear(slice)

		return slice
	}

	return make([]T, size)
}

// take removes one retained backing for size and reports whether it had one.
func (p *exactSlicePool[T]) take(size int) ([]T, bool) {
	p.mu.Lock()
	bucket := p.buckets[size]
	p.mu.Unlock()

	if bucket == nil {
		return nil, false
	}

	slice, ok := bucket.take(size)
	if !ok {
		return nil, false
	}

	p.retained.Add(-1)

	return slice, true
}

// put stores slice for bucket capacity. The caller must have cleared slice and
// sliced it to zero length already.
func (p *exactSlicePool[T]) put(capacity int, slice []T) {
	if capacity <= 0 || capacity > maxBoxAllocSliceCap {
		return
	}

	if p.retained.Load() >= maxBoxAllocRetained {
		return
	}

	p.mu.Lock()

	if p.buckets == nil {
		p.buckets = make(map[int]*holderPool[T])
	}

	bucket := p.buckets[capacity]
	if bucket == nil {
		if len(p.buckets) >= maxBoxAllocBuckets {
			p.mu.Unlock()

			return
		}

		bucket = new(holderPool[T])
		p.buckets[capacity] = bucket
	}
	p.mu.Unlock()

	bucket.put(slice)
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
func (p *exactSlicePool[T]) returnSlice(slice []T) {
	if slice == nil {
		return
	}

	capacity := cap(slice)
	if capacity <= 0 || capacity > maxBoxAllocSliceCap {
		return
	}

	// Clear the whole backing, not just the live length, so a slice returned
	// with len < cap cannot keep appended pointers alive in the pool.
	full := slice[:capacity]
	clear(full)
	p.put(capacity, full[:0])
}

//nolint:gochecknoglobals // package-level pools are the point of this file
var (
	boxPtrSlicePool exactSlicePool[*box]
	floatSlicePool  exactSlicePool[float64]
	uint64SlicePool exactSlicePool[uint64]
	boxRowSlicePool exactSlicePool[[]*box]
	// opIDSlicePool and placedSlicePool belong to the adopt contract below
	// and are reported by boxAllocRetained; no layout path rents them yet.
	opIDSlicePool   exactSlicePool[uint64]
	placedSlicePool exactSlicePool[PlacedElement]
)

//nolint:gochecknoglobals // package-level struct pools and immutable zero sentinels
var (
	boxStructPool = sync.Pool{
		New: func() any { return new(box) },
	}
	clipRectPool = sync.Pool{
		New: func() any { return new(clipRect) },
	}
	zeroBoxValue      box
	zeroClipRectValue clipRect
)

//nolint:gochecknoglobals // retained counters bound the struct pools
var (
	retainedBoxes atomic.Int64
	retainedClips atomic.Int64
)

// rentBox returns a zeroed box struct. The caller owns it until returnBox.
func rentBox() *box {
	if value, ok := boxStructPool.Get().(*box); ok && value != nil {
		retainedBoxes.Add(-1)

		*value = zeroBoxValue

		return value
	}

	return new(box)
}

// returnBox zeroes target and hands it back when the pool has room. A retained
// alias observes the zero value after this call, so callers must drop target.
func returnBox(target *box) {
	if target == nil {
		return
	}

	*target = zeroBoxValue

	if retainedBoxes.Load() >= maxBoxAllocBoxes {
		return
	}

	boxStructPool.Put(target)
	retainedBoxes.Add(1)
}

// rentClipRect returns a zeroed clip rect. The caller owns it until return.
func rentClipRect() *clipRect {
	if value, ok := clipRectPool.Get().(*clipRect); ok && value != nil {
		retainedClips.Add(-1)

		*value = zeroClipRectValue

		return value
	}

	return new(clipRect)
}

// returnClipRect zeroes rect and hands it back when the pool has room.
func returnClipRect(rect *clipRect) {
	if rect == nil {
		return
	}

	*rect = zeroClipRectValue

	if retainedClips.Load() >= maxBoxAllocClips {
		return
	}

	clipRectPool.Put(rect)
	retainedClips.Add(1)
}

// rentBoxPtrs returns len size with cap exactly size for children lists,
// flattened box lists, and table cell rows.
func rentBoxPtrs(size int) []*box {
	return boxPtrSlicePool.rent(size)
}

// returnBoxPtrs clears boxes and retains its backing when the bounds allow.
// Callers must not retain boxes after this call.
func returnBoxPtrs(boxes []*box) {
	boxPtrSlicePool.returnSlice(boxes)
}

// rentFloats returns len size with cap exactly size for rowTops, rowHeights,
// and table measure work slices.
func rentFloats(size int) []float64 {
	return floatSlicePool.rent(size)
}

// returnFloats clears values and retains their backing when the bounds allow.
// Callers must not retain values after this call.
func returnFloats(values []float64) {
	floatSlicePool.returnSlice(values)
}

// rentUint64s returns len size with cap exactly size for operation ID lists.
func rentUint64s(size int) []uint64 {
	return uint64SlicePool.rent(size)
}

// returnUint64s clears values and retains their backing when the bounds allow.
// Callers must not retain values after this call.
func returnUint64s(values []uint64) {
	uint64SlicePool.returnSlice(values)
}

// rentBoxRows returns len size with cap exactly size for the rows table on a
// table box. Each row remains nil until the caller fills it.
func rentBoxRows(size int) [][]*box {
	return boxRowSlicePool.rent(size)
}

// returnBoxRows clears rows and retains their backing when the bounds allow.
// It does not return the per-row slices; the caller must return each row
// with returnBoxPtrs first. Callers must not retain rows after this call.
func returnBoxRows(rows [][]*box) {
	boxRowSlicePool.returnSlice(rows)
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
