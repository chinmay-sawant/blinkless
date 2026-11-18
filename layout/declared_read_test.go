package layout_test

import "testing"

// TestDeclaredReadGrammar pins accept, normalize and reject for the read group.
func TestDeclaredReadGrammar(t *testing.T) {
	t.Parallel()

	runDeclaredCases(t, []declaredCase{
		{"reading-flow", "grid-rows", "grid-rows"},
		{"reading-flow", "Source-Order", "source-order"},
		{"reading-flow", "reverse", ""},
		{"reading-order", "3", "3"},
		{"reading-order", "-2", "-2"},
		{"reading-order", "1.5", ""},
		{"reading-order", "first", ""},
		{"overlay", "auto", "auto"},
		{"overlay", "top", ""},
		{"background-attachment", "local", "local"},
		{"background-attachment", "fixed, scroll", "fixed, scroll"},
		{"background-attachment", "sticky", ""},
		{"background-tbd", "none", "none"},
	})
}
