package svg

import (
	"bytes"
	"errors"
	"image/png"
	"math"
	"testing"
)

func TestCanvasDPMMBoundsBothAxes(t *testing.T) {
	t.Parallel()

	for _, dims := range [][4]float64{{10, 1000000, 100, 100}, {1000000, 10, 100, 200}} {
		dpmm := canvasDPMM(dims[0], dims[1], int(dims[2]), int(dims[3]))
		if dims[0]*dpmm > dims[2]+1e-6 || dims[1]*dpmm > dims[3]+1e-6 {
			t.Fatalf("canvasDPMM%v = %v exceeds target bounds", dims, dpmm)
		}
	}
}

func TestRasterizeMismatchedAspectRatioBounded(t *testing.T) {
	t.Parallel()

	src := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="1" height="100000" viewBox="0 0 100 100">` +
		`<rect width="100" height="100"/></svg>`)
	data, _, _, err := Rasterize(src, math.MaxInt)
	// Extreme canvas aspect ratios can round the short edge to zero. Either
	// a bounded raster or a structured error is acceptable.
	if errors.Is(err, errCanvasZeroPixel) {
		return
	}

	if err != nil {
		t.Fatalf("Rasterize returned unexpected error: %v", err)
	}

	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode raster dimensions: %v", err)
	}

	if config.Width > maxRasterSide || config.Height > maxRasterSide ||
		int64(config.Width)*int64(config.Height) > maxRasterSide*maxRasterSide {
		t.Fatalf("raster dimensions %dx%d exceed pixel budget", config.Width, config.Height)
	}
}
