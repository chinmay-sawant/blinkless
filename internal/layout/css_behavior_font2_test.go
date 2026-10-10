package layout

import (
	"path/filepath"
	"strings"
	"testing"

	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
)

// Font2 behavior tests: one test per property, asserted through USED values
// (selected faces, measured advances, op feature tags, paint fills, op sizes),
// never through stored style strings. Points are the engine unit and 1 CSS
// pixel = 0.75pt (ptPerCSSPx in css_review_02_test.go). Reference browser for
// every case: Chrome 143.0.7499.40.
//
// Shared helpers (boxByID, near, pxToPt, layoutHTML, layoutHTMLRegistry,
// sheet, opsOfKind, firstText, firstTextX, textOpIndex, joinedPaintText,
// textOpsContain) come from other files in this same package and are reused
// here, never redeclared. Local helpers use the behaviorFont2 prefix only.

// behaviorFont2TrkTag is the Turkic OpenType language tag used by the
// language-override test below.
const behaviorFont2TrkTag = "TRK"

// behaviorFont2AssertSameOps pins no-op behavior: two results must paint the
// same ops in the same order with identical geometry and text.
func behaviorFont2AssertSameOps(t *testing.T, first, second *Result) {
	t.Helper()

	if len(first.Ops) != len(second.Ops) {
		t.Fatalf("op count = %d vs %d, want identical geometry", len(first.Ops), len(second.Ops))
	}

	for i := range first.Ops {
		a, b := first.Ops[i], second.Ops[i]
		if a.Kind != b.Kind || a.Text != b.Text ||
			!near(a.X, b.X) || !near(a.Y, b.Y) ||
			!near(a.W, b.W) || !near(a.H, b.H) {
			t.Fatalf("op %d differs: %+v vs %+v, want identical geometry", i, a, b)
		}
	}
}

// TestBehaviorFontLanguageOverrideCarriesTag: font-language-override carries
// the OpenType language tag on the used text op, while the default run
// carries no override. Reference: Chrome 143.0.7499.40, shaping language
// handling with the bundled Liberation faces.
func TestBehaviorFontLanguageOverrideCarriesTag(t *testing.T) {
	t.Parallel()

	tagged := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-language-override:'TRK'">hello</p></body></html>`)
	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">hello</p></body></html>`)

	taggedOp, plainOp := firstText(tagged), firstText(plain)

	if taggedOp.Font == nil || taggedOp.Text == "" {
		t.Fatalf("tagged run has no face or text, got font=%v text=%q", taggedOp.Font, taggedOp.Text)
	}

	if got := taggedOp.TextLanguage(); got != behaviorFont2TrkTag {
		t.Errorf("language override = %q, want %q", got, behaviorFont2TrkTag)
	}

	if got := plainOp.TextLanguage(); got != "" {
		t.Errorf("default language override = %q, want empty (no override)", got)
	}

	if taggedOp.W <= 0 || plainOp.W <= 0 {
		t.Errorf("advances = %.4fpt and %.4fpt, want both positive", taggedOp.W, plainOp.W)
	}
}

// TestBehaviorFontOpticalSizingStaticKeepsFace: on the static bundled faces
// font-optical-sizing is a no-op, so auto and none resolve the same face with
// the same advance. Reference: Chrome 143.0.7499.40 would instance opsz on a
// variable face. Documented gap: bundled Liberation faces have no fvar table,
// so both values return the default instance (see
// TestResolveFontVariantsStaticBundledFaces and TestOpticalSizingAutoSetsOpsz
// for the variable-face path).
func TestBehaviorFontOpticalSizingStaticKeepsFace(t *testing.T) {
	t.Parallel()

	auto := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-optical-sizing:auto">hello</p></body></html>`)
	none := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-optical-sizing:none">hello</p></body></html>`)

	autoOp, noneOp := firstText(auto), firstText(none)

	if autoOp.Font != noneOp.Font {
		t.Errorf("optical-sizing faces = %p vs %p, want the same static face", autoOp.Font, noneOp.Font)
	}

	if !near(autoOp.W, noneOp.W) {
		t.Errorf("optical-sizing advances = %.4fpt vs %.4fpt, want identical", autoOp.W, noneOp.W)
	}
}

