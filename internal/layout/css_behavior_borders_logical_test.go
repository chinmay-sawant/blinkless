package layout

import "testing"

// This file covers the 24 logical border properties. Each test asserts used
// box geometry or emitted border ops, never stored strings. The engine lays
// out in points and 1 CSS pixel = 0.75pt (ptPerCSSPx in
// css_review_02_test.go), so expectations are written in CSS pixels and
// converted with pxToPt before comparing, which is how Chrome reports it.
// Reference browser for every case: Chrome 143.0.7499.40.
// Logical mapping under test is horizontal-tb ltr: block maps to
// top/bottom, inline maps to left/right, block-start to top, block-end to
// bottom, inline-start to left, inline-end to right.

// behaviorBordLogHoriz returns horizontal (W > 0, H == 0) OpLine ops.
func behaviorBordLogHoriz(ops []Op) []Op {
	var out []Op

	for _, paintOp := range ops {
		if paintOp.Kind == OpLine && paintOp.H == 0 && paintOp.W > 0 {
			out = append(out, paintOp)
		}
	}

	return out
}

// behaviorBordLogVert returns vertical (H > 0, W == 0) OpLine ops.
func behaviorBordLogVert(ops []Op) []Op {
	var out []Op

	for _, paintOp := range ops {
		if paintOp.Kind == OpLine && paintOp.W == 0 && paintOp.H > 0 {
			out = append(out, paintOp)
		}
	}

	return out
}

// behaviorBordLogHorizAt returns horizontal ops at the given Y edge.
func behaviorBordLogHorizAt(ops []Op, y float64) []Op {
	var out []Op

	for _, paintOp := range behaviorBordLogHoriz(ops) {
		if near(paintOp.Y, y) {
			out = append(out, paintOp)
		}
	}

	return out
}

// behaviorBordLogVertAt returns vertical ops at the given X edge.
func behaviorBordLogVertAt(ops []Op, x float64) []Op {
	var out []Op

	for _, paintOp := range behaviorBordLogVert(ops) {
		if near(paintOp.X, x) {
			out = append(out, paintOp)
		}
	}

	return out
}

// behaviorBordLogLayoutItem lays out one 200x40px box with style.
func behaviorBordLogLayoutItem(t *testing.T, style string) (*Result, *box) {
	t.Helper()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:200px;height:40px;`+style+`"></div>`+
		`</body></html>`)

	return res, boxByID(t, res, "item")
}

// behaviorBordLogEdgePair returns the first/second edge op lists of the item:
// top/bottom when horizontal, left/right otherwise.
func behaviorBordLogEdgePair(res *Result, item *box, horizontal bool) ([]Op, []Op) {
	if horizontal {
		return behaviorBordLogHorizAt(res.Ops, item.y),
			behaviorBordLogHorizAt(res.Ops, item.y+item.height)
	}

	return behaviorBordLogVertAt(res.Ops, item.x),
		behaviorBordLogVertAt(res.Ops, item.x+item.w)
}

// behaviorBordLogAssertPaintedEdges asserts every op in edges carries the
// wanted stroke width and color, and spans wantLen along its axis.
func behaviorBordLogAssertPaintedEdges(
	t *testing.T, edges []Op, wantPx, wantR, wantG, wantB, wantLen float64, horizontal bool, edgeName string,
) {
	t.Helper()

	for _, paintOp := range edges {
		if !near(paintOp.Width, pxToPt(wantPx)) {
			t.Errorf("%s edge stroke width = %.4fpt (%.2fpx), want %.2fpx",
				edgeName, paintOp.Width, paintOp.Width/ptPerCSSPx, wantPx)
		}

		if !near(paintOp.R, wantR) || !near(paintOp.G, wantG) || !near(paintOp.B, wantB) {
			t.Errorf("%s edge color = (%.2f, %.2f, %.2f), want (%.0f, %.0f, %.0f)",
				edgeName, paintOp.R, paintOp.G, paintOp.B, wantR, wantG, wantB)
		}

		length := paintOp.W
		if !horizontal {
			length = paintOp.H
		}

		if !near(length, wantLen) {
			t.Errorf("%s edge length = %.4fpt, want %.4fpt",
				edgeName, length, wantLen)
		}
	}
}

