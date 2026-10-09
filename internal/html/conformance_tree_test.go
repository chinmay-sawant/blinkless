//nolint:all // conformance harness: corpus format parsing and diff plumbing, not product code
package html

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// --- tree-construction category ---

type datDocument struct {
	lines []string
}

type datRecord struct {
	input      string
	fragment   string
	scriptOff  *datDocument
	scriptOn   *datDocument
	defaultDoc *datDocument
}

func runTreeCategory(t *testing.T, plan conformanceCategoryPlan, report *conformanceReport, scripting bool) {
	t.Helper()

	for _, path := range plan.files {
		rel := relativeSlash(report.dir, path)
		fileReason := plan.fileReasons[rel]

		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("conformance harness: read %s: %v", rel, err)
		}

		records := parseDatRecords(t, rel, string(raw))

		for i := range records {
			id := fmt.Sprintf("%s#%d", rel, i+1)
			record := records[i]

			switch {
			case plan.wholeReason != "":
				report.addCase(plan.category.ID, id, statusUnsupported, plan.wholeReason, "")
			case fileReason != "":
				report.addCase(plan.category.ID, id, statusUnsupported, fileReason, "")
			case record.fragment != "":
				report.addCase(plan.category.ID, id, statusUnsupported, reasonDocumentFragment, "")
			default:
				classifyTreeRecord(t, record, id, plan.category.ID, scripting, report)
			}
		}
	}
}

// parseDatRecords splits a tree-construction .dat file into records. The
// format (html5lib-tests tree-construction/README.md) is: #data, the input
// lines up to #errors, error lines up to the next marker, optional
// #new-errors, optional #document-fragment context, optional #script-off or
// #script-on, and #document followed by the dump. A dump section runs to the
// next "#" heading; dump nodes may span physical lines (text and comment
// data keep raw newlines), so continuation lines that do not start with "|"
// are folded back into the previous node.
func parseDatRecords(t *testing.T, path, content string) []datRecord {
	t.Helper()

	lines := strings.Split(content, "\n")
	records := make([]datRecord, 0)

	for i := 0; i < len(lines); {
		line := lines[i]

		if line != "#data" {
			if strings.HasPrefix(line, "#") {
				t.Fatalf("conformance harness: %s: unexpected marker %q at line %d", path, line, i+1)
			}

			i++

			continue
		}

		record := datRecord{}
		i++

		var dataLines []string

		for i < len(lines) && lines[i] != "#errors" {
			dataLines = append(dataLines, lines[i])
			i++
		}

		if i >= len(lines) {
			t.Fatalf("conformance harness: %s: #data without #errors", path)
		}

		record.input = strings.Join(dataLines, "\n")
		i++ // consume #errors
		i = skipDatSection(lines, i)

		for i < len(lines) {
			line = lines[i]

			if line == "" || line == "#data" {
				break
			}

			switch line {
			case "#new-errors":
				i = skipDatSection(lines, i+1)
			case "#document-fragment":
				if i+1 >= len(lines) {
					t.Fatalf("conformance harness: %s: #document-fragment without context", path)
				}

				record.fragment = lines[i+1]
				i += 2
			case "#script-off", "#script-on":
				mode := line
				i++

				if i >= len(lines) || lines[i] != "#document" {
					t.Fatalf("conformance harness: %s: %s without #document", path, mode)
				}

				i++

				dump, next := readDumpLines(lines, i)

				if mode == "#script-off" {
					record.scriptOff = &datDocument{lines: dump}
				} else {
					record.scriptOn = &datDocument{lines: dump}
				}

				i = next
			case "#document":
				i++

				dump, next := readDumpLines(lines, i)
				record.defaultDoc = &datDocument{lines: dump}
				i = next
			default:
				t.Fatalf("conformance harness: %s: unexpected marker %q at line %d", path, line, i+1)
			}
		}

		records = append(records, record)
	}

	return records
}

