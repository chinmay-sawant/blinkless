package layout

import "testing"

// TestBorderWidthLengthUnitsUsePoints is CAT-07 cluster 1: a border length in
// CSS pixels converts to points (1px = 0.75pt) in both the layout width and
// the paint width, so layout and paint agree. Before this fix the shorthand
// and the border-width longhand kept the raw parsed number, so `border: 1px`
// laid out as 1pt (1.33px) while painting 0.75pt. Reference: Chrome
// 143.0.7499.40 on test/chrome/cases/case-17-wpt-flex-basis-011.html, where
// the same fixture with `border: 0.75pt` matches the browser exactly.
func TestBorderWidthLengthUnitsUsePoints(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		style string
		want  float64 // points
	}{
		{name: "shorthand px", style: "border:1px solid black", want: pxToPt(1)},
		{name: "longhand px", style: "border-style:solid;border-width:1px", want: pxToPt(1)},
		{name: "shorthand pt", style: "border:1pt solid black", want: 1},
		{name: "longhand pt", style: "border-style:solid;border-width:2pt", want: 2},
		{name: "shorthand em", style: "border:0.1em solid black", want: 1.2}, // 12pt font
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html><body style="margin:0">`+
				`<div id="b" style="`+testCase.style+`"></div>`+
				`</body></html>`)

			boxNode := boxByID(t, res, "b")
			if !near(boxNode.style.BorderTop.Width, testCase.want) {
				t.Errorf("layout border width = %.4fpt, want %.4fpt",
					boxNode.style.BorderTop.Width, testCase.want)
			}

			if !near(boxNode.style.BorderTop.PaintWidth, testCase.want) {
				t.Errorf("paint border width = %.4fpt, want %.4fpt",
					boxNode.style.BorderTop.PaintWidth, testCase.want)
			}
		})
	}
}

// TestBorderPxWidthSizesBorderBox is the layout-level half of CAT-07 cluster 1:
// a content-box block with width:100px and border:1px reports a 102px border
// box (100 + 2 * 0.75pt), not the 102.67px the raw-number storage produced.
func TestBorderPxWidthSizesBorderBox(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="b" style="width:100px;border:1px solid black"></div>`+
		`</body></html>`)

	boxNode := boxByID(t, res, "b")
	if !near(boxNode.w, pxToPt(102)) {
		t.Errorf("border box width = %.4fpt (%.2fpx), want 102px",
			boxNode.w, boxNode.w/ptPerCSSPx)
	}
}

// TestImageMaxConstraintsKeepDefiniteAxis is CAT-07 cluster 3: max-height
// clamps only the height when the width is definite (a width attribute or a
// CSS width), and max-width clamps only the width when the height is
// definite. Only an auto axis follows the intrinsic ratio. Reference: Chrome
// 143.0.7499.40 on test/chrome/cases/case-28-wpt-flex-minimum-width-aspect.html
// (200x200 image with max-height:100pt is 200x133.33) and the p-maxwh2 /
// p-svg probes: 200x200 with max-width:100pt is 133.33x200, inline <svg> with
// width/height attributes behaves the same.
func TestImageMaxConstraintsKeepDefiniteAxis(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		attrs string
		style string
		wantW float64 // CSS px
		wantH float64
	}{
		{
			name: "max-height keeps attr width", attrs: `width="200" height="200"`,
			style: "max-height:100pt", wantW: 200, wantH: 133.33,
		},
		{
			name: "max-width keeps attr height", attrs: `width="200" height="200"`,
			style: "max-width:100pt", wantW: 133.33, wantH: 200,
		},
		{
			name: "max-height keeps css width", attrs: `width="200" height="200"`,
			style: "width:200px;height:200px;max-height:100pt", wantW: 200, wantH: 133.33,
		},
		{
			name: "max-width keeps css height", attrs: `width="200" height="200"`,
			style: "width:200px;height:200px;max-width:100pt", wantW: 133.33, wantH: 200,
		},
		{
			name: "auto width follows ratio", attrs: `height="200"`,
			style: "max-height:100pt", wantW: 133.33, wantH: 133.33,
		},
		{
			name: "auto height follows ratio", attrs: `width="200"`,
			style: "max-width:100pt", wantW: 133.33, wantH: 133.33,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTMLWithImages(t, `<html><body style="margin:0">`+
				`<img id="i" src="x.png" `+testCase.attrs+` style="display:block;`+testCase.style+`">`+
				`</body></html>`, tinyPNG(200, 200), "x.png")

			imgBox := boxByID(t, res, "i")
			if !near(imgBox.w, pxToPt(testCase.wantW)) || !near(imgBox.height, pxToPt(testCase.wantH)) {
				t.Errorf("image = %.4f x %.4f pt (%.2f x %.2f px), want %.2f x %.2f px",
					imgBox.w, imgBox.height, imgBox.w/ptPerCSSPx, imgBox.height/ptPerCSSPx,
					testCase.wantW, testCase.wantH)
			}
		})
	}
}

