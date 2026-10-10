package layout

import (
	"strings"
	"testing"
)

// Final CSS behavior wave. Each test covers one property and asserts used box
// geometry or emitted display-list ops, never stored strings. Reference
// browser for every case: Chrome 143.0.7499.40. Shared helpers boxByID,
// ptPerCSSPx, near, pxToPt, opsOfKind, layoutHTML, inlineLines, and
// behaviorSvgAssertSameOps come from layout_test.go, css_review_02_test.go,
// inline_balance_test.go, and css_behavior_svg_paint_test.go and are never
// redeclared here.

// behaviorFinalLetterBody returns the X and width of the large initial-letter
// op and the X of the first body run beside it. Missing runs report -1.
func behaviorFinalLetterBody(ops []Op) (float64, float64, float64) {
	letterX, bodyX := -1.0, -1.0
	letterW := 0.0

	for _, paintOp := range ops {
		if paintOp.Kind != OpText || strings.TrimSpace(paintOp.Text) == "" {
			continue
		}

		if letterX < 0 && strings.HasPrefix(paintOp.Text, "T") && paintOp.Size > 20 {
			letterX, letterW = paintOp.X, paintOp.W

			continue
		}

		if bodyX < 0 && paintOp.Size <= 20 {
			bodyX = paintOp.X
		}
	}

	return letterX, letterW, bodyX
}

// TestBehaviorBackgroundBlendModeMultiplyBlends is background-blend-mode:
// multiply on two gradient layers stamps multiply onto each emitted image
// op, while normal stamps no multiply mode. Reference: Chrome 143.0.7499.40
// multiplies the upper gradient through the lower one instead of covering
// it. The group pattern mirrors TestBehaviorMixBlendModeOpensGroup.
func TestBehaviorBackgroundBlendModeMultiplyBlends(t *testing.T) {
	t.Parallel()

	page := func(mode string) *Result {
		return layoutHTML(t, `<html><body style="margin:0">`+
			`<div id="bg" style="width:60pt;height:30pt;`+
			`background-image:linear-gradient(red, blue), linear-gradient(green, yellow);`+
			`background-blend-mode:`+mode+`"></div></body></html>`)
	}

	modes := func(res *Result) []string {
		imgs := opsOfKind(res, OpImage)
		out := make([]string, 0, len(imgs))

		for _, op := range imgs {
			out = append(out, op.BlendModeName())
		}

		return out
	}

	mult := modes(page("multiply"))
	if len(mult) != 2 {
		t.Fatalf("multiply image ops = %d, want 2 background layers", len(mult))
	}

	for _, mode := range mult {
		if mode != blendMultiply {
			t.Errorf("multiply layer mode = %q, want %q", mode, blendMultiply)
		}
	}

	norm := modes(page("normal"))
	if len(norm) != 2 {
		t.Fatalf("normal image ops = %d, want 2 background layers", len(norm))
	}

	for _, mode := range norm {
		if mode == blendMultiply {
			t.Errorf("normal layer mode = %q, want no multiply blend", mode)
		}
	}
}

// TestBehaviorTextBoxShorthandMatchesLonghands is text-box: the trim-both cap
// alphabetic shorthand produces the same used box height as the
// text-box-trim plus text-box-edge longhands. Reference: Chrome
// 143.0.7499.40 trims the same half-leading either way.
func TestBehaviorTextBoxShorthandMatchesLonghands(t *testing.T) {
	t.Parallel()

	doc := func(decl string) string {
		return `<html><body style="margin:0">` +
			`<p id="p" style="margin:0;font-size:20pt;line-height:2;` + decl + `">Hg</p></body></html>`
	}

	short := boxByID(t, layoutHTML(t, doc("text-box:trim-both cap alphabetic")), "p")
	long := boxByID(t, layoutHTML(t, doc("text-box-trim:trim-both;text-box-edge:cap alphabetic")), "p")

	if !near(short.height, long.height) {
		t.Errorf("shorthand height %.4fpt != longhand height %.4fpt", short.height, long.height)
	}
}

