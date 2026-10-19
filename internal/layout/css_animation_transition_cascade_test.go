package layout

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

const animationCascadeDuration = "250ms"

func TestAnimationTransitionCascadeRoundTrips(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div class="target"></div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.target {
			animation-name: slide;
			animation-duration: `+animationCascadeDuration+`;
			animation-timing-function: ease-in;
			transition-property: opacity;
			transition-duration: 400ms;
			transition-timing-function: steps(4, jump-end);
		}
	`)}, "print", testViewport, 800)
	target := findElementByName(root, "div")
	style := styles[target]

	if !style.Animation.Set || style.Animation.Name != "slide" ||
		style.Animation.Duration != animationCascadeDuration || style.Animation.TimingFunction != "ease-in" {
		t.Errorf("Animation = %+v, want name slide, duration 250ms, timing ease-in", style.Animation)
	}

	if !style.Transition.Set || style.Transition.Property != "opacity" ||
		style.Transition.Duration != "400ms" || style.Transition.TimingFunction != "steps(4, jump-end)" {
		t.Errorf("Transition = %+v, want opacity, 400ms, steps(4, jump-end)", style.Transition)
	}
}

func TestInvalidLaterAnimationTransitionDeclarationsDoNotWin(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div class="target"></div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.target {
			animation-duration: `+animationCascadeDuration+`;
			transition-duration: 400ms;
		}
		.target {
			animation-duration: -1s;
			transition-duration: -2s;
		}
	`)}, "print", testViewport, 800)
	style := styles[findElementByName(root, "div")]

	if got := style.Animation.Duration; got != animationCascadeDuration {
		t.Errorf("Animation.Duration = %q, want earlier valid 250ms", got)
	}

	if got := style.Transition.Duration; got != "400ms" {
		t.Errorf("Transition.Duration = %q, want earlier valid 400ms", got)
	}
}

func TestAnimationTransitionSupportsAcceptanceAndCascade(t *testing.T) {
	t.Parallel()

	cases := []struct {
		prop  string
		value string
		want  bool
	}{
		{prop: "animation-duration", value: "250ms", want: true},
		{prop: "animation-duration", value: "-1s", want: false},
		{prop: "transition-property", value: "opacity, transform", want: true},
		{prop: "transition-property", value: "none, opacity", want: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.prop+"/"+testCase.value, func(t *testing.T) {
			t.Parallel()

			accepted := supportedDeclaration(testCase.prop, testCase.value)
			supported := engineSupportsProperty(testCase.prop, testCase.value)

			if accepted != testCase.want {
				t.Errorf("supportedDeclaration(%q, %q) = %t, want %t",
					testCase.prop, testCase.value, accepted, testCase.want)
			}

			if supported != testCase.want {
				t.Errorf("engineSupportsProperty(%q, %q) = %t, want %t",
					testCase.prop, testCase.value, supported, testCase.want)
			}
		})
	}

	root := mustParse(t, `<html><body><div class="target"></div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.target {
			animation-name: baseline;
			transition-property: opacity;
		}
		@supports (animation-duration: 500ms) {
			.target { animation-name: supported-animation }
		}
		@supports (animation-duration: -1s) {
			.target { animation-name: rejected-animation }
		}
		@supports (transition-duration: 500ms) {
			.target { transition-property: width }
		}
		@supports (transition-duration: -1s) {
			.target { transition-property: height }
		}
	`)}, "print", testViewport, 800)
	style := styles[findElementByName(root, "div")]

	if got := style.Animation.Name; got != "supported-animation" {
		t.Errorf("Animation.Name = %q, want valid @supports value", got)
	}

	if got := style.Transition.Property; got != "width" {
		t.Errorf("Transition.Property = %q, want valid @supports value", got)
	}
}

func TestWebkitAnimationTransitionAliasesUseStandardHandlers(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div class="standard"></div><div class="prefixed"></div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.standard {
			animation-name: pulse;
			animation-duration: 300ms;
			transition-property: opacity;
			transition-duration: 500ms;
		}
		.prefixed {
			-webkit-animation-name: pulse;
			-webkit-animation-duration: 300ms;
			-webkit-transition-property: opacity;
			-webkit-transition-duration: 500ms;
		}
	`)}, "print", testViewport, 800)
	standard := findElementByClass(t, root, "standard")
	prefixed := findElementByClass(t, root, "prefixed")

	if got, want := styles[prefixed].Animation, styles[standard].Animation; got != want {
		t.Errorf("prefixed Animation = %+v, standard Animation = %+v", got, want)
	}

	if got, want := styles[prefixed].Transition, styles[standard].Transition; got != want {
		t.Errorf("prefixed Transition = %+v, standard Transition = %+v", got, want)
	}
}

func findElementByClass(t *testing.T, root *html.Node, class string) *html.Node {
	t.Helper()

	if found := findElementByClassNode(root, class); found != nil {
		return found
	}

	t.Fatalf("element class=%q not found", class)

	return nil
}

func findElementByClassNode(root *html.Node, class string) *html.Node {
	if root == nil {
		return nil
	}

	if root.Type == html.ElementNode && root.Attribute("class") == class {
		return root
	}

	for _, child := range root.Children {
		if found := findElementByClassNode(child, class); found != nil {
			return found
		}
	}

	return nil
}
