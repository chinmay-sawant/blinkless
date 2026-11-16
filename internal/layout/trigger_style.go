package layout

import (
	"math"
	"strconv"
	"strings"
)

// Declarations for scroll and pointer timeline scope, timeline triggers, event
// triggers, interest delay, and interactivity (CSS Scroll-driven Animations 1,
// CSS Animation Triggers 1, CSS UI 4).
//
// The engine owns no clock, no scroll offset, no pointer, and no event loop.
// It parses and validates each declaration, stores the cascaded value on
// ResolvedStyle.Triggers as a normalized string, and exports it on the public
// box. A host reads the export and computes progress or reacts to events.
//
// Every stored value is a normalized string so the style interning generator
// can compare and hash it. The empty string is always the property's initial
// value. The encodings are internal; trigger_export.go decodes them.
const (
	propTimelineScope                        = "timeline-scope"
	propPointerTimeline                      = "pointer-timeline"
	propPointerTimelineName                  = "pointer-timeline-name"
	propPointerTimelineAxis                  = "pointer-timeline-axis"
	propTimelineTrigger                      = "timeline-trigger"
	propTimelineTriggerName                  = "timeline-trigger-name"
	propTimelineTriggerSource                = "timeline-trigger-source"
	propTimelineTriggerActivationRange       = "timeline-trigger-activation-range"
	propTimelineTriggerActivationRangeStart  = "timeline-trigger-activation-range-start"
	propTimelineTriggerActivationRangeEnd    = "timeline-trigger-activation-range-end"
	propTimelineTriggerActiveRange           = "timeline-trigger-active-range"
	propTimelineTriggerActiveRangeStart      = "timeline-trigger-active-range-start"
	propTimelineTriggerActiveRangeEnd        = "timeline-trigger-active-range-end"
	propEventTrigger                         = "event-trigger"
	propEventTriggerName                     = "event-trigger-name"
	propEventTriggerSource                   = "event-trigger-source"
	propInterestDelay                        = "interest-delay"
	propInterestDelayStart                   = "interest-delay-start"
	propInterestDelayEnd                     = "interest-delay-end"
	propInteractivity                        = "interactivity"
	triggerKeywordNormal                     = "normal"
	triggerKeywordAll                        = "all"
	triggerKeywordInert                      = "inert"
	triggerRangeEdgeFields                   = 3
	triggerNamedEdgeTokens                   = 2
	triggerRangePlaceholder                  = "-"
	triggerShorthandRangeSeparator           = "/"
	triggerListSeparator                     = ", "
	triggerPercentBase                       = 100.0
	triggerValueEpsilon                      = 1e-9
	triggerNominalFontSize                   = 12.0
	triggerInterestShorthandMaxTokens        = 2
	triggerTimelineSourceDefault             = "auto"
	triggerRangeAuto                         = "auto"
	triggerEventSlash                        = "/"
	triggerEventSlashSpaced                  = " / "
	triggerShorthandSourceHeadTokens         = 1
	triggerShorthandMinTokensWithSourceRange = 2
)

// triggerStyle holds the cascaded values of the scope, pointer, trigger,
// interest, and interactivity properties. The empty string is the initial
// value of every field. Only InterestStart, InterestEnd, and Interactivity
// inherit.
type triggerStyle struct {
	// TimelineScope is "" for none, "all", or a comma list of dashed idents.
	TimelineScope string
	// PointerName is the pointer-timeline-name dashed ident, "" for none.
	PointerName string
	// PointerAxis is block, inline, x, or y; "" means the initial block.
	PointerAxis string
	// TriggerName is a comma list of dashed idents, "" for none.
	TriggerName string
	// TriggerSource is a comma list of source items; "" means auto.
	TriggerSource string
	// ActivationStart and ActivationEnd are comma lists of range edges.
	ActivationStart string
	ActivationEnd   string
	// ActiveStart and ActiveEnd are comma lists of range edges.
	ActiveStart string
	ActiveEnd   string
	// EventName is a comma list of dashed idents, "" for none.
	EventName string
	// EventSource is a comma list of "start [/ end]" event items; "" is none.
	EventSource string
	// InterestStart and InterestEnd are milliseconds as text; "" is normal.
	InterestStart string
	InterestEnd   string
	// Interactivity is "inert" or "" for auto.
	Interactivity string
}

