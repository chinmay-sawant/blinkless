package layout

import (
	"testing"
)

// 3D gated properties and dominant-baseline follow-ups. The engine is 2D by
// architecture (documentation/deferred.md): paint carries one Matrix2D per op
// (layout.go:402, op_extra.go:16) and the transform parser rejects 3D
// functions (transform.go:351). Reference browser for every case: Chrome
// 143.0.7499.40. Every test below asserts used values (box geometry, op
// fields, style used values), never stored strings.

// perspAssertSameOps pins no-op behavior: two results must paint the same ops
// in the same order with identical kind, geometry, and text.
func perspAssertSameOps(t *testing.T, first, second *Result) {
	t.Helper()

	if len(first.Ops) != len(second.Ops) {
		t.Fatalf("op count = %d vs %d, want identical geometry", len(first.Ops), len(second.Ops))
	}

	for i := range first.Ops {
		x, y := first.Ops[i], second.Ops[i]
		if x.Kind != y.Kind || x.Text != y.Text ||
			!near(x.X, y.X) || !near(x.Y, y.Y) ||
			!near(x.W, y.W) || !near(x.H, y.H) {
			t.Fatalf("op %d differs: %+v vs %+v, want identical geometry", i, x, y)
		}
	}
}

// perspSawXform reports whether any op carries a baked transform.
func perspSawXform(res *Result) bool {
	for _, op := range res.Ops {
		if op.XformSet {
			return true
		}
	}

	return false
}

// TestPerspPerspectiveNoEffectOn2DChild pins the correct 2D subset:
// perspective:500px on the container stores the camera but only 3D-bearing
// descendants project through it, so a 2D rotate child paints identically
// with or without the property. Chrome 143.0.7499.40 leaves 2D children
// unforeshortened under an ancestor perspective.
func TestPerspPerspectiveNoEffectOn2DChild(t *testing.T) {
	t.Parallel()

	doc := func(containerDecl string) string {
		return `<html><body style="margin:0">` +
			`<div id="outer" style="width:100px;height:100px;` + containerDecl + `">` +
			`<div id="inner" style="width:50px;height:50px;background-color:#00ff00;transform:rotate(10deg)"></div>` +
			`</div></body></html>`
	}

	plain := layoutHTML(t, doc(""))
	persp := layoutHTML(t, doc("perspective:500px;"))

	if got := boxByID(t, persp, "outer").style.HasTransform; got {
		t.Errorf("perspective container HasTransform = true, want false (property is dropped, not a 2D transform)")
	}

	inner := boxByID(t, persp, "inner")
	if !inner.style.HasTransform {
		t.Errorf("2D rotate child HasTransform = false, want true")
	}

	if inner.style.HasTransform3D {
		t.Errorf("2D rotate child HasTransform3D = true, want false (no 3D to project)")
	}

	if inner.hasPerspDist {
		t.Errorf("2D child hasPerspDist = true, want false (pre-pass fills 3D-bearing boxes only)")
	}

	perspAssertSameOps(t, plain, persp)
}

// TestPerspPerspectiveFunctionRejected pins the depthless function truth:
// perspective() parses onto the 3D side channel with an identity 2D
// contribution, and the 3D branch projects it with its own distance. A lone
// perspective() before a 2D rotate(10deg) carries no depth, so its bake stays
// identity and the list paints exactly like a 2D-only rotate. Chrome
// 143.0.7499.40 shows no foreshortening without depth; depthful lists pin
// their difference in css_behavior_perspective_build_test.go.
func TestPerspPerspectiveFunctionRejected(t *testing.T) {
	t.Parallel()

	rotated := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:100px;height:50px;transform:rotate(10deg)">`+
		`sample text here</div></body></html>`)
	funcForm := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:100px;height:50px;transform:perspective(500px) rotate(10deg)">`+
		`sample text here</div></body></html>`)

	if got := boxByID(t, funcForm, "a").style.HasTransform3D; !got {
		t.Errorf("perspective() list HasTransform3D = false, want true (3D parses onto the side channel)")
	}

	perspAssertSameOps(t, rotated, funcForm)
}

