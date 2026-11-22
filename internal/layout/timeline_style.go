package layout

import "strings"

// Named timeline declaration names (CSS Scroll-driven Animations 1). The
// scroll and view shorthands expand into the longhands here, at cascade time,
// so a shorthand and a longhand compete per property with each declaration's
// own origin, specificity, and source order.
const (
	propScrollTimeline     = "scroll-timeline"
	propScrollTimelineName = "scroll-timeline-name"
	propScrollTimelineAxis = "scroll-timeline-axis"
	propViewTimeline       = "view-timeline"
	propViewTimelineName   = "view-timeline-name"
	propViewTimelineAxis   = "view-timeline-axis"
	propViewTimelineInset  = "view-timeline-inset"

	timelineAxisBlock  = "block"
	timelineAxisInline = "inline"
	timelineAxisX      = "x"
	timelineAxisY      = "y"

	timelineKindScroll = "scroll"
	timelineKindView   = "view"
	// timelineKindPointer is a pointer-timeline declaration.
	timelineKindPointer = "pointer"

	timelineAutoKeyword           = "auto"
	timelineNoneKeyword           = "none"
	timelineInsetMaxCount         = 2
	timelineScrollShorthandTokens = 2
	timelineViewShorthandTokens   = 4
)

// TimelineDef is one named timeline an element declares for a host to resolve.
// An element exports one definition per name in each comma list it declares,
// so it can export several of each kind. ScrollerID is filled in by a pass
// over the placed boxes, because a box does not know its ancestors.
type TimelineDef struct {
	// Name is the dashed ident the element declares, such as --progress.
	Name string
	// Kind is timelineKindScroll or timelineKindView.
	Kind string
	// Axis is the used physical scroll axis: x or y. A block or inline
	// declaration is resolved through the element's writing mode.
	Axis string
	// SubjectID is the id of the element a view timeline observes. A scroll
	// timeline has no subject, so it stays empty.
	SubjectID string
	// ScrollerID is the id of the scroll container the timeline reads. Empty
	// means the page scroller.
	ScrollerID string
	// ScrollerAnonymous is true when the scroller is an element with no id, so
	// it cannot be named. A host cannot resolve such a timeline; the fix is to
	// give the scroll container an id.
	ScrollerAnonymous bool
	// InsetStart and InsetEnd are the view timeline insets in layout points.
	// They are zero when the declaration is auto or absent.
	InsetStart float64
	InsetEnd   float64
}

// timelineDefsFromStyle builds the named timeline definitions one element
// exports. Scroll comes before view so the order is stable across redraws. A
// nil style means the element declares no timeline.
func timelineDefsFromStyle(style *ResolvedStyle, subjectID string) []TimelineDef {
	if style == nil {
		return nil
	}

	scroll := timelineListText(style.motion().ScrollTimelineName)
	view := timelineListText(style.motion().ViewTimelineName)
	pointer := timelineListText(style.triggers().PointerName)

	if len(scroll)+len(view)+len(pointer) == 0 {
		return nil
	}

	defs := make([]TimelineDef, 0, len(scroll)+len(view)+len(pointer))
	defs = appendTimelineKind(defs, scroll, timelineKindSpec{
		kind: timelineKindScroll, axisList: style.motion().ScrollTimelineAxis, writingMode: style.positioning().WritingMode,
		subjectID: "", insets: nil,
	})
	defs = appendTimelineKind(defs, view, timelineKindSpec{
		kind: timelineKindView, axisList: style.motion().ViewTimelineAxis, writingMode: style.positioning().WritingMode,
		subjectID: subjectID, insets: viewInsetItems(style.motion().ViewTimelineInsets),
	})

	return appendTimelineKind(defs, pointer, timelineKindSpec{
		kind: timelineKindPointer, axisList: style.triggers().PointerAxis, writingMode: style.positioning().WritingMode,
		subjectID: subjectID, insets: nil,
	})
}

