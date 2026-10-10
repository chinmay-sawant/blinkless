package layout

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// Document-outline behavior tests for the GCPM bookmark properties
// (bookmark-label, bookmark-level, bookmark-state).
//
// Spec: https://www.w3.org/TR/css-gcpm-3/#bookmarks with the longhands in
// CSS Content 3 (https://drafts.csswg.org/css-content-3/#propdef-bookmark-label
// and the matching bookmark-level and bookmark-state sections). Chrome has no
// support for any of the three, so there is no browser reference to compare
// against: the drafts are the only oracle, and the catalog rows stay
// unsupported (testdata/css/catalog/properties.json) with "No handler" rows
// in documentation/compatibility-matrix.md:532-534.
//
// Every assertion below reads USED tree fields (Label, Level, Open and the
// Children nesting), never a stored style string. Rendering the tree into PDF
// bookmarks is out of scope (no PDF writer); the built tree with correct
// nesting and labels is the observable surface.

// bookmarkOutlineForTest parses src and builds the outline with the viewport
// the layout tests use plus optional author sheets.
func bookmarkOutlineForTest(t *testing.T, src string, sheets ...*css.Stylesheet) []*BookmarkNode {
	t.Helper()

	return BuildBookmarkOutline(mustParse(t, src), Options{
		Width: testViewport, Height: 800, Sheets: sheets,
	})
}

// assertBookmarkEntry checks one outline node's used fields: the title text,
// the nesting depth and the open flag.
func assertBookmarkEntry(t *testing.T, node *BookmarkNode, label string, level int, open bool) {
	t.Helper()

	if node == nil || node.Label != label || node.Level != level || node.Open != open {
		t.Errorf("outline entry = %+v, want {%s %d open=%v}", node, label, level, open)
	}
}

// TestBehaviorBookmarkOutlineNesting pins the core model: bookmark-level is
// the nesting depth, the used bookmark-label is the title and siblings nest
// under the nearest preceding entry with a smaller level.
func TestBehaviorBookmarkOutlineNesting(t *testing.T) {
	t.Parallel()

	tree := bookmarkOutlineForTest(t, `<html><body>`+
		`<h1 style="bookmark-level:1;bookmark-label:'Intro'">Intro</h1>`+
		`<h2 style="bookmark-level:2;bookmark-label:'Details'">Details</h2>`+
		`<h1 style="bookmark-level:1;bookmark-label:'Next'">Next</h1>`+
		`</body></html>`)

	if len(tree) != 2 {
		t.Fatalf("outline roots = %d, want 2 (Intro, Next)", len(tree))
	}

	assertBookmarkEntry(t, tree[0], "Intro", 1, true)

	if len(tree[0].Children) != 1 {
		t.Fatalf("Intro children = %d, want 1 (Details)", len(tree[0].Children))
	}

	assertBookmarkEntry(t, tree[0].Children[0], "Details", 2, true)

	if len(tree[0].Children[0].Children) != 0 {
		t.Errorf("Details children = %d, want 0", len(tree[0].Children[0].Children))
	}

	assertBookmarkEntry(t, tree[1], "Next", 1, true)

	if len(tree[1].Children) != 0 {
		t.Errorf("Next children = %d, want 0", len(tree[1].Children))
	}
}

// TestBehaviorBookmarkLabelContentUsesElementText pins the content keyword and
// the initial value: an explicit bookmark-label:content() and an absent
// bookmark-label both resolve to the used element text.
func TestBehaviorBookmarkLabelContentUsesElementText(t *testing.T) {
	t.Parallel()

	explicit := bookmarkOutlineForTest(t, `<html><body>`+
		`<h1 style="bookmark-level:1;bookmark-label:content()">Hello <span>World</span></h1>`+
		`</body></html>`)

	if len(explicit) != 1 || explicit[0].Label != "Hello World" {
		t.Errorf("content() outline = %+v, want one entry labeled Hello World", explicit)
	}

	implicit := bookmarkOutlineForTest(t, `<html><body>`+
		`<h1 style="bookmark-level:1">Plain Heading</h1>`+
		`</body></html>`)

	if len(implicit) != 1 || implicit[0].Label != "Plain Heading" {
		t.Errorf("default-label outline = %+v, want one entry labeled Plain Heading", implicit)
	}
}

// TestBehaviorBookmarkStateOpenClosed pins the used open flag: closed sets
// Open false while open and the absent initial resolve to true.
func TestBehaviorBookmarkStateOpenClosed(t *testing.T) {
	t.Parallel()

	tree := bookmarkOutlineForTest(t, `<html><body>`+
		`<h1 style="bookmark-level:1;bookmark-label:'A';bookmark-state:closed">A</h1>`+
		`<h1 style="bookmark-level:1;bookmark-label:'B';bookmark-state:open">B</h1>`+
		`<h1 style="bookmark-level:1;bookmark-label:'C'">C</h1>`+
		`</body></html>`)

	if len(tree) != 3 {
		t.Fatalf("outline entries = %d, want 3", len(tree))
	}

	assertBookmarkEntry(t, tree[0], "A", 1, false)
	assertBookmarkEntry(t, tree[1], "B", 1, true)
	assertBookmarkEntry(t, tree[2], "C", 1, true)
}

