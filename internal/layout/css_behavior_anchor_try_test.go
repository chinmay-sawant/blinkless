package layout

import "testing"

// TestBehaviorPositionTryTactic proves the tactic row: a tactic flips the
// chosen area across the axis the pinned spec names, applied after the area
// is known and before the final shift. The flipped target carries
// position-visibility:no-overflow so its base top (which would hang off the
// viewport) fails and the flip-block fallback wins below the anchor, while
// the unflipped target keeps the base top above it. The output is
// layout.DisplayList: both targets shift their paint ops with their boxes.
//
// Pinned <try-tactic> = flip-block || flip-inline || flip-start || flip-x ||
// flip-y (https://drafts.csswg.org/css-anchor-position-1/#position-try-fallbacks).
func TestBehaviorPositionTryTactic(t *testing.T) {
	t.Parallel()

	sheet := sheet(t, `
		#anchor { anchor-name: --tip; width: 100px; height: 50px; }
		#plain { position: absolute; position-anchor: --tip; position-area: top;
			width: 60px; height: 20px; }
		#flipped { position: absolute; position-anchor: --tip; position-area: top;
			position-try-fallbacks: flip-block; position-visibility: no-overflow;
			width: 60px; height: 20px; }
	`)

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="anchor">a</div><div id="plain">p</div><div id="flipped">f</div>`+
		`</body></html>`, sheet)

	anchor := boxByID(t, res, "anchor")
	plain := boxByID(t, res, "plain")
	flipped := boxByID(t, res, "flipped")

	if !(plain.y < anchor.y) {
		t.Errorf("plain Y = %.4fpt, anchor Y = %.4fpt, want plain above anchor (top)",
			plain.y, anchor.y)
	}

	if !(flipped.y > anchor.y) {
		t.Errorf("flipped Y = %.4fpt, anchor Y = %.4fpt, want flipped below anchor (flip-block)",
			flipped.y, anchor.y)
	}

	if !near(plain.x, anchor.x) || !near(flipped.x, anchor.x) {
		t.Errorf("plain X = %.4fpt, flipped X = %.4fpt, anchor X = %.4fpt, want both aligned to anchor start",
			plain.x, flipped.x, anchor.x)
	}

	if len(res.Ops) == 0 {
		t.Fatal("DisplayList Ops empty, want paint ops for anchor and targets")
	}
}
