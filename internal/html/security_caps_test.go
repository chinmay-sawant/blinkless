package html

import (
	"strconv"
	"strings"
	"testing"
)

// TestAttributeCountCap proves adversarial attribute counts cannot drive
// unbounded allocation: excess attributes are dropped during scanning.
func TestAttributeCountCap(t *testing.T) {
	t.Parallel()

	var src strings.Builder

	src.WriteString("<div")

	for i := range maxAttributesPerElement + 500 {
		src.WriteString(" a")
		src.WriteString(strings.Repeat("b", i/1000+1))
		src.WriteString(strings.Repeat("c", i%1000/100+1))
		src.WriteString(strings.Repeat("d", i%100/10+1))
		src.WriteString(strings.Repeat("e", i%10+1))
		src.WriteString(`="v"`)
	}

	src.WriteString(">x</div>")

	root := mustParse(t, src.String())
	div := root.FindFirst(func(n *Node) bool { return n.Name == "div" })

	if div == nil {
		t.Fatal("div not parsed")
	}

	if len(div.AttrList) != maxAttributesPerElement {
		t.Fatalf("AttrList = %d, want cap %d", len(div.AttrList), maxAttributesPerElement)
	}
}

// TestAttributeSizeCaps proves overlong names and values are truncated, and
// the per-element byte total stays bounded.
func TestAttributeSizeCaps(t *testing.T) {
	t.Parallel()

	longName := strings.Repeat("n", maxAttributeNameBytes+100)
	longValue := strings.Repeat("v", maxAttributeValueBytes+100)

	root := mustParse(t, `<div `+longName+`="x" short="`+longValue+`">x</div>`)
	div := root.FindFirst(func(n *Node) bool { return n.Name == "div" })

	if div == nil {
		t.Fatal("div not parsed")
	}

	total := 0

	for _, attr := range div.AttrList {
		if len(attr.Name) > maxAttributeNameBytes {
			t.Fatalf("attribute name %d bytes exceeds cap %d", len(attr.Name), maxAttributeNameBytes)
		}

		if len(attr.Value) > maxAttributeValueBytes {
			t.Fatalf("attribute value %d bytes exceeds cap %d", len(attr.Value), maxAttributeValueBytes)
		}

		total += len(attr.Name) + len(attr.Value)
	}

	if total > maxAttributeBytesPerElement {
		t.Fatalf("attribute total %d exceeds cap %d", total, maxAttributeBytesPerElement)
	}
}

// TestParseWithLimitRejectsOversize proves direct parsing of untrusted
// strings can enforce a byte bound.
func TestParseWithLimitRejectsOversize(t *testing.T) {
	t.Parallel()

	if _, err := ParseWithLimit(strings.Repeat("x", 100), 64); err == nil {
		t.Fatal("ParseWithLimit accepted input over the bound")
	}

	if _, err := ParseDocumentWithLimit([]byte(strings.Repeat("x", 100)), 64); err == nil {
		t.Fatal("ParseDocumentWithLimit accepted input over the bound")
	}

	if _, err := ParseWithLimit("<p>ok</p>", 64); err != nil {
		t.Fatalf("ParseWithLimit rejected small input: %v", err)
	}
}

// TestNumericRefDigitCap proves a long numeric reference consumes every
// digit with saturating arithmetic and yields one replacement rune.
func TestNumericRefDigitCap(t *testing.T) {
	t.Parallel()

	root := mustParse(t, "<p>&#x"+strings.Repeat("4", 100000)+";</p>")
	text := root.TextContent()

	if text != nulReplacement {
		t.Fatalf("numeric reference produced %d bytes, want one replacement rune", len(text))
	}
}

func TestNumericRefLongLeadingZeros(t *testing.T) {
	t.Parallel()

	for _, ref := range []string{"&#" + strings.Repeat("0", 100) + "65;", "&#x" + strings.Repeat("0", 100) + "41;"} {
		if got := UnescapeEntities(ref); got != "A" {
			t.Fatalf("long reference = %q, want A", got)
		}
	}
}

func TestSelectedContentEmptyNodesCap(t *testing.T) {
	t.Parallel()

	content := &Node{Type: ElementNode, Namespace: NamespaceHTML, Name: "selectedcontent"}
	option := &Node{Type: ElementNode, Namespace: NamespaceHTML, Name: "option"}
	selectNode := &Node{Type: ElementNode, Namespace: NamespaceHTML, Name: "select", Children: []*Node{content, option}}

	for range 10000 {
		option.Children = append(option.Children, &Node{Type: ElementNode, Name: "span"})
	}

	mirrorSelectedContent(selectNode)

	if len(content.Children) != 0 {
		t.Fatalf("cloned %d empty nodes past the budget", len(content.Children))
	}
}