// TestBehaviorBookmarkLevelNoneSkipsEntry pins the skip rule: bookmark-level
// none, out-of-range integers and unparsable values produce no entry even
// when a label is present.
func TestBehaviorBookmarkLevelNoneSkipsEntry(t *testing.T) {
	t.Parallel()

	tree := bookmarkOutlineForTest(t, `<html><body>`+
		`<h1 style="bookmark-level:none;bookmark-label:'Skipped'">Skipped</h1>`+
		`<h1 style="bookmark-level:0;bookmark-label:'Zero'">Zero</h1>`+
		`<h1 style="bookmark-level:bogus;bookmark-label:'Bogus'">Bogus</h1>`+
		`<h1 style="bookmark-label:'NoLevel'">NoLevel</h1>`+
		`<h1 style="bookmark-level:1;bookmark-label:'Kept'">Kept</h1>`+
		`</body></html>`)

	if len(tree) != 1 {
		t.Fatalf("outline entries = %d, want 1 (only Kept)", len(tree))
	}

	if tree[0].Label != "Kept" || tree[0].Level != 1 {
		t.Errorf("kept entry = %+v, want {Kept 1}", tree[0])
	}
}

// TestBehaviorBookmarkLabelNoneSkipsEntry pins the label skip rule: an
// explicit bookmark-label:none produces no entry even with a valid level.
func TestBehaviorBookmarkLabelNoneSkipsEntry(t *testing.T) {
	t.Parallel()

	tree := bookmarkOutlineForTest(t, `<html><body>`+
		`<h1 style="bookmark-level:1;bookmark-label:none">Hidden</h1>`+
		`<h1 style="bookmark-level:1;bookmark-label:'Shown'">Shown</h1>`+
		`</body></html>`)

	if len(tree) != 1 || tree[0].Label != "Shown" {
		t.Errorf("outline = %+v, want one entry labeled Shown", tree)
	}
}

// TestBehaviorBookmarkStylesheetCascade pins that used values come from the
// cascade, not just inline styles: sheet rules set level, label and state,
// and an inline declaration wins over the sheet for the same element.
func TestBehaviorBookmarkStylesheetCascade(t *testing.T) {
	t.Parallel()

	authorSheet := sheet(t, `#a{bookmark-level:2;bookmark-label:'Sheet Label';bookmark-state:closed}`+
		`#b{bookmark-level:1}`+
		`#c{bookmark-level:1;bookmark-label:'Sheet'}`)

	tree := bookmarkOutlineForTest(t, `<html><body>`+
		`<h1 id="a">Alpha</h1>`+
		`<h2 id="b">Beta</h2>`+
		`<h3 id="c" style="bookmark-level:3;bookmark-label:'Inline'">Gamma</h3>`+
		`</body></html>`, authorSheet)

	if len(tree) != 2 {
		t.Fatalf("outline roots = %d, want 2 (Sheet Label, Beta)", len(tree))
	}

	assertBookmarkEntry(t, tree[0], "Sheet Label", 2, false)

	if len(tree[0].Children) != 0 {
		t.Errorf("sheet entry children = %d, want 0 (level 2 has no deeper sibling)", len(tree[0].Children))
	}

	assertBookmarkEntry(t, tree[1], "Beta", 1, true)

	if len(tree[1].Children) != 1 {
		t.Fatalf("Beta children = %d, want 1 (Inline at level 3)", len(tree[1].Children))
	}

	assertBookmarkEntry(t, tree[1].Children[0], "Inline", 3, true)
}

// TestBehaviorBookmarkSkippedLevelNestsUnderNearestAncestor pins the skipped
// level policy: a level 3 heading with no level 2 ancestor nests under the
// nearest entry with a smaller level instead of becoming a root.
func TestBehaviorBookmarkSkippedLevelNestsUnderNearestAncestor(t *testing.T) {
	t.Parallel()

	tree := bookmarkOutlineForTest(t, `<html><body>`+
		`<h1 style="bookmark-level:1;bookmark-label:'Top'">Top</h1>`+
		`<h3 style="bookmark-level:3;bookmark-label:'Deep'">Deep</h3>`+
		`</body></html>`)

	if len(tree) != 1 || tree[0].Label != "Top" {
		t.Fatalf("outline roots = %+v, want one root Top", tree)
	}

	if len(tree[0].Children) != 1 {
		t.Fatalf("Top children = %d, want 1 (Deep)", len(tree[0].Children))
	}

	child := tree[0].Children[0]

	if child.Label != "Deep" || child.Level != 3 {
		t.Errorf("nested entry = %+v, want {Deep 3}", child)
	}
}
