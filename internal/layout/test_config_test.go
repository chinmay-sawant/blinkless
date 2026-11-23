package layout

import (
	"testing"
)

// TestConfigIsolatedAcrossParallelRuns lays out one document with every
// bypass on and off at the same time. The switches ride per-run configs, so
// the runs neither race (go test -race) nor change each other's output.
func TestConfigIsolatedAcrossParallelRuns(t *testing.T) {
	t.Parallel()

	doc := simpleFlexInvoiceDoc(12)
	want := layoutHTML(t, doc, sheet(t, simpleFlexInvoiceCSS))

	for name, cfg := range map[string]*TestConfig{
		"all-bypassed": {
			FlexRowMemoBypass: true, SimpleFlexHeightBypass: true,
			MulticolMemoBypass: true,
		},
		"default":      nil,
		"height-only":  {SimpleFlexHeightBypass: true},
		"row-memo-off": {FlexRowMemoBypass: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := layoutHTMLCfg(t, doc, cfg, sheet(t, simpleFlexInvoiceCSS))
			if !opsDeeplyEqual(want.Ops, got.Ops) || want.Height != got.Height {
				t.Fatalf("%s: layout differs from the default run (ops %d vs %d, height %v vs %v)",
					name, len(want.Ops), len(got.Ops), want.Height, got.Height)
			}
		})
	}
}

// TestConfigAbsentIsZero pins the production reading: no config means every
// switch is off and the shared cache bounds keep their defaults.
func TestConfigAbsentIsZero(t *testing.T) {
	t.Parallel()

	var cfg *TestConfig

	if cfg.flexRowMemoBypass() || cfg.simpleFlexHeightBypass() || cfg.multicolMemoBypass() || cfg.paintRestyleDisabled() {
		t.Fatal("nil config enabled a bypass")
	}

	if cfg.sharedTextBound() != maxTextWidthCacheEntries || cfg.sharedRuneBound() != maxRuneWidthCacheEntries {
		t.Fatal("nil config changed a shared cache bound")
	}
}
