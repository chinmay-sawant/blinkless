package layout

import (
	"strings"
	"testing"
)

// Font, list, and generated-content behavior tests: one test per property,
// asserted through USED values (selected faces, measured advances, paint
// order, emitted ops), never through stored style strings. Points are the
// engine unit and 1 CSS pixel = 0.75pt (ptPerCSSPx in css_review_02_test.go).
// Reference browser for every case: Chrome 143.0.7499.40.
//
// Shared helpers (boxByID, near, pxToPt, layoutHTML, sheet, opsOfKind,
// firstText, firstTextX, textOpIndex, joinedPaintText, textOpsContain,
// inlineLines) come from other files in this same package and are reused
// here, never redeclared.

// TestBehaviorFontFamilyFallbackSelectsFace: an unknown first family falls
// back to a bundled face (text still emits with a non-nil face), and the
// generic monospace stack measures wider than the serif stack for the same
// run. Reference: Chrome 143.0.7499.40, Liberation faces at 12pt.
func TestBehaviorFontFamilyFallbackSelectsFace(t *testing.T) {
	t.Parallel()

	fallback := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-family:'NoSuchFaceXYZ',serif">hello</p></body></html>`)
	mono := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-family:monospace">hello</p></body></html>`)
	serif := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-family:serif">hello</p></body></html>`)

	fallbackOp := firstText(fallback)
	if fallbackOp.Font == nil || fallbackOp.Text == "" {
		t.Fatalf("unknown family should fall back to a bundled face, got font=%v text=%q",
			fallbackOp.Font, fallbackOp.Text)
	}

	if wMono, wSerif := firstText(mono).W, firstText(serif).W; wMono <= wSerif {
		t.Errorf("monospace width %.4fpt should exceed serif width %.4fpt for the same run",
			wMono, wSerif)
	}
}

// TestBehaviorFontSizeScalesUsedSize: the used op size equals the declared
// font size, and doubling the size grows the measured advance.
// Reference: Chrome 143.0.7499.40, 12pt vs 24pt.
func TestBehaviorFontSizeScalesUsedSize(t *testing.T) {
	t.Parallel()

	small := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;white-space:nowrap">hello</p></body></html>`)
	large := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:24pt;white-space:nowrap">hello</p></body></html>`)

	smallOp, largeOp := firstText(small), firstText(large)

	if !near(smallOp.Size, 12) || !near(largeOp.Size, 24) {
		t.Errorf("used sizes = %.4fpt and %.4fpt, want 12pt and 24pt", smallOp.Size, largeOp.Size)
	}

	if largeOp.W <= smallOp.W*1.5 {
		t.Errorf("24pt width %.4fpt should be well above 12pt width %.4fpt", largeOp.W, smallOp.W)
	}
}

// TestBehaviorFontStyleItalicSelectsFace: font-style italic resolves a
// different (italic) face or synthesizes an oblique skew, so the used
// rendering differs from the normal run.
// Reference: Chrome 143.0.7499.40, Liberation Sans italic.
func TestBehaviorFontStyleItalicSelectsFace(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">Ag</p></body></html>`)
	italic := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-style:italic">Ag</p></body></html>`)

	plainOp, italicOp := firstText(plain), firstText(italic)

	if italicOp.Font == plainOp.Font && !italicOp.FakeOblique {
		t.Errorf("italic run should use another face or FakeOblique, got same face %v oblique=%v",
			italicOp.Font, italicOp.FakeOblique)
	}
}

// TestBehaviorFontWeightBoldSelectsFace: font-weight bold resolves the bold
// face (or flags the op bold for synthesis) at the same used size.
// Reference: Chrome 143.0.7499.40, Liberation Sans bold.
func TestBehaviorFontWeightBoldSelectsFace(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">Ag</p></body></html>`)
	bold := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-weight:bold">Ag</p></body></html>`)

	plainOp, boldOp := firstText(plain), firstText(bold)

	if boldOp.Font == plainOp.Font && !boldOp.Bold {
		t.Errorf("bold run should use another face or carry Bold, got same face %v bold=%v",
			boldOp.Font, boldOp.Bold)
	}

	if !near(boldOp.Size, plainOp.Size) {
		t.Errorf("bold used size %.4fpt should equal normal %.4fpt", boldOp.Size, plainOp.Size)
	}
}

// TestBehaviorFontVariantSmallCapsUppercases: the font-variant shorthand with
// small-caps paints the run uppercased (synthesized small caps).
// Reference: Chrome 143.0.7499.40, small-caps synthesis.
func TestBehaviorFontVariantSmallCapsUppercases(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-variant:small-caps">ag</p></body></html>`)

	if got := joinedPaintText(res); got != "AG" {
		t.Errorf("small-caps paint text = %q, want %q", got, "AG")
	}
}

// TestBehaviorFontSizeAdjustScalesUsedSize: font-size-adjust scales the used
// size by the ratio of the declared aspect to the face x-height aspect, so
// adjust 1.0 on Liberation Sans lays out much larger than no adjustment.
// Reference: Chrome 143.0.7499.40, font-size-adjust scaling.
func TestBehaviorFontSizeAdjustScalesUsedSize(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">Ag</p></body></html>`)
	adjusted := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-size-adjust:1.0">Ag</p></body></html>`)

	plainOp, adjustedOp := firstText(plain), firstText(adjusted)

	if adjustedOp.Size <= plainOp.Size*1.5 {
		t.Errorf("adjusted used size %.4fpt should exceed plain %.4fpt by well over half",
			adjustedOp.Size, plainOp.Size)
	}
}

