package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// This file holds the string-set behavior tests. The table tests assert the
// used value, not the declaration text. Paint tests assert OpText. A
// string() read on the same canvas paints the first assignment. Page-margin
// running headers are not painted. TestBehaviorStringSetNoPaintEffect is the
// case with no string() reader: the heading's drawing list does not change.

// TestBehaviorStringSetFirstAssignmentWins proves set-then-read on used
// values: the first element assigning chapter wins, and adjacent strings
// concatenate ('Guide ' + 'One' = 'Guide One'), so the test reads an
// evaluated value, not the declaration text.
func TestBehaviorStringSetFirstAssignmentWins(t *testing.T) {
	t.Parallel()

	src := `<html><body>` +
		`<h2 style="string-set: chapter 'Guide ' 'One'">Guide One</h2>` +
		`<h2 style="string-set: chapter 'Beta'">Beta</h2>` +
		`</body></html>`

	root := mustParse(t, src)

	if _, err := Layout(root, Options{Width: testViewport, Height: 800, Background: true}); err != nil {
		t.Fatalf("Layout: %v", err)
	}

	got, ok := LookupNamedString(root, "chapter")
	if !ok {
		t.Fatalf("Lookup(chapter) missed, want used value %q", "Guide One")
	}

	if got != "Guide One" {
		t.Errorf("Lookup(chapter) = %q, want used value %q (concatenated strings)", got, "Guide One")
	}
}

// TestBehaviorStringSetMultipleNames proves one declaration can assign two
// names at once via the comma list.
func TestBehaviorStringSetMultipleNames(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>`+
		`<div style="string-set: header 'Top', footer 'Bottom'">body</div>`+
		`</body></html>`)

	table := CollectNamedStrings(root)

	for name, want := range map[string]string{"header": "Top", "footer": "Bottom"} {
		got, ok := table.Lookup(name)
		if !ok {
			t.Errorf("Lookup(%s) missed, want used value %q", name, want)

			continue
		}

		if got != want {
			t.Errorf("Lookup(%s) = %q, want used value %q", name, got, want)
		}
	}
}

// TestBehaviorStringSetNoneAndInvalidIgnored proves none, a bare identifier
// with no string, a function value, and a non-identifier are all validly
// ignored (no table entry), while a later valid assignment still lands.
func TestBehaviorStringSetNoneAndInvalidIgnored(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>`+
		`<p style="string-set: none">a</p>`+
		`<p style="string-set: chapter">b</p>`+
		`<p style="string-set: chapter counter(page)">c</p>`+
		`<p style="string-set: 12px">d</p>`+
		`<p style="string-set: chapter 'Kept'">e</p>`+
		`</body></html>`)

	table := CollectNamedStrings(root)

	if table.Len() != 1 {
		t.Errorf("table length = %d, want 1 (only the valid assignment lands)", table.Len())
	}

	got, ok := table.Lookup("chapter")
	if !ok {
		t.Fatalf("Lookup(chapter) missed, want used value %q", "Kept")
	}

	if got != "Kept" {
		t.Errorf("Lookup(chapter) = %q, want used value %q", got, "Kept")
	}

	if _, ok := table.Lookup("missing"); ok {
		t.Errorf("Lookup(missing) hit, want miss")
	}
}

// TestBehaviorStringSetEscapesDecode proves the used value decodes escapes:
// 'a\'b' stores a'b, not the raw five characters.
func TestBehaviorStringSetEscapesDecode(t *testing.T) {
	t.Parallel()

	assignments, ok := ParseStringSet(`chapter 'a\'b'`)
	if !ok {
		t.Fatalf("ParseStringSet missed, want one assignment")
	}

	if len(assignments) != 1 {
		t.Fatalf("assignment count = %d, want 1", len(assignments))
	}

	if assignments[0].Value != "a'b" {
		t.Errorf("used value = %q, want %q (decoded escape)", assignments[0].Value, "a'b")
	}
}

// TestBehaviorStringSetNoPaintEffect pins the case with no string() reader:
// assigning a name does not change the heading's drawing list.
func TestBehaviorStringSetNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := `<html style="margin:0"><body style="margin:0"><h2>Guide One</h2></body></html>`
	styled := `<html style="margin:0"><body style="margin:0">` +
		`<h2 style="string-set: chapter 'Guide One'">Guide One</h2></body></html>`

	behaviorColumnsAssertSameOps(t, layoutHTML(t, plain), layoutHTML(t, styled))
}

