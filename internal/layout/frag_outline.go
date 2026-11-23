package layout

import (
	"strconv"
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// Frag helpers for the bookmark and copy-into properties. The outline model
// itself lives in bookmark_outline.go; this file adds attr() labels and the
// copy-into reader.

// attrCallLen is the length of the text "attr(".
const attrCallLen = len("attr(")

// skipQuotedString returns the index of the closing quote of the string that
// opens at raw[start], or the last byte when it never closes.
func skipQuotedString(raw string, start int) int {
	quote := raw[start]
	end := start + 1

	for end < len(raw) && raw[end] != quote {
		if raw[end] == '\\' {
			end++
		}

		end++
	}

	if end >= len(raw) {
		end = len(raw) - 1
	}

	return end
}

// attrCallAt reads an attr(name) call at raw[start]. It returns the quoted
// attribute value and the index after the call, or ok false when raw[start]
// does not begin a closed call.
func attrCallAt(raw string, start int, node *html.Node) (string, int, bool) {
	if len(raw)-start < attrCallLen || !strings.EqualFold(raw[start:start+attrCallLen], "attr(") {
		return "", 0, false
	}

	closeAt := strings.IndexByte(raw[start:], ')')
	if closeAt <= 0 {
		return "", 0, false
	}

	fields := strings.Fields(raw[start+attrCallLen : start+closeAt])
	value := ""

	if len(fields) > 0 {
		value = node.Attribute(fields[0])
	}

	return strconv.Quote(value), start + closeAt + 1, true
}

// substituteBookmarkAttr replaces each attr(name) in a bookmark-label list
// with a quoted string holding the element's attribute value, so the content
// list parser sees a plain string. A missing attribute becomes "".
func substituteBookmarkAttr(raw string, node *html.Node) string {
	if node == nil || !strings.Contains(strings.ToLower(raw), "attr(") {
		return raw
	}

	var out strings.Builder

	for pos := 0; pos < len(raw); {
		char := raw[pos]

		if char == '"' || char == '\'' {
			end := skipQuotedString(raw, pos)
			out.WriteString(raw[pos : end+1])

			pos = end + 1

			continue
		}

		if quoted, next, ok := attrCallAt(raw, pos, node); ok {
			out.WriteString(quoted)

			pos = next

			continue
		}

		out.WriteByte(char)

		pos++
	}

	return out.String()
}

// CopiedFragment is one named copy a copy-into declaration makes: the element
// text (or attribute value) under a name a host can place elsewhere, such as
// in a running header.
type CopiedFragment struct {
	// Name is the custom identifier of the copy.
	Name string
	// Level is the declared content level: element, content, text, or
	// attr(name).
	Level string
	// Text is the copied text.
	Text string
}

// CopiedFragments walks the tree in document order and returns every
// copy-into entry, in declaration order within an element. copy-into: none
// and an absent declaration add nothing.
func CopiedFragments(root *html.Node, opts Options) []CopiedFragment {
	if root == nil {
		return nil
	}

	ctx := &styleContext{
		sheets:     opts.Sheets,
		properties: registeredProperties(opts.Sheets),
		media:      opts.Media,
		viewportW:  opts.Width,
		viewportH:  opts.Height,
		state:      opts.State,
	}

	var out []CopiedFragment

	var walk func(node *html.Node)

	walk = func(node *html.Node) {
		if node == nil {
			return
		}

		if node.Type == html.ElementNode {
			raw := cascadeRaw(ctx, node, ctx.matchedRules(node, ""))
			value := raw["copy-into"]

			if text, ok := normalizeFragDeclared("copy-into", value); ok && text != fragNone {
				out = append(out, copiedFromList(text, node)...)
			}
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(root)

	return out
}

// copiedFromList turns a normalized copy-into list into fragments.
func copiedFromList(list string, node *html.Node) []CopiedFragment {
	items := strings.Split(list, ",")
	out := make([]CopiedFragment, 0, len(items))

	for _, item := range items {
		toks := strings.Fields(strings.TrimSpace(item))
		if len(toks) != fragQuoteLen {
			continue
		}

		text := bookmarkElementText(node)
		if arg, ok := funcArgs(toks[1], "attr"); ok {
			text = node.Attribute(arg)
		}

		out = append(out, CopiedFragment{Name: toks[0], Level: toks[1], Text: strings.TrimSpace(text)})
	}

	return out
}
