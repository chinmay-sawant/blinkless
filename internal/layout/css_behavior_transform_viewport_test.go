package layout

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// Transform-box viewport behavior. Each test stamps a synthetic box tree and
// asserts baked display-list matrices, never stored strings. Reference
// browser for every case: Chrome 143.0.7499.40. Shared helpers mustParse,
// near, and parseTransformList/parseTransformOrigin come from layout_test.go
// and transform.go and are never redeclared here.

// viewportTestRotate parses the shared rotate(90deg) fixture transform.
func viewportTestRotate(t *testing.T) Matrix2D {
	t.Helper()

	matrix, _, _, has, ok := parseTransformList("rotate(90deg)", 12)
	if !ok || !has {
		t.Fatal("parseTransformList(rotate 90deg) rejected, want a baked matrix")
	}

	return matrix
}

// viewportTestOrigin parses the shared 0 0 fixture origin.
func viewportTestOrigin(t *testing.T) transformOriginSpec {
	t.Helper()

	spec, ok := parseTransformOrigin("0 0", 12)
	if !ok {
		t.Fatal("parseTransformOrigin(0 0) rejected, want an absolute origin")
	}

	return spec
}

// viewportTestChildStyle builds the shared transformed-child style.
func viewportTestChildStyle(t *testing.T, transformBox string) *ResolvedStyle {
	t.Helper()

	return &ResolvedStyle{
		Transform:       viewportTestRotate(t),
		HasTransform:    true,
		TransformOrigin: viewportTestOrigin(t),
		TransformBox:    transformBox,
		Opacity:         1,
	}
}

// viewportTestTree builds a synthetic viewport fixture: a root box wrapping
// rootTag at (0,0) sized 200x100 holding one transformed 40x20 child at
// (10,20) owning one fill-rect op. layoutHTML never emits this nesting (an
// inline <svg> rasterizes to one image op, so its children get no boxes),
// which is why the tree is built by hand: it proves the stamp walk carries
// the nearest SVG viewport into boxTransformAccum for exactly this shape.
func viewportTestTree(t *testing.T, rootTag, childTag, transformBox string) (*box, []Op) {
	t.Helper()

	root := mustParse(t, `<`+rootTag+`><`+childTag+`/></`+rootTag+`>`)
	find := func(name string) *html.Node {
		node := root.FindFirst(func(n *html.Node) bool { return n.Type == html.ElementNode && n.Name == name })
		if node == nil {
			t.Fatalf("parsed fixture lacks <%s>", name)
		}

		return node
	}

	child := &box{
		node:    find(childTag),
		style:   viewportTestChildStyle(t, transformBox),
		x:       10,
		y:       20,
		w:       40,
		height:  20,
		opStart: 0,
		opEnd:   0,
	}
	parent := &box{
		node:     find(rootTag),
		style:    &ResolvedStyle{Opacity: 1},
		x:        0,
		y:        0,
		w:        200,
		height:   100,
		opStart:  0,
		opEnd:    -1,
		children: []*box{child},
	}
	ops := []Op{{Kind: OpFillRect, X: 10, Y: 20, W: 40, H: 20}}

	return parent, ops
}

// viewportTestStamped stamps one tree and returns the child op matrix.
func viewportTestStamped(t *testing.T, root *box, ops []Op) Matrix2D {
	t.Helper()

	stampBoxTransforms(root, IdentityMatrix(), ops)

	if !ops[0].XformSet {
		t.Fatal("stamped op carries no baked transform, want XformSet")
	}

	return ops[0].Transform()
}

// TestTransformBoxViewBoxUsesNearestSVGViewport is transform-box on SVG
// content: the same rotate(90deg) about 0 0 bakes a different matrix under
// fill-box (object bounding box: the 40x20 border box at 10,20, so
// E,F=(30,10)) than under view-box (nearest SVG viewport: the 200x100 svg
// box at 0,0, so E,F=(0,0)). Reference: Chrome 143.0.7499.40 resolves
// view-box against the nearest SVG viewport per CSS Transforms 2.
func TestTransformBoxViewBoxUsesNearestSVGViewport(t *testing.T) {
	t.Parallel()

	fillRoot, fillOps := viewportTestTree(t, "svg", "rect", "fill-box")
	viewRoot, viewOps := viewportTestTree(t, "svg", "rect", "view-box")

	fill := viewportTestStamped(t, fillRoot, fillOps)
	view := viewportTestStamped(t, viewRoot, viewOps)

	t.Logf("fill-box E,F=(%.3f,%.3f) view-box E,F=(%.3f,%.3f)", fill.E, fill.F, view.E, view.F)

	if !near(fill.E, 30) || !near(fill.F, 10) {
		t.Errorf("fill-box E,F=(%.3f,%.3f), want (30,10) from the border-box origin", fill.E, fill.F)
	}

	if !near(view.E, 0) || !near(view.F, 0) {
		t.Errorf("view-box E,F=(%.3f,%.3f), want (0,0) from the viewport origin", view.E, view.F)
	}

	if near(fill.E, view.E) && near(fill.F, view.F) {
		t.Errorf("fill-box and view-box baked identical E,F=(%.3f,%.3f), want viewport divergence",
			fill.E, fill.F)
	}
}

