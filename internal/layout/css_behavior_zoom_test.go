package layout

import (
	"os"
	"testing"
)

// TestBehaviorZoomDoublesBorderBox is the CSS zoom property on one element.
// zoom: 2 multiplies that element's used border box. A 100px by 40px box
// lays out 200px by 80px. The plain box stays 100px by 40px. Options.Zoom
// is left at the default, so this is not the document scale.
func TestBehaviorZoomDoublesBorderBox(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../testdata/css/v012/zoom/page.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	res := layoutHTML(t, string(raw))
	zoomed := boxByID(t, res, "zoomed")
	plain := boxByID(t, res, "plain")

	if !near(zoomed.w, pxToPt(200)) || !near(zoomed.height, pxToPt(80)) {
		t.Errorf("#zoomed used border box = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want 200px x 80px",
			zoomed.w, zoomed.height, zoomed.w/ptPerCSSPx, zoomed.height/ptPerCSSPx)
	}

	if !near(plain.w, pxToPt(100)) || !near(plain.height, pxToPt(40)) {
		t.Errorf("#plain used border box = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want 100px x 40px",
			plain.w, plain.height, plain.w/ptPerCSSPx, plain.height/ptPerCSSPx)
	}
}
