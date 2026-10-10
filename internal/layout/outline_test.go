package layout

import "testing"

// TestOutlinePaintsInflatedRect: the outline shorthand strokes four sides
// outside the border box, inflated by outline-offset plus half the width
// (internal/layout/outline.go:93-131); it never changes the layout box
// (internal/layout/outline.go:61-62). Browser reference: the Chrome 3px
// medium note at internal/layout/border_medium_default_test.go:193-206 and
// the fixture-62 rows 21-25.
func TestOutlinePaintsInflatedRect(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="o" style="width:60pt;height:20pt;outline:2pt solid red;outline-offset:3pt"></div>`+
		`</body></html>`)

	styled := boxByID(t, res, "o")
	if !near(styled.w, 60) || !near(styled.height, 20) {
		t.Fatalf("outline changed the layout box: %.2f x %.2f, want 60 x 20",
			styled.w, styled.height)
	}

	strokes := outlineOpsFrom(res.Ops)
	if len(strokes) != 4 {
		t.Fatalf("outline ops = %d, want 4 sides", len(strokes))
	}

	const inflate = 4 // outline-offset 3pt + outline-width 2pt / 2

	outX, outY := styled.x-inflate, styled.y-inflate
	outW, outH := styled.w+2*inflate, styled.height+2*inflate

	for _, stroke := range strokes {
		if stroke.Kind != OpLine || !near(stroke.Width, 2) {
			t.Fatalf("outline side = kind %v width %.2f, want OpLine width 2", stroke.Kind, stroke.Width)
		}

		if !outlineStrokeMatchesRect(stroke, outX, outY, outW, outH) {
			t.Fatalf("outline side at (%.2f, %.2f) %.2f x %.2f is outside the inflated rect (%.2f, %.2f) %.2f x %.2f",
				stroke.X, stroke.Y, stroke.W, stroke.H, outX, outY, outW, outH)
		}
	}
}

// TestOutlineStyleNoneEmitsNoOps: outline-style:none paints nothing even with
// a nonzero outline-width and a color set.
func TestOutlineStyleNoneEmitsNoOps(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="o" style="width:60pt;height:20pt;outline-style:none;`+
		`outline-width:4pt;outline-color:red"></div>`+
		`</body></html>`)

	for _, stroke := range res.Ops {
		if stroke.isOutline() {
			t.Errorf("outline-style:none emitted an outline op: %+v", stroke)
		}
	}
}

func outlineOpsFrom(ops []Op) []Op {
	var strokes []Op

	for _, stroke := range ops {
		if stroke.isOutline() {
			strokes = append(strokes, stroke)
		}
	}

	return strokes
}

func outlineStrokeMatchesRect(stroke Op, left, top, width, height float64) bool {
	if near(stroke.Y, top) && near(stroke.W, width) {
		return true
	}

	if near(stroke.Y, top+height) && near(stroke.W, width) {
		return true
	}

	if near(stroke.X, left) && near(stroke.H, height) {
		return true
	}

	return near(stroke.X, left+width) && near(stroke.H, height)
}
