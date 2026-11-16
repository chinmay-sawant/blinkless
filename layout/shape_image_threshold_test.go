package layout_test

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

// halfFaintPNG is 20x20: the left half is opaque, the right half has alpha 0.3.
func halfFaintPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, 20, 20))

	for y := range 20 {
		for x := range 20 {
			alpha := uint8(255)
			if x >= 10 {
				alpha = 77
			}

			img.SetNRGBA(x, y, color.NRGBA{R: 0, G: 128, B: 0, A: alpha})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

// firstTextX renders a left float with shape-outside: url(mask) and returns
// the x of the first text op that follows it.
func firstTextX(t *testing.T, threshold string) float64 {
	t.Helper()

	data := halfFaintPNG(t)
	page := fmt.Sprintf(`<body style="margin:0"><div style="float:left;width:80px;height:80px;`+
		`shape-outside:url('mask');%s"></div><p style="margin:0">Text beside the float</p></body>`, threshold)

	display, err := layout.DisplayListOptions(t.Context(), styledOf(t, page), layout.Options{
		Images: func(string) ([]byte, error) { return data, nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, op := range display.Ops {
		if op.Kind == layout.DisplayOpText {
			return op.X
		}
	}

	t.Fatal("no text op")

	return 0
}

// TestShapeImageThreshold: a faint right half counts as shape at threshold 0
// and drops out at 0.5, so the text beside the float moves left.
func TestShapeImageThreshold(t *testing.T) {
	t.Parallel()

	low := firstTextX(t, "shape-image-threshold:0")
	high := firstTextX(t, "shape-image-threshold:0.5")

	if math.Abs(low-60) > 0.5 {
		t.Errorf("threshold 0 text x = %.2f, want 60 (the float's right edge, 80px)", low)
	}

	if math.Abs(high-30) > 0.5 {
		t.Errorf("threshold 0.5 text x = %.2f, want 30 (the opaque half, 40px)", high)
	}
}
