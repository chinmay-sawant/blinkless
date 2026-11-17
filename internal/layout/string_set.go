// Named-string table for CSS string-set (css-content-3).
//
// The cascade stores the used value on __string_set. collectCascadedNamedStrings
// reads those values in document order. The first assignment of a name wins,
// which is what string() paints on this one canvas. string-set: none clears
// the table. A token ParseStringSet rejects does not write, so a bad token
// leaves an earlier assignment in place.
//
// Quoted strings concatenate. content() and content(text) read the element's
// own text. content(before) and content(after) read that pseudo-element, or
// nothing when it has no content. Other functions (counter(), attr()) stay
// invalid. There is no painted page-margin box, so running headers are not
// painted. The table is per document, not per page.
//
// CollectNamedStrings still reads inline style attributes only. Paint does
// not call it; it calls the cascaded collector so a stylesheet rule and an
// inline declaration share one table.
package layout

import (
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// StringSetAssignment is one parsed named-string assignment: the lower-cased
// identifier plus its used value (the concatenated decoded strings). pieces
// keeps content() terms so a later pass can read the element's text. Quoted
// strings still land in Value, which is what callers that only store literals
// assert.
type StringSetAssignment struct {
	Name  string
	Value string

	pieces []stringSetPiece
}

// stringSetPiece is one term after the name. fn is false for a quoted string.
// keyword is empty for content(), or text, before, after, first-letter, or
// marker. Empty and text both mean the element's own text.
type stringSetPiece struct {
	literal string
	keyword string
	fn      bool
}

// stringSetRevertLayer is the CSS-wide revert-layer keyword, kept as a named
// constant so the two keyword switches share one spelling.
const stringSetRevertLayer = "revert-layer"

// NamedStringTable is a per-document named-string store. The first assignment
// for a name wins; later assignments for the same name are ignored. This is
// the lite single-document form of the GCPM rule that the first assignment on
// a page supplies string(name, first).
type NamedStringTable struct {
	values map[string]string
	order  []string
}

// NewNamedStringTable returns an empty table.
func NewNamedStringTable() *NamedStringTable {
	return &NamedStringTable{
		values: make(map[string]string),
		order:  nil,
	}
}

// Set records name=value unless name is already present. Names match
// ASCII case-insensitively, consistent with CSS ident handling elsewhere
// (internal/layout/style_properties.go lower-cases the page name).
func (table *NamedStringTable) Set(name, value string) {
	if table == nil {
		return
	}

	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return
	}

	if table.values == nil {
		table.values = make(map[string]string)
	}

	if _, present := table.values[key]; present {
		return
	}

	table.values[key] = value
	table.order = append(table.order, key)
}

// Lookup reports the stored used value for name.
func (table *NamedStringTable) Lookup(name string) (string, bool) {
	if table == nil {
		return "", false
	}

	value, ok := table.values[strings.ToLower(strings.TrimSpace(name))]

	return value, ok
}

// Names lists stored names in first-assignment order.
func (table *NamedStringTable) Names() []string {
	if table == nil {
		return nil
	}

	return append([]string(nil), table.order...)
}

// Len reports how many names are stored.
func (table *NamedStringTable) Len() int {
	if table == nil {
		return 0
	}

	return len(table.values)
}

// Clear drops every name. string-set: none uses this so a later string()
// read does not paint a value assigned earlier in the document.
func (table *NamedStringTable) Clear() {
	if table == nil {
		return
	}

	table.values = map[string]string{}
	table.order = nil
}

// ParseStringSet parses a string-set declaration value into used-value
// assignments in source order. It reports false for anything outside the
// supported syntax, in which case the caller keeps the prior state and the
// table is untouched. none and initial are valid and yield zero assignments
// (initial is none). Quoted strings and content() are valid terms. Other
// functions, unquoted tokens, a bare identifier with no term, and CSS-wide
// keywords as names are invalid.
func ParseStringSet(raw string) ([]StringSetAssignment, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, false
	}

	switch strings.ToLower(trimmed) {
	case "none", cssKeywordInitial:
		return nil, true
	case inheritKeyword, cssKeywordUnset, flowKeywordDefault, cssKeywordRevert, stringSetRevertLayer:
		return nil, false
	}

	items := splitStringSetList(trimmed)

	assignments := make([]StringSetAssignment, 0, len(items))

	for _, item := range items {
		assignment, ok := parseStringSetItem(item)
		if !ok {
			return nil, false
		}

		assignments = append(assignments, assignment)
	}

	return assignments, true
}