// TestInlineSVGMaxConstraintsKeepDefiniteAxis is the SVG half of CAT-07
// cluster 3: inline <svg> width/height attributes are definite, so
// max-height:100pt on a 200x200 svg keeps the 200 width (Chrome 143 probe
// p-svg: 200x133.33).
func TestInlineSVGMaxConstraintsKeepDefiniteAxis(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<svg id="s" width="200" height="200" viewBox="0 0 200 200" `+
		`style="display:block;max-height:100pt" xmlns="http://www.w3.org/2000/svg">`+
		`<rect width="200" height="200" fill="green"/></svg>`+
		`</body></html>`)

	svgBox := boxByID(t, res, "s")
	if !near(svgBox.w, pxToPt(200)) || !near(svgBox.height, pxToPt(133.33)) {
		t.Errorf("svg = %.4f x %.4f pt (%.2f x %.2f px), want 200 x 133.33 px",
			svgBox.w, svgBox.height, svgBox.w/ptPerCSSPx, svgBox.height/ptPerCSSPx)
	}
}

// TestDefiniteHeightOverflowsContent is CAT-07 cluster 4: a definite height
// (length or resolvable percentage) sets the used height and taller content
// overflows visibly instead of growing the box. Before this fix the height
// acted as a floor. Reference: Chrome 143.0.7499.40 on
// test/chrome/cases/case-34-wpt-flex-container-max-content.html (inline-block
// ink with height:10pt is 13.33px tall) and
// case-40-wpt-flex-container-min-content.html (height:0 wrapper stays 0x0).
func TestDefiniteHeightOverflowsContent(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="wrap" style="height:0"><div id="tall" style="height:50pt"></div></div>`+
		`<div id="block" style="height:50pt"><div style="height:150pt"></div></div>`+
		`<p id="line" style="margin:0"><span id="ink" style="display:inline-block;height:10pt">X X</span></p>`+
		`</body></html>`)

	wrap := boxByID(t, res, "wrap")
	if !near(wrap.height, 0) {
		t.Errorf("height:0 wrapper = %.4fpt, want 0", wrap.height)
	}

	block := boxByID(t, res, "block")
	if !near(block.height, 50) {
		t.Errorf("height:50pt block = %.4fpt (%.2fpx), want 50pt (66.67px)",
			block.height, block.height/ptPerCSSPx)
	}

	ink := boxByID(t, res, "ink")
	if !near(ink.height, 10) {
		t.Errorf("height:10pt inline-block = %.4fpt (%.2fpx), want 10pt (13.33px)",
			ink.height, ink.height/ptPerCSSPx)
	}
}

// TestOrthogonalAutoWidthShrinkWraps is CAT-07 cluster 6: a block whose
// writing mode is orthogonal to its containing block resolves its auto
// inline size as fit-content, while a same-mode auto-width block still
// stretches. Reference: Chrome 143.0.7499.40 on
// case-30-wpt-auto-margins-column.html: the vertical-lr wrapper around the
// 300pt flex case is 402px (400 + 2 x 1px border), not the 1024px viewport.
func TestOrthogonalAutoWidthShrinkWraps(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="ortho" style="writing-mode:vertical-lr">`+
		`<div style="width:300pt;border:1px solid black"></div></div>`+
		`<div id="flat"><div style="width:300pt;border:1px solid black"></div></div>`+
		`</body></html>`)

	ortho := boxByID(t, res, "ortho")
	if !near(ortho.w, pxToPt(402)) {
		t.Errorf("orthogonal wrapper width = %.4fpt (%.2fpx), want 402px",
			ortho.w, ortho.w/ptPerCSSPx)
	}

	flat := boxByID(t, res, "flat")
	if !near(flat.w, testViewport) {
		t.Errorf("horizontal wrapper width = %.4fpt (%.2fpx), want %.0fpx",
			flat.w, flat.w/ptPerCSSPx, testViewport)
	}
}

// TestExplicitLineHeightAllowsNegativeHalfLeading is CAT-07 cluster 6:
// line-height:1 on a 9pt font makes a 9pt (12px) line box even though the
// face's raw ascent+descent is larger; the glyphs overflow. Before this fix
// the half-leading was clamped at zero and the box kept the raw metrics.
// Reference: Chrome 143.0.7499.40 p-lineheight probe (12px) and case-30's
// 20px line-height:1 items.
func TestExplicitLineHeightAllowsNegativeHalfLeading(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<p id="tight" style="margin:0;font-size:9pt;line-height:1">X</p>`+
		`<p id="explicit" style="margin:0;font-size:9pt;line-height:12pt">X</p>`+
		`</body></html>`)

	tight := boxByID(t, res, "tight")
	if !near(tight.height, 9) {
		t.Errorf("line-height:1 paragraph = %.4fpt (%.2fpx), want 9pt (12px)",
			tight.height, tight.height/ptPerCSSPx)
	}

	explicit := boxByID(t, res, "explicit")
	if !near(explicit.height, 12) {
		t.Errorf("line-height:12pt paragraph = %.4fpt, want 12pt", explicit.height)
	}
}

