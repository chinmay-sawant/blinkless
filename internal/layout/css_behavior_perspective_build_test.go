package layout

import (
	"testing"
)

// Perspective projection build tests: the pre-pass plus the boxTransformAccum
// 3D branch make perspective observable end to end. Every test asserts used
// baked matrices or painted geometry, never stored strings. Reference for
// all numeric choices: Chrome 143.0.7499.40.

// perspBuildXforms returns the baked transforms stamped onto ops.
func perspBuildXforms(res *Result) []Matrix2D {
	var out []Matrix2D

	for _, op := range res.Ops {
		if op.XformSet {
			out = append(out, op.Transform())
		}
	}

	return out
}

// perspBuildDoc layouts a 100x50px box with the given declarations.
func perspBuildDoc(t *testing.T, decl string) *Result {
	t.Helper()

	return layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:100px;height:50px;background-color:#00ff00;`+decl+`">sample text here</div></body></html>`)
}

// perspBuildRequireXforms fails when either side stamps nothing.
func perspBuildRequireXforms(t *testing.T, first, second []Matrix2D) {
	t.Helper()

	if len(first) == 0 || len(second) == 0 {
		t.Fatalf("xformed ops = %d/%d, want both to stamp", len(first), len(second))
	}
}

// perspBuildAssertCamera pins the pre-pass fill against perspectiveCenter.
func perspBuildAssertCamera(t *testing.T, inner *box, outer *box) {
	t.Helper()

	if !inner.hasPerspDist {
		t.Fatalf("perspective child hasPerspDist = false, want true (pre-pass fills ancestor camera)")
	}

	wantDist := pxToPt(500)

	if !near(inner.perspDist, wantDist) {
		t.Errorf("perspDist = %g, want %g (500px camera)", inner.perspDist, wantDist)
	}

	wantCX, wantCY := perspectiveCenter(outer)

	if !near(inner.perspCX, wantCX) || !near(inner.perspCY, wantCY) {
		t.Errorf("vanishing point = (%g, %g), want perspectiveCenter(outer) (%g, %g)",
			inner.perspCX, inner.perspCY, wantCX, wantCY)
	}
}

// perspBuildAssertDiffers fails when every baked pair matches.
func perspBuildAssertDiffers(t *testing.T, first, second []Matrix2D) {
	t.Helper()

	for _, left := range first {
		for _, right := range second {
			if !near(left.A, right.A) || !near(left.E, right.E) {
				return
			}
		}
	}

	t.Errorf("ortho A %g vs perspective A %g, want foreshortened difference",
		first[0].A, second[0].A)
}

// TestPerspBuildRotateYNarrowsOrtho pins the orthographic base: rotateY(60deg)
// with no perspective narrows the baked width to cos60 = 0.5 while keeping
// height 1. Chrome 143.0.7499.40 narrows the face without foreshortening.
func TestPerspBuildRotateYNarrowsOrtho(t *testing.T) {
	t.Parallel()

	plain := perspBuildDoc(t, "")

	if got := perspBuildXforms(plain); len(got) != 0 {
		t.Fatalf("plain xformed ops = %d, want 0 (no transform stamps identity)", len(got))
	}

	rotated := perspBuildDoc(t, "transform:rotateY(60deg)")
	rotatedBox := boxByID(t, rotated, "a")

	if !rotatedBox.style.HasTransform3D {
		t.Fatalf("rotateY(60deg) HasTransform3D = false, want true")
	}

	if rotatedBox.hasPerspDist {
		t.Errorf("rotateY without ancestor hasPerspDist = true, want false (orthographic)")
	}

	got := perspBuildXforms(rotated)

	if len(got) == 0 {
		t.Fatalf("rotateY(60deg) stamps no Xform ops, want narrowed bake")
	}

	for _, baked := range got {
		if !near(baked.A, 0.5) || !near(baked.D, 1) {
			t.Errorf("rotateY(60deg) ortho bake = A %g D %g, want 0.5 and 1", baked.A, baked.D)
		}
	}
}

