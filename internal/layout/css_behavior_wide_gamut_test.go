package layout

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// Wide-gamut author colors through dynamic-range-limit. Every case asserts
// numeric used paint values, never stored style strings. Reference browser
// for every case: Chrome 143.0.7499.40 preserves wide-gamut headroom under
// no-limit/high and folds it into sRGB under standard/constrained-high.

// wideGamutSpellings are author colors whose sRGB conversion leaves [0, 1]
// through css.ParseColorWide: display-p3, rec2020, prophoto and a98
// primaries, high-chroma lab/lch/oklch/oklab, and wide-gamut color-mix.
func wideGamutSpellings() []string {
	return []string{
		"color(display-p3 1 0 0)",
		"color(rec2020 1 0 0)",
		"color(prophoto-rgb 1 0 0)",
		"color(a98-rgb 0 1 0)",
		"lab(60% 70 60)",
		"lch(60% 90 30)",
		"oklch(0.7 0.3 30)",
		"oklab(0.9 -0.4 0.4)",
		"color-mix(in srgb, color(display-p3 1 0 0), color(display-p3 1 0 0))",
		"color-mix(in oklab, color(display-p3 1 0 0), color(display-p3 1 0 0))",
	}
}

// wideGamutHeadroom reports whether any used channel leaves sRGB [0, 1].
// The 1e-9 slack absorbs float dust from matrix round-trips.
func wideGamutHeadroom(rgb [3]float64) bool {
	const wideGamutSlack = 1e-9

	for _, channel := range rgb {
		if channel < -wideGamutSlack || channel > 1+wideGamutSlack {
			return true
		}
	}

	return false
}

// wideGamutFills resolves one author color to used fills under the four
// dynamic-range-limit values, failing the test on any parse rejection.
func wideGamutFills(t *testing.T, spell string) ([4]float64, [4]float64, [4]float64, [4]float64) {
	t.Helper()

	limits := [4]string{"standard", "no-limit", "high", "constrained-high"}
	fills := [4][4]float64{}

	for index, limit := range limits {
		fill, parsed := usedWideBGForPaint(spell, limit)
		if !parsed {
			t.Fatalf("usedWideBGForPaint(%q, %q) failed", spell, limit)
		}

		fills[index] = fill
	}

	return fills[0], fills[1], fills[2], fills[3]
}

// wideGamutInks resolves one author color to used text ink under standard
// and no-limit, failing the test on any parse rejection.
func wideGamutInks(t *testing.T, spell string) ([3]float64, [3]float64) {
	t.Helper()

	standard, parsed := usedWideTextForPaint(spell, "standard")
	if !parsed {
		t.Fatalf("usedWideTextForPaint(%q, standard) failed", spell)
	}

	kept, parsed := usedWideTextForPaint(spell, "no-limit")
	if !parsed {
		t.Fatalf("usedWideTextForPaint(%q, no-limit) failed", spell)
	}

	return standard, kept
}

// assertWideGamutFillDiverges checks one spelling's used fills: standard
// and constrained-high fold headroom into sRGB while no-limit and high
// preserve it, so the pairs agree and the halves diverge.
func assertWideGamutFillDiverges(t *testing.T, spell string, wide [3]float64) {
	t.Helper()

	standard, kept, high, constrained := wideGamutFills(t, spell)
	standardRGB := [3]float64{standard[0], standard[1], standard[2]}

	if wideGamutHeadroom(standardRGB) {
		t.Errorf("standard %q = %v, want folded into sRGB [0, 1]", spell, standard)
	}

	if kept != ([4]float64{wide[0], wide[1], wide[2], 1}) {
		t.Errorf("no-limit %q = %v, want headroom kept %v", spell, kept, wide)
	}

	if high != kept {
		t.Errorf("high %q = %v, want same headroom as no-limit %v", spell, high, kept)
	}

	if constrained != standard {
		t.Errorf("constrained-high %q = %v, want same fold as standard %v", spell, constrained, standard)
	}

	if standard == kept {
		t.Errorf("standard and no-limit identical %v for %q, want divergence", standard, spell)
	}

	if standard[3] != 1 || kept[3] != 1 {
		t.Errorf("%q alpha changed across the clamp: standard %v no-limit %v", spell, standard, kept)
	}
}

