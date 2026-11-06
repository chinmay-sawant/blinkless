package layout

import "testing"

// TestPaintOnlyBatchDisplayReusesBoxes pins the paint-only display build: it
// copies the previous display and swaps in the batch's operation slice and
// retained result, so it allocates only the display struct and shares the
// boxes, breaks, keyframes, sizes, and zoom. A new []Box would be the whole
// box walk the fast path exists to avoid.
func TestPaintOnlyBatchDisplayReusesBoxes(t *testing.T) {
	//nolint:paralleltest // AllocsPerRun refuses to run inside a parallel test

	tree := fastTree(t, `<div id="box" style="width:120px;height:40px;background-color:#eeeeee">Hello</div>`)
	styled := fastStyled(t, tree)
	prev := fastRetained(t, styled)

	if prev.result == nil || len(prev.Boxes) == 0 || len(prev.result.Ops) == 0 {
		t.Fatal("the fixture carries no retained boxes or operations")
	}

	allocs := testing.AllocsPerRun(100, func() {
		_ = paintOnlyBatchDisplay(prev, prev.result)
	})

	if allocs > 1 {
		t.Fatalf("paint-only display build allocated %.1f objects per run, want at most the display", allocs)
	}

	display := paintOnlyBatchDisplay(prev, prev.result)

	if &display.Boxes[0] != &prev.Boxes[0] {
		t.Fatal("paint-only display build allocated a new box list")
	}

	if &display.Ops[0] != &prev.result.Ops[0] {
		t.Fatal("paint-only display build did not install the retained operation slice")
	}

	if display.result != prev.result {
		t.Fatal("paint-only display build dropped the retained result")
	}

	if display.Width != prev.Width || display.Height != prev.Height ||
		display.PointsPerPixel != prev.PointsPerPixel ||
		display.PixelPerPoint != prev.PixelPerPoint ||
		display.zoom != prev.zoom {
		t.Fatal("paint-only display build changed the canvas or the zoom")
	}
}
