package css_test

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
	"github.com/chinmay-sawant/blinkless/internal/layout"
)

// Property names and values repeated across the @supports fixtures.
const (
	displayProp = "display"
	colorProp   = "color"
	blockValue  = "block"
)

func parseSheet(t *testing.T, src string) *css.Stylesheet {
	t.Helper()

	s, err := css.Parse(src)
	if err != nil {
		t.Fatalf("css.Parse(%q): %v", src, err)
	}

	return s
}

func resolveStyles(t *testing.T, src string, sheets ...*css.Stylesheet,
) (*html.Node, map[*html.Node]*layout.ResolvedStyle) {
	t.Helper()

	root, err := html.Parse(src)
	if err != nil {
		t.Fatalf("html.Parse(%q): %v", src, err)
	}

	styles, err := layout.ResolveStyles(t.Context(), root, layout.Options{
		Width: 600, Height: 800, Sheets: sheets,
	})
	if err != nil {
		t.Fatalf("layout.ResolveStyles: %v", err)
	}

	return root, styles
}

func findNode(n *html.Node, name string) *html.Node {
	if n.Name == name {
		return n
	}

	for _, child := range n.Children {
		if found := findNode(child, name); found != nil {
			return found
		}
	}

	return nil
}

// testColors holds the primary colors the layout tests compare against.
type testColors struct {
	red   [3]float64
	blue  [3]float64
	green [3]float64
	black [3]float64
}

// testPalette returns freshly initialized colors so parallel tests share no
// mutable global state.
func testPalette() testColors {
	return testColors{
		red:   [3]float64{1, 0, 0},
		blue:  [3]float64{0, 0, 1},
		green: [3]float64{0, 1, 0},
		black: [3]float64{0, 0, 0},
	}
}

func colorOf(t *testing.T, root *html.Node, styles map[*html.Node]*layout.ResolvedStyle, tag string) [3]float64 {
	t.Helper()

	node := findNode(root, tag)
	if node == nil {
		t.Fatalf("no <%s> node", tag)
	}

	style := styles[node]
	if style == nil {
		t.Fatalf("no resolved style for <%s>", tag)
	}

	return style.Color
}

func supportsCond(t *testing.T, cond string) *css.SupportsCondition {
	t.Helper()

	sheet := parseSheet(t, "@supports "+cond+" { p { color: #f00 } }")
	if len(sheet.Rules) != 1 {
		t.Fatalf("@supports %s: got %d rules, want 1", cond, len(sheet.Rules))
	}

	if sheet.Rules[0].Supports == nil {
		t.Fatalf("@supports %s: nil Supports condition", cond)
	}

	return sheet.Rules[0].Supports
}

// supportsGate parses one @supports block that gates a single style rule and
// returns the gate, or nil when the prelude is invalid and the block was
// dropped.
func supportsGate(t *testing.T, cond string) *css.SupportsCondition {
	t.Helper()

	sheet := parseSheet(t, "@supports "+cond+" { p { color: #f00 } }")
	if len(sheet.Rules) == 0 {
		return nil
	}

	if len(sheet.Rules) != 1 || sheet.Rules[0].Supports == nil {
		t.Fatalf("@supports %s: got %d rules, want 1 gated rule", cond, len(sheet.Rules))
	}

	return sheet.Rules[0].Supports
}

func TestSupportsConditionMatching(t *testing.T) {
	t.Parallel()

	supported := func(prop, _ string) bool {
		return prop == displayProp || prop == colorProp
	}

	cases := []struct {
		cond string
		want bool
	}{
		{"(display: grid)", true},
		{"(unknown-prop: 1)", false},
		{"(display: grid) and (color: #f00)", true},
		{"(display: grid) and (unknown-prop: 1)", false},
		{"(unknown-prop: 1) or (color: #f00)", true},
		{"not (unknown-prop: 1)", true},
		{"not (display: grid)", false},
		{"((display: grid) and (color: #f00)) or (unknown-prop: 1)", true},
		{"(display: grid) and ((color: #f00) or (unknown-prop: 1))", true},
		{"(display grid)", false},
		{"selector(p)", false},
	}

	for _, tc := range cases {
		if got := css.SupportsMatches(supportsCond(t, tc.cond), supported); got != tc.want {
			t.Errorf("SupportsMatches(%q) = %v, want %v", tc.cond, got, tc.want)
		}
	}
}

