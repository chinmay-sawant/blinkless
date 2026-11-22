package layout

import (
	"os"
	"testing"

	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
)

// TestMain pins the layout tests to the bundled faces. The operating-system
// CJK fallback is on by default and would otherwise make glyph advances, and
// so op counts and widths, depend on which fonts the test machine has
// installed. Tests that exercise the fallback switch it back on themselves.
func TestMain(m *testing.M) {
	pdf.SetSystemFallback(false)

	os.Exit(m.Run())
}
