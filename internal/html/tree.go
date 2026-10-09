// recovery conventions as html.go
//
//nolint:all // insertion-mode tree builder; same custom Node model and
package html

import "strings"

// maxElementDepth caps element nesting. Elements that would nest deeper are
// dropped by insertNode, so recursive walks stay bounded on adversarial input
// instead of exhausting the stack.
const maxElementDepth = 1024

// maxReprocess bounds mode-transition reprocessing so adversarial input cannot
// loop between insertion modes.
const maxReprocess = 8

// insertionMode is the standard's tree-construction mode.
type insertionMode uint8

const (
	modeInitial insertionMode = iota
	modeBeforeHTML
	modeBeforeHead
	modeInHead
	modeAfterHead
	modeInBody
	modeInText
	modeInTable
	modeInCaption
	modeInColumnGroup
	modeInTableBody
	modeInRow
	modeInCell
	modeAfterBody
	modeAfterAfterBody
)

// treeBuilder accumulates parsed tokens into a node tree.
type treeBuilder struct {
	root             *Node
	stack            []*Node
	mode             insertionMode
	textReturn       insertionMode // mode to restore after a raw-text element
	head             *Node         // head element pointer
	form             *Node         // form element pointer
	docMode          DocumentMode
	docModeSet       bool
	ignoreNextLF     bool // swallow the LF after <pre>, <listing>, <textarea>
	fosterParenting  bool // route insertions around the open table
	pendingTableText []string
	activeFormatting []*Node // nil entries are scope markers
}

func newTreeBuilder() *treeBuilder {
	root := &Node{Type: ElementNode, Name: "#document"} //nolint:exhaustruct

	return &treeBuilder{
		root:  root,
		stack: []*Node{root},
	}
}

// insertMarker pushes a scope marker onto the active formatting list.
func (b *treeBuilder) insertMarker() {
	b.activeFormatting = append(b.activeFormatting, nil)
}

// clearActiveFormattingToMarker removes active formatting entries up to and
// including the last scope marker, per the standard's clear-to-marker rule.
func (b *treeBuilder) clearActiveFormattingToMarker() {
	for idx := len(b.activeFormatting) - 1; idx >= 0; idx-- {
		if b.activeFormatting[idx] == nil {
			b.activeFormatting = b.activeFormatting[:idx]

			return
		}
	}

	b.activeFormatting = nil
}

func (b *treeBuilder) top() *Node {
	return b.stack[len(b.stack)-1]
}

// appendToken applies one scanned token, reprocessing it through insertion
// modes as the standard requires.
func (b *treeBuilder) appendToken(tokItem token) {
	if tokItem.kind != tokText {
		b.flushPendingTableText()
	}

	for range maxReprocess {
		if !b.processToken(&tokItem) {
			return
		}
	}
}

// finish runs the end-of-file mode chain: every mode before "in body" creates
// the document structure the standard inserts at EOF.
func (b *treeBuilder) finish() {
	b.flushPendingTableText()

	for {
		switch b.mode {
		case modeInitial:
			if !b.docModeSet {
				b.docMode = Quirks
				b.docModeSet = true
			}

			b.mode = modeBeforeHTML
		case modeBeforeHTML:
			b.insertHTMLElement("html", nil)
			b.mode = modeBeforeHead
		case modeBeforeHead:
			b.head = b.insertHTMLElement("head", nil)
			b.mode = modeInHead
		case modeInHead:
			b.popCurrentIfName("head")
			b.mode = modeAfterHead
		case modeAfterHead:
			b.insertHTMLElement("body", nil)
			b.mode = modeInBody
		case modeInText:
			// EOF in a raw-text element pops it and reprocesses under the
			// original insertion mode, which then builds head/body as needed.
			if len(b.stack) > 1 {
				b.stack = b.stack[:len(b.stack)-1]
			}

			b.mode = b.textReturn
		default:
			b.root.Mode = b.docMode

			return
		}
	}
}

// processToken applies the current insertion mode. It reports whether the
// token must be reprocessed after a mode switch.
func (b *treeBuilder) processToken(tokItem *token) bool {
	if b.foreignToken(tokItem) {
		return false
	}

	switch b.mode {
	case modeInitial:
		return b.processInitial(tokItem)
	case modeBeforeHTML:
		return b.processBeforeHTML(tokItem)
	case modeBeforeHead:
		return b.processBeforeHead(tokItem)
	case modeInHead:
		return b.processInHead(tokItem)
	case modeAfterHead:
		return b.processAfterHead(tokItem)
	case modeInBody:
		return b.processInBody(tokItem)
	case modeInText:
		return b.processInText(tokItem)
	case modeInTable:
		return b.processInTable(tokItem)
	case modeInCaption:
		return b.processInCaption(tokItem)
	case modeInColumnGroup:
		return b.processInColumnGroup(tokItem)
	case modeInTableBody:
		return b.processInTableBody(tokItem)
	case modeInRow:
		return b.processInRow(tokItem)
	case modeInCell:
		return b.processInCell(tokItem)
	case modeAfterBody:
		return b.processAfterBody(tokItem)
	case modeAfterAfterBody:
		return b.processAfterAfterBody(tokItem)
	default:
		return false
	}
}

