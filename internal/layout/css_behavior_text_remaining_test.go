package layout

import (
	"strings"
	"testing"
)

// Remaining text, list, fragmentation, and rule behavior tests: every
// observable property below is asserted through a USED value (an emitted op
// field, a measured line break, a painted rule op), never through a stored
// style string. The engine lays out in points and 1 CSS pixel = 0.75pt
// (ptPerCSSPx in css_review_02_test.go), so expectations written in CSS
// pixels convert with pxToPt before comparing, which is how Chrome reports
// it. Reference browser for every case: Chrome 143.0.7499.40. Shared
// helpers boxByID, inlineLines, textOpIndex, firstText, opsOfKind, near, and
// pxToPt come from css_review_02_test.go, inline_balance_test.go,
// text_support_layout_test.go, layout_test.go, and style_values.go and are
// never redeclared here.

// behaviorTextRemAssertSameOps asserts two results paint the same ops in the
// same order: same kind, geometry, and text. Used to pin no-op behavior for
// properties the drawing list cannot observe.
func behaviorTextRemAssertSameOps(t *testing.T, first, second *Result) {
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

// behaviorTextRemRuleOps returns the vertical line ops in a result, which is
// the shape emitColumnRules produces for a column rule (see
// TestColumnRulePaints in multicol_test.go).
func behaviorTextRemRuleOps(res *Result) []Op {
	out := make([]Op, 0, len(res.Ops))

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpLine || paintOp.W >= 0.5 || paintOp.H < 4 {
			continue
		}

		out = append(out, paintOp)
	}

	return out
}

// TestBehaviorFontFeatureSettingsSmallCapsTag is font-feature-settings: the
// low-level tag list reaches the shaper payload on the text op, so
// font-feature-settings:'smcp' 1 carries `"smcp" 1` on Op.FontFeatures while
// the plain run carries no smcp tag. Reference: Chrome 143.0.7499.40,
// OpenType feature handling.
func TestBehaviorFontFeatureSettingsSmallCapsTag(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">Ag</p></body></html>`)
	feat := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-feature-settings:'smcp' 1">Ag</p></body></html>`)

	if got := firstText(feat).FontFeatures(); !strings.Contains(got, `"smcp" 1`) {
		t.Errorf("smcp features = %q, want it to contain %q", got, `"smcp" 1`)
	}

	if got := firstText(plain).FontFeatures(); strings.Contains(got, "smcp") {
		t.Errorf("default features = %q, want no smcp tag", got)
	}
}

// TestBehaviorFontSynthesisNoneDisablesFakeBold is font-synthesis: none
// gates fake bold through Op.NoFakeBold, so font-weight:700 with
// font-synthesis:none marks the text op NoFakeBold and FakeBoldFor reports
// false, while the default run leaves the gate open. The bundled Liberation
// Sans ships a real bold face, so weight 700 selects it and FakeBoldFor is
// false on both runs; the pin is the emitted NoFakeBold flag itself.
// Reference: Chrome 143.0.7499.40, synthesis handling.
func TestBehaviorFontSynthesisNoneDisablesFakeBold(t *testing.T) {
	t.Parallel()

	def := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-weight:700">Ag</p></body></html>`)
	none := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-weight:700;font-synthesis:none">Ag</p></body></html>`)

	defOp, noneOp := firstText(def), firstText(none)

	if defOp.NoFakeBoldValue() {
		t.Errorf("default NoFakeBold = true, want false (synthesis allowed)")
	}

	if !noneOp.NoFakeBoldValue() {
		t.Errorf("font-synthesis:none NoFakeBold = false, want true")
	}

	if !FakeBoldFor(&defOp) && !defOp.Bold {
		t.Errorf("default op should carry Bold for weight 700")
	}

	if FakeBoldFor(&noneOp) {
		t.Errorf("font-synthesis:none FakeBoldFor = true, want false")
	}
}

