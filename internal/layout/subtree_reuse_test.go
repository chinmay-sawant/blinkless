package layout

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// TestReuseClassOnlyEditReusesOps proves the paint-only fast path: a class
// change that only repaints keeps the operation identities and matches a
// full relayout. The counter (not timing) proves the fast path ran.
func TestReuseClassOnlyEditReusesOps(t *testing.T) {
	root := mustParse(t, relayoutFixtureHTML)
	opts := relayoutOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	firstBox := relayoutBoxByID(t, prev, "first")
	oldStart, oldEnd := firstBox.opStart, firstBox.opEnd
	oldCount := oldEnd - oldStart + 1

	oldIDs := make([]uint64, 0, oldCount)
	for _, op := range prev.Ops[oldStart : oldEnd+1] {
		oldIDs = append(oldIDs, op.ID)
	}

	before := ReuseStatsSnapshot()

	first := relayoutNodeByID(t, root, "first")
	first.Attrs["class"] = "card selected"

	patch, err := RelayoutSubtreeWithOverrides(t.Context(), prev, first, nil, opts)
	if err != nil {
		t.Fatalf("RelayoutSubtreeWithOverrides: %v", err)
	}

	stats := ReuseStatsSnapshot()
	if stats.Hits-before.Hits != 1 {
		t.Fatalf("reuse hits delta = %d, want 1", stats.Hits-before.Hits)
	}

	if len(patch.Ops) != oldCount {
		t.Fatalf("patched operations = %d, want the kept %d", len(patch.Ops), oldCount)
	}

	for index, op := range patch.Ops {
		if op.ID != oldIDs[index] {
			t.Fatalf("operation %d ID = %d, want the kept %d", index, op.ID, oldIDs[index])
		}
	}

	fill, ok := relayoutFirstFill(patch.Ops)
	if !ok {
		t.Fatal("the patch carries no fill operation")
	}

	if fill != selectedBlue {
		t.Fatalf("patched background = %v, want the selected blue", fill)
	}

	relayoutAssertMatchesFullRelayout(t, root, opts, patch, "first")
}

// TestReuseTextEditChangingSizeFallsBack proves a content edit that changes
// the box size leaves the fast path and reports the documented full-layout
// signal instead of reusing stale operations.
func TestReuseTextEditChangingSizeFallsBack(t *testing.T) {
	const cssText = `.grow { width: 200px; background-color: #eeeeee; }`
	root := mustParse(t, `<html><body>`+
		`<div id="target" class="grow">Alpha</div>`+
		`<div id="other" class="grow">Beta</div>`+
		`</body></html>`)
	opts := relayoutOptionsForCSS(t, cssText)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	target := relayoutNodeByID(t, root, "target")
	target.Children[0].Text = strings.Repeat("lorem ipsum dolor sit amet ", 20)

	before := ReuseStatsSnapshot()

	_, err = RelayoutSubtreeWithOverrides(t.Context(), prev, target, nil, opts)
	if !errors.Is(err, errSubtreeResized) {
		t.Fatalf("error = %v, want %v", err, errSubtreeResized)
	}

	stats := ReuseStatsSnapshot()
	if stats.Misses-before.Misses < 1 {
		t.Fatalf("reuse misses delta = %d, want at least 1", stats.Misses-before.Misses)
	}

	if stats.Hits-before.Hits != 0 {
		t.Fatalf("reuse hits delta = %d, want 0 on a size change", stats.Hits-before.Hits)
	}
}