// triggerField returns the stored field a longhand property owns, or nil.
func triggerField(store *triggerStyle, prop string) *string {
	if field := triggerTimelineField(store, prop); field != nil {
		return field
	}

	if field := triggerEventField(store, prop); field != nil {
		return field
	}

	return triggerUIField(store, prop)
}

// triggerTimelineField covers the scope, pointer, and timeline trigger fields.
func triggerTimelineField(store *triggerStyle, prop string) *string {
	switch prop {
	case propTimelineScope:
		return &store.TimelineScope
	case propPointerTimelineName:
		return &store.PointerName
	case propPointerTimelineAxis:
		return &store.PointerAxis
	case propTimelineTriggerName:
		return &store.TriggerName
	case propTimelineTriggerSource:
		return &store.TriggerSource
	default:
		return triggerRangeField(store, prop)
	}
}

// triggerRangeField covers the four range edge fields.
func triggerRangeField(store *triggerStyle, prop string) *string {
	switch prop {
	case propTimelineTriggerActivationRangeStart:
		return &store.ActivationStart
	case propTimelineTriggerActivationRangeEnd:
		return &store.ActivationEnd
	case propTimelineTriggerActiveRangeStart:
		return &store.ActiveStart
	case propTimelineTriggerActiveRangeEnd:
		return &store.ActiveEnd
	default:
		return nil
	}
}

// triggerEventField covers the event trigger fields.
func triggerEventField(store *triggerStyle, prop string) *string {
	switch prop {
	case propEventTriggerName:
		return &store.EventName
	case propEventTriggerSource:
		return &store.EventSource
	default:
		return nil
	}
}

// triggerUIField covers the interest delay and interactivity fields.
func triggerUIField(store *triggerStyle, prop string) *string {
	switch prop {
	case propInterestDelayStart:
		return &store.InterestStart
	case propInterestDelayEnd:
		return &store.InterestEnd
	case propInteractivity:
		return &store.Interactivity
	default:
		return nil
	}
}

// triggerLonghandInherits reports whether a longhand inherits by default.
func triggerLonghandInherits(prop string) bool {
	switch prop {
	case propInterestDelayStart, propInterestDelayEnd, propInteractivity:
		return true
	default:
		return false
	}
}

// triggerShorthandProperty reports whether prop is one of the shorthands this
// file expands into longhands.
func triggerShorthandProperty(prop string) bool {
	switch prop {
	case propPointerTimeline, propTimelineTrigger, propTimelineTriggerActivationRange,
		propTimelineTriggerActiveRange, propEventTrigger, propInterestDelay:
		return true
	default:
		return false
	}
}

// triggerProperty reports whether prop is owned by this file.
func triggerProperty(prop string) bool {
	if triggerShorthandProperty(prop) {
		return true
	}

	var probe triggerStyle

	return triggerField(&probe, prop) != nil
}

// applyTriggerProps resolves the properties this file owns. It is reached
// from applyTimelineProps. A value that fails its grammar leaves the previous
// value, as every other handler here does.
func applyTriggerProps(
	style *ResolvedStyle, prop, value string, fsize float64, parent *ResolvedStyle, hasParent bool,
) bool {
	switch prop {
	case propTimelineScope, propPointerTimelineName, propPointerTimelineAxis,
		propTimelineTriggerName, propTimelineTriggerSource,
		propTimelineTriggerActivationRangeStart, propTimelineTriggerActivationRangeEnd,
		propTimelineTriggerActiveRangeStart, propTimelineTriggerActiveRangeEnd,
		propEventTriggerName, propEventTriggerSource,
		propInterestDelayStart, propInterestDelayEnd, propInteractivity:
		applyTriggerLonghand(style, prop, value, fsize, parent, hasParent)
	case propPointerTimeline, propTimelineTrigger, propTimelineTriggerActivationRange,
		propTimelineTriggerActiveRange, propEventTrigger, propInterestDelay:
		applyTriggerShorthand(style, prop, value, fsize, parent, hasParent)
	default:
		return false
	}

	return true
}

