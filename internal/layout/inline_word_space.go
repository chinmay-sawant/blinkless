// word-space-transform (CSS Text 4) replaces expandable separators in an
// element's text. none changes nothing. space writes U+0020. ideographic-space
// writes U+3000. auto-phrase is stored with either of those keywords and
// inserts no phrase breaks, because this engine has no phrase detector.
// A separator next to a forced line break stays as it is. A token outside
// the grammar leaves the previous value. The property inherits.
package layout

import (
	"strings"
	"unicode/utf8"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

const (
	wordSpaceTransformNone        = "none"
	wordSpaceTransformSpace       = "space"
	wordSpaceTransformIdeographic = "ideographic-space"
	wordSpaceTransformAutoPhrase  = "auto-phrase"
	wordSpaceTagWBR               = "wbr"

	// wordSpaceZWSP is U+200B, an expandable separator.
	wordSpaceZWSP = '\u200B'
	// wordSpaceIdeographic is U+3000, the ideographic-space replacement.
	wordSpaceIdeographic = '\u3000'
	// wordSpaceKept marks a U+200B that sits against a forced line break.
	// transformWordSpaceText restores it to U+200B so a later pass does not
	// expand that one separator.
	wordSpaceKept = '\uE000'
)

// applyWordSpaceTransformProps owns word-space-transform. Registered in
// styleGroups. A bad token leaves the stored value alone.
func applyWordSpaceTransformProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext,
	parent *ResolvedStyle, hasParent bool,
) bool {
	if prop != "word-space-transform" {
		return false
	}

	if next, ok := resolveWordSpaceTransform(value, parent, hasParent); ok {
		style.WordSpaceTransform = next
	}

	return true
}

// wordSpaceTransformValueAccepted is the cascade gate. CSS-wide keywords are
// accepted before this runs. auto-phrase alone is outside the grammar.
func wordSpaceTransformValueAccepted(value string) bool {
	_, ok := normalizeWordSpaceTransform(value)

	return ok
}

// resolveWordSpaceTransform returns the used value. ok is false when value
// is outside the grammar, so the caller keeps the previous value.
func resolveWordSpaceTransform(value string, parent *ResolvedStyle, hasParent bool) (string, bool) {
	parentValue := wordSpaceTransformNone
	if hasParent && parent != nil && parent.WordSpaceTransform != "" {
		parentValue = parent.WordSpaceTransform
	}

	if used, ok := cssWideValue(value, parentValue, wordSpaceTransformNone, hasParent); ok {
		return used, true
	}

	if normalizeCSSValue(value) == stringSetRevertLayer {
		return wordSpaceTransformNone, true
	}

	return normalizeWordSpaceTransform(value)
}

// normalizeWordSpaceTransform accepts none, space, ideographic-space, and
// either of the last two paired with auto-phrase in either order.
func normalizeWordSpaceTransform(value string) (string, bool) {
	fields := strings.Fields(normalizeCSSValue(value))

	switch len(fields) {
	case 1:
		switch fields[0] {
		case wordSpaceTransformNone, wordSpaceTransformSpace, wordSpaceTransformIdeographic:
			return fields[0], true
		default:
			return "", false
		}
	case 2:
		keyword := fields[0]
		phrase := fields[1]

		if keyword == wordSpaceTransformAutoPhrase {
			keyword, phrase = phrase, keyword
		}

		if phrase != wordSpaceTransformAutoPhrase {
			return "", false
		}

		switch keyword {
		case wordSpaceTransformSpace, wordSpaceTransformIdeographic:
			return keyword + " " + wordSpaceTransformAutoPhrase, true
		default:
			return "", false
		}
	default:
		return "", false
	}
}

// wordSpaceReplacement is the rune space or ideographic-space writes.
// none, an empty value, and auto-phrase with no separator keyword return 0.
func wordSpaceReplacement(style *ResolvedStyle) rune {
	if style == nil {
		return 0
	}

	switch style.WordSpaceTransform {
	case wordSpaceTransformSpace, wordSpaceTransformSpace + " " + wordSpaceTransformAutoPhrase:
		return ' '
	case wordSpaceTransformIdeographic, wordSpaceTransformIdeographic + " " + wordSpaceTransformAutoPhrase:
		return wordSpaceIdeographic
	default:
		return 0
	}
}

// transformWordSpaceText rewrites expandable separators before measure.
// A wordSpaceKept rune is a U+200B that must stay, because it sits against
// a forced line break. auto-phrase adds no characters.
func transformWordSpaceText(text string, style *ResolvedStyle) string {
	if text == "" || style == nil {
		return text
	}

	repl := wordSpaceReplacement(style)
	needsKept := strings.ContainsRune(text, wordSpaceKept)
	needsZWSP := repl != 0 && strings.ContainsRune(text, wordSpaceZWSP)

	if !needsKept && !needsZWSP {
		return text
	}

	var b strings.Builder
	b.Grow(len(text))

	for _, r := range text {
		switch r {
		case wordSpaceKept:
			r = wordSpaceZWSP
		case wordSpaceZWSP:
			if repl != 0 {
				r = repl
			}
		}

		b.WriteRune(r)
	}

	return b.String()
}

// guardWordSpaceEdges protects a U+200B at either end of text when that end
// sits against a forced line break. Other separators still expand.
func (e *engine) guardWordSpaceEdges(node *html.Node, text string, style *ResolvedStyle) string {
	if wordSpaceReplacement(style) == 0 || !strings.ContainsRune(text, wordSpaceZWSP) {
		return text
	}

	lead, trail := e.wordSpaceBreakNeighbors(node)

	return protectWordSpaceEdges(text, lead, trail)
}

