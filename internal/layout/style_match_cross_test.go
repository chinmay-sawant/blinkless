package layout

import (
	"slices"
	"sync"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// crossFixture is the shared document for the cross-layout tests: three
// .tick spans where the section span also matches a descendant rule and the
// second div span matches an adjacent-sibling rule.
const crossFixture = `<html><body><section><span class="tick">a</span></section>` +
	`<div><span class="tick">b</span><span class="tick">c</span></div></body></html>`

const crossCSS = `.tick { color: red } section .tick { color: blue } ` +
	`.tick + .tick { color: lime } span:first-child { color: black } ` +
	`.tick:not(.other) { color: gray }`

func crossSpans(t *testing.T, root *html.Node) []*html.Node {
	t.Helper()

	return tickSpans(t, root)
}

func crossColors(styles map[*html.Node]*ResolvedStyle, spans []*html.Node) [][3]float64 {
	out := make([][3]float64, len(spans))

	for i, span := range spans {
		out[i] = styles[span].Color
	}

	return out
}

// TestCrossLayoutSharesCompiledIndex proves two consecutive layouts of the
// same sheets reuse one compiled rule index: both contexts end up holding the
// identical index pointer, which only the shared bundle hands out (every
// fresh build allocates a new object).
func TestCrossLayoutSharesCompiledIndex(t *testing.T) {
	t.Parallel()

	styleSheet := sheet(t, crossCSS)
	sheets := []*css.Stylesheet{styleSheet}
	root := mustParse(t, crossFixture)
	spans := crossSpans(t, root)

	first := &styleContext{sheets: sheets}
	firstHits := slices.Clone(first.matchedRules(spans[0], ""))

	if first.index == nil {
		t.Fatal("first layout left no compiled index")
	}

	second := &styleContext{sheets: sheets}
	secondHits := slices.Clone(second.matchedRules(spans[0], ""))

	if second.index != first.index {
		t.Fatal("consecutive layouts of the same sheets hold different rule indexes")
	}

	if !ruleHitsEqual(firstHits, secondHits) {
		t.Fatal("shared index changed the matched rules")
	}

	if len(first.matchCache) == 0 || len(second.matchCache) == 0 {
		t.Fatal("per-layout match caches are empty after resolving")
	}

	// Per-document outcomes stay per layout: dropping a signature from one
	// context must not touch the other.
	for key := range first.matchCache {
		delete(first.matchCache, key)

		if _, ok := second.matchCache[key]; !ok {
			t.Fatal("per-layout match caches alias each other")
		}

		break
	}

	// The shared bundle interned the tag both layouts probed.
	bundle := sharedCompiledFor(sheets)
	if bundle == nil {
		t.Fatal("no shared bundle for the fixture sheets")
	}

	bundle.mu.RLock()
	interned, ok := bundle.tags["span"]
	bundle.mu.RUnlock()

	if !ok {
		t.Fatal("shared bundle did not intern the probed tag")
	}

	if first.matchTags["span"] != interned || second.matchTags["span"] != interned {
		t.Fatal("layouts do not reuse the shared interned tag")
	}

	// The single-class memo retains the encoding across layouts.
	singleAttrCacheMu.RLock()
	_, kept := singleAttrCache["tick"]
	singleAttrCacheMu.RUnlock()

	if !kept {
		t.Fatal("single-class memo did not retain the class encoding")
	}
}

// TestCrossLayoutInvalidatesOnSheetChange proves a changed stylesheet does
// not reuse the previous bundle: the new layout compiles its own index and
// resolves the new declarations.
func TestCrossLayoutInvalidatesOnSheetChange(t *testing.T) {
	t.Parallel()

	beforeSheet := sheet(t, `.tick { color: red }`)
	afterSheet := sheet(t, `.tick { color: blue }`)

	beforeCtx := &styleContext{sheets: []*css.Stylesheet{beforeSheet}}
	beforeCtx.matchedRules(mustParse(t, crossFixture), "")

	afterCtx := &styleContext{sheets: []*css.Stylesheet{afterSheet}}
	afterCtx.matchedRules(mustParse(t, crossFixture), "")

	if beforeCtx.index == nil || afterCtx.index == nil {
		t.Fatal("layouts left no compiled index")
	}

	if beforeCtx.index == afterCtx.index {
		t.Fatal("changed stylesheet reused the previous compiled index")
	}

	beforeRoot := mustParse(t, crossFixture)
	afterRoot := mustParse(t, crossFixture)
	beforeStyles := resolveStyles(beforeRoot, []*css.Stylesheet{beforeSheet}, "print", testViewport, 800)
	afterStyles := resolveStyles(afterRoot, []*css.Stylesheet{afterSheet}, "print", testViewport, 800)

	beforeSpans := crossSpans(t, beforeRoot)
	afterSpans := crossSpans(t, afterRoot)

	if got := beforeStyles[beforeSpans[1]].Color; got != [3]float64{1, 0, 0} {
		t.Fatalf("old sheet span color = %v, want red", got)
	}

	if got := afterStyles[afterSpans[1]].Color; got != [3]float64{0, 0, 1} {
		t.Fatalf("new sheet span color = %v, want blue", got)
	}
}

// TestCrossLayoutConcurrentLayouts drives shared bundles from many
// goroutines at once. The race detector owns the verdict: any unsynchronized
// sharing fails the -race run. A serial check afterwards confirms the shared
// paths still resolve correctly.
func TestCrossLayoutConcurrentLayouts(t *testing.T) {
	t.Parallel()

	styleSheet := sheet(t, `.tick { color: red } section .tick { color: blue } `+
		`.tick + .tick { color: lime }`)
	sheets := []*css.Stylesheet{styleSheet}

	var wg sync.WaitGroup

	for range 8 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			root, err := html.Parse(crossFixture)
			if err != nil {
				return
			}

			spans := tickSpansFor(root)
			if len(spans) != 3 {
				return
			}

			for round := range 25 {
				ctx := &styleContext{sheets: sheets}

				for _, span := range spans {
					ctx.matchedRules(span, "")
				}

				if round%5 == 0 {
					resolveStyles(root, sheets, "print", testViewport, 800)
				}
			}
		}()
	}

	wg.Wait()

	root := mustParse(t, crossFixture)
	styles := resolveStyles(root, sheets, "print", testViewport, 800)
	spans := crossSpans(t, root)

	if got := styles[spans[0]].Color; got != [3]float64{0, 0, 1} {
		t.Fatalf("section span color after concurrent layouts = %v, want blue", got)
	}

	if got := styles[spans[2]].Color; got != [3]float64{0, 1, 0} {
		t.Fatalf("second div span color after concurrent layouts = %v, want lime", got)
	}
}

