package layout

import "testing"

// WebKit aliases for the 3D transform properties. Each test declares the
// prefixed name and checks the same used value or baked matrix as the
// unprefixed property.

// TestBehaviorWebkitPerspectiveAliasForeshortens is the ancestor perspective
// fixture with -webkit-perspective:500px. The child's baked matrix must
// differ from the orthographic rotateY(60deg) bake.
func TestBehaviorWebkitPerspectiveAliasForeshortens(t *testing.T) {
	t.Parallel()

	doc := func(containerDecl string) string {
		return `<html><body style="margin:0">` +
			`<div id="outer" style="width:200px;height:100px;` + containerDecl + `">` +
			`<div id="inner" style="width:50px;height:50px;background-color:#00ff00;transform:rotateY(60deg)">sample</div>` +
			`</div></body></html>`
	}

	orthoRes := layoutHTML(t, doc(""))
	perspRes := layoutHTML(t, doc("-webkit-perspective:500px;"))

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

	if !outer.style.HasPerspective || outer.style.PerspectiveDist <= 0 {
		t.Errorf("outer HasPerspective=%v PerspectiveDist=%g, want a positive camera from -webkit-perspective",
			outer.style.HasPerspective, outer.style.PerspectiveDist)
	}

	perspBuildAssertCamera(t, perspInner, outer)

	orthoForms := perspBuildXforms(orthoRes)
	perspForms := perspBuildXforms(perspRes)

	perspBuildRequireXforms(t, orthoForms, perspForms)
	perspBuildAssertDiffers(t, orthoForms, perspForms)
}

// TestBehaviorWebkitPerspectiveOriginAlias stores 25% 75% as declared
// percents, the same fields perspective-origin writes. Paint resolves those
// later as a quarter of the width and three quarters of the height.
//
//nolint:cyclop // the origin fields are checked one by one
func TestBehaviorWebkitPerspectiveOriginAlias(t *testing.T) {
	t.Parallel()

	alias := originBuildBox(t, "-webkit-perspective-origin:25% 75%")
	plain := originBuildBox(t, "perspective-origin:25% 75%")

	if !alias.style.PerspectiveOriginSet {
		t.Errorf("PerspectiveOriginSet = false, want true")
	}

	if !alias.style.PerspectiveOriginXPct || !alias.style.PerspectiveOriginYPct {
		t.Errorf("origin percent flags = %v/%v, want true/true",
			alias.style.PerspectiveOriginXPct, alias.style.PerspectiveOriginYPct)
	}

	if !near(alias.style.PerspectiveOriginX, 25) || !near(alias.style.PerspectiveOriginY, 75) {
		t.Errorf("stored origin = (%g, %g), want declared percents (25, 75)",
			alias.style.PerspectiveOriginX, alias.style.PerspectiveOriginY)
	}

	sameX := near(alias.style.PerspectiveOriginX, plain.style.PerspectiveOriginX)
	sameY := near(alias.style.PerspectiveOriginY, plain.style.PerspectiveOriginY)
	sameFlags := alias.style.PerspectiveOriginXPct == plain.style.PerspectiveOriginXPct &&
		alias.style.PerspectiveOriginYPct == plain.style.PerspectiveOriginYPct &&
		alias.style.PerspectiveOriginSet == plain.style.PerspectiveOriginSet

	if !sameX || !sameY || !sameFlags {
		t.Errorf("webkit origin set=%v x=%g pct=%v y=%g pct=%v, unprefixed set=%v x=%g pct=%v y=%g pct=%v",
			alias.style.PerspectiveOriginSet, alias.style.PerspectiveOriginX, alias.style.PerspectiveOriginXPct,
			alias.style.PerspectiveOriginY, alias.style.PerspectiveOriginYPct,
			plain.style.PerspectiveOriginSet, plain.style.PerspectiveOriginX, plain.style.PerspectiveOriginXPct,
			plain.style.PerspectiveOriginY, plain.style.PerspectiveOriginYPct)
	}
}

// webkitPreserveDoc is preserveBuildDoc with the prefixed transform-style.
// preserveBuildDoc hardcodes the unprefixed property name.
func webkitPreserveDoc() string {
	return `<html><body style="margin:0">` +
		`<div id="outer" style="width:100px;height:100px;transform:rotateY(60deg);-webkit-transform-style:preserve-3d">` +
		`<div id="inner" style="width:50px;height:50px;background-color:#00ff00;transform:rotateY(-60deg)">x</div>` +
		`</div></body></html>`
}