// timelineKindSpec is what one kind of timeline needs to turn a name list into
// definitions: the axis list, the subject, and for a view timeline the insets.
type timelineKindSpec struct {
	kind, axisList, writingMode, subjectID string
	insets                                 []viewInset
}

// appendTimelineKind adds one definition per name. A none item exports nothing
// but still takes its index, so the axis and inset lists line up.
func appendTimelineKind(defs []TimelineDef, names []string, spec timelineKindSpec) []TimelineDef {
	for idx, name := range names {
		if name == timelineNoneKeyword {
			continue
		}

		inset := viewInsetAt(spec.insets, idx)

		defs = append(defs, TimelineDef{
			Name:              name,
			Kind:              spec.kind,
			Axis:              timelineAxisAt(spec.axisList, idx, spec.writingMode),
			SubjectID:         spec.subjectID,
			ScrollerID:        "",
			ScrollerAnonymous: false,
			InsetStart:        inset.start,
			InsetEnd:          inset.end,
		})
	}

	return defs
}

// timelineProperty reports whether prop is one of the named timeline
// declarations this engine parses.
func timelineProperty(prop string) bool {
	switch prop {
	case propScrollTimeline, propScrollTimelineName, propScrollTimelineAxis,
		propViewTimeline, propViewTimelineName, propViewTimelineAxis, propViewTimelineInset:
		return true
	default:
		return false
	}
}

// timelineDeclarationValueAccepted is the cascade gate for the timeline
// declarations. A CSS-wide keyword and a var() reference are valid at parse
// time; every other value must match the grammar.
func timelineDeclarationValueAccepted(prop, value string) (bool, bool) {
	if accepted, handled := familyDeclarationValueAccepted(prop, value); handled {
		return accepted, true
	}

	if !timelineProperty(prop) {
		return false, false
	}

	trimmed := strings.TrimSpace(value)
	if cssWideKeyword(strings.ToLower(trimmed)) || containsVarFunc(trimmed) {
		return true, true
	}

	switch prop {
	case propScrollTimeline:
		_, _, ok := parseScrollTimelineShorthand(trimmed)

		return ok, true
	case propViewTimeline:
		_, _, _, ok := parseViewTimelineShorthand(trimmed)

		return ok, true
	case propScrollTimelineName, propViewTimelineName:
		_, ok := parseTimelineName(trimmed)

		return ok, true
	case propScrollTimelineAxis, propViewTimelineAxis:
		_, ok := parseTimelineAxis(trimmed)

		return ok, true
	default:
		_, ok := parseViewTimelineInsetList(trimmed)

		return ok, true
	}
}

// parseTimelineName reads a comma list of timeline names. Each item is the
// none keyword or a dashed ident. A none item keeps its slot, so the axis and
// inset lists still line up with the names by index; a list of only none is
// the initial value, the empty string.
func parseTimelineName(value string) (string, bool) {
	items, ok := timelineListItems(value)
	if !ok {
		return "", false
	}

	allNone := true

	for idx, item := range items {
		if strings.EqualFold(item, timelineNoneKeyword) {
			items[idx] = timelineNoneKeyword

			continue
		}

		if !animationDashedIdent(item) {
			return "", false
		}

		allNone = false
	}

	if allNone {
		return "", true
	}

	return strings.Join(items, ", "), true
}

// parseTimelineAxis reads a comma list of axis keywords: block, inline, x, or
// y. Each keyword is lower-case and stays logical for block and inline; the
// writing mode picks the physical axis at export.
func parseTimelineAxis(value string) (string, bool) {
	items, ok := timelineListItems(value)
	if !ok {
		return "", false
	}

	for idx, item := range items {
		axis := strings.ToLower(item)

		switch axis {
		case timelineAxisBlock, timelineAxisInline, timelineAxisX, timelineAxisY:
			items[idx] = axis
		default:
			return "", false
		}
	}

	return strings.Join(items, ", "), true
}

