package layout

import (
	"testing"
)

// Ruby behavior tests: each property is asserted through USED annotation
// geometry (offsets and sizes in points), never through stored style strings.
// Reference browser for every case: Chrome 143.0.7499.40. Spec:
// https://drafts.csswg.org/css-ruby-1/
//
// The engine has no ruby formatting context yet (see the integration seam
// note in ruby.go), so these tests exercise the pure geometry helpers in
// ruby.go plus one gap test pinning the current flattened Layout output. When
// the cascade and inline seams land, matching Layout-level tests through
// layoutHTML should replace the helper calls without changing the numbers.

// TestBehaviorRubyAlignCentersNarrowAnnotation: a narrow annotation centers in a wide
// base column under center and the initial space-around, but pins to the
// start edge under start. Chrome centers short kana over kanji by default.
func TestBehaviorRubyAlignCentersNarrowAnnotation(t *testing.T) {
	t.Parallel()

	const baseW, annotW = 40.0, 20.0

	start := rubyAlignOffset(baseW, annotW, "start")
	center := rubyAlignOffset(baseW, annotW, "center")
	around := rubyAlignOffset(baseW, annotW, "space-around")

	if !near(start, 0) {
		t.Errorf("ruby-align:start offset = %.4fpt, want 0", start)
	}

	if !near(center, 10) {
		t.Errorf("ruby-align:center offset = %.4fpt, want 10", center)
	}

	if !near(around, 10) {
		t.Errorf("ruby-align:space-around offset = %.4fpt, want 10 (initial centers narrow annotation)", around)
	}

	if !near(center, around) {
		t.Errorf("center %.4fpt vs space-around %.4fpt, want identical for a lone annotation", center, around)
	}

	if near(start, center) {
		t.Errorf("start %.4fpt vs center %.4fpt, want distinct placement", start, center)
	}
}

// TestBehaviorRubyAlignWideAnnotationFillsColumn: when the annotation is wider than
// the base there is no extra space, so every value offsets 0 and the column
// grows to the annotation width instead.
func TestBehaviorRubyAlignWideAnnotationFillsColumn(t *testing.T) {
	t.Parallel()

	const baseW, annotW = 20.0, 30.0

	for _, align := range []string{"start", "center", "space-between", "space-around"} {
		if got := rubyAlignOffset(baseW, annotW, align); !near(got, 0) {
			t.Errorf("ruby-align:%s wide offset = %.4fpt, want 0 (no extra space)", align, got)
		}
	}
}

// TestBehaviorRubyPositionStacksOverAndUnder: over/alternate stack the annotation
// directly above the base (-annotH) while under stacks it directly below
// (+baseH). Chrome paints <rt> above the base by default and below for
// ruby-position:under.
func TestBehaviorRubyPositionStacksOverAndUnder(t *testing.T) {
	t.Parallel()

	const baseH, annotH = 13.5, 6.75

	over := rubyAnnotationStackOffset(baseH, annotH, rubyPositionOver)
	alternate := rubyAnnotationStackOffset(baseH, annotH, rubyPositionAlternate)
	under := rubyAnnotationStackOffset(baseH, annotH, rubyPositionUnder)
	side := rubyAnnotationStackOffset(baseH, annotH, rubyPositionInterCharacter)

	assertRubyStackSigns(t, over, alternate, under, side, baseH, annotH)

	if got := rubyAnnotationStackOffset(20, 10, rubyPositionUnder); !near(got, 20) {
		t.Errorf("ruby-position:under offset at 20/10 = %.4fpt, want 20", got)
	}

	if over >= 0 || under <= 0 {
		t.Errorf("over %.4fpt and under %.4fpt must sit on opposite sides of the base", over, under)
	}

	if rubyPositionIsUnder(rubyPositionUnder) == rubyPositionIsUnder(rubyPositionOver) {
		t.Errorf("under/over side predicate must differ")
	}

	if !rubyPositionIsInterCharacter(rubyPositionInterCharacter) || rubyPositionIsInterCharacter(rubyPositionOver) {
		t.Errorf("inter-character predicate must match only inter-character")
	}
}

// assertRubyStackSigns checks the four stacking offsets against the base and
// annotation heights.
func assertRubyStackSigns(t *testing.T, over, alternate, under, side, baseH, annotH float64) {
	t.Helper()

	if !near(over, -annotH) {
		t.Errorf("ruby-position:over offset = %.4fpt, want %.4fpt", over, -annotH)
	}

	if !near(alternate, -annotH) {
		t.Errorf("ruby-position:alternate offset = %.4fpt, want %.4fpt (first level behaves as over)", alternate, -annotH)
	}

	if !near(under, baseH) {
		t.Errorf("ruby-position:under offset = %.4fpt, want %.4fpt", under, baseH)
	}

	if !near(side, 0) {
		t.Errorf("ruby-position:inter-character vertical offset = %.4fpt, want 0 (side placement)", side)
	}
}

