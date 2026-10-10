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
