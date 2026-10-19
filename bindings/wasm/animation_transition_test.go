package main

import "testing"

func TestDrawingListAnimationAndTransition(t *testing.T) {
	t.Parallel()

	const source = `<style>
@keyframes slide { from { opacity: 0 } to { opacity: 1 } }
#item {
  animation: slide 2s ease-in 3 alternate both paused;
  animation-delay-start: -150ms;
  animation-delay-end: 150ms;
  trigger-scope: all;
  view-transition-name: card;
  transition: opacity 250ms ease-in -100ms allow-discrete;
}
</style><div id="item">item</div>`

	_, list := convertSource(t, source, 640, 480)
	if len(list.Keyframes) != 1 || list.Keyframes[0].Name != "slide" {
		t.Fatalf("keyframes = %#v", list.Keyframes)
	}

	item := drawingBoxByID(list.Boxes, "item")
	if item == nil {
		t.Fatal("item box missing")
	}

	assertDrawingListAnimation(t, item)
	assertDrawingListTransition(t, item)
	assertDrawingListIdentity(t, item)
}

func drawingBoxByID(boxes []drawBoxJSON, id string) *drawBoxJSON {
	for index := range boxes {
		if boxes[index].ID == id {
			return &boxes[index]
		}
	}

	return nil
}

func assertDrawingListAnimation(t *testing.T, item *drawBoxJSON) {
	t.Helper()

	if len(item.Animations) != 1 || item.Animations[0].Name != "slide" || item.Animations[0].DurationMS != 2000 ||
		item.Animations[0].DelayStartMS != -150 || item.Animations[0].DelayEndMS != 150 {
		t.Fatalf("animations = %#v", item.Animations)
	}
}

func assertDrawingListIdentity(t *testing.T, item *drawBoxJSON) {
	t.Helper()

	if item.TriggerScope != "all" || item.ViewTransitionName != "card" {
		t.Fatalf("identity fields: triggerScope=%q viewTransitionName=%q", item.TriggerScope, item.ViewTransitionName)
	}
}

func assertDrawingListTransition(t *testing.T, item *drawBoxJSON) {
	t.Helper()

	if len(item.Transitions) != 1 || item.Transitions[0].Property != "opacity" || item.Transitions[0].DurationMS != 250 ||
		item.Transitions[0].DelayMS != -100 || item.Transitions[0].Behavior != "allow-discrete" {
		t.Fatalf("transitions = %#v", item.Transitions)
	}
}