// behaviorBordLogAssertMixedSegments lays out one box and asserts the first
// edge splits into segments while the second stays single (or vice versa
// when firstDashed is false). firstName and secondName label the edges.
func behaviorBordLogAssertMixedSegments(
	t *testing.T, style string, horizontal, firstDashed bool, firstName, secondName string,
) {
	t.Helper()

	res, item := behaviorBordLogLayoutItem(t, style)
	first, second := behaviorBordLogEdgePair(res, item, horizontal)

	if firstDashed {
		if len(first) <= 1 {
			t.Errorf("dashed %s edge segments = %d, want more than 1", firstName, len(first))
		}

		if len(second) != 1 {
			t.Errorf("solid %s edge segments = %d, want 1", secondName, len(second))
		}

		return
	}

	if len(first) != 1 {
		t.Errorf("solid %s edge segments = %d, want 1", firstName, len(first))
	}

	if len(second) <= 1 {
		t.Errorf("dashed %s edge segments = %d, want more than 1", secondName, len(second))
	}
}
func behaviorBordLogAssertDashedExpands(t *testing.T, dashedStyle, solidStyle string, horizontal bool) {
	t.Helper()

	dashed, dashedBox := behaviorBordLogLayoutItem(t, dashedStyle)
	solid, solidBox := behaviorBordLogLayoutItem(t, solidStyle)
	dFirst, dSecond := behaviorBordLogEdgePair(dashed, dashedBox, horizontal)
	sFirst, sSecond := behaviorBordLogEdgePair(solid, solidBox, horizontal)

	if len(dFirst) <= 1 || len(dSecond) <= 1 {
		t.Errorf("dashed edge segments = %d / %d, want more than 1 each", len(dFirst), len(dSecond))
	}

	if len(sFirst) != 1 || len(sSecond) != 1 {
		t.Errorf("solid edge segments = %d / %d, want 1 each", len(sFirst), len(sSecond))
	}
}

// behaviorBordLogAssertEdgeWidths lays out one box and asserts the first and
// second edges carry the wanted stroke widths.
func behaviorBordLogAssertEdgeWidths(t *testing.T, style string, wantFirst, wantSecond float64, horizontal bool) {
	t.Helper()

	res, item := behaviorBordLogLayoutItem(t, style)
	first, second := behaviorBordLogEdgePair(res, item, horizontal)

	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("edges = first %d second %d, want 1 and 1", len(first), len(second))
	}

	if !near(first[0].Width, pxToPt(wantFirst)) {
		t.Errorf("first stroke width = %.4fpt (%.2fpx), want %.2fpx",
			first[0].Width, first[0].Width/ptPerCSSPx, wantFirst)
	}

	if !near(second[0].Width, pxToPt(wantSecond)) {
		t.Errorf("second stroke width = %.4fpt (%.2fpx), want %.2fpx",
			second[0].Width, second[0].Width/ptPerCSSPx, wantSecond)
	}
}

// behaviorBordLogCheckHoriz lays out one 200x40px box with style, collects
// the horizontal edge ops at its top and bottom, asserts the counts, and
// hands the lists to check for property-specific assertions.
func behaviorBordLogCheckHoriz(
	t *testing.T, style string, wantTop, wantBottom int,
	check func(top, bottom []Op, res *Result, item *box),
) {
	t.Helper()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:200px;height:40px;`+style+`"></div>`+
		`</body></html>`)

	item := boxByID(t, res, "item")
	top := behaviorBordLogHorizAt(res.Ops, item.y)
	bottom := behaviorBordLogHorizAt(res.Ops, item.y+item.height)

	if len(top) != wantTop {
		t.Fatalf("top edge ops = %d, want %d", len(top), wantTop)
	}

	if len(bottom) != wantBottom {
		t.Fatalf("bottom edge ops = %d, want %d", len(bottom), wantBottom)
	}

	check(top, bottom, res, item)
}

