package layout

import (
	"math"
	"strings"
	"testing"
)

// firstLineTextOps returns the text ops on the first visual line (same Y
// within half a point) in paint order, skipping whitespace-only runs.
func firstLineTextOps(res *Result) []Op {
	var line []Op

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || strings.TrimSpace(paintOp.Text) == "" {
			continue
		}

		if len(line) == 0 {
			line = append(line, paintOp)

			continue
		}

		if math.Abs(paintOp.Y-line[0].Y) < 0.5 {
			line = append(line, paintOp)

			continue
		}

		break
	}

	return line
}

// lineWidth spans the first to the last op of a visual line.
func lineWidth(line []Op) float64 {
	if len(line) == 0 {
		return 0
	}

	last := line[len(line)-1]

	return last.X + last.W - line[0].X
}

// maxWordGap is the largest X gap between consecutive word ops on a line.
// Word ops carry their trailing space in W, so the gap is the extra
// justification space (0 when not justifying).
func maxWordGap(line []Op) float64 {
	gap := 0.0

	for i := 1; i < len(line); i++ {
		if g := line[i].X - (line[i-1].X + line[i-1].W); g > gap {
			gap = g
		}
	}

	return gap
}

// behaviorJustifyDoc wraps the shared paragraph in a justified 220pt
// box with the given extra paragraph style.
func behaviorJustifyDoc(extra string) string {
	return `<html><body style="margin:0"><p style="margin:0;width:220pt;font-size:12pt;text-align:justify;` +
		extra + `">` + justifyPara + `</p></body></html>`
}

const justifyPara = "alpha bravo charlie delta echo foxtrot golf hotel india juliet kilo lorem ipsum"

// TestBehaviorTextJustifyNoneDisablesExpansion: text-align:justify stretches
// the first line to fill its box, while text-justify:none leaves it
// start-aligned with its natural width.
// Reference: Chrome 143.0.7499.40, text-justify none.
func TestBehaviorTextJustifyNoneDisablesExpansion(t *testing.T) {
	t.Parallel()

	base := behaviorJustifyDoc(``)
	none := behaviorJustifyDoc(`;text-justify:none`)

	justLine := firstLineTextOps(layoutHTML(t, base))
	noneLine := firstLineTextOps(layoutHTML(t, none))

	if len(justLine) < 2 {
		t.Fatalf("justified first line ops = %d, want at least 2 (per-word ops)", len(justLine))
	}

	if got := maxWordGap(justLine); got < 0.5 {
		t.Errorf("justified word gap = %.3fpt, want at least 0.5pt of expansion", got)
	}

	justW, noneW := lineWidth(justLine), lineWidth(noneLine)
	if justW < noneW+2 {
		t.Errorf("justified width %.3fpt should exceed unjustified %.3fpt by 2pt", justW, noneW)
	}
}

// TestBehaviorTextJustifyInterWordExpandsWordGaps: an explicit
// text-justify:inter-word justifies exactly like the default, expanding
// inter-word gaps while keeping glyph advances unchanged.
// Reference: Chrome 143.0.7499.40, text-justify inter-word.
func TestBehaviorTextJustifyInterWordExpandsWordGaps(t *testing.T) {
	t.Parallel()

	interWord := behaviorJustifyDoc(`;text-justify:inter-word`)
	none := behaviorJustifyDoc(`;text-justify:none`)

	wordLine := firstLineTextOps(layoutHTML(t, interWord))
	noneLine := firstLineTextOps(layoutHTML(t, none))

	if len(wordLine) < 2 {
		t.Fatalf("inter-word first line ops = %d, want at least 2", len(wordLine))
	}

	if got := maxWordGap(wordLine); got < 0.5 {
		t.Errorf("inter-word gap = %.3fpt, want at least 0.5pt", got)
	}

	if wordW, noneW := lineWidth(wordLine), lineWidth(noneLine); wordW < noneW+2 {
		t.Errorf("inter-word width %.3fpt should exceed none %.3fpt by 2pt", wordW, noneW)
	}
}