// TestBehaviorTextSpacingShorthandMirrorsTrim is text-spacing: the trim-start
// keyword mirrors onto text-spacing-trim (the only spacing consumer the paint
// path reads), so the shorthand hangs the leading fullwidth open quote at
// exactly the longhand X. Reference: Chrome 143.0.7499.40, spacing trim.
func TestBehaviorTextSpacingShorthandMirrorsTrim(t *testing.T) {
	t.Parallel()

	doc := func(decl string) string {
		return `<html><body style="margin:0">` +
			`<p style="margin:0;font-size:12pt;` + decl + `">「hello</p></body></html>`
	}

	shortTexts := opsOfKind(layoutHTML(t, doc("text-spacing:trim-start")), OpText)
	longTexts := opsOfKind(layoutHTML(t, doc("text-spacing-trim:trim-start")), OpText)

	if len(shortTexts) == 0 || len(longTexts) == 0 {
		t.Fatalf("want text ops, got shorthand=%d longhand=%d", len(shortTexts), len(longTexts))
	}

	if !near(shortTexts[0].X, longTexts[0].X) {
		t.Errorf("shorthand X %.4fpt != longhand X %.4fpt", shortTexts[0].X, longTexts[0].X)
	}
}

// TestBehaviorTextWrapModeNowrapFoldsOntoWhiteSpace is text-wrap-mode: nowrap
// folds onto white-space:nowrap, so the 43-rune prose stays on one
// overflowing line in a 150px block, while wrap lets it fold onto two or
// more lines. Reference: Chrome 143.0.7499.40, text-wrap-mode handling.
func TestBehaviorTextWrapModeNowrapFoldsOntoWhiteSpace(t *testing.T) {
	t.Parallel()

	prose := "The quick brown fox jumps over the lazy dog"
	doc := func(decl string) string {
		return `<html><body style="margin:0">` +
			`<p style="margin:0;width:150px;font-size:16px;` + decl + `">` + prose + `</p></body></html>`
	}

	wrapLines := inlineLines(layoutHTML(t, doc("text-wrap-mode:wrap")))
	nowrapLines := inlineLines(layoutHTML(t, doc("text-wrap-mode:nowrap")))

	if len(wrapLines) < 2 {
		t.Fatalf("wrap lines = %d, want at least 2 wrapped lines", len(wrapLines))
	}

	if len(nowrapLines) != 1 {
		t.Fatalf("nowrap lines = %d, want 1 overflowing line: %+v", len(nowrapLines), nowrapLines)
	}

	if nowrapLines[0].text != prose {
		t.Errorf("nowrap text = %q, want the full prose %q", nowrapLines[0].text, prose)
	}
}

// TestBehaviorTransformBoxFillBoxNoPaintEffect documents the current behavior
// of transform-box: the declaration is dropped (applyTransformGroup forwards
// it to applyLeftoversProps, which has no transform-box case and returns
// false), and resolveTransformOrigin always measures the border box, so
// fill-box paints identical geometry to view-box under the same rotation.
// GAP: unobservable in the drawing list, reported as a no-op, never failing.
// Reference: Chrome 143.0.7499.40 would rotate about the object bounding box
// versus the nearest SVG viewport.
func TestBehaviorTransformBoxFillBoxNoPaintEffect(t *testing.T) {
	t.Parallel()

	doc := func(box string) string {
		return `<html><body style="margin:0">` +
			`<div id="a" style="width:100px;height:50px;background-color:#ff0000;` +
			`transform:rotate(10deg);transform-origin:0 0;transform-box:` + box + `"></div>` +
			`</body></html>`
	}

	behaviorSvgAssertSameOps(t, layoutHTML(t, doc("fill-box")), layoutHTML(t, doc("view-box")))
}

// TestBehaviorTransformStylePreserve3DNoPaintEffect documents the current
// behavior of transform-style: the declaration is dropped through the same
// forwarding chain as transform-box, and the engine bakes only static 2D
// affine matrices, so preserve-3d paints identical geometry to flat on nested
// transformed elements. GAP: 3D transforms are a permanent print non-goal
// (documentation/deferred.md). Reported as a no-op, never failing.
// Reference: Chrome 143.0.7499.40 would keep the child in the parent 3D plane
// under preserve-3d and flatten it under flat.
func TestBehaviorTransformStylePreserve3DNoPaintEffect(t *testing.T) {
	t.Parallel()

	doc := func(style string) string {
		return `<html><body style="margin:0">` +
			`<div id="outer" style="width:100px;height:100px;transform:rotate(10deg);` +
			`transform-style:` + style + `">` +
			`<div id="inner" style="width:50px;height:50px;background-color:#00ff00;` +
			`transform:rotate(5deg)"></div></div></body></html>`
	}

	behaviorSvgAssertSameOps(t, layoutHTML(t, doc("flat")), layoutHTML(t, doc("preserve-3d")))
}