// CollectNamedStrings builds the per-document table from the inline style
// attributes in document order (html.Node.Walk is pre-order). When an element
// declares string-set twice in one attribute the last declaration wins, as in
// the cascade. Stylesheet rules are out of scope (see the file header); the
// future margin pipeline calls this once per layout and reads via Lookup.
func CollectNamedStrings(root *html.Node) *NamedStringTable {
	table := NewNamedStringTable()

	if root == nil {
		return table
	}

	root.Walk(func(node *html.Node) {
		if node.Type != html.ElementNode {
			return
		}

		styleAttr := node.Attribute("style")
		if styleAttr == "" {
			return
		}

		var latest []StringSetAssignment

		claimed := false

		for _, decl := range css.ParseInline(styleAttr) {
			if decl.Prop != "string-set" {
				continue
			}

			assignments, ok := ParseStringSet(decl.Value)
			if !ok {
				continue
			}

			latest = assignments
			claimed = true
		}

		if !claimed {
			return
		}

		for _, assignment := range latest {
			table.Set(assignment.Name, assignment.Value)
		}
	})

	return table
}

// LookupNamedString is the single-query accessor: collect the document table
// and return the used value stored for name.
func LookupNamedString(root *html.Node, name string) (string, bool) {
	return CollectNamedStrings(root).Lookup(name)
}

// splitStringSetList splits a declaration value on top-level commas.
// Commas inside quoted strings do not split. ok is false only when called
// with an empty value, which the caller already rejects.
func splitStringSetList(value string) []string {
	var items []string

	start := 0

	for idx := 0; idx < len(value); idx++ {
		switch value[idx] {
		case '"', '\'':
			end, ok := scanStringSetQuote(value, idx)
			if !ok {
				items = append(items, value[start:])

				return items
			}

			idx = end
		case '\\':
			idx++
		case ',':
			items = append(items, value[start:idx])
			start = idx + 1
		}
	}

	items = append(items, value[start:])

	return items
}

// parseStringSetItem parses one comma item: an identifier followed by one or
// more quoted strings or content() terms. Quoted strings concatenate into
// Value. content() stays in pieces until the element is known.
func parseStringSetItem(item string) (StringSetAssignment, bool) {
	empty := StringSetAssignment{Name: "", Value: "", pieces: nil}

	name, rest, ok := splitStringSetName(item)
	if !ok {
		return empty, false
	}

	var built strings.Builder

	pieces := make([]stringSetPiece, 0, stringSetInitialPieces)

	for rest != "" {
		if decoded, next, quoted := scanStringSetString(rest, 0); quoted {
			built.WriteString(decoded)
			pieces = append(pieces, stringSetPiece{literal: decoded, keyword: "", fn: false})
			rest = strings.TrimLeft(rest[next:], " \t\n\r\f")

			continue
		}

		keyword, next, isFn := scanStringSetContent(rest)
		if !isFn {
			return empty, false
		}

		pieces = append(pieces, stringSetPiece{literal: "", keyword: keyword, fn: true})
		rest = strings.TrimLeft(rest[next:], " \t\n\r\f")
	}

	if len(pieces) == 0 {
		return empty, false
	}

	return StringSetAssignment{
		Name:   strings.ToLower(name),
		Value:  built.String(),
		pieces: pieces,
	}, true
}

// stringSetInitialPieces is the starting capacity for the pieces of one value.
const stringSetInitialPieces = 2

// scanStringSetContent parses one content() term at the start of rest.
// keyword is empty for content(), which means the same thing as content(text).
func scanStringSetContent(rest string) (string, int, bool) {
	arg, end, found := scanStringSetContentCall(rest)
	if !found {
		return "", 0, false
	}

	if arg == "" {
		return "", end, true
	}

	keyword, valid := stringSetContentKeyword(arg)
	if !valid {
		return "", 0, false
	}

	return keyword, end, true
}

// scanStringSetContentCall reads "content(...)" at the start of rest and
// returns the trimmed argument and the index just past the closing parenthesis.
func scanStringSetContentCall(rest string) (string, int, bool) {
	idx := skipStringSetSpaces(rest, 0)

	wordEnd := idx

	for wordEnd < len(rest) && !isStringSetSpace(rest[wordEnd]) && rest[wordEnd] != '(' {
		wordEnd++
	}

	if wordEnd == idx || !strings.EqualFold(rest[idx:wordEnd], "content") {
		return "", 0, false
	}

	paren := skipStringSetSpaces(rest, wordEnd)
	if paren >= len(rest) || rest[paren] != '(' {
		return "", 0, false
	}

	end, ok := contentCallEnd(rest, paren)
	if !ok {
		return "", 0, false
	}

	return strings.TrimSpace(rest[paren+1 : end-1]), end, true
}

// contentCallEnd returns the index just past the ")" that closes the call
// whose "(" is at paren.
func contentCallEnd(rest string, paren int) (int, bool) {
	end := skipCSSFunction(rest, paren+1)
	if end <= paren+1 || end > len(rest) || rest[end-1] != ')' {
		return 0, false
	}

	return end, true
}