// TestBehaviorFontPaletteStaticKeepsColor: on the static bundled faces
// font-palette is a no-op, so dark keeps the CSS color exactly like normal.
// Reference: Chrome 143.0.7499.40 paints palette colors on COLR/CPAL fonts.
// Documented gap: bundled Liberation faces have no COLR+CPAL tables, so the
// lite fill stays off (see TestFontPaletteCPALFillDiffersFromNormal for the
// palette-demo face path).
func TestBehaviorFontPaletteStaticKeepsColor(t *testing.T) {
	t.Parallel()

	normal := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;color:#111111;font-palette:normal">Ag</p></body></html>`)
	dark := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;color:#111111;font-palette:dark">Ag</p></body></html>`)

	normalOp, darkOp := firstText(normal), firstText(dark)

	if !near(normalOp.R, darkOp.R) || !near(normalOp.G, darkOp.G) || !near(normalOp.B, darkOp.B) {
		t.Errorf("palette fills = (%.3f,%.3f,%.3f) vs (%.3f,%.3f,%.3f), want identical CSS #111",
			normalOp.R, normalOp.G, normalOp.B, darkOp.R, darkOp.G, darkOp.B)
	}

	if normalOp.R > 0.2 || normalOp.G > 0.2 || normalOp.B > 0.2 {
		t.Errorf("normal fill = (%.3f,%.3f,%.3f), want near CSS #111", normalOp.R, normalOp.G, normalOp.B)
	}
}

// TestBehaviorFontSynthesisPositionScalesSub: font-variant-position sub with
// font-synthesis-position auto scales the used size down, while none keeps
// the full size; both runs carry the subs feature tag. Reference: Chrome
// 143.0.7499.40, sub synthesis at 20pt.
func TestBehaviorFontSynthesisPositionScalesSub(t *testing.T) {
	t.Parallel()

	auto := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:20pt;font-variant-position:sub;font-synthesis-position:auto">2</p></body></html>`)
	none := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:20pt;font-variant-position:sub;font-synthesis-position:none">2</p></body></html>`)

	autoOp, noneOp := firstText(auto), firstText(none)

	if autoOp.Size >= noneOp.Size*0.9 {
		t.Errorf("sub auto size %.4fpt should be well below none size %.4fpt", autoOp.Size, noneOp.Size)
	}

	if got := autoOp.FontFeatures(); !strings.Contains(got, `"subs" 1`) {
		t.Errorf("sub auto features = %q, want it to contain %q", got, `"subs" 1`)
	}

	if got := noneOp.FontFeatures(); !strings.Contains(got, `"subs" 1`) {
		t.Errorf("sub none features = %q, want it to contain %q (synthesis gates size only)", got, `"subs" 1`)
	}
}

// TestBehaviorFontSynthesisSmallCapsGatesUppercase: small-caps with
// font-synthesis-small-caps auto paints uppercased at a reduced size, while
// none keeps the source lowercase at full size. Reference: Chrome
// 143.0.7499.40, small-caps synthesis at 16pt.
func TestBehaviorFontSynthesisSmallCapsGatesUppercase(t *testing.T) {
	t.Parallel()

	auto := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:16pt;font-variant-caps:small-caps;font-synthesis-small-caps:auto">ag</p></body></html>`)
	none := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:16pt;font-variant-caps:small-caps;font-synthesis-small-caps:none">ag</p></body></html>`)

	if got := joinedPaintText(auto); got != "AG" {
		t.Errorf("small-caps auto paint text = %q, want %q", got, "AG")
	}

	if got := joinedPaintText(none); got != "ag" {
		t.Errorf("small-caps none paint text = %q, want %q", got, "ag")
	}

	if autoOp, noneOp := firstText(auto), firstText(none); !(autoOp.Size < noneOp.Size) {
		t.Errorf("small-caps auto size %.4fpt should be below none size %.4fpt", autoOp.Size, noneOp.Size)
	}
}

