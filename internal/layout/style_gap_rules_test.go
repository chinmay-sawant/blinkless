package layout

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// rowRuleClass resolves one class the same way TestColumnRuleParse reads
// column-rule: parse the sheet, walk the tree, then look the class up.
func rowRuleClass(t *testing.T, cssText, className string) *ResolvedStyle {
	t.Helper()

	root := mustParse(t, `<html><body><div class="`+className+`">x</div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, cssText)}, "print", testViewport, 800)

	return styleByClass(t, styles, className)
}

// nearInset reports whether one inset stores the same length in points.
func nearInset(got ruleInset, want float64) bool {
	return !got.Percent && !got.OverlapJoin && near(got.Value, want)
}

// TestRowRuleWidthSixPx is row-rule-width: 6px stores 6px and a one-element list.
func TestRowRuleWidthSixPx(t *testing.T) {
	t.Parallel()

	sty := rowRuleClass(t, `.w { row-rule-width: 6px }`, "w")

	if !near(sty.RowRuleWidth, pxToPt(6)) {
		t.Fatalf("width=%.3f, want %.3f", sty.RowRuleWidth, pxToPt(6))
	}

	if len(sty.RowRuleWidths) != 1 || !near(sty.RowRuleWidths[0], pxToPt(6)) {
		t.Fatalf("widths=%v, want one %.3f", sty.RowRuleWidths, pxToPt(6))
	}
}

// TestRowRuleWidthListKeepsFirst is row-rule-width: 3px 6px stores both
// lengths. RowRuleWidth is the first item. The painter uses the first item.
func TestRowRuleWidthListKeepsFirst(t *testing.T) {
	t.Parallel()

	sty := rowRuleClass(t, `.w { row-rule-width: 3px 6px }`, "w")

	if len(sty.RowRuleWidths) != 2 ||
		!near(sty.RowRuleWidths[0], pxToPt(3)) ||
		!near(sty.RowRuleWidths[1], pxToPt(6)) {
		t.Fatalf("widths=%v, want %.3f then %.3f", sty.RowRuleWidths, pxToPt(3), pxToPt(6))
	}

	if !near(sty.RowRuleWidth, sty.RowRuleWidths[0]) {
		t.Fatalf("RowRuleWidth=%.3f, want first item %.3f", sty.RowRuleWidth, sty.RowRuleWidths[0])
	}
}

// TestRowRuleWidthBadTokenKeepsEarlier is row-rule-width: 4px followed by
// row-rule-width: nope. The bad token does not replace 4px.
func TestRowRuleWidthBadTokenKeepsEarlier(t *testing.T) {
	t.Parallel()

	sty := rowRuleClass(t, `.w { row-rule-width: 4px; row-rule-width: nope }`, "w")

	if !near(sty.RowRuleWidth, pxToPt(4)) {
		t.Fatalf("width=%.3f, want 4px kept (%.3f)", sty.RowRuleWidth, pxToPt(4))
	}
}

// TestRowRuleStyleDashed is row-rule-style: dashed stores dashed. A later bad
// style token does not replace dashed.
func TestRowRuleStyleDashed(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
<div class="dashed">A</div>
<div class="kept">B</div>
</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
.dashed { row-rule-style: dashed }
.kept { row-rule-style: dashed; row-rule-style: nope }
`)}, "print", testViewport, 800)

	if got := styleByClass(t, styles, "dashed").RowRuleStyle; got != borderStyleDashed {
		t.Fatalf("dashed style=%q, want dashed", got)
	}

	if got := styleByClass(t, styles, "kept").RowRuleStyle; got != borderStyleDashed {
		t.Fatalf("bad token replaced style with %q, want dashed kept", got)
	}
}

// TestRowRuleColorBlue is row-rule-color: #00f sets ColorSet and blue.
func TestRowRuleColorBlue(t *testing.T) {
	t.Parallel()

	sty := rowRuleClass(t, `.c { row-rule-color: #00f }`, "c")

	if !sty.RowRuleColorSet {
		t.Fatal("RowRuleColorSet false, want true")
	}

	if sty.RowRuleColor != ([3]float64{0, 0, 1}) {
		t.Fatalf("color=%v, want blue (0, 0, 1)", sty.RowRuleColor)
	}
}

