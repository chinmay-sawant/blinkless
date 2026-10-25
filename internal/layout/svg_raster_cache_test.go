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

	cache := &svgRasterCache{}

	cache.put(small, []byte{1}, 8, 8)
	cache.put(large, []byte{2}, 16, 16)

	gotSmall, wSmall, hSmall, okSmall := cache.get(small)
	if !okSmall || len(gotSmall) != 1 || gotSmall[0] != 1 || wSmall != 8 || hSmall != 8 {
		t.Fatal("small-cap raster missing or replaced by the large-cap entry")
	}

	gotLarge, wLarge, hLarge, okLarge := cache.get(large)
	if !okLarge || len(gotLarge) != 1 || gotLarge[0] != 2 || wLarge != 16 || hLarge != 16 {
		t.Fatal("large-cap raster missing or replaced by the small-cap entry")
	}
}
