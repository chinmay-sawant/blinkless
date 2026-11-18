package layout_test

import (
	"reflect"
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

func normalEdge() layout.RangeBound { return layout.RangeBound{Normal: true} }

func autoEdge() layout.RangeBound { return layout.RangeBound{Auto: true} }

func pctEdge(name string, percent float64) layout.RangeBound {
	return layout.RangeBound{Name: name, Percent: percent, HasPercent: true}
}

// TestBoxTimelineTriggerShorthand covers the timeline-trigger shorthand: the
// example from the draft, the slash form, none, lists, and invalid values.
func TestBoxTimelineTriggerShorthand(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		style string
		want  []layout.TimelineTrigger
	}{
		{"initial", "", nil},
		{"none", "timeline-trigger: none", nil},
		{
			"scroll source", "timeline-trigger: --play scroll() 0% 100%",
			[]layout.TimelineTrigger{{
				Name: "--play", Source: "scroll()", Sources: []string{"scroll()"},
				ActivationStart: layout.RangeBound{Percent: 0, HasPercent: true},
				ActivationEnd:   layout.RangeBound{Percent: 100, HasPercent: true},
				ActiveStart:     autoEdge(), ActiveEnd: autoEdge(),
			}},
		},
		{
			"named range and active range", "timeline-trigger: --t view() entry 0% entry 100% / contain 0% contain 100%",
			[]layout.TimelineTrigger{{
				Name: "--t", Source: "view()", Sources: []string{"view()"},
				ActivationStart: pctEdge("entry", 0), ActivationEnd: pctEdge("entry", 100),
				ActiveStart: pctEdge("contain", 0), ActiveEnd: pctEdge("contain", 100),
			}},
		},
		{
			"one activation edge", "timeline-trigger: --t --page cover",
			[]layout.TimelineTrigger{{
				Name: "--t", Source: "--page", Sources: []string{"--page"},
				ActivationStart: layout.RangeBound{Name: "cover"}, ActivationEnd: normalEdge(),
				ActiveStart: autoEdge(), ActiveEnd: autoEdge(),
			}},
		},
		{
			"two items", "timeline-trigger: --a auto normal, --b none 10px normal",
			[]layout.TimelineTrigger{
				{
					Name: "--a", Source: "auto", Sources: []string{"auto"},
					ActivationStart: normalEdge(), ActivationEnd: normalEdge(), ActiveStart: autoEdge(), ActiveEnd: autoEdge(),
				},
				{
					Name: "--b", Source: "none", Sources: []string{"none"},
					ActivationStart: layout.RangeBound{Length: 10, HasLength: true}, ActivationEnd: normalEdge(),
					ActiveStart: autoEdge(), ActiveEnd: autoEdge(),
				},
			},
		},
		{
			"two references", "timeline-trigger: --t --a view() 0% 50%",
			[]layout.TimelineTrigger{{
				Name: "--t", Source: "--a view()", Sources: []string{"--a", "view()"},
				ActivationStart: pctEdge("", 0), ActivationEnd: pctEdge("", 50),
				ActiveStart: autoEdge(), ActiveEnd: autoEdge(),
			}},
		},
		{"missing source dropped", "timeline-trigger: --t 0% 100%", nil},
		{"bad name dropped", "timeline-trigger: play scroll() 0% 100%", nil},
		{"none in list dropped", "timeline-trigger: --a scroll() 0% 1%, none", nil},
		{"empty active range dropped", "timeline-trigger: --a scroll() 0% 1% /", nil},
		{"auto in activation dropped", "timeline-trigger: --a scroll() auto", nil},
		{"initial keyword", "timeline-trigger: --a scroll() 0% 1%; timeline-trigger: initial", nil},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := boxWithStyle(t, test.style).TimelineTriggers; !reflect.DeepEqual(got, test.want) {
				t.Fatalf("TimelineTriggers =\n%#v\nwant\n%#v", got, test.want)
			}
		})
	}
}