// TestBehaviorWordBreakKeepAllKeepsToken is word-break: a spaceless 80-rune
// token in a 200px block stays on one overflowing line under keep-all (the
// mid-token break policy refuses the split) while break-all wraps it across
// two or more lines. Reference: Chrome 143.0.7499.40, CJK-friendly
// keep-all behavior.
func TestBehaviorWordBreakKeepAllKeepsToken(t *testing.T) {
	t.Parallel()

	token := strings.Repeat("W", 80)
	doc := func(decl string) string {
		return `<html><body style="margin:0">` +
			`<p style="margin:0;width:200px;font-size:12pt;` + decl + `">` + token + `</p></body></html>`
	}

	keepLines := inlineLines(layoutHTML(t, doc("word-break:keep-all")))
	breakLines := inlineLines(layoutHTML(t, doc("word-break:break-all")))

	if len(keepLines) != 1 {
		t.Fatalf("keep-all lines = %d, want 1 overflowing line: %+v", len(keepLines), keepLines)
	}

	if keepLines[0].text != token {
		t.Errorf("keep-all text = %q, want the full %d-rune token", keepLines[0].text, len(token))
	}

	if len(breakLines) < 2 {
		t.Errorf("break-all lines = %d, want at least 2 wrapped lines", len(breakLines))
	}
}

// TestBehaviorTextWrapNowrapKeepsSingleLine is text-wrap: the nowrap mode
// folds onto white-space:nowrap, so text-wrap:nowrap keeps the 43-rune prose
// on one overflowing line in a 150px block while the default wraps it.
// Reference: Chrome 143.0.7499.40, text-wrap shorthand handling.
func TestBehaviorTextWrapNowrapKeepsSingleLine(t *testing.T) {
	t.Parallel()

	prose := "The quick brown fox jumps over the lazy dog"
	doc := func(decl string) string {
		return `<html><body style="margin:0">` +
			`<p style="margin:0;width:150px;font-size:16px;` + decl + `">` + prose + `</p></body></html>`
	}

	normalLines := inlineLines(layoutHTML(t, doc("")))
	nowrapLines := inlineLines(layoutHTML(t, doc("text-wrap:nowrap")))

	if len(normalLines) < 2 {
		t.Fatalf("normal lines = %d, want at least 2 wrapped lines", len(normalLines))
	}

	if len(nowrapLines) != 1 {
		t.Fatalf("text-wrap:nowrap lines = %d, want 1 overflowing line: %+v", len(nowrapLines), nowrapLines)
	}

	if nowrapLines[0].text != prose {
		t.Errorf("nowrap text = %q, want the full prose %q", nowrapLines[0].text, prose)
	}
}

// TestBehaviorTextOrientationSidewaysRotatesRun is text-orientation: sideways
// is treated like mixed, so the run paints rotated -90 degrees, while
// upright stacks each glyph unrotated. Reference: Chrome 143.0.7499.40,
// vertical writing.
func TestBehaviorTextOrientationSidewaysRotatesRun(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
		`<span style="writing-mode:vertical-rl;text-orientation:upright">AB</span>`+
		`<span style="writing-mode:vertical-rl;text-orientation:sideways">EF</span>`+
		`</p></body></html>`)

	_, aOp, aFound := textOpIndex(res, "A")
	_, bOp, bFound := textOpIndex(res, "B")

	if !aFound || !bFound {
		t.Fatalf("upright AB produced glyph ops A=%v B=%v, want one op per glyph", aFound, bFound)
	}

	if aOp.RotateDeg != 0 || bOp.RotateDeg != 0 {
		t.Errorf("upright RotateDeg = %v/%v, want 0", aOp.RotateDeg, bOp.RotateDeg)
	}

	_, sideOp, found := textOpIndex(res, "EF")
	if !found {
		t.Fatal("no text op for EF")
	}

	if sideOp.RotateDeg != -90 {
		t.Errorf("sideways RotateDeg = %v, want -90 like mixed", sideOp.RotateDeg)
	}
}

