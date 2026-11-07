package layout

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// NOTE: this test is EXPECTED TO FAIL until the row-granularity splice fix
// lands. A tr places no box of its own, so today the patch escalates through
// tableAncestorTarget to the owning table and rebuilds all 60 rows before
// splicing the table back (the whole-table path the sibling
// BenchmarkRelayoutRowSplice60 measures). Expected red signature pre-fix:
// either RelayoutSubtreeWithOverrides refuses the patch (assertion 1 reports
// errSubtreeResized or errSubtreeNotSplicable), or the patch succeeds by
// rebuilding the table and assertion 3 reports the far row's cell boxes as
// rebuilt (pointer churn: same geometry, fresh allocations) instead of kept.
// A green run means one middle row rebuilt and spliced while the other 59
// rows kept their boxes.

// relayoutRowSpliceRows is the wider table size: 60 rows by 4 cells, the
// small bench size in relayout_row_splice_bench_test.go. Per-row content is
// fixed (span-in-td, nested percent div, two plain text cells), the same cell
// shapes as the inventory/compare grids.
const relayoutRowSpliceRows = 60

// relayoutRowSpliceCSS fixes the table width so the percent columns have a
// basis. The tint rule paints only the patched row's cells: background color
// keeps the row's geometry, so the patch must splice instead of reporting
// errSubtreeResized.
const relayoutRowSpliceCSS = `
	table.grid { width: 640px; table-layout: fixed; border-collapse: collapse; }
	td { padding: 4px; border: 1px solid #999999; }
	td.c-a { width: 30%; }
	td.c-b { width: 30%; }
	td.c-c { width: 20%; }
	td.c-d { width: 20%; }
	tr.tint td { background-color: #fff3bf; }
	.box { width: 50%; height: 18px; background-color: #eeeeee; }
`

// relayoutRowSpliceHTML builds a rows-by-four table: every row carries a span
// in its first cell, a nested percent-width div in its second, and plain
// text in its third and fourth. Every row is numbered so the test can patch
// the middle row and prove a far row kept its boxes.
func relayoutRowSpliceHTML(rows int) string {
	var b strings.Builder

	b.WriteString(`<html><body><table class="grid">`)

	for i := range rows {
		fmt.Fprintf(&b, `<tr id="row-%02d">`, i)
		fmt.Fprintf(&b, `<td class="c-a"><span>Item %04d</span></td>`, i)
		fmt.Fprintf(&b, `<td class="c-b"><div class="box">%d pcs</div></td>`, i)
		fmt.Fprintf(&b, `<td class="c-c">%d pcs</td>`, i)
		fmt.Fprintf(&b, `<td class="c-d">9.99</td>`)
		b.WriteString(`</tr>`)
	}

	b.WriteString(`</table></body></html>`)

	return b.String()
}

