package layout_test

import (
	"fmt"
	"testing"
)

// declaredCase is one inline declaration and the computed text it must give
// ("" means the declaration must be dropped).
type declaredCase struct {
	prop, value, want string
}

// declaredOn lays out one div with an inline style and returns its Declared map.
func declaredOn(t *testing.T, decl string) map[string]string {
	t.Helper()

	page := fmt.Sprintf(`<!DOCTYPE html><html><body><div id="a" style="%s">x</div></body></html>`, decl)

	return boxByID(t, renderDisplay(t, page, ""), "a").Declared
}

func runDeclaredCases(t *testing.T, cases []declaredCase) {
	t.Helper()

	for _, c := range cases {
		got := declaredOn(t, c.prop+": "+c.value)[c.prop]
		if got != c.want {
			t.Errorf("%s: %s => %q, want %q", c.prop, c.value, got, c.want)
		}
	}
}

// TestDeclaredNavGrammar pins accept, normalize and reject for the nav group.
func TestDeclaredNavGrammar(t *testing.T) {
	t.Parallel()

	runDeclaredCases(t, []declaredCase{
		{"caret-animation", "MANUAL", "manual"},
		{"caret-animation", "blink", ""},
		{"caret-shape", "Underscore", "underscore"},
		{"caret-shape", "block", "block"},
		{"caret-shape", "round", ""},
		{"caret", "red", "red"},
		{"caret", "block red", "red block"},
		{"caret", "manual bar #00f", "#00f manual bar"},
		{"caret", "block bar", ""},
		{"caret", "red blue", ""},
		{"caret", "nonsense", ""},
		{"field-sizing", "content", "content"},
		{"field-sizing", "auto", ""},
		{"input-security", "none", "none"},
		{"input-security", "hidden", ""},
		{"nav-up", "auto", "auto"},
		{"nav-down", "#next", "#next"},
		{"nav-left", "#a current", "#a current"},
		{"nav-right", "#a root", "#a root"},
		{"nav-right", "#a 'frame'", "#a 'frame'"},
		{"nav-up", "next", ""},
		{"nav-up", "#a bogus", ""},
		{"nav-up", "#a root extra", ""},
		{"slider-orientation", "bottom-to-top", "bottom-to-top"},
		{"slider-orientation", "up", ""},
		{"spatial-navigation-action", "scroll", "scroll"},
		{"spatial-navigation-action", "jump", ""},
		{"spatial-navigation-contain", "contain", "contain"},
		{"spatial-navigation-function", "grid", "grid"},
		{"spatial-navigation-function", "list", ""},
		{"window-drag", "move", "move"},
		{"window-drag", "resize", ""},
		{"will-change", "transform, opacity", "transform, opacity"},
		{"will-change", "auto", "auto"},
		{"will-change", "none", ""},
		{"will-change", "auto, transform", ""},
		{"-webkit-appearance", "button", "button"},
		{"-webkit-appearance", "weird", ""},
		{"-webkit-user-select", "NONE", "none"},
		{"-webkit-user-select", "element", ""},
		{"-webkit-text-size-adjust", "120%", "120%"},
		{"text-size-adjust", "auto", "auto"},
		{"text-size-adjust", "none", "none"},
		{"text-size-adjust", "-5%", ""},
		{"text-size-adjust", "100", ""},
		{"image-animation", "paused", "paused"},
		{"image-animation", "flying", ""},
	})
}

// TestDeclaredNavInvalidKeepsEarlier proves a bad value does not replace an
// earlier good one, and wide keywords reset or inherit.
func TestDeclaredNavInvalidKeepsEarlier(t *testing.T) {
	t.Parallel()

	if got := declaredOn(t, "field-sizing: content; field-sizing: bogus")["field-sizing"]; got != "content" {
		t.Errorf("invalid value replaced a good one: %q", got)
	}

	if got := declaredOn(t, "field-sizing: content; field-sizing: initial")["field-sizing"]; got != "" {
		t.Errorf("initial must clear, got %q", got)
	}

	const page = `<!DOCTYPE html><html><body>
		<div id="p" style="caret-shape: block; caret-animation: manual; -webkit-user-select: none;
			text-size-adjust: none; nav-up: #x">
		<p id="in" style="caret-shape: inherit; nav-up: inherit">a</p><p id="plain">b</p></div></body></html>`

	display := renderDisplay(t, page, "")

	in := boxByID(t, display, "in").Declared
	if in["caret-shape"] != "block" || in["nav-up"] != "#x" {
		t.Errorf("inherit keyword: %v", in)
	}

	plain := boxByID(t, display, "plain").Declared
	if plain["caret-shape"] != "block" || plain["caret-animation"] != "manual" {
		t.Errorf("caret-shape and caret-animation inherit: %v", plain)
	}

	if _, ok := plain["nav-up"]; ok {
		t.Error("nav-up must not inherit")
	}
}