// TestTransformBoxViewBoxFallsBackWithoutSVGViewport pins the plain-HTML
// fallback: with no <svg> ancestor, view-box resolves against the border box
// exactly like fill-box, so the baked matrices match. This keeps the Gap-C
// pin (plain divs paint identically) green after the viewport threading.
func TestTransformBoxViewBoxFallsBackWithoutSVGViewport(t *testing.T) {
	t.Parallel()

	fillRoot, fillOps := viewportTestTree(t, "div", "span", "fill-box")
	viewRoot, viewOps := viewportTestTree(t, "div", "span", "view-box")

	fill := viewportTestStamped(t, fillRoot, fillOps)
	view := viewportTestStamped(t, viewRoot, viewOps)

	if !near(fill.E, view.E) || !near(fill.F, view.F) {
		t.Errorf("no-viewport fallback fill E,F=(%.3f,%.3f) vs view E,F=(%.3f,%.3f), "+
			"want identical border-box origin", fill.E, fill.F, view.E, view.F)
	}
}

// viewportTestNestedNodes parses the nested-svg fixture and returns the
// outer svg, inner svg, and rect nodes in document order.
func viewportTestNestedNodes(t *testing.T) (*html.Node, *html.Node, *html.Node) {
	t.Helper()

	root := mustParse(t, `<svg><svg><rect/></svg></svg>`)

	rectNode := root.FindFirst(func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Name == "rect"
	})

	if rectNode == nil {
		t.Fatal("parsed fixture lacks <rect>")
	}

	// WalkUntil visits in document order, so the first svg is the outer one;
	// collect both to place the inner viewport at its offset.
	var svgs []*html.Node

	root.WalkUntil(func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Name == "svg" {
			svgs = append(svgs, n)
		}

		return true
	})

	if len(svgs) != 2 {
		t.Fatalf("parsed svg count = %d, want 2 nested", len(svgs))
	}

	return svgs[0], svgs[1], rectNode
}

// viewportTestNestedTree builds a two-viewport fixture: an outer 200x100 svg
// at (0,0) holding an inner 50x40 svg at (100,50) holding one transformed
// 20x10 rect at (110,60) owning one fill-rect op.
func viewportTestNestedTree(t *testing.T) (*box, []Op) {
	t.Helper()

	outerNode, innerNode, rectNode := viewportTestNestedNodes(t)

	child := &box{
		node:    rectNode,
		style:   viewportTestChildStyle(t, "view-box"),
		x:       110,
		y:       60,
		w:       20,
		height:  10,
		opStart: 0,
		opEnd:   0,
	}
	inner := &box{
		node:     innerNode,
		style:    &ResolvedStyle{Opacity: 1},
		x:        100,
		y:        50,
		w:        50,
		height:   40,
		opStart:  0,
		opEnd:    -1,
		children: []*box{child},
	}
	outer := &box{
		node:     outerNode,
		style:    &ResolvedStyle{Opacity: 1},
		x:        0,
		y:        0,
		w:        200,
		height:   100,
		opStart:  0,
		opEnd:    -1,
		children: []*box{inner},
	}
	ops := []Op{{Kind: OpFillRect, X: 110, Y: 60, W: 20, H: 10}}

	return outer, ops
}

// TestTransformBoxViewBoxPrefersNearestViewport nests two viewports: the
// transformed rect sits under an inner 50x40 svg at (100,50) inside the outer
// 200x100 svg. view-box about 0 0 must resolve against the inner viewport
// origin, baking E,F=(150,-50), not the outer origin. Reference: Chrome
// 143.0.7499.40 uses the nearest SVG viewport.
func TestTransformBoxViewBoxPrefersNearestViewport(t *testing.T) {
	t.Parallel()

	outer, ops := viewportTestNestedTree(t)
	got := viewportTestStamped(t, outer, ops)
	t.Logf("nested view-box E,F=(%.3f,%.3f)", got.E, got.F)

	// Origin (100,50) under rotate(90): E = 100+50 = 150, F = 50-100 = -50.
	if !near(got.E, 150) || !near(got.F, -50) {
		t.Errorf("nested view-box E,F=(%.3f,%.3f), want (150,-50) from the inner viewport origin",
			got.E, got.F)
	}
}
