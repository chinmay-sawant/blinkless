package layout

import (
	"testing"

	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
)

// Variable-font CSS behavior: font-variation-settings, font-optical-sizing,
// and font-palette observed through the display list on capable faces.
// Reference: Chrome 143.0.7499.40 instances wght/opsz axes and paints CPAL
// palette colors. The static bundled Liberation faces have no fvar or
// COLR+CPAL tables, so the same declarations stay no-ops there (see
// TestBehaviorFontVariationSettingsStaticKeepsFace,
// TestBehaviorFontOpticalSizingStaticKeepsFace, and
// TestBehaviorFontPaletteStaticKeepsColor). These tests cover the capable-face
// path end to end: every assertion compares used-face pointers, advance
// widths, or paint colors numerically, never stored strings.

// varfontGowkRegistry registers one parsed GowkVar variable face under
// "Gowk Var" so CSS font-family resolves it through Lookup.
func varfontGowkRegistry(t *testing.T) *pdf.Registry {
	t.Helper()

	reg := pdf.NewRegistry()
	reg.AddFamilyAlias("Gowk Var", loadGowkVar(t))

	return reg
}

// TestBehaviorVarFontVariationSettingsInstancesAdvance: font-variation-settings
// "wght" 900 on the GowkVar face resolves a different used face whose advance
// is wider than the default instance.
func TestBehaviorVarFontVariationSettingsInstancesAdvance(t *testing.T) {
	t.Parallel()

	reg := varfontGowkRegistry(t)

	plain := layoutHTMLRegistry(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-family:'Gowk Var';font-size:20pt">AAAA</p></body></html>`, reg)
	varied := layoutHTMLRegistry(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-family:'Gowk Var';font-size:20pt;`+
		`font-variation-settings:'wght' 900">AAAA</p></body></html>`, reg)

	plainOp, variedOp := firstText(plain), firstText(varied)

	if plainOp.Font == nil || variedOp.Font == nil {
		t.Fatalf("faces = %v vs %v, want non-nil used faces", plainOp.Font != nil, variedOp.Font != nil)
	}

	if plainOp.Font == variedOp.Font {
		t.Error("wght 900 resolved the default instance, want an instanced face")
	}

	if plainOp.W <= 0 || variedOp.W <= 0 {
		t.Fatalf("advances = %.4fpt vs %.4fpt, want both positive", plainOp.W, variedOp.W)
	}

	if variedOp.W <= plainOp.W {
		t.Errorf("wght 900 advance %.4fpt, want wider than default %.4fpt", variedOp.W, plainOp.W)
	}
}

// TestBehaviorVarFontOpticalSizingAutoInstancesOpsz: font-optical-sizing:auto
// at 72pt instances the opsz axis on the GowkVar face, so the used face and
// its advance differ from optical-sizing:none at the same size.
func TestBehaviorVarFontOpticalSizingAutoInstancesOpsz(t *testing.T) {
	t.Parallel()

	reg := varfontGowkRegistry(t)

	none := layoutHTMLRegistry(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-family:'Gowk Var';font-size:72pt;font-optical-sizing:none">AAAA</p></body></html>`, reg)
	auto := layoutHTMLRegistry(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-family:'Gowk Var';font-size:72pt;font-optical-sizing:auto">AAAA</p></body></html>`, reg)

	noneOp, autoOp := firstText(none), firstText(auto)

	if noneOp.Font == nil || autoOp.Font == nil {
		t.Fatalf("faces = %v vs %v, want non-nil used faces", noneOp.Font != nil, autoOp.Font != nil)
	}

	if noneOp.Font == autoOp.Font {
		t.Error("optical-sizing:auto at 72pt resolved the default instance, want an opsz instance")
	}

	if noneOp.W <= 0 || autoOp.W <= 0 {
		t.Fatalf("advances = %.4fpt vs %.4fpt, want both positive", noneOp.W, autoOp.W)
	}

	if near(noneOp.W, autoOp.W) {
		t.Errorf("opsz advances = %.4fpt vs %.4fpt, want them to differ", noneOp.W, autoOp.W)
	}
}

// TestBehaviorVarFontPaletteDarkDiffersFromNormal: font-palette dark on the
// Palette Demo COLR+CPAL face paints the selected palette color instead of
// the CSS color, while normal keeps the CSS color.
func TestBehaviorVarFontPaletteDarkDiffersFromNormal(t *testing.T) {
	t.Parallel()

	reg := paletteDemoRegistry(t)

	normal := layoutHTMLRegistry(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-family:'Palette Demo';font-size:18pt;`+
		`color:#111111;font-palette:normal">Ag</p></body></html>`, reg)
	dark := layoutHTMLRegistry(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-family:'Palette Demo';font-size:18pt;`+
		`color:#111111;font-palette:dark">Ag</p></body></html>`, reg)

	normalOp, darkOp := firstText(normal), firstText(dark)

	if colorsNear(normalOp, darkOp, 0.05) {
		t.Errorf("dark fill (%.3f,%.3f,%.3f) must differ from normal (%.3f,%.3f,%.3f)",
			darkOp.R, darkOp.G, darkOp.B, normalOp.R, normalOp.G, normalOp.B)
	}

	if normalOp.R > 0.2 || normalOp.G > 0.2 || normalOp.B > 0.2 {
		t.Errorf("normal fill (%.3f,%.3f,%.3f), want near CSS #111", normalOp.R, normalOp.G, normalOp.B)
	}
}
