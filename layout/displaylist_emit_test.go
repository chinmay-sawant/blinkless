package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	ilayout "github.com/chinmay-sawant/blinkless/internal/layout"
)

// Emission-step pins for layout/displaylist.go. newDisplay issues the owner
// mapping and the paint order as one fused step (emitOwnersAndOrder) from one
// shared op slice capture, with every slice presized from the count known
// before emission (boxes/parents from BoxCount, owners/order from len(Ops),
// keyframes rules from the pre-counted total). These tests pin that the fused
// step changes no op, ID, owner, order, or box against values recorded from
// the passing pre-edit run, and that the fused step costs no more allocs than
// the two separate derivations it replaces.

// emitTableRows and emitTableCols size the golden fixture. 300x4 yields 7200
// ops and 1204 boxes, the same shape as the allocation pin in
// displaylist_alloc_test.go, large enough to expose per-row slice growth.
const (
	emitTableRows = 300
	emitTableCols = 4

	wantEmitTableOps   = 7200
	wantEmitTableBoxes = 1204
)

// buildEmitTableSource returns a table document with the given rows and
// columns. Every cell carries paint (background plus text) so the display
// list grows with the row count.
func buildEmitTableSource(rows, cols int) string {
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

func renderEmitDisplay(t *testing.T, rows, cols int, retain bool) *Display {
	t.Helper()

	source := buildEmitTableSource(rows, cols)

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

	display, err := DisplayListOptions(t.Context(), styled, Options{Images: nil, RetainState: retain})
	if err != nil {
		t.Fatalf("display: %v", err)
	}

	return display
}

func emitApproxEqual(got, want float64) bool {
	const epsilon = 1e-6
	diff := got - want
	if diff < 0 {
		diff = -diff
	}

	return diff <= epsilon
}

// TestDisplaylistEmitTableGolden pins the fused emission step against values
// recorded from the passing pre-edit run. Any change to op emission, owner
// mapping, paint order, or box windowing in layout/displaylist.go must keep
// these identical: same op count, same first and last op fields, same box
// count and edge boxes.
func TestDisplaylistEmitTableGolden(t *testing.T) {
	t.Parallel()

	display := renderEmitDisplay(t, emitTableRows, emitTableCols, false)

	if len(display.Ops) != wantEmitTableOps {
		t.Fatalf("ops = %d, want %d", len(display.Ops), wantEmitTableOps)
	}

	if len(display.Boxes) != wantEmitTableBoxes {
		t.Fatalf("boxes = %d, want %d", len(display.Boxes), wantEmitTableBoxes)
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
	if first.Kind != DisplayOpFillRect {
		t.Errorf("first op kind = %d, want fill %d", first.Kind, DisplayOpFillRect)
	}
	if first.ID != 1 {
		t.Errorf("first op id = %d, want 1", first.ID)
	}
	if !emitApproxEqual(first.X, 7.5) || !emitApproxEqual(first.Y, 7.5) ||
		!emitApproxEqual(first.W, 21.005859) || !emitApproxEqual(first.H, 16.5) {
		t.Errorf("first op rect = (%.6f, %.6f, %.6f, %.6f), want (7.5, 7.5, 21.005859, 16.5)",
			first.X, first.Y, first.W, first.H)
	}

	last := display.Ops[len(display.Ops)-1]
	if last.Kind != DisplayOpText {
		t.Errorf("last op kind = %d, want text %d", last.Kind, DisplayOpText)
	}
	if last.ID != uint64(wantEmitTableOps) {
		t.Errorf("last op id = %d, want %d", last.ID, wantEmitTableOps)
	}
	if last.Text != "cell" {
		t.Errorf("last op text = %q, want %q", last.Text, "cell")
	}
	if !emitApproxEqual(last.X, 76.517578) || !emitApproxEqual(last.Y, 5401.910156) ||
		!emitApproxEqual(last.W, 18.005859) || !emitApproxEqual(last.H, 13.5) {
		t.Errorf("last op rect = (%.6f, %.6f, %.6f, %.6f), want (76.517578, 5401.910156, 18.005859, 13.5)",
			last.X, last.Y, last.W, last.H)
	}

	if display.Boxes[0].Tag != "#document" {
		t.Errorf("first box tag = %q, want %q", display.Boxes[0].Tag, "#document")
	}
	if !emitApproxEqual(display.Boxes[0].W, 640) {
		t.Errorf("first box width = %.3f, want 640.000", display.Boxes[0].W)
	}

	lastBox := display.Boxes[len(display.Boxes)-1]
	if lastBox.Tag != "td" {
		t.Errorf("last box tag = %q, want %q", lastBox.Tag, "td")
	}
	if !emitApproxEqual(lastBox.X, 100.023438) || !emitApproxEqual(lastBox.Y, 7186) ||
		!emitApproxEqual(lastBox.W, 28.007812) || !emitApproxEqual(lastBox.H, 22) {
		t.Errorf("last box rect = (%.6f, %.6f, %.6f, %.6f), want (100.023438, 7186, 28.007812, 22)",
			lastBox.X, lastBox.Y, lastBox.W, lastBox.H)
	}
}

// TestDisplaylistEmitAllocs pins the fused emission helper at the cost of the
// two separate derivations it replaces: one alloc for the owners slice and
// one for the order index slice on the table ops. It also checks the fused
// step against the separate calls element by element, so the fuse provably
// computes the same owners and order. It must not be parallel:
// AllocsPerRun refuses to run inside a parallel test.
func TestDisplaylistEmitAllocs(t *testing.T) {
	display := renderEmitDisplay(t, emitTableRows, emitTableCols, true)

	if display.result == nil {
		t.Fatal("retained result missing")
	}

	parents := make([]int, len(display.Boxes))
	for i := range display.Boxes {
		parents[i] = display.Boxes[i].ParentIndex
	}

	owners, order := emitOwnersAndOrder(display.result, parents)

	wantOwners := ilayout.OperationOwnersFromParents(display.result, parents)
	wantOrder := ilayout.PaintOrder(display.Ops)

	if len(owners) != len(wantOwners) || len(order) != len(wantOrder) {
		t.Fatalf("fused lengths = (%d, %d), want (%d, %d)",
			len(owners), len(order), len(wantOwners), len(wantOrder))
	}

	for i := range owners {
		if owners[i] != wantOwners[i] {
			t.Fatalf("fused owner [%d] = %d, want %d", i, owners[i], wantOwners[i])
		}
	}

	for i := range order {
		if order[i] != wantOrder[i] {
			t.Fatalf("fused order [%d] = %d, want %d", i, order[i], wantOrder[i])
		}
	}

	allocs := testing.AllocsPerRun(100, func() {
		fusedOwners, fusedOrder := emitOwnersAndOrder(display.result, parents)
		if len(fusedOwners) != len(display.Ops) || len(fusedOrder) != len(display.Ops) {
			t.Errorf("fused lengths = (%d, %d), want one per op (%d)",
				len(fusedOwners), len(fusedOrder), len(display.Ops))
		}
	})

	if allocs > 2 {
		t.Errorf("emitOwnersAndOrder allocs = %.0f, want at most 2", allocs)
	}
}
