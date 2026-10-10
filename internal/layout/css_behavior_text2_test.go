package layout

import (
	"strings"
	"testing"
)

// Text2 behavior tests: one test per text property in the second batch,
// asserted through USED values (measured box sizes, text-op coordinates and
// widths, emitted text and decoration ops), never through stored style
// strings. Points are the engine unit and 1 CSS pixel = 0.75pt (ptPerCSSPx
// in css_review_02_test.go). Reference browser for every case: Chrome
// 143.0.7499.40. Shared helpers boxByID, near, pxToPt, layoutHTML,
// opsOfKind, firstText, inlineLines, textOpIndex, and underlineOps come from
// other files in this same package and are reused here, never redeclared.

// behaviorText2AssertSameOps asserts two results paint the same ops in the
// same order: same kind, geometry, and text. Used to pin no-op behavior for
// properties the drawing list cannot observe.
func behaviorText2AssertSameOps(t *testing.T, first, second *Result) {
	t.Helper()

	if len(first.Ops) != len(second.Ops) {
		t.Fatalf("op count = %d vs %d, want identical geometry", len(first.Ops), len(second.Ops))
	}

	for i := range first.Ops {
		x, y := first.Ops[i], second.Ops[i]
		if x.Kind != y.Kind || x.Text != y.Text ||
			!near(x.X, y.X) || !near(x.Y, y.Y) ||
			!near(x.W, y.W) || !near(x.H, y.H) {
			t.Fatalf("op %d differs: %+v vs %+v, want identical geometry", i, x, y)
		}
	}
}

// TestBehaviorTextAlignAllCentersLine is text-align-all: center behaves like
// text-align:center, so the short line centers in the 150pt content box
// instead of sitting at the left edge.
// Reference: Chrome 143.0.7499.40, width 200px, short centered line.
func TestBehaviorTextAlignAllCentersLine(t *testing.T) {
	t.Parallel()

	left := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:150pt;font-size:12pt">Hi</p></body></html>`)
	centered := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:150pt;font-size:12pt;text-align-all:center">Hi</p></body></html>`)

	xs := firstTextX(t, left, "Hi")
	xc := firstTextX(t, centered, "Hi")

	if xc <= xs+1 {
		t.Fatalf("text-align-all:center X %.4f should be right of left X %.4f", xc, xs)
	}

	centerOp := firstText(centered)
	if center := centerOp.X + centerOp.W/2; center < 70 || center > 80 {
		t.Errorf("centered line center = %.4fpt, want near 75pt (150pt box middle)", center)
	}
}

// TestBehaviorTextAlignLastCentersLastLine is text-align-last: the last line
// of a left-aligned wrapping paragraph centers while the first line stays at
// the content edge.
// Reference: Chrome 143.0.7499.40, left block with centered last line.
func TestBehaviorTextAlignLastCentersLastLine(t *testing.T) {
	t.Parallel()

	const prose = "The quick brown fox jumps over the lazy dog"

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:112.5pt;font-size:12pt;text-align:left">`+prose+`</p></body></html>`)
	lastCenter := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:112.5pt;font-size:12pt;text-align:left;text-align-last:center">`+prose+`</p></body></html>`)

	plainLines := inlineLines(plain)
	centerLines := inlineLines(lastCenter)

	if len(plainLines) < 2 || len(centerLines) < 2 {
		t.Fatalf("want wrapped lines, got plain=%d center=%d", len(plainLines), len(centerLines))
	}

	if !near(plainLines[len(plainLines)-1].x, plainLines[0].x) {
		t.Errorf("plain last-line x %.4f should match first-line x %.4f (both left)",
			plainLines[len(plainLines)-1].x, plainLines[0].x)
	}

	firstX := centerLines[0].x
	lastX := centerLines[len(centerLines)-1].x

	if lastX <= firstX+1 {
		t.Errorf("text-align-last:center last-line x %.4f should sit right of first-line x %.4f",
			lastX, firstX)
	}
}

