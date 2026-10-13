package layout

import "testing"

func TestPaintOrderSharedPolicyKeepsStableMetadataOrder(t *testing.T) {
	t.Parallel()

	ops := []Op{
		{Kind: OpText, ZIndexSet: true},
		{Kind: OpFillRect, ZIndexSet: true},
		{Kind: OpText, ZIndexSet: true},
		{Kind: OpFillRect, ZIndexSet: true},
		{Kind: OpLinkURI, ZIndexSet: true},
	}
	// The test intentionally sets the actual z-index values below after the
	// compact literals keep the operation kinds easy to scan.
	ops[0].ZIndex = 0
	ops[1].ZIndex = 0
	ops[2].ZIndex = 2
	ops[3].ZIndex = -1
	ops[4].ZIndex = 0

	if got, want := PaintOrder(ops), []int{3, 1, 0, 4, 2}; !sameIndices(got, want) {
		t.Fatalf("paint order = %v, want %v", got, want)
	}

	subset := []int{4, 0, 1, 3}
	if got, want := paintOrderSubset(ops, subset), []int{3, 1, 0, 4}; !sameIndices(got, want) {
		t.Fatalf("subset paint order = %v, want %v", got, want)
	}
}

func TestPaintOrderOutlinesAboveDescendantChrome(t *testing.T) {
	t.Parallel()

	ops := []Op{
		{Kind: OpFillRect},
		{Kind: OpLine},
		{Kind: OpFillRect},
	}
	ops[1].setOutline()

	if got, want := PaintOrder(ops), []int{0, 2, 1}; !sameIndices(got, want) {
		t.Fatalf("paint order = %v, want outline after descendant fill %v", got, want)
	}
}

func TestPaintOrderKeepsGroupedDescendantInsideContext(t *testing.T) {
	t.Parallel()

	group := &BlendGroup{ID: 1, Isolate: true}
	ops := []Op{
		{Kind: OpUnknown, ZIndexSet: true},
		{Kind: OpText, ZIndex: 5, ZIndexSet: true},
		{Kind: OpUnknown, ZIndexSet: true},
		{Kind: OpText, ZIndex: 1, ZIndexSet: true},
	}
	ops[0].setGroupMark(group, groupMarkBegin)
	ops[1].setBlendGroup(group)
	ops[2].setGroupMark(group, groupMarkEnd)

	if got, want := PaintOrder(ops), []int{0, 1, 2, 3}; !sameIndices(got, want) {
		t.Fatalf("paint order = %v, want grouped child and boundaries before sibling %v", got, want)
	}
}

func TestPaintOrderUsesGroupBeginSlotBetweenChromeAndContent(t *testing.T) {
	t.Parallel()

	group := &BlendGroup{ID: 1, Isolate: true}
	ops := []Op{
		{Kind: OpFillRect, ZIndexSet: true},
		{Kind: OpUnknown, ZIndexSet: true},
		{Kind: OpFillRect, ZIndexSet: true},
		{Kind: OpText, ZIndexSet: true},
		{Kind: OpUnknown, ZIndexSet: true},
	}
	ops[1].setGroupMark(group, groupMarkBegin)
	ops[2].setBlendGroup(group)
	ops[4].setGroupMark(group, groupMarkEnd)

	if got, want := PaintOrder(ops), []int{0, 1, 2, 4, 3}; !sameIndices(got, want) {
		t.Fatalf("paint order = %v, want group at begin-marker slot between chrome and content %v", got, want)
	}
}

func TestPaintOrderUsesGroupOwnershipInsteadOfMarkerRange(t *testing.T) {
	t.Parallel()

	group := &BlendGroup{ID: 1, Isolate: true}
	ops := []Op{
		{Kind: OpUnknown, ZIndexSet: true},
		{Kind: OpText, ZIndexSet: true},
		{Kind: OpUnknown, ZIndexSet: true},
		{Kind: OpLine, ZIndexSet: true},
		{Kind: OpText, ZIndex: 1, ZIndexSet: true},
	}
	ops[0].setGroupMark(group, groupMarkBegin)
	ops[1].setBlendGroup(group)
	ops[2].setGroupMark(group, groupMarkEnd)
	ops[3].setBlendGroup(group)

	if got, want := PaintOrder(ops), []int{0, 3, 1, 2, 4}; !sameIndices(got, want) {
		t.Fatalf("paint order = %v, want late group-owned op before its end marker %v", got, want)
	}

	if got, want := paintOrderSubset(ops, []int{3, 4}), []int{3, 4}; !sameIndices(got, want) {
		t.Fatalf("subset paint order = %v, want exact selected ops %v", got, want)
	}
}

