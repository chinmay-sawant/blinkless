package layout

import (
	"strings"
	"testing"
)

// Remaining CSS behavior tests: every property below is asserted through a
// used value (box geometry or an emitted op field), never through a stored
// style string. Lengths are written in CSS px and converted with pxToPt
// (1px = 0.75pt, ptPerCSSPx in css_review_02_test.go), which is how Chrome
// reports them. Helpers boxByID, near, pxToPt, opsOfKind, inlineLines,
// layoutHTMLWithImages, and tinyPNG come from css_review_02_test.go,
// layout_test.go, and inline_balance_test.go in this same package and are
// never redeclared here. Reference browser for every case: Chrome
// 143.0.7499.40.

// behaviorRemAssertSameOps pins no-op behavior: two results must paint the
// same ops in the same order with identical geometry and text.
func behaviorRemAssertSameOps(t *testing.T, first, second *Result) {
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

// TestBehaviorLineBreakAnywhereWrapsToken is line-break: anywhere behaves
// like overflow-wrap:anywhere and overrides word-break:keep-all, so an
// 80-rune spaceless token in a 200px block stays on one overflowing line
// under keep-all alone but wraps across two or more lines once
// line-break:anywhere is added. Reference: Chrome 143.0.7499.40 keeps the
// keep-all token on 1 overflowing line of 80 runes at 200px and 12pt type,
// while keep-all plus anywhere wraps to 7 lines of about 13 runes each.
func TestBehaviorLineBreakAnywhereWrapsToken(t *testing.T) {
	t.Parallel()

	token := strings.Repeat("W", 80)
	doc := func(decl string) string {
		return `<html><body style="margin:0">` +
			`<p style="margin:0;width:200px;font-size:12pt;` + decl + `">` + token + `</p></body></html>`
	}

	keepLines := inlineLines(layoutHTML(t, doc("word-break:keep-all")))
	anyLines := inlineLines(layoutHTML(t, doc("word-break:keep-all;line-break:anywhere")))

	if len(keepLines) != 1 {
		t.Fatalf("keep-all lines = %d, want 1 overflowing line: %+v", len(keepLines), keepLines)
	}

	if len(anyLines) < 2 {
		t.Fatalf("line-break:anywhere lines = %d, want at least 2 wrapped lines: %+v", len(anyLines), anyLines)
	}

	if anyLines[0].text == token {
		t.Errorf("anywhere first line holds the full token, want a mid-token split")
	}
}

// TestBehaviorLineClampTwoLimitsLines is line-clamp: a 200px block of 16px
// prose that paints 4 lines by default keeps exactly 2 lines under
// line-clamp:2 with an ellipsis marker on the last line. Reference: Chrome
// 143.0.7499.40 shows lines 1-2 plus an ellipsis and hides lines 3-4 at
// 200px width.
func TestBehaviorLineClampTwoLimitsLines(t *testing.T) {
	t.Parallel()

	prose := "The quick brown fox jumps over the lazy dog near the river bank " +
		"while the red fox watches the slow green turtle cross the road again."
	doc := func(decl string) string {
		return `<html><body style="margin:0">` +
			`<p style="margin:0;width:200px;font-size:16px;` + decl + `">` + prose + `</p></body></html>`
	}

	defLines := inlineLines(layoutHTML(t, doc("")))
	if len(defLines) < 3 {
		t.Fatalf("default lines = %d, want at least 3 unclamped lines: %+v", len(defLines), defLines)
	}

	clamped := layoutHTML(t, doc("line-clamp:2"))
	clampLines := inlineLines(clamped)

	if len(clampLines) != 2 {
		t.Fatalf("line-clamp:2 lines = %d, want exactly 2: %+v", len(clampLines), clampLines)
	}

	joined := ""

	for _, paintOp := range clamped.Ops {
		if paintOp.Kind == OpText {
			joined += paintOp.Text
		}
	}

	if !strings.Contains(joined, "…") {
		t.Errorf("clamped paint text %q has no ellipsis, want trailing … on line 2", joined)
	}
}

// TestBehaviorMaxLinesTwoLimitsLines is max-lines: the same 200px prose
// keeps exactly 2 lines under max-lines:2 with an ellipsis, matching the
// line-clamp path when line-clamp is unset. Reference: Chrome 143.0.7499.40
// truncates the 4-line 200px block to 2 lines plus ellipsis under
// max-lines:2.
func TestBehaviorMaxLinesTwoLimitsLines(t *testing.T) {
	t.Parallel()

	prose := "The quick brown fox jumps over the lazy dog near the river bank " +
		"while the red fox watches the slow green turtle cross the road again."
	doc := func(decl string) string {
		return `<html><body style="margin:0">` +
			`<p style="margin:0;width:200px;font-size:16px;` + decl + `">` + prose + `</p></body></html>`
	}

	clamped := layoutHTML(t, doc("max-lines:2"))
	clampLines := inlineLines(clamped)

	if len(clampLines) != 2 {
		t.Fatalf("max-lines:2 lines = %d, want exactly 2: %+v", len(clampLines), clampLines)
	}

	joined := ""

	for _, paintOp := range clamped.Ops {
		if paintOp.Kind == OpText {
			joined += paintOp.Text
		}
	}

	if !strings.Contains(joined, "…") {
		t.Errorf("max-lines paint text %q has no ellipsis, want trailing … on line 2", joined)
	}
}

// TestBehaviorListStyleNoneHidesMarker is list-style: the none shorthand
// suppresses the marker, so a disc list paints one OpBullet while
// list-style:none paints zero. Reference: Chrome 143.0.7499.40 paints a
// 5px disc at the gutter for the default 16px item and paints no marker box
// under list-style:none.
func TestBehaviorListStyleNoneHidesMarker(t *testing.T) {
	t.Parallel()

	disc := layoutHTML(t, `<html><body style="margin:0">`+
		`<ul style="margin:0;font-size:12pt"><li id="item">one</li></ul></body></html>`)
	none := layoutHTML(t, `<html><body style="margin:0">`+
		`<ul style="margin:0;font-size:12pt;list-style:none"><li id="item">one</li></ul></body></html>`)

	if bullets := opsOfKind(disc, OpBullet); len(bullets) != 1 {
		t.Fatalf("disc bullets = %d, want exactly one marker", len(bullets))
	} else if bullets[0].Text == "" {
		t.Errorf("disc bullet text empty, want a marker glyph")
	}

	if bullets := opsOfKind(none, OpBullet); len(bullets) != 0 {
		t.Errorf("list-style:none bullets = %d, want zero markers, got %+v", len(bullets), bullets)
	}

	itemBox := boxByID(t, disc, "item")
	if itemBox.w <= 0 || itemBox.height <= 0 {
		t.Errorf("li box = %.4fpt x %.4fpt, want positive content box", itemBox.w, itemBox.height)
	}

	// Pin the 1px = 0.75pt scale used across these behavior tests.
	if got := 8 * ptPerCSSPx; !near(got, pxToPt(8)) {
		t.Errorf("8px = %.4fpt, want %.4fpt", got, pxToPt(8))
	}
}

// TestBehaviorObjectViewBoxXywhCropsImage is object-view-box: the xywh crop
// selects a source sub-rect, so a 10x10 image under
// object-view-box:xywh(0px 0px 5px 5px) emits a 5x5 intrinsic payload while
// the plain image emits 10x10. Reference: Chrome 143.0.7499.40 scales the
// 5x5 top-left crop into the full 10px content box.
func TestBehaviorObjectViewBoxXywhCropsImage(t *testing.T) {
	t.Parallel()

	png := tinyPNG(10, 10)
	doc := func(decl string) string {
		return `<html><body style="margin:0">` +
			`<img id="pic" src="x.png" style="` + decl + `"></body></html>`
	}

	plain := layoutHTMLWithImages(t, doc(""), png, "x.png")
	cropped := layoutHTMLWithImages(t, doc("object-view-box:xywh(0px 0px 5px 5px)"), png, "x.png")

	plainImgs := opsOfKind(plain, OpImage)
	if len(plainImgs) != 1 {
		t.Fatalf("plain images = %d, want 1", len(plainImgs))
	}

	if plainImgs[0].ImgW != 10 || plainImgs[0].ImgH != 10 {
		t.Fatalf("plain intrinsic = %dx%d, want 10x10", plainImgs[0].ImgW, plainImgs[0].ImgH)
	}

	if plainImgs[0].W <= 0 || plainImgs[0].H <= 0 {
		t.Errorf("plain used size = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want positive replaced box",
			plainImgs[0].W, plainImgs[0].H, plainImgs[0].W/ptPerCSSPx, plainImgs[0].H/ptPerCSSPx)
	}

	cropImgs := opsOfKind(cropped, OpImage)
	if len(cropImgs) != 1 {
		t.Fatalf("cropped images = %d, want 1", len(cropImgs))
	}

	if cropImgs[0].ImgW != 5 || cropImgs[0].ImgH != 5 {
		t.Errorf("cropped intrinsic = %dx%d, want 5x5", cropImgs[0].ImgW, cropImgs[0].ImgH)
	}
}

// TestBehaviorPerspectiveNoLayoutEffect documents the current behavior of
// perspective: the declaration is dropped (style_properties.go forwards it to
// applyLeftoversProps, which has no perspective case and returns false) and
// no layout or paint pass reads it, so perspective:500px paints identical
// geometry to the default. Unobservable in the drawing list: reported as a
// no-op, never failing. Reference: Chrome 143.0.7499.40 would foreshorten a
// rotated child inside a 200x100px perspective container.
func TestBehaviorPerspectiveNoLayoutEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:100px;height:50px">sample text here</div></body></html>`)
	persp := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:100px;height:50px;perspective:500px">sample text here</div></body></html>`)

	behaviorRemAssertSameOps(t, plain, persp)
}

