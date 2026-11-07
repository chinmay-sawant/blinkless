package layout_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

// TestRelayoutElementsEmptyReturnsPrev pins that an empty batch is a no-op:
// it returns the input display unchanged with a nil error.
func TestRelayoutElementsEmptyReturnsPrev(t *testing.T) {
	t.Parallel()

	tree, err := html.Parse([]byte(`<div id="box" style="width:120px;height:40px">Hello</div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")
	display := layRetained(t, styled)

	got, err := layout.RelayoutElements(t.Context(), styled, display, nil, layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElements: %v", err)
	}

	if got != display {
		t.Fatalf("empty batch returned %p, want the input display %p", got, display)
	}

	got, err = layout.RelayoutElements(t.Context(), styled, display, []string{}, layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElements: %v", err)
	}

	if got != display {
		t.Fatalf("empty batch returned %p, want the input display %p", got, display)
	}
}

// TestRelayoutElementsMatchesSequentialChain pins that one batch call threads
// the display exactly as sequential RelayoutElement calls would, op for op.
func TestRelayoutElementsMatchesSequentialChain(t *testing.T) {
	t.Parallel()

	const source = `
<style>
  .card { width: 200px; height: 60px; background-color: #eeeeee; border: 2px solid #333333; }
  .card.selected { background-color: #3366cc; }
</style>
<div id="a" class="card">Alpha</div>
<div id="b" class="card">Beta</div>
<div id="c" class="card">Gamma</div>
`

	tree, err := html.Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")
	batchStart := layRetained(t, styled)
	seqStart := layRetained(t, styled)

	for _, id := range []string{"a", "b", "c"} {
		if !tree.SetAttribute(id, "class", "card selected") {
			t.Fatalf("SetAttribute found no element with id %s", id)
		}
	}

	ids := []string{"a", "b", "c"}

	batched, err := layout.RelayoutElements(t.Context(), styled, batchStart, ids, layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElements: %v", err)
	}

	current := seqStart

	for _, id := range ids {
		patch, err := layout.RelayoutElement(t.Context(), styled, current, id, layout.Options{})
		if err != nil {
			t.Fatalf("RelayoutElement %s: %v", id, err)
		}

		current = patch.Display
	}

	sequential := current

	compareDisplays(t, batched, sequential)

	if !reflect.DeepEqual(batched.OpOwners, sequential.OpOwners) {
		t.Fatalf("batch owners differ from the sequential chain")
	}

	if !reflect.DeepEqual(batched.Order, sequential.Order) {
		t.Fatalf("batch paint order differs from the sequential chain")
	}

	compareOpSlices(t, "batch against sequential", batched.Ops, sequential.Ops)
}

// TestRelayoutElementsRefusalLeavesPrevUntouched proves a size-changing
// middle id now splices the way the single call does: widening the middle
// card escalates to the nearest size-stable ancestor instead of refusing, so
// the batch and the sequential chain both succeed, agree with each other,
// and match a full relayout of the same document.
func TestRelayoutElementsRefusalLeavesPrevUntouched(t *testing.T) {
	t.Parallel()

	const source = `
<style>
  .card { width: 200px; height: 60px; background-color: #eeeeee; border: 2px solid #333333; }
  .card.selected { background-color: #3366cc; }
  .card.wide { width: 260px; }
</style>
<div id="a" class="card">Alpha</div>
<div id="b" class="card">Beta</div>
<div id="c" class="card">Gamma</div>
`

	tree, err := html.Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")
	batchStart := layRetained(t, styled)
	singleStart := layRetained(t, styled)

	beforeW := boxByID(t, batchStart, "b").W

	if !tree.SetAttribute("a", "class", "card selected") {
		t.Fatal("SetAttribute found no element with id a")
	}

	if !tree.SetAttribute("b", "class", "card wide") {
		t.Fatal("SetAttribute found no element with id b")
	}

	if !tree.SetAttribute("c", "class", "card selected") {
		t.Fatal("SetAttribute found no element with id c")
	}

	ids := []string{"a", "b", "c"}

	batched, err := layout.RelayoutElements(t.Context(), styled, batchStart, ids, layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElements: %v", err)
	}

	current := singleStart

	for _, id := range ids {
		patch, err := layout.RelayoutElement(t.Context(), styled, current, id, layout.Options{})
		if err != nil {
			t.Fatalf("RelayoutElement %s: %v", id, err)
		}

		current = patch.Display
	}

	sequential := current

	compareDisplays(t, batched, sequential)

	if !reflect.DeepEqual(batched.OpOwners, sequential.OpOwners) {
		t.Fatalf("batch owners differ from the sequential chain")
	}

	if !reflect.DeepEqual(batched.Order, sequential.Order) {
		t.Fatalf("batch paint order differs from the sequential chain")
	}

	compareOpSlices(t, "batch against sequential", batched.Ops, sequential.Ops)

	full := layOf(t, styled)
	compareDisplays(t, batched, full)

	if got := boxByID(t, batched, "b").W; got <= beforeW {
		t.Fatalf("widened card width = %g, want more than the kept %g", got, beforeW)
	}
}

// TestRelayoutElementsLargeTableCommitsAllFills builds a 200-row table
// inline and patches one block div per cell in a single batch. Every
// patched div keeps its place and commits its new background fill.
func TestRelayoutElementsLargeTableCommitsAllFills(t *testing.T) {
	t.Parallel()

	const rowCount = 200

	var page strings.Builder

	page.WriteString(`<style>
  .cell { width: 200px; height: 20px; background-color: #eeeeee; }
  .cell.selected { background-color: #3366cc; }
</style><table>`)

	ids := make([]string, 0, rowCount)

	for row := range rowCount {
		id := "cell-" + itoa(row)
		ids = append(ids, id)
		page.WriteString(`<tr><td><div id="` + id + `" class="cell">Row ` + itoa(row) + `</div></td></tr>`)
	}

	page.WriteString(`</table>`)

	tree, err := html.Parse([]byte(page.String()))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 600, 800, "", "", "")
	display := layRetained(t, styled)

	beforeFills := make(map[string][3]float64, rowCount)

	for _, id := range ids {
		rgb, found := fillRGB(opsOfElement(display, id))
		if !found {
			t.Fatalf("cell %s carries no fill before the batch", id)
		}

		beforeFills[id] = rgb
	}

	for _, id := range ids {
		if !tree.SetAttribute(id, "class", "cell selected") {
			t.Fatalf("SetAttribute found no element with id %s", id)
		}
	}

	batched, err := layout.RelayoutElements(t.Context(), styled, display, ids, layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElements: %v", err)
	}

	full := layOf(t, styled)

	if len(batched.Ops) != len(full.Ops) {
		t.Fatalf("batch ops = %d, want the %d of the full relayout", len(batched.Ops), len(full.Ops))
	}

	for _, id := range ids {
		batchRGB, found := fillRGB(opsOfElement(batched, id))
		if !found {
			t.Fatalf("cell %s carries no fill after the batch", id)
		}

		fullRGB, found := fillRGB(opsOfElement(full, id))
		if !found {
			t.Fatalf("full relayout carries no fill for cell %s", id)
		}

		if batchRGB != fullRGB {
			t.Fatalf("cell %s fill %v, want %v", id, batchRGB, fullRGB)
		}

		if batchRGB == beforeFills[id] {
			t.Fatalf("cell %s fill %v, want a change from %v", id, batchRGB, beforeFills[id])
		}
	}
}

// itoa formats a small non-negative row index without pulling in strconv for
// a test-only table builder.
func itoa(value int) string {
	if value == 0 {
		return "0"
	}

	var digits []byte

	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}

	return string(digits)
}
