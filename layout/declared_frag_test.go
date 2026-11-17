package layout_test

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

func fragDoc(t *testing.T, page string) *css.Document {
	t.Helper()

	doc, err := html.Parse([]byte(page))
	if err != nil {
		t.Fatal(err)
	}

	styled, err := css.Apply(t.Context(), doc, css.Options{WidthPx: 300, HeightPx: 200, Media: "screen"})
	if err != nil {
		t.Fatal(err)
	}

	return styled
}

const outlinePage = `<!DOCTYPE html><html><body>
<h1 style="bookmark-level:1">Intro</h1>
<h2 style="bookmark-level:2">Part A</h2>
<h2 style="bookmark-level:2; bookmark-label: 'Part ' attr(data-n); bookmark-state: closed" data-n="B">raw</h2>
<h3 style="bookmark-level:3; bookmark-label: content 'x'">Deep</h3>
<h2 style="bookmark-level:none">Hidden</h2>
<h2 style="bookmark-level:2; bookmark-label:none">NoLabel</h2>
<h1 style="bookmark-level:1; bookmark-state:open">Next</h1>
</body></html>`

// TestOutlineFromBookmarks: levels nest, labels come from text, strings and
// attr(), state closed is carried, and none leaves an element out.
func TestOutlineFromBookmarks(t *testing.T) {
	t.Parallel()

	got, err := layout.Outline(t.Context(), fragDoc(t, outlinePage))
	if err != nil {
		t.Fatal(err)
	}

	want := []layout.OutlineEntry{
		{Label: "Intro", Level: 1, Open: true, Parent: -1},
		{Label: "Part A", Level: 2, Open: true, Parent: 0},
		{Label: "Part B", Level: 2, Open: false, Parent: 0},
		{Label: "Deepx", Level: 3, Open: true, Parent: 2},
		{Label: "Next", Level: 1, Open: true, Parent: -1},
	}

	if len(got) != len(want) {
		t.Fatalf("outline = %+v, want %d entries", got, len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestOutlineLeavesPaintAlone: the heading text on the canvas is its own text,
// not the label.
func TestOutlineLeavesPaintAlone(t *testing.T) {
	t.Parallel()

	display := renderDisplay(t, `<h1 style="bookmark-level:1; bookmark-label:'Other'">Shown</h1>`, "")

	var text strings.Builder

	for _, op := range display.Ops {
		text.WriteString(op.Text)
	}

	if !strings.Contains(text.String(), "Shown") || strings.Contains(text.String(), "Other") {
		t.Errorf("painted text %q", text.String())
	}
}

// TestCopiedFragments: copy-into hands the host named text; none adds nothing.
func TestCopiedFragments(t *testing.T) {
	t.Parallel()

	page := `<h1 id="a" style="copy-into: chapter-title content, chapter-id attr(data-id)" ` +
		`data-id="c7">Chapter Seven</h1>` +
		`<h1 style="copy-into: none">Skipped</h1><p>plain</p>`

	got, err := layout.CopiedFragments(t.Context(), fragDoc(t, page))
	if err != nil {
		t.Fatal(err)
	}

	want := []layout.CopiedFragment{
		{Name: "chapter-title", Level: "content", Text: "Chapter Seven"},
		{Name: "chapter-id", Level: "attr(data-id)", Text: "c7"},
	}

	if len(got) != len(want) {
		t.Fatalf("fragments = %+v", got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("fragment %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestFragDeclaredValues: grammar of the seven frag properties.
func TestFragDeclaredValues(t *testing.T) {
	t.Parallel()

	good := map[string]string{
		"bookmark-label":   "'A' content attr(title)",
		"bookmark-level":   "3",
		"bookmark-state":   "closed",
		"footnote-display": "compact",
		"footnote-policy":  "line",
		"copy-into":        "name content, other attr(x)",
		"block-ellipsis":   "' [more]'",
	}

	bad := map[string]string{
		"bookmark-label":   "bogus(1)",
		"bookmark-level":   "0",
		"bookmark-state":   "half",
		"footnote-display": "float",
		"footnote-policy":  "never",
		"copy-into":        "name",
		"block-ellipsis":   "unquoted",
	}

	for prop, value := range good {
		d := renderDisplay(t, `<div id="x" style="`+prop+`: `+value+`">t</div>`, "")
		if boxByID(t, d, "x").Declared[prop] == "" {
			t.Errorf("%s: %s should be accepted", prop, value)
		}
	}

	for prop, value := range bad {
		d := renderDisplay(t, `<div id="x" style="`+prop+`: `+value+`">t</div>`, "")
		if got := boxByID(t, d, "x").Declared[prop]; got != "" {
			t.Errorf("%s: %s should be dropped, got %q", prop, value, got)
		}
	}

	d := renderDisplay(t, `<div style="block-ellipsis: ellipsis"><div id="c">t</div></div>`+
		`<div id="n" style="bookmark-level: 2"><div id="m">t</div></div>`, "")
	if boxByID(t, d, "c").Declared["block-ellipsis"] == "" {
		t.Error("block-ellipsis should inherit")
	}

	if _, ok := boxByID(t, d, "m").Declared["bookmark-level"]; ok {
		t.Error("bookmark-level must not inherit")
	}
}

// ellipsisText lays out a clamped paragraph and returns the painted text.
func ellipsisText(t *testing.T, decl string) string {
	t.Helper()

	prose := "The quick brown fox jumps over the lazy dog near the river bank " +
		"while the red fox watches the slow green turtle cross the road again."
	d := renderDisplay(t, `<p style="margin:0;width:200px;font-size:16px;`+decl+`">`+prose+`</p>`, "")

	var out strings.Builder

	for _, op := range d.Ops {
		if op.Kind == layout.DisplayOpText {
			out.WriteString(op.Text)
		}
	}

	return out.String()
}

// TestBlockEllipsisMarker: on a block cut by line-clamp, the last visible line
// ends with the marker block-ellipsis names. Without a clamp nothing is added.
func TestBlockEllipsisMarker(t *testing.T) {
	t.Parallel()

	if got := ellipsisText(t, "line-clamp:2"); !strings.Contains(got, "…") {
		t.Errorf("default marker missing: %q", got)
	}

	got := ellipsisText(t, "line-clamp:2; block-ellipsis: ' [more]'")
	if !strings.HasSuffix(got, " [more]") || strings.Contains(got, "…") {
		t.Errorf("string marker: %q", got)
	}

	if got := ellipsisText(t, "line-clamp:2; block-ellipsis: no-ellipsis"); strings.Contains(got, "…") {
		t.Errorf("no-ellipsis still painted a marker: %q", got)
	}

	if got := ellipsisText(t, "block-ellipsis: ' [more]'"); strings.Contains(got, "[more]") {
		t.Errorf("an unclamped block got a marker: %q", got)
	}
}