// applyTriggerLonghand stores one longhand. The CSS-wide keywords resolve
// here: inherit takes the parent value, unset takes it only for an inherited
// property, and every other wide keyword resets to the initial value.
func applyTriggerLonghand(
	style *ResolvedStyle, prop, value string, fsize float64, parent *ResolvedStyle, hasParent bool,
) {
	field := triggerField(&style.Triggers, prop)
	if field == nil {
		return
	}

	trimmed := strings.TrimSpace(value)
	lower := strings.ToLower(trimmed)

	if cssWideKeyword(lower) {
		*field = triggerWideValue(lower, prop, parent, hasParent)

		return
	}

	if normalized, ok := normalizeTriggerLonghand(prop, trimmed, fsize); ok {
		*field = normalized
	}
}

// triggerWideValue resolves a CSS-wide keyword for one longhand.
func triggerWideValue(keyword, prop string, parent *ResolvedStyle, hasParent bool) string {
	takeParent := keyword == inheritKeyword || (keyword == cssKeywordUnset && triggerLonghandInherits(prop))
	if !takeParent || !hasParent || parent == nil {
		return ""
	}

	if field := triggerField(&parent.Triggers, prop); field != nil {
		return *field
	}

	return ""
}

// applyTriggerShorthand expands a shorthand that reached the handler through
// var() substitution and applies each longhand.
func applyTriggerShorthand(
	style *ResolvedStyle, prop, value string, fsize float64, parent *ResolvedStyle, hasParent bool,
) {
	declarations, ok := expandTriggerDeclaration(prop, value, fsize)
	if !ok {
		return
	}

	for _, declaration := range declarations {
		applyTriggerLonghand(style, declaration.prop, declaration.val, fsize, parent, hasParent)
	}
}

// triggerShorthandLonghands lists the longhands a shorthand resets, in the
// order the grammar names them.
func triggerShorthandLonghands(prop string) []string {
	switch prop {
	case propPointerTimeline:
		return []string{propPointerTimelineName, propPointerTimelineAxis}
	case propTimelineTrigger:
		return []string{
			propTimelineTriggerName, propTimelineTriggerSource,
			propTimelineTriggerActivationRangeStart, propTimelineTriggerActivationRangeEnd,
			propTimelineTriggerActiveRangeStart, propTimelineTriggerActiveRangeEnd,
		}
	case propTimelineTriggerActivationRange:
		return []string{propTimelineTriggerActivationRangeStart, propTimelineTriggerActivationRangeEnd}
	case propTimelineTriggerActiveRange:
		return []string{propTimelineTriggerActiveRangeStart, propTimelineTriggerActiveRangeEnd}
	case propEventTrigger:
		return []string{propEventTriggerName, propEventTriggerSource}
	case propInterestDelay:
		return []string{propInterestDelayStart, propInterestDelayEnd}
	default:
		return nil
	}
}

// expandTriggerDeclaration expands a winning shorthand into longhand
// declarations in grammar order. It returns false for a property it does not
// own and for a value that fails the grammar, so an invalid shorthand
// changes nothing. A CSS-wide keyword fans out to every longhand.
func expandTriggerDeclaration(prop, value string, fsize float64) ([]logicalPropDecl, bool) {
	longhands := triggerShorthandLonghands(prop)
	if longhands == nil {
		return nil, false
	}

	trimmed := strings.TrimSpace(value)
	if cssWideKeyword(strings.ToLower(trimmed)) {
		out := make([]logicalPropDecl, len(longhands))
		for idx, longhand := range longhands {
			out[idx] = logicalPropDecl{prop: longhand, val: trimmed}
		}

		return out, true
	}

	values, ok := triggerShorthandValues(prop, trimmed, fsize)
	if !ok || len(values) != len(longhands) {
		return nil, false
	}

	out := make([]logicalPropDecl, len(longhands))
	for idx, longhand := range longhands {
		out[idx] = logicalPropDecl{prop: longhand, val: values[idx]}
	}

	return out, true
}