// behaviorBordLogCheckVert is the vertical twin: left and right edge ops of
// a 200x40px box.
func behaviorBordLogCheckVert(
	t *testing.T, style string, wantLeft, wantRight int,
	check func(left, right []Op, res *Result, item *box),
) {
	t.Helper()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:200px;height:40px;`+style+`"></div>`+
		`</body></html>`)

	item := boxByID(t, res, "item")
	left := behaviorBordLogVertAt(res.Ops, item.x)
	right := behaviorBordLogVertAt(res.Ops, item.x+item.w)

	if len(left) != wantLeft {
		t.Fatalf("left edge ops = %d, want %d", len(left), wantLeft)
	}

	if len(right) != wantRight {
		t.Fatalf("right edge ops = %d, want %d", len(right), wantRight)
	}

	check(left, right, res, item)
}

// TestBehaviorBorderBlockPaintsTopBottomEdges is border-block: one shorthand
// paints the top and bottom edges and leaves left and right empty.
// Reference: Chrome 143.0.7499.40, 200px by 40px box, 4px solid red block
// borders paint 200px top and bottom edges, no vertical edges.
func TestBehaviorBorderBlockPaintsTopBottomEdges(t *testing.T) {
	t.Parallel()

	behaviorBordLogCheckHoriz(t, `border-block:4px solid red`, 1, 1,
		func(top, bottom []Op, res *Result, item *box) {
			behaviorBordLogAssertPaintedEdges(t, append(top, bottom...),
				4, 1, 0, 0, item.w, true, "block")

			if got := len(behaviorBordLogVert(res.Ops)); got != 0 {
				t.Errorf("vertical border ops = %d, want 0 (inline sides empty)", got)
			}
		})
}

// TestBehaviorBorderBlockColorPaintsBlockEdges is border-block-color: the
// color applies to the top and bottom edges only.
// Reference: Chrome 143.0.7499.40, 200px by 40px box, 4px solid edges,
// border-block-color red paints both horizontal edges red.
func TestBehaviorBorderBlockColorPaintsBlockEdges(t *testing.T) {
	t.Parallel()

	behaviorBordLogCheckHoriz(t,
		`border-top-width:4px;border-top-style:solid;`+
			`border-bottom-width:4px;border-bottom-style:solid;border-block-color:red`,
		1, 1, func(top, bottom []Op, res *Result, _ *box) {
			for _, paintOp := range append(top, bottom...) {
				if !near(paintOp.R, 1) || !near(paintOp.G, 0) || !near(paintOp.B, 0) {
					t.Errorf("block edge color = (%.2f, %.2f, %.2f), want red", paintOp.R, paintOp.G, paintOp.B)
				}

				if !near(paintOp.Width, pxToPt(4)) {
					t.Errorf("block edge width = %.4fpt, want 4px", paintOp.Width)
				}
			}

			if got := len(behaviorBordLogVert(res.Ops)); got != 0 {
				t.Errorf("vertical border ops = %d, want 0", got)
			}
		})
}

// TestBehaviorBorderBlockStyleExpandsBlockSegments is border-block-style:
// dashed expands both horizontal edges into segments, solid keeps one each.
// Reference: Chrome 143.0.7499.40, 200px block edges, 3px width.
func TestBehaviorBorderBlockStyleExpandsBlockSegments(t *testing.T) {
	t.Parallel()

	behaviorBordLogAssertDashedExpands(t,
		`border-top-width:3px;border-bottom-width:3px;border-block-style:dashed`,
		`border-top-width:3px;border-bottom-width:3px;border-block-style:solid`, true)
}

// TestBehaviorBorderBlockWidthSetsBlockStrokeWidth is border-block-width:
// the width applies to the top and bottom strokes.
// Reference: Chrome 143.0.7499.40, 200px by 40px box, 6px block width
// paints 6px top and bottom strokes.
func TestBehaviorBorderBlockWidthSetsBlockStrokeWidth(t *testing.T) {
	t.Parallel()

	behaviorBordLogAssertEdgeWidths(t,
		`border-top-style:solid;border-bottom-style:solid;border-block-width:6px`, 6, 6, true)
}

// TestBehaviorBorderBlockStartPaintsTopEdge is border-block-start: only the
// top edge paints.
// Reference: Chrome 143.0.7499.40, 200px by 40px box, 4px solid red
// block-start paints the 200px top edge only.
func TestBehaviorBorderBlockStartPaintsTopEdge(t *testing.T) {
	t.Parallel()

	behaviorBordLogCheckHoriz(t, `border-block-start:4px solid red`, 1, 0,
		func(top, _ []Op, res *Result, _ *box) {
			if !near(top[0].Width, pxToPt(4)) {
				t.Errorf("top stroke width = %.4fpt, want 4px", top[0].Width)
			}

			if !near(top[0].R, 1) || !near(top[0].G, 0) || !near(top[0].B, 0) {
				t.Errorf("top color = (%.2f, %.2f, %.2f), want red", top[0].R, top[0].G, top[0].B)
			}

			if got := len(behaviorBordLogVert(res.Ops)); got != 0 {
				t.Errorf("vertical border ops = %d, want 0", got)
			}
		})
}

