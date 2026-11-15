package layout

import "testing"

// allPage is a parent with one child. The stylesheet paints the parent and
// sets the child's all value.
func allPage() string {
	return `<html><body style="margin:0"><div id="parent"><p id="child">Hi</p></div></body></html>`
}

func allLayout(t *testing.T, parentCSS, childCSS string) *Result {
	t.Helper()

	return layoutHTML(t, allPage(), sheet(t,
		"#parent { "+parentCSS+" } #child { "+childCSS+" }"))
}

func allBoxOps(t *testing.T, res *Result, elementID string) []Op {
	t.Helper()

	elementBox := boxByID(t, res, elementID)
	if elementBox.opEnd < elementBox.opStart || elementBox.opStart < 0 {
		return nil
	}

	end := elementBox.opEnd
	if end >= len(res.Ops) {
		end = len(res.Ops) - 1
	}

	if elementBox.opStart > end {
		return nil
	}

	return res.Ops[elementBox.opStart : end+1]
}

func allText(t *testing.T, ops []Op) Op {
	t.Helper()

	for _, op := range ops {
		if op.Kind == OpText && op.Text != "" {
			return op
		}
	}

	t.Fatal("no text op on the child")

	return Op{}
}

func allFills(ops []Op) []Op {
	var out []Op

	for _, op := range ops {
		if op.Kind == OpFillRect {
			out = append(out, op)
		}
	}

	return out
}

// allParentOwnFills returns fill ops on the parent that are outside the child
// op span. The parent span includes the child subtree.
func allParentOwnFills(t *testing.T, res *Result) []Op {
	t.Helper()

	parent := boxByID(t, res, "parent")
	child := boxByID(t, res, "child")
	if parent.opEnd < parent.opStart {
		return nil
	}

	var out []Op

	for i := parent.opStart; i <= parent.opEnd && i < len(res.Ops); i++ {
		if child.opEnd >= child.opStart && i >= child.opStart && i <= child.opEnd {
			continue
		}

		if res.Ops[i].Kind == OpFillRect {
			out = append(out, res.Ops[i])
		}
	}

	return out
}

func allRGBNear(op Op, r, g, b float64) bool {
	return near(op.R, r) && near(op.G, g) && near(op.B, b)
}

// TestBehaviorAllInitialResetsColorAndKeepsDirection: all: initial paints the
// child text black and emits no child fill. direction is not reset, so the
// parent's rtl stays.
func TestBehaviorAllInitialResetsColorAndKeepsDirection(t *testing.T) {
	t.Parallel()

	res := allLayout(t,
		"color: red; background-color: blue; direction: rtl;",
		"all: initial;")

	child := boxByID(t, res, "child")
	if child.style == nil || child.style.Direction != cssDirectionRTL {
		got := ""
		if child.style != nil {
			got = child.style.Direction
		}

		t.Fatalf("direction = %q, want rtl", got)
	}

	text := allText(t, allBoxOps(t, res, "child"))
	if !allRGBNear(text, 0, 0, 0) {
		t.Fatalf("text rgb = %.3f,%.3f,%.3f, want 0,0,0", text.R, text.G, text.B)
	}

	if fills := allFills(allBoxOps(t, res, "child")); len(fills) != 0 {
		t.Fatalf("child OpFillRect count = %d, want 0", len(fills))
	}

	parentFills := allParentOwnFills(t, res)
	if len(parentFills) == 0 || !allRGBNear(parentFills[0], 0, 0, 1) {
		t.Fatal("parent blue fill missing")
	}
}

// TestBehaviorAllInheritPaintsParentBackground: all: inherit copies the
// parent's background onto the child even though background-color is not
// in the inherit table.
func TestBehaviorAllInheritPaintsParentBackground(t *testing.T) {
	t.Parallel()

	res := allLayout(t,
		"color: red; background-color: blue; direction: rtl;",
		"all: inherit;")

	childFills := allFills(allBoxOps(t, res, "child"))
	parentFills := allParentOwnFills(t, res)
	if len(childFills) == 0 || len(parentFills) == 0 {
		t.Fatalf("fills child=%d parent=%d, want a fill on each", len(childFills), len(parentFills))
	}

	if !allRGBNear(childFills[0], parentFills[0].R, parentFills[0].G, parentFills[0].B) {
		t.Fatalf("child fill rgb = %.3f,%.3f,%.3f, parent = %.3f,%.3f,%.3f",
			childFills[0].R, childFills[0].G, childFills[0].B,
			parentFills[0].R, parentFills[0].G, parentFills[0].B)
	}
}

// TestBehaviorAllLonghandAfterWins: a color longhand after all: initial
// keeps the text red.
func TestBehaviorAllLonghandAfterWins(t *testing.T) {
	t.Parallel()

	res := allLayout(t,
		"color: red; background-color: blue; direction: rtl;",
		"all: initial; color: red;")

	text := allText(t, allBoxOps(t, res, "child"))
	if !allRGBNear(text, 1, 0, 0) {
		t.Fatalf("text rgb = %.3f,%.3f,%.3f, want red", text.R, text.G, text.B)
	}
}

// TestBehaviorAllUnsetInheritsColor: unset inherits color and resets
// background-color to transparent, so the child text is the parent's red
// and the child has no fill.
func TestBehaviorAllUnsetInheritsColor(t *testing.T) {
	t.Parallel()

	res := allLayout(t,
		"color: red; background-color: blue;",
		"color: #00ff00; background-color: #00ff00; all: unset;")

	text := allText(t, allBoxOps(t, res, "child"))
	if !allRGBNear(text, 1, 0, 0) {
		t.Fatalf("unset color rgb = %.3f,%.3f,%.3f, want inherited red", text.R, text.G, text.B)
	}

	if fills := allFills(allBoxOps(t, res, "child")); len(fills) != 0 {
		t.Fatalf("unset background fills = %d, want 0 (initial)", len(fills))
	}
}

// TestBehaviorAllBadTokenKeepsColor: a token after all makes the declaration
// invalid, so an earlier color stays.
func TestBehaviorAllBadTokenKeepsColor(t *testing.T) {
	t.Parallel()

	res := allLayout(t,
		"color: red; background-color: blue;",
		"color: red; all: initial extra;")

	text := allText(t, allBoxOps(t, res, "child"))
	if !allRGBNear(text, 1, 0, 0) {
		t.Fatalf("text rgb = %.3f,%.3f,%.3f, want the earlier red", text.R, text.G, text.B)
	}
}

// TestBehaviorAllRevertDoesNotWalkOrigins: revert, revert-layer, and
// revert-rule do not roll an author color or fill back to a user-agent or
// parent value.
func TestBehaviorAllRevertDoesNotWalkOrigins(t *testing.T) {
	t.Parallel()

	for _, keyword := range []string{"revert", "revert-layer", "revert-rule"} {
		res := allLayout(t,
			"color: red; background-color: blue;",
			"color: #ff00ff; background-color: #00ff00; all: "+keyword+";")

		text := allText(t, allBoxOps(t, res, "child"))
		if !allRGBNear(text, 1, 0, 1) {
			t.Fatalf("all: %s text rgb = %.3f,%.3f,%.3f, want the author magenta",
				keyword, text.R, text.G, text.B)
		}

		fills := allFills(allBoxOps(t, res, "child"))
		if len(fills) == 0 || !allRGBNear(fills[0], 0, 1, 0) {
			t.Fatalf("all: %s fill = %v, want the author green", keyword, fills)
		}
	}
}
