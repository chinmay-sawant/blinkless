package layout

import "strings"

// blockEllipsisMarker is the text added to the last visible line of a block
// that line-clamp or max-lines cut short. With no block-ellipsis declared the
// marker is the single character U+2026, the old line-clamp behavior.
// no-ellipsis adds nothing, ellipsis adds U+2026, and a string adds that
// string. The marker is appended to the line as text, so a long string can
// run past the line end.
func blockEllipsisMarker(style *ResolvedStyle) string {
	if style == nil {
		return "…"
	}

	switch value := style.Declared.BlockEllipsis; value {
	case "", textOverflowEllipsis:
		return "…"
	case fragNoEllipsis:
		return ""
	default:
		return fragUnquote(value)
	}
}

// fragQuoteLen is the length of an empty quoted string: just the two quotes.
const fragQuoteLen = 2

// fragUnquote removes the quotes and backslash escapes of one string.
func fragUnquote(text string) string {
	if len(text) < fragQuoteLen {
		return text
	}

	var out strings.Builder

	for i := 1; i < len(text)-1; i++ {
		if text[i] == '\\' && i+1 < len(text)-1 {
			i++
		}

		out.WriteByte(text[i])
	}

	return out.String()
}