// TestBehaviorFontKerningNoneDisablesKern: font-kerning none carries a kern-off
// feature tag on the text op for the shaper, while the default run carries no
// kern tag. Reference: Chrome 143.0.7499.40, kerning feature handling.
func TestBehaviorFontKerningNoneDisablesKern(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">AV</p></body></html>`)
	off := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-kerning:none">AV</p></body></html>`)

	if got := firstText(off).FontFeatures(); !strings.Contains(got, `"kern" 0`) {
		t.Errorf("kerning:none features = %q, want it to contain %q", got, `"kern" 0`)
	}

	if got := firstText(plain).FontFeatures(); strings.Contains(got, "kern") {
		t.Errorf("default kerning features = %q, want no kern tag", got)
	}
}

// TestBehaviorDirectionRtlRightAlignsLine: direction rtl right-aligns the
// short line in a wide block, so its used X sits right of the ltr run.
// Reference: Chrome 143.0.7499.40, rtl default alignment.
func TestBehaviorDirectionRtlRightAlignsLine(t *testing.T) {
	t.Parallel()

	ltr := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:150pt;font-size:12pt">Hi</p></body></html>`)
	rtl := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:150pt;font-size:12pt;direction:rtl">Hi</p></body></html>`)

	ltrX, rtlX := firstTextX(t, ltr, "Hi"), firstTextX(t, rtl, "Hi")

	if rtlX <= ltrX+1 {
		t.Errorf("rtl X %.4fpt should sit right of ltr X %.4fpt", rtlX, ltrX)
	}
}

// TestBehaviorUnicodeBidiOverrideReversesOrder: direction rtl with
// unicode-bidi bidi-override reverses the visual run order, while the same
// markup without the override keeps logical order.
// Reference: Chrome 143.0.7499.40, bidi override.
// unicode-bidi override order is already pinned by
// TestUnicodeBidiOverrideReversesRunOrder in text_support_layout_test.go, so
// no duplicate lives here. The direction test below covers the rtl half of
// the pair.

// TestBehaviorWritingModeVerticalRotatesRun: writing-mode vertical-rl paints
// the sideways run rotated -90 degrees.
// Reference: Chrome 143.0.7499.40, vertical writing.
func TestBehaviorWritingModeVerticalRotatesRun(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
		`<span style="writing-mode:vertical-rl">CD</span>`+
		`</p></body></html>`)

	_, textOp, found := textOpIndex(res, "CD")
	if !found {
		t.Fatal("no text op for CD")
	}

	if textOp.RotateDeg != -90 {
		t.Fatalf("vertical-rl RotateDeg = %v, want -90", textOp.RotateDeg)
	}
}

// TestBehaviorListStyleTypeDecimalEmitsNumbers: list-style-type decimal emits
// numbered OpBullet markers (1., 2.) while square emits the square glyph.
// Reference: Chrome 143.0.7499.40, list markers.
func TestBehaviorListStyleTypeDecimalEmitsNumbers(t *testing.T) {
	t.Parallel()

	ordered := layoutHTML(t, `<html><body style="margin:0">`+
		`<ol style="margin:0"><li>one</li><li>two</li></ol></body></html>`)
	square := layoutHTML(t, `<html><body style="margin:0">`+
		`<ul style="margin:0;list-style-type:square"><li>one</li></ul></body></html>`)

	bullets := opsOfKind(ordered, OpBullet)
	if len(bullets) != 2 || bullets[0].Text != "1." || bullets[1].Text != "2." {
		t.Errorf("decimal bullets = %v, want [1. 2.]", bullets)
	}

	squareBullets := opsOfKind(square, OpBullet)
	if len(squareBullets) != 1 {
		t.Errorf("square bullets = %v, want exactly one marker", squareBullets)
	} else if mark := squareBullets[0].Text; len([]rune(mark)) != 1 || []rune(mark)[0] < 0x80 {
		t.Errorf("square bullet = %q, want one non-ASCII marker glyph unlike the decimal numbers", mark)
	}
}

// TestBehaviorContentBeforeEmitsGeneratedText: p::before content paints the
// generated marker before the host text in paint order.
// Reference: Chrome 143.0.7499.40, generated content.
func TestBehaviorContentBeforeEmitsGeneratedText(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `p::before { content: ">>"; }`)
	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">hello</p></body></html>`, cssSheet)

	if !textOpsContain(res, ">>") {
		t.Fatalf("no generated ::before text, paint text = %q", joinedPaintText(res))
	}

	markerIdx, hostIdx := behaviorFontListMarkerOrder(res.Ops, ">>", "hello")

	if markerIdx < 0 || hostIdx < 0 {
		t.Fatalf("marker idx %d host idx %d, want both painted", markerIdx, hostIdx)
	}

	if markerIdx > hostIdx {
		t.Errorf("::before marker at op %d paints after host text at op %d, want before",
			markerIdx, hostIdx)
	}
}

// behaviorFontListMarkerOrder returns the first op indexes holding the
// marker and host texts, or -1 for either when absent.
func behaviorFontListMarkerOrder(ops []Op, marker, host string) (int, int) {
	markerIdx, hostIdx := -1, -1

	for idx, textOp := range ops {
		if textOp.Kind != OpText {
			continue
		}

		if strings.Contains(textOp.Text, marker) && markerIdx < 0 {
			markerIdx = idx
		}

		if strings.Contains(textOp.Text, host) && hostIdx < 0 {
			hostIdx = idx
		}
	}

	return markerIdx, hostIdx
}