// TestRowRuleShorthand is row-rule: 3px solid #c00 sets width, solid, and red.
func TestRowRuleShorthand(t *testing.T) {
	t.Parallel()

	sty := rowRuleClass(t, `.a { row-rule: 3px solid #c00 }`, "a")

	if !near(sty.RowRuleWidth, pxToPt(3)) {
		t.Fatalf("width=%.3f, want %.3f", sty.RowRuleWidth, pxToPt(3))
	}

	if sty.RowRuleStyle != solidKeyword {
		t.Fatalf("style=%q, want solid", sty.RowRuleStyle)
	}

	if !sty.RowRuleColorSet {
		t.Fatal("RowRuleColorSet false, want true")
	}

	if !near(sty.RowRuleColor[0], 0.8) || !near(sty.RowRuleColor[1], 0) || !near(sty.RowRuleColor[2], 0) {
		t.Fatalf("color=%v, want #c00 red", sty.RowRuleColor)
	}
}

// TestRowRuleShorthandResetsWidth is row-rule: solid after a previous width.
// The omitted width resets to medium and the style becomes solid.
func TestRowRuleShorthandResetsWidth(t *testing.T) {
	t.Parallel()

	sty := rowRuleClass(t, `.a { row-rule-width: 4px; row-rule: solid }`, "a")
	want := borderWidth(mediumKeyword, 12)

	if !near(sty.RowRuleWidth, want) {
		t.Fatalf("width=%.3f, want medium %.3f", sty.RowRuleWidth, want)
	}

	if sty.RowRuleStyle != solidKeyword {
		t.Fatalf("style=%q, want solid", sty.RowRuleStyle)
	}
}

// rowRuleBadTokenCase is one row in TestRowRuleBadTokenKeepsEarlier: a
// property, a good value, and the check that the stored field survives a
// later bad token.
type rowRuleBadTokenCase struct {
	prop  string
	good  string
	check func(*ResolvedStyle) bool
}

