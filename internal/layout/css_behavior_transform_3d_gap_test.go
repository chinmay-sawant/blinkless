package layout

import (
	"math"
	"strconv"
	"testing"
)

// Gap tests for the 3D leftovers: translate3d, scale3d, the Z component of
// the translate and scale properties, the rotate axis form, and the layout
// stamps that were parse-only. Reference: Chrome 143.0.7499.40.

func gap3DFirst(t *testing.T, res *Result) Matrix2D {
	t.Helper()

	forms := perspBuildXforms(res)
	if len(forms) == 0 {
		t.Fatal("want a stamped matrix, got none")
	}

	return forms[0]
}

//nolint:wsl // each fixture's matrix is checked immediately
func TestBehaviorTranslate3dAndScale3d(t *testing.T) {
	t.Parallel()

	moved := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:40px;height:20px;background:#f00;transform:translate3d(10px, 0, 0)"></div>`+
		`</body></html>`)
	if got := gap3DFirst(t, moved).E; !near(got, pxToPt(10)) && math.Abs(got) < 1 {
		t.Errorf("translate3d E = %g, want a non-zero shift of 10px", got)
	}

	scaled := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:40px;height:20px;background:#f00;transform:scale3d(2, 3, 1)"></div>`+
		`</body></html>`)
	got := gap3DFirst(t, scaled)
	if !near(got.A, 2) || !near(got.D, 3) {
		t.Errorf("scale3d linear = A %g D %g, want 2 and 3", got.A, got.D)
	}

	rejected := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:40px;height:20px;background:#f00;transform:scale3d(2)"></div>`+
		`</body></html>`)
	if boxByID(t, rejected, "a").style.HasTransform {
		t.Error("scale3d(2) applied, want the one-number form rejected")
	}
}

func TestBehaviorTranslateZUnderPerspective(t *testing.T) {
	t.Parallel()

	doc := func(camera string) string {
		return `<html><body style="margin:0"><div id="outer" style="width:200px;height:100px;` + camera + `">` +
			`<div id="inner" style="width:40px;height:20px;background:#f00;translate:0 0 100px"></div>` +
			`</div></body></html>`
	}

	plain := layoutHTML(t, doc(""))
	if perspSawXform(plain) {
		t.Error("translate:0 0 100px with no perspective stamped a matrix, want identity depth")
	}

	seen := gap3DFirst(t, layoutHTML(t, doc("perspective:500px")))
	if near(seen.A, 1) {
		t.Errorf("translate Z under perspective A = %g, want foreshortening away from 1", seen.A)
	}
}

func TestBehaviorScaleZUnderPerspective(t *testing.T) {
	t.Parallel()

	doc := func(camera, scale string) string {
		return `<html><body style="margin:0"><div id="outer" style="width:200px;height:100px;` + camera + `">` +
			`<div id="inner" style="width:40px;height:20px;background:#f00;scale:` + scale + `"></div>` +
			`</div></body></html>`
	}

	flat := layoutHTML(t, doc("", "1 1 2"))
	if perspSawXform(flat) {
		t.Error("scale:1 1 2 with no perspective stamped a matrix, want identity depth")
	}

	if !boxByID(t, flat, "inner").style.HasTransform3D {
		t.Error("scale:1 1 2 HasTransform3D = false, want the Z side matrix stored")
	}

	// The painted plane is z=0. scaleZ does not move those corners, so the
	// perspective camera has nothing to foreshorten. translateZ in the same
	// camera does, and that fixture is TestBehaviorTranslateZUnderPerspective.
	seen := layoutHTML(t, doc("perspective:500px", "1 1 2"))
	if perspSawXform(seen) {
		t.Error("scale:1 1 2 under perspective stamped a matrix, want the z=0 plane left alone")
	}
}

//nolint:wsl // the baked matrix follows the layout call
func TestBehaviorRotateAxisForm(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:40px;height:20px;background:#f00;rotate:y 60deg"></div>`+
		`</body></html>`)
	got := gap3DFirst(t, res)
	if !near(got.A, 0.5) || !near(got.D, 1) {
		t.Errorf("rotate:y 60deg A %g D %g, want 0.5 and 1", got.A, got.D)
	}
}

//nolint:wsl // the baked matrix follows the layout call
func TestBehaviorRotateXNarrowsHeight(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:40px;height:20px;background:#f00;transform:rotateX(60deg)"></div>`+
		`</body></html>`)
	got := gap3DFirst(t, res)
	if !near(got.A, 1) || !near(got.D, 0.5) {
		t.Errorf("rotateX(60deg) A %g D %g, want 1 and 0.5", got.A, got.D)
	}
}

