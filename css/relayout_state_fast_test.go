package css

import (
	"context"
	"testing"

	pubhtml "github.com/chinmay-sawant/blinkless/html"
)

func stateFastDoc(t *testing.T) *Document {
	t.Helper()

	tree, err := pubhtml.Parse([]byte(`<style>a:hover{color:red}</style><a id="x">hi</a>`))
	if err != nil {
		t.Fatal(err)
	}

	doc, err := Apply(context.Background(), tree, Options{WidthPx: 300, HeightPx: 200, Media: "screen"})
	if err != nil {
		t.Fatal(err)
	}

	return doc
}

// TestRelayoutStateOnlyReusesSheets: hover/focus/active changes at an
// unchanged viewport must reuse the sheet slice itself (no tree walk, no new
// loader), not merely equal sheets.
func TestRelayoutStateOnlyReusesSheets(t *testing.T) {
	t.Parallel()

	doc := stateFastDoc(t)

	next, err := Relayout(context.Background(), doc, 300, 200, "", "x", "")
	if err != nil {
		t.Fatal(err)
	}

	if len(next.sheets) == 0 || &next.sheets[0] != &doc.sheets[0] {
		t.Fatal("state-only relayout rebuilt the sheet slice")
	}

	if next.registry != doc.registry || next.state.Hover != "x" {
		t.Fatal("registry not shared or state not applied")
	}

	// A viewport change still goes through collection.
	wide, err := Relayout(context.Background(), doc, 640, 200, "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if &wide.sheets[0] == &doc.sheets[0] {
		t.Fatal("viewport change must re-run collection")
	}
}

// TestRelayoutAfterMutationRecollects: a tree edit forces a fresh collection.
func TestRelayoutAfterMutationRecollects(t *testing.T) {
	t.Parallel()

	doc := stateFastDoc(t)
	doc.root.MutationVersion++

	next, err := Relayout(context.Background(), doc, 300, 200, "", "x", "")
	if err != nil {
		t.Fatal(err)
	}

	if &next.sheets[0] == &doc.sheets[0] {
		t.Fatal("mutated tree reused stale sheets")
	}
}

// TestRelayoutLeafMutationReusesSheets verifies that non-sheet leaf mutations
// (SetText, AddClass, RemoveAttribute) reuse collected sheets, while editing a
// <style> element forces fresh sheet collection.
func TestRelayoutLeafMutationReusesSheets(t *testing.T) {
	t.Parallel()

	tree, err := pubhtml.Parse([]byte(`<style id="s">a.on{color:red}</style><a id="x" data-k="1">hi</a>`))
	if err != nil {
		t.Fatal(err)
	}

	doc, err := Apply(context.Background(), tree, Options{WidthPx: 300, HeightPx: 200, Media: "screen"})
	if err != nil {
		t.Fatal(err)
	}

	if !tree.SetText("x", "hello") || !tree.AddClass("x", "on") || !tree.RemoveAttribute("x", "data-k") {
		t.Fatal("leaf mutations failed")
	}

	next, err := Relayout(context.Background(), doc, 300, 200, "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if len(next.sheets) == 0 || &next.sheets[0] != &doc.sheets[0] {
		t.Fatal("non-sheet leaf mutations rebuilt the sheet slice")
	}

	if !tree.SetText("s", "a.on{color:blue}") {
		t.Fatal("style element SetText failed")
	}

	recollected, err := Relayout(context.Background(), next, 300, 200, "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if &recollected.sheets[0] == &next.sheets[0] {
		t.Fatal("<style> edit reused stale sheets")
	}
}

func TestUsesStatePseudoMemoized(t *testing.T) {
	t.Parallel()

	doc := stateFastDoc(t)

	if !doc.UsesStatePseudo("hover") || doc.UsesStatePseudo("active") {
		t.Fatal("wrong state pseudo usage")
	}

	next, err := Relayout(context.Background(), doc, 300, 200, "", "x", "")
	if err != nil {
		t.Fatal(err)
	}

	if next.states != doc.states || doc.states == nil {
		t.Fatal("state pseudo memo not shared across a sheet-reusing relayout")
	}
}

func BenchmarkRelayoutHoverOnly(b *testing.B) {
	tree, err := pubhtml.Parse([]byte(`<style>a:hover{color:red}</style><a id="x">hi</a>`))
	if err != nil {
		b.Fatal(err)
	}

	doc, err := Apply(context.Background(), tree, Options{WidthPx: 300, HeightPx: 200, Media: "screen"})
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()

	for i := range b.N {
		hover := ""
		if i%2 == 0 {
			hover = "x"
		}

		if _, err := Relayout(context.Background(), doc, 300, 200, "", hover, ""); err != nil {
			b.Fatal(err)
		}
	}
}

// TestApplyClonesExtra: editing the caller's Extra slice after Apply must not
// change what Relayout re-appends.
func TestApplyClonesExtra(t *testing.T) {
	t.Parallel()

	one, err := Parse(`p{color:red}`)
	if err != nil {
		t.Fatal(err)
	}

	two, err := Parse(`p{color:blue}`)
	if err != nil {
		t.Fatal(err)
	}

	extra := []*Sheet{one}

	tree, err := pubhtml.Parse([]byte(`<p>x</p>`))
	if err != nil {
		t.Fatal(err)
	}

	doc, err := Apply(context.Background(), tree, Options{WidthPx: 300, HeightPx: 200, Extra: extra})
	if err != nil {
		t.Fatal(err)
	}

	extra[0] = two

	next, err := Relayout(context.Background(), doc, 640, 200, "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if len(next.sheets) != 1 || next.sheets[0] != one.sheet {
		t.Fatal("Relayout saw the caller's later edit of Options.Extra")
	}
}
