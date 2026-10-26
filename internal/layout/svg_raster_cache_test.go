package layout

import "testing"

// TestSVGRasterCacheKeyIncludesSize pins the coordination contract with the
// rasterizer: the cache key carries the cap that sized the raster, so the
// same document requested under two caps holds two entries and can never
// serve the wrong pixels.
func TestSVGRasterCacheKeyIncludesSize(t *testing.T) {
	t.Parallel()

	doc := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16"></svg>`)

	small := svgInlineCacheKey(doc, 64)
	large := svgInlineCacheKey(doc, 1024)

	if small == large {
		t.Fatal("cache key ignores the raster cap")
	}

	if again := svgInlineCacheKey(doc, 64); again != small {
		t.Fatal("cache key is not deterministic")
	}
}

// TestSVGRasterCacheServesBothCaps pins the two-entry behavior behind the
// key contract: the small- and large-cap rasters coexist and each get
// returns its own pixels.
func TestSVGRasterCacheServesBothCaps(t *testing.T) {
	t.Parallel()

	doc := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16"></svg>`)

	small := svgInlineCacheKey(doc, 64)
	large := svgInlineCacheKey(doc, 1024)

	cache := &svgRasterCache{}

	cache.put(small, []byte{1}, 8, 8)
	cache.put(large, []byte{2}, 16, 16)

	requireCachedRaster(t, cache, small, 1, 8, 8)
	requireCachedRaster(t, cache, large, 2, 16, 16)
}

// requireCachedRaster fails the test unless key resolves to the expected
// payload and dimensions. Shared by the two-cap tests so each pin stays a
// flat sequence of puts and checks.
func requireCachedRaster(
	t *testing.T, cache *svgRasterCache, key string, wantByte byte, wantW, wantH int,
) {
	t.Helper()

	got, w, h, ok := cache.get(key)
	if !ok || len(got) != 1 || got[0] != wantByte || w != wantW || h != wantH {
		t.Fatalf("raster for cap = %v %dx%d ok=%v, want [%d] %dx%d",
			got, w, h, ok, wantByte, wantW, wantH)
	}
}