// --- insertion modes ---

func (b *treeBuilder) processInitial(tokItem *token) bool {
	switch tokItem.kind {
	case tokComment:
		b.appendCommentTo(b.root, tokItem.data)

		return false
	case tokDoctype:
		if !b.docModeSet {
			b.docMode = classifyDocumentMode(tokItem.doctype)
			b.docModeSet = true
			b.root.Children = append(b.root.Children, &Node{Type: DoctypeNode, Text: tokItem.data, Doctype: tokItem.doctype}) //nolint:exhaustruct
		}

		b.mode = modeBeforeHTML

		return false
	case tokText:
		_, rest := splitLeadingWhitespace(tokItem.data)
		if rest == "" {
			return false
		}

		tokItem.data = rest
	case tokStart, tokEnd:
	}

	if !b.docModeSet {
		b.docMode = Quirks
		b.docModeSet = true
	}

	b.mode = modeBeforeHTML

	return true
}

func (b *treeBuilder) processBeforeHTML(tokItem *token) bool {
	switch tokItem.kind {
	case tokDoctype:
		return false
	case tokComment:
		b.appendCommentTo(b.root, tokItem.data)

		return false
	case tokText:
		_, rest := splitLeadingWhitespace(tokItem.data)
		if rest == "" {
			return false
		}

		tokItem.data = rest
	case tokStart:
		if tokItem.data == "html" {
			b.insertHTMLElement("html", tokItem.attrs)
			b.mode = modeBeforeHead

			return false
		}
	case tokEnd:
		switch tokItem.data {
		case "head", "body", "html", "br":
		default:
			return false
		}
	}

	b.insertHTMLElement("html", nil)
	b.mode = modeBeforeHead

	return true
}

func (b *treeBuilder) processBeforeHead(tokItem *token) bool {
	switch tokItem.kind {
	case tokDoctype:
		return false
	case tokComment:
		b.appendCommentTo(b.top(), tokItem.data)

		return false
	case tokText:
		_, rest := splitLeadingWhitespace(tokItem.data)
		if rest == "" {
			return false
		}

		tokItem.data = rest
	case tokStart:
		switch tokItem.data {
		case "html":
			b.mergeIntoHTMLElement(tokItem.attrs)

			return false
		case "head":
			b.head = b.insertHTMLElement("head", tokItem.attrs)
			b.mode = modeInHead

			return false
		}
	case tokEnd:
		switch tokItem.data {
		case "head", "body", "html", "br":
		default:
			return false
		}
	}

	b.head = b.insertHTMLElement("head", nil)
	b.mode = modeInHead

	return true
}

func (b *treeBuilder) processInHead(tokItem *token) bool {
	switch tokItem.kind {
	case tokDoctype:
		return false
	case tokComment:
		b.appendCommentTo(b.top(), tokItem.data)

		return false
	case tokText:
		prefix, rest := splitLeadingWhitespace(tokItem.data)
		if rest == "" {
			b.appendTextToken(tokItem.data)

			return false
		}

		if prefix != "" {
			b.appendTextToken(prefix)
		}

		tokItem.data = rest
		b.popCurrentIfName("head")
		b.mode = modeAfterHead

		return true
	case tokStart:
		switch tokItem.data {
		case "html":
			b.mergeIntoHTMLElement(tokItem.attrs)

			return false
		case "base", "basefont", "bgsound", "link", "meta":
			b.insertHTMLElement(tokItem.data, tokItem.attrs)

			return false
		case "title", "style", "noframes", "script", "template", "noscript":
			b.insertHTMLElement(tokItem.data, tokItem.attrs)
			if textModeElement(tokItem.data) {
				b.textReturn = b.mode
				b.mode = modeInText
			}

			return false
		}
	case tokEnd:
		if tokItem.data == "head" {
			b.popCurrentIfName("head")
			b.mode = modeAfterHead

			return false
		}

		if tokItem.data == "template" {
			// The in-head template end tag pops through the open template.
			// The template contents model and its marker stay out of scope.
			if openInStack(b.stack, "template") {
				b.popUntilName("template")
			}

			return false
		}

		if tokItem.data != "body" && tokItem.data != "html" && tokItem.data != "br" {
			b.popCurrentIfName(tokItem.data)

			return false
		}
	}

	b.popCurrentIfName("head")
	b.mode = modeAfterHead

	return true
}

func (b *treeBuilder) processAfterHead(tokItem *token) bool {
	switch tokItem.kind {
	case tokDoctype:
		return false
	case tokComment:
		b.appendCommentTo(b.top(), tokItem.data)

		return false
	case tokText:
		prefix, rest := splitLeadingWhitespace(tokItem.data)
		if rest == "" {
			b.appendTextToken(tokItem.data)

			return false
		}

		if prefix != "" {
			b.appendTextToken(prefix)
		}

		tokItem.data = rest
		b.insertHTMLElement("body", nil)
		b.mode = modeInBody

		return true
	case tokStart:
		switch tokItem.data {
		case "html":
			b.mergeIntoHTMLElement(tokItem.attrs)

			return false
		case "body":
			b.insertHTMLElement("body", tokItem.attrs)
			b.mode = modeInBody

			return false
		case "base", "basefont", "bgsound", "link", "meta", "noframes", "script", "style", "template", "title":
			b.processHeadContent(tokItem)

			return false
		}
	case tokEnd:
		switch tokItem.data {
		case "body", "html", "br":
		default:
			return false
		}
	}

	b.insertHTMLElement("body", nil)
	b.mode = modeInBody

	return true
}