// TestBehaviorFontSynthesisStyleGatesOblique: italic on the upright-only
// Cactus face synthesizes an oblique skew when font-synthesis-style is auto
// and paints upright when none. Reference: Chrome 143.0.7499.40, synthetic
// italics with testdata/fonts/implemented-audit faces.
func TestBehaviorFontSynthesisStyleGatesOblique(t *testing.T) {
	t.Parallel()

	audit := filepath.Join("..", "..", "testdata", "fonts", "implemented-audit")
	reg := pdf.ScanFontDirs([]string{audit})

	auto := layoutHTMLRegistry(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-family:'Cactus Classical Serif';`+
		`font-style:italic;font-synthesis-style:auto">Ag</p></body></html>`, reg)
	none := layoutHTMLRegistry(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-family:'Cactus Classical Serif';`+
		`font-style:italic;font-synthesis-style:none">Ag</p></body></html>`, reg)

	autoOp, noneOp := firstText(auto), firstText(none)

	if !autoOp.FakeOblique {
		t.Errorf("italic auto FakeOblique = false, want true on the upright-only face")
	}

	if noneOp.FakeOblique {
		t.Errorf("italic none FakeOblique = true, want false (synthesis forbidden)")
	}

	if autoOp.Font != noneOp.Font {
		t.Errorf("italic faces = %p vs %p, want the same upright face", autoOp.Font, noneOp.Font)
	}
}

// TestBehaviorFontSynthesisWeightGatesFakeBold: font-weight 700 with
// font-synthesis-weight none marks the used op NoFakeBold, while the default
// leaves the gate open. Reference: Chrome 143.0.7499.40, synthesis handling.
// The bundled Liberation Sans ships a real bold face, so both runs carry Bold
// and FakeBoldFor stays false; the pin is the emitted NoFakeBold flag itself.
func TestBehaviorFontSynthesisWeightGatesFakeBold(t *testing.T) {
	t.Parallel()

	def := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-weight:700">Ag</p></body></html>`)
	none := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-weight:700;font-synthesis-weight:none">Ag</p></body></html>`)

	defOp, noneOp := firstText(def), firstText(none)

	if defOp.NoFakeBoldValue() {
		t.Errorf("default NoFakeBold = true, want false (synthesis allowed)")
	}

	if !noneOp.NoFakeBoldValue() {
		t.Errorf("synthesis-weight:none NoFakeBold = false, want true")
	}

	if FakeBoldFor(&noneOp) {
		t.Errorf("synthesis-weight:none FakeBoldFor = true, want false")
	}

	if !defOp.Bold || !noneOp.Bold {
		t.Errorf("bold flags = %v/%v, want both true for weight 700", defOp.Bold, noneOp.Bold)
	}
}

// TestBehaviorFontVariantAlternatesEmitsHistTag: historical-forms reaches the
// shaper payload as the hist tag, while the plain run carries no hist tag.
// Reference: Chrome 143.0.7499.40, OpenType alternate handling.
func TestBehaviorFontVariantAlternatesEmitsHistTag(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">Ag</p></body></html>`)
	hist := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-variant-alternates:historical-forms">Ag</p></body></html>`)

	if got := firstText(hist).FontFeatures(); !strings.Contains(got, `"hist" 1`) {
		t.Errorf("historical-forms features = %q, want it to contain %q", got, `"hist" 1`)
	}

	if got := firstText(plain).FontFeatures(); strings.Contains(got, "hist") {
		t.Errorf("default features = %q, want no hist tag", got)
	}
}

// TestBehaviorFontVariantCapsEmitsSmcpTag: small-caps reaches the shaper
// payload as the smcp tag, while the plain run carries no smcp tag.
// Reference: Chrome 143.0.7499.40, caps handling with Liberation faces.
func TestBehaviorFontVariantCapsEmitsSmcpTag(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">Ag</p></body></html>`)
	caps := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-variant-caps:small-caps">Ag</p></body></html>`)

	if got := firstText(caps).FontFeatures(); !strings.Contains(got, `"smcp" 1`) {
		t.Errorf("small-caps features = %q, want it to contain %q", got, `"smcp" 1`)
	}

	if got := firstText(plain).FontFeatures(); strings.Contains(got, "smcp") {
		t.Errorf("default features = %q, want no smcp tag", got)
	}
}

// TestBehaviorFontVariantEastAsianEmitsJp78Tag: jis78 reaches the shaper
// payload as the jp78 tag, while the plain run carries no jp78 tag.
// Reference: Chrome 143.0.7499.40, East Asian feature handling.
func TestBehaviorFontVariantEastAsianEmitsJp78Tag(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">Ag</p></body></html>`)
	east := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-variant-east-asian:jis78">Ag</p></body></html>`)

	if got := firstText(east).FontFeatures(); !strings.Contains(got, `"jp78" 1`) {
		t.Errorf("jis78 features = %q, want it to contain %q", got, `"jp78" 1`)
	}

	if got := firstText(plain).FontFeatures(); strings.Contains(got, "jp78") {
		t.Errorf("default features = %q, want no jp78 tag", got)
	}
}

// TestBehaviorFontVariantEmojiAppliesFill: emoji presentation paints the lite
// gold fill over CSS color, while text presentation keeps the CSS color; the
// emoji run also keeps at least the text advance. Reference: Chrome
// 143.0.7499.40, emoji presentation on U+263A at 18pt.
func TestBehaviorFontVariantEmojiAppliesFill(t *testing.T) {
	t.Parallel()

	text := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:18pt;color:#111111;font-variant-emoji:text">`+"☺"+`</p></body></html>`)
	emoji := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:18pt;color:#111111;font-variant-emoji:emoji">`+"☺"+`</p></body></html>`)

	textOp, emojiOp := firstText(text), firstText(emoji)

	if textOp.R > 0.2 || textOp.G > 0.2 || textOp.B > 0.2 {
		t.Errorf("text fill = (%.3f,%.3f,%.3f), want near CSS #111", textOp.R, textOp.G, textOp.B)
	}

	if emojiOp.R < 0.8 || emojiOp.G < 0.5 || emojiOp.B > 0.3 {
		t.Errorf("emoji fill = (%.3f,%.3f,%.3f), want gold-ish", emojiOp.R, emojiOp.G, emojiOp.B)
	}

	if emojiOp.W+0.01 < textOp.W {
		t.Errorf("emoji width %.4fpt should be at least text width %.4fpt", emojiOp.W, textOp.W)
	}
}

// TestBehaviorFontVariantLigaturesDisablesLiga: no-common-ligatures carries
// liga-off on the text op for the shaper, while the default run carries no
// liga tag. Reference: Chrome 143.0.7499.40, ligature feature handling.
func TestBehaviorFontVariantLigaturesDisablesLiga(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">fi</p></body></html>`)
	off := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-variant-ligatures:no-common-ligatures">fi</p></body></html>`)

	if got := firstText(off).FontFeatures(); !strings.Contains(got, `"liga" 0`) {
		t.Errorf("no-common-ligatures features = %q, want it to contain %q", got, `"liga" 0`)
	}

	if got := firstText(plain).FontFeatures(); strings.Contains(got, "liga") {
		t.Errorf("default features = %q, want no liga tag", got)
	}
}

// TestBehaviorFontVariantNumericEmitsTnumTag: tabular-nums reaches the shaper
// payload as the tnum tag, while the plain run carries no tnum tag.
// Reference: Chrome 143.0.7499.40, numeric feature handling.
func TestBehaviorFontVariantNumericEmitsTnumTag(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">123</p></body></html>`)
	tab := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-variant-numeric:tabular-nums">123</p></body></html>`)

	if got := firstText(tab).FontFeatures(); !strings.Contains(got, `"tnum" 1`) {
		t.Errorf("tabular-nums features = %q, want it to contain %q", got, `"tnum" 1`)
	}

	if got := firstText(plain).FontFeatures(); strings.Contains(got, "tnum") {
		t.Errorf("default features = %q, want no tnum tag", got)
	}
}

// TestBehaviorFontVariantPositionEmitsSubsTag: sub reaches the shaper payload
// as the subs tag and scales the used size down under the default auto
// synthesis gate. Reference: Chrome 143.0.7499.40, sub handling at 20pt.
func TestBehaviorFontVariantPositionEmitsSubsTag(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:20pt">2</p></body></html>`)
	sub := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:20pt;font-variant-position:sub">2</p></body></html>`)

	plainOp, subOp := firstText(plain), firstText(sub)

	if got := subOp.FontFeatures(); !strings.Contains(got, `"subs" 1`) {
		t.Errorf("sub features = %q, want it to contain %q", got, `"subs" 1`)
	}

	if got := plainOp.FontFeatures(); strings.Contains(got, "subs") {
		t.Errorf("default features = %q, want no subs tag", got)
	}

	if subOp.Size >= plainOp.Size*0.9 {
		t.Errorf("sub size %.4fpt should be well below plain size %.4fpt", subOp.Size, plainOp.Size)
	}
}

// TestBehaviorFontVariationSettingsStaticKeepsFace: on the static bundled
// faces font-variation-settings is a no-op, so the axis list resolves the
// same face with the same advance as the default. Reference: Chrome
// 143.0.7499.40 would instance wght on a variable face. Documented gap:
// bundled Liberation faces have no fvar table (see
// TestVariationSettingsChangesAdvance for the GowkVar instancing path).
func TestBehaviorFontVariationSettingsStaticKeepsFace(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">hello</p></body></html>`)
	varied := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;font-variation-settings:'wght' 700">hello</p></body></html>`)

	plainOp, variedOp := firstText(plain), firstText(varied)

	if plainOp.Font != variedOp.Font {
		t.Errorf("variation faces = %p vs %p, want the same static face", plainOp.Font, variedOp.Font)
	}

	if !near(plainOp.W, variedOp.W) {
		t.Errorf("variation advances = %.4fpt vs %.4fpt, want identical", plainOp.W, variedOp.W)
	}
}

// TestBehaviorFontWidthCondensesAdvance: font-width condensed scales the used
// advance below the normal width for the same run. Reference: Chrome
// 143.0.7499.40, condensed 75 percent against normal at 20pt.
func TestBehaviorFontWidthCondensesAdvance(t *testing.T) {
	t.Parallel()

	wide := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:20pt;font-width:100%">MMMM</p></body></html>`)
	narrow := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:20pt;font-width:condensed">MMMM</p></body></html>`)

	wideOp, narrowOp := firstText(wide), firstText(narrow)

	if narrowOp.W >= wideOp.W*0.9 {
		t.Errorf("condensed width %.4fpt (%.2fpx) should be below normal %.4fpt (%.2fpx)",
			narrowOp.W, narrowOp.W/ptPerCSSPx, wideOp.W, wideOp.W/ptPerCSSPx)
	}
}

// TestBehaviorRubyAlignNoPaintEffect documents the current behavior of
// ruby-align: the declaration is dropped (style_paint_props.go forwards it to
// applyLeftoversProps, which has no ruby case and returns false) and no paint
// pass reads it, so ruby-align:center paints identical geometry to the
// default. Unobservable in the drawing list: reported as a no-op, never
// failing. Reference: Chrome 143.0.7499.40 would center the ruby annotation.
func TestBehaviorRubyAlignNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p id="a" style="margin:0;font-size:12pt">hello</p></body></html>`)
	styled := layoutHTML(t, `<html><body style="margin:0">`+
		`<p id="a" style="margin:0;font-size:12pt;ruby-align:center">hello</p></body></html>`)

	behaviorFont2AssertSameOps(t, plain, styled)

	plainBox, styledBox := boxByID(t, plain, "a"), boxByID(t, styled, "a")
	if !near(plainBox.w, styledBox.w) || !near(plainBox.x, styledBox.x) {
		t.Errorf("ruby-align boxes = (x %.4fpt w %.4fpt) vs (x %.4fpt w %.4fpt), want identical",
			plainBox.x, plainBox.w, styledBox.x, styledBox.w)
	}
}

// TestBehaviorRubyMergeNoPaintEffect documents the current behavior of
// ruby-merge: the declaration is dropped (style_paint_props.go forwards it to
// applyLeftoversProps, which has no ruby case and returns false) and no paint
// pass reads it, so ruby-merge:collapse paints identical geometry to the
// default. Unobservable in the drawing list: reported as a no-op, never
// failing. Reference: Chrome 143.0.7499.40 would merge adjacent annotations.
func TestBehaviorRubyMergeNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">hello</p></body></html>`)
	styled := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;ruby-merge:collapse">hello</p></body></html>`)

	behaviorFont2AssertSameOps(t, plain, styled)
}

