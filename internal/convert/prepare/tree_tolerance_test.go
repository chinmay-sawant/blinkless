package prepare_test

// These tests pin the downstream side of HTML-INTEGRATION-01: stylesheet
// collection and element routing must tolerate the trees the public parser
// produces (implicit html/head/body wrappers, a parser-inserted tbody,
// namespaced SVG and MathML subtrees).
//
// Template contents live in Node.Contents, outside Children, and the Walk and
// FindFirst helpers skip them (internal/html/html.go), so collectSheets must
// not pick up a <style> inside a template.
// TestCollectSheetsSkipsTemplateContents pins that.
//
// MathML decision: MathML subtrees render downstream as generic elements.
// Nothing marks them unsupported; layout dispatches on element name only, so
// collection just walks through them.

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/convert/prepare"
	"github.com/chinmay-sawant/blinkless/internal/html"
	"github.com/chinmay-sawant/blinkless/internal/load"
	"github.com/chinmay-sawant/blinkless/internal/settings"
)

// TestCollectSheetsImplicitDocumentTree proves routing tolerates the standard
// parser tree: a synthetic #document root, a doctype sibling, and implicit
// html/head/body wrappers. The doctype node is not an element, so the walk
// must skip it while still reaching the <style> in head and the <link> in
// body.
func TestCollectSheetsImplicitDocumentTree(t *testing.T) {
	t.Parallel()

	const source = `<!DOCTYPE html><title>t</title><style>.in-head { color: #f00 }</style>` +
		`<p>x</p><link rel="stylesheet" href="linked.css">`

	root, err := html.Parse(source)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	assertImplicitDocumentShape(t, root)

	sheets, logBuf, hits := collectImportSheets(t, importFixture{
		html:  source,
		files: map[string]string{"/linked.css": `.linked { color: #00f }`},
		media: "print",
	})

	if hits["/linked.css"] != 1 {
		t.Fatalf("linked stylesheet hits = %v, want one fetch", hits)
	}

	assertSheetClasses(t, sheets, []string{"in-head", "linked"})

	if logBuf.Len() != 0 {
		t.Fatalf("collection warnings = %q", logBuf.String())
	}
}

// assertImplicitDocumentShape pins the standard tree the collector must walk:
// a synthetic root whose first child is the doctype, then html with implicit
// head (holding the style) and body (holding the link).
func assertImplicitDocumentShape(t *testing.T, root *html.Node) {
	t.Helper()

	if len(root.Children) < 2 || root.Children[0].Type != html.DoctypeNode {
		t.Fatalf("first root child = %+v, want the doctype node", root.Children)
	}

	htmlElement := root.FirstChild("html")
	if htmlElement == nil {
		t.Fatal("no implicit html element")
	}

	head := htmlElement.FirstChild("head")
	body := htmlElement.FirstChild("body")

	if head == nil || body == nil {
		t.Fatalf("implicit wrappers missing: head=%v body=%v", head != nil, body != nil)
	}

	if head.FirstChild("style") == nil {
		t.Fatal("style not under the implicit head")
	}

	if body.FirstChild("link") == nil {
		t.Fatal("link not under the implicit body")
	}
}

// TestCollectSheetsParserInsertedTbody proves routing reaches a <style> inside
// a cell under a parser-inserted tbody. The source has no tbody, so the walk
// only works if the collector follows whatever wrappers the parser inserted.
func TestCollectSheetsParserInsertedTbody(t *testing.T) {
	t.Parallel()

	const source = `<table><tr><td><style>.in-cell { color: #f00 }</style></td></tr></table>`

	root, err := html.Parse(source)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	table := root.FindFirst(func(node *html.Node) bool { return node.Name == "table" })
	if table == nil {
		t.Fatal("table not found")
	}

	if table.FirstChild("tbody") == nil {
		t.Fatal("parser did not insert tbody")
	}

	sheets, logBuf, _ := collectImportSheets(t, importFixture{html: source, media: "print"})
	assertSheetClasses(t, sheets, []string{"in-cell"})

	if logBuf.Len() != 0 {
		t.Fatalf("collection warnings = %q", logBuf.String())
	}
}

// TestCollectSheetsStyleInTableContext proves a <style> that stays directly in
// the table during the in-table insertion mode is still routed, even with the
// tbody inserted after it.
func TestCollectSheetsStyleInTableContext(t *testing.T) {
	t.Parallel()

	const source = `<table><style>.in-table { color: #f00 }</style><tr><td>x</td></tr></table>`

	root, err := html.Parse(source)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	table := root.FindFirst(func(node *html.Node) bool { return node.Name == "table" })
	if table == nil || table.FirstChild("style") == nil {
		t.Fatal("style not a direct child of table")
	}

	sheets, logBuf, _ := collectImportSheets(t, importFixture{html: source, media: "print"})
	assertSheetClasses(t, sheets, []string{"in-table"})

	if logBuf.Len() != 0 {
		t.Fatalf("collection warnings = %q", logBuf.String())
	}
}

