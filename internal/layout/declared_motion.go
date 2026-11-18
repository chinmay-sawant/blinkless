package layout

import "strings"

// normalizeMotionDeclared checks and normalizes the value of one motion path
// property: `offset`, `offset-anchor`, `offset-distance`, `offset-path`,
// `offset-position`, `offset-rotate` (CSS Motion Path 1). It returns the
// trimmed text with white space collapsed, or false for an invalid value.
func normalizeMotionDeclared(prop, value string) (string, bool) {
	text := normalizeMotionText(value)
	if text == "" {
		return "", false
	}

	toks := splitTop(text)

	switch prop {
	case "offset":
		_, ok := parseOffsetShorthand(text, motionNormFontSize)

		return text, ok
	case "offset-path":
		_, _, ok := parseMotionPathTokens(toks)

		return text, ok
	case "offset-distance":
		_, ok := parseMotionLen(text, motionNormFontSize)

		return text, ok && len(toks) == 1
	case "offset-rotate":
		_, ok := parseMotionRotate(toks)

		return text, ok
	case "offset-position":
		_, _, ok := parseMotionPositionValue(append([]string(nil), toks...), motionNormFontSize)

		return text, ok
	case "offset-anchor":
		_, _, ok := parseAnchorValue(toks, motionNormFontSize)

		return text, ok
	}

	return "", strings.TrimSpace(text) != ""
}
