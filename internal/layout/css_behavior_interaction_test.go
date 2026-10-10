package layout

import (
	"strings"
	"testing"
)

// This file holds the interaction behavior tests: properties that change how
// a box responds to overflowing content or to the user (clipping on one
// axis, breaking long words, truncating with an ellipsis, masking an image,
// form control tinting). Every observable property below is asserted through
// a USED value (a clipped op width, a wrapped text op, an ellipsis suffix, a
// masked alpha, a widget fill color), never through a stored style string.
// The engine lays out in points and 1 CSS pixel = 0.75pt (ptPerCSSPx in
// css_review_02_test.go), so every expectation is written in CSS pixels and
// converted with pxToPt before comparing, which is how Chrome reports it.
// Reference browser for every case: Chrome 143.0.7499.40.
//
// Properties with no drawing-list consumer (pointer-events, cursor,
// user-select, caret-color, resize, appearance) pin the current no-op
// behavior with identical-geometry comparisons and are reported as
// unobservable, never as failing tests.

// behaviorInteractAssertSameOps asserts two results paint the same ops in the
// same order: same kind, geometry, and text. Used to pin no-op behavior for
// properties the drawing list cannot observe.
func behaviorInteractAssertSameOps(t *testing.T, first, second *Result) {
	t.Helper()

	if len(first.Ops) != len(second.Ops) {
		t.Fatalf("op count = %d vs %d, want identical geometry", len(first.Ops), len(second.Ops))
	}

	for i := range first.Ops {
		x, y := first.Ops[i], second.Ops[i]
		if x.Kind != y.Kind || x.Text != y.Text ||
			!near(x.X, y.X) || !near(x.Y, y.Y) ||
			!near(x.W, y.W) || !near(x.H, y.H) {
			t.Fatalf("op %d differs: %+v vs %+v, want identical geometry", i, x, y)
		}
	}
}

// behaviorInteractFillWidths returns the widths of every live fill-rect op.
func behaviorInteractFillWidths(res *Result) []float64 {
	var out []float64

	for _, op := range res.Ops {
		if op.Kind == OpFillRect {
			out = append(out, op.W)
		}
	}

	return out
}

// TestBehaviorOverflowXHiddenClipsWideChild is overflow-x: a 100px container
// with overflow-x:hidden clips its 300px child background to the padding box
// on the x axis, while overflow-x:visible keeps the full 300px fill.
// Reference: Chrome 143.0.7499.40.
func TestBehaviorOverflowXHiddenClipsWideChild(t *testing.T) {
	t.Parallel()

	src := func(overflow string) string {
		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="clip" style="width:100px;height:50px;overflow-x:` + overflow + `">` +
			`<div id="wide" style="width:300px;height:20px;background-color:#ff0000"></div>` +
			`</div></body></html>`
	}

	hidden := layoutHTML(t, src("hidden"))
	visible := layoutHTML(t, src("visible"))

	for _, w := range behaviorInteractFillWidths(hidden) {
		if w > pxToPt(100)+0.01 {
			t.Errorf("hidden fill width = %.4fpt (%.2fpx), want at most 100px",
				w, w/ptPerCSSPx)
		}
	}

	sawFull := false

	for _, w := range behaviorInteractFillWidths(visible) {
		if near(w, pxToPt(300)) {
			sawFull = true
		}
	}

	if !sawFull {
		t.Errorf("visible fill widths = %v, want a 300px child fill", behaviorInteractFillWidths(visible))
	}
}

// TestBehaviorOverflowYScrollClipsTallChild is overflow-y: a 200x50px scroll
// container clips its tall child to the padding box on the y axis (the engine
// has no user scroll, so scroll clips like hidden). Paint past the bottom
// edge is deactivated and the first lines stay live. Reference: Chrome
// 143.0.7499.40.
func TestBehaviorOverflowYScrollClipsTallChild(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="clip" style="width:200px;height:50px;overflow-y:scroll">`+
		`<div id="tall" style="font-size:16px">`+
		`<p id="l1" style="margin:0">tall line one</p>`+
		`<p id="l2" style="margin:0">tall line two</p>`+
		`<p id="l3" style="margin:0">tall line three</p>`+
		`<p id="l4" style="margin:0">tall line four</p>`+
		`<p id="l5" style="margin:0">tall line five</p>`+
		`<p id="l6" style="margin:0">tall line six</p>`+
		`</div></div></body></html>`)

	behaviorClipAssertDeactivation(t, res, "clip", "tall")
}

