package layout

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// continue (CSS Overflow 4) against testdata/css/v012/continue/page.html.
// Each paragraph is 120px wide with 20px lines, so max-height or height of
// 40px holds two lines. Chrome 143.0.7499.40 does not ship continue; the
// expectation follows the draft: collapse and discard drop the lines past
// the box, and normal keeps them. Each case lays out one paragraph alone and
// counts the distinct baselines of its text operations.

func continueFixture(t *testing.T) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "css", "v012", "continue", "page.html"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	return string(raw)
}

var continueParaRE = regexp.MustCompile(`(?s)<p id="[^"]+">.*?</p>\s*`)

// continueLines lays out the fixture with only the paragraph id kept and
// returns how many distinct text lines it paints.
func continueLines(t *testing.T, src, id, extraCSS string) int {
	t.Helper()

	kept := ""

	for _, p := range continueParaRE.FindAllString(src, -1) {
		if strings.Contains(p, `id="`+id+`"`) {
			kept = p
		}
	}

	if kept == "" {
		t.Fatalf("paragraph %q missing from fixture", id)
	}

	doc := continueParaRE.ReplaceAllString(src, "")
	doc = strings.Replace(doc, "</body>", kept+"</body>", 1)
	doc = strings.Replace(doc, "</style>", extraCSS+"</style>", 1)
	res := layoutHTML(t, doc, sheet(t, svgGeomStyleText(t, doc)))
	ys := map[string]bool{}

	for _, op := range opsOfKind(res, OpText) {
		if strings.TrimSpace(op.Text) != "" {
			ys[fmt.Sprintf("%.2f", op.Y)] = true
		}
	}

	return len(ys)
}

func TestBehaviorContinueCollapseDropsLinesPastMaxHeight(t *testing.T) {
	t.Parallel()

	src := continueFixture(t)
	plain := continueLines(t, src, "plain", "")
	collapse := continueLines(t, src, "collapse", "")
	normal := continueLines(t, src, "normal", "")

	if plain < 3 {
		t.Fatalf("plain paragraph has %d lines, want at least 3 so the clamp is visible", plain)
	}

	if collapse != 2 {
		t.Errorf("continue:collapse kept %d lines, want 2", collapse)
	}

	if normal != plain {
		t.Errorf("continue:normal kept %d lines, want the %d of the plain paragraph", normal, plain)
	}
}

func TestBehaviorContinueDiscardDropsLinesPastHeight(t *testing.T) {
	t.Parallel()

	src := continueFixture(t)
	discard := continueLines(t, src, "discard", "")

	if discard != 2 {
		t.Errorf("continue:discard with height:40px kept %d lines, want 2", discard)
	}

	// discard with no definite height has nothing to cut at.
	open := continueLines(t, src, "discard", "#discard{height:auto}")
	if open < 3 {
		t.Errorf("continue:discard with height:auto kept %d lines, want all of them", open)
	}

	// collapse ignores a definite height: only max-height cuts it.
	heightOnly := continueLines(t, src, "collapse", "#collapse{max-height:none;height:40px}")
	if heightOnly < 3 {
		t.Errorf("continue:collapse with only height kept %d lines, want all of them", heightOnly)
	}
}

func TestBehaviorContinueWebkitLegacyClampsVerticalBox(t *testing.T) {
	t.Parallel()

	src := continueFixture(t)
	vertical := continueLines(t, src, "collapse",
		"#collapse{display:-webkit-box;-webkit-box-orient:vertical;continue:-webkit-legacy}")
	horizontal := continueLines(t, src, "collapse",
		"#collapse{display:-webkit-box;-webkit-box-orient:horizontal;continue:-webkit-legacy}")

	if vertical != 2 {
		t.Errorf("-webkit-legacy in a vertical -webkit-box kept %d lines, want 2", vertical)
	}

	if horizontal == 2 {
		t.Errorf("-webkit-legacy in a horizontal -webkit-box clamped to %d lines, want no clamp", horizontal)
	}
}
