package layout

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// Forced-colors and dynamic-range behavior through used paint values.
// Every case asserts numeric used colors on the paint path, never stored
// style strings. Reference browser for every case: Chrome 143.0.7499.40.

func forcedDynamicDoc(adjust string) string {
	return `<html style="margin:0;forced-color-adjust:` + adjust + `"><body style="margin:0">` +
		`<div id="c" style="width:50px;height:20px;background-color:#0000ff">` +
		`<p id="p" style="margin:0;color:#ffff00">hi</p></div></body></html>`
}

// forcedModeFillDiverges asserts used background fills diverge under the
// test-only forced-colors mode: auto maps blue to system black while none
// keeps author blue.
func forcedModeFillDiverges(t *testing.T) {
	t.Helper()

	blueBG := func(adjust string) ResolvedStyle {
		return ResolvedStyle{
			BGColor:           [4]float64{0, 0, 1, 1},
			ForcedColorAdjust: adjust,
			DynamicRangeLimit: "no-limit",
		}
	}

	autoFill := usedBGForPaint(blueBG("auto"))
	noneFill := usedBGForPaint(blueBG("none"))

	if autoFill != ([4]float64{0, 0, 0, 1}) {
		t.Errorf("forced auto fill = %v, want system black (0, 0, 0, 1)", autoFill)
	}

	if noneFill != ([4]float64{0, 0, 1, 1}) {
		t.Errorf("forced none fill = %v, want author blue (0, 0, 1, 1)", noneFill)
	}

	if autoFill == noneFill {
		t.Errorf("forced fills identical %v, want auto vs none to diverge", autoFill)
	}
}

// forcedModeInkDiverges asserts used text ink diverges under the test-only
// forced-colors mode: auto maps yellow to system black while none keeps it.
func forcedModeInkDiverges(t *testing.T) {
	t.Helper()

	yellow := [3]float64{1, 1, 0}

	autoInk := usedTextForPaint(yellow, "normal", "auto", "no-limit")
	noneInk := usedTextForPaint(yellow, "normal", "none", "no-limit")

	if autoInk != ([3]float64{0, 0, 0}) {
		t.Errorf("forced auto ink = %v, want system black", autoInk)
	}

	if noneInk != yellow {
		t.Errorf("forced none ink = %v, want author yellow %v", noneInk, yellow)
	}

	if autoInk == noneInk {
		t.Errorf("forced inks identical %v, want auto vs none to diverge", autoInk)
	}
}

// forcedModeLayoutDiverges asserts real documents diverge under the test-only
// forced-colors mode: the none root keeps the blue fill and yellow text while
// the auto root maps both to system black.
func forcedModeLayoutDiverges(t *testing.T) {
	t.Helper()

	blue := [3]float64{0, 0, 1}
	yellow := [3]float64{1, 1, 0}
	black := [3]float64{0, 0, 0}

	autoRes := layoutHTML(t, forcedDynamicDoc("auto"))
	noneRes := layoutHTML(t, forcedDynamicDoc("none"))

	if fill := wireFindFill(t, noneRes, blue); fill == nil {
		t.Fatal("forced none painted no blue author fill")
	}

	if text := wireFindText(t, noneRes, yellow); text == nil {
		t.Fatal("forced none painted no yellow author text")
	}

	if fill := wireFindFill(t, autoRes, blue); fill != nil {
		t.Errorf("forced auto kept author blue fill %+v, want system black", *fill)
	}

	if text := wireFindText(t, autoRes, yellow); text != nil {
		t.Errorf("forced auto kept author yellow text %+v, want system black", *text)
	}

	if fill := wireFindFill(t, autoRes, black); fill == nil {
		t.Fatal("forced auto painted no system black fill")
	}

	if text := wireFindText(t, autoRes, black); text == nil {
		t.Fatal("forced auto painted no system black text")
	}
}

// TestBehaviorForcedColorAdjustTestModeDiverges is forced-color-adjust: with
// the explicit test-only forced-colors mode on, auto maps author colors to
// the system color (black) while none preserves them. Print keeps the mode
// off, so both values paint author colors there. Reference: Chrome
// 143.0.7499.40 keeps author colors under none when forced-colors is active.
//
//nolint:paralleltest // flips a global test-only paint override
func TestBehaviorForcedColorAdjustTestModeDiverges(t *testing.T) {
	setForcedColorsTestActive(true)
	defer setForcedColorsTestActive(false)

	forcedModeFillDiverges(t)
	forcedModeInkDiverges(t)
	forcedModeLayoutDiverges(t)
}

