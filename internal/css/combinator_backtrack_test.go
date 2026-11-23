package css

import (
	"strings"
	"testing"
	"time"
)

// TestCombinatorBacktracking covers selectors where the nearest candidate for
// a descendant or general-sibling combinator fails a later constraint, so the
// matcher must try the next candidate instead of giving up.
func TestCombinatorBacktracking(t *testing.T) {
	t.Parallel()

	deep := `<div class="a"><div class="b"><div class="x"><div class="b"><span id="t" class="c"></span></div></div></div></div>`
	sib := `<div><p class="a"></p><p class="b"></p><p class="b"></p><p id="t" class="c"></p></div>`
	sibFail := `<div><p class="b"></p><p class="b"></p><p class="a"></p><p id="t" class="c"></p></div>`

	cases := []struct {
		name, src, sel string
		want           bool
	}{
		{"child then descendant", deep, ".a > .b .c", true},
		{"descendant three", deep, ".a .b .c", true},
		{"no outer match", strings.Replace(deep, `class="a"`, `class="z"`, 1), ".a > .b .c", false},
		{"adjacent then general sibling", sib, ".a + .b ~ .c", true},
		{"adjacent fails", sibFail, ".a + .b ~ .c", false},
		{"sibling backtrack", sib, ".a ~ .b ~ .c", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := treeFor(t, tc.src)

			sel, ok := parseSelector(tc.sel)
			if !ok {
				t.Fatalf("parse %q", tc.sel)
			}

			target := byID(root, "t")
			if got := Match(sel, target); got != tc.want {
				t.Fatalf("Match(%q) = %v, want %v", tc.sel, got, tc.want)
			}
		})
	}
}

// TestPseudoCombinatorBacktracking is the ::before variant of the same walk.
func TestPseudoCombinatorBacktracking(t *testing.T) {
	t.Parallel()

	root := treeFor(t, `<div class="a"><div class="b"><div class="x"><div class="b"><span id="t" class="c"></span></div></div></div></div>`)

	sel, ok := parseSelector(".a > .b .c::before")
	if !ok {
		t.Fatal("parse")
	}

	if !MatchPseudo(sel, byID(root, "t"), "before") {
		t.Fatal(".a > .b .c::before should match through the outer .b")
	}
}

// TestDeepAncestorNonMatchBounded keeps the failing case polynomial.
func TestDeepAncestorNonMatchBounded(t *testing.T) {
	t.Parallel()

	var b strings.Builder
	for range 200 {
		b.WriteString(`<div class="b">`)
	}

	b.WriteString(`<span id="t" class="c"></span>`)

	for range 200 {
		b.WriteString(`</div>`)
	}

	root := treeFor(t, b.String())

	sel, ok := parseSelector(".b .b .b .b .b .b .b .b .a .c")
	if !ok {
		t.Fatal("parse")
	}

	start := time.Now()
	if Match(sel, byID(root, "t")) {
		t.Fatal("unexpected match")
	}

	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("non-match took %v", d)
	}
}
