package layout

import "testing"

// TestTriggerDeclarationValueAccepted pins the cascade gate for the scope,
// pointer, trigger, interest, and interactivity properties.
func TestTriggerDeclarationValueAccepted(t *testing.T) {
	t.Parallel()

	cases := []struct {
		prop, value string
		want        bool
	}{
		{"timeline-scope", "none", true},
		{"timeline-scope", "all", true},
		{"timeline-scope", "--a, --b", true},
		{"timeline-scope", "all, --a", false},
		{"timeline-scope", "a", false},
		{"pointer-timeline", "--p x", true},
		{"pointer-timeline", "x --p", false},
		{"pointer-timeline-name", "none", true},
		{"pointer-timeline-axis", "inline", true},
		{"pointer-timeline-axis", "z", false},
		{"timeline-trigger", "none", true},
		{"timeline-trigger", "--a scroll() 0% 100% / 10% 90%", true},
		{"timeline-trigger", "--a scroll() 0% 100% / ", false},
		{"timeline-trigger", "--a 0% 100%", false},
		{"timeline-trigger", "--a scroll()", true},
		{"timeline-trigger-name", "none", true},
		{"timeline-trigger-name", "none, --a", false},
		{"timeline-trigger-source", "scroll(root x), view(), --a", true},
		{"timeline-trigger-source", "scroll(bogus)", false},
		{"timeline-trigger-activation-range", "entry 10% exit 20px", true},
		{"timeline-trigger-activation-range", "auto", false},
		{"timeline-trigger-activation-range-start", "normal", true},
		{"timeline-trigger-activation-range-end", "cover", true},
		{"timeline-trigger-activation-range-end", "10% 20%", false},
		{"timeline-trigger-active-range", "auto normal", true},
		{"timeline-trigger-active-range-start", "auto", true},
		{"timeline-trigger-active-range-end", "bogus", false},
		{"event-trigger", "--a click / keydown(Enter)", true},
		{"event-trigger", "--a", false},
		{"event-trigger-name", "--a, --b", true},
		{"event-trigger-source", "none", true},
		{"event-trigger-source", "click focus / blur, none", true},
		{"event-trigger-source", "none click", false},
		{"interest-delay", "1s", true},
		{"interest-delay", "1s normal", true},
		{"interest-delay", "1s 2s 3s", false},
		{"interest-delay-start", "normal", true},
		{"interest-delay-end", "-1ms", false},
		{"interactivity", "inert", true},
		{"interactivity", "none", false},
		{"interactivity", "inherit", true},
		{"timeline-trigger-source", "var(--x)", true},
	}

	for _, test := range cases {
		got, handled := timelineDeclarationValueAccepted(test.prop, test.value)
		if !handled || got != test.want {
			t.Errorf("%s: %q = (%t, %t), want (%t, true)", test.prop, test.value, got, handled, test.want)
		}
	}
}

// TestTriggerShorthandThroughVar covers the var() path, where a shorthand is
// substituted after the cascade and expands in the applier.
func TestTriggerShorthandThroughVar(t *testing.T) {
	t.Parallel()

	style := styleForDecl(t, "--t: --play scroll() 10% 20%; timeline-trigger: var(--t);"+
		"--e: --press click / blur; event-trigger: var(--e);"+
		"--d: 1s 2s; interest-delay: var(--d);--p: --ptr x; pointer-timeline: var(--p);"+
		"--r: entry 0% exit 100%; timeline-trigger-activation-range: var(--r)")

	if style.Triggers.TriggerName != "--play" || style.Triggers.TriggerSource != "scroll()" {
		t.Errorf("trigger name/source = %q/%q", style.Triggers.TriggerName, style.Triggers.TriggerSource)
	}

	if style.Triggers.EventName != "--press" || style.Triggers.EventSource != "click / blur" {
		t.Errorf("event name/source = %q/%q", style.Triggers.EventName, style.Triggers.EventSource)
	}

	if style.Triggers.InterestStart != "1000" || style.Triggers.InterestEnd != "2000" {
		t.Errorf("interest = %q/%q", style.Triggers.InterestStart, style.Triggers.InterestEnd)
	}

	if style.Triggers.PointerName != "--ptr" || style.Triggers.PointerAxis != "x" {
		t.Errorf("pointer = %q/%q", style.Triggers.PointerName, style.Triggers.PointerAxis)
	}

	if style.Triggers.ActivationStart != "entry 0 -" || style.Triggers.ActivationEnd != "exit 100 -" {
		t.Errorf("activation = %q/%q", style.Triggers.ActivationStart, style.Triggers.ActivationEnd)
	}
}

// TestPlacedTriggerExport checks the engine-side payloads on a placed
// element and that an element with no declaration exports the initial values.
func TestPlacedTriggerExport(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">
		<div id="plain">plain</div>
		<div id="all" style="timeline-scope: --a; timeline-trigger: --t view() cover 10% / contain;
			event-trigger: --e click / blur; interest-delay: 3s; interactivity: inert">x</div>
	</body></html>`)

	placed := PlacedElements(res)

	plain := placedTimelineElement(t, placed, "plain")
	initial := plain.Triggers
	if !initial.TimelineScope.None || initial.TimelineTriggers != nil || initial.EventTriggers != nil ||
		!initial.InterestDelay.StartNormal || !initial.InterestDelay.EndNormal || initial.Interactivity != "auto" {
		t.Fatalf("plain element exports non-initial values: %#v", plain)
	}

	full := placedTimelineElement(t, placed, "all")
	got := full.Triggers
	if len(got.TimelineScope.Names) != 1 || len(got.TimelineTriggers) != 1 || len(got.EventTriggers) != 1 ||
		got.InterestDelay.StartMs != 3000 || got.Interactivity != "inert" {
		t.Fatalf("declared element exports = %#v", full)
	}

	trigger := full.Triggers.TimelineTriggers[0]
	if trigger.ActivationStart.Name != "cover" || !trigger.ActivationStart.HasPercent ||
		trigger.ActivationEnd.Name != "" || !trigger.ActivationEnd.Normal ||
		trigger.ActiveStart.Name != "contain" || trigger.ActiveStart.HasPercent {
		t.Fatalf("trigger ranges = %#v", trigger)
	}
}
