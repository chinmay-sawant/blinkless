package layout

import "testing"

// TestPlacedOverflowClipRecordsAncestorBox checks that an element inside an
// overflow:hidden ancestor records that ancestor's padding box. Hosts
// painting the element through a transform intersect the paint with it,
// because geometry-baked clipping moves with the matrix.
func TestPlacedOverflowClipRecordsAncestorBox(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body>
<div id="outer"><div id="inner">x</div></div>
<div id="free">y</div>
</body></html>`, sheet(t, `
html, body { margin: 0; padding: 0; }
div { margin: 0; padding: 0; }
#outer { width: 200px; height: 100px; overflow: hidden; background: #c33; }
#inner { width: 300px; height: 40px; background: #33c; }
#free { width: 50px; height: 50px; background: #3c3; }
`))

	placed := PlacedElements(res)
	outer := placedByElementID(t, placed, "outer")
	inner := placedByElementID(t, placed, "inner")
	free := placedByElementID(t, placed, "free")

	if outer.HasOverflowClip {
		t.Error("outer HasOverflowClip = true, want false (own overflow does not clip self)")
	}

	if !inner.HasOverflowClip {
		t.Fatal("inner HasOverflowClip = false, want true")
	}

	clip := inner.OverflowClip
	if clip[2] != outer.W || clip[3] != outer.H {
		t.Errorf("inner clip size = %.2fx%.2f, want outer %.2fx%.2f", clip[2], clip[3], outer.W, outer.H)
	}

	if clip[0] != outer.X || clip[1] != outer.Y {
		t.Errorf("inner clip origin = %.2f,%.2f, want outer %.2f,%.2f", clip[0], clip[1], outer.X, outer.Y)
	}

	if free.HasOverflowClip {
		t.Error("free HasOverflowClip = true, want false")
	}
}
