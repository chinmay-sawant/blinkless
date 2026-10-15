package css

import (
	"errors"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/html"
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

// TestNotArgumentsShareHasBudget proves :not, :is, and :where arguments share
// one visit budget. A fresh budget per :has(*) would still find a child.
func TestNotArgumentsShareHasBudget(t *testing.T) {
	t.Parallel()

	var body strings.Builder

	body.WriteString(`<div id="root">`)

	for range 30 {
		body.WriteString(`<span></span>`)
	}

	body.WriteString(`</div>`)

	root := treeFor(t, body.String())
	subject := byID(root, "root")

	notSel, notOK := parseSelector("div:not(" + repeatedHasArgs(12) + ")")
	isSel, isOK := parseSelector("div:is(" + repeatedHasArgs(12) + ")")
	whereSel, whereOK := parseSelector("div:where(" + repeatedHasArgs(12) + ")")

	if !notOK || !isOK || !whereOK || subject == nil {
		t.Fatal("fixture did not parse")
	}

	assertSharedHasBudget(t, subject, notSel, isSel, whereSel)
}

func assertSharedHasBudget(t *testing.T, subject *html.Node, notSel, isSel, whereSel Selector) {
	t.Helper()

	if Match(notSel, subject) {
		t.Fatal("full budget let :not(:has(*)) match a div with children")
	}

	if !Match(isSel, subject) || !Match(whereSel, subject) {
		t.Fatal("full budget missed :is/:where(:has(*)) on a div with children")
	}

	// Zero visits is smaller than the tree. Shared across every argument,
	// :has(*) fails closed. A private full budget per argument would not.
	limited := MatchState{Focus: "", Hover: "", Active: "", budget: &hasBudget{left: 0}}

	if !limited.Matches(notSel, subject) {
		t.Fatal("each :not(:has(*)) argument got its own visit budget")
	}

	if limited.Matches(isSel, subject) || limited.Matches(whereSel, subject) {
		t.Fatal(":is or :where arguments did not share the tiny budget")
	}
}

func repeatedHasArgs(count int) string {
	var args strings.Builder

	for i := range count {
		if i > 0 {
			args.WriteByte(',')
		}

		args.WriteString(":has(*)")
	}

	return args.String()
}

func TestIsHoverIgnoresOuterState(t *testing.T) {
	t.Parallel()

	root := treeFor(t, `<div id="root"></div>`)
	subject := byID(root, "root")
	hoverSel, hoverOK := parseSelector(":is(:hover)")

	if !hoverOK || subject == nil {
		t.Fatal("fixture did not parse")
	}

	hover := MatchState{Focus: "", Hover: "root", Active: "", budget: nil}
	if hover.Matches(hoverSel, subject) {
		t.Fatal(":is(:hover) used the outer hover state")
	}
}

func TestParseRejectsHugeStylesheet(t *testing.T) {
	t.Parallel()

	src := "a{color:red}" + strings.Repeat(" ", maxStylesheetBytes)
	if _, err := Parse(src); !errors.Is(err, ErrStylesheetTooLarge) {
		t.Fatalf("error = %v, want stylesheet byte cap", err)
	}
}
