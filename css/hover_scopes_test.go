package css_test

import (
	"slices"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
)

func TestHoverChangedScopes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name, rules, previous, next string
		want                        []string
	}{
		{"plain panel to button", ".button:hover{color:red}", "panel", "a", []string{"a"}},
		{"between buttons", ".button:hover{color:red}", "a", "b", []string{"a", "b"}},
		{"descendant without ID", ".button:hover .label{color:red}", "panel", "a", []string{"a"}},
		{"inherited parent", ".panel:hover{color:red}.button:hover{color:blue}", "panel", "b", []string{"panel", "b"}},
		{"generated content", ".button:hover::before{content:'x'}", "panel", "a", []string{"a"}},
		{"no changed selector", ".button:hover{color:red}", "panel", "", nil},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			doc := mustDoc(t, `<style>`+test.rules+`</style>`+
				`<div class="panel" id="panel"><button class="button" id="a">`+
				`<span class="label">A</span></button></div><button class="button" id="b">B</button>`)

			styled, err := css.Apply(t.Context(), doc, css.Options{
				WidthPx: 320, HeightPx: 200, Hover: test.next,
			})
			if err != nil {
				t.Fatal(err)
			}

			got, ok := styled.HoverChangedScopes(test.previous)
			if !ok || !slices.Equal(got, test.want) {
				t.Fatalf("scopes = %v, %v; want %v, true", got, ok, test.want)
			}
		})
	}
}

func TestHoverChangedScopesRequiresAnID(t *testing.T) {
	t.Parallel()

	doc := mustDoc(t, `<style>button:hover + div{color:red}</style><button id="a">A</button><div>B</div>`)

	styled, err := css.Apply(t.Context(), doc, css.Options{WidthPx: 320, HeightPx: 200, Hover: "a"})
	if err != nil {
		t.Fatal(err)
	}

	if ids, ok := styled.HoverChangedScopes(""); ok {
		t.Fatalf("unidentified sibling gave scopes %v", ids)
	}

	var absent *css.Document

	if _, ok := absent.HoverChangedScopes(""); ok {
		t.Fatal("nil document accepted")
	}
}
