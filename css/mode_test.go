package css_test

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/internal/pubstate"
)

// TestApplyCarriesDocumentMode proves the doctype-derived document mode the
// HTML parser records reaches the style/layout handoff (pubstate.Styled).
func TestApplyCarriesDocumentMode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		src  string
		want string
	}{
		{`<!DOCTYPE html><p>x`, "no-quirks"},
		{
			`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" ` +
				`"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd"><p>x`,
			"limited-quirks",
		},
		{`<p>x`, "quirks"},
	}

	for _, testCase := range cases {
		doc, err := html.Parse([]byte(testCase.src))
		if err != nil {
			t.Fatalf("Parse(%q): %v", testCase.src, err)
		}

		styled, err := css.Apply(t.Context(), doc, css.Options{WidthPx: 200, HeightPx: 100, Media: "screen"})
		if err != nil {
			t.Fatalf("Apply(%q): %v", testCase.src, err)
		}

		state, ok := pubstate.StyledOf(styled)
		if !ok {
			t.Fatalf("StyledOf(%q): document not registered", testCase.src)
		}

		if got := state.Mode.String(); got != testCase.want {
			t.Errorf("Apply(%q) mode = %s, want %s", testCase.src, got, testCase.want)
		}
	}
}
