package layout_test

import "testing"

// TestDeclaredSvgGrammar pins accept, normalize and reject for the SVG leftovers.
func TestDeclaredSvgGrammar(t *testing.T) {
	t.Parallel()

	runDeclaredCases(t, []declaredCase{
		{"text-rendering", "optimizelegibility", "optimizeLegibility"},
		{"text-rendering", "AUTO", "auto"},
		{"text-rendering", "crisp", ""},
		{"lighting-color", "#336699", "#336699"},
		{"lighting-color", "Red", "red"},
		{"lighting-color", "currentColor", "currentcolor"},
		{"lighting-color", "nonsense", ""},
		{"glyph-orientation-vertical", "90", "90deg"},
		{"glyph-orientation-vertical", "0deg", "0deg"},
		{"glyph-orientation-vertical", "180", ""},
		{"baseline-shift", "super", "super"},
		{"baseline-shift", "30%", "30%"},
		{"baseline-shift", "-2px", "-2px"},
		{"baseline-shift", "0", "0"},
		{"baseline-shift", "2", ""},
		{"baseline-shift", "sideways", ""},
		{"marker-side", "match-parent", "match-parent"},
		{"marker-side", "left", ""},
	})
}

// TestDeclaredSvgInherit: text-rendering and glyph-orientation-vertical
// inherit, baseline-shift and marker-side do not.
func TestDeclaredSvgInherit(t *testing.T) {
	t.Parallel()

	const page = `<!DOCTYPE html><html><body>
		<div style="text-rendering:optimizeSpeed;baseline-shift:sub;glyph-orientation-vertical:90;
			marker-side:match-parent"><p id="c">x</p></div>
		</body></html>`

	child := boxByID(t, renderDisplay(t, page, ""), "c").Declared

	if child["text-rendering"] != "optimizeSpeed" || child["glyph-orientation-vertical"] != "90deg" {
		t.Errorf("inheriting names lost: %v", child)
	}

	if child["baseline-shift"] != "" || child["marker-side"] != "" {
		t.Errorf("non-inheriting names leaked: %v", child)
	}
}
