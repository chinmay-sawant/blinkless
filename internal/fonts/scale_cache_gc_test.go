package fonts

import (
	"runtime"
	"testing"
	"weak"
)

// TestScaledFaceCacheDoesNotPinFont proves an unloaded Font is collectable
// even after SharedScaledFace cached scaled faces for it: the cache lives on
// the Font, so no process-global map holds the SFNT bytes alive.
func TestScaledFaceCacheDoesNotPinFont(t *testing.T) { //nolint:paralleltest // GC timing
	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	data := append([]byte(nil), faces.Regular.Bytes()...)

	collect := func() weak.Pointer[Font] {
		font, err := ParseTTF(data)
		if err != nil {
			t.Fatal(err)
		}

		for size := 8.0; size < 40; size++ {
			SharedScaledFace(font, size).Advance('A')
		}

		return weak.Make(font)
	}

	ref := collect()

	for range 10 {
		runtime.GC()

		if ref.Value() == nil {
			return
		}
	}

	t.Fatal("Font stayed reachable after its last user dropped it; the scaled-face cache pins it")
}

// TestScaledFaceCacheBoundedPerFont keeps a hostile size sweep from growing
// one Font's cache without bound while still measuring correctly.
func TestScaledFaceCacheBoundedPerFont(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	font, err := ParseTTF(append([]byte(nil), faces.Regular.Bytes()...))
	if err != nil {
		t.Fatal(err)
	}

	for i := range 2*maxScaledFaces + 10 {
		got := SharedScaledFace(font, 10+float64(i)/7)
		if got == nil || got.Advance('A') <= 0 {
			t.Fatalf("size %d: unusable scaled face", i)
		}
	}

	font.scaledMu.RLock()
	n := len(font.scaledFaces)
	font.scaledMu.RUnlock()

	if n > maxScaledFaces {
		t.Fatalf("cached %d scaled faces, cap %d", n, maxScaledFaces)
	}
}