// timelineUsedAxis resolves a declared axis keyword to the scroll axis the
// host reads: block and inline follow the writing mode, as the engine's
// logical property mapping does.
func timelineUsedAxis(axis, writingMode string) string {
	vertical := isVerticalWritingMode(writingMode)

	switch strings.ToLower(axis) {
	case timelineAxisX:
		return timelineAxisX
	case timelineAxisY:
		return timelineAxisY
	case timelineAxisInline:
		if vertical {
			return timelineAxisY
		}

		return timelineAxisX
	default: // the initial block axis
		if vertical {
			return timelineAxisX
		}

		return timelineAxisY
	}
}

// parseScrollTimelineShorthand reads a comma list of <name> <axis>? items and
// returns the name list and the axis list as comma-joined text, one entry per
// item. The pointer timeline shorthand shares it.
func parseScrollTimelineShorthand(value string) (string, string, bool) {
	items, ok := timelineListItems(value)
	if !ok {
		return "", "", false
	}

	names := make([]string, len(items))
	axes := make([]string, len(items))

	for idx, item := range items {
		name, axis, valid := parseScrollTimelineItem(item)
		if !valid {
			return "", "", false
		}

		names[idx], axes[idx] = name, axis
	}

	return strings.Join(names, ", "), strings.Join(axes, ", "), true
}

// parseScrollTimelineItem reads one <name> <axis>? item. The name comes first,
// as the grammar requires. A missing axis resets the longhand to the initial
// block axis.
func parseScrollTimelineItem(value string) (string, string, bool) {
	tokens := strings.Fields(value)
	if len(tokens) == 0 || len(tokens) > timelineScrollShorthandTokens {
		return "", "", false
	}

	if _, ok := parseTimelineName(tokens[0]); !ok {
		return "", "", false
	}

	axis := timelineAxisBlock

	if len(tokens) == timelineScrollShorthandTokens {
		parsed, ok := parseTimelineAxis(tokens[1])
		if !ok {
			return "", "", false
		}

		axis = parsed
	}

	return tokens[0], axis, true
}

// parseViewTimelineShorthand reads a comma list of
// <name> [ <axis> || <inset> ]? items and returns the name, axis, and inset
// lists as comma-joined text, one entry per item.
func parseViewTimelineShorthand(value string) (string, string, string, bool) {
	items, ok := timelineListItems(value)
	if !ok {
		return "", "", "", false
	}

	names := make([]string, len(items))
	axes := make([]string, len(items))
	insets := make([]string, len(items))

	for idx, item := range items {
		name, axis, inset, valid := parseViewTimelineItem(item)
		if !valid {
			return "", "", "", false
		}

		names[idx], axes[idx], insets[idx] = name, axis, inset
	}

	return strings.Join(names, ", "), strings.Join(axes, ", "), strings.Join(insets, ", "), true
}

// parseViewTimelineItem reads one <name> [ <axis> || <inset> ]? item. The axis
// keyword may appear before or after the one or two inset values. A missing
// axis or inset keeps the longhand initial value, so the shorthand resets a
// previous longhand from a weaker origin.
func parseViewTimelineItem(value string) (string, string, string, bool) {
	tokens := strings.Fields(value)
	if len(tokens) == 0 || len(tokens) > timelineViewShorthandTokens {
		return "", "", "", false
	}

	if _, ok := parseTimelineName(tokens[0]); !ok {
		return "", "", "", false
	}

	axis, axisSeen := timelineAxisBlock, false
	insets := make([]string, 0, timelineInsetMaxCount)

	for _, token := range tokens[1:] {
		if parsed, ok := parseTimelineAxis(token); ok {
			if axisSeen {
				return "", "", "", false
			}

			axis, axisSeen = parsed, true

			continue
		}

		insets = append(insets, token)
	}

	inset := timelineAutoKeyword

	if len(insets) > 0 {
		inset = strings.Join(insets, " ")
		if _, _, ok := parseViewTimelineInset(inset); !ok {
			return "", "", "", false
		}
	}

	return tokens[0], axis, inset, true
}