func skipDatSection(lines []string, i int) int {
	for i < len(lines) && lines[i] != "" && !isDatMarker(lines[i]) {
		i++
	}

	return i
}

func isDatMarker(line string) bool {
	switch line {
	case "#data", "#errors", "#new-errors", "#document-fragment", "#script-off", "#script-on", "#document":
		return true
	}

	return false
}

// readDumpLines reads a #document dump section: every line up to the next
// "#" heading, including blank continuation lines that belong to text or
// comment data. Exactly one trailing blank line separates records and is
// stripped; the corpus audit shows every section has exactly that one.
func readDumpLines(lines []string, i int) ([]string, int) {
	dump := make([]string, 0)

	for i < len(lines) && !strings.HasPrefix(lines[i], "#") {
		dump = append(dump, lines[i])
		i++
	}

	if len(dump) > 0 && dump[len(dump)-1] == "" {
		dump = dump[:len(dump)-1]
	}

	return dump, i
}

// selectDatDocument picks the expected document for the pinned scripting
// mode. Records with only the other mode's document are skipped by the
// caller (ok=false). A nil document with ok=true means the record is
// malformed and the caller fails the harness.
func selectDatDocument(record datRecord, scripting bool) (*datDocument, bool) {
	if scripting {
		if record.scriptOn != nil {
			return record.scriptOn, true
		}

		if record.scriptOff != nil {
			return nil, false
		}

		return record.defaultDoc, true
	}

	if record.scriptOff != nil {
		return record.scriptOff, true
	}

	if record.scriptOn != nil {
		return nil, false
	}

	return record.defaultDoc, true
}

func classifyTreeRecord(t *testing.T, record datRecord, id, category string, scripting bool, report *conformanceReport) {
	t.Helper()

	doc, ok := selectDatDocument(record, scripting)
	if !ok {
		report.addCase(category, id, statusSkipped, reasonScriptingOnOnly, "")
		return
	}

	if doc == nil {
		t.Fatalf("conformance harness: %s: record has no #document section", id)
	}

	root, err := Parse(record.input)
	if err != nil {
		report.addCase(category, id, statusFailed, reasonEngineError+err.Error(), "")
		return
	}

	want := parseTreeDump(t, id, doc.lines)
	got := engineTree(root)
	flags := &cfFlags{}

	if mismatch := compareTree(want, got, kindDocument, flags); mismatch != nil {
		report.addCase(category, id, statusFailed, "tree-mismatch", mismatchDetail(mismatch))
		return
	}

	switch {
	case flags.templateContents:
		report.addCase(category, id, statusUnsupported, reasonTemplateContents, "")
	case flags.pi:
		report.addCase(category, id, statusUnsupported, reasonProcessingInstr, "")
	default:
		report.addCase(category, id, statusPassed, "", "")
	}
}

// --- tree dump parsing ---

type dumpNode struct {
	depth   int
	content string
}

func parseTreeDump(t *testing.T, id string, dump []string) *cfNode {
	t.Helper()

	nodes := groupDumpNodes(t, id, dump)
	root := &cfNode{kind: kindDocument}
	stack := []*cfNode{root}

	for _, node := range nodes {
		if isDumpAttr(node.content) {
			attachDumpAttr(t, id, stack, node)

			continue
		}

		parsed := parseDumpNode(t, id, node)
		if node.depth >= len(stack) {
			t.Fatalf("conformance harness: %s: dump depth %d has no parent node", id, node.depth)
		}

		parent := stack[node.depth]
		parent.children = append(parent.children, parsed)
		stack = stack[:node.depth+1]

		if parsed.kind == kindElement || parsed.kind == kindContent {
			stack = append(stack, parsed)
		}
	}

	return root
}