// TestBehaviorForcedColorAdjustPrintKeepsAuthor is the print default: with
// the test mode off, auto and none both paint author colors. Reference:
// Chrome 143.0.7499.40 with forced-colors off.
func TestBehaviorForcedColorAdjustPrintKeepsAuthor(t *testing.T) {
	t.Parallel()

	blue := [3]float64{0, 0, 1}
	yellow := [3]float64{1, 1, 0}

	for _, adjust := range []string{"auto", "none"} {
		res := layoutHTML(t, forcedDynamicDoc(adjust))

		if fill := wireFindFill(t, res, blue); fill == nil {
			t.Fatalf("print forced-color-adjust:%s painted no blue author fill", adjust)
		}

		if text := wireFindText(t, res, yellow); text == nil {
			t.Fatalf("print forced-color-adjust:%s painted no yellow author text", adjust)
		}
	}
}

// dynamicRangeWideBG builds a synthetic out-of-range computed background for
// one dynamic-range-limit value. No author syntax parses to this: the CSS
// parser clamps every channel into 0..1.
func dynamicRangeWideBG(limit string) ResolvedStyle {
	return ResolvedStyle{
		BGColor:           [4]float64{1.4, -0.2, 0.6, 1},
		ForcedColorAdjust: "auto",
		DynamicRangeLimit: limit,
	}
}

// TestBehaviorDynamicRangeLimitPaintClampDiverges is dynamic-range-limit:
// standard folds out-of-range computed channels into sRGB while no-limit and
// high keep HDR headroom. The values below are synthetic computed colors,
// not parseable CSS. Reference: Chrome 143.0.7499.40 clamps headroom under
// standard.
func TestBehaviorDynamicRangeLimitPaintClampDiverges(t *testing.T) {
	t.Parallel()

	standardFill := usedBGForPaint(dynamicRangeWideBG("standard"))
	keptFill := usedBGForPaint(dynamicRangeWideBG("no-limit"))
	highFill := usedBGForPaint(dynamicRangeWideBG("high"))
	wantClamped := [4]float64{1, 0, 0.6, 1}
	wantKept := [4]float64{1.4, -0.2, 0.6, 1}

	if standardFill != wantClamped {
		t.Errorf("standard fill = %v, want clamped %v", standardFill, wantClamped)
	}

	if keptFill != wantKept {
		t.Errorf("no-limit fill = %v, want headroom kept %v", keptFill, wantKept)
	}

	if highFill != wantKept {
		t.Errorf("high fill = %v, want headroom kept %v", highFill, wantKept)
	}

	if standardFill == keptFill {
		t.Errorf("fills identical %v, want standard vs no-limit to diverge", standardFill)
	}

	wide := [3]float64{1.4, -0.2, 0.6}
	standardInk := usedTextForPaint(wide, "normal", "auto", "standard")
	keptInk := usedTextForPaint(wide, "normal", "auto", "no-limit")

	if standardInk != ([3]float64{1, 0, 0.6}) {
		t.Errorf("standard ink = %v, want clamped (1, 0, 0.6)", standardInk)
	}

	if keptInk != wide {
		t.Errorf("no-limit ink = %v, want headroom kept %v", keptInk, wide)
	}

	if standardInk == keptInk {
		t.Errorf("inks identical %v, want standard vs no-limit to diverge", standardInk)
	}
}

// TestBehaviorDynamicRangeLimitParserYieldsOnlyInGamut pins the engine gap:
// css.ParseColor clamps every channel (clampByte in values.go, oklabToRGB in
// color_modern.go), so no author syntax reaches the clamp with headroom.
// Wide-gamut spellings still paint identical ops under standard vs no-limit.
// Reference: Chrome 143.0.7499.40 would diverge only with real HDR headroom.
func TestBehaviorDynamicRangeLimitParserYieldsOnlyInGamut(t *testing.T) {
	t.Parallel()

	colors := []string{
		"#ff0000",
		"rgb(300, -20, 50)",
		"oklch(0.7 0.35 30)",
		"oklab(0.9 -0.4 0.4)",
		"color-mix(in srgb, red 50%, blue 50%)",
	}

	for _, color := range colors {
		doc := func(limit string) string {
			return `<html><body style="margin:0">` +
				`<div id="c" style="width:50px;height:20px;background-color:` + color + `;` +
				`dynamic-range-limit:` + limit + `"></div></body></html>`
		}

		behaviorSvgAssertSameOps(t, layoutHTML(t, doc("no-limit")), layoutHTML(t, doc("standard")))
	}
}

