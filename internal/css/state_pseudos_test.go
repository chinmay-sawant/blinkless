package css

import "testing"

func TestUsesStatePseudos(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		source     string
		hover, act bool
		focus      bool
	}{
		{"none", `a { color: red } .x:hovered { color: blue }`, false, false, false},
		{"hover", `a:hover { color: red }`, true, false, false},
		{"active", `button:active { color: red }`, false, true, false},
		{"focus-visible", `:is(a, button):focus-visible { outline: 1px }`, false, false, true},
		{"not active", `a:not(:active) { color: red }`, false, true, false},
		{"has hover", `div:has(> .item:hover) { color: red }`, true, false, false},
	}

	for _, testCase := range cases {
		sheet, err := Parse(testCase.source)
		if err != nil {
			t.Fatalf("%s: %v", testCase.name, err)
		}

		got := UsesStatePseudos([]*Stylesheet{sheet})
		if got.Hover != testCase.hover || got.Active != testCase.act || got.Focus != testCase.focus {
			t.Fatalf("%s: got %+v, want hover=%v active=%v focus=%v",
				testCase.name, got, testCase.hover, testCase.act, testCase.focus)
		}
	}
}
