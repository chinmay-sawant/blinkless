package layout_test

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

// Allocation sites in layout/displaylist.go (read, not changed, except where
// noted):
//
//   - boxes and parents in newDisplay are already exact upper bounds:
//     make([]Box, 0, boxCount) and make([]int, 0, boxCount) where boxCount is
//     Result.BoxCount, the flattened box count, which covers every element
//     box WalkPlaced can emit. Append never grows them and never truncates.
//   - OpOwners comes from ilayout.OperationOwnersFromParents as
//     make([]int, len(res.Ops)) with in-place fill, no append growth.
//   - Order comes from ilayout.PaintOrder as make([]int, len(ops)) with an
//     already-sorted fast path, so a plain table pays exactly one alloc.
//   - keyframesFrom grew its rules slice from nil by append (changed here to
//     a pre-counted exact capacity; nil is kept when there are no rules).
//
// The table below has no keyframes, so it pins zero regression on the hot
// path; the keyframes change only removes append growth there.

// allocTableRows and allocTableCols size the golden fixture. 300x4 yields
// 7200 ops and 1204 boxes, large enough to expose per-row slice growth.
const (
	allocTableRows = 300
	allocTableCols = 4

	wantTableOps   = 7200
	wantTableBoxes = 1204
)

// buildAllocTableSource returns a table document with the given rows and
// columns. Every cell carries paint (background plus text) so the display
// list grows with the row count.
func buildAllocTableSource(rows, cols int) string {
	var sb strings.Builder

	sb.WriteString("<table>")
	for r := 0; r < rows; r++ {
		sb.WriteString("<tr>")
		for c := 0; c < cols; c++ {
			sb.WriteString(`<td style="background:#eee;border:1px solid #333">cell</td>`)
		}
		sb.WriteString("</tr>")
	}
	sb.WriteString("</table>")

	return sb.String()
}

func renderAllocTable(t *testing.T, rows, cols int) *layout.Display {
	t.Helper()

	source := buildAllocTableSource(rows, cols)

	doc, err := html.Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled, err := css.Apply(t.Context(), doc, css.Options{
		WidthPx:  640,
		HeightPx: 480,
		Media:    "screen",
		Extra:    nil,
	})
	if err != nil {
		t.Fatalf("css: %v", err)
	}

	display, err := layout.DisplayList(t.Context(), styled)
	if err != nil {
		t.Fatalf("display: %v", err)
	}

	return display
}

func approxEqual(got, want float64) bool {
	const epsilon = 1e-6
	diff := got - want
	if diff < 0 {
		diff = -diff
	}

	return diff <= epsilon
}

// TestDisplaylistTableGolden pins the table display against values recorded
// from the unmodified code path. Any change to op emission, owner mapping,
// paint order, or box windowing in layout/displaylist.go must keep these
// identical: same op count, same first and last op fields, same box count
// and edge boxes.
func TestDisplaylistTableGolden(t *testing.T) {
	t.Parallel()

	display := renderAllocTable(t, allocTableRows, allocTableCols)

	if len(display.Ops) != wantTableOps {
		t.Fatalf("ops = %d, want %d", len(display.Ops), wantTableOps)
	}

	if len(display.Boxes) != wantTableBoxes {
		t.Fatalf("boxes = %d, want %d", len(display.Boxes), wantTableBoxes)
	}

	if len(display.OpOwners) != len(display.Ops) {
		t.Fatalf("owners = %d, want one per op (%d)", len(display.OpOwners), len(display.Ops))
	}

	if len(display.Order) != len(display.Ops) {
		t.Fatalf("order = %d, want one per op (%d)", len(display.Order), len(display.Ops))
	}

	seen := make([]bool, len(display.Ops))
	for _, index := range display.Order {
		if index < 0 || index >= len(display.Ops) {
			t.Fatalf("order index %d out of range for %d ops", index, len(display.Ops))
		}
		if seen[index] {
			t.Fatalf("order repeats index %d", index)
		}
		seen[index] = true
	}

	first := display.Ops[0]
	if first.Kind != layout.DisplayOpFillRect {
		t.Errorf("first op kind = %d, want fill %d", first.Kind, layout.DisplayOpFillRect)
	}
	if first.ID != 1 {
		t.Errorf("first op id = %d, want 1", first.ID)
	}
	if !approxEqual(first.X, 7.5) || !approxEqual(first.Y, 7.5) ||
		!approxEqual(first.W, 21.005859) || !approxEqual(first.H, 16.5) {
		t.Errorf("first op rect = (%.6f, %.6f, %.6f, %.6f), want (7.5, 7.5, 21.005859, 16.5)",
			first.X, first.Y, first.W, first.H)
	}

	last := display.Ops[len(display.Ops)-1]
	if last.Kind != layout.DisplayOpText {
		t.Errorf("last op kind = %d, want text %d", last.Kind, layout.DisplayOpText)
	}
	if last.ID != uint64(wantTableOps) {
		t.Errorf("last op id = %d, want %d", last.ID, wantTableOps)
	}
	if last.Text != "cell" {
		t.Errorf("last op text = %q, want %q", last.Text, "cell")
	}
	if !approxEqual(last.X, 76.517578) || !approxEqual(last.Y, 5401.910156) ||
		!approxEqual(last.W, 18.005859) || !approxEqual(last.H, 13.5) {
		t.Errorf("last op rect = (%.6f, %.6f, %.6f, %.6f), want (76.517578, 5401.910156, 18.005859, 13.5)",
			last.X, last.Y, last.W, last.H)
	}

	if display.Boxes[0].Tag != "#document" {
		t.Errorf("first box tag = %q, want %q", display.Boxes[0].Tag, "#document")
	}
	if !approxEqual(display.Boxes[0].W, 640) {
		t.Errorf("first box width = %.3f, want 640.000", display.Boxes[0].W)
	}

	lastBox := display.Boxes[len(display.Boxes)-1]
	if lastBox.Tag != "td" {
		t.Errorf("last box tag = %q, want %q", lastBox.Tag, "td")
	}
	if !approxEqual(lastBox.X, 100.023438) || !approxEqual(lastBox.Y, 7186) ||
		!approxEqual(lastBox.W, 28.007812) || !approxEqual(lastBox.H, 22) {
		t.Errorf("last box rect = (%.6f, %.6f, %.6f, %.6f), want (100.023438, 7186, 28.007812, 22)",
			lastBox.X, lastBox.Y, lastBox.W, lastBox.H)
	}
}

// TestDisplaylistOrderAllocs pins the DisplayOrder helper at one alloc per
// run for the table ops. PaintOrder pre-sizes its index slice exactly and
// takes the already-sorted fast path here, so this documents zero regression
// on the owner/order path. It must not be parallel: AllocsPerRun refuses to
// run inside a parallel test.
func TestDisplaylistOrderAllocs(t *testing.T) {
	display := renderAllocTable(t, allocTableRows, allocTableCols)

	allocs := testing.AllocsPerRun(100, func() {
		_ = layout.DisplayOrder(display.Ops)
	})

	if allocs > 1 {
		t.Errorf("DisplayOrder allocs = %.0f, want at most 1", allocs)
	}
}
