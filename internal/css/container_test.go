package css

import "testing"

const cardName = "card"

func TestParseContainerShorthand(t *testing.T) {
	t.Parallel()

	name, ctype := ParseContainerShorthand("card / inline-size")
	if name != cardName || ctype != "inline-size" {
		t.Fatalf("got name=%q type=%q", name, ctype)
	}

	name, ctype = ParseContainerShorthand("sidebar / size")
	if name != "sidebar" || ctype != "size" {
		t.Fatalf("got name=%q type=%q", name, ctype)
	}

	name, ctype = ParseContainerShorthand("none")
	if name != "" || ctype != "" {
		t.Fatalf("none: name=%q type=%q", name, ctype)
	}

	name, ctype = ParseContainerShorthand("a b / normal")
	if name != "a b" || ctype != "normal" {
		t.Fatalf("multi name: name=%q type=%q", name, ctype)
	}
}

func TestParseContainerNameValue(t *testing.T) {
	t.Parallel()

	if ParseContainerNameValue("none") != "" {
		t.Fatal("none should clear")
	}

	if ParseContainerNameValue("Card") != cardName {
		t.Fatalf("got %q", ParseContainerNameValue("Card"))
	}
}

func TestParseContainerRules(t *testing.T) { //nolint:cyclop,funlen // per-variant structural checks
	t.Parallel()
	sty := mustSheet(t, `
		.card { container: card / inline-size; width: 400px }
		@container card (inline-size > 20em) {
			.title { font-size: 2em }
		}
		@container (width > 300px) {
			.wide { color: red }
		}
		@container card (min-width: 10em) and (max-width: 1000px) {
			.mid { color: blue }
		}
		@container not (inline-size < 5em) {
			.ok { display: block }
		}
		@container card (inline-size > 10em) or (width < 1px) {
			.or { color: green }
		}
	`)
	// 1 normal + 5 container rules
	if len(sty.Rules) != 6 {
		t.Fatalf("rules = %d: %+v", len(sty.Rules), sty.Rules)
	}

	if sty.Rules[0].Container != nil {
		t.Fatal("first rule should not be container-conditional")
	}

	r1Val := sty.Rules[1]
	if r1Val.Container == nil || r1Val.Container.Name != cardName {
		t.Fatalf("rule1 container = %+v", r1Val.Container)
	}

	if r1Val.Container.Cond.Kind != "feat" || r1Val.Container.Cond.Feat == nil {
		t.Fatalf("rule1 cond = %+v", r1Val.Container.Cond)
	}

	if r1Val.Container.Cond.Feat.Name != "inline-size" || r1Val.Container.Cond.Feat.Op != ">" {
		t.Fatalf("feat = %+v", r1Val.Container.Cond.Feat)
	}

	if len(r1Val.Selectors) != 1 || r1Val.Selectors[0].Parts[0].Classes[0] != "title" {
		t.Fatalf("selectors = %+v", r1Val.Selectors)
	}

	r2 := sty.Rules[2]
	if r2.Container == nil || r2.Container.Name != "" {
		t.Fatalf("unnamed container = %+v", r2.Container)
	}

	r3 := sty.Rules[3]
	if r3.Container == nil || r3.Container.Cond.Kind != "and" || len(r3.Container.Cond.Kids) != 2 {
		t.Fatalf("and cond = %+v", r3.Container)
	}

	r4 := sty.Rules[4]
	if r4.Container == nil || r4.Container.Cond.Kind != "not" {
		t.Fatalf("not cond = %+v", r4.Container)
	}

	r5 := sty.Rules[5]
	if r5.Container == nil || r5.Container.Cond.Kind != "or" {
		t.Fatalf("or cond = %+v", r5.Container)
	}
}

func TestContainerCondMatches(t *testing.T) {
	t.Parallel()

	cases := []struct {
		prelude string
		size    float64
		want    bool
	}{
		{"card (inline-size > 20em)", 241, true},                // 20em at 12pt = 240pt
		{"card (inline-size > 20em)", 239, false},               // 20em at 12pt = 240pt
		{"(min-width: 100px)", 75, true},                        // 100px = 75pt
		{"(min-width: 100px)", 74, false},                       // 100px = 75pt
		{"(width > 50pt) and (inline-size < 200pt)", 100, true}, // mid-range matches both
		{"(width > 50pt) and (inline-size < 200pt)", 40, false}, // low fails the > 50pt arm
		{"not (inline-size < 10pt)", 10, true},                  // == 10 is not < 10
		{"not (inline-size < 10pt)", 9, false},                  // 9 is < 10
		{"(width < 1pt) or (inline-size > 5pt)", 6, true},       // second arm matches
	}
	for _, testCase := range cases {
		cq, found := parseContainerPrelude(testCase.prelude)
		if !found {
			t.Fatalf("parse prelude %q", testCase.prelude)
		}

		if got := cq.Cond.Matches(testCase.size, 12); got != testCase.want {
			t.Errorf("Matches(%q, %g) = %v, want %v", testCase.prelude, testCase.size, got, testCase.want)
		}
	}
}