// TestSelectedContentCloneCap proves a huge selected subtree is not cloned
// into selectedcontent.
func TestSelectedContentCloneCap(t *testing.T) {
	t.Parallel()

	big := strings.Repeat("x", maxSelectedContentBytes+100)
	root := mustParse(t, `<select><button><selectedcontent></button><option selected>`+big+`</option></select>`)
	content := root.FindFirst(func(n *Node) bool { return n.Name == "selectedcontent" })

	if content == nil {
		t.Fatal("selectedcontent missing from test input")
	}

	if len(content.Children) != 0 {
		t.Fatal("selectedcontent cloned a subtree over the byte cap")
	}
}

// TestForeignRawTextReparseFinishes proves nested svg style content is
// reparsed, and that reparse stops at maxForeignReparse instead of recursing
// for every nested raw-text element.
func TestForeignRawTextReparseFinishes(t *testing.T) {
	t.Parallel()

	shallow := mustParse(t, `<svg><style><svg><rect/></style></svg>`)
	if firstElement(shallow, "rect") == nil {
		t.Fatalf("nested svg style did not reparse:\n%s", treeString(shallow))
	}

	const nests = maxForeignReparse + 8

	var src strings.Builder

	for range nests {
		src.WriteString("<svg><style>")
	}

	src.WriteString("x")

	root := mustParse(t, src.String())
	svgs := 0

	root.Walk(func(n *Node) {
		if n.Type == ElementNode && n.Name == "svg" {
			svgs++
		}
	})

	if svgs == 0 {
		t.Fatal("deep svg style produced no svg element")
	}

	if svgs > maxForeignReparse+1 {
		t.Fatalf("svg count %d, reparse continued past cap %d", svgs, maxForeignReparse)
	}
}

// TestFosterParentedOneCharTokensFinish proves many one-character table text
// tokens still merge into the fostered text node and the parse returns.
func TestFosterParentedOneCharTokensFinish(t *testing.T) {
	t.Parallel()

	const tokenCount = 4000

	var src strings.Builder

	src.Grow(tokenCount * 10)
	src.WriteString("<table>")

	for range tokenCount {
		src.WriteString("x<!--c-->")
	}

	src.WriteString("</table>")

	root := mustParse(t, src.String())
	body := firstElement(root, "body")

	if body == nil || len(body.Children) == 0 || body.Children[0].Type != TextNode {
		t.Fatalf("foster text missing:\n%s", treeString(root))
	}

	if got := body.Children[0].Text; got != strings.Repeat("x", tokenCount) {
		t.Fatalf("foster text length %d, want %d", len(got), tokenCount)
	}
}

// TestActiveFormattingListCap proves distinct formatting elements cannot
// grow the active list past the cap, and a normal bold/italic misnest still
// reconstructs.
func TestActiveFormattingListCap(t *testing.T) {
	t.Parallel()

	builder := newTreeBuilder()

	for i := range maxActiveFormatting + 500 {
		builder.pushActiveFormatting(&Node{
			Type:      ElementNode,
			Name:      "b",
			Namespace: NamespaceHTML,
			AttrList:  []Attr{{Namespace: "", Name: "id", Value: strconv.Itoa(i)}},
		})
	}

	if got := len(builder.activeFormatting); got != maxActiveFormatting {
		t.Fatalf("active formatting list = %d, want cap %d", got, maxActiveFormatting)
	}

	const nests = 1500

	var src strings.Builder

	src.WriteString("<body>")

	for i := range nests {
		src.WriteString(`<b id="`)
		src.WriteString(strconv.Itoa(i))
		src.WriteString(`"><p>`)
	}

	parsed := newTreeBuilder()
	scanTokens(src.String(), parsed.appendToken, parsed.foreignCDATAAllowed)
	parsed.finish()

	if got := len(parsed.activeFormatting); got > maxActiveFormatting {
		t.Fatalf("parsed active formatting list = %d, over cap %d", got, maxActiveFormatting)
	}

	root := mustParse(t, `<b><i>one</b>two</i>`)
	body := firstElement(root, "body")

	if body == nil || len(body.Children) < 2 {
		t.Fatalf("reconstructed tree:\n%s", treeString(root))
	}

	if got := body.Children[1].TextContent(); got != "two" {
		t.Fatalf("reconstructed italic text = %q, want %q", got, "two")
	}
}
