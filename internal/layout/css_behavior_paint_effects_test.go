package layout

import (
	"math"
	"testing"
)

// CSS paint-effect behavior tests. Each test covers one property and asserts
// the emitted display-list ops (or their used values), never stored strings.
// Reference browser for expected geometry: Chrome 143.0.7499.40.

// behaviorPaintFxXformed returns the ops that carry a baked CSS transform.
func behaviorPaintFxXformed(res *Result) []Op {
	var out []Op

	for _, op := range res.Ops {
		if op.XformSet {
			out = append(out, op)
		}
	}

	return out
}

// behaviorPaintFxDecoBox layouts one paragraph holding a single decorated
// span and returns the result.
func behaviorPaintFxDecoBox(t *testing.T, spanStyle, text string) *Result {
	t.Helper()

	return layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:14pt"><span style="`+spanStyle+`">`+text+`</span></p>`+
		`</body></html>`)
}

// TestBehaviorTextDecorationLineUnderlineStrokes is text-decoration-line:
// underline emits decoration strokes while none emits none. Reference:
// Chrome 143.0.7499.40 underlines the span text.
func TestBehaviorTextDecorationLineUnderlineStrokes(t *testing.T) {
	t.Parallel()

	under := behaviorPaintFxDecoBox(t, "text-decoration-line:underline", "acemnorstu")
	if got := len(underlineOps(under)); got == 0 {
		t.Errorf("text-decoration-line:underline: want >0 decoration ops, got 0")
	}

	plain := behaviorPaintFxDecoBox(t, "text-decoration-line:none", "acemnorstu")
	if got := len(underlineOps(plain)); got != 0 {
		t.Errorf("text-decoration-line:none: want 0 decoration ops, got %d", got)
	}

	// line-through strokes above the baseline, underline below it.
	const word = "acemnorstu"
	resU := behaviorPaintFxDecoBox(t, "text-decoration-line:underline", word)
	resS := behaviorPaintFxDecoBox(t, "text-decoration-line:line-through", word)
	underOps := underlineOps(resU)
	strikeOps := underlineOps(resS)

	if len(underOps) == 0 || len(strikeOps) == 0 {
		t.Fatalf("want decoration ops for both lines, got underline=%d strike=%d",
			len(underOps), len(strikeOps))
	}

	_, textOp, found := textOpIndex(resU, word)
	if !found {
		t.Fatal("no text op for the underline fixture word")
	}

	if !(strikeOps[0].Y < textOp.Y && textOp.Y < underOps[0].Y) {
		t.Errorf("strike Y %.3f baseline Y %.3f underline Y %.3f, want strike < baseline < underline",
			strikeOps[0].Y, textOp.Y, underOps[0].Y)
	}
}

// TestBehaviorTextDecorationStyleDashedSplits is text-decoration-style:
// dashed splits the underline into dash segments, so the same two-word text
// emits more ops than the solid style. Reference: Chrome 143.0.7499.40
// renders a broken dash run where solid draws one stroke per word.
func TestBehaviorTextDecorationStyleDashedSplits(t *testing.T) {
	t.Parallel()

	const text = "alpha beta"

	solid := behaviorPaintFxDecoBox(t, "text-decoration-line:underline", text)
	dashed := behaviorPaintFxDecoBox(t,
		"text-decoration-line:underline;text-decoration-style:dashed", text)

	solidOps := underlineOps(solid)
	dashedOps := underlineOps(dashed)

	if len(solidOps) == 0 {
		t.Fatal("solid underline: want >0 decoration ops, got 0")
	}

	if len(dashedOps) <= len(solidOps) {
		t.Errorf("dashed = %d ops, solid = %d ops, want dashed > solid",
			len(dashedOps), len(solidOps))
	}
}

// TestBehaviorTextDecorationColorOverridesInk is text-decoration-color: the
// decoration stroke uses the declared color, not the text ink. Reference:
// Chrome 143.0.7499.40 paints a red rule under black text.
func TestBehaviorTextDecorationColorOverridesInk(t *testing.T) {
	t.Parallel()

	res := behaviorPaintFxDecoBox(t,
		"color:black;text-decoration-line:underline;text-decoration-color:red", "acemnorstu")

	lines := underlineOps(res)
	if len(lines) == 0 {
		t.Fatal("want >0 decoration ops, got 0")
	}

	for _, line := range lines {
		if !near(line.R, 1) || !near(line.G, 0) || !near(line.B, 0) {
			t.Errorf("decoration color = (%.3f, %.3f, %.3f), want (1, 0, 0)",
				line.R, line.G, line.B)
		}
	}
}