// parseViewTimelineInsetList reads a comma list of inset items, each one or
// two values, and returns the start and end token of every item.
func parseViewTimelineInsetList(value string) ([][2]string, bool) {
	items, ok := timelineListItems(value)
	if !ok {
		return nil, false
	}

	pairs := make([][2]string, len(items))

	for idx, item := range items {
		start, end, valid := parseViewTimelineInset(item)
		if !valid {
			return nil, false
		}

		pairs[idx] = [2]string{start, end}
	}

	return pairs, true
}

// parseViewTimelineInset splits the one or two inset values of one item. One
// value sets both edges, matching the property grammar.
func parseViewTimelineInset(value string) (string, string, bool) {
	tokens := strings.Fields(value)
	if len(tokens) == 0 || len(tokens) > timelineInsetMaxCount {
		return "", "", false
	}

	for _, token := range tokens {
		if !timelineInsetTokenAccepted(token) {
			return "", "", false
		}
	}

	if len(tokens) == 1 {
		return tokens[0], tokens[0], true
	}

	return tokens[0], tokens[1], true
}

// timelineInsetTokenAccepted reports whether one inset token is auto or a
// length-percentage. Negative values are legal: they move the range outside
// the scrollport.
func timelineInsetTokenAccepted(token string) bool {
	if strings.EqualFold(token, timelineAutoKeyword) {
		return true
	}

	return animationLengthPercentageValueAccepted(token)
}

// expandScrollTimelineDeclaration expands the winning scroll-timeline
// shorthand into its name and axis longhands.
func expandScrollTimelineDeclaration(prop, value string) ([]logicalPropDecl, bool) {
	if prop != propScrollTimeline {
		return nil, false
	}

	trimmed := strings.TrimSpace(value)
	if cssWideKeyword(strings.ToLower(trimmed)) {
		return []logicalPropDecl{
			{prop: propScrollTimelineName, val: trimmed},
			{prop: propScrollTimelineAxis, val: trimmed},
		}, true
	}

	name, axis, ok := parseScrollTimelineShorthand(trimmed)
	if !ok {
		return nil, false
	}

	return []logicalPropDecl{
		{prop: propScrollTimelineName, val: name},
		{prop: propScrollTimelineAxis, val: axis},
	}, true
}

// expandViewTimelineDeclaration expands the winning view-timeline shorthand
// into its name, axis, and inset longhands.
func expandViewTimelineDeclaration(prop, value string) ([]logicalPropDecl, bool) {
	if prop != propViewTimeline {
		return nil, false
	}

	trimmed := strings.TrimSpace(value)
	if cssWideKeyword(strings.ToLower(trimmed)) {
		return []logicalPropDecl{
			{prop: propViewTimelineName, val: trimmed},
			{prop: propViewTimelineAxis, val: trimmed},
			{prop: propViewTimelineInset, val: trimmed},
		}, true
	}

	name, axis, inset, ok := parseViewTimelineShorthand(trimmed)
	if !ok {
		return nil, false
	}

	return []logicalPropDecl{
		{prop: propViewTimelineName, val: name},
		{prop: propViewTimelineAxis, val: axis},
		{prop: propViewTimelineInset, val: inset},
	}, true
}

// applyTimelineProps resolves the named timeline declarations. The shorthands
// reach this handler only through var() substitution; a literal shorthand is
// already expanded at cascade time.
func applyTimelineProps(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext,
	parent *ResolvedStyle, hasParent bool,
) bool {
	if applyTriggerProps(style, prop, value, fsize, parent, hasParent) {
		return true
	}

	if applyDeclaredProps(style, prop, value, parent, hasParent) {
		motionAfterDeclared(style, prop)

		return true
	}

	if !timelineProperty(prop) {
		return false
	}

	applyTimelineProp(style, prop, value, fsize, ctx, parent, hasParent)

	return true
}

