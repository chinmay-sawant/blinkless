package layout

import (
	"strings"
	"testing"
)

// Container query behavior tests: container-type, container-name, and the
// container shorthand are observable through @container width thresholds.
// Every property is asserted through a used value (a measured box edge or an
// emitted text op color), never through a stored style string. Lengths are
// written in CSS px and converted with pxToPt (1px = 0.75pt). Reference
// browser for every case: Chrome 143.0.7499.40.
//
// A bare container declaration with no @container rule paints identical
// geometry (see TestBehaviorContainerShorthandNoLayoutEffect and friends);
// these tests close that gap by pairing each declaration form with a
// matching query at a 200px threshold: a 400px container matches while a
// 100px container does not.

// containerTextByColor joins the text of every text op painted with the
// given ink (red = query matched, black = query missed). Narrow queried
// boxes wrap per character, so one logical word spans many ops and no single
// op holds the full word; joining by ink keeps the assertion on used paint.
func containerTextByColor(res *Result, red bool) string {
	var out strings.Builder

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		if containerInkMatches(paintOp, red) {
			out.WriteString(paintOp.Text)
		}
	}

	return out.String()
}

// containerInkMatches reports whether a text op carries the wanted ink: red
// for a matched query, black for a missed one.
func containerInkMatches(paintOp Op, red bool) bool {
	isRed := paintOp.R > 0.9 && paintOp.G < 0.1 && paintOp.B < 0.1
	isBlack := paintOp.R < 0.1 && paintOp.G < 0.1 && paintOp.B < 0.1

	return red && isRed || !red && isBlack
}

// containerCheckThresholdItem asserts the used width switch at the 200px
// threshold: the wide item takes the queried 50px width and red paint while
// the narrow item keeps its 10px width and black paint.
func containerCheckThresholdItem(t *testing.T, res *Result, wideID, narrowID, wideText, narrowText string) {
	t.Helper()

	wide := boxByID(t, res, wideID)
	if !near(wide.w, pxToPt(50)) {
		t.Errorf("#%s used width = %.4fpt (%.2fpx), want 50px (query matched)", wideID, wide.w, wide.w/ptPerCSSPx)
	}

	narrow := boxByID(t, res, narrowID)
	if !near(narrow.w, pxToPt(10)) {
		t.Errorf("#%s used width = %.4fpt (%.2fpx), want 10px (query missed)", narrowID, narrow.w, narrow.w/ptPerCSSPx)
	}

	wideOp := containerTextByColor(res, true)
	if wideOp != wideText {
		t.Errorf("red text = %q, want %q (only the wide container matched)", wideOp, wideText)
	}

	narrowOp := containerTextByColor(res, false)
	if narrowOp != narrowText {
		t.Errorf("black text = %q, want %q (only the narrow container missed)", narrowOp, narrowText)
	}
}

// TestBehaviorContainerTypeInlineSizeQuerySwitch is container-type: a
// 400px inline-size container matches (min-width:200px) so its child takes
// the queried width and paint, while a 100px container misses and keeps the
// base style. Reference: Chrome 143.0.7499.40.
func TestBehaviorContainerTypeInlineSizeQuerySwitch(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
		.wide { width: 400px; container-type: inline-size }
		.narrow { width: 100px; container-type: inline-size }
		.item { width: 10px; height: 20px; color: black }
		@container (min-width: 200px) {
			.item { width: 50px; color: red }
		}
	`)
	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div class="wide"><div class="item" id="wideItem">WideType</div></div>`+
		`<div class="narrow"><div class="item" id="narrowItem">NarrowType</div></div>`+
		`</body></html>`, cssSheet)

	containerCheckThresholdItem(t, res, "wideItem", "narrowItem", "WideType", "NarrowType")
}

// TestBehaviorContainerShorthandNamedQuerySwitch is container: the shorthand
// registers name card with inline-size containment in one declaration, so a
// named @container card query switches the descendant style at the same
// 200px threshold. Reference: Chrome 143.0.7499.40.
func TestBehaviorContainerShorthandNamedQuerySwitch(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
		.wide { width: 400px; container: card / inline-size }
		.narrow { width: 100px; container: card / inline-size }
		.item { width: 10px; height: 20px; color: black }
		@container card (min-width: 200px) {
			.item { width: 50px; color: red }
		}
	`)
	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div class="wide"><div class="item" id="wideCard">WideCard</div></div>`+
		`<div class="narrow"><div class="item" id="narrowCard">NarrowCard</div></div>`+
		`</body></html>`, cssSheet)

	containerCheckThresholdItem(t, res, "wideCard", "narrowCard", "WideCard", "NarrowCard")
}

// TestBehaviorContainerNameLonghandQuerySwitch is container-name: a name
// alone never establishes a size container, but paired with
// container-type:inline-size it selects the named query, switching the
// descendant style at the 200px threshold. Reference: Chrome 143.0.7499.40.
func TestBehaviorContainerNameLonghandQuerySwitch(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
		.wide { width: 400px; container-name: card; container-type: inline-size }
		.narrow { width: 100px; container-name: card; container-type: inline-size }
		.item { width: 10px; height: 20px; color: black }
		@container card (min-width: 200px) {
			.item { width: 50px; color: red }
		}
	`)
	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div class="wide"><div class="item" id="wideName">WideName</div></div>`+
		`<div class="narrow"><div class="item" id="narrowName">NarrowName</div></div>`+
		`</body></html>`, cssSheet)

	containerCheckThresholdItem(t, res, "wideName", "narrowName", "WideName", "NarrowName")
}
