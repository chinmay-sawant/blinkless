package layout_test

import (
	"math"
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

// want is the part of a TimelineDef these tests compare.
type want struct{ name, kind, axis string }

func assertTimelines(t *testing.T, got []layout.TimelineDef, wants []want) {
	t.Helper()

	if len(got) != len(wants) {
		t.Fatalf("got %d timelines %#v, want %d %#v", len(got), got, len(wants), wants)
	}

	for idx, def := range got {
		if def.Name != wants[idx].name || def.Kind != wants[idx].kind || def.Axis != wants[idx].axis {
			t.Fatalf("timeline %d = %s %s %s, want %s %s %s",
				idx, def.Name, def.Kind, def.Axis, wants[idx].name, wants[idx].kind, wants[idx].axis)
		}
	}
}

// TestBoxTimelineLists covers the comma lists of scroll-timeline, view-timeline
// and pointer-timeline: how many timelines a box exports, in what order, and
// how the axis list lines up with the names.
func TestBoxTimelineLists(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		style string
		want  []want
	}{
		{"two names", "scroll-timeline-name: --a, --b", []want{{"--a", "scroll", "y"}, {"--b", "scroll", "y"}}},
		{
			"axis list repeats to the name list",
			"scroll-timeline-name: --a, --b, --c; scroll-timeline-axis: x, y",
			[]want{{"--a", "scroll", "x"}, {"--b", "scroll", "y"}, {"--c", "scroll", "x"}},
		},
		{
			"axis list longer than the names is cut",
			"scroll-timeline-name: --a; scroll-timeline-axis: x, y",
			[]want{{"--a", "scroll", "x"}},
		},
		{
			"a none item keeps its slot",
			"scroll-timeline-name: none, --b; scroll-timeline-axis: x, y",
			[]want{{"--b", "scroll", "y"}},
		},
		{"only none exports nothing", "scroll-timeline-name: none, none", nil},
		{
			"axis keywords are case insensitive",
			"scroll-timeline-name: --a, --b; scroll-timeline-axis: X, Y",
			[]want{{"--a", "scroll", "x"}, {"--b", "scroll", "y"}},
		},
		{
			"scroll shorthand list",
			"scroll-timeline: --a x, --b",
			[]want{{"--a", "scroll", "x"}, {"--b", "scroll", "y"}},
		},
		{
			"view shorthand list",
			"view-timeline: --a x, --b block",
			[]want{{"--a", "view", "x"}, {"--b", "view", "y"}},
		},
		{
			"pointer shorthand list",
			"pointer-timeline: --a x, --b",
			[]want{{"--a", "pointer", "x"}, {"--b", "pointer", "y"}},
		},
		{
			"pointer longhand lists",
			"pointer-timeline-name: --a, --b; pointer-timeline-axis: y, x",
			[]want{{"--a", "pointer", "y"}, {"--b", "pointer", "x"}},
		},
		{
			"order is scroll, then view, then pointer",
			"pointer-timeline: --p1 x, --p2; view-timeline: --v; scroll-timeline: --s1, --s2",
			[]want{
				{"--s1", "scroll", "y"}, {"--s2", "scroll", "y"}, {"--v", "view", "y"},
				{"--p1", "pointer", "x"}, {"--p2", "pointer", "y"},
			},
		},
		{
			"vertical writing mode maps block and inline per item",
			"writing-mode: vertical-rl; scroll-timeline: --a block, --b inline",
			[]want{{"--a", "scroll", "x"}, {"--b", "scroll", "y"}},
		},
		{
			"initial resets the list",
			"scroll-timeline-name: --a, --b; scroll-timeline-name: initial",
			nil,
		},
		{
			"a later valid list replaces an earlier one",
			"scroll-timeline-name: --a, --b; scroll-timeline-name: --c",
			[]want{{"--c", "scroll", "y"}},
		},
		{
			"trailing comma is dropped and the earlier list stays",
			"scroll-timeline-name: --a; scroll-timeline-name: --b,",
			[]want{{"--a", "scroll", "y"}},
		},
		{
			"empty item is dropped and the earlier list stays",
			"scroll-timeline-name: --a; scroll-timeline-name: --b,,--c",
			[]want{{"--a", "scroll", "y"}},
		},
		{
			"bad ident in a list drops the whole list",
			"scroll-timeline-name: --a; scroll-timeline-name: --b, c",
			[]want{{"--a", "scroll", "y"}},
		},
		{
			"bad axis in a list drops the whole list",
			"scroll-timeline-name: --a, --b; scroll-timeline-axis: x; scroll-timeline-axis: x, z",
			[]want{{"--a", "scroll", "x"}, {"--b", "scroll", "x"}},
		},
		{
			"bad shorthand item drops the whole shorthand",
			"scroll-timeline: --a x; scroll-timeline: --b, --c z",
			[]want{{"--a", "scroll", "x"}},
		},
		{
			"shorthand resets the axis list",
			"scroll-timeline-axis: x, x; scroll-timeline: --a, --b",
			[]want{{"--a", "scroll", "y"}, {"--b", "scroll", "y"}},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assertTimelines(t, boxWithStyle(t, test.style).Timelines, test.want)
		})
	}
}