// TestBehaviorRubyOverhangNoPaintEffect documents the current behavior of
// ruby-overhang: the declaration is dropped (style_paint_props.go forwards it
// to applyLeftoversProps, which has no ruby case and returns false) and no
// paint pass reads it, so ruby-overhang:none paints identical geometry to the
// default. Unobservable in the drawing list: reported as a no-op, never
// failing. Reference: Chrome 143.0.7499.40 would clip annotation overhang.
func TestBehaviorRubyOverhangNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">hello</p></body></html>`)
	styled := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;ruby-overhang:none">hello</p></body></html>`)

	behaviorFont2AssertSameOps(t, plain, styled)
}

// TestBehaviorRubyPositionNoPaintEffect documents the current behavior of
// ruby-position: the declaration is dropped (style_paint_props.go forwards it
// to applyLeftoversProps, which has no ruby case and returns false) and no
// paint pass reads it, so ruby-position:under paints identical geometry to
// the default. Unobservable in the drawing list: reported as a no-op, never
// failing. Reference: Chrome 143.0.7499.40 would place the annotation below.
func TestBehaviorRubyPositionNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">hello</p></body></html>`)
	styled := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;ruby-position:under">hello</p></body></html>`)

	behaviorFont2AssertSameOps(t, plain, styled)
}
