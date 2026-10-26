package css

import (
	"testing"
)

func TestSelectorIsElementLocal(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		rule  string
		local bool
	}{
		{name: "class", rule: `.a { color: red }`, local: true},
		{name: "compound", rule: `div.a#b[x="y"] { color: red }`, local: true},
		{name: "universal", rule: `* { color: red }`, local: true},
		{name: "attribute presence", rule: `[disabled] { color: red }`, local: true},
		{name: "hover", rule: `a:hover { color: red }`, local: true},
		{name: "checked", rule: `input:checked { color: red }`, local: true},
		{name: "link", rule: `a:link { color: red }`, local: true},
		{name: "pseudo element", rule: `p.note::before { color: red }`, local: true},
		{name: "descendant", rule: `section .a { color: red }`, local: false},
		{name: "child", rule: `div > .a { color: red }`, local: false},
		{name: "adjacent", rule: `.a + .a { color: red }`, local: false},
		{name: "sibling", rule: `p ~ p { color: red }`, local: false},
		{name: "first child", rule: `span:first-child { color: red }`, local: false},
		{name: "nth child", rule: `span:nth-child(2) { color: red }`, local: false},
		{name: "root", rule: `:root { color: red }`, local: false},
		{name: "has", rule: `div:has(.a) { color: red }`, local: false},
		{name: "not", rule: `.a:not(.b) { color: red }`, local: false},
		{name: "is", rule: `:is(.a, .b) { color: red }`, local: false},
		{name: "where", rule: `:where(.a) { color: red }`, local: false},
		{name: "unknown", rule: `span:unknown-thing { color: red }`, local: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			sheet, err := Parse(testCase.rule)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}

			if len(sheet.Rules) != 1 || len(sheet.Rules[0].Selectors) != 1 {
				t.Fatalf("fixture parsed to %d rules", len(sheet.Rules))
			}

			if got := SelectorIsElementLocal(sheet.Rules[0].Selectors[0]); got != testCase.local {
				t.Fatalf("SelectorIsElementLocal(%q) = %v, want %v", testCase.rule, got, testCase.local)
			}
		})
	}
}