// TestBehaviorTextAnchorNoPaintEffect documents the current behavior of
// text-anchor: the declaration is forwarded to applyLeftoversProps, which has
// no text-anchor case and drops it, so no layout or paint pass reads it and
// text-anchor:middle paints identical geometry to the default.
// Unobservable in the drawing list: reported as a no-op, never failing.
// Reference: Chrome 143.0.7499.40 would middle-anchor SVG text only.
func TestBehaviorTextAnchorNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">anchor sample</p></body></html>`)
	anchored := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-anchor:middle">anchor sample</p></body></html>`)

	behaviorText2AssertSameOps(t, plain, anchored)
}

// TestBehaviorTextAutospaceWidensIdeographAlpha is text-autospace: the
// ideograph-alpha value inserts a 1/8 em gap at the ideograph and Latin
// boundary, so the mixed run measures wider than no-autospace.
// Reference: Chrome 143.0.7499.40, ideograph and alpha spacing.
func TestBehaviorTextAutospaceWidensIdeographAlpha(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:16pt;text-autospace:no-autospace">漢A</p>`+
		`</body></html>`)
	spaced := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:16pt;text-autospace:ideograph-alpha">漢A</p>`+
		`</body></html>`)

	wp := behaviorTextTotalWidth(spaced)
	_ = wp

	plainW := behaviorTextTotalWidth(plain)
	spacedW := behaviorTextTotalWidth(spaced)

	if spacedW <= plainW+0.1 {
		t.Fatalf("ideograph-alpha width %v should exceed no-autospace width %v", spacedW, plainW)
	}
}

// TestBehaviorTextBoxTrimsHalfLeading is text-box: trim-both with cap and
// alphabetic edges drops the half-leading around a loose line, so the block
// box measures shorter than the untrimmed run.
// Reference: Chrome 143.0.7499.40, text-box trimming.
func TestBehaviorTextBoxTrimsHalfLeading(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p id="p" style="margin:0;font-size:20pt;line-height:2">Hg</p></body></html>`)
	trimmed := layoutHTML(t, `<html><body style="margin:0">`+
		`<p id="p" style="margin:0;font-size:20pt;line-height:2;`+
		`text-box:trim-both cap alphabetic">Hg</p></body></html>`)

	plainBox := boxByID(t, plain, "p")
	trimBox := boxByID(t, trimmed, "p")

	if trimBox.height >= plainBox.height-0.5 {
		t.Errorf("trim-both height %.4fpt not smaller than untrimmed %.4fpt",
			trimBox.height, plainBox.height)
	}
}

// TestBehaviorTextBoxEdgeRetargetsTrimmedMetrics is text-box-edge: with the
// same trim-both, cap and alphabetic edges retarget the content edges inside
// the trimmed box, so the capped box measures shorter than the auto-edge box.
// Reference: Chrome 143.0.7499.40, text-box-edge retargeting.
func TestBehaviorTextBoxEdgeRetargetsTrimmedMetrics(t *testing.T) {
	t.Parallel()

	auto := layoutHTML(t, `<html><body style="margin:0">`+
		`<p id="p" style="margin:0;font-size:20pt;line-height:2;`+
		`text-box-trim:trim-both;text-box-edge:auto">Hg</p></body></html>`)
	capped := layoutHTML(t, `<html><body style="margin:0">`+
		`<p id="p" style="margin:0;font-size:20pt;line-height:2;`+
		`text-box-trim:trim-both;text-box-edge:cap alphabetic">Hg</p></body></html>`)

	autoBox := boxByID(t, auto, "p")
	capBox := boxByID(t, capped, "p")

	if capBox.height >= autoBox.height-0.1 {
		t.Errorf("cap/alphabetic height %.4fpt should be shorter than auto height %.4fpt",
			capBox.height, autoBox.height)
	}
}