// TestAtomicInlineLineMetrics is CAT-07 cluster 5: atomic inlines put their
// margin box on the line, every line carries the block strut, an empty
// inline-block is 0x0 and sits on the baseline, and an inline-block whose
// last child uses a vertical writing mode falls back to its bottom margin
// edge. Reference: Chrome 143.0.7499.40 on /tmp/opencode/p-inlineblock.html
// (lb height 72.33 = 53.33 + 16 margin + 3 strut; inner1 0x0 at the
// baseline) and case-13-legacy-flex-flow-auto-margins.html.
func TestAtomicInlineLineMetrics(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0;font:12px sans-serif">`+
		`<div id="outer1" style="width:100px"><div id="inner1" style="display:inline-block"></div></div>`+
		`<div id="outer2" style="width:100px">`+
		`<span id="inner2" style="display:inline-block;width:30pt;height:10pt"></span></div>`+
		`<div id="lb" style="width:300pt">`+
		`<div id="ib" style="display:inline-block;width:50pt;height:40pt;margin-bottom:12pt"></div></div>`+
		`<div id="after" style="margin-top:12pt;height:10pt"></div>`+
		`</body></html>`)

	outer1 := boxByID(t, res, "outer1")
	inner1 := boxByID(t, res, "inner1")
	outer2 := boxByID(t, res, "outer2")
	inner2 := boxByID(t, res, "inner2")
	lineBox := boxByID(t, res, "lb")
	innerBlock := boxByID(t, res, "ib")
	after := boxByID(t, res, "after")

	// The empty inline-block is 0x0 (no 1pt floor) and rests on the strut
	// baseline, so its top is inside the line box, not at the line top.
	if !near(inner1.w, 0) || !near(inner1.height, 0) {
		t.Errorf("empty inline-block = %.4f x %.4f, want 0 x 0", inner1.w, inner1.height)
	}

	if inner1.y <= outer1.y || inner1.y >= outer1.y+outer1.height {
		t.Errorf("empty inline-block y = %.4f, want inside line [%.4f, %.4f]",
			inner1.y, outer1.y, outer1.y+outer1.height)
	}

	// The strut descent extends a line whose tallest box is a definite-height
	// inline-block by up to one line of descent.
	if outer2.height <= inner2.height || outer2.height > inner2.height+pxToPt(4) {
		t.Errorf("outer2 height = %.4f, want %.4f < h <= %.4f",
			outer2.height, inner2.height, inner2.height+pxToPt(4))
	}

	// The inline-block margin box is on the line: lineBox = innerBlock +
	// margin-bottom + strut descent, and the next block's margin starts below
	// that.
	if lineBox.height <= innerBlock.height+pxToPt(16) ||
		lineBox.height > innerBlock.height+pxToPt(16)+pxToPt(4) {
		t.Errorf("line box height = %.4f, want inner block %.4f + 16px margin + strut descent",
			lineBox.height, innerBlock.height)
	}

	if !near(after.y-lineBox.y-lineBox.height, pxToPt(16)) {
		t.Errorf("after gap = %.4f, want 16px margin-top", after.y-lineBox.y-lineBox.height)
	}
}

