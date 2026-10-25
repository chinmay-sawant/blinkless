package svg

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// gradientNRGBA builds one deterministic test raster of the given side
// length: red ramps with x and green with y, so every pixel differs. The
// channel math stays in 0..255 because both coordinates stay below size.
func gradientNRGBA(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	for pixelY := range size {
		for pixelX := range size {
			//nolint:gosec // pixel coordinates stay below size, so each channel stays in 0..255
			img.SetNRGBA(pixelX, pixelY, color.NRGBA{
				R: uint8(pixelX * 255 / (size - 1)),
				G: uint8(pixelY * 255 / (size - 1)),
				B: 128,
				A: 255,
			})
		}
	}

	return img
}

func TestDpmmScaleFactorCapsSmallIcons(t *testing.T) {
	t.Parallel()

	for _, sample := range []struct {
		name             string
		targetW, targetH int
		maxSide          int
		want             float64
	}{
		{"icon5", 5, 5, 512, 2.0},
		{"icon8", 8, 8, 512, 2.0},
		{"icon16", 16, 16, 1024, 2.0},
		{"icon31", 31, 31, 512, 2.0},
		{"boundary32", 32, 32, 1024, 4.0},
		{"rect40x20", 40, 20, 256, 4.0},
		{"large", 300, 300, 1024, 1.0},
		{"empty", 0, 0, 512, 1.0},
		{"capBelowTarget", 100, 100, 50, 1.0},
	} {
		t.Run(sample.name, func(t *testing.T) {
			t.Parallel()

			got := dpmmScaleFactor(sample.targetW, sample.targetH, sample.maxSide)

			if got != sample.want {
				t.Fatalf("dpmmScaleFactor(%d, %d, %d) = %v, want %v",
					sample.targetW, sample.targetH, sample.maxSide, got, sample.want)
			}
		})
	}
}

func TestEncodeRasterPNGKeepsPixels(t *testing.T) {
	t.Parallel()

	img := gradientNRGBA(64)

	var def, fast bytes.Buffer

	if err := png.Encode(&def, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}

	if err := encodeRasterPNG(&fast, img, 64, 64); err != nil {
		t.Fatalf("encodeRasterPNG: %v", err)
	}

	assertSamePixels(t, def.Bytes(), fast.Bytes())
}

func TestEncodeRasterPNGLargeKeepsPixels(t *testing.T) {
	t.Parallel()

	img := gradientNRGBA(200)

	var def, fast bytes.Buffer

	if err := png.Encode(&def, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}

	if err := encodeRasterPNG(&fast, img, 200, 200); err != nil {
		t.Fatalf("encodeRasterPNG: %v", err)
	}

	assertSamePixels(t, def.Bytes(), fast.Bytes())
}

func assertSamePixels(t *testing.T, first, second []byte) {
	t.Helper()

	firstImg, err := png.Decode(bytes.NewReader(first))
	if err != nil {
		t.Fatalf("decode first: %v", err)
	}

	secondImg, err := png.Decode(bytes.NewReader(second))
	if err != nil {
		t.Fatalf("decode second: %v", err)
	}

	if !firstImg.Bounds().Eq(secondImg.Bounds()) {
		t.Fatalf("bounds = %v vs %v", firstImg.Bounds(), secondImg.Bounds())
	}

	for pixelY := firstImg.Bounds().Min.Y; pixelY < firstImg.Bounds().Max.Y; pixelY++ {
		for pixelX := firstImg.Bounds().Min.X; pixelX < firstImg.Bounds().Max.X; pixelX++ {
			firstR, firstG, firstB, firstA := firstImg.At(pixelX, pixelY).RGBA()
			secondR, secondG, secondB, secondA := secondImg.At(pixelX, pixelY).RGBA()

			if firstR != secondR || firstG != secondG || firstB != secondB || firstA != secondA {
				t.Fatalf("pixel (%d, %d) differs: %04x %04x %04x %04x vs %04x %04x %04x %04x",
					pixelX, pixelY, firstR, firstG, firstB, firstA, secondR, secondG, secondB, secondA)
			}
		}
	}
}

func TestRasterMemoServesIdenticalInputs(t *testing.T) {
	t.Parallel()

	src := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16"><!--` +
		t.Name() + `--><rect width="16" height="16" fill="#0645ad"/></svg>`)

	first, firstWidth, firstHeight, err := Rasterize(src, 1024)
	if err != nil {
		t.Fatalf("first Rasterize: %v", err)
	}

	second, secondWidth, secondHeight, err := Rasterize(src, 1024)
	if err != nil {
		t.Fatalf("second Rasterize: %v", err)
	}

	if firstWidth != secondWidth || firstHeight != secondHeight {
		t.Fatalf("logical size changed: %dx%d vs %dx%d",
			firstWidth, firstHeight, secondWidth, secondHeight)
	}

	if !bytes.Equal(first, second) {
		t.Fatal("identical inputs encoded twice instead of reusing the memo")
	}

	key := rasterMemoKeyFor(src, normalizeRasterMaxSide(1024))

	if _, _, _, ok := sharedRasterMemo.get(key); !ok {
		t.Fatal("raster memo holds no entry for the rasterized input")
	}
}

func TestRasterMemoKeyIncludesSize(t *testing.T) {
	t.Parallel()

	src := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16"><!--` +
		t.Name() + `--><rect width="16" height="16" fill="#0645ad"/></svg>`)

	full, _, _, err := Rasterize(src, 1024)
	if err != nil {
		t.Fatalf("Rasterize at cap 1024: %v", err)
	}

	capped, _, _, err := Rasterize(src, 16)
	if err != nil {
		t.Fatalf("Rasterize at cap 16: %v", err)
	}

	fullImg, err := png.Decode(bytes.NewReader(full))
	if err != nil {
		t.Fatalf("decode full: %v", err)
	}

	cappedImg, err := png.Decode(bytes.NewReader(capped))
	if err != nil {
		t.Fatalf("decode capped: %v", err)
	}

	if fullImg.Bounds().Eq(cappedImg.Bounds()) {
		t.Fatalf("caps 1024 and 16 shared encoded size %v", fullImg.Bounds())
	}
}