func applyTimelineProp(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext,
	parent *ResolvedStyle, hasParent bool,
) {
	switch prop {
	case propScrollTimeline, propViewTimeline:
		applyTimelineShorthand(style, prop, value, fsize, ctx, parent, hasParent)
	case propScrollTimelineName, propViewTimelineName:
		applyTimelineNameProp(style, prop, value, parent, hasParent)
	case propScrollTimelineAxis, propViewTimelineAxis:
		applyTimelineAxisProp(style, prop, value, parent, hasParent)
	case propViewTimelineInset:
		applyViewTimelineInsetProp(style, value, fsize, ctx, parent, hasParent)
	}
}

func applyTimelineShorthand(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext,
	parent *ResolvedStyle, hasParent bool,
) {
	var (
		declarations []logicalPropDecl
		valid        bool
	)

	if prop == propScrollTimeline {
		declarations, valid = expandScrollTimelineDeclaration(prop, value)
	} else {
		declarations, valid = expandViewTimelineDeclaration(prop, value)
	}

	if !valid {
		return
	}

	for _, declaration := range declarations {
		applyTimelineProp(style, declaration.prop, declaration.val, fsize, ctx, parent, hasParent)
	}
}

// applyTimelineNameProp stores one name longhand. Neither name is inherited,
// so only inherit copies the parent value; the other wide keywords reset to
// none.
func applyTimelineNameProp(style *ResolvedStyle, prop, value string, parent *ResolvedStyle, hasParent bool) {
	var parentValue string

	if prop == propScrollTimelineName {
		parentValue = parentScrollTimelineName(parent, hasParent)
	} else {
		parentValue = parentViewTimelineName(parent, hasParent)
	}

	trimmed := strings.TrimSpace(value)

	var name string

	switch {
	case cssWideKeyword(strings.ToLower(trimmed)):
		name = timelineNameWideValue(strings.ToLower(trimmed), parentValue, hasParent)
	default:
		parsed, valid := parseTimelineName(trimmed)
		if !valid {
			return
		}

		name = parsed
	}

	if prop == propScrollTimelineName {
		style.editMotion().ScrollTimelineName = name
	} else {
		style.editMotion().ViewTimelineName = name
	}
}

// applyTimelineAxisProp stores one axis longhand. The initial block axis is
// re-applied for every wide keyword except inherit.
func applyTimelineAxisProp(style *ResolvedStyle, prop, value string, parent *ResolvedStyle, hasParent bool) {
	var parentValue string

	if prop == propScrollTimelineAxis {
		parentValue = parentScrollTimelineAxis(parent, hasParent)
	} else {
		parentValue = parentViewTimelineAxis(parent, hasParent)
	}

	trimmed := strings.TrimSpace(value)

	var axis string

	switch {
	case cssWideKeyword(strings.ToLower(trimmed)):
		axis = timelineAxisWideValue(strings.ToLower(trimmed), parentValue, hasParent)
	default:
		parsed, valid := parseTimelineAxis(trimmed)
		if !valid {
			return
		}

		axis = parsed
	}

	if prop == propScrollTimelineAxis {
		style.editMotion().ScrollTimelineAxis = axis
	} else {
		style.editMotion().ViewTimelineAxis = axis
	}
}

// applyViewTimelineInsetProp stores the start and end insets in layout points.
// auto leaves a Set flag false; percentages resolve against the page
// scrollport dimension the timeline axis reads.
func applyViewTimelineInsetProp(
	style *ResolvedStyle, value string, fsize float64, ctx *styleContext,
	parent *ResolvedStyle, hasParent bool,
) {
	trimmed := strings.TrimSpace(value)
	if cssWideKeyword(strings.ToLower(trimmed)) {
		applyViewTimelineInsetWide(style, strings.ToLower(trimmed), parent, hasParent)

		return
	}

	pairs, ok := parseViewTimelineInsetList(trimmed)
	if !ok {
		return
	}

	items := make([]viewInset, len(pairs))

	for idx, pair := range pairs {
		scrollport := timelineScrollport(timelineAxisAt(style.motion().ViewTimelineAxis, idx, style.positioning().WritingMode), ctx)
		items[idx].start, items[idx].startSet = timelineInsetValue(pair[0], fsize, scrollport)
		items[idx].end, items[idx].endSet = timelineInsetValue(pair[1], fsize, scrollport)
	}

	motion := style.editMotion()
	motion.ViewTimelineInsets = encodeViewInsets(items)
	motion.ViewTimelineInsetStart, motion.ViewTimelineInsetStartSet = items[0].start, items[0].startSet
	motion.ViewTimelineInsetEnd, motion.ViewTimelineInsetEndSet = items[0].end, items[0].endSet
}

