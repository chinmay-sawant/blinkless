package layout

import "testing"

// Backface culling at stamp time (internal/layout/transform.go:
// stampExclusiveTransformOps). A box with HasTransform3D whose flattened
// facing is negative (back face toward the viewer) paints nothing under
// backface-visibility:hidden; visible paints. 2D mirrors never cull.
// Reference: Chrome 143.0.7499.40.

func backfaceRedFills(res *Result) []Op {
	var out []Op

	for _, op := range res.Ops {
		if op.Kind == OpFillRect && op.R > 0.9 && op.G < 0.1 && op.B < 0.1 {
			out = append(out, op)
		}
	}

	return out
}

func backfaceDoc(transform, vis string) string {
	return `<html><body style="margin:0">` +
		`<div id="a" style="width:100px;height:50px;background-color:#ff0000;transform:` + transform +
		`;backface-visibility:` + vis + `"></div></body></html>`
}

// TestBackfaceHiddenCullRotateY180 is rotateY(180deg) with hidden: the back
// face looks at the viewer, so exclusive fill geometry is zeroed.
// Reference: Chrome 143.0.7499.40 hides the back face.
func TestBackfaceHiddenCullRotateY180(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, backfaceDoc("rotateY(180deg)", "hidden"))
	fills := backfaceRedFills(res)

	if len(fills) == 0 {
		t.Fatalf("want red fill ops, got 0")
	}

	for _, op := range fills {
		if op.W != 0 || op.H != 0 {
			t.Errorf("hidden rotateY(180deg) fill = %.4fpt x %.4fpt, want 0 x 0 (culled)",
				op.W, op.H)
		}
	}
}

// TestBackfaceVisiblePaintsRotateY180 is rotateY(180deg) with visible: the
// turned face still paints its fill geometry.
// Reference: Chrome 143.0.7499.40 paints the back face when visible.
func TestBackfaceVisiblePaintsRotateY180(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, backfaceDoc("rotateY(180deg)", "visible"))
	fills := backfaceRedFills(res)

	if len(fills) == 0 {
		t.Fatalf("want red fill ops, got 0")
	}

	sawPaint := false

	for _, op := range fills {
		if op.W > 1 && op.H > 1 {
			sawPaint = true
		}
	}

	if !sawPaint {
		t.Errorf("visible rotateY(180deg) paints no live red fill, want at least one %.4fpt x %.4fpt fill",
			fills[0].W, fills[0].H)
	}
}

// TestBackfaceHiddenKeeps2DMirror is scaleX(-1) with hidden: a 2D mirror is
// not a 3D-rotated face, so the fill stays painted.
// Reference: Chrome 143.0.7499.40 keeps 2D scaleX(-1) content visible.
func TestBackfaceHiddenKeeps2DMirror(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, backfaceDoc("scaleX(-1)", "hidden"))
	fills := backfaceRedFills(res)

	if len(fills) == 0 {
		t.Fatalf("want red fill ops, got 0")
	}

	sawPaint := false

	for _, op := range fills {
		if op.W > 1 && op.H > 1 {
			sawPaint = true
		}
	}

	if !sawPaint {
		t.Errorf("hidden scaleX(-1) paints no live red fill, want at least one (2D never culls)")
	}
}
