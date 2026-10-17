package layout

import (
	"bytes"
	"image"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// fixture59 loaders and page geometry. The consumer lays this fixture out at
// A4 minus the default 10mm margins: 538.58pt x 785.2pt of content, which is
// also the page-1 canvas band.
const (
	fixture59Name       = "fixture-59-apex-digital-landing.html"
	fixture59ContentW   = 538.58
	fixture59ContentH   = 785.2
	fixture59ContentX   = 43.09 // 8% section padding, the hero content left edge
	fixture59ContentRgt = 495.5 // hero content right edge (W minus the padding)
)

// TestFixture59HeroCopyStaysOnFirstPage pins the hero section of fixture 59.
// Two engine defects used to push the whole hero copy to page 2 and hide it
// under the hero bitmap:
//
//  1. The grid's intrinsic pass resolved the hero image's width:100% against
//     the viewport, so the single 1fr track was floored at the border-box
//     width and the heading wrapped past the section's content box.
//  2. The image's object-fit:cover paint rect (538.58 x 807.87, centered)
//     spilled ~246pt above and below its 315pt content box and crossed the
//     page boundary. A consumer paginator snaps a crossing image op to the
//     next page and drags every op below its Y with it: the hero heading,
//     paragraph, and button all landed on page 2 behind the picture.
//
// The test asserts the resolved drawing list: the heading wraps at the
// content box like the browser, every hero copy op sits inside page 1, and
// the cover paint op is cropped to the content box instead of spilling.
// Reference: Chrome 143.0.7499.40 print layout of the same fixture.
func TestFixture59HeroCopyStaysOnFirstPage(t *testing.T) { //nolint:cyclop,funlen // fixture assertions
	t.Parallel()

	res := layoutFixture59(t)

	// The heading breaks into three lines at the content-box width. Before
	// the fix it broke into two lines at the border-box width (513.4pt wide,
	// past the 495.5pt content right edge).
	for _, line := range []string{"Design and Build", "High-Performance Digital", "Products"} {
		paintOp, ok := findTextOp(res, line)
		if !ok {
			t.Fatalf("hero heading line %q missing from the drawing list", line)
		}

		if !near(paintOp.Size, 30) {
			t.Errorf("heading %q size = %.1fpt, want 30pt", line, paintOp.Size)
		}

		// color: var(--dark) = #0f172a, opaque.
		if !near(paintOp.R, 0x0f/255.0) || !near(paintOp.G, 0x17/255.0) || !near(paintOp.B, 0x2a/255.0) {
			t.Errorf("heading %q fill = %.3f,%.3f,%.3f, want #0f172a", line, paintOp.R, paintOp.G, paintOp.B)
		}

		if math.IsNaN(paintOp.Alpha) || paintOp.Alpha < 0 || paintOp.Alpha > 1 {
			t.Errorf("heading %q alpha = %.2f, want a paintable 0..1", line, paintOp.Alpha)
		}
	}

	// Every hero copy op stays inside the hero content box and page 1.
	heroCopy := []string{"Design and Build", "High-Performance Digital", "Products",
		"We combine design systems, responsive layouts, and performance", "View Plans"}

	for _, want := range heroCopy {
		paintOp, ok := findTextOp(res, want)
		if !ok {
			t.Fatalf("hero copy %q missing from the drawing list", want)
		}

		if paintOp.X < fixture59ContentX-0.01 || paintOp.X+paintOp.W > fixture59ContentRgt+0.01 {
			t.Errorf("hero copy %q x-range = %.2f..%.2f, want inside %.2f..%.2f",
				want, paintOp.X, paintOp.X+paintOp.W, fixture59ContentX, fixture59ContentRgt)
		}

		if paintOp.Y > fixture59ContentH {
			t.Errorf("hero copy %q baseline y = %.2f, want on page 1 (below %.1f)", want, paintOp.Y, fixture59ContentH)
		}
	}

	// Exactly one hero bitmap, cropped to its content box and page 1. The
	// source is 1024x1536; cover into the 452.41 x 315 box crops the payload
	// and the op must not cross the 785.2pt page boundary.
	img, ok := findImageOp(res, "Workspace display with code and mockups")
	if !ok {
		t.Fatal("hero image op missing from the drawing list")
	}

	if !near(img.W, 452.41) || !near(img.H, 315) {
		t.Errorf("hero image paint = %.2fx%.2fpt, want 452.41x315 (content box)", img.W, img.H)
	}

	if img.X < fixture59ContentX-0.01 || img.X+img.W > fixture59ContentRgt+0.01 {
		t.Errorf("hero image x-range = %.2f..%.2f, want inside the content box", img.X, img.X+img.W)
	}

	if img.Y < 0 || img.Y+img.H > fixture59ContentH+0.01 {
		t.Errorf("hero image y-range = %.2f..%.2f crosses page 1 (0..%.1f)",
			img.Y, img.Y+img.H, fixture59ContentH)
	}

	if img.ImgH >= 1536 {
		t.Errorf("hero image payload height = %dpx, want the cover crop (< 1536px source)", img.ImgH)
	}

	// The cropped payload must keep the drawn rect's aspect ratio, so the
	// pixels map 1:1 onto the visible region instead of stretching.
	cfg, _, err := image.DecodeConfig(bytes.NewReader(img.Image))
	if err != nil {
		t.Fatalf("decode hero image payload: %v", err)
	}

	if want := img.W / img.H; math.Abs(float64(cfg.Width)/float64(cfg.Height)-want) > 0.01 {
		t.Errorf("hero image payload aspect = %.3f (%dx%d), want the paint rect aspect %.3f",
			float64(cfg.Width)/float64(cfg.Height), cfg.Width, cfg.Height, want)
	}
}

// TestGridPercentWidthImageDoesNotStretchTrack pins the intrinsic sizing
// rule behind fixture 59's hero: a width:100% image inside a 1fr track is
// cyclic during intrinsic sizing (CSS Grid L1 §11.5), so the track must size
// from the image's intrinsic width and the fr share of the content box, not
// from the percentage resolved against the container. Before the fix the
// track was floored at the 400pt border-box width and the image overflowed
// its column by the 20pt padding on each side.
func TestGridPercentWidthImageDoesNotStretchTrack(t *testing.T) {
	t.Parallel()

	src := `<html style="margin:0"><body style="margin:0">` +
		`<section id="g" style="display:grid;grid-template-columns:1fr;width:400pt;` +
		`padding:20pt;box-sizing:border-box">` +
		`<img id="i" src="x.png" style="width:100%">` +
		`</section></body></html>`

	res := layoutHTMLWithImages(t, src, tinyPNG(40, 20), "")

	imgBox := findBox(t, res, "img")
	if !near(imgBox.w, 360) {
		t.Errorf("grid image width = %.2fpt, want 360pt (400 minus two 20pt paddings)", imgBox.w)
	}

	if !near(imgBox.x, 20) {
		t.Errorf("grid image x = %.2fpt, want 20pt (content left edge)", imgBox.x)
	}
}

// layoutFixture59 lays the fixture out at the consumer's A4 content box with
// its own <style> sheets and asset images resolved from testdata/golden.
func layoutFixture59(t *testing.T) *Result {
	t.Helper()

	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", fixture59Name))
	if err != nil {
		t.Skip(err)
	}

	root, err := html.Parse(string(b))
	if err != nil {
		t.Fatal(err)
	}

	var sheets []*css.Stylesheet

	root.Walk(func(node *html.Node) {
		if node.Name != "style" {
			return
		}

		var sbox strings.Builder

		for _, c := range node.Children {
			if c.Type == html.TextNode {
				sbox.WriteString(c.Text)
			}
		}

		if s, err := css.Parse(sbox.String()); err == nil {
			sheets = append(sheets, s)
		}
	})

	resolver := func(src string) ([]byte, error) {
		return os.ReadFile(filepath.Join("..", "..", "testdata", "golden", filepath.FromSlash(src)))
	}

	res, err := Layout(root, Options{
		Width: fixture59ContentW, Height: fixture59ContentH,
		Sheets: sheets, Background: true, Images: resolver,
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	return res
}

// findTextOp returns the first text op with exactly the given content.
func findTextOp(res *Result, text string) (Op, bool) {
	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text == text {
			return op, true
		}
	}

	return Op{}, false
}

// findImageOp returns the first image op carrying the given alt text.
func findImageOp(res *Result, alt string) (Op, bool) {
	for _, op := range res.Ops {
		if op.Kind == OpImage && op.ImageAlt() == alt {
			return op, true
		}
	}

	return Op{}, false
}
