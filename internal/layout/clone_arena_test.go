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

// TestCloneResultIsolatesStylesAndChunkOwnership proves that mutating a clone's
// style map does not affect the source result and that releasing both clone and
// source never double-returns shared style or store chunks to pools.
func TestCloneResultIsolatesStylesAndChunkOwnership(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0"><div id="a" style="width:80px;height:20px;background:#eee">a</div></body></html>`)
	if len(res.styles) == 0 {
		t.Fatal("layout produced no styles")
	}

	node := res.boxForID("a").node
	origStyle := res.styles[node]
	if origStyle == nil {
		t.Fatal("missing retained style for #a")
	}

	clone := CloneResult(res)
	if clone.styleChunks != nil || clone.storeChunks != nil {
		t.Fatal("CloneResult must not share source styleChunks or storeChunks ownership")
	}

	replacement := *origStyle
	replacement.BGColor = [4]float64{1, 0, 0, 1}
	clone.styles[node] = &replacement

	if res.styles[node] != origStyle {
		t.Fatal("mutating clone.styles mutated source res.styles")
	}

	// Releasing clone must not clear or return chunks still owned by res.
	clone.Release()

	if res.styles[node].Width != origStyle.Width {
		t.Fatal("releasing clone corrupted source style backing")
	}

	res.Release()
}

// TestCloneResultArenaAllocationBudget pins the reason for the arena: a large
// clone pays for chunks and slices, not one heap object per box.
func TestCloneResultArenaAllocationBudget(t *testing.T) {
	res := cloneArenaFixture(1000)

	allocs := testing.AllocsPerRun(5, func() {
		_ = CloneResult(res)
	})
	if allocs > 64 {
		t.Fatalf("CloneResult allocated %.0f objects for 1000 boxes, want at most 64", allocs)
	}
}

// TestCloneResultRemapsMotionParent checks that a cloned box does not keep a
// pointer into the source result. The source arena is recycled when the
// source is released, so a stale motionParent turns into a data race.
func TestCloneResultRemapsMotionParent(t *testing.T) {
	t.Parallel()

	res := cloneArenaFixture(8)
	for _, child := range res.root.children {
		child.motionParent = res.root
	}

	clone := CloneResult(res)
	if clone == nil || clone.root == res.root {
		t.Fatal("CloneResult must return a distinct tree")
	}

	for i, child := range clone.root.children {
		if child.motionParent != clone.root {
			t.Fatalf("child %d motionParent = %p, want the cloned root %p (source root %p)", i, child.motionParent, clone.root, res.root)
		}
	}
}
