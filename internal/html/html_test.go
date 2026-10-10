package html

import (
	"fmt"
	"strings"
	"testing"
)

// fffdText is the shared U+FFFD replacement sample for the NUL and
// invalid-UTF-8 tests.
const fffdText = "a\uFFFDb"

func mustParse(t *testing.T, src string) *Node {
	t.Helper()

	root, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q) returned error: %v", src, err)
	}

	return root
}

// treeString renders the tree for failure messages.
func treeString(node *Node) string {
	var buf strings.Builder

	var rec func(*Node, int)
	rec = func(node *Node, depth int) {
		buf.WriteString(strings.Repeat("  ", depth))

		switch node.Type {
		case ElementNode:
			fmt.Fprintf(&buf, "<%s>", node.Name)
		case TextNode:
			fmt.Fprintf(&buf, "#text %q", node.Text)
		case CommentNode:
			fmt.Fprintf(&buf, "<!--%s-->", node.Text)
		case DoctypeNode:
			fmt.Fprintf(&buf, "<!%s>", node.Text)
		case NodeUnknown:
			fmt.Fprintf(&buf, "#unknown %q", node.Text)
		}

		buf.WriteByte('\n')

		for _, c := range node.Children {
			rec(c, depth+1)
		}
	}
	rec(node, 0)

	return buf.String()
}

func assertChildren(t *testing.T, node *Node, names ...string) {
	t.Helper()

	var got []*Node

	for _, c := range node.Children {
		if c.Type == ElementNode {
			got = append(got, c)
		}
	}

	if len(got) != len(names) {
		t.Errorf("children of <%s>: got %d elements, want %d\n%s", node.Name, len(got), len(names), treeString(node))

		return
	}

	for i, want := range names {
		if got[i].Name != want {
			t.Errorf("child %d of <%s>: got <%s>, want <%s>\n%s", i, node.Name, got[i].Name, want, treeString(node))
		}
	}
}

// --- tokenizer-level tests ---

func TestTokenizeAttributes(t *testing.T) {
	t.Parallel()

	toks, err := tokenize(`<p id="a1" class='b2' data-x=unquoted hidden checked="yes">`)
	if err != nil {
		t.Fatal(err)
	}

	if len(toks) != 1 {
		t.Fatalf("got %d tokens, want 1: %+v", len(toks), toks)
	}

	got := toks[0]
	if got.kind != tokStart || got.data != "p" {
		t.Fatalf("token = %+v, want start <p>", got)
	}

	want := []string{"id", "a1", "class", "b2", "data-x", "unquoted", "hidden", "", "checked", "yes"}
	if len(got.attrs) != len(want) {
		t.Fatalf("attrs = %v, want %v", got.attrs, want)
	}

	for i := range want {
		if got.attrs[i] != want[i] {
			t.Fatalf("attr %d = %q, want %q", i, got.attrs[i], want[i])
		}
	}
}

func TestTokenizeWhitespaceAroundEquals(t *testing.T) {
	t.Parallel()

	toks, err := tokenize(`<div a = "x" b= y>`)
	if err != nil {
		t.Fatal(err)
	}

	if len(toks) != 1 {
		t.Fatalf("got %d tokens, want 1: %+v", len(toks), toks)
	}

	attrs := toks[0].attrs
	want := []string{"a", "x", "b", "y"}

	if len(attrs) != len(want) {
		t.Fatalf("attrs = %v, want %v", attrs, want)
	}

	for i := range want {
		if attrs[i] != want[i] {
			t.Fatalf("attr %d = %q, want %q", i, attrs[i], want[i])
		}
	}
}

func TestTokenizeGreaterThanInQuotedValue(t *testing.T) {
	t.Parallel()

	toks, err := tokenize(`<p title="a > b" data-x='1>0'>x</p>`)
	if err != nil {
		t.Fatal(err)
	}

	if len(toks) != 3 {
		t.Fatalf("got %d tokens, want 3: %+v", len(toks), toks)
	}

	attrs := toks[0].attrs
	want := []string{"title", "a > b", "data-x", "1>0"}

	if len(attrs) != len(want) {
		t.Fatalf("attrs = %v, want %v", attrs, want)
	}

	for i := range want {
		if attrs[i] != want[i] {
			t.Fatalf("attr %d = %q, want %q", i, attrs[i], want[i])
		}
	}
}

func TestTokenizeComments(t *testing.T) {
	t.Parallel()

	toks, err := tokenize(`a<!-- hello -->b<!---->c`)
	if err != nil {
		t.Fatal(err)
	}

	want := []struct {
		kind tokenKind
		data string
	}{
		{tokText, "a"},
		{tokComment, " hello "},
		{tokText, "b"},
		{tokComment, ""},
		{tokText, "c"},
	}
	if len(toks) != len(want) {
		t.Fatalf("got %d tokens, want %d: %+v", len(toks), len(want), toks)
	}

	for i, wantTok := range want {
		if toks[i].kind != wantTok.kind || toks[i].data != wantTok.data {
			t.Errorf("token %d = %+v, want %+v", i, toks[i], wantTok)
		}
	}
}

// assertPlainDoctypeToken checks a doctype token that must carry only a name:
// the expected raw text and structured name match, and no identifier or
// force-quirks flag is set.
func assertPlainDoctypeToken(t *testing.T, src string, tok token, wantName, wantText string) {
	t.Helper()

	if tok.kind != tokDoctype || tok.data != wantText || tok.doctype.Name != wantName ||
		tok.doctype.ForceQuirks || tok.doctype.HasPublicID || tok.doctype.HasSystemID {
		t.Errorf("tokenize(%q): token 0 = %+v", src, tok)
	}
}

func TestTokenizeDoctype(t *testing.T) {
	t.Parallel()

	cases := []struct {
		src       string
		wantKinds []tokenKind
		wantName  string
		wantText  string
	}{
		{
			`<!DOCTYPE html><p>x</p>`,
			[]tokenKind{tokDoctype, tokStart, tokText, tokEnd},
			"html",
			"DOCTYPE html",
		},
		{
			// The structured name is lowercased; token data keeps the raw
			// declaration body (the case written in the source).
			`<!DoCtYpE html>ok`,
			[]tokenKind{tokDoctype, tokText},
			"html",
			"DoCtYpE html",
		},
	}
	for _, testCase := range cases {
		toks, err := tokenize(testCase.src)
		if err != nil {
			t.Fatal(err)
		}

		if len(toks) != len(testCase.wantKinds) {
			t.Fatalf("tokenize(%q): got %d tokens, want %d: %+v",
				testCase.src, len(toks), len(testCase.wantKinds), toks)
		}

		assertPlainDoctypeToken(t, testCase.src, toks[0], testCase.wantName, testCase.wantText)

		for i, wantKind := range testCase.wantKinds {
			if toks[i].kind != wantKind {
				t.Errorf("tokenize(%q): token %d kind = %v, want %v", testCase.src, i, toks[i].kind, wantKind)
			}
		}
	}
}

// tokenDoctypeCase is one structured doctype expectation for the tokenizer.
type tokenDoctypeCase struct {
	src         string
	name        string
	publicID    string
	hasPublic   bool
	systemID    string
	hasSystem   bool
	forceQuirks bool
}

// sameTokenDoctype compares the engine doctype against one expectation.
func sameTokenDoctype(got Doctype, want tokenDoctypeCase) bool {
	return got.Name == want.name &&
		got.PublicID == want.publicID && got.HasPublicID == want.hasPublic &&
		got.SystemID == want.systemID && got.HasSystemID == want.hasSystem &&
		got.ForceQuirks == want.forceQuirks
}

