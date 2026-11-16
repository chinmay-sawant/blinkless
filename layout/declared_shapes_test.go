package layout_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

// TestDeclaredShapesGrammar pins accept, normalize and reject for the shapes group.
func TestDeclaredShapesGrammar(t *testing.T) {
	t.Parallel()

	runDeclaredCases(t, []declaredCase{
		{"border-clip", "none", "none"},
		{"border-clip", "20px 10px", "20px 10px"},
		{"border-top-clip", "10% 1FR 2fr", "10% 1fr 2fr"},
		{"border-left-clip", "0 5px", "0 5px"},
		{"border-block-start-clip", "1em", "1em"},
		{"border-inline-end-clip", "-2px", ""},
		{"border-right-clip", "5", ""},
		{"border-bottom-clip", "none 4px", ""},
		{"border-inline-clip", "4vw", ""},
		{"border-limit", "all", ""},
		{"border-limit", "left 4em", "left 4em"},
		{"border-limit", "sides 50%", "sides 50%"},
		{"border-limit", "corners", "corners"},
		{"border-limit", "top", ""},
		{"border-limit", "sides 1px 2px", ""},
		{"border-boundary", "PARENT", "parent"},
		{"border-boundary", "display", "display"},
		{"border-boundary", "screen", ""},
		{"border-shape", "ellipse(50% 50%)", "ellipse(50% 50%)"},
		{"border-shape", "circle(40%) border-box ellipse(30% 30%)", "circle(40%) border-box ellipse(30% 30%)"},
		{"border-shape", "inset(0 round 4px) border-box circle(1px) circle(2px) circle(3px)", ""},
		{"border-shape", "border-box", ""},
		{"border-shape", "blob(1px)", ""},
		{"border-shape", "circle(4px", ""},
		{"text-fit", "grow", "grow"},
		{"text-fit", "shrink per-line 80%", "shrink per-line 80%"},
		{"text-fit", "scale", ""},
		{"text-fit", "grow 80", ""},
		{"float-defer", "2", "2"},
		{"float-defer", "last", "last"},
		{"float-defer", "1.5", ""},
		{"shape-image-threshold", "0.5", "0.5"},
		{"shape-image-threshold", "30%", "0.3"},
		{"shape-image-threshold", "4", "1"},
		{"shape-image-threshold", "red", ""},
		{"column-rule-visibility-items", "AROUND", "around"},
		{"column-rule-visibility-items", "between", "between"},
		{"column-rule-visibility-items", "edges", ""},
	})
}

// TestDeclaredShapesWideKeywords proves inherit and initial on a shapes name.
func TestDeclaredShapesWideKeywords(t *testing.T) {
	t.Parallel()

	const page = `<!DOCTYPE html><html><body>
		<div style="border-limit: left 4em"><p id="i" style="border-limit: inherit">x</p>
		<p id="n">y</p><p id="z" style="border-limit: inherit; border-limit: initial">z</p></div>
		</body></html>`

	display := renderDisplay(t, page, "")

	if got := boxByID(t, display, "i").Declared["border-limit"]; got != "left 4em" {
		t.Errorf("inherit = %q", got)
	}

	if got := boxByID(t, display, "n").Declared["border-limit"]; got != "" {
		t.Errorf("non-inheriting child = %q", got)
	}

	if got := boxByID(t, display, "z").Declared["border-limit"]; got != "" {
		t.Errorf("initial = %q", got)
	}
}

// edgeParts returns the stroke ops of a 100x40 px box (in pt) as [start, end]
// pairs in px along the edge, keyed by side letter, with the border drawn at
// 10px, 10px.
func edgeParts(display *layout.Display, color [3]float64) map[string][][2]float64 {
	out := map[string][][2]float64{}

	for _, op := range display.Ops {
		if op.Kind != layout.DisplayOpLine || math.Abs(op.R-color[0]) > 0.01 || math.Abs(op.B-color[2]) > 0.01 {
			continue
		}

		x, y, w, h := op.X/0.75, op.Y/0.75, op.W/0.75, op.H/0.75

		switch {
		case h == 0 && math.Abs(y-10) < 0.1:
			out["top"] = append(out["top"], [2]float64{x - 10, x - 10 + w})
		case h == 0:
			out["bottom"] = append(out["bottom"], [2]float64{x - 10, x - 10 + w})
		case math.Abs(x-10) < 0.1:
			out["left"] = append(out["left"], [2]float64{y - 10, y - 10 + h})
		default:
			out["right"] = append(out["right"], [2]float64{y - 10, y - 10 + h})
		}
	}

	return out
}

func borderBoxDisplay(t *testing.T, decl string) map[string][][2]float64 {
	t.Helper()

	page := fmt.Sprintf(`<!DOCTYPE html><html><body style="margin:0"><div id="b" style="position:absolute;`+
		`left:10px;top:10px;box-sizing:border-box;width:100px;height:40px;border:4px solid #c00;%s"></div>`+
		`</body></html>`, decl)

	return edgeParts(renderDisplay(t, page, ""), [3]float64{0.8, 0, 0})
}

func wantParts(t *testing.T, label string, got [][2]float64, want ...[2]float64) {
	t.Helper()

	if len(got) != len(want) {
		t.Errorf("%s: parts = %v, want %v", label, got, want)

		return
	}

	for i := range want {
		if math.Abs(got[i][0]-want[i][0]) > 0.1 || math.Abs(got[i][1]-want[i][1]) > 0.1 {
			t.Errorf("%s: parts = %v, want %v", label, got, want)

			return
		}
	}
}

