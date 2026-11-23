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

func TestNestedMediaAllAndScreenMatchesScreenOnly(t *testing.T) {
	t.Parallel()

	sheet, err := Parse(`@media all and (min-width: 400px) { @media screen and (max-width: 800px) { p { color: red } } }`)
	if err != nil {
		t.Fatal(err)
	}

	if len(sheet.Rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(sheet.Rules))
	}

	q := sheet.Rules[0].Media
	// 400px = 300pt, 800px = 600pt at 96dpi (72/96 pt/px).
	if !MediaMatches(q, "screen", 450, 800) {
		t.Errorf("media %q did not match screen at width 450pt (600px)", q)
	}

	if MediaMatches(q, "screen", 200, 800) {
		t.Errorf("media %q matched screen at width 200pt (below min-width 400px = 300pt)", q)
	}

	if MediaMatches(q, "screen", 700, 800) {
		t.Errorf("media %q matched screen at width 700pt (above max-width 800px = 600pt)", q)
	}

	if MediaMatches(q, "print", 450, 800) {
		t.Errorf("media %q matched print, want screen-only", q)
	}
}

func TestNestedMediaCommaCartesianProductCapped(t *testing.T) {
	t.Parallel()

	// 6 levels of 4-way comma lists would produce 4^6 = 4096 combinations without a cap.
	src := `p { color: red }`
	for range 6 {
		src = `@media (min-width: 100px), (min-width: 200px), (min-width: 300px), (min-width: 400px) { ` + src + ` }`
	}

	sheet, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	if len(sheet.Rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(sheet.Rules))
	}

	parts := splitMediaList(sheet.Rules[0].Media)
	if len(parts) > maxCombinedMediaQueries {
		t.Fatalf("got %d combined media queries, want <= %d", len(parts), maxCombinedMediaQueries)
	}

	if !MediaMatches(sheet.Rules[0].Media, "screen", 500, 800) {
		t.Errorf("capped media query %q should match at width 500", sheet.Rules[0].Media)
	}
}
