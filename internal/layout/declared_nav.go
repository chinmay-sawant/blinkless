package layout

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// navKeywordSets lists the keyword grammar of every nav property that takes
// one keyword from a fixed set. It is built once on first use.
var navKeywordSets = sync.OnceValue(buildNavKeywordSets)

// buildNavKeywordSets builds the keyword table behind navKeywordSets.
func buildNavKeywordSets() map[string][]string {
	// The legacy appearance values WebKit accepts.
	appearance := []string{
		"none", "auto", "button", "checkbox", "radio", "push-button", "square-button", "textfield", "textarea",
		"searchfield", "menulist", "menulist-button", "listbox", "slider-horizontal", "slider-vertical",
		"sliderthumb-horizontal", "sliderthumb-vertical", "progress-bar", "meter", "caret", "inner-spin-button",
		"media-slider", "media-volume-slider", "searchfield-cancel-button",
	}

	return map[string][]string{
		"caret-animation":             {"auto", hyphenationManual},
		"caret-shape":                 {"auto", "bar", timelineAxisBlock, "underscore"},
		"field-sizing":                {"fixed", propContent},
		"input-security":              {"auto", "none"},
		"slider-orientation":          {"auto", "left-to-right", "right-to-left", "top-to-bottom", "bottom-to-top"},
		"spatial-navigation-action":   {"auto", "focus", "scroll"},
		"spatial-navigation-contain":  {"auto", "contain"},
		"spatial-navigation-function": {"normal", "grid"},
		"window-drag":                 {"none", "move"},
		"-webkit-user-select":         {"auto", "text", "none", "all", "contain"},
		"image-animation":             {"normal", "paused", "stopped", "running"},
		"-webkit-appearance":          appearance,
	}
}

// normalizeNavDeclared checks and normalizes the value of one nav property.
// It returns the normalized text, or false when the value does not match the
// property's grammar.
func normalizeNavDeclared(prop, value string) (string, bool) {
	text := strings.TrimSpace(value)
	if text == "" {
		return "", false
	}

	if set, ok := navKeywordSets()[prop]; ok {
		return navOneKeyword(text, set)
	}

	switch prop {
	case "caret":
		return normalizeCaretShorthand(text)
	case "nav-up", "nav-down", "nav-left", "nav-right":
		return normalizeNavTarget(text)
	case declPropWillChange:
		return normalizeWillChange(text)
	case "text-size-adjust", "-webkit-text-size-adjust":
		return normalizeTextSizeAdjust(text)
	}

	return "", false
}

// navOneKeyword accepts exactly one keyword from set, in lower case.
func navOneKeyword(text string, set []string) (string, bool) {
	lower := strings.ToLower(text)
	for _, k := range set {
		if lower == k {
			return k, true
		}
	}

	return "", false
}

// navScanner walks a value rune by rune and records where each field starts.
type navScanner struct {
	out   []string
	depth int
	quote rune
	start int
}

// open marks the start of a field at byte offset i when none is open.
func (scan *navScanner) open(i int) {
	if scan.start < 0 {
		scan.start = i
	}
}

// closeField ends the open field before byte offset i.
func (scan *navScanner) closeField(text string, i int) {
	if scan.start >= 0 {
		scan.out = append(scan.out, text[scan.start:i])
		scan.start = -1
	}
}

// isNavSpace reports whether r is white space that splits fields.
func isNavSpace(r rune) bool { return r == ' ' || r == '\t' || r == '\n' }

// closeParen lowers the parenthesis depth, never below zero.
func (scan *navScanner) closeParen() {
	if scan.depth > 0 {
		scan.depth--
	}
}

// step takes one rune at byte offset pos.
func (scan *navScanner) step(text string, pos int, char rune) {
	if scan.quote != 0 {
		if char == scan.quote {
			scan.quote = 0
		}

		return
	}

	switch {
	case char == '"' || char == '\'':
		scan.quote = char

		scan.open(pos)
	case char == '(':
		scan.depth++

		scan.open(pos)
	case char == ')':
		scan.closeParen()
	case isNavSpace(char) && scan.depth == 0:
		scan.closeField(text, pos)
	default:
		scan.open(pos)
	}
}

// navFields splits text on white space that sits outside parentheses and
// quotes.
func navFields(text string) []string {
	scan := navScanner{out: nil, depth: 0, quote: 0, start: -1}

	for pos, r := range text {
		scan.step(text, pos, r)
	}

	scan.closeField(text, len(text))

	return scan.out
}

// caretParts holds the three slots of the caret shorthand.
type caretParts struct {
	color, anim, shape string
}

// add places one field in its slot. It returns false when the field has no
// free slot or is not a caret value.
func (cp *caretParts) add(field string) bool {
	lower := strings.ToLower(field)

	switch {
	case lower == hyphenationManual:
		return cp.setOnce(&cp.anim, lower)
	case lower == "bar" || lower == timelineAxisBlock || lower == "underscore":
		return cp.setOnce(&cp.shape, lower)
	case lower == "auto":
		// auto is legal for the color and for the shape and for the
		// animation. The first free slot takes it, in the order color,
		// animation, shape.
		for _, slot := range []*string{&cp.color, &cp.anim, &cp.shape} {
			if *slot == "" {
				*slot = lower

				return true
			}
		}

		return false
	}

	if cp.color != "" || !navColorValue(field) {
		return false
	}

	cp.color = field

	return true
}

