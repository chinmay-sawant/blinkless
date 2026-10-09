package layout

import "testing"

// ptPerCSSPx converts layout points to CSS pixels in error messages (1px = 0.75pt).
const ptPerCSSPx = 0.75

// boxByID returns the first box whose element carries the given id.
func boxByID(t *testing.T, res *Result, elementID string) *box {
	t.Helper()

	var found *box

	var walk func(*box)

	walk = func(boxNode *box) {
		if found != nil || boxNode == nil {
			return
		}

		if boxNode.node != nil && boxNode.node.Attribute("id") == elementID {
			found = boxNode

			return
		}

		for _, child := range boxNode.children {
			walk(child)
		}
	}
	walk(res.root)

	if found == nil {
		t.Fatalf("no box with id %q", elementID)
	}

	return found
}

// TestFlexDistributedJustifyKeepsGap is CSS-REVIEW-02-C1: space-evenly and
// space-around must subtract the fixed gaps from the free space and keep the
// gap between items. Reference: Chrome 143.0.7499.40, 400px row, three 50px
// items, gap 20px.
func TestFlexDistributedJustifyKeepsGap(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		justify    string
		x1, x2, x3 float64 // CSS pixels
	}{
		{justify: "space-evenly", x1: 52.5, x2: 175, x3: 297.5},
		{justify: "space-around", x1: 35, x2: 175, x3: 315},
		{justify: "space-between", x1: 0, x2: 175, x3: 350},
	}

	for _, testCase := range testCases {
		t.Run(testCase.justify, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html><body style="margin:0">`+
				`<div id="row" style="display:flex;width:300pt;gap:15pt;justify-content:`+testCase.justify+`">`+
				`<div id="a" style="width:37.5pt;height:10pt"></div>`+
				`<div id="b" style="width:37.5pt;height:10pt"></div>`+
				`<div id="c" style="width:37.5pt;height:10pt"></div>`+
				`</div></body></html>`)

			for elementID, wantPx := range map[string]float64{
				"a": testCase.x1, "b": testCase.x2, "c": testCase.x3,
			} {
				itemBox := boxByID(t, res, elementID)
				if !near(itemBox.x, pxToPt(wantPx)) {
					t.Errorf("%s.x = %.4fpt (%.2fpx), want %.2fpx",
						elementID, itemBox.x, itemBox.x/ptPerCSSPx, wantPx)
				}
			}
		})
	}
}

// TestGridFrTracksShareFreeSpace is CSS-REVIEW-02-C2: 1fr and minmax(...,
// 1fr) tracks divide the free space equally, and a minmax floor larger than
// the equal share keeps its size while the rest share the remainder.
// Reference: Chrome 143.0.7499.40, 300px grid.
func TestGridFrTracksShareFreeSpace(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		template string
		x        [3]float64
		w        [3]float64
	}{
		{
			name:     "fr",
			template: "1fr 1fr 1fr",
			x:        [3]float64{0, 100, 200},
			w:        [3]float64{100, 100, 100},
		},
		{
			name:     "minmax",
			template: "minmax(50pt,1fr) minmax(60pt,1fr) minmax(70pt,1fr)",
			x:        [3]float64{0, 100, 200},
			w:        [3]float64{100, 100, 100},
		},
		{
			name:     "minmax-floor",
			template: "minmax(150pt,1fr) 1fr 1fr",
			x:        [3]float64{0, 200, 250},
			w:        [3]float64{200, 50, 50},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html><body style="margin:0">`+
				`<div id="g" style="display:grid;width:225pt;grid-template-columns:`+testCase.template+`">`+
				`<div id="g1">several words of text</div><div id="g2">x</div><div id="g3">x</div>`+
				`</div></body></html>`)

			for idx, elementID := range []string{"g1", "g2", "g3"} {
				itemBox := boxByID(t, res, elementID)
				if !near(itemBox.x, pxToPt(testCase.x[idx])) || !near(itemBox.w, pxToPt(testCase.w[idx])) {
					t.Errorf("%s = x %.4fpt w %.4fpt, want x %.2fpx w %.2fpx",
						elementID, itemBox.x, itemBox.w, testCase.x[idx], testCase.w[idx])
				}
			}
		})
	}
}

