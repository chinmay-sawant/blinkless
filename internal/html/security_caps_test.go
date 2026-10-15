package html

import (
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
