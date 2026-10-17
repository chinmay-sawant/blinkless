package layout_test

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	bcss "github.com/chinmay-sawant/blinkless/css"
	bhtml "github.com/chinmay-sawant/blinkless/html"
	blayout "github.com/chinmay-sawant/blinkless/layout"
)

// TestFixture47CertificateFitsOnePrintedPage pins fixture-47 as a one-page
// print composition whose out-of-flow background image covers the page.
//
// The defect this guards: .certificate-bg is a translated and scaled
// absolute image whose authored box is a full page tall (.certificate and
// .certificate-bg are both 1123px; the transform is
// translate(-111px,-132px) scale(1.23)). With the viewport at the fixture's
// 794x1123 CSS-pixel A4, the un-fitted op list drew the image box from
// y=86.25pt to 928.5pt, 86.25pt past the one-page 842.25pt canvas. A
// paginating consumer that snaps leading-edge-crossing ink (gopkg convert
// snapCrossingTextOps) moved that image and every following flow op to
// page 2, so page 1 printed blank with the header text clipped on page 2.
//
// The print width fit (DisplayList) re-lays the document out at
// width/contentWidth, the native PDF path's smart shrink, which pulls the
// background back inside one page while its transformed bounds still cover
// the whole canvas.
func TestFixture47CertificateFitsOnePrintedPage(t *testing.T) {
	t.Parallel()

	fixture := filepath.Join("..", "..", "testdata", "golden", "fixture-47-certificate.html")

	src, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	doc, err := bhtml.Parse(src)
	if err != nil {
		t.Fatalf("html.Parse: %v", err)
	}

	// The print path's viewport: A4 at 96 dpi is 793.7x1122.5 CSS pixels,
	// which the PDF bridge rounds to 794x1123 before css.Apply.
	styled, err := bcss.Apply(t.Context(), doc, bcss.Options{
		WidthPx:  794,
		HeightPx: 1123,
		Media:    "print",
	})
	if err != nil {
		t.Fatalf("css.Apply: %v", err)
	}

	assetDir := filepath.Dir(fixture)

	display, err := blayout.DisplayListOptions(t.Context(), styled, blayout.Options{
		Images: func(imageSrc string) ([]byte, error) {
			imageSrc = strings.TrimPrefix(imageSrc, "file://")

			return os.ReadFile(filepath.Join(assetDir, filepath.Base(imageSrc)))
		},
	})
	if err != nil {
		t.Fatalf("DisplayListOptions: %v", err)
	}

	if display.Width != 794 || display.Height != 1123 {
		t.Fatalf("canvas = %dx%d, want 794x1123 (one A4 page at 96 dpi)",
			display.Width, display.Height)
	}

	pagePt := float64(display.Height) * display.PointsPerPixel
	widthPt := float64(display.Width) * display.PointsPerPixel

	image := fixture47ScanOps(t, display, pagePt)

	fixture47AssertBackgroundCoversPage(t, image, widthPt, pagePt)
}

// fixture47ScanOps fails every visible op that draws past the one-page
// canvas and returns the image op, which is the certificate background.
func fixture47ScanOps(t *testing.T, display *blayout.Display, pagePt float64) *blayout.DisplayOp {
	t.Helper()

	var image *blayout.DisplayOp

	for idx := range display.Ops {
		paintOp := &display.Ops[idx]
		if paintOp.Kind == blayout.DisplayOpNoop || paintOp.Kind == blayout.DisplayOpUnknown {
			continue
		}

		if paintOp.Y+paintOp.H > pagePt+0.01 {
			t.Errorf("op[%d] kind=%v bottom=%.2fpt exceeds the one-page %.2fpt canvas",
				idx, paintOp.Kind, paintOp.Y+paintOp.H, pagePt)
		}

		if paintOp.Kind == blayout.DisplayOpImage {
			image = paintOp
		}
	}

	return image
}

// fixture47AssertBackgroundCoversPage pins that the background image op kept
// its transform and covers the full page after the print width fit.
func fixture47AssertBackgroundCoversPage(t *testing.T, image *blayout.DisplayOp, widthPt, pagePt float64) {
	t.Helper()

	if image == nil {
		t.Fatal("no image op for the certificate background")
	}

	if !image.XformSet {
		t.Fatal("background image op lost its translate/scale transform")
	}

	// The transformed box must cover the page: the fixture's whole premise
	// is a full-bleed background, so a clipped corner fails the layout even
	// though the untransformed box may sit inside the canvas.
	minX, minY, maxX, maxY := fixture47TransformedBounds(*image)
	if minX > 0.01 || minY > 0.01 || maxX < widthPt-0.01 || maxY < pagePt-0.01 {
		t.Errorf("background transformed bounds [%.2f,%.2f..%.2f,%.2f] do not cover the page [0,0..%.2f,%.2f]",
			minX, minY, maxX, maxY, widthPt, pagePt)
	}
}

// fixture47TransformedBounds maps the op rectangle through its baked display
// matrix and returns the axis-aligned canvas bounds.
func fixture47TransformedBounds(paintOp blayout.DisplayOp) (float64, float64, float64, float64) {
	matrix := paintOp.Transform()

	corners := [4][2]float64{
		{paintOp.X, paintOp.Y},
		{paintOp.X + paintOp.W, paintOp.Y},
		{paintOp.X + paintOp.W, paintOp.Y + paintOp.H},
		{paintOp.X, paintOp.Y + paintOp.H},
	}

	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)

	for _, corner := range corners {
		x := matrix.A*corner[0] + matrix.C*corner[1] + matrix.E
		y := matrix.B*corner[0] + matrix.D*corner[1] + matrix.F

		minX = math.Min(minX, x)
		minY = math.Min(minY, y)
		maxX = math.Max(maxX, x)
		maxY = math.Max(maxY, y)
	}

	return minX, minY, maxX, maxY
}
