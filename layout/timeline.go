package layout

import ilayout "github.com/chinmay-sawant/blinkless/internal/layout"

// TimelineDef is one named timeline an element box declares for a host to
// resolve. CSS Scroll-driven Animations declares it with scroll-timeline or
// view-timeline, and animation-timeline references it by name. Each property
// is a comma list, and a box exports one definition per name: every scroll
// definition first, then every view one, then every pointer one, each in the
// order the names are written. A none item exports nothing but keeps its
// place, so the axis and inset lists still line up with the names, and a
// shorter axis or inset list repeats to the length of the name list.
//
// The engine owns no clock and no scroll offset: it does not compute timeline
// progress. A host resolves the name against the nearest ancestor box, then
// reads the scroll state itself. timeline-scope is not applied here; the host
// resolves a name through Box.TimelineScope.
//
// ScrollerID names the scroll container a timeline reads. A scroll timeline
// reads its own element when that element is a scroll container (overflow
// hidden, scroll, or auto), and the page otherwise. A view timeline reads the
// nearest ancestor scroll container, or the page when there is none. The root
// and body elements never count, because their overflow belongs to the page.
type TimelineDef struct {
	// Name is the dashed ident the element declares, such as --progress.
	Name string
	// Kind is "scroll", "view", or "pointer". A pointer timeline tracks the
	// pointer over the element named by SubjectID; the engine has no
	// pointer, so a host supplies the position.
	Kind string
	// Axis is the used physical scroll axis: "x" or "y". A block or inline
	// declaration is resolved through the element's writing mode.
	Axis string
	// SubjectID is the id of the element a view timeline observes. A scroll
	// timeline has no subject, so it is empty.
	SubjectID string
	// ScrollerID is the id of the scroll container the timeline reads. Empty
	// means the page scroller.
	ScrollerID string
	// ScrollerAnonymous is true when the scroller is a scroll container with no
	// id attribute. It cannot be named, so a host cannot resolve the timeline;
	// give that container an id.
	ScrollerAnonymous bool
	// InsetStart and InsetEnd are the view timeline insets in CSS pixels, the
	// same unit as Box.X/Y/W/H. They are zero when the declaration is auto or
	// absent.
	InsetStart float64
	InsetEnd   float64
}

// timelinesFrom maps the engine-side definitions to the public payload. A box
// with no declaration keeps a nil slice. Insets convert from layout points to
// CSS pixels with the border box, so a host compares them with Box.X/Y/W/H
// directly.
func timelinesFrom(defs []ilayout.TimelineDef) []TimelineDef {
	if len(defs) == 0 {
		return nil
	}

	timelines := make([]TimelineDef, len(defs))

	for i, def := range defs {
		timelines[i] = TimelineDef{
			Name:              def.Name,
			Kind:              def.Kind,
			Axis:              def.Axis,
			SubjectID:         def.SubjectID,
			ScrollerID:        def.ScrollerID,
			ScrollerAnonymous: def.ScrollerAnonymous,
			InsetStart:        def.InsetStart * ptToPx,
			InsetEnd:          def.InsetEnd * ptToPx,
		}
	}

	return timelines
}
