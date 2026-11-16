package layout_test

import "testing"

// TestBoxDeclared proves a stored-only property reaches the box, and that an
// inheriting one reaches a child that declares nothing while a non-inheriting
// one does not.
func TestBoxDeclared(t *testing.T) {
	t.Parallel()

	const page = `<!DOCTYPE html><html><body>
		<div id="a" style="nav-up: #b; text-rendering: optimizeLegibility"><p id="c">x</p></div>
		</body></html>`

	display := renderDisplay(t, page, "")

	if got := boxByID(t, display, "a").Declared["nav-up"]; got != "#b" {
		t.Errorf("nav-up = %q, want #b", got)
	}

	child := boxByID(t, display, "c").Declared

	if child["text-rendering"] == "" {
		t.Error("text-rendering must inherit")
	}

	if _, ok := child["nav-up"]; ok {
		t.Error("nav-up must not inherit")
	}
}