// TestBehaviorWideGamutDynamicRangeDiverges is dynamic-range-limit over
// wide-gamut author colors: standard and constrained-high fold headroom
// into sRGB [0, 1] while no-limit and high preserve the out-of-range used
// channels. Reference: Chrome 143.0.7499.40.
func TestBehaviorWideGamutDynamicRangeDiverges(t *testing.T) {
	t.Parallel()

	for _, spell := range wideGamutSpellings() {
		wide, alpha, parsed := css.ParseColorWide(spell)
		if !parsed {
			t.Fatalf("ParseColorWide(%q) failed to parse", spell)
		}

		if alpha != 1 {
			t.Errorf("ParseColorWide(%q) alpha = %v, want 1", spell, alpha)
		}

		if !wideGamutHeadroom(wide) {
			t.Errorf("ParseColorWide(%q) = %v, want out-of-gamut headroom", spell, wide)
		}

		assertWideGamutFillDiverges(t, spell, wide)
	}
}

// TestBehaviorWideGamutTextDiverges is the text-ink half: used ink folds
// under standard and keeps headroom under no-limit. Reference: Chrome
// 143.0.7499.40.
func TestBehaviorWideGamutTextDiverges(t *testing.T) {
	t.Parallel()

	for _, spell := range wideGamutSpellings() {
		wide, _, parsed := css.ParseColorWide(spell)
		if !parsed {
			t.Fatalf("ParseColorWide(%q) failed to parse", spell)
		}

		standard, kept := wideGamutInks(t, spell)

		if wideGamutHeadroom(standard) {
			t.Errorf("standard ink %q = %v, want folded into sRGB [0, 1]", spell, standard)
		}

		if kept != wide {
			t.Errorf("no-limit ink %q = %v, want headroom kept %v", spell, kept, wide)
		}

		if standard == kept {
			t.Errorf("standard and no-limit ink identical %v for %q, want divergence", standard, spell)
		}
	}
}

// TestBehaviorWideGamutInGamutAgrees pins the other side: in-gamut author
// colors paint identical used values under standard and no-limit, so only
// headroom diverges. Reference: Chrome 143.0.7499.40.
func TestBehaviorWideGamutInGamutAgrees(t *testing.T) {
	t.Parallel()

	for _, spell := range []string{"#ff0000", "hwb(0 0% 0%)", "color(display-p3 0 0 0)"} {
		standard, parsed := usedWideBGForPaint(spell, "standard")
		if !parsed {
			t.Fatalf("usedWideBGForPaint(%q, standard) failed", spell)
		}

		kept, parsed := usedWideBGForPaint(spell, "no-limit")
		if !parsed {
			t.Fatalf("usedWideBGForPaint(%q, no-limit) failed", spell)
		}

		if standard != kept {
			t.Errorf("in-gamut %q diverges: standard %v vs no-limit %v", spell, standard, kept)
		}
	}
}

// TestBehaviorWideGamutBridgeBoundary pins the pipeline boundary the
// alongside representation works around: the legacy css.ParseColor byte
// pipeline (values.go ParseColor, 0..255 int return, divide-by-255 storage
// in style_properties.go and style_paint_props.go) still rejects the
// wide-gamut functions and clamps rgb() overflow, so only ParseColorWide
// reaches the dynamic-range-limit clamp with headroom. Reference: Chrome
// 143.0.7499.40.
func TestBehaviorWideGamutBridgeBoundary(t *testing.T) {
	t.Parallel()

	if fill, parsed := usedWideBGForPaint("not-a-color", "standard"); parsed {
		t.Errorf("usedWideBGForPaint(not-a-color) = %v, want rejection", fill)
	}

	if ink, parsed := usedWideTextForPaint("color(unknown-space 1 0 0)", "no-limit"); parsed {
		t.Errorf("usedWideTextForPaint(unknown-space) = %v, want rejection", ink)
	}

	for _, spell := range []string{
		"color(display-p3 1 0 0)",
		"lab(60% 70 60)",
		"lch(60% 90 30)",
		"hwb(0 0% 0%)",
		"color-mix(in oklab, red, blue)",
	} {
		if red, green, blue, alpha, ok := css.ParseColor(spell); ok {
			t.Errorf("ParseColor(%q) = (%v, %v, %v, %v), want rejection", spell, red, green, blue, alpha)
		}
	}

	if red, green, blue, _, ok := css.ParseColor("rgb(300, -20, 50)"); !ok || red != 255 || green != 0 || blue != 50 {
		t.Errorf("ParseColor(rgb(300, -20, 50)) = (%v, %v, %v), want clamped (255, 0, 50)", red, green, blue)
	}
}

