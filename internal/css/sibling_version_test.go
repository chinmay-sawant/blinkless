package css

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// TestSiblingEntryRebuildsOnSameLengthReplacement replaces the only element
// child with a text node. The child count stays 1, so only ChildVersion tells
// the memo that its element list is stale.
func TestSiblingEntryRebuildsOnSameLengthReplacement(t *testing.T) {
	t.Parallel()

	root := treeFor(t, `<div id="p"><span id="s"></span></div>`)
	parent := byID(root, "p")

	if got := siblingEntryFor(parent); got == nil || len(got.elems) != 1 {
		t.Fatalf("initial entry = %+v", got)
	}

	parent.Children[0].Parent = nil
	parent.Children = []*html.Node{{Type: html.TextNode, Text: "x", Parent: parent}}
	parent.ChildVersion++

	got := siblingEntryFor(parent)
	if got == nil || len(got.elems) != 0 {
		t.Fatalf("entry after replacement kept %d stale elements", len(got.elems))
	}
}