// triggerShorthandValues parses a shorthand into one text value per longhand.
func triggerShorthandValues(prop, value string, fsize float64) ([]string, bool) {
	switch prop {
	case propPointerTimeline:
		name, axis, ok := parseScrollTimelineShorthand(value)
		if !ok {
			return nil, false
		}

		return []string{name, axis}, true
	case propTimelineTrigger:
		return parseTimelineTriggerShorthand(value, fsize)
	case propTimelineTriggerActivationRange:
		return parseRangePairList(value, false, fsize)
	case propTimelineTriggerActiveRange:
		return parseRangePairList(value, true, fsize)
	case propEventTrigger:
		return parseEventTriggerShorthand(value)
	default: // interest-delay
		return parseInterestDelayShorthand(value)
	}
}

// normalizeTriggerLonghand validates one longhand value and returns its
// stored form.
func normalizeTriggerLonghand(prop, value string, fsize float64) (string, bool) {
	switch prop {
	case propTimelineScope:
		return normalizeTimelineScope(value)
	case propPointerTimelineName:
		return parseTimelineName(value)
	case propPointerTimelineAxis:
		return parseTimelineAxis(value)
	case propTimelineTriggerName, propEventTriggerName:
		return normalizeTriggerNames(value)
	case propTimelineTriggerSource:
		return normalizeTimelineSources(value)
	default:
		return normalizeTriggerValueLonghand(prop, value, fsize)
	}
}

// normalizeTriggerValueLonghand covers the range, event source, interest
// delay, and interactivity longhands.
func normalizeTriggerValueLonghand(prop, value string, fsize float64) (string, bool) {
	switch prop {
	case propTimelineTriggerActivationRangeStart, propTimelineTriggerActivationRangeEnd:
		return normalizeRangeEdgeList(value, false, fsize)
	case propTimelineTriggerActiveRangeStart, propTimelineTriggerActiveRangeEnd:
		return normalizeRangeEdgeList(value, true, fsize)
	case propEventTriggerSource:
		return normalizeEventSources(value)
	case propInterestDelayStart, propInterestDelayEnd:
		return normalizeInterestTime(value)
	case propInteractivity:
		return normalizeInteractivity(value)
	default:
		return "", false
	}
}

// triggerDeclarationValueAccepted is the cascade gate. It reports (accepted,
// handled); handled is false for a property this file does not own. A wide
// keyword and a var() reference are valid at parse time.
func triggerDeclarationValueAccepted(prop, value string) (bool, bool) {
	if !triggerProperty(prop) {
		return false, false
	}

	trimmed := strings.TrimSpace(value)
	if cssWideKeyword(strings.ToLower(trimmed)) || containsVarFunc(trimmed) {
		return true, true
	}

	if triggerShorthandProperty(prop) {
		_, ok := expandTriggerDeclaration(prop, trimmed, triggerNominalFontSize)

		return ok, true
	}

	_, ok := normalizeTriggerLonghand(prop, trimmed, triggerNominalFontSize)

	return ok, true
}

// splitStrictList splits a comma list and rejects an empty item, a leading
// comma, and a trailing comma.
func splitStrictList(value string) ([]string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.HasSuffix(trimmed, ",") {
		return nil, false
	}

	parts := splitParenArgs(trimmed, ',')
	if len(parts) == 0 {
		return nil, false
	}

	for _, part := range parts {
		if part == "" {
			return nil, false
		}
	}

	return parts, true
}

// normalizeTimelineScope reads none | all | <dashed-ident>#.
func normalizeTimelineScope(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case timelineNoneKeyword:
		return "", true
	case triggerKeywordAll:
		return triggerKeywordAll, true
	}

	return normalizeDashedList(value)
}

// normalizeDashedList reads a comma list of dashed idents.
func normalizeDashedList(value string) (string, bool) {
	parts, ok := splitStrictList(value)
	if !ok {
		return "", false
	}

	for _, part := range parts {
		if !animationDashedIdent(part) {
			return "", false
		}
	}

	return strings.Join(parts, triggerListSeparator), true
}

