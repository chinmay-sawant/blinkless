package layout

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// Wire tests for the six GCPM cascade arms: footnote-display and
// footnote-policy (CSS Generated Content for Paged Media 3, section 2,
// https://drafts.csswg.org/css-gcpm-3/#footnotes), string-set (CSS Content 3,
// https://drafts.csswg.org/css-content-3/#propdef-string-set) and
// bookmark-label, bookmark-level, bookmark-state (GCPM bookmarks,
// https://www.w3.org/TR/css-gcpm-3/#bookmarks with longhands in CSS Content
// 3). Chrome has no footnote area, no string-set support (Chrome
// 143.0.7499.40 per string_set.go) and no bookmark support, so the drafts
// are the oracle in every case below.
//
// What these tests prove, per property: a real author stylesheet changes the
// USED value (the canonical cascade storage plus the standalone system's
// output) while the same document without the declaration keeps the initial.
// Every assertion reads a used value (lowercase keyword, decoded and
// concatenated string, outline title, depth, open flag, footnote display and
// policy, table lookup), never the stored declaration text. Uppercase
// declarations, split adjacent strings, and backslash escapes are used on
// purpose so a verbatim stored string could not match.
func gcpmWireByID(root *html.Node, ident string) *html.Node {
	var found *html.Node

	root.Walk(func(node *html.Node) {
		if found != nil || node.Type != html.ElementNode {
			return
		}

		if node.Attribute("id") == ident {
			found = node
		}
	})

	return found
}

func gcpmWireStyles(t *testing.T, src string, sheets ...*css.Stylesheet) (*html.Node, map[*html.Node]*ResolvedStyle) {
	t.Helper()

	root := mustParse(t, src)

	return root, resolveStyles(root, sheets, "print", testViewport, 800)
}

func gcpmWireCustom(t *testing.T, styles map[*html.Node]*ResolvedStyle, node *html.Node, key string) string {
	t.Helper()

	style := styles[node]
	if style == nil || style.CustomProps == nil {
		return ""
	}

	return style.CustomProps[key]
}

// TestBehaviorGCPMWireFootnoteDisplay pins that an author sheet drives the
// used footnote-display: uppercase INLINE from the sheet stores canonical
// inline, the undeclared sibling keeps no key (initial block), and the
// collector reports the used display from the same sheet.
func TestBehaviorGCPMWireFootnoteDisplay(t *testing.T) {
	t.Parallel()

	const src = `<html><body>` +
		`<span id="a" class="fn">note</span>` +
		`<span id="b">plain</span>` +
		`</body></html>`

	authorSheet := sheet(t, `.fn { float: footnote; footnote-display: INLINE }`)

	root, styles := gcpmWireStyles(t, src, authorSheet)

	if got := gcpmWireCustom(t, styles, gcpmWireByID(root, "a"), footnoteDisplayCustomKey); got != "inline" {
		t.Errorf("sheet footnote-display stores %q, want used %q (lowercase canonical)", got, "inline")
	}

	if got := gcpmWireCustom(t, styles, gcpmWireByID(root, "b"), footnoteDisplayCustomKey); got != "" {
		t.Errorf("undeclared footnote-display stores %q, want no key (initial block)", got)
	}

	items := CollectFootnotes(root, authorSheet)
	if len(items) != 1 || items[0].Display != "inline" {
		t.Errorf("collector from sheet = %+v, want one footnote with used display inline", items)
	}

	bogusRoot, bogusStyles := gcpmWireStyles(t,
		`<html><body><span id="a" style="float:footnote;footnote-display:whatever">odd</span></body></html>`)

	if got := gcpmWireCustom(t, bogusStyles, gcpmWireByID(bogusRoot, "a"), footnoteDisplayCustomKey); got != "" {
		t.Errorf("bogus footnote-display stores %q, want no key (initial block)", got)
	}

	if items := CollectFootnotes(bogusRoot); len(items) != 1 || items[0].Display != "block" {
		t.Errorf("bogus collector = %+v, want one footnote with used display block", items)
	}
}

// TestBehaviorGCPMWireFootnotePolicy pins that an author sheet drives the
// used footnote-policy: BLOCK from the sheet stores canonical block, and the
// grouping system merges the two block footnotes into one group while the
// same document without the sheet keeps two groups.
func TestBehaviorGCPMWireFootnotePolicy(t *testing.T) {
	t.Parallel()

	const src = `<html><body>` +
		`<span id="a" class="fn" style="float:footnote">first</span>` +
		`<span id="b" class="fn" style="float:footnote">second</span>` +
		`</body></html>`

	authorSheet := sheet(t, `.fn { footnote-policy: BLOCK }`)

	root, styles := gcpmWireStyles(t, src, authorSheet)

	for _, ident := range []string{"a", "b"} {
		if got := gcpmWireCustom(t, styles, gcpmWireByID(root, ident), footnotePolicyCustomKey); got != "block" {
			t.Errorf("sheet footnote-policy stores %q for %s, want used %q", got, ident, "block")
		}
	}

	if groups := GroupFootnotesByPolicy(CollectFootnotes(root, authorSheet)); len(groups) != 1 || len(groups[0]) != 2 {
		t.Errorf("sheet policy groups = %d groups, want 1 group of 2 (block merges)", len(groups))
	}

	plainRoot := mustParse(t, src)
	if groups := GroupFootnotesByPolicy(CollectFootnotes(plainRoot)); len(groups) != 2 {
		t.Errorf("absent policy groups = %d groups, want 2 (initial auto keeps one per footnote)", len(groups))
	}
}

