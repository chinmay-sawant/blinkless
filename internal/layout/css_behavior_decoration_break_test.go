package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBehaviorBoxDecorationBreakClonePaintsEachFragment reads
// testdata/css/v012/box-decoration-break/page.html and lays it out with
// layoutHTML.
//
// slice paints one red fill that covers every wrapped line of that phrase.
// clone paints one red fill per line. Each clone fill is about one line
// tall, and wider than its glyphs by the padding and the border.
func TestBehaviorBoxDecorationBreakClonePaintsEachFragment(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.Join(
		"..", "..", "testdata", "css", "v012", "box-decoration-break", "page.html",
	))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	res := layoutHTML(t, string(raw))
	sliceText, cloneText, sliceFills, cloneFills := decorationBreakParts(res)

	if len(sliceText) < 2 {
		t.Fatalf("slice lines = %d, want a wrap", len(sliceText))
	}

	if len(sliceFills) != 1 {
		t.Fatalf("slice red fills = %d, want 1", len(sliceFills))
	}

	sliceFill := sliceFills[0]

	for _, text := range sliceText {
		if !decorationBreakFillHitsText(sliceFill, []Op{text}) {
			t.Fatalf("slice fill misses %q at y=%.2f", text.Text, text.Y)
		}
	}

	if len(cloneText) < 2 {
		t.Fatalf("clone lines = %d, want a wrap", len(cloneText))
	}

	if len(cloneFills) != len(cloneText) {
		t.Fatalf("clone fills = %d, lines = %d, want one fill per line", len(cloneFills), len(cloneText))
	}

	// 8px padding and 2px border on the left and the right.
	chromeW := pxToPt(8) + pxToPt(8) + pxToPt(2) + pxToPt(2)

	for idx := range cloneText {
		text := cloneText[idx]
		fill := cloneFills[idx]

		if fill.H >= sliceFill.H {
			t.Fatalf("clone %q fill height %.2f is not one line; slice bar is %.2f",
				text.Text, fill.H, sliceFill.H)
		}

		if !near(fill.W, text.W+chromeW) {
			t.Fatalf("clone %q fill width %.2f, want %.2f (glyphs plus padding and border)",
				text.Text, fill.W, text.W+chromeW)
		}

		if !decorationBreakFillHitsText(fill, []Op{text}) {
			t.Fatalf("clone fill %d misses %q", idx, text.Text)
		}
	}
}

// decorationBreakParts splits the fixture into the slice paragraph and the
// clone paragraph. The clone span's first text op starts with "clone".
func decorationBreakParts(res *Result) ([]Op, []Op, []Op, []Op) {
	var sliceText, cloneText, sliceFills, cloneFills []Op

	var texts []Op

	var fills []Op

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpText && paintOp.Text != "" {
			texts = append(texts, paintOp)
		}

		if decorationBreakRedFill(paintOp) {
			fills = append(fills, paintOp)
		}
	}

	sliceText = decorationBreakRun(texts, "slice")
	cloneText = decorationBreakRun(texts, "clone")

	for _, fill := range fills {
		if decorationBreakFillHitsText(fill, sliceText) {
			sliceFills = append(sliceFills, fill)
		}

		if decorationBreakFillHitsText(fill, cloneText) {
			cloneFills = append(cloneFills, fill)
		}
	}

	return sliceText, cloneText, sliceFills, cloneFills
}

// decorationBreakRun returns the wrapped lines of the sample that starts
// with prefix, and stops when the next line is more than one line away.
func decorationBreakRun(texts []Op, prefix string) []Op {
	start := -1

	for idx, text := range texts {
		if strings.HasPrefix(text.Text, prefix) {
			start = idx
			break
		}
	}

	if start < 0 {
		return nil
	}

	out := []Op{texts[start]}

	for idx := start + 1; idx < len(texts); idx++ {
		if texts[idx].Y-texts[idx-1].Y > 32 {
			break
		}

		out = append(out, texts[idx])
	}

	return out
}

func decorationBreakRedFill(paintOp Op) bool {
	return paintOp.Kind == OpFillRect && paintOp.R > 0.9 && paintOp.G < 0.1 && paintOp.B < 0.1 &&
		paintOp.Alpha > 0 && paintOp.W > 0 && paintOp.H > 0
}

func decorationBreakFillHitsText(fill Op, texts []Op) bool {
	for _, text := range texts {
		probeX := text.X + 0.5
		probeY := text.Y - 1

		if probeX >= fill.X && probeX <= fill.X+fill.W && probeY >= fill.Y && probeY <= fill.Y+fill.H {
			return true
		}
	}

	return false
}
