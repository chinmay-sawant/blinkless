package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/layout"
)

// marginParents are the containing blocks a hovered root can sit in. The
// flex row wraps three fixed boxes; the block parent stacks them.
//
// The fixed variants pin the holder's height, so a margin change on the root
// cannot resize the holder and the escalated splice has to commit; the other
// two grow with the margin, so the patch is refused and the host lays out in
// full.
type marginParent struct {
	name       string
	parent     string
	holderFits bool
}

func marginParents() []marginParent {
	return []marginParent{
		{"flex", `.g{display:flex;flex-wrap:wrap;gap:4px;padding:4px}`, false},
		{"block", `.g{padding:4px}`, false},
		{"flex-fixed", `.g{display:flex;flex-wrap:wrap;gap:4px;padding:4px;height:120px}`, true},
		{"block-fixed", `.g{padding:4px;height:120px}`, true},
	}
}

// marginRules change the hovered root's own margin through every spelling
// the cascade accepts. The box never changes size, only where it sits and
// how much room it takes in its parent.
func marginRules() []string {
	return []string{
		"margin-top:8px",
		"margin-bottom:8px",
		"margin-left:8px",
		"margin-right:8px",
		"margin:8px",
		"margin:2px 6px",
		"margin-block:8px",
		"margin-inline:8px",
		"margin-block-start:8px",
		"margin-inline-start:8px",
	}
}

func rootMarginSource(parent, rule string) string {
	return `<style>` + parent + `.i{width:84px;height:22px;box-sizing:border-box;background:#eee}` +
		`#a:hover{` + rule + `}</style><div class="g">` +
		`<div id="a" class="i">a</div><div id="b" class="i">b</div><div id="c" class="i">c</div></div>`
}

// restyleHover restyles doc for a new hover target over the same tree, the
// way a host that keeps one parsed page does, so the patched display and the
// retained one share their nodes.
func restyleHover(t *testing.T, doc *css.Document, hover string) *css.Document {
	t.Helper()

	next, err := css.Relayout(t.Context(), doc, 320, 200, "", hover, "")
	if err != nil {
		t.Fatalf("relayout style: %v", err)
	}

	return next
}

// TestRelayoutElementsRootMarginMatchesFullLayout pins that a hover patch
// never draws a wrong picture for a root whose own margin changes. A
// refusal is a correct answer, because the host then lays the page out in
// full. An accepted patch must equal that full layout, boxes, operations,
// owners, and paint order, on the enter and on the leave.
func TestRelayoutElementsRootMarginMatchesFullLayout(t *testing.T) {
	t.Parallel()

	for _, parent := range marginParents() {
		for _, rule := range marginRules() {
			t.Run(parent.name+"/"+rule, func(t *testing.T) {
				t.Parallel()

				source := rootMarginSource(parent.parent, rule)
				resting := applyAt(t, source, 320, 200, "", "", "")
				hovered := restyleHover(t, resting, "a")

				prev := layRetained(t, resting)

				enter, err := layout.RelayoutElements(t.Context(), hovered, prev, []string{"a"}, layout.Options{RetainState: true})
				if err == nil {
					compareDisplays(t, enter, layRetained(t, hovered))
				} else if parent.holderFits {
					t.Fatalf("enter was refused although the holder keeps its size: %v", err)
				}

				// The leave patches a display laid out for the hovered state,
				// the way a page that hovers and unhovers in turn does.
				base := layRetained(t, hovered)
				left := restyleHover(t, hovered, "")
				leave, err := layout.RelayoutElements(t.Context(), left, base, []string{"a"}, layout.Options{RetainState: true})
				if err == nil {
					compareDisplays(t, leave, layRetained(t, left))
				} else if parent.holderFits {
					t.Fatalf("leave was refused although the holder keeps its size: %v", err)
				}
			})
		}
	}
}

// TestRelayoutElementRootMarginMatchesFullLayout is the single element form
// of the same gate: the patch's display must equal a full layout.
func TestRelayoutElementRootMarginMatchesFullLayout(t *testing.T) {
	t.Parallel()

	for _, parent := range marginParents() {
		for _, rule := range marginRules() {
			t.Run(parent.name+"/"+rule, func(t *testing.T) {
				t.Parallel()

				source := rootMarginSource(parent.parent, rule)
				resting := applyAt(t, source, 320, 200, "", "", "")
				hovered := restyleHover(t, resting, "a")

				prev := layRetained(t, resting)

				patch, err := layout.RelayoutElement(t.Context(), hovered, prev, "a", layout.Options{RetainState: true})
				if err == nil {
					compareDisplays(t, patch.Display, layRetained(t, hovered))
				}
			})
		}
	}
}
