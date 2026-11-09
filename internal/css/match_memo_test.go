package css

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

func memoTreeFor(t *testing.T, src string) *html.Node {
	t.Helper()

	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	return root
}

func memoByID(root *html.Node, idVal string) *html.Node {
	var found *html.Node

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil || n == nil {
			return
		}

		if n.Type == html.ElementNode && n.Attribute("id") == idVal {
			found = n

			return
		}

		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)

	return found
}

func memoMustParse(t *testing.T, src string) Selector {
	t.Helper()

	sel, ok := parseSelector(src)
	if !ok {
		t.Fatalf("parseSelector(%q) failed", src)
	}

	return sel
}

// TestMemoGoldenMatches checks match results on generated docs covering
// :has, sibling pseudos, and i-flag attributes. Expected values are the
// long-standing correct outcomes (same as the focused existing tests), so a
// semantic change fails here.
//
//nolint:paralleltest // shares the process-wide match memo and its counters
func TestMemoGoldenMatches(t *testing.T) {
	resetMatchMemoForTest()

	root := memoTreeFor(t, `<html><body>
		<div id="a">A</div><p id="p1">P</p><span id="s">S</span><p id="p2">P2</p>
		<ul id="list"><li id="li0">0</li><li id="li1">1</li><li id="li2">2</li><li id="li3">3</li></ul>
		<section id="sec"><h2 id="h">Title</h2><p id="sec-p">body</p></section>
		<section id="empty"><p id="only">only</p></section>
		<article id="art"><span class="footnote" id="fn">fn</span></article>
		<article id="plain">no fn</article>
		<input id="up" type="FOO">
		<input id="low" type="foo">
		<a id="pdf" href="/files/report.PDF">PDF</a>
		<a id="png" href="/files/report.png">PNG</a>
		<span id="cls" class="Bar baz"></span>
		<span id="en" lang="EN-us"></span>
		<span id="fr" lang="fr"></span>
		<DIV id="upper">U</DIV>
	</body></html>`)

	cases := []struct {
		sel  string
		id   string
		want bool
	}{
		// :has
		{`div:has(+ p)`, "a", true},
		{`span:has(~ p)`, "s", true},
		{`div:has(~ p)`, "a", true},
		{`section:has(> h2)`, "sec", true},
		{`section:has(> h2)`, "empty", false},
		{`article:has(.footnote)`, "art", true},
		{`article:has(.footnote)`, "plain", false},
		// sibling pseudos
		{`li:first-child`, "li0", true},
		{`li:first-child`, "li1", false},
		{`li:last-child`, "li3", true},
		{`li:last-child`, "li0", false},
		{`li:nth-child(2)`, "li1", true},
		{`li:nth-child(2)`, "li0", false},
		{`li:nth-child(even)`, "li1", true},
		{`li:nth-child(even)`, "li0", false},
		{`li:nth-child(odd)`, "li0", true},
		{`li:nth-child(2n+1)`, "li2", true},
		{`li:nth-of-type(2)`, "li1", true},
		{`li:nth-of-type(2)`, "li2", false},
		{`li:nth-last-of-type(1)`, "li3", true},
		// sibling combinators
		{`div + p`, "p1", true},
		{`div + span`, "s", false},
		{`div ~ p`, "p1", true},
		{`div ~ p`, "p2", true},
		{`span ~ p`, "p2", true},
		{`p + p`, "p2", false},
		// i-flag attributes
		{`[type=foo i]`, "up", true},
		{`[type=foo i]`, "low", true},
		{`[type=foo]`, "up", false},
		{`[type=foo]`, "low", true},
		{`[type=FOO i]`, "low", true},
		{`a[href$=".pdf" i]`, "pdf", true},
		{`a[href$=".pdf" i]`, "png", false},
		{`a[href$=".pdf"]`, "pdf", false},
		{`[class~="BAR" i]`, "cls", true},
		{`[class~="BAR"]`, "cls", false},
		{`[class*="AR" i]`, "cls", true},
		{`[class^="bar" i]`, "cls", true},
		{`[lang|="en" i]`, "en", true},
		{`[lang|="en" i]`, "fr", false},
		{`[lang|="en"]`, "en", false},
		// ASCII tag folding
		{`div`, "upper", true},
		{`DIV`, "upper", true},
		{`DiV`, "a", true},
	}

	for _, tc := range cases {
		sel := memoMustParse(t, tc.sel)
		node := memoByID(root, tc.id)

		if node == nil {
			t.Fatalf("missing #%s", tc.id)
		}

		if got := Match(sel, node); got != tc.want {
			t.Errorf("Match(%q, #%s) = %v, want %v", tc.sel, tc.id, got, tc.want)
		}
	}
}

