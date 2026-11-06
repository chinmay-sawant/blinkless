package html

// Allocation regression tests for the HTML parser.
//
// TestParseGoldenStable pins the parsed tree shape (ordered AttrList, Attrs
// map contents, text, namespaces, template contents) over a corpus that
// exercises every touched path: duplicate/cased/boolean attributes, foreign
// attribute adjustment, select/selectedcontent mirroring, raw text, CRLF/BOM/
// invalid-UTF-8 normalization, and entity decoding. The expected strings were
// captured before the allocation work and must not change.
//
// TestParseAllocReduction proves the allocation win with
// testing.AllocsPerRun on Parse of a generated document. It is deliberately
// sequential: AllocsPerRun reads process-wide counters, so running beside
// other tests would pollute the measurement.
//
// TestParseNoAttrNilMaps pins item 2: elements without attributes carry nil
// maps instead of empty per-element allocations.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

func canonicalDump(n *Node) string {
	var b strings.Builder

	canonDumpNode(&b, n, 0)

	return b.String()
}

func canonDumpNode(b *strings.Builder, n *Node, depth int) {
	pad := strings.Repeat("  ", depth)

	switch n.Type {
	case TextNode:
		fmt.Fprintf(b, "%sT %q\n", pad, n.Text)
	case CommentNode:
		fmt.Fprintf(b, "%sC %q\n", pad, n.Text)
	case DoctypeNode:
		fmt.Fprintf(b, "%sD %q mode=%d\n", pad, n.Text, n.Mode)
	case ElementNode:
		fmt.Fprintf(b, "%sE %s ns=%d", pad, n.Name, n.Namespace)
		for _, a := range n.AttrList {
			fmt.Fprintf(b, " [%s|%s=%q]", a.Namespace, a.Name, a.Value)
		}
		if n.Attrs == nil {
			b.WriteString(" map=nil")
		} else {
			keys := make([]string, 0, len(n.Attrs))
			for k := range n.Attrs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			b.WriteString(" map={")
			for i, k := range keys {
				if i > 0 {
					b.WriteString(",")
				}
				fmt.Fprintf(b, "%s=%q", k, n.Attrs[k])
			}
			b.WriteString("}")
		}
		b.WriteString("\n")
		for _, c := range n.Children {
			canonDumpNode(b, c, depth+1)
		}
		if len(n.Contents) > 0 {
			fmt.Fprintf(b, "%s  contents:\n", pad)
			for _, c := range n.Contents {
				canonDumpNode(b, c, depth+2)
			}
		}
	}
}

// allocCorpus returns the golden corpus sources plus, last, the document-bytes
// input (BOM, CRLF, invalid UTF-8) parsed with ParseDocument.
func allocCorpus() ([]string, []byte) {
	long := strings.Repeat("v", 200)

	docs := []string{
		`<div ID="a" class="b" data-x="1" hidden id="dup">x</div>`,
		`<select><option value="1">one<option value="2" selected>two<selectedcontent></selectedcontent></select>`,
		`<p>hi <b>there</b> &amp; you</p>`,
		`<svg viewBox="0 0 10 10"><linearGradient id="g"><stop offset="0"/></linearGradient><a xlink:href="u" xml:lang="en">t</a></svg>`,
		`<table><tr><td>cell</td></tr></table><template><p>in tpl</p></template>`,
		"<p>a\r\nb\rc&#65;d\x00e</p><!--c--><!DOCTYPE html>",
		"<title>t &amp; t</title><script>if(a<b){c()}</script><textarea>x</textarea>",
		`<div a=b/ c="d" e>`,
		`<div class="x" id="y" data-a="1" data-b="2" data-c="3">` + long + `</div>`,
		`<select><option>a<option SELECTED>b</select>`,
	}

	raw := append([]byte("\xef\xbb\xbf<p>bom</p>\r\n"), 0xff, 0xfe)

	return docs, raw
}