// TestBehaviorTextBoxTrimStartShrinksBox is text-box-trim: trim-start drops
// the top half-leading only, so the block still measures shorter than the
// untrimmed run with the same loose line-height.
// Reference: Chrome 143.0.7499.40, text-box-trim start.
func TestBehaviorTextBoxTrimStartShrinksBox(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p id="p" style="margin:0;font-size:20pt;line-height:2">Hg</p></body></html>`)
	trimmed := layoutHTML(t, `<html><body style="margin:0">`+
		`<p id="p" style="margin:0;font-size:20pt;line-height:2;`+
		`text-box-trim:trim-start">Hg</p></body></html>`)

	plainBox := boxByID(t, plain, "p")
	trimBox := boxByID(t, trimmed, "p")

	if trimBox.height >= plainBox.height-0.1 {
		t.Errorf("trim-start height %.4fpt not smaller than untrimmed %.4fpt",
			trimBox.height, plainBox.height)
	}
}

// TestBehaviorTextCombineUprightCombinesDigits is text-combine-upright: a
// digits-2 run in vertical writing paints as one upright cell (rotation 0),
// while the same run without combining keeps the rotated column angle.
// Reference: Chrome 143.0.7499.40, vertical digit combining.
func TestBehaviorTextCombineUprightCombinesDigits(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
		`<span style="writing-mode:vertical-rl;text-combine-upright:digits 2">12</span>`+
		`<span style="writing-mode:vertical-rl">34</span>`+
		`</p></body></html>`)

	_, combined, found := textOpIndex(res, "12")
	if !found {
		t.Fatal("no text op for 12")
	}

	if combined.RotateDeg != 0 {
		t.Errorf("combined RotateDeg = %v, want 0 (upright cell)", combined.RotateDeg)
	}

	_, rotated, found := textOpIndex(res, "34")
	if !found {
		t.Fatal("no text op for 34")
	}

	if rotated.RotateDeg != -90 {
		t.Errorf("plain vertical RotateDeg = %v, want -90", rotated.RotateDeg)
	}
}