func (b *treeBuilder) processInText(tokItem *token) bool {
	switch tokItem.kind {
	case tokText:
		b.appendTextToken(tokItem.data)
	case tokEnd:
		if len(b.stack) > 1 {
			b.stack = b.stack[:len(b.stack)-1]
		}

		b.mode = b.textReturn
	case tokStart, tokComment, tokDoctype:
		// Raw text is consumed by the tokenizer; these cannot occur here.
	}

	return false
}

func (b *treeBuilder) processInBody(tokItem *token) bool {
	switch tokItem.kind {
	case tokComment:
		b.appendCommentTo(b.top(), tokItem.data)

		return false
	case tokDoctype:
		return false
	case tokText:
		data := tokItem.data
		if b.ignoreNextLF {
			b.ignoreNextLF = false
			data = strings.TrimPrefix(data, "\n")
		}

		if data != "" {
			b.reconstructActiveFormatting()
			b.appendTextToken(data)
		}

		return false
	case tokStart:
		return b.processInBodyStart(tokItem)
	case tokEnd:
		return b.processInBodyEnd(tokItem)
	default:
		return false
	}
}

func (b *treeBuilder) processInBodyStart(tokItem *token) bool {
	name := tokItem.data

	switch name {
	case "html":
		b.mergeIntoHTMLElement(tokItem.attrs)

		return false
	case "base", "basefont", "bgsound", "link", "meta", "noframes", "script", "style", "template", "title":
		b.processInHead(tokItem)

		return false
	case "body":
		if body := b.findInScope("body", defaultScopeStops); body != nil {
			applyAttributes(body, tokItem.attrs)
		}

		return false
	case "address", "article", "aside", "blockquote", "center", "details", "dialog",
		"dir", "div", "dl", "fieldset", "figcaption", "figure", "footer", "header",
		"hgroup", "main", "menu", "nav", "ol", "p", "search", "section", "summary", "ul":
		b.closePElementIfOpen()
		b.insertHTMLElement(name, tokItem.attrs)

		return false
	case "h1", "h2", "h3", "h4", "h5", "h6":
		b.closePElementIfOpen()

		if isHeadingElement(b.top()) {
			b.stack = b.stack[:len(b.stack)-1]
		}

		b.insertHTMLElement(name, tokItem.attrs)

		return false
	case "pre", "listing":
		b.closePElementIfOpen()
		b.insertHTMLElement(name, tokItem.attrs)

		b.ignoreNextLF = true

		return false
	case "form":
		if b.form != nil {
			return false
		}

		b.closePElementIfOpen()
		b.form = b.insertHTMLElement(name, tokItem.attrs)

		return false
	case "li":
		b.closeOpenListItem()
		b.closePElementIfOpen()
		b.insertHTMLElement(name, tokItem.attrs)

		return false
	case "dd", "dt":
		b.closeOpenDefinitionItem()
		b.closePElementIfOpen()
		b.insertHTMLElement(name, tokItem.attrs)

		return false
	case "plaintext":
		b.closePElementIfOpen()
		b.insertHTMLElement(name, tokItem.attrs)

		return false
	case "button":
		if b.hasInScope("button", defaultScopeStops) {
			b.generateImpliedEndTags("")
			b.popUntilName("button")
		}

		b.reconstructActiveFormatting()
		b.insertHTMLElement(name, tokItem.attrs)

		return false
	case "a":
		if existing := b.lastActiveFormatting("a"); existing != nil {
			b.adoptionAgency(tokItem)

			if b.activeFormattingContains(existing) {
				b.activeFormattingRemove(existing)
			}

			b.removeFromStack(existing)
		}

		b.reconstructActiveFormatting()
		b.pushActiveFormatting(b.insertHTMLElement(name, tokItem.attrs))

		return false
	case "b", "big", "code", "em", "font", "i", "s", "small", "strike", "strong", "tt", "u":
		b.reconstructActiveFormatting()
		b.pushActiveFormatting(b.insertHTMLElement(name, tokItem.attrs))

		return false
	case "nobr":
		b.reconstructActiveFormatting()

		if b.hasInScope("nobr", defaultScopeStops) {
			b.adoptionAgency(tokItem)
			b.reconstructActiveFormatting()
		}

		b.pushActiveFormatting(b.insertHTMLElement(name, tokItem.attrs))

		return false
	case "applet", "marquee", "object":
		b.reconstructActiveFormatting()
		b.insertHTMLElement(name, tokItem.attrs)
		b.insertMarker()

		return false
	case "area", "br", "embed", "img", "keygen", "wbr", "input":
		b.reconstructActiveFormatting()
		b.insertHTMLElement(name, tokItem.attrs)

		return false
	case "param", "source", "track":
		b.insertHTMLElement(name, tokItem.attrs)

		return false
	case "hr":
		b.closePElementIfOpen()
		b.insertHTMLElement(name, tokItem.attrs)

		return false
	case "image":
		tokItem.data = "img"

		return true
	case "textarea":
		b.insertHTMLElement(name, tokItem.attrs)

		b.ignoreNextLF = true
		b.textReturn = b.mode
		b.mode = modeInText

		return false
	case "xmp":
		b.closePElementIfOpen()
		b.reconstructActiveFormatting()
		b.insertHTMLElement(name, tokItem.attrs)

		b.textReturn = b.mode
		b.mode = modeInText

		return false
	case "iframe", "noembed":
		b.insertHTMLElement(name, tokItem.attrs)

		b.textReturn = b.mode
		b.mode = modeInText

		return false
	case "select", "option", "optgroup":
		b.legacyOptionAutoClose(name)
		b.insertHTMLElement(name, tokItem.attrs)

		return false
	case "math":
		b.insertForeignElement(name, NamespaceMathML, tokItem)

		return false
	case "svg":
		b.insertForeignElement(name, NamespaceSVG, tokItem)

		return false
	case "table":
		if b.docMode != Quirks {
			b.closePElementIfOpen()
		}

		b.insertHTMLElement(name, tokItem.attrs)
		b.mode = modeInTable

		return false
	case "caption", "col", "colgroup", "tbody", "td", "tfoot", "th", "thead", "tr":
		// A table-structure tag outside its table is a parse error; the
		// table insertion modes own these tags.
		return false
	default:
		if !defaultNoReconstruct[name] {
			b.reconstructActiveFormatting()
		}

		b.insertHTMLElement(name, tokItem.attrs)

		return false
	}
}