// TestBehaviorGCPMWireStringSet pins that an author sheet drives the used
// named-string table: adjacent strings concatenate, escapes decode, and the
// undeclared heading keeps no key. The collector reads inline styles only,
// so the cascade storage is the proof the sheet lands.
func TestBehaviorGCPMWireStringSet(t *testing.T) {
	t.Parallel()

	const src = `<html><body>` +
		`<h2 id="a">Guide One</h2>` +
		`<h2 id="b" style="string-set: tricky 'a\'b'">Tricky</h2>` +
		`<h2 id="c">Plain</h2>` +
		`</body></html>`

	authorSheet := sheet(t, `#a { string-set: chapter 'Guide ' 'One' }`)

	root, styles := gcpmWireStyles(t, src, authorSheet)

	if got := gcpmWireCustom(t, styles, gcpmWireByID(root, "a"), stringSetCustomKey); got != `chapter "Guide One"` {
		t.Errorf("sheet string-set stores %q, want used %q (concatenated canonical)", got, `chapter "Guide One"`)
	}

	if got := gcpmWireCustom(t, styles, gcpmWireByID(root, "b"), stringSetCustomKey); got != `tricky "a'b"` {
		t.Errorf("inline string-set stores %q, want used %q (decoded escape via the same arm)", got, `tricky "a'b"`)
	}

	if got := gcpmWireCustom(t, styles, gcpmWireByID(root, "c"), stringSetCustomKey); got != "" {
		t.Errorf("undeclared string-set stores %q, want no key (initial none)", got)
	}

	assignments, ok := ParseStringSet(gcpmWireCustom(t, styles, gcpmWireByID(root, "a"), stringSetCustomKey))
	if !ok || len(assignments) != 1 || assignments[0].Value != "Guide One" {
		t.Errorf("stored string-set parses to %+v, want one used value %q", assignments, "Guide One")
	}

	plainRoot, plainStyles := gcpmWireStyles(t, src)
	for _, ident := range []string{"a", "c"} {
		if got := gcpmWireCustom(t, plainStyles, gcpmWireByID(plainRoot, ident), stringSetCustomKey); got != "" {
			t.Errorf("absent string-set stores %q for %s, want no key", got, ident)
		}
	}

	if got := gcpmWireCustom(t, plainStyles, gcpmWireByID(plainRoot, "b"), stringSetCustomKey); got != `tricky "a'b"` {
		t.Errorf("inline string-set without sheets stores %q, want used %q", got, `tricky "a'b"`)
	}
}

// TestBehaviorGCPMWireBookmarkLevel pins that an author sheet drives the
// used bookmark level: 2 stores canonical 2, a bogus inline value stores
// nothing, and the outline built from the sheet nests the level 2 entry
// under the level 1 entry.
func TestBehaviorGCPMWireBookmarkLevel(t *testing.T) {
	t.Parallel()

	const src = `<html><body>` +
		`<h1 id="t">Top</h1>` +
		`<h2 id="d">Deep</h2>` +
		`<h1 id="x" style="bookmark-level:bogus;bookmark-label:'Bogus'">Bogus</h1>` +
		`</body></html>`

	authorSheet := sheet(t, `#t { bookmark-level: 1; bookmark-label: 'Top' }`+
		`#d { bookmark-level: 2; bookmark-label: 'Deep' }`)

	root, styles := gcpmWireStyles(t, src, authorSheet)

	if got := gcpmWireCustom(t, styles, gcpmWireByID(root, "d"), bookmarkLevelCustomKey); got != "2" {
		t.Errorf("sheet bookmark-level stores %q, want used %q", got, "2")
	}

	if got := gcpmWireCustom(t, styles, gcpmWireByID(root, "x"), bookmarkLevelCustomKey); got != "" {
		t.Errorf("bogus bookmark-level stores %q, want no key (no bookmark)", got)
	}

	tree := BuildBookmarkOutline(root, Options{Width: testViewport, Height: 800, Sheets: []*css.Stylesheet{authorSheet}})
	if len(tree) != 1 || tree[0].Label != "Top" {
		t.Fatalf("sheet outline roots = %+v, want one root Top", tree)
	}

	if len(tree[0].Children) != 1 || tree[0].Children[0].Label != "Deep" || tree[0].Children[0].Level != 2 {
		t.Errorf("sheet outline children = %+v, want one nested Deep at level 2", tree[0].Children)
	}
}

