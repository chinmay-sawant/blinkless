package layout

import "testing"

// Color-adjust behavior tests. Each property below is asserted through used
// paint values (0..1 RGB floats), never through stored style strings. The
// live Layout paint path has no forced-colors mode (print) and emits no root
// canvas fill, so forced-color-adjust and the color-scheme canvas default are
// pinned at the paint helper they will feed; dynamic-range-limit additionally
// asserts a real fill op through Layout. Reference browser for every case:
// Chrome 143.0.7499.40.

// TestBehaviorColorSchemeDarkDefaults is color-scheme: a dark root resolves
// the #121212 canvas fill with #e8e8e8 default text while normal and light
// keep the transparent canvas with black text. Only the html root's value is
// consumed; nested values stay parsed and inherited (fixture-63 prose).
func TestBehaviorColorSchemeDarkDefaults(t *testing.T) {
	t.Parallel()

	canvasChannel := float64(darkSchemeCanvasByte) / 255
	textChannel := float64(darkSchemeTextByte) / 255

	for _, scheme := range []string{"dark", "light dark", "dark light", "only dark"} {
		canvas := defaultCanvasForScheme(scheme)
		if canvas != [4]float64{canvasChannel, canvasChannel, canvasChannel, 1} {
			t.Errorf("canvas for %q = %v, want opaque #121212", scheme, canvas)
		}

		if text := defaultTextForScheme(scheme); text != [3]float64{textChannel, textChannel, textChannel} {
			t.Errorf("default text for %q = %v, want #e8e8e8", scheme, text)
		}
	}

	for _, scheme := range []string{"normal", "light", "only light"} {
		if canvas := defaultCanvasForScheme(scheme); canvas != [4]float64{} {
			t.Errorf("canvas for %q = %v, want transparent (paper)", scheme, canvas)
		}

		if text := defaultTextForScheme(scheme); text != [3]float64{} {
			t.Errorf("default text for %q = %v, want black", scheme, text)
		}
	}
}

// TestBehaviorDynamicRangeLimitClampsBrightColors is dynamic-range-limit:
// standard folds out-of-range channels into sRGB [0, 1] while no-limit and
// high preserve headroom. In-gamut red passes through untouched, which is why
// the pinned no-op layout case (red fill, standard vs no-limit) still paints
// identical ops. The tail asserts that real fill op through Layout.
// assertDynamicRangeClamp checks a clamped color against the want value.
func assertDynamicRangeClamp(t *testing.T, wide, want [3]float64, limit string) {
	t.Helper()

	if got := clampDynamicRangeColor(wide, limit); got != want {
		t.Errorf("%s clamp = %v, want %v", limit, got, want)
	}
}

// assertDynamicRangeKept checks headroom values pass through unchanged.
func assertDynamicRangeKept(t *testing.T, wide [3]float64, limit string) {
	t.Helper()

	if got := clampDynamicRangeColor(wide, limit); got != wide {
		t.Errorf("%s clamp = %v, want headroom kept %v", limit, got, wide)
	}
}

func TestBehaviorDynamicRangeLimitClampsBrightColors(t *testing.T) {
	t.Parallel()

	wide := [3]float64{1.4, -0.2, 0.6}

	assertDynamicRangeClamp(t, wide, [3]float64{1, 0, 0.6}, "standard")
	assertDynamicRangeClamp(t, wide, [3]float64{1, 0, 0.6}, "constrained-high")
	assertDynamicRangeKept(t, wide, "no-limit")
	assertDynamicRangeKept(t, wide, "high")

	red := [3]float64{1, 0, 0}
	if got := clampDynamicRangeColor(red, "standard"); got != red {
		t.Errorf("standard clamp of sRGB red = %v, want unchanged %v", got, red)
	}

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="c" style="width:50px;height:20px;background-color:#ff0000;`+
		`dynamic-range-limit:standard"></div></body></html>`)

	found := false

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpFillRect {
			continue
		}

		if near(paintOp.R, 1) && near(paintOp.G, 0) && near(paintOp.B, 0) {
			found = true
		}
	}

	if !found {
		t.Fatal("no sRGB red fill op under dynamic-range-limit:standard")
	}
}

// TestBehaviorForcedColorAdjustNonePreservesAuthor is forced-color-adjust:
// under forced-colors mode, none keeps the author color while auto maps to
// the system color. With forced mode off (print has none) the author color
// always wins, which is why the pinned no-op layout case (auto vs none)
// still paints identical ops.
func TestBehaviorForcedColorAdjustNonePreservesAuthor(t *testing.T) {
	t.Parallel()

	author := [3]float64{0, 0, 1}
	system := [3]float64{1, 1, 1}

	if got := forcedColorsPaintColor(true, "none", author, system); got != author {
		t.Errorf("forced none = %v, want author %v", got, author)
	}

	if got := forcedColorsPaintColor(true, "auto", author, system); got != system {
		t.Errorf("forced auto = %v, want system %v", got, system)
	}

	if got := forcedColorsPaintColor(true, "preserve-parent-color", author, system); got != system {
		t.Errorf("forced preserve-parent-color = %v, want system %v", got, system)
	}

	if got := forcedColorsPaintColor(false, "auto", author, system); got != author {
		t.Errorf("unforced auto = %v, want author %v", got, author)
	}

	if got := forcedColorsPaintColor(false, "none", author, system); got != author {
		t.Errorf("unforced none = %v, want author %v", got, author)
	}
}
