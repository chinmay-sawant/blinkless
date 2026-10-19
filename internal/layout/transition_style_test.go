package layout

import (
	"reflect"
	"testing"
)

type transitionValueCase struct {
	prop  string
	value string
	want  bool
}

func TestTransitionDeclarationValueAccepted(t *testing.T) {
	t.Parallel()
	runTransitionValueCases(t, transitionValueCases())
}

func transitionValueCases() []transitionValueCase {
	return []transitionValueCase{
		{"transition-property", "width, opacity, --theme-color", true},
		{"transition-property", "all, opacity", true},
		{"transition-property", "none", true},
		{"transition-property", "none, width", false},
		{"transition-property", "initial", true},
		{"transition-property", "default", false},
		{"transition-property", "width,", false},
		{"transition-duration", "unset", true},
		{"transition-duration", "0s, 250ms, 1.25e2ms", true},
		{"transition-duration", "-1s", false},
		{"transition-duration", "1", false},
		{"transition-duration", "1 s", false},
		{"transition-duration", "1e308s", false},
		{"transition-delay", "-250ms, 0s", true},
		{"transition-delay", "NaNs", false},
		{"transition-timing-function", "ease-in, steps(4, jump-end)", true},
		{"transition-timing-function", "steps(0)", false},
		{"transition-behavior", "normal, allow-discrete", true},
		{"transition-behavior", "allow-animated", false},
		{"transition", "opacity 250ms ease-out 100ms allow-discrete", true},
		{"transition", "width -1s", false},
		{"transition", "none, opacity 1s", false},
	}
}

func runTransitionValueCases(t *testing.T, cases []transitionValueCase) {
	t.Helper()

	for _, testCase := range cases {
		t.Run(testCase.prop+"/"+testCase.value, func(t *testing.T) {
			t.Parallel()

			got, handled := transitionDeclarationValueAccepted(testCase.prop, testCase.value)
			if !handled {
				t.Fatalf("handled = false for transition property %q", testCase.prop)
			}

			if got != testCase.want {
				t.Fatalf("accepted = %t, want %t", got, testCase.want)
			}
		})
	}
}

func TestTransitionDeclarationValueAcceptedLeavesOtherProperties(t *testing.T) {
	t.Parallel()

	accepted, handled := transitionDeclarationValueAccepted("opacity", "0.5")
	if accepted || handled {
		t.Fatalf("(accepted, handled) = (%t, %t), want (false, false)", accepted, handled)
	}
}

type transitionShorthandCase struct {
	name  string
	value string
	want  map[string]string
	ok    bool
}

func TestExpandTransitionShorthand(t *testing.T) {
	t.Parallel()
	runTransitionShorthandCases(t, transitionShorthandCases())
}

func transitionShorthandCases() []transitionShorthandCase {
	cases := transitionShorthandDefaultsCases()
	cases = append(cases, transitionShorthandEasingCase())
	cases = append(cases, transitionShorthandWideCase())
	cases = append(cases, transitionShorthandInvalidCases()...)

	return cases
}

func transitionShorthandDefaultsCases() []transitionShorthandCase {
	return []transitionShorthandCase{
		{
			name:  "all components in arbitrary order",
			value: "opacity 250ms ease-out 100ms allow-discrete",
			want: map[string]string{
				"transition-property":        "opacity",
				"transition-duration":        "250ms",
				"transition-delay":           "100ms",
				"transition-timing-function": "ease-out",
				"transition-behavior":        "allow-discrete",
			},
			ok: true,
		},
		{
			name:  "defaults",
			value: "width",
			want: map[string]string{
				"transition-property":        "width",
				"transition-duration":        "0s",
				"transition-delay":           "0s",
				"transition-timing-function": "ease",
				"transition-behavior":        "normal",
			},
			ok: true,
		},
		{
			name:  "none is a valid single transition",
			value: "none 500ms",
			want: map[string]string{
				"transition-property":        "none",
				"transition-duration":        "500ms",
				"transition-delay":           "0s",
				"transition-timing-function": "ease",
				"transition-behavior":        "normal",
			},
			ok: true,
		},
	}
}

func transitionShorthandEasingCase() transitionShorthandCase {
	return transitionShorthandCase{
		name:  "comma items keep easing arguments together",
		value: "opacity steps(4, jump-end), transform 1s -50ms cubic-bezier(0.2, 0, 0.8, 1)",
		want: map[string]string{
			"transition-property":        "opacity, transform",
			"transition-duration":        "0s, 1s",
			"transition-delay":           "0s, -50ms",
			"transition-timing-function": "steps(4, jump-end), cubic-bezier(0.2, 0, 0.8, 1)",
			"transition-behavior":        "normal, normal",
		},
		ok: true,
	}
}

