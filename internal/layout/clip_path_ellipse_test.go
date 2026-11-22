package layout

import "testing"

// TestResolveEllipseRadiiIndependentAxes verifies that ellipse() resolves rx
// and ry independently when mixing explicit lengths and side keywords.
func TestResolveEllipseRadiiIndependentAxes(t *testing.T) {
	t.Parallel()

	const (
		w = 100.0
		h = 120.0
	)

	cases := []struct {
		expr   string
		wantRX float64
		wantRY float64
	}{
		{
			// cx = 30 (closest 30, farthest 70), cy = 40 (closest 40, farthest 80)
			// 20px converts to 15pt (20 * 0.75).
			expr:   "ellipse(20px closest-side at 30pt 40pt)",
			wantRX: pxToPt(20),
			wantRY: 40,
		},
		{
			expr:   "ellipse(closest-side farthest-side at 30pt 40pt)",
			wantRX: 30,
			wantRY: 80,
		},
		{
			expr:   "ellipse(farthest-side 25% at 30pt 40pt)",
			wantRX: 70,
			wantRY: 30,
		},
	}

	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			t.Parallel()

			shape, ok := parseClipPathShape(tc.expr, 12)
			if !ok {
				t.Fatalf("parseClipPathShape(%q) failed", tc.expr)
			}

			cx, cy := shape.centerX.resolve(w), shape.centerY.resolve(h)
			rx, ry := resolveEllipseRadii(shape.radiusX, shape.radiusY, cx, cy, w, h)

			if !near(rx, tc.wantRX) || !near(ry, tc.wantRY) {
				t.Fatalf("resolveEllipseRadii(%q) = (%.4f, %.4f), want (%.4f, %.4f)",
					tc.expr, rx, ry, tc.wantRX, tc.wantRY)
			}
		})
	}
}
