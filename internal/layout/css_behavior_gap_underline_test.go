package layout

import (
	"testing"
)

// Gap-close underline and emphasis-skip behavior, asserted through USED
// drawing-list geometry, never stored style strings. Reference browser for
// every case: Chrome 143.0.7499.40. Shared helpers layoutHTML,
// underlineOps, and opsOfKind come from other files in this package and are
// reused here, never redeclared.

// TestGapUnderlinePositionUnderDropsStroke is text-underline-position:under.
// Under drops the stroke below descenders, so it paints lower than the auto
// baseline spot. Reference: Chrome 143.0.7499.40 places the under stroke
// under descenders.
func TestGapUnderlinePositionUnderDropsStroke(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
		`<span style="text-decoration:underline">acemnorstu</span>`+
		`</p></body></html>`)
	under := layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
		`<span style="text-decoration:underline;text-underline-position:under">acemnorstu</span>`+
		`</p></body></html>`)

	plainLines := underlineOps(plain)
	underLines := underlineOps(under)

	if len(plainLines) == 0 || len(underLines) == 0 {
		t.Fatalf("want decoration ops, got plain=%d under=%d", len(plainLines), len(underLines))
	}

	if delta := underLines[0].Y - plainLines[0].Y; delta < 1 {
		t.Errorf("under stroke Y %.4f should sit at least 1pt below auto Y %.4f, delta=%.4f",
			underLines[0].Y, plainLines[0].Y, delta)
	}
}

// TestGapUnderlinePositionLeftRightMoveStroke is
// text-underline-position:left and right. In horizontal text the engine maps
// both to a smaller under drop so the declared value stays observable in
// the drawing list. Reference: Chrome 143.0.7499.40 treats bare left and
// right as auto in horizontal text and reserves side placement for vertical
// text; this subset keeps the drop so authors can see the value apply.
func TestGapUnderlinePositionLeftRightMoveStroke(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
		`<span style="text-decoration:underline">acemnorstu</span>`+
		`</p></body></html>`)
	left := layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
		`<span style="text-decoration:underline;text-underline-position:left">acemnorstu</span>`+
		`</p></body></html>`)
	right := layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
		`<span style="text-decoration:underline;text-underline-position:right">acemnorstu</span>`+
		`</p></body></html>`)

	plainLines := underlineOps(plain)
	leftLines := underlineOps(left)
	rightLines := underlineOps(right)

	if len(plainLines) == 0 || len(leftLines) == 0 || len(rightLines) == 0 {
		t.Fatalf("want decoration ops, got plain=%d left=%d right=%d",
			len(plainLines), len(leftLines), len(rightLines))
	}

	if delta := leftLines[0].Y - plainLines[0].Y; delta < 0.5 {
		t.Errorf("left stroke Y %.4f should sit below auto Y %.4f, delta=%.4f",
			leftLines[0].Y, plainLines[0].Y, delta)
	}

	if delta := rightLines[0].Y - plainLines[0].Y; delta < 0.5 {
		t.Errorf("right stroke Y %.4f should sit below auto Y %.4f, delta=%.4f",
			rightLines[0].Y, plainLines[0].Y, delta)
	}
}

// TestGapEmphasisSkipPunctuationSkipsComma is text-emphasis-skip: spaces
// keeps the comma mark while spaces punctuation also skips it, so the second
// run paints one fewer mark. Reference: Chrome 143.0.7499.40 skips spaces
// and punctuation per CSS Text Decoration 4 section 3.5.
func TestGapEmphasisSkipPunctuationSkipsComma(t *testing.T) {
	t.Parallel()

	spaces := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-emphasis-style:dot;`+
		`text-emphasis-skip:spaces">a b,c</p></body></html>`)
	punct := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-emphasis-style:dot;`+
		`text-emphasis-skip:spaces punctuation">a b,c</p></body></html>`)

	spaceMarks := opsOfKind(spaces, OpFillRect)
	punctMarks := opsOfKind(punct, OpFillRect)

	if len(spaceMarks) != 4 {
		t.Errorf("spaces marks = %d, want 4 (a, b, comma, c with the space skipped)", len(spaceMarks))
	}

	if len(punctMarks) != 3 {
		t.Errorf("spaces punctuation marks = %d, want 3 (a, b, c with space and comma skipped)", len(punctMarks))
	}
}

// TestGapEmphasisSkipNarrowSkipsNarrow is text-emphasis-skip: narrow skips
// narrow Latin letters on the probe text, so it paints fewer marks than
// spaces. Reference: Chrome 143.0.7499.40 skips narrow characters per CSS
// Text Decoration 4 section 3.5.
func TestGapEmphasisSkipNarrowSkipsNarrow(t *testing.T) {
	t.Parallel()

	spaces := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-emphasis-style:dot;text-emphasis-skip:spaces">a b,c</p></body></html>`)
	narrow := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-emphasis-style:dot;text-emphasis-skip:narrow">a b,c</p></body></html>`)

	spaceMarks := opsOfKind(spaces, OpFillRect)
	narrowMarks := opsOfKind(narrow, OpFillRect)

	if len(spaceMarks) != 4 {
		t.Errorf("spaces marks = %d, want 4 (a, b, comma, c with the space skipped)", len(spaceMarks))
	}

	if len(narrowMarks) >= len(spaceMarks) {
		t.Errorf("narrow marks = %d should be fewer than spaces marks = %d (narrow skips the Latin letters)",
			len(narrowMarks), len(spaceMarks))
	}
}