// TestBoxTimelineListsInherit checks that inherit copies a whole list, and
// that an unset list is not inherited.
func TestBoxTimelineListsInherit(t *testing.T) {
	t.Parallel()

	const page = `<!DOCTYPE html><html><body>
		<div id="p" style="scroll-timeline: --a x, --b y; view-timeline: --v1 x, --v2 y 4px 6px;
			pointer-timeline: --p1 x, --p2 y">
			<div id="plain">a</div>
			<div id="copy" style="scroll-timeline-name: inherit; scroll-timeline-axis: inherit;
				view-timeline-name: inherit; view-timeline-axis: inherit; view-timeline-inset: inherit;
				pointer-timeline-name: inherit; pointer-timeline-axis: inherit">b</div>
			<div id="reset" style="scroll-timeline-name: unset; view-timeline-name: unset">c</div>
		</div></body></html>`

	display := renderDisplay(t, page, "")

	if got := boxByID(t, display, "plain").Timelines; len(got) != 0 {
		t.Fatalf("lists must not inherit by default, got %#v", got)
	}

	if got := boxByID(t, display, "reset").Timelines; len(got) != 0 {
		t.Fatalf("unset must reset the lists, got %#v", got)
	}

	assertTimelines(t, boxByID(t, display, "copy").Timelines, []want{
		{"--a", "scroll", "x"}, {"--b", "scroll", "y"},
		{"--v1", "view", "x"}, {"--v2", "view", "y"},
		{"--p1", "pointer", "x"}, {"--p2", "pointer", "y"},
	})

	views := boxByID(t, display, "copy").Timelines[2:4]
	if views[0].InsetStart != 0 || views[1].InsetStart < 3.9 || views[1].InsetEnd < 5.9 {
		t.Fatalf("inherited insets = %#v", views)
	}
}

func nearPx(got, want float64) bool { return math.Abs(got-want) < 0.01 }

// TestBoxViewTimelineInsetLists covers view-timeline-inset as a list: one
// item per timeline, repeating when it is shorter than the name list.
func TestBoxViewTimelineInsetLists(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		style string
		start []float64
		end   []float64
	}{
		{
			"one item per name",
			"view-timeline-name: --a, --b; view-timeline-inset: 20px 10px, auto",
			[]float64{20, 0}, []float64{10, 0},
		},
		{
			"short list repeats",
			"view-timeline-name: --a, --b, --c; view-timeline-inset: 5px",
			[]float64{5, 5, 5}, []float64{5, 5, 5},
		},
		{
			"two items repeat over three names",
			"view-timeline-name: --a, --b, --c; view-timeline-inset: 1px 2px, 3px 4px",
			[]float64{1, 3, 1}, []float64{2, 4, 2},
		},
		{
			"shorthand carries an inset per item",
			"view-timeline: --a x 20px 10px, --b inline auto",
			[]float64{20, 0}, []float64{10, 0},
		},
		{
			"axis and inset in either order in a shorthand item",
			"view-timeline: --a 7px 8px y, --b block 9px",
			[]float64{7, 9}, []float64{8, 9},
		},
		{
			"negative inset",
			"view-timeline-name: --a; view-timeline-inset: auto -10px",
			[]float64{0}, []float64{-10},
		},
		{
			"bad item drops the whole list",
			"view-timeline-name: --a, --b; view-timeline-inset: 3px; view-timeline-inset: 20px 10px, 1px 2px 3px",
			[]float64{3, 3}, []float64{3, 3},
		},
		{
			"trailing comma drops the list",
			"view-timeline-name: --a; view-timeline-inset: 4px; view-timeline-inset: 9px,",
			[]float64{4}, []float64{4},
		},
		{
			"none name keeps its inset slot",
			"view-timeline-name: none, --b; view-timeline-inset: 1px, 2px",
			[]float64{2}, []float64{2},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := boxWithStyle(t, test.style).Timelines
			if len(got) != len(test.start) {
				t.Fatalf("got %d timelines, want %d: %#v", len(got), len(test.start), got)
			}

			for idx, def := range got {
				if !nearPx(def.InsetStart, test.start[idx]) || !nearPx(def.InsetEnd, test.end[idx]) {
					t.Fatalf("timeline %d inset = %v %v, want %v %v",
						idx, def.InsetStart, def.InsetEnd, test.start[idx], test.end[idx])
				}
			}
		})
	}
}

