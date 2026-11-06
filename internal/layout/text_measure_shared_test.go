//nolint:all // shared text cache tests run sequentially and use unexported internals.
package layout

import (
	"math"
	"sync"
	"testing"

	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
)

func sharedTestEngine(t *testing.T, scale float64) (*engine, *ResolvedStyle) {
	t.Helper()

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatalf("LoadDefaultFaces: %v", err)
	}

	eng := &engine{font: faces.Regular, faces: faces, scale: scale}
	style := initialStyle()
	style.FontSize = 12

	return eng, &style
}

func sharedTestStyleCopy(style *ResolvedStyle) *ResolvedStyle {
	out := *style

	return &out
}

func TestSharedTextWidthHitOnIdenticalRepeat(t *testing.T) {
	clearSharedMeasureCaches()
	resetSharedMeasureCounters()

	eng1, style1 := sharedTestEngine(t, 1)
	const text = "shared-cache-hit-label-123"

	w1 := eng1.measureTextFace(text, style1)
	if w1 <= 0 {
		t.Fatalf("first width = %g, want positive", w1)
	}

	if got := sharedTextMisses.Load(); got != 1 {
		t.Fatalf("misses after first = %d, want 1", got)
	}

	eng2, _ := sharedTestEngine(t, 1)
	style2 := sharedTestStyleCopy(style1)
	resetSharedMeasureCounters()

	w2 := eng2.measureTextFace(text, style2)
	if w2 != w1 {
		t.Fatalf("second width = %g, want %g", w2, w1)
	}

	if got := sharedTextHits.Load(); got != 1 {
		t.Fatalf("shared hits after repeat = %d, want 1", got)
	}

	resetSharedMeasureCounters()
	w3 := eng2.measureTextFace(text, style2)
	if w3 != w1 {
		t.Fatalf("third width = %g, want %g", w3, w1)
	}

	if got := sharedTextHits.Load(); got != 0 {
		t.Fatalf("shared hits after per-layout repeat = %d, want 0", got)
	}

	if got := sharedTextMisses.Load(); got != 0 {
		t.Fatalf("shared misses after per-layout repeat = %d, want 0", got)
	}
}

func TestSharedTextWidthMissOnEachKeyComponent(t *testing.T) {
	const baseText = "shared-cache-miss-base"

	type variant struct {
		name  string
		apply func(style *ResolvedStyle) string
		scale float64
	}

	variants := []variant{
		{name: "content", apply: func(style *ResolvedStyle) string { return baseText + "!" }, scale: 1},
		{name: "size", apply: func(style *ResolvedStyle) string {
			style.FontSize = 13
			return baseText
		}, scale: 1},
		{name: "letter", apply: func(style *ResolvedStyle) string {
			style.LetterSpacing = 1
			return baseText
		}, scale: 1},
		{name: "word", apply: func(style *ResolvedStyle) string {
			style.WordSpacing = 1
			return baseText
		}, scale: 1},
		{name: "tab", apply: func(style *ResolvedStyle) string {
			style.TabSize = 4
			return "a\tb-shared"
		}, scale: 1},
		{name: "tablen", apply: func(style *ResolvedStyle) string {
			if style.CustomProps == nil {
				style.CustomProps = map[string]string{}
			}
			style.CustomProps["__tab_size_is_length"] = "1"
			return "a\tb-shared"
		}, scale: 1},
		{name: "autospace", apply: func(style *ResolvedStyle) string {
			style.TextAutospace = "ideograph-alpha"
			return baseText
		}, scale: 1},
		{name: "emoji", apply: func(style *ResolvedStyle) string {
			style.FontVariantEmoji = "emoji"
			return baseText
		}, scale: 1},
		{name: "weight", apply: func(style *ResolvedStyle) string {
			style.FontWeight = 700
			return baseText
		}, scale: 1},
		{name: "italic", apply: func(style *ResolvedStyle) string {
			style.FontItalic = true
			return baseText
		}, scale: 1},
		{name: "optical", apply: func(style *ResolvedStyle) string {
			style.FontOpticalSizing = "none"
			return baseText
		}, scale: 1},
		{name: "variations", apply: func(style *ResolvedStyle) string {
			style.FontVariationSettings = `"wght" 700`
			return baseText
		}, scale: 1},
		{name: "family", apply: func(style *ResolvedStyle) string {
			style.FontFamily = []string{"serif"}
			style.famHash = hashFontFamily(style.FontFamily)
			return baseText
		}, scale: 1},
		{name: "width", apply: func(style *ResolvedStyle) string { return baseText }, scale: 2},
	}

	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			clearSharedMeasureCaches()
			resetSharedMeasureCounters()

			engBase, styleBase := sharedTestEngine(t, 1)
			base := baseText
			if v.name == "tab" || v.name == "tablen" {
				base = "a\tb-shared"
			}
			_ = engBase.measureTextFace(base, styleBase)

			eng, style := sharedTestEngine(t, v.scale)
			text := v.apply(style)
			resetSharedMeasureCounters()

			_ = eng.measureTextFace(text, style)
			if got := sharedTextHits.Load(); got != 0 {
				t.Fatalf("%s: shared hits = %d, want 0", v.name, got)
			}
		})
	}
}

