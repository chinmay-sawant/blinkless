package css

import "testing"

func nestedVarColor(depth int) string {
	value := "red"

	for range depth {
		value = "var(--a, " + value + ")"
	}

	return value
}

func colorIsRed(t *testing.T, value string) {
	t.Helper()

	red, green, blue, alpha, parsed := ParseColor(value)
	if !parsed || red != 255 || green != 0 || blue != 0 || alpha != 1 {
		t.Fatalf("%s = (%d,%d,%d,%v,%v), want red", value, red, green, blue, alpha, parsed)
	}
}

func TestParseColorVarFallbackDepth(t *testing.T) {
	t.Parallel()

	colorIsRed(t, "var(--a, red)")
	colorIsRed(t, nestedVarColor(maxColorVarDepth))

	if _, _, _, _, parsed := ParseColor(nestedVarColor(maxColorVarDepth + 1)); parsed {
		t.Fatalf("ParseColor accepted var fallback depth %d", maxColorVarDepth+1)
	}
}

func TestParseColorWideVarFallbackDepth(t *testing.T) {
	t.Parallel()

	rgb, alpha, parsed := ParseColorWide("var(--a, red)")
	if !parsed || rgb != [3]float64{1, 0, 0} || alpha != 1 {
		t.Fatalf("wide var(--a, red) = (%v,%v,%v), want red", rgb, alpha, parsed)
	}

	rgb, alpha, parsed = ParseColorWide(nestedVarColor(maxColorVarDepth))
	if !parsed || rgb != [3]float64{1, 0, 0} || alpha != 1 {
		t.Fatalf("wide var fallback depth %d = (%v,%v,%v), want red", maxColorVarDepth, rgb, alpha, parsed)
	}

	if _, _, parsed = ParseColorWide(nestedVarColor(maxColorVarDepth + 1)); parsed {
		t.Fatalf("ParseColorWide accepted var fallback depth %d", maxColorVarDepth+1)
	}
}
