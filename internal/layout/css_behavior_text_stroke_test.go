package layout

import "testing"

// -webkit-text-stroke and its longhands are carried on the OpText extra. The
// engine does not rasterize the outline, so the drawing-list read
// (Op.TextStroke) is the observable result. 1px is 0.75pt.

func textStrokeOp(t *testing.T, style string) Op {
	t.Helper()

	res := layoutHTML(t, `<html><body style="margin:0"><p id="p" style="margin:0;color:#000000;`+style+`">hi</p></body></html>`)

	for _, op := range opsOfKind(res, OpText) {
		if op.Text == "hi" {
			return op
		}
	}

	t.Fatal("no text op for the paragraph")

	return Op{}
}

func TestBehaviorTextStrokeWidthAndColor(t *testing.T) {
	t.Parallel()

	width, color, ok := textStrokeOp(t, "-webkit-text-stroke-width:1px;-webkit-text-stroke-color:#ff0000").TextStroke()
	if !ok || !near(width, pxToPt(1)) || color != [3]float64{1, 0, 0} {
		t.Errorf("1px red stroke = (%.3fpt, %v, %v), want 0.75pt red", width, color, ok)
	}

	// No color longhand: the stroke follows the black fill.
	width, color, ok = textStrokeOp(t, "-webkit-text-stroke-width:1px").TextStroke()
	if !ok || !near(width, pxToPt(1)) || color != [3]float64{0, 0, 0} {
		t.Errorf("1px stroke with no color = (%.3fpt, %v, %v), want 0.75pt in the fill color", width, color, ok)
	}

	// Width 0, no width, a negative width, and a bad token paint no stroke.
	for _, style := range []string{
		"-webkit-text-stroke-width:0;-webkit-text-stroke-color:red",
		"-webkit-text-stroke-color:red",
		"-webkit-text-stroke-width:-1px",
		"-webkit-text-stroke-width:bogus",
	} {
		if _, _, ok = textStrokeOp(t, style).TextStroke(); ok {
			t.Errorf("%q produced a stroke, want none", style)
		}
	}

	// Keywords are line widths.
	if w, _, ok := textStrokeOp(t, "-webkit-text-stroke-width:thick").TextStroke(); !ok || w <= pxToPt(1) {
		t.Errorf("thick stroke width = %.3fpt, want more than 1px", w)
	}
}

func TestBehaviorTextStrokeShorthand(t *testing.T) {
	t.Parallel()

	type want struct {
		width float64
		color [3]float64
		ok    bool
	}

	cases := map[string]want{
		"-webkit-text-stroke:1px red":          {pxToPt(1), [3]float64{1, 0, 0}, true},
		"-webkit-text-stroke:red 1px":          {pxToPt(1), [3]float64{1, 0, 0}, true},
		"-webkit-text-stroke:1px":              {pxToPt(1), [3]float64{0, 0, 0}, true},
		"-webkit-text-stroke:red":              {0, [3]float64{}, false},
		"-webkit-text-stroke:2px rgb(0,0,255)": {pxToPt(2), [3]float64{0, 0, 1}, true},
		"-webkit-text-stroke:1px red blue":     {0, [3]float64{}, false},
	}

	for style, expect := range cases {
		width, color, ok := textStrokeOp(t, style).TextStroke()
		if ok != expect.ok || (ok && (!near(width, expect.width) || color != expect.color)) {
			t.Errorf("%q = (%.3fpt, %v, %v), want (%.3fpt, %v, %v)", style, width, color, ok,
				expect.width, expect.color, expect.ok)
		}
	}
}

func TestBehaviorTextStrokeInherits(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0"><div style="color:#000;-webkit-text-stroke:1px red">`+
		`<p style="margin:0">hi</p></div></body></html>`)

	for _, op := range opsOfKind(res, OpText) {
		if op.Text == "hi" {
			if width, color, ok := op.TextStroke(); !ok || !near(width, pxToPt(1)) || color != [3]float64{1, 0, 0} {
				t.Errorf("child stroke = (%.3fpt, %v, %v), want the parent's 1px red", width, color, ok)
			}

			return
		}
	}

	t.Fatal("no text op")
}
