package layout

import (
	"testing"

	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
)

// TestSystemFallbackFaceForCJK checks the layout hook end to end: with the
// fallback on, a CJK rune that no bundled face covers resolves to an installed
// face that covers it, and with it off the lookup does not return one.
func TestSystemFallbackFaceForCJK(t *testing.T) {
	pdf.SetSystemFallback(true)
	t.Cleanup(func() { pdf.SetSystemFallback(false) })

	if pdf.FindSystemGlyph('か', 400, false) == nil {
		t.Skip("no system font with Japanese coverage on this host")
	}

	eng := &engine{}
	style := &ResolvedStyle{FontWeight: 400}

	face := eng.registryGlyphFallback(style, 'か')
	if face == nil || face.GlyphID('か') == 0 {
		t.Fatalf("registryGlyphFallback(か) = %v, want a covering system face", face)
	}

	if latin := eng.registryGlyphFallback(style, 'a'); latin != nil {
		t.Fatalf("registryGlyphFallback(a) = %v, want nil: Latin must not use the system fallback", latin)
	}

	pdf.SetSystemFallback(false)

	if off := eng.registryGlyphFallback(style, 'か'); off != nil {
		t.Fatalf("registryGlyphFallback(か) with fallback off = %v, want nil", off)
	}
}
