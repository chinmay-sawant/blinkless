package layout_test

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

// tablePatchSource is a fixed-layout table: the columns come from the table
// width, so a short cell edit keeps every column where it stood, and the
// patch must splice instead of refusing.
const tablePatchSource = `<html><body>` +
	`<table id="grid" style="table-layout:fixed;width:200px"><tr id="row">` +
	`<td id="cell"><span id="inner">Alpha</span></td>` +
	`<td>Beta</td>` +
	`</tr></table>` +
	`<div id="after">After</div>` +
	`</body></html>`

// tablePatchAutoSource is an auto-width table: a long unbreakable edit widens
// a column, so the grid moves and the patch must refuse.
const tablePatchAutoSource = `<html><body>` +
	`<table id="grid"><tr id="row">` +
	`<td id="cell"><span id="inner">Alpha</span></td>` +
	`<td>Beta</td>` +
	`</tr></table>` +
	`<div id="after">After</div>` +
	`</body></html>`

// layTableRetained parses source and lays it out with retained state.
func layTableRetained(t *testing.T, source string) (*html.Document, *layout.Display) {
	t.Helper()

	tree, err := html.Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")

	return tree, layRetained(t, styled)
}

// displayTextPresent reports whether any text operation carries text.
func displayTextPresent(display *layout.Display, text string) bool {
	for i := range display.Ops {
		if display.Ops[i].Kind == layout.DisplayOpText && display.Ops[i].Text == text {
			return true
		}
	}

	return false
}

// TestRelayoutElementSplicesFittingCellText proves a td text edit that does
// not move the table grid splices in place instead of refusing.
func TestRelayoutElementSplicesFittingCellText(t *testing.T) {
	t.Parallel()

	tree, display := layTableRetained(t, tablePatchSource)

	if !tree.SetText("cell", "Gamma") {
		t.Fatal("SetText reported no element")
	}

	patch, err := layout.RelayoutElement(
		t.Context(), mustApply(t, tree, 400, 300, "", "", ""), display, "cell", layout.Options{},
	)
	if err != nil {
		t.Fatalf("relayout element: %v", err)
	}

	if got := patchText(patch); got != "Gamma" {
		t.Fatalf("patch text = %q, want %q", got, "Gamma")
	}

	if patch.Old.Empty() || patch.New.Empty() {
		t.Fatalf("patch bounds = %v/%v, want the cell box", patch.Old, patch.New)
	}

	if patch.Old != patch.New {
		t.Fatalf("patch moved the cell: old %v, new %v", patch.Old, patch.New)
	}

	if !displayTextPresent(patch.Display, "Gamma") {
		t.Fatal("adopted display does not paint the edited text")
	}
}

// TestRelayoutElementSplicesRowClassChange proves a tr attribute edit that
// keeps the grid splices: the row owns no box, so the patch covers the table.
func TestRelayoutElementSplicesRowClassChange(t *testing.T) {
	t.Parallel()

	tree, display := layTableRetained(t, tablePatchSource)

	if !tree.SetAttribute("row", "class", "highlight") {
		t.Fatal("SetAttribute reported no element")
	}

	patch, err := layout.RelayoutElement(
		t.Context(), mustApply(t, tree, 400, 300, "", "", ""), display, "row", layout.Options{},
	)
	if err != nil {
		t.Fatalf("relayout element: %v", err)
	}

	if len(patch.Ops) == 0 {
		t.Fatal("patch has no operations")
	}

	if patch.Old.Empty() || patch.New.Empty() {
		t.Fatalf("patch bounds = %v/%v, want the table damage", patch.Old, patch.New)
	}
}

// TestRelayoutElementSplicesSpanInsideCell proves a text edit of inline
// content inside a cell splices: the span owns no box, so the patch covers
// the table.
func TestRelayoutElementSplicesSpanInsideCell(t *testing.T) {
	t.Parallel()

	tree, display := layTableRetained(t, tablePatchSource)

	if !tree.SetText("inner", "Gamma") {
		t.Fatal("SetText reported no element")
	}

	patch, err := layout.RelayoutElement(
		t.Context(), mustApply(t, tree, 400, 300, "", "", ""), display, "inner", layout.Options{},
	)
	if err != nil {
		t.Fatalf("relayout element: %v", err)
	}

	if got := patchText(patch); got != "Gamma" {
		t.Fatalf("patch text = %q, want %q", got, "Gamma")
	}

	if patch.Old.Empty() || patch.New.Empty() {
		t.Fatalf("patch bounds = %v/%v, want the table damage", patch.Old, patch.New)
	}

	if !displayTextPresent(patch.Display, "Gamma") {
		t.Fatal("adopted display does not paint the edited text")
	}
}

// TestRelayoutElementRefusesMovedTableGrid proves an edit that widens a
// column still refuses: only a grid that does not move splices.
func TestRelayoutElementRefusesMovedTableGrid(t *testing.T) {
	t.Parallel()

	tree, display := layTableRetained(t, tablePatchAutoSource)

	if !tree.SetText("cell", "averylongunbreakablewordthatforcescolumnstogrow") {
		t.Fatal("SetText reported no element")
	}

	_, err := layout.RelayoutElement(
		t.Context(), mustApply(t, tree, 400, 300, "", "", ""), display, "cell", layout.Options{},
	)
	if err == nil {
		t.Fatal("relayout element succeeded, want the grid-move refusal")
	}

	if got := err.Error(); !strings.Contains(got, "changed size") {
		t.Fatalf("error = %q, want the grid-move refusal", got)
	}
}