func TestSharedTextWidthFontChangeMisses(t *testing.T) {
	clearSharedMeasureCaches()
	resetSharedMeasureCounters()

	eng1, style1 := sharedTestEngine(t, 1)
	const text = "shared-font-change"
	_ = eng1.measureTextFace(text, style1)

	eng2, style2 := sharedTestEngine(t, 1)
	style2.FontWeight = 700
	resetSharedMeasureCounters()

	_ = eng2.measureTextFace(text, style2)
	if got := sharedTextHits.Load(); got != 0 {
		t.Fatalf("font change hits = %d, want 0", got)
	}
}

func TestSharedTextWidthBoundedEviction(t *testing.T) {
	oldTextBound := maxSharedTextWidthEntries
	oldRuneBound := maxSharedRuneWidthEntries
	maxSharedTextWidthEntries = 8
	maxSharedRuneWidthEntries = 8
	defer func() {
		maxSharedTextWidthEntries = oldTextBound
		maxSharedRuneWidthEntries = oldRuneBound
	}()

	clearSharedMeasureCaches()
	resetSharedMeasureCounters()

	eng, style := sharedTestEngine(t, 1)
	for i := 0; i < 32; i++ {
		text := "evict-label-" + string(rune('a'+i%26)) + "-" + string(rune('0'+i/26))
		_ = eng.measureTextFace(text, style)
	}

	if got := sharedTextCacheLen(); got > 8 {
		t.Fatalf("shared text entries = %d, want at most 8", got)
	}

	for i := 0; i < 32; i++ {
		_ = eng.measureRuneFace(rune(0x61+i), style)
	}

	if got := sharedRuneCacheLen(); got > 8 {
		t.Fatalf("shared rune entries = %d, want at most 8", got)
	}
}

func TestSharedTextWidthConcurrentUse(t *testing.T) {
	clearSharedMeasureCaches()
	resetSharedMeasureCounters()

	eng, style := sharedTestEngine(t, 1)
	texts := []string{"concurrent-alpha", "concurrent-beta", "concurrent-gamma", "price 12.99"}
	want := make([]float64, len(texts))
	for i, text := range texts {
		want[i] = eng.measureTextFace(text, style)
	}

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan string, workers)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e, s := sharedTestEngine(t, 1)
			for i, text := range texts {
				if got := e.measureTextFace(text, s); got != want[i] {
					errs <- "mismatch"
					return
				}
				if got := e.measureRuneFace('x', s); got <= 0 {
					errs <- "bad rune"
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatalf("concurrent use: %s", e)
	}
}

func TestSharedTextWidthParityWithAndWithoutCache(t *testing.T) {
	clearSharedMeasureCaches()
	resetSharedMeasureCounters()

	eng, style := sharedTestEngine(t, 1)
	texts := []string{"parity label", "price 12.99", "a\tb", "Hello World"}
	styles := []*ResolvedStyle{style}
	wide := sharedTestStyleCopy(style)
	wide.LetterSpacing = 1
	wide.WordSpacing = 1
	styles = append(styles, wide)

	for _, s := range styles {
		for _, text := range texts {
			primary := eng.faceFor(s)
			if primary == nil {
				primary = eng.font
			}
			size := usedFontSize(s, primary) * eng.scale
			direct := eng.measureTextFaceUncached(text, s, primary, size)
			cached := eng.measureTextFace(text, s)
			if math.Abs(direct-cached) > 1e-9 {
				t.Fatalf("parity %q = %g cached vs %g direct", text, cached, direct)
			}
		}
	}

	r := 'R'
	directRune := eng.measureRuneFaceUncached(r, style)
	cachedRune := eng.measureRuneFace(r, style)
	if directRune != cachedRune {
		t.Fatalf("rune parity = %g vs %g", cachedRune, directRune)
	}

	long := ""
	for i := 0; i < 200; i++ {
		long += "x"
	}
	primary := eng.faceFor(style)
	if primary == nil {
		primary = eng.font
	}
	size := usedFontSize(style, primary) * eng.scale
	if got := eng.measureTextFace(long, style); got != eng.measureTextFaceUncached(long, style, primary, size) {
		t.Fatalf("long text bypass differs")
	}
}

func TestSharedTextWidthAcrossLayouts(t *testing.T) {
	clearSharedMeasureCaches()
	resetSharedMeasureCounters()

	root := mustParse(t, relayoutFixtureHTML)
	opts := relayoutOptions(t)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	hitsAfterFirst := sharedTextHits.Load()
	if hitsAfterFirst == 0 && sharedTextMisses.Load() == 0 {
		t.Fatalf("first layout recorded no shared activity")
	}

	resetSharedMeasureCounters()
	root2 := mustParse(t, relayoutFixtureHTML)
	if _, err := Layout(root2, opts); err != nil {
		t.Fatalf("second Layout: %v", err)
	}

	if got := sharedTextHits.Load(); got == 0 {
		t.Fatalf("second layout shared hits = 0, want > 0")
	}

	resetSharedMeasureCounters()
	first := relayoutNodeByID(t, root, "first")
	first.Attrs["class"] = relayoutSelectedClass
	if _, err := RelayoutSubtree(t.Context(), prev, first, opts); err != nil {
		t.Fatalf("RelayoutSubtree: %v", err)
	}

	if got := sharedTextHits.Load() + sharedRuneHits.Load(); got == 0 {
		t.Fatalf("subtree relayout shared hits = 0, want > 0")
	}
}
