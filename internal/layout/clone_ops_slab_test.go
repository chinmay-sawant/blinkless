package layout

import "testing"

// TestCloneOpsOwnsEveryPayload pins the slab clone: a payload write on the
// clone never reaches the source, the shared empty singleton and nil payloads
// stay as they were, and the image bytes are copied instead of aliased.
func TestCloneOpsOwnsEveryPayload(t *testing.T) {
	t.Parallel()

	src := make([]Op, 6)
	src[0].SetXform(Matrix2D{A: 1, D: 1, E: 7})
	src[1].bindEmptyExtra()
	// src[2] keeps a nil payload.
	src[3].setImage([]byte{1, 2, 3}, 1, 1, "alt")
	src[4].SetOverflowClip(1, 2, 3, 4)
	src[5].setDisplayPath([]DisplayPoint{{X: 1, Y: 2}, {X: 3, Y: 4}}, true)

	dst := cloneOps(src)
	if len(dst) != len(src) {
		t.Fatalf("clone length = %d, want %d", len(dst), len(src))
	}

	if dst[1].opExtra != emptyExtra {
		t.Fatal("the shared empty payload was copied")
	}

	if dst[2].opExtra != nil {
		t.Fatal("a nil payload gained a value")
	}

	for _, i := range []int{0, 3, 4, 5} {
		if dst[i].opExtra == src[i].opExtra {
			t.Fatalf("op %d shares its payload with the source", i)
		}
	}

	dst[0].SetXform(Matrix2D{A: 2, D: 2, E: 99})
	dst[4].SetOverflowClip(9, 9, 9, 9)

	if got := src[0].Transform().E; got != 7 {
		t.Fatalf("source transform moved to E=%v after a clone write", got)
	}

	if x, _, _, _, ok := src[4].OverflowClip(); !ok || x != 1 {
		t.Fatalf("source overflow clip changed to %v", x)
	}

	dst[3].opExtra.Image[0] = 42

	if src[3].opExtra.Image[0] != 1 {
		t.Fatal("the clone aliases the source image bytes")
	}

	dst[5].opExtra.Path.Points[0].X = 99

	if src[5].opExtra.Path.Points[0].X != 1 {
		t.Fatalf("the clone aliases the source display path points: got %v, want 1", src[5].opExtra.Path.Points[0].X)
	}
}

// TestCloneOpsAllocatesPayloadsOnce pins the point of the slab: cloning a
// page of operations that all carry a payload costs a fixed number of
// allocations, not one per operation.
func TestCloneOpsAllocatesPayloadsOnce(t *testing.T) { //nolint:paralleltest // AllocsPerRun is not parallel-safe
	src := make([]Op, 512)
	for i := range src {
		src[i].SetXform(Matrix2D{A: 1, D: 1, E: float64(i)})
	}

	allocs := testing.AllocsPerRun(20, func() {
		returnOpSlice(cloneOps(src))
	})

	if allocs > 4 {
		t.Fatalf("clone of %d payload ops allocated %.0f objects, want a handful", len(src), allocs)
	}
}