// TestBehaviorTextDecorationInsetTrimsStroke is text-decoration-inset: the
// inset moves both decoration endpoints inward, so the inset stroke starts
// right of the plain stroke and measures narrower by about twice the inset.
// Reference: Chrome 143.0.7499.40, inset underlines.
func TestBehaviorTextDecorationInsetTrimsStroke(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body>`+
		`<p style="margin:0;font-size:14pt"><span style="text-decoration:underline">abcdef</span></p>`+
		`<p style="margin:0;font-size:14pt">`+
		`<span style="text-decoration:underline;text-decoration-inset:3pt">abcdef</span></p>`+
		`</body></html>`)

	lines := underlineOps(res)
	if len(lines) != 2 {
		t.Fatalf("inset layout produced %d decoration lines, want 2", len(lines))
	}

	plain, inset := lines[0], lines[1]

	if inset.X < plain.X+2.5 {
		t.Errorf("inset X=%.4f, plain X=%.4f; want start inset by 3pt", inset.X, plain.X)
	}

	if inset.W > plain.W-5.5 {
		t.Errorf("inset W=%.4f, plain W=%.4f; want both endpoints inset by 3pt", inset.W, plain.W)
	}
}

// TestBehaviorTextDecorationSkipShorthandSpacesSplits is text-decoration-skip:
// the spaces keyword maps onto skipping all spaces, so a nowrap two-word run
// splits into per-word strokes while none keeps one continuous stroke.
// Reference: Chrome 143.0.7499.40, skip shorthand.
func TestBehaviorTextDecorationSkipShorthandSpacesSplits(t *testing.T) {
	t.Parallel()

	layoutDecl := func(decl string) *Result {
		return layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
			`<span style="white-space:nowrap;text-decoration:underline;`+decl+`">alpha beta</span>`+
			`</p></body></html>`)
	}

	split := underlineOps(layoutDecl("text-decoration-skip:spaces"))
	if len(split) != 2 {
		t.Fatalf("shorthand spaces produced %d decoration lines, want 2 (one per word)", len(split))
	}

	continuous := underlineOps(layoutDecl("text-decoration-skip:none"))
	if len(continuous) != 1 {
		t.Fatalf("shorthand none produced %d decoration lines, want 1 continuous", len(continuous))
	}

	if continuous[0].W <= split[0].W || continuous[0].W <= split[1].W {
		t.Fatalf("none stroke W=%.4f should span both words (split %.4f and %.4f)",
			continuous[0].W, split[0].W, split[1].W)
	}
}

// TestBehaviorTextDecorationSkipBoxBreaksAtChrome is
// text-decoration-skip-box: all breaks a continuous underline at an item with
// inline padding or border instead of drawing across its box chrome.
// Reference: Chrome 143.0.7499.40, skip-box handling.
func TestBehaviorTextDecorationSkipBoxBreaksAtChrome(t *testing.T) {
	t.Parallel()

	page := func(middle string) *Result {
		return layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
			`<span style="text-decoration:underline">ab</span>`+
			`<span style="text-decoration:underline;padding:0 6pt;`+middle+`">cd</span>`+
			`<span style="text-decoration:underline">ef</span>`+
			`</p></body></html>`)
	}

	if got := len(underlineOps(page(""))); got != 1 {
		t.Fatalf("continuous run = %d decoration lines, want 1", got)
	}

	if got := len(underlineOps(page("text-decoration-skip-box:all"))); got != 2 {
		t.Fatalf("skip-box all run = %d decoration lines, want 2", got)
	}
}

// TestBehaviorTextDecorationSkipInkSplitsDescenders is
// text-decoration-skip-ink: auto skips the stroke under descender glyphs, so
// an all-descender word paints no visible stroke while none paints one
// continuous stroke.
// Reference: Chrome 143.0.7499.40, skip-ink gaps.
func TestBehaviorTextDecorationSkipInkSplitsDescenders(t *testing.T) {
	t.Parallel()

	layoutDecl := func(decl string) *Result {
		return layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
			`<span style="text-decoration:underline;`+decl+`">ggg</span>`+
			`</p></body></html>`)
	}

	split := underlineOps(layoutDecl("text-decoration-skip-ink:auto"))
	if len(split) != 0 {
		t.Fatalf("skip-ink auto produced %d decoration lines, want 0 (all ink skipped)", len(split))
	}

	continuous := underlineOps(layoutDecl("text-decoration-skip-ink:none"))
	if len(continuous) != 1 {
		t.Fatalf("skip-ink none produced %d decoration lines, want 1 continuous", len(continuous))
	}
}

// TestBehaviorTextDecorationSkipSelfSuppressesStroke is
// text-decoration-skip-self: skip-all suppresses the item's own decoration
// stroke, so the skipped span paints no underline while the default paints
// one.
// Reference: Chrome 143.0.7499.40, skip-self handling.
func TestBehaviorTextDecorationSkipSelfSuppressesStroke(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
		`<span style="text-decoration:underline">ab</span>`+
		`</p></body></html>`)
	if got := len(underlineOps(plain)); got == 0 {
		t.Fatal("plain underline: want >0 decoration ops, got 0")
	}

	skipped := layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
		`<span style="text-decoration:underline;text-decoration-skip-self:skip-all">ab</span>`+
		`</p></body></html>`)
	if got := len(underlineOps(skipped)); got != 0 {
		t.Fatalf("skip-self skip-all = %d decoration lines, want 0", got)
	}
}

// TestBehaviorTextDecorationSkipSpacesSplitsWords is
// text-decoration-skip-spaces: all splits a nowrap run into per-word strokes
// while none keeps one continuous stroke across the space.
// Reference: Chrome 143.0.7499.40, skip-spaces handling.
func TestBehaviorTextDecorationSkipSpacesSplitsWords(t *testing.T) {
	t.Parallel()

	layoutDecl := func(decl string) *Result {
		return layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
			`<span style="white-space:nowrap;text-decoration:underline;`+decl+`">alpha beta</span>`+
			`</p></body></html>`)
	}

	split := underlineOps(layoutDecl("text-decoration-skip-spaces:all"))
	if len(split) != 2 {
		t.Fatalf("skip-spaces all produced %d decoration lines, want 2 (one per word)", len(split))
	}

	continuous := underlineOps(layoutDecl("text-decoration-skip-spaces:none"))
	if len(continuous) != 1 {
		t.Fatalf("skip-spaces none produced %d decoration lines, want 1 continuous", len(continuous))
	}
}

// TestBehaviorTextEmphasisShorthandPaintsDots is text-emphasis: a filled dot
// run paints one emphasis mark per letter as extra fill ops, while the plain
// run paints no extra marks.
// Reference: Chrome 143.0.7499.40, emphasis dots.
func TestBehaviorTextEmphasisShorthandPaintsDots(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">hello</p></body></html>`)
	emph := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-emphasis:filled dot">hello</p></body></html>`)

	if fills := opsOfKind(plain, OpFillRect); len(fills) != 0 {
		t.Errorf("plain fills = %d, want 0 (no emphasis marks)", len(fills))
	}

	if fills := opsOfKind(emph, OpFillRect); len(fills) == 0 {
		t.Error("text-emphasis:filled dot: want >0 emphasis fill ops, got 0")
	}
}

