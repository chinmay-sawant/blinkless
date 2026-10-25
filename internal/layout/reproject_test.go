package layout

import (
	"errors"
	"testing"
)

// Restamp tests: a restamp pass has to produce exactly the operations a fresh
// layout with the same declaration produces, without re-cascading or
// re-placing. The comparison is the acceptance gate from the priority-0 plan.

// restampCompare pins one restamped display against a fresh layout op by op.
// Both sides are finalized, so a projected op is compared by the geometry it
// paints, not by the matrix it was projected from.
func restampCompare(t *testing.T, got, want *Result) {
	t.Helper()

	if len(got.Ops) != len(want.Ops) {
		t.Fatalf("ops = %d, want %d", len(got.Ops), len(want.Ops))
	}

	for idx := range want.Ops {
		restampCompareOp(t, idx, &got.Ops[idx], &want.Ops[idx])
	}
}

// restampCompareOp compares one operation's painted form.
func restampCompareOp(t *testing.T, index int, got, want *Op) {
	t.Helper()

	restampCompareGeometry(t, index, got, want)
	restampCompareTransform(t, index, got, want)
	restampCompareProjection(t, index, got, want)
	restampComparePayload(t, index, got, want)
}

// restampCompareGeometry compares the shape and the flat geometry.
func restampCompareGeometry(t *testing.T, index int, got, want *Op) {
	t.Helper()

	if got.Kind != want.Kind || got.X != want.X || got.Y != want.Y ||
		got.W != want.W || got.H != want.H || got.Text != want.Text {
		t.Fatalf("op %d: kind=%v geom=(%g,%g,%g,%g) text=%q, want kind=%v geom=(%g,%g,%g,%g) text=%q",
			index, got.Kind, got.X, got.Y, got.W, got.H, got.Text,
			want.Kind, want.X, want.Y, want.W, want.H, want.Text)
	}
}

// restampCompareTransform compares the affine bake.
func restampCompareTransform(t *testing.T, index int, got, want *Op) {
	t.Helper()

	if got.XformSet != want.XformSet || got.Transform() != want.Transform() {
		t.Fatalf("op %d: xform set=%v %+v, want set=%v %+v",
			index, got.XformSet, got.Transform(), want.XformSet, want.Transform())
	}
}

// restampCompareProjection compares the retained projective payload.
func restampCompareProjection(t *testing.T, index int, got, want *Op) {
	t.Helper()

	gotProjective, gotHasProjective := got.ProjectiveTransformValue()
	wantProjective, wantHasProjective := want.ProjectiveTransformValue()

	if gotHasProjective != wantHasProjective || gotProjective != wantProjective {
		t.Fatalf("op %d: projective set=%v %+v, want set=%v %+v",
			index, gotHasProjective, gotProjective, wantHasProjective, wantProjective)
	}
}

// restampComparePayload compares the geometry finalize produces from a
// projection: the display path and the text glyph basis.
func restampComparePayload(t *testing.T, index int, got, want *Op) {
	t.Helper()

	gotPath, gotHasPath := got.PathValue()
	wantPath, wantHasPath := want.PathValue()

	if gotHasPath != wantHasPath || len(gotPath.Points) != len(wantPath.Points) {
		t.Fatalf("op %d: path set=%v points=%d, want set=%v points=%d",
			index, gotHasPath, len(gotPath.Points), wantHasPath, len(wantPath.Points))
	}

	for pointIdx := range wantPath.Points {
		if gotPath.Points[pointIdx] != wantPath.Points[pointIdx] {
			t.Fatalf("op %d: path point %d = %+v, want %+v",
				index, pointIdx, gotPath.Points[pointIdx], wantPath.Points[pointIdx])
		}
	}

	gotBasis, gotHasBasis := got.GlyphBasisValue()
	wantBasis, wantHasBasis := want.GlyphBasisValue()

	if gotHasBasis != wantHasBasis || gotBasis != wantBasis {
		t.Fatalf("op %d: glyph basis set=%v %+v, want set=%v %+v",
			index, gotHasBasis, gotBasis, wantHasBasis, wantBasis)
	}
}

