package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

const animationSlideName = "slide"

func TestDisplayAnimationAndTransitionMetadata(t *testing.T) {
	t.Parallel()

	const source = `<style>
@keyframes slide { from { opacity: 0 } to { opacity: 1 } }
@keyframes slide { 0% { transform: translateX(0px) } 100% { transform: translateX(20px) } }
#moving {
  animation-name: slide, pulse;
  animation-duration: 2s, 500ms;
  animation-timing-function: ease-in, steps(4, end);
  animation-iteration-count: 2, infinite;
  animation-direction: alternate, reverse;
  animation-fill-mode: both, forwards;
  animation-play-state: paused, running;
  animation-composition: add, accumulate;
  animation-delay-start: -150ms, 0ms;
  animation-delay-end: 150ms;
  animation-timeline: auto, scroll();
  animation-range-start: normal, entry 25%;
  animation-range-end: normal, exit 75%;
  trigger-scope: all;
  view-transition-name: card;
  view-transition-class: old new;
  view-transition-group: contain;
  view-transition-scope: all;
  transition-property: opacity, transform;
  transition-duration: 250ms, 1s;
  transition-delay: -100ms;
  transition-timing-function: ease-in, steps(2, jump-end);
  transition-behavior: normal, allow-discrete;
}

</style><div id="moving">motion</div>`

	display := displayOf(t, source)

	box := animationBoxByID(display, "moving")
	if box == nil {
		t.Fatal("moving box missing")
	}

	assertAnimationMetadata(t, box)
	assertTransitionMetadata(t, box)
	assertKeyframeMetadata(t, display.Keyframes)
}

func animationBoxByID(display *layout.Display, id string) *layout.Box {
	for index := range display.Boxes {
		if display.Boxes[index].ID == id {
			return &display.Boxes[index]
		}
	}

	return nil
}

func assertAnimationMetadata(t *testing.T, box *layout.Box) {
	t.Helper()

	if len(box.Animations) != 2 {
		t.Fatalf("animations = %#v, want two list items", box.Animations)
	}

	first, second := box.Animations[0], box.Animations[1]
	assertFirstAnimationCoreFields(t, first)
	assertFirstAnimationPlaybackFields(t, first)
	assertFirstAnimationTimingFields(t, first)
	assertSecondAnimationCoreFields(t, second)
	assertSecondAnimationListFields(t, second)
	assertSecondAnimationRangeFields(t, second)
}

func assertFirstAnimationCoreFields(t *testing.T, animation layout.Animation) {
	t.Helper()

	if animation.Name != animationSlideName || animation.DurationMS != 2000 ||
		animation.TimingFunction != "ease-in" || animation.IterationCount != 2 {
		t.Fatalf("first animation = %#v", animation)
	}
}

func assertFirstAnimationPlaybackFields(t *testing.T, animation layout.Animation) {
	t.Helper()

	if animation.Direction != "alternate" || animation.FillMode != "both" || animation.PlayState != "paused" ||
		animation.Composition != "add" {
		t.Fatalf("first animation flags = %#v", animation)
	}
}

func assertFirstAnimationTimingFields(t *testing.T, animation layout.Animation) {
	t.Helper()

	if animation.DelayStartMS != -150 || animation.DelayEndMS != 150 || animation.Timeline != "auto" {
		t.Fatalf("first animation timing = %#v", animation)
	}
}

func assertSecondAnimationCoreFields(t *testing.T, animation layout.Animation) {
	t.Helper()

	if animation.Name != "pulse" || animation.DurationMS != 500 || !animation.IterationInfinite ||
		animation.IterationCount != 0 {
		t.Fatalf("second animation = %#v", animation)
	}
}

func assertSecondAnimationListFields(t *testing.T, animation layout.Animation) {
	t.Helper()

	if animation.TimingFunction != "steps(4, end)" || animation.DelayEndMS != 150 || animation.Timeline != "scroll()" {
		t.Fatalf("second animation list alignment = %#v", animation)
	}
}

func assertSecondAnimationRangeFields(t *testing.T, animation layout.Animation) {
	t.Helper()

	if animation.RangeStart != "entry 25%" || animation.RangeEnd != "exit 75%" {
		t.Fatalf("second animation range = %#v", animation)
	}
}

