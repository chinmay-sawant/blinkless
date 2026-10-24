package layout

import (
	"math"
	"testing"
)

// Transform metadata population tests: the placed walk must report the same
// matrices and origin the stamp walk baked onto the operations.
// Reference: Chrome 143.0.7499.40.

// placedByElementID returns the placed element with the given id.
func placedByElementID(t *testing.T, placed []PlacedElement, elementID string) PlacedElement {
	t.Helper()

	for _, item := range placed {
		if item.ID == elementID {
			return item
		}
	}

	t.Fatalf("placed element #%s not found", elementID)

	return PlacedElement{}
}

// placedOpByID returns the display-list operation with the given id.
func placedOpByID(t *testing.T, res *Result, opID uint64) Op {
	t.Helper()

	for _, item := range res.Ops {
		if item.ID == opID {
			return item
		}
	}

	t.Fatalf("op id %d not found", opID)

	return Op{}
}

// matricesNear reports whether two matrices match within floating-point noise.
func matricesNear(a, b Matrix2D) bool {
	const eps = 1e-9

	return math.Abs(a.A-b.A) < eps && math.Abs(a.B-b.B) < eps &&
		math.Abs(a.C-b.C) < eps && math.Abs(a.D-b.D) < eps &&
		math.Abs(a.E-b.E) < eps && math.Abs(a.F-b.F) < eps
}

// TestPlacedTransformWorldMatchesStampedOps checks that World equals the matrix
// the stamp walk wrote to the element's first operation, that Parent * Own
// reconstructs World, and that an untransformed descendant still reports its
// parent accumulation and identity Own while its ops carry the ancestor matrix.
//
//nolint:cyclop,funlen // one nested-transform assertion block
func TestPlacedTransformWorldMatchesStampedOps(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body>
<div id="outer"><div id="inner"><div id="leaf"></div></div></div>
</body></html>`, sheet(t, `
html, body { margin: 0; padding: 0; }
div { margin: 0; padding: 0; }
#outer { width: 200px; height: 100px; transform: rotate(10deg); background: #c33; }
#inner { width: 80px; height: 40px; transform: rotate(20deg) translate(5px, 7px); background: #33c; }
#leaf { width: 20px; height: 20px; background: #3c3; }
`))

	placed := PlacedElements(res)
	outer := placedByElementID(t, placed, "outer")
	inner := placedByElementID(t, placed, "inner")
	leaf := placedByElementID(t, placed, "leaf")

	for _, item := range []PlacedElement{outer, inner} {
		if !item.Transform.HasMatrix {
			t.Fatalf("#%s HasMatrix = false, want true", item.ID)
		}

		if len(item.OperationIDs) == 0 {
			t.Fatalf("#%s owns no ops", item.ID)
		}

		stamped := placedOpByID(t, res, item.OperationIDs[0])

		if !stamped.XformSet {
			t.Fatalf("#%s first op XformSet = false, want true", item.ID)
		}

		if !matricesNear(item.Transform.World, stamped.Transform()) {
			t.Errorf("#%s World = %+v, stamped op = %+v",
				item.ID, item.Transform.World, stamped.Transform())
		}
	}

	if got := inner.Transform.Parent.Mul(inner.Transform.Own); !matricesNear(got, inner.Transform.World) {
		t.Errorf("inner Parent*Own = %+v, want World %+v", got, inner.Transform.World)
	}

	if !matricesNear(inner.Transform.Parent, outer.Transform.World) {
		t.Errorf("inner Parent = %+v, want outer World %+v",
			inner.Transform.Parent, outer.Transform.World)
	}

	if leaf.Transform.HasMatrix {
		t.Error("leaf HasMatrix = true, want false")
	}

	if !leaf.Transform.Own.IsIdentity() {
		t.Errorf("leaf Own = %+v, want identity", leaf.Transform.Own)
	}

	if !matricesNear(leaf.Transform.Parent, inner.Transform.World) {
		t.Errorf("leaf Parent = %+v, want inner World %+v",
			leaf.Transform.Parent, inner.Transform.World)
	}

	if !matricesNear(leaf.Transform.World, inner.Transform.World) {
		t.Errorf("leaf World = %+v, want inner World %+v",
			leaf.Transform.World, inner.Transform.World)
	}

	// Metadata must not change op emission: the untransformed leaf's ops still
	// carry the ancestor matrix.
	if len(leaf.OperationIDs) == 0 {
		t.Fatal("leaf owns no ops")
	}

	stampedLeaf := placedOpByID(t, res, leaf.OperationIDs[0])

	if !stampedLeaf.XformSet {
		t.Error("leaf op XformSet = false, want the ancestor matrix")
	}

	if !matricesNear(stampedLeaf.Transform(), inner.Transform.World) {
		t.Errorf("leaf op matrix = %+v, want inner World %+v",
			stampedLeaf.Transform(), inner.Transform.World)
	}
}

// TestPlacedTransformOriginZeroIsLayoutPoints checks a transform-origin of 0 0
// on a box at the page origin: the reported origin is 0,0 in layout points.
func TestPlacedTransformOriginZeroIsLayoutPoints(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div id="zero"></div></body></html>`, sheet(t, `
html, body { margin: 0; padding: 0; }
div { margin: 0; padding: 0; }
#zero { width: 80px; height: 40px; transform: rotate(30deg); transform-origin: 0 0; background: #e33; }
`))

	element := placedByElementID(t, PlacedElements(res), "zero")

	if element.X != 0 || element.Y != 0 {
		t.Fatalf("#zero box at (%.3f,%.3f), want the page origin", element.X, element.Y)
	}

	if !element.Transform.HasMatrix {
		t.Fatal("#zero HasMatrix = false, want true")
	}

	if !near(element.Transform.OriginX, 0) || !near(element.Transform.OriginY, 0) {
		t.Errorf("#zero origin = (%.3f,%.3f), want (0,0)",
			element.Transform.OriginX, element.Transform.OriginY)
	}
}