// TestBoxTimelineTriggerLonghands covers each longhand and list repetition.
func TestBoxTimelineTriggerLonghands(t *testing.T) {
	t.Parallel()

	got := boxWithStyle(t, "timeline-trigger-name: --a, --b, --c;"+
		"timeline-trigger-source: scroll(), --page;"+
		"timeline-trigger-activation-range-start: entry 25%, 10px;"+
		"timeline-trigger-activation-range-end: exit;"+
		"timeline-trigger-active-range-start: contain 0%;"+
		"timeline-trigger-active-range-end: normal, auto").TimelineTriggers

	if len(got) != 3 {
		t.Fatalf("triggers = %d, want 3 (one per name)", len(got))
	}

	wantSources := []string{"scroll()", "--page", "scroll()"}
	for i, trigger := range got {
		if trigger.Source != wantSources[i] {
			t.Errorf("trigger %d source = %q, want %q (sources repeat)", i, trigger.Source, wantSources[i])
		}

		if want := (layout.RangeBound{Name: "exit"}); trigger.ActivationEnd != want {
			t.Errorf("trigger %d activation end = %#v, want %#v", i, trigger.ActivationEnd, want)
		}

		if want := pctEdge("contain", 0); trigger.ActiveStart != want {
			t.Errorf("trigger %d active start = %#v, want %#v", i, trigger.ActiveStart, want)
		}
	}

	if want := pctEdge("entry", 25); got[0].ActivationStart != want || got[2].ActivationStart != want {
		t.Errorf("activation start 0 and 2 = %#v, %#v, want %#v", got[0].ActivationStart, got[2].ActivationStart, want)
	}

	if want := (layout.RangeBound{Length: 10, HasLength: true}); got[1].ActivationStart != want {
		t.Errorf("activation start 1 = %#v, want %#v", got[1].ActivationStart, want)
	}

	if got[0].ActiveEnd != normalEdge() || got[1].ActiveEnd != autoEdge() || got[2].ActiveEnd != normalEdge() {
		t.Errorf("active ends = %#v %#v %#v, want normal auto normal", got[0].ActiveEnd, got[1].ActiveEnd, got[2].ActiveEnd)
	}
}

// TestBoxTimelineTriggerNameAndSource covers name and source defaults, none,
// and invalid values.
func TestBoxTimelineTriggerNameAndSource(t *testing.T) {
	t.Parallel()

	only := boxWithStyle(t, "timeline-trigger-name: --play").TimelineTriggers
	if len(only) != 1 || only[0].Source != "auto" || only[0].ActivationStart != normalEdge() ||
		only[0].ActivationEnd != normalEdge() || only[0].ActiveStart != autoEdge() || only[0].ActiveEnd != autoEdge() {
		t.Fatalf("name only = %#v, want initial source auto, activation normal, active auto", only)
	}

	cases := map[string]string{
		"timeline-trigger-name: none":                                       "",
		"timeline-trigger-name: --a; timeline-trigger-name: none":           "",
		"timeline-trigger-name: --a; timeline-trigger-name: a":              "--a",
		"timeline-trigger-name: --a; timeline-trigger-name: none, --b":      "--a",
		"timeline-trigger-name: --a; timeline-trigger-name: initial":        "",
		"timeline-trigger-name: --a; timeline-trigger-name: unset":          "",
		"timeline-trigger-name: --a; timeline-trigger-name: --b; color:red": "--b",
	}

	for style, wantName := range cases {
		got := boxWithStyle(t, style).TimelineTriggers

		switch {
		case wantName == "" && got != nil:
			t.Errorf("%q = %#v, want no trigger", style, got)
		case wantName != "" && (len(got) != 1 || got[0].Name != wantName):
			t.Errorf("%q = %#v, want name %q", style, got, wantName)
		}
	}

	sources := map[string]string{
		"timeline-trigger-source: none":                                  "none",
		"timeline-trigger-source: auto":                                  "auto",
		"timeline-trigger-source: view()":                                "view()",
		"timeline-trigger-source: scroll(root x)":                        "scroll(root x)",
		"timeline-trigger-source: --a --b":                               "--a --b",
		"timeline-trigger-source: bogus":                                 "auto",
		"timeline-trigger-source: none --a":                              "auto",
		"timeline-trigger-source: view(bogus)":                           "auto",
		"timeline-trigger-source: --a, , --b":                            "auto",
		"timeline-trigger-source: --a; color: red":                       "--a",
		"timeline-trigger-source: --a; timeline-trigger-source: inherit": "auto",
	}

	for style, want := range sources {
		got := boxWithStyle(t, "timeline-trigger-name: --t;"+style).TimelineTriggers
		if len(got) != 1 || got[0].Source != want {
			t.Errorf("%q source = %#v, want %q", style, got, want)
		}
	}
}

