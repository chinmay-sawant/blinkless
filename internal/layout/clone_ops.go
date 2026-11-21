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

	for i := range src {
		if src[i].opExtra != nil {
			dst[i].opExtra = cloneOpExtra(src[i].opExtra)
		}
	}

	return dst
}
