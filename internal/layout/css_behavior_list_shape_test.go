package layout

import (
	"strings"
	"testing"
)

// List marker image and shape-inside behavior tests: every observable below
// is asserted through a USED value (an emitted op field or painted geometry),
// never through a stored style string. Reference browser for every case:
// Chrome 143.0.7499.40. Shared helpers layoutHTML, layoutHTMLWithImages,
// tinyPNG, opsOfKind, boxByID, near, pxToPt, and
// behaviorTextRemAssertSameOps come from layout_test.go, css_review_02_test.go,
// style_values.go, and css_behavior_text_remaining_test.go and are never
// redeclared here.

// TestBehaviorListStyleImageResolvedEmitsImageMarker is list-style-image:
// with an image resolver configured, a URL on the <ul> is inherited by the
// <li> (list-style-image is inherited per CSS Lists 3) and the resolved
// payload replaces the type marker: one OpImage, no OpBullet. The 20x10px
// fixture paints 15pt by 7.5pt (1 CSS pixel = 0.75pt) and the default
// outside marker hangs left of the li box. Reference: Chrome 143.0.7499.40
// paints the image instead of the disc.
func TestBehaviorListStyleImageResolvedEmitsImageMarker(t *testing.T) {
	t.Parallel()

	res := layoutHTMLWithImages(t, `<html><body style="margin:0">`+
		`<ul style="margin:0;padding-left:30pt;font-size:12pt;list-style-image:url(marker.png)">`+
		`<li id="li1">one</li></ul></body></html>`, tinyPNG(20, 10), "marker.png")

	if bullets := opsOfKind(res, OpBullet); len(bullets) != 0 {
		t.Fatalf("bullets = %d, want 0 (resolved image replaces the type marker)", len(bullets))
	}

	imgs := opsOfKind(res, OpImage)
	if len(imgs) != 1 {
		t.Fatalf("images = %d, want exactly one list-style-image marker", len(imgs))
	}

	if !near(imgs[0].W, pxToPt(20)) {
		t.Errorf("marker width = %.4fpt, want 15pt (20px)", imgs[0].W)
	}

	if !near(imgs[0].H, pxToPt(10)) {
		t.Errorf("marker height = %.4fpt, want 7.5pt (10px)", imgs[0].H)
	}

	if liBox := boxByID(t, res, "li1"); !(imgs[0].X < liBox.x) {
		t.Errorf("marker x %.4fpt should hang left of the li box x %.4fpt (outside)",
			imgs[0].X, liBox.x)
	}
}

// TestBehaviorListStyleShorthandImageEmitsImageMarker is the list-style
// shorthand with a url(): the shorthand carries the image while the type and
// position tokens still apply, so the marker is an image. Reference: Chrome
// 143.0.7499.40, list-style shorthand marker.
func TestBehaviorListStyleShorthandImageEmitsImageMarker(t *testing.T) {
	t.Parallel()

	res := layoutHTMLWithImages(t, `<html><body style="margin:0">`+
		`<ul style="margin:0;padding-left:30pt;font-size:12pt;list-style:url(marker.png) square inside">`+
		`<li>one</li></ul></body></html>`, tinyPNG(8, 8), "marker.png")

	if bullets := opsOfKind(res, OpBullet); len(bullets) != 0 {
		t.Fatalf("bullets = %d, want 0 (shorthand image replaces the square marker)", len(bullets))
	}

	imgs := opsOfKind(res, OpImage)
	if len(imgs) != 1 {
		t.Fatalf("images = %d, want exactly one shorthand image marker", len(imgs))
	}

	if !near(imgs[0].W, pxToPt(8)) || !near(imgs[0].H, pxToPt(8)) {
		t.Errorf("marker size = %.4fpt x %.4fpt, want 6pt x 6pt (8px)",
			imgs[0].W, imgs[0].H)
	}
}

// TestBehaviorShapeInsideNoLayoutEffect documents the current behavior of
// shape-inside: the declaration is dropped (applyShapeProps in
// style_shape_props.go has no shape-inside arm and ResolvedStyle carries no
// ShapeInside field), so the styled document paints identical geometry to
// the unstyled one. GAP: a shape-inside implementation needs interior
// exclusion layout, narrowing the container's own line boxes to the shape
// contour per line. The float-wrapping exclusion integration exists for the
// mirror feature (floatState.exclusion in float.go consults the
// shape-outside contour from shape_exclusion.go via lineBounds in
// inline.go:576), but there is no hook that narrows a block's own content
// boxes, which would touch inline.go (line-box layout), style.go (new
// stored field), and style_intern_gen.go (interning). Unobservable in the
// drawing list: reported as a no-op, never failing. Reference: Chrome
// 143.0.7499.40 also ignores shape-inside (unsupported, no BCD support).
func TestBehaviorShapeInsideNoLayoutEffect(t *testing.T) {
	t.Parallel()

	doc := func(extra string) *Result {
		return layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
			`<div id="box" style="margin:0;width:200px;font-size:12pt;`+extra+`">`+
			strings.Repeat("wrap ", 10)+`end</div></body></html>`)
	}

	plain := doc("")
	shaped := doc("shape-inside:circle();")

	behaviorTextRemAssertSameOps(t, plain, shaped)
}