func TestSupportsInvalidPreludeDropsBlock(t *testing.T) {
	t.Parallel()

	sheet := parseSheet(t, "@supports display: grid { p { color: #f00 } }")

	if len(sheet.Rules) != 0 {
		t.Fatalf("invalid @supports kept %d rules, want 0", len(sheet.Rules))
	}
}

// TestSupportsDeclarationValueMatching covers positive and negative
// property/value cases with a callback that mirrors the intended engine
// contract: display:block is supported, while display:bogus and unknown
// properties are not.
func TestSupportsDeclarationValueMatching(t *testing.T) {
	t.Parallel()

	supported := func(prop, value string) bool {
		return prop == displayProp && value == blockValue
	}

	cases := []struct {
		cond string
		want bool
	}{
		{"(display: block)", true},
		{"(display:block)", true},
		{"(display:\tblock)", true},
		{"(display:\nblock)", true},
		{"( display : block )", true},
		{"(Display: block)", true},
		{"(display: bogus)", false},
		{"(display:grid)", false},
		{"(unknown-prop: 1)", false},
		{"(display: block) and (display: block)", true},
		{"(display: block) and (display: bogus)", false},
		{"(display: bogus) or (display: block)", true},
		{"(display: bogus) or (unknown-prop: 1)", false},
		{"not (display: bogus)", true},
		{"not (display: block)", false},
		{"not (unknown-prop: 1)", true},
		{"((display: block) and (not (display: bogus)))", true},
		{"((display: bogus) or (display: block)) and (not (unknown-prop: 1))", true},
	}

	for _, tc := range cases {
		if got := css.SupportsMatches(supportsCond(t, tc.cond), supported); got != tc.want {
			t.Errorf("SupportsMatches(%q) = %v, want %v", tc.cond, got, tc.want)
		}
	}
}

// TestSupportsOperatorsCaseInsensitive checks that and, or, and not match
// regardless of ASCII case.
func TestSupportsOperatorsCaseInsensitive(t *testing.T) {
	t.Parallel()

	supported := func(prop, value string) bool {
		return prop == displayProp && value == blockValue
	}

	cases := []struct {
		cond string
		want bool
	}{
		{"(display: block) AnD (display: block)", true},
		{"(display: block) aNd (display: bogus)", false},
		{"(display: bogus) OR (display: block)", true},
		{"(display: bogus) oR (display: bogus)", false},
		{"NOT (display: bogus)", true},
		{"nOt (display: block)", false},
	}

	for _, tc := range cases {
		if got := css.SupportsMatches(supportsCond(t, tc.cond), supported); got != tc.want {
			t.Errorf("SupportsMatches(%q) = %v, want %v", tc.cond, got, tc.want)
		}
	}
}

// TestSupportsMixedOperatorsRejectBlock checks that preludes mixing and/or at
// one level are invalid and drop their block.
func TestSupportsMixedOperatorsRejectBlock(t *testing.T) {
	t.Parallel()

	invalid := []string{
		"(display: block) and (display: block) or (display: bogus)",
		"(display: block) or (display: block) and (display: bogus)",
		"not (display: block) and (display: block)",
		"not (display: block) or (display: block)",
		"(display: block) and not (display: bogus)",
	}

	for _, cond := range invalid {
		if gate := supportsGate(t, cond); gate != nil {
			t.Errorf("@supports %q: gate = %+v, want dropped block", cond, gate)
		}
	}
}

