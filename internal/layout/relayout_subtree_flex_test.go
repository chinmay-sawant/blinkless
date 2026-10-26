package layout

import (
	"errors"
	"strings"
	"testing"
)

// relayoutFlexGrowCSS is the shape the trade blotter uses: a definite column
// flex container, a rows container that takes the free block space, and rows
// that share that space equally through flex sizing. A row's content is one
// text line, so its used height is several times its content height.
const relayoutFlexGrowCSS = `
	html, body { margin: 0; height: 900px; }
	#app { display: flex; flex-direction: column; height: 900px; }
	#rows { flex: 1 1 auto; min-height: 0; display: flex; flex-direction: column; overflow: hidden; }
	.row { flex: 1 1 0; min-height: 0; border-bottom: 1px solid #222222; }
	.row.hot { background-color: #eeeeee; }
`

// relayoutFlexGrowHTML holds three rows in the flex column.
const relayoutFlexGrowHTML = `<html><body><div id="app"><div id="rows">` +
	`<div class="row" id="row-a">a</div>` +
	`<div class="row" id="row-b">b</div>` +
	`<div class="row" id="row-c">c</div>` +
	`</div></div></body></html>`

// relayoutFlexStretchCSS is a row flex container whose item has an auto
// height: the item's used block size is the line's cross size, not its
// content height.
const relayoutFlexStretchCSS = `
	#strip { display: flex; flex-direction: row; height: 120px; }
	.cell { flex: 1 1 0; border-left: 1px solid #222222; }
	.cell.hot { background-color: #eeeeee; }
`

// relayoutFlexStretchHTML holds one stretched item next to another.
const relayoutFlexStretchHTML = `<html><body><div id="strip">` +
	`<div class="cell" id="cell-a">a</div>` +
	`<div class="cell" id="cell-b">b</div>` +
	`</div></body></html>`

// relayoutContentCSS gives a block a width that makes text wrap, so the
// element's height is the one its content measures.
const relayoutContentCSS = `
	#box { width: 120px; font-size: 16px; }
	#box.hot { background-color: #eeeeee; }
`

// relayoutContentHTML holds one content-sized block.
const relayoutContentHTML = `<html><body><div id="box">short</div></body></html>`

// relayoutIndefiniteFlexCSS is a column flex container without a definite
// height: every item keeps the height its content measures.
const relayoutIndefiniteFlexCSS = `
	#col { display: flex; flex-direction: column; }
	#item { min-height: 0; }
	#item.hot { background-color: #eeeeee; }
`

// relayoutIndefiniteFlexHTML holds one item in a content-sized column.
const relayoutIndefiniteFlexHTML = `<html><body><div id="col"><div id="item">short</div></div></body></html>`

// relayoutDefiniteFlexCSS is a column flex container with a definite height
// and free space: the item does not grow, so its height stays the one its
// content measures.
const relayoutDefiniteFlexCSS = `
	#col { display: flex; flex-direction: column; height: 300px; }
	#item { min-height: 0; }
	#item.hot { background-color: #eeeeee; }
`

// relayoutDefiniteFlexHTML holds one item in a definite-height column that
// leaves free space below it.
const relayoutDefiniteFlexHTML = `<html><body><div id="col"><div id="item">short</div></div></body></html>`

// relayoutFlexGrowOptions builds the layout options of the flex column
// fixture.
func relayoutFlexGrowOptions(t *testing.T) Options {
	t.Helper()

	return relayoutOptionsForCSS(t, relayoutFlexGrowCSS)
}

// TestRelayoutSubtreePatchesAFlexGrownRow proves the splice accepts a row
// whose used height the flex container assigned: the row keeps the height it
// had even though its content is one short line, so a hover patch applies
// instead of falling back to a full layout.
func TestRelayoutSubtreePatchesAFlexGrownRow(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutFlexGrowHTML)
	opts := relayoutFlexGrowOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	rowBefore := *relayoutBoxByID(t, prev, "row-a")

	// The fixture really is the stretched case: three rows share the
	// container, so each row is far taller than its single text line.
	if rowBefore.height <= 100 {
		t.Fatalf("row height = %g, want the flex-assigned height", rowBefore.height)
	}

	row := relayoutNodeByID(t, root, "row-a")
	row.Attrs["class"] = "row hot"

	patch, err := RelayoutSubtree(t.Context(), prev, row, opts)
	if err != nil {
		t.Fatalf("RelayoutSubtree: %v", err)
	}

	patched := patch.rootBox
	if patched.height != rowBefore.height {
		t.Fatalf("patched row height = %g, want the kept %g", patched.height, rowBefore.height)
	}

	if patched.x != rowBefore.x || patched.y != rowBefore.y || patched.w != rowBefore.w {
		t.Fatalf("patched row moved: got (%g, %g) %g wide, want (%g, %g) %g wide",
			patched.x, patched.y, patched.w, rowBefore.x, rowBefore.y, rowBefore.w)
	}

	fill, ok := relayoutFirstFill(patch.Ops)
	if !ok {
		t.Fatal("the patch carries no fill operation")
	}

	if fill != [3]float64{238.0 / 255, 238.0 / 255, 238.0 / 255} {
		t.Fatalf("patched background = %v, want the hover grey", fill)
	}
}