// restampFinalized finalizes every operation the way DisplayListOptions does.
func restampFinalized(t *testing.T, res *Result) *Result {
	t.Helper()

	if err := FinalizeDisplayOps(res.Ops); err != nil {
		t.Fatalf("FinalizeDisplayOps: %v", err)
	}

	return res
}

// restampCardSource is one card on a stage. The card has no transform so an
// override replaces a clean element.
const restampCardSource = `<html><body style="margin:0">` +
	`<div id="stage" style="width:300px;height:200px">` +
	`<div id="card" style="width:100px;height:60px;background-color:#ff0000">sample card</div>` +
	`</div></body></html>`

// restampCardTransformed is the same page with the declaration the override
// has to reproduce.
func restampCardTransformed(decl string) string {
	return `<html><body style="margin:0">` +
		`<div id="stage" style="width:300px;height:200px">` +
		`<div id="card" style="width:100px;height:60px;background-color:#ff0000;` + decl + `">sample card</div>` +
		`</div></body></html>`
}

// TestRestampRotateYMatchesFreshLayout pins the 3D path against a fresh layout.
func TestRestampRotateYMatchesFreshLayout(t *testing.T) {
	t.Parallel()

	restamped := layoutHTML(t, restampCardSource)

	patch, err := RestampTransforms(t.Context(), restamped, []RestampOverride{{
		ElementID:    "card",
		Transform:    "rotateY(45deg)",
		HasTransform: true,
	}}, Options{Width: testViewport, Height: 800, Background: true})
	if err != nil {
		t.Fatalf("RestampTransforms: %v", err)
	}

	if !patch.Changed {
		t.Fatal("patch reports no change, want the override applied")
	}

	fresh := restampFinalized(t, layoutHTML(t, restampCardTransformed("transform:rotateY(45deg)")))
	restampFinalized(t, restamped)

	restampCompare(t, restamped, fresh)

	if !restampHasNarrowedXform(restamped) {
		t.Fatal("restamped ops carry no rotateY bake, want a narrowed op matrix")
	}
}

// restampHasNarrowedXform reports whether some operation carries a horizontal
// scale that is not 1, the fingerprint of an orthographic rotateY bake.
func restampHasNarrowedXform(res *Result) bool {
	for idx := range res.Ops {
		op := &res.Ops[idx]

		if op.XformSet && !near(op.Transform().A, 1) {
			return true
		}
	}

	return false
}

// restampHasPath reports whether some operation was finalized into projected
// geometry.
func restampHasPath(res *Result) bool {
	for idx := range res.Ops {
		if _, ok := res.Ops[idx].PathValue(); ok {
			return true
		}
	}

	return false
}

// TestRestampMovesBetweenFrames pins that a second restamp starts from the
// geometry the first one restored, not from the flattened path it produced.
func TestRestampMovesBetweenFrames(t *testing.T) {
	t.Parallel()

	restamped := layoutHTML(t, restampCardSource)

	frames := []struct {
		transform string
		decl      string
	}{
		{"rotateY(20deg) translateZ(30px)", "transform:rotateY(20deg) translateZ(30px)"},
		{"rotateY(55deg) translateZ(10px)", "transform:rotateY(55deg) translateZ(10px)"},
		{"rotate3d(1, 2, 3, 40deg)", "transform:rotate3d(1, 2, 3, 40deg)"},
	}

	for _, frame := range frames {
		_, err := RestampTransforms(t.Context(), restamped, []RestampOverride{{
			ElementID:    "card",
			Transform:    frame.transform,
			HasTransform: true,
		}}, Options{Width: testViewport, Height: 800, Background: true})
		if err != nil {
			t.Fatalf("RestampTransforms(%s): %v", frame.transform, err)
		}

		fresh := restampFinalized(t, layoutHTML(t, restampCardTransformed(frame.decl)))
		restampFinalized(t, restamped)

		restampCompare(t, restamped, fresh)
	}
}

