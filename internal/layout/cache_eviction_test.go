package layout

import (
	"fmt"
	"strings"
	"testing"
)

// TestGlobalCachesKeepAdmittingPastCapacity inserts well over twice each
// cache's capacity and checks that late entries are still cached (the caches
// used to freeze on their first entries) and that sizes stay bounded.
func TestGlobalCachesKeepAdmittingPastCapacity(t *testing.T) { //nolint:paralleltest // global caches
	resetAcceptVerdictForTest()
	resetCascadeCachesForTest()

	for i := range 2*maxAcceptVerdictEntries + 100 {
		supportedDeclaration("width", fmt.Sprintf("%dpx", i))
	}

	_, hitsBefore := acceptVerdictStatsForTest()
	last := fmt.Sprintf("%dpx", 2*maxAcceptVerdictEntries+99)
	supportedDeclaration("width", last)

	if _, hits := acceptVerdictStatsForTest(); hits != hitsBefore+1 {
		t.Fatal("latest accept verdict was not cached after the cache filled")
	}

	acceptVerdictMu.RLock()
	size := len(acceptVerdicts)
	acceptVerdictMu.RUnlock()

	if size > maxAcceptVerdictEntries {
		t.Fatalf("accept verdicts = %d, cap %d", size, maxAcceptVerdictEntries)
	}

	for i := range 2*maxSupportsMemoEntries + 100 {
		engineSupportsProperty("height", fmt.Sprintf("%dpx", i))
	}

	_, hitsBefore = supportsProbeStatsForTest()
	engineSupportsProperty("height", fmt.Sprintf("%dpx", 2*maxSupportsMemoEntries+99))

	if _, hits := supportsProbeStatsForTest(); hits != hitsBefore+1 {
		t.Fatal("latest @supports verdict was not cached after the cache filled")
	}

	for i := range 2*maxInlineDeclEntries + 100 {
		cachedInlineDecls(fmt.Sprintf("width:%dpx", i))
	}

	before := inlineDeclParseCountForTest()
	cachedInlineDecls(fmt.Sprintf("width:%dpx", 2*maxInlineDeclEntries+99))

	if inlineDeclParseCountForTest() != before {
		t.Fatal("latest inline declaration was parsed again; cache froze")
	}

	// Oversized style attributes are parsed but never retained.
	huge := "width:1px;" + strings.Repeat("/*x*/", maxInlineDeclKeyBytes)
	cachedInlineDecls(huge)

	inlineDeclsMu.RLock()
	_, kept := inlineDecls[huge]
	inlineDeclsMu.RUnlock()

	if kept {
		t.Fatal("oversized inline style was retained")
	}
}

func TestSingleAttrCacheKeepsAdmittingPastCapacity(t *testing.T) { //nolint:paralleltest // global cache
	for i := range 2*maxSingleAttrCacheSize + 50 {
		canonicalSingleAttrKey(classAttrName, fmt.Sprintf("c%d", i))
	}

	last := fmt.Sprintf("c%d", 2*maxSingleAttrCacheSize+49)

	singleAttrCacheMu.RLock()
	_, ok := singleAttrCache[last]
	size := len(singleAttrCache)
	singleAttrCacheMu.RUnlock()

	if !ok || size > maxSingleAttrCacheSize {
		t.Fatalf("latest cached=%t size=%d (cap %d)", ok, size, maxSingleAttrCacheSize)
	}
}