// TestBehaviorWideGamutCascadeEndToEnd closes the storage loop the bridge
// tests leave open: background-color:color(display-p3 1 0 0) cascades
// headroom into storage through a real document, and the production paint
// resolver folds it under standard while keeping it under no-limit.
// Reference: Chrome 143.0.7499.40.
func TestBehaviorWideGamutCascadeEndToEnd(t *testing.T) {
	t.Parallel()

	doc := func(limit string) string {
		return `<html><body style="margin:0">` +
			`<div id="w" style="width:100px;height:50px;` +
			`background-color:color(display-p3 1 0 0);dynamic-range-limit:` + limit +
			`"></div></body></html>`
	}

	stdStyle := boxByID(t, layoutHTML(t, doc("standard")), "w").style
	keptStyle := boxByID(t, layoutHTML(t, doc("no-limit")), "w").style

	if !wideGamutHeadroom([3]float64{keptStyle.BGColor[0], keptStyle.BGColor[1], keptStyle.BGColor[2]}) {
		t.Errorf("stored BGColor = %v, want out-of-gamut headroom", keptStyle.BGColor)
	}

	stdUsed := usedBGForPaintActive(*stdStyle, false)
	keptUsed := usedBGForPaintActive(*keptStyle, false)

	if wideGamutHeadroom([3]float64{stdUsed[0], stdUsed[1], stdUsed[2]}) {
		t.Errorf("standard used fill = %v, want folded into sRGB [0, 1]", stdUsed)
	}

	if !wideGamutHeadroom([3]float64{keptUsed[0], keptUsed[1], keptUsed[2]}) {
		t.Errorf("no-limit used fill = %v, want headroom kept", keptUsed)
	}

	if stdUsed == keptUsed {
		t.Errorf("standard and no-limit used fills identical %v, want divergence", stdUsed)
	}
}

// TestBehaviorWideGamutTextCascadeEndToEnd is the text-ink half: color:lab()
// cascades headroom into the used color and the production ink resolver
// folds it under standard while keeping it under no-limit. Reference:
// Chrome 143.0.7499.40.
func TestBehaviorWideGamutTextCascadeEndToEnd(t *testing.T) {
	t.Parallel()

	doc := func(limit string) string {
		return `<html><body style="margin:0">` +
			`<div id="w" style="width:100px;height:50px;color:lab(60% 70 60);dynamic-range-limit:` + limit +
			`">text</div></body></html>`
	}

	stdStyle := boxByID(t, layoutHTML(t, doc("standard")), "w").style
	keptStyle := boxByID(t, layoutHTML(t, doc("no-limit")), "w").style

	stdUsed := usedTextForPaintActive(stdStyle.Color, "", stdStyle.ForcedColorAdjust,
		stdStyle.DynamicRangeLimit, false)
	keptUsed := usedTextForPaintActive(keptStyle.Color, "", keptStyle.ForcedColorAdjust,
		keptStyle.DynamicRangeLimit, false)

	if wideGamutHeadroom(stdUsed) {
		t.Errorf("standard used ink = %v, want folded into sRGB [0, 1]", stdUsed)
	}

	if !wideGamutHeadroom(keptUsed) {
		t.Errorf("no-limit used ink = %v, want headroom kept", keptUsed)
	}

	if stdUsed == keptUsed {
		t.Errorf("standard and no-limit used inks identical %v, want divergence", stdUsed)
	}
}
