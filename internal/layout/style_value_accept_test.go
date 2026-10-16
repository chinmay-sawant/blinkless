package layout

import "testing"

type declarationValueCase struct {
	prop  string
	value string
	want  bool
}

func assertSupportedDeclarationCases(t *testing.T, cases []declarationValueCase) {
	t.Helper()

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

// TestSupportedDeclarationBackgroundFontGrammar (CSS-01b): the acceptance
// gate mirrors the value parser behind each property's apply arm, so an
// invalid ordinary value cannot win the cascade or satisfy @supports. This
// half of the table covers the color, background, border, font, and
// transform families.
func TestSupportedDeclarationBackgroundFontGrammar(t *testing.T) {
	t.Parallel()

	cases := []declarationValueCase{
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

	assertSupportedDeclarationCases(t, cases)
}

// TestSupportedDeclarationMaskGrammar (CSS-01b): the acceptance gate mirrors
// the value parser behind each property's apply arm, so an invalid ordinary
// value cannot win the cascade or satisfy @supports. The cases run in
// cohesive groups: the mask shorthand and layer properties, the position,
// size, and repeat tuning properties, the box and composite properties, and
// the -webkit-mask legacy aliases.
func TestSupportedDeclarationMaskGrammar(t *testing.T) {
	t.Parallel()

	runMaskShorthandLayerCases(t)
	runMaskPositionSizeRepeatCases(t)
	runMaskBoxCompositeCases(t)
	runWebkitMaskCases(t)
}

func runMaskShorthandLayerCases(t *testing.T) {
	t.Helper()

	cases := []declarationValueCase{
		{"mask", "none", true},
		{"mask", `url(mask.png) center / cover no-repeat alpha padding-box intersect`, true},
		{"mask", `url(a.png), url(b.png) right bottom / 10px 20px repeat-y`, true},
		{"mask", `url(a.png) / cover 10px`, false},
		{"mask", `url(a.png),`, false},
		{"mask", "url(a.png) space round", true},
		{"mask", "url(a.png) left 10px top", true},
		{"mask", "url(a.png) top 10px left", true},
		{"mask", "url(a.png) view-box content-box", true},
		{"mask", "url(a.png) no-clip", true},
		{"mask", "url(a.png) round", true},
		{"mask", "url(a.png) left 10px top 20px", true},
		{"mask", "url(a.png) space left top round", false},
		{"mask", "url(a.png) content-box no-clip view-box", false},
		{"mask-image", "none, url(Mask.png)", true},
		{"mask-image", "linear-gradient(red, blue)", true},
		{"mask-image", "not-an-image()", false},
		{"mask-image", "url(mask.png) junk", false},
		{"mask-mode", "match-source, alpha, luminance", true},
		{"mask-mode", "bogus", false},
	}

	assertSupportedDeclarationCases(t, cases)
}

func runMaskPositionSizeRepeatCases(t *testing.T) {
	t.Helper()

	cases := []declarationValueCase{
		{"mask-position", "left top, 10px 20%", true},
		{"mask-position", "center", true},
		{"mask-position", "10px", true},
		{"mask-position", "calc(10px + 5%)", true},
		{"mask-position", "10px top", true},
		{"mask-position", "left 10px top 20px", true},
		{"mask-position", "top 10px left 20px", true},
		{"mask-position", "left calc(10px + 5%)", true},
		{"mask-position", "left 10px top calc(10px + 5%)", true},
		{"mask-position", "left 10px top 20px, top 10px left 20px", true},
		{"mask-position", "left top right", false},
		{"mask-position", "left 10px top", false},
		{"mask-position", "left top 10px", false},
		{"mask-position", "10px left", false},
		{"mask-position", "top 10px", false},
		{"mask-position", "30% 40% 50% 60%", false},
		{"mask-position", "left top, left 10px top", false},
		{"mask-position", "left right", false},
		{"mask-position", "top bottom", false},
		{"mask-position", "left 10px top 20px 30px", false},
		{"mask-size", "auto, cover, contain, 10px 20%", true},
		{"mask-size", "-1px", false},
		{"mask-size", "cover 10px", false},
		{"mask-repeat", "repeat, repeat-x, repeat-y, no-repeat", true},
		{"mask-repeat", "round", true},
		{"mask-repeat", "space", true},
		{"mask-repeat", "space round", true},
		{"mask-repeat", "round no-repeat", true},
		{"mask-repeat", "no-repeat, space round", true},
		{"mask-repeat", "space, round", true},
		{"mask-repeat", "repeat-x, repeat-y", true},
		{"mask-repeat", "repeat-x repeat-y", false},
		{"mask-repeat", "space repeat-x", false},
		{"mask-repeat", "repeat-x space", false},
	}

	assertSupportedDeclarationCases(t, cases)
}

func runMaskBoxCompositeCases(t *testing.T) {
	t.Helper()

	cases := []declarationValueCase{
		{"mask-origin", "border-box, padding-box, content-box", true},
		{"mask-origin", "fill-box, view-box", true},
		{"mask-origin", "view-box", true},
		{"mask-origin", "stroke-box", true},
		{"mask-origin", "no-clip", false},
		{"mask-origin", "no-clip, view-box", false},
		{"mask-clip", "border-box, padding-box, content-box, no-clip", true},
		{"mask-clip", "view-box", true},
		{"mask-clip", "no-clip, view-box", true},
		{"mask-clip", "content-box, no-clip", true},
		{"mask-composite", "add, subtract, intersect, exclude", true},
		{"mask-composite", "source-over", false},
		{"mask-type", "alpha", true},
		{"mask-type", "match-source", false},
	}

	assertSupportedDeclarationCases(t, cases)
}

func runWebkitMaskCases(t *testing.T) {
	t.Helper()

	cases := []declarationValueCase{
		{"-webkit-mask", `url(mask.png) center / cover no-repeat`, true},
		{"-webkit-mask", "url(a.png) view-box content-box", false},
		{"-webkit-mask", "url(a.png) no-clip", false},
		{"-webkit-mask-clip", "no-clip", false},
		{"-webkit-mask-clip", "fill-box", false},
		{"-webkit-mask-clip", "stroke-box", false},
		{"-webkit-mask-clip", "view-box", false},
		{"-webkit-mask-composite", "source-over, source-out, source-in, xor", true},
		{"-webkit-mask-composite", "add", false},
		{"-webkit-mask-composite", "destination-out", false},
		{"-webkit-mask-image", "url(mask.png)", true},
		{"-webkit-mask-origin", "content-box", true},
		{"-webkit-mask-origin", "view-box", false},
		{"-webkit-mask-position", "right bottom", true},
		{"-webkit-mask-position", "left 10px top 20px", true},
		{"-webkit-mask-repeat", "repeat-x", true},
		{"-webkit-mask-repeat", "space round", true},
		{"-webkit-mask-size", "contain", true},
	}

	assertSupportedDeclarationCases(t, cases)
}

// TestSupportedDeclarationFlexGridGeneratedGrammar (CSS-01b): the second half
// of the table covers flex-flow, grid placement, counters, quotes, and
// hyphenate-character, reusing the same parsers as their apply arms.
func TestSupportedDeclarationFlexGridGeneratedGrammar(t *testing.T) {
	t.Parallel()

	cases := []declarationValueCase{
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

	assertSupportedDeclarationCases(t, cases)
}

// TestSupportedDeclarationGridTemplateGrammar (CSS-01b): this table covers the
// grid and grid-template shorthands, the last two advertised implemented
// properties without an acceptance arm. The grammar mirrors
// parseGridTemplateShorthand and parseGridShorthand: none, masonry, the areas
// form, track lists with line names / minmax() / repeat(), and the grid
// auto-flow form. Invalid ordinary values must not win the cascade.
func TestSupportedDeclarationGridTemplateGrammar(t *testing.T) {
	t.Parallel()

	cases := []struct {
		prop  string
		value string
		want  bool
	}{
		// grid-template: none, track lists, rows / columns, and areas.
		{"grid-template", "none", true},
		{"grid-template", "100px 1fr / 50px 2fr", true},
		{"grid-template", "auto / 1fr 1fr", true},
		{"grid-template", `"a a" 40px "b c" 1fr / 1fr 2fr`, true},
		{"grid-template", `"a" "b"`, true},
		{"grid-template", "masonry / 1fr", true},
		{"grid-template", "minmax(0, 1.2fr) minmax(0, .8fr)", true},
		{"grid-template", "repeat(3, minmax(0, 1fr))", true},
		{"grid-template", "repeat(auto-fit, minmax(130px, 1fr))", true},
		{"grid-template", "[main] 1fr [side] 2fr", true},
		{"grid-template", "40% 1fr", true},
		{"grid-template", "none / 1fr", true},
		{"grid-template", "bogus", false},
		{"grid-template", "1fr bogus", false},
		{"grid-template", "1fr / bogus", false},
		{"grid-template", "1fr /", false},
		{"grid-template", `40px "a"`, false},
		{"grid-template", `"a" bogus`, false},
		{"grid-template", "minmax(1fr)", false},
		{"grid-template", "minmax(1fr, bogus)", false},
		{"grid-template", "repeat(0, 1fr)", false},
		{"grid-template", "repeat(2, bogus)", false},
		{"grid-template", "-10px 1fr", false},

		// grid: template form, auto-flow form, none, and masonry.
		{"grid", "none", true},
		{"grid", "80px / 1fr 2fr", true},
		{"grid", "auto-flow dense / 1fr 2fr", true},
		{"grid", "100px / auto-flow dense", true},
		{"grid", "auto-flow 100px / 1fr", true},
		{"grid", "masonry", true},
		{"grid", "bogus", false},
		{"grid", "auto-flow", false},
		{"grid", "auto-flow bogus / 1fr", false},
		{"grid", "100px / auto-flow bogus", false},
		{"grid", "auto-flow / auto-flow", false},
		{"grid", "100px /", false},
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
