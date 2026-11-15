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
// Both paragraphs are 80px wide. Each span is red, with 2px padding and a
// 1px border. The copy wraps to 5 lines. 1px is 0.75pt, so the padding and
// border add 4.5pt to a fragment's width (2px + 1px on each side).
//
// slice paints one red fill: x=0, y=0, 45.19pt wide, 90pt tall. That single
// box covers all 5 slice lines (baselines 13.16, 31.16, 49.16, 67.16, and
// 85.16). 45.19pt is the widest slice fragment: "slice aa" is 40.69pt of
// glyphs plus the 4.5pt chrome.
//
// clone paints 5 red fills, one per line, each 18pt tall:
//
//	"clone aa" 49.87pt, "bb cc" 33.18pt, "dd ee ff" 44.53pt,
//	"gg hh ii" 43.20pt, "jj kk ll" 33.83pt.
//
// Each width is that line's glyph width plus the same 4.5pt.
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

	if len(sliceText) != 5 {
		t.Fatalf("slice lines = %d, want 5", len(sliceText))
	}

	if len(sliceFills) != 1 {
		t.Fatalf("slice red fills = %d, want 1", len(sliceFills))
	}

	sliceFill := sliceFills[0]
	if !near(sliceFill.X, 0) || !near(sliceFill.Y, 0) || !near(sliceFill.W, 45.19) || !near(sliceFill.H, 90) {
		t.Fatalf("slice fill = (%.2f, %.2f) %.2f by %.2f, want (0, 0) 45.19 by 90",
			sliceFill.X, sliceFill.Y, sliceFill.W, sliceFill.H)
	}

	for _, text := range sliceText {
		if !decorationBreakFillHitsText(sliceFill, []Op{text}) {
			t.Fatalf("slice fill misses %q at y=%.2f", text.Text, text.Y)
		}
	}

	if len(cloneFills) != 5 {
		t.Fatalf("clone red fills = %d, want 5 (more than one)", len(cloneFills))
	}

	if len(cloneText) != len(cloneFills) {
		t.Fatalf("clone lines = %d, fills = %d, want one fill per line", len(cloneText), len(cloneFills))
	}

	// 2px padding and 1px border on the left and the right.
	chromeW := pxToPt(2) + pxToPt(2) + pxToPt(1) + pxToPt(1)

	for idx := range cloneText {
		text := cloneText[idx]
		fill := cloneFills[idx]

		if !near(fill.H, 18) || !near(fill.W, text.W+chromeW) {
			t.Fatalf("clone %q fill %.2f by %.2f, want width %.2f (glyphs plus 4.5pt) and height 18",
				text.Text, fill.W, fill.H, text.W+chromeW)
		}

		if !decorationBreakFillHitsText(fill, []Op{text}) {
			t.Fatalf("clone fill %d misses %q", idx, text.Text)
		}
	}
}

// decorationBreakParts splits the fixture into the slice paragraph and the
// clone paragraph. The clone span's first text op starts with "clone".
func decorationBreakParts(res *Result) (sliceText, cloneText, sliceFills, cloneFills []Op) {
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

	cut := len(texts)

	for idx, text := range texts {
		if strings.HasPrefix(text.Text, "clone") {
			cut = idx

			break
		}
	}

	sliceText = texts[:cut]
	cloneText = texts[cut:]

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