// TestPlacedTransformOriginPercentStaysInPoints guards the unit rule: an 80px
// (60 layout point) wide box at the page origin resolves 50% 50% to (30,15)
// layout points. Reporting CSS pixels would give (40,20) for the same box.
func TestPlacedTransformOriginPercentStaysInPoints(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div id="pct"></div></body></html>`, sheet(t, `
html, body { margin: 0; padding: 0; }
div { margin: 0; padding: 0; }
#pct { width: 80px; height: 40px; transform: rotate(30deg); transform-origin: 50% 50%; background: #3e3; }
`))

	element := placedByElementID(t, PlacedElements(res), "pct")

	if element.X != 0 || element.Y != 0 {
		t.Fatalf("#pct box at (%.3f,%.3f), want the page origin", element.X, element.Y)
	}

	if !near(element.W, 60) || !near(element.H, 30) {
		t.Fatalf("#pct box = %.3fx%.3f points, want 60x30 from 80x40px", element.W, element.H)
	}

	wantX, wantY := element.W/2, element.H/2

	if !near(element.Transform.OriginX, wantX) || !near(element.Transform.OriginY, wantY) {
		t.Errorf("#pct origin = (%.3f,%.3f), want (%.3f,%.3f) layout points",
			element.Transform.OriginX, element.Transform.OriginY, wantX, wantY)
	}
}

// TestPlacedTransform3DMetadataFields checks the non-matrix metadata copied
// from the resolved style: the 3D flag, perspective distance and origin,
// transform style, and backface visibility.
//
//nolint:cyclop // one table of 3D metadata fields
func TestPlacedTransform3DMetadataFields(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div id="three"></div></body></html>`, sheet(t, `
html, body { margin: 0; padding: 0; }
div { margin: 0; padding: 0; }
#three {
	width: 80px; height: 40px; transform: rotateY(30deg); background: #e3e;
	perspective: 500px; perspective-origin: 25% 75%;
	transform-style: preserve-3d; backface-visibility: hidden;
}
`))

	element := placedByElementID(t, PlacedElements(res), "three")

	if element.X != 0 || element.Y != 0 {
		t.Fatalf("#three box at (%.3f,%.3f), want the page origin", element.X, element.Y)
	}

	if !element.Transform.HasMatrix {
		t.Error("#three HasMatrix = false, want true")
	}

	if !element.Transform.HasTransform3D {
		t.Error("#three HasTransform3D = false, want true")
	}

	if !element.Transform.HasPerspective {
		t.Error("#three HasPerspective = false, want true")
	}

	if !near(element.Transform.Perspective, pxToPt(500)) {
		t.Errorf("#three Perspective = %.3f, want %.3f layout points",
			element.Transform.Perspective, pxToPt(500))
	}

	wantPerspectiveX := element.X + element.W*0.25
	wantPerspectiveY := element.Y + element.H*0.75

	if !near(element.Transform.PerspectiveOriginX, wantPerspectiveX) ||
		!near(element.Transform.PerspectiveOriginY, wantPerspectiveY) {
		t.Errorf("#three perspective origin = (%.3f,%.3f), want (%.3f,%.3f) layout points",
			element.Transform.PerspectiveOriginX, element.Transform.PerspectiveOriginY,
			wantPerspectiveX, wantPerspectiveY)
	}

	if element.Transform.Style != "preserve-3d" {
		t.Errorf("#three Style = %q, want preserve-3d", element.Transform.Style)
	}

	if element.Transform.Backface != "hidden" {
		t.Errorf("#three Backface = %q, want hidden", element.Transform.Backface)
	}
}

// TestPlacedTransformAbsentKeepsParentAndOrigin pins the no-static-transform
// case: Parent and origin are still available so a keyframed transform can be
// composed, while HasMatrix stays false and Own is the identity.
func TestPlacedTransformAbsentKeepsParentAndOrigin(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div id="plain"></div></body></html>`, sheet(t, `
html, body { margin: 0; padding: 0; }
div { margin: 0; padding: 0; }
#plain { width: 40px; height: 20px; background: #ccc; }
`))

	element := placedByElementID(t, PlacedElements(res), "plain")

	if element.Transform.HasMatrix {
		t.Error("#plain HasMatrix = true, want false")
	}

	if !element.Transform.Own.IsIdentity() {
		t.Errorf("#plain Own = %+v, want identity", element.Transform.Own)
	}

	if element.Transform.Parent != element.Transform.World {
		t.Errorf("#plain Parent = %+v, World = %+v, want equal", element.Transform.Parent, element.Transform.World)
	}

	if !element.Transform.World.IsIdentity() {
		t.Errorf("#plain World = %+v, want identity for a top-level box", element.Transform.World)
	}

	// 40x20 CSS px is 30x15 layout points, so the default 50% 50% origin is
	// (15, 7.5) points.
	if element.Transform.OriginX != 15 || element.Transform.OriginY != 7.5 {
		t.Errorf("#plain origin = (%v, %v), want (15, 7.5)", element.Transform.OriginX, element.Transform.OriginY)
	}
}