// TestBoxViewTimelineInsetPercentPerAxis checks a percentage inset resolves
// against the scrollport on that item's own axis.
func TestBoxViewTimelineInsetPercentPerAxis(t *testing.T) {
	t.Parallel()

	got := boxWithStyle(t, "view-timeline: --y y 10%, --x x 10%").Timelines
	if len(got) != 2 {
		t.Fatalf("got %#v", got)
	}

	if got[0].InsetStart <= 0 || got[1].InsetStart <= 0 || nearPx(got[0].InsetStart, got[1].InsetStart) {
		t.Fatalf("percent insets must differ by axis, got y=%v x=%v", got[0].InsetStart, got[1].InsetStart)
	}
}

// TestTimelineScrollers covers which scroll container a timeline reads.
func TestTimelineScrollers(t *testing.T) {
	t.Parallel()

	const page = `<!DOCTYPE html><html><body>
		<div id="plain" style="scroll-timeline: --plain">a</div>
		<div id="sc" style="overflow:auto;height:50px;scroll-timeline: --own, --own2">
			<div id="inner" style="view-timeline: --v1">b</div>
			<div id="deep" style="overflow:hidden">
				<div id="deeper" style="view-timeline: --v2; pointer-timeline: --p">c</div>
			</div>
		</div>
		<div style="overflow:scroll">
			<div id="anon" style="view-timeline: --v3">d</div>
		</div>
		<div id="free" style="view-timeline: --v4">e</div>
		</body></html>`

	display := renderDisplay(t, page, "")

	type check struct {
		id        string
		index     int
		scroller  string
		anonymous bool
	}

	for _, test := range []check{
		{"plain", 0, "", false}, // not a scroll container: the page scrolls
		{"sc", 0, "sc", false},  // a scroll timeline on a scroll container reads that box
		{"sc", 1, "sc", false},  // every name in the list reads the same scroller
		{"inner", 0, "sc", false},
		{"deeper", 0, "deep", false}, // nearest scroll container wins, hidden counts
		{"anon", 0, "", true},        // a scroll container with no id cannot be named
		{"free", 0, "", false},       // no scroll container above: the page
	} {
		defs := boxByID(t, display, test.id).Timelines
		if test.index >= len(defs) {
			t.Fatalf("#%s has %d timelines, want index %d", test.id, len(defs), test.index)
		}

		if defs[test.index].ScrollerID != test.scroller || defs[test.index].ScrollerAnonymous != test.anonymous {
			t.Fatalf("#%s timeline %d scroller = %q anonymous %t, want %q %t", test.id, test.index,
				defs[test.index].ScrollerID, defs[test.index].ScrollerAnonymous, test.scroller, test.anonymous)
		}
	}

	pointer := boxByID(t, display, "deeper").Timelines[1]
	if pointer.Kind != "pointer" || pointer.ScrollerID != "" {
		t.Fatalf("a pointer timeline has no scroller, got %#v", pointer)
	}
}

// TestTimelineScrollerIgnoresRootAndBody checks that overflow on html or body
// is the page's own scroll, not a nested scroller.
func TestTimelineScrollerIgnoresRootAndBody(t *testing.T) {
	t.Parallel()

	const page = `<!DOCTYPE html><html style="overflow:auto"><body style="overflow:auto">
		<div id="a" style="view-timeline: --v">x</div></body></html>`

	def := boxByID(t, renderDisplay(t, page, ""), "a").Timelines[0]
	if def.ScrollerID != "" || def.ScrollerAnonymous {
		t.Fatalf("root and body overflow belong to the page, got %#v", def)
	}
}
