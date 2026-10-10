package layout

import (
	"testing"
)

// perspective-origin build tests: a declared perspective-origin moves the
// vanishing point returned by perspectiveCenter, resolved against the
// perspective box border box. Percentages resolve against border-box width
// (X) and height (Y), absolute lengths are pt offsets from the border-box
// top-left, and the parser already folds keywords to positions (left/top
// 0%, center 50%, right/bottom 100%). Unset keeps the default center.
// The stamp-time projection that consumes this vanishing point has not
// landed yet (no production caller of perspectiveCenter), so the projection
// shift is asserted at the flatten3D contract level and layout-level
// divergence stays pending-reconcile. Reference: Chrome 143.0.7499.40.
// Every test below asserts used values (vanishing-point points and baked
// projected geometry), never stored strings.

// originBuildBox lays out a 200x100px box with the given container
// declarations and returns its border box.
func originBuildBox(t *testing.T, decl string) *box {
	t.Helper()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="width:200px;height:100px;`+decl+`">sample text here</div></body></html>`)

	return boxByID(t, res, "a")
}

// TestOriginBuildDefaultIsCenter pins the unset case: no perspective-origin
// declaration keeps the border-box center.
func TestOriginBuildDefaultIsCenter(t *testing.T) {
	t.Parallel()

	target := originBuildBox(t, "")

	if target.style.PerspectiveOriginSet {
		t.Errorf("PerspectiveOriginSet = true, want false (no declaration)")
	}

	vx, vy := perspectiveCenter(target)

	if !near(vx, target.x+target.w/2) || !near(vy, target.y+target.height/2) {
		t.Errorf("perspectiveCenter = (%g, %g), want center (%g, %g)",
			vx, vy, target.x+target.w/2, target.y+target.height/2)
	}

	if vx, vy := perspectiveCenter(nil); vx != 0 || vy != 0 {
		t.Errorf("perspectiveCenter(nil) = (%g, %g), want (0, 0)", vx, vy)
	}
}

// TestOriginBuildPercentages resolves X% of width and Y% of height from the
// border-box top-left.
func TestOriginBuildPercentages(t *testing.T) {
	t.Parallel()

	target := originBuildBox(t, "perspective-origin:25% 75%")

	if !target.style.PerspectiveOriginSet {
		t.Errorf("PerspectiveOriginSet = false, want true (declared)")
	}

	vx, vy := perspectiveCenter(target)

	if !near(vx, target.x+0.25*target.w) || !near(vy, target.y+0.75*target.height) {
		t.Errorf("perspectiveCenter = (%g, %g), want (%g, %g)",
			vx, vy, target.x+0.25*target.w, target.y+0.75*target.height)
	}
}

// TestOriginBuildKeywords folds the origin keywords to their positions:
// left 0%, center 50%, right 100% on X; top 0%, center 50%, bottom 100%
// on Y. A single horizontal keyword centers Y; a single vertical keyword
// centers X.
func TestOriginBuildKeywords(t *testing.T) {
	t.Parallel()

	left := originBuildBox(t, "perspective-origin:left")
	leftX, leftY := perspectiveCenter(left)

	if !near(leftX, left.x) || !near(leftY, left.y+left.height/2) {
		t.Errorf("left center = (%g, %g), want (%g, %g)", leftX, leftY, left.x, left.y+left.height/2)
	}

	right := originBuildBox(t, "perspective-origin:right bottom")
	rightX, rightY := perspectiveCenter(right)

	if !near(rightX, right.x+right.w) || !near(rightY, right.y+right.height) {
		t.Errorf("right bottom = (%g, %g), want (%g, %g)", rightX, rightY, right.x+right.w, right.y+right.height)
	}

	top := originBuildBox(t, "perspective-origin:top")
	tx, ty := perspectiveCenter(top)

	if !near(tx, top.x+top.w/2) || !near(ty, top.y) {
		t.Errorf("top center = (%g, %g), want (%g, %g)", tx, ty, top.x+top.w/2, top.y)
	}

	if !near(rightX-leftX, left.w) {
		t.Errorf("right-left vx gap = %g, want one box width %g", rightX-leftX, left.w)
	}
}

// TestOriginBuildLengths resolves absolute lengths as pt offsets from the
// border-box top-left.
func TestOriginBuildLengths(t *testing.T) {
	t.Parallel()

	target := originBuildBox(t, "perspective-origin:10px 20px")
	vx, vy := perspectiveCenter(target)

	if !near(vx, target.x+pxToPt(10)) || !near(vy, target.y+pxToPt(20)) {
		t.Errorf("perspectiveCenter = (%g, %g), want (%g, %g)",
			vx, vy, target.x+pxToPt(10), target.y+pxToPt(20))
	}
}

// TestOriginBuildVanishingPointShiftsProjection asserts the contract the
// stamp-time projection will consume: the same 3D state flattened around
// the left-origin vanishing point bakes different geometry than around the
// right-origin one. rotateY(45deg) gives the x-axis samples depth, so the
// perspective divide pulls the baked x scale toward each vanishing point
// differently. (The baked translation E is the projection of the z = 0
// local origin, which the divide leaves fixed at any vanishing point, so
// the shift shows in the baked x scale A instead.)
func TestOriginBuildVanishingPointShiftsProjection(t *testing.T) {
	t.Parallel()

	left := originBuildBox(t, "perspective-origin:left")
	right := originBuildBox(t, "perspective-origin:right")

	leftX, leftY := perspectiveCenter(left)
	rightX, rightY := perspectiveCenter(right)

	const dist = 375.0 // 500px camera distance in pt

	fromLeft, _ := rotateY3D(45).flatten3D(leftX, leftY, dist)
	fromRight, _ := rotateY3D(45).flatten3D(rightX, rightY, dist)

	if near(fromLeft.A, fromRight.A) {
		t.Errorf("baked A = %g vs %g, want left and right origins to shift foreshortened output",
			fromLeft.A, fromRight.A)
	}
}

// TestOriginBuildShiftsProjectedOutput is the layout-level close: one
// rotateY child under a perspective ancestor bakes different Xform matrices
// for left vs right perspective-origin values, proving the vanishing point
// moves end to end. Reference: Chrome 143.0.7499.40.
func TestOriginBuildShiftsProjectedOutput(t *testing.T) {
	t.Parallel()

	doc := func(origin string) string {
		return `<html><body style="margin:0">` +
			`<div id="outer" style="width:200px;height:100px;perspective:500px;perspective-origin:` + origin + `">` +
			`<div id="inner" style="width:40px;height:20px;transform:rotateY(45deg)">x</div>` +
			`</div></body></html>`
	}

	left := layoutHTML(t, doc("left center"))
	right := layoutHTML(t, doc("right center"))

	var leftSeen, rightSeen bool

	var leftE, rightE float64

	for _, op := range left.Ops {
		if op.XformSet {
			leftSeen = true
			leftE = op.Xform.E
		}
	}

	for _, op := range right.Ops {
		if op.XformSet {
			rightSeen = true
			rightE = op.Xform.E
		}
	}

	if !leftSeen || !rightSeen {
		t.Fatalf("Xform stamped = left %v right %v, want both true", leftSeen, rightSeen)
	}

	if near(leftE, rightE) {
		t.Errorf("projected E = %g for both origins, want left/right vanishing points to diverge", leftE)
	}
}