func TestPaintOrderNestsGroupBoundaries(t *testing.T) {
	t.Parallel()

	parent := &BlendGroup{ID: 1, Isolate: true}
	child := &BlendGroup{ID: 2, Isolate: true, Parent: parent}
	ops := []Op{
		{Kind: OpUnknown, ZIndexSet: true},
		{Kind: OpText, ZIndexSet: true},
		{Kind: OpUnknown, ZIndex: 2, ZIndexSet: true},
		{Kind: OpText, ZIndex: 5, ZIndexSet: true},
		{Kind: OpUnknown, ZIndex: 2, ZIndexSet: true},
		{Kind: OpUnknown, ZIndexSet: true},
		{Kind: OpText, ZIndex: 1, ZIndexSet: true},
	}
	ops[0].setGroupMark(parent, groupMarkBegin)
	ops[1].setBlendGroup(parent)
	ops[2].setGroupMark(child, groupMarkBegin)
	ops[3].setBlendGroup(child)
	ops[4].setGroupMark(child, groupMarkEnd)
	ops[5].setGroupMark(parent, groupMarkEnd)

	if got, want := PaintOrder(ops), []int{0, 1, 2, 3, 4, 5, 6}; !sameIndices(got, want) {
		t.Fatalf("paint order = %v, want nested groups to stay inside their parent %v", got, want)
	}
}

func TestPaintOrderKeepsNestedMemberAfterEndMarkerInsideGroup(t *testing.T) {
	t.Parallel()

	parent := &BlendGroup{ID: 1, Isolate: true}
	child := &BlendGroup{ID: 2, Isolate: true, Parent: parent}
	ops := []Op{
		{Kind: OpUnknown, ZIndexSet: true},
		{Kind: OpUnknown, ZIndexSet: true},
		{Kind: OpText, ZIndexSet: true},
		{Kind: OpUnknown, ZIndexSet: true},
		{Kind: OpLine, ZIndexSet: true},
		{Kind: OpUnknown, ZIndexSet: true},
		{Kind: OpText, ZIndex: 1, ZIndexSet: true},
	}
	ops[0].setGroupMark(parent, groupMarkBegin)
	ops[1].setGroupMark(child, groupMarkBegin)
	ops[2].setBlendGroup(child)
	ops[3].setGroupMark(child, groupMarkEnd)
	ops[4].setBlendGroup(child)
	ops[5].setGroupMark(parent, groupMarkEnd)

	if got, want := PaintOrder(ops), []int{0, 1, 4, 2, 3, 5, 6}; !sameIndices(got, want) {
		t.Fatalf("paint order = %v, want late nested member inside its group %v", got, want)
	}
}

func TestPaintOrderSubsetUsesOwnerAnchorAndPreservesIndices(t *testing.T) {
	t.Parallel()

	parent := &BlendGroup{ID: 1, Isolate: true}
	child := &BlendGroup{ID: 2, Isolate: true, Parent: parent}
	ops := []Op{
		{Kind: OpUnknown, ZIndexSet: true},
		{Kind: OpUnknown, ZIndex: 5, ZIndexSet: true},
		{Kind: OpText, ZIndex: 9, ZIndexSet: true},
		{Kind: OpUnknown, ZIndex: 5, ZIndexSet: true},
		{Kind: OpUnknown, ZIndexSet: true},
		{Kind: OpText, ZIndex: 3, ZIndexSet: true},
	}
	ops[0].setGroupMark(parent, groupMarkBegin)
	ops[1].setGroupMark(child, groupMarkBegin)
	ops[2].setBlendGroup(child)
	ops[3].setGroupMark(child, groupMarkEnd)
	ops[4].setGroupMark(parent, groupMarkEnd)

	// The subset leaves out both groups' markers. The selected child still
	// sorts at the parent's z-index, and no missing marker index is injected.
	subset := []int{5, 2}
	if got, want := paintOrderSubset(ops, subset), []int{2, 5}; !sameIndices(got, want) {
		t.Fatalf("subset paint order = %v, want selected indices %v", got, want)
	}

	if len(subset) != 2 || subset[0] != 5 || subset[1] != 2 {
		t.Fatalf("paintOrderSubset changed input subset: %v", subset)
	}
}

func sameIndices(got, want []int) bool {
	if len(got) != len(want) {
		return false
	}

	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}
