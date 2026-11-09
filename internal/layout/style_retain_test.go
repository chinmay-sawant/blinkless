package layout

import (
	"sync"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// testDirtySet is the in-test stand-in for the dirty-set worker's
// html/dirty.go set: a thread-safe set with Mark, MarkSubtree, Clear, and
// Snapshot. Subtree expansion happens in the resolver's tree walk, so
// MarkSubtree(id) and Mark(id) carry the same information.
type testDirtySet struct {
	mu  sync.Mutex
	ids map[string]struct{}
}

var _ styleDirtySet = (*testDirtySet)(nil)

func newTestDirtySet() *testDirtySet {
	return &testDirtySet{ids: make(map[string]struct{})}
}

func (d *testDirtySet) Mark(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.ids[id] = struct{}{}
}

func (d *testDirtySet) MarkSubtree(id string) {
	d.Mark(id)
}

func (d *testDirtySet) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.ids = make(map[string]struct{})
}

func (d *testDirtySet) Snapshot() []string {
	d.mu.Lock()
	defer d.mu.Unlock()

	out := make([]string, 0, len(d.ids))
	for id := range d.ids {
		out = append(out, id)
	}

	return out
}

func retainTestRoot(t *testing.T) *html.Node {
	t.Helper()

	return mustParse(t, `<html><body><div id="a">alpha <span id="c">deep</span></div>`+
		`<div id="b">beta</div></body></html>`)
}

func retainTestOpts(sheets []*css.Stylesheet) Options {
	return Options{
		Width: testViewport, Height: 800, Media: "print", Sheets: sheets,
	}
}

