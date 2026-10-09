package layout

import (
	"math"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// Chrome 143.0.7499.40 references for text-wrap-style: balance, captured with
// temps/css-review/wrap/line_rects.js against
// temps/css-review/wrap/fixtures/balance-cases.html (Liberation Sans 16px,
// viewport 1024x768). Raw line rects live in
// temps/css-review/wrap/out/balance-cases.json. Expected widths below are the
// browser line widths in CSS pixels times 0.75, the engine's point-per-pixel
// factor, and are checked with a 0.6pt tolerance for font-metric rounding.

const (
	balanceProse = "The quick brown fox jumps over the lazy dog"
	balanceLong  = balanceProse + " again and again and again"
)

const balanceSheet = `
body { margin: 0; font-family: 'Liberation Sans', sans-serif; font-size: 16px; }
p { margin: 0; }
.w90 { width: 90px; }
.w150 { width: 150px; }
.w180 { width: 180px; }
.w220 { width: 220px; }
.w240 { width: 240px; }
.w260 { width: 260px; }
.w300 { width: 300px; }
.bal { text-wrap-style: balance; }
.nw { text-wrap-mode: nowrap; }
.ctr { text-align: center; }
b { font-weight: bold; }
`

// inlineLine is one visual line reconstructed from text ops that share a Y.
type inlineLine struct {
	text string
	x, w float64
	y    float64
}

// inlineLines groups text ops into visual lines (same Y within half a point)
// in paint order, so a style boundary inside one line joins back into the
// browser's line text.
func inlineLines(res *Result) []inlineLine {
	lines := make([]inlineLine, 0, len(res.Ops))

	for _, operation := range res.Ops {
		if operation.Kind != OpText || strings.TrimSpace(operation.Text) == "" {
			continue
		}

		text := strings.TrimSpace(operation.Text)

		if n := len(lines); n > 0 && math.Abs(lines[n-1].y-operation.Y) < 0.5 {
			lines[n-1].text += " " + text
			lines[n-1].w = operation.X + operation.W - lines[n-1].x

			continue
		}

		lines = append(lines, inlineLine{text: text, x: operation.X, w: operation.W, y: operation.Y})
	}

	return lines
}

func checkBalanceLines(t *testing.T, res *Result, want []inlineLine) {
	t.Helper()

	got := inlineLines(res)
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d: %+v", len(got), len(want), got)
	}

	for idx := range got {
		if got[idx].text != want[idx].text {
			t.Errorf("line %d text = %q, want %q", idx, got[idx].text, want[idx].text)
		}

		if math.Abs(got[idx].w-want[idx].w) > 0.6 {
			t.Errorf("line %d width = %.2fpt, want %.2fpt", idx, got[idx].w, want[idx].w)
		}
	}
}

// TestBalanceLineWidthBisection pins the bisection arithmetic on synthetic
// items so the algorithm is covered without font metrics: four 40pt items in a
// 130pt box wrap 3+1 normally, and balance re-breaks them 2+2 at 80.5pt.
func TestBalanceLineWidthBisection(t *testing.T) {
	t.Parallel()

	items := []inlineItem{
		{text: "a", w: 40},
		{text: "b", w: 40},
		{text: "c", w: 40},
		{text: "d", w: 40},
	}

	lines, widthSum, ok := countInlineLines(items, 130)
	if !ok || lines != 2 || !near(widthSum, 160) {
		t.Fatalf("countInlineLines(130) = (%d, %.2f, %v), want (2, 160, true)", lines, widthSum, ok)
	}

	if got := balanceLineWidth(items, 130, 1); !near(got, 80.5) {
		t.Fatalf("balanceLineWidth(130) = %.2f, want 80.5", got)
	}

	// One line needs no balancing; a width that cannot hold an item whole and
	// a forced break opt out.
	if got := balanceLineWidth(items, 200, 1); got != 0 {
		t.Fatalf("balanceLineWidth(200) = %.2f, want 0 (single line)", got)
	}

	if got := balanceLineWidth(items, 30, 1); got != 0 {
		t.Fatalf("balanceLineWidth(30) = %.2f, want 0 (item cannot fit)", got)
	}

	broken := []inlineItem{{text: "a", w: 40}, {forceBreak: true}, {text: "b", w: 40}}
	if got := balanceLineWidth(broken, 130, 1); got != 0 {
		t.Fatalf("balanceLineWidth with forced break = %.2f, want 0", got)
	}

	if got := inlineSegmentEnd(broken, 0); got != 1 {
		t.Fatalf("inlineSegmentEnd = %d, want 1", got)
	}
}

// balanceCase is one recorded Chrome line-break reference.
type balanceCase struct {
	name string
	page string
	want []inlineLine
}