func TestTokenizeDoctypeIdentifiers(t *testing.T) {
	t.Parallel()

	assertTokenizeDoctypeIdentifiers(t, []tokenDoctypeCase{
		{
			src: `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" ` +
				`"http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">`,
			name:      "html",
			publicID:  "-//W3C//DTD XHTML 1.0 Strict//EN",
			hasPublic: true,
			systemID:  "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd",
			hasSystem: true,
		},
		{
			src:       `<!DOCTYPE html SYSTEM "about:legacy-compat">`,
			name:      "html",
			systemID:  "about:legacy-compat",
			hasSystem: true,
		},
		{
			src:       `<!DOCTYPE html PUBLIC 'one' 'two'>`,
			name:      "html",
			publicID:  "one",
			hasPublic: true,
			systemID:  "two",
			hasSystem: true,
		},
		{
			src:       `<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.01//EN">`,
			name:      "html",
			publicID:  "-//W3C//DTD HTML 4.01//EN",
			hasPublic: true,
		},
	})
}

func TestTokenizeDoctypeIdentifiersMalformed(t *testing.T) {
	t.Parallel()

	assertTokenizeDoctypeIdentifiers(t, []tokenDoctypeCase{
		{
			src:         `<!DOCTYPE html PUBLIC>`,
			name:        "html",
			forceQuirks: true,
		},
		{
			src:         `<!DOCTYPE>`,
			forceQuirks: true,
		},
		{
			src:         `<!DOCTYPE html x>`,
			name:        "html",
			forceQuirks: true,
		},
		{
			src:         `<!DOCTYPE html SYSTEM "x" y>`,
			name:        "html",
			systemID:    "x",
			hasSystem:   true,
			forceQuirks: false, // unexpected char after the system id: bogus, no force-quirks
		},
		{
			src:         `<!DOCTYPE html PUBLIC "a`,
			name:        "html",
			publicID:    "a",
			hasPublic:   true,
			forceQuirks: true, // EOF inside the public identifier
		},
	})
}

// assertTokenizeDoctypeIdentifiers tokenizes each case and compares its single
// doctype token against the structured expectation.
func assertTokenizeDoctypeIdentifiers(t *testing.T, cases []tokenDoctypeCase) {
	t.Helper()

	for _, testCase := range cases {
		toks, err := tokenize(testCase.src)
		if err != nil {
			t.Fatalf("tokenize(%q): %v", testCase.src, err)
		}

		if len(toks) != 1 || toks[0].kind != tokDoctype {
			t.Fatalf("tokenize(%q) = %+v, want one doctype token", testCase.src, toks)
		}

		got := toks[0].doctype
		if !sameTokenDoctype(got, testCase) {
			t.Errorf("tokenize(%q) doctype = %+v, want name=%q public=%q/%t system=%q/%t forceQuirks=%t",
				testCase.src, got, testCase.name, testCase.publicID, testCase.hasPublic,
				testCase.systemID, testCase.hasSystem, testCase.forceQuirks)
		}
	}
}

func TestDocumentModeClassification(t *testing.T) {
	t.Parallel()

	cases := []struct {
		src  string
		want DocumentMode
	}{
		{`<!DOCTYPE html>`, NoQuirks},
		{`<!DOCTYPE HTML>`, NoQuirks},
		{`<!DOCTYPE html SYSTEM "about:legacy-compat">`, NoQuirks},
		{`<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd">`, NoQuirks},
		{
			`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" ` +
				`"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">`,
			LimitedQuirks,
		},
		{
			`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Frameset//EN" ` +
				`"http://www.w3.org/TR/xhtml1/DTD/xhtml1-frameset.dtd">`,
			LimitedQuirks,
		},
		{
			`<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" ` +
				`"http://www.w3.org/TR/html4/loose.dtd">`,
			LimitedQuirks,
		},
		{
			`<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.01 Frameset//EN" ` +
				`"http://www.w3.org/TR/html4/frameset.dtd">`,
			LimitedQuirks,
		},
		// The same public id without a system id is quirks.
		{`<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN">`, Quirks},
		// A system id that is present but empty is not missing: no-quirks.
		{`<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" "">`, Quirks},
		{`<!DOCTYPE html PUBLIC "HTML">`, Quirks},
		{`<!DOCTYPE html PUBLIC "-//W3O//DTD W3 HTML Strict 3.0//EN//">`, Quirks},
		{`<!DOCTYPE html PUBLIC "-//IETF//DTD HTML 2.0//EN">`, Quirks},
		{`<!DOCTYPE html PUBLIC "+//Silmaril//dtd html Pro v0r11 19970101//">`, Quirks},
		{`<!DOCTYPE html SYSTEM "http://www.ibm.com/data/dtd/v11/ibmxhtml1-transitional.dtd">`, Quirks},
		{`<!DOCTYPE>`, Quirks},
		{`<!DOCTYPE html PUBLIC>`, Quirks},
		{`<!DOCTYPE other>`, Quirks},
		// No doctype, or content before it, is quirks mode.
		{``, Quirks},
		{`<p>x`, Quirks},
		{`<!DOCTYPE html><p>x`, NoQuirks},
		// Comments and whitespace do not leave the initial mode.
		{`<!-- c --><!DOCTYPE html>`, NoQuirks},
		{`  <!DOCTYPE html>`, NoQuirks},
	}
	for _, testCase := range cases {
		root := mustParse(t, testCase.src)
		if got := root.Mode; got != testCase.want {
			t.Errorf("Parse(%q) mode = %s, want %s", testCase.src, got, testCase.want)
		}
	}
}

func TestTokenizeDeclarationsAndPI(t *testing.T) {
	t.Parallel()

	toks, err := tokenize(`<?xml version="1.0"?><!bogus stuff><p>x</p>`)
	if err != nil {
		t.Fatal(err)
	}

	// "<?...>" and "<!bogus...>" are bogus comments per the tokenizer spec.
	want := []struct {
		kind tokenKind
		data string
	}{
		{tokComment, `?xml version="1.0"?`},
		{tokComment, "bogus stuff"},
		{tokStart, "p"},
		{tokText, "x"},
		{tokEnd, "p"},
	}
	if len(toks) != len(want) {
		t.Fatalf("got %d tokens, want %d: %+v", len(toks), len(want), toks)
	}

	for i, wantTok := range want {
		if toks[i].kind != wantTok.kind || toks[i].data != wantTok.data {
			t.Errorf("token %d = %+v, want kind %v data %q", i, toks[i], wantTok.kind, wantTok.data)
		}
	}
}

func TestTokenizeRawText(t *testing.T) {
	t.Parallel()

	cases := []struct {
		src, content string
	}{
		{`<script>if (a < b) { x(); }</script>`, "if (a < b) { x(); }"},
		{`<style>p > b { color: red; }</style>`, "p > b { color: red; }"},
		{`<textarea><b>not bold</b></textarea>`, "<b>not bold</b>"},
		{`<title>My <Page></title>`, "My <Page>"},
		{`<SCRIPT>var a = 1;</script>`, "var a = 1;"},
		{`<script src="x.js"></script>`, ""},
		// A self-closing flag on a raw-text element is ignored: content is
		// still consumed as script data.
		{`<script src="x.js"/>ok`, "ok"},
		{`<script>a</SCRIPT>b`, "ab"},
		{`<script>var x = 1;`, "var x = 1;"},
		// RAWTEXT and script data keep character references literal.
		{`<script>var x = "&amp;";</script>`, `var x = "&amp;";`},
		{`<style>a{content:"&amp;"}</style>`, `a{content:"&amp;"}`},
		{`<xmp>a &amp; b</xmp>`, "a &amp; b"},
		// RCDATA decodes character references.
		{`<title>Tom &amp; Jerry</title>`, "Tom & Jerry"},
		{`<textarea>&lt;b&gt;not bold&lt;/b&gt;</textarea>`, "<b>not bold</b>"},
		// A trailing solidus still closes a raw-text element.
		{`<script>a</script/>b`, "ab"},
	}
	for _, testCase := range cases {
		toks, err := tokenize(testCase.src)
		if err != nil {
			t.Fatalf("tokenize(%q): %v", testCase.src, err)
		}

		var content string

		for _, tk := range toks {
			if tk.kind == tokText {
				content += tk.data
			}
		}

		if content != testCase.content {
			t.Errorf("tokenize(%q): raw text = %q, want %q (tokens %+v)", testCase.src, content, testCase.content, toks)
		}
	}
}