func groupDumpNodes(t *testing.T, id string, dump []string) []dumpNode {
	t.Helper()

	nodes := make([]dumpNode, 0, len(dump))

	for _, line := range dump {
		if strings.HasPrefix(line, "|") {
			spaces := 0

			for spaces < len(line)-1 && line[1+spaces] == ' ' {
				spaces++
			}

			if spaces < 1 || (spaces-1)%dumpIndentStep != 0 {
				t.Fatalf("conformance harness: %s: malformed dump indent in %q", id, line)
			}

			nodes = append(nodes, dumpNode{depth: (spaces - 1) / dumpIndentStep, content: line[1+spaces:]})

			continue
		}

		if len(nodes) == 0 {
			t.Fatalf("conformance harness: %s: dump continuation line without a node: %q", id, line)
		}

		nodes[len(nodes)-1].content += "\n" + line
	}

	return nodes
}

// isDumpAttr distinguishes attribute lines (name="value") from node lines.
// Attribute names may start with "<" (webkit01.dat has one named "<"), so
// the node shapes are matched first and only the remaining "=" forms are
// treated as attributes.
func isDumpAttr(content string) bool {
	if !strings.Contains(content, `="`) {
		return false
	}

	switch {
	case strings.HasPrefix(content, `"`):
		return false
	case strings.HasPrefix(content, "<!--"):
		return false
	case strings.HasPrefix(content, "<!DOCTYPE"):
		return false
	case strings.HasPrefix(content, "<?"):
		return false
	case strings.HasPrefix(content, "<") && strings.HasSuffix(content, ">"):
		return false
	case content == kindContent:
		return false
	}

	return true
}

func parseDumpNode(t *testing.T, id string, node dumpNode) *cfNode {
	t.Helper()

	content := node.content

	switch {
	case strings.HasPrefix(content, dumpCommentOpen):
		return &cfNode{kind: kindComment, data: trimDumpComment(t, id, content)}
	case strings.HasPrefix(content, "<!DOCTYPE"):
		return parseDumpDoctype(t, id, content)
	case strings.HasPrefix(content, "<?"):
		return &cfNode{kind: kindPI, data: strings.TrimSuffix(strings.TrimPrefix(content, "<?"), ">")}
	case strings.HasPrefix(content, "<"):
		return parseDumpElement(t, id, content)
	case strings.HasPrefix(content, `"`):
		if len(content) < 2 || !strings.HasSuffix(content, `"`) {
			t.Fatalf("conformance harness: %s: malformed text dump %q", id, content)
		}

		return &cfNode{kind: kindText, data: content[1 : len(content)-1]}
	case content == kindContent:
		return &cfNode{kind: kindContent}
	default:
		t.Fatalf("conformance harness: %s: unrecognized dump line %q", id, content)
	}

	return nil
}

func trimDumpComment(t *testing.T, id, content string) string {
	t.Helper()

	if !strings.HasSuffix(content, dumpCommentClose) || len(content) < len(dumpCommentOpen)+len(dumpCommentClose) {
		t.Fatalf("conformance harness: %s: malformed comment dump %q", id, content)
	}

	return content[len(dumpCommentOpen) : len(content)-len(dumpCommentClose)]
}

func parseDumpElement(t *testing.T, id, content string) *cfNode {
	t.Helper()

	if len(content) < 3 || !strings.HasSuffix(content, ">") {
		t.Fatalf("conformance harness: %s: malformed element dump %q", id, content)
	}

	inner := content[1 : len(content)-1]
	node := &cfNode{kind: kindElement, name: inner}

	parts := strings.Split(inner, " ")

	switch len(parts) {
	case 1:
	case 2:
		if parts[0] != "svg" && parts[0] != "math" {
			t.Fatalf("conformance harness: %s: unknown namespace designator in %q", id, content)
		}

		node.ns = parts[0]
		node.name = parts[1]
	default:
		t.Fatalf("conformance harness: %s: malformed element dump %q", id, content)
	}

	return node
}

