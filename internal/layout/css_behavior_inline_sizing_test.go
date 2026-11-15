package layout

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBehaviorInlineSizingStretchGrowsFill is inline-sizing. The line box is
// 32px tall and the spans use a 16px font with line-height:normal, so the
// normal red fill stays on the font. stretch grows that fill to the line box.
// The two text baselines stay on the same Y.
// Reference: CSS Inline 3, Chrome 143.0.7499.40.
func TestBehaviorInlineSizingStretchGrowsFill(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "testdata", "css", "v012", "inline-sizing", "page.html")

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	res := layoutHTML(t, string(raw))
	line := boxByID(t, res, "line")
	normalText := inlineSizingText(t, res, "Na")
	stretchText := inlineSizingText(t, res, "St")
	normalFill := inlineSizingRedFill(t, res, normalText.X)
	stretchFill := inlineSizingRedFill(t, res, stretchText.X)

	if near(normalFill.X, stretchFill.X) {
		t.Fatalf("expected two red span fills, both at X %.4f", normalFill.X)
	}

	if stretchFill.H <= normalFill.H {
		t.Fatalf("stretch fill height %.4fpt is not taller than normal %.4fpt",
			stretchFill.H, normalFill.H)
	}

	if !near(stretchFill.H, line.height) {
		t.Fatalf("stretch fill height %.4fpt, want line box %.4fpt", stretchFill.H, line.height)
	}

	if !near(normalText.Y, stretchText.Y) {
		t.Fatalf("text Y normal %.4fpt stretch %.4fpt, want the same baseline",
			normalText.Y, stretchText.Y)
	}
}

func inlineSizingText(t *testing.T, res *Result, text string) Op {
	t.Helper()

	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text == text {
			return op
		}
	}

	t.Fatalf("no text op %q", text)

	return Op{}
}

func inlineSizingRedFill(t *testing.T, res *Result, textX float64) Op {
	t.Helper()

	var found Op

	matched := false
	best := 0.0

	for _, op := range res.Ops {
		if op.Kind != OpFillRect || !near(op.R, 1) || !near(op.G, 0) || !near(op.B, 0) {
			continue
		}

		delta := op.X - textX
		if delta < 0 {
			delta = -delta
		}

		if matched && delta >= best {
			continue
		}

		found = op
		best = delta
		matched = true
	}

	if !matched {
		t.Fatal("no red fill")
	}

	return found
}