// TestBehaviorPerspectiveOriginNoLayoutEffect documents the current behavior
// of perspective-origin: like perspective it is dropped by
// applyLeftoversProps and no consumer reads it, so
// perspective-origin:25% 75% paints identical geometry to the default.
// Unobservable in the drawing list: reported as a no-op, never failing.
// Reference: Chrome 143.0.7499.40 would move the vanishing point to
// (50px, 75px) of a 200x100px perspective container.
func TestBehaviorPerspectiveOriginNoLayoutEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:100px;height:50px">sample text here</div></body></html>`)
	origin := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:100px;height:50px;perspective:500px;`+
		`perspective-origin:25% 75%">sample text here</div></body></html>`)

	behaviorRemAssertSameOps(t, plain, origin)
}

// TestBehaviorDominantBaselineNoPaintEffect pins the remaining no-op pair
// for dominant-baseline: auto and alphabetic share the alphabetic baseline,
// so they paint identical text positions. Hanging now raises the run (see
// TestGapBDominantBaselineHangingRaisesText); other baselines pin to
// alphabetic until font baseline tables exist.
// Reference: Chrome 143.0.7499.40 keeps auto and alphabetic on the same
// baseline and aligns the 16px hanging run about 5px above alphabetic.
func TestBehaviorDominantBaselineNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:16px">Ag</p></body></html>`)
	alpha := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:16px;dominant-baseline:alphabetic">Ag</p></body></html>`)

	behaviorRemAssertSameOps(t, plain, alpha)
}

