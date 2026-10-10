package layout

import "testing"

// Chrome evidence for this file: Google Chrome 143.0.7499.40 headless, a
// probe document read through getComputedStyle. `border: solid red` and
// `border-top: solid red` both resolve border-top-width to 3px (the initial
// medium), `border: thin|medium|thick solid red` resolve to 1px/3px/5px, and
// `border: 0px solid red` resolves to 0px. `border-style: solid;
// border-color: red` with no width also resolves to 3px; that longhand-only
// path still paints nothing here (see the audit note on parseBorder's
// explicit-width tracking).

// TestBorderShorthandOmittedWidthUsesMedium: a widthless border shorthand
// takes the CSS initial width medium, 3 CSS px = 2.25pt, on every side. It
// used to default to 1pt.
func TestBorderShorthandOmittedWidthUsesMedium(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="b" style="border:solid red"></div>`+
		`</body></html>`)

	sty := boxByID(t, res, "b").style
	want := pxToPt(3)

	for idx, side := range []border{sty.BorderTop, sty.BorderRight, sty.BorderBottom, sty.BorderLeft} {
		if !near(side.Width, want) || !near(side.PaintWidth, want) {
			t.Errorf("side %d width %.4f paint %.4f, want %.4f", idx, side.Width, side.PaintWidth, want)
		}

		if side.Style != solidKeyword {
			t.Errorf("side %d style %q, want solid", idx, side.Style)
		}
	}

	if sty.BorderTop.Color != [3]float64{1, 0, 0} {
		t.Errorf("top color %v, want red", sty.BorderTop.Color)
	}
}

// TestBorderShorthandOmittedWidthSizesBox is the layout half: Chrome gives a
// width:100px content box with `border: solid black` a 106px border box
// (100 + 2 * 3px), not the 101.33px the old 1pt default produced.
func TestBorderShorthandOmittedWidthSizesBox(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="b" style="width:100px;border:solid black"></div>`+
		`</body></html>`)

	boxNode := boxByID(t, res, "b")
	if !near(boxNode.w, pxToPt(106)) {
		t.Errorf("border box = %.4fpt (%.2fpx), want 106px",
			boxNode.w, boxNode.w/ptPerCSSPx)
	}
}

// TestBorderSideShorthandsOmittedWidthUseMedium: border-top/right/bottom/left
// share parseBorder, so a widthless side shorthand also takes medium and
// leaves the other sides alone.
func TestBorderSideShorthandsOmittedWidthUseMedium(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="top" style="border-top:solid red"></div>`+
		`<div id="right" style="border-right:dashed blue"></div>`+
		`<div id="bottom" style="border-bottom:dotted green"></div>`+
		`<div id="left" style="border-left:solid black"></div>`+
		`</body></html>`)

	want := pxToPt(3)

	top := boxByID(t, res, "top").style
	checkBorderSide(t, "border-top", top.BorderTop, want, solidKeyword)

	if top.BorderBottom.Width != 0 {
		t.Errorf("border-top shorthand touched border-bottom: %.4f", top.BorderBottom.Width)
	}

	checkBorderSide(t, "border-right", boxByID(t, res, "right").style.BorderRight, want, borderStyleDashed)
	checkBorderSide(t, "border-bottom", boxByID(t, res, "bottom").style.BorderBottom, want, borderStyleDotted)
	checkBorderSide(t, "border-left", boxByID(t, res, "left").style.BorderLeft, want, solidKeyword)
}

// checkBorderSide asserts one side's layout width, paint width, and style.
// An empty wantStyle skips the style check.
func checkBorderSide(t *testing.T, name string, side border, want float64, wantStyle string) {
	t.Helper()

	if !near(side.Width, want) || !near(side.PaintWidth, want) {
		t.Errorf("%s width %.4f paint %.4f, want %.4f", name, side.Width, side.PaintWidth, want)
	}

	if wantStyle != "" && side.Style != wantStyle {
		t.Errorf("%s style %q, want %q", name, side.Style, wantStyle)
	}
}

// TestBorderShorthandKeywordWidthsMatchChrome: thin, medium, and thick are
// explicit widths in the shorthand, not omitted ones. The old parser ignored
// the keyword and fell back to its default; Chrome resolves them to
// 1px/3px/5px.
func TestBorderShorthandKeywordWidthsMatchChrome(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="thin" style="border:thin solid red"></div>`+
		`<div id="medium" style="border:medium solid red"></div>`+
		`<div id="thick" style="border:thick solid red"></div>`+
		`</body></html>`)

	testCases := []struct {
		id   string
		want float64
	}{
		{id: "thin", want: pxToPt(1)},
		{id: "medium", want: pxToPt(3)},
		{id: "thick", want: pxToPt(5)},
	}

	for _, testCase := range testCases {
		sty := boxByID(t, res, testCase.id).style
		if !near(sty.BorderTop.Width, testCase.want) || !near(sty.BorderTop.PaintWidth, testCase.want) {
			t.Errorf("%s width %.4f paint %.4f, want %.4f",
				testCase.id, sty.BorderTop.Width, sty.BorderTop.PaintWidth, testCase.want)
		}
	}
}

