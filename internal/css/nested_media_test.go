package css

import "testing"

func TestNestedMediaCombinesQueries(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name, src string
		match     map[float64]bool // viewport width in points -> rule applies
	}{
		{
			"inside layer",
			`@layer base { @media (min-width: 600px) { p { color: red } } }`,
			map[float64]bool{300: false, 600: true, 900: true},
		},
		{
			"media in media",
			`@media screen { @media (max-width: 800px) { p { color: red } } }`,
			map[float64]bool{300: true, 600: true, 900: false},
		},
		{
			"media in supports",
			`@supports (display: grid) { @media (min-width: 600px) { p { color: red } } }`,
			map[float64]bool{300: false, 700: true},
		},
		{
			"conflicting types never match",
			`@media print { @media screen { p { color: red } } }`,
			map[float64]bool{300: false, 700: false},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sheet, err := Parse(tc.src)
			if err != nil {
				t.Fatal(err)
			}

			var found bool

			for _, rule := range sheet.Rules {
				if len(rule.Selectors) == 0 {
					continue
				}

				found = true

				for width, want := range tc.match {
					if got := MediaMatches(rule.Media, "screen", width, 800); got != want {
						t.Errorf("width %v: media %q matches=%v, want %v", width, rule.Media, got, want)
					}
				}
			}

			if !found && tc.name != "conflicting types never match" {
				t.Fatal("nested @media produced no rule")
			}
		})
	}
}
