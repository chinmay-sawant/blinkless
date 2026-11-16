package layout

import (
	"strconv"
	"strings"
)

// Engine-side payloads for the trigger family. layout/trigger.go and
// layout/interest.go map them to the public Box fields. Lengths are CSS
// pixels and times are milliseconds, the units the public API documents.

// TimelineScopeDef is the computed timeline-scope of one element.
type TimelineScopeDef struct {
	// None is true when the element exposes no names, the initial value.
	None bool
	// All is true for timeline-scope: all.
	All bool
	// Names lists the dashed idents of a named scope.
	Names []string
}

// RangeBoundDef is one range edge of a timeline trigger.
type RangeBoundDef struct {
	// Normal is true for the normal keyword.
	Normal bool
	// Auto is true for the auto keyword, legal only on an active range.
	Auto bool
	// Name is the timeline range name (cover, contain, entry, exit,
	// entry-crossing, exit-crossing), or empty.
	Name string
	// Percent is the percentage part; valid when HasPercent is true.
	Percent float64
	// Length is the length part in CSS pixels; valid when HasLength is true.
	Length     float64
	HasPercent bool
	HasLength  bool
}

// TimelineTriggerDef is one timeline trigger an element declares.
type TimelineTriggerDef struct {
	// Name is the dashed ident of the trigger.
	Name string
	// Source is the raw timeline reference text: scroll(), view(), a dashed
	// ident, none, or auto. Several references are joined with a space.
	Source string
	// Sources is Source split into single references.
	Sources []string
	// ActivationStart and ActivationEnd bound the activation range.
	ActivationStart RangeBoundDef
	ActivationEnd   RangeBoundDef
	// ActiveStart and ActiveEnd bound the active range.
	ActiveStart RangeBoundDef
	ActiveEnd   RangeBoundDef
}

// EventTriggerDef is one event trigger an element declares.
type EventTriggerDef struct {
	// Name is the dashed ident of the trigger.
	Name string
	// Start lists the event types that start the trigger.
	Start []string
	// End lists the event types that end the trigger.
	End []string
}

// InterestDelayDef is the computed interest delay of one element.
type InterestDelayDef struct {
	// StartNormal and EndNormal are true for the normal keyword, which means
	// the host picks its own delay.
	StartNormal bool
	EndNormal   bool
	// StartMs and EndMs are the delays in milliseconds when not normal.
	StartMs float64
	EndMs   float64
}

// timelineScopeFromStyle decodes the stored timeline-scope.
func timelineScopeFromStyle(style *ResolvedStyle) TimelineScopeDef {
	if style == nil || style.Triggers.TimelineScope == "" {
		return TimelineScopeDef{None: true, All: false, Names: nil}
	}

	if style.Triggers.TimelineScope == triggerKeywordAll {
		return TimelineScopeDef{None: false, All: true, Names: nil}
	}

	return TimelineScopeDef{
		None:  false,
		All:   false,
		Names: splitParenArgs(style.Triggers.TimelineScope, ','),
	}
}

// pointerTimelineDef returns the pointer timeline an element declares, or
// false when it declares none.
func pointerTimelineDef(style *ResolvedStyle, subjectID string) (TimelineDef, bool) {
	if style == nil || style.Triggers.PointerName == "" {
		return TimelineDef{Name: "", Kind: "", Axis: "", SubjectID: "", ScrollerID: "", InsetStart: 0, InsetEnd: 0}, false
	}

	return TimelineDef{
		Name:       style.Triggers.PointerName,
		Kind:       timelineKindPointer,
		Axis:       timelineUsedAxis(style.Triggers.PointerAxis, style.WritingMode),
		SubjectID:  subjectID,
		ScrollerID: "",
		InsetStart: 0,
		InsetEnd:   0,
	}, true
}

// timelineTriggersFromStyle builds one trigger per declared name. The other
// lists repeat to the length of the name list, as the draft requires. A nil
// result means no trigger.
func timelineTriggersFromStyle(style *ResolvedStyle) []TimelineTriggerDef {
	if style == nil || style.Triggers.TriggerName == "" {
		return nil
	}

	store := &style.Triggers
	names := splitParenArgs(store.TriggerName, ',')
	sources := triggerListOr(store.TriggerSource, triggerTimelineSourceDefault)
	actStart := triggerListOr(store.ActivationStart, triggerKeywordNormal)
	actEnd := triggerListOr(store.ActivationEnd, triggerKeywordNormal)
	activeStart := triggerListOr(store.ActiveStart, triggerRangeAuto)
	activeEnd := triggerListOr(store.ActiveEnd, triggerRangeAuto)

	out := make([]TimelineTriggerDef, len(names))

	for idx, name := range names {
		source := sources[idx%len(sources)]

		out[idx] = TimelineTriggerDef{
			Name:            name,
			Source:          source,
			Sources:         strings.Fields(source),
			ActivationStart: decodeRangeEdge(actStart[idx%len(actStart)]),
			ActivationEnd:   decodeRangeEdge(actEnd[idx%len(actEnd)]),
			ActiveStart:     decodeRangeEdge(activeStart[idx%len(activeStart)]),
			ActiveEnd:       decodeRangeEdge(activeEnd[idx%len(activeEnd)]),
		}
	}

	return out
}

