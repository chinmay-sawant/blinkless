//nolint:all // targeted shape-inside / shape-padding storage tests
package layout

import (
	"context"
	"testing"
)

// shapeInsideStoreCtx is the minimal cascade context for direct applier calls.
func shapeInsideStoreCtx() *styleContext {
	return &styleContext{ctx: context.Background(), viewportW: 800}
}

// shapeInsideApply runs the group applier for one declaration and requires
// the group to claim the property. A rejected value still returns true: the
// group owns the property even when the parser leaves the earlier stored
// value in place.
func shapeInsideApply(t *testing.T, sty *ResolvedStyle, prop, value string) {
	t.Helper()

	if !applyShapeInsideProps(sty, prop, value, 16, shapeInsideStoreCtx(), nil, false) {
		t.Fatalf("applyShapeInsideProps(%q, %q) = false, want true", prop, value)
	}
}

// TestShapeInsideStoreInitials pins the CSS initial values: shape-inside is
// auto (no contour), content-box reference, no outside-shape flag, no
// padding, and the percent slot reads -1 for a length.
func TestShapeInsideStoreInitials(t *testing.T) {
	t.Parallel()

	sty := initialStyle()

	if sty.ShapeInside != shapeInsideAuto {
		t.Errorf("initial ShapeInside = %q, want %q", sty.ShapeInside, shapeInsideAuto)
	}

	if sty.ShapeInsideBox != "" || sty.ShapeInsideOutside {
		t.Errorf("initial box/outside = %q/%v, want empty/false", sty.ShapeInsideBox, sty.ShapeInsideOutside)
	}

	if sty.ShapePadding != 0 || sty.ShapePaddingPercent != -1 {
		t.Errorf("initial padding = %v/%v, want 0/-1", sty.ShapePadding, sty.ShapePaddingPercent)
	}
}

// TestShapeInsideStoreAcceptsSubset stores the accepted subset: auto,
// outside-shape, circle()/ellipse()/inset() with an optional shape-box
// keyword in either order, canonicalized like shape-outside.
func TestShapeInsideStoreAcceptsSubset(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		value       string
		wantShape   string
		wantBox     string
		wantOutside bool
	}{
		{value: "auto", wantShape: shapeInsideAuto},
		{value: "outside-shape", wantShape: shapeInsideAuto, wantOutside: true},
		{value: "circle()", wantShape: "circle()"},
		{value: "CIRCLE(10px at 20% 30%)", wantShape: "circle(10px at 20% 30%)"},
		{value: "ellipse(2px 3px at center)", wantShape: "ellipse(2px 3px at center)"},
		{value: "inset(1px 2px 3px 4px)", wantShape: "inset(1px 2px 3px 4px)"},
		{value: "inset(1px) padding-box", wantShape: "inset(1px)", wantBox: "padding-box"},
		{value: "circle() margin-box", wantShape: "circle()", wantBox: "margin-box"},
		{value: "border-box ellipse()", wantShape: "ellipse()", wantBox: "border-box"},
	}

	for _, testCase := range testCases {
		sty := initialStyle()
		shapeInsideApply(t, &sty, "shape-inside", testCase.value)

		if sty.ShapeInside != testCase.wantShape || sty.ShapeInsideBox != testCase.wantBox ||
			sty.ShapeInsideOutside != testCase.wantOutside {
			t.Errorf("shape-inside:%s stored %q/%q/%v, want %q/%q/%v",
				testCase.value, sty.ShapeInside, sty.ShapeInsideBox, sty.ShapeInsideOutside,
				testCase.wantShape, testCase.wantBox, testCase.wantOutside)
		}
	}
}

// TestShapeInsideStoreRejectedTokensKeepEarlierValue: polygon(), path(),
// url() images, display, none, a bare shape-box, and junk all leave the
// earlier circle() and its shape-box untouched.
func TestShapeInsideStoreRejectedTokensKeepEarlierValue(t *testing.T) {
	t.Parallel()

	sty := initialStyle()
	shapeInsideApply(t, &sty, "shape-inside", "circle(10px at 20% 30%) padding-box")

	rejected := []string{
		"polygon(0 0, 10 0, 10 10)",
		"path('M 0 0 L 10 10 Z')",
		"url(pic.png)",
		"display",
		"none",
		"content-box",
		"bogus",
	}

	for _, bad := range rejected {
		shapeInsideApply(t, &sty, "shape-inside", bad)

		if sty.ShapeInside != "circle(10px at 20% 30%)" || sty.ShapeInsideBox != "padding-box" ||
			sty.ShapeInsideOutside {
			t.Errorf("shape-inside:%s replaced the earlier value with %q/%q/%v",
				bad, sty.ShapeInside, sty.ShapeInsideBox, sty.ShapeInsideOutside)
		}
	}
}

