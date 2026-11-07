package layout

import (
	"reflect"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
)

// TestNewDisplayForPatchReusesBoxesAndSlices verifies that newDisplayForPatch
// reuses prev.Boxes, pools parents, and reuses prev.OpOwners and prev.Order
// when boxCount and opCount match.
func TestNewDisplayForPatchReusesBoxesAndSlices(t *testing.T) {
	display := renderEmitDisplay(t, emitTableRows, emitTableCols, true)
	if display.result == nil || len(display.Boxes) == 0 {
		t.Fatal("retained result missing")
	}

	patched := newDisplayForPatch(display.result, display, display.ForcedBreaks, display.KeepTogether, true)
	if patched == nil {
		t.Fatal("newDisplayForPatch returned nil")
	}

	if len(patched.Boxes) != len(display.Boxes) {
		t.Fatalf("patched boxes = %d, want %d", len(patched.Boxes), len(display.Boxes))
	}
	if &patched.Boxes[0] != &display.Boxes[0] {
		t.Fatal("patched display did not reuse prev.Boxes backing array")
	}

	if len(patched.OpOwners) != len(display.OpOwners) {
		t.Fatalf("patched owners = %d, want %d", len(patched.OpOwners), len(display.OpOwners))
	}
	if &patched.OpOwners[0] != &display.OpOwners[0] {
		t.Fatal("patched display did not reuse prev.OpOwners backing array")
	}

	if len(patched.Order) != len(display.Order) {
		t.Fatalf("patched order = %d, want %d", len(patched.Order), len(display.Order))
	}
	if &patched.Order[0] != &display.Order[0] {
		t.Fatal("patched display did not reuse prev.Order backing array")
	}
}

// TestNewDisplayForPatchParityWithFresh verifies that the display produced by
// newDisplayForPatch has identical contents (Boxes, OpOwners, Order) to a fresh
// display built from the same placement.
func TestNewDisplayForPatchParityWithFresh(t *testing.T) {
	fresh := renderEmitDisplay(t, 20, 4, true)
	base := renderEmitDisplay(t, 20, 4, true)

	patched := newDisplayForPatch(base.result, base, base.ForcedBreaks, base.KeepTogether, true)
	if patched == nil {
		t.Fatal("patched display is nil")
	}

	if len(patched.Boxes) != len(fresh.Boxes) {
		t.Fatalf("boxes count = %d, want %d", len(patched.Boxes), len(fresh.Boxes))
	}
	for i := range fresh.Boxes {
		if !reflect.DeepEqual(patched.Boxes[i], fresh.Boxes[i]) {
			t.Fatalf("box %d mismatch: got %+v, want %+v", i, patched.Boxes[i], fresh.Boxes[i])
		}
	}

	if !reflect.DeepEqual(patched.OpOwners, fresh.OpOwners) {
		t.Fatal("patched OpOwners differs from fresh")
	}
	if !reflect.DeepEqual(patched.Order, fresh.Order) {
		t.Fatal("patched Order differs from fresh")
	}
}

// TestNewDisplayForPatchNilPrev verifies that newDisplayForPatch handles a nil
// prev safely by renting or allocating slices.
func TestNewDisplayForPatchNilPrev(t *testing.T) {
	display := renderEmitDisplay(t, 10, 2, true)
	if display.result == nil {
		t.Fatal("retained result missing")
	}

	dummyPrev := &Display{
		Width:          display.Width,
		Height:         display.Height,
		PointsPerPixel: display.PointsPerPixel,
		PixelPerPoint:  display.PixelPerPoint,
		zoom:           display.zoom,
	}

	patched := newDisplayForPatch(display.result, dummyPrev, nil, nil, true)
	if patched == nil {
		t.Fatal("newDisplayForPatch returned nil")
	}

	if len(patched.Boxes) != display.result.BoxCount() {
		t.Fatalf("boxes = %d, want %d", len(patched.Boxes), display.result.BoxCount())
	}
	if len(patched.OpOwners) != len(display.result.Ops) {
		t.Fatalf("owners = %d, want %d", len(patched.OpOwners), len(display.result.Ops))
	}
	if len(patched.Order) != len(display.result.Ops) {
		t.Fatalf("order = %d, want %d", len(patched.Order), len(display.result.Ops))
	}
}

// TestNewDisplayForPatchDeriveOwnersFalse verifies that when deriveOwners is
// false, OpOwners and Order remain nil as expected by the batch splice path.
func TestNewDisplayForPatchDeriveOwnersFalse(t *testing.T) {
	display := renderEmitDisplay(t, 10, 2, true)
	if display.result == nil {
		t.Fatal("retained result missing")
	}

	patched := newDisplayForPatch(display.result, display, nil, nil, false)
	if patched == nil {
		t.Fatal("newDisplayForPatch returned nil")
	}

	if patched.OpOwners != nil {
		t.Fatalf("OpOwners = %v, want nil", patched.OpOwners)
	}
	if patched.Order != nil {
		t.Fatalf("Order = %v, want nil", patched.Order)
	}
}

// BenchmarkNewDisplayForPatch measures the allocations and CPU cost of
// newDisplayForPatch on the 300x4 golden table (1204 boxes, 7200 ops).
func BenchmarkNewDisplayForPatch(b *testing.B) {
	source := buildEmitTableSource(emitTableRows, emitTableCols)
	doc, err := html.Parse([]byte(source))
	if err != nil {
		b.Fatalf("parse: %v", err)
	}

	styled, err := css.Apply(b.Context(), doc, css.Options{
		WidthPx:  640,
		HeightPx: 480,
		Media:    "screen",
	})
	if err != nil {
		b.Fatalf("css: %v", err)
	}

	display, err := DisplayListOptions(b.Context(), styled, Options{RetainState: true})
	if err != nil {
		b.Fatalf("display: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = newDisplayForPatch(display.result, display, display.ForcedBreaks, display.KeepTogether, true)
	}
}
