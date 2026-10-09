package layout

import "testing"

// TestSupportedDeclarationBackgroundFontGrammar (CSS-01b): the acceptance
// gate mirrors the value parser behind each property's apply arm, so an
// invalid ordinary value cannot win the cascade or satisfy @supports. This
// half of the table covers the color, background, border, font, and
// transform families.
func TestSupportedDeclarationBackgroundFontGrammar(t *testing.T) {
	t.Parallel()

	cases := []struct {
		prop  string
		value string
		want  bool
	}{
		// CSS-wide keywords and var() are universal.
		{"background", "inherit", true},
		{"font", "var(--f)", true},

		// background shorthand: a readable color or image must be present.
		{"background", "red", true},
		{"background", "url(x.png) no-repeat center", true},
		{"background", "none", true},
		{"background", "no-repeat center", false},
		{"background", "bogus", false},

		// background-image: none, url(), or a gradient per comma layer.
		{"background-image", "none", true},
		{"background-image", "url(x.png)", true},
		{"background-image", "linear-gradient(red, blue)", true},
		{"background-image", "url(a.png), url(b.png)", true},
		{"background-image", "url(a.png), bogus", false},
		{"background-image", "bogus", false},

		// font shorthand: prefixes, then a readable size with optional
		// line-height; system font keywords are rejected.
		{"font", "12pt serif", true},
		{"font", "italic bold 14pt/20pt serif", true},
		{"font", "16pt/1.5 Helvetica, Arial, sans-serif", true},
		{"font", "caption", false},
		{"font", "bogus", false},
		{"font", "bold", false},

		// Existing priority families.
		{"color", "currentColor", true},
		{"color", "bogus", false},
		{"border", "1px solid red", true},
		{"border", "bogus", false},
		{"font-size", "12pt", true},
		{"font-size", "bogus", false},
		{"transform", "rotate(10deg)", true},
		{"transform", "bogus", false},
	}

	for _, testCase := range cases {
		t.Run(testCase.prop+"/"+testCase.value, func(t *testing.T) {
			t.Parallel()

			if got := supportedDeclaration(testCase.prop, testCase.value); got != testCase.want {
				t.Errorf("supportedDeclaration(%q, %q) = %v, want %v",
					testCase.prop, testCase.value, got, testCase.want)
			}
		})
	}
}

// TestSupportedDeclarationFlexGridGeneratedGrammar (CSS-01b): the second half
// of the table covers flex-flow, grid placement, counters, quotes, and
// hyphenate-character, reusing the same parsers as their apply arms.
func TestSupportedDeclarationFlexGridGeneratedGrammar(t *testing.T) {
	t.Parallel()

	cases := []struct {
		prop  string
		value string
		want  bool
	}{
		// flex-flow: direction and wrap keywords in any order.
		{"flex-flow", "row wrap", true},
		{"flex-flow", "wrap column-reverse", true},
		{"flex-flow", "row bogus", false},
		{"flex-flow", "bogus", false},

		// grid placement: the engine's recognized grid-line forms.
		{"grid-row", "span 2", true},
		{"grid-row", "2 / span 3", true},
		{"grid-row", "1/3", true},
		{"grid-row", "auto", true},
		{"grid-row", "span bogus", false},
		{"grid-row", "bogus", false},
		{"grid-row-start", "3", true},
		{"grid-row-start", "auto", true},
		{"grid-row-start", "bogus", false},
		{"grid-row-end", "auto", true},
		{"grid-row-end", "span 2", true},
		{"grid-row-end", "bogus", false},

		// counters: none or identifier/integer pairs.
		{"counter-reset", "section", true},
		{"counter-reset", "audit-section req-clause", true},
		{"counter-reset", "section 0", true},
		{"counter-reset", "none", true},
		{"counter-reset", "section none", false},
		{"counter-reset", "5 section", false},
		{"counter-set", "item 5", true},
		{"counter-set", "5", false},
		{"counter-increment", "none", true},
		{"counter-increment", "bogus()", false},

		// quotes: none or open/close string pairs.
		{"quotes", `"a" "b"`, true},
		{"quotes", `"a" "b" "c" "d"`, true},
		{"quotes", "none", true},
		{"quotes", `"a"`, false},
		{"quotes", "auto", false},

		// hyphenate-character: auto or one string token.
		{"hyphenate-character", "auto", true},
		{"hyphenate-character", "'~'", true},
		{"hyphenate-character", "two tokens", false},

		// Existing flex shorthand.
		{"flex", "1 1 auto", true},
		{"flex", "bogus", false},
	}

	for _, testCase := range cases {
		t.Run(testCase.prop+"/"+testCase.value, func(t *testing.T) {
			t.Parallel()

			if got := supportedDeclaration(testCase.prop, testCase.value); got != testCase.want {
				t.Errorf("supportedDeclaration(%q, %q) = %v, want %v",
					testCase.prop, testCase.value, got, testCase.want)
			}
		})
	}
}
