package layout

import (
	"errors"
	"reflect"
	"testing"
)

// NOTE: this test is EXPECTED TO FAIL until the engine fix lands. Patching a
// table row to `visibility: hidden` refuses with errSubtreeResized even though
// geometry is unchanged: the rebuilt row subtree drops a `<span>` child box
// that the full layout keeps (sameBoxGeometry count mismatch on a `td`: kept
// `[span]` vs rebuilt `[]`). A plain-text-cell table patches fine, so the
// trigger is specifically element children (span) inside the patched row's
// cells.

// relayoutRowSpanCSS fixes the table width so a paint-only row patch cannot
// shift columns, and gives one cell nested percent-width content mimicking the
// inventory stock row.
const relayoutRowSpanCSS = `
	body { margin: 0; font-size: 10pt; }
	table { width: 400px; border-collapse: collapse; }
	td { border: 1px solid #999; padding: 4px; }
	.meter { width: 100%; background-color: #eeeeee; }
	.fill { width: 42%; background-color: #3366cc; }
	.badge { display: inline-block; background-color: #3366cc; }
`

// relayoutRowSpanHTML is a 2 row x 3 cell table. The patched row holds one
// cell with a span element child (the dropped box), one cell with a nested
// percent-width div, and one plain-text cell.
const relayoutRowSpanHTML = `<html><body><table>` +
	`<tr id="target">` +
	`<td><span id="badge" class="badge">In stock</span></td>` +
	`<td><div class="meter"><div class="fill">42%</div></div></td>` +
	`<td>plain</td>` +
	`</tr>` +
	`<tr><td>a</td><td>b</td><td>c</td></tr>` +
	`</table></body></html>`

// TestRelayoutRowSpanVisibilityPatch pins the inventory-app repro: hiding a
// row whose cell holds a span must splice through the single-element subtree
// path (RelayoutSubtree, the element patch path behind the public
// RelayoutElement entry) and match a fresh full Layout of the same tree.
func TestRelayoutRowSpanVisibilityPatch(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutRowSpanHTML)
	opts := relayoutOptionsForCSS(t, relayoutRowSpanCSS)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	row := relayoutNodeByID(t, root, "target")
	row.Attrs["style"] = "visibility: hidden"

	patch, err := RelayoutSubtree(t.Context(), prev, row, opts)
	if err != nil {
		if errors.Is(err, errSubtreeResized) {
			t.Fatalf("RelayoutSubtree(row#target style=visibility:hidden) = %v, want success "+
				"(hiding keeps geometry); kept cells hold children %q: the span-bearing td "+
				"keeps [span] in the full layout but the rebuilt row subtree drops it "+
				"(kept [span] vs rebuilt [])",
				err, relayoutRowSpanCellChildren(t, prev, "target"))
		}

		t.Fatalf("RelayoutSubtree(row#target style=visibility:hidden) = %v, want success", err)
	}

	if patch == nil || patch.rootBox == nil {
		t.Fatal("RelayoutSubtree returned no patched subtree")
	}

	full, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout (full relayout): %v", err)
	}

	if !reflect.DeepEqual(relayoutStripIDs(prev.Ops), relayoutStripIDs(full.Ops)) {
		t.Fatalf("patched operations differ from a full relayout:\n got %+v\nwant %+v",
			relayoutStripIDs(prev.Ops), relayoutStripIDs(full.Ops))
	}

	relayoutRowSpanAssertGeometry(t, prev.boxTreeRoot(), full.boxTreeRoot())
}

// relayoutRowSpanCellChildren lists, in row-major order, the element names of
// the target row's cell child boxes, so a red run names the cell that lost a
// child in the rebuild.
func relayoutRowSpanCellChildren(t *testing.T, res *Result, rowID string) []string {
	t.Helper()

	table := res.boxTreeRoot()
	if table == nil {
		t.Fatal("no box tree in the retained result")
	}

	// A tr places no box of its own, so its cells are found through the
	// table's row list by matching the cell node's parent row.
	var out []string

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

				cellName := cell.node.Name
				if id := cell.node.Attribute("id"); id != "" {
					cellName += "#" + id
				}

				kids := make([]string, 0, len(cell.children))
				for _, child := range cell.children {
					if child.node != nil {
						kids = append(kids, child.node.Name)
					} else {
						kids = append(kids, "?")
					}
				}

				out = append(out, cellName+"->[")
				out = append(out, kids...)
				out = append(out, "]")
			}
		}

		for _, child := range b.children {
			walk(child)
		}
	}
	walk(table)

	return out
}

// relayoutRowSpanAssertGeometry proves the patched page kept the boxes a
// fresh full layout produces: the same tree shape with the same geometry,
// descending into table rows and cells as well as the children chain.
func relayoutRowSpanAssertGeometry(t *testing.T, got, want *box) {
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
		relayoutRowSpanAssertGeometry(t, got.children[index], want.children[index])
	}

	for index := range got.rows {
		if len(got.rows[index]) != len(want.rows[index]) {
			t.Fatalf("box <%s> row %d cells = %d, want %d",
				name(got), index, len(got.rows[index]), len(want.rows[index]))
		}

		for cell := range got.rows[index] {
			relayoutRowSpanAssertGeometry(t, got.rows[index][cell], want.rows[index][cell])
		}
	}
}
