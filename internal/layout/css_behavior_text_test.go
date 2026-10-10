package layout

import (
	"fmt"
	"testing"
)

// Text behavior tests: one test per text property, asserted through USED
// values (measured box sizes, text-op coordinates and widths, emitted text),
// never through stored style strings. Points are the engine unit and
// 1 CSS pixel = 0.75pt (ptPerCSSPx in css_review_02_test.go).
// Reference browser for every case: Chrome 143.0.7499.40.
//
// text-wrap-style: balance itself is pinned in inline_balance_test.go, so the
// text-wrap-style test here only pins auto-equals-normal plus the pretty
// rejection and does not duplicate those fixtures.

// behaviorTextTotalWidth sums the measured widths of all emitted text ops.
func behaviorTextTotalWidth(res *Result) float64 {
	var total float64

	for _, op := range res.Ops {
		if op.Kind == OpText {
			total += op.W
		}
	}

	return total
}

// TestBehaviorLineHeightExplicitEnlargesBox: an explicit line-height sets the
// used line box height. A 12pt paragraph with line-height 24pt lays out 24pt
// (32px) tall, taller than the normal-height sibling.
// Reference: Chrome 143.0.7499.40, font-size 12pt, line-height 24pt.
func TestBehaviorLineHeightExplicitEnlargesBox(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<p id="plain" style="margin:0;font-size:12pt">Hg</p>`+
		`<p id="tall" style="margin:0;font-size:12pt;line-height:24pt">Hg</p>`+
		`</body></html>`)

	plain := boxByID(t, res, "plain")
	tall := boxByID(t, res, "tall")

	if !near(tall.height, pxToPt(32)) {
		t.Errorf("line-height:24pt box height = %.4fpt (%.2fpx), want 32px",
			tall.height, tall.height/ptPerCSSPx)
	}

	if tall.height <= plain.height+1 {
		t.Errorf("explicit line-height height %.4fpt should exceed normal %.4fpt",
			tall.height, plain.height)
	}
}

// TestBehaviorLetterSpacingWidensText: letter-spacing adds per-rune advance,
// so the measured text width grows. 3pt over "hello" adds about 15pt.
// Reference: Chrome 143.0.7499.40, letter-spacing 3pt.
func TestBehaviorLetterSpacingWidensText(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;white-space:nowrap">hello</p></body></html>`)
	wide := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;white-space:nowrap;letter-spacing:3pt">hello</p></body></html>`)

	delta := behaviorTextTotalWidth(wide) - behaviorTextTotalWidth(plain)
	if delta < 10 {
		t.Errorf("letter-spacing:3pt widened text by %.4fpt, want at least 10pt", delta)
	}
}

// TestBehaviorWordSpacingWidensText: word-spacing adds advance after each
// space, so "hello world" measures wider with word-spacing 4pt.
// Reference: Chrome 143.0.7499.40, word-spacing 4pt.
func TestBehaviorWordSpacingWidensText(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;white-space:nowrap">hello world</p></body></html>`)
	wide := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;white-space:nowrap;word-spacing:4pt">hello world</p></body></html>`)

	delta := behaviorTextTotalWidth(wide) - behaviorTextTotalWidth(plain)
	if delta < 3.5 {
		t.Errorf("word-spacing:4pt widened text by %.4fpt, want at least 3.5pt", delta)
	}
}

// TestBehaviorTextAlignCentersLine: text-align center puts the line in the
// middle of the 200pt content box instead of at its left edge.
// Reference: Chrome 143.0.7499.40, width 200px, short centered line.
func TestBehaviorTextAlignCentersLine(t *testing.T) {
	t.Parallel()

	left := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:150pt;font-size:12pt">Hi</p></body></html>`)
	centered := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:150pt;font-size:12pt;text-align:center">Hi</p></body></html>`)

	xs := firstTextX(t, left, "Hi")
	xc := firstTextX(t, centered, "Hi")

	if xc <= xs+1 {
		t.Fatalf("centered X %.4f should be right of left X %.4f", xc, xs)
	}

	op := firstText(centered)
	if center := op.X + op.W/2; center < 70 || center > 80 {
		t.Errorf("centered line center = %.4fpt, want near 75pt (150pt box middle)", center)
	}
}