//nolint:wsl // the two matrices are compared together
func TestBehaviorRotateZMatchesRotate(t *testing.T) {
	t.Parallel()

	turn := func(decl string) Matrix2D {
		res := layoutHTML(t, `<html><body style="margin:0">`+
			`<div id="a" style="width:40px;height:20px;background:#f00;`+decl+`"></div></body></html>`)

		return gap3DFirst(t, res)
	}

	plain := turn("transform:rotate(90deg)")
	side := turn("transform:rotateZ(90deg)")
	if !near(plain.A, side.A) || !near(plain.B, side.B) || !near(plain.C, side.C) || !near(plain.D, side.D) {
		t.Errorf("rotateZ linear (%g,%g,%g,%g) != rotate (%g,%g,%g,%g)",
			side.A, side.B, side.C, side.D, plain.A, plain.B, plain.C, plain.D)
	}
}

//nolint:wsl // the baked matrix follows the layout call
func TestBehaviorRotate3dMatchesRotateY(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:40px;height:20px;background:#f00;transform:rotate3d(0, 1, 0, 60deg)"></div>`+
		`</body></html>`)
	got := gap3DFirst(t, res)
	if !near(got.A, 0.5) || !near(got.D, 1) {
		t.Errorf("rotate3d Y A %g D %g, want 0.5 and 1", got.A, got.D)
	}
}

func TestBehaviorScaleZFunction(t *testing.T) {
	t.Parallel()

	alone := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:40px;height:20px;background:#f00;transform:scaleZ(2)"></div></body></html>`)
	if perspSawXform(alone) {
		t.Error("scaleZ(2) with no perspective stamped a matrix, want none")
	}

	doc := func(camera string) string {
		return `<html><body style="margin:0"><div id="outer" style="width:200px;height:100px;` + camera + `">` +
			`<div id="inner" style="width:40px;height:20px;background:#f00;transform:scaleZ(2)"></div>` +
			`</div></body></html>`
	}

	under := layoutHTML(t, doc("perspective:500px"))
	if !boxByID(t, under, "inner").style.HasTransform3D {
		t.Fatal("scaleZ(2) HasTransform3D = false, want the side matrix stored")
	}

	if perspSawXform(under) {
		t.Error("scaleZ(2) under perspective stamped a matrix, want the z=0 plane left alone")
	}
}

func TestBehaviorTranslateZLayoutForeshortens(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="outer" style="width:200px;height:100px;perspective:500px">`+
		`<div id="inner" style="width:40px;height:20px;background:#f00;transform:translateZ(250px)"></div>`+
		`</div></body></html>`)
	if got := gap3DFirst(t, res).A; !near(got, 2) {
		t.Errorf("translateZ(250px) under 500px A = %g, want 2", got)
	}
}

//nolint:wsl // the baked matrix follows the layout call
func TestBehaviorMatrix3DRotateY(t *testing.T) {
	t.Parallel()

	cos := math.Cos(60 * math.Pi / 180)
	sin := math.Sin(60 * math.Pi / 180)
	num := func(v float64) string {
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	matrix := "matrix3d(" + num(cos) + ",0," + num(-sin) + ",0,0,1,0,0," + num(sin) + ",0," + num(cos) + ",0,0,0,0,1)"
	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:40px;height:20px;background:#f00;transform:`+matrix+`"></div></body></html>`)
	if got := gap3DFirst(t, res).A; !near(got, 0.5) {
		t.Errorf("matrix3d rotateY A = %g, want 0.5", got)
	}
}

//nolint:cyclop,wsl // hidden, visible, and the 2D parent stay one test
func TestBehaviorBackfaceHiddenCullsComposedChild(t *testing.T) {
	t.Parallel()

	doc := func(parentTransform, childVis string) string {
		return `<html><body style="margin:0">` +
			`<div id="outer" style="width:100px;height:80px;transform-style:preserve-3d;backface-visibility:visible;transform:` +
			parentTransform + `">` +
			`<div id="inner" style="width:80px;height:40px;background-color:#ff0000;backface-visibility:` +
			childVis + `">child</div></div></body></html>`
	}

	hidden := backfaceRedFills(layoutHTML(t, doc("rotateY(180deg)", "hidden")))
	if len(hidden) == 0 {
		t.Fatal("want red fill ops, got 0")
	}

	for _, op := range hidden {
		if op.W != 0 || op.H != 0 {
			t.Errorf("composed hidden child fill = %.4f x %.4f, want 0 x 0", op.W, op.H)
		}
	}

	visible := backfaceRedFills(layoutHTML(t, doc("rotateY(180deg)", "visible")))
	saw := false
	for _, op := range visible {
		if op.W > 1 && op.H > 1 {
			saw = true
		}
	}

	if !saw {
		t.Error("visible composed child paints no live red fill")
	}

	flatParent := backfaceRedFills(layoutHTML(t, `<html><body style="margin:0">`+
		`<div style="width:100px;height:80px;transform:rotate(10deg)">`+
		`<div style="width:80px;height:40px;background-color:#ff0000;backface-visibility:hidden">child</div>`+
		`</div></body></html>`))
	kept := false
	for _, op := range flatParent {
		if op.W > 1 && op.H > 1 {
			kept = true
		}
	}

	if !kept {
		t.Error("2D rotate parent culled the hidden child, want the fill kept")
	}
}