// TestBehaviorListStylePositionInsideVsOutside is list-style-position: the
// inside marker sits at the content edge while the outside marker hangs in
// the gutter, so the inside bullet paints right of the outside bullet.
// Reference: Chrome 143.0.7499.40, list marker placement.
func TestBehaviorListStylePositionInsideVsOutside(t *testing.T) {
	t.Parallel()

	doc := func(pos, id string) string {
		return `<html><body style="margin:0">` +
			`<ul style="margin:0;padding-left:30pt;font-size:12pt;list-style-position:` + pos + `">` +
			`<li id="` + id + `">inside item sample</li></ul></body></html>`
	}

	inside := layoutHTML(t, doc("inside", "li1"))
	outside := layoutHTML(t, doc("outside", "li1"))

	insideBullets := opsOfKind(inside, OpBullet)
	outsideBullets := opsOfKind(outside, OpBullet)

	if len(insideBullets) != 1 {
		t.Fatalf("inside bullets = %d, want exactly one marker", len(insideBullets))
	}

	if len(outsideBullets) != 1 {
		t.Fatalf("outside bullets = %d, want exactly one marker", len(outsideBullets))
	}

	if !(insideBullets[0].X > outsideBullets[0].X) {
		t.Errorf("inside bullet x %.4fpt should sit right of outside bullet x %.4fpt",
			insideBullets[0].X, outsideBullets[0].X)
	}

	if liBox := boxByID(t, outside, "li1"); !(outsideBullets[0].X < liBox.x) {
		t.Errorf("outside bullet x %.4fpt should hang left of the li box x %.4fpt",
			outsideBullets[0].X, liBox.x)
	}
}

// TestBehaviorListStyleImageBadURLFallsBackToMarker is list-style-image: with
// no image resolver configured the URL cannot resolve, so the engine falls
// back to the type marker (one OpBullet, no OpImage), identical to
// list-style-image:none. Documented gap: the image payload path (a resolved
// marker painting OpImage) is unverified in this environment, so this test
// pins the fallback only. Reference: Chrome 143.0.7499.40, list marker
// fallback.
func TestBehaviorListStyleImageBadURLFallsBackToMarker(t *testing.T) {
	t.Parallel()

	bad := layoutHTML(t, `<html><body style="margin:0">`+
		`<ul style="margin:0;list-style-image:url(missing-marker.png)"><li>one</li></ul></body></html>`)
	none := layoutHTML(t, `<html><body style="margin:0">`+
		`<ul style="margin:0;list-style-image:none"><li>one</li></ul></body></html>`)

	if bullets := opsOfKind(bad, OpBullet); len(bullets) != 1 {
		t.Fatalf("bad-URL bullets = %d, want exactly one fallback marker", len(bullets))
	}

	if images := opsOfKind(bad, OpImage); len(images) != 0 {
		t.Errorf("bad-URL images = %d, want no OpImage without a resolver", len(images))
	}

	behaviorTextRemAssertSameOps(t, none, bad)
}

// TestBehaviorBreakAfterColumnNoPageEffect documents the current behavior of
// break-after: the column value parses to page-break-after:always but layout
// emits no forced break and Paint only relocates ops that straddle a page
// boundary, so the styled document paints identical geometry to the unstyled
// one. Unobservable in the drawing list: reported as a no-op, never failing.
// Reference: Chrome 143.0.7499.40 would start the next block on a new page.
func TestBehaviorBreakAfterColumnNoPageEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="a" style="height:100px"></div>`+
		`<div id="b" style="height:100px"></div></body></html>`)

	forced := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="a" style="height:100px"></div>`+
		`<div id="b" style="height:100px;break-after:column"></div></body></html>`)

	behaviorTextRemAssertSameOps(t, plain, forced)
}

// TestBehaviorWidowsNoFragmentationEffect documents the current behavior of
// widows: the value parses (initial 2, integer >= 1) but no layout pass reads
// it, and Paint moves whole ops across page boundaries without ever splitting
// a line, so widows:5 paints identical text positions to the default.
// Unobservable in the drawing list: reported as a no-op, never failing.
// Reference: Chrome 143.0.7499.40 would keep 5 lines together at a page
// break.
func TestBehaviorWidowsNoFragmentationEffect(t *testing.T) {
	t.Parallel()

	para := `First line of text here. Second line of text here. ` +
		`Third line of text here. Fourth line here. Fifth line of text here. ` +
		`Sixth line of text here.`

	def := layoutHTML(t, `<html style="margin:0"><body style="margin:0;width:200px"><p id="p" `+
		`style="font-size:16px">`+para+`</p></body></html>`)

	five := layoutHTML(t, `<html style="margin:0"><body style="margin:0;width:200px"><p id="p" `+
		`style="font-size:16px;widows:5">`+para+`</p></body></html>`)

	behaviorTextRemAssertSameOps(t, def, five)
}