// TestBehaviorTextIndentShiftsFirstLine: text-indent 24pt starts the first
// line one indent inside the content edge.
// Reference: Chrome 143.0.7499.40, text-indent 24pt (32px).
func TestBehaviorTextIndentShiftsFirstLine(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">hello</p></body></html>`)
	indented := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-indent:24pt">hello</p></body></html>`)

	delta := firstTextX(t, indented, "hello") - firstTextX(t, plain, "hello")
	if !near(delta, 24) {
		t.Errorf("text-indent shift = %.4fpt, want 24pt", delta)
	}
}

// TestBehaviorTextTransformUppercaseEmitsCaps: text-transform uppercase is
// carried on the text op for paint time (Op.Text keeps source text) and the
// measured width is the uppercased run's width, wider than lowercase "hello".
// Reference: Chrome 143.0.7499.40, text-transform uppercase.
func TestBehaviorTextTransformUppercaseEmitsCaps(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-transform:uppercase">hello</p></body></html>`)
	lower := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">hello</p></body></html>`)

	upperOp := firstText(res)

	if got := upperOp.TextTransformValue(); got != "uppercase" {
		t.Errorf("text op transform = %q, want %q (carried for paint time)", got, "uppercase")
	}

	if upperOp.W <= firstText(lower).W {
		t.Errorf("uppercased width %.4f should exceed lowercase width %.4f",
			upperOp.W, firstText(lower).W)
	}

	if got := TransformInlineText(upperOp.Text, upperOp.TextTransformValue()); got != "HELLO" {
		t.Errorf("paint-time transform = %q, want %q", got, "HELLO")
	}
}

// TestBehaviorWhiteSpaceNowrapKeepsSingleLine: a long run wraps in a 60pt box
// under normal white-space but stays on one overflowing line with nowrap.
// Reference: Chrome 143.0.7499.40, white-space nowrap.
func TestBehaviorWhiteSpaceNowrapKeepsSingleLine(t *testing.T) {
	t.Parallel()

	const page = `<p style="margin:0;width:45pt;font-size:12pt%s">alpha bravo charlie delta echo</p>`

	wrapped := layoutHTML(t, `<html><body style="margin:0">`+fmt.Sprintf(page, "")+`</body></html>`)
	single := layoutHTML(t, `<html><body style="margin:0">`+fmt.Sprintf(page, ";white-space:nowrap")+`</body></html>`)

	if got := len(inlineLines(wrapped)); got < 2 {
		t.Errorf("normal white-space lines = %d, want at least 2 (wrapped)", got)
	}

	if got := len(inlineLines(single)); got != 1 {
		t.Errorf("nowrap lines = %d, want 1 (single overflowing line)", got)
	}
}

// TestBehaviorVerticalAlignSuperRaisesText: vertical-align super lifts the run
// above the baseline and sub drops it below it, measured as paint Y.
// Reference: Chrome 143.0.7499.40, super/sub shifts.
func TestBehaviorVerticalAlignSuperRaisesText(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">base`+
		`<span id="sup" style="vertical-align:super">sup</span>`+
		`<span id="sub" style="vertical-align:sub">sub</span></p></body></html>`)

	baseY := textY(t, res, "base")
	supY := textY(t, res, "sup")
	subY := textY(t, res, "sub")

	if supY >= baseY-0.5 {
		t.Errorf("super Y %.4f should be above baseline Y %.4f", supY, baseY)
	}

	if subY <= baseY+0.5 {
		t.Errorf("sub Y %.4f should be below baseline Y %.4f", subY, baseY)
	}
}

// TestBehaviorHyphensNoneSuppressesSoftHyphenBreak: a soft hyphen breaks the
// word with a visible hyphen under hyphens manual, while hyphens none never
// emits the hyphen at the soft hyphen (the narrow box still wraps the long
// word mid-word, but with no hyphen mark).
// Reference: Chrome 143.0.7499.40, narrow 14pt box, authored soft hyphen.
func TestBehaviorHyphensNoneSuppressesSoftHyphenBreak(t *testing.T) {
	t.Parallel()

	const word = "super\u00adcalifragilistic"

	manual := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:48pt;font-size:14pt;hyphens:manual">`+word+`</p></body></html>`)
	none := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:48pt;font-size:14pt;hyphens:none">`+word+`</p></body></html>`)

	if got := len(inlineLines(manual)); got < 2 {
		t.Errorf("hyphens:manual lines = %d, want at least 2 (SHY break)", got)
	}

	if !textOpsContain(manual, "-") {
		t.Errorf("hyphens:manual should emit a hyphen at the SHY break, got %q", joinedPaintText(manual))
	}

	if textOpsContain(none, "-") {
		t.Errorf("hyphens:none should emit no hyphen at the SHY break, got %q", joinedPaintText(none))
	}
}