// TestRowRuleBadTokenKeepsEarlier is one entry per in-scope name. A bad token
// does not replace the earlier valid value.
//
//nolint:lll // each name is one table row
func TestRowRuleBadTokenKeepsEarlier(t *testing.T) {
	t.Parallel()

	cases := []rowRuleBadTokenCase{
		{"row-rule-width", "4px", func(s *ResolvedStyle) bool { return near(s.RowRuleWidth, pxToPt(4)) }},
		{"row-rule-style", "dashed", func(s *ResolvedStyle) bool { return s.RowRuleStyle == borderStyleDashed }},
		{"row-rule-color", "#00f", func(s *ResolvedStyle) bool { return s.RowRuleColorSet && s.RowRuleColor == [3]float64{0, 0, 1} }},
		{"row-rule", "3px solid #c00", func(s *ResolvedStyle) bool { return s.RowRuleStyle == solidKeyword }},
		{"row-rule-break", "none", func(s *ResolvedStyle) bool { return s.RowRuleBreak == "none" }},
		{"row-rule-visibility-items", "around", func(s *ResolvedStyle) bool { return s.RowRuleVisibility == "around" }},
		{"row-rule-inset", "10px", func(s *ResolvedStyle) bool { return nearInset(s.RowRuleInsetCapStart, pxToPt(10)) }},
		{"row-rule-inset-cap", "10px", func(s *ResolvedStyle) bool { return nearInset(s.RowRuleInsetCapStart, pxToPt(10)) }},
		{"row-rule-inset-cap-start", "10px", func(s *ResolvedStyle) bool { return nearInset(s.RowRuleInsetCapStart, pxToPt(10)) }},
		{"row-rule-inset-cap-end", "10px", func(s *ResolvedStyle) bool { return nearInset(s.RowRuleInsetCapEnd, pxToPt(10)) }},
		{"row-rule-inset-start", "10px", func(s *ResolvedStyle) bool { return nearInset(s.RowRuleInsetCapStart, pxToPt(10)) }},
		{"row-rule-inset-end", "10px", func(s *ResolvedStyle) bool { return nearInset(s.RowRuleInsetCapEnd, pxToPt(10)) }},
		{"row-rule-inset-junction", "4px", func(s *ResolvedStyle) bool { return nearInset(s.RowRuleInsetJunctionStart, pxToPt(4)) }},
		{"row-rule-inset-junction-start", "4px", func(s *ResolvedStyle) bool { return nearInset(s.RowRuleInsetJunctionStart, pxToPt(4)) }},
		{"row-rule-inset-junction-end", "4px", func(s *ResolvedStyle) bool { return nearInset(s.RowRuleInsetJunctionEnd, pxToPt(4)) }},
		{"rule", "3px solid #c00", func(s *ResolvedStyle) bool {
			return s.RowRuleStyle == solidKeyword && s.ColumnRuleStyle == solidKeyword
		}},
		{"rule-width", "6px", func(s *ResolvedStyle) bool {
			return near(s.RowRuleWidth, pxToPt(6)) && near(s.ColumnRuleWidth, pxToPt(6))
		}},
		{"rule-style", "dashed", func(s *ResolvedStyle) bool {
			return s.RowRuleStyle == borderStyleDashed && s.ColumnRuleStyle == borderStyleDashed
		}},
		{"rule-color", "#00f", func(s *ResolvedStyle) bool { return s.RowRuleColorSet && s.ColumnRuleColorSet }},
		{"rule-break", "none", func(s *ResolvedStyle) bool {
			return s.RowRuleBreak == "none" && s.ColumnRuleBreak == "none"
		}},
		{"rule-visibility-items", "around", func(s *ResolvedStyle) bool { return s.RowRuleVisibility == "around" }},
		{"rule-inset", "10px", func(s *ResolvedStyle) bool { return nearInset(s.RowRuleInsetCapStart, pxToPt(10)) }},
		{"rule-inset-cap", "10px", func(s *ResolvedStyle) bool { return nearInset(s.RowRuleInsetCapEnd, pxToPt(10)) }},
		{"rule-inset-start", "10px", func(s *ResolvedStyle) bool { return nearInset(s.RowRuleInsetCapStart, pxToPt(10)) }},
		{"rule-inset-end", "10px", func(s *ResolvedStyle) bool { return nearInset(s.RowRuleInsetCapEnd, pxToPt(10)) }},
		{"rule-inset-junction", "4px", func(s *ResolvedStyle) bool { return nearInset(s.RowRuleInsetJunctionStart, pxToPt(4)) }},
		{"column-rule-break", "intersection", func(s *ResolvedStyle) bool { return s.ColumnRuleBreak == "intersection" }},
		{"column-rule-inset", "10px", func(s *ResolvedStyle) bool { return nearInset(s.ColumnRuleInsetCapStart, pxToPt(10)) }},
		{"column-rule-inset-cap", "10px", func(s *ResolvedStyle) bool { return nearInset(s.ColumnRuleInsetCapStart, pxToPt(10)) }},
		{"column-rule-inset-cap-start", "10px", func(s *ResolvedStyle) bool { return nearInset(s.ColumnRuleInsetCapStart, pxToPt(10)) }},
		{"column-rule-inset-cap-end", "10px", func(s *ResolvedStyle) bool { return nearInset(s.ColumnRuleInsetCapEnd, pxToPt(10)) }},
		{"column-rule-inset-junction", "4px", func(s *ResolvedStyle) bool { return nearInset(s.ColumnRuleInsetJunctionStart, pxToPt(4)) }},
		{"column-rule-inset-junction-start", "4px", func(s *ResolvedStyle) bool { return nearInset(s.ColumnRuleInsetJunctionStart, pxToPt(4)) }},
		{"column-rule-inset-junction-end", "4px", func(s *ResolvedStyle) bool { return nearInset(s.ColumnRuleInsetJunctionEnd, pxToPt(4)) }},
		{"column-rule-inset-start", "10px", func(s *ResolvedStyle) bool { return nearInset(s.ColumnRuleInsetCapStart, pxToPt(10)) }},
		{"column-rule-inset-end", "10px", func(s *ResolvedStyle) bool { return nearInset(s.ColumnRuleInsetCapEnd, pxToPt(10)) }},
		{"rule-overlap", "row-over-column", func(s *ResolvedStyle) bool { return s.RuleOverlap == "row-over-column" }},
	}

	runRowRuleBadTokenCases(t, cases)
}