// normalizeTriggerNames reads none | <dashed-ident>#. none cannot join a list.
func normalizeTriggerNames(value string) (string, bool) {
	if strings.EqualFold(strings.TrimSpace(value), timelineNoneKeyword) {
		return "", true
	}

	return normalizeDashedList(value)
}

// normalizeTimelineSources reads [ none | auto | [ <dashed-ident> | scroll() |
// view() ]+ ]#.
func normalizeTimelineSources(value string) (string, bool) {
	parts, ok := splitStrictList(value)
	if !ok {
		return "", false
	}

	items := make([]string, len(parts))

	for idx, part := range parts {
		item, valid := normalizeTimelineSourceItem(part)
		if !valid {
			return "", false
		}

		items[idx] = item
	}

	return strings.Join(items, triggerListSeparator), true
}

// normalizeTimelineSourceItem reads one source item: none, auto, or one or
// more timeline references.
func normalizeTimelineSourceItem(item string) (string, bool) {
	tokens, ok := animationSpaceTokens(item)
	if !ok || len(tokens) == 0 {
		return "", false
	}

	if len(tokens) == 1 {
		switch lower := strings.ToLower(tokens[0]); lower {
		case timelineNoneKeyword, triggerTimelineSourceDefault:
			return lower, true
		}
	}

	for _, token := range tokens {
		if !timelineReferenceToken(token) {
			return "", false
		}
	}

	return strings.Join(tokens, " "), true
}

// timelineReferenceToken reports whether token is a dashed ident or a
// scroll() or view() function with valid arguments.
func timelineReferenceToken(token string) bool {
	if animationDashedIdent(token) {
		return true
	}

	name, _, ok := animationFunctionParts(token)

	return ok && (name == timelineKindScroll || name == timelineKindView) &&
		animationTimelineFunctionValueAccepted(token)
}

// normalizeRangeEdgeList reads a comma list where each item is exactly one
// range edge.
func normalizeRangeEdgeList(value string, allowAuto bool, fsize float64) (string, bool) {
	parts, ok := splitStrictList(value)
	if !ok {
		return "", false
	}

	edges := make([]string, len(parts))

	for idx, part := range parts {
		tokens, valid := animationSpaceTokens(part)
		if !valid {
			return "", false
		}

		edge, used, valid := parseRangeEdge(tokens, allowAuto, fsize)
		if !valid || used != len(tokens) {
			return "", false
		}

		edges[idx] = edge
	}

	return strings.Join(edges, triggerListSeparator), true
}

// parseRangePairList reads [ <start> <end>? ]# for the two range shorthands
// and returns the start list and the end list. A missing end is the
// property's initial value: normal for activation, auto for active.
func parseRangePairList(value string, allowAuto bool, fsize float64) ([]string, bool) {
	parts, ok := splitStrictList(value)
	if !ok {
		return nil, false
	}

	starts := make([]string, len(parts))
	ends := make([]string, len(parts))

	for idx, part := range parts {
		tokens, valid := animationSpaceTokens(part)
		if !valid {
			return nil, false
		}

		start, end, valid := parseRangePair(tokens, allowAuto, fsize)
		if !valid {
			return nil, false
		}

		starts[idx], ends[idx] = start, end
	}

	return []string{strings.Join(starts, triggerListSeparator), strings.Join(ends, triggerListSeparator)}, true
}

// parseRangePair reads one start edge and an optional end edge that together
// use every token. It returns the source text of each edge, so the values can
// be applied through the longhand parsers; a missing end is the property's
// initial value, normal for activation and auto for active.
func parseRangePair(tokens []string, allowAuto bool, fsize float64) (string, string, bool) {
	_, used, valid := parseRangeEdge(tokens, allowAuto, fsize)
	if !valid {
		return "", "", false
	}

	start := strings.Join(tokens[:used], " ")

	rest := tokens[used:]
	if len(rest) == 0 {
		return start, triggerRangeInitial(allowAuto), true
	}

	_, usedEnd, valid := parseRangeEdge(rest, allowAuto, fsize)
	if !valid || usedEnd != len(rest) {
		return "", "", false
	}

	return start, strings.Join(rest, " "), true
}

