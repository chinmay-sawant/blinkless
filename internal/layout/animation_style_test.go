package layout

import (
	"reflect"
	"testing"
)

type animationAcceptanceCase struct {
	prop  string
	value string
	want  bool
}

func TestAnimationDeclarationValueAccepted(t *testing.T) {
	t.Parallel()
	assertAnimationDeclarationCases(t, animationBaseAcceptanceCases())
}

func TestAnimationExtendedDeclarationValueAccepted(t *testing.T) {
	t.Parallel()
	assertAnimationDeclarationCases(t, animationExtendedAcceptanceCases())
}

func animationBaseAcceptanceCases() []animationAcceptanceCase {
	return []animationAcceptanceCase{
		{prop: "animation-duration", value: "auto, 250ms", want: true},
		{prop: "animation", value: "slide", want: true},
		{prop: "animation-duration", value: "-1s", want: false},
		{prop: "animation-duration", value: "1", want: false},
		{prop: "animation-timing-function", value: "ease-in, steps(4, jump-end)", want: true},
		{prop: "animation-timing-function", value: "steps(0)", want: false},
		{prop: "animation-iteration-count", value: "2.5, infinite", want: true},
		{prop: "animation-iteration-count", value: "-1", want: false},
		{prop: "animation-direction", value: "alternate-reverse", want: true},
		{prop: "animation-direction", value: "sideways", want: false},
		{prop: "animation-fill-mode", value: "backwards, both", want: true},
		{prop: "animation-fill-mode", value: "forward", want: false},
		{prop: "animation-play-state", value: "paused", want: true},
		{prop: "animation-play-state", value: "stopped", want: false},
		{prop: "animation-composition", value: "add, accumulate", want: true},
		{prop: "animation-composition", value: "merge", want: false},
		{prop: "animation-delay-start", value: "-250ms, 1s", want: true},
		{prop: "animation-delay-end", value: "0s, 2s", want: true},
		{prop: "animation-delay-end", value: "-1ms", want: false},
		{prop: "animation-name", value: "none, slide-in, \"quoted name\"", want: true},
		{prop: "animation-name", value: "bus", want: true},
		{prop: "animation-name", value: "none, initial", want: false},
	}
}

//nolint:dupl // shares the acceptance-case shape with the timeline grammar.
func animationExtendedAcceptanceCases() []animationAcceptanceCase {
	return []animationAcceptanceCase{
		{prop: "animation-timeline", value: "auto, --scroll, scroll(root x), view(block 10%)", want: true},
		{prop: "animation-timeline", value: "scroll(root nearest)", want: false},
		{prop: "animation-range-start", value: "normal, entry 25%, 10%", want: true},
		{prop: "animation-range-end", value: "exit, cover calc(100% - 10px)", want: true},
		{prop: "animation-range-center", value: "normal, source 25%, target", want: true},
		{prop: "animation-range-center", value: "middle", want: false},
		{prop: "animation-trigger", value: "--ready play, --done pause reset", want: true},
		{prop: "animation-trigger", value: "ready play", want: false},
		{prop: "trigger-scope", value: "all", want: true},
		{prop: "trigger-scope", value: "--local, --nested", want: true},
		{prop: "trigger-scope", value: "all, --local", want: false},
		{prop: "trigger-scope", value: "local", want: false},
		{prop: "view-transition-name", value: "none", want: true},
		{prop: "view-transition-name", value: "card", want: true},
		{prop: "view-transition-name", value: "card, other", want: false},
		{prop: "view-transition-name", value: "auto", want: false},
		{prop: "view-transition-class", value: "card compact", want: true},
		{prop: "view-transition-class", value: "card none", want: false},
		{prop: "view-transition-class", value: "card, compact", want: false},
		{prop: "view-transition-group", value: "normal", want: true},
		{prop: "view-transition-group", value: "parent", want: true},
		{prop: "view-transition-group", value: "none", want: true},
		{prop: "view-transition-group", value: "normal, contain", want: false},
		{prop: "view-transition-group", value: "unknown", want: true},
		{prop: "view-transition-scope", value: "none", want: true},
		{prop: "view-transition-scope", value: "all", want: true},
		{prop: "view-transition-scope", value: "none, all", want: false},
		{prop: "view-transition-scope", value: "parent", want: false},
		{prop: "animation", value: "slide-in 250ms ease-out 100ms 2 alternate both paused", want: true},
		{prop: "animation", value: "slide-in 250ms nowhere", want: false},
		{prop: "animation", value: "slide-in 250ms add", want: false},
	}
}

func assertAnimationDeclarationCases(t *testing.T, tests []animationAcceptanceCase) {
	t.Helper()

	for _, test := range tests {
		t.Run(test.prop+"/"+test.value, func(t *testing.T) {
			t.Parallel()

			got, handled := animationDeclarationValueAccepted(test.prop, test.value)
			if !handled {
				t.Fatalf("handled = false for animation property %q", test.prop)
			}

			if got != test.want {
				t.Fatalf("accepted = %t, want %t", got, test.want)
			}
		})
	}
}