// TestBehaviorInitialLetterAlignHangingRaisesLetter is initial-letter-align:
// the hanging metric sits above alphabetic, so a 3-line drop cap under
// hanging paints its T higher (smaller Y) than under alphabetic at 12pt
// type with a 16pt parent line. Reference: Chrome 143.0.7499.40 places the
// hanging cap about 4px above the alphabetic cap for a 3-line sink.
func TestBehaviorInitialLetterAlignHangingRaisesLetter(t *testing.T) {
	t.Parallel()

	body := `he rest of this paragraph wraps beside the drop cap across several ` +
		`lines so geometry can be checked against the large letter box.`
	doc := func(align string) string {
		return `<html><body style="margin:0">` +
			`<p style="margin:0;font-size:12pt;line-height:16pt;width:220pt">` +
			`<span style="initial-letter:3;initial-letter-align:` + align +
			`;initial-letter-wrap:none">T</span>` + body + `</p></body></html>`
	}

	alpha := layoutHTML(t, doc("alphabetic"))
	hang := layoutHTML(t, doc("hanging"))

	alphaY, alphaFound := behaviorRemBigLetterY(alpha.Ops)
	hangY, hangFound := behaviorRemBigLetterY(hang.Ops)

	if !alphaFound || !hangFound {
		t.Fatalf("drop-cap letter missing: alphabetic Y=%.2f hanging Y=%.2f", alphaY, hangY)
	}

	if !(hangY < alphaY) {
		t.Errorf("hanging Y=%.4f should sit above alphabetic Y=%.4f", hangY, alphaY)
	}
}