// triggerRangeInitial is the initial range edge: auto for the active range,
// normal for the activation range.
func triggerRangeInitial(allowAuto bool) string {
	if allowAuto {
		return triggerRangeAuto
	}

	return triggerKeywordNormal
}

// parseRangeEdge reads one range edge from the front of tokens and returns
// its stored encoding and the number of tokens it used. The encoding is
// normal, auto, or "<name|-> <percent|-> <px|->": a timeline range name, a
// percentage as a plain number, and a length in CSS pixels.
func parseRangeEdge(tokens []string, allowAuto bool, fsize float64) (string, int, bool) {
	if len(tokens) == 0 {
		return "", 0, false
	}

	first := strings.ToLower(tokens[0])

	switch {
	case first == triggerKeywordNormal:
		return triggerKeywordNormal, 1, true
	case allowAuto && first == triggerRangeAuto:
		return triggerRangeAuto, 1, true
	case animationTimelineRangeName(first):
		return parseNamedRangeEdge(first, tokens, fsize)
	case animationLengthPercentageValueAccepted(tokens[0]):
		percent, length, valid := rangeOffset(tokens[0], fsize)
		if !valid {
			return "", 0, false
		}

		return triggerRangePlaceholder + " " + percent + " " + length, 1, true
	}

	return "", 0, false
}

// parseNamedRangeEdge reads a timeline range name and its optional offset.
func parseNamedRangeEdge(name string, tokens []string, fsize float64) (string, int, bool) {
	if len(tokens) > 1 && animationLengthPercentageValueAccepted(tokens[1]) {
		percent, length, valid := rangeOffset(tokens[1], fsize)
		if !valid {
			return "", 0, false
		}

		return name + " " + percent + " " + length, triggerNamedEdgeTokens, true
	}

	return name + " " + triggerRangePlaceholder + " " + triggerRangePlaceholder, 1, true
}

// rangeOffset splits a length-percentage into a percent number and a length
// in CSS pixels. Either is "-" when the value has no such part. A calc()
// that mixes both is split by evaluating it at two container sizes, which is
// exact for sums of a length and a percentage.
func rangeOffset(token string, fsize float64) (string, string, bool) {
	trimmed := strings.TrimSpace(token)

	if number, isPercent := strings.CutSuffix(trimmed, "%"); isPercent {
		if value, ok := cssNumberValue(number); ok {
			return formatTriggerNumber(value), triggerRangePlaceholder, true
		}
	}

	atZero, okZero := triggerLengthPoints(trimmed, fsize, 0)
	atBase, okBase := triggerLengthPoints(trimmed, fsize, triggerPercentBase)

	if !okZero || !okBase {
		return "", "", false
	}

	percent, length := triggerRangePlaceholder, triggerRangePlaceholder

	if delta := atBase - atZero; math.Abs(delta) > triggerValueEpsilon {
		percent = formatTriggerNumber(delta)
	}

	if math.Abs(atZero) > triggerValueEpsilon || percent == triggerRangePlaceholder {
		length = formatTriggerNumber(atZero / pxToPt(1))
	}

	return percent, length, true
}

// triggerLengthPoints resolves a length-percentage to layout points against a
// container size.
func triggerLengthPoints(token string, fsize, containing float64) (float64, bool) {
	if points, ok := calcLength(token, fsize, containing); ok {
		return points, true
	}

	return plainLength(token, fsize, containing)
}

// formatTriggerNumber writes a number without a trailing exponent or zeros.
func formatTriggerNumber(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}

