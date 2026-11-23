package layout

import (
	"math"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// TestCustomPropertyCaseSensitiveInCascade checks the cascade keeps
// --sizeBig and --sizebig apart while ordinary property names stay
// case-insensitive.
func TestCustomPropertyCaseSensitiveInCascade(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.a { --sizeBig: 20px; --sizebig: 10px; FONT-SIZE: var(--sizeBig); }
.b { --sizeBig: 20px; --sizebig: 10px; font-size: var(--sizebig); }
`)

	root, err := html.Parse(`<html><body><p class="a" id="a">x</p><p class="b" id="b">y</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "screen", 800, 600)

	var sizes []float64

	for _, id := range []string{"a", "b"} {
		var found *html.Node

		var walk func(n *html.Node)

		walk = func(n *html.Node) {
			if n.AttrExact("id") == id {
				found = n
			}

			for _, c := range n.Children {
				walk(c)
			}
		}
		walk(root)

		if found == nil {
			t.Fatalf("missing %s", id)
		}

		sizes = append(sizes, styles[found].FontSize)
	}

	if math.Abs(sizes[0]-15) > 0.1 || math.Abs(sizes[1]-7.5) > 0.1 {
		t.Fatalf("font sizes = %v, want 15pt (20px) and 7.5pt (10px)", sizes)
	}
}
