package layout

import (
	"reflect"
	"testing"
)

func TestFlexBuildKeepsSharedStyleImmutable(t *testing.T) {
	t.Parallel()
	for _, declaration := range []string{
		"display:flex;width:200px;aspect-ratio:2/1",
		"display:flex;width:max-content",
		"display:-webkit-box",
		"display:flex;width:200px;flex-direction:column",
		"display:flex;width:200px;writing-mode:vertical-rl",
	} {
		t.Run(declaration, func(t *testing.T) {
			t.Parallel()
			root := mustParse(t, `<div id="flex" style="`+declaration+`">direct text<span>child</span></div>`)
			opts := Options{Width: 400, Height: 300}
			styles, err := resolveStylesWithContext(t.Context(), root, opts, nil)
			if err != nil {
				t.Fatal(err)
			}
			node := findByID(root, "flex")
			before := *styles[node]
			// Layout's normal entry initializes fonts and caches; its resolved style
			// must still carry the same declared values after building the flex box.
			result, err := Layout(root, opts)
			if err != nil {
				t.Fatal(err)
			}
			defer result.Release()
			got := result.styles[node]
			if got == nil || !reflect.DeepEqual(before, *got) {
				t.Fatal("flex layout changed its shared cascade style")
			}
		})
	}
}
