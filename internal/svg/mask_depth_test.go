package svg_test

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/svg"
)

// TestRasterizeMaskDepthCap proves a pathologically nested mask document is
// rejected instead of driving unbounded parser recursion.
func TestRasterizeMaskDepthCap(t *testing.T) {
	t.Parallel()

	var doc strings.Builder

	doc.WriteString(`<svg xmlns="http://www.w3.org/2000/svg"><mask id="m">`)

	for range 2000 {
		doc.WriteString("<g>")
	}

	for range 2000 {
		doc.WriteString("</g>")
	}

	doc.WriteString(`</mask></svg>`)

	if _, _, _, err := svg.RasterizeMask([]byte(doc.String()), "m", 8, 2, "alpha"); err == nil {
		t.Fatal("RasterizeMask accepted nesting past the depth cap")
	}
}