func TestTokenizeRawTextClosesOnlyRealEndTag(t *testing.T) {
	t.Parallel()

	cases := []struct{ src, want string }{
		{`<script>a</scriptx>b</script>`, "a</scriptx>b"},
		{`<script>a</script/>b`, "ab"},
		{`<script>a</script b>c`, "ac"},
		{`<script>a</script `, "a"},
		{`<script>a</script`, "a</script"},
		{`<script>a</scr`, "a</scr"},
	}
	for _, testCase := range cases {
		toks, err := tokenize(testCase.src)
		if err != nil {
			t.Fatalf("tokenize(%q): %v", testCase.src, err)
		}

		var text string

		for _, tk := range toks {
			if tk.kind == tokText {
				text += tk.data
			}
		}

		if text != testCase.want {
			t.Errorf("tokenize(%q): raw text = %q, want %q (tokens %+v)", testCase.src, text, testCase.want, toks)
		}
	}
}

func TestTokenizeNormalizesNewlines(t *testing.T) {
	t.Parallel()

	toks, err := tokenize("a\r\nb\rc\r\rd\n\re")
	if err != nil {
		t.Fatal(err)
	}

	if len(toks) != 1 || toks[0].kind != tokText || toks[0].data != "a\nb\nc\n\nd\n\ne" {
		t.Fatalf("tokens = %+v, want one text token with normalized newlines", toks)
	}

	toks, err = tokenize("<p title=\"x\r\ny\">")
	if err != nil {
		t.Fatal(err)
	}

	if len(toks) != 1 || len(toks[0].attrs) != 2 || toks[0].attrs[1] != "x\ny" {
		t.Fatalf("tokens = %+v, want attribute value with normalized newline", toks)
	}
}

//nolint:cyclop // sequential scenario assertions, not branch logic
func TestTokenizeNullHandling(t *testing.T) {
	t.Parallel()

	// Data-state NUL is kept in the token stream; the tree drops it.
	toks, err := tokenize("a\x00b")
	if err != nil {
		t.Fatal(err)
	}

	if len(toks) != 1 || toks[0].kind != tokText || toks[0].data != "a\x00b" {
		t.Fatalf("data tokens = %+v, want text with U+0000 preserved", toks)
	}

	if got := mustParse(t, "a\x00b").TextContent(); got != "ab" {
		t.Errorf("tree text = %q, want %q (U+0000 dropped)", got, "ab")
	}

	// Comment, tag name, attribute, and raw-text states substitute U+FFFD.
	cases := []struct {
		src       string
		wantIndex int
		wantKind  tokenKind
		wantData  string
	}{
		{"<!--a\x00b-->", 0, tokComment, fffdText},
		{"<p\x00 a\x00b=\"c\x00d\">", 0, tokStart, "p\ufffd"},
		{"<style>a\x00b</style>", 1, tokText, fffdText},
		{"<title>a\x00b</title>", 1, tokText, fffdText},
	}
	for _, testCase := range cases {
		toks, err := tokenize(testCase.src)
		if err != nil {
			t.Fatalf("tokenize(%q): %v", testCase.src, err)
		}

		if len(toks) <= testCase.wantIndex {
			t.Errorf("tokenize(%q): got %d tokens %+v, want token %d", testCase.src, len(toks), toks, testCase.wantIndex)

			continue
		}

		got := toks[testCase.wantIndex]
		if got.kind != testCase.wantKind || got.data != testCase.wantData {
			t.Errorf("tokenize(%q): token %d = %+v, want kind %v data %q",
				testCase.src, testCase.wantIndex, got, testCase.wantKind, testCase.wantData)
		}
	}

	toks, err = tokenize("<p a\x00b=\"c\x00d\">")
	if err != nil {
		t.Fatal(err)
	}

	if len(toks) != 1 || len(toks[0].attrs) != 2 || toks[0].attrs[0] != fffdText || toks[0].attrs[1] != "c\ufffdd" {
		t.Fatalf("attribute tokens = %+v, want NUL replaced with U+FFFD in name and value", toks)
	}
}

func TestParseReplacesInvalidUTF8(t *testing.T) {
	t.Parallel()

	if got := mustParse(t, "a\xffb").TextContent(); got != fffdText {
		t.Errorf("TextContent = %q, want %q", got, fffdText)
	}
}

func TestTokenizeAttributeCharacterReferences(t *testing.T) {
	t.Parallel()

	cases := []struct{ src, want string }{
		{`<h a="&noti;">`, "&noti;"},
		{`<h a='&noti'>`, "&noti"},
		{`<h a='&notx'>`, "&notx"},
		{`<h a='&not1'>`, "&not1"},
		{`<h a='&COPY'>`, "\u00a9"},
		{`<h a="&notin;">`, "\u2209"},
		{`<h a="&not=">`, "&not="},
		{`<h a="&lang=">`, "&lang="},
		{`<h a="&amp;">`, "&"},
		{`<h a="&amp">`, "&"},
		{`<h a="&ampx">`, "&ampx"},
		{`<h a="&semi;">`, ";"},
		{`<h a="&nGt;">`, "\u226B\u20D2"},
		{`<h a="&#x3f;">`, "?"},
		{`<h a="&#38">`, "&"},
		{`<h a="&#0;">`, "\uFFFD"},
		{`<s o=& t>`, "&"},
		{`<a a=a&>foo`, "a&"},
	}
	for _, testCase := range cases {
		toks, err := tokenize(testCase.src)
		if err != nil {
			t.Fatalf("tokenize(%q): %v", testCase.src, err)
		}

		if len(toks) == 0 || toks[0].kind != tokStart {
			t.Fatalf("tokenize(%q): first token = %+v, want start tag", testCase.src, toks)
		}

		if len(toks[0].attrs) < 2 || toks[0].attrs[1] != testCase.want {
			t.Errorf("tokenize(%q): first attribute = %v, want value %q", testCase.src, toks[0].attrs, testCase.want)
		}
	}
}

func TestUnescapeEntitiesNumericEdgeCases(t *testing.T) {
	t.Parallel()

	cases := []struct{ src, want string }{
		{"&#0", "\uFFFD"},
		{"&#x0", "\uFFFD"},
		{"&#x100000041;", "\uFFFD"},
		{"&#4294967361;", "\uFFFD"},
		{"&#xD800;", "\uFFFD"},
		{"&#x41;", "A"},
		{"&#97a", "aa"},
		{"&nGt;", "\u226B\u20D2"},
		{"&noti;", "\u00ACi;"},
	}
	for _, testCase := range cases {
		if got := mustParse(t, testCase.src).TextContent(); got != testCase.want {
			t.Errorf("Parse(%q) text = %q, want %q", testCase.src, got, testCase.want)
		}
	}
}

func TestParseTextContexts(t *testing.T) {
	t.Parallel()

	cases := []struct{ src, elem, want string }{
		{`<style>a{content:"&amp;"}</style>`, "style", `a{content:"&amp;"}`},
		{`<script>var x = "&amp;";</script>`, "script", `var x = "&amp;";`},
		{`<title>Tom &amp; Jerry</title>`, "title", "Tom & Jerry"},
		{`<textarea>&lt;b&gt;not bold&lt;/b&gt;</textarea>`, "textarea", "<b>not bold</b>"},
		{`<p>Tom &amp; Jerry</p>`, "p", "Tom & Jerry"},
	}
	for _, testCase := range cases {
		root := mustParse(t, testCase.src)

		elem := firstElement(root, testCase.elem)
		if elem == nil {
			t.Fatalf("Parse(%q): no <%s>:\n%s", testCase.src, testCase.elem, treeString(root))
		}

		if got := elem.TextContent(); got != testCase.want {
			t.Errorf("Parse(%q): <%s> text = %q, want %q", testCase.src, testCase.elem, got, testCase.want)
		}
	}
}