// TestBehaviorRubyMergeSharesAnnotationSpace: with two columns whose first annotation
// is wider than its base, separate keeps per-column maxima while merge fits
// the wider of the summed runs, so merged is narrower. Auto matches separate
// when everything fits and merge otherwise (the spec jukugo rule).
func TestBehaviorRubyMergeSharesAnnotationSpace(t *testing.T) {
	t.Parallel()

	baseWs := []float64{20, 20}
	wideAnnots := []float64{30, 10}
	fitAnnots := []float64{10, 10}

	separate := rubyMergeSegmentWidth(baseWs, wideAnnots, "separate")
	merged := rubyMergeSegmentWidth(baseWs, wideAnnots, "merge")
	autoWide := rubyMergeSegmentWidth(baseWs, wideAnnots, "auto")
	autoFit := rubyMergeSegmentWidth(baseWs, fitAnnots, "auto")
	separateFit := rubyMergeSegmentWidth(baseWs, fitAnnots, "separate")

	// separate = max(20,30)+max(20,10) = 50; merge = max(40,40) = 40.
	if !near(separate, 50) {
		t.Errorf("ruby-merge:separate width = %.4fpt, want 50", separate)
	}

	if !near(merged, 40) {
		t.Errorf("ruby-merge:merge width = %.4fpt, want 40", merged)
	}

	if merged >= separate {
		t.Errorf("merged %.4fpt vs separate %.4fpt, want merged narrower when sharing space", merged, separate)
	}

	if !near(autoWide, merged) {
		t.Errorf("ruby-merge:auto wide width = %.4fpt, want merged %.4fpt", autoWide, merged)
	}

	if !near(autoFit, separateFit) {
		t.Errorf("ruby-merge:auto fit width = %.4fpt, want separate %.4fpt", autoFit, separateFit)
	}
}

// TestBehaviorRubyOverhangGrowsBaseWhenForbidden: auto allows a wider annotation to
// overlap neighbors (no growth) while spaces forces the segment wider to fit
// it. Chrome lets long kana spill over neighbors by default and contains it
// under ruby-overhang:spaces (legacy none).
func TestBehaviorRubyOverhangGrowsBaseWhenForbidden(t *testing.T) {
	t.Parallel()

	const baseW, annotW = 20.0, 30.0

	auto := rubyOverhangExpand(baseW, annotW, "auto")
	spaces := rubyOverhangExpand(baseW, annotW, "spaces")
	noneAlias := rubyOverhangExpand(baseW, annotW, "none")

	if !near(auto, 0) {
		t.Errorf("ruby-overhang:auto expand = %.4fpt, want 0 (overlap allowed)", auto)
	}

	if !near(spaces, 10) {
		t.Errorf("ruby-overhang:spaces expand = %.4fpt, want 10", spaces)
	}

	if !near(noneAlias, 10) {
		t.Errorf("ruby-overhang:none expand = %.4fpt, want 10 (legacy alias of spaces)", noneAlias)
	}

	if near(auto, spaces) {
		t.Errorf("auto %.4fpt vs spaces %.4fpt, want distinct growth", auto, spaces)
	}

	if got := rubyOverhangExpand(baseW, 10.0, "spaces"); !near(got, 0) {
		t.Errorf("narrow annotation spaces expand = %.4fpt, want 0 (nothing to contain)", got)
	}

	if got := rubyOverhangExpand(30.0, 50.0, "spaces"); !near(got, 20) {
		t.Errorf("wide annotation spaces expand = %.4fpt, want 20", got)
	}
}

// TestBehaviorRubyParsersAcceptSpecValues: the four grammars accept their spec
// keywords and reject neighbors. This stays geometry-adjacent: every accepted
// value feeds one helper above, every rejected value keeps its initial.
func TestBehaviorRubyParsersAcceptSpecValues(t *testing.T) {
	t.Parallel()

	if got, ok := parseRubyAlign(" center "); !ok || got != "center" {
		t.Errorf("ruby-align:' center ' = %q,%v, want center,true (trimmed and lowered)", got, ok)
	}

	if _, ok := parseRubyAlign("bogus"); ok {
		t.Errorf("ruby-align:bogus accepted, want reject")
	}

	if got, ok := parseRubyMerge("Merge"); !ok || got != "merge" {
		t.Errorf("ruby-merge:'Merge' = %q,%v, want merge,true", got, ok)
	}

	if _, ok := parseRubyMerge("collapse"); ok {
		t.Errorf("ruby-merge:collapse accepted, want reject (not in separate|merge|auto)")
	}

	if got, ok := parseRubyOverhang("none"); !ok || got != "spaces" {
		t.Errorf("ruby-overhang:none = %q,%v, want spaces,true (legacy alias)", got, ok)
	}

	assertRubyPositionParses(t)
}

// assertRubyPositionParses checks the position grammar accepts.
func assertRubyPositionParses(t *testing.T) {
	t.Helper()

	if got, ok := parseRubyPosition("alternate under"); !ok || got != rubyPositionUnder {
		t.Errorf("ruby-position:alternate under = %q,%v, want under,true", got, ok)
	}

	if got, ok := parseRubyPosition("inter-character"); !ok || got != "inter-character" {
		t.Errorf("ruby-position:inter-character = %q,%v, want inter-character,true", got, ok)
	}

	if _, ok := parseRubyPosition("over under"); ok {
		t.Errorf("ruby-position:over under accepted, want reject (at most one side)")
	}
}
