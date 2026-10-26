package layout_test

import (
	"testing"
)

// TestAbsoluteTopBottomStretches guards CSS 2.1 §10.6.4: an absolutely
// positioned box with auto height and both top and bottom set fills the
// containing block instead of collapsing to its content. Overlays that sweep
// across a bar (sheens, scan lines) depend on the stretched height; at zero
// height their animations paint nothing.
func TestAbsoluteTopBottomStretches(t *testing.T) {
	t.Parallel()

	const source = `<style>
.row{position:relative;height:24px;display:block;}
.stretch{position:absolute;top:0;bottom:0;left:0;width:38%;}
.explicit{position:absolute;top:0;bottom:0;left:0;width:38%;height:10px;}
.toponly{position:absolute;top:0;left:0;width:38%;}
</style><div class="row"><span>Bid</span><span class="stretch" id="stretch"></span></div>
<div class="row"><span>Bid</span><span class="explicit" id="explicit"></span></div>
<div class="row"><span>Bid</span><span class="toponly" id="toponly"></span></div>`

	display := displayOf(t, source)

	heights := make(map[string]float64)

	for index := range display.Boxes {
		box := &display.Boxes[index]
		heights[box.ID] = box.H
	}

	// 24pt row, no chrome on the overlays: top:0 plus bottom:0 fills it.
	if got := heights["stretch"]; got != 24 {
		t.Fatalf("stretched overlay height = %.1f, want 24", got)
	}

	// An explicit height still wins over the top/bottom stretch.
	if got := heights["explicit"]; got != 10 {
		t.Fatalf("explicit-height overlay = %.1f, want 10", got)
	}

	// With only top set the box keeps its content height (empty here).
	if got := heights["toponly"]; got != 0 {
		t.Fatalf("top-only overlay height = %.1f, want 0", got)
	}
}
