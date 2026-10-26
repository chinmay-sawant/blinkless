package layout

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// findContentProbeNodes returns the #marked and #plain fixture nodes,
// failing the test when either is absent.
func findContentProbeNodes(t *testing.T, root *html.Node) (*html.Node, *html.Node) {
	t.Helper()

	var marked, plain *html.Node

	var walk func(node *html.Node)
	walk = func(node *html.Node) {
		if node == nil {
			return
		}

		if node.Type == html.ElementNode {
			switch node.Attribute("id") {
			case "marked":
				marked = node
			case "plain":
				plain = node
			}
		}

		for _, c := range node.Children {
			walk(c)
		}
	}

	walk(root)

	if marked == nil || plain == nil {
		t.Fatal("test nodes not found")
	}

	return marked, plain
}

// TestCachedContentDeclAgreesAcrossProbes pins the pseudo-content memo: the
// first probe runs the selector match and later probes for the same element
// serve the same declaration without re-matching, while an element with no
// content rule keeps reporting none.
func TestCachedContentDeclAgreesAcrossProbes(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
<div id="marked">x</div>
<div id="plain">y</div>
</body></html>`)

	sheets := []*css.Stylesheet{sheet(t, `
html, body { margin: 0; padding: 0; }
#marked::before { content: "M"; }
`)}

	ctx := &styleContext{
		ctx:    t.Context(),
		sheets: sheets,
	}

	marked, plain := findContentProbeNodes(t, root)

	eng := &engine{}

	first := eng.cachedContentDecl(ctx, marked, pseudoBefore)
	second := eng.cachedContentDecl(ctx, marked, pseudoBefore)

	if first == nil || first.value != `"M"` {
		t.Fatalf("marked content = %+v, want M", first)
	}

	if second == nil || second.value != first.value {
		t.Fatalf("second probe = %+v, want %+v", second, first)
	}

	if _, ok := eng.pseudoContentCache[pseudoContentKey{node: marked, pseudo: pseudoBefore}]; !ok {
		t.Fatal("marked probe not memoized")
	}

	if got := eng.cachedContentDecl(ctx, plain, pseudoBefore); got != nil {
		t.Fatalf("plain content = %+v, want nil", got)
	}

	if _, ok := eng.pseudoContentCache[pseudoContentKey{node: plain, pseudo: pseudoBefore}]; !ok {
		t.Fatal("plain miss not memoized")
	}
}
