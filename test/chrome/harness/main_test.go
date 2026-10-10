package main

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

// TestHarnessSmokeMatchFlow pins the join contract without a browser: two
// elements, two boxes, exact matches, one report. It guards the helpers
// against refactors, not the Chrome comparison itself.
func TestHarnessSmokeMatchFlow(t *testing.T) {
	t.Parallel()

	elems := []domElem{
		{Pos: 0, Path: "html/body/div:nth-of-type(1)", Tag: "div", ID: "a", Text: "alpha"},
		{Pos: 1, Path: "html/body/div:nth-of-type(2)", Tag: "div", ID: "b", Text: "beta"},
	}
	boxes := []layout.Box{
		{Tag: "div", ID: "a", Text: "alpha", X: 0, Y: 0, W: 100, H: 20},
		{Tag: "div", ID: "b", Text: "beta", X: 0, Y: 20, W: 100, H: 20},
		{Tag: "#document"},
	}

	rep := newReport("smoke.html", 1024, 768, 100, 40, len(elems), len(boxes))
	joinBoxes(&rep, elems, boxes)

	if len(rep.Boxes) != 2 {
		t.Fatalf("joined boxes = %d, want 2 (the #document box only bumps the skip counter)", len(rep.Boxes))
	}

	if rep.Boxes[0].Match != "exact" || rep.Boxes[0].Path == "" {
		t.Errorf("first box match = %q path = %q, want exact with a path",
			rep.Boxes[0].Match, rep.Boxes[0].Path)
	}

	if rep.Matches["exact"] != 2 {
		t.Errorf("exact matches = %d, want 2", rep.Matches["exact"])
	}

	if rep.Matches["structural-skip"] != 1 {
		t.Errorf("structural skips = %d, want 1", rep.Matches["structural-skip"])
	}

	if rep.ElemCount != 2 || rep.BoxCount != 3 {
		t.Errorf("counts = %d/%d, want 2/3", rep.ElemCount, rep.BoxCount)
	}
}

// TestHarnessSmokeUnmatchedKeepsBox pins the unmatched path: a box with no
// element stays in the output with an empty path so the join stays complete.
func TestHarnessSmokeUnmatchedKeepsBox(t *testing.T) {
	t.Parallel()

	elems := []domElem{
		{Pos: 0, Path: "html/body/div:nth-of-type(1)", Tag: "div", ID: "a", Text: "alpha"},
	}
	boxes := []layout.Box{
		{Tag: "span", ID: "ghost", Text: "ghost", X: 5, Y: 5, W: 10, H: 10},
	}

	rep := newReport("smoke.html", 1024, 768, 10, 10, len(elems), len(boxes))
	joinBoxes(&rep, elems, boxes)

	if len(rep.Boxes) != 1 {
		t.Fatalf("joined boxes = %d, want 1", len(rep.Boxes))
	}

	if rep.Boxes[0].Match != "unmatched" || rep.Boxes[0].Path != "" {
		t.Errorf("ghost box match = %q path = %q, want unmatched with empty path",
			rep.Boxes[0].Match, rep.Boxes[0].Path)
	}
}
