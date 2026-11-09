package css

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// matchHotFixture builds a hover-evaluation row: one <div id="list"> holding
// count <span> siblings with ids hs0..hs(count-1). It returns the row, the
// spans in order, and the hovered id (the middle span).
func matchHotFixture(tb testing.TB, count int) (*html.Node, []*html.Node, string) {
	tb.Helper()

	var sb strings.Builder

	sb.WriteString(`<html><body><div id="list">`)

	for i := range count {
		sb.WriteString(`<span id="hs`)
		sb.WriteString(itoa(i))
		sb.WriteString(`" class="item">x</span>`)
	}

	sb.WriteString(`</div></body></html>`)

	root, err := html.Parse(sb.String())
	if err != nil {
		tb.Fatal(err)
	}

	row := root.FirstChild("html").FirstChild("body").FirstChild("div")

	var spans []*html.Node

	for _, child := range row.Children {
		if child.Type == html.ElementNode && child.Name == "span" {
			spans = append(spans, child)
		}
	}

	if len(spans) != count {
		tb.Fatalf("want %d spans, got %d", count, len(spans))
	}

	return row, spans, "hs" + itoa(count/2)
}

// matchHotSelectors parses the three hover shapes once: a direct hover
// compound, a :has() ancestor over a hovered descendant, and a sibling
// after a hovered sibling.
func matchHotSelectors(tb testing.TB) (Selector, Selector, Selector) {
	tb.Helper()

	hover, ok := parseSelector("span:hover")
	if !ok {
		tb.Fatal("parse span:hover")
	}

	hasHover, ok := parseSelector("div:has(:hover)")
	if !ok {
		tb.Fatal("parse div:has(:hover)")
	}

	afterHover, ok := parseSelector("span:hover + span")
	if !ok {
		tb.Fatal("parse span:hover + span")
	}

	return hover, hasHover, afterHover
}

// BenchmarkMatchHotHoverDirect matches span:hover against every span of a
// 200-node row with the middle span hovered: one hit, the rest fail on the
// hover id compare after the tag check passes.
func BenchmarkMatchHotHoverDirect(b *testing.B) {
	_, spans, hoverID := matchHotFixture(b, 200)
	hover, _, _ := matchHotSelectors(b)
	state := MatchState{Hover: hoverID}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		for _, span := range spans {
			state.Matches(hover, span)
		}
	}
}

// BenchmarkMatchHotHasHover matches div:has(:hover) against the row: the
// descendant walk visits spans until the hovered middle span, exercising the
// single-walk :has() descendant path with a hover leaf.
func BenchmarkMatchHotHasHover(b *testing.B) {
	row, _, hoverID := matchHotFixture(b, 200)
	_, hasHover, _ := matchHotSelectors(b)
	state := MatchState{Hover: hoverID}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if !state.Matches(hasHover, row) {
			b.Fatal("expected :has(:hover) match")
		}
	}
}

// BenchmarkMatchHotSiblingAfterHover matches span:hover + span against every
// span: only the span right after the hovered one matches, exercising the
// adjacent-sibling step plus the hover id compare on the previous sibling.
func BenchmarkMatchHotSiblingAfterHover(b *testing.B) {
	_, spans, hoverID := matchHotFixture(b, 200)
	_, _, afterHover := matchHotSelectors(b)
	state := MatchState{Hover: hoverID}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		for _, span := range spans {
			state.Matches(afterHover, span)
		}
	}
}