// runRowRuleBadTokenCases applies each case as a good declaration followed
// by the same property with the bad token nope, then runs the case check.
func runRowRuleBadTokenCases(t *testing.T, cases []rowRuleBadTokenCase) {
	t.Helper()

	if len(cases) != 37 {
		t.Fatalf("bad-token cases = %d, want 37", len(cases))
	}

	for _, testCase := range cases {
		t.Run(testCase.prop, func(t *testing.T) {
			t.Parallel()

			cssText := ".a { " + testCase.prop + ": " + testCase.good + "; " + testCase.prop + ": nope }"

			sty := rowRuleClass(t, cssText, "a")
			if !testCase.check(sty) {
				t.Fatalf("%s: nope replaced %s", testCase.prop, testCase.good)
			}
		})
	}
}

// TestRuleInsetCopySemantics mirrors Chrome's computed values for the
// css-gaps-1 inset shorthands: an omitted junction half copies the cap
// half, per-field pairs set both axes, and overlap-join and percentages
// survive as stored keywords.
func TestRuleInsetCopySemantics(t *testing.T) {
	t.Parallel()

	cssText := `.a { row-rule-inset: 10px 4px }
.b { rule-inset: 10px / 2px 1px }
.c { row-rule-inset-start: 5px }
.d { row-rule-inset-cap: 3px }
.e { rule-inset-junction: overlap-join }
.f { row-rule-inset: -6px / 50% }`

	checkRuleInsetCapCopiedToJunction(t, cssText)
	checkRuleInsetPairSetBothAxes(t, cssText)
	checkRuleInsetStartSide(t, cssText)
	checkRuleInsetCapOnly(t, cssText)
	checkRuleInsetOverlapJoin(t, cssText)
	checkRuleInsetPercentJunction(t, cssText)
}

// checkRuleInsetCapCopiedToJunction covers row-rule-inset: 10px 4px. Each
// omitted junction half copies its cap half and the column axis stays zero.
func checkRuleInsetCapCopiedToJunction(t *testing.T, cssText string) {
	t.Helper()

	copied := rowRuleClass(t, cssText, "a")
	if !nearInset(copied.RowRuleInsetCapStart, pxToPt(10)) ||
		!nearInset(copied.RowRuleInsetCapEnd, pxToPt(4)) ||
		!nearInset(copied.RowRuleInsetJunctionStart, pxToPt(10)) ||
		!nearInset(copied.RowRuleInsetJunctionEnd, pxToPt(4)) {
		t.Error("row-rule-inset: 10px 4px did not copy each cap field to its junction field")
	}

	if !nearInset(copied.ColumnRuleInsetCapStart, 0) {
		t.Error("row-rule-inset set a column inset")
	}
}