// TestShapeInsideStoreOutsideShapeFlagAndAutoReset: outside-shape stores the
// copy flag and clears the box keyword; auto clears everything back to the
// initial state.
func TestShapeInsideStoreOutsideShapeFlagAndAutoReset(t *testing.T) {
	t.Parallel()

	sty := initialStyle()
	shapeInsideApply(t, &sty, "shape-inside", "circle() border-box")
	shapeInsideApply(t, &sty, "shape-inside", "outside-shape")

	if sty.ShapeInside != shapeInsideAuto || sty.ShapeInsideBox != "" || !sty.ShapeInsideOutside {
		t.Fatalf("outside-shape stored %q/%q/%v, want auto/empty/true",
			sty.ShapeInside, sty.ShapeInsideBox, sty.ShapeInsideOutside)
	}

	shapeInsideApply(t, &sty, "shape-inside", "auto")

	if sty.ShapeInside != shapeInsideAuto || sty.ShapeInsideBox != "" || sty.ShapeInsideOutside {
		t.Errorf("auto stored %q/%q/%v, want auto/empty/false",
			sty.ShapeInside, sty.ShapeInsideBox, sty.ShapeInsideOutside)
	}
}

// TestShapePaddingStoreLengthPercentAndReject: a length stores in points
// with the percent slot at -1, a percentage stores in the percent slot, and
// a negative or junk value never replaces the earlier padding.
func TestShapePaddingStoreLengthPercentAndReject(t *testing.T) {
	t.Parallel()

	sty := initialStyle()
	shapeInsideApply(t, &sty, "shape-padding", "10px")

	if !near(sty.ShapePadding, pxToPt(10)) || sty.ShapePaddingPercent != -1 {
		t.Errorf("shape-padding:10px stored %v/%v, want %v/-1",
			sty.ShapePadding, sty.ShapePaddingPercent, pxToPt(10))
	}

	shapeInsideApply(t, &sty, "shape-padding", "5%")

	if sty.ShapePadding != 0 || sty.ShapePaddingPercent != 5 {
		t.Errorf("shape-padding:5%% stored %v/%v, want 0/5",
			sty.ShapePadding, sty.ShapePaddingPercent)
	}

	for _, bad := range []string{"-2px", "bogus", ""} {
		shapeInsideApply(t, &sty, "shape-padding", bad)

		if sty.ShapePadding != 0 || sty.ShapePaddingPercent != 5 {
			t.Errorf("shape-padding:%q replaced the earlier percentage with %v/%v",
				bad, sty.ShapePadding, sty.ShapePaddingPercent)
		}
	}

	shapeInsideApply(t, &sty, "shape-padding", "0")

	if sty.ShapePadding != 0 || sty.ShapePaddingPercent != -1 {
		t.Errorf("shape-padding:0 stored %v/%v, want 0/-1",
			sty.ShapePadding, sty.ShapePaddingPercent)
	}
}

// TestShapeInsideStoreGroupOwnership: the group claims shape-inside and
// shape-padding only; shape-outside, shape-margin, and other properties stay
// with their own groups.
func TestShapeInsideStoreGroupOwnership(t *testing.T) {
	t.Parallel()

	sty := initialStyle()

	for _, prop := range []string{"shape-outside", "shape-margin", "float", "margin-left"} {
		if applyShapeInsideProps(&sty, prop, "circle()", 16, shapeInsideStoreCtx(), nil, false) {
			t.Errorf("applyShapeInsideProps(%q) = true, want false", prop)
		}
	}
}

// TestShapeInsideStoreThroughCascadeDispatch proves the styleGroups
// registration: applyRestProps routes both properties into the inside group.
func TestShapeInsideStoreThroughCascadeDispatch(t *testing.T) {
	t.Parallel()

	sty := initialStyle()
	applyRestProps(&sty, map[string]string{
		"shape-inside":  "inset(2px) padding-box",
		"shape-padding": "4px",
	}, shapeInsideStoreCtx(), nil)

	if sty.ShapeInside != "inset(2px)" || sty.ShapeInsideBox != "padding-box" {
		t.Errorf("cascade dispatch stored %q/%q, want inset(2px)/padding-box",
			sty.ShapeInside, sty.ShapeInsideBox)
	}

	if !near(sty.ShapePadding, pxToPt(4)) || sty.ShapePaddingPercent != -1 {
		t.Errorf("cascade dispatch stored padding %v/%v, want %v/-1",
			sty.ShapePadding, sty.ShapePaddingPercent, pxToPt(4))
	}
}
