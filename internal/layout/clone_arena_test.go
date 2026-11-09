package layout

import "testing"

// cloneArenaFixture builds one root with many children. It is the shape a
// patch clone walks, with enough boxes that one allocation per box would show
// up in the allocation count.
func cloneArenaFixture(count int) *Result {
	root := &box{kind: boxKindBlock, w: 100, height: 1000}
	boxes := make([]*box, 0, count)
	boxes = append(boxes, root)

	for i := 1; i < count; i++ {
		child := &box{kind: boxKindBlock, y: float64(i), w: 100, height: 1}
		root.children = append(root.children, child)
		boxes = append(boxes, child)
	}

	return &Result{
		Ops:     make([]Op, count),
		root:    root,
		rootBox: root,
		boxes:   boxes,
	}
}

// TestCloneResultArenaCopiesEveryBox checks that the chunked clone still gives
// every box its own storage and preserves the tree links.
func TestCloneResultArenaCopiesEveryBox(t *testing.T) {
	t.Parallel()

	res := cloneArenaFixture(1000)
	clone := CloneResult(res)

	if clone == nil || clone == res {
		t.Fatal("CloneResult must return a distinct result")
	}
	if len(clone.boxes) != len(res.boxes) {
		t.Fatalf("box count = %d, want %d", len(clone.boxes), len(res.boxes))
	}
	if len(clone.root.children) != len(res.root.children) {
		t.Fatalf("root children = %d, want %d", len(clone.root.children), len(res.root.children))
	}

	for i := range res.boxes {
		got, want := clone.boxes[i], res.boxes[i]
		if got == want {
			t.Fatalf("box %d aliases the source", i)
		}
		if got.x != want.x || got.y != want.y || got.w != want.w || got.height != want.height {
			t.Fatalf("box %d geometry = %v, want %v", i, got, want)
		}
		if len(got.children) != len(want.children) {
			t.Fatalf("box %d children = %d, want %d", i, len(got.children), len(want.children))
		}
	}

	if !PageBreaksEqual(res, clone) {
		t.Fatal("clone changed the page breaks")
	}
}

// TestCloneResultArenaAllocationBudget pins the reason for the arena: a large
// clone pays for chunks and slices, not one heap object per box.
func TestCloneResultArenaAllocationBudget(t *testing.T) { //nolint:paralleltest // reads process-wide allocation stats
	res := cloneArenaFixture(1000)

	allocs := testing.AllocsPerRun(5, func() {
		_ = CloneResult(res)
	})
	if allocs > 64 {
		t.Fatalf("CloneResult allocated %.0f objects for 1000 boxes, want at most 64", allocs)
	}
}