// checkRuleInsetPairSetBothAxes covers rule-inset: 10px / 2px 1px. Each
// per-field pair reaches both the row and the column axis.
func checkRuleInsetPairSetBothAxes(t *testing.T, cssText string) {
	t.Helper()

	bothAxes := rowRuleClass(t, cssText, "b")

	for _, probe := range []struct {
		name string
		got  ruleInset
		want float64
	}{
		{"row cap-start", bothAxes.RowRuleInsetCapStart, pxToPt(10)},
		{"row cap-end", bothAxes.RowRuleInsetCapEnd, pxToPt(10)},
		{"row junction-start", bothAxes.RowRuleInsetJunctionStart, pxToPt(2)},
		{"row junction-end", bothAxes.RowRuleInsetJunctionEnd, pxToPt(1)},
		{"column cap-start", bothAxes.ColumnRuleInsetCapStart, pxToPt(10)},
		{"column cap-end", bothAxes.ColumnRuleInsetCapEnd, pxToPt(10)},
		{"column junction-start", bothAxes.ColumnRuleInsetJunctionStart, pxToPt(2)},
		{"column junction-end", bothAxes.ColumnRuleInsetJunctionEnd, pxToPt(1)},
	} {
		if !nearInset(probe.got, probe.want) {
			t.Errorf("rule-inset %s = %+v, want %.3fpt", probe.name, probe.got, probe.want)
		}
	}
}

// checkRuleInsetStartSide covers row-rule-inset-start: 5px. The start side
// sets the row start cap and junction fields and leaves the rest zero.
func checkRuleInsetStartSide(t *testing.T, cssText string) {
	t.Helper()

	sided := rowRuleClass(t, cssText, "c")
	if !nearInset(sided.RowRuleInsetCapStart, pxToPt(5)) ||
		!nearInset(sided.RowRuleInsetJunctionStart, pxToPt(5)) {
		t.Error("row-rule-inset-start: 5px did not set the row start cap and junction fields")
	}

	if !nearInset(sided.RowRuleInsetCapEnd, 0) || !nearInset(sided.ColumnRuleInsetCapStart, 0) {
		t.Error("row-rule-inset-start: 5px reached the row end or the column axis")
	}
}

// checkRuleInsetCapOnly covers row-rule-inset-cap: 3px. Both row cap fields
// get 3px and the junction fields stay zero.
func checkRuleInsetCapOnly(t *testing.T, cssText string) {
	t.Helper()

	capsOnly := rowRuleClass(t, cssText, "d")
	if !nearInset(capsOnly.RowRuleInsetCapStart, pxToPt(3)) ||
		!nearInset(capsOnly.RowRuleInsetCapEnd, pxToPt(3)) ||
		!nearInset(capsOnly.RowRuleInsetJunctionStart, 0) {
		t.Error("row-rule-inset-cap: 3px did not set only the row cap fields")
	}
}

// checkRuleInsetOverlapJoin covers rule-inset-junction: overlap-join. All
// four junction fields keep the keyword and no cap field does.
func checkRuleInsetOverlapJoin(t *testing.T, cssText string) {
	t.Helper()

	joined := rowRuleClass(t, cssText, "e")
	if !joined.RowRuleInsetJunctionStart.OverlapJoin ||
		!joined.RowRuleInsetJunctionEnd.OverlapJoin ||
		!joined.ColumnRuleInsetJunctionStart.OverlapJoin ||
		!joined.ColumnRuleInsetJunctionEnd.OverlapJoin {
		t.Error("rule-inset-junction: overlap-join did not reach both axes")
	}

	if joined.RowRuleInsetCapStart.OverlapJoin || joined.ColumnRuleInsetCapEnd.OverlapJoin {
		t.Error("rule-inset-junction reached a cap field")
	}
}

// checkRuleInsetPercentJunction covers row-rule-inset: -6px / 50%. The cap
// halves keep the negative length and the junction halves keep the percent.
func checkRuleInsetPercentJunction(t *testing.T, cssText string) {
	t.Helper()

	mixed := rowRuleClass(t, cssText, "f")
	if !nearInset(mixed.RowRuleInsetCapStart, pxToPt(-6)) ||
		!nearInset(mixed.RowRuleInsetCapEnd, pxToPt(-6)) {
		t.Error("row-rule-inset: -6px / 50% did not store the negative cap length")
	}

	if !mixed.RowRuleInsetJunctionStart.Percent || !mixed.RowRuleInsetJunctionEnd.Percent ||
		!near(mixed.RowRuleInsetJunctionStart.Value, 50) {
		t.Error("row-rule-inset: -6px / 50% did not store the percentage junction halves")
	}
}