// TestBoxTimelineTriggerRangeShorthands covers the two range shorthands and
// their expansion into the start and end longhands.
func TestBoxTimelineTriggerRangeShorthands(t *testing.T) {
	t.Parallel()

	got := boxWithStyle(t, "timeline-trigger-name: --t;"+
		"timeline-trigger-activation-range: entry 0% entry 100%;"+
		"timeline-trigger-active-range: contain 0% contain 100%").TimelineTriggers
	if len(got) != 1 {
		t.Fatalf("triggers = %d, want 1", len(got))
	}

	if got[0].ActivationStart != pctEdge("entry", 0) || got[0].ActivationEnd != pctEdge("entry", 100) ||
		got[0].ActiveStart != pctEdge("contain", 0) || got[0].ActiveEnd != pctEdge("contain", 100) {
		t.Fatalf("ranges = %#v", got[0])
	}

	// One value sets the start and leaves the end at its initial value.
	got = boxWithStyle(t, "timeline-trigger-name: --t;"+
		"timeline-trigger-activation-range-end: exit; timeline-trigger-activation-range: 25%;"+
		"timeline-trigger-active-range-end: exit; timeline-trigger-active-range: cover").TimelineTriggers
	if got[0].ActivationStart != pctEdge("", 25) || got[0].ActivationEnd != normalEdge() ||
		got[0].ActiveStart != (layout.RangeBound{Name: "cover"}) || got[0].ActiveEnd != autoEdge() {
		t.Fatalf("one-value ranges = %#v", got[0])
	}

	// A longhand after the shorthand wins, a shorthand after a longhand resets.
	got = boxWithStyle(t, "timeline-trigger-name: --t;"+
		"timeline-trigger-activation-range: 10% 20%; timeline-trigger-activation-range-end: 90%").TimelineTriggers
	if got[0].ActivationStart != pctEdge("", 10) || got[0].ActivationEnd != pctEdge("", 90) {
		t.Fatalf("longhand after shorthand = %#v", got[0])
	}

	// Invalid values leave the earlier declaration alone.
	got = boxWithStyle(t, "timeline-trigger-name: --t;"+
		"timeline-trigger-activation-range: 10% 20%; timeline-trigger-activation-range: 1% 2% 3%;"+
		"timeline-trigger-active-range: 10% 20%; timeline-trigger-active-range: bogus").TimelineTriggers
	if got[0].ActivationEnd != pctEdge("", 20) || got[0].ActiveEnd != pctEdge("", 20) {
		t.Fatalf("invalid range shorthand changed the value: %#v", got[0])
	}

	// auto is legal on the active range and not on the activation range.
	got = boxWithStyle(t, "timeline-trigger-name: --t;"+
		"timeline-trigger-activation-range-start: 5%; timeline-trigger-activation-range-start: auto;"+
		"timeline-trigger-active-range-start: 5%; timeline-trigger-active-range-start: auto").TimelineTriggers
	if got[0].ActivationStart != pctEdge("", 5) || got[0].ActiveStart != autoEdge() {
		t.Fatalf("auto handling = %#v", got[0])
	}
}

