package layout_test

import (
	"reflect"
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

// boxWithStyle renders one element with an inline style and returns its box.
func boxWithStyle(t *testing.T, style string) layout.Box {
	t.Helper()

	page := `<!DOCTYPE html><html><body><div id="t" style="` + style + `">x</div></body></html>`

	return boxByID(t, renderDisplay(t, page, ""), "t")
}

// TestBoxTimelineScope covers timeline-scope: the initial none, all, a name
// list, an invalid value that is dropped, and the wide keywords.
func TestBoxTimelineScope(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		style string
		want  layout.TimelineScope
	}{
		{"initial", "", layout.TimelineScope{None: true}},
		{"none", "timeline-scope: none", layout.TimelineScope{None: true}},
		{"all", "timeline-scope: all", layout.TimelineScope{All: true}},
		{"one name", "timeline-scope: --card", layout.TimelineScope{Names: []string{"--card"}}},
		{"two names", "timeline-scope: --a, --b", layout.TimelineScope{Names: []string{"--a", "--b"}}},
		{"initial keyword", "timeline-scope: --a; timeline-scope: initial", layout.TimelineScope{None: true}},
		{"unset keyword", "timeline-scope: --a; timeline-scope: unset", layout.TimelineScope{None: true}},
		{
			"invalid name dropped",
			"timeline-scope: --a; timeline-scope: card",
			layout.TimelineScope{Names: []string{"--a"}},
		},
		{
			"none in list dropped",
			"timeline-scope: --a; timeline-scope: none, --b",
			layout.TimelineScope{Names: []string{"--a"}},
		},
		{
			"all in list dropped",
			"timeline-scope: --a; timeline-scope: all, --b",
			layout.TimelineScope{Names: []string{"--a"}},
		},
		{
			"trailing comma dropped",
			"timeline-scope: --a; timeline-scope: --b,",
			layout.TimelineScope{Names: []string{"--a"}},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := boxWithStyle(t, test.style).TimelineScope; !reflect.DeepEqual(got, test.want) {
				t.Fatalf("TimelineScope = %#v, want %#v", got, test.want)
			}
		})
	}
}

// TestBoxTimelineScopeInherit checks that timeline-scope does not inherit and
// that inherit copies the parent value.
func TestBoxTimelineScopeInherit(t *testing.T) {
	t.Parallel()

	const page = `<!DOCTYPE html><html><body>
		<div id="p" style="timeline-scope: --card">
			<div id="plain">a</div>
			<div id="copy" style="timeline-scope: inherit">b</div>
		</div></body></html>`

	display := renderDisplay(t, page, "")

	if got := boxByID(t, display, "plain").TimelineScope; !got.None {
		t.Fatalf("plain child TimelineScope = %#v, want none (not inherited)", got)
	}

	want := []string{"--card"}
	if got := boxByID(t, display, "copy").TimelineScope.Names; !reflect.DeepEqual(got, want) {
		t.Fatalf("inherit child Names = %v, want %v", got, want)
	}
}

// pointerOf returns the pointer timeline of a box, or false.
func pointerOf(box layout.Box) (layout.TimelineDef, bool) {
	for _, def := range box.Timelines {
		if def.Kind == "pointer" {
			return def, true
		}
	}

	return layout.TimelineDef{}, false
}

// TestBoxPointerTimeline covers pointer-timeline and its two longhands.
func TestBoxPointerTimeline(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		style    string
		wantName string
		wantAxis string
		wantNone bool
	}{
		{"initial", "", "", "", true},
		{"shorthand name only", "pointer-timeline: --ptr", "--ptr", "y", false},
		{"shorthand with x", "pointer-timeline: --ptr x", "--ptr", "x", false},
		{"shorthand inline", "pointer-timeline: --ptr inline", "--ptr", "x", false},
		{"shorthand none", "pointer-timeline: none", "", "", true},
		{"longhands", "pointer-timeline-name: --p; pointer-timeline-axis: x", "--p", "x", false},
		{"axis without name", "pointer-timeline-axis: x", "", "", true},
		{
			"invalid axis dropped",
			"pointer-timeline-name: --p; pointer-timeline-axis: x; pointer-timeline-axis: z",
			"--p",
			"x",
			false,
		},
		{"invalid name dropped", "pointer-timeline-name: --p; pointer-timeline-name: p", "--p", "y", false},
		{"list rejected", "pointer-timeline-name: --p; pointer-timeline-name: --a, --b", "--p", "y", false},
		{"shorthand resets axis", "pointer-timeline-axis: x; pointer-timeline: --p", "--p", "y", false},
		{"longhand after shorthand", "pointer-timeline: --p x; pointer-timeline-axis: y", "--p", "y", false},
		{"shorthand bad token", "pointer-timeline: --p; pointer-timeline: --q z", "--p", "y", false},
		{"initial keyword", "pointer-timeline: --p x; pointer-timeline: initial", "", "", true},
		{"name none", "pointer-timeline-name: --p; pointer-timeline-name: none", "", "", true},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			def, ok := pointerOf(boxWithStyle(t, test.style))
			if ok == test.wantNone {
				t.Fatalf("pointer timeline present = %t, want %t", ok, !test.wantNone)
			}

			if ok && (def.Name != test.wantName || def.Axis != test.wantAxis || def.SubjectID != "t") {
				t.Fatalf("pointer timeline = %#v, want name %q axis %q subject t", def, test.wantName, test.wantAxis)
			}
		})
	}
}

