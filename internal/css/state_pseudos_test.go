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

	for _, c := range cases {
		sheet, err := Parse(c.source)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}

		got := UsesStatePseudos([]*Stylesheet{sheet})
		if got.Hover != c.hover || got.Active != c.act || got.Focus != c.focus {
			t.Fatalf("%s: got %+v, want hover=%v active=%v focus=%v",
				c.name, got, c.hover, c.act, c.focus)
		}
	}
}