func transitionShorthandWideCase() transitionShorthandCase {
	return transitionShorthandCase{
		name:  "wide keyword",
		value: "initial",
		want: map[string]string{
			"transition-property":        "initial",
			"transition-duration":        "initial",
			"transition-delay":           "initial",
			"transition-timing-function": "initial",
			"transition-behavior":        "initial",
		},
		ok: true,
	}
}

func transitionShorthandInvalidCases() []transitionShorthandCase {
	return []transitionShorthandCase{
		{name: "none cannot share a list", value: "none, opacity 1s"},
		{name: "duplicate time", value: "width 1s 2s 3s"},
		{name: "duplicate property", value: "width height"},
		{name: "empty list item", value: "width 1s,"},
		{name: "invalid duration", value: "width -1s"},
	}
}

func runTransitionShorthandCases(t *testing.T, cases []transitionShorthandCase) {
	t.Helper()

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			declarations, ok := expandTransitionShorthand(testCase.value)
			if ok != testCase.ok {
				t.Fatalf("expandTransitionShorthand(%q) ok = %t, want %t", testCase.value, ok, testCase.ok)
			}

			if !ok {
				return
			}

			got := make(map[string]string, len(declarations))
			for _, declaration := range declarations {
				got[declaration.prop] = declaration.val
			}

			if !reflect.DeepEqual(got, testCase.want) {
				t.Fatalf("expanded declarations = %#v, want %#v", got, testCase.want)
			}
		})
	}
}

func TestApplyTransitionPropsStoresDefaults(t *testing.T) {
	t.Parallel()

	style := ResolvedStyle{}
	if !applyTransitionProps(&style, "transition", "width 250ms", 16, nil, nil, false) {
		t.Fatal("applyTransitionProps(transition) = false")
	}

	transition := style.Transition
	if !transition.Set || transition.Property != "width" || transition.Duration != "250ms" ||
		transition.Delay != "0s" || transition.TimingFunction != "ease" || transition.Behavior != "normal" {
		t.Fatalf("transition style = %#v", transition)
	}
}

func TestApplyTransitionPropsWideKeywords(t *testing.T) {
	t.Parallel()

	parent := ResolvedStyle{
		Transition: transitionStyle{
			Set:            true,
			Property:       "opacity",
			Duration:       "2s",
			Delay:          "-100ms",
			TimingFunction: "ease-in",
			Behavior:       "allow-discrete",
		},
	}
	style := ResolvedStyle{}

	if !applyTransitionProps(&style, "transition-duration", "inherit", 16, nil, &parent, true) {
		t.Fatal("applyTransitionProps(inherit) = false")
	}

	if style.Transition.Duration != "2s" {
		t.Fatalf("inherited duration = %q, want 2s", style.Transition.Duration)
	}

	if !applyTransitionProps(&style, "transition-behavior", "initial", 16, nil, &parent, true) {
		t.Fatal("applyTransitionProps(initial) = false")
	}

	if style.Transition.Behavior != "normal" {
		t.Fatalf("initial behavior = %q, want normal", style.Transition.Behavior)
	}
}

type easingAcceptanceCase struct {
	value string
	want  bool
}

func TestEasingValueAccepted(t *testing.T) {
	t.Parallel()
	runEasingAcceptanceCases(t, easingAcceptanceCases())
}

func easingAcceptanceCases() []easingAcceptanceCase {
	return []easingAcceptanceCase{
		{"linear", true},
		{"ease-in-out", true},
		{"step-start", true},
		{"cubic-bezier(0, 0.2, 1, 1)", true},
		{"cubic-bezier(0.2, -3, 0.8, 4)", true},
		{"steps(4)", true},
		{"steps(2, jump-none)", true},
		{"steps(10, jump-none)", true},
		{"steps(+2, jump-none)", true},
		{"steps(1, start)", true},
		{"linear(0, 0.5 20% 40%, 1 100%)", true},
		{"linear(1e-3 0%, 1.2 100%)", true},
		{"cubic-bezier(-0.1, 0, 0.5, 1)", false},
		{"cubic-bezier(0, 0, 1.1, 1)", false},
		{"cubic-bezier(NaN, 0, 1, 1)", false},
		{"cubic-bezier(0, 0, 1, 1, 2)", false},
		{"steps(0)", false},
		{"steps(-2)", false},
		{"steps(1, jump-none)", false},
		{"steps(2, middle)", false},
		{"steps(1.5)", false},
		{"steps(999999999999999999999999)", true},
		{"linear()", false},
		{"linear(0,)", false},
		{"linear(0 0% 20% 30%)", false},
		{"linear(1s, 2)", false},
		{"ease(0)", false},
	}
}

func runEasingAcceptanceCases(t *testing.T, cases []easingAcceptanceCase) {
	t.Helper()

	for _, testCase := range cases {
		t.Run(testCase.value, func(t *testing.T) {
			t.Parallel()

			if got := easingValueAccepted(testCase.value); got != testCase.want {
				t.Fatalf("easingValueAccepted(%q) = %t, want %t", testCase.value, got, testCase.want)
			}
		})
	}
}
