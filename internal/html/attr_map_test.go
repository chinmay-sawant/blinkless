package html

import "testing"

// TestAttrMapDoesNotMutate pins the read-only contract that makes shared
// trees safe for concurrent passes: AttrMap must leave n.Attrs nil.
func TestAttrMapDoesNotMutate(t *testing.T) {
	t.Parallel()

	n := &Node{Type: ElementNode, Name: "p", AttrList: []Attr{{Name: "id", Value: "a"}, {Name: "title", Value: "t"}}}

	got := n.AttrMap()
	if got["id"] != "a" || got["title"] != "t" {
		t.Fatalf("AttrMap = %v", got)
	}

	if n.Attrs != nil {
		t.Fatal("AttrMap materialized n.Attrs; read-only passes would race")
	}
}
