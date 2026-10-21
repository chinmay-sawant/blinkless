package layout

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

func indexNode(t *testing.T, source, name string) *html.Node {
	t.Helper()

	root, err := html.Parse(source)
	if err != nil {
		t.Fatal(err)
	}

	node := findElementByName(root, name)
	if node == nil {
		t.Fatalf("no %s element", name)
	}

	return node
}

func TestRuleIndexKeepsSheetOrderOnEqualRuleOrder(t *testing.T) {
	t.Parallel()

	first := sheet(t, `div { color: red }`)
	second := sheet(t, `#x { color: blue }`)
	node := indexNode(t, `<html><body><div id="x"></div></body></html>`, "div")

	index := newRuleIndex([]*css.Stylesheet{first, second})
	cands := index.candidates(nil, node, "")
	sortCandidates(cands)

	if len(cands) != 2 || cands[0].sheet != 0 || cands[1].sheet != 1 {
		t.Fatalf("candidates = %+v, want sheet 0 before sheet 1", cands)
	}
}

func TestRuleIndexFindsSelectorsByAnyClass(t *testing.T) {
	t.Parallel()

	index := newRuleIndex([]*css.Stylesheet{sheet(t, `.second { color: red }`)})
	node := indexNode(t, `<html><body><div class="first second"></div></body></html>`, "div")

	if cands := index.candidates(nil, node, ""); len(cands) != 1 {
		t.Fatalf("class candidates = %d, want 1", len(cands))
	}
}

func TestRuleIndexSeparatesPseudoElements(t *testing.T) {
	t.Parallel()

	index := newRuleIndex([]*css.Stylesheet{sheet(t, `p::before { content: "x" }`)})
	node := indexNode(t, `<html><body><p></p></body></html>`, "p")

	if cands := index.candidates(nil, node, "before"); len(cands) != 1 {
		t.Fatalf("before candidates = %d, want 1", len(cands))
	}

	if cands := index.candidates(nil, node, ""); len(cands) != 0 {
		t.Fatalf("host candidates = %d, want 0", len(cands))
	}

	if !index.content {
		t.Fatal("content flag is false for a pseudo content rule")
	}
}

func TestRuleIndexMarksGroupedSelectorOnce(t *testing.T) {
	t.Parallel()

	index := newRuleIndex([]*css.Stylesheet{sheet(t, `h1, .missing { color: red }`)})
	node := indexNode(t, `<html><body><h1 class="title"></h1></body></html>`, "h1")

	if cands := index.candidates(nil, node, ""); len(cands) != 1 {
		t.Fatalf("grouped candidates = %d, want 1", len(cands))
	}
}

func TestRuleIndexContentFlagWithoutContentDecl(t *testing.T) {
	t.Parallel()

	index := newRuleIndex([]*css.Stylesheet{sheet(t, `p::after { color: red }`)})
	if index.content {
		t.Fatal("content flag is true without a content declaration")
	}
}