// restampPerspectiveSource is a card under a perspective stage.
const restampPerspectiveSource = `<html><body style="margin:0">` +
	`<div id="stage" style="width:300px;height:200px;perspective:500px;perspective-origin:30% 20%">` +
	`<div id="card" style="width:100px;height:60px;background-color:#ff0000;transform:rotateY(45deg)">card</div>` +
	`</div></body></html>`

// TestRestampPerspectiveMatchesFreshLayout pins the camera distance override.
func TestRestampPerspectiveMatchesFreshLayout(t *testing.T) {
	t.Parallel()

	restamped := layoutHTML(t, restampPerspectiveSource)

	patch, err := RestampTransforms(t.Context(), restamped, []RestampOverride{{
		ElementID:      "stage",
		Perspective:    pxToPt(1000),
		HasPerspective: true,
	}}, Options{Width: testViewport, Height: 800, Background: true})
	if err != nil {
		t.Fatalf("RestampTransforms: %v", err)
	}

	if !patch.Changed {
		t.Fatal("patch reports no change, want the perspective applied")
	}

	freshSource := `<html><body style="margin:0">` +
		`<div id="stage" style="width:300px;height:200px;perspective:1000px;perspective-origin:30% 20%">` +
		`<div id="card" style="width:100px;height:60px;background-color:#ff0000;transform:rotateY(45deg)">card</div>` +
		`</div></body></html>`

	fresh := restampFinalized(t, layoutHTML(t, freshSource))
	restampFinalized(t, restamped)

	restampCompare(t, restamped, fresh)

	if !restampHasPath(restamped) {
		t.Fatal("restamped ops carry no projected path, want the 1000px camera bake")
	}
}

// TestRestampTransformFunctionsMatchFreshLayout runs every 3D function the
// override accepts through the same equality gate. Each function is compared
// under a perspective camera so it actually projects.
func TestRestampTransformFunctionsMatchFreshLayout(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		decl   string
		camera bool
	}{
		{"rotateX", "rotateX(35deg)", true},
		{"rotateY", "rotateY(35deg)", true},
		{"rotateZ", "rotateZ(35deg)", true},
		{"rotate3d", "rotate3d(1, 2, 3, 35deg)", true},
		{"translate3d", "translate3d(10px, 20px, 30px)", true},
		{"translateZ", "translateZ(30px)", true},
		{"scale3d", "scale3d(0.6, 0.8, 0.5)", true},
		{"scaleZ", "scaleZ(0.5)", true},
		{"matrix3d", "matrix3d(1,0,0,0, 0,1,0,0, 0,0,1,0, 0,0,0,1)", true},
		{"perspectiveFunction", "perspective(500px) rotateY(30deg)", false},
		{"none", "", false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			restamped := layoutHTML(t, restampFunctionBase(testCase.camera))

			_, err := RestampTransforms(t.Context(), restamped, []RestampOverride{{
				ElementID:    "card",
				Transform:    testCase.decl,
				HasTransform: true,
			}}, Options{Width: testViewport, Height: 800, Background: true})
			if err != nil {
				t.Fatalf("RestampTransforms: %v", err)
			}

			fresh := restampFinalized(t, layoutHTML(t, restampFunctionPage(testCase.camera, testCase.decl)))
			restampFinalized(t, restamped)

			restampCompare(t, restamped, fresh)
		})
	}
}

// restampFunctionBase is the page an override starts from: a card with no
// transform, under the camera the function needs to project.
func restampFunctionBase(camera bool) string {
	if !camera {
		return restampCardSource
	}

	return `<html><body style="margin:0;perspective:800px">` +
		`<div id="stage" style="width:300px;height:200px">` +
		`<div id="card" style="width:100px;height:60px;background-color:#ff0000">sample card</div>` +
		`</div></body></html>`
}

// restampFunctionPage is the fresh layout with the declaration in place.
func restampFunctionPage(camera bool, decl string) string {
	declaration := "transform:" + decl
	if decl == "" {
		declaration = "transform:none"
	}

	body := `<body style="margin:0">`
	if camera {
		body = `<body style="margin:0;perspective:800px">`
	}

	return `<html>` + body +
		`<div id="stage" style="width:300px;height:200px">` +
		`<div id="card" style="width:100px;height:60px;background-color:#ff0000;` + declaration + `">sample card</div>` +
		`</div></body></html>`
}