// TestPerspBuildAncestorPerspectiveForeshortens pins the property path: the
// same rotateY(60deg) child foreshortens differently under a
// perspective:500px ancestor than orthographically. The pre-pass fills the
// camera from the nearest ancestor via perspectiveCenter. Chrome
// 143.0.7499.40 foreshortens the child toward the ancestor vanishing point.
func TestPerspBuildAncestorPerspectiveForeshortens(t *testing.T) {
	t.Parallel()

	doc := func(containerDecl string) string {
		return `<html><body style="margin:0">` +
			`<div id="outer" style="width:200px;height:100px;` + containerDecl + `">` +
			`<div id="inner" style="width:50px;height:50px;background-color:#00ff00;transform:rotateY(60deg)">sample</div>` +
			`</div></body></html>`
	}

	orthoRes := layoutHTML(t, doc(""))
	perspRes := layoutHTML(t, doc("perspective:500px;"))

	orthoInner := boxByID(t, orthoRes, "inner")
	perspInner := boxByID(t, perspRes, "inner")
	outer := boxByID(t, perspRes, "outer")

	if !orthoInner.style.HasTransform3D || !perspInner.style.HasTransform3D {
		t.Fatalf("rotateY child HasTransform3D = %v/%v, want true/true",
			orthoInner.style.HasTransform3D, perspInner.style.HasTransform3D)
	}

	if orthoInner.hasPerspDist {
		t.Errorf("orthographic child hasPerspDist = true, want false")
	}

	perspBuildAssertCamera(t, perspInner, outer)

	orthoForms := perspBuildXforms(orthoRes)
	perspForms := perspBuildXforms(perspRes)

	perspBuildRequireXforms(t, orthoForms, perspForms)
	perspBuildAssertDiffers(t, orthoForms, perspForms)
}

// TestPerspBuildPerspectiveFunctionAppliesOwnDistance pins the transform-level
// path: perspective(500px) rotateY(60deg) bakes different geometry than
// rotateY(60deg) alone, with no ancestor involved. The 3D branch falls back
// to the function's own distance. Chrome 143.0.7499.40 applies the
// function-level camera to its own element.
func TestPerspBuildPerspectiveFunctionAppliesOwnDistance(t *testing.T) {
	t.Parallel()

	alone := perspBuildDoc(t, "transform:rotateY(60deg)")
	funcForm := perspBuildDoc(t, "transform:perspective(500px) rotateY(60deg)")

	aloneBox := boxByID(t, alone, "a")
	funcBox := boxByID(t, funcForm, "a")

	if !aloneBox.style.HasTransform3D || !funcBox.style.HasTransform3D {
		t.Fatalf("HasTransform3D = %v/%v, want true/true",
			aloneBox.style.HasTransform3D, funcBox.style.HasTransform3D)
	}

	if aloneBox.hasPerspDist || funcBox.hasPerspDist {
		t.Errorf("hasPerspDist = %v/%v, want false/false (no ancestor property)",
			aloneBox.hasPerspDist, funcBox.hasPerspDist)
	}

	aloneForms := perspBuildXforms(alone)
	funcForms := perspBuildXforms(funcForm)

	perspBuildRequireXforms(t, aloneForms, funcForms)

	if near(aloneForms[0].A, funcForms[0].A) && near(aloneForms[0].E, funcForms[0].E) {
		t.Errorf("perspective() bake A %g E %g matches plain rotateY, want own-distance difference",
			funcForms[0].A, funcForms[0].E)
	}
}

// TestPerspBuildPreserveComposedSkipsProjection pins the transform-style
// contract: when preserveComposed is set, boxTransformAccum keeps the 2D path
// and skips its own 3D branch. A pure rotateY(60deg) therefore stays identity
// instead of narrowing. Chrome 143.0.7499.40 composes preserve-3d chains
// elsewhere; this engine must not double-project.
func TestPerspBuildPreserveComposedSkipsProjection(t *testing.T) {
	t.Parallel()

	res := perspBuildDoc(t, "transform:rotateY(60deg)")
	rotatedBox := boxByID(t, res, "a")

	var viewport svgViewport

	accum := boxTransformAccum(rotatedBox, IdentityMatrix(), viewport)

	if !near(accum.A, 0.5) {
		t.Fatalf("projected accum A = %g, want 0.5 (ortho narrow)", accum.A)
	}

	rotatedBox.preserveComposed = true

	skipped := boxTransformAccum(rotatedBox, IdentityMatrix(), viewport)

	if !skipped.IsIdentity() {
		t.Errorf("preserveComposed accum = %+v, want identity (3D branch skipped)", skipped)
	}
}
