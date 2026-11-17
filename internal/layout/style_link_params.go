package layout

import "strings"

// link-parameters (CSS Link Parameters) passes named values into a linked
// resource. Syntax: none | <param()>#. A param() holds a custom property name
// and a value, written "param(--color: #ff0000)" or "param(--color, #ff0000)".
// The property does not inherit and does not touch an <a href>.
//
// ResolvedStyle.LinkParameters holds the canonical list ("name<TAB>value" per
// line) that svg.ApplyLinkParams reads. A param() that fails the grammar is
// skipped and the rest of the list is kept. A list with no valid param() is
// refused, so the previous value stays. none clears the list.

const (
	linkParametersProp = "link-parameters"
)

// applyLinkParams owns link-parameters. Registered in styleGroups
// (style_cascade.go).
func applyLinkParams(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	if prop != linkParametersProp {
		return false
	}

	trimmed := strings.TrimSpace(value)
	if strings.EqualFold(trimmed, svgValueNone) {
		style.LinkParameters = ""

		return true
	}

	if canon, ok := parseLinkParameters(trimmed); ok {
		style.LinkParameters = canon
	}

	return true
}

// parseLinkParameters turns a comma list of param() into the canonical form.
func parseLinkParameters(value string) (string, bool) {
	var lines []string

	for _, item := range splitTopLevelCommas(value) {
		name, val, ok := parseOneParam(item)
		if ok {
			lines = append(lines, name+"\t"+val)
		}
	}

	return strings.Join(lines, "\n"), len(lines) > 0
}

func parseOneParam(item string) (string, string, bool) {
	item = strings.TrimSpace(item)
	if len(item) < len("param()") || !strings.EqualFold(item[:len("param(")], "param(") ||
		!strings.HasSuffix(item, ")") {
		return "", "", false
	}

	inner := item[len("param(") : len(item)-1]

	cut := strings.IndexAny(inner, ":,")
	if cut < 0 {
		return "", "", false
	}

	name := strings.TrimSpace(inner[:cut])
	val := strings.Join(strings.Fields(inner[cut+1:]), " ")

	if len(name) < len("--x") || !strings.HasPrefix(name, "--") || val == "" || strings.ContainsAny(name, " \t\n()") {
		return "", "", false
	}

	return name, val, true
}

// splitTopLevelCommas splits on commas outside parentheses.
func splitTopLevelCommas(text string) []string {
	var parts []string

	depth, start := 0, 0

	for pos := range len(text) {
		switch text[pos] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, text[start:pos])
				start = pos + 1
			}
		}
	}

	return append(parts, text[start:])
}
