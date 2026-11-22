package layout

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// timelineAcceptanceCase is one declaration the cascade gate must accept or
// reject.
type timelineAcceptanceCase struct {
	prop  string
	value string
	want  bool
}

func TestTimelineDeclarationValueAccepted(t *testing.T) {
	t.Parallel()

	cases := []timelineAcceptanceCase{
		{prop: "scroll-timeline", value: "--page", want: true},
		{prop: "scroll-timeline", value: "--page y", want: true},
		{prop: "scroll-timeline", value: "none inline", want: true},
		{prop: "scroll-timeline", value: "y --page", want: false},
		{prop: "scroll-timeline", value: "--page w", want: false},
		{prop: "scroll-timeline", value: "page y", want: false},
		{prop: "scroll-timeline", value: "--a --b", want: false},
		{prop: "scroll-timeline-name", value: "--page", want: true},
		{prop: "scroll-timeline-name", value: "none", want: true},
		{prop: "scroll-timeline-name", value: "--a, --b", want: true},
		{prop: "scroll-timeline-name", value: "none, --b", want: true},
		{prop: "scroll-timeline-name", value: "--a,", want: false},
		{prop: "scroll-timeline-name", value: "--a,,--b", want: false},
		{prop: "scroll-timeline-name", value: "--a, b", want: false},
		{prop: "scroll-timeline-axis", value: "x, y", want: true},
		{prop: "scroll-timeline-axis", value: "x, z", want: false},
		{prop: "scroll-timeline", value: "--a x, --b", want: true},
		{prop: "scroll-timeline", value: "--a x, --b z", want: false},
		{prop: "view-timeline", value: "--a x 10px, --b 5px 6px inline", want: true},
		{prop: "view-timeline", value: "--a, ", want: false},
		{prop: "view-timeline-inset", value: "20px 10px, auto", want: true},
		{prop: "view-timeline-inset", value: "20px 10px, 1px 2px 3px", want: false},
		{prop: "view-timeline-inset", value: "5px,", want: false},
		{prop: "scroll-timeline-name", value: "page", want: false},
		{prop: "scroll-timeline-axis", value: "inline", want: true},
		{prop: "scroll-timeline-axis", value: "block", want: true},
		{prop: "scroll-timeline-axis", value: "z", want: false},
		{prop: "view-timeline", value: "--card", want: true},
		{prop: "view-timeline", value: "--card block", want: true},
		{prop: "view-timeline", value: "--card block 10px 20px", want: true},
		{prop: "view-timeline", value: "--card 10px x", want: true},
		{prop: "view-timeline", value: "none 20% 200px", want: true},
		// A second axis keyword is the invalid part.
		{prop: "view-timeline", value: "--card block block", want: false}, //nolint:dupword // second block
		{prop: "view-timeline", value: "--card 10px 20px 30px", want: false},
		{prop: "view-timeline", value: "card block", want: false},
		{prop: "view-timeline-name", value: "--card", want: true},
		{prop: "view-timeline-name", value: "card", want: false},
		{prop: "view-timeline-axis", value: "block", want: true},
		{prop: "view-timeline-axis", value: "both", want: false},
		{prop: "view-timeline-inset", value: "auto", want: true},
		{prop: "view-timeline-inset", value: "20px 10px", want: true},
		{prop: "view-timeline-inset", value: "auto -100px", want: true},
		{prop: "view-timeline-inset", value: "20px 10px auto", want: false},
		{prop: "view-timeline-inset", value: "wide", want: false},
	}

	for _, test := range cases {
		t.Run(test.prop+"/"+test.value, func(t *testing.T) {
			t.Parallel()

			got, handled := timelineDeclarationValueAccepted(test.prop, test.value)
			if !handled {
				t.Fatalf("handled = false for timeline property %q", test.prop)
			}

			if got != test.want {
				t.Fatalf("accepted = %t, want %t", got, test.want)
			}
		})
	}
}

func TestTimelineDeclarationValueAcceptedLeavesOtherProperties(t *testing.T) {
	t.Parallel()

	accepted, handled := timelineDeclarationValueAccepted("opacity", "0.5")
	if accepted || handled {
		t.Fatalf("(accepted, handled) = (%t, %t), want (false, false)", accepted, handled)
	}
}

