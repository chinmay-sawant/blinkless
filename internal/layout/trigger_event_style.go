package layout

import (
	"strings"
)

// Event triggers, interest delay, and interactivity. They share the storage
// and the handler in trigger_style.go.

// normalizeEventSources reads [ none | <event-trigger-event>+ [ /
// <event-trigger-event>+ ]? ]#. The draft leaves the event list open, so an
// event type is any identifier, or an identifier with a balanced argument
// list such as keydown(Enter). The stored form of one item is "start events"
// or "start events / end events"; none is stored as none.
func normalizeEventSources(value string) (string, bool) {
	parts, ok := splitStrictList(value)
	if !ok {
		return "", false
	}

	items := make([]string, len(parts))

	for idx, part := range parts {
		item, valid := normalizeEventSourceItem(part)
		if !valid {
			return "", false
		}

		items[idx] = item
	}

	return strings.Join(items, triggerListSeparator), true
}

// normalizeEventSourceItem reads one source item.
func normalizeEventSourceItem(item string) (string, bool) {
	tokens := tokenizeWithSlash(item)
	if len(tokens) == 0 {
		return "", false
	}

	if len(tokens) == 1 && strings.EqualFold(tokens[0], timelineNoneKeyword) {
		return timelineNoneKeyword, true
	}

	slash := indexOfToken(tokens, triggerEventSlash)
	if slash < 0 {
		return joinEventTypes(tokens)
	}

	if slash == 0 || slash == len(tokens)-1 || indexOfToken(tokens[slash+1:], triggerEventSlash) >= 0 {
		return "", false
	}

	start, startOK := joinEventTypes(tokens[:slash])
	end, endOK := joinEventTypes(tokens[slash+1:])

	if !startOK || !endOK {
		return "", false
	}

	return start + triggerEventSlashSpaced + end, true
}

// joinEventTypes validates event type tokens and joins them with spaces.
func joinEventTypes(tokens []string) (string, bool) {
	for _, token := range tokens {
		if !eventTypeToken(token) {
			return "", false
		}
	}

	return strings.Join(tokens, " "), true
}

// eventTypeToken reports whether token is an identifier, optionally followed
// by a balanced argument list. The CSS-wide keywords and none are not event
// types.
func eventTypeToken(token string) bool {
	name := token

	if open := strings.IndexByte(token, '('); open >= 0 {
		if _, _, valid := animationFunctionParts(token); !valid {
			return false
		}

		name = token[:open]
	}

	if cssWideKeyword(strings.ToLower(name)) || strings.EqualFold(name, timelineNoneKeyword) {
		return false
	}

	return eventTypeIdent(name)
}

// eventTypeIdent reports whether name is a letter followed by letters,
// digits, hyphens, or underscores.
func eventTypeIdent(name string) bool {
	if name == "" || !asciiLetter(name[0]) {
		return false
	}

	for pos := 1; pos < len(name); pos++ {
		if char := name[pos]; !asciiLetter(char) && !asciiIdentRest(char) {
			return false
		}
	}

	return true
}

// asciiLetter reports whether char is an ASCII letter.
func asciiLetter(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
}

// asciiIdentRest reports whether char is a digit, hyphen, or underscore.
func asciiIdentRest(char byte) bool {
	return char >= '0' && char <= '9' || char == '-' || char == '_'
}

// parseEventTriggerShorthand reads none | [ <name> <source> ]# into the name
// list and the source list.
func parseEventTriggerShorthand(value string) ([]string, bool) {
	if strings.EqualFold(strings.TrimSpace(value), timelineNoneKeyword) {
		return []string{timelineNoneKeyword, "initial"}, true
	}

	parts, ok := splitStrictList(value)
	if !ok {
		return nil, false
	}

	names := make([]string, len(parts))
	sources := make([]string, len(parts))

	for idx, part := range parts {
		trimmed := strings.TrimSpace(part)

		space := strings.IndexAny(trimmed, " \t\n\r\f")
		if space < 0 || !animationDashedIdent(trimmed[:space]) {
			return nil, false
		}

		source, valid := normalizeEventSourceItem(trimmed[space:])
		if !valid {
			return nil, false
		}

		names[idx], sources[idx] = trimmed[:space], source
	}

	return []string{
		strings.Join(names, triggerListSeparator),
		strings.Join(sources, triggerListSeparator),
	}, true
}

// normalizeInterestTime reads normal | <time>. normal is stored as the empty
// string. A time is stored as milliseconds. A negative time is refused, since
// a delay cannot end before the pointer arrives.
func normalizeInterestTime(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if strings.EqualFold(trimmed, triggerKeywordNormal) {
		return "", true
	}

	millis, ok := cssTimeMilliseconds(trimmed, false)
	if !ok {
		return "", false
	}

	return formatTriggerNumber(millis), true
}

// parseInterestDelayShorthand reads <'interest-delay-start'>{1,2}. One value
// sets both delays.
func parseInterestDelayShorthand(value string) ([]string, bool) {
	tokens, ok := animationSpaceTokens(value)
	if !ok || len(tokens) == 0 || len(tokens) > triggerInterestShorthandMaxTokens {
		return nil, false
	}

	for _, token := range tokens {
		if _, valid := normalizeInterestTime(token); !valid {
			return nil, false
		}
	}

	if len(tokens) == 1 {
		return []string{tokens[0], tokens[0]}, true
	}

	return []string{tokens[0], tokens[1]}, true
}

// normalizeInteractivity reads auto | inert. auto is stored as the empty
// string, the initial value.
func normalizeInteractivity(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "auto":
		return "", true
	case triggerKeywordInert:
		return triggerKeywordInert, true
	default:
		return "", false
	}
}

// inheritTriggerProps copies the inherited properties of this family from the
// parent unless the element declares them: the two interest delays and
// interactivity. A declared value, including a CSS-wide keyword, is applied
// later by applyTriggerLonghand.
func inheritTriggerProps(dst, parent *ResolvedStyle, raw map[string]string) {
	_, shorthand := raw[propInterestDelay]

	if _, ok := raw[propInterestDelayStart]; !ok && !shorthand {
		dst.Triggers.InterestStart = parent.Triggers.InterestStart
	}

	if _, ok := raw[propInterestDelayEnd]; !ok && !shorthand {
		dst.Triggers.InterestEnd = parent.Triggers.InterestEnd
	}

	if _, ok := raw[propInteractivity]; !ok {
		dst.Triggers.Interactivity = parent.Triggers.Interactivity
	}
}