// TestBehaviorBorderBlockStartColorPaintsTopEdge is
// border-block-start-color: only the top edge takes the color.
// Reference: Chrome 143.0.7499.40, 200px by 40px box, 4px solid edges,
// block-start red leaves the top red and the bottom black.
func TestBehaviorBorderBlockStartColorPaintsTopEdge(t *testing.T) {
	t.Parallel()

	behaviorBordLogCheckHoriz(t,
		`border-top-width:4px;border-top-style:solid;`+
			`border-bottom-width:4px;border-bottom-style:solid;border-bottom-color:black;`+
			`border-block-start-color:red`,
		1, 1, func(top, bottom []Op, _ *Result, _ *box) {
			if !near(top[0].R, 1) || !near(top[0].G, 0) || !near(top[0].B, 0) {
				t.Errorf("top color = (%.2f, %.2f, %.2f), want red", top[0].R, top[0].G, top[0].B)
			}

			if !near(bottom[0].R, 0) || !near(bottom[0].G, 0) || !near(bottom[0].B, 0) {
				t.Errorf("bottom color = (%.2f, %.2f, %.2f), want black (untouched)", bottom[0].R, bottom[0].G, bottom[0].B)
			}
		})
}

// TestBehaviorBorderBlockStartStyleExpandsTopSegments is
// border-block-start-style: dashed expands only the top edge.
// Reference: Chrome 143.0.7499.40, 200px edges, 3px width, dashed top and
// solid bottom.
func TestBehaviorBorderBlockStartStyleExpandsTopSegments(t *testing.T) {
	t.Parallel()

	behaviorBordLogAssertMixedSegments(t,
		`border-top-width:3px;border-bottom-width:3px;`+
			`border-bottom-style:solid;border-block-start-style:dashed`,
		true, true, "top", "bottom")
}

// TestBehaviorBorderBlockStartWidthSetsTopStrokeWidth is
// border-block-start-width: only the top stroke takes the width.
// Reference: Chrome 143.0.7499.40, 200px by 40px box, 6px block-start width
// over a 2px bottom width.
func TestBehaviorBorderBlockStartWidthSetsTopStrokeWidth(t *testing.T) {
	t.Parallel()

	behaviorBordLogAssertEdgeWidths(t,
		`border-top-style:solid;border-bottom-style:solid;border-bottom-width:2px;border-block-start-width:6px`, 6, 2, true)
}

// TestBehaviorBorderBlockEndPaintsBottomEdge is border-block-end: only the
// bottom edge paints.
// Reference: Chrome 143.0.7499.40, 200px by 40px box, 4px solid red
// block-end paints the 200px bottom edge only.
func TestBehaviorBorderBlockEndPaintsBottomEdge(t *testing.T) {
	t.Parallel()

	behaviorBordLogCheckHoriz(t, `border-block-end:4px solid red`, 0, 1,
		func(_, bottom []Op, _ *Result, _ *box) {
			if !near(bottom[0].Width, pxToPt(4)) {
				t.Errorf("bottom stroke width = %.4fpt, want 4px", bottom[0].Width)
			}

			if !near(bottom[0].R, 1) || !near(bottom[0].G, 0) || !near(bottom[0].B, 0) {
				t.Errorf("bottom color = (%.2f, %.2f, %.2f), want red", bottom[0].R, bottom[0].G, bottom[0].B)
			}
		})
}

