package layout

import (
	"slices"
	"strconv"
	"strings"
)

// Values shared by the frag normalizers.
const (
	fragNone       = "none"
	fragStateClose = "closed"
	fragNoEllipsis = "no-ellipsis"
)

// fragEnumValues returns the keyword set of a frag property whose grammar is
// one keyword, or nil for any other property.
func fragEnumValues(prop string) []string {
	switch prop {
	case bookmarkStateProp:
		return []string{"open", fragStateClose}
	case footnotePropDisplay:
		return []string{timelineAxisBlock, "inline", "compact"}
	case footnotePropPolicy:
		return []string{"auto", "line", timelineAxisBlock}
	}

	return nil
}

// normalizeFragDeclared checks and normalizes the value of one frag property:
// `bookmark-label`, `bookmark-level`, `bookmark-state`, `footnote-display`,
// `footnote-policy`, `copy-into`, `block-ellipsis`. It returns the trimmed
// text, or false for a value outside the grammar.
func normalizeFragDeclared(prop, value string) (string, bool) {
	text := strings.TrimSpace(value)
	if text == "" {
		return "", false
	}

	low := strings.ToLower(text)

	if words := fragEnumValues(prop); words != nil {
		return low, slices.Contains(words, low)
	}

	switch prop {
	case bookmarkLevelProp:
		return normalizeBookmarkLevel(low)
	case bookmarkLabelProp:
		return normalizeBookmarkLabelList(text)
	case "copy-into":
		return normalizeCopyInto(text)
	case "block-ellipsis":
		return normalizeBlockEllipsis(text, low)
	}

	return "", false
}

// normalizeBookmarkLevel accepts none or an integer of at least 1.
func normalizeBookmarkLevel(low string) (string, bool) {
	if low == fragNone {
		return low, true
	}

	level, err := strconv.Atoi(strings.TrimPrefix(low, "+"))

	return strconv.Itoa(level), err == nil && level >= 1
}

// normalizeBlockEllipsis accepts no-ellipsis, ellipsis or a quoted string.
func normalizeBlockEllipsis(text, low string) (string, bool) {
	if low == fragNoEllipsis || low == textOverflowEllipsis {
		return low, true
	}

	return text, isQuotedString(text)
}

// isQuotedString reports whether text is one complete quoted string.
func isQuotedString(text string) bool {
	if len(text) < 2 || (text[0] != '"' && text[0] != '\'') {
		return false
	}

	for i := 1; i < len(text); i++ {
		switch text[i] {
		case '\\':
			i++
		case text[0]:
			return i == len(text)-1
		}
	}

	return false
}

// bookmarkLabelToken reports whether tok is one item of a <content-list>
// this property accepts: a string, content, or a known function call.
func bookmarkLabelToken(tok string) bool {
	if isQuotedString(tok) || strings.EqualFold(tok, propContent) {
		return true
	}

	for _, fn := range []string{propContent, "attr", "counter", "counters", "string", "text"} {
		if arg, ok := funcArgs(tok, fn); ok && (arg != "" || fn == propContent) {
			return true
		}
	}

	return false
}

// normalizeBookmarkLabelList accepts none or a space list of label tokens.
func normalizeBookmarkLabelList(text string) (string, bool) {
	toks := splitTop(text)
	if len(toks) == 1 && strings.EqualFold(toks[0], fragNone) {
		return fragNone, true
	}

	for _, tok := range toks {
		if !bookmarkLabelToken(tok) {
			return "", false
		}
	}

	return strings.Join(toks, " "), len(toks) > 0
}

// isCSSIdentByte reports whether the byte at index may appear there in an
// identifier.
func isCSSIdentByte(text string, index int) bool {
	c := text[index]

	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c == '_', c == '-':
		return true
	case c >= '0' && c <= '9':
		return index > 0 && (index != 1 || text[0] != '-')
	}

	return false
}

// isCSSIdent reports whether text is a plain identifier that is not a
// CSS-wide keyword or none.
func isCSSIdent(text string) bool {
	if text == "" || cssWideKeyword(strings.ToLower(text)) || strings.EqualFold(text, fragNone) {
		return false
	}

	for index := range len(text) {
		if !isCSSIdentByte(text, index) {
			return false
		}
	}

	return true
}

// copyIntoLevel normalizes a <content-level>: element, content, text, or
// attr(name).
func copyIntoLevel(tok string) (string, bool) {
	low := strings.ToLower(tok)

	switch low {
	case "element", propContent, "text":
		return low, true
	}

	if arg, ok := funcArgs(tok, "attr"); ok && isCSSIdent(arg) {
		return "attr(" + arg + ")", true
	}

	return "", false
}

// normalizeCopyInto accepts none or a comma list of `<ident> <content-level>`.
func normalizeCopyInto(text string) (string, bool) {
	if strings.EqualFold(text, fragNone) {
		return fragNone, true
	}

	items := strings.Split(text, ",")
	parts := make([]string, 0, len(items))

	for _, item := range items {
		toks := splitTop(strings.TrimSpace(item))
		if len(toks) != 2 || !isCSSIdent(toks[0]) {
			return "", false
		}

		level, ok := copyIntoLevel(toks[1])
		if !ok {
			return "", false
		}

		parts = append(parts, toks[0]+" "+level)
	}

	return strings.Join(parts, ", "), len(parts) > 0
}

// mirrorFragDeclared copies the bookmark and footnote longhands into the
// declared store as well. Their older handlers keep a private canonical value
// in CustomProps and return first, so the declared store would never see
// them; the mirror is what Box.Declared exports. Other properties pass
// through untouched.
func mirrorFragDeclared(style *ResolvedStyle, prop, value string, parent *ResolvedStyle) {
	switch prop {
	case bookmarkLabelProp, bookmarkLevelProp, bookmarkStateProp, footnotePropDisplay, footnotePropPolicy:
		applyDeclaredProps(style, prop, value, parent, parent != nil)
	}
}
