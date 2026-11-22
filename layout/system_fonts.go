package layout

import "github.com/chinmay-sawant/blinkless/internal/fonts"

// SetSystemFontFallback turns the operating-system CJK font fallback on or
// off for the whole process. It is on by default: when a CJK character is not
// covered by the document's fonts or the bundled faces, layout borrows an
// installed system face for it (Yu Gothic or Meiryo on Windows, Hiragino on
// macOS, common CJK families on Linux). Set the environment variable
// BLINKLESS_SYSTEM_FONTS=off to start with it disabled.
func SetSystemFontFallback(enabled bool) { fonts.SetSystemFallback(enabled) }

// SystemFontFallbackEnabled reports whether the operating-system CJK font
// fallback is on.
func SystemFontFallbackEnabled() bool { return fonts.SystemFallbackEnabled() }
