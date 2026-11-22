package layout

import (
	"strings"
	"testing"
)

// attrQuote picks the attribute quote that does not appear in the declaration,
// so a quoted OpenType tag survives the HTML parser.
func attrQuote(decl string) string {
	if strings.Contains(decl, `"`) {
		return `'`
	}

	return `"`
}

// featuresFor lays out one paragraph with an inline style and returns the
// feature string of its first text run.
func featuresFor(t *testing.T, rule string) string {
	t.Helper()

	decl := "margin:0;font-size:12pt;" + rule
	q := attrQuote(decl)
	d := layoutHTML(t, `<html><body style="margin:0"><p style=`+q+decl+q+`>Hello</p></body></html>`)

	return firstText(d).FontFeatures()
}

// TestFontFeatureSettingsReachTheTextOp pins the declaration forms that must
// produce the feature string the shaper parses, and the ones CSS drops.
func TestFontFeatureSettingsReachTheTextOp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		rule string
		want string
	}{
		{"liga off and smcp", `font-feature-settings:"liga" 0, "smcp" 1`, `"liga" 0, "smcp" 1`},
		{"single quotes", `font-feature-settings:'liga' 0, 'smcp' 1`, `"liga" 0, "smcp" 1`},
		{"off keyword", `font-feature-settings:"liga" off`, `"liga" 0`},
		{"on keyword", `font-feature-settings:"liga" on`, `"liga" 1`},
		{"bare tag defaults to one", `font-feature-settings:"smcp"`, `"smcp" 1`},
		{"uppercase tag keeps case", `font-feature-settings:"SMCP" 1`, `"SMCP" 1`},
		{"numeric value", `font-feature-settings:"salt" 3`, `"salt" 3`},
		{"normal", `font-feature-settings:normal`, ``},
		{"invalid tag length dropped", `font-feature-settings:"lig" 1`, ``},
		{"unquoted tag dropped", `font-feature-settings:liga 1`, ``},
		{"one bad entry drops the declaration", `font-feature-settings:"liga" 0, bad`, ``},
		{"kerning none", `font-kerning:none`, `"kern" 0`},
		{"ligatures none", `font-variant-ligatures:none`, `"liga" 0, "clig" 0, "dlig" 0, "hlig" 0, "calt" 0`},
		{"no-common-ligatures", `font-variant-ligatures:no-common-ligatures`, `"liga" 0, "clig" 0`},
		{"variant shorthand none", `font-variant:none`, `"liga" 0, "clig" 0, "dlig" 0, "hlig" 0, "calt" 0`},
		{"small caps", `font-variant:small-caps`, `"smcp" 1`},
		{
			"feature settings win on a tag collision",
			`font-variant-caps:small-caps;font-feature-settings:"smcp" 0`,
			`"smcp" 0`,
		},
		{
			"variant tags come first then settings",
			`font-variant-numeric:tabular-nums;font-feature-settings:"liga" 0`,
			`"tnum" 1, "liga" 0`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := featuresFor(t, tc.rule); got != tc.want {
				t.Fatalf("%s: features = %q, want %q", tc.rule, got, tc.want)
			}
		})
	}
}

// TestFontFeatureSettingsInheritAndReset checks that a child inherits the
// parent's list and that normal clears it.
func TestFontFeatureSettingsInheritAndReset(t *testing.T) {
	t.Parallel()

	inherit := layoutHTML(t, `<html><body style="margin:0">`+
		`<div style='font-size:12pt;font-feature-settings:"liga" 0'><p style="margin:0">Hello</p></div></body></html>`)
	if got := firstText(inherit).FontFeatures(); got != `"liga" 0` {
		t.Errorf("inherited features = %q, want %q", got, `"liga" 0`)
	}

	reset := layoutHTML(t, `<html><body style="margin:0">`+
		`<div style='font-size:12pt;font-feature-settings:"liga" 0'>`+
		`<p style="margin:0;font-feature-settings:normal">Hello</p></div></body></html>`)
	if got := firstText(reset).FontFeatures(); got != "" {
		t.Errorf("normal left features = %q, want none", got)
	}
}
