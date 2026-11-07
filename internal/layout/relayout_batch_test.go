package layout

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Twin-match test for a BATCH of 3 paint-only row patches on the 60-row table
// shape (span-in-td, nested div, and plain-text cells under a fixed layout,
// the same cell shapes as the single-row splice test).
//
// Baseline: a 3-patch batch on this wide table costs ~5.1MB, from
// whole-table escalation per element plus per-patch display cloning. The goal
// is KBs per batch: one batch call shares its setup across all three ids,
// and each id rebuilds and splices only its own row while untouched rows keep
// their boxes.

// relayoutBatchRows is the wider table size: 60 rows by 4 cells. Per-row
// content is fixed (span-in-td, nested percent div, two plain text cells),
// matching the single-row splice fixture's cell shapes.
const relayoutBatchRows = 60

// relayoutBatchCSS fixes the table width so the percent columns have a basis.
// The tint rule paints only patched rows' cells: background color keeps each
// row's geometry, so every patch must splice instead of reporting
// errSubtreeResized.
const relayoutBatchCSS = `
	table.grid { width: 640px; table-layout: fixed; border-collapse: collapse; }
	td { padding: 4px; border: 1px solid #999999; }
	td.c-a { width: 30%; }
	td.c-b { width: 30%; }
	td.c-c { width: 20%; }
	td.c-d { width: 20%; }
	tr.tint td { background-color: #fff3bf; }
	.box { width: 50%; height: 18px; background-color: #eeeeee; }
`

// relayoutBatchHTML builds a rows-by-four table: every row carries a span in
// its first cell, a nested percent-width div in its second, and plain text in
// its third and fourth. Every row is numbered so the test can tint three
// middle rows in one batch and prove a far row kept its boxes.
func relayoutBatchHTML(rows int) string {
	var b strings.Builder

	b.WriteString(`<html><body><table class="grid">`)

	for i := range rows {
		fmt.Fprintf(&b, `<tr id="brow-%02d">`, i)
		fmt.Fprintf(&b, `<td class="c-a"><span>Item %04d</span></td>`, i)
		fmt.Fprintf(&b, `<td class="c-b"><div class="box">%d pcs</div></td>`, i)
		fmt.Fprintf(&b, `<td class="c-c">%d pcs</td>`, i)
		fmt.Fprintf(&b, `<td class="c-d">9.99</td>`)
		b.WriteString(`</tr>`)
	}

	b.WriteString(`</table></body></html>`)

	return b.String()
}