// TestRestampAncestorProjectionIsReapplied pins a subtree under a flat 3D
// ancestor: the ancestor's projection covers every operation below it and the
// restamp has to wrap the new matrices in the same projection.
func TestRestampAncestorProjectionIsReapplied(t *testing.T) {
	t.Parallel()

	const nested = `<html><body style="margin:0;perspective:800px">` +
		`<div id="outer" style="width:300px;height:200px;transform:rotateY(30deg)">` +
		`<div id="card" style="width:100px;height:60px;background-color:#ff0000;transform:rotateY(20deg)">card</div>` +
		`</div></body></html>`

	restamped := layoutHTML(t, nested)

	_, err := RestampTransforms(t.Context(), restamped, []RestampOverride{{
		ElementID:    "card",
		Transform:    "rotateY(45deg)",
		HasTransform: true,
	}}, Options{Width: testViewport, Height: 800, Background: true})
	if err != nil {
		t.Fatalf("RestampTransforms: %v", err)
	}

	freshSource := `<html><body style="margin:0;perspective:800px">` +
		`<div id="outer" style="width:300px;height:200px;transform:rotateY(30deg)">` +
		`<div id="card" style="width:100px;height:60px;background-color:#ff0000;transform:rotateY(45deg)">card</div>` +
		`</div></body></html>`

	fresh := restampFinalized(t, layoutHTML(t, freshSource))
	restampFinalized(t, restamped)

	restampCompare(t, restamped, fresh)
}

// TestRestampFlatBoundaryCompositionIsReapplied pins a subtree under a
// preserve-3d chain whose flat boundary composes over the whole subtree after
// its children stamp.
func TestRestampFlatBoundaryCompositionIsReapplied(t *testing.T) {
	t.Parallel()

	const nested = `<html><body style="margin:0">` +
		`<div id="outer" style="width:300px;height:200px;transform-style:preserve-3d;transform:rotateX(15deg)">` +
		`<div id="mid" style="width:260px;height:180px;transform-style:preserve-3d;transform:rotateY(20deg)">` +
		`<div id="flat" style="width:220px;height:150px;transform:rotateZ(5deg)">` +
		`<div id="card" style="width:100px;height:60px;background-color:#ff0000;transform:translateZ(10px)">card</div>` +
		`</div></div></div></body></html>`

	restamped := layoutHTML(t, nested)

	_, err := RestampTransforms(t.Context(), restamped, []RestampOverride{{
		ElementID:    "card",
		Transform:    "translateZ(40px) rotateX(25deg)",
		HasTransform: true,
	}}, Options{Width: testViewport, Height: 800, Background: true})
	if err != nil {
		t.Fatalf("RestampTransforms: %v", err)
	}

	freshSource := `<html><body style="margin:0">` +
		`<div id="outer" style="width:300px;height:200px;transform-style:preserve-3d;transform:rotateX(15deg)">` +
		`<div id="mid" style="width:260px;height:180px;transform-style:preserve-3d;transform:rotateY(20deg)">` +
		`<div id="flat" style="width:220px;height:150px;transform:rotateZ(5deg)">` +
		`<div id="card" style="width:100px;height:60px;background-color:#ff0000;` +
		`transform:translateZ(40px) rotateX(25deg)">card</div>` +
		`</div></div></div></body></html>`

	fresh := restampFinalized(t, layoutHTML(t, freshSource))
	restampFinalized(t, restamped)

	restampCompare(t, restamped, fresh)
}

