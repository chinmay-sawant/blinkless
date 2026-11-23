package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/html"
)

// TestTransitionsNotSharedAcrossDisplays proves a caller mutating
// Box.Transitions cannot corrupt later display lists of the same document.
func TestTransitionsNotSharedAcrossDisplays(t *testing.T) {
	t.Parallel()

	tree, err := html.Parse([]byte(`<div id="a" style="width:50px;height:20px;transition:opacity 200ms ease 10ms">x</div>`))
	if err != nil {
		t.Fatal(err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")

	find := func() []string {
		var out []string

		for _, box := range layOf(t, styled).Boxes {
			if len(box.Transitions) > 0 {
				box.Transitions[0].Property = "mutated"
				out = append(out, "mutated")

				return out
			}
		}

		return out
	}

	if len(find()) == 0 {
		t.Fatal("no box carried a transition")
	}

	for _, box := range layOf(t, styled).Boxes {
		for _, tr := range box.Transitions {
			if tr.Property == "mutated" {
				t.Fatal("transition slice was shared through the global cache")
			}
		}
	}
}