// TestBehaviorInitialLetterExclusionWidth is initial-letter: the 3-line T
// opens a rectangular exclusion whose width equals the sized letter advance
// (about 40px in Chrome 143.0.7499.40), pushing the first body run right of
// the letter. TestInitialLetterSpansThreeLines pins the letter size and the
// rightward wrap; this pins the exclusion width itself, which that test never
// measures.
func TestBehaviorInitialLetterExclusionWidth(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;line-height:16pt;width:220pt">`+
		`<span style="initial-letter:3;initial-letter-wrap:all">T</span>`+
		`he rest of this paragraph wraps beside the drop cap across several `+
		`lines so geometry can be checked. More words fill a second and third `+
		`line beside the letter box.</p></body></html>`)

	letterX, letterW, bodyX := behaviorFinalLetterBody(res.Ops)
	if letterX < 0 || bodyX < 0 {
		t.Fatalf("missing letter/body runs: letterX=%.2f bodyX=%.2f", letterX, bodyX)
	}

	exclusion := bodyX - letterX
	if exclusion <= 0 {
		t.Fatalf("exclusion width %.4fpt <= 0, want the letter advance", exclusion)
	}

	if !near(exclusion, letterW) {
		t.Errorf("exclusion width %.4fpt != letter advance %.4fpt", exclusion, letterW)
	}

	if widthPx := exclusion / ptPerCSSPx; widthPx < 25 || widthPx > 65 {
		t.Errorf("exclusion width %.4fpt (%.1fpx), want about 40px per Chrome", exclusion, widthPx)
	}
}

// TestBehaviorDynamicRangeLimitStandardNoClampEffect documents the current
// behavior of dynamic-range-limit: the value parses and stores
// (DynamicRangeLimit, style_color_adjust_props.go:72) but no sRGB clamp
// consumer reads it, so standard paints identical ops to no-limit. GAP:
// missing behavior is an sRGB channel clamp. Reported as a no-op, never
// failing. Reference: Chrome 143.0.7499.40 would clamp HDR headroom under
// standard.
func TestBehaviorDynamicRangeLimitStandardNoClampEffect(t *testing.T) {
	t.Parallel()

	doc := func(limit string) string {
		return `<html><body style="margin:0">` +
			`<div id="c" style="width:50px;height:20px;background-color:#ff0000;` +
			`dynamic-range-limit:` + limit + `"></div></body></html>`
	}

	behaviorSvgAssertSameOps(t, layoutHTML(t, doc("no-limit")), layoutHTML(t, doc("standard")))
}

// TestBehaviorForcedColorAdjustNoneNoPaintEffect documents the current
// behavior of forced-color-adjust: the value parses, stores, and inherits
// (ForcedColorAdjust, style_color_adjust_props.go:54) but no forced-colors
// consumer reads it (print has no forced-colors mode), so none paints
// identical ops to auto. GAP: missing behavior is a forced-colors opt-out.
// Reported as a no-op, never failing. Reference: Chrome 143.0.7499.40 would
// keep author colors under none in forced-colors mode.
func TestBehaviorForcedColorAdjustNoneNoPaintEffect(t *testing.T) {
	t.Parallel()

	doc := func(adjust string) string {
		return `<html><body style="margin:0">` +
			`<div id="c" style="width:50px;height:20px;background-color:#0000ff;` +
			`forced-color-adjust:` + adjust + `"><p id="p" style="margin:0;color:#ffff00">hi</p></div>` +
			`</body></html>`
	}

	behaviorSvgAssertSameOps(t, layoutHTML(t, doc("auto")), layoutHTML(t, doc("none")))
}