// timelineStyleCase is one declaration whose used timeline values are checked.
type timelineStyleCase struct {
	name          string
	decl          string
	scrollName    string
	scrollAxis    string
	viewName      string
	viewAxis      string
	insetStart    float64
	insetEnd      float64
	insetStartSet bool
	insetEndSet   bool
}

func TestTimelineShorthandAndLonghandParsing(t *testing.T) {
	t.Parallel()

	for _, test := range timelineStyleCases() {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			style := styleForDecl(t, test.decl)
			assertTimelineStyle(t, style, test)
		})
	}
}

// timelineStyleCases are the shorthand and longhand declarations whose used
// values the parsing test checks.
func timelineStyleCases() []timelineStyleCase {
	return append(scrollTimelineStyleCases(), viewTimelineStyleCases()...)
}

// scrollTimelineStyleCases are the scroll-timeline shorthand and longhand
// declarations.
func scrollTimelineStyleCases() []timelineStyleCase {
	return []timelineStyleCase{
		{
			name:       "scroll shorthand with axis",
			decl:       "scroll-timeline: --page y",
			scrollName: "--page",
			scrollAxis: timelineAxisY,
			viewAxis:   timelineAxisBlock,
		},
		{
			name:       "scroll shorthand keeps the block initial",
			decl:       "scroll-timeline: --page",
			scrollName: "--page",
			scrollAxis: timelineAxisBlock,
			viewAxis:   timelineAxisBlock,
		},
		{
			name:       "scroll longhands",
			decl:       "scroll-timeline-name: --page; scroll-timeline-axis: inline",
			scrollName: "--page",
			scrollAxis: timelineAxisInline,
			viewAxis:   timelineAxisBlock,
		},
		{
			name:       "none clears the name",
			decl:       "scroll-timeline-name: --page; scroll-timeline-name: none",
			scrollAxis: timelineAxisBlock,
			viewAxis:   timelineAxisBlock,
		},
	}
}

// viewTimelineStyleCases are the view-timeline shorthand and longhand
// declarations.
func viewTimelineStyleCases() []timelineStyleCase {
	return []timelineStyleCase{
		{
			name:       "view shorthand name only",
			decl:       "view-timeline: --card",
			scrollAxis: timelineAxisBlock,
			viewName:   "--card",
			viewAxis:   timelineAxisBlock,
		},
		{
			name:          "view shorthand axis then inset",
			decl:          "view-timeline: --card inline 10px 20px",
			scrollAxis:    timelineAxisBlock,
			viewName:      "--card",
			viewAxis:      timelineAxisInline,
			insetStart:    pxToPt(10),
			insetEnd:      pxToPt(20),
			insetStartSet: true,
			insetEndSet:   true,
		},
		{
			name:          "view shorthand inset then axis",
			decl:          "view-timeline: --card 10px x",
			scrollAxis:    timelineAxisBlock,
			viewName:      "--card",
			viewAxis:      timelineAxisX,
			insetStart:    pxToPt(10),
			insetEnd:      pxToPt(10),
			insetStartSet: true,
			insetEndSet:   true,
		},
		{
			name:        "view longhands",
			decl:        "view-timeline-name: --card; view-timeline-axis: y; view-timeline-inset: auto 20px",
			scrollAxis:  timelineAxisBlock,
			viewName:    "--card",
			viewAxis:    timelineAxisY,
			insetEnd:    pxToPt(20),
			insetEndSet: true,
		},
	}
}

// assertTimelineStyle compares the four timeline fields plus the inset values.
func assertTimelineStyle(t *testing.T, style *ResolvedStyle, want timelineStyleCase) {
	t.Helper()

	if style.motion().ScrollTimelineName != want.scrollName {
		t.Errorf("ScrollTimelineName = %q, want %q", style.motion().ScrollTimelineName, want.scrollName)
	}

	if style.motion().ScrollTimelineAxis != want.scrollAxis {
		t.Errorf("ScrollTimelineAxis = %q, want %q", style.motion().ScrollTimelineAxis, want.scrollAxis)
	}

	if style.motion().ViewTimelineName != want.viewName {
		t.Errorf("ViewTimelineName = %q, want %q", style.motion().ViewTimelineName, want.viewName)
	}

	if style.motion().ViewTimelineAxis != want.viewAxis {
		t.Errorf("ViewTimelineAxis = %q, want %q", style.motion().ViewTimelineAxis, want.viewAxis)
	}

	if !near(style.motion().ViewTimelineInsetStart, want.insetStart) {
		t.Errorf("ViewTimelineInsetStart = %v, want %v", style.motion().ViewTimelineInsetStart, want.insetStart)
	}

	if !near(style.motion().ViewTimelineInsetEnd, want.insetEnd) {
		t.Errorf("ViewTimelineInsetEnd = %v, want %v", style.motion().ViewTimelineInsetEnd, want.insetEnd)
	}

	if style.motion().ViewTimelineInsetStartSet != want.insetStartSet {
		t.Errorf("ViewTimelineInsetStartSet = %t, want %t", style.motion().ViewTimelineInsetStartSet, want.insetStartSet)
	}

	if style.motion().ViewTimelineInsetEndSet != want.insetEndSet {
		t.Errorf("ViewTimelineInsetEndSet = %t, want %t", style.motion().ViewTimelineInsetEndSet, want.insetEndSet)
	}
}