func allocGolden() []string {
	long := strings.Repeat("v", 200)

	return []string{
		"E #document ns=0 map=nil\n" +
			"  E html ns=0 map=nil\n" +
			"    E head ns=0 map=nil\n" +
			"    E body ns=0 map=nil\n" +
			"      E div ns=0 [|id=\"a\"] [|class=\"b\"] [|data-x=\"1\"] [|hidden=\"\"] map={class=\"b\",data-x=\"1\",hidden=\"\",id=\"a\"}\n" +
			"        T \"x\"\n",
		"E #document ns=0 map=nil\n" +
			"  E html ns=0 map=nil\n" +
			"    E head ns=0 map=nil\n" +
			"    E body ns=0 map=nil\n" +
			"      E select ns=0 map=nil\n" +
			"        E option ns=0 [|value=\"1\"] map={value=\"1\"}\n" +
			"          T \"one\"\n" +
			"        E option ns=0 [|value=\"2\"] [|selected=\"\"] map={selected=\"\",value=\"2\"}\n" +
			"          T \"two\"\n" +
			"          E selectedcontent ns=0 map=nil\n" +
			"            T \"two\"\n" +
			"            E selectedcontent ns=0 map=nil\n" +
			"              T \"two\"\n",
		"E #document ns=0 map=nil\n" +
			"  E html ns=0 map=nil\n" +
			"    E head ns=0 map=nil\n" +
			"    E body ns=0 map=nil\n" +
			"      E p ns=0 map=nil\n" +
			"        T \"hi \"\n" +
			"        E b ns=0 map=nil\n" +
			"          T \"there\"\n" +
			"        T \" & you\"\n",
		"E #document ns=0 map=nil\n" +
			"  E html ns=0 map=nil\n" +
			"    E head ns=0 map=nil\n" +
			"    E body ns=0 map=nil\n" +
			"      E svg ns=1 [|viewBox=\"0 0 10 10\"] map={viewBox=\"0 0 10 10\"}\n" +
			"        E linearGradient ns=1 [|id=\"g\"] map={id=\"g\"}\n" +
			"          E stop ns=1 [|offset=\"0\"] map={offset=\"0\"}\n" +
			"        E a ns=1 [xlink|href=\"u\"] [xml|lang=\"en\"] map={xlink:href=\"u\",xml:lang=\"en\"}\n" +
			"          T \"t\"\n",
		"E #document ns=0 map=nil\n" +
			"  E html ns=0 map=nil\n" +
			"    E head ns=0 map=nil\n" +
			"    E body ns=0 map=nil\n" +
			"      E table ns=0 map=nil\n" +
			"        E tbody ns=0 map=nil\n" +
			"          E tr ns=0 map=nil\n" +
			"            E td ns=0 map=nil\n" +
			"              T \"cell\"\n" +
			"      E template ns=0 map=nil\n" +
			"        contents:\n" +
			"          E p ns=0 map=nil\n" +
			"            T \"in tpl\"\n",
		"E #document ns=0 map=nil\n" +
			"  E html ns=0 map=nil\n" +
			"    E head ns=0 map=nil\n" +
			"    E body ns=0 map=nil\n" +
			"      E p ns=0 map=nil\n" +
			"        T \"a\\nb\\ncAde\"\n" +
			"      C \"c\"\n",
		"E #document ns=0 map=nil\n" +
			"  E html ns=0 map=nil\n" +
			"    E head ns=0 map=nil\n" +
			"      E title ns=0 map=nil\n" +
			"        T \"t & t\"\n" +
			"      E script ns=0 map=nil\n" +
			"        T \"if(a<b){c()}\"\n" +
			"    E body ns=0 map=nil\n" +
			"      E textarea ns=0 map=nil\n" +
			"        T \"x\"\n",
		"E #document ns=0 map=nil\n" +
			"  E html ns=0 map=nil\n" +
			"    E head ns=0 map=nil\n" +
			"    E body ns=0 map=nil\n" +
			"      E div ns=0 [|a=\"b/\"] [|c=\"d\"] [|e=\"\"] map={a=\"b/\",c=\"d\",e=\"\"}\n",
		"E #document ns=0 map=nil\n" +
			"  E html ns=0 map=nil\n" +
			"    E head ns=0 map=nil\n" +
			"    E body ns=0 map=nil\n" +
			"      E div ns=0 [|class=\"x\"] [|id=\"y\"] [|data-a=\"1\"] [|data-b=\"2\"] [|data-c=\"3\"] map={class=\"x\",data-a=\"1\",data-b=\"2\",data-c=\"3\",id=\"y\"}\n" +
			"        T \"" + long + "\"\n",
		"E #document ns=0 map=nil\n" +
			"  E html ns=0 map=nil\n" +
			"    E head ns=0 map=nil\n" +
			"    E body ns=0 map=nil\n" +
			"      E select ns=0 map=nil\n" +
			"        E option ns=0 map=nil\n" +
			"          T \"a\"\n" +
			"        E option ns=0 [|selected=\"\"] map={selected=\"\"}\n" +
			"          T \"b\"\n",
		"E #document ns=0 map=nil\n" +
			"  E html ns=0 map=nil\n" +
			"    E head ns=0 map=nil\n" +
			"    E body ns=0 map=nil\n" +
			"      E p ns=0 map=nil\n" +
			"        T \"bom\"\n" +
			"      T \"\\n\ufffd\ufffd\"\n",
	}
}