// TestBoxTimelineTriggerLengths checks percentages versus lengths, calc, and
// relative units, so 25% is never read as 25px.
func TestBoxTimelineTriggerLengths(t *testing.T) {
	t.Parallel()

	edge := func(value string) layout.RangeBound {
		got := boxWithStyle(t, "font-size: 10px; timeline-trigger-name: --t; timeline-trigger-activation-range-start: "+
			value).TimelineTriggers

		if len(got) != 1 {
			t.Fatalf("%q: triggers = %d, want 1", value, len(got))
		}

		return got[0].ActivationStart
	}

	if got := edge("25%"); got != pctEdge("", 25) {
		t.Errorf("25%% = %#v, want a percentage", got)
	}

	if got := edge("25px"); got != (layout.RangeBound{Length: 25, HasLength: true}) {
		t.Errorf("25px = %#v, want a length", got)
	}

	if got := edge("2em"); got != (layout.RangeBound{Length: 20, HasLength: true}) {
		t.Errorf("2em = %#v, want 20px", got)
	}

	if got := edge("0"); got != (layout.RangeBound{HasLength: true}) {
		t.Errorf("0 = %#v, want a zero length", got)
	}

	mixed := edge("calc(50% + 10px)")
	if !mixed.HasPercent || !mixed.HasLength || mixed.Percent < 49.99 || mixed.Percent > 50.01 ||
		mixed.Length < 9.99 || mixed.Length > 10.01 {
		t.Errorf("calc(50%% + 10px) = %#v, want 50%% and 10px", mixed)
	}

	if got := edge("cover 10px"); got != (layout.RangeBound{Name: "cover", Length: 10, HasLength: true}) {
		t.Errorf("cover 10px = %#v", got)
	}
}

// TestBoxTimelineTriggerNotInherited checks the trigger family does not
// inherit and that inherit copies the parent value.
func TestBoxTimelineTriggerNotInherited(t *testing.T) {
	t.Parallel()

	const page = `<!DOCTYPE html><html><body>
		<div id="p" style="timeline-trigger: --play scroll() 0% 100%; event-trigger: --press click">
			<div id="plain">a</div>
			<div id="copy" style="timeline-trigger-name: inherit; timeline-trigger-source: inherit;
				timeline-trigger-activation-range-start: inherit; timeline-trigger-activation-range-end: inherit;
				event-trigger-name: inherit; event-trigger-source: inherit">b</div>
		</div></body></html>`

	display := renderDisplay(t, page, "")

	plain := boxByID(t, display, "plain")
	if plain.TimelineTriggers != nil || plain.EventTriggers != nil {
		t.Fatalf("plain child inherited a trigger: %#v %#v", plain.TimelineTriggers, plain.EventTriggers)
	}

	copied := boxByID(t, display, "copy")
	if len(copied.TimelineTriggers) != 1 || copied.TimelineTriggers[0].Name != "--play" ||
		copied.TimelineTriggers[0].Source != "scroll()" ||
		copied.TimelineTriggers[0].ActivationEnd != (layout.RangeBound{Percent: 100, HasPercent: true}) {
		t.Fatalf("inherit child timeline triggers = %#v", copied.TimelineTriggers)
	}

	if len(copied.EventTriggers) != 1 || copied.EventTriggers[0].Name != "--press" {
		t.Fatalf("inherit child event triggers = %#v", copied.EventTriggers)
	}
}

// TestBoxTimelineTriggerFixturePair reproduces the scroll fixture element
// trigger-range-pair: a shorthand with no range, then the two range
// shorthands overriding it.
func TestBoxTimelineTriggerFixturePair(t *testing.T) {
	t.Parallel()

	got := boxWithStyle(t, "timeline-trigger:--show3 scroll();timeline-trigger-activation-range:25% 75%;"+
		"timeline-trigger-active-range:cover 0% cover 100%").TimelineTriggers
	want := []layout.TimelineTrigger{{
		Name: "--show3", Source: "scroll()", Sources: []string{"scroll()"},
		ActivationStart: pctEdge("", 25), ActivationEnd: pctEdge("", 75),
		ActiveStart: pctEdge("cover", 0), ActiveEnd: pctEdge("cover", 100),
	}}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("TimelineTriggers = %#v, want %#v", got, want)
	}

	// The shorthand alone, with no range, still declares the trigger.
	alone := boxWithStyle(t, "timeline-trigger:--show3 scroll()").TimelineTriggers
	if len(alone) != 1 || alone[0].ActivationStart != normalEdge() || alone[0].ActiveEnd != autoEdge() {
		t.Fatalf("shorthand without a range = %#v, want one trigger with initial ranges", alone)
	}
}
