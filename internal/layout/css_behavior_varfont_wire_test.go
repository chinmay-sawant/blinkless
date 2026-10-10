package layout

import (
	"os"
	"path/filepath"
	"testing"

	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
)

// Variable-font wire observability on the vendored file faces.
// Reference: Chrome 143.0.7499.40 instances fvar axes and selects CPAL
// palettes; static faces keep the default instance and CSS color.
//
// Capability split (checked in each test setup):
//   - testdata/fonts/implemented-audit/GowkVar-VF.ttf carries fvar
//     wght/opsz/wdth and no COLR/CPAL, so it covers
//     font-variation-settings and font-optical-sizing but not font-palette.
//   - testdata/fonts/implemented-audit/PaletteDemo-Regular.ttf carries
//     COLR+CPAL and no fvar, so it covers font-palette but not the axes.
// PaletteDemo alone cannot cover all three; both TTF faces are needed.
// Loader note: ScanFontDirs reads .ttf/.otf only
// (internal/fonts/registry.go), @font-face skips .woff2/.eot by extension
// (internal/convert/prepare/styles.go), and ParseFontBytes rejects woff2
// signatures (internal/fonts/woff.go), so no Brotli dep is added. Both
// faces below load through ParseTTF.
//
// Every assertion compares used-face pointers, advance widths, or paint
// fills numerically, never stored style strings.

// loadPaletteDemoFile parses the vendored COLR/CPAL file face.
func loadPaletteDemoFile(t *testing.T) *pdf.Font {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "fonts", "implemented-audit", "PaletteDemo-Regular.ttf")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read PaletteDemo-Regular: %v", err)
	}

	face, err := pdf.ParseTTF(data)
	if err != nil {
		t.Fatalf("ParseTTF PaletteDemo: %v", err)
	}

	return face
}

// wirePaletteFileRegistry registers the file COLR/CPAL face under its
// name-table family so CSS font-family resolves it through Lookup.
func wirePaletteFileRegistry(t *testing.T) *pdf.Registry {
	t.Helper()

	face := loadPaletteDemoFile(t)
	if !face.HasColorPalette() {
		t.Fatal("PaletteDemo-Regular.ttf must advertise COLR+CPAL")
	}

	if face.HasVariationAxes() {
		t.Fatal("PaletteDemo-Regular.ttf must have no fvar; axes need GowkVar")
	}

	reg := pdf.NewRegistry()
	reg.AddFamilyAlias("Palette Demo", face)

	return reg
}

// TestBehaviorVarFontWireWghtDiverges: font-variation-settings 'wght' 900 on
// the GowkVar file face resolves a different used face whose advance is
// wider than the default instance. Reference: Chrome 143.0.7499.40.
func TestBehaviorVarFontWireWghtDiverges(t *testing.T) {
	t.Parallel()

	base := loadGowkVar(t)
	if !base.HasVariationAxes() {
		t.Fatal("GowkVar-VF.ttf must advertise fvar")
	}

	if base.HasColorPalette() {
		t.Fatal("GowkVar-VF.ttf must have no COLR+CPAL; palette needs PaletteDemo")
	}

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

// TestBehaviorVarFontWireOpticalAutoDiverges: font-optical-sizing:auto at
// 72pt instances the opsz axis on the GowkVar file face, so the used face
// and its advance differ from optical-sizing:none at the same size.
// Reference: Chrome 143.0.7499.40.
func TestBehaviorVarFontWireOpticalAutoDiverges(t *testing.T) {
	t.Parallel()

	base := loadGowkVar(t)
	if !base.HasVariationAxes() {
		t.Fatal("GowkVar-VF.ttf must advertise fvar")
	}

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

// TestBehaviorVarFontWirePaletteFileDarkDiverges: font-palette dark on the
// vendored PaletteDemo-Regular.ttf file face paints the selected CPAL color
// instead of the CSS color, while normal keeps the CSS color.
// Reference: Chrome 143.0.7499.40.
func TestBehaviorVarFontWirePaletteFileDarkDiverges(t *testing.T) {
	t.Parallel()

	reg := wirePaletteFileRegistry(t)

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