func TestTimelineLonghandOverridesShorthand(t *testing.T) {
	t.Parallel()
	// A longhand declared after the shorthand wins its own property.
	style := styleForDecl(t,
		"view-timeline: --card block 10px; "+
			"view-timeline-name: --late; view-timeline-axis: inline; view-timeline-inset: 4px")
	assertTimelineStyle(t, style, timelineStyleCase{
		name:          "later longhand wins",
		scrollAxis:    timelineAxisBlock,
		viewName:      "--late",
		viewAxis:      timelineAxisInline,
		insetStart:    pxToPt(4),
		insetEnd:      pxToPt(4),
		insetStartSet: true,
		insetEndSet:   true,
	})

	// A shorthand declared after the longhands resets them.
	style = styleForDecl(t,
		"view-timeline-name: --early; view-timeline-axis: inline; view-timeline-inset: 4px; "+
			"view-timeline: --card x 10px")
	assertTimelineStyle(t, style, timelineStyleCase{
		name:          "later shorthand wins",
		scrollAxis:    timelineAxisBlock,
		viewName:      "--card",
		viewAxis:      timelineAxisX,
		insetStart:    pxToPt(10),
		insetEnd:      pxToPt(10),
		insetStartSet: true,
		insetEndSet:   true,
	})
}

// TestTimelineShorthandThroughVar covers the var() path, where the shorthand
// is substituted after the cascade and therefore expands in the applier.
func TestTimelineShorthandThroughVar(t *testing.T) {
	t.Parallel()

	style := styleForDecl(t, "--timeline: --card block 10px; view-timeline: var(--timeline)")
	assertTimelineStyle(t, style, timelineStyleCase{
		name:          "var shorthand",
		scrollAxis:    timelineAxisBlock,
		viewName:      "--card",
		viewAxis:      timelineAxisBlock,
		insetStart:    pxToPt(10),
		insetEnd:      pxToPt(10),
		insetStartSet: true,
		insetEndSet:   true,
	})
}