// TestRestampPerspectiveNoneRestoresOuterCamera pins the seed: removing an
// inner perspective falls back to the outer one instead of orthographic.
func TestRestampPerspectiveNoneRestoresOuterCamera(t *testing.T) {
	t.Parallel()

	const inner = `<html><body style="margin:0">` +
		`<div id="outer" style="width:300px;height:200px;perspective:400px">` +
		`<div id="stage" style="width:250px;height:180px;perspective:500px">` +
		`<div id="card" style="width:100px;height:60px;background-color:#ff0000;transform:rotateY(45deg)">card</div>` +
		`</div></div></body></html>`

	restamped := layoutHTML(t, inner)

	_, err := RestampTransforms(t.Context(), restamped, []RestampOverride{{
		ElementID:      "stage",
		Perspective:    0,
		HasPerspective: true,
	}}, Options{Width: testViewport, Height: 800, Background: true})
	if err != nil {
		t.Fatalf("RestampTransforms: %v", err)
	}

	freshSource := `<html><body style="margin:0">` +
		`<div id="outer" style="width:300px;height:200px;perspective:400px">` +
		`<div id="stage" style="width:250px;height:180px;perspective:none">` +
		`<div id="card" style="width:100px;height:60px;background-color:#ff0000;transform:rotateY(45deg)">card</div>` +
		`</div></div></body></html>`

	fresh := restampFinalized(t, layoutHTML(t, freshSource))
	restampFinalized(t, restamped)

	restampCompare(t, restamped, fresh)
}

// TestRestampNestedPerspectiveAndTransformOverrides pins the outer and inner
// override interaction against a fresh layout with both declarations.
func TestRestampNestedPerspectiveAndTransformOverrides(t *testing.T) {
	t.Parallel()

	restamped := layoutHTML(t, restampPerspectiveSource)

	_, err := RestampTransforms(t.Context(), restamped, []RestampOverride{
		{
			ElementID:      "stage",
			Perspective:    pxToPt(700),
			HasPerspective: true,
		},
		{
			ElementID:    "card",
			Transform:    "rotateY(30deg)",
			HasTransform: true,
		},
	}, Options{Width: testViewport, Height: 800, Background: true})
	if err != nil {
		t.Fatalf("RestampTransforms: %v", err)
	}

	freshSource := `<html><body style="margin:0">` +
		`<div id="stage" style="width:300px;height:200px;perspective:700px;perspective-origin:30% 20%">` +
		`<div id="card" style="width:100px;height:60px;background-color:#ff0000;transform:rotateY(30deg)">card</div>` +
		`</div></body></html>`

	fresh := restampFinalized(t, layoutHTML(t, freshSource))
	restampFinalized(t, restamped)

	restampCompare(t, restamped, fresh)
}

// TestRestampPreserve3DChainMatchesFreshLayout pins preserve-3d flattening
// through a subtree root that sits inside another 3D chain.
func TestRestampPreserve3DChainMatchesFreshLayout(t *testing.T) {
	t.Parallel()

	const chain = `<html><body style="margin:0">` +
		`<div id="outer" style="width:300px;height:200px;transform-style:preserve-3d;transform:rotateX(20deg)">` +
		`<div id="mid" style="width:200px;height:150px;transform-style:preserve-3d;transform:rotateY(30deg)">` +
		`<div id="card" style="width:120px;height:80px;background-color:#00ff00;transform:translateZ(20px)">card</div>` +
		`</div></div></body></html>`

	restamped := layoutHTML(t, chain)

	_, err := RestampTransforms(t.Context(), restamped, []RestampOverride{{
		ElementID:    "mid",
		Transform:    "rotateY(55deg)",
		HasTransform: true,
	}}, Options{Width: testViewport, Height: 800, Background: true})
	if err != nil {
		t.Fatalf("RestampTransforms: %v", err)
	}

	freshSource := `<html><body style="margin:0">` +
		`<div id="outer" style="width:300px;height:200px;transform-style:preserve-3d;transform:rotateX(20deg)">` +
		`<div id="mid" style="width:200px;height:150px;transform-style:preserve-3d;transform:rotateY(55deg)">` +
		`<div id="card" style="width:120px;height:80px;background-color:#00ff00;transform:translateZ(20px)">card</div>` +
		`</div></div></body></html>`

	fresh := restampFinalized(t, layoutHTML(t, freshSource))
	restampFinalized(t, restamped)

	restampCompare(t, restamped, fresh)
}

