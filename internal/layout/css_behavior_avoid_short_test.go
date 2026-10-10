package layout

import (
	"strings"
	"testing"
)

// TestAvoidShortWireRebreaksOrphan pins the avoid-short-last-line end to end
// wiring on a three-line paragraph: text-wrap-style: avoid-short-last-line
// reaches the pack loop through the cascade table and repairs a greedy
// one-word tail without adding lines. It asserts used line breaks, never
// stored strings. Reference: Chrome 143.0.7499.40 pulls another word down
// instead of leaving the single-word tail.
func TestAvoidShortWireRebreaksOrphan(t *testing.T) {
	t.Parallel()

	const prose = "Pack my box with five dozen liquor jugs and a few extra things for the trip"

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:200pt;font-family:'Liberation Sans',sans-serif;`+
		`font-size:16px">`+prose+`</p></body></html>`)
	fixed := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:200pt;font-family:'Liberation Sans',sans-serif;`+
		`font-size:16px;text-wrap-style:avoid-short-last-line">`+prose+`</p></body></html>`)

	plainLines := inlineLines(plain)
	fixedLines := inlineLines(fixed)

	t.Logf("plain lines=%q", lineTexts(plainLines))
	t.Logf("fixed lines=%q", lineTexts(fixedLines))

	if len(plainLines) == 0 || len(fixedLines) == 0 {
		t.Fatalf("want lines for both layouts, got plain=%d fixed=%d", len(plainLines), len(fixedLines))
	}

	plainTail := plainLines[len(plainLines)-1].text
	if len(strings.Fields(plainTail)) != 1 {
		t.Fatalf("plain last line = %q, want a one-word tail fixture", plainTail)
	}

	if len(fixedLines) != len(plainLines) {
		t.Fatalf("avoid-short lines = %d, want plain %d (no added lines)", len(fixedLines), len(plainLines))
	}

	fixedTail := fixedLines[len(fixedLines)-1].text
	if len(strings.Fields(fixedTail)) < avoidShortMinLastWords {
		t.Errorf("avoid-short last line = %q, want at least %d words", fixedTail, avoidShortMinLastWords)
	}

	same := true

	for idx := range plainLines {
		if plainLines[idx].text != fixedLines[idx].text {
			same = false
		}
	}

	if same {
		t.Errorf("avoid-short breaks %q identical to greedy, want a repaired tail", lineTexts(fixedLines))
	}
}

// TestAvoidShortHelperRebreaksOrphan pins the width search on synthetic items:
// seven 40pt words in a 130pt box wrap 3+3+1 greedily, and
// avoid-short-last-line re-breaks them at a narrower width so the last line
// keeps two words without changing the line count. Reference: Chrome
// 143.0.7499.40 pulls another word down instead of leaving a one-word tail.
func TestAvoidShortHelperRebreaksOrphan(t *testing.T) {
	t.Parallel()

	items := []inlineItem{
		{text: "a", w: 40},
		{text: "b", w: 40},
		{text: "c", w: 40},
		{text: "d", w: 40},
		{text: "e", w: 40},
		{text: "f", w: 40},
	}

	greedy, packOK := prettyPackLines(items, 220)
	if !packOK || len(greedy) != 2 {
		t.Fatalf("prettyPackLines(220) lines = %+v, %v; want 2 lines", greedy, packOK)
	}

	if greedy[1].words != 1 {
		t.Fatalf("greedy last line words = %d, want 1 (tail fixture)", greedy[1].words)
	}

	got := avoidShortLineWidth(items, 220, 1)
	if got <= 0 || got >= 220 {
		t.Fatalf("avoidShortLineWidth(220) = %.2f, want a narrower width in (0, 220)", got)
	}

	fixed, fixOK := prettyPackLines(items, got)
	if !fixOK || len(fixed) != len(greedy) {
		t.Fatalf("prettyPackLines(%.2f) lines = %+v, %v; want %d lines", got, fixed, fixOK, len(greedy))
	}

	if fixed[len(fixed)-1].words < avoidShortMinLastWords {
		t.Errorf("avoid-short last line words = %d, want at least %d", fixed[len(fixed)-1].words, avoidShortMinLastWords)
	}
}

// TestAvoidShortKeepsGoodBreaks pins the no-op sides: an already even tail, a
// single line, an unfittable width, and a forced break all return 0 so greedy
// stands.
func TestAvoidShortKeepsGoodBreaks(t *testing.T) {
	t.Parallel()

	items := []inlineItem{
		{text: "a", w: 40},
		{text: "b", w: 40},
		{text: "c", w: 40},
		{text: "d", w: 40},
	}

	if got := avoidShortLineWidth(items, 90, 0.5); got != 0 {
		t.Errorf("avoidShortLineWidth(90, even 2+2) = %.2f, want 0", got)
	}

	if got := avoidShortLineWidth(items, 200, 0.5); got != 0 {
		t.Errorf("avoidShortLineWidth(200, single line) = %.2f, want 0", got)
	}

	if got := avoidShortLineWidth(items, 30, 0.5); got != 0 {
		t.Errorf("avoidShortLineWidth(30, unfittable) = %.2f, want 0", got)
	}

	broken := []inlineItem{{text: "a", w: 40}, {forceBreak: true}, {text: "b", w: 40}}
	if got := avoidShortLineWidth(broken, 130, 0.5); got != 0 {
		t.Errorf("avoidShortLineWidth with forced break = %.2f, want 0", got)
	}
}

// TestAvoidShortApplyGates mirrors the pretty gates: floats, line clamp,
// indents, and non-avoid styles opt out.
func TestAvoidShortApplyGates(t *testing.T) {
	t.Parallel()

	avoided := &ResolvedStyle{TextWrapStyle: "avoid-short-last-line"}
	pretty := &ResolvedStyle{TextWrapStyle: "pretty"}

	if !avoidShortCanApply(nil, 0, avoided) {
		t.Error("avoidShortCanApply(nil, 0, avoid) = false, want true")
	}

	if avoidShortCanApply(&floatState{hasLeft: true}, 0, avoided) {
		t.Error("avoidShortCanApply with active float = true, want false")
	}

	if avoidShortCanApply(nil, 2, avoided) {
		t.Error("avoidShortCanApply with line clamp = true, want false")
	}

	indented := &ResolvedStyle{TextWrapStyle: "avoid-short-last-line", TextIndent: 12}
	if avoidShortCanApply(nil, 0, indented) {
		t.Error("avoidShortCanApply with indent = true, want false")
	}

	if avoidShortCanApply(nil, 0, pretty) {
		t.Error("avoidShortCanApply(pretty style) = true, want false")
	}

	if avoidShortCanApply(nil, 0, nil) {
		t.Error("avoidShortCanApply(nil style) = true, want false")
	}
}