// TestReuseMovedSiblingTranslates proves coordinate translation moves a
// clean sibling without rebuilding it, matches a full layout of the shifted
// page, and round-trips. It also proves the batch entry (overrides) falls
// back to the normal build instead of reusing.
func TestReuseMovedSiblingTranslates(t *testing.T) {
	root := mustParse(t, relayoutFixtureHTML)
	opts := relayoutOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	secondBox := relayoutBoxByID(t, prev, "second")
	winStart, winEnd := secondBox.opStart, secondBox.opEnd
	before := cloneOps(prev.Ops)

	countsBefore := ReuseStatsSnapshot()

	const dx, dy = 0.0, 15.5

	if !translateOpsRange(prev.Ops, winStart, winEnd, dx, dy) {
		t.Fatal("translateOpsRange refused a plain fill and text window")
	}

	stats := ReuseStatsSnapshot()
	if stats.Translated-countsBefore.Translated != 1 {
		t.Fatalf("reuse translations delta = %d, want 1", stats.Translated-countsBefore.Translated)
	}

	for index := winStart; index <= winEnd; index++ {
		if prev.Ops[index].ID != before[index].ID {
			t.Fatalf("operation %d changed identity on translation", index)
		}

		if prev.Ops[index].X != before[index].X+dx || prev.Ops[index].Y != before[index].Y+dy {
			t.Fatalf("operation %d = (%g, %g), want (%g, %g)",
				index, prev.Ops[index].X, prev.Ops[index].Y,
				before[index].X+dx, before[index].Y+dy)
		}
	}

	for index := range prev.Ops {
		if index >= winStart && index <= winEnd {
			continue
		}

		if !reflect.DeepEqual(prev.Ops[index], before[index]) {
			t.Fatalf("operation %d outside the window changed", index)
		}
	}

	if !translateOpsRange(prev.Ops, winStart, winEnd, -dx, -dy) {
		t.Fatal("translateOpsRange refused the way back")
	}

	if !reflect.DeepEqual(prev.Ops, before) {
		t.Fatal("translating back did not restore the display list")
	}

	shifted := mustParse(t, `<html><body>`+
		`<div id="first" class="card">Alpha</div>`+
		`<div id="second" class="card" style="margin-top:20px">Beta</div>`+
		`</body></html>`)
	fullShifted, err := Layout(shifted, opts)
	if err != nil {
		t.Fatalf("Layout (shifted): %v", err)
	}

	plain := mustParse(t, relayoutFixtureHTML)
	plainRes, err := Layout(plain, opts)
	if err != nil {
		t.Fatalf("Layout (plain): %v", err)
	}

	plainSecond := plainRes.boxForID("second")
	shiftedSecond := fullShifted.boxForID("second")
	if plainSecond == nil || shiftedSecond == nil {
		t.Fatal("missing second box in one of the layouts")
	}

	shiftY := shiftedSecond.y - plainSecond.y
	if shiftY <= 0 {
		t.Fatalf("shifted second y = %g, want it below %g", shiftedSecond.y, plainSecond.y)
	}

	moved := cloneOps(plainRes.Ops)
	if !translateOpsRange(moved, plainSecond.opStart, plainSecond.opEnd, 0, shiftY) {
		t.Fatal("translateOpsRange refused the golden shift")
	}

	shiftedBox := fullShifted.boxForID("second")
	got := relayoutStripIDs(moved[plainSecond.opStart : plainSecond.opEnd+1])
	want := relayoutStripIDs(fullShifted.Ops[shiftedBox.opStart : shiftedBox.opEnd+1])

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("translated operations differ from a full layout of the shifted page:\n got %+v\nwant %+v", got, want)
	}
}

// TestReuseBatchOverrideFallsBack proves the batch entry (a width override)
// does not reuse: overrides name layout properties, so the attempt records a
// miss and the normal override build runs.
func TestReuseBatchOverrideFallsBack(t *testing.T) {
	root := mustParse(t, overrideFlexHTML)
	opts := overrideOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	before := ReuseStatsSnapshot()

	card := relayoutNodeByID(t, root, "card")

	_, err = RelayoutSubtreeWithOverrides(t.Context(), prev, card,
		[]StyleOverride{{Property: "width", Value: "180px"}}, opts)
	if err != nil {
		t.Fatalf("RelayoutSubtreeWithOverrides: %v", err)
	}

	stats := ReuseStatsSnapshot()
	if stats.Misses-before.Misses < 1 {
		t.Fatalf("reuse misses delta = %d, want at least 1 for a layout override", stats.Misses-before.Misses)
	}

	if stats.Hits-before.Hits != 0 {
		t.Fatalf("reuse hits delta = %d, want 0 for a layout override", stats.Hits-before.Hits)
	}
}

// TestReuseGoldenParityFewHundredRows proves the patched page equals a full
// layout on a few-hundred-row document: same operations (up to identity),
// after a paint-only edit to one row.
func TestReuseGoldenParityFewHundredRows(t *testing.T) {
	const rows = 300
	const cssText = `
		.row { width: 400px; height: 20px; background-color: #eeeeee; }
		.row.sel { background-color: #3366cc; }
	`

	var body strings.Builder
	for index := range rows {
		body.WriteString(`<div id="row-` + itoa(index) + `" class="row">Row ` + itoa(index) + `</div>`)
	}

	root := mustParse(t, `<html><body>`+body.String()+`</body></html>`)
	opts := relayoutOptionsForCSS(t, cssText)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	middle := relayoutNodeByID(t, root, "row-150")
	middle.Attrs["class"] = "row sel"

	countsBefore := ReuseStatsSnapshot()

	patch, err := RelayoutSubtreeWithOverrides(t.Context(), prev, middle, nil, opts)
	if err != nil {
		t.Fatalf("RelayoutSubtreeWithOverrides: %v", err)
	}

	if stats := ReuseStatsSnapshot(); stats.Hits-countsBefore.Hits != 1 {
		t.Fatalf("reuse hits delta = %d, want 1", stats.Hits-countsBefore.Hits)
	}

	if len(patch.Ops) == 0 {
		t.Fatal("the patch carries no operations")
	}

	full, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout (full): %v", err)
	}

	if len(prev.Ops) != len(full.Ops) {
		t.Fatalf("patched page carries %d operations, want the %d of a full layout",
			len(prev.Ops), len(full.Ops))
	}

	if !reflect.DeepEqual(relayoutStripIDs(prev.Ops), relayoutStripIDs(full.Ops)) {
		t.Fatal("patched page differs from a full layout")
	}
}

// itoa formats a small non-negative index without importing strconv for one
// test helper.
func itoa(index int) string {
	if index == 0 {
		return "0"
	}

	var digits []byte
	for value := index; value > 0; value /= 10 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
	}

	return string(digits)
}
