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

// pathBounds counts the fill and stroke OpPath ops of a display and returns
// the x range their points cover.
func pathBounds(display *layout.Display) (int, int, float64, float64) {
	fills, strokes := 0, 0
	minX, maxX := math.Inf(1), math.Inf(-1)

	for _, op := range display.Ops {
		if op.Kind != layout.DisplayOpPath {
			continue
		}

		path, ok := op.CornerPathValue()
		if !ok {
			continue
		}

		if path.Stroke {
			strokes++
		} else {
			fills++
		}

		for _, cmd := range path.Cmds {
			if cmd.Verb == layout.PathClose {
				continue
			}

			minX, maxX = math.Min(minX, cmd.Pts[0].X), math.Max(maxX, cmd.Pts[0].X)
		}
	}

	return fills, strokes, minX, maxX
}

// TestBorderShapeEllipsePath: border-shape: ellipse paints the fill and the
// border as path ops that stay inside the border box, and none keeps rects.
func TestBorderShapeEllipsePath(t *testing.T) {
	t.Parallel()

	page := func(decl string) string {
		return fmt.Sprintf(`<!DOCTYPE html><html><body style="margin:0"><div style="position:absolute;`+
			`left:10px;top:10px;box-sizing:border-box;width:100px;height:60px;border:4px solid #00f;`+
			`background:#eef;border-radius:50%%;%s"></div></body></html>`, decl)
	}

	fills, strokes, minX, maxX := pathBounds(renderDisplay(t, page("border-shape: ellipse(50% 50%)"), ""))
	if fills != 1 || strokes != 1 {
		t.Fatalf("fills=%d strokes=%d, want 1 and 1", fills, strokes)
	}

	if math.Abs(minX) > 0.1 || math.Abs(maxX-75) > 0.1 {
		t.Errorf("ellipse x range = %.2f..%.2f, want 0..75pt, the 100px box", minX, maxX)
	}

	fills, strokes, _, _ = pathBounds(renderDisplay(t, page("border-shape: none"), ""))
	if fills != 0 || strokes != 0 {
		t.Errorf("border-shape none emitted %d fill and %d stroke paths", fills, strokes)
	}

	fills, strokes, _, _ = pathBounds(renderDisplay(t, page("border-shape: ellipse(50% 50%) ellipse(40% 40%)"), ""))
	if fills != 0 || strokes != 0 {
		t.Errorf("two-shape band is not painted yet, got %d fill and %d stroke paths", fills, strokes)
	}
}

// textSize returns the font size of the first text op.
func textSize(display *layout.Display) float64 {
	for _, op := range display.Ops {
		if op.Kind == layout.DisplayOpText {
			return op.Size
		}
	}

	return 0
}

// TestTextFitScalesOneLine: grow enlarges a short line toward the box width,
// shrink reduces a long one to fit, the percentage caps the scale, and none
// leaves the font alone.
func TestTextFitScalesOneLine(t *testing.T) {
	t.Parallel()

	size := func(text, decl string) float64 {
		page := fmt.Sprintf(`<!DOCTYPE html><html><body style="margin:0"><div style="width:200px;`+
			`font-size:12px;%s">%s</div></body></html>`, decl, text)

		return textSize(renderDisplay(t, page, ""))
	}

	plain := size("Fit me", "")
	if plain <= 0 {
		t.Fatal("no text op")
	}

	if grown := size("Fit me", "text-fit: grow"); grown < plain*2 {
		t.Errorf("grow size = %.2f, plain %.2f, want a clear enlargement", grown, plain)
	}

	if capped := size("Fit me", "text-fit: grow consistent 150%"); math.Abs(capped-plain*1.5) > 0.05 {
		t.Errorf("grow 150%% size = %.2f, want %.2f", capped, plain*1.5)
	}

	long := "A much longer line of text that cannot fit in two hundred pixels at twelve"
	longPlain := size(long, "")

	if shrunk := size(long, "text-fit: shrink"); !(shrunk < longPlain*0.9) {
		t.Errorf("shrink size = %.2f, plain %.2f, want smaller", shrunk, longPlain)
	}

	if size("Fit me", "text-fit: shrink") != plain {
		t.Error("shrink changed a line that already fits")
	}

	if size("Fit me", "text-fit: none") != plain {
		t.Error("text-fit none changed the font")
	}
}

// TestFloatDeferMovesFloatToLaterColumn: in a three column box a float with
// float-defer: 1 sits one column (100px plus the 10px gap) to the right of the same float without it,
// `last` puts it in the third column, and a count past the end stops there.
func TestFloatDeferMovesFloatToLaterColumn(t *testing.T) {
	t.Parallel()

	left := func(decl string) float64 {
		page := fmt.Sprintf(`<!DOCTYPE html><html><body style="margin:0"><div style="column-count:3;column-gap:10px;`+
			`width:320px"><div id="f" style="float:left;width:20px;height:20px;background:#f00;%s"></div>`+
			`<p>one two three</p></div></body></html>`, decl)

		for _, box := range renderDisplay(t, page, "").Boxes {
			if box.ID == "f" {
				return box.X
			}
		}

		t.Fatal("no float box")

		return 0
	}

	base := left("")

	for decl, want := range map[string]float64{
		"float-defer:none": 0, "float-defer:1": 110, "float-defer:last": 220, "float-defer:5": 220,
	} {
		if got := left(decl) - base; math.Abs(got-want) > 0.1 {
			t.Errorf("%s moved the float by %.2fpx, want %.0fpx", decl, got, want)
		}
	}
}

// TestBorderBoundaryParentTracesCurve: a child in the corner of a rounded
// parent gets a border path along the parent's curve inside its own box, and
// none/display or a child clear of the curve get no path.
func TestBorderBoundaryParentTracesCurve(t *testing.T) {
	t.Parallel()

	paths := func(boundary, offset string) (int, float64, float64) {
		page := fmt.Sprintf(`<!DOCTYPE html><html><body style="margin:0"><div style="width:150px;height:60px;`+
			`border-radius:50%%;background:#eee"><div id="c" style="position:relative;%s;width:50px;height:20px;`+
			`border:4px solid #c0392b;%s"></div></div></body></html>`, offset, boundary)

		count := 0
		minX, maxX := math.Inf(1), math.Inf(-1)

		for _, op := range renderDisplay(t, page, "").Ops {
			path, ok := op.CornerPathValue()
			if op.Kind != layout.DisplayOpPath || !ok || !path.Stroke {
				continue
			}

			count++

			for _, cmd := range path.Cmds {
				minX, maxX = math.Min(minX, cmd.Pts[0].X), math.Max(maxX, cmd.Pts[0].X)
			}
		}

		return count, minX, maxX
	}

	count, minX, maxX := paths("border-boundary:parent", "left:0;top:0")
	if count == 0 {
		t.Fatal("child in the parent's corner got no boundary path")
	}

	if minX < -0.1 || maxX > 58*0.75+0.1 {
		t.Errorf("boundary path x range %.2f..%.2f leaves the child's 58px box", minX, maxX)
	}

	for _, tc := range []struct{ boundary, offset string }{
		{"border-boundary:none", "left:0;top:0"},
		{"border-boundary:display", "left:0;top:0"},
		{"border-boundary:parent", "left:50px;top:20px"},
	} {
		if got, _, _ := paths(tc.boundary, tc.offset); got != 0 {
			t.Errorf("%s at %s gave %d boundary paths, want 0", tc.boundary, tc.offset, got)
		}
	}
}