func attachDumpAttr(t *testing.T, id string, stack []*cfNode, node dumpNode) {
	t.Helper()

	if node.depth == 0 || node.depth >= len(stack) {
		t.Fatalf("conformance harness: %s: attribute without an element parent: %q", id, node.content)
	}

	parent := stack[node.depth]
	if parent.kind != kindElement {
		t.Fatalf("conformance harness: %s: attribute on non-element %q", id, node.content)
	}

	eq := strings.Index(node.content, "=")
	name := node.content[:eq]
	value := node.content[eq+1:]

	if len(value) < 2 || !strings.HasPrefix(value, `"`) || !strings.HasSuffix(value, `"`) {
		t.Fatalf("conformance harness: %s: malformed attribute dump %q", id, node.content)
	}

	attr := cfAttr{value: value[1 : len(value)-1]}
	attr.ns, attr.name = splitDumpAttrName(name)
	parent.attrs = append(parent.attrs, attr)
}

func splitDumpAttrName(name string) (string, string) {
	for _, ns := range []string{"xlink", "xml", "xmlns"} {
		if strings.HasPrefix(name, ns+" ") {
			return ns, name[len(ns)+1:]
		}
	}

	return "", name
}

func parseDumpDoctype(t *testing.T, id, content string) *cfNode {
	t.Helper()

	s := strings.TrimPrefix(content, "<!DOCTYPE")
	s = strings.TrimPrefix(s, " ")

	if !strings.HasSuffix(s, ">") {
		t.Fatalf("conformance harness: %s: malformed doctype dump %q", id, content)
	}

	s = strings.TrimSuffix(s, ">")

	doctype := cfDoctype{}
	doctype.name, s = cutWord(s)
	s = strings.TrimLeft(s, " ")

	if s != "" {
		public, rest, ok := cutQuoted(s)
		if !ok {
			t.Fatalf("conformance harness: %s: malformed doctype public id in %q", id, content)
		}

		doctype.public, doctype.publicSet = public, true

		// The dump serializer does not escape quotes inside ids, so the
		// system id runs from its opening quote to the final quote of the
		// line (the corpus has ids like "taco"").
		rest = strings.TrimLeft(rest, " ")

		if len(rest) < 2 || !strings.HasPrefix(rest, `"`) || !strings.HasSuffix(rest, `"`) {
			t.Fatalf("conformance harness: %s: malformed doctype system id in %q", id, content)
		}

		doctype.system, doctype.systemSet = rest[1:len(rest)-1], true
	}

	return &cfNode{kind: kindDoctype, doctype: doctype, doctypeOK: true}
}

// --- engine tree conversion ---

func engineTree(root *Node) *cfNode {
	doc := &cfNode{kind: kindDocument}

	for _, child := range root.Children {
		doc.children = append(doc.children, engineNode(child))
	}

	return doc
}

func engineNode(node *Node) *cfNode {
	switch node.Type {
	case ElementNode:
		out := &cfNode{kind: kindElement, name: node.Name, ns: namespaceDesignator(node.Namespace), attrs: engineAttrs(node)}

		for _, child := range node.Children {
			out.children = append(out.children, engineNode(child))
		}

		return out
	case TextNode:
		return &cfNode{kind: kindText, data: node.Text}
	case CommentNode:
		return &cfNode{kind: kindComment, data: node.Text}
	case DoctypeNode:
		return &cfNode{kind: kindDoctype, data: node.Text, doctypeOK: true, doctype: cfDoctype{
			name:      node.Doctype.Name,
			public:    node.Doctype.PublicID,
			publicSet: node.Doctype.HasPublicID,
			system:    node.Doctype.SystemID,
			systemSet: node.Doctype.HasSystemID,
		}}
	case NodeUnknown:
		return &cfNode{kind: "unknown", data: node.Text}
	}

	return &cfNode{kind: "unknown", data: node.Text}
}

