package svg

import (
	"testing"
	"time"
)

// FuzzRasterizeSVG feeds arbitrary bytes to Rasterize. Any input must return
// either a PNG with positive dimensions or an error with nothing else set,
// and must not panic or run past a generous wall-clock bound.
func FuzzRasterizeSVG(f *testing.F) {
	seeds := []string{
		`<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"/>`,
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10" fill="red"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20"><path d="M1 1L19 1L10 19z" stroke="#000"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20"><circle cx="10" cy="10" r="8" fill="url(#g)"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><g transform="rotate(30)"><text>hi</text></g></svg>`,
		`<svg width="-1" height="1e999"></svg>`,
		`not svg at all`,
		``,
	}
	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 16<<10 {
			return
		}

		started := time.Now()
		png, width, height, err := Rasterize(data, 64)

		if err != nil && (png != nil || width != 0 || height != 0) {
			t.Fatalf("error %v returned alongside png=%d w=%d h=%d", err, len(png), width, height)
		}

		if err == nil && (len(png) == 0 || width <= 0 || height <= 0) {
			t.Fatalf("nil error with empty result: png=%d w=%d h=%d", len(png), width, height)
		}

		if elapsed := time.Since(started); elapsed > 10*time.Second {
			t.Fatalf("Rasterize took %v on %d bytes", elapsed, len(data))
		}
	})
}