// TestBehaviorTabSizeFixedAdvanceIgnoresValue is a DOCUMENTED GAP: the paint
// fast path (primaryFaceRun in inline_paint.go) measures a tab by the font's
// raw tab glyph advance and never consults TabSize, so tab-size 2 and tab-size
// 8 emit equal used widths. Chrome 143.0.7499.40 would advance 2 vs 8 space
// widths. The test pins the current behavior: equal widths, and a positive
// used advance for the tab versus the same run without it.
func TestBehaviorTabSizeFixedAdvanceIgnoresValue(t *testing.T) {
	t.Parallel()

	const page = "<pre class=\"%s\" style=\"margin:0;font-size:12pt;white-space:pre\">%s</pre>"

	cssSheet := sheet(t, `.t2 { tab-size: 2 } .t8 { tab-size: 8 }`)
	narrow := layoutHTML(t, `<html><body style="margin:0">`+fmt.Sprintf(page, "t2", "a\tb")+`</body></html>`, cssSheet)
	wide := layoutHTML(t, `<html><body style="margin:0">`+fmt.Sprintf(page, "t8", "a\tb")+`</body></html>`, cssSheet)
	untabbed := layoutHTML(t, `<html><body style="margin:0">`+fmt.Sprintf(page, "t8", "ab")+`</body></html>`, cssSheet)

	if !near(behaviorTextTotalWidth(wide), behaviorTextTotalWidth(narrow)) {
		t.Errorf("tab-size:8 width %.4f != tab-size:2 width %.4f (gap: value ignored)",
			behaviorTextTotalWidth(wide), behaviorTextTotalWidth(narrow))
	}

	if behaviorTextTotalWidth(narrow) <= behaviorTextTotalWidth(untabbed) {
		t.Errorf("tabbed width %.4f should exceed untabbed %.4f (tab has used advance)",
			behaviorTextTotalWidth(narrow), behaviorTextTotalWidth(untabbed))
	}
}

// TestBehaviorQuotesEmitGeneratedMarks: content open-quote/close-quote paints
// the quotes pair around the element text.
// Reference: Chrome 143.0.7499.40, quotes pair with generated content.
func TestBehaviorQuotesEmitGeneratedMarks(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
q { quotes: "‹" "›"; }
q::before { content: open-quote; }
q::after { content: close-quote; }
`)
	res := layoutHTML(t, `<html><body><q>hello</q></body></html>`, cssSheet)

	if got := joinedPaintText(res); got != "‹hello›" {
		t.Errorf("quotes paint text = %q, want %q", got, "‹hello›")
	}
}

// TestBehaviorTextWrapStyleAutoMatchesNormal: text-wrap-style auto wraps
// greedily exactly like normal wrapping, while pretty is rejected (a
// documented gap: the engine keeps greedy breaks there).
// Reference: Chrome 143.0.7499.40; balance cases live in
// inline_balance_test.go and are not repeated here.
func TestBehaviorTextWrapStyleAutoMatchesNormal(t *testing.T) {
	t.Parallel()

	const prose = "The quick brown fox jumps over the lazy dog"

	normal := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:112.5pt;font-size:12pt">`+prose+`</p></body></html>`)
	auto := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:112.5pt;font-size:12pt;text-wrap-style:auto">`+prose+`</p></body></html>`)

	normalLines := inlineLines(normal)
	autoLines := inlineLines(auto)

	if len(normalLines) != len(autoLines) {
		t.Fatalf("auto lines = %d, want normal %d", len(autoLines), len(normalLines))
	}

	for idx := range normalLines {
		if normalLines[idx].text != autoLines[idx].text {
			t.Errorf("line %d = %q, want normal %q", idx, autoLines[idx].text, normalLines[idx].text)
		}
	}

	// Documented gap: pretty is not accepted, so it stays greedy.
	if supportedDeclaration("text-wrap-style", "pretty") {
		t.Errorf("supportedDeclaration(text-wrap-style, pretty) = true, want false (documented gap)")
	}
}
