package layout

import (
	"bytes"
	"slices"
	"testing"
)

func TestImageFilterCacheHitsAndDifferentiatesInputs(t *testing.T) {
	t.Parallel()

	pngA, _, _, ok := renderGradientPNG("linear-gradient(#c0392b, #2980b9)", 32, 24, [3]float64{})
	if !ok {
		t.Fatal("renderGradientPNG A failed")
	}

	pngB, _, _, ok := renderGradientPNG("linear-gradient(#27ae60, #8e44ad)", 32, 24, [3]float64{})
	if !ok {
		t.Fatal("renderGradientPNG B failed")
	}

	blur4 := []parsedFilter{{kind: filterBlur, val: 4}}
	blur2 := []parsedFilter{{kind: filterBlur, val: 2}}
	gray := []parsedFilter{{kind: filterGrayscale, val: 1}}

	first := applyImageFilterToImageWithInterp(pngA, blur4, "srgb")
	second := applyImageFilterToImageWithInterp(pngA, blur4, "srgb")
	if !bytes.Equal(first, second) {
		t.Fatal("cached filter result differed from first result")
	}

	if &first[0] != &second[0] {
		t.Fatal("expected second call to return cached slice")
	}

	diffRadius := applyImageFilterToImageWithInterp(pngA, blur2, "srgb")
	if bytes.Equal(first, diffRadius) {
		t.Fatal("different blur radius unexpectedly matched cached output")
	}

	diffKind := applyImageFilterToImageWithInterp(pngA, gray, "srgb")
	if bytes.Equal(first, diffKind) {
		t.Fatal("different filter kind unexpectedly matched cached output")
	}

	diffImage := applyImageFilterToImageWithInterp(pngB, blur4, "srgb")
	if bytes.Equal(first, diffImage) {
		t.Fatal("different source image unexpectedly matched cached output")
	}

	diffSpace := applyImageFilterToImageWithInterp(pngA, blur4, "linearrgb")
	if &first[0] == &diffSpace[0] {
		t.Fatal("different interpolation space unexpectedly shared cache entry")
	}
}

func TestClipPathMaskCacheHitsAcrossTranslatedBoxes(t *testing.T) {
	t.Parallel()

	pngData, _, _, ok := renderGradientPNG("linear-gradient(#c0392b, #2980b9)", 40, 20, [3]float64{})
	if !ok {
		t.Fatal("renderGradientPNG failed")
	}

	shape, ok := parseClipPathShape("inset(4px round 8px)", 12)
	if !ok {
		t.Fatal("parseClipPathShape failed")
	}

	box1 := maskImageWithClipPath(pngData, shape, 10, 20, 40, 20, 10, 20, 40, 20)
	box2 := maskImageWithClipPath(pngData, shape, 150, 280, 40, 20, 150, 280, 40, 20)
	if len(box1) == 0 || !bytes.Equal(box1, box2) {
		t.Fatal("translated box with same relative geometry produced different mask")
	}

	if &box1[0] != &box2[0] {
		t.Fatal("expected translated box with identical relative geometry to hit mask cache")
	}

	shifted := maskImageWithClipPath(pngData, shape, 14, 20, 40, 20, 10, 20, 40, 20)
	if bytes.Equal(box1, shifted) {
		t.Fatal("shifted relative offset unexpectedly matched cached mask")
	}

	circle, ok := parseClipPathShape("circle(8px at center)", 12)
	if !ok {
		t.Fatal("parseClipPathShape circle failed")
	}

	circleMask := maskImageWithClipPath(pngData, circle, 10, 20, 40, 20, 10, 20, 40, 20)
	if bytes.Equal(box1, circleMask) {
		t.Fatal("different clip-path shape unexpectedly matched cached mask")
	}
}

func TestSplitTextToWidthASCIIMatchesRunePath(t *testing.T) {
	t.Parallel()

	eng, sty := sharedTestEngine(t, 1)

	cases := []struct {
		text string
		mode softBreakMode
	}{
		{"abcdefghijklmno", softBreakNone},
		{"https://example.com/a/b/c", softBreakURL},
		{"inner-alpha-beta-gamma", softBreakWord},
	}

	for _, tc := range cases {
		got := eng.splitTextToWidthASCII(tc.text, sty, 24, 24, tc.mode)
		if len(got) < 2 {
			t.Fatalf("expected %q to split into multiple chunks, got %v", tc.text, got)
		}

		// Compare against the rune path directly.
		runes := []rune(tc.text)
		var want []string
		for len(runes) > 0 {
			node := eng.fittingPrefix(runes, 24, sty)
			if tc.mode != softBreakNone && node < len(runes) && node > 1 {
				if soft := lastSoftBreak(runes[:node], tc.mode); soft > 0 {
					node = soft
				}
			}

			want = append(want, string(runes[:node]))
			runes = runes[node:]
		}

		if !slices.Equal(got, want) {
			t.Fatalf("splitTextToWidthASCII(%q, %v) = %v, want %v", tc.text, tc.mode, got, want)
		}
	}
}