func (b *treeBuilder) processInBodyEnd(tokItem *token) bool {
	name := tokItem.data

	if isFormattingTag(name) {
		b.adoptionAgency(tokItem)

		return false
	}

	switch name {
	case "body":
		if !b.hasInScope("body", defaultScopeStops) {
			return false
		}

		b.mode = modeAfterBody

		return false
	case "html":
		if !b.hasInScope("body", defaultScopeStops) {
			return false
		}

		b.mode = modeAfterBody

		return false
	case "p":
		if !b.hasInButtonScope("p") {
			b.insertHTMLElement("p", nil)
		}

		b.closePElement()

		return false
	case "li":
		if !b.hasInListScope("li") {
			return false
		}

		b.generateImpliedEndTags("li")
		b.popUntilName("li")

		return false
	case "dd", "dt":
		if !b.hasInScope(name, defaultScopeStops) {
			return false
		}

		b.generateImpliedEndTags(name)
		b.popUntilName(name)

		return false
	case "h1", "h2", "h3", "h4", "h5", "h6":
		if !b.hasInScopeAny(headingNames, defaultScopeStops) {
			return false
		}

		b.generateImpliedEndTags("")
		b.popUntilAnyName(headingNames...)

		return false
	case "br":
		b.reconstructActiveFormatting()
		b.insertHTMLElement("br", nil)

		return false
	case "applet", "marquee", "object":
		if !b.hasInScope(name, defaultScopeStops) {
			return false
		}

		b.generateImpliedEndTags("")
		b.popUntilName(name)
		b.clearActiveFormattingToMarker()

		return false
	case "form":
		node := b.form
		if node == nil {
			node = b.findInScope("form", defaultScopeStops)
		}

		b.form = nil

		if node == nil {
			return false
		}

		b.generateImpliedEndTags("")
		b.removeFromStack(node)

		return false
	case "template":
		b.processInHead(tokItem)

		return false
	}

	if bodyEndTagsWithScope[name] {
		if !b.hasInScope(name, defaultScopeStops) {
			return false
		}

		b.generateImpliedEndTags("")
		b.popUntilName(name)

		return false
	}

	b.closeHTMLElement(name)

	return false
}

func (b *treeBuilder) processAfterBody(tokItem *token) bool {
	switch tokItem.kind {
	case tokComment:
		b.appendCommentTo(b.firstStackElement(), tokItem.data)

		return false
	case tokDoctype:
		return false
	case tokText:
		if isAllWhitespace(tokItem.data) {
			return b.processInBody(tokItem)
		}
	case tokStart:
		if tokItem.data == "html" {
			return b.processInBody(tokItem)
		}
	case tokEnd:
		if tokItem.data == "html" {
			b.mode = modeAfterAfterBody

			return false
		}
	}

	b.mode = modeInBody

	return true
}

func (b *treeBuilder) processAfterAfterBody(tokItem *token) bool {
	switch tokItem.kind {
	case tokComment:
		b.appendCommentTo(b.root, tokItem.data)

		return false
	case tokDoctype:
		return false
	case tokText:
		if isAllWhitespace(tokItem.data) {
			return b.processInBody(tokItem)
		}
	case tokStart:
		if tokItem.data == "html" {
			return b.processInBody(tokItem)
		}
	}

	b.mode = modeInBody

	return true
}

// --- foreign content dispatch ---

