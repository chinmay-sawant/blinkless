package layout_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

// acceptanceViewportPx is the 200px viewport the CSS acceptance regressions
// lay out in.
const acceptanceViewportPx = 200

// renderBoxID lays one page out at a 200px viewport and returns the border-box
// width of the element with id "box", in CSS pixels.
func renderBoxID(t *testing.T, page, sheetText string) float64 {
	t.Helper()

	doc, err := html.Parse([]byte(page))
	if err != nil {
		t.Fatal(err)
	}

	options := css.Options{WidthPx: acceptanceViewportPx, HeightPx: 100, Media: "screen"}

	if sheetText != "" {
		parsed, err := css.Parse(sheetText)
		if err != nil {
			t.Fatal(err)
		}

		options.Extra = []*css.Sheet{parsed}
	}

	styled, err := css.Apply(t.Context(), doc, options)
	if err != nil {
		t.Fatal(err)
	}

	display, err := layout.DisplayList(t.Context(), styled)
	if err != nil {
		t.Fatal(err)
	}

	for _, box := range display.Boxes {
		if box.ID == "box" {
			return box.W
		}
	}

	t.Fatal("box element not found in display list")

	return 0
}

// TestDisplayRejectsInvalidDeclarations (CSS-01b): the stylesheet and inline
// paths both reject an invalid ordinary value before it replaces a valid
// declaration, verified through the public DisplayList. 80px is the width the
// valid declaration asks for in a 200px viewport.
func TestDisplayRejectsInvalidDeclarations(t *testing.T) {
	t.Parallel()

	const page = `<!DOCTYPE html><html><head></head><body><div id="box">x</div></body></html>`

	const inlinePage = `<!DOCTYPE html><html><head></head><body><div id="box" style="%s">x</div></body></html>`

	cases := []struct {
		name  string
		page  string
		sheet string
		want  float64
	}{
		{"stylesheet later invalid", page, `#box { width: 80px; width: bogus }`, 80},
		{"stylesheet invalid higher specificity", page, `#box { width: 80px } body #box { width: bogus }`, 80},
		{"stylesheet invalid important", page, `#box { width: 80px } body #box { width: bogus !important }`, 80},
		{"inline later invalid", fmt.Sprintf(inlinePage, "width: 80px; width: bogus"), "", 80},
		{"inline invalid loses to stylesheet", fmt.Sprintf(inlinePage, "width: bogus"), `#box { width: 80px }`, 80},
		{
			"inline invalid important loses to stylesheet",
			fmt.Sprintf(inlinePage, "width: bogus !important"),
			`#box { width: 80px }`,
			80,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := renderBoxID(t, testCase.page, testCase.sheet); math.Abs(got-testCase.want) > 1 {
				t.Fatalf("box width = %.2fpx, want %.2fpx", got, testCase.want)
			}
		})
	}
}

// TestDisplaySupportsQueryValueAcceptance (CSS-02c): @supports must not apply
// its block when the engine owns the property but rejects the value. The base
// 11px width stays for display:bogus; a valid supported query overrides it.
// The background, flex-flow, grid, counter, quotes, and hyphenate-character
// rows cover the CSS-01b acceptance table through the public DisplayList.
func TestDisplaySupportsQueryValueAcceptance(t *testing.T) {
	t.Parallel()

	const page = `<!DOCTYPE html><html><head></head><body><div id="box">x</div></body></html>`

	cases := []struct {
		name  string
		query string
		want  float64
	}{
		{"invalid value blocked", `@supports (display: bogus) { #box { width: 120px } }`, 11},
		{"unknown property blocked", `@supports (unknown-prop: 1) { #box { width: 120px } }`, 11},
		{"valid value applies", `@supports (display: grid) { #box { width: 120px } }`, 120},
		{"negated invalid value applies", `@supports not (display: bogus) { #box { width: 120px } }`, 120},
		{"valid background applies", `@supports (background: red) { #box { width: 120px } }`, 120},
		{"invalid background blocked", `@supports (background: no-repeat center) { #box { width: 120px } }`, 11},
		{"valid background-image applies", `@supports (background-image: url(x.png)) { #box { width: 120px } }`, 120},
		{"invalid background-image blocked", `@supports (background-image: bogus) { #box { width: 120px } }`, 11},
		{"valid flex-flow applies", `@supports (flex-flow: row wrap) { #box { width: 120px } }`, 120},
		{"invalid flex-flow blocked", `@supports (flex-flow: row bogus) { #box { width: 120px } }`, 11},
		{"valid grid-row applies", `@supports (grid-row: span 2) { #box { width: 120px } }`, 120},
		{"invalid grid-row blocked", `@supports (grid-row: span bogus) { #box { width: 120px } }`, 11},
		{"valid counter-reset applies", `@supports (counter-reset: item 1) { #box { width: 120px } }`, 120},
		{"invalid counter-reset blocked", `@supports (counter-reset: 5 item) { #box { width: 120px } }`, 11},
		{"valid quotes applies", `@supports (quotes: "a" "b") { #box { width: 120px } }`, 120},
		{"invalid quotes blocked", `@supports (quotes: "a") { #box { width: 120px } }`, 11},
		{"valid hyphenate-character applies", `@supports (hyphenate-character: auto) { #box { width: 120px } }`, 120},
		{"invalid hyphenate-character blocked", `@supports (hyphenate-character: two tokens) { #box { width: 120px } }`, 11},
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
