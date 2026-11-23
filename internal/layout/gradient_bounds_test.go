package layout

import (
	"fmt"
	"math"
	"testing"
)

func TestRenderGradientRejectsNonFinite(t *testing.T) { //nolint:paralleltest // global gradient cache
	gradCacheMu.Lock()
	clear(gradCache)

	gradCacheBytes = 0

	gradCacheMu.Unlock()

	const raw = "linear-gradient(red, blue)"

	good := [3]float64{0, 0, 0}

	cases := []struct {
		name string
		w, h float64
		cur  [3]float64
	}{
		{"nan width", math.NaN(), 10, good},
		{"inf height", 10, math.Inf(1), good},
		{"nan color", 10, 10, [3]float64{math.NaN(), 0, 0}},
		{"inf color", 10, 10, [3]float64{0, math.Inf(-1), 0}},
	}

	for _, tc := range cases {
		if data, _, _, ok := renderGradientPNG(raw, tc.w, tc.h, tc.cur); ok || data != nil {
			t.Errorf("%s: rendered a gradient", tc.name)
		}
	}

	gradCacheMu.RLock()
	n := len(gradCache)
	gradCacheMu.RUnlock()

	if n != 0 {
		t.Fatalf("non-finite inputs inserted %d cache entries", n)
	}
}

func TestGradientCacheByteBounded(t *testing.T) { //nolint:paralleltest // global gradient cache
	old := gradCacheByteCap
	gradCacheByteCap = 64 << 10

	defer func() { gradCacheByteCap = old }()

	for i := range 40 {
		raw := fmt.Sprintf("linear-gradient(%ddeg, red, blue)", i)
		renderGradientPNG(raw, 64, 64, [3]float64{})
	}

	gradCacheMu.RLock()
	bytes, n := gradCacheBytes, len(gradCache)
	gradCacheMu.RUnlock()

	if bytes > gradCacheByteCap || n > maxGradCacheEntries || n == 0 {
		t.Fatalf("gradient cache holds %d bytes in %d entries (cap %d)", bytes, n, gradCacheByteCap)
	}
}