// namespaceDesignator maps an engine namespace to the html5lib dump
// designator ("svg", "math") and HTML to the empty string.
func namespaceDesignator(ns Namespace) string {
	switch ns {
	case NamespaceSVG:
		return "svg"
	case NamespaceMathML:
		return "math"
	case NamespaceHTML:
		return ""
	default:
		return ""
	}
}

func engineAttrs(node *Node) []cfAttr {
	out := make([]cfAttr, 0, len(node.AttrList))
	for _, attr := range node.AttrList {
		out = append(out, cfAttr{ns: attr.Namespace, name: attr.Name, value: attr.Value})
	}

	if len(out) == 0 && len(node.Attrs) > 0 {
		// Synthetic nodes built without AttrList: fall back to the flat map.
		for name, value := range node.Attrs {
			out = append(out, cfAttr{name: name, value: value})
		}
	}

	sortCFAttrs(out)

	return out
}

func sortCFAttrs(attrs []cfAttr) {
	sort.Slice(attrs, func(i, j int) bool {
		if attrs[i].ns != attrs[j].ns {
			return attrs[i].ns < attrs[j].ns
		}

		return attrs[i].name < attrs[j].name
	})
}

// --- tree comparison ---

type cfFlags struct {
	templateContents bool
	pi               bool
}

type cfMismatch struct {
	path string
	want []string
	got  []string
}

func compareTree(want, got *cfNode, path string, flags *cfFlags) *cfMismatch {
	if want.kind != got.kind {
		return &cfMismatch{path: path, want: []string{want.kind}, got: []string{got.kind}}
	}

	switch want.kind {
	case kindDocument:
		return compareTreeChildren(want, got, path, flags)
	case kindElement:
		if want.ns != got.ns {
			return &cfMismatch{
				path: path + " namespace",
				want: []string{namespacePrefix(want.ns) + want.name},
				got:  []string{namespacePrefix(got.ns) + got.name},
			}
		}

		if want.name != got.name {
			return &cfMismatch{path: path + " name", want: []string{want.name}, got: []string{got.name}}
		}

		if mismatch := compareTreeAttrs(want.attrs, got.attrs, path); mismatch != nil {
			return mismatch
		}

		return compareTreeChildren(want, got, path, flags)
	case kindText, kindComment:
		if want.data != got.data {
			return &cfMismatch{path: path, want: []string{quoteDump(want.data)}, got: []string{quoteDump(got.data)}}
		}
	case kindDoctype:
		return compareDoctypes(want, got, path)
	case kindPI:
		flags.pi = true
	case kindContent:
		flags.templateContents = true
	}

	return nil
}

func compareTreeChildren(want, got *cfNode, path string, flags *cfFlags) *cfMismatch {
	wantChildren := flattenWantChildren(want.children, flags)

	if len(wantChildren) != len(got.children) {
		return &cfMismatch{path: path + " children", want: renderTree(want), got: renderTree(got)}
	}

	for i := range wantChildren {
		childPath := fmt.Sprintf("%s > %s[%d]", path, wantChildren[i].kind, i)

		if mismatch := compareTree(wantChildren[i], got.children[i], childPath, flags); mismatch != nil {
			return mismatch
		}
	}

	return nil
}

// flattenWantChildren unwraps template "content" nodes (flagging the missing
// engine field) and drops expected processing instructions, which the HTML
// parser cannot emit and the engine therefore cannot represent.
func flattenWantChildren(children []*cfNode, flags *cfFlags) []*cfNode {
	out := make([]*cfNode, 0, len(children))

	for _, child := range children {
		switch child.kind {
		case kindContent:
			flags.templateContents = true

			out = append(out, flattenWantChildren(child.children, flags)...)
		case kindPI:
			flags.pi = true
		default:
			out = append(out, child)
		}
	}

	return out
}