// timelineInsetValue resolves one inset token. auto resolves to zero with a
// false Set flag so the host can tell it apart from an explicit 0px.
func timelineInsetValue(token string, fsize, scrollport float64) (float64, bool) {
	if strings.EqualFold(token, timelineAutoKeyword) {
		return 0, false
	}

	if points, ok := calcLength(token, fsize, scrollport); ok {
		return points, true
	}

	if points, ok := plainLength(token, fsize, scrollport); ok {
		return points, true
	}

	return 0, false
}

// timelineScrollport is the page scrollport dimension on a view timeline's
// used axis, in layout points. Percentages resolve against it.
func timelineScrollport(usedAxis string, ctx *styleContext) float64 {
	if ctx == nil {
		return 0
	}

	if usedAxis == timelineAxisX {
		return ctx.viewportW
	}

	return ctx.viewportH
}

func applyViewTimelineInsetWide(style *ResolvedStyle, keyword string, parent *ResolvedStyle, hasParent bool) {
	style.editMotion().ViewTimelineInsets = ""
	style.editMotion().ViewTimelineInsetStart = 0
	style.editMotion().ViewTimelineInsetEnd = 0
	style.editMotion().ViewTimelineInsetStartSet = false
	style.editMotion().ViewTimelineInsetEndSet = false

	if keyword == inheritKeyword && hasParent && parent != nil {
		style.editMotion().ViewTimelineInsets = parent.motion().ViewTimelineInsets
		style.editMotion().ViewTimelineInsetStart = parent.motion().ViewTimelineInsetStart
		style.editMotion().ViewTimelineInsetEnd = parent.motion().ViewTimelineInsetEnd
		style.editMotion().ViewTimelineInsetStartSet = parent.motion().ViewTimelineInsetStartSet
		style.editMotion().ViewTimelineInsetEndSet = parent.motion().ViewTimelineInsetEndSet
	}
}

// timelineNameWideValue resolves a wide keyword for a non-inherited name.
func timelineNameWideValue(keyword, parentValue string, hasParent bool) string {
	if keyword == inheritKeyword && hasParent {
		return parentValue
	}

	return ""
}

// timelineAxisWideValue resolves a wide keyword for a non-inherited axis. The
// initial value is the block axis.
func timelineAxisWideValue(keyword, parentValue string, hasParent bool) string {
	if keyword == inheritKeyword && hasParent && parentValue != "" {
		return parentValue
	}

	return timelineAxisBlock
}

func parentScrollTimelineName(parent *ResolvedStyle, hasParent bool) string {
	if !hasParent || parent == nil {
		return ""
	}

	return parent.motion().ScrollTimelineName
}

func parentScrollTimelineAxis(parent *ResolvedStyle, hasParent bool) string {
	if !hasParent || parent == nil {
		return ""
	}

	return parent.motion().ScrollTimelineAxis
}

func parentViewTimelineName(parent *ResolvedStyle, hasParent bool) string {
	if !hasParent || parent == nil {
		return ""
	}

	return parent.motion().ViewTimelineName
}

func parentViewTimelineAxis(parent *ResolvedStyle, hasParent bool) string {
	if !hasParent || parent == nil {
		return ""
	}

	return parent.motion().ViewTimelineAxis
}
