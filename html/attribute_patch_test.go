package html

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	ihtml "github.com/chinmay-sawant/blinkless/internal/html"
)

// budgetPatch1000 bounds 1000 SetAttribute calls on a 1000-element document.
// Measured about 0.4ms on the dev host; the budget is double that rounded
// up with a 2s floor so loaded CI stays green. It guards against a return
// to superlinear lookup cost, not small constant shifts.
const budgetPatch1000 = 2 * time.Second

// buildPatchDoc parses one document with count id-bearing divs.
func buildPatchDoc(t *testing.T, count int) *Document {
	t.Helper()

	var sb strings.Builder

	sb.WriteString("<html><body>")

	for i := range count {
		fmt.Fprintf(&sb, `<div id="e%04d" class="c">text %d</div>`, i, i)
	}

	sb.WriteString("</body></html>")

	doc, err := Parse([]byte(sb.String()))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	return doc
}

func TestPatchIDLookupStaysCheap(t *testing.T) {
	t.Parallel()

	doc := buildPatchDoc(t, 1000)

	start := time.Now()

	for i := range 1000 {
		id := fmt.Sprintf("e%04d", i)
		if !doc.SetAttribute(id, "data-n", "v") {
			t.Fatalf("SetAttribute(%q) reported no element", id)
		}
	}

	elapsed := time.Since(start)
	t.Logf("1000 SetAttribute calls on a 1000-element doc took %v", elapsed)

	if elapsed > budgetPatch1000 {
		t.Fatalf("1000 patches took %v, budget %v", elapsed, budgetPatch1000)
	}

	node := doc.elementByID("e0999")
	if node == nil || node.Attribute("data-n") != "v" {
		t.Fatalf("last patch did not land: node=%+v", node)
	}
}

func TestPatchSemanticsParity(t *testing.T) {
	t.Parallel()

	t.Run("set keeps stored spelling", func(t *testing.T) {
		t.Parallel()

		patchSetKeepsStoredSpelling(t)
	})

	t.Run("missing element and empty name", func(t *testing.T) {
		t.Parallel()

		patchMissingElementAndEmptyName(t)
	})

	t.Run("empty text clears children", func(t *testing.T) {
		t.Parallel()

		patchEmptyTextClearsChildren(t)
	})

	t.Run("text replace reuses the single text node", func(t *testing.T) {
		t.Parallel()

		patchTextReplaceReusesNode(t)
	})

	t.Run("renamed id resolves to the same node", func(t *testing.T) {
		t.Parallel()

		patchRenamedIDResolves(t)
	})

	t.Run("text replace forgets dropped ids only", func(t *testing.T) {
		t.Parallel()

		patchTextReplaceForgetsDroppedIDs(t)
	})

	t.Run("duplicate ids resolve to the first", func(t *testing.T) {
		t.Parallel()

		patchDuplicateIDsResolveFirst(t)
	})
}

func patchSetKeepsStoredSpelling(t *testing.T) {
	t.Helper()

	doc, err := Parse([]byte(`<div id="box" data-x="1">Hi</div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if !doc.SetAttribute("box", "DATA-X", "2") {
		t.Fatal("SetAttribute reported no element")
	}

	node := doc.elementByID("box")
	if node == nil {
		t.Fatal("box not found")
	}

	if got := node.Attribute("data-x"); got != "2" {
		t.Fatalf("data-x = %q, want %q", got, "2")
	}

	if len(node.AttrList) != 2 {
		t.Fatalf("AttrList = %+v, want no duplicate entry", node.AttrList)
	}
}

func patchMissingElementAndEmptyName(t *testing.T) {
	t.Helper()

	doc, err := Parse([]byte(`<div id="box">Hi</div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

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
}

func patchEmptyTextClearsChildren(t *testing.T) {
	t.Helper()

	doc, err := Parse([]byte(`<div id="box"><span>Hi</span></div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if !doc.SetText("box", "") {
		t.Fatal("SetText reported no element")
	}

	node := doc.elementByID("box")
	if node == nil {
		t.Fatal("box not found")
	}

	if len(node.Children) != 0 || node.TextContent() != "" {
		t.Fatalf("children = %+v, want none", node.Children)
	}
}

func patchTextReplaceReusesNode(t *testing.T) {
	t.Helper()

	doc, err := Parse([]byte(`<div id="box">Hello</div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	before := doc.elementByID("box").Children[0]

	if !doc.SetText("box", "Bye") {
		t.Fatal("SetText reported no element")
	}

	node := doc.elementByID("box")
	if len(node.Children) != 1 || node.Children[0].Type != ihtml.TextNode {
		t.Fatalf("children = %+v, want one text node", node.Children)
	}

	if node.Children[0] != before {
		t.Fatal("SetText rebuilt a text node it could have reused")
	}

	if node.TextContent() != "Bye" {
		t.Fatalf("text = %q, want %q", node.TextContent(), "Bye")
	}
}

func patchRenamedIDResolves(t *testing.T) {
	t.Helper()

	doc, err := Parse([]byte(`<div id="box" class="c">Hi</div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if !doc.SetAttribute("box", "id", "box2") {
		t.Fatal("SetAttribute reported no element")
	}

	if node := doc.elementByID("box"); node != nil {
		t.Fatal("old id still resolves")
	}

	node := doc.elementByID("box2")
	if node == nil {
		t.Fatal("new id not found")
	}

	if got := node.Attribute("class"); got != "c" {
		t.Fatalf("class = %q, want %q", got, "c")
	}
}

func patchTextReplaceForgetsDroppedIDs(t *testing.T) {
	t.Helper()

	doc, err := Parse([]byte(`<div id="outer"><span id="inner">x</span></div><div id="sib">y</div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if !doc.SetText("outer", "hi") {
		t.Fatal("SetText reported no element")
	}

	if node := doc.elementByID("inner"); node != nil {
		t.Fatal("dropped id still resolves")
	}

	if node := doc.elementByID("sib"); node == nil {
		t.Fatal("sibling id lost")
	}

	if node := doc.elementByID("outer"); node == nil || node.TextContent() != "hi" {
		t.Fatalf("outer text wrong: node=%+v", node)
	}
}

func patchDuplicateIDsResolveFirst(t *testing.T) {
	t.Helper()

	doc, err := Parse([]byte(`<div id="dup" class="first">a</div><div id="dup" class="second">b</div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	node := doc.elementByID("dup")
	if node == nil || node.Attribute("class") != "first" {
		t.Fatalf("dup resolved to %+v, want the first element", node)
	}
}

func TestPatchConcurrentReadsRaceClean(t *testing.T) {
	t.Parallel()

	doc := buildPatchDoc(t, 200)

	const readers = 4
	const rounds = 500

	var wg sync.WaitGroup

	for r := range readers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			id := fmt.Sprintf("e%04d", r)

			for range rounds {
				if node := doc.elementByID(id); node == nil {
					t.Error("element vanished under concurrent reads")
					return
				}
			}
		}()
	}

	for i := range rounds {
		id := fmt.Sprintf("e%04d", 100+(i%50))
		if !doc.SetAttribute(id, "data-k", "v") {
			t.Fatalf("SetAttribute(%q) reported no element", id)
		}

		if i%100 == 0 && !doc.SetText("e0199", "scratch") {
			t.Fatal("SetText reported no element")
		}
	}

	wg.Wait()
}