func TestTimelineNamesNotInherited(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div id="parent">
			<div id="child"></div>
			<div id="inherit"></div>
		</div>
	</body></html>`)

	sheets := []*css.Stylesheet{sheet(t, `
		#parent {
			scroll-timeline: --page y;
			view-timeline: --card block 10px;
		}
		#inherit {
			scroll-timeline-name: inherit;
			scroll-timeline-axis: inherit;
			view-timeline-name: inherit;
			view-timeline-axis: inherit;
			view-timeline-inset: inherit;
		}
	`)}

	styles := resolveStyles(root, sheets, "print", testViewport, 800)

	assertTimelineNotInherited(t, styles[findElementByID(t, root, "child")])
	assertTimelineInherited(t, styles[findElementByID(t, root, "inherit")])
}

// assertTimelineNotInherited checks a child that declares nothing keeps the
// initial timeline values even though its parent declares both kinds.
func assertTimelineNotInherited(t *testing.T, child *ResolvedStyle) {
	t.Helper()

	if child.motion().ScrollTimelineName != "" || child.motion().ViewTimelineName != "" {
		t.Fatalf("child names = (%q, %q), want empty", child.motion().ScrollTimelineName, child.motion().ViewTimelineName)
	}

	if child.motion().ScrollTimelineAxis != timelineAxisBlock || child.motion().ViewTimelineAxis != timelineAxisBlock {
		t.Fatalf(
			"child axes = (%q, %q), want the block initial",
			child.motion().ScrollTimelineAxis, child.motion().ViewTimelineAxis,
		)
	}

	if child.motion().ViewTimelineInsetStartSet || child.motion().ViewTimelineInsetEndSet {
		t.Fatalf(
			"child inset flags = (%t, %t), want false",
			child.motion().ViewTimelineInsetStartSet, child.motion().ViewTimelineInsetEndSet,
		)
	}
}

// assertTimelineInherited checks the explicit inherit keyword copies every
// timeline declaration from the parent.
func assertTimelineInherited(t *testing.T, inherit *ResolvedStyle) {
	t.Helper()

	if inherit.motion().ScrollTimelineName != "--page" || inherit.motion().ViewTimelineName != "--card" {
		t.Fatalf(
			"inherit names = (%q, %q), want (--page, --card)",
			inherit.motion().ScrollTimelineName, inherit.motion().ViewTimelineName,
		)
	}

	if inherit.motion().ScrollTimelineAxis != timelineAxisY ||
		inherit.motion().ViewTimelineAxis != timelineAxisBlock {
		t.Fatalf(
			"inherit axes = (%q, %q), want (y, block)",
			inherit.motion().ScrollTimelineAxis, inherit.motion().ViewTimelineAxis,
		)
	}

	if !near(inherit.motion().ViewTimelineInsetStart, pxToPt(10)) || !inherit.motion().ViewTimelineInsetStartSet {
		t.Fatalf(
			"inherit inset start = (%v, %t), want (%v, true)",
			inherit.motion().ViewTimelineInsetStart, inherit.motion().ViewTimelineInsetStartSet, pxToPt(10),
		)
	}
}

func TestTimelineLogicalAxisExport(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		decl string
		want string
	}{
		{
			name: "block in horizontal writing",
			decl: "scroll-timeline: --page block",
			want: timelineAxisY,
		},
		{
			name: "inline in horizontal writing",
			decl: "scroll-timeline: --page inline",
			want: timelineAxisX,
		},
		{
			name: "block in vertical writing",
			decl: "writing-mode: vertical-rl; scroll-timeline: --page block",
			want: timelineAxisX,
		},
		{
			name: "inline in vertical writing",
			decl: "writing-mode: vertical-lr; scroll-timeline: --page inline",
			want: timelineAxisY,
		},
		{
			name: "view timeline block in vertical writing",
			decl: "writing-mode: vertical-rl; view-timeline: --card block",
			want: timelineAxisX,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			style := styleForDecl(t, test.decl)
			defs := timelineDefsFromStyle(style, "subject")

			if len(defs) != 1 {
				t.Fatalf("definitions = %d, want 1", len(defs))
			}

			if defs[0].Axis != test.want {
				t.Fatalf("Axis = %q, want %q", defs[0].Axis, test.want)
			}
		})
	}
}

func TestViewTimelineInsetValues(t *testing.T) {
	t.Parallel()

	for _, test := range viewTimelineInsetCases() {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assertViewTimelineInset(t, styleForDecl(t, test.decl), test)
		})
	}
}

// viewTimelineInsetCase is one view-timeline-inset declaration and the values
// it must store.
type viewTimelineInsetCase struct {
	name          string
	decl          string
	insetStart    float64
	insetEnd      float64
	insetStartSet bool
	insetEndSet   bool
}

// viewTimelineInsetCases covers the one and two value forms, auto, negatives,
// and the scrollport dimension each percentage resolves against.
func viewTimelineInsetCases() []viewTimelineInsetCase {
	return []viewTimelineInsetCase{
		{
			name: "auto keeps no value",
			decl: "view-timeline-inset: auto",
		},
		{
			name:          "one value sets both edges",
			decl:          "view-timeline-inset: 20px",
			insetStart:    pxToPt(20),
			insetEnd:      pxToPt(20),
			insetStartSet: true,
			insetEndSet:   true,
		},
		{
			name:          "two values set start then end",
			decl:          "view-timeline-inset: 20px 10px",
			insetStart:    pxToPt(20),
			insetEnd:      pxToPt(10),
			insetStartSet: true,
			insetEndSet:   true,
		},
		{
			name:        "auto then a negative outset",
			decl:        "view-timeline-inset: auto -100px",
			insetEnd:    pxToPt(-100),
			insetEndSet: true,
		},
		{
			name:          "block percentage uses the scrollport height",
			decl:          "view-timeline-inset: -10% 50%",
			insetStart:    -80,
			insetEnd:      400,
			insetStartSet: true,
			insetEndSet:   true,
		},
		{
			name:          "inline percentage uses the scrollport width",
			decl:          "view-timeline-axis: inline; view-timeline-inset: 50%",
			insetStart:    250,
			insetEnd:      250,
			insetStartSet: true,
			insetEndSet:   true,
		},
	}
}

// assertViewTimelineInset compares the stored inset values and Set flags.
func assertViewTimelineInset(t *testing.T, style *ResolvedStyle, test viewTimelineInsetCase) {
	t.Helper()

	if !near(style.motion().ViewTimelineInsetStart, test.insetStart) {
		t.Fatalf("inset start = %v, want %v", style.motion().ViewTimelineInsetStart, test.insetStart)
	}

	if !near(style.motion().ViewTimelineInsetEnd, test.insetEnd) {
		t.Fatalf("inset end = %v, want %v", style.motion().ViewTimelineInsetEnd, test.insetEnd)
	}

	if style.motion().ViewTimelineInsetStartSet != test.insetStartSet {
		t.Fatalf("inset start set = %t, want %t", style.motion().ViewTimelineInsetStartSet, test.insetStartSet)
	}

	if style.motion().ViewTimelineInsetEndSet != test.insetEndSet {
		t.Fatalf("inset end set = %t, want %t", style.motion().ViewTimelineInsetEndSet, test.insetEndSet)
	}
}

func TestPlacedTimelineDefs(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">
		<div id="plain">plain</div>
		<div id="both" style="scroll-timeline: --page y; view-timeline: --card block 20px 10px">both</div>
	</body></html>`)

	placed := PlacedElements(res)

	plain := placedTimelineElement(t, placed, "plain")
	if plain.Timelines != nil {
		t.Fatalf("plain element Timelines = %#v, want nil", plain.Timelines)
	}

	assertPlacedTimelines(t, placedTimelineElement(t, placed, "both"))
}