func TestParseGoldenStable(t *testing.T) {
	t.Parallel()

	docs, raw := allocCorpus()
	want := allocGolden()

	for i, src := range docs {
		root, err := Parse(src)
		if err != nil {
			t.Fatalf("doc %d: %v", i, err)
		}
		if got := canonicalDump(root); got != want[i] {
			t.Errorf("doc %d mismatch:\n got:\n%s\nwant:\n%s", i, got, want[i])
		}
	}

	root, err := ParseDocument(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got, wantRaw := canonicalDump(root), want[len(want)-1]; got != wantRaw {
		t.Errorf("raw doc mismatch:\n got:\n%s\nwant:\n%s", got, wantRaw)
	}
}

// allocBenchDoc builds the generated document for the allocation test: many
// small elements with a few attributes each, the shape that dominated
// per-tag tokenizer scratch before pooling.
func allocBenchDoc() string {
	var sb strings.Builder

	sb.WriteString("<html><body>")
	for range 500 {
		sb.WriteString(`<div class="c" id="x" data-v="v">text &amp; more</div>`)
	}
	sb.WriteString(`<select><option>a<option selected>b</select>`)
	sb.WriteString("</body></html>")

	return sb.String()
}

func TestParseAllocReduction(t *testing.T) {
	// Sequential on purpose: AllocsPerRun reads process-wide allocation
	// counters, so a parallel test beside the suite would pollute it.
	doc := allocBenchDoc()

	// Sanity: the document parses to the expected shape before measuring.
	root, err := Parse(doc)
	if err != nil {
		t.Fatal(err)
	}
	divs := 0
	root.Walk(func(n *Node) {
		if n.Type == ElementNode && n.Name == "div" {
			divs++
		}
	})
	if divs != 500 {
		t.Fatalf("bench doc has %d divs, want 500", divs)
	}

	// Pre-edit baseline: 9037 allocs per Parse. The tokenizer scratch pooling
	// (item 5), running attr-byte total (item 9), and pre-sized attribute
	// stores (items 1-2) cut this to ~7035. The ceiling leaves headroom for
	// allocator noise while still failing if the pooling regresses.
	const allocCeiling = 8000.0

	if got := testing.AllocsPerRun(100, func() {
		if _, err := Parse(doc); err != nil {
			t.Error(err)
		}
	}); got >= allocCeiling {
		t.Errorf("AllocsPerRun(Parse) = %v, want below %v", got, allocCeiling)
	}
}

func TestParseNoAttrNilMaps(t *testing.T) {
	t.Parallel()

	root, err := Parse(`<div><p>hi</p><br><img src="y.png"><span title="t">x</span></div>`)
	if err != nil {
		t.Fatal(err)
	}

	root.Walk(func(n *Node) {
		if n.Type != ElementNode {
			return
		}
		if len(n.AttrList) == 0 && (n.Attrs != nil || n.AttrList != nil) {
			t.Errorf("<%s> without attributes carries allocated stores: Attrs=%v AttrList=%v",
				n.Name, n.Attrs, n.AttrList)
		}
		if len(n.AttrList) > 0 && n.Attrs == nil {
			t.Errorf("<%s> with attributes has nil Attrs map", n.Name)
		}
	})
}