func TestTokenizeBareLessThanIsText(t *testing.T) {
	t.Parallel()

	cases := []struct {
		src, want string
	}{
		{"1 < 2", "1 < 2"},
		{"a < b > c", "a < b > c"},
		{"<3>", "<3>"},
		{"<>empty<>", "<>empty<>"},
		{"< -x-", "< -x-"},
	}
	for _, testCase := range cases {
		toks, err := tokenize(testCase.src)
		if err != nil {
			t.Fatalf("tokenize(%q): %v", testCase.src, err)
		}

		var text string

		for _, tk := range toks {
			if tk.kind != tokText {
				t.Fatalf("tokenize(%q): unexpected non-text token %+v", testCase.src, tk)
			}

			text += tk.data
		}

		if text != testCase.want {
			t.Errorf("tokenize(%q) = %q, want %q", testCase.src, text, testCase.want)
		}
	}
}

func TestTokenizeRecoversUnterminated(t *testing.T) {
	t.Parallel()

	// The spec's recovery rules: unfinished comments become comment tokens,
	// EOF inside a tag drops the tag, stray "</" stays text, and unfinished
	// declarations and processing instructions become bogus comments.
	cases := []struct {
		src             string
		wantKind        tokenKind
		wantData        string
		wantForceQuirks bool
		wantCount       int
	}{
		{"<!-- unterminated", tokComment, " unterminated", false, 1},
		{"</div", 0, "", false, 0},
		{`<div a="x`, 0, "", false, 0},
		{`<div a='x`, 0, "", false, 0},
		{`<div a="x>`, 0, "", false, 0},
		{"<!DOCTYPE", tokDoctype, "DOCTYPE", true, 1},
		{"<!bogus", tokComment, "bogus", false, 1},
		{"<?pi", tokComment, "?pi", false, 1},
	}
	for _, testCase := range cases {
		if _, err := Parse(testCase.src); err != nil {
			t.Errorf("Parse(%q): unexpected error %v", testCase.src, err)
		}

		toks, err := tokenize(testCase.src)
		if err != nil {
			t.Fatalf("tokenize(%q): %v", testCase.src, err)
		}

		if len(toks) != testCase.wantCount {
			t.Errorf("tokenize(%q): got %d tokens %+v, want %d", testCase.src, len(toks), toks, testCase.wantCount)

			continue
		}

		if testCase.wantCount == 0 {
			continue
		}

		if toks[0].kind != testCase.wantKind || toks[0].data != testCase.wantData {
			t.Errorf("tokenize(%q): token 0 = %+v, want kind %v data %q",
				testCase.src, toks[0], testCase.wantKind, testCase.wantData)
		}

		if testCase.wantKind == tokDoctype && toks[0].doctype.ForceQuirks != testCase.wantForceQuirks {
			t.Errorf("tokenize(%q): force-quirks = %t, want %t",
				testCase.src, toks[0].doctype.ForceQuirks, testCase.wantForceQuirks)
		}
	}
}

func TestParseMatchesCollectedTokenBuilder(t *testing.T) {
	t.Parallel()

	const source = `<!DOCTYPE html><html><body><!-- note --><p data-x="a &amp; b">x < y</p>` +
		`<script>if (a < b) {}</script></body></html>`

	streamed := mustParse(t, source)

	tokens, err := tokenize(source)
	if err != nil {
		t.Fatal(err)
	}

	builder := newTreeBuilder()

	for _, token := range tokens {
		builder.appendToken(token)
	}

	builder.finish()

	collected := builder.root

	if got, want := treeString(streamed), treeString(collected); got != want {
		t.Fatalf("streamed Parse tree differs from collected-token builder:\nstreamed:\n%scollected:\n%s", got, want)
	}

	if got := streamed.FirstChild("html").FirstChild("body").FirstChild("p").Attribute("data-x"); got != "a & b" {
		t.Fatalf("streamed attribute = %q, want decoded value", got)
	}
}

// --- foreign content (SVG / MathML) ---

// firstElement returns the first element named name anywhere in root, so
// tests survive the implicit html/head/body wrappers.
func firstElement(root *Node, name string) *Node {
	return root.FindFirst(func(n *Node) bool { return n.Type == ElementNode && n.Name == name })
}

func TestParseForeignNamespaces(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<svg viewBox="0 0 10 10"><linearGradient id="g"/></svg>`)
	svg := firstElement(root, "svg")

	if svg == nil || svg.Namespace != NamespaceSVG {
		t.Fatalf("svg element = %+v, want SVG namespace\n%s", svg, treeString(root))
	}

	if got := svg.Attribute("viewBox"); got != "0 0 10 10" {
		t.Errorf("viewBox = %q, want adjusted attribute preserved", got)
	}

	gradient := svg.FirstChild("linearGradient")
	if gradient == nil || gradient.Namespace != NamespaceSVG {
		t.Fatalf("linearGradient = %+v, want adjusted SVG name\n%s", gradient, treeString(root))
	}

	if len(gradient.AttrList) != 1 || gradient.AttrList[0].Name != "id" || gradient.AttrList[0].Namespace != "" {
		t.Errorf("gradient AttrList = %+v, want one plain id attribute", gradient.AttrList)
	}
}

func TestParseForeignIntegrationPointSVG(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<svg><foreignObject><div>x</div></foreignObject></svg>`)
	svg := firstElement(root, "svg")
	foreign := svg.FirstChild("foreignObject")

	if foreign == nil || foreign.Namespace != NamespaceSVG {
		t.Fatalf("foreignObject = %+v, want adjusted SVG name\n%s", foreign, treeString(root))
	}

	div := foreign.FirstChild("div")
	if div == nil || div.Namespace != NamespaceHTML {
		t.Fatalf("div inside foreignObject = %+v, want HTML namespace\n%s", div, treeString(root))
	}
}

func TestParseForeignIntegrationPointMathML(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<math><mi>x</mi><annotation-xml encoding="text/html"><p>y</p></annotation-xml></math>`)
	math := firstElement(root, "math")

	if math == nil || math.Namespace != NamespaceMathML {
		t.Fatalf("math = %+v, want MathML namespace\n%s", math, treeString(root))
	}

	mi := math.FirstChild("mi")
	if mi == nil || mi.Namespace != NamespaceMathML {
		t.Fatalf("mi = %+v, want MathML namespace\n%s", mi, treeString(root))
	}

	annotation := math.FirstChild("annotation-xml")
	if annotation == nil {
		t.Fatalf("no annotation-xml:\n%s", treeString(root))
	}

	p := annotation.FirstChild("p")
	if p == nil || p.Namespace != NamespaceHTML {
		t.Fatalf("p inside annotation-xml = %+v, want HTML namespace\n%s", p, treeString(root))
	}
}

func TestParseForeignAttributesHTML(t *testing.T) {
	t.Parallel()

	// HTML attributes keep qualified names literal and un-namespaced.
	root := mustParse(t, `<body xlink:href="a"></body>`)
	body := firstElement(root, "body")

	if got := body.Attribute("xlink:href"); got != "a" {
		t.Errorf("HTML xlink:href = %q, want literal qualified name kept", got)
	}

	if len(body.AttrList) != 1 || body.AttrList[0].Namespace != "" || body.AttrList[0].Name != "xlink:href" {
		t.Errorf("HTML AttrList = %+v, want one un-namespaced qualified attribute", body.AttrList)
	}
}

func TestParseForeignAttributesSVG(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<svg xlink:href="b" xml:lang="en"></svg>`)
	svg := firstElement(root, "svg")

	if len(svg.AttrList) != 2 {
		t.Fatalf("SVG AttrList = %+v, want two namespaced attributes", svg.AttrList)
	}

	byName := map[string]Attr{}
	for _, attr := range svg.AttrList {
		byName[attr.Namespace+" "+attr.Name] = attr
	}

	if attr, found := byName["xlink href"]; !found || attr.Value != "b" {
		t.Errorf("SVG xlink:href = %+v, want xlink namespace with local name href", svg.AttrList)
	}

	if attr, found := byName["xml lang"]; !found || attr.Value != "en" {
		t.Errorf("SVG xml:lang = %+v, want xml namespace with local name lang", svg.AttrList)
	}
}