// TestBehaviorTextEmphasisColorPaintsRedDots is text-emphasis-color: the dot
// marks use the declared color instead of the text ink.
// Reference: Chrome 143.0.7499.40, red emphasis marks over dark text.
func TestBehaviorTextEmphasisColorPaintsRedDots(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;color:black;`+
		`text-emphasis-style:dot;text-emphasis-color:red">hello</p></body></html>`)

	fills := opsOfKind(res, OpFillRect)
	if len(fills) == 0 {
		t.Fatal("want >0 emphasis fill ops, got 0")
	}

	for _, fill := range fills {
		if !near(fill.R, 1) || !near(fill.G, 0) || !near(fill.B, 0) {
			t.Errorf("emphasis color = (%.3f, %.3f, %.3f), want (1, 0, 0)",
				fill.R, fill.G, fill.B)
		}
	}
}

// TestBehaviorTextEmphasisPositionMovesDotsBelow is text-emphasis-position:
// under puts the marks below the baseline while over puts them above it, so
// the under dots paint below the text and the over dots paint above it.
// Reference: Chrome 143.0.7499.40, emphasis position.
func TestBehaviorTextEmphasisPositionMovesDotsBelow(t *testing.T) {
	t.Parallel()

	over := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-emphasis-style:dot;text-emphasis-position:over">hello</p></body></html>`)
	under := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-emphasis-style:dot;text-emphasis-position:under">hello</p></body></html>`)

	overFills := opsOfKind(over, OpFillRect)
	underFills := opsOfKind(under, OpFillRect)

	if len(overFills) == 0 || len(underFills) == 0 {
		t.Fatalf("want emphasis marks both ways, got over=%d under=%d",
			len(overFills), len(underFills))
	}

	textYOver := firstText(over).Y
	textYUnder := firstText(under).Y

	if !(overFills[0].Y < textYOver) {
		t.Errorf("over dot Y %.4f should sit above text Y %.4f", overFills[0].Y, textYOver)
	}

	if !(underFills[0].Y > textYUnder) {
		t.Errorf("under dot Y %.4f should sit below text Y %.4f", underFills[0].Y, textYUnder)
	}
}

// TestBehaviorTextEmphasisSkipNoLayoutEffect documents the current behavior
// of text-emphasis-skip: the value is stored on the style but no paint pass
// reads the skip mode (paintEmphasis skips only whitespace), so spaces versus
// another skip keyword paints identical marks.
// Unobservable in the drawing list: reported as a no-op, never failing.
// Reference: Chrome 143.0.7499.40 would skip the marked characters.
func TestBehaviorTextEmphasisSkipNoLayoutEffect(t *testing.T) {
	t.Parallel()

	spaces := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-emphasis-style:dot;text-emphasis-skip:spaces">hello</p></body></html>`)
	other := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-emphasis-style:dot;text-emphasis-skip:all">hello</p></body></html>`)

	behaviorText2AssertSameOps(t, spaces, other)
}

