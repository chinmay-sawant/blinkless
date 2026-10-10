package layout

import (
	"fmt"
	"testing"
)

// Tab-size through the paint fast path: \t expands to TabSize space widths,
// matching the slow path (tabStopAdvance in inline_paint.go).
// Reference: Chrome 143.0.7499.40, tab-size 2 vs 8 over "a\tb" in pre.

// TestBehaviorWaveGTabSizeValueChangesWidth pins that tab-size 2 vs 8 emit
// different used widths (the fast path consults TabSize).
func TestBehaviorWaveGTabSizeValueChangesWidth(t *testing.T) {
	t.Parallel()

	const page = "<pre class=\"%s\" style=\"margin:0;font-size:12pt;white-space:pre\">%s</pre>"

	cssSheet := sheet(t, `.t2 { tab-size: 2 } .t8 { tab-size: 8 }`)
	narrow := layoutHTML(t, `<html><body style="margin:0">`+fmt.Sprintf(page, "t2", "a\tb")+`</body></html>`, cssSheet)
	wide := layoutHTML(t, `<html><body style="margin:0">`+fmt.Sprintf(page, "t8", "a\tb")+`</body></html>`, cssSheet)
	untabbed := layoutHTML(t, `<html><body style="margin:0">`+fmt.Sprintf(page, "t8", "ab")+`</body></html>`, cssSheet)

	narrowW := behaviorTextTotalWidth(narrow)
	wideW := behaviorTextTotalWidth(wide)

	if near(wideW, narrowW) {
		t.Errorf("tab-size:8 width %.4f should differ from tab-size:2 width %.4f", wideW, narrowW)
	}

	if wideW <= narrowW {
		t.Errorf("tab-size:8 width %.4f should exceed tab-size:2 width %.4f", wideW, narrowW)
	}

	if narrowW <= behaviorTextTotalWidth(untabbed) {
		t.Errorf("tabbed width %.4f should exceed untabbed %.4f", narrowW, behaviorTextTotalWidth(untabbed))
	}
}

// TestBehaviorWaveGTabSizeSameValueDeterministic pins that the same tab-size
// value lays out identically on repeat.
func TestBehaviorWaveGTabSizeSameValueDeterministic(t *testing.T) {
	t.Parallel()

	const page = "<pre class=\"%s\" style=\"margin:0;font-size:12pt;white-space:pre\">%s</pre>"

	cssSheet := sheet(t, `.t8 { tab-size: 8 }`)
	first := layoutHTML(t, `<html><body style="margin:0">`+fmt.Sprintf(page, "t8", "a\tb")+`</body></html>`, cssSheet)
	second := layoutHTML(t, `<html><body style="margin:0">`+fmt.Sprintf(page, "t8", "a\tb")+`</body></html>`, cssSheet)

	if !near(behaviorTextTotalWidth(first), behaviorTextTotalWidth(second)) {
		t.Errorf("same tab-size widths differ: %.4f vs %.4f",
			behaviorTextTotalWidth(first), behaviorTextTotalWidth(second))
	}
}
