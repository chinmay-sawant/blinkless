package html

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	ihtml "github.com/chinmay-sawant/blinkless/internal/html"
)

func mustParseDirty(t *testing.T, src string) *Document {
	t.Helper()

	doc, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	return doc
}

func dirtyHas(t *testing.T, doc *Document, id string) {
	t.Helper()

	for _, got := range doc.DirtyIDs() {
		if got == id {
			return
		}
	}

	t.Fatalf("dirty ids = %q, want %q marked", doc.DirtyIDs(), id)
}

func TestDirtySetMarkAndClear(t *testing.T) {
	t.Parallel()

	set := ihtml.NewDirtySet()
	if !set.IsEmpty() || set.Len() != 0 || set.Structural() {
		t.Fatal("new set is not empty")
	}

	set.Mark("b")
	set.Mark("a")
	set.Mark("a")
	set.Mark("")

	if set.Len() != 2 || !set.Contains("a") || !set.Contains("b") {
		t.Fatalf("marks did not land: %q", set.Snapshot())
	}

	if got := set.Snapshot(); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("snapshot = %q, want sorted [a b]", got)
	}

	set.MarkStructural()
	if !set.Structural() || set.IsEmpty() {
		t.Fatal("structural flag did not land")
	}

	// Marking is monotonic: a second mark keeps the entry and the flag.
	set.Mark("a")
	if set.Len() != 2 || !set.Structural() {
		t.Fatal("marking was not monotonic")
	}

	set.Clear()
	if !set.IsEmpty() || set.Len() != 0 || set.Structural() {
		t.Fatal("Clear did not reset the set")
	}

	// A nil set never panics and reads empty.
	var nilSet *ihtml.DirtySet
	nilSet.Mark("x")
	nilSet.MarkStructural()
	nilSet.Clear()
	if nilSet.Snapshot() != nil || nilSet.Contains("x") || nilSet.Len() != 0 ||
		nilSet.Structural() || !nilSet.IsEmpty() {
		t.Fatal("nil set did not read empty")
	}
}

func TestDirtyMarkOnSetAttribute(t *testing.T) {
	t.Parallel()

	doc := mustParseDirty(t, `<div id="box" class="old">Hello</div>`)
	if len(doc.DirtyIDs()) != 0 || doc.DirtyStructural() {
		t.Fatalf("fresh document is dirty: %q", doc.DirtyIDs())
	}

	if !doc.SetAttribute("box", "class", "new") {
		t.Fatal("SetAttribute reported no element")
	}

	dirtyHas(t, doc, "box")
	if doc.DirtyStructural() {
		t.Fatal("an attribute write set the structural flag")
	}
}

func TestDirtyMarkOnSetText(t *testing.T) {
	t.Parallel()

	doc := mustParseDirty(t, `<div id="outer"><span id="inner">x</span></div>`)

	if !doc.SetText("inner", "hi") {
		t.Fatal("SetText reported no element")
	}

	// The span keeps its single text node, so only the target and the
	// nearest block ancestor are dirty, without the structural flag.
	dirtyHas(t, doc, "inner")
	dirtyHas(t, doc, "outer")
	if doc.DirtyStructural() {
		t.Fatal("an in-place text update set the structural flag")
	}

	doc.ClearDirty()

	if !doc.SetText("outer", "hi") {
		t.Fatal("SetText reported no element")
	}

	// Dropping the span removes nodes, so the structural flag lands too.
	dirtyHas(t, doc, "outer")
	if !doc.DirtyStructural() {
		t.Fatal("dropping children did not set the structural flag")
	}
}

func TestDirtyParentChainStopsAtNearestBlock(t *testing.T) {
	t.Parallel()

	doc := mustParseDirty(t, `<div id="a"><div id="b"><span id="c">x</span></div></div>`)

	if !doc.SetText("c", "y") {
		t.Fatal("SetText reported no element")
	}

	dirtyHas(t, doc, "c")
	dirtyHas(t, doc, "b")

	for _, got := range doc.DirtyIDs() {
		if got == "a" {
			t.Fatalf("dirty ids = %q, want the chain to stop at block b", doc.DirtyIDs())
		}
	}
}