// layoutHTMLForcedColors lays out src with the document-wide forced-colors
// mode switch (Options.ForcedColorsActive): the non-test-only route to the
// forced palette, on the same seam as the Options.Background print setting.
func layoutHTMLForcedColors(t *testing.T, src string, forced bool) *Result {
	t.Helper()

	root := mustParse(t, src)

	res, err := Layout(root, Options{
		Width: testViewport, Height: 800, Background: true, ForcedColorsActive: forced,
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	return res
}

// TestBehaviorForcedColorAdjustOptionsModeDiverges is forced-color-adjust:
// with Options.ForcedColorsActive set on real documents, auto maps the blue
// fill and yellow text to the system color (black) while none preserves the
// author colors. Every assertion reads used paint values, never stored style
// strings. Reference: Chrome 143.0.7499.40 keeps author colors under none
// when forced-colors is active.
func TestBehaviorForcedColorAdjustOptionsModeDiverges(t *testing.T) {
	t.Parallel()

	if forcedColorsActive() {
		t.Fatal("test-only forced-colors override leaked on, want the Options route alone")
	}

	blue := [3]float64{0, 0, 1}
	yellow := [3]float64{1, 1, 0}
	black := [3]float64{0, 0, 0}

	forcedAuto := layoutHTMLForcedColors(t, forcedDynamicDoc("auto"), true)
	if fill := wireFindFill(t, forcedAuto, blue); fill != nil {
		t.Errorf("forced auto kept author blue fill %+v, want system black", *fill)
	}

	if text := wireFindText(t, forcedAuto, yellow); text != nil {
		t.Errorf("forced auto kept author yellow text %+v, want system black", *text)
	}

	if fill := wireFindFill(t, forcedAuto, black); fill == nil {
		t.Fatal("forced auto painted no system black fill")
	}

	if text := wireFindText(t, forcedAuto, black); text == nil {
		t.Fatal("forced auto painted no system black text")
	}

	forcedNone := layoutHTMLForcedColors(t, forcedDynamicDoc("none"), true)
	if fill := wireFindFill(t, forcedNone, blue); fill == nil {
		t.Fatal("forced none painted no blue author fill")
	}

	if text := wireFindText(t, forcedNone, yellow); text == nil {
		t.Fatal("forced none painted no yellow author text")
	}

	plainAuto := layoutHTMLForcedColors(t, forcedDynamicDoc("auto"), false)
	if fill := wireFindFill(t, plainAuto, blue); fill == nil {
		t.Fatal("unforced auto painted no blue author fill")
	}

	if text := wireFindText(t, plainAuto, yellow); text == nil {
		t.Fatal("unforced auto painted no yellow author text")
	}
}

// TestBehaviorForcedColorAdjustActiveHelpersDiverge is forced-color-adjust:
// the explicit-flag paint helpers diverge between auto and none without the
// test-only override, proving the document setting (not the global) selects
// the mapping. Reference: Chrome 143.0.7499.40.
func TestBehaviorForcedColorAdjustActiveHelpersDiverge(t *testing.T) {
	t.Parallel()

	blueBG := func(adjust string) ResolvedStyle {
		return ResolvedStyle{
			BGColor:           [4]float64{0, 0, 1, 1},
			ForcedColorAdjust: adjust,
			DynamicRangeLimit: "no-limit",
		}
	}

	autoFill := usedBGForPaintActive(blueBG("auto"), true)
	noneFill := usedBGForPaintActive(blueBG("none"), true)

	if autoFill != ([4]float64{0, 0, 0, 1}) {
		t.Errorf("forced auto fill = %v, want system black (0, 0, 0, 1)", autoFill)
	}

	if noneFill != ([4]float64{0, 0, 1, 1}) {
		t.Errorf("forced none fill = %v, want author blue (0, 0, 1, 1)", noneFill)
	}

	if autoFill == noneFill {
		t.Errorf("forced fills identical %v, want auto vs none to diverge", autoFill)
	}

	yellow := [3]float64{1, 1, 0}
	autoInk := usedTextForPaintActive(yellow, "normal", "auto", "no-limit", true)
	noneInk := usedTextForPaintActive(yellow, "normal", "none", "no-limit", true)

	if autoInk != ([3]float64{0, 0, 0}) {
		t.Errorf("forced auto ink = %v, want system black", autoInk)
	}

	if noneInk != yellow {
		t.Errorf("forced none ink = %v, want author yellow %v", noneInk, yellow)
	}

	offFill := usedBGForPaintActive(blueBG("auto"), false)
	if offFill != ([4]float64{0, 0, 1, 1}) {
		t.Errorf("unforced auto fill = %v, want author blue", offFill)
	}

	// The scheme default selects initial black only: author yellow survives
	// every root scheme through the shared wrapper (mode off here).
	for _, scheme := range []string{"normal", "dark", "light dark"} {
		if ink := usedTextForPaint(yellow, scheme, "auto", "no-limit"); ink != yellow {
			t.Errorf("unforced auto ink under %q = %v, want author yellow %v", scheme, ink, yellow)
		}
	}
}

// TestBehaviorDynamicRangeLimitParserClampsRGB pins the rgb() clamp site:
// channels fold through clampByte (values.go parseRGBColor), so overflow
// spellings cannot carry headroom to the paint clamp. The assertion reads
// numeric parse outputs, never stored strings. Reference: Chrome
// 143.0.7499.40 would diverge only with real HDR headroom.
func TestBehaviorDynamicRangeLimitParserClampsRGB(t *testing.T) {
	t.Parallel()

	if red, green, blue, _, ok := css.ParseColor("rgb(300, -20, 50)"); !ok || red != 255 || green != 0 || blue != 50 {
		t.Errorf("ParseColor(rgb(300, -20, 50)) = (%v, %v, %v, %v), want (255, 0, 50, true)", red, green, blue, ok)
	}
}

// TestBehaviorDynamicRangeLimitParserRejectsWideGamut pins the missing-parser
// gap: parseModernColor dispatches only oklch(), oklab(), color-mix(in srgb,
// ...), and light-dark(), so the color() function, lab(), lch(), and hwb()
// never parse and no author syntax yields out-of-gamut channels.
func TestBehaviorDynamicRangeLimitParserRejectsWideGamut(t *testing.T) {
	t.Parallel()

	for _, spell := range []string{
		"color(display-p3 1 0 0)",
		"color(rec2020 1 0 0)",
		"lab(50% 20 30)",
		"lch(50% 40 30)",
		"hwb(0 0% 0%)",
		"color-mix(in oklab, red, blue)",
	} {
		if red, green, blue, alpha, ok := css.ParseColor(spell); ok {
			t.Errorf("ParseColor(%q) = (%v, %v, %v, %v), want unsupported (no headroom source)", spell, red, green, blue, alpha)
		}
	}
}

// TestBehaviorDynamicRangeLimitParserClampsMixAndOklch pins the remaining
// clamp sites: color-mix percentages clamp (clampPercent in color_modern.go)
// so weights above 100% cannot overshoot, and oklab lightness clamps to 0..1
// (parseOKLabLight) with gamut folding into bytes inside oklabToRGB, so even
// an extreme oklch spelling yields in-gamut bytes.
func TestBehaviorDynamicRangeLimitParserClampsMixAndOklch(t *testing.T) {
	t.Parallel()

	mixed := "color-mix(in srgb, red 200%, blue -50%)"
	if red, green, blue, _, ok := css.ParseColor(mixed); !ok || red != 255 || green != 0 || blue != 0 {
		t.Errorf("ParseColor(200%%/-50%% mix) = (%v, %v, %v, %v), want (255, 0, 0, true)", red, green, blue, ok)
	}
}

// TestBehaviorDynamicRangeLimitParserClampsOklch pins the oklab clamp sites:
// lightness clamps to 0..1 (parseOKLabLight) with gamut folding into bytes
// inside oklabToRGB (color_modern.go), so even an extreme oklch spelling
// yields in-gamut bytes.
func TestBehaviorDynamicRangeLimitParserClampsOklch(t *testing.T) {
	t.Parallel()

	red, green, blue, _, ok := css.ParseColor("oklch(150% 0.4 30)")
	if !ok {
		t.Fatal("ParseColor(oklch(150% 0.4 30)) failed to parse")
	}

	if red < 0 || red > 255 || green < 0 || green > 255 || blue < 0 || blue > 255 {
		t.Errorf("ParseColor(oklch(150%% 0.4 30)) = (%v, %v, %v), want in-gamut bytes", red, green, blue)
	}
}