// TestBorderShorthandExplicitWidthsStayExplicit: an explicit width token is
// never replaced by the medium default, including an explicit zero. Chrome
// resolves `border: 0px solid red` to 0px and `border: 2px solid red` to 2px.
func TestBorderShorthandExplicitWidthsStayExplicit(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="zero" style="border:0px solid red"></div>`+
		`<div id="two" style="border:2px solid red"></div>`+
		`<div id="pt" style="border:1pt solid red"></div>`+
		`</body></html>`)

	zero := boxByID(t, res, "zero").style
	if zero.BorderTop.Width != 0 || zero.BorderTop.PaintWidth != 0 {
		t.Errorf("border:0px = width %.4f paint %.4f, want 0",
			zero.BorderTop.Width, zero.BorderTop.PaintWidth)
	}

	two := boxByID(t, res, "two").style
	if !near(two.BorderTop.Width, pxToPt(2)) || !near(two.BorderTop.PaintWidth, pxToPt(2)) {
		t.Errorf("border:2px = width %.4f paint %.4f, want %.4f",
			two.BorderTop.Width, two.BorderTop.PaintWidth, pxToPt(2))
	}

	pt := boxByID(t, res, "pt").style
	if !near(pt.BorderTop.Width, 1) || !near(pt.BorderTop.PaintWidth, 1) {
		t.Errorf("border:1pt = width %.4f paint %.4f, want 1",
			pt.BorderTop.Width, pt.BorderTop.PaintWidth)
	}
}

// TestBorderWidthLonghandKeywordsMatchChrome: the border-width longhands and
// the border-width shorthand already route through borderWidth, which maps
// thin/medium/thick to 1px/3px/5px. The longhand appliers only run when a
// value is present, so there is no omitted-width default to change there.
func TestBorderWidthLonghandKeywordsMatchChrome(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="thin" style="border-style:solid;border-width:thin"></div>`+
		`<div id="medium" style="border-style:solid;border-width:medium"></div>`+
		`<div id="thick" style="border-style:solid;border-width:thick"></div>`+
		`</body></html>`)

	testCases := []struct {
		id   string
		want float64
	}{
		{id: "thin", want: pxToPt(1)},
		{id: "medium", want: pxToPt(3)},
		{id: "thick", want: pxToPt(5)},
	}

	for _, testCase := range testCases {
		sty := boxByID(t, res, testCase.id).style
		if !near(sty.BorderTop.Width, testCase.want) || !near(sty.BorderTop.PaintWidth, testCase.want) {
			t.Errorf("%s width %.4f paint %.4f, want %.4f",
				testCase.id, sty.BorderTop.Width, sty.BorderTop.PaintWidth, testCase.want)
		}
	}
}

// TestOutlineOmittedWidthUsesMedium: outline has its own path and already
// applies the CSS initial medium width when a visible style is set with no
// width, matching Chrome's 3px for `outline: solid red`.
func TestOutlineOmittedWidthUsesMedium(t *testing.T) {
	t.Parallel()

	width, style := effectiveOutline(&ResolvedStyle{OutlineStyle: solidKeyword})
	if style != solidKeyword {
		t.Fatalf("outline style %q, want solid", style)
	}

	if !near(width, pxToPt(3)) {
		t.Errorf("outline width %.4f, want %.4f", width, pxToPt(3))
	}
}