// assertPlacedTimelines checks the two definitions a box that declares both
// kinds exports, in the stable scroll then view order.
func assertPlacedTimelines(t *testing.T, both PlacedElement) {
	t.Helper()

	if len(both.Timelines) != 2 {
		t.Fatalf("both element definitions = %d, want 2", len(both.Timelines))
	}

	assertPlacedScrollTimeline(t, both.Timelines[0])
	assertPlacedViewTimeline(t, both.Timelines[1])
}

func assertPlacedScrollTimeline(t *testing.T, scroll TimelineDef) {
	t.Helper()

	if scroll.Kind != timelineKindScroll || scroll.Name != "--page" || scroll.Axis != timelineAxisY {
		t.Fatalf("scroll definition = %#v, want kind scroll, name --page, axis y", scroll)
	}

	if scroll.SubjectID != "" || scroll.ScrollerID != "" {
		t.Fatalf("scroll subject/scroller = (%q, %q), want empty", scroll.SubjectID, scroll.ScrollerID)
	}
}

func assertPlacedViewTimeline(t *testing.T, view TimelineDef) {
	t.Helper()

	if view.Kind != timelineKindView || view.Name != "--card" || view.Axis != timelineAxisY {
		t.Fatalf("view definition = %#v, want kind view, name --card, axis y", view)
	}

	if view.SubjectID != "both" {
		t.Fatalf("view SubjectID = %q, want %q", view.SubjectID, "both")
	}

	if view.ScrollerID != "" {
		t.Fatalf("view ScrollerID = %q, want the page scroller", view.ScrollerID)
	}

	if !near(view.InsetStart, pxToPt(20)) || !near(view.InsetEnd, pxToPt(10)) {
		t.Fatalf("view insets = (%v, %v), want (%v, %v)",
			view.InsetStart, view.InsetEnd, pxToPt(20), pxToPt(10))
	}
}

// placedTimelineElement finds one placed element by id.
func placedTimelineElement(t *testing.T, placed []PlacedElement, identifier string) PlacedElement {
	t.Helper()

	for _, element := range placed {
		if element.ID == identifier {
			return element
		}
	}

	t.Fatalf("placed element id=%q not found", identifier)

	return PlacedElement{}
}
