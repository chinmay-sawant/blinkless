package layout_test

import (
	"reflect"
	"testing"

	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

// breaksPatchSource is a page with a heading keep range and a forced break
// above the cards, plus a paint-only class and a class that adds another
// forced break on the patched card. None of the three classes moves a box, so
// each change splices.
const breaksPatchSource = `
<style>
  .card { width: 200px; height: 60px; background-color: #eeeeee; }
  .card.selected { background-color: #3366cc; }
  .card.brk { page-break-before: always; }
</style>
<div id="lead" style="height:20px">Lead</div>
<h2 id="head">Heading</h2>
<p id="body">Body copy</p>
<div id="a" class="card">Alpha</div>
<div id="b" class="card brk">Beta</div>
`

// TestRelayoutElementsKeepsBreaksWhenPaginationCannotChange pins that a
// paint-only batch patch reuses the page-break metadata the page already
// carries, and that a patch able to change pagination recomputes it to the
// values a full layout produces.
func TestRelayoutElementsKeepsBreaksWhenPaginationCannotChange(t *testing.T) {
	t.Parallel()

	tree, err := html.Parse([]byte(breaksPatchSource))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")
	display := layRetained(t, styled)

	if len(display.ForcedBreaks) == 0 {
		t.Fatal("the fixture carries no forced break")
	}

	if len(display.KeepTogether) == 0 {
		t.Fatal("the fixture carries no keep-together range")
	}

	beforeRGB, found := fillRGB(opsOfElement(display, "a"))
	if !found {
		t.Fatal("the first layout carries no fill for the card")
	}

	if !tree.SetAttribute("a", "class", "card selected") {
		t.Fatal("SetAttribute found no element with id a")
	}

	paintOnly, err := layout.RelayoutElements(t.Context(), styled, display, []string{"a"}, layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElements: %v", err)
	}

	if afterRGB, _ := fillRGB(opsOfElement(paintOnly, "a")); afterRGB == beforeRGB {
		t.Fatalf("patched fill %v, want a change from %v", afterRGB, beforeRGB)
	}

	// The unchanged page keeps the exact slices the previous display held: a
	// recomputation would return fresh backing even when the values match.
	if &paintOnly.ForcedBreaks[0] != &display.ForcedBreaks[0] {
		t.Fatal("a paint-only patch recomputed the forced breaks")
	}

	if &paintOnly.KeepTogether[0] != &display.KeepTogether[0] {
		t.Fatal("a paint-only patch recomputed the keep-together ranges")
	}

	if !tree.SetAttribute("a", "class", "card brk") {
		t.Fatal("SetAttribute found no element with id a")
	}

	withBreak, err := layout.RelayoutElements(t.Context(), styled, paintOnly, []string{"a"}, layout.Options{})
	if err != nil {
		t.Fatalf("RelayoutElements: %v", err)
	}

	if len(withBreak.ForcedBreaks) != len(display.ForcedBreaks)+1 {
		t.Fatalf("forced breaks %v, want one more than %v",
			withBreak.ForcedBreaks, display.ForcedBreaks)
	}

	full := layOf(t, styled)

	if !reflect.DeepEqual(withBreak.ForcedBreaks, full.ForcedBreaks) {
		t.Fatalf("forced breaks %v, want the full layout %v",
			withBreak.ForcedBreaks, full.ForcedBreaks)
	}

	if !reflect.DeepEqual(withBreak.KeepTogether, full.KeepTogether) {
		t.Fatalf("keep-together ranges %v, want the full layout %v",
			withBreak.KeepTogether, full.KeepTogether)
	}
}
