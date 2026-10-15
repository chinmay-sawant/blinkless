package css

import (
	"errors"
	"strings"
	"testing"
)

func TestHasVisitBudgetStopsBeforeLateMatch(t *testing.T) {
	t.Parallel()

	var body strings.Builder

	body.WriteString(`<div id="root">`)

	for range 80 {
		body.WriteString("<span></span>")
	}

	body.WriteString(`<span class="hit"></span></div>`)

	root := treeFor(t, body.String())
	subject := byID(root, "root")
	sel, ok := parseSelector("div:has(.hit)")

	if !ok || subject == nil {
		t.Fatal("fixture did not parse")
	}

	limited := MatchState{Focus: "", Hover: "", Active: "", budget: &hasBudget{left: 8}}
	if limited.Matches(sel, subject) {
		t.Fatal("a tiny :has budget matched a target past the cap")
	}

	if !Match(sel, subject) {
		t.Fatal("the normal budget missed a real :has target")
	}
}

func TestParseRejectsHugeStylesheet(t *testing.T) {
	t.Parallel()

	src := "a{color:red}" + strings.Repeat(" ", maxStylesheetBytes)
	if _, err := Parse(src); !errors.Is(err, ErrStylesheetTooLarge) {
		t.Fatalf("error = %v, want stylesheet byte cap", err)
	}
}
