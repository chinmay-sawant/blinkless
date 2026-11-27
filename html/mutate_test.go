package html

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	ihtml "github.com/chinmay-sawant/blinkless/internal/html"
)

func TestRemoveAttributeHTMLAndNoOp(t *testing.T) {
	t.Parallel()

	doc := mustParseDirty(t, `<div id="box" data-role="banner" class="a">Hi</div>`)
	ver0 := doc.root.MutationVersion

	// Missing element, missing attr, empty args are no-ops.
	if doc.RemoveAttribute("missing", "class") ||
		doc.RemoveAttribute("box", "nonexistent") ||
		doc.RemoveAttribute("box", "") ||
		doc.RemoveAttribute("", "class") {
		t.Fatal("RemoveAttribute reported true on no-op")
	}

	var nilDoc *Document
	if nilDoc.RemoveAttribute("box", "class") {
		t.Fatal("nil Document.RemoveAttribute reported true")
	}

	if doc.root.MutationVersion != ver0 || len(doc.DirtyIDs()) != 0 || doc.DirtyStructural() {
		t.Fatalf("no-op mutated state: ver=%d->%d dirty=%q structural=%v",
			ver0, doc.root.MutationVersion, doc.DirtyIDs(), doc.DirtyStructural())
	}

	// Case-insensitive match on HTML elements removes from both Attrs and AttrList.
	if !doc.RemoveAttribute("box", "DATA-ROLE") {
		t.Fatal("RemoveAttribute(DATA-ROLE) reported false")
	}

	if doc.root.MutationVersion != ver0+1 {
		t.Fatalf("MutationVersion = %d, want %d", doc.root.MutationVersion, ver0+1)
	}

	dirtyHas(t, doc, "box")
	if doc.DirtyStructural() {
		t.Fatal("RemoveAttribute set structural flag")
	}

	node := doc.elementByID("box")
	if node.HasAttr("data-role") || node.Attribute("data-role") != "" {
		t.Fatalf("data-role still present: attrs=%v list=%+v", node.Attrs, node.AttrList)
	}

	// Removing again is a no-op.
	ver1 := doc.root.MutationVersion
	if doc.RemoveAttribute("box", "data-role") {
		t.Fatal("second RemoveAttribute reported true")
	}

	if doc.root.MutationVersion != ver1 {
		t.Fatalf("second RemoveAttribute bumped MutationVersion: %d -> %d", ver1, doc.root.MutationVersion)
	}

	// Also works after SetAttribute materializes node.Attrs.
	if !doc.SetAttribute("box", "title", "tip") {
		t.Fatal("SetAttribute failed")
	}

	if !doc.RemoveAttribute("box", "TITLE") {
		t.Fatal("RemoveAttribute(TITLE) failed")
	}

	if node.HasAttr("title") || len(node.Attrs) != len(node.AttrList) {
		t.Fatalf("Attrs and AttrList out of sync: attrs=%v list=%+v", node.Attrs, node.AttrList)
	}
}

func TestRemoveAttributeSVGCaseSensitive(t *testing.T) {
	t.Parallel()

	doc := mustParseDirty(t, `<svg id="icon" viewBox="0 0 24 24"></svg>`)
	ver0 := doc.root.MutationVersion

	if doc.RemoveAttribute("icon", "viewbox") {
		t.Fatal("RemoveAttribute matched wrong case on SVG element")
	}

	if doc.root.MutationVersion != ver0 || len(doc.DirtyIDs()) != 0 {
		t.Fatal("wrong-case SVG RemoveAttribute mutated state")
	}

	if !doc.RemoveAttribute("icon", "viewBox") {
		t.Fatal("RemoveAttribute(viewBox) on SVG reported false")
	}

	node := doc.elementByID("icon")
	if node.HasAttr("viewBox") || len(node.AttrList) != 1 {
		t.Fatalf("viewBox still present on SVG: attrs=%v list=%+v", node.Attrs, node.AttrList)
	}
}

func TestRemoveAttributeIDInvalidatesIndexAndFallsBackToDuplicate(t *testing.T) {
	t.Parallel()

	doc := mustParseDirty(t, `<div id="dup">first</div><div id="dup">second</div>`)

	if !doc.RemoveAttribute("dup", "ID") {
		t.Fatal("RemoveAttribute(ID) reported false")
	}

	dirtyHas(t, doc, "dup")

	elem, ok := doc.Find("dup")
	if !ok || elem.Text != "second" {
		t.Fatalf("Find(dup) = %+v, %v, want second element", elem, ok)
	}

	if !doc.RemoveAttribute("dup", "id") {
		t.Fatal("second RemoveAttribute(id) reported false")
	}

	if _, ok := doc.Find("dup"); ok {
		t.Fatal("Find(dup) still hit after removing last id")
	}
}