// foreignToken applies the foreign-content rules when the current node is in
// a foreign namespace. It reports whether the token was consumed. A breakout
// token is popped back to HTML or integration-point content and reprocessed
// under the current HTML mode.
func (b *treeBuilder) foreignToken(tokItem *token) bool {
	switch tokItem.kind {
	case tokText:
		if !b.currentIsForeignText() {
			return false
		}

		b.appendTextToken(tokItem.data)

		return true
	case tokDoctype:
		return b.currentIsForeign()
	case tokStart:
		if !b.inForeignStartContext(tokItem) {
			return false
		}

		if isForeignBreakout(tokItem) {
			b.popForeignBreakout()

			return false
		}

		b.insertForeignElement(tokItem.data, b.top().Namespace, tokItem)

		return true
	case tokEnd:
		if !b.currentIsForeign() {
			return false
		}

		if tokItem.data == "br" || tokItem.data == "p" {
			b.popForeignBreakout()

			return false
		}

		return b.closeForeignElement(tokItem.data)
	case tokComment:
		return false
	}

	return false
}

func (b *treeBuilder) inForeignStartContext(tokItem *token) bool {
	top := b.top()
	if top.Namespace == NamespaceHTML {
		return false
	}

	if isMathMLTextIntegrationPoint(top) && tokItem.data != "mglyph" && tokItem.data != "malignmark" {
		return false
	}

	if top.Namespace == NamespaceMathML && top.Name == "annotation-xml" && tokItem.data == "svg" {
		return false
	}

	return !isHTMLIntegrationPoint(top)
}

func (b *treeBuilder) currentIsForeign() bool {
	return b.top().Namespace != NamespaceHTML
}

func (b *treeBuilder) currentIsForeignText() bool {
	top := b.top()
	if top.Namespace == NamespaceHTML {
		return false
	}

	return !isMathMLTextIntegrationPoint(top) && !isHTMLIntegrationPoint(top)
}

func (b *treeBuilder) popForeignBreakout() {
	for len(b.stack) > 1 {
		top := b.top()
		if top.Namespace == NamespaceHTML || isMathMLTextIntegrationPoint(top) || isHTMLIntegrationPoint(top) {
			return
		}

		b.stack = b.stack[:len(b.stack)-1]
	}
}

// --- insertion helpers ---

// insertHTMLElement inserts an HTML element at the current position.
func (b *treeBuilder) insertHTMLElement(name string, attrs []string) *Node {
	return b.insertNode(name, NamespaceHTML, attrs, false)
}

// insertionPoint is where the next node goes: a parent plus an optional
// reference child to insert before (nil appends at the end).
type insertionPoint struct {
	parent *Node
	before *Node
}

// appropriatePlace implements the standard's "appropriate place for inserting
// a node" for the current node: the top of the open-element stack, or, with
// foster parenting enabled over a table section, the position just before the
// open table.
func (b *treeBuilder) appropriatePlace() insertionPoint {
	return b.appropriatePlaceFor(b.top())
}

// appropriatePlaceFor is appropriatePlace relative to target, for the
// adoption agency's adjusted insertion location.
func (b *treeBuilder) appropriatePlaceFor(target *Node) insertionPoint {
	if !b.fosterParenting || !isFosterTarget(target) {
		return insertionPoint{parent: target}
	}

	for i := len(b.stack) - 1; i >= 0; i-- {
		table := b.stack[i]
		if table.Namespace == NamespaceHTML && table.Name == "table" {
			if table.Parent != nil {
				return insertionPoint{parent: table.Parent, before: table}
			}

			if i > 0 {
				return insertionPoint{parent: b.stack[i-1]}
			}

			break
		}
	}

	return insertionPoint{parent: b.stack[0]}
}

// isFosterTarget reports whether inserting into n must be rerouted around the
// table: table, tbody, tfoot, thead, and tr are the foster-parenting targets.
func isFosterTarget(n *Node) bool {
	if n.Namespace != NamespaceHTML {
		return false
	}

	switch n.Name {
	case "table", "tbody", "tfoot", "thead", "tr":
		return true
	default:
		return false
	}
}

// insertNode appends an element node at the appropriate place and pushes it
// unless it is void or, in foreign content, self-closing. Elements deeper than
// the depth cap are dropped so recursive walks stay bounded.
func (b *treeBuilder) insertNode(name string, ns Namespace, attrs []string, selfClosing bool) *Node {
	if len(b.stack)-1 >= maxElementDepth {
		return nil // deeper than the cap: drop the element, content flattens up
	}

	point := b.appropriatePlace()

	node := &Node{Type: ElementNode, Name: name, Namespace: ns} //nolint:exhaustruct
	applyAttributes(node, attrs)

	insertChildAt(point, node)
	node.Parent = point.parent

	// The self-closing flag is a parse error and is ignored on ordinary HTML
	// elements; void HTML elements never take content, and foreign elements
	// honor the flag.
	if (ns == NamespaceHTML && isVoidElement(name)) || (ns != NamespaceHTML && selfClosing) {
		return node // no child content
	}

	b.stack = append(b.stack, node)

	return node
}

// insertChildAt places node at point, either before the reference child or at
// the end of the parent's children.
func insertChildAt(point insertionPoint, node *Node) {
	if point.before != nil {
		for i, child := range point.parent.Children {
			if child == point.before {
				point.parent.Children = append(point.parent.Children, nil)
				copy(point.parent.Children[i+1:], point.parent.Children[i:])
				point.parent.Children[i] = node

				return
			}
		}
	}

	point.parent.Children = append(point.parent.Children, node)
}

