package layout

import ilayout "github.com/chinmay-sawant/blinkless/internal/layout"

// The engine owns no clock, no scroll offset, no pointer, and no event loop.
// It parses and validates the trigger declarations, applies the cascade and
// inheritance rules, and exports the result on Box. A host computes timeline
// progress, watches events, and decides when a trigger is armed or active.

// TimelineScope is the computed timeline-scope of a box. It lets a timeline
// declared on a descendant be named by an animation on this box or on a
// sibling subtree. The engine does not resolve names; a host reads this value
// when it looks a dashed name up.
type TimelineScope struct {
	// None is true when the box exposes no names. It is the initial value, so
	// a box that never declares timeline-scope reports None as true.
	None bool
	// All is true for timeline-scope: all. Every timeline name declared in the
	// subtree is visible from this box.
	All bool
	// Names lists the dashed idents of a named scope, such as "--card". It is
	// nil unless the box declares names.
	Names []string
}

// RangeBound is one edge of a timeline trigger range. A range edge is one of
// the normal keyword, the auto keyword, or an optional timeline range name
// plus an optional offset.
//
// A host reads an edge in this order: Normal and Auto first, then Name (a
// timeline range such as "cover", "contain", "entry", "exit",
// "entry-crossing", or "exit-crossing"), then the offset. A Name with no
// offset leaves HasPercent and HasLength false, and the host picks the
// start or end of that range. A calc() that mixes a percentage and a length
// sets both HasPercent and HasLength; the offset is their sum.
type RangeBound struct {
	// Normal is true for the normal keyword.
	Normal bool
	// Auto is true for the auto keyword. Only the active range accepts it.
	Auto bool
	// Name is the timeline range name in lower case, or empty.
	Name string
	// Percent is the percentage part as a plain number, so 50 means 50%. It
	// is valid when HasPercent is true.
	Percent float64
	// Length is the length part in CSS pixels. It is valid when HasLength is
	// true.
	Length float64
	// HasPercent and HasLength say which offset parts were declared.
	HasPercent bool
	HasLength  bool
}

// TimelineTrigger is one timeline trigger a box declares with
// timeline-trigger or its longhands. The engine does not compute progress; a
// host reads the source timeline and compares its progress with the two
// ranges. The trigger arms while the progress is inside the activation range
// and stays on while it is inside the active range.
//
// When the longhand lists differ in length, they repeat to the length of the
// name list, as the draft requires.
type TimelineTrigger struct {
	// Name is the dashed ident of the trigger, such as "--play".
	Name string
	// Source is the raw timeline reference text: "scroll()", "view()", a
	// dashed name such as "--page", "auto", or "none". Several references are
	// joined with a space.
	Source string
	// Sources is Source split into single references.
	Sources []string
	// ActivationStart and ActivationEnd bound the activation range. Their
	// initial value is normal.
	ActivationStart RangeBound
	ActivationEnd   RangeBound
	// ActiveStart and ActiveEnd bound the active range. Their initial value
	// is auto, which means the active range matches the activation range.
	ActiveStart RangeBound
	ActiveEnd   RangeBound
}

// EventTrigger is one event trigger a box declares with event-trigger or its
// longhands. A host starts the trigger when an event in Start fires on the
// box and ends it when an event in End fires. Event types are the text the
// author wrote, such as "click" or "keydown(Enter)"; the engine does not
// check them against a list of known events.
type EventTrigger struct {
	// Name is the dashed ident of the trigger, such as "--press".
	Name string
	// Start lists the event types before the slash. It is nil for none.
	Start []string
	// End lists the event types after the slash. It is nil when no slash is
	// given.
	End []string
}

// timelineScopeFrom maps the engine-side scope to the public one.
func timelineScopeFrom(scope ilayout.TimelineScopeDef) TimelineScope {
	return TimelineScope{None: scope.None, All: scope.All, Names: scope.Names}
}

// rangeBoundFrom maps one engine-side edge to the public one. Lengths are
// already CSS pixels.
func rangeBoundFrom(bound ilayout.RangeBoundDef) RangeBound {
	return RangeBound{
		Normal:     bound.Normal,
		Auto:       bound.Auto,
		Name:       bound.Name,
		Percent:    bound.Percent,
		Length:     bound.Length,
		HasPercent: bound.HasPercent,
		HasLength:  bound.HasLength,
	}
}

// timelineTriggersFrom maps the engine-side triggers to the public payload. A
// box with no trigger keeps a nil slice.
func timelineTriggersFrom(defs []ilayout.TimelineTriggerDef) []TimelineTrigger {
	if len(defs) == 0 {
		return nil
	}

	out := make([]TimelineTrigger, len(defs))

	for i, def := range defs {
		out[i] = TimelineTrigger{
			Name:            def.Name,
			Source:          def.Source,
			Sources:         def.Sources,
			ActivationStart: rangeBoundFrom(def.ActivationStart),
			ActivationEnd:   rangeBoundFrom(def.ActivationEnd),
			ActiveStart:     rangeBoundFrom(def.ActiveStart),
			ActiveEnd:       rangeBoundFrom(def.ActiveEnd),
		}
	}

	return out
}

// eventTriggersFrom maps the engine-side event triggers to the public
// payload. A box with no trigger keeps a nil slice.
func eventTriggersFrom(defs []ilayout.EventTriggerDef) []EventTrigger {
	if len(defs) == 0 {
		return nil
	}

	out := make([]EventTrigger, len(defs))

	for i, def := range defs {
		out[i] = EventTrigger{Name: def.Name, Start: def.Start, End: def.End}
	}

	return out
}
