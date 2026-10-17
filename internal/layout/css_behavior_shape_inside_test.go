package layout

import (
	"strings"
	"testing"
)

// shape-inside / shape-padding behavior tests: every observable below is
// asserted through a USED value (an emitted text op X/W grouped into visual
// lines by inlineLines), never through a stored style string. No browser
// implements shape-inside (Chrome 143 has no BCD support), so the expectations
// encode this ledger's lite rule: the element's own lines narrow to the
// inside contour, a definite content height is required (an auto-height box
// has no final height during line layout), and floats beside the element
// still narrow the line after the inside inset. Shared helpers layoutHTML,
// boxByID, inlineLines, near, pxToPt, and behaviorTextRemAssertSameOps come
// from layout_test.go, css_review_02_test.go, inline_balance_test.go, and
// css_behavior_text_remaining_test.go and are never redeclared here.

// shapeInsideTestSlack is the point slack allowed between compared line
// widths or positions.
const shapeInsideTestSlack = 1

// shapeInsideTestBody returns a deterministic paragraph body. words counts
// "wrap" tokens before the trailing "end".
func shapeInsideTestBody(words int) string {
	return strings.Repeat("wrap ", words) + "end"
}

// shapeInsideTestDoc lays the body into a 200x200 CSS pixel box at 12pt.
func shapeInsideTestDoc(t *testing.T, body, extra string) *Result {
	t.Helper()

	return layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="box" style="margin:0;width:200px;height:200px;font-size:12pt;`+extra+`">`+
		body+`</div></body></html>`)
}

// shapeInsideTestSameLines reports whether two visual line lists paint the
// same text at the same x and width.
func shapeInsideTestSameLines(first, second []inlineLine) bool {
	if len(first) != len(second) {
		return false
	}

	for i := range first {
		if first[i].text != second[i].text ||
			!near(first[i].x, second[i].x) || !near(first[i].w, second[i].w) {
			return false
		}
	}

	return true
}

// TestBehaviorShapeInsideCircleTapersLines is the phase 2 gate: a circle()
// anchored at the top of a definite-height box makes a middle line shorter
// than the first line, moves the first line's x right, and narrows it versus
// auto. The default circle() (closest-side against the 200x200px content box
// = 45pt radius centered at midline) also changes the painted geometry, so a
// definite-height box is not a no-op for it.
func TestBehaviorShapeInsideCircleTapersLines(t *testing.T) {
	t.Parallel()

	body := shapeInsideTestBody(6)
	plain := inlineLines(shapeInsideTestDoc(t, body, ""))
	shaped := inlineLines(shapeInsideTestDoc(t, body, "shape-inside:circle(60px at 50% 0px);"))

	if len(shaped) < 3 {
		t.Fatalf("shaped lines = %d, want >= 3 (%+v)", len(shaped), shaped)
	}

	if !(shaped[1].w < shaped[0].w-shapeInsideTestSlack) {
		t.Errorf("middle line width = %.3fpt, want shorter than first line %.3fpt",
			shaped[1].w, shaped[0].w)
	}

	if !(shaped[0].w < plain[0].w-shapeInsideTestSlack) {
		t.Errorf("shaped first line width = %.3fpt, want narrower than auto %.3fpt",
			shaped[0].w, plain[0].w)
	}

	if !(shaped[0].x > plain[0].x+shapeInsideTestSlack) {
		t.Errorf("shaped first line x = %.3fpt, want right of auto x = %.3fpt",
			shaped[0].x, plain[0].x)
	}

	defaultShaped := inlineLines(shapeInsideTestDoc(t, body, "shape-inside:circle();"))
	if shapeInsideTestSameLines(defaultShaped, plain) {
		t.Errorf("shape-inside:circle() on a definite-height box painted like auto: %+v", defaultShaped)
	}
}

// TestBehaviorShapeInsidePaddingShrinksMiddleLine is the phase 3 gate:
// shape-padding 10px (and the equivalent 5% of the 150pt reference diagonal)
// makes a middle line shorter than shape-padding 0px, and 0px paints
// exactly like no shape-padding declaration.
func TestBehaviorShapeInsidePaddingShrinksMiddleLine(t *testing.T) {
	t.Parallel()

	body := shapeInsideTestBody(10)
	shape := "shape-inside:circle(90px at 50% 0px);"

	noPadding := shapeInsideTestDoc(t, body, shape)
	zero := shapeInsideTestDoc(t, body, shape+"shape-padding:0px;")
	ten := shapeInsideTestDoc(t, body, shape+"shape-padding:10px;")
	percent := shapeInsideTestDoc(t, body, shape+"shape-padding:5%;")

	// 0 matches the phase-2 layout, and 5% of the 150pt diagonal is 7.5pt,
	// the same shrink as 10px.
	behaviorTextRemAssertSameOps(t, noPadding, zero)
	behaviorTextRemAssertSameOps(t, ten, percent)

	zeroLines := inlineLines(zero)
	tenLines := inlineLines(ten)

	if len(zeroLines) < 3 || len(tenLines) < 3 {
		t.Fatalf("lines = %d / %d, want >= 3 for both padding values", len(zeroLines), len(tenLines))
	}

	if !(tenLines[1].w < zeroLines[1].w-shapeInsideTestSlack) {
		t.Errorf("padding 10px middle line width = %.3fpt, want shorter than padding 0 %.3fpt",
			tenLines[1].w, zeroLines[1].w)
	}

	if !(tenLines[1].x > zeroLines[1].x+shapeInsideTestSlack) {
		t.Errorf("padding 10px middle line x = %.3fpt, want right of padding 0 x = %.3fpt",
			tenLines[1].x, zeroLines[1].x)
	}

	// The bare circle() form: on a definite-height box the closest-side
	// radius resolves, and padding 10px drops the second line by one word.
	bare := inlineLines(shapeInsideTestDoc(t, body, "shape-inside:circle();"))
	barePadded := inlineLines(shapeInsideTestDoc(t, body, "shape-inside:circle();shape-padding:10px;"))

	if len(bare) < 3 || len(barePadded) < 3 {
		t.Fatalf("bare circle() lines = %d / %d, want >= 3 for both padding values", len(bare), len(barePadded))
	}

	if !(barePadded[1].w < bare[1].w-shapeInsideTestSlack) {
		t.Errorf("bare circle() padding 10px middle line width = %.3fpt, want shorter than padding 0 %.3fpt",
			barePadded[1].w, bare[1].w)
	}
}

// TestBehaviorShapeInsidePaddingAloneNoEffect: shape-padding without a
// shape-inside contour does nothing, including on the first line's x.
func TestBehaviorShapeInsidePaddingAloneNoEffect(t *testing.T) {
	t.Parallel()

	body := shapeInsideTestBody(10)

	behaviorTextRemAssertSameOps(t,
		shapeInsideTestDoc(t, body, ""),
		shapeInsideTestDoc(t, body, "shape-padding:100px;"))
}

// TestBehaviorShapeInsideOversizePaddingKeepsZeroWidthLines: shape-padding
// larger than the shape leaves an empty interval, so every line stays at its
// Y with width 0 and packs exactly one word. No page break is invented and
// no word is dropped.
func TestBehaviorShapeInsideOversizePaddingKeepsZeroWidthLines(t *testing.T) {
	t.Parallel()

	body := shapeInsideTestBody(10)
	plain := inlineLines(shapeInsideTestDoc(t, body, ""))
	oversize := inlineLines(shapeInsideTestDoc(t, body, "shape-inside:inset(0);shape-padding:100px;"))

	if len(oversize) != 11 {
		t.Fatalf("oversize lines = %d, want 11 (one word per zero-width line)", len(oversize))
	}

	for i, line := range oversize {
		if strings.Contains(line.text, " ") {
			t.Errorf("oversize line %d = %q, want a single word", i, line.text)
		}
	}

	if !near(oversize[0].y, plain[0].y) {
		t.Errorf("oversize first line y = %.3f, want the plain first line y %.3f (no page break)",
			oversize[0].y, plain[0].y)
	}
}

// TestBehaviorShapeInsideShapeBoxMovesReference: the optional shape-box
// keyword changes the reference rectangle, unlike shape-outside where the
// keyword is stored and ignored. With 20px padding, a 2px border, and a 10px
// margin, the first line starts at the named box's left edge.
func TestBehaviorShapeInsideShapeBoxMovesReference(t *testing.T) {
	t.Parallel()

	body := shapeInsideTestBody(10)

	doc := func(box string) []inlineLine {
		return inlineLines(shapeInsideTestDoc(t, body,
			"margin:10px;border:2px solid black;padding:20px;shape-inside:inset(0) "+box+";"))
	}

	// content left = margin 10px + border 2px + padding 20px = 32px.
	testCases := []struct {
		box   string
		wantX float64
	}{
		{"content-box", pxToPt(32)},
		{"padding-box", pxToPt(12)},
		{"border-box", pxToPt(10)},
		{"margin-box", 0},
	}

	content := doc("content-box")

	for _, testCase := range testCases {
		lines := doc(testCase.box)

		if len(lines) == 0 {
			t.Fatalf("shape-box %s produced no lines", testCase.box)
		}

		if !near(lines[0].x, testCase.wantX) {
			t.Errorf("shape-box %s first line x = %.3fpt, want %.3fpt",
				testCase.box, lines[0].x, testCase.wantX)
		}
	}

	padding := doc("padding-box")
	if !(padding[0].w > content[0].w+20) {
		t.Errorf("padding-box first line width = %.3fpt, want over 20pt wider than content-box %.3fpt",
			padding[0].w, content[0].w)
	}
}

// TestBehaviorShapeInsideOutsideShapeCopiesOutside: outside-shape reuses the
// element's supported shape-outside contour as the inside contour and is a
// no-op when shape-outside is none (or missing).
func TestBehaviorShapeInsideOutsideShapeCopiesOutside(t *testing.T) {
	t.Parallel()

	body := shapeInsideTestBody(6)
	direct := shapeInsideTestDoc(t, body, "shape-inside:circle(60px at 50% 0px);")
	copied := shapeInsideTestDoc(t, body,
		"shape-outside:circle(60px at 50% 0px);shape-inside:outside-shape;")

	behaviorTextRemAssertSameOps(t, direct, copied)

	behaviorTextRemAssertSameOps(t,
		shapeInsideTestDoc(t, body, ""),
		shapeInsideTestDoc(t, body, "shape-inside:outside-shape;"))
}

// TestBehaviorShapeInsideDoesNotInherit: the contour applies to the
// declaring element's own inline content only. A child block inside a shaped
// parent lays its own lines out at full width.
func TestBehaviorShapeInsideDoesNotInherit(t *testing.T) {
	t.Parallel()

	childBody := shapeInsideTestBody(6)

	doc := func(style string) *Result {
		return layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
			`<div style="margin:0;width:200px;height:200px;font-size:12pt;`+style+`">`+
			`<div id="child" style="margin:0;font-size:12pt">`+childBody+`</div></div></body></html>`)
	}

	behaviorTextRemAssertSameOps(t, doc(""), doc("shape-inside:circle(60px at 50% 0px);"))
}

// TestBehaviorShapeInsideStacksWithFloatExclusion: floats beside the element
// still apply after the inside inset. The first line keeps the float's edge
// as its x and is narrower than the float-only line, because both the float
// and the circle constrain it.
func TestBehaviorShapeInsideStacksWithFloatExclusion(t *testing.T) {
	t.Parallel()

	body := shapeInsideTestBody(6)

	doc := func(extra string) *Result {
		return layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
			`<div style="float:left;width:100px;height:100px"></div>`+
			`<div id="box" style="margin:0;width:200px;height:200px;font-size:12pt;`+extra+`">`+
			body+`</div></body></html>`)
	}

	floatOnly := inlineLines(doc(""))
	both := inlineLines(doc("shape-inside:circle(60px at 50% 0px);"))

	if len(floatOnly) == 0 || len(both) == 0 {
		t.Fatalf("float lines = %d / %d, want both non-empty", len(floatOnly), len(both))
	}

	if !near(both[0].x, floatOnly[0].x) {
		t.Errorf("shaped first line x = %.3fpt, want the float edge %.3fpt still applied",
			both[0].x, floatOnly[0].x)
	}

	if !(both[0].w < floatOnly[0].w-shapeInsideTestSlack) {
		t.Errorf("shaped first line width = %.3fpt, want narrower than float-only %.3fpt",
			both[0].w, floatOnly[0].w)
	}
}