func TestParseForeignSelfClosingAndBreakout(t *testing.T) {
	t.Parallel()

	// A self-closing foreign element closes immediately; following text stays
	// in the foreign parent.
	root := mustParse(t, `<svg><g/>after</svg>`)
	svg := firstElement(root, "svg")

	if got := svg.TextContent(); got != "after" {
		t.Errorf("svg text = %q, want %q", got, "after")
	}

	if len(svg.FirstChild("g").Children) != 0 {
		t.Errorf("self-closing g has children:\n%s", treeString(svg))
	}

	// A breakout start tag pops the foreign elements and is reprocessed as
	// HTML, so the div becomes a sibling of svg.
	root = mustParse(t, `<svg><g><div>x</div></svg>`)
	svg = firstElement(root, "svg")

	if svg == nil || len(svg.Children) != 1 || svg.FirstChild("g") == nil {
		t.Fatalf("svg tree:\n%s", treeString(root))
	}

	div := firstElement(root, "div")
	if div == nil || div.Namespace != NamespaceHTML || div.TextContent() != "x" {
		t.Fatalf("breakout div = %+v\n%s", div, treeString(root))
	}

	// Foreign text replaces U+0000 with U+FFFD.
	if got := firstElement(mustParse(t, "<svg>a\x00b</svg>"), "svg").TextContent(); got != fffdText {
		t.Errorf("foreign text = %q, want U+FFFD replacement", got)
	}
}

// --- tree-level tests ---

func TestUnescapeEntitiesInText(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<html><body><h2>Docs &amp; forms</h2><p>a &lt; b &#38; c</p></body></html>`)
	body := root.FirstChild("body")

	if body == nil {
		body = root.FirstChild("html").FirstChild("body")
	}

	h2 := body.FirstChild("h2")
	if h2 == nil || h2.TextContent() != "Docs & forms" {
		t.Fatalf("h2 text = %q, want Docs & forms", h2.TextContent())
	}

	p := body.FirstChild("p")
	if p == nil || p.TextContent() != "a < b & c" {
		t.Fatalf("p text = %q", p.TextContent())
	}
}

func TestParseNesting(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<html><head><title>t</title></head><body><div><p>hi</p></div></body></html>`)
	html := root.FirstChild("html")

	if html == nil {
		t.Fatalf("no <html>:\n%s", treeString(root))
	}

	assertChildren(t, html, "head", "body")
	assertChildren(t, html.FirstChild("head"), "title")
	assertChildren(t, html.FirstChild("body"), "div")
	assertChildren(t, html.FirstChild("body").FirstChild("div"), "p")

	if got := html.FirstChild("body").TextContent(); got != "hi" {
		t.Errorf("TextContent = %q, want %q", got, "hi")
	}
}

func TestParseParentPointers(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<html><body><div>x</div><p>y</p></body></html>`)

	var walk func(*Node)
	walk = func(n *Node) {
		for _, c := range n.Children {
			if c.Parent != n {
				t.Errorf("Parent of %s = %v, want %s", c.Name, c.Parent, n.Name)
			}

			walk(c)
		}
	}
	walk(root)
}

func TestParseVoidElements(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<p>a<br>x<img src="y.png" alt="y"><input type="text" disabled><hr></p>`)
	body := firstElement(root, "body")

	// <hr> closes the open p, and the trailing </p> creates an empty p.
	assertChildren(t, body, "p", "hr", "p")

	para := body.FirstChild("p")

	// text a, br, text x, img, input - br/img/input must not consume the
	// following content.
	if len(para.Children) != 5 {
		t.Fatalf("<p> has %d children, want 5:\n%s", len(para.Children), treeString(para))
	}

	assertChildren(t, para, "br", "img", "input")

	if got := para.TextContent(); got != "ax" {
		t.Errorf("TextContent = %q, want %q", got, "ax")
	}

	img := para.FirstChild("img")
	if img.Attribute("src") != "y.png" || img.Attribute("alt") != "y" {
		t.Errorf("img attrs = %v", img.Attrs)
	}

	if len(img.Children) != 0 {
		t.Errorf("void <img> has children:\n%s", treeString(img))
	}

	if got := body.Children[2].TextContent(); got != "" {
		t.Errorf("trailing p text = %q, want empty", got)
	}
}

func TestParseAutoCloseTable(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<table><tr><td>a</td><td>b</td></tr><tr><td>c</td></tr></table>`)
	table := firstElement(root, "table")
	tbody := table.FirstChild("tbody")
	assertChildren(t, table, "tbody")
	assertChildren(t, tbody, "tr", "tr")
	assertChildren(t, tbody.Children[0], "td", "td")
	assertChildren(t, tbody.Children[1], "td")

	// <tr> closes an open <tr>/<td>/<th>
	root = mustParse(t, `<table><tr><td>a<tr><td>b</table>`)
	table = firstElement(root, "table")
	tbody = table.FirstChild("tbody")
	assertChildren(t, table, "tbody")
	assertChildren(t, tbody, "tr", "tr")
	assertChildren(t, tbody.Children[0], "td")
	assertChildren(t, tbody.Children[1], "td")

	if got := table.TextContent(); got != "ab" {
		t.Errorf("TextContent = %q, want %q", got, "ab")
	}

	// <td> closes an open <td>/<th>/<tr>
	root = mustParse(t, `<table><tr><td>a<td>b</table>`)
	table = firstElement(root, "table")
	tbody = table.FirstChild("tbody")
	assertChildren(t, table, "tbody")
	assertChildren(t, tbody, "tr")
	assertChildren(t, tbody.FirstChild("tr"), "td", "td")
}

func TestParseAutoCloseP(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<div><p>a<p>b</div>`)
	div := firstElement(root, "div")
	assertChildren(t, div, "p", "p")

	if got := div.TextContent(); got != "ab" {
		t.Errorf("TextContent = %q, want %q", got, "ab")
	}
}

func TestParseAutoCloseList(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<ul><li>a<li>b</ul>`)
	ul := firstElement(root, "ul")
	assertChildren(t, ul, "li", "li")

	root = mustParse(t, `<select><option>a<option>b</select>`)
	sel := firstElement(root, "select")
	assertChildren(t, sel, "option", "option")

	root = mustParse(t, `<dl><dt>t<dd>d<dt>t2</dl>`)
	dl := firstElement(root, "dl")
	assertChildren(t, dl, "dt", "dd", "dt")
}