// TestBehaviorTextJustifyInterCharacterExpandsWithinWord: with
// text-justify:inter-character the slack spreads between every character as
// extra letter spacing, so the first word op measures wider than its
// inter-word twin while the extra gap between word ops stays near zero.
// Reference: Chrome 143.0.7499.40, text-justify inter-character.
func TestBehaviorTextJustifyInterCharacterExpandsWithinWord(t *testing.T) {
	t.Parallel()

	interWord := behaviorJustifyDoc(`;text-justify:inter-word`)
	interChar := behaviorJustifyDoc(`;text-justify:inter-character`)

	wordLine := firstLineTextOps(layoutHTML(t, interWord))
	charLine := firstLineTextOps(layoutHTML(t, interChar))

	if len(wordLine) < 2 || len(charLine) < 2 {
		t.Fatalf("first line ops inter-word=%d inter-character=%d, want at least 2 each",
			len(wordLine), len(charLine))
	}

	firstWord := strings.TrimSpace(wordLine[0].Text)
	firstChar := strings.TrimSpace(charLine[0].Text)

	if firstWord != firstChar {
		t.Fatalf("first word %q vs %q, want same wrapping", firstWord, firstChar)
	}

	if charLine[0].W < wordLine[0].W+0.5 {
		t.Errorf("inter-character W %.3fpt should exceed inter-word W %.3fpt by 0.5pt",
			charLine[0].W, wordLine[0].W)
	}

	if wordGap, charGap := maxWordGap(wordLine), maxWordGap(charLine); wordGap < charGap+0.5 {
		t.Errorf("inter-word gap %.3fpt should exceed inter-character gap %.3fpt by 0.5pt",
			wordGap, charGap)
	}
}

// firstPreText returns the first text op, keeping its preserved spaces.
func firstPreText(t *testing.T, res *Result) string {
	t.Helper()

	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text != "" {
			return op.Text
		}
	}

	t.Fatal("no text op")

	return ""
}

// TestBehaviorWhiteSpaceTrimDiscardBeforeTrimsLeading: with white-space:pre
// preserved, discard-before drops leading spaces while the untrimmed run
// keeps them.
// Reference: Chrome 143.0.7499.40, white-space-trim discard-before.
func TestBehaviorWhiteSpaceTrimDiscardBeforeTrimsLeading(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<pre style="margin:0;font-size:12pt;white-space:pre">   hello</pre></body></html>`)
	trimmed := layoutHTML(t, `<html><body style="margin:0">`+
		`<pre style="margin:0;font-size:12pt;white-space:pre;white-space-trim:discard-before">   hello</pre></body></html>`)

	plainText := firstPreText(t, plain)
	trimText := firstPreText(t, trimmed)

	if !strings.HasPrefix(plainText, " ") {
		t.Errorf("untrimmed pre text %q should keep leading spaces", plainText)
	}

	if strings.HasPrefix(trimText, " ") {
		t.Errorf("discard-before text %q should lose leading spaces", trimText)
	}

	if strings.TrimSpace(trimText) != "hello" {
		t.Errorf("discard-before text %q, want %q", trimText, "hello")
	}
}

// TestBehaviorWhiteSpaceTrimDiscardAfterTrimsTrailing: discard-after drops
// trailing tabs from a preserved pre run. Tabs are used because the line
// box already drops trailing ASCII spaces (trimTrailingSpace), while tabs
// survive to paint without the trim.
// Reference: Chrome 143.0.7499.40, white-space-trim discard-after.
func TestBehaviorWhiteSpaceTrimDiscardAfterTrimsTrailing(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<pre style="margin:0;font-size:12pt;white-space:pre">hello`+"\t\t"+`</pre></body></html>`)
	trimmed := layoutHTML(t, `<html><body style="margin:0">`+
		`<pre style="margin:0;font-size:12pt;white-space:pre;`+
		`white-space-trim:discard-after">hello`+"\t\t"+`</pre></body></html>`)

	plainText := firstPreText(t, plain)
	trimText := firstPreText(t, trimmed)

	if !strings.HasSuffix(plainText, "\t") {
		t.Errorf("untrimmed pre text %q should keep trailing tabs", plainText)
	}

	if strings.HasSuffix(trimText, "\t") || strings.HasSuffix(trimText, " ") {
		t.Errorf("discard-after text %q should lose trailing tabs", trimText)
	}

	if strings.TrimSpace(trimText) != "hello" {
		t.Errorf("discard-after text %q, want %q", trimText, "hello")
	}
}

// TestBehaviorWhiteSpaceTrimDiscardInnerTrimsBothEdges: discard-inner drops
// both edges of a preserved run while keeping inner spacing intact. Trailing
// tabs prove the end trim because the line box already drops trailing ASCII
// spaces on its own.
// Reference: Chrome 143.0.7499.40, white-space-trim discard-inner.
func TestBehaviorWhiteSpaceTrimDiscardInnerTrimsBothEdges(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<pre style="margin:0;font-size:12pt;white-space:pre">   a   b`+"\t\t"+`</pre></body></html>`)
	trimmed := layoutHTML(t, `<html><body style="margin:0">`+
		`<pre style="margin:0;font-size:12pt;white-space:pre;`+
		`white-space-trim:discard-inner">   a   b`+"\t\t"+`</pre></body></html>`)

	plainText := firstPreText(t, plain)
	trimText := firstPreText(t, trimmed)

	if plainText != "   a   b\t\t" {
		t.Errorf("untrimmed pre text %q, want %q", plainText, "   a   b\t\t")
	}

	if trimText != "a   b" {
		t.Errorf("discard-inner text %q, want %q", trimText, "a   b")
	}
}