func TestAnimationDeclarationValueAcceptedLeavesOtherProperties(t *testing.T) {
	t.Parallel()

	accepted, handled := animationDeclarationValueAccepted("opacity", "0.5")
	if accepted || handled {
		t.Fatalf("(accepted, handled) = (%t, %t), want (false, false)", accepted, handled)
	}
}

func TestApplyAnimationPropsStoresCommaLists(t *testing.T) {
	t.Parallel()

	style := ResolvedStyle{}
	declarations := []struct {
		prop  string
		value string
	}{
		{prop: "animation-duration", value: "250ms, auto"},
		{prop: "animation-name", value: "slide-in, \"fade out\""},
		{prop: "animation-range-start", value: "entry 10%, calc(100% - 10px)"},
	}

	for _, declaration := range declarations {
		if !applyAnimationProps(&style, declaration.prop, declaration.value, 16, nil, nil, false) {
			t.Fatalf("applyAnimationProps(%q) = false", declaration.prop)
		}
	}

	if !style.Animation.Set {
		t.Fatal("Animation.Set = false, want true")
	}

	if style.Animation.Duration != "250ms, auto" {
		t.Errorf("Animation.Duration = %q, want %q", style.Animation.Duration, "250ms, auto")
	}

	if style.Animation.Name != "slide-in, \"fade out\"" {
		t.Errorf("Animation.Name = %q, want %q", style.Animation.Name, "slide-in, \"fade out\"")
	}

	if style.Animation.RangeStart != "entry 10%, calc(100% - 10px)" {
		t.Errorf("Animation.RangeStart = %q", style.Animation.RangeStart)
	}
}

func TestApplyAnimationPropsCSSWideKeywords(t *testing.T) {
	t.Parallel()

	parent := ResolvedStyle{
		Animation: animationStyle{Set: true, Duration: "2s", Name: "parent-animation"},
	}
	style := ResolvedStyle{}

	if !applyAnimationProps(&style, "animation-duration", "inherit", 16, nil, &parent, true) {
		t.Fatal("applyAnimationProps(inherit) = false")
	}

	if style.Animation.Duration != "2s" {
		t.Fatalf("inherited duration = %q, want 2s", style.Animation.Duration)
	}

	if !applyAnimationProps(&style, "animation-name", "initial", 16, nil, &parent, true) {
		t.Fatal("applyAnimationProps(initial) = false")
	}

	if style.Animation.Name != "none" {
		t.Errorf("initial animation-name = %q, want none", style.Animation.Name)
	}

	if !applyAnimationProps(&style, "animation-duration", "initial", 16, nil, &parent, true) {
		t.Fatal("applyAnimationProps(animation-duration: initial) = false")
	}

	if style.Animation.Duration != "auto" {
		t.Errorf("initial animation-duration = %q, want auto", style.Animation.Duration)
	}
}

func TestApplyAnimationPropsShorthandDefaultsDurationToAuto(t *testing.T) {
	t.Parallel()

	style := ResolvedStyle{}
	if !applyAnimationProps(&style, "animation", "slide", 16, nil, nil, false) {
		t.Fatal("applyAnimationProps(animation: slide) = false")
	}

	if style.Animation.Duration != "auto" {
		t.Fatalf("animation duration = %q, want auto", style.Animation.Duration)
	}

	records := animationRecordsFromStyle(style.Animation)

	if len(records) != 1 || !records[0].DurationAuto {
		t.Fatalf("animation records = %#v, want one auto-duration record", records)
	}
}

func TestApplyAnimationPropsInheritsViewTransitionWithoutAnimationRecord(t *testing.T) {
	t.Parallel()

	parent := ResolvedStyle{
		Animation: animationStyle{ViewTransitionName: "parent-transition"},
	}
	style := ResolvedStyle{}

	if !applyAnimationProps(&style, "view-transition-name", "inherit", 16, nil, &parent, true) {
		t.Fatal("applyAnimationProps(view-transition-name: inherit) = false")
	}

	if style.Animation.ViewTransitionName != "parent-transition" {
		t.Fatalf("inherited view-transition-name = %q, want %q", style.Animation.ViewTransitionName, "parent-transition")
	}
}

type animationExpansionCase struct {
	name  string
	prop  string
	value string
	want  map[string]string
	ok    bool
}

func TestExpandAnimationDeclaration(t *testing.T) {
	t.Parallel()
	runAnimationExpansionCases(t, animationExpansionCases())
}

func animationExpansionCases() []animationExpansionCase {
	cases := animationShorthandExpansionCases()
	cases = append(cases, animationShorthandListExpansionCases()...)
	cases = append(cases, animationDelayExpansionCases()...)
	cases = append(cases, animationRangeExpansionCases()...)
	cases = append(cases, animationWideAndInvalidExpansionCases()...)

	return cases
}