// behaviorRemLetterAndBodyX returns the X of the large initial letter and
// the X of the first body run in a result.
func behaviorRemLetterAndBodyX(ops []Op) (float64, float64) {
	letterX, bodyX := -1.0, -1.0

	for _, paintOp := range ops {
		if paintOp.Kind != OpText || strings.TrimSpace(paintOp.Text) == "" {
			continue
		}

		if (paintOp.Text == "T" || strings.HasPrefix(paintOp.Text, "T")) && paintOp.Size > 20 && letterX < 0 {
			letterX = paintOp.X

			continue
		}

		if bodyX < 0 && !(paintOp.Size > 20) {
			bodyX = paintOp.X
		}
	}

	return letterX, bodyX
}

// behaviorRemBigLetterY returns the Y of the large initial letter op.
func behaviorRemBigLetterY(ops []Op) (float64, bool) {
	for _, paintOp := range ops {
		if paintOp.Kind == OpText && (paintOp.Text == "T" || strings.HasPrefix(paintOp.Text, "T")) && paintOp.Size > 20 {
			return paintOp.Y, true
		}
	}

	return 0, false
}

// TestBehaviorInitialLetterWrapAllExcludesLaterLines is initial-letter-wrap:
// all keeps the rectangular exclusion over every overlapping sink line while
// none excludes nothing, so the first body run beside a 3-line T sits right
// of the letter under wrap:all and starts at the content edge under
// wrap:none. Reference: Chrome 143.0.7499.40 keeps about 40px of exclusion
// beside the 3-line cap under wrap:all and zero under wrap:none.
func TestBehaviorInitialLetterWrapAllExcludesLaterLines(t *testing.T) {
	t.Parallel()

	body := `he rest of this paragraph wraps beside the drop cap across several ` +
		`lines so geometry can be checked. More words fill a second and third ` +
		`line beside the letter box so wrap keywords can diverge.`
	doc := func(wrap string) string {
		return `<html><body style="margin:0">` +
			`<p style="margin:0;font-size:12pt;line-height:16pt;width:220pt">` +
			`<span style="initial-letter:3;initial-letter-wrap:` + wrap + `">T</span>` + body + `</p></body></html>`
	}

	noneRes := layoutHTML(t, doc("none"))
	allRes := layoutHTML(t, doc("all"))

	noneLetterX, noneBodyX := behaviorRemLetterAndBodyX(noneRes.Ops)
	allLetterX, allBodyX := behaviorRemLetterAndBodyX(allRes.Ops)

	if noneLetterX < 0 || noneBodyX < 0 || allLetterX < 0 || allBodyX < 0 {
		t.Fatalf("missing letter/body runs: none L=%.2f B=%.2f all L=%.2f B=%.2f",
			noneLetterX, noneBodyX, allLetterX, allBodyX)
	}

	if !(noneBodyX <= noneLetterX+2) {
		t.Errorf("wrap:none first body x=%.4f should start at the letter edge x=%.4f (no exclusion)",
			noneBodyX, noneLetterX)
	}

	if !(allBodyX > allLetterX+2) {
		t.Errorf("wrap:all first body x=%.4f should sit right of the letter x=%.4f",
			allBodyX, allLetterX)
	}
}