func findByID(root *html.Node, id string) *html.Node {
	var found *html.Node

	var walk func(node *html.Node)
	walk = func(node *html.Node) {
		if found != nil {
			return
		}

		if node.Type == html.ElementNode && node.Attribute("id") == id {
			found = node

			return
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(root)

	return found
}

func retainFirstPass(
	t *testing.T, root *html.Node, opts Options, dirty *testDirtySet, ret *styleRetainer,
) map[*html.Node]*ResolvedStyle {
	t.Helper()

	styles, err := resolveStylesRetainedWithContext(t.Context(), root, opts, nil, dirty, ret)
	if err != nil {
		t.Fatalf("retained resolve: %v", err)
	}

	return styles
}

// TestRetainCleanSubtreeReuse proves a clean subtree reuses retained styles
// verbatim: the second resolve returns the same interned pointers the full
// path stocked, with no re-resolution.
func TestRetainCleanSubtreeReuse(t *testing.T) {
	t.Parallel()

	root := retainTestRoot(t)
	sheets := []*css.Stylesheet{sheet(t, `#a{color:red} #b{color:blue} #c{font-size:20pt}`)}
	opts := retainTestOpts(sheets)
	dirty := newTestDirtySet()
	ret := newStyleRetainer()

	first := retainFirstPass(t, root, opts, dirty, ret)
	if got := ret.fullCount(); got != 1 {
		t.Fatalf("first resolve took the full path %d times, want 1", got)
	}

	nodeA, nodeB, nodeC := findByID(root, "a"), findByID(root, "b"), findByID(root, "c")
	ptrA, ptrB, ptrC := first[nodeA], first[nodeB], first[nodeC]
	if ptrA == nil || ptrB == nil || ptrC == nil {
		t.Fatal("missing styles for id-bearing elements")
	}

	if first[nodeA].Color == first[nodeB].Color {
		t.Fatal("sibling subtrees share one color, want distinct sheet colors")
	}

	if first[nodeC].Color != first[nodeA].Color {
		t.Fatal("child did not inherit the retained parent color")
	}

	second := retainFirstPass(t, root, opts, dirty, ret)
	if got := ret.retainedCount(); got != 1 {
		t.Fatalf("incremental resolves = %d, want 1", got)
	}

	if second[nodeA] != ptrA || second[nodeB] != ptrB || second[nodeC] != ptrC {
		t.Fatal("clean subtrees did not reuse retained styles verbatim")
	}

	if got := ret.reusedCount(); got < 3 {
		t.Fatalf("reused clean subtrees = %d, want at least 3", got)
	}
}

// TestRetainDirtySubtreeReresolves proves a dirty subtree re-resolves
// itself and its descendants against the retained parent chain while a
// clean sibling reuses its retained style verbatim.
func TestRetainDirtySubtreeReresolves(t *testing.T) {
	t.Parallel()

	root := retainTestRoot(t)
	sheets := []*css.Stylesheet{sheet(t, `#a{color:red} #b{color:blue}`)}
	opts := retainTestOpts(sheets)
	dirty := newTestDirtySet()
	ret := newStyleRetainer()

	first := retainFirstPass(t, root, opts, dirty, ret)
	nodeA, nodeB, nodeC := findByID(root, "a"), findByID(root, "b"), findByID(root, "c")
	ptrA, ptrB, ptrC := first[nodeA], first[nodeB], first[nodeC]

	nodeA.Attrs["style"] = "color: green"
	dirty.Mark("a")

	second := retainFirstPass(t, root, opts, dirty, ret)
	if got := ret.retainedCount(); got != 1 {
		t.Fatalf("incremental resolves = %d, want 1", got)
	}

	if second[nodeA] == ptrA {
		t.Fatal("dirty subtree root reused its retained style")
	}

	if second[nodeA].Color == ptrA.Color {
		t.Fatal("dirty subtree root kept its old color after the inline edit")
	}

	if second[nodeB] != ptrB {
		t.Fatal("clean sibling did not reuse its retained style verbatim")
	}

	if second[nodeC] == ptrC {
		t.Fatal("descendant of a changed parent reused its retained style")
	}

	if second[nodeC].Color != second[nodeA].Color {
		t.Fatal("re-resolved descendant did not inherit the new parent color")
	}
}

// TestRetainStructuralFallsBack proves a structural edit (an inserted
// element changes the document element count) falls back to the existing
// full resolve path and still styles the new node.
func TestRetainStructuralFallsBack(t *testing.T) {
	t.Parallel()

	root := retainTestRoot(t)
	sheets := []*css.Stylesheet{sheet(t, `#a{color:red} #n{color:blue}`)}
	opts := retainTestOpts(sheets)
	dirty := newTestDirtySet()
	ret := newStyleRetainer()

	first := retainFirstPass(t, root, opts, dirty, ret)
	nodeA := findByID(root, "a")
	wantColor := first[nodeA].Color

	body := findElementByName(root, "body")
	if body == nil {
		t.Fatal("no body element")
	}

	added := &html.Node{
		Type: html.ElementNode, Name: "p",
		Attrs:  map[string]string{"id": "n"},
		Parent: body,
	}
	body.Children = append(body.Children, added)

	second := retainFirstPass(t, root, opts, dirty, ret)
	if got := ret.fullCount(); got != 2 {
		t.Fatalf("full resolves = %d, want 2 (stock plus structural fallback)", got)
	}

	if second[nodeA].Color != wantColor {
		t.Fatal("structural fallback changed the untouched subtree style")
	}

	fresh := resolveStyles(root, sheets, "print", testViewport, 800)
	if second[added] == nil || fresh[added] == nil {
		t.Fatal("missing style for the inserted element")
	}

	if second[added].Color != fresh[added].Color {
		t.Fatal("fallback resolve disagrees with the full path on the inserted element")
	}
}

// TestRetainRuleMutationNeedsFreshParse preserves the existing contract:
// sheets are keyed by identity, so a fresh parse of changed rules misses
// the retainer, falls back to the full path, and serves the new values.
func TestRetainRuleMutationNeedsFreshParse(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div id="t" class="t">x</div></body></html>`)
	firstSheets := []*css.Stylesheet{sheet(t, `.t{color:red}`)}
	opts := retainTestOpts(firstSheets)
	dirty := newTestDirtySet()
	ret := newStyleRetainer()

	first := retainFirstPass(t, root, opts, dirty, ret)
	nodeT := findByID(root, "t")
	red := first[nodeT].Color

	secondSheets := []*css.Stylesheet{sheet(t, `.t{color:blue}`)}
	second := retainFirstPass(t, root, retainTestOpts(secondSheets), dirty, ret)
	if got := ret.fullCount(); got != 2 {
		t.Fatalf("full resolves = %d, want 2 (stock plus fresh-sheet fallback)", got)
	}

	if second[nodeT].Color == red {
		t.Fatal("retained resolve served the stale sheet after a fresh parse")
	}

	fresh := resolveStyles(root, secondSheets, "print", testViewport, 800)
	if second[nodeT].Color != fresh[nodeT].Color {
		t.Fatal("fallback resolve disagrees with the full path after a fresh parse")
	}
}

// TestRetainConcurrentLayouts proves concurrent layouts share one retainer
// race-clean: every goroutine resolves the stocked document and reads the
// same retained styles.
func TestRetainConcurrentLayouts(t *testing.T) {
	t.Parallel()

	root := retainTestRoot(t)
	sheets := []*css.Stylesheet{sheet(t, `#a{color:red} #b{color:blue}`)}
	opts := retainTestOpts(sheets)
	ret := newStyleRetainer()

	first := retainFirstPass(t, root, opts, newTestDirtySet(), ret)
	nodeA := findByID(root, "a")
	want := first[nodeA].Color

	const workers = 8

	var wg sync.WaitGroup

	errs := make(chan string, workers)

	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			styles, err := resolveStylesRetainedWithContext(
				t.Context(), root, opts, nil, newTestDirtySet(), ret,
			)
			if err != nil {
				errs <- err.Error()

				return
			}

			if styles[nodeA].Color != want {
				errs <- "concurrent resolve disagrees on the retained color"
			}
		}()
	}

	wg.Wait()
	close(errs)

	for msg := range errs {
		t.Fatal(msg)
	}
}