// insertForeignElement inserts an element in ns, applying the namespace's
// element-name adjustment (the standard's SVG tag-name table).
func (b *treeBuilder) insertForeignElement(name string, ns Namespace, tokItem *token) {
	b.insertNode(adjustForeignElementName(ns, name), ns, tokItem.attrs, tokItem.selfClosing)
}

// appendCommentTo attaches a comment node to parent.
func (b *treeBuilder) appendCommentTo(parent *Node, data string) {
	parent.Children = append(parent.Children, &Node{Type: CommentNode, Text: data}) //nolint:exhaustruct
}

// appendTextToken attaches text at the appropriate place, merging into an
// adjacent text node when present. Token data is already decoded by the
// tokenizer; U+0000 characters are dropped in HTML content and replaced with
// U+FFFD in foreign content.
func (b *treeBuilder) appendTextToken(data string) {
	if b.currentIsForeignText() {
		data = replaceNUL(data)
	} else {
		data = strings.ReplaceAll(data, "\x00", "")
	}

	if data == "" {
		return
	}

	point := b.appropriatePlace()

	if point.before != nil {
		for i, child := range point.parent.Children {
			if child != point.before {
				continue
			}

			if i > 0 {
				if prev := point.parent.Children[i-1]; prev.Type == TextNode {
					prev.Text += data

					return
				}
			}

			node := &Node{Type: TextNode, Text: data, Parent: point.parent} //nolint:exhaustruct
			insertChildAt(point, node)

			return
		}
	}

	top := point.parent

	if len(top.Children) > 0 {
		last := top.Children[len(top.Children)-1]
		if last.Type == TextNode {
			var merged strings.Builder
			merged.Grow(len(last.Text) + len(data))
			merged.WriteString(last.Text)
			merged.WriteString(data)
			last.Text = merged.String()

			return
		}
	}

	node := &Node{Type: TextNode, Text: data} //nolint:exhaustruct
	node.Parent = top
	top.Children = append(top.Children, node)
}

// applyAttributes stores the interleaved name/value pairs on node, keeping
// the first value of a duplicated attribute. Names are already lowercased and
// values already character-reference decoded by the tokenizer; foreign
// elements additionally adjust names per the standard's tables.
func applyAttributes(node *Node, attrs []string) {
	if len(attrs) == 0 {
		return
	}

	if node.Attrs == nil {
		const attrPairSize = 2 // attrs slice interleaves name and value

		node.Attrs = make(map[string]string, len(attrs)/attrPairSize)
	}

	for i := 0; i+1 < len(attrs); i += 2 {
		key, attr := adjustAttributeName(node.Namespace, attrs[i], attrs[i+1])
		if _, dup := node.Attrs[key]; dup {
			continue
		}

		node.Attrs[key] = attr.Value
		node.AttrList = append(node.AttrList, attr)
	}
}

// processHeadContent handles a head-content start tag seen after the head
// element was closed: the head is pushed back on the stack, the in-head rules
// run, and the head is removed again.
func (b *treeBuilder) processHeadContent(tokItem *token) {
	pushed := false

	if b.head != nil && !openInStack(b.stack, "head") {
		b.stack = append(b.stack, b.head)
		pushed = true
	}

	b.processInHead(tokItem)

	if pushed {
		b.removeFromStack(b.head)
	}
}

// mergeIntoHTMLElement merges attributes into the existing html element.
func (b *treeBuilder) mergeIntoHTMLElement(attrs []string) {
	if len(b.stack) < 2 {
		return
	}

	html := b.stack[1]
	if html.Namespace == NamespaceHTML && html.Name == "html" {
		applyAttributes(html, attrs)
	}
}

func (b *treeBuilder) removeFromStack(node *Node) {
	for i := len(b.stack) - 1; i > 0; i-- {
		if b.stack[i] == node {
			b.stack = append(b.stack[:i], b.stack[i+1:]...)

			return
		}
	}
}

func (b *treeBuilder) popCurrentIfName(name string) {
	if b.top().Namespace == NamespaceHTML && b.top().Name == name {
		b.stack = b.stack[:len(b.stack)-1]
	}
}

func (b *treeBuilder) firstStackElement() *Node {
	for _, node := range b.stack {
		if node != b.root && node.Type == ElementNode {
			return node
		}
	}

	return b.root
}

// closeHTMLElement applies the in-body "any other end tag" rule: walk the
// stack from the current node; a matching HTML element is popped together
// with everything above it, but a special element that does not match makes
// the end tag a no-op.
func (b *treeBuilder) closeHTMLElement(data string) {
	for i := len(b.stack) - 1; i > 0; i-- {
		node := b.stack[i]
		if node.Namespace == NamespaceHTML && node.Name == data {
			b.generateImpliedEndTags(data)
			b.stack = b.stack[:i]

			return
		}

		if isSpecialElement(node) {
			return
		}
	}
}

