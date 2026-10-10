package layout

import "testing"

// Color-adjust pipeline wiring tests. Each property is asserted through used
// paint values on real documents (0..1 RGB floats on emitted ops), never
// through stored style strings or helper-only outputs. Reference browser for
// every case: Chrome 143.0.7499.40.

// wireDarkCanvasChannel is the used #121212 canvas channel for a dark
// color-scheme root; wireDarkTextChannel is the used #e8e8e8 default text.
const (
	wireDarkCanvasChannel = float64(darkSchemeCanvasByte) / 255
	wireDarkTextChannel   = float64(darkSchemeTextByte) / 255
)

// wireFindFill returns the first fill op whose RGB is near want.
func wireFindFill(t *testing.T, res *Result, want [3]float64) *Op {
	t.Helper()

	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Kind != OpFillRect {
			continue
		}

		if near(paintOp.R, want[0]) && near(paintOp.G, want[1]) && near(paintOp.B, want[2]) {
			return paintOp
		}
	}

	return nil
}

// wireFindText returns the first text or bullet op whose RGB is near want.
func wireFindText(t *testing.T, res *Result, want [3]float64) *Op {
	t.Helper()

	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Kind != OpText && paintOp.Kind != OpBullet {
			continue
		}

		if near(paintOp.R, want[0]) && near(paintOp.G, want[1]) && near(paintOp.B, want[2]) {
			return paintOp
		}
	}

	return nil
}

// TestBehaviorColorSchemeDarkWireCanvasAndText is color-scheme: a dark html
// root emits an opaque #121212 canvas fill and paints default (initial black)
// text as #e8e8e8, while a normal root keeps the transparent paper with black
// text. Only the html root's value is consumed. Reference: Chrome
// 143.0.7499.40 swaps defaults under dark.
func TestBehaviorColorSchemeDarkWireCanvasAndText(t *testing.T) {
	t.Parallel()

	doc := func(scheme string) string {
		return `<html style="margin:0;color-scheme:` + scheme + `"><body style="margin:0">` +
			`<p id="p" style="margin:0;font-size:12pt">hello world</p></body></html>`
	}

	dark := layoutHTML(t, doc("dark"))
	normal := layoutHTML(t, doc("normal"))

	darkCanvas := [3]float64{wireDarkCanvasChannel, wireDarkCanvasChannel, wireDarkCanvasChannel}
	if fill := wireFindFill(t, dark, darkCanvas); fill == nil {
		t.Fatal("dark color-scheme root emitted no #121212 canvas fill")
	} else if !near(fill.Alpha, 1) {
		t.Errorf("dark canvas alpha = %v, want opaque 1", fill.Alpha)
	}

	darkText := [3]float64{wireDarkTextChannel, wireDarkTextChannel, wireDarkTextChannel}
	if text := wireFindText(t, dark, darkText); text == nil {
		t.Fatal("dark color-scheme root painted no #e8e8e8 default text")
	}

	if fill := wireFindFill(t, normal, darkCanvas); fill != nil {
		t.Errorf("normal color-scheme emitted dark canvas fill %+v, want transparent paper", *fill)
	}

	black := [3]float64{}
	if text := wireFindText(t, normal, black); text == nil {
		t.Fatal("normal color-scheme painted no black default text")
	}
}

// TestBehaviorColorSchemeDarkWireAuthorBGWins is color-scheme with an author
// background: the html root's own background-color wins over the dark canvas,
// so a white root paints a white fill and no #121212 fill. Reference: Chrome
// 143.0.7499.40.
func TestBehaviorColorSchemeDarkWireAuthorBGWins(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0;color-scheme:dark;background-color:#ffffff">`+
		`<body style="margin:0"><p id="p" style="margin:0;font-size:12pt;color:#ff0000">hello</p></body></html>`)

	white := [3]float64{1, 1, 1}
	if fill := wireFindFill(t, res, white); fill == nil {
		t.Fatal("dark root with author white background painted no white fill")
	}

	darkCanvas := [3]float64{wireDarkCanvasChannel, wireDarkCanvasChannel, wireDarkCanvasChannel}
	if fill := wireFindFill(t, res, darkCanvas); fill != nil {
		t.Errorf("author background should win over dark canvas, found %+v", *fill)
	}

	red := [3]float64{1, 0, 0}
	if text := wireFindText(t, res, red); text == nil {
		t.Fatal("author red text under dark scheme painted no red text op")
	}
}

// TestBehaviorForcedColorAdjustWirePreservesAuthor is forced-color-adjust:
// print has no forced-colors mode, so both auto and none preserve author
// colors through the forced-mode mapping. The test asserts the used blue
// background fill and yellow text on real documents for both values.
// Reference: Chrome 143.0.7499.40 keeps author colors under none when forced
// mode is on; with the mode off both values paint the author colors.
func TestBehaviorForcedColorAdjustWirePreservesAuthor(t *testing.T) {
	t.Parallel()

	doc := func(adjust string) string {
		return `<html><body style="margin:0">` +
			`<div id="c" style="width:50px;height:20px;background-color:#0000ff;` +
			`forced-color-adjust:` + adjust + `"><p id="p" style="margin:0;color:#ffff00">hi</p></div>` +
			`</body></html>`
	}

	blue := [3]float64{0, 0, 1}
	yellow := [3]float64{1, 1, 0}

	for _, adjust := range []string{"auto", "none"} {
		res := layoutHTML(t, doc(adjust))

		if fill := wireFindFill(t, res, blue); fill == nil {
			t.Fatalf("forced-color-adjust:%s painted no blue author fill", adjust)
		}

		if text := wireFindText(t, res, yellow); text == nil {
			t.Fatalf("forced-color-adjust:%s painted no yellow author text", adjust)
		}
	}
}

// TestBehaviorDynamicRangeLimitWireClamps is dynamic-range-limit: standard
// folds out-of-range computed channels into sRGB [0, 1] while no-limit and
// high preserve headroom. The CSS parser only produces 0..1 channels, so an
// in-gamut sRGB red fill paints red under both standard and no-limit on a
// real document; the clamp call site is what the test exercises. Reference:
// Chrome 143.0.7499.40 clamps HDR headroom under standard.
func TestBehaviorDynamicRangeLimitWireClamps(t *testing.T) {
	t.Parallel()

	doc := func(limit string) string {
		return `<html><body style="margin:0">` +
			`<div id="c" style="width:50px;height:20px;background-color:#ff0000;` +
			`dynamic-range-limit:` + limit + `"></div></body></html>`
	}

	red := [3]float64{1, 0, 0}

	for _, limit := range []string{"no-limit", "standard"} {
		res := layoutHTML(t, doc(limit))

		if fill := wireFindFill(t, res, red); fill == nil {
			t.Fatalf("dynamic-range-limit:%s painted no sRGB red fill", limit)
		}
	}
}
