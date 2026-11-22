package layout

import "sync"

// cloneOpsSpareDivisor and cloneOpsSpareFloor size the spare capacity a cloned
// display list keeps for the window splices a patch commits. One click patches
// a handful of ids against thousands of operations, so len/16 plus a small
// floor covers those windows without pinning another page-sized backing.
const (
	cloneOpsSpareDivisor = 16
	cloneOpsSpareFloor   = 16
	maxPooledOpSliceCap  = 8192
)

//nolint:exhaustruct,gochecknoglobals // pools deliberately have no New so cold rents allocate exact size
var (
	opSlicePool    = sync.Pool{}
	opSliceHdrPool = sync.Pool{}
)

func rentOpSlice(length, capacity int) []Op {
	if capacity <= maxPooledOpSliceCap {
		if hdr, ok := opSlicePool.Get().(*[]Op); ok {
			scratch := *hdr
			*hdr = nil

			opSliceHdrPool.Put(hdr)

			if cap(scratch) >= capacity && cap(scratch) <= capacity*2 {
				return scratch[:length]
			}
		}
	}

	return make([]Op, length, capacity)
}

func returnOpSlice(scratch []Op) {
	if cap(scratch) == 0 || cap(scratch) > maxPooledOpSliceCap {
		return
	}

	clear(scratch[:cap(scratch)])
	scratch = scratch[:0]

	header, ok := opSliceHdrPool.Get().(*[]Op)
	if !ok {
		header = new([]Op)
	}

	*header = scratch
	opSlicePool.Put(header)
}

// cloneOps returns an independent copy of one operation list with room for the
// splices a patch pass commits.
func cloneOps(src []Op) []Op {
	if src == nil {
		return nil
	}

	capacity := len(src) + len(src)/cloneOpsSpareDivisor + cloneOpsSpareFloor
	dst := rentOpSlice(len(src), capacity)
	copy(dst, src)

	// Every payload the clone must own comes out of one slab. A page of
	// thousands of operations then pays one allocation for the payloads
	// instead of one per operation, and the collector scans one object.
	slab := make([]opExtra, countOwnedExtras(src))
	next := 0

	for index := range src {
		extra := src[index].opExtra
		if extra == nil || extra == emptyExtra {
			continue
		}

		cloneOpExtraInto(&slab[next], extra)
		dst[index].opExtra = &slab[next]
		next++
	}

	return dst
}

// countOwnedExtras counts the operations whose payload is a real, unshared
// extra: the ones cloneOps must copy. A nil payload and the shared empty
// singleton stay as they are.
func countOwnedExtras(ops []Op) int {
	count := 0

	for i := range ops {
		if extra := ops[i].opExtra; extra != nil && extra != emptyExtra {
			count++
		}
	}

	return count
}