// TestBehaviorOverflowWrapBreakWordWrapsToken is overflow-wrap: break-word
// lets a long slash-separated token wrap inside a 100px block, so the text
// paints as at least two ops and no op crosses the content edge. Reference:
// Chrome 143.0.7499.40.
func TestBehaviorOverflowWrapBreakWordWrapsToken(t *testing.T) {
	t.Parallel()

	token := strings.Repeat("abcdefghij/", 12)
	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<p id="wrap" style="margin:0;width:100px;font-size:12pt;overflow-wrap:break-word">`+
		token+`</p></body></html>`)

	texts := make([]string, 0, len(res.Ops))

	var maxRight float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		texts = append(texts, paintOp.Text)

		if r := paintOp.X + paintOp.W; r > maxRight {
			maxRight = r
		}
	}

	if len(texts) < 2 {
		t.Fatalf("expected wrap into >=2 text ops, got %v", texts)
	}

	if joined := strings.Join(texts, ""); !strings.Contains(joined, "abcdefghij") {
		t.Fatalf("missing token text in ops: %q", joined)
	}

	if maxRight > pxToPt(100)+1 {
		t.Errorf("text crosses content edge: maxRight=%.4fpt (%.2fpx), want at most 100px",
			maxRight, maxRight/ptPerCSSPx)
	}
}

// TestBehaviorTextOverflowEllipsisTruncatesLine is text-overflow: a nowrap
// line wider than its 120px hidden block is truncated with a "..." suffix in
// the emitted text op instead of painting past the edge. Reference: Chrome
// 143.0.7499.40.
func TestBehaviorTextOverflowEllipsisTruncatesLine(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<p id="trunc" style="margin:0;width:120px;white-space:nowrap;overflow:hidden;`+
		`text-overflow:ellipsis;font-size:16px">ellipsis truncation line that is far too long</p>`+
		`</body></html>`)

	sawEllipsis := false

	var maxRight float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		if strings.HasSuffix(paintOp.Text, "...") {
			sawEllipsis = true
		}

		if r := paintOp.X + paintOp.W; r > maxRight {
			maxRight = r
		}
	}

	if !sawEllipsis {
		t.Errorf("no text op ends with ..., want ellipsis truncation")
	}

	if maxRight > pxToPt(120)+1 {
		t.Errorf("text crosses content edge: maxRight=%.4fpt (%.2fpx), want at most 120px",
			maxRight, maxRight/ptPerCSSPx)
	}
}

// TestBehaviorClipPathEllipseMasksImage is clip-path: ellipse() on an img
// zeroes the decoded alpha outside the shape while the center stays opaque.
// Reference: Chrome 143.0.7499.40.
func TestBehaviorClipPathEllipseMasksImage(t *testing.T) {
	t.Parallel()

	res := layoutHTMLWithImages(t,
		`<html><body><img src="x.png" style="clip-path:ellipse(25% 10% at 50% 50%)"></body></html>`,
		tinyPNG(40, 40), "x.png")

	imgs := opsOfKind(res, OpImage)
	if len(imgs) != 1 {
		t.Fatalf("img ops = %d, want 1", len(imgs))
	}

	img := decodeNRGBA(t, imgs[0].Image)

	if got := img.NRGBAAt(20, 20); got.A != 255 {
		t.Errorf("center alpha = %d, want 255", got.A)
	}

	if got := img.NRGBAAt(2, 2); got.A != 0 {
		t.Errorf("corner alpha = %d, want 0 (masked outside ellipse)", got.A)
	}
}

// TestBehaviorPointerEventsNoneKeepsPaint is pointer-events: the engine has
// no hit-testing, so pointer-events:none cannot change any emitted op. This
// pins the current no-op. Reference: Chrome 143.0.7499.40 keeps painting the
// link unchanged.
func TestBehaviorPointerEventsNoneKeepsPaint(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<a id="link" href="https://example.com" style="display:block;width:100px;height:20px;`+
		`background-color:#0000ff">x</a></body></html>`)
	none := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<a id="link" href="https://example.com" style="display:block;width:100px;height:20px;`+
		`background-color:#0000ff;pointer-events:none">x</a></body></html>`)

	behaviorInteractAssertSameOps(t, plain, none)
}

