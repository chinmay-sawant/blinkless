package layout

import "testing"

// TestRelayoutEnterAncestorPaintStackingContext verifies that an ancestor with
// an empty MixBlendMode does not spuriously create a stacking context, while
// backdrop-filter, non-normal mix-blend-mode, and isolation:isolate do.
func TestRelayoutEnterAncestorPaintStackingContext(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		style    ResolvedStyle
		wantZSet bool
		wantZVal int
	}{
		{
			name:     "empty mix-blend-mode does not force stacking context",
			style:    ResolvedStyle{Opacity: 1, MixBlendMode: ""},
			wantZSet: false,
		},
		{
			name:     "normal mix-blend-mode does not force stacking context",
			style:    ResolvedStyle{Opacity: 1, MixBlendMode: blendNormal},
			wantZSet: false,
		},
		{
			name:     "backdrop-filter forces stacking context",
			style:    ResolvedStyle{Opacity: 1, MixBlendMode: blendNormal, BackdropFilter: "blur(4px)"},
			wantZSet: true,
			wantZVal: 0,
		},
		{
			name:     "multiply mix-blend-mode forces stacking context",
			style:    ResolvedStyle{Opacity: 1, MixBlendMode: "multiply"},
			wantZSet: true,
			wantZVal: 0,
		},
		{
			name:     "isolation isolate forces stacking context",
			style:    ResolvedStyle{Opacity: 1, MixBlendMode: blendNormal, Isolation: unicodeBidiIsolate},
			wantZSet: true,
			wantZVal: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			eng := &engine{}
			style := tc.style
			target := &relayoutTarget{
				ancestors: []*box{{style: &style}},
			}

			relayoutEnterAncestorPaint(eng, target)

			if eng.zIndexSet != tc.wantZSet {
				t.Fatalf("zIndexSet = %v, want %v", eng.zIndexSet, tc.wantZSet)
			}

			if tc.wantZSet && eng.zIndex != tc.wantZVal {
				t.Fatalf("zIndex = %d, want %d", eng.zIndex, tc.wantZVal)
			}
		})
	}
}
