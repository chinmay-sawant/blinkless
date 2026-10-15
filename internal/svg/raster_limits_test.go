package svg

import (
	"bytes"
	"context"
	"errors"
	"image/png"
	"math"
	"testing"
	"time"
)

//nolint:paralleltest // this test holds the package canvas mutex
func TestRasterizeContextStopsWaitingForMutex(t *testing.T) {
	canvasMu.Lock()
	t.Cleanup(canvasMu.Unlock)

	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Millisecond)
	t.Cleanup(cancel)

	started := time.Now()
	src := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"/>`)
	png, width, height, err := RasterizeContext(ctx, src, 16)

	if err == nil || png != nil || width != 0 || height != 0 {
		t.Fatal("rasterize waited for a held mutex")
	}

	if time.Since(started) > time.Second {
		t.Fatal("mutex wait ignored the caller deadline")
	}
}

func TestEmulateTagsStopsAtTagCap(t *testing.T) {
	t.Parallel()

	var body bytes.Buffer

	body.WriteString(`<svg fill-rule="evenodd">`)

	for range maxEmulateTags + 2 {
		body.WriteString(`<rect fill-rule="evenodd"/>`)
	}

	body.WriteString(`</svg>`)

	if emulateTags(body.Bytes()) != nil {
		t.Fatal("tag cap still rewrote the document")
	}
}

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