func compareTreeAttrs(want, got []cfAttr, path string) *cfMismatch {
	if len(want) != len(got) {
		return &cfMismatch{path: path + " attributes", want: renderAttrs(want), got: renderAttrs(got)}
	}

	for _, wantAttr := range want {
		gotAttr, ok := findCFAttr(got, wantAttr.ns, wantAttr.name)
		if !ok || gotAttr.value != wantAttr.value {
			return &cfMismatch{
				path: path + " attribute " + namespacePrefix(wantAttr.ns) + wantAttr.name,
				want: []string{quoteDump(wantAttr.value)},
				got:  []string{quoteDump(attrValueOrMissing(gotAttr, ok))},
			}
		}
	}

	return nil
}

func compareDoctypes(want, got *cfNode, path string) *cfMismatch {
	if !got.doctypeOK {
		return &cfMismatch{path: path, want: []string{renderDoctype(want.doctype)}, got: []string{got.data}}
	}

	if !sameTreeDoctype(want.doctype, got.doctype) {
		return &cfMismatch{path: path, want: []string{renderDoctype(want.doctype)}, got: []string{renderDoctype(got.doctype)}}
	}

	return nil
}

// sameTreeDoctype compares dump-derived doctypes. The .dat format prints a
// missing identifier as an empty quoted string once the other identifier is
// present, so the missing/empty distinction is not comparable here; names and
// identifier values are.
func sameTreeDoctype(want, got cfDoctype) bool {
	return strings.EqualFold(want.name, got.name) &&
		want.public == got.public && want.system == got.system
}

func findCFAttr(attrs []cfAttr, ns, name string) (cfAttr, bool) {
	for _, attr := range attrs {
		if attr.ns == ns && attr.name == name {
			return attr, true
		}
	}

	return cfAttr{}, false
}

func attrValueOrMissing(attr cfAttr, ok bool) string {
	if !ok {
		return "(missing)"
	}

	return attr.value
}

func renderTree(node *cfNode) []string {
	lines := make([]string, 0)
	appendTreeLines(&lines, node, 0)

	return lines
}

func appendTreeLines(lines *[]string, node *cfNode, depth int) {
	if node.kind == kindDocument {
		for _, child := range node.children {
			appendTreeLines(lines, child, depth)
		}

		return
	}

	*lines = append(*lines, dumpLine(depth, node))

	if node.kind == kindElement {
		attrs := append([]cfAttr(nil), node.attrs...)
		sortCFAttrs(attrs)

		for _, attr := range attrs {
			*lines = append(*lines, dumpAttrLine(depth+1, attr))
		}
	}

	for _, child := range node.children {
		appendTreeLines(lines, child, depth+1)
	}
}

func dumpLine(depth int, node *cfNode) string {
	prefix := "| " + strings.Repeat("  ", depth)

	switch node.kind {
	case kindElement:
		return prefix + "<" + namespacePrefix(node.ns) + node.name + ">"
	case kindText:
		return prefix + quoteDump(node.data)
	case kindComment:
		return prefix + dumpCommentOpen + escapeDumpText(node.data) + dumpCommentClose
	case kindDoctype:
		return prefix + renderDoctype(node.doctype)
	case kindContent:
		return prefix + kindContent
	case kindPI:
		return prefix + "<?" + escapeDumpText(node.data) + ">"
	default:
		return prefix + node.kind
	}
}

func dumpAttrLine(depth int, attr cfAttr) string {
	prefix := "| " + strings.Repeat("  ", depth)

	return prefix + namespacePrefix(attr.ns) + attr.name + `="` + escapeDumpText(attr.value) + `"`
}

func namespacePrefix(ns string) string {
	if ns == "" {
		return ""
	}

	return ns + " "
}

func renderAttrs(attrs []cfAttr) []string {
	lines := make([]string, 0, len(attrs))
	for _, attr := range attrs {
		lines = append(lines, dumpAttrLine(0, attr))
	}

	return lines
}