// TestPerspRotate3DRejected pins the projected orthographic truth.
// rotateX and rotateY now narrow through the boxTransformAccum 3D branch
// (baked Xform A/D differ from plain) while layout geometry (X/Y/W/H) stays
// put because the bake rides the separate Xform channel. translateZ and the
// identity matrix3d stay identity orthographically: depth alone only shows
// through a perspective divide. One 2D control (rotate) proves the harness
// observes a transform when one is kept. Reference: Chrome 143.0.7499.40.
func TestPerspRotate3DRejected(t *testing.T) {
	t.Parallel()

	plainDoc := `<html><body style="margin:0">` +
		`<div id="a" style="width:100px;height:50px">sample text here</div></body></html>`
	plain := layoutHTML(t, plainDoc)

	for _, decl := range []string{
		"transform:rotateX(45deg)",
		"transform:rotateY(45deg)",
		"transform:translateZ(100px)",
		"transform:matrix3d(1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1)",
	} {
		res := layoutHTML(t, `<html><body style="margin:0">`+
			`<div id="a" style="width:100px;height:50px;`+decl+`">sample text here</div></body></html>`)
		if got := boxByID(t, res, "a").style.HasTransform3D; !got {
			t.Errorf("%s HasTransform3D = false, want true (3D parses onto the side channel)", decl)
		}

		perspAssertSameOps(t, plain, res)

		sawXform := perspSawXform(res)

		switch decl {
		case "transform:rotateX(45deg)", "transform:rotateY(45deg)":
			if !sawXform {
				t.Errorf("%s stamps no Xform, want narrowed orthographic bake", decl)
			}
		default:
			if sawXform {
				t.Errorf("%s stamps Xform, want identity orthographic bake (depth needs perspective)", decl)
			}
		}
	}

	control := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:100px;height:50px;transform:rotate(10deg)">sample text here</div></body></html>`)
	if got := boxByID(t, control, "a").style.HasTransform; !got {
		t.Errorf("2D rotate control HasTransform = false, want true")
	}
}

// TestPerspPerspectiveOriginNoEffect pins the perspective-origin gap.
// perspective-origin stores onto the style (applyPerspectiveOriginProperty)
// but perspectiveCenter still returns the default center. Chrome
// 143.0.7499.40 would move the vanishing point of a perspective projection,
// but with no origin consumer the 2D drawing list is identical, alone or
// combined with perspective.
func TestPerspPerspectiveOriginNoEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:100px;height:50px">sample text here</div></body></html>`)
	origin := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:100px;height:50px;perspective-origin:25% 75%">sample text here</div></body></html>`)
	combined := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:100px;height:50px;perspective:500px;perspective-origin:25% 75%">`+
		`sample text here</div></body></html>`)

	if got := boxByID(t, origin, "a").style.HasTransform; got {
		t.Errorf("perspective-origin HasTransform = true, want false (property is dropped)")
	}

	perspAssertSameOps(t, plain, origin)
	perspAssertSameOps(t, plain, combined)
}

// TestPerspTransformStyleFlatteningIdentical pins the 2D-only subset: nested
// 2D transforms carry no 3D side channel, so the preserve-3d chain helpers in
// transform.go pass them through and flat and preserve-3d still paint
// identical geometry. Both values store canonically
// (style_leftovers.go:139-148); stampBoxTransformsRec threads TransformStyle
// and only diverges when HasTransform3D enters the chain (see
// css_behavior_preserve_build_test.go). Reference: Chrome 143.0.7499.40 keeps
// a 3D child in the parent plane under preserve-3d and flattens it under
// flat; with no 3D functions that difference cannot appear.
func TestPerspTransformStyleFlatteningIdentical(t *testing.T) {
	t.Parallel()

	doc := func(style string) string {
		return `<html><body style="margin:0">` +
			`<div id="outer" style="width:100px;height:100px;transform:rotate(10deg);transform-style:` + style + `">` +
			`<div id="mid" style="width:80px;height:80px;transform:rotate(-4deg)">` +
			`<div id="inner" style="width:50px;height:50px;background-color:#00ff00;transform:scale(1.2)"></div>` +
			`</div></div></body></html>`
	}

	flat := layoutHTML(t, doc("flat"))
	preserved := layoutHTML(t, doc("preserve-3d"))

	if got := boxByID(t, flat, "outer").style.TransformStyle; got != "flat" {
		t.Errorf("transform-style used = %q, want flat", got)
	}

	if got := boxByID(t, preserved, "outer").style.TransformStyle; got != "preserve-3d" {
		t.Errorf("transform-style used = %q, want preserve-3d", got)
	}

	perspAssertSameOps(t, flat, preserved)

	sawXform := false

	for _, op := range preserved.Ops {
		if op.XformSet {
			sawXform = true

			break
		}
	}

	if !sawXform {
		t.Errorf("nested 2D rotates should stamp at least one Xform op, saw none")
	}

	bogus := boxByID(t, layoutHTML(t, doc("bogus")), "outer")
	if bogus.style.TransformStyle != "flat" {
		t.Errorf("transform-style:bogus used = %q, want initial flat", bogus.style.TransformStyle)
	}
}

