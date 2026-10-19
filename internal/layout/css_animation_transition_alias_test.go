package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

const animationTransitionAliasHTML = `<html><body><div id="prefixed" class="prefixed"></div>` +
	`<div id="standard" class="standard"></div></body></html>`

type animationTransitionAliasCase struct {
	name      string
	prefixed  string
	standard  string
	value     string
	animation bool
}

func TestWebkitAnimationTransitionAliasesMatchStandardProperties(t *testing.T) {
	t.Parallel()
	runAnimationTransitionAliasCases(t, animationTransitionAliasCases())
}

func animationTransitionAliasCases() []animationTransitionAliasCase {
	cases := animationAliasCases()
	cases = append(cases, transitionAliasCases()...)

	return cases
}

func animationAliasCases() []animationTransitionAliasCase {
	return []animationTransitionAliasCase{
		{
			name: "animation shorthand", prefixed: "-webkit-animation", standard: "animation",
			value: "pulse 250ms ease-out 100ms 2 alternate both paused", animation: true,
		},
		{
			name: "animation delay", prefixed: "-webkit-animation-delay", standard: "animation-delay",
			value: "-100ms 50ms", animation: true,
		},
		{
			name: "animation direction", prefixed: "-webkit-animation-direction", standard: "animation-direction",
			value: "alternate-reverse", animation: true,
		},
		{
			name: "animation duration", prefixed: "-webkit-animation-duration", standard: "animation-duration",
			value: "250ms", animation: true,
		},
		{
			name: "animation fill mode", prefixed: "-webkit-animation-fill-mode", standard: "animation-fill-mode",
			value: "both", animation: true,
		},
		{
			name:     "animation iteration count",
			prefixed: "-webkit-animation-iteration-count",
			standard: "animation-iteration-count",
			value:    "2.5", animation: true,
		},
		{
			name: "animation name", prefixed: "-webkit-animation-name", standard: "animation-name",
			value: `"quoted name"`, animation: true,
		},
		{
			name: "animation play state", prefixed: "-webkit-animation-play-state", standard: "animation-play-state",
			value: "paused", animation: true,
		},
		{
			name:     "animation timing function",
			prefixed: "-webkit-animation-timing-function",
			standard: "animation-timing-function",
			value:    "steps(4, jump-end)", animation: true,
		},
	}
}

func transitionAliasCases() []animationTransitionAliasCase {
	return []animationTransitionAliasCase{
		{
			name: "transition shorthand", prefixed: "-webkit-transition", standard: "transition",
			value: "opacity 250ms ease-out 100ms allow-discrete",
		},
		{
			name: "transition delay", prefixed: "-webkit-transition-delay", standard: "transition-delay",
			value: "-100ms",
		},
		{
			name: "transition duration", prefixed: "-webkit-transition-duration", standard: "transition-duration",
			value: "250ms",
		},
		{
			name: "transition property", prefixed: "-webkit-transition-property", standard: "transition-property",
			value: "width, opacity",
		},
		{
			name:     "transition timing function",
			prefixed: "-webkit-transition-timing-function",
			standard: "transition-timing-function",
			value:    "steps(4, jump-end)",
		},
	}
}

func runAnimationTransitionAliasCases(t *testing.T, cases []animationTransitionAliasCase) {
	t.Helper()

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assertAnimationTransitionAliasMatches(t, testCase)
		})
	}
}

func assertAnimationTransitionAliasMatches(t *testing.T, testCase animationTransitionAliasCase) {
	t.Helper()

	root := mustParse(t, animationTransitionAliasHTML)
	styleSheet := sheet(t, ".prefixed { "+testCase.prefixed+": "+testCase.value+" }\n"+
		".standard { "+testCase.standard+": "+testCase.value+" }")
	styles := resolveStyles(root, []*css.Stylesheet{styleSheet}, "print", testViewport, 800)
	prefixed := styles[findAnimationTransitionAliasElement(t, root, "prefixed")]
	standard := styles[findAnimationTransitionAliasElement(t, root, "standard")]

	if testCase.animation {
		assertAnimationAliasStylesMatch(t, prefixed.Animation, standard.Animation)

		return
	}

	assertTransitionAliasStylesMatch(t, prefixed.Transition, standard.Transition)
}

func assertAnimationAliasStylesMatch(t *testing.T, prefixed, standard animationStyle) {
	t.Helper()

	if !prefixed.Set || !standard.Set {
		t.Fatalf("animation alias application = prefixed %t, standard %t; want both set", prefixed.Set, standard.Set)
	}

	if prefixed != standard {
		t.Errorf("prefixed Animation = %+v, standard Animation = %+v", prefixed, standard)
	}
}

func assertTransitionAliasStylesMatch(t *testing.T, prefixed, standard transitionStyle) {
	t.Helper()

	if !prefixed.Set || !standard.Set {
		t.Fatalf("transition alias application = prefixed %t, standard %t; want both set", prefixed.Set, standard.Set)
	}

	if prefixed != standard {
		t.Errorf("prefixed Transition = %+v, standard Transition = %+v", prefixed, standard)
	}
}

func TestWebkitAnimationTransitionAliasesRespectCascadeOrder(t *testing.T) {
	t.Parallel()
	runAnimationTransitionCascadeCases(t, animationTransitionCascadeCases())
}

type animationTransitionCascadeCase struct {
	name  string
	style string
	want  string
}

func animationTransitionCascadeCases() []animationTransitionCascadeCase {
	return []animationTransitionCascadeCase{
		{"prefixed animation longhand comes later", "animation-duration: 100ms; -webkit-animation-duration: 250ms", "250ms"},
		{"standard animation longhand comes later", "-webkit-animation-duration: 250ms; animation-duration: 100ms", "100ms"},
		{
			"prefixed transition longhand comes later",
			"transition-duration: 100ms; -webkit-transition-duration: 250ms",
			"250ms",
		},
		{
			"standard transition longhand comes later",
			"-webkit-transition-duration: 250ms; transition-duration: 100ms",
			"100ms",
		},
	}
}

func runAnimationTransitionCascadeCases(t *testing.T, cases []animationTransitionCascadeCase) {
	t.Helper()

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assertAnimationTransitionCascade(t, testCase)
		})
	}
}

func assertAnimationTransitionCascade(t *testing.T, testCase animationTransitionCascadeCase) {
	t.Helper()

	root := mustParse(t, `<html><body><div id="target"></div></body></html>`)
	node := findAnimationTransitionAliasElement(t, root, "target")
	node.Attrs["style"] = testCase.style
	styles := resolveStyles(root, nil, "print", testViewport, 800)
	style := styles[node]

	if strings.Contains(testCase.name, "animation") {
		if got := style.Animation.Duration; got != testCase.want {
			t.Errorf("Animation.Duration = %q, want %q", got, testCase.want)
		}

		return
	}

	if got := style.Transition.Duration; got != testCase.want {
		t.Errorf("Transition.Duration = %q, want %q", got, testCase.want)
	}
}

func findAnimationTransitionAliasElement(t *testing.T, root *html.Node, elementID string) *html.Node {
	t.Helper()

	if found := findAnimationTransitionAliasElementNode(root, elementID); found != nil {
		return found
	}

	t.Fatalf("element id=%q not found", elementID)

	return nil
}

func findAnimationTransitionAliasElementNode(root *html.Node, elementID string) *html.Node {
	if root == nil {
		return nil
	}

	if root.Type == html.ElementNode && root.Attribute("id") == elementID {
		return root
	}

	for _, child := range root.Children {
		if found := findAnimationTransitionAliasElementNode(child, elementID); found != nil {
			return found
		}
	}

	return nil
}
