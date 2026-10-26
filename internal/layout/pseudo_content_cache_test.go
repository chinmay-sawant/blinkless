package layout

import (
	"context"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

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

	ctx := &styleContext{ //nolint:exhaustruct // unit harness fills what the matcher reads
		ctx:    context.Background(),
		sheets: sheets,
	}

	var marked, plain *html.Node

	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n == nil {
			return
		}

		if n.Type == html.ElementNode {
			switch n.Attribute("id") {
			case "marked":
				marked = n
			case "plain":
				plain = n
			}
		}

		for _, c := range n.Children {
			walk(c)
		}
	}

	walk(root)

	if marked == nil || plain == nil {
		t.Fatal("test nodes not found")
	}

	eng := &engine{} //nolint:exhaustruct // memo starts empty, like a fresh Layout run

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