// runBalanceCases lays each case out and checks it against the recorded
// Chrome line breaks.
func runBalanceCases(t *testing.T, sheetText *css.Stylesheet, cases []balanceCase) {
	t.Helper()

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html><body>`+testCase.page+`</body></html>`, sheetText)
			checkBalanceLines(t, res, testCase.want)
		})
	}
}

// TestTextWrapBalanceMatchesChromeLineBreaks pins balance against the
// recorded Chrome line breaks, including the 220px case whose balanced result
// differs from normal wrapping and the 150px case where balance reflows all
// three lines.
func TestTextWrapBalanceMatchesChromeLineBreaks(t *testing.T) {
	t.Parallel()

	runBalanceCases(t, sheet(t, balanceSheet), []balanceCase{
		{
			name: "normal wraps greedily at 220px",
			page: `<p class="w220">` + balanceProse + `</p>`,
			want: []inlineLine{
				{text: "The quick brown fox jumps", w: 142.73},
				{text: "over the lazy dog", w: 91.39},
			},
		},
		{
			name: "balance splits 220px differently from normal",
			page: `<p class="w220 bal">` + balanceProse + `</p>`,
			want: []inlineLine{
				{text: "The quick brown fox", w: 107.39},
				{text: "jumps over the lazy dog", w: 126.74},
			},
		},
		{
			name: "balance reflows three lines at 150px",
			page: `<p class="w150 bal">` + balanceProse + `</p>`,
			want: []inlineLine{
				{text: "The quick brown", w: 88.04},
				{text: "fox jumps over", w: 78.04},
				{text: "the lazy dog", w: 64.71},
			},
		},
		{
			name: "balance keeps the two-line split at 180px",
			page: `<p class="w180 bal">` + balanceProse + `</p>`,
			want: []inlineLine{
				{text: "The quick brown fox", w: 107.39},
				{text: "jumps over the lazy dog", w: 126.74},
			},
		},
	})
}

// TestTextWrapBalanceMatchesChromeWideWidths: 260px and 300px keep the same
// balanced split as 220px in Chrome.
func TestTextWrapBalanceMatchesChromeWideWidths(t *testing.T) {
	t.Parallel()

	runBalanceCases(t, sheet(t, balanceSheet), []balanceCase{
		{
			name: "balance keeps the two-line split at 260px",
			page: `<p class="w260 bal">` + balanceProse + `</p>`,
			want: []inlineLine{
				{text: "The quick brown fox", w: 107.39},
				{text: "jumps over the lazy dog", w: 126.74},
			},
		},
		{
			name: "balance keeps the two-line split at 300px",
			page: `<p class="w300 bal">` + balanceProse + `</p>`,
			want: []inlineLine{
				{text: "The quick brown fox", w: 107.39},
				{text: "jumps over the lazy dog", w: 126.74},
			},
		},
	})
}

// TestTextWrapBalanceMatchesChromeEdgeCases: forced breaks, mixed inline
// styles, the text-wrap shorthand, and stable greedy wrapping.
func TestTextWrapBalanceMatchesChromeEdgeCases(t *testing.T) {
	t.Parallel()

	runBalanceCases(t, sheet(t, balanceSheet), []balanceCase{
		{
			name: "balance applies per forced-break segment at 150px",
			page: `<p class="w150 bal">The quick brown fox jumps<br>over the lazy dog</p>`,
			want: []inlineLine{
				{text: "The quick", w: 52.04},
				{text: "brown fox jumps", w: 87.38},
				{text: "over the lazy dog", w: 91.40},
			},
		},
		{
			name: "balance mixed inline styles at 240px",
			page: `<p class="w240 bal">Alpha <b>Beta</b> gamma delta epsilon zeta eta theta</p>`,
			want: []inlineLine{
				{text: "Alpha Beta gamma delta", w: 132.75},
				{text: "epsilon zeta eta theta", w: 114.08},
			},
		},
		{
			name: "balance from the text-wrap shorthand at 220px",
			page: `<p class="w220" style="text-wrap: balance">` + balanceProse + `</p>`,
			want: []inlineLine{
				{text: "The quick brown fox", w: 107.39},
				{text: "jumps over the lazy dog", w: 126.74},
			},
		},
		{
			// Chrome 143 treats stable as greedy wrap; probe2.html #stable at
			// 300px records "The quick brown fox jumps over the lazy" / "dog".
			name: "stable wraps greedily at 300px",
			page: `<p class="w300" style="text-wrap-style: stable">` + balanceProse + `</p>`,
			want: []inlineLine{
				{text: "The quick brown fox jumps over the lazy", w: 214.10},
				{text: "dog", w: 20.02},
			},
		},
	})
}

// TestTextWrapBalanceLeavesLongParagraphAlone: Chrome does not balance past
// six lines; the 90px fixture wraps seven lines and balance must equal normal.
func TestTextWrapBalanceLeavesLongParagraphAlone(t *testing.T) {
	t.Parallel()

	sheetText := sheet(t, balanceSheet)

	normal := layoutHTML(t, `<html><body><p class="w90">`+balanceLong+`</p></body></html>`, sheetText)
	balanced := layoutHTML(t, `<html><body><p class="w90 bal">`+balanceLong+`</p></body></html>`, sheetText)

	normalLines := inlineLines(normal)
	balancedLines := inlineLines(balanced)

	if len(normalLines) != 7 {
		t.Fatalf("normal lines = %d, want 7 (Chrome reference)", len(normalLines))
	}

	if len(balancedLines) != len(normalLines) {
		t.Fatalf("balanced lines = %d, want %d", len(balancedLines), len(normalLines))
	}

	for idx := range normalLines {
		if normalLines[idx].text != balancedLines[idx].text {
			t.Errorf("line %d text = %q, want normal %q", idx, balancedLines[idx].text, normalLines[idx].text)
		}

		if math.Abs(normalLines[idx].w-balancedLines[idx].w) > 0.01 {
			t.Errorf("line %d width = %.2fpt, want normal %.2fpt", idx, balancedLines[idx].w, normalLines[idx].w)
		}
	}
}

// TestTextWrapBalanceLeavesNowrapAlone: text-wrap-mode:nowrap keeps the whole
// run on one overflowing line in Chrome, and balance must not break it.
func TestTextWrapBalanceLeavesNowrapAlone(t *testing.T) {
	t.Parallel()

	sheetText := sheet(t, balanceSheet)

	res := layoutHTML(
		t,
		`<html><body><p class="w300 nw bal">`+balanceProse+`</p></body></html>`,
		sheetText,
	)

	checkBalanceLines(t, res, []inlineLine{
		{text: balanceProse, w: 237.46},
	})
}

// TestTextWrapBalanceCenterAlignmentUsesFullWidth: Chrome narrows only the
// line breaker for balance; center alignment still centers each line in the
// full 300px block (recorded centers 150px for both lines).
func TestTextWrapBalanceCenterAlignmentUsesFullWidth(t *testing.T) {
	t.Parallel()

	sheetText := sheet(t, balanceSheet)

	res := layoutHTML(
		t,
		`<html><body><p class="w300 ctr bal">`+balanceProse+`</p></body></html>`,
		sheetText,
	)

	lines := inlineLines(res)
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2: %+v", len(lines), lines)
	}

	checkBalanceLines(t, res, []inlineLine{
		{text: "The quick brown fox", w: 107.39},
		{text: "jumps over the lazy dog", w: 126.74},
	})

	const wantCenter = 112.5 // 225pt content width / 2

	for idx, line := range lines {
		if center := line.x + line.w/2; math.Abs(center-wantCenter) > 0.5 {
			t.Errorf("line %d center = %.2fpt, want %.2fpt", idx, center, wantCenter)
		}
	}
}

// TestTextWrapStyleAcceptanceGate: feature queries follow the shipped
// behavior. auto, balance, and stable are accepted (stable is greedy wrap);
// pretty and avoid-short-last-line are rejected because the engine does not
// break them differently from normal wrapping.
func TestTextWrapStyleAcceptanceGate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		prop, value string
		want        bool
	}{
		{"text-wrap-style", "auto", true},
		{"text-wrap-style", "balance", true},
		{"text-wrap-style", "stable", true},
		{"text-wrap-style", "pretty", false},
		{"text-wrap-style", "avoid-short-last-line", false},
		{"text-wrap-style", "bogus", false},
		{"text-wrap", "balance", true},
		{"text-wrap", "wrap balance", true},
		{"text-wrap", "balance wrap", true},
		{"text-wrap", "nowrap", true},
		{"text-wrap", "pretty", false},
		{"text-wrap", textWrapStyleBalance + " " + textWrapStyleBalance, false},
		{"text-wrap", "bogus", false},
	}

	for _, testCase := range cases {
		if got := supportedDeclaration(testCase.prop, testCase.value); got != testCase.want {
			t.Errorf("supportedDeclaration(%q, %q) = %v, want %v",
				testCase.prop, testCase.value, got, testCase.want)
		}

		if got := engineSupportsProperty(testCase.prop, testCase.value); got != testCase.want {
			t.Errorf("engineSupportsProperty(%q, %q) = %v, want %v",
				testCase.prop, testCase.value, got, testCase.want)
		}
	}
}