// triggerListOr splits a stored comma list, or returns the one initial item
// when the list is empty.
func triggerListOr(stored, initial string) []string {
	if stored == "" {
		return []string{initial}
	}

	return splitParenArgs(stored, ',')
}

// decodeRangeEdge reads the stored encoding written by parseRangeEdge.
func decodeRangeEdge(edge string) RangeBoundDef {
	switch edge {
	case triggerKeywordNormal:
		return emptyRangeBound(true, false)
	case triggerRangeAuto:
		return emptyRangeBound(false, true)
	}

	fields := strings.Fields(edge)
	if len(fields) != triggerRangeEdgeFields {
		return emptyRangeBound(true, false)
	}

	bound := emptyRangeBound(false, false)

	if fields[0] != triggerRangePlaceholder {
		bound.Name = fields[0]
	}

	value, err := strconv.ParseFloat(fields[1], 64)
	if err == nil && fields[1] != triggerRangePlaceholder {
		bound.Percent, bound.HasPercent = value, true
	}

	value, err = strconv.ParseFloat(fields[2], 64)
	if err == nil && fields[2] != triggerRangePlaceholder {
		bound.Length, bound.HasLength = value, true
	}

	return bound
}

// eventTriggersFromStyle builds one trigger per declared name, repeating the
// source list to the length of the name list. A nil result means no trigger.
func eventTriggersFromStyle(style *ResolvedStyle) []EventTriggerDef {
	if style == nil || style.Triggers.EventName == "" {
		return nil
	}

	names := splitParenArgs(style.Triggers.EventName, ',')
	sources := triggerListOr(style.Triggers.EventSource, timelineNoneKeyword)

	out := make([]EventTriggerDef, len(names))

	for idx, name := range names {
		start, end := decodeEventSource(sources[idx%len(sources)])
		out[idx] = EventTriggerDef{Name: name, Start: start, End: end}
	}

	return out
}

// decodeEventSource splits a stored source item into start and end events.
func decodeEventSource(item string) ([]string, []string) {
	if item == timelineNoneKeyword {
		return nil, nil
	}

	startText, endText, hasEnd := strings.Cut(item, triggerEventSlashSpaced)
	start := eventTypesFrom(startText)

	if !hasEnd {
		return start, nil
	}

	return start, eventTypesFrom(endText)
}

// eventTypesFrom splits event types on whitespace outside parentheses.
func eventTypesFrom(text string) []string {
	tokens, ok := animationSpaceTokens(text)
	if !ok || len(tokens) == 0 {
		return nil
	}

	return tokens
}

// interestDelayFromStyle decodes the stored interest delays.
func interestDelayFromStyle(style *ResolvedStyle) InterestDelayDef {
	def := InterestDelayDef{StartNormal: true, EndNormal: true, StartMs: 0, EndMs: 0}
	if style == nil {
		return def
	}

	if value, err := strconv.ParseFloat(style.Triggers.InterestStart, 64); err == nil {
		def.StartNormal, def.StartMs = false, value
	}

	if value, err := strconv.ParseFloat(style.Triggers.InterestEnd, 64); err == nil {
		def.EndNormal, def.EndMs = false, value
	}

	return def
}

// interactivityFromStyle is the computed interactivity keyword.
func interactivityFromStyle(style *ResolvedStyle) string {
	if style != nil && style.Triggers.Interactivity == triggerKeywordInert {
		return triggerKeywordInert
	}

	return "auto"
}

// PlacedTriggers groups the scope, trigger, interest delay, and
// interactivity values of one placed element.
type PlacedTriggers struct {
	// TimelineScope is the computed timeline-scope.
	TimelineScope TimelineScopeDef
	// TimelineTriggers and EventTriggers hold the declared triggers, nil
	// when the element declares none.
	TimelineTriggers []TimelineTriggerDef
	EventTriggers    []EventTriggerDef
	// InterestDelay is the computed interest delay, inherited.
	InterestDelay InterestDelayDef
	// Interactivity is the computed keyword, auto or inert, inherited.
	Interactivity string
}

// placedTriggersFromStyle decodes every trigger-family value of one style.
func placedTriggersFromStyle(style *ResolvedStyle) PlacedTriggers {
	return PlacedTriggers{
		TimelineScope:    timelineScopeFromStyle(style),
		TimelineTriggers: timelineTriggersFromStyle(style),
		EventTriggers:    eventTriggersFromStyle(style),
		InterestDelay:    interestDelayFromStyle(style),
		Interactivity:    interactivityFromStyle(style),
	}
}

// emptyRangeBound is a range edge with no name and no offset.
func emptyRangeBound(normal, auto bool) RangeBoundDef {
	return RangeBoundDef{
		Normal:     normal,
		Auto:       auto,
		Name:       "",
		Percent:    0,
		Length:     0,
		HasPercent: false,
		HasLength:  false,
	}
}