// TestMemoSiblingHits proves the per-parent memo is built once and reused:
// after matching many siblings, hits are nonzero and results stay correct.
//
//nolint:paralleltest // shares the process-wide match memo and its counters
func TestMemoSiblingHits(t *testing.T) {
	resetMatchMemoForTest()

	var sb strings.Builder
	sb.WriteString(`<div id="row"><zzz id="z"></zzz>`)

	for i := range 200 {
		sb.WriteString(`<span id="s`)
		sb.WriteString(itoa(i))
		sb.WriteString(`"></span>`)
	}

	sb.WriteString(`</div>`)
	root := memoTreeFor(t, sb.String())

	z := memoByID(root, "z")
	if z == nil {
		t.Fatal("missing #z")
	}

	// Narrow general-sibling selector: every span after zzz matches.
	genSel := memoMustParse(t, "zzz ~ span")
	// Structural selectors that all read the same parent entry.
	nthSel := memoMustParse(t, "span:nth-child(even)")
	firstSel := memoMustParse(t, "span:first-child")

	var spans []*html.Node

	var collect func(*html.Node)
	collect = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == "span" {
			spans = append(spans, n)
		}

		for _, c := range n.Children {
			collect(c)
		}
	}
	collect(root)

	if len(spans) != 200 {
		t.Fatalf("want 200 spans, got %d", len(spans))
	}

	for _, sp := range spans {
		if !Match(genSel, sp) {
			t.Fatalf("zzz ~ span should match #%s", sp.Attribute("id"))
		}

		// Exercise index lookups on the same parent entry.
		_ = Match(nthSel, sp)
		_ = Match(firstSel, sp)
	}

	if Match(genSel, z) {
		t.Fatal("zzz ~ span should not match the zzz itself")
	}

	hits, misses := siblingMemoStatsForTest()
	if misses == 0 {
		t.Fatal("sibling memo never built (misses == 0)")
	}

	if hits == 0 {
		t.Fatalf("sibling memo never hit (hits == 0, misses == %d)", misses)
	}
}

// TestMemoNoBudgetAllocs proves a match without :has allocates nothing for
// the budget, while a :has match allocates. Nil budget means unlimited.
//
//nolint:paralleltest // shares the process-wide match memo and its counters
func TestMemoNoBudgetAllocs(t *testing.T) {
	resetMatchMemoForTest()

	root := memoTreeFor(t, `<div id="a" class="x"><p>hi</p></div>`)
	node := memoByID(root, "a")

	if node == nil {
		t.Fatal("missing #a")
	}

	plainSel := memoMustParse(t, "div.x")

	allocs := testing.AllocsPerRun(100, func() {
		if !Match(plainSel, node) {
			t.Fatal("div.x should match")
		}
	})
	if allocs != 0 {
		t.Fatalf("Match without :has allocated %v per run, want 0", allocs)
	}

	// selectorContainsHas gates the budget: plain selectors need none.
	if selectorContainsHas(plainSel) {
		t.Fatal("selectorContainsHas(div.x) = true, want false")
	}

	hasSel := memoMustParse(t, "div:has(p:nth-child(1))")
	if !selectorContainsHas(hasSel) {
		t.Fatal("selectorContainsHas(div:has(...)) = false, want true")
	}

	notSel := memoMustParse(t, "div:not(.y)")
	if selectorContainsHas(notSel) {
		t.Fatal("selectorContainsHas(div:not(.y)) = true, want false")
	}

	notHasSel := memoMustParse(t, "div:not(:has(p))")
	if !selectorContainsHas(notHasSel) {
		t.Fatal("selectorContainsHas(div:not(:has(p))) = false, want true")
	}

	// sharedBudgetState keeps a nil budget nil: no per-list allocation.
	empty := MatchState{Focus: "", Hover: "", Active: "", budget: nil}
	if inner := sharedBudgetState(empty); inner.budget != nil {
		t.Fatal("sharedBudgetState allocated a budget with no :has")
	}

	sharedAllocs := testing.AllocsPerRun(100, func() {
		_ = sharedBudgetState(empty)
	})
	if sharedAllocs != 0 {
		t.Fatalf("sharedBudgetState allocated %v per run, want 0", sharedAllocs)
	}

	// First :has descent allocates; nil means unlimited.
	budgeted := empty.ensureHasBudget()
	if budgeted.budget == nil || budgeted.budget.left != maxHasVisits {
		t.Fatal("ensureHasBudget did not allocate the full visit budget")
	}

	// Nil budget stays unlimited: a plain match never fails closed.
	plainState := MatchState{Focus: "", Hover: "", Active: "", budget: nil}
	if !plainState.Matches(plainSel, node) {
		t.Fatal("nil-budget plain match should succeed")
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	var buf [20]byte
	pos := len(buf)

	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}

	return string(buf[pos:])
}