func TestDirtyRenameMarksOldAndNew(t *testing.T) {
	t.Parallel()

	doc := mustParseDirty(t, `<div id="box">Hi</div>`)

	if !doc.SetAttribute("box", "id", "box2") {
		t.Fatal("SetAttribute reported no element")
	}

	dirtyHas(t, doc, "box")
	dirtyHas(t, doc, "box2")
}

func TestDirtyClearResets(t *testing.T) {
	t.Parallel()

	doc := mustParseDirty(t, `<div id="box"><span>x</span></div>`)

	if !doc.SetAttribute("box", "class", "c") {
		t.Fatal("SetAttribute reported no element")
	}

	if !doc.SetText("box", "hi") {
		t.Fatal("SetText reported no element")
	}

	if len(doc.DirtyIDs()) == 0 || !doc.DirtyStructural() {
		t.Fatal("edits did not dirty the document")
	}

	doc.ClearDirty()

	if len(doc.DirtyIDs()) != 0 || doc.DirtyStructural() {
		t.Fatalf("ClearDirty did not reset: ids=%q structural=%v", doc.DirtyIDs(), doc.DirtyStructural())
	}

	// The next run starts clean and marks again.
	if !doc.SetAttribute("box", "class", "d") {
		t.Fatal("SetAttribute reported no element")
	}

	dirtyHas(t, doc, "box")
}

func TestDirtyNoMarkOnFailedWrites(t *testing.T) {
	t.Parallel()

	doc := mustParseDirty(t, `<div id="box">Hello</div>`)

	if doc.SetAttribute("other", "class", "new") {
		t.Fatal("SetAttribute found an element that is not there")
	}

	if doc.SetAttribute("box", "", "new") {
		t.Fatal("SetAttribute accepted an empty name")
	}

	if doc.SetAttribute("", "class", "new") {
		t.Fatal("SetAttribute accepted an empty id")
	}

	if doc.SetText("other", "Bye") {
		t.Fatal("SetText found an element that is not there")
	}

	if len(doc.DirtyIDs()) != 0 || doc.DirtyStructural() {
		t.Fatalf("failed writes dirtied the document: %q", doc.DirtyIDs())
	}

	// Nil receivers never panic and read clean.
	var nilDoc *Document
	if nilDoc.SetAttribute("box", "class", "new") || nilDoc.SetText("box", "hi") {
		t.Fatal("nil document reported a write")
	}

	if nilDoc.DirtyIDs() != nil || nilDoc.DirtyStructural() {
		t.Fatal("nil document reads dirty")
	}

	nilDoc.ClearDirty()
}

func TestDirtyConcurrentMarkingRaceClean(t *testing.T) {
	var sb strings.Builder

	sb.WriteString("<html><body>")
	for i := range 50 {
		fmt.Fprintf(&sb, `<div id="e%02d"><span id="s%02d">text %d</span></div>`, i, i, i)
	}
	sb.WriteString("</body></html>")

	doc := mustParseDirty(t, sb.String())

	var wg sync.WaitGroup

	for w := range 4 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for i := range 200 {
				id := fmt.Sprintf("e%02d", (w+i)%50)
				doc.SetAttribute(id, "data-k", "v")

				if i%50 == 0 {
					doc.SetText(fmt.Sprintf("s%02d", (w+i)%50), "scratch")
				}

				_ = doc.DirtyIDs()
				_ = doc.DirtyStructural()
			}
		}()
	}

	wg.Wait()

	// The final state only needs to be readable; concurrent marks make the
	// exact contents nondeterministic, and the race detector guards sharing.
	_ = doc.DirtyIDs()
	doc.ClearDirty()

	if len(doc.DirtyIDs()) != 0 || doc.DirtyStructural() {
		t.Fatal("ClearDirty did not reset after concurrent marks")
	}
}