// TestBoxPointerTimelineNotInherited checks the pointer timeline is not
// inherited but copies on inherit, and sits after scroll and view in order.
func TestBoxPointerTimelineNotInherited(t *testing.T) {
	t.Parallel()

	const page = `<!DOCTYPE html><html><body>
		<div id="p" style="pointer-timeline: --ptr x; scroll-timeline: --s; view-timeline: --v">
			<div id="plain">a</div>
			<div id="copy" style="pointer-timeline-name: inherit; pointer-timeline-axis: inherit">b</div>
		</div></body></html>`

	display := renderDisplay(t, page, "")

	if _, ok := pointerOf(boxByID(t, display, "plain")); ok {
		t.Fatal("plain child has a pointer timeline, want none")
	}

	def, ok := pointerOf(boxByID(t, display, "copy"))
	if !ok || def.Name != "--ptr" || def.Axis != "x" {
		t.Fatalf("inherit child pointer timeline = %#v (%t), want --ptr on x", def, ok)
	}

	kinds := []string{}
	for _, def := range boxByID(t, display, "p").Timelines {
		kinds = append(kinds, def.Kind)
	}

	if want := []string{"scroll", "view", "pointer"}; !reflect.DeepEqual(kinds, want) {
		t.Fatalf("timeline kinds = %v, want %v", kinds, want)
	}
}

// TestBoxScrollAndViewTimelineCascade verifies the seven scroll and view
// properties through the public path: longhand and shorthand, wide keywords,
// invalid values dropped, and expansion order.
func TestBoxScrollAndViewTimelineCascade(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		style string
		kind  string
		want  layout.TimelineDef
		none  bool
	}{
		{
			"scroll shorthand",
			"scroll-timeline: --a x",
			"scroll",
			layout.TimelineDef{Name: "--a", Kind: "scroll", Axis: "x"},
			false,
		},
		{
			"scroll longhand order",
			"scroll-timeline-name: --a; scroll-timeline-axis: x",
			"scroll",
			layout.TimelineDef{Name: "--a", Kind: "scroll", Axis: "x"},
			false,
		},
		{
			"scroll shorthand resets axis",
			"scroll-timeline-axis: x; scroll-timeline: --a",
			"scroll",
			layout.TimelineDef{Name: "--a", Kind: "scroll", Axis: "y"},
			false,
		},
		{"scroll none", "scroll-timeline: --a; scroll-timeline-name: none", "scroll", layout.TimelineDef{}, true},
		{
			"scroll initial",
			"scroll-timeline: --a; scroll-timeline-name: initial",
			"scroll",
			layout.TimelineDef{},
			true,
		},
		{
			"scroll invalid dropped",
			"scroll-timeline-name: --a; scroll-timeline-name: a, b",
			"scroll",
			layout.TimelineDef{Name: "--a", Kind: "scroll", Axis: "y"},
			false,
		},
		{
			"view shorthand",
			"view-timeline: --v inline 10px 20px",
			"view",
			layout.TimelineDef{Name: "--v", Kind: "view", Axis: "x", SubjectID: "t", InsetStart: 10, InsetEnd: 20},
			false,
		},
		{
			"view longhands",
			"view-timeline-name: --v; view-timeline-axis: x; view-timeline-inset: 5px",
			"view",
			layout.TimelineDef{Name: "--v", Kind: "view", Axis: "x", SubjectID: "t", InsetStart: 5, InsetEnd: 5},
			false,
		},
		{
			"view inset auto",
			"view-timeline: --v; view-timeline-inset: auto",
			"view",
			layout.TimelineDef{Name: "--v", Kind: "view", Axis: "y", SubjectID: "t"},
			false,
		},
		{
			"view invalid inset dropped",
			"view-timeline: --v 4px; view-timeline-inset: 1px 2px 3px",
			"view",
			layout.TimelineDef{Name: "--v", Kind: "view", Axis: "y", SubjectID: "t", InsetStart: 4, InsetEnd: 4},
			false,
		},
		{"view unset", "view-timeline: --v; view-timeline: unset", "view", layout.TimelineDef{}, true},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var got layout.TimelineDef

			found := false

			for _, def := range boxWithStyle(t, test.style).Timelines {
				if def.Kind == test.kind {
					got, found = def, true
				}
			}

			if found == test.none {
				t.Fatalf("%s timeline present = %t, want %t", test.kind, found, !test.none)
			}

			if found && !reflect.DeepEqual(got, test.want) {
				t.Fatalf("timeline = %#v, want %#v", got, test.want)
			}
		})
	}
}