// parseTimelineTriggerShorthand reads none | [ <name> <source>
// <activation-range> [ / <active-range> ]? ]# into one value per longhand:
// name, source, activation start and end, active start and end.
func parseTimelineTriggerShorthand(value string, fsize float64) ([]string, bool) {
	if strings.EqualFold(strings.TrimSpace(value), timelineNoneKeyword) {
		return []string{timelineNoneKeyword, "initial", "initial", "initial", "initial", "initial"}, true
	}

	parts, ok := splitStrictList(value)
	if !ok {
		return nil, false
	}

	columns := make([][]string, 6) //nolint:mnd // name, source, four range edges

	for _, part := range parts {
		item, valid := parseTimelineTriggerItem(part, fsize)
		if !valid {
			return nil, false
		}

		for col := range columns {
			columns[col] = append(columns[col], item[col])
		}
	}

	out := make([]string, len(columns))
	for col := range columns {
		out[col] = strings.Join(columns[col], triggerListSeparator)
	}

	return out, true
}

// parseTimelineTriggerItem reads one comma item of the timeline-trigger
// shorthand and returns its six longhand texts.
func parseTimelineTriggerItem(item string, fsize float64) ([6]string, bool) {
	var out [6]string

	tokens := tokenizeWithSlash(item)
	if len(tokens) < triggerShorthandMinTokensWithSourceRange || !animationDashedIdent(tokens[0]) {
		return out, false
	}

	out[0] = tokens[0]
	rest := tokens[1:]

	sourceLen := timelineSourceTokenCount(rest)
	if sourceLen == 0 {
		return out, false
	}

	source, valid := normalizeTimelineSourceItem(strings.Join(rest[:sourceLen], " "))
	if !valid {
		return out, false
	}

	out[1] = source

	ranges, valid := parseTriggerRanges(rest[sourceLen:], fsize)
	if !valid {
		return out, false
	}

	copy(out[2:], ranges[:])

	return out, true
}

// parseTriggerRanges reads [ <activation-range> ]? [ / <active-range> ]? and
// returns the activation start and end then the active start and end. A
// missing active range is auto.
func parseTriggerRanges(tokens []string, fsize float64) ([4]string, bool) {
	out := [4]string{"", "", triggerRangeAuto, triggerRangeAuto}
	activation, active := tokens, []string(nil)

	if slash := indexOfToken(tokens, triggerShorthandRangeSeparator); slash >= 0 {
		activation, active = tokens[:slash], tokens[slash+1:]
		if len(active) == 0 {
			return out, false
		}
	}

	if len(activation) == 0 {
		// The activation range is optional here; omitted means normal.
		out[0], out[1] = triggerKeywordNormal, triggerKeywordNormal

		return out, active == nil
	}

	var valid bool

	out[0], out[1], valid = parseRangePair(activation, false, fsize)
	if !valid {
		return out, false
	}

	if active != nil {
		out[2], out[3], valid = parseRangePair(active, true, fsize)
	}

	return out, valid
}

// timelineSourceTokenCount counts the leading tokens that form one source
// item: a single none or auto, or a run of timeline references.
func timelineSourceTokenCount(tokens []string) int {
	if len(tokens) == 0 {
		return 0
	}

	switch strings.ToLower(tokens[0]) {
	case timelineNoneKeyword, triggerTimelineSourceDefault:
		return triggerShorthandSourceHeadTokens
	}

	count := 0

	for _, token := range tokens {
		if !timelineReferenceToken(token) {
			break
		}

		count++
	}

	return count
}

// indexOfToken returns the index of the first token equal to target, or -1.
func indexOfToken(tokens []string, target string) int {
	for idx, token := range tokens {
		if token == target {
			return idx
		}
	}

	return -1
}

// tokenizeWithSlash splits on whitespace outside parentheses and makes every
// slash outside parentheses its own token.
func tokenizeWithSlash(value string) []string {
	var spaced strings.Builder

	depth := 0

	for _, char := range value {
		switch {
		case char == '(':
			depth++
		case char == ')' && depth > 0:
			depth--
		}

		if char == '/' && depth == 0 {
			spaced.WriteString(" / ")

			continue
		}

		spaced.WriteRune(char)
	}

	tokens, ok := animationSpaceTokens(spaced.String())
	if !ok {
		return nil
	}

	return tokens
}