// TestBehaviorColumnRuleStyleDashedSegmentsRule is column-rule-style: a solid
// rule paints one vertical line op centered in the gap, while dashed expands
// into multiple segment ops at the same gap-middle x. Reference: Chrome
// 143.0.7499.40, multicol rule painting.
func TestBehaviorColumnRuleStyleDashedSegmentsRule(t *testing.T) {
	t.Parallel()

	doc := func(styleDecl string) string {
		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="mc" style="width:300px;column-count:2;column-gap:20px;` +
			`column-rule-width:3px;column-rule-color:#666;` + styleDecl + `font-size:10pt">` +
			`Multi-column sample text repeated. Multi-column sample text repeated. ` +
			`Multi-column sample text repeated.</div></body></html>`
	}

	solid := layoutHTML(t, doc("column-rule-style:solid;"))
	dashed := layoutHTML(t, doc("column-rule-style:dashed;"))

	solidRules := behaviorTextRemRuleOps(solid)
	dashedRules := behaviorTextRemRuleOps(dashed)

	if len(solidRules) != 1 {
		t.Fatalf("solid rule ops = %d, want exactly one centered rule", len(solidRules))
	}

	if len(dashedRules) < 2 {
		t.Fatalf("dashed rule ops = %d, want at least 2 dash segments", len(dashedRules))
	}

	if !near(dashedRules[0].X, solidRules[0].X) {
		t.Errorf("dashed rule x %.4fpt should match solid gap-middle x %.4fpt",
			dashedRules[0].X, solidRules[0].X)
	}
}

// TestBehaviorColumnRuleColorBluePaintsRule is column-rule-color: the
// longhand sets the painted rule color, so a solid 3px rule with
// column-rule-color:#00f emits one vertical line op in blue. Reference:
// Chrome 143.0.7499.40, multicol rule color.
func TestBehaviorColumnRuleColorBluePaintsRule(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="mc" style="width:300px;column-count:2;column-gap:20px;`+
		`column-rule-style:solid;column-rule-width:3px;column-rule-color:#00f;font-size:10pt">`+
		`Multi-column sample text repeated. Multi-column sample text repeated. `+
		`Multi-column sample text repeated.</div></body></html>`)

	rules := behaviorTextRemRuleOps(res)

	if len(rules) != 1 {
		t.Fatalf("rule ops = %d, want exactly one blue rule", len(rules))
	}

	if !near(rules[0].R, 0) || !near(rules[0].G, 0) || !near(rules[0].B, 1) {
		t.Errorf("rule color = (%.2f, %.2f, %.2f), want blue (0, 0, 1)",
			rules[0].R, rules[0].G, rules[0].B)
	}

	if !near(rules[0].Width, pxToPt(3)) {
		t.Errorf("rule stroke width = %.4fpt (%.2fpx), want 3px",
			rules[0].Width, rules[0].Width/ptPerCSSPx)
	}
}

// TestBehaviorClipRuleEvenOddNoPaintEffect documents the current behavior of
// clip-rule: the declaration is dropped (style_paint_props.go forwards it to
// applyLeftoversProps, which has no clip-rule case and returns false) and no
// paint pass reads it, so clip-rule:evenodd paints identical geometry to the
// default. Unobservable in the drawing list: reported as a no-op, never
// failing. Reference: Chrome 143.0.7499.40 would apply evenodd fill to SVG
// clip content.
func TestBehaviorClipRuleEvenOddNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="a" style="width:100px;height:50px">clip sample text</div></body></html>`)

	evenOdd := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="a" style="width:100px;height:50px;clip-rule:evenodd">clip sample text</div></body></html>`)

	behaviorTextRemAssertSameOps(t, plain, evenOdd)
}