// TestBehaviorBorderBlockEndColorPaintsBottomEdge is
// border-block-end-color: only the bottom edge takes the color.
// Reference: Chrome 143.0.7499.40, 200px by 40px box, 4px solid edges,
// block-end blue leaves the bottom blue and the top black.
func TestBehaviorBorderBlockEndColorPaintsBottomEdge(t *testing.T) {
	t.Parallel()

	behaviorBordLogCheckHoriz(t,
		`border-top-width:4px;border-top-style:solid;`+
			`border-top-color:black;border-bottom-width:4px;border-bottom-style:solid;`+
			`border-block-end-color:blue`,
		1, 1, func(top, bottom []Op, _ *Result, _ *box) {
			if !near(bottom[0].B, 1) || !near(bottom[0].R, 0) || !near(bottom[0].G, 0) {
				t.Errorf("bottom color = (%.2f, %.2f, %.2f), want blue", bottom[0].R, bottom[0].G, bottom[0].B)
			}

			if !near(top[0].R, 0) || !near(top[0].G, 0) || !near(top[0].B, 0) {
				t.Errorf("top color = (%.2f, %.2f, %.2f), want black (untouched)", top[0].R, top[0].G, top[0].B)
			}
		})
}

// TestBehaviorBorderBlockEndStyleExpandsBottomSegments is
// border-block-end-style: dashed expands only the bottom edge.
// Reference: Chrome 143.0.7499.40, 200px edges, 3px width, solid top and
// dashed bottom.
func TestBehaviorBorderBlockEndStyleExpandsBottomSegments(t *testing.T) {
	t.Parallel()

	behaviorBordLogAssertMixedSegments(t,
		`border-top-width:3px;border-bottom-width:3px;`+
			`border-top-style:solid;border-block-end-style:dashed`,
		true, false, "top", "bottom")
}

// TestBehaviorBorderBlockEndWidthSetsBottomStrokeWidth is
// border-block-end-width: only the bottom stroke takes the width.
// Reference: Chrome 143.0.7499.40, 200px by 40px box, 2px top width and 6px
// block-end width.
func TestBehaviorBorderBlockEndWidthSetsBottomStrokeWidth(t *testing.T) {
	t.Parallel()

	behaviorBordLogAssertEdgeWidths(t,
		`border-top-style:solid;border-bottom-style:solid;border-top-width:2px;border-block-end-width:6px`, 2, 6, true)
}

// TestBehaviorBorderInlinePaintsLeftRightEdges is border-inline: one shorthand
// paints the left and right edges and leaves top and bottom empty.
// Reference: Chrome 143.0.7499.40, 200px by 40px box, 4px solid blue inline
// borders paint 40px left and right edges, no horizontal edges.
func TestBehaviorBorderInlinePaintsLeftRightEdges(t *testing.T) {
	t.Parallel()

	behaviorBordLogCheckVert(t, `border-inline:4px solid blue`, 1, 1,
		func(left, right []Op, res *Result, item *box) {
			behaviorBordLogAssertPaintedEdges(t, append(left, right...),
				4, 0, 0, 1, item.height, false, "inline")

			if got := len(behaviorBordLogHoriz(res.Ops)); got != 0 {
				t.Errorf("horizontal border ops = %d, want 0 (block sides empty)", got)
			}
		})
}

// TestBehaviorBorderInlineColorPaintsInlineEdges is border-inline-color: the
// color applies to the left and right edges only.
// Reference: Chrome 143.0.7499.40, 200px by 40px box, 4px solid edges,
// border-inline-color blue paints both vertical edges blue.
func TestBehaviorBorderInlineColorPaintsInlineEdges(t *testing.T) {
	t.Parallel()

	behaviorBordLogCheckVert(t,
		`border-left-width:4px;border-left-style:solid;`+
			`border-right-width:4px;border-right-style:solid;border-inline-color:blue`,
		1, 1, func(left, right []Op, res *Result, _ *box) {
			for _, paintOp := range append(left, right...) {
				if !near(paintOp.B, 1) || !near(paintOp.R, 0) || !near(paintOp.G, 0) {
					t.Errorf("inline edge color = (%.2f, %.2f, %.2f), want blue", paintOp.R, paintOp.G, paintOp.B)
				}

				if !near(paintOp.Width, pxToPt(4)) {
					t.Errorf("inline edge width = %.4fpt, want 4px", paintOp.Width)
				}
			}

			if got := len(behaviorBordLogHoriz(res.Ops)); got != 0 {
				t.Errorf("horizontal border ops = %d, want 0", got)
			}
		})
}

