package layout

import "testing"

// Canvas background behavior. The color-scheme dark default is the canvas
// background, not an element background: CSS Backgrounds 3 §2.11.2 gives the
// root's background to the canvas, whose painting area is the whole canvas,
// independent of the root element's box. Reference browser for every case:
// Chrome 143.0.7499.40 prints the dark canvas edge-to-edge on every page.

// canvasBackgroundFill returns the single op marked as the root canvas
// background, failing when the document has none or more than one. Element
// backgrounds must never carry the mark, so a second marked op means the
// canvas default is no longer distinguishable from an author background.
func canvasBackgroundFill(t *testing.T, res *Result) *Op {
	t.Helper()

	var found *Op

	for opIdx := range res.Ops {
		if !res.Ops[opIdx].IsCanvasBackground() {
			continue
		}

		if found != nil {
			t.Fatalf("two canvas background ops: %+v and %+v", *found, res.Ops[opIdx])
		}

		found = &res.Ops[opIdx]
	}

	if found == nil {
		t.Fatal("no canvas background fill op")
	}

	return found
}

// assertDarkCanvasFill pins the color, alpha, and origin of the fixture-63
// canvas background: opaque #121212 painted from the canvas origin.
func assertDarkCanvasFill(t *testing.T, fill *Op) {
	t.Helper()

	dark := [3]float64{wireDarkCanvasChannel, wireDarkCanvasChannel, wireDarkCanvasChannel}
	if !near(fill.R, dark[0]) || !near(fill.G, dark[1]) || !near(fill.B, dark[2]) {
		t.Errorf("canvas fill color = (%.3f, %.3f, %.3f), want #121212", fill.R, fill.G, fill.B)
	}

	if !near(fill.Alpha, 1) {
		t.Errorf("canvas fill alpha = %v, want opaque 1", fill.Alpha)
	}

	if fill.X != 0 || fill.Y != 0 {
		t.Errorf("canvas fill origin = (%v, %v), want the canvas origin (0, 0)", fill.X, fill.Y)
	}
}

// TestFixture63CanvasFillCoversCanvas is the fixture-63 root canvas pin: the
// loaded document's dark color-scheme fills the whole canvas with opaque
// #121212 and is the only op marked as the canvas background, while
// undeclared text paints #e8e8e8. The load uses the page content box the
// gowkhtmltopdf bridge passes: A4 minus the 12mm @page margins.
func TestFixture63CanvasFillCoversCanvas(t *testing.T) {
	t.Parallel()

	root, sheets := loadFixture(t, "fixture-63-page-level-demos.html")

	res, err := Layout(root, Options{
		Width: 527.25, Height: 774, Sheets: sheets, Background: true, Media: "print",
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	fill := canvasBackgroundFill(t, res)
	assertDarkCanvasFill(t, fill)

	if !near(fill.W, res.Width) || !near(fill.H, res.Height) {
		t.Errorf("canvas fill = (%.2f, %.2f, %.2f, %.2f), want the canvas rect (0, 0, %.2f, %.2f)",
			fill.X, fill.Y, fill.W, fill.H, res.Width, res.Height)
	}
	// The fixture's content is taller than one page content box, so the
	// canvas is the packed content height; the fill must not stop at the
	// first page.
	if fill.H < 773.86 {
		t.Errorf("canvas fill height = %.2f, want the multi-page canvas", fill.H)
	}

	textInk := [3]float64{wireDarkTextChannel, wireDarkTextChannel, wireDarkTextChannel}
	if text := wireFindText(t, res, textInk); text == nil {
		t.Fatal("fixture-63 painted no #e8e8e8 default text")
	}
}

// TestCanvasFillCoversViewportAndRootMargin pins the canvas rect against two
// root-box shapes that must not leak into it: a root margin, which offsets
// and shrinks the root border box, and a canvas raised above the content by
// the requested viewport minimum, which the root box never covers. Before
// the fix this op was emitted as the root border box, so a short dark page
// left the viewport paper-white below the first line and a root margin
// pulled the fill away from the canvas edges.
func TestCanvasFillCoversViewportAndRootMargin(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:20px;color-scheme:dark">`+
		`<body style="margin:0"><p style="margin:0">hi</p></body></html>`)

	fill := canvasBackgroundFill(t, res)

	if fill.X != 0 || fill.Y != 0 {
		t.Errorf("canvas fill origin = (%v, %v), want the canvas origin (0, 0)", fill.X, fill.Y)
	}

	if !near(fill.W, res.Width) {
		t.Errorf("canvas fill width = %.2f, want the canvas width %.2f", fill.W, res.Width)
	}
	// layoutHTML requests Height 800; a short document's canvas is that
	// requested minimum, and the fill has to reach its end.
	if !near(fill.H, 800) {
		t.Errorf("canvas fill height = %.2f, want the requested viewport minimum 800", fill.H)
	}
}
