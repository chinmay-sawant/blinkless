package layout

import (
	"reflect"
	"testing"
)

const (
	spliceTestTotal = 24
	spliceTestStart = 5
)

// spliceTestOps returns a filled operation list whose entries are easy to tell
// apart, so a test can compare a splice against the copy it replaced. capacity
// sizes the backing past the count when a case needs spare room.
func spliceTestOps(count, capacity int, baseID uint64) []Op {
	ops := make([]Op, count, capacity)
	id := baseID

	for index := range ops {
		ops[index] = Op{ID: id, X: float64(index), Kind: OpFillRect}
		id++
	}

	return ops
}

// spliceAllocatingCopy is the allocate-and-copy splice the in-place path
// replaces. A test compares the two so the in-place path cannot drift from the
// readable reference.
func spliceAllocatingCopy(ops []Op, start, oldCount int, newOps []Op) []Op {
	out := make([]Op, 0, len(ops)-oldCount+len(newOps))
	out = append(out, ops[:start]...)
	out = append(out, newOps...)
	out = append(out, ops[start+oldCount:]...)

	return out
}

// TestRelayoutSpliceOpsMatchesTheAllocatingCopy proves the in-place splice
// keeps the operations of the allocating copy for a same-count overwrite, a
// grow, and a shrink, whether the kept list has spare capacity or has to be
// allocated again.
func TestRelayoutSpliceOpsMatchesTheAllocatingCopy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		oldCount int
		newCount int
		spare    int
		inPlace  bool
	}{
		{name: "same count", oldCount: 4, newCount: 4, spare: 0, inPlace: true},
		{name: "grow inside the capacity", oldCount: 4, newCount: 7, spare: 9, inPlace: true},
		{name: "shrink inside the capacity", oldCount: 7, newCount: 3, spare: 9, inPlace: true},
		{name: "shrink at the capacity", oldCount: 7, newCount: 2, spare: 0, inPlace: true},
		{name: "grow past the capacity", oldCount: 4, newCount: 7, spare: 0, inPlace: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			base := spliceTestOps(spliceTestTotal, spliceTestTotal+testCase.spare, 0)
			newOps := spliceTestOps(testCase.newCount, testCase.newCount, spliceTestTotal)
			want := spliceAllocatingCopy(base, spliceTestStart, testCase.oldCount, newOps)

			prev := &Result{Ops: base}
			relayoutSpliceOpsInPlace(prev, spliceTestStart, testCase.oldCount, newOps)

			if !reflect.DeepEqual(prev.Ops, want) {
				t.Fatalf("spliced operations differ from the allocating copy:\n got %+v\nwant %+v", prev.Ops, want)
			}

			if sameBacking := &prev.Ops[0] == &base[0]; sameBacking != testCase.inPlace {
				t.Fatalf("kept the old backing = %v, want %v", sameBacking, testCase.inPlace)
			}

			// A shrink inside the capacity clears the dropped operations so
			// their rare payloads do not stay reachable through the backing.
			if testCase.inPlace && testCase.newCount < testCase.oldCount {
				for index, op := range base[len(prev.Ops):] {
					if !reflect.DeepEqual(op, Op{}) {
						t.Fatalf("dropped operation %d still holds %+v", index, op)
					}
				}
			}
		})
	}
}

// TestRelayoutSpliceOpsSameLengthDoesNotAllocate pins the batch fast path: a
// splice that keeps the operation count overwrites the window and copies
// nothing page-wide. It is deliberately sequential: AllocsPerRun refuses to run
// inside a parallel test.
func TestRelayoutSpliceOpsSameLengthDoesNotAllocate(t *testing.T) { //nolint:paralleltest // AllocsPerRun needs serial
	const (
		total    = 2048
		start    = 100
		newCount = 8
	)

	base := spliceTestOps(total, total, 0)
	newOps := spliceTestOps(newCount, newCount, total)
	prev := &Result{Ops: base}

	allocs := testing.AllocsPerRun(100, func() {
		prev.Ops = base
		relayoutSpliceOpsInPlace(prev, start, newCount, newOps)
	})

	if allocs != 0 {
		t.Fatalf("same-length splice allocated %.1f objects per run, want 0", allocs)
	}

	if &prev.Ops[0] != &base[0] {
		t.Fatal("same-length splice replaced the operation backing")
	}
}

// TestRelayoutSpliceOpsKeepsTheOldBacking pins the single-element contract: the
// allocating splice leaves the operation list the display still holds alone.
func TestRelayoutSpliceOpsKeepsTheOldBacking(t *testing.T) {
	t.Parallel()

	base := spliceTestOps(spliceTestTotal, spliceTestTotal, 0)
	before := spliceTestOps(spliceTestTotal, spliceTestTotal, 0)
	newOps := spliceTestOps(4, 4, spliceTestTotal)

	prev := &Result{Ops: base}
	relayoutSpliceOps(prev, spliceTestStart, 4, newOps)

	if !reflect.DeepEqual(base, before) {
		t.Fatal("the allocating splice changed the operation backing the old display holds")
	}

	if &prev.Ops[0] == &base[0] {
		t.Fatal("the allocating splice reused the old operation backing")
	}
}

// TestCloneResultReservesSpliceCapacity proves a patch clone carries room for
// a batch's windows, so a grow splice runs without reallocating the page list.
func TestCloneResultReservesSpliceCapacity(t *testing.T) {
	t.Parallel()

	const opCount = 512

	res := &Result{Ops: spliceTestOps(opCount, opCount, 0)}
	clone := CloneResult(res)

	if clone == nil {
		t.Fatal("CloneResult returned nil")
	}

	if len(clone.Ops) != opCount {
		t.Fatalf("clone carries %d operations, want %d", len(clone.Ops), opCount)
	}

	wantCap := opCount + opCount/cloneOpsSpareDivisor + cloneOpsSpareFloor
	if cap(clone.Ops) < wantCap {
		t.Fatalf("clone capacity = %d, want at least %d", cap(clone.Ops), wantCap)
	}

	clone.Ops[0].X = -1
	if res.Ops[0].X == -1 {
		t.Fatal("the clone shares the source operation backing")
	}

	backing := &clone.Ops[0]

	relayoutSpliceOpsInPlace(clone, 10, 4, spliceTestOps(6, 6, opCount))

	if &clone.Ops[0] != backing {
		t.Fatal("the grow splice reallocated despite the clone's spare capacity")
	}

	if len(clone.Ops) != opCount+2 {
		t.Fatalf("clone carries %d operations after the grow, want %d", len(clone.Ops), opCount+2)
	}
}

// TestRelayoutFuseSpliceSizesTheBoxListExactly pins the fused walk's capacity
// hint: a splice that keeps the box count rebuilds the flattened list at its
// exact length instead of growing it from zero.
func TestRelayoutFuseSpliceSizesTheBoxListExactly(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutFixtureHTML)
	opts := relayoutOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	first := relayoutNodeByID(t, root, "first")
	first.Attrs["class"] = relayoutSelectedClass

	if _, err := RelayoutSubtree(t.Context(), prev, first, opts); err != nil {
		t.Fatalf("RelayoutSubtree: %v", err)
	}

	if cap(prev.boxes) != len(prev.boxes) {
		t.Fatalf(
			"flattened box list has capacity %d for %d boxes, want an exact list",
			cap(prev.boxes), len(prev.boxes),
		)
	}
}