// TestInlineBlockBaselineVerticalChild is CAT-07 cluster 5: a non-empty
// inline-block whose last child is horizontal keeps the margin box on the
// line, while a vertical-writing-mode last child has no horizontal baseline
// and falls back to the bottom margin edge, adding the strut descent.
// Reference: Chrome 143.0.7499.40 probes p-gap13c and p-vwbase.
func TestInlineBlockBaselineVerticalChild(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0;font:12px sans-serif">`+
		`<div id="ch" style="display:inline-block;width:100px;margin-bottom:12pt">`+
		`<div id="bh" style="height:50pt"></div></div>`+
		`<div id="th" style="margin-top:12pt;height:10pt"></div>`+
		`<div id="cv" style="display:inline-block;width:100px;margin-bottom:12pt">`+
		`<div id="bv" style="height:50pt;writing-mode:vertical-lr"></div></div>`+
		`<div id="tv" style="margin-top:12pt;height:10pt"></div>`+
		`</body></html>`)

	horizontal := boxByID(t, res, "ch")
	afterHorizontal := boxByID(t, res, "th")
	vertical := boxByID(t, res, "cv")
	afterVertical := boxByID(t, res, "tv")

	// Horizontal child: line = margin box (50pt + 12pt), so the next block
	// starts 50 + 12 + 12 = 74pt below.
	if !near(afterHorizontal.y-horizontal.y, 74) {
		t.Errorf("horizontal gap = %.4f, want 74pt", afterHorizontal.y-horizontal.y)
	}

	// Vertical child: the strut descent adds below the margin box, so the
	// gap is strictly larger but still under one extra line.
	if afterVertical.y-vertical.y <= 74 || afterVertical.y-vertical.y > 74+pxToPt(4) {
		t.Errorf("vertical gap = %.4f, want 74pt < gap <= 74pt + strut descent", afterVertical.y-vertical.y)
	}
}

// TestRootMarginCollapsesThroughNonzeroBodyMargin is CAT-07 cluster 2: the
// body's nonzero top margin collapses with its first in-flow block child
// chain to their maximum, so the body border edge and the whole chain share
// the collapsed position. Before this fix the engine stacked the margins
// (body at 8px, child at 24px where Chrome has both at 16px). Reference:
// Chrome 143.0.7499.40 on test/chrome/cases/case-18-wpt-rtl-flow-reverse.html
// and the probes under /tmp/opencode (p-body-margin, p-body-margin-chain).
func TestRootMarginCollapsesThroughNonzeroBodyMargin(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		body     string
		child    string
		inner    string // optional grandchild
		wantY    float64
		wantHTML float64
	}{
		{
			name:     "child margin wins",
			body:     "margin:8px",
			child:    "margin-top:16px;height:75pt",
			wantY:    16,
			wantHTML: 124,
		},
		{
			name:     "body margin wins",
			body:     "margin:22.5pt 0 0",
			child:    "margin-top:15pt;height:75pt",
			wantY:    30,
			wantHTML: 130,
		},
		{
			name:     "chain collapses to deepest",
			body:     "margin:8px",
			child:    "margin-top:10px",
			inner:    "margin-top:20px;height:75pt",
			wantY:    20,
			wantHTML: 128,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assertRootMarginCollapseCase(t, testCase.body, testCase.child, testCase.inner,
				testCase.wantY, testCase.wantHTML)
		})
	}
}

// assertRootMarginCollapseCase lays out one root margin collapse fixture and
// checks that the body, its first child, an optional grandchild, and the html
// height all land on the collapsed top margin.
func assertRootMarginCollapseCase(
	t *testing.T, body, child, inner string, wantY, wantHTML float64,
) {
	t.Helper()

	innerHTML := ""
	if inner != "" {
		innerHTML = `<div id="inner" style="` + inner + `"></div>`
	}

	res := layoutHTML(t, `<html style="margin:0"><body style="`+body+`">`+
		`<div id="child" style="`+child+`">`+innerHTML+`</div>`+
		`</body></html>`)

	bodyBox := findBox(t, res, "body")
	childBox := boxByID(t, res, "child")

	if !near(bodyBox.y, pxToPt(wantY)) {
		t.Errorf("body y = %.4fpt (%.2fpx), want %.2fpx",
			bodyBox.y, bodyBox.y/ptPerCSSPx, wantY)
	}

	if !near(childBox.y, pxToPt(wantY)) {
		t.Errorf("child y = %.4fpt (%.2fpx), want %.2fpx",
			childBox.y, childBox.y/ptPerCSSPx, wantY)
	}

	if inner != "" {
		innerBox := boxByID(t, res, "inner")
		if !near(innerBox.y, pxToPt(wantY)) {
			t.Errorf("grandchild y = %.4fpt (%.2fpx), want %.2fpx",
				innerBox.y, innerBox.y/ptPerCSSPx, wantY)
		}
	}

	htmlBox := findBox(t, res, "html")
	if !near(htmlBox.height, pxToPt(wantHTML)) {
		t.Errorf("html height = %.4fpt (%.2fpx), want %.2fpx",
			htmlBox.height, htmlBox.height/ptPerCSSPx, wantHTML)
	}
}
