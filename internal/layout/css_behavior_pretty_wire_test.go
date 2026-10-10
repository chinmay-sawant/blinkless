package layout

import (
	"strings"
	"testing"
)

// TestPrettyWireRebreaksOrphan pins the pretty end-to-end wiring:
// text-wrap-style: pretty reaches the pack loop through the cascade table
// and re-breaks a greedy one-word orphan without adding lines. Reference:
// Chrome 143.0.7499.40 pulls another word down instead of leaving the
// single-word tail.
func TestPrettyWireRebreaksOrphan(t *testing.T) {
	t.Parallel()

	const prose = "The quick brown fox jumps over the lazy dog"

	normal := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:225pt;font-family:'Liberation Sans',sans-serif;`+
		`font-size:16px">`+prose+`</p></body></html>`)
	pretty := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:225pt;font-family:'Liberation Sans',sans-serif;`+
		`font-size:16px;text-wrap-style:pretty">`+prose+`</p></body></html>`)

	normalLines := inlineLines(normal)
	prettyLines := inlineLines(pretty)

	t.Logf("normal lines=%q", lineTexts(normalLines))
	t.Logf("pretty lines=%q", lineTexts(prettyLines))

	if len(normalLines) == 0 || len(prettyLines) == 0 {
		t.Fatalf("want lines for both layouts, got normal=%d pretty=%d", len(normalLines), len(prettyLines))
	}

	lastNormal := normalLines[len(normalLines)-1].text
	if len(strings.Fields(lastNormal)) != 1 {
		t.Fatalf("normal last line = %q, want a one-word orphan fixture", lastNormal)
	}

	if len(prettyLines) != len(normalLines) {
		t.Fatalf("pretty lines = %d, want normal %d (no added lines)", len(prettyLines), len(normalLines))
	}

	lastPretty := prettyLines[len(prettyLines)-1].text
	if len(strings.Fields(lastPretty)) < prettyMinLastWords {
		t.Errorf("pretty last line = %q, want at least %d words", lastPretty, prettyMinLastWords)
	}

	differs := false

	for idx := range normalLines {
		if normalLines[idx].text != prettyLines[idx].text {
			differs = true
		}
	}

	if !differs {
		t.Errorf("pretty breaks %q identical to greedy, want a re-broken tail", lineTexts(prettyLines))
	}
}

func lineTexts(lines []inlineLine) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, line.text)
	}

	return out
}