// TestBorderClipSplitsEveryEdge: border-clip: 20px 10px paints 20px, skips
// 10px, paints the rest, on all four edges of a 100 by 40 box.
func TestBorderClipSplitsEveryEdge(t *testing.T) {
	t.Parallel()

	whole := borderBoxDisplay(t, "")
	wantParts(t, "plain top", whole["top"], [2]float64{0, 100})

	clipped := borderBoxDisplay(t, "border-clip: 20px 10px")
	wantParts(t, "top", clipped["top"], [2]float64{0, 20}, [2]float64{30, 100})
	wantParts(t, "bottom", clipped["bottom"], [2]float64{0, 20}, [2]float64{30, 100})
	wantParts(t, "left", clipped["left"], [2]float64{0, 20}, [2]float64{30, 40})
	wantParts(t, "right", clipped["right"], [2]float64{0, 20}, [2]float64{30, 40})

	none := borderBoxDisplay(t, "border-clip: none")
	wantParts(t, "none top", none["top"], [2]float64{0, 100})
}

// TestBorderClipLonghandsAndLogical: each longhand cuts only its own edge,
// and the logical and axis names map to the matching physical edges.
func TestBorderClipLonghandsAndLogical(t *testing.T) {
	t.Parallel()

	cut20 := [][2]float64{{0, 20}, {30, 100}}
	cutV := [][2]float64{{0, 20}, {30, 40}}
	whole := map[string][][2]float64{
		"top": {{0, 100}}, "bottom": {{0, 100}}, "left": {{0, 40}}, "right": {{0, 40}},
	}

	cases := []struct {
		name string
		cut  []string
	}{
		{"border-top-clip", []string{"top"}},
		{"border-right-clip", []string{"right"}},
		{"border-bottom-clip", []string{"bottom"}},
		{"border-left-clip", []string{"left"}},
		{"border-block-start-clip", []string{"top"}},
		{"border-block-end-clip", []string{"bottom"}},
		{"border-inline-start-clip", []string{"left"}},
		{"border-inline-end-clip", []string{"right"}},
		{"border-block-clip", []string{"top", "bottom"}},
		{"border-inline-clip", []string{"left", "right"}},
	}

	for _, c := range cases {
		got := borderBoxDisplay(t, c.name+": 20px 10px")
		cutSet := map[string]bool{}

		for _, side := range c.cut {
			cutSet[side] = true
		}

		for side, edge := range whole {
			want := edge

			if cutSet[side] {
				want = cut20
				if side == "left" || side == "right" {
					want = cutV
				}
			}

			wantParts(t, c.name+" "+side, got[side], want...)
		}
	}

	// A physical longhand beats the shorthand.
	over := borderBoxDisplay(t, "border-clip: 20px 10px; border-top-clip: none")
	wantParts(t, "override top", over["top"], [2]float64{0, 100})
	wantParts(t, "override bottom", over["bottom"], cut20...)
}

// TestBorderClipFlexAndPercent: percentages use the edge length and fr shares
// what the fixed parts leave.
func TestBorderClipFlexAndPercent(t *testing.T) {
	t.Parallel()

	got := borderBoxDisplay(t, "border-top-clip: 10% 10px 1fr 20px 1fr")
	// 10 paint, 10 skip, free 60 split 30/30: 30 paint, 20 skip, 30 skip? parts alternate.
	wantParts(t, "flex", got["top"], [2]float64{0, 10}, [2]float64{20, 50}, [2]float64{70, 100})
}

// TestBorderLimit: border-limit left 4em keeps 4em of each horizontal edge from
// the left, and all keeps the full edge.
func TestBorderLimit(t *testing.T) {
	t.Parallel()

	got := borderBoxDisplay(t, "border-limit: left 40px")
	wantParts(t, "left 40px top", got["top"], [2]float64{0, 40})
	wantParts(t, "left 40px left edge", got["left"], [2]float64{0, 40})

	sides := borderBoxDisplay(t, "border-limit: sides 50%")
	wantParts(t, "sides top", sides["top"], [2]float64{25, 75})
	wantParts(t, "sides left", sides["left"], [2]float64{10, 30})

	all := borderBoxDisplay(t, "border-limit: all")
	wantParts(t, "all", all["top"], [2]float64{0, 100})

	corners := borderBoxDisplay(t, "border-limit: corners 10px")
	wantParts(t, "corners", corners["top"], [2]float64{0, 10}, [2]float64{90, 100})
}

// TestColumnRuleVisibilityItemsAround: around adds a column rule on each
// content edge, between keeps only the gap rules.
func TestColumnRuleVisibilityItemsAround(t *testing.T) {
	t.Parallel()

	count := func(vis string) int {
		page := fmt.Sprintf(`<!DOCTYPE html><html><body style="margin:0"><div style="column-count:2;`+
			`column-gap:20px;column-rule:3px solid #c00;%s;width:200px"><p>one two three four five six `+
			`seven eight nine ten eleven twelve</p></div></body></html>`, vis)
		display := renderDisplay(t, page, "")
		n := 0

		for _, op := range display.Ops {
			if op.Kind == layout.DisplayOpLine && op.W == 0 && op.H > 0 && math.Abs(op.R-0.8) < 0.01 {
				n++
			}
		}

		return n
	}

	between, around, all, normal := count("column-rule-visibility-items:between"),
		count("column-rule-visibility-items:around"), count("column-rule-visibility-items:all"),
		count("column-rule-visibility-items:normal")

	if between != 1 || normal != 1 {
		t.Errorf("between = %d normal = %d, want 1 gap rule", between, normal)
	}

	if around != 3 || all != 3 {
		t.Errorf("around = %d all = %d, want gap plus two edges", around, all)
	}
}