// TestBehaviorTextDecorationThicknessSetsWidth is text-decoration-thickness:
// the declared thickness becomes the decoration stroke width. Reference:
// Chrome 143.0.7499.40 draws a 2pt rule for thickness:2pt.
func TestBehaviorTextDecorationThicknessSetsWidth(t *testing.T) {
	t.Parallel()

	res := behaviorPaintFxDecoBox(t,
		"text-decoration-line:underline;text-decoration-thickness:2pt", "acemnorstu")

	lines := underlineOps(res)
	if len(lines) == 0 {
		t.Fatal("want >0 decoration ops, got 0")
	}

	for _, line := range lines {
		if !near(line.Width, 2) {
			t.Errorf("decoration width = %.4fpt, want 2pt", line.Width)
		}
	}
}

// TestBehaviorTransformTranslateMovesPaint is transform:translate(): the
// translation bakes into the painted ops. Reference: Chrome 143.0.7499.40
// shifts the border box by the translated offset.
func TestBehaviorTransformTranslateMovesPaint(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="box" style="width:40pt;height:20pt;background:red;`+
		`transform:translate(10pt,5pt)"></div></body></html>`)

	moved := behaviorPaintFxXformed(res)
	if len(moved) == 0 {
		t.Fatal("transform:translate: want ops with a baked transform, got 0")
	}

	got := moved[0].Transform()
	if !near(got.E, 10) || !near(got.F, 5) {
		t.Errorf("baked translation = (%.4f, %.4f), want (10, 5)", got.E, got.F)
	}
}

// TestBehaviorTransformOriginShiftsRotation is transform-origin: rotating
// about the top left corner bakes a different matrix than rotating about the
// default center. Reference: Chrome 143.0.7499.40 moves the painted box when
// the origin changes under the same rotation.
func TestBehaviorTransformOriginShiftsRotation(t *testing.T) {
	t.Parallel()

	page := func(origin string) *Result {
		return layoutHTML(t, `<html><body style="margin:0">`+
			`<div id="box" style="width:40pt;height:20pt;background:red;`+
			`transform:rotate(90deg);transform-origin:`+origin+`"></div></body></html>`)
	}

	center := behaviorPaintFxXformed(page("50% 50%"))
	corner := behaviorPaintFxXformed(page("0 0"))

	if len(center) == 0 || len(corner) == 0 {
		t.Fatalf("want transformed ops for both origins, got center=%d corner=%d",
			len(center), len(corner))
	}

	c, k := center[0].Transform(), corner[0].Transform()
	if math.Abs(c.E-k.E) < 1 && math.Abs(c.F-k.F) < 1 {
		t.Errorf("origin shift too small: center E,F = (%.3f, %.3f), corner E,F = (%.3f, %.3f)",
			c.E, c.F, k.E, k.F)
	}
}

// TestBehaviorTranslatePropertyMovesPaint is the translate longhand: lengths
// bake as a translation on the painted ops. Reference: Chrome 143.0.7499.40
// shifts the border box by (tx, ty).
func TestBehaviorTranslatePropertyMovesPaint(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="box" style="width:40pt;height:20pt;background:red;`+
		`translate:10pt 5pt"></div></body></html>`)

	moved := behaviorPaintFxXformed(res)
	if len(moved) == 0 {
		t.Fatal("translate: want ops with a baked transform, got 0")
	}

	got := moved[0].Transform()
	if !near(got.E, 10) || !near(got.F, 5) {
		t.Errorf("baked translation = (%.4f, %.4f), want (10, 5)", got.E, got.F)
	}
}

// TestBehaviorRotatePropertyBakesMatrix is the rotate longhand: a 90deg turn
// bakes the y-down rotation into the painted ops. Reference: Chrome
// 143.0.7499.40 turns the border box a quarter turn clockwise.
func TestBehaviorRotatePropertyBakesMatrix(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="box" style="width:40pt;height:20pt;background:red;`+
		`rotate:90deg"></div></body></html>`)

	turned := behaviorPaintFxXformed(res)
	if len(turned) == 0 {
		t.Fatal("rotate: want ops with a baked transform, got 0")
	}

	got := turned[0].Transform()
	if got.IsIdentity() {
		t.Error("rotate:90deg baked identity, want a quarter-turn matrix")
	}

	if !near(got.A, 0) || !near(got.D, 0) || !near(got.B, 1) || !near(got.C, -1) {
		t.Errorf("rotate:90deg linear part = (%.4f, %.4f, %.4f, %.4f), want (0, 1, -1, 0)",
			got.A, got.B, got.C, got.D)
	}
}

