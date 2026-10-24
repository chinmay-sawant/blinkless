package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

// TestBoxTimelines covers the public export: a box with no declaration keeps a
// nil slice, and a box that declares both kinds exports two definitions in a
// stable order with the subject id and pixel insets.
func TestBoxTimelines(t *testing.T) {
	t.Parallel()

	const page = `<!DOCTYPE html><html><body>
		<div id="plain">plain</div>
		<div id="tracks" style="scroll-timeline: --page y; view-timeline: --card block 20px 10px">card</div>
	</body></html>`

	display := renderDisplay(t, page, "")

	plain := boxByID(t, display, "plain")
	if plain.Timelines != nil {
		t.Fatalf("plain Timelines = %#v, want nil", plain.Timelines)
	}

	assertBoxTimelines(t, boxByID(t, display, "tracks"))
}

// assertBoxTimelines checks the scroll then view order and each definition.
func assertBoxTimelines(t *testing.T, tracks layout.Box) {
	t.Helper()

	if len(tracks.Timelines) != 2 {
		t.Fatalf("tracks Timelines = %d, want 2", len(tracks.Timelines))
	}

	assertBoxScrollTimeline(t, tracks.Timelines[0])
	assertBoxViewTimeline(t, tracks.Timelines[1])
}

func assertBoxScrollTimeline(t *testing.T, scroll layout.TimelineDef) {
	t.Helper()

	if scroll.Name != "--page" || scroll.Kind != "scroll" || scroll.Axis != "y" {
		t.Fatalf("scroll definition = %#v, want name --page, kind scroll, axis y", scroll)
	}

	if scroll.SubjectID != "" || scroll.ScrollerID != "" {
		t.Fatalf("scroll subject/scroller = (%q, %q), want empty", scroll.SubjectID, scroll.ScrollerID)
	}
}

func assertBoxViewTimeline(t *testing.T, view layout.TimelineDef) {
	t.Helper()

	if view.Name != "--card" || view.Kind != "view" || view.Axis != "y" {
		t.Fatalf("view definition = %#v, want name --card, kind view, axis y", view)
	}

	if view.SubjectID != "tracks" {
		t.Fatalf("view SubjectID = %q, want %q", view.SubjectID, "tracks")
	}

	if view.ScrollerID != "" {
		t.Fatalf("view ScrollerID = %q, want the page scroller", view.ScrollerID)
	}

	nearBox(t, "inset start", view.InsetStart, 20)
	nearBox(t, "inset end", view.InsetEnd, 10)
}
