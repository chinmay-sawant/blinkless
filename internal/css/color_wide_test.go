package css

import (
	"math"
	"testing"
)

// Wide-gamut parsing with HDR headroom. Every case asserts numeric parse
// outputs on the 0..1 wide scale, never stored strings. Reference browser
// for every case: Chrome 143.0.7499.40.

// wideHasHeadroom reports whether any channel leaves the sRGB 0..1 range.
// The 1e-9 slack absorbs float dust from matrix round-trips (for example
// display-p3 white decodes to 1.000000000008).
func wideHasHeadroom(rgb [3]float64) bool {
	const wideHeadroomSlack = 1e-9

	return rgb[0] < -wideHeadroomSlack || rgb[0] > 1+wideHeadroomSlack ||
		rgb[1] < -wideHeadroomSlack || rgb[1] > 1+wideHeadroomSlack ||
		rgb[2] < -wideHeadroomSlack || rgb[2] > 1+wideHeadroomSlack
}

// wideAssertNear checks one parsed triplet against want within tolerance.
func wideAssertNear(t *testing.T, spell string, rgb, want [3]float64, tol float64) {
	t.Helper()

	for i := range oklabChannelCount {
		if math.Abs(rgb[i]-want[i]) > tol {
			t.Errorf("ParseColorWide(%q) = %v, want near %v (tol %v)", spell, rgb, want, tol)

			return
		}
	}
}

func TestParseColorWideGamut(t *testing.T) {
	t.Parallel()

	inGamut := []struct {
		spell string
		want  [3]float64
		alpha float64
	}{
		{"color(display-p3 0 0 0)", [3]float64{0, 0, 0}, 1},
		{"color(display-p3 1 1 1)", [3]float64{1, 1, 1}, 1},
		{"color(srgb 1 0 0)", [3]float64{1, 0, 0}, 1},
		{"hwb(0 0% 0%)", [3]float64{1, 0, 0}, 1},
		{"hwb(0 0% 0% / 50%)", [3]float64{1, 0, 0}, 0.5},
		{"lab(100% 0 0)", [3]float64{1, 1, 1}, 1},
		{"lch(0% 0 0)", [3]float64{0, 0, 0}, 1},
		{"red", [3]float64{1, 0, 0}, 1},
		{"transparent", [3]float64{0, 0, 0}, 0},
	}

	for _, testCase := range inGamut {
		rgb, alpha, ok := ParseColorWide(testCase.spell)
		if !ok {
			t.Errorf("ParseColorWide(%q) failed to parse", testCase.spell)

			continue
		}

		if alpha != testCase.alpha {
			t.Errorf("ParseColorWide(%q) alpha = %v, want %v", testCase.spell, alpha, testCase.alpha)
		}

		wideAssertNear(t, testCase.spell, rgb, testCase.want, 0.02)

		if wideHasHeadroom(rgb) {
			t.Errorf("ParseColorWide(%q) = %v, want in-gamut (no headroom)", testCase.spell, rgb)
		}
	}
}

func TestParseColorWideHeadroom(t *testing.T) {
	t.Parallel()

	headroom := []struct {
		spell string
		want  [3]float64
	}{
		{"color(display-p3 1 0 0)", [3]float64{1.093, -0.227, -0.150}},
		{"color(rec2020 1 0 0)", [3]float64{1.248, -0.388, -0.144}},
		{"color(prophoto-rgb 1 0 0)", [3]float64{1.363, -0.516, -0.090}},
		{"color(a98-rgb 0 1 0)", [3]float64{-0.664, 1.0, -0.229}},
		{"color(display-p3 1 0 0 / 50%)", [3]float64{1.093, -0.227, -0.150}},
		{"lab(60% 70 60)", [3]float64{1.020, 0.284, 0.162}},
		{"lch(60% 90 30)", [3]float64{1.053, 0.214, 0.283}},
		{"oklch(0.7 0.3 30)", [3]float64{1.173, -0.139, -0.128}},
		{"oklab(0.9 -0.4 0.4)", [3]float64{-0.504, 1.109, -0.679}},
		{"var(--gap, color(display-p3 1 0 0))", [3]float64{1.093, -0.227, -0.150}},
	}

	for _, testCase := range headroom {
		rgb, alpha, ok := ParseColorWide(testCase.spell)
		if !ok {
			t.Errorf("ParseColorWide(%q) failed to parse", testCase.spell)

			continue
		}

		wantAlpha := 1.0
		if testCase.spell == "color(display-p3 1 0 0 / 50%)" {
			wantAlpha = 0.5
		}

		if alpha != wantAlpha {
			t.Errorf("ParseColorWide(%q) alpha = %v, want %v", testCase.spell, alpha, wantAlpha)
		}

		wideAssertNear(t, testCase.spell, rgb, testCase.want, 0.03)

		if !wideHasHeadroom(rgb) {
			t.Errorf("ParseColorWide(%q) = %v, want out-of-gamut headroom", testCase.spell, rgb)
		}
	}
}

