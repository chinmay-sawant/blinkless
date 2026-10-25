package layout_test

import (
	"testing"
)

// TestFlexAbsoluteChildOverlays guards the out-of-flow contract for flex
// containers (CSS Flexbox §4): an absolutely positioned child takes no line
// space and positions against the flex container's padding box. Laying it
// out as a flex item collapses overlays such as sheen sweeps to zero height
// at the item's static position, and their animations paint nothing.
func TestFlexAbsoluteChildOverlays(t *testing.T) {
	t.Parallel()

	const source = `<style>
.row{position:relative;height:24px;display:flex;align-items:center;}
.sheen{position:absolute;top:0;bottom:0;left:0;width:38%;}
</style><div class="row" id="row"><span id="label">Bid</span><span class="sheen" id="sheen"></span></div>`

	display := displayOf(t, source)

	boxes := make(map[string]struct{ x, y, w, h float64 })
	for index := range display.Boxes {
		box := &display.Boxes[index]
		if box.ID == "row" || box.ID == "sheen" || box.ID == "label" {
			boxes[box.ID] = struct{ x, y, w, h float64 }{box.X, box.Y, box.W, box.H}
		}
	}

	row, sheen, label := boxes["row"], boxes["sheen"], boxes["label"]

	// The overlay fills the row's height from the row's content edge.
	if sheen.w != 0.38*row.w {
		t.Fatalf("overlay width = %.1f, want 38%% of row (%.1f)", sheen.w, 0.38*row.w)
	}

	if sheen.h != row.h {
		t.Fatalf("overlay height = %.1f, want row height %.1f", sheen.h, row.h)
	}

	if sheen.x != row.x || sheen.y != row.y {
		t.Fatalf("overlay at (%.1f,%.1f), want row content edge (%.1f,%.1f)", sheen.x, sheen.y, row.x, row.y)
	}

	// The overlay takes no line space: the label starts at the row edge.
	if label.x != row.x {
		t.Fatalf("label x = %.1f, want row edge %.1f (overlay must not consume line space)", label.x, row.x)
	}
}
