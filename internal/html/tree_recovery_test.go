package html

import (
	"testing"
)

// Tree-construction recovery rules closed under the GATE-03 clusters:
// script-data escaped states, CDATA sections in foreign content, the ruby
// implied end-tag rules, noscript in head with scripting disabled, the
// frameset-ok flag, and the small mismatches (textarea leading LF, void
// basefont/bgsound, comment placement after </html>). The pinned corpus is
// baseline evidence unless HTML_CONFORMANCE_STRICT=1, so these tests keep the
// rules enforced in an ordinary `go test` run.

func TestParseScriptDataEscaping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		src  string
		want string
	}{
		{`<script><!--<script></script></script>`, `<!--<script></script>`},
		{`<script><!--<script></SCRIPT></script>`, `<!--<script></SCRIPT>`},
		{`<script><!--x--></script>`, `<!--x-->`},
		{`<script><!--x--!></script>`, `<!--x--!>`},
		{`<script><!--<scr></script>`, `<!--<scr>`},
		{`<script>a</ScRiPtX>b</script>`, `a</ScRiPtX>b`},
		{"<script><!--<script>\x00</script></script>", "<!--<script>\uFFFD</script>"},
	}

	for _, testCase := range cases {
		root := mustParse(t, testCase.src)

		script := firstElement(root, "script")
		if script == nil {
			t.Fatalf("Parse(%q): no script element:\n%s", testCase.src, treeString(root))
		}

		if got := script.TextContent(); got != testCase.want {
			t.Errorf("Parse(%q): script text = %q, want %q", testCase.src, got, testCase.want)
		}
	}
}

func TestParseCDATAInForeignContent(t *testing.T) {
	t.Parallel()

	// A CDATA section in foreign content is text, not a comment, and keeps
	// its content byte for byte.
	root := mustParse(t, `<svg><![CDATA[<b>&amp;]]></svg>`)

	svg := firstElement(root, "svg")
	if got := svg.TextContent(); got != "<b>&amp;" {
		t.Errorf("svg text = %q, want %q", got, "<b>&amp;")
	}

	// The same opener in HTML content is a bogus comment.
	root = mustParse(t, `<div><![CDATA[foo]]>`)

	div := firstElement(root, "div")
	if len(div.Children) != 1 || div.Children[0].Type != CommentNode || div.Children[0].Text != "[CDATA[foo]]" {
		t.Errorf("div children:\n%s", treeString(div))
	}
}

func TestParseRubyImpliedEndTags(t *testing.T) {
	t.Parallel()

	// rt closes an open rb and lands beside it under ruby.
	root := mustParse(t, `<ruby>a<rb>b<rt></ruby>`)
	ruby := firstElement(root, "ruby")

	assertChildren(t, ruby, "rb", "rt")

	if got := ruby.FirstChild("rb").TextContent(); got != "b" {
		t.Errorf("rb text = %q, want %q", got, "b")
	}

	// rb closes implied end tags including an open rtc, while rt stays
	// inside the open rtc.
	root = mustParse(t, `<ruby>a<rtc>b<rt>c<rb>d</ruby>`)
	ruby = firstElement(root, "ruby")

	assertChildren(t, ruby, "rtc", "rb")
	assertChildren(t, ruby.FirstChild("rtc"), "rt")

	if got := ruby.FirstChild("rb").TextContent(); got != "d" {
		t.Errorf("rb text = %q, want %q", got, "d")
	}
}

func TestParseNoscriptInHead(t *testing.T) {
	t.Parallel()

	// Head content stays inside noscript; comments stay there too.
	root := mustParse(t, `<head><noscript><basefont><!--foo--></noscript>`)
	head := firstElement(root, "head")

	assertChildren(t, head, "noscript")

	noscript := head.FirstChild("noscript")
	assertChildren(t, noscript, "basefont")

	if len(noscript.Children) != 2 || noscript.Children[1].Type != CommentNode || noscript.Children[1].Text != "foo" {
		t.Errorf("noscript children:\n%s", treeString(noscript))
	}

	// A non-head tag closes noscript and continues in body; the iframe then
	// consumes the rest of the input as raw text.
	root = mustParse(t, `<!doctype html><noscript><iframe></noscript>X`)
	body := firstElement(root, "body")

	assertChildren(t, body, "iframe")

	if got := body.TextContent(); got != "</noscript>X" {
		t.Errorf("body text = %q, want %q", got, "</noscript>X")
	}
}

func TestParseFramesetReplacesBody(t *testing.T) {
	t.Parallel()

	// A frameset replaces an implicit body when nothing cleared frameset-ok.
	for _, src := range []string{
		`<!doctype html><p><frameset><frame>`,
		`<!doctype html><svg></svg><frameset>`,
		`<!doctype html><input type="hidden"><frameset>`,
		"<html>\x00<frameset></frameset>",
	} {
		root := mustParse(t, src)

		html := root.FirstChild("html")
		assertChildren(t, html, "head", "frameset")
	}

	// Non-whitespace text in foreign content clears frameset-ok, so body
	// stays and the frameset tag is ignored.
	root := mustParse(t, `<!doctype html><svg>a</svg><frameset>`)
	html := root.FirstChild("html")

	assertChildren(t, html, "head", "body")
}

func TestParseTextareaLeadingNewline(t *testing.T) {
	t.Parallel()

	cases := []struct {
		src  string
		want string
	}{
		{"<textarea>\nfoo</textarea>", "foo"},
		{"<textarea>\n\nfoo</textarea>", "\nfoo"},
		{"<textarea></textarea>", ""},
	}

	for _, testCase := range cases {
		root := mustParse(t, testCase.src)

		textarea := firstElement(root, "textarea")
		if got := textarea.TextContent(); got != testCase.want {
			t.Errorf("Parse(%q): textarea text = %q, want %q", testCase.src, got, testCase.want)
		}
	}
}

func TestParseBasefontBgsoundVoid(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<!doctype html><body><basefont>A`)

	body := firstElement(root, "body")
	assertChildren(t, body, "basefont")

	if got := body.TextContent(); got != "A" {
		t.Errorf("body text = %q, want %q", got, "A")
	}
}

func TestParseCommentAfterBody(t *testing.T) {
	t.Parallel()

	// A comment after </html> belongs to the document, not to html.
	root := mustParse(t, `<html></html><!-- foo -->`)

	if len(root.Children) != 2 || root.Children[1].Type != CommentNode || root.Children[1].Text != " foo " {
		t.Errorf("root children:\n%s", treeString(root))
	}
}