// TestBehaviorScalePropertyScalesPaint is the scale longhand: the scale
// factors bake into the linear part of the painted ops. Reference: Chrome
// 143.0.7499.40 doubles the border box for scale:2.
func TestBehaviorScalePropertyScalesPaint(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="box" style="width:40pt;height:20pt;background:red;`+
		`scale:2 3"></div></body></html>`)

	scaled := behaviorPaintFxXformed(res)
	if len(scaled) == 0 {
		t.Fatal("scale: want ops with a baked transform, got 0")
	}

	got := scaled[0].Transform()
	if !near(got.A, 2) || !near(got.D, 3) {
		t.Errorf("scale:2 3 linear part = (%.4f, %.4f), want (2, 3)", got.A, got.D)
	}
}

// TestBehaviorFilterOpacityFoldsIntoPaint is filter:opacity(): the filter
// folds into the element opacity stamped on its painted ops, while no filter
// paints fully opaque. Reference: Chrome 143.0.7499.40 fades the box at 50%.
func TestBehaviorFilterOpacityFoldsIntoPaint(t *testing.T) {
	t.Parallel()

	page := func(extra string) *Result {
		return layoutHTML(t, `<html><body style="margin:0">`+
			`<div id="box" style="width:40pt;height:20pt;background:red;`+extra+`"></div></body></html>`)
	}

	findFill := func(res *Result) Op {
		for _, op := range res.Ops {
			if op.Kind == OpFillRect {
				return op
			}
		}

		return Op{}
	}

	faded := findFill(page("filter:opacity(50%)"))
	if faded.Kind != OpFillRect {
		t.Fatal("filter:opacity: no fill rect op painted")
	}

	if got := faded.Opacity(); !near(got, 0.5) {
		t.Errorf("filter:opacity(50%%): op opacity = %.4f, want 0.5", got)
	}

	solid := findFill(page(""))
	if got := solid.Opacity(); !near(got, 1) {
		t.Errorf("no filter: op opacity = %.4f, want 1", got)
	}
}

// TestBehaviorContentVisibilityAutoPaintsDescendants is
// content-visibility:auto: descendants still paint, unlike hidden. This
// locks the visible half of the hidden behavior (hidden skips paint per
// TestContentVisibilityHiddenSkipsDescendantLayout). Reference: Chrome
// 143.0.7499.40 paints auto content on first view.
func TestBehaviorContentVisibilityAutoPaintsDescendants(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="wrap" style="content-visibility:auto"><p id="in" style="margin:0">`+
		`shown words</p></div></body></html>`)

	found := false

	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text == "shown words" {
			found = true
		}
	}

	if !found {
		t.Error("content-visibility:auto: descendant text painted nowhere, want it painted")
	}

	inner := boxByID(t, res, "in")
	if inner.height <= 0 {
		t.Errorf("content-visibility:auto: inner height = %.4f, want > 0", inner.height)
	}
}

// TestBehaviorMixBlendModeOpensGroup is mix-blend-mode:multiply: the element
// opens a blend group whose markers bound its painted ops, while normal
// blending opens no multiply group. Reference: Chrome 143.0.7499.40 isolates
// the blended element from its backdrop.
func TestBehaviorMixBlendModeOpensGroup(t *testing.T) {
	t.Parallel()

	page := func(mode string) *Result {
		return layoutHTML(t, `<html><body style="margin:0;background:white">`+
			`<div id="blend" style="mix-blend-mode:`+mode+`;background:red;`+
			`width:40pt;height:20pt"><p id="in" style="margin:0">x</p></div></body></html>`)
	}

	groupMode := func(res *Result, wantBegin bool) string {
		begins, ends := 0, 0
		mode := ""

		for idx := range res.Ops {
			if res.Ops[idx].IsGroupBegin() {
				begins++

				if group := res.Ops[idx].Group(); group != nil {
					mode = group.Mode
				}
			}

			if res.Ops[idx].IsGroupEnd() {
				ends++
			}
		}

		if wantBegin && (begins == 0 || ends == 0) {
			t.Errorf("mix-blend-mode:multiply: begins=%d ends=%d, want both >0", begins, ends)
		}

		return mode
	}

	if got := groupMode(page("multiply"), true); got != "multiply" {
		t.Errorf("blend group mode = %q, want %q", got, "multiply")
	}

	if got := groupMode(page("normal"), false); got == "multiply" {
		t.Errorf("normal blending opened a multiply group, want none")
	}
}