// TestRestampNoOverridesLeavesMatricesUntouched pins the empty pass.
func TestRestampNoOverridesLeavesMatricesUntouched(t *testing.T) {
	t.Parallel()

	restamped := layoutHTML(t, restampPerspectiveSource)
	before := restamped.Ops

	patch, err := RestampTransforms(t.Context(), restamped, nil, Options{Width: testViewport, Height: 800})
	if err != nil {
		t.Fatalf("RestampTransforms: %v", err)
	}

	if patch.Changed || len(patch.Windows) != 0 {
		t.Fatalf("patch changed=%v windows=%d, want an empty patch", patch.Changed, len(patch.Windows))
	}

	for idx := range before {
		restampCompareOp(t, idx, &restamped.Ops[idx], &before[idx])
	}
}

// TestRestampSameValuesLeavesMatricesUntouched pins that an override equal to
// the resolved style costs nothing.
func TestRestampSameValuesLeavesMatricesUntouched(t *testing.T) {
	t.Parallel()

	restamped := layoutHTML(t, restampPerspectiveSource)
	before := restamped.Ops

	patch, err := RestampTransforms(t.Context(), restamped, []RestampOverride{
		{
			ElementID:    "card",
			Transform:    "rotateY(45deg)",
			HasTransform: true,
		},
		{
			ElementID:      "stage",
			Perspective:    pxToPt(500),
			HasPerspective: true,
		},
	}, Options{Width: testViewport, Height: 800})
	if err != nil {
		t.Fatalf("RestampTransforms: %v", err)
	}

	if patch.Changed || len(patch.Windows) != 0 {
		t.Fatalf("patch changed=%v windows=%d, want an empty patch", patch.Changed, len(patch.Windows))
	}

	for idx := range before {
		restampCompareOp(t, idx, &restamped.Ops[idx], &before[idx])
	}
}

// TestRestampRejectsUnknownElement pins the error for a missing id.
func TestRestampRejectsUnknownElement(t *testing.T) {
	t.Parallel()

	restamped := layoutHTML(t, restampCardSource)

	_, err := RestampTransforms(t.Context(), restamped, []RestampOverride{{
		ElementID:    "missing",
		Transform:    "rotateY(10deg)",
		HasTransform: true,
	}}, Options{Width: testViewport, Height: 800})

	if !errors.Is(err, ErrNoElement) {
		t.Fatalf("error = %v, want %v", err, ErrNoElement)
	}
}

// TestRestampRejectsTableCell pins the table-cell guard: the static path does
// not stamp a transform declared on a cell, so a restamp must not invent one.
func TestRestampRejectsTableCell(t *testing.T) {
	t.Parallel()

	restamped := layoutHTML(t, `<html><body style="margin:0">`+
		`<table style="width:200px"><tr>`+
		`<td id="cell" style="width:100px;height:40px;background-color:#ff0000">cell</td>`+
		`</tr></table></body></html>`)

	_, err := RestampTransforms(t.Context(), restamped, []RestampOverride{{
		ElementID:    "cell",
		Transform:    "rotateY(30deg)",
		HasTransform: true,
	}}, Options{Width: testViewport, Height: 800})

	if !errors.Is(err, ErrNotStampable) {
		t.Fatalf("error = %v, want %v", err, ErrNotStampable)
	}
}

// TestRestampRejectsMalformedTransform pins that a bad list is reported and
// nothing is stamped.
func TestRestampRejectsMalformedTransform(t *testing.T) {
	t.Parallel()

	restamped := layoutHTML(t, restampCardSource)
	before := restamped.Ops

	_, err := RestampTransforms(t.Context(), restamped, []RestampOverride{{
		ElementID:    "card",
		Transform:    "rotateY(45deg",
		HasTransform: true,
	}}, Options{Width: testViewport, Height: 800})

	if !errors.Is(err, ErrInvalidTransform) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidTransform)
	}

	for idx := range before {
		restampCompareOp(t, idx, &restamped.Ops[idx], &before[idx])
	}
}
