package layout

import (
	"testing"
)

// Preserve-3d build behavior. Each test lays out a nested rotateY pair and
// asserts the baked display-list matrix, never stored strings. Reference
// browser for every case: Chrome 143.0.7499.40.
//
// Canonical divergence: parent rotateY(60) narrows to scaleX 0.5 under
// orthographic projection. A child rotateY(-60) counter-rotates back to the
// viewer plane under preserve-3d (chain composes in 3D, then flattens once:
// net identity) but compounds to scaleX 0.25 under flat (each level flattens
// in order via boxTransformAccum in transform.go).

// preserveBuildDoc nests a 3D child under a 3D outer with the given outer
// transform-style. The inner fill rect carries the composed matrix.
func preserveBuildDoc(outerStyle string) string {
	return `<html><body style="margin:0">` +
		`<div id="outer" style="width:100px;height:100px;transform:rotateY(60deg);transform-style:` + outerStyle + `">` +
		`<div id="inner" style="width:50px;height:50px;background-color:#00ff00;transform:rotateY(-60deg)">x</div>` +
		`</div></body></html>`
}

// preserveBuildInnerXform returns the baked matrix of the inner fill op.
func preserveBuildInnerXform(t *testing.T, res *Result) Matrix2D {
	t.Helper()

	for _, op := range res.Ops {
		if op.Kind == OpFillRect && op.XformSet {
			return op.Transform()
		}
	}

	t.Fatal("want an inner fill op with a baked transform, got none")

	return IdentityMatrix()
}

// TestPreserveBuildNestedCounterRotationDiverges is the canonical divergence:
// rotateY(60) over rotateY(-60) nets to identity under preserve-3d but
// compounds to scaleX 0.25 under flat. Reference: Chrome 143.0.7499.40.
func TestPreserveBuildNestedCounterRotationDiverges(t *testing.T) {
	t.Parallel()

	flat := layoutHTML(t, preserveBuildDoc("flat"))
	preserved := layoutHTML(t, preserveBuildDoc("preserve-3d"))

	flatBox := boxByID(t, flat, "outer")
	if got := flatBox.style.TransformStyle; got != "flat" {
		t.Errorf("outer transform-style used = %q, want flat", got)
	}

	preservedBox := boxByID(t, preserved, "outer")
	if got := preservedBox.style.TransformStyle; got != "preserve-3d" {
		t.Errorf("outer transform-style used = %q, want preserve-3d", got)
	}

	flatInner := boxByID(t, flat, "inner")
	if !flatInner.style.HasTransform3D {
		t.Errorf("inner HasTransform3D = false, want true (rotateY parses onto the side channel)")
	}

	flatM := preserveBuildInnerXform(t, flat)
	preservedM := preserveBuildInnerXform(t, preserved)

	t.Logf("flat inner A=%.4f D=%.4f; preserve inner A=%.4f D=%.4f", flatM.A, flatM.D, preservedM.A, preservedM.D)

	if !near(flatM.A, 0.25) {
		t.Errorf("flat nested rotateY A = %.4f, want about 0.25 (0.5 per level compounded)", flatM.A)
	}

	if !near(flatM.D, 1) {
		t.Errorf("flat nested rotateY D = %.4f, want 1 (no vertical foreshorten)", flatM.D)
	}

	if !near(preservedM.A, 1) {
		t.Errorf("preserve-3d nested rotateY A = %.4f, want about 1 (counter-rotation nets to identity)", preservedM.A)
	}

	if !near(preservedM.D, 1) {
		t.Errorf("preserve-3d nested rotateY D = %.4f, want 1", preservedM.D)
	}

	if near(flatM.A, preservedM.A) {
		t.Errorf("flat A=%.4f vs preserve A=%.4f look identical, want observable divergence", flatM.A, preservedM.A)
	}

	checkPreserveComposedFlags(t, preserved, flat)
}

// checkPreserveComposedFlags pins the per-level skip contract: the preserve
// chain marks composed boxes so boxTransformAccum keeps the 2D path, while
// flat boxes stay unmarked for per-level flattening.
func checkPreserveComposedFlags(t *testing.T, preserved, flat *Result) {
	t.Helper()

	outerPreserved := boxByID(t, preserved, "outer")
	innerPreserved := boxByID(t, preserved, "inner")

	if !outerPreserved.preserveComposed {
		t.Errorf("preserve outer preserveComposed = false, want true (chain owns the projection)")
	}

	if !innerPreserved.preserveComposed {
		t.Errorf("preserve inner preserveComposed = false, want true (chain owns the projection)")
	}

	outerFlat := boxByID(t, flat, "outer")
	innerFlat := boxByID(t, flat, "inner")

	if outerFlat.preserveComposed {
		t.Errorf("flat outer preserveComposed = true, want false (per-level flattening owns it)")
	}

	if innerFlat.preserveComposed {
		t.Errorf("flat inner preserveComposed = true, want false (per-level flattening owns it)")
	}
}

// TestPreserveBuildFlatBoundaryClearsChain pins the flat boundary rule: a
// plain flat middle clears the ancestor chain, so the grandchild starts fresh
// from the flattened ancestors. With a preserve-3d middle the chain continues
// and the grandchild still nets to identity. Reference: Chrome 143.0.7499.40.
func TestPreserveBuildFlatBoundaryClearsChain(t *testing.T) {
	t.Parallel()

	doc := func(midStyle string) string {
		return `<html><body style="margin:0">` +
			`<div id="outer" style="width:100px;height:100px;transform:rotateY(60deg);transform-style:preserve-3d">` +
			`<div id="mid" style="width:80px;height:80px;transform-style:` + midStyle + `">` +
			`<div id="inner" style="width:50px;height:50px;background-color:#00ff00;transform:rotateY(-60deg)">x</div>` +
			`</div></div></body></html>`
	}

	flatMid := layoutHTML(t, doc("flat"))
	preservedMid := layoutHTML(t, doc("preserve-3d"))

	flatM := preserveBuildInnerXform(t, flatMid)
	preservedM := preserveBuildInnerXform(t, preservedMid)

	t.Logf("flat-mid inner A=%.4f; preserve-mid inner A=%.4f", flatM.A, preservedM.A)

	if !near(preservedM.A, 1) {
		t.Errorf("preserve-mid inner A = %.4f, want about 1 (chain continues through the middle)", preservedM.A)
	}

	if !near(flatM.A, 0.25) {
		t.Errorf("flat-mid inner A = %.4f, want about 0.25 (middle clears the chain, fresh per-level)", flatM.A)
	}

	if near(flatM.A, preservedM.A) {
		t.Errorf("flat-mid A=%.4f vs preserve-mid A=%.4f look identical, want boundary divergence",
			flatM.A, preservedM.A)
	}
}
