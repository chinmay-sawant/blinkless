package layout

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// stateFixtureHTML nests three block elements so each one owns a box, and
// adds an inline element and a display:none element, which own no box.
const stateFixtureHTML = `<div id="outer">
<div id="inner">
<p id="leaf">Hello <span id="inline">world</span></p>
</div>
<p id="hidden" style="display:none">gone</p>
</div>`

// TestResultCarriesLayoutState proves the layout state survives the run: the
// resolved styles and the root box stay on the Result, so a later run can
// re-lay out one subtree from them.
func TestResultCarriesLayoutState(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, stateFixtureHTML, sheet(t, "#inner { color: rgb(255, 0, 0); }"))

	if res.styles == nil {
		t.Fatal("Result.styles is nil, want the resolved cascade of the layout run")
	}

	if res.rootBox == nil {
		t.Fatal("Result.rootBox is nil, want the root box of the layout run")
	}

	if res.rootBox != res.root {
		t.Fatal("Result.rootBox and Result.root point at different boxes, want the same box")
	}

	inner := stateNodeByID(t, res, "inner")

	sty, ok := res.styles[inner]
	if !ok {
		t.Fatal("Result.styles has no entry for the nested element node")
	}

	if sty.Color[0] < 0.99 || sty.Color[1] > 0.01 || sty.Color[2] > 0.01 {
		t.Fatalf("resolved color of the nested element = %v, want red", sty.Color)
	}
}

// TestResultStateFindsSubtreeByID proves a subtree can be located in the kept
// state: the box of an element by id, and the html node behind it.
func TestResultStateFindsSubtreeByID(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, stateFixtureHTML)

	outerBox := stateBoxByID(t, res, "outer")
	innerBox := stateBoxByID(t, res, "inner")
	leafBox := stateBoxByID(t, res, "leaf")

	if !boxHasDescendant(outerBox, innerBox) {
		t.Fatal("the nested box is not inside the outer box, want it in the outer subtree")
	}

	if !boxHasDescendant(innerBox, leafBox) {
		t.Fatal("the paragraph box is not inside the nested box, want it in the nested subtree")
	}

	innerNode := stateNodeByID(t, res, "inner")

	if innerNode.Name != divElementName {
		t.Fatalf("nodeForID(\"inner\") returned a %q node, want the nested div", innerNode.Name)
	}

	if innerNode.Attribute("id") != "inner" {
		t.Fatalf("nodeForID(\"inner\") returned the node with id %q, want \"inner\"", innerNode.Attribute("id"))
	}

	if innerNode != innerBox.node {
		t.Fatal("nodeForID and boxForID disagree on the node of the nested element")
	}

	// The style map and the box tree must agree: the box points at the same
	// resolved style the map carries for that node.
	if innerBox.style != res.styles[innerNode] {
		t.Fatal("the nested box and the style map hold different styles for the same node")
	}
}

// TestResultStateLookupMisses proves the lookups report a miss instead of
// guessing: an unknown or empty id has no box and no node.
func TestResultStateLookupMisses(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, stateFixtureHTML)

	for _, elementID := range []string{"missing", ""} {
		if res.boxForID(elementID) != nil {
			t.Fatalf("boxForID(%q) returned a box, want nil", elementID)
		}

		if res.nodeForID(elementID) != nil {
			t.Fatalf("nodeForID(%q) returned a node, want nil", elementID)
		}
	}
}

// TestResultStateLookupWithoutBox proves an element that was never laid out
// still resolves to its node, while the box lookup reports the miss.
func TestResultStateLookupWithoutBox(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, stateFixtureHTML)

	// An inline element is flattened into its parent block, so it owns no box.
	if res.boxForID("inline") != nil {
		t.Fatal("boxForID(\"inline\") returned a box, want nil (inline content has no box)")
	}

	inlineNode := stateNodeByID(t, res, "inline")
	if inlineNode.Name != "span" {
		t.Fatalf("nodeForID(\"inline\") returned a %q node, want the span", inlineNode.Name)
	}

	// A display:none element is never laid out, so it has no box either.
	if res.boxForID("hidden") != nil {
		t.Fatal("boxForID(\"hidden\") returned a box, want nil (display:none has no box)")
	}

	hiddenNode := stateNodeByID(t, res, "hidden")
	if hiddenNode.Type != html.ElementNode {
		t.Fatalf("nodeForID(\"hidden\") returned a node of type %v, want an element", hiddenNode.Type)
	}
}

// stateBoxByID returns the box of the element with the given id and fails the
// test when there is none.
func stateBoxByID(t *testing.T, res *Result, elementID string) *box {
	t.Helper()

	found := res.boxForID(elementID)
	if found == nil {
		t.Fatalf("boxForID(%q) = nil, want a box", elementID)
	}

	if found.node == nil || found.node.Attribute("id") != elementID {
		t.Fatalf("boxForID(%q) did not return the box of that element", elementID)
	}

	return found
}

// stateNodeByID returns the node of the element with the given id and fails
// the test when there is none.
func stateNodeByID(t *testing.T, res *Result, elementID string) *html.Node {
	t.Helper()

	found := res.nodeForID(elementID)
	if found == nil {
		t.Fatalf("nodeForID(%q) = nil, want a node", elementID)
	}

	return found
}

// boxHasDescendant reports whether want is root itself or sits in its subtree.
func boxHasDescendant(root, want *box) bool {
	if root == nil || want == nil {
		return false
	}

	if root == want {
		return true
	}

	for _, child := range root.children {
		if boxHasDescendant(child, want) {
			return true
		}
	}

	return false
}
