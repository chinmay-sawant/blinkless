package layout

import (
	"runtime"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// TestSubtreeTagCoversEverySheetAndSurvivesAddressReuse: the tag must differ
// when a later sheet differs (only Sheets[0] used to count), and a collected
// sheet's tag must not equal a new sheet's, whatever its address.
func TestSubtreeTagCoversEverySheetAndSurvivesAddressReuse(t *testing.T) {
	t.Parallel()

	first, err := css.Parse(`.a{color:red}`)
	if err != nil {
		t.Fatal(err)
	}

	secondA, err := css.Parse(`.b{color:blue}`)
	if err != nil {
		t.Fatal(err)
	}

	secondB, err := css.Parse(`.b{color:green}`)
	if err != nil {
		t.Fatal(err)
	}

	tagA := subtreeStyleDocTagOf(nil, Options{Sheets: []*css.Stylesheet{first, secondA}})
	tagB := subtreeStyleDocTagOf(nil, Options{Sheets: []*css.Stylesheet{first, secondB}})

	if tagA.equal(tagB) {
		t.Fatal("tags ignore every sheet after the first")
	}

	if !tagA.equal(subtreeStyleDocTagOf(nil, Options{Sheets: []*css.Stylesheet{first, secondA}})) {
		t.Fatal("identical documents must tag equal")
	}

	// Drop secondA, collect it, and allocate a replacement: even if the
	// allocator reuses the address, the old tag must not match.
	oldTag := tagA
	secondA = nil               //nolint:ineffassign,wastedassign // drops the last reference so the GC can collect the sheet
	tagA = subtreeStyleDocTag{} //nolint:wastedassign // drops the tag's copy of the sheet reference

	for range 5 {
		runtime.GC()
	}

	replacement, err := css.Parse(`.b{color:blue}`)
	if err != nil {
		t.Fatal(err)
	}

	newTag := subtreeStyleDocTagOf(nil, Options{Sheets: []*css.Stylesheet{first, replacement}})
	if oldTag.equal(newTag) {
		t.Fatal("a recycled sheet address matched a collected sheet's tag")
	}

	runtime.KeepAlive(first)
	runtime.KeepAlive(replacement)
}