// TestSupportsGeneralEnclosedEvaluatesFalse checks that features the engine
// cannot evaluate keep the block but gate it false, and still combine with
// boolean operators.
func TestSupportsGeneralEnclosedEvaluatesFalse(t *testing.T) {
	t.Parallel()

	supported := func(prop, value string) bool {
		return prop == displayProp && value == blockValue
	}

	cases := []struct {
		cond string
		want bool
	}{
		{"selector(p)", false},
		{"font-tech(color-COLRv1)", false},
		{"(display grid)", false},
		{"(unknown-func(1))", false},
		{"(display: block) or selector(p)", true},
		{"(display: bogus) and selector(p)", false},
		{"not selector(p)", true},
	}

	for _, tc := range cases {
		if got := css.SupportsMatches(supportsCond(t, tc.cond), supported); got != tc.want {
			t.Errorf("SupportsMatches(%q) = %v, want %v", tc.cond, got, tc.want)
		}
	}
}

// TestSupportsEmptyParensEvaluatesFalse checks that empty parens stay valid
// general-enclosed syntax and evaluate false.
func TestSupportsEmptyParensEvaluatesFalse(t *testing.T) {
	t.Parallel()

	supported := func(string, string) bool { return true }

	for _, cond := range []string{"()", "( )", "(\t)"} {
		gate := supportsGate(t, cond)
		if gate == nil {
			t.Fatalf("@supports %q: block dropped, want kept", cond)
		}

		if css.SupportsMatches(gate, supported) {
			t.Errorf("@supports %q: matched, want false", cond)
		}
	}
}

// TestSupportsUnbalancedParensRejectBlock checks that an unbalanced prelude is
// invalid and drops its block.
func TestSupportsUnbalancedParensRejectBlock(t *testing.T) {
	t.Parallel()

	invalid := []string{
		"(display: block",
		"((display: block)",
		"(display: block))",
		"((display: block)))",
	}

	for _, cond := range invalid {
		if gate := supportsGate(t, cond); gate != nil {
			t.Errorf("@supports %q: gate = %+v, want dropped block", cond, gate)
		}
	}
}

// TestSupportsDeclarationWithoutValue covers an empty or missing declaration
// value: the prelude stays valid (WPT css-supports-022) and the feature
// evaluates false because no empty value is supported.
func TestSupportsDeclarationWithoutValue(t *testing.T) {
	t.Parallel()

	supported := func(prop, value string) bool {
		return prop == displayProp && value == blockValue
	}

	for _, cond := range []string{"(display:)", "(display: )", "(display)"} {
		gate := supportsGate(t, cond)
		if gate == nil {
			t.Fatalf("@supports %q: block dropped, want kept", cond)
		}

		if css.SupportsMatches(gate, supported) {
			t.Errorf("@supports %q: matched, want false", cond)
		}
	}
}

// TestSupportsOperatorWithoutSpaceIsFunctionToken covers not(, and(, and or(:
// the parenthesis binds to the keyword, so the feature is general-enclosed and
// must not act as an operator (WPT at-supports-014 and at-supports-043). A
// space before the keyword is fine because ')' already ended the previous
// token.
func TestSupportsOperatorWithoutSpaceIsFunctionToken(t *testing.T) {
	t.Parallel()

	supported := func(prop, value string) bool {
		return prop == displayProp && value == blockValue
	}

	gate := supportsGate(t, "not(display: bogus)")
	if gate == nil {
		t.Fatal("@supports not(display: bogus): block dropped, want kept and false")
	}

	if css.SupportsMatches(gate, supported) {
		t.Fatal("@supports not(display: bogus): matched, want false")
	}

	invalid := []string{
		"(display: block)and(display: block)",
		"(display: block)or(display: block)",
		"(display: block) and(display: block)",
		"(display: block) or(display: block)",
	}

	for _, cond := range invalid {
		if gate := supportsGate(t, cond); gate != nil {
			t.Errorf("@supports %q: gate = %+v, want dropped block", cond, gate)
		}
	}

	valid := []string{
		"(display: block)and (display: block)",
		"(display: block)or (display: block)",
	}

	for _, cond := range valid {
		if !css.SupportsMatches(supportsCond(t, cond), supported) {
			t.Errorf("SupportsMatches(%q) = false, want true", cond)
		}
	}
}