// TestBehaviorBorderInlineStyleExpandsInlineSegments is border-inline-style:
// dashed expands both vertical edges into segments, solid keeps one each.
// Reference: Chrome 143.0.7499.40, 40px inline edges, 3px width.
func TestBehaviorBorderInlineStyleExpandsInlineSegments(t *testing.T) {
	t.Parallel()

	dashed := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:200px;height:40px;border-left-width:3px;border-right-width:3px;`+
		`border-inline-style:dashed"></div>`+
		`</body></html>`)
	solid := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:200px;height:40px;border-left-width:3px;border-right-width:3px;`+
		`border-inline-style:solid"></div>`+
		`</body></html>`)

	dashedBox := boxByID(t, dashed, "item")
	solidBox := boxByID(t, solid, "item")

	if got := len(behaviorBordLogVertAt(dashed.Ops, dashedBox.x)); got <= 1 {
		t.Errorf("dashed left edge segments = %d, want more than 1", got)
	}

	if got := len(behaviorBordLogVertAt(dashed.Ops, dashedBox.x+dashedBox.w)); got <= 1 {
		t.Errorf("dashed right edge segments = %d, want more than 1", got)
	}

	if got := len(behaviorBordLogVertAt(solid.Ops, solidBox.x)); got != 1 {
		t.Errorf("solid left edge segments = %d, want 1", got)
	}

	if got := len(behaviorBordLogVertAt(solid.Ops, solidBox.x+solidBox.w)); got != 1 {
		t.Errorf("solid right edge segments = %d, want 1", got)
	}
}

// TestBehaviorBorderInlineWidthSetsInlineStrokeWidth is border-inline-width:
// the width applies to the left and right strokes.
// Reference: Chrome 143.0.7499.40, 200px by 40px box, 7px inline width
// paints 7px left and right strokes.
func TestBehaviorBorderInlineWidthSetsInlineStrokeWidth(t *testing.T) {
	t.Parallel()

	behaviorBordLogAssertEdgeWidths(t,
		`border-left-style:solid;border-right-style:solid;border-inline-width:7px`, 7, 7, false)
}

// TestBehaviorBorderInlineStartPaintsLeftEdge is border-inline-start: only the
// left edge paints.
// Reference: Chrome 143.0.7499.40, 200px by 40px box, 5px solid red
// inline-start paints the 40px left edge only.
func TestBehaviorBorderInlineStartPaintsLeftEdge(t *testing.T) {
	t.Parallel()

	behaviorBordLogCheckVert(t, `border-inline-start:5px solid red`, 1, 0,
		func(left, _ []Op, _ *Result, _ *box) {
			if !near(left[0].Width, pxToPt(5)) {
				t.Errorf("left stroke width = %.4fpt, want 5px", left[0].Width)
			}

			if !near(left[0].R, 1) || !near(left[0].G, 0) || !near(left[0].B, 0) {
				t.Errorf("left color = (%.2f, %.2f, %.2f), want red", left[0].R, left[0].G, left[0].B)
			}
		})
}

// TestBehaviorBorderInlineStartColorPaintsLeftEdge is
// border-inline-start-color: only the left edge takes the color.
// Reference: Chrome 143.0.7499.40, 200px by 40px box, 4px solid edges,
// inline-start red leaves the left red and the right black.
func TestBehaviorBorderInlineStartColorPaintsLeftEdge(t *testing.T) {
	t.Parallel()

	behaviorBordLogCheckVert(t,
		`border-left-width:4px;border-left-style:solid;`+
			`border-right-width:4px;border-right-style:solid;border-right-color:black;`+
			`border-inline-start-color:red`,
		1, 1, func(left, right []Op, _ *Result, _ *box) {
			if !near(left[0].R, 1) || !near(left[0].G, 0) || !near(left[0].B, 0) {
				t.Errorf("left color = (%.2f, %.2f, %.2f), want red", left[0].R, left[0].G, left[0].B)
			}

			if !near(right[0].R, 0) || !near(right[0].G, 0) || !near(right[0].B, 0) {
				t.Errorf("right color = (%.2f, %.2f, %.2f), want black (untouched)", right[0].R, right[0].G, right[0].B)
			}
		})
}

