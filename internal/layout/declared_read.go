package layout

import (
	"strconv"
	"strings"
)

// normalizeReadDeclared checks and normalizes the value of one read property.
// background-tbd is the Backgrounds 4 layer list without a color, so a value
// that carries a color token is invalid.
func normalizeReadDeclared(prop, value string) (string, bool) {
	text := strings.TrimSpace(value)
	if text == "" {
		return "", false
	}

	switch prop {
	case "reading-flow":
		return navOneKeyword(text, []string{
			"normal", "source-order", "flex-visual", "flex-flow", "grid-rows", "grid-columns", "grid-order",
		})
	case "overlay":
		return navOneKeyword(text, []string{"none", "auto"})
	case "reading-order":
		n, err := strconv.Atoi(text)
		if err != nil {
			return "", false
		}

		return strconv.Itoa(n), true
	case declPropBackgroundAttachment:
		return normalizeBackgroundAttachment(text)
	case declPropBackgroundTbd:
		if backgroundTbdHasColor(text) {
			return "", false
		}

		return text, true
	}

	return "", false
}

// normalizeBackgroundAttachment accepts a comma list of scroll | fixed | local.
func normalizeBackgroundAttachment(text string) (string, bool) {
	parts := strings.Split(text, ",")
	out := make([]string, 0, len(parts))

	for _, part := range parts {
		k, ok := navOneKeyword(strings.TrimSpace(part), []string{"scroll", "fixed", "local"})
		if !ok {
			return "", false
		}

		out = append(out, k)
	}

	return strings.Join(out, ", "), true
}
