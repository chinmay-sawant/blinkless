package prepare

import "testing"

// TestHasFontExtensionGuardsQueryEvasion proves cache-busting suffixes
// cannot hide a skipped font type from the pre-fetch filter.
func TestHasFontExtensionGuardsQueryEvasion(t *testing.T) {
	t.Parallel()

	skipped := map[string]string{
		"https://example.test/face.woff2":     ".woff2",
		"https://example.test/face.woff2?v=3": ".woff2",
		"https://example.test/face.woff2#x":   ".woff2",
		"https://example.test/face.EOT":       ".eot",
		"https://example.test/face.eot?v=3":   ".eot",
	}

	for uri, ext := range skipped {
		if !hasFontExtension(uri, ext) {
			t.Fatalf("hasFontExtension(%q, %q) = false, want true", uri, ext)
		}
	}

	if hasFontExtension("https://example.test/face.woff", ".woff2") {
		t.Fatal("hasFontExtension matched .woff against .woff2")
	}

	if hasFontExtension("https://example.test/woff2/face.ttf", ".woff2") {
		t.Fatal("hasFontExtension matched a path segment against the extension")
	}
}
