package layout

import (
	"math"
	"testing"
)

// Foundation tests for the 3D core: matrix math, orthographic and perspective
// flattening, facing signs, and 3D function parsing. No layout involved.
// Reference: Chrome 143.0.7499.40.

func TestTransform3DIdentityFlattensToIdentity(t *testing.T) {
	t.Parallel()

	baked, facing := identity3D().flatten3D(0, 0, 0)

	if !baked.IsIdentity() {
		t.Errorf("identity flatten = %+v, want identity", baked)
	}

	if facing <= 0 {
		t.Errorf("identity facing = %g, want positive", facing)
	}
}

func TestTransform3DRotateYNarrowsOrthographic(t *testing.T) {
	t.Parallel()

	baked, facing := rotateY3D(60).flatten3D(0, 0, 0)

	if !near(baked.A, 0.5) || !near(baked.D, 1) {
		t.Errorf("rotateY(60) ortho = A %g D %g, want 0.5 and 1", baked.A, baked.D)
	}

	if facing <= 0 {
		t.Errorf("rotateY(60) facing = %g, want positive", facing)
	}
}

func TestTransform3DRotateYHalfTurnFacesAway(t *testing.T) {
	t.Parallel()

	baked, facing := rotateY3D(180).flatten3D(0, 0, 0)

	if !near(math.Abs(baked.A), 1) {
		t.Errorf("rotateY(180) ortho |A| = %g, want 1", math.Abs(baked.A))
	}

	if facing >= 0 {
		t.Errorf("rotateY(180) facing = %g, want negative", facing)
	}
}

func TestTransform3DPerspectiveForeshortens(t *testing.T) {
	t.Parallel()

	// translateZ toward the camera doubles apparent size at half the camera
	// distance; orthographic projection keeps size 1.
	ortho, _ := translateZ3D(pxToPt(250)).flatten3D(0, 0, 0)
	proj, _ := translateZ3D(pxToPt(250)).flatten3D(0, 0, pxToPt(500))

	if !near(ortho.A, 1) {
		t.Errorf("translateZ ortho A = %g, want 1", ortho.A)
	}

	if !near(proj.A, 2) {
		t.Errorf("translateZ perspective A = %g, want 2", proj.A)
	}
}

func TestTransform3DParseAccepts3DFuncs(t *testing.T) {
	t.Parallel()

	for _, src := range []string{
		"rotateX(30deg)", "rotateY(30deg)", "rotateZ(30deg)",
		"rotate3d(0, 1, 0, 30deg)", "translateZ(10px)", "scaleZ(2)",
		"perspective(500px)", "matrix3d(1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1)",
	} {
		name, args, _, ok := splitTransformFunc(src)
		if !ok {
			t.Fatalf("splitTransformFunc(%q) failed", src)
		}

		if _, ok := parse3DFunc(name, args); !ok {
			t.Errorf("parse3DFunc(%q) rejected, want accepted", src)
		}
	}
}

func TestTransform3DParseRejectsBad3D(t *testing.T) {
	t.Parallel()

	for _, src := range []string{
		"rotateY()", "rotate3d(0, 0, 0, 30deg)",
		"translateZ(10%)", "perspective(0px)", "perspective(-5px)",
		"matrix3d(1,0,0,0)", "bogus3d(1deg)",
	} {
		name, args, _, ok := splitTransformFunc(src)
		if !ok {
			continue
		}

		if _, ok := parse3DFunc(name, args); ok {
			t.Errorf("parse3DFunc(%q) accepted, want rejected", src)
		}
	}
}
