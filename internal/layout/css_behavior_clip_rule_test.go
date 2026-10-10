package layout

import (
	"testing"
)

// Wave G clip-rule tests. Reference browser for every case: Chrome
// 143.0.7499.40. Shared helpers layoutHTMLWithImages, tinyPNG, decodeNRGBA,
// opsOfKind, and behaviorSvgAssertSameOps come from layout_test.go and
// css_behavior_svg_paint_test.go and are never redeclared here.

// behaviorClipStarPolygon is a self-intersecting pentagram (star drawn in one
// stroke, not convex order) over a 100x100 reference box. Its center
// (50%, 50%) sits deep inside the inner pentagon, which the path winds
// twice: nonzero keeps it painted, evenodd cuts it out.
const behaviorClipStarPolygon = "polygon(50% 10%, 73.5% 82.4%, 12% 37.6%, 88% 37.6%, 26.5% 82.4%)"

// TestBehaviorClipRuleInlineFillRuleMasksStar proves the clip model
// has a working fill concept: polygonContains (clip_path.go) honors the
// inline polygon() fill rule when masking image ops. The default (nonzero)
// keeps the twice-wound star center opaque; inline evenodd zeroes it.
// Reference: Chrome 143.0.7499.40 renders the same difference.
func TestBehaviorClipRuleInlineFillRuleMasksStar(t *testing.T) {
	t.Parallel()

	nonzero := layoutHTMLWithImages(t,
		`<html><body><img src="x.png" style="clip-path:`+behaviorClipStarPolygon+`"></body></html>`,
		tinyPNG(40, 40), "x.png")
	evenodd := layoutHTMLWithImages(t,
		`<html><body><img src="x.png" style="clip-path:polygon(evenodd, 50% 10%, 73.5% 82.4%,`+
			`12% 37.6%, 88% 37.6%, 26.5% 82.4%)"></body></html>`,
		tinyPNG(40, 40), "x.png")

	nonzeroImgs := opsOfKind(nonzero, OpImage)
	evenoddImgs := opsOfKind(evenodd, OpImage)

	if len(nonzeroImgs) != 1 || len(evenoddImgs) != 1 {
		t.Fatalf("img ops = %d vs %d, want 1 each", len(nonzeroImgs), len(evenoddImgs))
	}

	if got := decodeNRGBA(t, nonzeroImgs[0].Image).NRGBAAt(20, 20); got.A != 255 {
		t.Errorf("nonzero center alpha = %d, want 255 (twice-wound center stays)", got.A)
	}

	if got := decodeNRGBA(t, evenoddImgs[0].Image).NRGBAAt(20, 20); got.A != 0 {
		t.Errorf("evenodd center alpha = %d, want 0 (twice-wound center cut out)", got.A)
	}
}

// TestBehaviorClipRulePropertyNoPaintEffect pins the current gap: the
// standalone clip-rule property never reaches the mask. It is forwarded by
// applySVGPresentationProps (style_paint_props.go) to applyLeftoversProps
// (style_leftovers.go), which has no clip-rule case and returns false, and
// there is no ClipRule field on ResolvedStyle (style.go), so all three
// variants below paint byte-identical output. Per MDN and SVG 1.1, clip-rule
// only applies to graphics elements inside a <clipPath> element, which this
// engine does not model (inline SVG is opaque raster; layout_svg.go bakes
// only fill/stroke presentation attrs), so there is no spec-valid consumer
// to hook without new style storage plus a bake or mask-site change.
// Wiring seam, in order: add ResolvedStyle.ClipRule next to ClipPath in
// style.go, add the case in applyLeftoversProps in style_leftovers.go, add
// cascade and intern entries in style_cascade.go and style_intern_gen.go,
// then thread the effective rule at the mask sites in layout_images.go,
// background_image.go, and inline_image.go (or bake it in layout_svg.go).
// When that lands, this test must flip: evenodd must cut the center out.
func TestBehaviorClipRulePropertyNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTMLWithImages(t,
		`<html><body><img src="x.png" style="clip-path:`+behaviorClipStarPolygon+`"></body></html>`,
		tinyPNG(40, 40), "x.png")
	evenodd := layoutHTMLWithImages(t,
		`<html><body><img src="x.png" style="clip-path:`+behaviorClipStarPolygon+`;clip-rule:evenodd"></body></html>`,
		tinyPNG(40, 40), "x.png")
	nonzero := layoutHTMLWithImages(t,
		`<html><body><img src="x.png" style="clip-path:`+behaviorClipStarPolygon+`;clip-rule:nonzero"></body></html>`,
		tinyPNG(40, 40), "x.png")

	behaviorSvgAssertSameOps(t, plain, evenodd)
	behaviorSvgAssertSameOps(t, plain, nonzero)

	for name, res := range map[string]*Result{"plain": plain, "evenodd": evenodd, "nonzero": nonzero} {
		imgs := opsOfKind(res, OpImage)
		if len(imgs) != 1 {
			t.Fatalf("%s img ops = %d, want 1", name, len(imgs))
		}

		if got := decodeNRGBA(t, imgs[0].Image).NRGBAAt(20, 20); got.A != 255 {
			t.Errorf("%s center alpha = %d, want 255 (clip-rule dropped, nonzero default)", name, got.A)
		}
	}
}