func assertTransitionMetadata(t *testing.T, box *layout.Box) {
	t.Helper()

	if box.TriggerScope != "all" || box.ViewTransitionName != "card" || box.ViewTransitionClass != "old new" ||
		box.ViewTransitionGroup != "contain" || box.ViewTransitionScope != "all" {
		t.Fatalf("view transition metadata = %#v", box)
	}

	if len(box.Transitions) != 2 {
		t.Fatalf("transitions = %#v, want two list items", box.Transitions)
	}

	assertFirstTransitionMetadata(t, box.Transitions[0])
	assertSecondTransitionMetadata(t, box.Transitions[1])
}

func assertFirstTransitionMetadata(t *testing.T, transition layout.Transition) {
	t.Helper()

	if transition.Property != "opacity" || transition.DurationMS != 250 || transition.DelayMS != -100 ||
		transition.TimingFunction != "ease-in" || transition.Behavior != "normal" {
		t.Fatalf("first transition = %#v", transition)
	}
}

func assertSecondTransitionMetadata(t *testing.T, transition layout.Transition) {
	t.Helper()

	if transition.Property != "transform" || transition.DurationMS != 1000 || transition.DelayMS != -100 ||
		transition.TimingFunction != "steps(2, jump-end)" || transition.Behavior != "allow-discrete" {
		t.Fatalf("second transition = %#v", transition)
	}
}

func assertKeyframeMetadata(t *testing.T, keyframes []layout.KeyframesRule) {
	t.Helper()

	if len(keyframes) != 2 || keyframes[0].Name != animationSlideName || keyframes[1].Name != animationSlideName {
		t.Fatalf("keyframes = %#v, want both source-ordered rules", keyframes)
	}

	last := keyframes[1]
	if len(last.Keyframes) != 2 || last.Keyframes[0].Offset != 0 || last.Keyframes[1].Offset != 1 {
		t.Fatalf("last slide keyframes = %#v", last.Keyframes)
	}

	if len(last.Keyframes[0].Declarations) != 1 ||
		last.Keyframes[0].Declarations[0] != (layout.KeyframeDeclaration{
			Property: "transform",
			Value:    "translateX(0px)",
		}) {
		t.Fatalf("from declarations = %#v", last.Keyframes[0].Declarations)
	}
}

func TestDisplayQuotedAnimationNameMatchesKeyframesName(t *testing.T) {
	t.Parallel()

	display := displayOf(t, `<style>
@keyframes "fade in" { from { opacity: 0 } to { opacity: 1 } }
#item { animation-name: "fade in" }
</style><div id="item"></div>`)

	if len(display.Keyframes) != 1 || display.Keyframes[0].Name != "fade in" {
		t.Fatalf("keyframes = %#v, want one unquoted name", display.Keyframes)
	}

	for _, box := range display.Boxes {
		if box.ID != "item" {
			continue
		}

		if len(box.Animations) != 1 || box.Animations[0].Name != display.Keyframes[0].Name {
			t.Fatalf("animation = %#v, keyframe name = %q", box.Animations, display.Keyframes[0].Name)
		}

		return
	}

	t.Fatal("item box missing")
}

func TestDisplayAnimationTriggerAndRangeCenter(t *testing.T) {
	t.Parallel()

	display := displayOf(t, `<style>
#item {
  animation-name: pulse;
  animation-trigger: --ready play-once;
  animation-range-center: target 25%;
}
</style><div id="item"></div>`)

	box := animationBoxByID(display, "item")
	if box == nil {
		t.Fatal("item box missing")
	}

	if len(box.Animations) != 1 {
		t.Fatalf("animations = %#v, want one list item", box.Animations)
	}

	if box.Animations[0].Trigger != "--ready play-once" {
		t.Errorf("animation trigger = %q, want %q", box.Animations[0].Trigger, "--ready play-once")
	}

	if box.Animations[0].RangeCenter != "target 25%" {
		t.Errorf("animation range center = %q, want %q", box.Animations[0].RangeCenter, "target 25%")
	}
}