// closeForeignElement applies the foreign-content end-tag rules: a matching
// element name anywhere in the foreign run pops through it. It reports
// whether the token was consumed; reaching an HTML element first returns
// false so the token is reprocessed under the HTML insertion mode, whose
// scope rules decide whether the end tag applies.
func (b *treeBuilder) closeForeignElement(data string) bool {
	if data == "script" && b.top().Namespace == NamespaceSVG && strings.EqualFold(b.top().Name, "script") {
		b.stack = b.stack[:len(b.stack)-1]

		return true
	}

	i := len(b.stack) - 1
	for {
		if i <= 1 {
			return true // topmost element reached: ignore the stray end tag
		}

		node := b.stack[i]
		if strings.EqualFold(node.Name, data) {
			b.stack = b.stack[:i]

			return true
		}

		i--
		if b.stack[i].Namespace == NamespaceHTML {
			return false // process the token under the HTML rules
		}
	}
}

// --- scope and implied end tags ---

// integrationPointStops is the sentinel key that adds the foreign scope
// boundaries to a stops map: the default-derived scopes (default, button,
// list item) also end at MathML text integration points and HTML integration
// points. Table scope omits it, because only HTML html, table, and template
// elements end a table-scope search.
const integrationPointStops = "#integration-points"

var (
	defaultScopeStops = map[string]bool{
		"applet": true, "caption": true, "html": true, "table": true, "td": true,
		"th": true, "marquee": true, "object": true, "select": true, "template": true,
		integrationPointStops: true,
	}

	buttonScopeStops = map[string]bool{
		"applet": true, "caption": true, "html": true, "table": true, "td": true,
		"th": true, "marquee": true, "object": true, "select": true, "template": true,
		"button":              true,
		integrationPointStops: true,
	}

	listItemScopeStops = map[string]bool{
		"applet": true, "caption": true, "html": true, "table": true, "td": true,
		"th": true, "marquee": true, "object": true, "select": true, "template": true,
		"ol": true, "ul": true,
		integrationPointStops: true,
	}

	impliedEndTags = map[string]bool{
		"dd": true, "dt": true, "li": true, "optgroup": true, "option": true,
		"p": true, "rb": true, "rp": true, "rt": true, "rtc": true,
	}

	headingNames = []string{"h1", "h2", "h3", "h4", "h5", "h6"}

	bodyEndTagsWithScope = map[string]bool{
		"address": true, "article": true, "aside": true, "blockquote": true,
		"button": true, "center": true, "details": true, "dialog": true,
		"dir": true, "div": true, "dl": true, "fieldset": true, "figcaption": true,
		"figure": true, "footer": true, "header": true, "hgroup": true,
		"listing": true, "main": true, "menu": true, "nav": true, "ol": true,
		"pre": true, "search": true, "section": true, "summary": true, "ul": true,
	}

	specialTags = map[string]bool{
		"address": true, "applet": true, "area": true, "article": true, "aside": true,
		"base": true, "basefont": true, "bgsound": true, "blockquote": true, "body": true,
		"br": true, "button": true, "caption": true, "center": true, "col": true,
		"colgroup": true, "dd": true, "details": true, "dir": true, "div": true,
		"dl": true, "dt": true, "embed": true, "fieldset": true, "figcaption": true,
		"figure": true, "footer": true, "form": true, "frame": true, "frameset": true,
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"head": true, "header": true, "hgroup": true, "hr": true, "html": true,
		"iframe": true, "img": true, "input": true, "keygen": true, "li": true,
		"link": true, "listing": true, "main": true, "marquee": true, "menu": true,
		"meta": true, "nav": true, "noembed": true, "noframes": true, "noscript": true,
		"object": true, "ol": true, "p": true, "param": true, "plaintext": true,
		"pre": true, "script": true, "search": true, "section": true, "select": true,
		"source": true, "style": true, "summary": true, "table": true, "tbody": true,
		"td": true, "template": true, "textarea": true, "tfoot": true, "th": true,
		"thead": true, "title": true, "tr": true, "track": true, "ul": true,
		"wbr": true, "xmp": true,
	}
)

func (b *treeBuilder) findInScope(name string, stops map[string]bool) *Node {
	for i := len(b.stack) - 1; i > 0; i-- {
		node := b.stack[i]
		if node.Namespace == NamespaceHTML && node.Name == name {
			return node
		}

		if isScopeBoundary(node, stops) {
			return nil
		}
	}

	return nil
}

func (b *treeBuilder) hasInScope(name string, stops map[string]bool) bool {
	return b.findInScope(name, stops) != nil
}

func (b *treeBuilder) hasInScopeAny(names []string, stops map[string]bool) bool {
	for i := len(b.stack) - 1; i > 0; i-- {
		node := b.stack[i]
		if node.Namespace == NamespaceHTML {
			for _, name := range names {
				if node.Name == name {
					return true
				}
			}
		}

		if isScopeBoundary(node, stops) {
			return false
		}
	}

	return false
}

func isScopeBoundary(node *Node, stops map[string]bool) bool {
	if node.Namespace == NamespaceHTML {
		return stops[node.Name]
	}

	return stops[integrationPointStops] &&
		(isMathMLTextIntegrationPoint(node) || isHTMLIntegrationPoint(node))
}

func (b *treeBuilder) hasInButtonScope(name string) bool {
	return b.hasInScope(name, buttonScopeStops)
}

