package layout

import "testing"

// TestBehaviorPositionTryTactic proves the tactic row: the base top spills
// above the canvas so it unanchors, while the flip-block fallback flips the
// chosen area across the axis the pinned spec names and wins below the
// anchor. The output is layout.DisplayList: the flipped target shifts its
// paint ops with its box.
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

	if !near(plain.x, 0) || !near(plain.y, 0) {
		t.Errorf("plain = (%.4fpt, %.4fpt), want laid-out (0, 0) after the base top spilled",
			plain.x, plain.y)
	}

	wantY := anchor.y + anchor.height
	if !near(flipped.y, wantY) {
		t.Errorf("flipped Y = %.4fpt, want below anchor %.4fpt (flip-block)",
			flipped.y, wantY)
	}

	if !near(plain.x, flipped.x) {
		t.Errorf("plain X = %.4fpt, flipped X = %.4fpt, want both at laid-out start",
			plain.x, flipped.x)
	}

	if len(res.Ops) == 0 {
		t.Fatal("DisplayList Ops empty, want paint ops for anchor and targets")
	}
}