// TestBehaviorBorderInlineStartStyleExpandsLeftSegments is
// border-inline-start-style: dashed expands only the left edge.
// Reference: Chrome 143.0.7499.40, 40px edges, 3px width, dashed left and
// solid right.
func TestBehaviorBorderInlineStartStyleExpandsLeftSegments(t *testing.T) {
	t.Parallel()

	behaviorBordLogAssertMixedSegments(t,
		`border-left-width:3px;border-right-width:3px;`+
			`border-right-style:solid;border-inline-start-style:dashed`,
		false, true, "left", "right")
}

// TestBehaviorBorderInlineStartWidthSetsLeftStrokeWidth is
// border-inline-start-width: only the left stroke takes the width.
// Reference: Chrome 143.0.7499.40, 200px by 40px box, 7px inline-start width
// over a 2px right width.
func TestBehaviorBorderInlineStartWidthSetsLeftStrokeWidth(t *testing.T) {
	t.Parallel()

	behaviorBordLogAssertEdgeWidths(t,
		`border-left-style:solid;border-right-style:solid;border-right-width:2px;border-inline-start-width:7px`, 7, 2, false)
}

// TestBehaviorBorderInlineEndPaintsRightEdge is border-inline-end: only the
// right edge paints.
// Reference: Chrome 143.0.7499.40, 200px by 40px box, 5px solid blue
// inline-end paints the 40px right edge only.
func TestBehaviorBorderInlineEndPaintsRightEdge(t *testing.T) {
	t.Parallel()

	behaviorBordLogCheckVert(t, `border-inline-end:5px solid blue`, 0, 1,
		func(_, right []Op, _ *Result, _ *box) {
			if !near(right[0].Width, pxToPt(5)) {
				t.Errorf("right stroke width = %.4fpt, want 5px", right[0].Width)
			}

			if !near(right[0].B, 1) || !near(right[0].R, 0) || !near(right[0].G, 0) {
				t.Errorf("right color = (%.2f, %.2f, %.2f), want blue", right[0].R, right[0].G, right[0].B)
			}
		})
}

// TestBehaviorBorderInlineEndColorPaintsRightEdge is
// border-inline-end-color: only the right edge takes the color.
// Reference: Chrome 143.0.7499.40, 200px by 40px box, 4px solid edges,
// inline-end blue leaves the right blue and the left black.
func TestBehaviorBorderInlineEndColorPaintsRightEdge(t *testing.T) {
	t.Parallel()

	behaviorBordLogCheckVert(t,
		`border-left-width:4px;border-left-style:solid;`+
			`border-left-color:black;border-right-width:4px;border-right-style:solid;`+
			`border-inline-end-color:blue`,
		1, 1, func(left, right []Op, _ *Result, _ *box) {
			if !near(right[0].B, 1) || !near(right[0].R, 0) || !near(right[0].G, 0) {
				t.Errorf("right color = (%.2f, %.2f, %.2f), want blue", right[0].R, right[0].G, right[0].B)
			}

			if !near(left[0].R, 0) || !near(left[0].G, 0) || !near(left[0].B, 0) {
				t.Errorf("left color = (%.2f, %.2f, %.2f), want black (untouched)", left[0].R, left[0].G, left[0].B)
			}
		})
}

// TestBehaviorBorderInlineEndStyleExpandsRightSegments is
// border-inline-end-style: dashed expands only the right edge.
// Reference: Chrome 143.0.7499.40, 40px edges, 3px width, solid left and
// dashed right.
func TestBehaviorBorderInlineEndStyleExpandsRightSegments(t *testing.T) {
	t.Parallel()

	behaviorBordLogAssertMixedSegments(t,
		`border-left-width:3px;border-right-width:3px;`+
			`border-left-style:solid;border-inline-end-style:dashed`,
		false, false, "left", "right")
}

// TestBehaviorBorderInlineEndWidthSetsRightStrokeWidth is
// border-inline-end-width: only the right stroke takes the width.
// Reference: Chrome 143.0.7499.40, 200px by 40px box, 2px left width and 7px
// inline-end width.
func TestBehaviorBorderInlineEndWidthSetsRightStrokeWidth(t *testing.T) {
	t.Parallel()

	behaviorBordLogAssertEdgeWidths(t,
		`border-left-style:solid;border-right-style:solid;border-left-width:2px;border-inline-end-width:7px`, 2, 7, false)
}