func TestParseAutoCloseTableSections(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<table><thead>h<tbody>b<tfoot>f</table>`)
	table := firstElement(root, "table")
	assertChildren(t, table, "thead", "tbody", "tfoot")

	root = mustParse(t, `<table><tbody>b<thead>h</table>`)
	table = firstElement(root, "table")
	assertChildren(t, table, "tbody", "thead")

	root = mustParse(t, `<table><tfoot>f<tbody>b</table>`)
	table = firstElement(root, "table")
	assertChildren(t, table, "tfoot", "tbody")
}

func TestParseTableFosterParenting(t *testing.T) {
	t.Parallel()

	// Text around cells is foster-parented before the table; the cell text
	// stays in the cell.
	root := mustParse(t, `<table>A<td>B</td>C</table>`)
	body := firstElement(root, "body")
	assertChildren(t, body, "table")

	if got := body.Children[0].Text; got != "AC" {
		t.Errorf("foster text = %q, want %q", got, "AC")
	}

	table := firstElement(root, "table")
	td := table.FirstChild("tbody").FirstChild("tr").FirstChild("td")

	if got := td.TextContent(); got != "B" {
		t.Errorf("cell text = %q, want %q", got, "B")
	}

	// An element that cannot sit in a table is foster-parented as a whole.
	root = mustParse(t, `<table><div>x</div><tr><td>y`)
	body = firstElement(root, "body")
	assertChildren(t, body, "div", "table")

	if got := body.FirstChild("div").TextContent(); got != "x" {
		t.Errorf("div text = %q, want %q", got, "x")
	}

	// A select start tag under an open table moves before the table.
	root = mustParse(t, `<table><select><option>3</select></table>`)
	body = firstElement(root, "body")
	assertChildren(t, body, "select", "table")
}

func TestParseTableEndTags(t *testing.T) {
	t.Parallel()

	// </table> closes an open cell, row, section, and table in one pass.
	root := mustParse(t, `<table><tbody><tr><td>a</table>`)
	table := firstElement(root, "table")
	assertChildren(t, table, "tbody")
	assertChildren(t, table.FirstChild("tbody"), "tr")
	assertChildren(t, table.FirstChild("tbody").FirstChild("tr"), "td")

	// A stray section end tag after the row is ignored.
	root = mustParse(t, `<table><tr><td>a</td></tr></tbody></table>`)
	table = firstElement(root, "table")
	assertChildren(t, table, "tbody")

	// A colgroup closes on the first row start tag.
	root = mustParse(t, `<table><colgroup><col><tr><td>x`)
	table = firstElement(root, "table")
	assertChildren(t, table, "colgroup", "tbody")

	// A caption after the rows stays inside the table.
	root = mustParse(t, `<table><tr><td>a</td></tr><caption>c`)
	table = firstElement(root, "table")
	assertChildren(t, table, "tbody", "caption")

	// An end tag for an ordinary element does not pop through the table:
	// the table is special, so </kbd> is ignored.
	root = mustParse(t, `<kbd><table></kbd><tr><td>x`)
	kbd := firstElement(root, "kbd")
	assertChildren(t, kbd, "table")
}

func TestParseTableScopeThroughForeign(t *testing.T) {
	t.Parallel()

	// Table scope sees through foreign integration points: the second <td>
	// closes the open cell instead of being ignored.
	root := mustParse(t, `<table><tr><td><svg><desc><td>`)
	table := firstElement(root, "table")
	tr := table.FirstChild("tbody").FirstChild("tr")
	assertChildren(t, tr, "td", "td")

	// A nested table start tag closes the outer table through the foreign
	// subtree; the trailing <s> is foster-parented before the new table.
	root = mustParse(t, `<div><table><svg><foreignObject><select><table><s>`)
	div := firstElement(root, "div")
	assertChildren(t, div, "svg", "table", "s", "table")
}

func TestParseMisnestedFormatting(t *testing.T) {
	t.Parallel()

	// The standard's example: the unclosed <i> is reconstructed for "4".
	root := mustParse(t, `<p>1<b>2<i>3</b>4</i>5</p>`)
	paragraph := firstElement(root, "p")
	assertChildren(t, paragraph, "b", "i")

	if got := paragraph.TextContent(); got != "12345" {
		t.Errorf("p text = %q, want %q", got, "12345")
	}

	b := paragraph.FirstChild("b")
	assertChildren(t, b, "i")

	if got := b.TextContent(); got != "23" {
		t.Errorf("b text = %q, want %q", got, "23")
	}

	if got := paragraph.FirstChild("i").TextContent(); got != "4" {
		t.Errorf("reconstructed i text = %q, want %q", got, "4")
	}

	// A second <a> closes the first through the adoption agency; the
	// following "3" reopens the <b>.
	root = mustParse(t, `<a>1<b>2</a>3</b>`)
	body := firstElement(root, "body")
	assertChildren(t, body, "a", "b")

	a := body.FirstChild("a")
	assertChildren(t, a, "b")

	if got := body.TextContent(); got != "123" {
		t.Errorf("body text = %q, want %q", got, "123")
	}

	// A block inside formatting moves out: the <p> becomes a sibling and a
	// clone of the <a> stays inside it.
	root = mustParse(t, `<a><p></a></p>`)
	body = firstElement(root, "body")
	assertChildren(t, body, "a", "p")
	assertChildren(t, body.FirstChild("p"), "a")

	// The unclosed <i> reopens for "two" only.
	root = mustParse(t, `<b><i>one</b>two</i>`)
	body = firstElement(root, "body")
	assertChildren(t, body, "b", "i")

	if got := body.FirstChild("b").TextContent(); got != "one" {
		t.Errorf("b text = %q, want %q", got, "one")
	}

	if got := body.Children[1].TextContent(); got != "two" {
		t.Errorf("reconstructed i text = %q, want %q", got, "two")
	}
}

func TestParseHtmlHeadBodyMerge(t *testing.T) {
	t.Parallel()

	// Non-whitespace text in head content closes the head and lands in body;
	// the second <head> is ignored.
	root := mustParse(t, `<html><head>a</head><head>b</head></html>`)
	html := root.FirstChild("html")

	if html == nil {
		t.Fatalf("no <html>:\n%s", treeString(root))
	}

	assertChildren(t, html, "head", "body")

	if got := html.FirstChild("head").TextContent(); got != "" {
		t.Errorf("head text = %q, want empty", got)
	}

	if got := html.FirstChild("body").TextContent(); got != "ab" {
		t.Errorf("body text = %q, want %q", got, "ab")
	}

	// A second <body> merges into the existing one; its text stays in body.
	root = mustParse(t, `<body>x</body><body>y</body>`)
	assertChildren(t, firstElement(root, "html"), "head", "body")

	if got := firstElement(root, "body").TextContent(); got != "xy" {
		t.Errorf("body text = %q, want %q", got, "xy")
	}

	// A second <html> merges attributes; text stays in the one body.
	root = mustParse(t, `<html>a</html><html>b</html>`)
	assertChildren(t, root, "html")

	if got := firstElement(root, "body").TextContent(); got != "ab" {
		t.Errorf("body text = %q, want %q", got, "ab")
	}

	// A nested duplicate <body> is ignored, not nested and not closed.
	root = mustParse(t, `<body><body>z</body>`)
	assertChildren(t, firstElement(root, "html"), "head", "body")

	if got := firstElement(root, "body").TextContent(); got != "z" {
		t.Errorf("body text = %q, want %q", got, "z")
	}

	// Attributes on repeated html/body start tags merge into the existing
	// elements, keeping the first value.
	root = mustParse(t, `<html lang="en"><body class="a">x<body class="b">y</html>`)
	html = root.FirstChild("html")

	if got := html.Attribute("lang"); got != "en" {
		t.Errorf("html lang = %q, want %q", got, "en")
	}

	if got := firstElement(root, "body").Attribute("class"); got != "a" {
		t.Errorf("body class = %q, want first value %q", got, "a")
	}

	if got := firstElement(root, "body").TextContent(); got != "xy" {
		t.Errorf("body text = %q, want %q", got, "xy")
	}
}

func TestParseHeadBodyTransition(t *testing.T) {
	t.Parallel()

	// <body> closes an open <head>
	root := mustParse(t, `<head><title>t</title></head><body>b</body>`)
	html := firstElement(root, "html")
	assertChildren(t, html, "head", "body")

	root = mustParse(t, `<html><head><title>t</title></head><body>b</body></html>`)
	html = root.FirstChild("html")
	assertChildren(t, html, "head", "body")
}

func TestParseImplicitDocumentStructure(t *testing.T) {
	t.Parallel()

	for _, src := range []string{"", "<p>x", "plain text", "<!-- c -->", "<title>t</title>"} {
		root := mustParse(t, src)

		html := root.FirstChild("html")
		if html == nil {
			t.Fatalf("Parse(%q): no html wrapper:\n%s", src, treeString(root))
		}

		if html.FirstChild("head") == nil || html.FirstChild("body") == nil {
			t.Fatalf("Parse(%q): missing head or body:\n%s", src, treeString(root))
		}
	}

	// Content that belongs to head is routed there before body starts.
	root := mustParse(t, `<title>t</title><meta charset="utf-8"><link rel="x"><p>p`)
	html := root.FirstChild("html")

	assertChildren(t, html, "head", "body")
	assertChildren(t, html.FirstChild("head"), "title", "meta", "link")
	assertChildren(t, html.FirstChild("body"), "p")

	// A doctype stays a document child ahead of the html element.
	root = mustParse(t, `<!DOCTYPE html><p>x`)

	if len(root.Children) != 2 || root.Children[0].Type != DoctypeNode || root.Children[1].Name != "html" {
		t.Fatalf("doctype placement:\n%s", treeString(root))
	}
}

func TestParseClosesParagraphsListsAndDefinitions(t *testing.T) {
	t.Parallel()

	// A block start tag closes an open p.
	root := mustParse(t, `<!DOCTYPE html><p>one<div>two</div>three`)
	body := firstElement(root, "body")

	assertChildren(t, body, "p", "div")

	if got := body.FirstChild("p").TextContent(); got != "one" {
		t.Errorf("p text = %q, want %q", got, "one")
	}

	if got := body.FirstChild("div").TextContent(); got != "two" {
		t.Errorf("div text = %q, want %q", got, "two")
	}

	if got := body.Children[2].Text; got != "three" {
		t.Errorf("trailing text = %q, want %q", got, "three")
	}

	// An end tag for a scoped block closes an open p through implied ends.
	root = mustParse(t, `<div><p>foo</div>bar`)

	body = firstElement(root, "body")
	assertChildren(t, body, "div")

	if got := body.FirstChild("div").TextContent(); got != "foo" {
		t.Errorf("div text = %q, want %q", got, "foo")
	}

	if got := body.TextContent(); got != "foobar" {
		t.Errorf("body text = %q, want %q", got, "foobar")
	}

	// A new li closes the previous li even through a div.
	root = mustParse(t, `<ul><li>a<div><li>b</ul>`)

	ul := firstElement(root, "ul")
	assertChildren(t, ul, "li", "li")
	assertChildren(t, ul.Children[0], "div")

	if got := ul.TextContent(); got != "ab" {
		t.Errorf("ul text = %q, want %q", got, "ab")
	}

	// A new dt/dd closes the previous definition item.
	root = mustParse(t, `<dl><dt>t<dd>d<dt>t2</dl>`)

	dl := firstElement(root, "dl")
	assertChildren(t, dl, "dt", "dd", "dt")
}

//nolint:cyclop // sequential scenario assertions, not branch logic
func TestParseTextMerging(t *testing.T) {
	t.Parallel()

	// adjacent text tokens merge into a single TextNode inside body
	root := mustParse(t, `1 < 2`)

	body := firstElement(root, "body")
	if len(body.Children) != 1 {
		t.Fatalf("body has %d children, want 1:\n%s", len(body.Children), treeString(root))
	}

	txt := body.Children[0]
	if txt.Type != TextNode || txt.Text != "1 < 2" {
		t.Errorf("child = %+v, want single TextNode \"1 < 2\"", txt)
	}

	// text around comments stays as separate nodes
	root = mustParse(t, `a<!-- c -->b`)

	body = firstElement(root, "body")
	if len(body.Children) != 3 {
		t.Fatalf("body has %d children, want 3:\n%s", len(body.Children), treeString(root))
	}

	if body.Children[0].Type != TextNode || body.Children[0].Text != "a" {
		t.Errorf("child 0 = %+v", body.Children[0])
	}

	if body.Children[2].Type != TextNode || body.Children[2].Text != "b" {
		t.Errorf("child 2 = %+v", body.Children[2])
	}

	// bare '<' sequences inside an element merge
	root = mustParse(t, `<p>x <3>y</p>`)

	p := firstElement(root, "p")
	if len(p.Children) != 1 || p.Children[0].Type != TextNode || p.Children[0].Text != "x <3>y" {
		t.Fatalf("p children = %+v, want one TextNode \"x <3>y\"\n%s", p.Children, treeString(p))
	}
}

func TestParseAttrDuplicates(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<div ID="a" class="b" data-x="1" hidden id="dup">x</div>`)

	div := firstElement(root, "div")
	if len(div.Attrs) != 4 {
		t.Errorf("Attrs = %v, want 4 entries", div.Attrs)
	}
	// duplicate keeps the FIRST value
	if div.Attribute("id") != "a" {
		t.Errorf("id = %q, want first value %q", div.Attribute("id"), "a")
	}
	// attribute names are lowercased
	if div.Attribute("ID") != "a" || div.Attribute("Class") != "b" {
		t.Errorf("case-sensitive lookup failed: %v", div.Attrs)
	}

	if div.Attribute("hidden") != "" {
		t.Errorf("boolean attr hidden = %q, want \"\"", div.Attribute("hidden"))
	}

	if div.Attribute("data-x") != "1" {
		t.Errorf("data-x = %q, want %q", div.Attribute("data-x"), "1")
	}
}