// TestSplitCondKeywordTokenBoundary pins the token boundary rule for and/or
// from the CSS Conditional 3 changes note: whitespace is not required before
// the keyword, but an opening parenthesis directly after it makes a function
// token, so `and(` and `or(` are not operators. @supports applies the same
// rule in supportsBoundary.
func TestSplitCondKeywordTokenBoundary(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cond string
		kw   string
		want int
	}{
		{name: "spaced and", cond: "(width > 10px) and (height > 10px)", kw: condKindAnd, want: 2},
		{name: "space before keyword is optional", cond: "(width > 10px)and (height > 10px)", kw: condKindAnd, want: 2},
		{name: "three ands", cond: "(a) and (b) and (c)", kw: condKindAnd, want: 3},
		{name: "and function token", cond: "(width > 10px) and(height > 10px)", kw: condKindAnd, want: 1},
		{name: "or function token", cond: "(width > 10px) or(height > 10px)", kw: condKindOr, want: 1},
		{name: "uppercase keyword", cond: "(width > 10px) AND (height > 10px)", kw: condKindAnd, want: 2},
		{
			name: "nested and stays intact",
			cond: "((width > 10px) and (height > 10px)) or (width < 1px)",
			kw:   condKindAnd,
			want: 1,
		},
		{name: "outer or splits", cond: "((width > 10px) and (height > 10px)) or (width < 1px)", kw: condKindOr, want: 2},
		{name: "keyword inside ident", cond: "(width > 10px) brandy (height > 10px)", kw: condKindAnd, want: 1},
		{name: "quoted keyword", cond: `"a and b" and (width > 10px)`, kw: condKindAnd, want: 2},
	}

	for _, testCase := range cases {
		parts, ok := splitCondKeyword(testCase.cond, testCase.kw)
		if !ok {
			t.Fatalf("%s: split failed for %q", testCase.name, testCase.cond)
		}

		if len(parts) != testCase.want {
			t.Fatalf("%s: parts=%q, want %d", testCase.name, parts, testCase.want)
		}
	}

	if _, ok := splitCondKeyword("and (width > 10px)", condKindAnd); ok {
		t.Fatal("leading keyword must fail (empty first part)")
	}

	if _, ok := splitCondKeyword("(width > 10px) and", condKindAnd); ok {
		t.Fatal("trailing keyword must fail (empty last part)")
	}
}

// TestContainerCondFunctionTokensRejected covers the CSS Conditional 5
// @container grammar: `not(...)`, `and(...)`, `or(...)`, and `style(...)`
// written without a space are function tokens, not operators or size
// conditions. The engine rejects the unsupported function forms; a browser
// evaluates them as unknown, which also never applies the rule.
func TestContainerCondFunctionTokensRejected(t *testing.T) {
	t.Parallel()

	preludes := []string{
		"(width > 10px) and(inline-size > 20px)",
		"(width > 10px) or(inline-size > 20px)",
		"(width > 10px) AND(inline-size > 20px)",
		"(width > 10px) and((inline-size > 20px))",
		"not(width > 10px)",
		"NOT(width > 10px)",
		"and(width > 10px)",
		"or(width > 10px)",
		"style(--foo: bar)",
		"style(width > 10px)",
		"scroll-state(width > 10px)",
		"(width > 10px) and style(--foo: bar)",
		"style(--foo: bar) or (width > 10px)",
	}

	for _, prelude := range preludes {
		if _, ok := parseContainerPrelude(prelude); ok {
			t.Errorf("prelude %q must be rejected", prelude)
		}
	}
}

