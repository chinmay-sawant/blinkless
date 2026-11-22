package fonts

import "testing"

// TestSystemFallbackHostCJK exercises the real host: on a machine with a
// Japanese system font (Windows, macOS, WSL, or a Linux box with a TrueType
// CJK family) hiragana, kanji, and Hangul or Chinese must resolve to a face
// that actually covers them, and Latin text must never reach the fallback.
// Machines without such a font skip instead of failing.
func TestSystemFallbackHostCJK(t *testing.T) {
	SetSystemFallback(true)

	hira := FindSystemGlyph('か', 400, false)
	if hira == nil {
		t.Skip("no system font with Japanese coverage on this host")
	}

	for _, r := range []rune{'か', 'ナ', '漢'} {
		face := FindSystemGlyph(r, 400, false)
		if face == nil || face.GlyphID(r) == 0 {
			t.Errorf("U+%04X: system fallback returned %v, want a covering face", r, face)
		}
	}

	if bold := FindSystemGlyph('か', 700, false); bold == nil || bold.GlyphID('か') == 0 {
		t.Errorf("bold hiragana: got %v, want a covering face", bold)
	}

	if FindSystemGlyph('a', 400, false) != nil {
		t.Error("Latin must not reach the system fallback")
	}

	// The extracted face must be a standalone SFNT other shapers can load.
	if data := hira.Bytes(); len(data) < sfntHeaderSize || IsTTC(data) {
		t.Errorf("face bytes are not a standalone SFNT (len %d)", len(data))
	}
}