func (b *treeBuilder) hasInListScope(name string) bool {
	return b.hasInScope(name, listItemScopeStops)
}

func (b *treeBuilder) generateImpliedEndTags(except string) {
	for len(b.stack) > 1 {
		top := b.top()
		if top.Namespace != NamespaceHTML || !impliedEndTags[top.Name] || top.Name == except {
			return
		}

		b.stack = b.stack[:len(b.stack)-1]
	}
}

func (b *treeBuilder) popUntilName(name string) {
	for len(b.stack) > 1 {
		top := b.top()
		b.stack = b.stack[:len(b.stack)-1]

		if top.Namespace == NamespaceHTML && top.Name == name {
			return
		}
	}
}

func (b *treeBuilder) popUntilAnyName(names ...string) {
	for len(b.stack) > 1 {
		top := b.top()
		b.stack = b.stack[:len(b.stack)-1]

		if top.Namespace != NamespaceHTML {
			continue
		}

		for _, name := range names {
			if top.Name == name {
				return
			}
		}
	}
}

// closePElement closes the p element in button scope: implied end tags except
// p, then pop until p has been popped.
func (b *treeBuilder) closePElement() {
	b.generateImpliedEndTags("p")
	b.popUntilName("p")
}

func (b *treeBuilder) closePElementIfOpen() {
	if !b.hasInButtonScope("p") {
		return
	}

	b.closePElement()
}

// closeOpenListItem implements the standard's list-item loop: a new li closes
// the nearest open li, continuing past address, div, and p elements.
func (b *treeBuilder) closeOpenListItem() {
	for i := len(b.stack) - 1; i > 0; i-- {
		node := b.stack[i]
		if node.Namespace == NamespaceHTML && node.Name == "li" {
			b.generateImpliedEndTags("li")
			b.popUntilName("li")

			return
		}

		if isSpecialElement(node) && !isListContinuationElement(node) {
			return
		}
	}
}

// closeOpenDefinitionItem is closeOpenListItem for dd and dt.
func (b *treeBuilder) closeOpenDefinitionItem() {
	for i := len(b.stack) - 1; i > 0; i-- {
		node := b.stack[i]
		if node.Namespace == NamespaceHTML && (node.Name == "dd" || node.Name == "dt") {
			b.generateImpliedEndTags(node.Name)
			b.popUntilAnyName("dd", "dt")

			return
		}

		if isSpecialElement(node) && !isListContinuationElement(node) {
			return
		}
	}
}

func isSpecialElement(node *Node) bool {
	if node.Namespace == NamespaceHTML {
		return specialTags[node.Name]
	}

	return isMathMLTextIntegrationPoint(node) || isHTMLIntegrationPoint(node)
}

// isListContinuationElement reports whether the li/dd/dt search continues past
// this element: address, div, and p do not stop the list-item loop.
func isListContinuationElement(node *Node) bool {
	if node.Namespace != NamespaceHTML {
		return false
	}

	switch node.Name {
	case "address", "div", "p":
		return true
	default:
		return false
	}
}

// --- legacy option closing (HTML-CONTEXT-01 owns the select insertion modes) ---

// legacyOptionAutoClose closes an open option before a new option or
// optgroup, and an open optgroup before a new optgroup.
func (b *treeBuilder) legacyOptionAutoClose(name string) {
	if b.top().Namespace != NamespaceHTML {
		return
	}

	switch name {
	case "option":
		if b.top().Name == "option" {
			b.stack = b.stack[:len(b.stack)-1]
		}
	case "optgroup":
		if b.top().Name == "option" {
			b.stack = b.stack[:len(b.stack)-1]
		}

		if b.top().Namespace == NamespaceHTML && b.top().Name == "optgroup" {
			b.stack = b.stack[:len(b.stack)-1]
		}
	}
}

// --- small predicates ---

func isAllWhitespace(data string) bool {
	for i := 0; i < len(data); i++ {
		if !isWhitespace(data[i]) {
			return false
		}
	}

	return true
}

// splitLeadingWhitespace splits data into its leading ASCII whitespace and the
// rest, so modes that ignore or relocate leading whitespace can process the
// remainder correctly.
func splitLeadingWhitespace(data string) (string, string) {
	i := 0
	for i < len(data) && isWhitespace(data[i]) {
		i++
	}

	return data[:i], data[i:]
}

func isHeadingElement(node *Node) bool {
	if node.Namespace != NamespaceHTML {
		return false
	}

	for _, name := range headingNames {
		if node.Name == name {
			return true
		}
	}

	return false
}

// textModeElement reports whether a start tag's content is consumed by the
// tokenizer as raw text, so the tree builder enters the "text" mode until the
// matching end tag.
func textModeElement(name string) bool {
	_, raw := rawTextMode(name)

	return raw
}

// openInStack reports whether an element with name is currently open.
func openInStack(stack []*Node, name string) bool {
	for i := len(stack) - 1; i > 0; i-- {
		if stack[i].Name == name {
			return true
		}
	}

	return false
}

// isVoidElement reports whether name never takes content.
func isVoidElement(name string) bool {
	switch name {
	case "area", "base", "br", "col", "embed", "hr", "img", "input",
		"link", "meta", "param", "source", "track", "wbr":
		return true
	}

	return false
}