func TestParseSelfClosing(t *testing.T) {
	t.Parallel()

	// The flag is ignored on ordinary HTML elements: the span stays inside
	// the still-open div.
	root := mustParse(t, `<div/><span>x</span>`)
	div := firstElement(root, "div")

	span := div.FirstChild("span")
	if span == nil || span.TextContent() != "x" {
		t.Fatalf("self-closing <div/> did not stay open:\n%s", treeString(root))
	}

	// A self-closing raw-text element still starts raw text: the flag is
	// ignored, so the following text is script data.
	root = mustParse(t, `<script src="x.js"/>ok`)

	script := firstElement(root, "script")
	if got := script.TextContent(); got != "ok" {
		t.Errorf("script text = %q, want %q", got, "ok")
	}

	// Foreign elements honor the flag: <g/> closes immediately.
	root = mustParse(t, `<svg><g/>after</svg>`)

	if got := firstElement(root, "svg").TextContent(); got != "after" {
		t.Errorf("svg text = %q, want %q", got, "after")
	}
}

func TestParseDoctypeNode(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<!DOCTYPE html><html><body>x</body></html>`)
	if len(root.Children) != 2 {
		t.Fatalf("root has %d children, want 2:\n%s", len(root.Children), treeString(root))
	}

	doctype := root.Children[0]
	if doctype.Type != DoctypeNode || doctype.Text != "DOCTYPE html" {
		t.Errorf("child 0 = %+v", doctype)
	}

	if doctype.Doctype.Name != "html" || doctype.Doctype.HasPublicID || doctype.Doctype.HasSystemID {
		t.Errorf("doctype fields = %+v, want name html and no identifiers", doctype.Doctype)
	}

	if root.Mode != NoQuirks {
		t.Errorf("document mode = %s, want no-quirks", root.Mode)
	}
}

func TestParseCommentsInBody(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<body><!-- hello -->x</body>`)
	body := firstElement(root, "body")

	if len(body.Children) != 2 {
		t.Fatalf("body has %d children, want 2:\n%s", len(body.Children), treeString(body))
	}

	if body.Children[0].Type != CommentNode || body.Children[0].Text != " hello " {
		t.Errorf("child 0 = %+v", body.Children[0])
	}

	if body.Children[1].Type != TextNode || body.Children[1].Text != "x" {
		t.Errorf("child 1 = %+v", body.Children[1])
	}
}

