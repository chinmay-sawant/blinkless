//nolint:all // targeted unit tests for font prop cascade wiring
package layout

import (
	"context"
	"testing"
)

func TestFontPropsWave4(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       context.Background(),
		viewportW: 800,
	}
	style := initialStyle()
	raw := map[string]string{
		"font-feature-settings": `"liga" 1, "smcp" 1`,
		"font-kerning":          "none",
		"font-variant-caps":     "small-caps",
		"font-stretch":          "condensed",
		"font-size-adjust":      "0.58",
	}
	applyFontProps(&style, raw, 12, ctx)
	applyRestProps(&style, raw, ctx, nil)

	if style.font().FontFeatureSettings != `"liga" 1, "smcp" 1` {
		t.Fatalf("feature-settings = %q", style.font().FontFeatureSettings)
	}

	if style.font().FontKerning != "none" {
		t.Fatalf("kerning = %q", style.font().FontKerning)
	}

	if style.font().FontVariantCaps != "small-caps" {
		t.Fatalf("caps = %q", style.font().FontVariantCaps)
	}

	if style.font().FontWidth != 75 {
		t.Fatalf("stretch/width = %v", style.font().FontWidth)
	}

	if !style.font().FontSizeAdjustSet || style.font().FontSizeAdjust != 0.58 {
		t.Fatalf("size-adjust = set=%v value=%v", style.font().FontSizeAdjustSet, style.font().FontSizeAdjust)
	}
}