// tickSpansFor collects .tick spans without test assertions, for goroutines
// that cannot fail the test from a non-test goroutine.
func tickSpansFor(root *html.Node) []*html.Node {
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

	return out
}

// TestCrossLayoutSameResultsAcrossBundles proves sharing is behavior-neutral:
// the same CSS parsed twice compiles two independent bundles, and layouts
// through the reused bundle resolve exactly what the independent bundle
// resolves, span by span.
func TestCrossLayoutSameResultsAcrossBundles(t *testing.T) {
	t.Parallel()

	shared := sheet(t, crossCSS)
	separate := sheet(t, crossCSS)

	sharedSheets := []*css.Stylesheet{shared}
	separateSheets := []*css.Stylesheet{separate}

	// Warm the shared bundle, then resolve through the reuse path twice.
	warmRoot := mustParse(t, crossFixture)
	resolveStyles(warmRoot, sharedSheets, "print", testViewport, 800)

	firstRoot := mustParse(t, crossFixture)
	firstStyles := resolveStyles(firstRoot, sharedSheets, "print", testViewport, 800)

	secondRoot := mustParse(t, crossFixture)
	secondStyles := resolveStyles(secondRoot, separateSheets, "print", testViewport, 800)

	firstColors := crossColors(firstStyles, crossSpans(t, firstRoot))
	secondColors := crossColors(secondStyles, crossSpans(t, secondRoot))

	if len(firstColors) != len(secondColors) {
		t.Fatalf("color counts differ: %d vs %d", len(firstColors), len(secondColors))
	}

	for i := range firstColors {
		if firstColors[i] != secondColors[i] {
			t.Fatalf("span %d color differs with and without sharing: %v vs %v",
				i, firstColors[i], secondColors[i])
		}
	}
}