func TestParseColorWideSDRConsistency(t *testing.T) {
	t.Parallel()

	// SDR spellings keep the clamped byte semantics: rgb() overflow folds,
	// so only the wide-gamut functions carry headroom.
	rgb, alpha, ok := ParseColorWide("rgb(300, -20, 50)")
	if !ok {
		t.Fatal("ParseColorWide(rgb(300, -20, 50)) failed to parse")
	}

	wideAssertNear(t, "rgb(300, -20, 50)", rgb, [3]float64{1, 0, 50.0 / 255.0}, 0.01)

	if alpha != 1 {
		t.Errorf("ParseColorWide(rgb(300, -20, 50)) alpha = %v, want 1", alpha)
	}

	if wideHasHeadroom(rgb) {
		t.Errorf("ParseColorWide(rgb(300, -20, 50)) = %v, want clamped SDR bytes", rgb)
	}
}

func TestParseColorWideMix(t *testing.T) {
	t.Parallel()

	p3Red := [3]float64{1.093, -0.227, -0.150}

	// Mixing identical wide stops returns the stop, headroom intact.
	for _, spell := range []string{
		"color-mix(in srgb, color(display-p3 1 0 0), color(display-p3 1 0 0))",
		"color-mix(in oklab, color(display-p3 1 0 0), color(display-p3 1 0 0))",
	} {
		rgb, alpha, ok := ParseColorWide(spell)
		if !ok {
			t.Errorf("ParseColorWide(%q) failed to parse", spell)

			continue
		}

		if alpha != 1 {
			t.Errorf("ParseColorWide(%q) alpha = %v, want 1", spell, alpha)
		}

		wideAssertNear(t, spell, rgb, p3Red, 0.03)

		if !wideHasHeadroom(rgb) {
			t.Errorf("ParseColorWide(%q) = %v, want headroom preserved through the mix", spell, rgb)
		}
	}

	// New interpolation spaces parse, including the previously rejected
	// in-oklab form.
	for _, spell := range []string{
		"color-mix(in oklab, red, blue)",
		"color-mix(in display-p3, red, blue)",
		"color-mix(in lab, red 30%, blue)",
		"color-mix(in oklch, red, blue)",
		"color-mix(in hsl, red, blue)",
		"color-mix(in srgb, red 200%, blue -50%)",
	} {
		if _, alpha, ok := ParseColorWide(spell); !ok || alpha != 1 {
			t.Errorf("ParseColorWide(%q) = (alpha %v, %v), want parsed with alpha 1", spell, alpha, ok)
		}
	}

	// Round-tripping sRGB red through display-p3 mixing stays red.
	rgb, _, ok := ParseColorWide("color-mix(in display-p3, red, red)")
	if !ok {
		t.Fatal("ParseColorWide(color-mix(in display-p3, red, red)) failed to parse")
	}

	wideAssertNear(t, "color-mix(in display-p3, red, red)", rgb, [3]float64{1, 0, 0}, 0.01)
}

func TestParseColorWideRejects(t *testing.T) {
	t.Parallel()

	for _, spell := range []string{
		"",
		"not-a-color",
		"color(unknown-space 1 0 0)",
		"color(display-p3 1 0)",
		"color(display-p3 1 0 0 / x)",
		"lab(50%)",
		"lch(50% 10)",
		"hwb(0 0%)",
		"oklch(0.5 0.1)",
		"color-mix(in xyz, red, blue)",
		"color-mix(in srgb, red)",
		"light-dark(red)",
	} {
		if rgb, alpha, ok := ParseColorWide(spell); ok {
			t.Errorf("ParseColorWide(%q) = (%v, %v), want rejection", spell, rgb, alpha)
		}
	}
}

func TestParseColorLegacyStaysClamped(t *testing.T) {
	t.Parallel()

	// The alongside boundary: ParseColor's 0..255 int return (values.go)
	// cannot carry headroom, so it keeps rejecting the wide-gamut functions
	// and keeps clamping rgb() overflow. Only ParseColorWide feeds the
	// dynamic-range-limit clamp with headroom.
	for _, spell := range []string{
		"color(display-p3 1 0 0)",
		"lab(60% 70 60)",
		"lch(60% 90 30)",
		"hwb(0 0% 0%)",
		"color-mix(in oklab, red, blue)",
	} {
		if red, green, blue, alpha, ok := ParseColor(spell); ok {
			t.Errorf("ParseColor(%q) = (%v, %v, %v, %v), want rejection", spell, red, green, blue, alpha)
		}
	}

	red, green, blue, _, ok := ParseColor("rgb(300, -20, 50)")
	if !ok || red != 255 || green != 0 || blue != 50 {
		t.Errorf("ParseColor(rgb(300, -20, 50)) = (%v, %v, %v), want clamped (255, 0, 50)", red, green, blue)
	}
}