func TestClassListHelpers(t *testing.T) {
	t.Parallel()

	doc := mustParseDirty(t, `<div id="box" class="a  b">Hi</div><div id="bare">Empty</div>`)
	ver0 := doc.root.MutationVersion

	// Empty, whitespace-only, multi-token, missing element, nil doc are no-ops.
	if doc.AddClass("box", "") || doc.AddClass("box", "   ") || doc.AddClass("box", "x y") ||
		doc.AddClass("missing", "x") || doc.AddClass("box", "a") ||
		doc.RemoveClass("box", "") || doc.RemoveClass("box", "   ") || doc.RemoveClass("box", "z") ||
		doc.RemoveClass("missing", "a") {
		t.Fatal("no-op AddClass/RemoveClass reported true")
	}

	if added, ok := doc.ToggleClass("box", "   "); added || ok {
		t.Fatalf("ToggleClass(whitespace) = (%v, %v), want (false, false)", added, ok)
	}

	if added, ok := doc.ToggleClass("missing", "x"); added || ok {
		t.Fatalf("ToggleClass(missing) = (%v, %v), want (false, false)", added, ok)
	}

	var nilDoc *Document
	if nilDoc.AddClass("box", "x") || nilDoc.RemoveClass("box", "a") {
		t.Fatal("nil Document class helper reported true")
	}

	if added, ok := nilDoc.ToggleClass("box", "x"); added || ok {
		t.Fatal("nil Document.ToggleClass reported true")
	}

	if doc.root.MutationVersion != ver0 || len(doc.DirtyIDs()) != 0 {
		t.Fatalf("no-op class calls mutated state: ver=%d->%d dirty=%q",
			ver0, doc.root.MutationVersion, doc.DirtyIDs())
	}

	// AddClass appends without duplicates.
	if !doc.AddClass("box", "c") {
		t.Fatal("AddClass(c) reported false")
	}

	node := doc.elementByID("box")
	if got := node.Attribute("class"); got != "a b c" {
		t.Fatalf("class = %q, want %q", got, "a b c")
	}

	if len(node.Attrs) != len(node.AttrList) {
		t.Fatalf("Attrs and AttrList out of sync: attrs=%v list=%+v", node.Attrs, node.AttrList)
	}

	dirtyHas(t, doc, "box")
	if doc.DirtyStructural() {
		t.Fatal("AddClass set structural flag")
	}

	// AddClass on element with no class attribute yet.
	if !doc.AddClass("bare", "active") {
		t.Fatal("AddClass(bare, active) reported false")
	}

	if got := doc.elementByID("bare").Attribute("class"); got != "active" {
		t.Fatalf("bare class = %q, want %q", got, "active")
	}

	// RemoveClass removes existing class.
	if !doc.RemoveClass("box", "b") {
		t.Fatal("RemoveClass(b) reported false")
	}

	if got := node.Attribute("class"); got != "a c" {
		t.Fatalf("class after RemoveClass = %q, want %q", got, "a c")
	}

	// ToggleClass removes when present, adds when absent.
	added, ok := doc.ToggleClass("box", "a")
	if !ok || added {
		t.Fatalf("ToggleClass(a) = (%v, %v), want (false, true)", added, ok)
	}

	if got := node.Attribute("class"); got != "c" {
		t.Fatalf("class after toggle off = %q, want %q", got, "c")
	}

	added, ok = doc.ToggleClass("box", "a")
	if !ok || !added {
		t.Fatalf("ToggleClass(a) = (%v, %v), want (true, true)", added, ok)
	}

	if got := node.Attribute("class"); got != "c a" {
		t.Fatalf("class after toggle on = %q, want %q", got, "c a")
	}
}

