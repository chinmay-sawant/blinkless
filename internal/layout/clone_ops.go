package layout

// cloneOpsSpareDivisor and cloneOpsSpareFloor size the spare capacity a cloned
// display list keeps for the window splices a patch commits. One click patches
// a handful of ids against thousands of operations, so len/16 plus a small
// floor covers those windows without pinning another page-sized backing.
const (
	cloneOpsSpareDivisor = 16
	cloneOpsSpareFloor   = 16
)

// cloneOps returns an independent copy of one operation list with room for the
// splices a patch pass commits.
func cloneOps(src []Op) []Op {
	if src == nil {
		return nil
	}

	dst := make([]Op, len(src), len(src)+len(src)/cloneOpsSpareDivisor+cloneOpsSpareFloor)
	for i := range src {
		dst[i] = src[i]
		dst[i].opExtra = cloneOpExtra(src[i].opExtra)
	}

	return dst
}