// TestCollectSheetsForeignSubtrees proves the walk crosses SVG and MathML
// subtrees without error and keeps collecting HTML stylesheets after them.
// Routing keys on element name only, so the SVG <style> also joins the
// document sheets; MathML has no style element and its subtree is traversed
// as generic elements.
func TestCollectSheetsForeignSubtrees(t *testing.T) {
	t.Parallel()

	const source = `<svg viewBox="0 0 10 10"><style>.svg-rule { fill: #f00 }</style>` +
		`<foreignObject><div>x</div></foreignObject></svg>` +
		`<math><mi>x</mi></math><style>.html-rule { color: #00f }</style>`

	root, err := html.Parse(source)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	svg := root.FindFirst(func(node *html.Node) bool { return node.Name == "svg" })
	if svg == nil || svg.Namespace != html.NamespaceSVG {
		t.Fatalf("svg subtree not in the SVG namespace: %+v", svg)
	}

	math := root.FindFirst(func(node *html.Node) bool { return node.Name == "math" })
	if math == nil || math.Namespace != html.NamespaceMathML {
		t.Fatalf("math subtree not in the MathML namespace: %+v", math)
	}

	sheets, logBuf, _ := collectImportSheets(t, importFixture{html: source, media: "print"})
	assertSheetClasses(t, sheets, []string{"svg-rule", "html-rule"})

	if logBuf.Len() != 0 {
		t.Fatalf("collection warnings = %q", logBuf.String())
	}
}

// TestCollectSheetsSkipsTemplateContents proves the walk does not collect a
// <style> that lives in a template's content fragment. Template content is
// inert until instantiated, so its styles must not join the document sheets.
func TestCollectSheetsSkipsTemplateContents(t *testing.T) {
	t.Parallel()

	const source = `<div><template><style>.in-template { color: #f00 }</style></template>` +
		`<style>.active { color: #0f0 }</style></div>`

	root, err := html.Parse(source)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	template := root.FindFirst(func(node *html.Node) bool { return node.Name == "template" })
	if template == nil {
		t.Fatal("template node not found")
	}

	if len(template.Contents) == 0 {
		t.Fatal("template has no content fragment")
	}

	if len(template.Children) != 0 {
		t.Fatalf("template Children = %d, want content kept out of Children", len(template.Children))
	}

	sheets, logBuf, _ := collectImportSheets(t, importFixture{html: source, media: "print"})
	assertSheetClasses(t, sheets, []string{"active"})

	if logBuf.Len() != 0 {
		t.Fatalf("collection warnings = %q", logBuf.String())
	}
}

// TestDocumentCollectsSheetsOnStandardTree runs the production
// prepare.Document path (load, ParseDocument, collectSheets) over one page
// that combines the implicit wrappers, a doctype, a parser-inserted tbody, an
// SVG subtree, and a linked stylesheet in body.
func TestDocumentCollectsSheetsOnStandardTree(t *testing.T) {
	t.Parallel()

	const page = `<!DOCTYPE html><title>t</title>` +
		`<style>.in-head { color: #f00 }</style>` +
		`<link rel="stylesheet" href="linked.css">` +
		`<table><tr><td><style>.in-cell { color: #0f0 }</style></td></tr></table>` +
		`<svg><path d="M0 0"/></svg>`

	pageURL, hits := serveIntegrationPage(t, page)

	loader, err := load.NewLoaderWithError(settings.LoadGlobal{})
	if err != nil {
		t.Fatalf("new loader: %v", err)
	}

	prep, err := prepare.Document(
		t.Context(),
		loader,
		pageURL,
		settings.DefaultLoadPage(),
		nil,
		prepare.Options{ViewportW: 600, ViewportH: 800, MediaType: "print"},
		io.Discard,
	)
	if err != nil {
		t.Fatalf("prepare.Document: %v", err)
	}

	if prep.Root == nil {
		t.Fatal("no parsed root")
	}

	if prep.Root.Mode != html.NoQuirks {
		t.Errorf("root mode = %s, want no-quirks", prep.Root.Mode)
	}

	htmlElement := prep.Root.FirstChild("html")
	if htmlElement == nil {
		t.Fatal("no implicit html element")
	}

	if htmlElement.FirstChild("head") == nil || htmlElement.FirstChild("body") == nil {
		t.Fatal("implicit head/body wrappers missing from the prepared root")
	}

	assertSheetClasses(t, prep.Sheets, []string{"in-head", "linked", "in-cell"})

	if hits["/linked.css"] != 1 {
		t.Fatalf("linked.css fetched %d times, want 1", hits["/linked.css"])
	}
}

// serveIntegrationPage serves the test page and its linked stylesheet,
// returning the page URL and the per-path fetch counts.
func serveIntegrationPage(t *testing.T, page string) (string, map[string]int) {
	t.Helper()

	hits := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		hits[request.URL.Path]++

		switch request.URL.Path {
		case "/page.html":
			responseWriter.Header().Set("Content-Type", "text/html")
			_, _ = responseWriter.Write([]byte(page))
		case "/linked.css":
			responseWriter.Header().Set("Content-Type", "text/css")
			_, _ = responseWriter.Write([]byte(`.linked { color: #00f }`))
		default:
			http.NotFound(responseWriter, request)
		}
	}))
	t.Cleanup(server.Close)

	return server.URL + "/page.html", hits
}
