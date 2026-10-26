package layout

import (
	"strings"
	"testing"
)

// TestCloneContentEnvIsolatesQuoteDepth pins the clone contract the counter
// environment memo relies on: a probe may mutate quote depth while it
// evaluates, so sharing the cached canonical object would corrupt the next
// probe. The clone carries the computed state and diverges safely.
func TestCloneContentEnvIsolatesQuoteDepth(t *testing.T) {
	t.Parallel()

	base := defaultContentEnv()
	base.counters.byName["item"] = []int{0, 2}
	base.quoteDepth = 1

	clone := base.cloneContentEnv()

	if clone == base {
		t.Fatal("clone shares the canonical object")
	}

	clone.quoteDepth = 9
	clone.counters.byName["item"][1] = 7
	clone.counters.byName["new"] = []int{1}

	if base.quoteDepth != 1 {
		t.Fatalf("canonical depth = %d, want 1", base.quoteDepth)
	}

	if got := base.counters.byName["item"]; len(got) != 2 || got[1] != 2 {
		t.Fatalf("canonical counters = %v, want [0 2]", got)
	}

	if _, ok := base.counters.byName["new"]; ok {
		t.Fatal("clone leaked a counter into the canonical map")
	}
}

// TestQuoteContentSurvivesRepeatedProbes exercises the memo plus clone on the
// production path: one layout probes the same quote content several times
// across measure and emit, and the rendered glyphs must match a single walk.
func TestQuoteContentSurvivesRepeatedProbes(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><p id="q">say <span id="w">hi</span> now</p></body></html>`, sheet(t, `
html, body { margin: 0; padding: 0; }
#q::before { content: open-quote; }
#q::after { content: close-quote; }
#w::before { content: open-quote; }
#w::after { content: close-quote; }
`))

	var builder strings.Builder

	for _, op := range res.Ops {
		if op.Kind == OpText {
			builder.WriteString(op.Text)
			builder.WriteString("|")
		}
	}

	got := builder.String()

	if strings.Count(got, "\u201c") != 2 || strings.Count(got, "\u201d") != 2 {
		t.Fatalf("quote glyphs unbalanced: %q", got)
	}

	if !strings.Contains(got, "say ") || !strings.Contains(got, "hi") || !strings.Contains(got, " now") {
		t.Fatalf("words missing: %q", got)
	}
}