// TestBehaviorCursorPointerKeepsPaint is cursor: the engine paints no mouse
// pointer, so cursor:pointer cannot change any emitted op. This pins the
// current no-op. Reference: Chrome 143.0.7499.40 only changes the pointer
// glyph, not the page paint.
func TestBehaviorCursorPointerKeepsPaint(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="btn" style="width:100px;height:20px;background-color:#00ff00">x</div></body></html>`)
	pointed := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="btn" style="width:100px;height:20px;background-color:#00ff00;cursor:pointer">x</div></body></html>`)

	behaviorInteractAssertSameOps(t, plain, pointed)
}

// TestBehaviorUserSelectNoneKeepsPaint is user-select: the engine has no
// selection model, so user-select:none cannot change any emitted op. This
// pins the current no-op. Reference: Chrome 143.0.7499.40 keeps painting the
// same text.
func TestBehaviorUserSelectNoneKeepsPaint(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<p id="sel" style="margin:0;font-size:16px">selectable text here</p></body></html>`)
	locked := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<p id="sel" style="margin:0;font-size:16px;user-select:none">selectable text here</p></body></html>`)

	behaviorInteractAssertSameOps(t, plain, locked)
}

// TestBehaviorCaretColorKeepsPaint is caret-color: the engine paints no text
// caret, so caret-color cannot change any emitted op. This pins the current
// no-op. Reference: Chrome 143.0.7499.40 only recolors the caret glyph.
func TestBehaviorCaretColorKeepsPaint(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<p id="caret" style="margin:0;font-size:16px;color:#000000">caret text here</p></body></html>`)
	tinted := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<p id="caret" style="margin:0;font-size:16px;color:#000000;caret-color:#ff0000">caret text here</p></body></html>`)

	behaviorInteractAssertSameOps(t, plain, tinted)
}

// TestBehaviorResizeBothKeepsPaint is resize: the engine has no interactive
// resizing, so resize:both with overflow:auto keeps the used box and every
// emitted op. This pins the current no-op. Reference: Chrome 143.0.7499.40
// would add a drag handle without reflowing the box.
func TestBehaviorResizeBothKeepsPaint(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="rz" style="width:100px;height:50px;overflow:auto;`+
		`background-color:#ffff00">x</div></body></html>`)
	sized := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="rz" style="width:100px;height:50px;overflow:auto;resize:both;`+
		`background-color:#ffff00">x</div></body></html>`)

	behaviorInteractAssertSameOps(t, plain, sized)
}

// TestBehaviorAppearanceNoneKeepsRangeThumb is appearance: the engine paints
// the range thumb from the widget path regardless of appearance:none, so the
// declaration cannot change any emitted op. This pins the current no-op.
// Reference: Chrome 143.0.7499.40 restyles the control chrome instead.
func TestBehaviorAppearanceNoneKeepsRangeThumb(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<input id="rg" type="range" value="50" style="display:block;width:100px;height:12px"></body></html>`)
	none := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<input id="rg" type="range" value="50" style="display:block;width:100px;height:12px;appearance:none"></body></html>`)

	if len(plain.Ops) == 0 || len(none.Ops) == 0 {
		t.Fatalf("op counts = %d vs %d, want range widget ops in both", len(plain.Ops), len(none.Ops))
	}

	behaviorInteractAssertSameOps(t, plain, none)
}

// TestBehaviorAccentColorTintsMeterFill is accent-color: a meter with an
// authored accent-color paints its indicator fill with that color in the
// drawing list. Reference: Chrome 143.0.7499.40.
func TestBehaviorAccentColorTintsMeterFill(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<meter id="m" value="0.5" style="display:block;width:100px;height:12px;accent-color:#3366cc"></meter>`+
		`</body></html>`)

	saw := false

	for _, op := range res.Ops {
		if op.Kind != OpFillRect {
			continue
		}

		if approx(op.R, 0x33/255.0) && approx(op.G, 0x66/255.0) && approx(op.B, 0xcc/255.0) {
			saw = true

			break
		}
	}

	if !saw {
		t.Errorf("no meter fill with accent-color #3366cc")
	}
}
