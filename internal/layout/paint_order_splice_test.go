package layout

import (
	"reflect"
	"testing"
)

// TestSplicePaintOrderReusesEqualNeutralWindow pins the early return: a
// paint-only splash over a window that kept its length and its
// ordering-neutral keys leaves the maintained order in place, with no
// allocation and no sort. A window that changed length still merges, and the
// merged order must equal the full sort.
func TestSplicePaintOrderReusesEqualNeutralWindow(t *testing.T) {
	t.Parallel()

	ops := []Op{
		{Kind: OpFillRect},
		{Kind: OpText},
		{Kind: OpText},
		{Kind: OpFillRect},
		{Kind: OpText, Positioned: true},
	}

	order := PaintOrder(ops)

	splashed := append([]Op(nil), ops...)
	splashed[1].R, splashed[1].G, splashed[1].B = 0.1, 0.2, 0.3
	splashed[2].R, splashed[2].G, splashed[2].B = 0.4, 0.5, 0.6

	reused := splicePaintOrder(splashed, order, 1, 2, 2, true)
	if &reused[0] != &order[0] {
		t.Fatal("an equal neutral window allocated a new order, want the maintained one")
	}

	if want := PaintOrder(splashed); !reflect.DeepEqual(reused, want) {
		t.Fatalf("reused order = %v, want the full sort %v", reused, want)
	}

	inserted := make([]Op, 0, len(ops)+1)
	inserted = append(inserted, ops[:2]...)
	inserted = append(inserted, Op{Kind: OpText})
	inserted = append(inserted, ops[2:]...)

	merged := splicePaintOrder(inserted, order, 2, 1, 2, true)
	if want := PaintOrder(inserted); !reflect.DeepEqual(merged, want) {
		t.Fatalf("merged order = %v, want the full sort %v", merged, want)
	}
}