func TestRemoveElement(t *testing.T) {
	t.Parallel()

	doc := mustParseDirty(t, `<div id="parent"><span id="child"><b id="grand">hi</b></span><em id="keep">ok</em></div>`)

	parent := doc.elementByID("parent")
	childVer0 := parent.ChildVersion
	mutVer0 := doc.root.MutationVersion

	// Missing, empty, nil are no-ops.
	if doc.RemoveElement("missing") || doc.RemoveElement("") {
		t.Fatal("RemoveElement reported true on missing/empty")
	}

	var nilDoc *Document
	if nilDoc.RemoveElement("child") {
		t.Fatal("nil Document.RemoveElement reported true")
	}

	if doc.root.MutationVersion != mutVer0 || parent.ChildVersion != childVer0 || doc.DirtyStructural() {
		t.Fatal("no-op RemoveElement mutated state")
	}

	// Removing child detaches subtree, clears both child and descendant ids,
	// bumps versions, marks parent dirty and structural.
	if !doc.RemoveElement("child") {
		t.Fatal("RemoveElement(child) reported false")
	}

	if parent.ChildVersion != childVer0+1 {
		t.Fatalf("parent.ChildVersion = %d, want %d", parent.ChildVersion, childVer0+1)
	}

	if doc.root.MutationVersion != mutVer0+1 {
		t.Fatalf("MutationVersion = %d, want %d", doc.root.MutationVersion, mutVer0+1)
	}

	if _, ok := doc.Find("child"); ok {
		t.Fatal("removed child still found")
	}

	if _, ok := doc.Find("grand"); ok {
		t.Fatal("removed descendant still found")
	}

	if elem, ok := doc.Find("keep"); !ok || elem.Text != "ok" {
		t.Fatalf("sibling lost: %+v, %v", elem, ok)
	}

	dirtyHas(t, doc, "parent")
	if !doc.DirtyStructural() {
		t.Fatal("RemoveElement did not set structural dirty flag")
	}

	// Refuse removing d.root even if someone attaches an id to it.
	doc.root.Attrs = map[string]string{"id": "docroot"}
	doc.root.AttrList = []ihtml.Attr{{Name: "id", Value: "docroot"}}
	doc.index.mu.Lock()
	doc.index.complete = false
	doc.index.mu.Unlock()

	if doc.RemoveElement("docroot") {
		t.Fatal("RemoveElement removed d.root")
	}
}

func TestRemoveElementDuplicateIDResolvesNext(t *testing.T) {
	t.Parallel()

	doc := mustParseDirty(t, `<div id="wrap"><span id="dup">first</span><span id="dup">second</span></div>`)

	if !doc.RemoveElement("dup") {
		t.Fatal("RemoveElement(dup) reported false")
	}

	elem, ok := doc.Find("dup")
	if !ok || elem.Text != "second" {
		t.Fatalf("Find(dup) after removing first = %+v, %v, want second", elem, ok)
	}
}

func TestSetInnerHTMLAndAppendHTML(t *testing.T) {
	t.Parallel()

	doc := mustParseDirty(t, `<div id="host"><span id="old">bye</span></div><div id="after"><p id="dup">later</p></div>`)

	// Warm index with existing dup.
	if elem, ok := doc.Find("dup"); !ok || elem.Text != "later" {
		t.Fatalf("initial Find(dup) = %+v, %v", elem, ok)
	}

	host := doc.elementByID("host")
	childVer0 := host.ChildVersion
	mutVer0 := doc.root.MutationVersion

	// Missing parent returns false, nil.
	if ok, err := doc.SetInnerHTML("missing", "<p>x</p>"); ok || err != nil {
		t.Fatalf("SetInnerHTML(missing) = (%v, %v), want (false, nil)", ok, err)
	}

	if ok, err := doc.AppendHTML("missing", "<p>x</p>"); ok || err != nil {
		t.Fatalf("AppendHTML(missing) = (%v, %v), want (false, nil)", ok, err)
	}

	// SetInnerHTML replaces children without html/head/body wrappers.
	ok, err := doc.SetInnerHTML("host", `<p id="dup">earlier</p><span id="a">one</span>`)
	if err != nil || !ok {
		t.Fatalf("SetInnerHTML = (%v, %v), want (true, nil)", ok, err)
	}

	if host.ChildVersion == childVer0 || doc.root.MutationVersion != mutVer0+1 {
		t.Fatalf("versions did not bump: child=%d->%d mut=%d->%d",
			childVer0, host.ChildVersion, mutVer0, doc.root.MutationVersion)
	}

	if len(host.Children) != 2 || host.Children[0].Name != "p" || host.Children[1].Name != "span" {
		t.Fatalf("unexpected children: %+v", host.Children)
	}

	for _, child := range host.Children {
		if child.Parent != host {
			t.Fatalf("child %q Parent != host", child.Name)
		}
	}

	if _, ok := doc.Find("old"); ok {
		t.Fatal("old child id still found after SetInnerHTML")
	}

	if elem, ok := doc.Find("a"); !ok || elem.Text != "one" {
		t.Fatalf("Find(a) = %+v, %v, want one", elem, ok)
	}

	// Newly inserted dup precedes later dup in document order.
	if elem, ok := doc.Find("dup"); !ok || elem.Text != "earlier" {
		t.Fatalf("Find(dup) = %+v, %v, want earlier", elem, ok)
	}

	dirtyHas(t, doc, "host")
	if !doc.DirtyStructural() {
		t.Fatal("SetInnerHTML did not set structural dirty flag")
	}

	// AppendHTML appends without dropping existing children.
	doc.ClearDirty()
	childVer1 := host.ChildVersion
	mutVer1 := doc.root.MutationVersion

	ok, err = doc.AppendHTML("host", `<em id="b">two</em>plain`)
	if err != nil || !ok {
		t.Fatalf("AppendHTML = (%v, %v), want (true, nil)", ok, err)
	}

	if host.ChildVersion != childVer1+1 || doc.root.MutationVersion != mutVer1+1 {
		t.Fatalf("AppendHTML versions: child=%d->%d mut=%d->%d",
			childVer1, host.ChildVersion, mutVer1, doc.root.MutationVersion)
	}

	if len(host.Children) != 4 {
		t.Fatalf("len(host.Children) = %d, want 4", len(host.Children))
	}

	if host.Children[3].Type != ihtml.TextNode || host.Children[3].Text != "plain" || host.Children[3].Parent != host {
		t.Fatalf("appended text node wrong: %+v", host.Children[3])
	}

	if elem, ok := doc.Find("b"); !ok || elem.Text != "two" {
		t.Fatalf("Find(b) = %+v, %v, want two", elem, ok)
	}

	dirtyHas(t, doc, "host")
	if !doc.DirtyStructural() {
		t.Fatal("AppendHTML did not set structural dirty flag")
	}

	// Empty SetInnerHTML clears children; calling again on empty is a no-op.
	doc.ClearDirty()
	if ok, err := doc.SetInnerHTML("host", ""); err != nil || !ok {
		t.Fatalf("SetInnerHTML(empty) = (%v, %v), want (true, nil)", ok, err)
	}

	if len(host.Children) != 0 {
		t.Fatalf("children not cleared: %+v", host.Children)
	}

	doc.ClearDirty()
	mutVerEmpty := doc.root.MutationVersion
	if ok, err := doc.SetInnerHTML("host", ""); err != nil || !ok {
		t.Fatalf("second SetInnerHTML(empty) = (%v, %v)", ok, err)
	}

	if doc.root.MutationVersion != mutVerEmpty || doc.DirtyStructural() {
		t.Fatal("SetInnerHTML(empty) on already empty element mutated state")
	}
}