// guardWordSpaceLine protects the ends of one preserved line. A newline on
// that side is a forced break, and so is a br outside the text node.
func guardWordSpaceLine(line string, style *ResolvedStyle, leadBreak, trailBreak bool) string {
	if wordSpaceReplacement(style) == 0 || !strings.ContainsRune(line, wordSpaceZWSP) {
		return line
	}

	return protectWordSpaceEdges(line, leadBreak, trailBreak)
}

// protectWordSpaceEdges swaps an edge U+200B for wordSpaceKept.
func protectWordSpaceEdges(text string, keepLead, keepTrail bool) string {
	if text == "" || (!keepLead && !keepTrail) {
		return text
	}

	lead, leadSize := utf8.DecodeRuneInString(text)
	trail, trailSize := utf8.DecodeLastRuneInString(text)
	replaceLead := keepLead && lead == wordSpaceZWSP
	replaceTrail := keepTrail && trail == wordSpaceZWSP

	if !replaceLead && !replaceTrail {
		return text
	}

	if replaceLead && replaceTrail && leadSize == len(text) {
		return string(wordSpaceKept)
	}

	var b strings.Builder
	b.Grow(len(text))

	start := 0
	end := len(text)

	if replaceLead {
		b.WriteRune(wordSpaceKept)
		start = leadSize
	}

	if replaceTrail {
		end -= trailSize
	}

	if start < end {
		b.WriteString(text[start:end])
	}

	if replaceTrail {
		b.WriteRune(wordSpaceKept)
	}

	return b.String()
}

// collectWbr emits the replacement for a wbr element. none adds nothing.
// A wbr next to a forced line break stays empty.
func (e *engine) collectWbr(node *html.Node, sty *ResolvedStyle, out *[]inlineItem) {
	repl := wordSpaceReplacement(sty)
	if repl == 0 || e.wordSpaceBesideBreak(node) {
		return
	}

	*out = append(*out, e.textItem(string(repl), sty))
}

// wordSpaceBesideBreak reports that node touches a br, ignoring inline boxes.
func (e *engine) wordSpaceBesideBreak(node *html.Node) bool {
	lead, trail := e.wordSpaceBreakNeighbors(node)

	return lead || trail
}

// wordSpaceBreakNeighbors reports a forced break before and after node.
func (e *engine) wordSpaceBreakNeighbors(node *html.Node) (bool, bool) {
	if node == nil {
		return false, false
	}

	return e.wordSpaceBreakFrom(node, false), e.wordSpaceBreakFrom(node, true)
}

// wordSpaceBreakFrom walks to the next significant sibling. Inline boxes and
// empty text are not content, so a br on the far side of them still counts.
func (e *engine) wordSpaceBreakFrom(node *html.Node, forward bool) bool {
	for node != nil {
		sib := wordSpaceSibling(node, forward)
		if sib == nil {
			parent := node.Parent
			if !e.wordSpaceInlineBoundary(parent) {
				return false
			}

			node = parent

			continue
		}

		hit, cont := e.wordSpaceBreakAt(sib, forward)
		if !cont {
			return hit
		}

		node = sib
	}

	return false
}

// wordSpaceBreakAt reports whether node is a br, or whether the walk should
// keep going through an inline box. cont is false when the answer is final.
func (e *engine) wordSpaceBreakAt(node *html.Node, forward bool) (bool, bool) {
	if node == nil {
		return false, false
	}

	switch node.Type {
	case html.CommentNode, html.DoctypeNode:
		return false, true
	case html.TextNode:
		if node.Text == "" {
			return false, true
		}

		return false, false
	case html.ElementNode:
		if node.Name == cssTagBR {
			return true, false
		}

		if !e.wordSpaceInlineBoundary(node) || len(node.Children) == 0 {
			return false, e.wordSpaceInlineBoundary(node)
		}

		var inner *html.Node
		if forward {
			inner = node.Children[0]
		} else {
			inner = node.Children[len(node.Children)-1]
		}

		return e.wordSpaceBreakInside(inner, forward)
	case html.NodeUnknown:
		return false, false
	default:
		return false, false
	}
}

// wordSpaceBreakInside continues the search from a node already entered.
func (e *engine) wordSpaceBreakInside(node *html.Node, forward bool) (bool, bool) {
	hit, cont := e.wordSpaceBreakAt(node, forward)
	if !cont {
		return hit, false
	}

	return e.wordSpaceBreakFrom(node, forward), false
}

// wordSpaceInlineBoundary reports an inline box the break walk may cross.
func (e *engine) wordSpaceInlineBoundary(node *html.Node) bool {
	if node == nil || node.Type != html.ElementNode || node.Name == cssTagBR {
		return false
	}

	sty := e.stylePtr(node)

	return sty != nil && sty.Display == cssDisplayInline
}

// wordSpaceSibling returns the adjacent child of node's parent.
func wordSpaceSibling(node *html.Node, forward bool) *html.Node {
	if node == nil || node.Parent == nil {
		return nil
	}

	kids := node.Parent.Children

	for i, child := range kids {
		if child != node {
			continue
		}

		if forward && i+1 < len(kids) {
			return kids[i+1]
		}

		if !forward && i > 0 {
			return kids[i-1]
		}

		return nil
	}

	return nil
}
