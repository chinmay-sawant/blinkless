package layout_test

import (
	"math"
	"testing"
)

// TestDisplaySupportsTextWrapStyleAcceptance: @supports must follow the
// catalog decision for text-wrap-style. balance, stable, auto, and pretty
// have layout behavior (balance narrows line breaking, pretty re-breaks
// orphans, stable and auto wrap greedily), so their queries apply;
// avoid-short-last-line falls back to normal wrapping in this engine and must
// not satisfy the query. The text-wrap shorthand follows the same rule for
// its style component, except pretty which the shorthand does not accept.
func TestDisplaySupportsTextWrapStyleAcceptance(t *testing.T) {
	t.Parallel()

	const page = `<!DOCTYPE html><html><head></head><body><div id="box">x</div></body></html>`

	cases := []struct {
		name  string
		query string
		want  float64
	}{
		{"balance accepted", `@supports (text-wrap-style: balance) { #box { width: 120px } }`, 120},
		{"auto accepted", `@supports (text-wrap-style: auto) { #box { width: 120px } }`, 120},
		{"stable accepted", `@supports (text-wrap-style: stable) { #box { width: 120px } }`, 120},
		{"pretty accepted", `@supports (text-wrap-style: pretty) { #box { width: 120px } }`, 120},
		{
			"avoid-short-last-line rejected",
			`@supports (text-wrap-style: avoid-short-last-line) { #box { width: 120px } }`,
			11,
		},
		{"bogus rejected", `@supports (text-wrap-style: bogus) { #box { width: 120px } }`, 11},
		{"shorthand balance accepted", `@supports (text-wrap: balance) { #box { width: 120px } }`, 120},
		{"shorthand nowrap balance accepted", `@supports (text-wrap: nowrap balance) { #box { width: 120px } }`, 120},
		{"shorthand pretty rejected", `@supports (text-wrap: pretty) { #box { width: 120px } }`, 11},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := renderBoxID(t, page, `#box { width: 11px } `+testCase.query)
			if math.Abs(got-testCase.want) > 1 {
				t.Fatalf("box width = %.2fpx, want %.2fpx", got, testCase.want)
			}
		})
	}
}
