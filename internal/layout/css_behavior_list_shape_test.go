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

// TestBehaviorShapeInsideNoLayoutEffect pins the no-op cases on an
// auto-height box: shape-inside:auto (the initial value, full content
// width), rejected tokens (polygon(), path(), url(), and display: storage
// stays at auto, the same rule shape-outside follows), and circle().
// shape_inside.go builds an inside contour only from a definite content box
// height; during line layout an auto-height box has no final height yet, so
// the contour is nil and the lines keep the full content width. A
// definite-height box does narrow (see
// TestBehaviorShapeInsideCircleTapersLines). Reference: Chrome
// 143.0.7499.40 also ignores shape-inside (unsupported, no BCD support).
func TestBehaviorShapeInsideNoLayoutEffect(t *testing.T) {
	t.Parallel()

	doc := func(extra string) *Result {
		return layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
			`<div id="box" style="margin:0;width:200px;font-size:12pt;`+extra+`">`+
			strings.Repeat("wrap ", 10)+`end</div></body></html>`)
	}

	plain := doc("")

	for _, extra := range []string{
		"shape-inside:auto;",
		"shape-inside:circle();",
		"shape-inside:polygon(0 0, 10 0, 10 10);",
		"shape-inside:path('M 0 0 L 10 10 Z');",
		"shape-inside:url(pic.png);",
		"shape-inside:display;",
	} {
		behaviorTextRemAssertSameOps(t, plain, doc(extra))
	}
}