// TestBehaviorGCPMWireBookmarkLabel pins the label cascade win: the inline
// declaration beats the sheet rule for the same element, the sheet-only
// element keeps the sheet title, and the undeclared element keeps no key so
// the outline falls back to the used element text.
func TestBehaviorGCPMWireBookmarkLabel(t *testing.T) {
	t.Parallel()

	const src = `<html><body>` +
		`<h1 id="a" style="bookmark-level:1;bookmark-label:'Inline'">Alpha</h1>` +
		`<h1 id="b">Beta</h1>` +
		`<h1 id="c" style="bookmark-level:1">Gamma Text</h1>` +
		`</body></html>`

	authorSheet := sheet(t, `#a { bookmark-level: 1; bookmark-label: 'Sheet' }`+
		`#b { bookmark-level: 1; bookmark-label: 'Sheet Beta' }`)

	root, styles := gcpmWireStyles(t, src, authorSheet)

	if got := gcpmWireCustom(t, styles, gcpmWireByID(root, "a"), bookmarkLabelCustomKey); got != "'Inline'" {
		t.Errorf("inline bookmark-label stores %q, want %q (inline wins over sheet)", got, "'Inline'")
	}

	if got := gcpmWireCustom(t, styles, gcpmWireByID(root, "b"), bookmarkLabelCustomKey); got != "'Sheet Beta'" {
		t.Errorf("sheet bookmark-label stores %q, want %q", got, "'Sheet Beta'")
	}

	if got := gcpmWireCustom(t, styles, gcpmWireByID(root, "c"), bookmarkLabelCustomKey); got != "" {
		t.Errorf("absent bookmark-label stores %q, want no key (initial reads element text)", got)
	}

	tree := BuildBookmarkOutline(root, Options{Width: testViewport, Height: 800, Sheets: []*css.Stylesheet{authorSheet}})
	if len(tree) != 3 || tree[0].Label != "Inline" || tree[1].Label != "Sheet Beta" || tree[2].Label != "Gamma Text" {
		t.Errorf("outline labels = %+v, want [Inline Sheet Beta Gamma Text] (used titles)", tree)
	}
}

// TestBehaviorGCPMWireBookmarkLabelNone pins the label skip rules: an
// element with no level produces no entry even with text, and an explicit
// bookmark-label:none produces no entry even with a valid level.
func TestBehaviorGCPMWireBookmarkLabelNone(t *testing.T) {
	t.Parallel()

	plainRoot, plainStyles := gcpmWireStyles(t, `<html><body><h1 id="p">Plain Heading</h1></body></html>`)
	if got := gcpmWireCustom(t, plainStyles, gcpmWireByID(plainRoot, "p"), bookmarkLabelCustomKey); got != "" {
		t.Errorf("absent bookmark-label stores %q, want no key (initial reads element text)", got)
	}

	plainTree := BuildBookmarkOutline(plainRoot, Options{Width: testViewport, Height: 800})
	if len(plainTree) != 0 {
		t.Errorf("undeclared level outline = %+v, want no entries (no level, no bookmark)", plainTree)
	}

	noneRoot, _ := gcpmWireStyles(t,
		`<html><body><h1 id="n" style="bookmark-level:1;bookmark-label:none">Hidden</h1></body></html>`)
	noneTree := BuildBookmarkOutline(noneRoot, Options{Width: testViewport, Height: 800})

	if len(noneTree) != 0 {
		t.Errorf("none label outline = %+v, want no entries", noneTree)
	}
}

// TestBehaviorGCPMWireBookmarkState pins that an author sheet drives the
// used open flag: closed from the sheet stores canonical closed and reports
// Open false, while the undeclared sibling keeps no key and reports Open
// true (the initial).
func TestBehaviorGCPMWireBookmarkState(t *testing.T) {
	t.Parallel()

	const src = `<html><body>` +
		`<h1 id="a">Alpha</h1>` +
		`<h1 id="b">Beta</h1>` +
		`</body></html>`

	authorSheet := sheet(t, `#a { bookmark-level: 1; bookmark-label: 'A'; bookmark-state: closed }`+
		`#b { bookmark-level: 1; bookmark-label: 'B' }`)

	root, styles := gcpmWireStyles(t, src, authorSheet)

	if got := gcpmWireCustom(t, styles, gcpmWireByID(root, "a"), bookmarkStateCustomKey); got != "closed" {
		t.Errorf("sheet bookmark-state stores %q, want used %q", got, "closed")
	}

	if got := gcpmWireCustom(t, styles, gcpmWireByID(root, "b"), bookmarkStateCustomKey); got != "" {
		t.Errorf("absent bookmark-state stores %q, want no key (initial open)", got)
	}

	tree := BuildBookmarkOutline(root, Options{Width: testViewport, Height: 800, Sheets: []*css.Stylesheet{authorSheet}})
	if len(tree) != 2 {
		t.Fatalf("sheet outline entries = %d, want 2", len(tree))
	}

	if tree[0].Open {
		t.Errorf("closed entry Open = true, want false (used flag from sheet)")
	}

	if !tree[1].Open {
		t.Errorf("absent entry Open = false, want true (initial open)")
	}
}