// TestBehaviorStringSetPaintsOnSameCanvas proves a later string() read paints
// the assigned text on this canvas. The heading's own text op stays. A later
// assignment does not replace the first. none clears the name. An unknown
// name adds nothing. A bad token leaves the earlier value. content() and
// content(text) both read the element text, and content(before) with no
// ::before contributes nothing.
func TestBehaviorStringSetPaintsOnSameCanvas(t *testing.T) {
	t.Parallel()

	t.Run("fixture", func(t *testing.T) {
		t.Parallel()

		texts := stringSetTextOps(t, layoutStringSetFixture(t))
		if got := stringSetCount(texts, "Guide One"); got != 2 {
			t.Fatalf("Guide One ops = %d, texts %q, want 2 (heading plus reader)", got, texts)
		}
	})

	t.Run("first assignment wins", func(t *testing.T) {
		t.Parallel()

		body := `<html style="margin:0"><body style="margin:0">` +
			`<h1 style="string-set: chapter content(text)">Guide One</h1>` +
			`<h1 style="string-set: chapter content(text)">Beta</h1>` +
			`<p id="read"></p></body></html>`
		texts := stringSetTextOps(t, layoutHTML(t, body, sheet(t, `#read::before { content: string(chapter); }`)))

		if got := stringSetCount(texts, "Guide One"); got != 2 {
			t.Fatalf("Guide One ops = %d, texts %q, want 2 (heading plus reader, Beta does not replace)", got, texts)
		}

		if got := stringSetCount(texts, "Beta"); got != 1 {
			t.Fatalf("Beta ops = %d, texts %q, want 1 (the heading only)", got, texts)
		}
	})

	t.Run("none clears", func(t *testing.T) {
		t.Parallel()

		body := `<html style="margin:0"><body style="margin:0">` +
			`<h1 style="string-set: chapter content(text)">Guide One</h1>` +
			`<p style="string-set: none"></p>` +
			`<p id="read"></p></body></html>`
		texts := stringSetTextOps(t, layoutHTML(t, body, sheet(t, `#read::before { content: "X" string(chapter) "Y"; }`)))

		if got := stringSetCount(texts, "Guide One"); got != 1 {
			t.Fatalf("Guide One ops = %d, texts %q, want 1 (heading only, none cleared the reader)", got, texts)
		}

		if got := stringSetCount(texts, "XY"); got != 1 {
			t.Fatalf("reader ops = %d, texts %q, want XY (string() contributed nothing)", got, texts)
		}
	})

	t.Run("unknown name", func(t *testing.T) {
		t.Parallel()

		body := `<html style="margin:0"><body style="margin:0">` +
			`<h1 style="string-set: chapter content(text)">Guide One</h1>` +
			`<p id="read"></p></body></html>`
		texts := stringSetTextOps(t, layoutHTML(t, body, sheet(t, `#read::before { content: "X" string(missing) "Y"; }`)))

		if got := stringSetCount(texts, "XY"); got != 1 {
			t.Fatalf("reader ops = %d, texts %q, want XY", got, texts)
		}

		if got := stringSetCount(texts, "Guide One"); got != 1 {
			t.Fatalf("Guide One ops = %d, texts %q, want 1 (the heading; the unknown name is not extra text)", got, texts)
		}

		for _, text := range texts {
			if strings.Contains(text, "missing") {
				t.Fatalf("unknown name painted in %q", text)
			}
		}
	})

	t.Run("bad token keeps earlier", func(t *testing.T) {
		t.Parallel()

		body := `<html style="margin:0"><body style="margin:0">` +
			`<h1 style="string-set: chapter content(text)">Guide One</h1>` +
			`<h1 style="string-set: chapter @@">Beta</h1>` +
			`<p id="read"></p></body></html>`
		texts := stringSetTextOps(t, layoutHTML(t, body, sheet(t, `#read::before { content: string(chapter); }`)))

		if got := stringSetCount(texts, "Guide One"); got != 2 {
			t.Fatalf("Guide One ops = %d, texts %q, want 2 (bad token did not replace the first assignment)", got, texts)
		}

		if got := stringSetCount(texts, "Beta"); got != 1 {
			t.Fatalf("Beta ops = %d, texts %q, want 1 (the heading's own text)", got, texts)
		}
	})

	t.Run("content keywords", func(t *testing.T) {
		t.Parallel()

		body := `<html style="margin:0"><body style="margin:0">` +
			`<h1 style="string-set: plain content(), typed content(text), bare content(before)">Guide One</h1>` +
			`<p id="plain"></p><p id="typed"></p><p id="bare"></p></body></html>`
		rule := `#plain::before { content: string(plain); }` +
			`#typed::before { content: string(typed); }` +
			`#bare::before { content: "B" string(bare) "E"; }`
		texts := stringSetTextOps(t, layoutHTML(t, body, sheet(t, rule)))

		if got := stringSetCount(texts, "Guide One"); got != 3 {
			t.Fatalf("Guide One ops = %d, texts %q, want 3 (heading, content(), and content(text))", got, texts)
		}

		if got := stringSetCount(texts, "BE"); got != 1 {
			t.Fatalf("bare reader = %d, texts %q, want BE (content(before) contributed nothing)", got, texts)
		}
	})
}

// layoutStringSetFixture lays out testdata/css/v012/string-set/page.html
// with the <style> block applied. layoutHTML does not collect document styles.
func layoutStringSetFixture(t *testing.T) *Result {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(
		"..", "..", "testdata", "css", "v012", "string-set", "page.html",
	))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	root := mustParse(t, string(raw))

	var sheets []*css.Stylesheet

	root.Walk(func(node *html.Node) {
		if node.Type != html.ElementNode || node.Name != "style" {
			return
		}

		var text string

		for _, child := range node.Children {
			if child.Type == html.TextNode {
				text += child.Text
			}
		}

		parsed, parseErr := css.Parse(text)
		if parseErr != nil {
			t.Fatalf("css.Parse: %v", parseErr)
		}

		sheets = append(sheets, parsed)
	})

	res, err := Layout(root, Options{
		Width: testViewport, Height: 800, Sheets: sheets, Background: true,
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	return res
}

// stringSetTextOps returns the non-empty text operations in paint order.
func stringSetTextOps(t *testing.T, res *Result) []string {
	t.Helper()

	if res == nil {
		t.Fatal("nil result")
	}

	texts := make([]string, 0, 4)

	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text != "" {
			texts = append(texts, op.Text)
		}
	}

	return texts
}

// stringSetCount counts text operations equal to want.
func stringSetCount(texts []string, want string) int {
	count := 0

	for _, text := range texts {
		if text == want {
			count++
		}
	}

	return count
}
