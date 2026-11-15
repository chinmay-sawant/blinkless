package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBehaviorBaselineSourceFirstAlignsFirstLine reads the v0.1.2 fixture and
// checks text-op baselines. baseline-source:first puts the first Ag on the
// same baseline as the neighboring go. last and auto put the second Ag there.
func TestBehaviorBaselineSourceFirstAlignsFirstLine(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "css", "v012", "baseline-source", "page.html"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	res := layoutHTML(t, string(raw))

	assertBaselineSourceLine(t, res, "src-first", 0)
	assertBaselineSourceLine(t, res, "src-last", 1)
	assertBaselineSourceLine(t, res, "src-auto", 1)
}

// assertBaselineSourceLine checks that the Ag line at alignIndex (0 is the
// first line, 1 the second) shares Y with go, and that the other Ag does not.
func assertBaselineSourceLine(t *testing.T, res *Result, paragraphID string, alignIndex int) {
	t.Helper()

	paragraph := boxByID(t, res, paragraphID)
	goY, ags := baselineSourceTextYs(t, res, paragraph, paragraphID)

	if alignIndex < 0 || alignIndex >= len(ags) {
		t.Fatalf("%s: align index %d outside %d Ag lines", paragraphID, alignIndex, len(ags))
	}

	if !near(ags[alignIndex], goY) {
		t.Errorf("%s: Ag line %d Y = %.4f, go Y = %.4f", paragraphID, alignIndex, ags[alignIndex], goY)
	}

	other := 1 - alignIndex
	if other < 0 || other >= len(ags) {
		return
	}

	if near(ags[other], goY) {
		t.Errorf("%s: the other Ag Y = %.4f also matches go %.4f", paragraphID, ags[other], goY)
	}
}

// baselineSourceTextYs returns the go baseline and the Ag baselines sorted
// top to bottom (smaller Y first).
func baselineSourceTextYs(t *testing.T, res *Result, paragraph *box, paragraphID string) (float64, []float64) {
	t.Helper()

	if paragraph.opEnd < paragraph.opStart {
		t.Fatalf("%s: empty op range", paragraphID)
	}

	var goY float64

	var ags []float64

	sawGo := false

	for index := paragraph.opStart; index <= paragraph.opEnd && index < len(res.Ops); index++ {
		op := res.Ops[index]
		if op.Kind != OpText {
			continue
		}

		text := strings.TrimSpace(op.Text)
		switch text {
		case "go":
			goY = op.Y
			sawGo = true
		case "Ag":
			ags = append(ags, op.Y)
		}
	}

	if !sawGo {
		t.Fatalf("%s: no go text op", paragraphID)
	}

	if len(ags) < 2 {
		t.Fatalf("%s: Ag lines = %d, want 2", paragraphID, len(ags))
	}

	if ags[0] > ags[1] {
		ags[0], ags[1] = ags[1], ags[0]
	}

	return goY, ags
}