// TestRelayoutSubtreePatchesAFlexCrossStretchedCell proves the same rule on
// the row axis: an item stretched to the line's cross size keeps that size
// when its subtree is patched.
func TestRelayoutSubtreePatchesAFlexCrossStretchedCell(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutFlexStretchHTML)
	opts := relayoutOptionsForCSS(t, relayoutFlexStretchCSS)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	cellBefore := *relayoutBoxByID(t, prev, "cell-a")

	strip := relayoutBoxByID(t, prev, "strip")
	if cellBefore.height != strip.height {
		t.Fatalf("cell height = %g, want the line's %g", cellBefore.height, strip.height)
	}

	cell := relayoutNodeByID(t, root, "cell-a")
	cell.Attrs["class"] = "cell hot"

	patch, err := RelayoutSubtree(t.Context(), prev, cell, opts)
	if err != nil {
		t.Fatalf("RelayoutSubtree: %v", err)
	}

	if patch.rootBox.height != cellBefore.height {
		t.Fatalf("patched cell height = %g, want the kept %g", patch.rootBox.height, cellBefore.height)
	}

	relayoutAssertMatchesFullRelayout(t, root, opts, patch, "cell-a")
}

// TestRelayoutSubtreeFlexGrownRowMatchesAFullRelayout proves the assigned
// height does not change the display list: the patched row paints exactly
// what a full relayout of the same document paints.
func TestRelayoutSubtreeFlexGrownRowMatchesAFullRelayout(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutFlexGrowHTML)
	opts := relayoutFlexGrowOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	rowBefore := *relayoutBoxByID(t, prev, "row-a")

	row := relayoutNodeByID(t, root, "row-a")
	row.Attrs["class"] = "row hot"

	patch, err := RelayoutSubtree(t.Context(), prev, row, opts)
	if err != nil {
		t.Fatalf("RelayoutSubtree: %v", err)
	}

	relayoutAssertMatchesFullRelayout(t, root, opts, patch, "row-a")

	full, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout (full relayout): %v", err)
	}

	fullRow := relayoutBoxByID(t, full, "row-a")
	if !relayoutSameGeometry(patch.rootBox, fullRow) {
		t.Fatalf("patched row = (%g, %g) %gx%g, want the full relayout's (%g, %g) %gx%g",
			patch.rootBox.x, patch.rootBox.y, patch.rootBox.w, patch.rootBox.height,
			fullRow.x, fullRow.y, fullRow.w, fullRow.height)
	}

	if patch.rootBox.height != rowBefore.height {
		t.Fatalf("patched row height = %g, want the kept %g",
			patch.rootBox.height, rowBefore.height)
	}
}

// TestRelayoutSubtreeRejectsAContentHeightChange proves the size guard keeps
// its power for elements whose height their content decides: a block that
// grows when its text wraps, and a flex item in a column without a definite
// height, are both refused.
func TestRelayoutSubtreeRejectsAContentHeightChange(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		html string
		css  string
		id   string
	}{
		{name: "block", html: relayoutContentHTML, css: relayoutContentCSS, id: "box"},
		{name: "indefinite flex column", html: relayoutIndefiniteFlexHTML, css: relayoutIndefiniteFlexCSS, id: "item"},
		{
			name: "definite flex column with free space",
			html: relayoutDefiniteFlexHTML,
			css:  relayoutDefiniteFlexCSS,
			id:   "item",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := mustParse(t, testCase.html)
			opts := relayoutOptionsForCSS(t, testCase.css)

			prev, err := Layout(root, opts)
			if err != nil {
				t.Fatalf("Layout: %v", err)
			}

			before := *relayoutBoxByID(t, prev, testCase.id)

			node := relayoutNodeByID(t, root, testCase.id)
			node.Attrs["class"] = "hot"
			node.Children[0].Text = strings.Repeat("wrapping text ", 20)

			_, err = RelayoutSubtree(t.Context(), prev, node, opts)
			if !errors.Is(err, errSubtreeResized) {
				t.Fatalf("error = %v, want %v (the height grew from %g)",
					err, errSubtreeResized, before.height)
			}
		})
	}
}
