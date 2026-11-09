package layout

import (
	"math"
	"testing"
)

// testMeasureStyleID returns the interned ID for sty at its used text size.
func testMeasureStyleID(t *testing.T, eng *engine, sty *ResolvedStyle) uint64 {
	t.Helper()

	primary := eng.faceFor(sty)
	if primary == nil {
		primary = eng.font
	}

	sizeBits := math.Float64bits(usedFontSize(sty, primary) * eng.scale)

	return eng.measureStyleID(sty, primary, sizeBits)
}

// TestMeasureStyleIDSameStyleOneID pins that a repeated style gets one stable
// ID, in one engine and across two engines with identical descriptors.
//
//nolint:paralleltest // the shared measure-style intern table is process-wide
func TestMeasureStyleIDSameStyleOneID(t *testing.T) {
	eng1, style1 := sharedTestEngine(t, 1)
	style1.FontSize = 17.125

	before := internedMeasureStyleCount()

	id1 := testMeasureStyleID(t, eng1, style1)
	if id1 == 0 {
		t.Fatal("first style ID = 0, want nonzero")
	}

	if again := testMeasureStyleID(t, eng1, style1); again != id1 {
		t.Fatalf("repeat ID = %d, want %d", again, id1)
	}

	eng2, style2 := sharedTestEngine(t, 1)
	style2.FontSize = 17.125

	if id2 := testMeasureStyleID(t, eng2, style2); id2 != id1 {
		t.Fatalf("second engine ID = %d, want %d", id2, id1)
	}

	if after := internedMeasureStyleCount(); after-before > 1 {
		t.Fatalf("intern entries grew by %d for one style, want at most 1", after-before)
	}
}

// TestMeasureStyleIDDiffersForOpticalVariationEmoji pins that the descriptor
// separates the fields that previously lived as strings on the width key.
//
//nolint:paralleltest // the shared measure-style intern table is process-wide
func TestMeasureStyleIDDiffersForOpticalVariationEmoji(t *testing.T) {
	eng, base := sharedTestEngine(t, 1)

	baseID := testMeasureStyleID(t, eng, base)
	if baseID == 0 {
		t.Fatal("base style ID = 0, want nonzero")
	}

	variants := []struct {
		name  string
		apply func(*ResolvedStyle)
	}{
		{name: "optical", apply: func(style *ResolvedStyle) { style.FontOpticalSizing = "none" }},
		{name: "variations", apply: func(style *ResolvedStyle) {
			style.FontVariationSettings = `"wght" 700`
		}},
		{name: "emoji", apply: func(style *ResolvedStyle) { style.FontVariantEmoji = "emoji" }},
	}

	seen := map[uint64]string{baseID: "base"}

	for _, variant := range variants {
		style := sharedTestStyleCopy(base)
		variant.apply(style)

		id := testMeasureStyleID(t, eng, style)
		if id == 0 {
			t.Fatalf("%s: style ID = 0, want nonzero", variant.name)
		}

		if other, ok := seen[id]; ok {
			t.Fatalf("%s shares ID %d with %s", variant.name, id, other)
		}

		seen[id] = variant.name
	}
}

// TestSharedTextWidthSeparatesOpticalVariationEmoji proves two styles that
// differ only in optical sizing, variation settings, or emoji presentation
// never read each other's cached width.
//
//nolint:paralleltest // clears the process-wide shared measure caches and counters
func TestSharedTextWidthSeparatesOpticalVariationEmoji(t *testing.T) {
	clearSharedMeasureCaches()
	resetSharedMeasureCounters()

	const text = "descriptor-width-split-label"

	engBase, base := sharedTestEngine(t, 1)
	if width := engBase.measureTextFace(text, base); width <= 0 {
		t.Fatalf("base width = %g, want positive", width)
	}

	variants := []struct {
		name  string
		apply func(*ResolvedStyle)
	}{
		{name: "optical", apply: func(style *ResolvedStyle) { style.FontOpticalSizing = "none" }},
		{name: "variations", apply: func(style *ResolvedStyle) {
			style.FontVariationSettings = `"wght" 700`
		}},
		{name: "emoji", apply: func(style *ResolvedStyle) { style.FontVariantEmoji = "emoji" }},
	}

	//nolint:paralleltest // each subtest resets the process-wide measure counters
	for _, variant := range variants {
		t.Run(variant.name, func(t *testing.T) {
			eng, style := sharedTestEngine(t, 1)
			variantStyle := sharedTestStyleCopy(style)
			variant.apply(variantStyle)

			resetSharedMeasureCounters()

			width := eng.measureTextFace(text, variantStyle)
			if width <= 0 {
				t.Fatalf("%s width = %g, want positive", variant.name, width)
			}

			if hits := sharedTextHits.Load(); hits != 0 {
				t.Fatalf("%s shared hits = %d, want 0", variant.name, hits)
			}

			if misses := sharedTextMisses.Load(); misses == 0 {
				t.Fatalf("%s shared misses = 0, want at least one", variant.name)
			}
		})
	}
}

// TestMeasureStyleInternBoundReturnsZero pins the table bound: existing
// descriptors keep their ID, a new descriptor at the bound gets 0 so callers
// bypass the shared cache instead of reusing an ID.
func TestMeasureStyleInternBoundReturnsZero(t *testing.T) {
	t.Parallel()

	table := &measureStyleTable{
		ids:  make(map[textMeasureDesc]uint64),
		next: 1,
	}

	first := textMeasureDesc{weight: 400}
	second := textMeasureDesc{weight: 500}
	third := textMeasureDesc{weight: 600}

	if id := table.intern(first, 2); id != 1 {
		t.Fatalf("first intern = %d, want 1", id)
	}

	if id := table.intern(first, 2); id != 1 {
		t.Fatalf("repeat intern = %d, want 1", id)
	}

	if id := table.intern(second, 2); id != 2 {
		t.Fatalf("second intern = %d, want 2", id)
	}

	if id := table.intern(third, 2); id != 0 {
		t.Fatalf("full-table intern = %d, want 0", id)
	}

	if got := len(table.ids); got != 2 {
		t.Fatalf("interned entries = %d, want 2", got)
	}
}