// stringSetContentKeyword returns the lowercase keyword of a content() argument
// that names a supported source.
func stringSetContentKeyword(arg string) (string, bool) {
	if strings.ContainsAny(arg, " \t\n\r\f,()") {
		return "", false
	}

	switch strings.ToLower(arg) {
	case "text", "before", "after", "first-letter", "marker":
		return strings.ToLower(arg), true
	default:
		return "", false
	}
}

// skipStringSetSpaces returns the index of the first non-space byte at or after i.
func skipStringSetSpaces(rest string, i int) int {
	for i < len(rest) && isStringSetSpace(rest[i]) {
		i++
	}

	return i
}

// splitStringSetName cuts one comma item into its identifier and the
// remaining string list. ok is false for an empty item, a non-identifier, or
// a reserved keyword.
func splitStringSetName(item string) (string, string, bool) {
	rest := strings.TrimLeft(item, " \t\n\r\f")
	if rest == "" {
		return "", "", false
	}

	identEnd := 0

	for identEnd < len(rest) && !isStringSetSpace(rest[identEnd]) {
		identEnd++
	}

	name := rest[:identEnd]

	if !css.IsIdentToken(name) || isStringSetReserved(name) {
		return "", "", false
	}

	return name, rest[identEnd:], true
}

// isStringSetReserved reports CSS-wide keywords and none, which cannot be
// custom identifiers.
func isStringSetReserved(name string) bool {
	switch strings.ToLower(name) {
	case "none", cssKeywordInitial, inheritKeyword, cssKeywordUnset, "default", cssKeywordRevert, stringSetRevertLayer:
		return true
	default:
		return false
	}
}

// scanStringSetString scans one quoted string at the start of rest (after
// leading whitespace) and returns its decoded used value plus the offset just
// past the closing quote.
func scanStringSetString(rest string, start int) (string, int, bool) {
	idx := start

	for idx < len(rest) && isStringSetSpace(rest[idx]) {
		idx++
	}

	if idx >= len(rest) || (rest[idx] != '"' && rest[idx] != '\'') {
		return "", start, false
	}

	end, ok := scanStringSetQuote(rest, idx)
	if !ok {
		return "", start, false
	}

	return decodeStringSetString(rest[idx+1 : end]), end + 1, true
}

// scanStringSetQuote returns the index of the closing quote matching the
// quote at open. Backslash escapes skip the next byte.
func scanStringSetQuote(value string, open int) (int, bool) {
	quote := value[open]

	for idx := open + 1; idx < len(value); idx++ {
		if value[idx] == '\\' {
			idx++

			if idx >= len(value) {
				return 0, false
			}

			continue
		}

		if value[idx] == quote {
			return idx, true
		}
	}

	return 0, false
}

// decodeStringSetString resolves backslash escapes the way quoted page-box
// content does (internal/css/page_margin.go): a backslash quotes the next
// byte, so \' inside a single-quoted string is a literal quote.
func decodeStringSetString(src string) string {
	if !strings.ContainsRune(src, '\\') {
		return src
	}

	var built strings.Builder

	built.Grow(len(src))

	for idx := 0; idx < len(src); idx++ {
		if src[idx] != '\\' || idx+1 >= len(src) {
			built.WriteByte(src[idx])

			continue
		}

		idx++

		built.WriteByte(src[idx])
	}

	return built.String()
}

// isStringSetSpace reports CSS whitespace between string-set tokens.
func isStringSetSpace(char byte) bool {
	switch char {
	case ' ', '\t', '\n', '\r', '\f':
		return true
	default:
		return false
	}
}

// encodeStringSetAssignment re-encodes one assignment. Adjacent quoted
// strings become one quoted literal, which is the stored form the cascade
// already uses for string-only values. content() stays a function so the
// collector can read the element later.
func encodeStringSetAssignment(assignment StringSetAssignment) string {
	if len(assignment.pieces) == 0 {
		return assignment.Name + " " + quoteStringSetValue(assignment.Value)
	}

	var built strings.Builder

	built.WriteString(assignment.Name)

	var literal strings.Builder

	flush := func() {
		if literal.Len() == 0 {
			return
		}

		built.WriteByte(' ')
		built.WriteString(quoteStringSetValue(literal.String()))
		literal.Reset()
	}

	for _, piece := range assignment.pieces {
		if !piece.fn {
			literal.WriteString(piece.literal)

			continue
		}

		flush()
		built.WriteString(" content(")
		built.WriteString(piece.keyword)
		built.WriteByte(')')
	}

	flush()

	return built.String()
}

