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

// TestRowRuleBadTokenKeepsEarlier is one entry per in-scope name. A bad token
// does not replace the earlier valid value.
//
//nolint:lll,wsl // each name is one table row
func TestRowRuleBadTokenKeepsEarlier(t *testing.T) {
	t.Parallel()

	cases := []struct {
		prop  string
		good  string
		check func(*ResolvedStyle) bool
	}{
		{"row-rule-width", "4px", func(s *ResolvedStyle) bool { return near(s.RowRuleWidth, pxToPt(4)) }},
		{"row-rule-style", "dashed", func(s *ResolvedStyle) bool { return s.RowRuleStyle == borderStyleDashed }},
		{"row-rule-color", "#00f", func(s *ResolvedStyle) bool { return s.RowRuleColorSet && s.RowRuleColor == [3]float64{0, 0, 1} }},
		{"row-rule", "3px solid #c00", func(s *ResolvedStyle) bool { return s.RowRuleStyle == solidKeyword }},
		{"row-rule-break", "none", func(s *ResolvedStyle) bool { return s.RowRuleBreak == "none" }},
		{"row-rule-visibility-items", "around", func(s *ResolvedStyle) bool { return s.RowRuleVisibility == "around" }},
		{"row-rule-inset", "10px", func(s *ResolvedStyle) bool { return near(s.RowRuleInsetCapStart, pxToPt(10)) }},
		{"row-rule-inset-cap", "10px", func(s *ResolvedStyle) bool { return near(s.RowRuleInsetCapStart, pxToPt(10)) }},
		{"row-rule-inset-cap-start", "10px", func(s *ResolvedStyle) bool { return near(s.RowRuleInsetCapStart, pxToPt(10)) }},
		{"row-rule-inset-cap-end", "10px", func(s *ResolvedStyle) bool { return near(s.RowRuleInsetCapEnd, pxToPt(10)) }},
		{"row-rule-inset-start", "10px", func(s *ResolvedStyle) bool { return near(s.RowRuleInsetCapStart, pxToPt(10)) }},
		{"row-rule-inset-end", "10px", func(s *ResolvedStyle) bool { return near(s.RowRuleInsetCapEnd, pxToPt(10)) }},
		{"row-rule-inset-junction", "4px", func(s *ResolvedStyle) bool { return near(s.RowRuleInsetJunctionStart, pxToPt(4)) }},
		{"row-rule-inset-junction-start", "4px", func(s *ResolvedStyle) bool { return near(s.RowRuleInsetJunctionStart, pxToPt(4)) }},
		{"row-rule-inset-junction-end", "4px", func(s *ResolvedStyle) bool { return near(s.RowRuleInsetJunctionEnd, pxToPt(4)) }},
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
		{"rule-break", "none", func(s *ResolvedStyle) bool { return s.RowRuleBreak == "none" }},
		{"rule-visibility-items", "around", func(s *ResolvedStyle) bool { return s.RowRuleVisibility == "around" }},
		{"rule-inset", "10px", func(s *ResolvedStyle) bool { return near(s.RowRuleInsetCapStart, pxToPt(10)) }},
		{"rule-inset-cap", "10px", func(s *ResolvedStyle) bool { return near(s.RowRuleInsetCapEnd, pxToPt(10)) }},
		{"rule-inset-start", "10px", func(s *ResolvedStyle) bool { return near(s.RowRuleInsetCapStart, pxToPt(10)) }},
		{"rule-inset-end", "10px", func(s *ResolvedStyle) bool { return near(s.RowRuleInsetCapEnd, pxToPt(10)) }},
		{"rule-inset-junction", "4px", func(s *ResolvedStyle) bool { return near(s.RowRuleInsetJunctionStart, pxToPt(4)) }},
		{"rule-overlap", "row-over-column", func(s *ResolvedStyle) bool { return s.RuleOverlap == "row-over-column" }},
	}

	if len(cases) != 27 {
		t.Fatalf("bad-token cases = %d, want 27", len(cases))
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