// TestNormalLineHeightUsesFaceMetrics is CSS-REVIEW-02-C3: line-height:
// normal comes from the face's hhea ascent, descent, and line gap rounded to
// whole CSS pixels, not the 1.2em constant. Reference: Chrome 143.0.7499.40
// with the bundled Liberation Sans metrics.
func TestNormalLineHeightUsesFaceMetrics(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		fontSize string
		wantPx   float64
	}{
		{fontSize: "16px", wantPx: 18},
		{fontSize: "12px", wantPx: 14},
		{fontSize: "32px", wantPx: 37},
		{fontSize: "12pt", wantPx: 18},
	}

	for _, testCase := range testCases {
		t.Run(testCase.fontSize, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html><body style="margin:0">`+
				`<p id="t" style="font-size:`+testCase.fontSize+`">Ag</p>`+
				`</body></html>`)

			textBox := boxByID(t, res, "t")
			if !near(textBox.height, pxToPt(testCase.wantPx)) {
				t.Errorf("line height = %.4fpt (%.2fpx), want %.2fpx",
					textBox.height, textBox.height/ptPerCSSPx, testCase.wantPx)
			}
		})
	}
}

// TestRootMarginFirstChildTopOffset is CSS-REVIEW-02-C4: the collapsed
// first-child top margin offsets the body box, matching Chrome, and the
// in-flow child keeps its position.
func TestRootMarginFirstChildTopOffset(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="one" style="margin:15pt 0 22.5pt;height:75pt"></div>`+
		`</body></html>`)

	htmlBox := findBox(t, res, "html")
	bodyBox := findBox(t, res, "body")
	one := boxByID(t, res, "one")

	// Chrome: html 0..150px, body 20..120px, child 20..120px.
	if !near(htmlBox.y, 0) || !near(htmlBox.height, pxToPt(150)) {
		t.Errorf("html = y %.4fpt h %.4fpt (%.2fpx), want h 150px",
			htmlBox.y, htmlBox.height, htmlBox.height/ptPerCSSPx)
	}

	if !near(bodyBox.y, pxToPt(20)) || !near(bodyBox.height, pxToPt(100)) {
		t.Errorf("body = y %.4fpt h %.4fpt, want y 20px h 100px", bodyBox.y, bodyBox.height)
	}

	if !near(one.y, pxToPt(20)) || !near(one.height, pxToPt(100)) {
		t.Errorf("child = y %.4fpt h %.4fpt, want y 20px h 100px", one.y, one.height)
	}
}

// TestRootMarginLastChildBottomExtendsHTML is CSS-REVIEW-02-C4: the collapsed
// last-child bottom margin extends the html height, matching Chrome.
func TestRootMarginLastChildBottomExtendsHTML(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="one" style="margin:0 0 22.5pt;height:75pt"></div>`+
		`</body></html>`)

	htmlBox := findBox(t, res, "html")
	bodyBox := findBox(t, res, "body")

	// Chrome: html 0..130px, body 0..100px.
	if !near(htmlBox.height, pxToPt(130)) {
		t.Errorf("html height = %.4fpt (%.2fpx), want 130px",
			htmlBox.height, htmlBox.height/ptPerCSSPx)
	}

	if !near(bodyBox.y, 0) || !near(bodyBox.height, pxToPt(100)) {
		t.Errorf("body = y %.4fpt h %.4fpt, want y 0 h 100px", bodyBox.y, bodyBox.height)
	}
}

// TestRootMarginNonzeroBodyHTMLHeight is CSS-REVIEW-02-C4: with a nonzero body
// top margin the engine keeps its stacked body box (out of the row's scope),
// but the html height still reports Chrome's collapsed geometry.
func TestRootMarginNonzeroBodyHTMLHeight(t *testing.T) {
	t.Parallel()

	// Chrome: body 30..130px, html 130px.
	res := layoutHTML(t, `<html style="margin:0"><body style="margin:22.5pt 0 0">`+
		`<div id="one" style="margin:15pt 0 0;height:75pt"></div>`+
		`</body></html>`)

	htmlBox := findBox(t, res, "html")
	if !near(htmlBox.height, pxToPt(130)) {
		t.Errorf("html height = %.4fpt (%.2fpx), want 130px",
			htmlBox.height, htmlBox.height/ptPerCSSPx)
	}
}

// TestFlexItemAutoHeightEnclosesFloats is CSS-REVIEW-02-C5: a flex item is a
// BFC root, so its auto height includes floating descendants. Reference:
// Chrome 143.0.7499.40, 50x100px float in a column flex item.
func TestFlexItemAutoHeightEnclosesFloats(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="flex" style="display:flex;flex-direction:column;width:150pt">`+
		`<div id="item"><div id="float" style="float:left;width:37.5pt;height:75pt"></div></div>`+
		`</div></body></html>`)

	item := boxByID(t, res, "item")
	floating := boxByID(t, res, "float")

	if !near(item.height, pxToPt(100)) {
		t.Errorf("flex item height = %.4fpt (%.2fpx), want 100px",
			item.height, item.height/ptPerCSSPx)
	}

	if !near(floating.height, pxToPt(100)) {
		t.Errorf("float height = %.4fpt, want 100px", floating.height)
	}
}

// TestAbsPositionedRootBodyAnchorsToICB is CSS-REVIEW-02-C6: an absolutely
// positioned box with no positioned ancestor anchors to the initial
// containing block, not the body content box. Reference: Chrome 143.0.7499.40
// on test/chrome/cases/case-16-wpt-align-items-stretch.html, where the
// description paragraph's 12pt inset lands at (16, 16) CSS px despite the
// default 8px body margin. A positioned body is the containing block, so its
// padding box stays the origin there.
func TestAbsPositionedRootBodyAnchorsToICB(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		bodyStyle string
		wantPx    float64
	}{
		{"no positioned ancestor", "", 16},
		{"relative body", "position:relative", 24},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html><body style="`+testCase.bodyStyle+`">`+
				`<p id="abs" style="position:absolute;top:12pt;left:12pt;margin:0">x</p>`+
				`</body></html>`)

			abs := boxByID(t, res, "abs")

			if !near(abs.x, pxToPt(testCase.wantPx)) || !near(abs.y, pxToPt(testCase.wantPx)) {
				t.Errorf("abs p = (%.4fpt, %.4fpt) = (%.2fpx, %.2fpx), want (%.0fpx, %.0fpx)",
					abs.x, abs.y, abs.x/ptPerCSSPx, abs.y/ptPerCSSPx,
					testCase.wantPx, testCase.wantPx)
			}
		})
	}
}