// TestRelayoutRowSplicePatchesOneRowOfWideTable pins row-granular splicing on
// a 60-row table: tinting one middle row (a geometry-neutral
// background-color change) patches through the single-element subtree path
// (RelayoutSubtreeWithOverrides with no overrides, which fronts the
// subtree-reuse fingerprint check and delegates to RelayoutSubtree), matches
// a fresh full Layout, and leaves a far row's boxes untouched.
func TestRelayoutRowSplicePatchesOneRowOfWideTable(t *testing.T) {
	t.Parallel()

	rows := relayoutRowSpliceRows
	targetID := fmt.Sprintf("row-%02d", rows/2)
	farID := fmt.Sprintf("row-%02d", rows-1)

	root := mustParse(t, relayoutRowSpliceHTML(rows))
	opts := relayoutOptionsForCSS(t, relayoutRowSpliceCSS)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	farBefore := relayoutRowSpliceRowCells(t, prev, farID)
	if len(farBefore) != 4 {
		t.Fatalf("far row %q cells = %d, want 4", farID, len(farBefore))
	}

	farBoxes := make([]box, len(farBefore))
	for i, cell := range farBefore {
		farBoxes[i] = *cell
	}

	row := relayoutNodeByID(t, root, targetID)
	row.Attrs["class"] = "tint"

	patch, err := RelayoutSubtreeWithOverrides(t.Context(), prev, row, nil, opts)
	if err != nil {
		if errors.Is(err, errSubtreeResized) {
			t.Fatalf("RelayoutSubtreeWithOverrides(%s class=tint) = %v, want success "+
				"(background-color keeps geometry); the rebuilt grid moved",
				targetID, err)
		}

		if errors.Is(err, errSubtreeNotSplicable) {
			t.Fatalf("RelayoutSubtreeWithOverrides(%s class=tint) = %v, want a row-granular "+
				"splice, not a refusal", targetID, err)
		}

		t.Fatalf("RelayoutSubtreeWithOverrides(%s class=tint) = %v, want success", targetID, err)
	}

	if patch == nil || patch.rootBox == nil {
		t.Fatal("RelayoutSubtreeWithOverrides returned no patched subtree")
	}

	full, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout (full relayout): %v", err)
	}

	if !reflect.DeepEqual(relayoutStripIDs(prev.Ops), relayoutStripIDs(full.Ops)) {
		t.Fatalf("patched operations differ from a full relayout:\n got %+v\nwant %+v",
			relayoutStripIDs(prev.Ops), relayoutStripIDs(full.Ops))
	}

	relayoutRowSpliceAssertGeometry(t, prev.boxTreeRoot(), full.boxTreeRoot())

	farAfter := relayoutRowSpliceRowCells(t, prev, farID)
	if len(farAfter) != len(farBefore) {
		t.Fatalf("far row %q cells = %d, want the kept %d", farID, len(farAfter), len(farBefore))
	}

	for i := range farAfter {
		if farAfter[i] != farBefore[i] {
			t.Fatalf("far row %q cell %d was rebuilt, want the kept box: "+
				"the patch escalated to the owning table instead of splicing at row granularity",
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

// relayoutRowSpliceRowCells returns the cell boxes of one table row in
// column order. A tr places no box of its own, so the row's cells are found
// through the table's row list by matching the cell node's parent row, the
// same lookup relayoutRowSpanCellChildren uses.
func relayoutRowSpliceRowCells(t *testing.T, res *Result, rowID string) []*box {
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

// relayoutRowSpliceAssertGeometry proves the patched page kept the boxes a
// fresh full layout produces: the same tree shape with the same geometry,
// descending into table rows and cells as well as the children chain.
func relayoutRowSpliceAssertGeometry(t *testing.T, got, want *box) {
	t.Helper()

	if got == nil || want == nil {
		if got != want {
			t.Fatalf("box tree mismatch: got nil=%v, want nil=%v", got == nil, want == nil)
		}

		return
	}

	name := func(b *box) string {
		if b.node == nil {
			return "?"
		}

		if id := b.node.Attribute("id"); id != "" {
			return b.node.Name + "#" + id
		}

		return b.node.Name
	}

	if !relayoutNear(got.x, want.x) || !relayoutNear(got.y, want.y) ||
		!relayoutNear(got.w, want.w) || !relayoutNear(got.height, want.height) {
		t.Fatalf("box <%s> geometry = (%g, %g, %gx%g), want (%g, %g, %gx%g)",
			name(got), got.x, got.y, got.w, got.height,
			want.x, want.y, want.w, want.height)
	}

	if len(got.children) != len(want.children) {
		t.Fatalf("box <%s> children = %d, want %d", name(got), len(got.children), len(want.children))
	}

	if len(got.rows) != len(want.rows) {
		t.Fatalf("box <%s> rows = %d, want %d", name(got), len(got.rows), len(want.rows))
	}

	for index := range got.children {
		relayoutRowSpliceAssertGeometry(t, got.children[index], want.children[index])
	}

	for index := range got.rows {
		if len(got.rows[index]) != len(want.rows[index]) {
			t.Fatalf("box <%s> row %d cells = %d, want %d",
				name(got), index, len(got.rows[index]), len(want.rows[index]))
		}

		for cell := range got.rows[index] {
			relayoutRowSpliceAssertGeometry(t, got.rows[index][cell], want.rows[index][cell])
		}
	}
}