func animationShorthandExpansionCases() []animationExpansionCase {
	return []animationExpansionCase{
		{
			name:  "animation components and reset-only longhands",
			prop:  "animation",
			value: "slide-in 2s ease-in 250ms 3 alternate both paused --scroll",
			want: map[string]string{
				"animation-name":            "slide-in",
				"animation-duration":        "2s",
				"animation-timing-function": "ease-in",
				"animation-delay-start":     "250ms",
				"animation-iteration-count": "3",
				"animation-direction":       "alternate",
				"animation-fill-mode":       "both",
				"animation-play-state":      "paused",
				"animation-timeline":        "--scroll",
				"animation-range-start":     "normal",
				"animation-range-end":       "normal",
				"animation-trigger":         "none",
			},
			ok: true,
		},
		{
			name:  "omitted shorthand duration defaults to auto",
			prop:  "animation",
			value: "slide",
			want: map[string]string{
				"animation-name":            "slide",
				"animation-duration":        "auto",
				"animation-timing-function": "ease",
				"animation-delay-start":     "0s",
				"animation-iteration-count": "1",
				"animation-direction":       "normal",
				"animation-fill-mode":       "none",
				"animation-play-state":      "running",
				"animation-timeline":        "auto",
				"animation-range-start":     "normal",
				"animation-range-end":       "normal",
				"animation-trigger":         "none",
			},
			ok: true,
		},
	}
}

func animationShorthandListExpansionCases() []animationExpansionCase {
	return []animationExpansionCase{{
		name:  "animation shorthand list values",
		prop:  "animation",
		value: "fade 1s, spin 2s",
		want: map[string]string{
			"animation-name":            "fade, spin",
			"animation-duration":        "1s, 2s",
			"animation-timing-function": "ease, ease",
			"animation-delay-start":     "0s, 0s",
			"animation-iteration-count": "1, 1",
			"animation-direction":       "normal, normal",
			"animation-fill-mode":       "none, none",
			"animation-play-state":      "running, running",
			"animation-timeline":        "auto, auto",
			"animation-range-start":     "normal, normal",
			"animation-range-end":       "normal, normal",
			"animation-trigger":         "none, none",
		},
		ok: true,
	}}
}

func animationDelayExpansionCases() []animationExpansionCase {
	return []animationExpansionCase{{
		name:  "delay resets omitted end delay",
		prop:  "animation-delay",
		value: "-100ms 50ms, 2s",
		want: map[string]string{
			"animation-delay-start": "-100ms, 2s",
			"animation-delay-end":   "50ms, 0s",
		},
		ok: true,
	}}
}

func animationRangeExpansionCases() []animationExpansionCase {
	return []animationExpansionCase{
		{
			name:  "range shorthand defaults named end",
			prop:  "animation-range",
			value: "entry 10%, 20% 80%",
			want: map[string]string{
				"animation-range-start": "entry 10%, 20%",
				"animation-range-end":   "entry 100%, 80%",
			},
			ok: true,
		},
		{
			name:  "range shorthand pairs named ranges",
			prop:  "animation-range",
			value: "entry exit 90%",
			want: map[string]string{
				"animation-range-start": "entry 0%",
				"animation-range-end":   "exit 90%",
			},
			ok: true,
		},
	}
}

func animationWideAndInvalidExpansionCases() []animationExpansionCase {
	return []animationExpansionCase{
		{
			name:  "wide keyword expands to all animation subproperties",
			prop:  "animation",
			value: "unset",
			want: map[string]string{
				"animation-name":            "unset",
				"animation-duration":        "unset",
				"animation-timing-function": "unset",
				"animation-delay-start":     "unset",
				"animation-iteration-count": "unset",
				"animation-direction":       "unset",
				"animation-fill-mode":       "unset",
				"animation-play-state":      "unset",
				"animation-timeline":        "unset",
				"animation-range-start":     "unset",
				"animation-range-end":       "unset",
				"animation-trigger":         "unset",
			},
			ok: true,
		},
		{name: "invalid duplicate component", prop: "animation", value: "slide 1s 2s 3s", ok: false},
		{name: "composition not in pinned animation shorthand", prop: "animation", value: "throb 400ms add", ok: false},
		{name: "unowned shorthand", prop: "transition", value: "opacity 1s", ok: false},
	}
}

func runAnimationExpansionCases(t *testing.T, cases []animationExpansionCase) {
	t.Helper()

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			declarations, gotOK := expandAnimationDeclaration(test.prop, test.value)
			if gotOK != test.ok {
				t.Fatalf("ok = %t, want %t", gotOK, test.ok)
			}

			if !test.ok {
				return
			}

			got := make(map[string]string, len(declarations))
			for _, declaration := range declarations {
				got[declaration.prop] = declaration.val
			}

			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("expanded declarations = %#v, want %#v", got, test.want)
			}
		})
	}
}