// forgetInheritedStringSet drops __string_set that mergeCustomProps copied
// from the parent. string-set does not inherit. The map may be the parent's,
// so this replaces the pointer instead of deleting in place.
func forgetInheritedStringSet(style *ResolvedStyle) {
	if style == nil || style.CustomProps == nil {
		return
	}

	if _, ok := style.CustomProps[stringSetCustomKey]; !ok {
		return
	}

	if len(style.CustomProps) == 1 {
		style.CustomProps = nil

		return
	}

	cloned := make(map[string]string, len(style.CustomProps)-1)

	for key, value := range style.CustomProps {
		if key == stringSetCustomKey {
			continue
		}

		cloned[key] = value
	}

	style.CustomProps = cloned
}

// documentStringTable is the named-string table visible to a string() read
// on node. Assignments after node are ignored, so a later string-set: none
// does not erase a reader that already passed, and a reader after none does
// not paint the cleared value.
func (e *engine) documentStringTable(node *html.Node) *NamedStringTable {
	root := documentRoot(node)
	if e == nil || root == nil {
		return NewNamedStringTable()
	}

	return e.collectCascadedNamedStrings(root, node)
}

// collectCascadedNamedStrings walks from root in document order and stops
// before stop. It applies each element's own __string_set. The first
// assignment of a name wins. none clears every name. An unparsable stored
// value is skipped.
func (e *engine) collectCascadedNamedStrings(root, stop *html.Node) *NamedStringTable {
	table := NewNamedStringTable()
	if e == nil || root == nil {
		return table
	}

	root.WalkUntil(func(node *html.Node) bool {
		if node == stop {
			return false
		}

		e.applyCascadedStringSet(table, node)

		return true
	})

	return table
}

// applyCascadedStringSet records one element's cascaded string-set, if it
// declared one. Inherited values are already stripped.
func (e *engine) applyCascadedStringSet(table *NamedStringTable, node *html.Node) {
	if e == nil || table == nil || node == nil || node.Type != html.ElementNode {
		return
	}

	raw, ok := e.cascadedStringSetRaw(node)
	if !ok {
		return
	}

	if raw == "none" {
		table.Clear()

		return
	}

	assignments, parsed := ParseStringSet(raw)
	if !parsed {
		return
	}

	for _, assignment := range assignments {
		table.Set(assignment.Name, e.resolveStringSetAssignment(node, assignment))
	}
}

// cascadedStringSetRaw returns the string-set text the cascade stored on node.
// ok is false when node has none or it is empty.
func (e *engine) cascadedStringSetRaw(node *html.Node) (string, bool) {
	sty := e.styles[node]
	if sty == nil || sty.CustomProps == nil {
		return "", false
	}

	raw, ok := sty.CustomProps[stringSetCustomKey]
	if !ok || raw == "" {
		return "", false
	}

	return raw, true
}

// resolveStringSetAssignment turns one stored assignment into the text
// string() will paint. Quoted pieces are already decoded. content() reads
// the element.
func (e *engine) resolveStringSetAssignment(node *html.Node, assignment StringSetAssignment) string {
	if len(assignment.pieces) == 0 {
		return assignment.Value
	}

	var built strings.Builder

	for _, piece := range assignment.pieces {
		if !piece.fn {
			built.WriteString(piece.literal)

			continue
		}

		built.WriteString(e.stringSetKeyword(node, piece.keyword))
	}

	return built.String()
}

// stringSetKeyword resolves one content() argument. An empty keyword is
// content(), which reads the same text as content(text). content(before)
// with no ::before contributes nothing.
func (e *engine) stringSetKeyword(node *html.Node, keyword string) string {
	switch keyword {
	case "", "text":
		return bookmarkElementText(node)
	case "before":
		return e.stringSetPseudoText(node, pseudoBefore)
	case "after":
		return e.stringSetPseudoText(node, pseudoAfter)
	case "first-letter":
		text := bookmarkElementText(node)
		if text == "" {
			return ""
		}

		return string([]rune(text)[:1])
	default:
		return ""
	}
}

// stringSetPseudoText reads ::before or ::after for content(before) and
// content(after). No matching rule contributes nothing. A pseudo that itself
// calls string() contributes nothing, so building the table cannot recurse
// through the same reader.
func (e *engine) stringSetPseudoText(node *html.Node, pseudo string) string {
	if e == nil || node == nil || !e.mayHavePseudoContent(node, pseudo) {
		return ""
	}

	best := e.cachedContentDecl(e.pseudoStyleContext(), node, pseudo)
	if best == nil {
		return ""
	}

	if strings.Contains(strings.ToLower(best.value), "string(") {
		return ""
	}

	if contentNeedsEnv(best.value) {
		return evalContentValue(best.value, node, e.contentEnvAt(node, pseudo), nil)
	}

	return evalContentValue(best.value, node, nil, nil)
}