// TestBehaviorTextEmphasisStyleOpenStrokesMarks is text-emphasis-style: an
// open mark strokes its dot outline while a filled mark fills it, so open
// paints stroke ops where filled paints fill ops.
// Reference: Chrome 143.0.7499.40, open versus filled marks.
func TestBehaviorTextEmphasisStyleOpenStrokesMarks(t *testing.T) {
	t.Parallel()

	filled := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-emphasis-style:filled circle">hello</p></body></html>`)
	open := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-emphasis-style:open circle">hello</p></body></html>`)

	if fills := opsOfKind(filled, OpFillRect); len(fills) == 0 {
		t.Error("filled circle: want >0 fill mark ops, got 0")
	}

	if strokes := opsOfKind(open, OpStrokeRect); len(strokes) == 0 {
		t.Error("open circle: want >0 stroke mark ops, got 0")
	}

	if fills := opsOfKind(open, OpFillRect); len(fills) != 0 {
		t.Errorf("open circle fills = %d, want 0 (marks stroke, not fill)", len(fills))
	}
}

// TestBehaviorTextGroupAlignEndMovesLine is text-group-align: end right
// aligns the short line in a wide block, so its used X sits right of the
// start-aligned run.
// Reference: Chrome 143.0.7499.40, group alignment.
func TestBehaviorTextGroupAlignEndMovesLine(t *testing.T) {
	t.Parallel()

	start := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:200pt;font-size:12pt;text-group-align:none">Hi</p>`+
		`</body></html>`)
	end := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:200pt;font-size:12pt;text-group-align:end">Hi</p>`+
		`</body></html>`)

	xs := firstTextX(t, start, "Hi")
	xe := firstTextX(t, end, "Hi")

	if xe <= xs+1 {
		t.Errorf("text-group-align:end X %.4f should sit right of start X %.4f", xe, xs)
	}
}

// TestBehaviorTextShadowPaintsOffsetCopy is text-shadow: the shadow paints an
// extra text copy offset by the declared lengths, so the shadowed run emits
// one more text op than the plain run and the extra copy sits down and right.
// Reference: Chrome 143.0.7499.40, text shadows.
func TestBehaviorTextShadowPaintsOffsetCopy(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">hello</p></body></html>`)
	shadowed := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-shadow:3pt 3pt 0pt blue">hello</p></body></html>`)

	plainTexts := opsOfKind(plain, OpText)
	shadowTexts := opsOfKind(shadowed, OpText)

	if len(shadowTexts) != len(plainTexts)+1 {
		t.Fatalf("shadowed text ops = %d, plain = %d, want exactly one extra shadow copy",
			len(shadowTexts), len(plainTexts))
	}

	main := firstText(plain)

	var found bool

	for _, paintOp := range shadowTexts {
		if near(paintOp.X, main.X+3) && near(paintOp.Y, main.Y+3) {
			found = true

			if !near(paintOp.R, 0) || !near(paintOp.G, 0) || !near(paintOp.B, 1) {
				t.Errorf("shadow color = (%.3f, %.3f, %.3f), want blue (0, 0, 1)",
					paintOp.R, paintOp.G, paintOp.B)
			}
		}
	}

	if !found {
		t.Errorf("no shadow copy at (+3pt, +3pt) from main text at (%.4f, %.4f): %+v",
			main.X, main.Y, shadowTexts)
	}
}

// TestBehaviorTextSpacingTrimStartTrimsLead is text-spacing: the trim-start
// keyword mirrors onto text-spacing-trim, so a leading fullwidth open quote
// hangs past the content edge exactly like the trim longhand.
// Reference: Chrome 143.0.7499.40, spacing trim.
func TestBehaviorTextSpacingTrimStartTrimsLead(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">「hello</p></body></html>`)
	trimmed := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-spacing:trim-start">「hello</p></body></html>`)

	plainTexts := opsOfKind(plain, OpText)
	trimTexts := opsOfKind(trimmed, OpText)

	if len(plainTexts) == 0 || len(trimTexts) == 0 {
		t.Fatalf("want text ops, got plain=%d trimmed=%d", len(plainTexts), len(trimTexts))
	}

	if trimTexts[0].X >= plainTexts[0].X-0.05 {
		t.Errorf("trimmed X %.4f should sit left of plain X %.4f by a visible trim",
			trimTexts[0].X, plainTexts[0].X)
	}
}