func TestFragmentParsingTableAndSVGContexts(t *testing.T) {
	t.Parallel()

	doc := mustParseDirty(t, `<table><tbody id="tb"></tbody></table><svg id="canvas"></svg>`)

	if ok, err := doc.AppendHTML("tb", `<tr id="r1"><td id="c1">cell</td></tr>`); err != nil || !ok {
		t.Fatalf("AppendHTML(tbody) = (%v, %v)", ok, err)
	}

	if elem, ok := doc.Find("c1"); !ok || elem.Tag != "td" || elem.Text != "cell" {
		t.Fatalf("Find(c1) = %+v, %v, want td cell", elem, ok)
	}

	if ok, err := doc.SetInnerHTML("canvas", `<circle id="dot" cx="5" cy="5" r="3"></circle>`); err != nil || !ok {
		t.Fatalf("SetInnerHTML(svg) = (%v, %v)", ok, err)
	}

	dot := doc.elementByID("dot")
	if dot == nil || dot.Namespace != ihtml.NamespaceSVG {
		t.Fatalf("svg child = %+v, want NamespaceSVG", dot)
	}
}

func TestRetainedMutationsConcurrentRaceClean(t *testing.T) {
	t.Parallel()

	var sb strings.Builder
	sb.WriteString("<html><body>")
	for i := range 20 {
		fmt.Fprintf(&sb, `<div id="row%02d" class="item"><span id="lbl%02d">v%d</span></div>`, i, i, i)
	}
	sb.WriteString("</body></html>")

	doc := mustParseDirty(t, sb.String())

	var wg sync.WaitGroup

	for r := range 4 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for i := range 100 {
				_, _ = doc.Find(fmt.Sprintf("row%02d", (r+i)%20))
				_ = doc.DirtyIDs()
				_ = doc.DirtyStructural()
			}
		}()
	}

	for w := range 2 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for i := range 50 {
				id := fmt.Sprintf("row%02d", (w*10+i)%20)
				doc.AddClass(id, "active")
				doc.ToggleClass(id, "selected")
				doc.RemoveClass(id, "active")
				doc.SetAttribute(id, "data-i", fmt.Sprintf("%d", i))
				doc.RemoveAttribute(id, "data-i")

				if i%10 == 0 {
					_, _ = doc.AppendHTML(id, `<em>x</em>`)
					_, _ = doc.SetInnerHTML(id, fmt.Sprintf(`<span id="dyn%d_%d">ok</span>`, w, i))
					doc.RemoveElement(fmt.Sprintf("dyn%d_%d", w, i))
				}
			}
		}()
	}

	wg.Wait()

	if !slices.Contains(doc.DirtyIDs(), "row00") {
		t.Fatalf("expected row00 in DirtyIDs, got %q", doc.DirtyIDs())
	}
}