// TestSupportsNestedValueConditions covers nested @supports blocks where the
// inner value decides support.
func TestSupportsNestedValueConditions(t *testing.T) {
	t.Parallel()

	supported := func(prop, value string) bool {
		return prop == displayProp && value == blockValue
	}

	cases := []struct {
		name  string
		sheet string
		want  bool
	}{
		{
			name: "inner bogus value gates false",
			sheet: `
				@supports (display: block) {
					@supports (display: bogus) {
						p { color: #f00 }
					}
				}`,
			want: false,
		},
		{
			name: "supported value under case-insensitive not",
			sheet: `
				@supports (display: block) {
					@supports (NOT (display: bogus)) {
						p { color: #f00 }
					}
				}`,
			want: true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			sheet := parseSheet(t, testCase.sheet)
			if len(sheet.Rules) != 1 {
				t.Fatalf("nested @supports: got %d rules, want 1", len(sheet.Rules))
			}

			cond := sheet.Rules[0].Supports
			if cond == nil {
				t.Fatal("nested @supports: nil gate")
			}

			if got := css.SupportsMatches(cond, supported); got != testCase.want {
				t.Fatalf("nested @supports matched = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestSupportsNestedConditionCombines(t *testing.T) {
	t.Parallel()

	sheet := parseSheet(t, `
		@supports (display: grid) {
			@supports (color: #f00) {
				p { color: #f00 }
			}
		}`)
	if len(sheet.Rules) != 1 {
		t.Fatalf("nested @supports: got %d rules, want 1", len(sheet.Rules))
	}

	cond := sheet.Rules[0].Supports
	supported := func(prop, _ string) bool { return prop == displayProp }

	if css.SupportsMatches(cond, supported) {
		t.Fatal("nested @supports matched with color unsupported, want false")
	}

	supported = func(prop, _ string) bool { return prop == displayProp || prop == colorProp }

	if !css.SupportsMatches(cond, supported) {
		t.Fatal("nested @supports did not match with both supported, want true")
	}
}

func TestSupportsGatesRulesInLayout(t *testing.T) {
	t.Parallel()

	colors := testPalette()
	sheet := parseSheet(t, `
		@supports (display: grid) { p { color: #f00 } }
		@supports (unknown-prop: 1) { h1 { color: #f00 } }`)

	root, styles := resolveStyles(t, "<p>x</p><h1>y</h1>", sheet)

	if got := colorOf(t, root, styles, "p"); got != colors.red {
		t.Fatalf("supported @supports rule: p color = %v, want red", got)
	}

	if got := colorOf(t, root, styles, "h1"); got != colors.black {
		t.Fatalf("unsupported @supports rule: h1 color = %v, want black", got)
	}
}

func TestSupportsInsideMedia(t *testing.T) {
	t.Parallel()

	colors := testPalette()
	sheet := parseSheet(t, `
		@media all {
			@supports (unknown-prop: 1) { p { color: #f00 } }
			@supports (display: grid) { h1 { color: #f00 } }
		}`)

	root, styles := resolveStyles(t, "<p>x</p><h1>y</h1>", sheet)

	if got := colorOf(t, root, styles, "p"); got != colors.black {
		t.Fatalf("unsupported nested @supports: p color = %v, want black", got)
	}

	if got := colorOf(t, root, styles, "h1"); got != colors.red {
		t.Fatalf("supported nested @supports: h1 color = %v, want red", got)
	}
}

func TestLayerRanksAndOrder(t *testing.T) {
	t.Parallel()

	sheet := parseSheet(t, `
		@layer base, theme;
		@layer theme { p { color: #00f } }
		@layer base { p { color: #f00 } }
		p { color: #0f0 }`)

	ranks := map[string]int{}

	for _, r := range sheet.Rules {
		if len(r.Decls) == 1 && r.Decls[0].Prop == colorProp {
			ranks[r.Decls[0].Value] = r.Layer
		}
	}

	if ranks["#f00"] != 1 || ranks["#00f"] != 2 || ranks["#0f0"] != 0 {
		t.Fatalf("layer ranks = %v, want base 1, theme 2, unlayered 0", ranks)
	}

	if len(sheet.Layers) != 2 || sheet.Layers[0] != "base" || sheet.Layers[1] != "theme" {
		t.Fatalf("Layers = %v, want [base theme]", sheet.Layers)
	}
}

func TestLayerCascadeOrder(t *testing.T) {
	t.Parallel()

	colors := testPalette()

	cases := []struct {
		name  string
		sheet string
		want  [3]float64
	}{
		{
			name:  "unlayered beats layered",
			sheet: `@layer a { p { color: #00f } } p { color: #0f0 }`,
			want:  colors.green,
		},
		{
			name:  "later layer beats earlier",
			sheet: `@layer a, b; @layer a { p { color: #f00 } } @layer b { p { color: #00f } }`,
			want:  colors.blue,
		},
		{
			name:  "anonymous layers follow order",
			sheet: `@layer { p { color: #f00 } } @layer { p { color: #00f } }`,
			want:  colors.blue,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root, styles := resolveStyles(t, "<p>x</p>", parseSheet(t, testCase.sheet))

			if got := colorOf(t, root, styles, "p"); got != testCase.want {
				t.Fatalf("color = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestPropertyRegistrationParse(t *testing.T) {
	t.Parallel()

	sheet := parseSheet(t, `
		@property --brand {
			syntax: "<color>";
			initial-value: #f00;
			inherits: false;
		}`)

	if len(sheet.Properties) != 1 {
		t.Fatalf("got %d @property registrations, want 1", len(sheet.Properties))
	}

	prop := sheet.Properties[0]
	if prop.Name != "--brand" || prop.Syntax != "<color>" || prop.Initial != "#f00" {
		t.Fatalf("@property = %+v, want --brand <color> #f00", prop)
	}

	if prop.Inherits || !prop.InheritsSet {
		t.Fatalf("inherits = %v set = %v, want false set", prop.Inherits, prop.InheritsSet)
	}
}

func TestPropertyInitialValueUsedByVar(t *testing.T) {
	t.Parallel()

	colors := testPalette()
	sheet := parseSheet(t, `
		@property --brand { syntax: "<color>"; initial-value: #f00; inherits: false; }
		p { color: var(--brand) }`)

	root, styles := resolveStyles(t, "<p>x</p>", sheet)

	if got := colorOf(t, root, styles, "p"); got != colors.red {
		t.Fatalf("var(--brand) color = %v, want initial red", got)
	}
}

func TestPropertyVarFallbackWithoutInitial(t *testing.T) {
	t.Parallel()

	colors := testPalette()
	sheet := parseSheet(t, `
		@property --brand { syntax: "*"; inherits: false; }
		p { color: var(--brand, #0f0) }`)

	root, styles := resolveStyles(t, "<p>x</p>", sheet)

	if got := colorOf(t, root, styles, "p"); got != colors.green {
		t.Fatalf("var(--brand, #0f0) color = %v, want fallback green", got)
	}
}

func TestPropertyInheritsFlag(t *testing.T) {
	t.Parallel()

	colors := testPalette()

	cases := []struct {
		name     string
		inherits string
		want     [3]float64
	}{
		{name: "inherits true", inherits: "true", want: colors.blue},
		{name: "inherits false", inherits: "false", want: colors.red},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			sheet := parseSheet(t, `
				@property --brand { syntax: "<color>"; initial-value: #f00; inherits: `+testCase.inherits+`; }
				div { --brand: #00f }
				p { color: var(--brand) }`)

			root, styles := resolveStyles(t, "<div><p>x</p></div>", sheet)

			if got := colorOf(t, root, styles, "p"); got != testCase.want {
				t.Fatalf("child color = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestUnknownAtRulesStillSkipped(t *testing.T) {
	t.Parallel()

	sheet := parseSheet(t, "@unknown foo { p { color: #f00 } } p { color: #0f0 }")

	if len(sheet.Rules) != 1 || sheet.Rules[0].Decls[0].Value != "#0f0" {
		t.Fatalf("unknown at-rule changed rule list: %+v", sheet.Rules)
	}
}
