package layout_test

import (
	"testing"
)

// TestFlexItemMaxContentOverflowsLane guards the marquee contract: a flex
// item with an explicit intrinsic width sizes to its content and may
// overflow a narrower container. Clamping it to the lane breaks percentage
// loop points such as a tape that translates by -50% of its own width.
func TestFlexItemMaxContentOverflowsLane(t *testing.T) {
	t.Parallel()

	const source = `<style>
.lane{width:500px;display:flex;overflow:hidden;}
.track{flex:0 0 auto;width:max-content;display:flex;}
.set{display:flex;flex-shrink:0;}
.tick{flex:0 0 auto;display:inline-block;width:60px;height:10px;}
.fit{flex:0 0 auto;width:auto;display:flex;}
</style><div class="lane"><div class="track" id="track"><div class="set" id="set1">` +
		`<span class="tick"></span><span class="tick"></span><span class="tick"></span><span class="tick"></span></div>` +
		`<div class="set" id="set2"><span class="tick"></span><span class="tick"></span><span class="tick"></span><span class="tick"></span></div>` +
		`</div></div><div class="lane"><div class="fit" id="fit"><div class="set"><span class="tick"></span><span class="tick"></span><span class="tick"></span><span class="tick"></span><span class="tick"></span><span class="tick"></span><span class="tick"></span><span class="tick"></span><span class="tick"></span><span class="tick"></span></div></div></div>`

	display := displayOf(t, source)

	byID := make(map[string]float64)
	var set1X, set1W, set2X float64

	for index := range display.Boxes {
		box := &display.Boxes[index]
		byID[box.ID] = box.W

		switch box.ID {
		case "set1":
			set1X, set1W = box.X, box.W
		case "set2":
			set2X = box.X
		}
	}

	// Four fixed 60px ticks per set: each set is 240px and the track holds
	// both, overflowing the 500px lane.
	if got := byID["set1"]; got != 240 {
		t.Fatalf("set1 width = %.1f, want 240", got)
	}

	if got := set2X - set1X; got != 240 {
		t.Fatalf("set2 offset from set1 = %.1f, want 240", got)
	}

	if got, want := byID["track"], 2*set1W; got != want {
		t.Fatalf("track width = %.1f, want both sets (%.1f)", got, want)
	}

	// width:auto keeps the fit-content clamp: the wide row still fits the
	// 500px lane instead of overflowing it.
	if got := byID["fit"]; got != 500 {
		t.Fatalf("auto flex item width = %.1f, want clamped 500", got)
	}
}

// TestFlexItemMinContentOverflowsNarrowLane guards the other explicit
// intrinsic width: min-content may overflow a lane narrower than the longest
// unbreakable word instead of crushing to the lane.
func TestFlexItemMinContentOverflowsNarrowLane(t *testing.T) {
	t.Parallel()

	const source = `<style>
.lane{width:80px;display:flex;}
.item{flex:0 0 auto;width:min-content;font-size:10px;}
</style><div class="lane"><div class="item" id="item">Supercalifragilisticexpialidocious</div></div>`

	display := displayOf(t, source)

	for index := range display.Boxes {
		box := &display.Boxes[index]
		if box.ID != "item" {
			continue
		}

		if box.W <= 80 {
			t.Fatalf("min-content item width = %.1f, want wider than the 80px lane", box.W)
		}
	}
}
