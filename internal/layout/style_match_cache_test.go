package layout

import (
	"slices"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// tickSpans collects the .tick spans of the cache fixture in document order.
func tickSpans(t *testing.T, root *html.Node) []*html.Node {
	t.Helper()

	var out []*html.Node

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Attribute("class") == "tick" {
			out = append(out, node)
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(root)

	if len(out) != 3 {
		t.Fatalf("fixture has %d .tick spans, want 3", len(out))
	}

	return out
}

func TestMatchCacheKeepsContextualOutcomes(t *testing.T) {
	t.Parallel()

	s := sheet(t, `.tick { color: red } section .tick { color: blue } .tick + .tick { color: lime }`)
	root := mustParse(t, `<html><body><section><span class="tick">a</span></section>`+
		`<div><span class="tick">b</span><span class="tick">c</span></div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{s}, "print", testViewport, 800)
	spans := tickSpans(t, root)

	// All three spans share one element signature, so the second and third
	// resolve through the match cache. The descendant and sibling rules still
	// depend on position and must not leak across signatures.
	if got := styles[spans[0]].Color; got != [3]float64{0, 0, 1} {
		t.Fatalf("section span color = %v, want blue", got)
	}

	if got := styles[spans[1]].Color; got != [3]float64{1, 0, 0} {
		t.Fatalf("first div span color = %v, want red", got)
	}

	if got := styles[spans[2]].Color; got != [3]float64{0, 1, 0} {
		t.Fatalf("second div span color = %v, want lime", got)
	}
}

func TestMatchedRulesCacheReplayMatchesFreshScan(t *testing.T) {
	t.Parallel()

	s := sheet(t, `.tick { color: red } section .tick { color: blue } `+
		`.tick + .tick { color: lime } span:first-child { color: black } `+
		`div:has(.tick) { color: white } .tick:not(.other) { color: gray }`)
	root := mustParse(t, `<html><body><section><span class="tick">a</span></section>`+
		`<div><span class="tick">b</span><span class="tick">c</span></div></body></html>`)
	spans := tickSpans(t, root)

	ctx := &styleContext{sheets: []*css.Stylesheet{s}}
	got := make([][]ruleHit, len(spans))

	for i, span := range spans {
		got[i] = slices.Clone(ctx.matchedRules(span, ""))
	}

	// Section span: class, descendant, first-child, and :not rules hit; the
	// sibling rule has no preceding tick and the :has rule targets divs.
	if len(got[0]) != 4 {
		t.Fatalf("section span hits = %d, want 4", len(got[0]))
	}

	// First div span: class, first-child, and :not rules hit.
	if len(got[1]) != 3 {
		t.Fatalf("first div span hits = %d, want 3", len(got[1]))
	}

	// Second div span: class, sibling, and :not rules hit.
	if len(got[2]) != 3 {
		t.Fatalf("second div span hits = %d, want 3", len(got[2]))
	}

	if ruleHitsEqual(got[1], got[2]) {
		t.Fatal("div spans share hits despite different sibling and child positions")
	}

	if ctx.matchCache == nil || len(ctx.matchCache) == 0 {
		t.Fatal("match cache is empty after resolving repeated signatures")
	}

	// A fresh context without any cache must produce identical hit sequences.
	for i, span := range spans {
		fresh := &styleContext{sheets: []*css.Stylesheet{s}}
		if dup := fresh.matchedRules(span, ""); !ruleHitsEqual(got[i], dup) {
			t.Fatalf("span %d replay hits differ from a fresh scan", i)
		}
	}
}

func TestMatchCachePseudoElementReplay(t *testing.T) {
	t.Parallel()

	s := sheet(t, `p.note::before { color: red } section p.note::before { color: blue }`)
	root := mustParse(t, `<html><body><section><p class="note">a</p></section>`+
		`<div><p class="note">b</p></div></body></html>`)

	var paras []*html.Node

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Name == "p" {
			paras = append(paras, node)
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(root)

	if len(paras) != 2 {
		t.Fatalf("fixture has %d paragraphs, want 2", len(paras))
	}

	ctx := &styleContext{sheets: []*css.Stylesheet{s}}
	first := cascadePseudoRaw(ctx, paras[0], "before")
	second := cascadePseudoRaw(ctx, paras[1], "before")

	if first["color"] != "blue" {
		t.Fatalf("section paragraph ::before color = %q, want blue", first["color"])
	}

	if second["color"] != "red" {
		t.Fatalf("div paragraph ::before color = %q, want red", second["color"])
	}
}