// TestRelayoutBatchThreeRowTintsMatchFullLayout pins the batch path for three
// paint-only row patches: tinting three middle rows (geometry-neutral
// background-color changes) in ONE RelayoutSubtrees call must succeed, match
// a fresh full Layout, and leave a far row's boxes untouched.
func TestRelayoutBatchThreeRowTintsMatchFullLayout(t *testing.T) {
	t.Parallel()

	rows := relayoutBatchRows
	ids := []string{
		fmt.Sprintf("brow-%02d", rows/2-2),
		fmt.Sprintf("brow-%02d", rows/2),
		fmt.Sprintf("brow-%02d", rows/2+2),
	}
	farID := fmt.Sprintf("brow-%02d", rows-1)

	root := mustParse(t, relayoutBatchHTML(rows))
	opts := relayoutOptionsForCSS(t, relayoutBatchCSS)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	farBefore := relayoutBatchRowCells(t, prev, farID)
	if len(farBefore) != 4 {
		t.Fatalf("far row %q cells = %d, want 4", farID, len(farBefore))
	}

	farBoxes := make([]box, len(farBefore))
	for i, cell := range farBefore {
		farBoxes[i] = *cell
	}

	for _, id := range ids {
		relayoutNodeByID(t, root, id).Attrs["class"] = "tint"
	}

	// One batch call shares its setup across all three ids (per-batch shared
	// work: one box thread, one ordering-relevant count, one font baseline
	// for the text-bearing rows, and the owners/order slices threaded through
	// every splice), while each id still pays its own row rebuild plus one
	// splice window (per-element work). Three separate single-element calls
	// would repeat the shared work per id and clone the display per patch.
	out, err := RelayoutSubtrees(t.Context(), prev, ids, opts,
		OperationOwners(prev), PaintOrder(prev.Ops))

	// Assertion 1: the batch succeeds. A refusal (errSubtreeResized or
	// errSubtreeNotSplicable) would force the whole-table fallback the
	// ~5.1MB baseline measures, so success pins that all three patches took
	// the row-granular splice instead of escalating.
	if err != nil {
		if errors.Is(err, errSubtreeResized) {
			t.Fatalf("RelayoutSubtrees(%v class=tint) = %v, want success "+
				"(background-color keeps geometry); a rebuilt grid moved", ids, err)
		}

		if errors.Is(err, errSubtreeNotSplicable) {
			t.Fatalf("RelayoutSubtrees(%v class=tint) = %v, want row-granular "+
				"splices, not a refusal", ids, err)
		}

		t.Fatalf("RelayoutSubtrees(%v class=tint) = %v, want success", ids, err)
	}

	if out == nil || len(out.Windows) != len(ids) {
		t.Fatalf("RelayoutSubtrees windows = %+v, want one committed window per id", out)
	}

	full, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout (full relayout): %v", err)
	}

	// Assertion 2: the spliced result twins a fresh full Layout of the same
	// tree. Operation equality (ids stripped) proves no per-patch display
	// cloning leaked extra or stale ops into the batch commit, and the
	// recursive geometry walk proves the shared commit placed every box where
	// the full layout does, descending into table rows and cells as well as
	// the children chain.
	if !reflect.DeepEqual(relayoutStripIDs(prev.Ops), relayoutStripIDs(full.Ops)) {
		t.Fatalf("patched operations differ from a full relayout:\n got %+v\nwant %+v",
			relayoutStripIDs(prev.Ops), relayoutStripIDs(full.Ops))
	}

	relayoutRowSpanAssertGeometry(t, prev.boxTreeRoot(), full.boxTreeRoot())

	// Assertion 3: a far row's cell box POINTERS are unchanged. Pointer
	// identity pins no-clone/no-rebuild of untouched ranges: the batch
	// rebuilt only the three tinted rows, so the other 57 rows kept their
	// boxes. Whole-table escalation per element would rebuild all 60 rows and
	// fail this check with the same geometry on fresh allocations. This is
	// the granularity pin that moves the batch from ~5.1MB toward KBs.
	farAfter := relayoutBatchRowCells(t, prev, farID)
	if len(farAfter) != len(farBefore) {
		t.Fatalf("far row %q cells = %d, want the kept %d", farID, len(farAfter), len(farBefore))
	}

	for i := range farAfter {
		if farAfter[i] != farBefore[i] {
			t.Fatalf("far row %q cell %d was rebuilt, want the kept box: "+
				"a patch escalated to the owning table instead of splicing at row granularity",
				farID, i)
		}

		if !relayoutSameGeometry(farAfter[i], &farBoxes[i]) {
			t.Fatalf("far row %q cell %d moved: got (%g, %g) %gx%g, want (%g, %g) %gx%g",
				farID, i,
				farAfter[i].x, farAfter[i].y, farAfter[i].w, farAfter[i].height,
				farBoxes[i].x, farBoxes[i].y, farBoxes[i].w, farBoxes[i].height)
		}
	}
}

// relayoutBatchRowCells returns the cell boxes of one table row in column
// order. A tr places no box of its own, so the row's cells are found through
// the table's row list by matching the cell node's parent row.
func relayoutBatchRowCells(t *testing.T, res *Result, rowID string) []*box {
	t.Helper()

	table := res.boxTreeRoot()
	if table == nil {
		t.Fatal("no box tree in the retained result")
	}

	var out []*box

	var walk func(b *box)
	walk = func(b *box) {
		if b == nil {
			return
		}

		for _, row := range b.rows {
			for _, cell := range row {
				if cell.node == nil || cell.node.Parent == nil ||
					cell.node.Parent.Attribute("id") != rowID {
					continue
				}

				out = append(out, cell)
			}
		}

		for _, child := range b.children {
			walk(child)
		}
	}
	walk(table)

	return out
}