func TestParseRawTextTree(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<script>if (a < b) { f(); }</script><p>ok</p>`)
	script := firstElement(root, "script")

	if len(script.Children) != 1 || script.Children[0].Type != TextNode {
		t.Fatalf("script children = %+v, want one TextNode\n%s", script.Children, treeString(root))
	}

	if script.Children[0].Text != "if (a < b) { f(); }" {
		t.Errorf("script text = %q", script.Children[0].Text)
	}

	// A leading script lands in head; the paragraph starts the body.
	assertChildren(t, firstElement(root, "head"), "script")
	assertChildren(t, firstElement(root, "body"), "p")

	if got := root.TextContent(); got != "if (a < b) { f(); }ok" {
		t.Errorf("TextContent = %q", got)
	}
}

// malformedCase is one malformed-input scenario: the source and the tree
// assertions to run on its parse.
type malformedCase struct {
	src   string
	check func(t *testing.T, root *Node)
}

// runMalformedCases parses each scenario and runs its assertions.
func runMalformedCases(t *testing.T, cases []malformedCase) {
	t.Helper()

	for _, testCase := range cases {
		root := mustParse(t, testCase.src)
		testCase.check(t, root)
	}
}

func TestParseMalformed(t *testing.T) {
	t.Parallel()

	runMalformedCases(t, []malformedCase{
		{
			src: `<p><b>bold`,
			check: func(t *testing.T, root *Node) {
				t.Helper()

				p := firstElement(root, "p")
				assertChildren(t, p, "b")
				if got := root.TextContent(); got != "bold" {
					t.Errorf("TextContent = %q", got)
				}
			},
		},
		{
			src: `<div><span><p>x</div>`,
			check: func(t *testing.T, root *Node) {
				t.Helper()

				div := firstElement(root, "div")
				assertChildren(t, div, "span")
				assertChildren(t, div.FirstChild("span"), "p")
			},
		},
		{
			src: `<ul><li>a<li>b`,
			check: func(t *testing.T, root *Node) {
				t.Helper()

				assertChildren(t, firstElement(root, "ul"), "li", "li")
			},
		},
	})
}

func TestParseMalformedStrayTextAndTags(t *testing.T) {
	t.Parallel()

	runMalformedCases(t, []malformedCase{
		{
			src: `</div>text`,
			check: func(t *testing.T, root *Node) {
				t.Helper()

				body := firstElement(root, "body")
				for _, c := range body.Children {
					if c.Type == ElementNode {
						t.Errorf("stray end tag produced element:\n%s", treeString(root))

						return
					}
				}
				if body.TextContent() != "text" {
					t.Errorf("stray end tag:\n%s", treeString(root))
				}
			},
		},
		{
			src: `<>empty<>`,
			check: func(t *testing.T, root *Node) {
				t.Helper()

				body := firstElement(root, "body")
				if got := body.TextContent(); got != "<>empty<>" {
					t.Errorf("TextContent = %q, want %q", got, "<>empty<>")
				}
				if len(body.Children) != 1 || body.Children[0].Type != TextNode {
					t.Errorf("children:\n%s", treeString(root))
				}
			},
		},
		{
			src: `<div a=b/ c="d" e>`,
			check: func(t *testing.T, root *Node) {
				t.Helper()

				div := firstElement(root, "div")
				if div == nil {
					t.Fatalf("no <div>:\n%s", treeString(root))
				}
				if div.Attribute("c") != "d" || div.Attribute("e") != "" {
					t.Errorf("attrs = %v", div.Attrs)
				}
			},
		},
	})
}

func TestParseMalformedUnclosedTable(t *testing.T) {
	t.Parallel()

	runMalformedCases(t, []malformedCase{
		{
			src: `<table><tr><td>cell`,
			check: func(t *testing.T, root *Node) {
				t.Helper()

				table := firstElement(root, "table")
				tbody := table.FirstChild("tbody")
				assertChildren(t, table, "tbody")
				assertChildren(t, tbody, "tr")
				assertChildren(t, tbody.FirstChild("tr"), "td")
				if got := root.TextContent(); got != "cell" {
					t.Errorf("TextContent = %q", got)
				}
			},
		},
	})
}

func TestParseMalformedEOFRecovery(t *testing.T) {
	t.Parallel()

	runMalformedCases(t, []malformedCase{
		{
			src: `<!--comment`,
			check: func(t *testing.T, root *Node) {
				t.Helper()

				// The unfinished comment stays on the document; EOF still
				// creates the html/head/body structure.
				if len(root.Children) != 2 || root.Children[0].Type != CommentNode || root.Children[0].Text != "comment" {
					t.Errorf("unfinished comment tree:\n%s", treeString(root))
				}

				if firstElement(root, "html") == nil {
					t.Errorf("no html wrapper:\n%s", treeString(root))
				}
			},
		},
		{
			src: `<!bogus`,
			check: func(t *testing.T, root *Node) {
				t.Helper()

				if len(root.Children) != 2 || root.Children[0].Type != CommentNode || root.Children[0].Text != "bogus" {
					t.Errorf("bogus declaration tree:\n%s", treeString(root))
				}
			},
		},
		{
			src: `</div`,
			check: func(t *testing.T, root *Node) {
				t.Helper()

				// EOF in an end tag drops the token; the empty document still
				// gets the html wrapper.
				if len(root.Children) != 1 || root.Children[0].Name != "html" {
					t.Errorf("EOF end tag produced children:\n%s", treeString(root))
				}
			},
		},
	})
}

func TestParseUsableTreeNoPanic(t *testing.T) {
	t.Parallel()

	inputs := []string{
		`<div><div><div><div>x`,
		`<b><i><u>deep</b>`,
		`<table><thead><tr><th>h<td>d`,
		`<p a=1><p a=2><p a=3>`,
		`<script><script></script>`,
		`<style>a{}</style>`,
		"",
		`plain text only`,
		`<html><head><meta charset="utf-8"><title>t</title></head><body><br><img></body></html>`,
	}
	for _, src := range inputs {
		if _, err := Parse(src); err != nil {
			t.Errorf("Parse(%q): %v", src, err)
		}
	}
}

func TestWalkPreOrder(t *testing.T) {
	t.Parallel()
	root := mustParse(t, `<html><head><title>t</title></head><body><h1>x</h1><p>y</p></body></html>`)

	var names []string

	root.Walk(func(n *Node) {
		if n.Type == ElementNode {
			names = append(names, n.Name)
		}
	})

	want := []string{"#document", "html", "head", "title", "body", "h1", "p"}
	if len(names) != len(want) {
		t.Fatalf("walk order = %v, want %v", names, want)
	}

	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("walk order = %v, want %v", names, want)
		}
	}
}

func TestTextContentOf(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><head><title>One</title><title>Two</title></head><body><p>body text</p></body></html>`)
	if got := root.TextContentOf("title"); got != "One" {
		t.Errorf("TextContentOf(title) = %q, want %q", got, "One")
	}

	if got := root.TextContentOf("section"); got != "" {
		t.Errorf("TextContentOf(section) = %q, want %q", got, "")
	}

	if got := root.TextContentOf("p"); got != "body text" {
		t.Errorf("TextContentOf(p) = %q, want %q", got, "body text")
	}
}

func TestParseDocument(t *testing.T) {
	t.Parallel()

	src := "<html><body>ok</body></html>"
	for _, body := range [][]byte{[]byte(src), append([]byte("\ufeff"), src...)} {
		root, err := ParseDocument(body)
		if err != nil {
			t.Fatalf("ParseDocument: %v", err)
		}

		if root.FirstChild("html") == nil {
			t.Errorf("ParseDocument(%q): no html element", body[:4])
		}
	}
}

func TestParseDocumentBOMStripsWholeBOM(t *testing.T) {
	t.Parallel()

	// Regression: the old s = s[1:] cut one byte of the 3-byte UTF-8 BOM,
	// leaving two stray bytes as a leading text node.
	root, err := ParseDocument([]byte("\ufeff<html><body>ok</body></html>"))
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}

	for _, c := range root.Children {
		if c.Type == TextNode {
			t.Errorf("ParseDocument left a leading text node %q", c.Text)
		}
	}

	if got := root.TextContent(); strings.Contains(got, "\xbb\xbf") {
		t.Errorf("ParseDocument left BOM residue bytes in text content %q", got)
	}
}

func TestParseDeepNesting(t *testing.T) {
	t.Parallel()

	// A 100k-deep input must parse without exhausting the stack, and the
	// tree must stay within maxElementDepth so recursive walks stay bounded.
	const depth = 100000

	src := strings.Repeat("<div>", depth) + "deep" + strings.Repeat("</div>", depth)

	root, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	maxSeen := 0

	var measure func(*Node, int)
	measure = func(n *Node, level int) {
		if level > maxSeen {
			maxSeen = level
		}

		for _, c := range n.Children {
			measure(c, level+1)
		}
	}
	measure(root, 0)

	if maxSeen > maxElementDepth+1 {
		t.Fatalf("tree depth %d exceeds cap %d", maxSeen, maxElementDepth)
	}

	visits := 0

	root.Walk(func(*Node) { visits++ })

	if visits > (maxElementDepth+1)*2 {
		t.Fatalf("Walk visited %d nodes, want bounded by depth cap", visits)
	}
}

func TestNodeTypeZeroIsUnknown(t *testing.T) {
	t.Parallel()

	var zeroNode Node

	if got := zeroNode.Type; got != NodeUnknown {
		t.Fatalf("zero Node Type = %v, want NodeUnknown", got)
	}

	root, err := Parse(`<div>text</div>`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if got := root.Type; got != ElementNode {
		t.Fatalf("parsed root Type = %v, want ElementNode", got)
	}
}