// TestContainerCondKeywordForms covers ASCII case-insensitive keywords, the
// optional space before a keyword, and parenthesized operator mixing, which
// the grammar allows across nesting levels (same-level mixing is rejected by
// TestContainerCondMixedOperatorsRejected).
func TestContainerCondKeywordForms(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		prelude string
		qname   string
		kind    string
		kids    int
		kidKind string
	}{
		{name: "lowercase and", qname: "", kind: condKindAnd, kids: 2, kidKind: "",
			prelude: "(width > 10px) and (inline-size > 20px)"},
		{name: "no space before and", qname: "", kind: condKindAnd, kids: 2, kidKind: "",
			prelude: "(width > 10px)and (inline-size > 20px)"},
		{name: "uppercase and", qname: "", kind: condKindAnd, kids: 2, kidKind: "",
			prelude: "(width > 10px) AND (inline-size > 20px)"},
		{name: "uppercase or", qname: "", kind: condKindOr, kids: 2, kidKind: "",
			prelude: "(width > 10px) OR (inline-size > 20px)"},
		{name: "uppercase not", qname: "", kind: condKindNot, kids: 1, kidKind: "",
			prelude: "NOT (width > 10px)"},
		{name: "spaced style is a name", qname: "style", kind: condKindFeat, kids: 0, kidKind: "",
			prelude: "style (width > 10px)"},
		{name: "nested and under or", qname: "", kind: condKindOr, kids: 2, kidKind: condKindAnd,
			prelude: "((width > 10px) and (inline-size > 20px)) or (width < 1px)"},
		{name: "nested or under and", qname: "", kind: condKindAnd, kids: 2, kidKind: "",
			prelude: "(width > 10px) and ((inline-size > 20px) or (width < 1px))"},
		{name: "not wraps nested or", qname: "", kind: condKindNot, kids: 1, kidKind: "",
			prelude: "not ((width > 10px) or (inline-size > 20px))"},
	}

	for _, testCase := range cases {
		query, ok := parseContainerPrelude(testCase.prelude)
		if !ok {
			t.Fatalf("%s: prelude %q rejected", testCase.name, testCase.prelude)
		}

		if query.Name != testCase.qname {
			t.Fatalf("%s: name=%q, want %q", testCase.name, query.Name, testCase.qname)
		}

		if query.Cond.Kind != testCase.kind || len(query.Cond.Kids) != testCase.kids {
			t.Fatalf("%s: kind=%q kids=%d, want %q/%d",
				testCase.name, query.Cond.Kind, len(query.Cond.Kids), testCase.kind, testCase.kids)
		}

		if testCase.kidKind != "" && query.Cond.Kids[0].Kind != testCase.kidKind {
			t.Fatalf("%s: kids[0].kind=%q, want %q", testCase.name, query.Cond.Kids[0].Kind, testCase.kidKind)
		}
	}
}

// TestContainerCondMixedOperatorsRejected covers CSS Conditional 3 section 6:
// mixing and/or at one level without parentheses is invalid.
func TestContainerCondMixedOperatorsRejected(t *testing.T) {
	t.Parallel()

	preludes := []string{
		"(width > 10px) and (inline-size > 20px) or (width < 1px)",
		"(width > 10px) or (inline-size > 20px) and (width < 1px)",
		"(width > 10px) AND (inline-size > 20px) OR (width < 1px)",
		"(width > 10px) or (inline-size > 20px) or (width < 1px) and (inline-size < 1px)",
	}

	for _, prelude := range preludes {
		if _, ok := parseContainerPrelude(prelude); ok {
			t.Errorf("prelude %q must be rejected: and/or cannot mix at one level", prelude)
		}
	}
}

// TestInvalidContainerCondDropsRule proves the rejected keyword forms drop the
// whole @container block through Parse, the real entry point.
func TestInvalidContainerCondDropsRule(t *testing.T) {
	t.Parallel()

	sty := mustSheet(t, `
		@container (width > 10px) and(inline-size > 20px) { .a { color: red } }
		@container not(width > 10px) { .b { color: red } }
		@container (width > 10px) and (inline-size > 20px) or (width < 1px) { .c { color: red } }
		p { color: green }
	`)
	if len(sty.Rules) != 1 {
		t.Fatalf("rules = %d, want only the p rule: %+v", len(sty.Rules), sty.Rules)
	}

	survivor := sty.Rules[0]
	if survivor.Container != nil || len(survivor.Selectors) != 1 || survivor.Selectors[0].Parts[0].Tag != "p" {
		t.Fatalf("survivor = %+v", survivor)
	}
}

func TestContainerWithoutSizeRejectedAtEval(t *testing.T) {
	t.Parallel()
	// Parsing succeeds; layout refuses to treat non-size-containers as
	// query containers. Here we only prove unknown features don't match.
	cq, ok := parseContainerPrelude("(height > 10px)")
	if ok && cq.Cond.Kind == "feat" {
		if cq.Cond.Matches(1000, 12) {
			t.Error("height feature must not match size-lite evaluator")
		}
	}
}

func TestHasContainerRules(t *testing.T) {
	t.Parallel()

	s := mustSheet(t, `p { color: red }`)
	if HasContainerRules([]*Stylesheet{s}) {
		t.Fatal("expected no container rules")
	}

	s2 := mustSheet(t, `@container (width > 1px) { p { color: red } }`)
	if !HasContainerRules([]*Stylesheet{s2}) {
		t.Fatal("expected container rules")
	}
}

func TestInvalidContainerPreludeSkipped(t *testing.T) {
	t.Parallel()
	sty := mustSheet(t, `
		@container { .x { color: red } }
		@container !!! { .y { color: blue } }
		p { color: green }
	`)
	// invalid @container bodies skipped; p remains
	found := false

	for _, r := range sty.Rules {
		if r.Container != nil {
			t.Fatalf("unexpected container rule: %+v", r)
		}

		if len(r.Selectors) > 0 && r.Selectors[0].Parts[0].Tag == "p" {
			found = true
		}
	}

	if !found {
		t.Fatalf("rules = %+v", sty.Rules)
	}
}