// setOnce fills an empty slot and refuses a second value.
func (cp *caretParts) setOnce(slot *string, value string) bool {
	if *slot != "" {
		return false
	}

	*slot = value

	return true
}

// normalizeCaretShorthand accepts <caret-color> || <caret-animation> ||
// <caret-shape>: each part at most once, in any order, at least one part.
// The parts are written back in the order color, animation, shape.
func normalizeCaretShorthand(text string) (string, bool) {
	var parts caretParts

	for _, field := range navFields(text) {
		if !parts.add(field) {
			return "", false
		}
	}

	var out []string

	for _, part := range []string{parts.color, parts.anim, parts.shape} {
		if part != "" {
			out = append(out, part)
		}
	}

	if len(out) == 0 {
		return "", false
	}

	return strings.Join(out, " "), true
}

// navColorValue reports whether text is a color the engine can read.
func navColorValue(text string) bool {
	if isCurrentColor(text) {
		return true
	}

	_, ok := parseUsedColor(text, [3]float64{})

	return ok
}

// navQuotedString reports whether frame is a non-empty quoted string.
func navQuotedString(frame string) bool {
	return len(frame) > 2 && (frame[0] == '"' || frame[0] == '\'') && frame[len(frame)-1] == frame[0]
}

// normalizeNavTarget accepts auto | <id> [ current | root | <target-name> ]?
// where <id> is written as #name. The target name is a quoted string.
func normalizeNavTarget(text string) (string, bool) {
	fields := navFields(text)

	if len(fields) == 1 && strings.EqualFold(fields[0], "auto") {
		return "auto", true
	}

	if len(fields) == 0 || len(fields) > 2 || !navIDToken(fields[0]) {
		return "", false
	}

	if len(fields) == 1 {
		return fields[0], true
	}

	frame := fields[1]
	lower := strings.ToLower(frame)

	switch {
	case lower == "current" || lower == "root":
		return fields[0] + " " + lower, true
	case navQuotedString(frame):
		return fields[0] + " " + frame, true
	}

	return "", false
}

// navIDRune reports whether r may appear in an #id token.
func navIDRune(r rune) bool {
	return r == '-' || r == '_' || r >= 0x80 || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9')
}

// navIDToken reports whether token is #ident.
func navIDToken(token string) bool {
	if len(token) < 2 || token[0] != '#' {
		return false
	}

	for _, r := range token[1:] {
		if !navIDRune(r) {
			return false
		}
	}

	return true
}

// willChangeName normalizes one will-change feature name. The keywords that
// spec forbids as a feature name are rejected.
func willChangeName(part string) (string, bool) {
	name := strings.TrimSpace(part)
	if name == "" || strings.ContainsAny(name, " \t\n(") {
		return "", false
	}

	reserved := []string{
		"auto", "none", "all", declPropWillChange, "initial", "inherit", "unset", "revert", stringSetRevertLayer,
		"default",
	}
	if slices.Contains(reserved, strings.ToLower(name)) || !navIdent(name) {
		return "", false
	}

	return strings.ToLower(name), true
}

// normalizeWillChange accepts auto or a comma list of property names.
func normalizeWillChange(text string) (string, bool) {
	if strings.EqualFold(text, "auto") {
		return "auto", true
	}

	items := strings.Split(text, ",")
	out := make([]string, 0, len(items))

	for _, part := range items {
		name, ok := willChangeName(part)
		if !ok {
			return "", false
		}

		out = append(out, name)
	}

	return strings.Join(out, ", "), len(out) > 0
}

// navIdentRune reports whether r may sit at byte offset pos of an identifier.
func navIdentRune(pos int, r rune) bool {
	if r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
		return true
	}

	return pos > 0 && r >= '0' && r <= '9'
}

// navIdent reports whether text is a plain identifier (a property name such as
// transform or --custom).
func navIdent(text string) bool {
	for pos, r := range text {
		if !navIdentRune(pos, r) {
			return false
		}
	}

	return text != ""
}

// normalizeTextSizeAdjust accepts auto | none | <percentage [0,inf]>.
func normalizeTextSizeAdjust(text string) (string, bool) {
	lower := strings.ToLower(text)
	if lower == "auto" || lower == "none" {
		return lower, true
	}

	if !strings.HasSuffix(lower, "%") {
		return "", false
	}

	n, err := strconv.ParseFloat(strings.TrimSuffix(lower, "%"), 64)
	if err != nil || n < 0 || math.IsNaN(n) || math.IsInf(n, 0) {
		return "", false
	}

	return strconv.FormatFloat(n, 'f', -1, 64) + "%", true
}