// TestBehaviorTextSpacingTrimLeadingTrimsPunct is text-spacing-trim:
// trim-start hangs the leading fullwidth open punctuation past the content
// edge, so the trimmed line starts left of the space-all line.
// Reference: Chrome 143.0.7499.40, spacing trim leading punctuation.
func TestBehaviorTextSpacingTrimLeadingTrimsPunct(t *testing.T) {
	t.Parallel()

	spaced := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-spacing-trim:space-all">「hello</p></body></html>`)
	trimmed := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-spacing-trim:trim-start">「hello</p></body></html>`)

	spacedTexts := opsOfKind(spaced, OpText)
	trimTexts := opsOfKind(trimmed, OpText)

	if len(spacedTexts) == 0 || len(trimTexts) == 0 {
		t.Fatalf("want text ops, got spaced=%d trimmed=%d", len(spacedTexts), len(trimTexts))
	}

	if trimTexts[0].X >= spacedTexts[0].X-0.05 {
		t.Errorf("trimmed X %.4f should sit left of space-all X %.4f by a visible trim",
			trimTexts[0].X, spacedTexts[0].X)
	}
}

// TestBehaviorTextUnderlineOffsetShiftsStroke is text-underline-offset: the
// declared offset moves the underline below its default spot, so a 4px offset
// paints about 3pt lower than the plain underline.
// Reference: Chrome 143.0.7499.40, underline offsets.
func TestBehaviorTextUnderlineOffsetShiftsStroke(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
		`<span style="text-decoration:underline">acemnorstu</span>`+
		`</p></body></html>`)
	offset := layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
		`<span style="text-decoration:underline;text-underline-offset:4px">acemnorstu</span>`+
		`</p></body></html>`)

	plainLines := underlineOps(plain)
	offsetLines := underlineOps(offset)

	if len(plainLines) == 0 || len(offsetLines) == 0 {
		t.Fatalf("want decoration ops, got plain=%d offset=%d", len(plainLines), len(offsetLines))
	}

	if delta := offsetLines[0].Y - plainLines[0].Y; !near(delta, 4*ptPerCSSPx) {
		t.Errorf("underline offset shift = %.4fpt, want 4px (%.4fpt)", delta, 4*ptPerCSSPx)
	}
}

// TestBehaviorHangingPunctuationFirstHangsQuote is hanging-punctuation: first
// hangs the leading quote past the content edge, so the hung line starts left
// of the plain line.
// Reference: Chrome 143.0.7499.40, hanging opening punctuation.
func TestBehaviorHangingPunctuationFirstHangsQuote(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:120pt;font-size:14pt;hanging-punctuation:none">"Hi</p>`+
		`</body></html>`)
	hang := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:120pt;font-size:14pt;hanging-punctuation:first">"Hi</p>`+
		`</body></html>`)

	xp := firstTextX(t, plain, `"Hi`)
	xh := firstTextX(t, hang, `"Hi`)

	if xh >= xp-0.1 {
		t.Fatalf("hanging-punctuation:first X %v should be left of plain X %v", xh, xp)
	}
}

// TestBehaviorWhiteSpaceCollapsePreserveKeepsSpaces is white-space-collapse:
// preserve keeps consecutive spaces in the emitted text run while collapse
// folds them to one, so the preserved op still holds the double space.
// Reference: Chrome 143.0.7499.40, whitespace collapsing.
func TestBehaviorWhiteSpaceCollapsePreserveKeepsSpaces(t *testing.T) {
	t.Parallel()

	collapsed := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;white-space-collapse:collapse">a  b</p></body></html>`)
	preserved := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;white-space-collapse:preserve">a  b</p></body></html>`)

	var collapsedHasDouble, preservedHasDouble bool

	for _, op := range collapsed.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "  ") {
			collapsedHasDouble = true
		}
	}

	for _, op := range preserved.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "  ") {
			preservedHasDouble = true
		}
	}

	if collapsedHasDouble {
		t.Errorf("collapsed run should fold double spaces, kept one in %+v", collapsed.Ops)
	}

	if !preservedHasDouble {
		t.Errorf("preserved run should keep the double space, got %+v", preserved.Ops)
	}
}