// TestBehaviorWebkitTransformStyleAliasPreserve3D nests rotateY(60deg) over
// rotateY(-60deg). The prefixed preserve-3d alias composes to a baked A of
// about 1, the same net identity as the unprefixed property.
func TestBehaviorWebkitTransformStyleAliasPreserve3D(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, webkitPreserveDoc())
	outer := boxByID(t, res, "outer")

	if got := outer.style.TransformStyle; got != "preserve-3d" {
		t.Errorf("outer transform-style used = %q, want preserve-3d", got)
	}

	inner := preserveBuildInnerXform(t, res)

	if !near(inner.A, 1) {
		t.Errorf("preserve-3d nested rotateY A = %.4f, want about 1 (counter-rotation nets to identity)", inner.A)
	}

	if !near(inner.D, 1) {
		t.Errorf("preserve-3d nested rotateY D = %.4f, want 1", inner.D)
	}
}

// webkitBackfaceDoc is backfaceDoc with the prefixed visibility property.
func webkitBackfaceDoc(vis string) string {
	return `<html><body style="margin:0">` +
		`<div id="a" style="width:100px;height:50px;background-color:#ff0000;transform:rotateY(180deg);` +
		`-webkit-backface-visibility:` + vis + `"></div></body></html>`
}

// TestBehaviorWebkitBackfaceAliasHides culls a rotateY(180deg) red box when
// the prefixed visibility is hidden, and still paints it when the alias is
// visible.
//
//nolint:cyclop // hidden and visible are one alias test
func TestBehaviorWebkitBackfaceAliasHides(t *testing.T) {
	t.Parallel()

	hidden := layoutHTML(t, webkitBackfaceDoc("hidden"))
	hiddenFills := backfaceRedFills(hidden)

	if len(hiddenFills) == 0 {
		t.Fatal("want red fill ops for hidden, got 0")
	}

	for _, op := range hiddenFills {
		if op.W != 0 || op.H != 0 {
			t.Errorf("hidden rotateY(180deg) fill = %.4fpt x %.4fpt, want 0 x 0 (culled)",
				op.W, op.H)
		}
	}

	if got := boxByID(t, hidden, "a").style.BackfaceVisibility; got != "hidden" {
		t.Errorf("backface-visibility used = %q, want hidden", got)
	}

	visible := layoutHTML(t, webkitBackfaceDoc("visible"))
	visibleFills := backfaceRedFills(visible)

	if len(visibleFills) == 0 {
		t.Fatal("want red fill ops for visible, got 0")
	}

	sawPaint := false

	for _, op := range visibleFills {
		if op.W > 0 && op.H > 0 {
			sawPaint = true
		}
	}

	if !sawPaint {
		t.Errorf("visible rotateY(180deg) paints no live red fill, want a non-zero fill (first is %.4fpt x %.4fpt)",
			visibleFills[0].W, visibleFills[0].H)
	}

	if got := boxByID(t, visible, "a").style.BackfaceVisibility; got != "visible" {
		t.Errorf("backface-visibility used = %q, want visible", got)
	}
}

// TestBehaviorWebkitPerspectiveCascadeOrder checks both inline orders on one
// element. The later declaration wins after the alias maps to perspective
// (500px is 375pt, 100px is 75pt).
func TestBehaviorWebkitPerspectiveCascadeOrder(t *testing.T) {
	t.Parallel()

	boxFor := func(decl string) *box {
		t.Helper()

		res := layoutHTML(t, `<html><body style="margin:0">`+
			`<div id="a" style="`+decl+`"></div></body></html>`)

		return boxByID(t, res, "a")
	}

	standardLast := boxFor("-webkit-perspective:100px;perspective:500px")
	aliasLast := boxFor("perspective:500px;-webkit-perspective:100px")

	if !standardLast.style.HasPerspective || !near(standardLast.style.PerspectiveDist, pxToPt(500)) {
		t.Errorf("standard last HasPerspective=%v dist=%g, want true and %g (500px)",
			standardLast.style.HasPerspective, standardLast.style.PerspectiveDist, pxToPt(500))
	}

	if !aliasLast.style.HasPerspective || !near(aliasLast.style.PerspectiveDist, pxToPt(100)) {
		t.Errorf("alias last HasPerspective=%v dist=%g, want true and %g (100px)",
			aliasLast.style.HasPerspective, aliasLast.style.PerspectiveDist, pxToPt(100))
	}
}
