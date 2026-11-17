package layout

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBehaviorLinePaddingInsetsEveryLine reads the line-padding fixture and
// checks that each text run of the padded paragraph starts 20 CSS pixels to
// the right of the same run in the zero paragraph. Both lines move; this is
// not text-indent.
func TestBehaviorLinePaddingInsetsEveryLine(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "testdata", "css", "v012", "line-padding", "page.html")

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	res := layoutHTML(t, string(raw))
	padded := linePaddingTextRuns(t, res, "padded")
	zero := linePaddingTextRuns(t, res, "zero")

	if len(padded) < 2 || len(zero) < 2 {
		t.Fatalf("padded runs = %d %q, zero runs = %d %q, want at least two each",
			len(padded), linePaddingRunText(padded), len(zero), linePaddingRunText(zero))
	}

	if len(padded) != len(zero) {
		t.Fatalf("run count padded = %d %q, zero = %d %q",
			len(padded), linePaddingRunText(padded), len(zero), linePaddingRunText(zero))
	}

	want := pxToPt(20)

	for i := range padded {
		if padded[i].Text != zero[i].Text {
			t.Errorf("run %d text = %q, zero run = %q", i, padded[i].Text, zero[i].Text)
		}

		delta := padded[i].X - zero[i].X
		if !near(delta, want) {
			t.Errorf("run %d X shift = %.4fpt (%.2fpx), want 20px", i, delta, delta/ptPerCSSPx)
		}
	}
}

func linePaddingTextRuns(t *testing.T, res *Result, id string) []Op {
	t.Helper()

	boxNode := boxByID(t, res, id)
	if boxNode.opEnd < boxNode.opStart {
		t.Fatalf("#%s emitted no ops", id)
	}

	var runs []Op

	for i := boxNode.opStart; i <= boxNode.opEnd && i < len(res.Ops); i++ {
		if res.Ops[i].Kind == OpText {
			runs = append(runs, res.Ops[i])
		}
	}

	if len(runs) == 0 {
		t.Fatalf("#%s has no text runs", id)
	}

	return runs
}

func linePaddingRunText(runs []Op) []string {
	out := make([]string, len(runs))

	for i := range runs {
		out[i] = runs[i].Text
	}

	return out
}