// TestPerspDominantBaselineBeyondHanging is the observable follow-up: every
// stored selector beyond hanging shifts the run from font metrics in
// baselineShiftFromMetrics (inline_vertical_align.go:133-157), applied only
// on the baseline default via alignmentBaselineShift
// (inline_vertical_align.go:169-187). At 16px type Chrome 143.0.7499.40 puts
// hanging about 5px above alphabetic; ideographic sits below alphabetic while
// middle, central, and the before edge sit above it.
func TestPerspDominantBaselineBeyondHanging(t *testing.T) {
	t.Parallel()

	doc := func(decl string) string {
		return `<html><body style="margin:0">` +
			`<p id="p" style="margin:0;font-size:16px;` + decl + `">Ag</p></body></html>`
	}

	plainRes := layoutHTML(t, doc(""))
	plainTexts := opsOfKind(plainRes, OpText)

	if len(plainTexts) == 0 {
		t.Fatalf("want plain text ops, got 0")
	}

	plainY := plainTexts[0].Y

	raises := []string{"middle", "central", "mathematical", "text-before-edge"}
	for _, sel := range raises {
		res := layoutHTML(t, doc("dominant-baseline:"+sel))
		if got := boxByID(t, res, "p").style.DominantBaseline; got != sel {
			t.Errorf("dominant-baseline used = %q, want %q", got, sel)
		}

		texts := opsOfKind(res, OpText)
		if len(texts) == 0 {
			t.Fatalf("dominant-baseline:%s: want text ops, got 0", sel)
		}

		if lift := plainY - texts[0].Y; lift <= 0.5 {
			t.Errorf("dominant-baseline:%s lift = %.4fpt, want above alphabetic (positive lift)", sel, lift)
		}
	}

	lowers := []string{"ideographic", "text-after-edge"}
	for _, sel := range lowers {
		res := layoutHTML(t, doc("dominant-baseline:"+sel))
		if got := boxByID(t, res, "p").style.DominantBaseline; got != sel {
			t.Errorf("dominant-baseline used = %q, want %q", got, sel)
		}

		texts := opsOfKind(res, OpText)
		if len(texts) == 0 {
			t.Fatalf("dominant-baseline:%s: want text ops, got 0", sel)
		}

		if lift := plainY - texts[0].Y; lift >= -0.5 {
			t.Errorf("dominant-baseline:%s lift = %.4fpt, want below alphabetic (negative lift)", sel, lift)
		}
	}
}

// TestPerspDominantBaselineInitialAndBogus pins the selector edges: auto is
// the initial (style.go initial values), alphabetic shares its zero shift in
// baselineShiftFromMetrics (inline_vertical_align.go:135), and an unknown
// token leaves the previous declaration intact
// (style_leftovers.go:192-198). auto and alphabetic therefore paint
// identical text positions.
func TestPerspDominantBaselineInitialAndBogus(t *testing.T) {
	t.Parallel()

	doc := func(decl string) string {
		return `<html><body style="margin:0">` +
			`<p id="p" style="margin:0;font-size:16px;` + decl + `">Ag</p></body></html>`
	}

	defStyle := boxByID(t, layoutHTML(t, doc("")), "p")
	if defStyle.style.DominantBaseline != "auto" {
		t.Errorf("dominant-baseline initial used = %q, want auto", defStyle.style.DominantBaseline)
	}

	alphaRes := layoutHTML(t, doc("dominant-baseline:alphabetic"))
	if got := boxByID(t, alphaRes, "p").style.DominantBaseline; got != "alphabetic" {
		t.Errorf("dominant-baseline used = %q, want alphabetic", got)
	}

	perspAssertSameOps(t, layoutHTML(t, doc("")), alphaRes)

	bogus := boxByID(t, layoutHTML(t, doc("dominant-baseline:bogus")), "p")
	if bogus.style.DominantBaseline != "auto" {
		t.Errorf("dominant-baseline:bogus used = %q, want initial auto", bogus.style.DominantBaseline)
	}
}

// TestPerspDominantBaselineInherits pins the inherited 2D subset:
// style_cascade.go:517 copies DominantBaseline from the parent when the
// element does not declare it, so a hanging declaration on the parent raises
// the child run without a declaration of its own.
func TestPerspDominantBaselineInherits(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<div><p id="p" style="margin:0;font-size:16px">Ag</p></div></body></html>`)
	inherited := layoutHTML(t, `<html><body style="margin:0">`+
		`<div style="dominant-baseline:hanging"><p id="p" style="margin:0;font-size:16px">Ag</p></div></body></html>`)

	child := boxByID(t, inherited, "p")
	if child.style.DominantBaseline != "hanging" {
		t.Errorf("inherited dominant-baseline used = %q, want hanging", child.style.DominantBaseline)
	}

	plainTexts := opsOfKind(plain, OpText)
	inhTexts := opsOfKind(inherited, OpText)

	if len(plainTexts) == 0 || len(inhTexts) == 0 {
		t.Fatalf("want text ops, got plain=%d inherited=%d", len(plainTexts), len(inhTexts))
	}

	if lift := plainTexts[0].Y - inhTexts[0].Y; lift <= 0.5 {
		t.Errorf("inherited hanging lift = %.4fpt, want positive lift above alphabetic", lift)
	}
}
