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

func TestForEachAttrZeroAllocAndNoMutation(t *testing.T) {
	t.Parallel()

	n := &Node{
		Type: ElementNode,
		Name: "svg",
		AttrList: []Attr{
			{Name: "id", Value: "icon"},
			{Namespace: "xlink", Name: "href", Value: "#g"},
		},
	}

	if !n.HasAttrs() || n.AttrCount() != 2 {
		t.Fatalf("HasAttrs=%v AttrCount=%d, want true and 2", n.HasAttrs(), n.AttrCount())
	}

	var keys, vals []string

	n.ForEachAttr(func(k, v string) {
		keys = append(keys, k)
		vals = append(vals, v)
	})

	if len(keys) != 2 || keys[0] != "id" || vals[0] != "icon" || keys[1] != "xlink:href" || vals[1] != "#g" {
		t.Fatalf("ForEachAttr = %v / %v", keys, vals)
	}

	if n.Attrs != nil {
		t.Fatal("ForEachAttr materialized n.Attrs")
	}
}
