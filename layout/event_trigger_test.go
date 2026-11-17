package layout_test

import (
	"reflect"
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

// TestBoxEventTriggerShorthand covers the event-trigger shorthand.
func TestBoxEventTriggerShorthand(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		style string
		want  []layout.EventTrigger
	}{
		{"initial", "", nil},
		{"none", "event-trigger: none", nil},
		{"click", "event-trigger: --press click", []layout.EventTrigger{{Name: "--press", Start: []string{"click"}}}},
		{
			"start and end", "event-trigger: --press click / keydown",
			[]layout.EventTrigger{{Name: "--press", Start: []string{"click"}, End: []string{"keydown"}}},
		},
		{
			"slash without spaces", "event-trigger: --press click/keydown",
			[]layout.EventTrigger{{Name: "--press", Start: []string{"click"}, End: []string{"keydown"}}},
		},
		{
			"several events", "event-trigger: --p click dblclick / keydown(Enter) blur",
			[]layout.EventTrigger{{
				Name: "--p", Start: []string{"click", "dblclick"}, End: []string{"keydown(Enter)", "blur"},
			}},
		},
		{"source none", "event-trigger: --p none", []layout.EventTrigger{{Name: "--p"}}},
		{
			"two items", "event-trigger: --a click, --b focus / blur",
			[]layout.EventTrigger{
				{Name: "--a", Start: []string{"click"}},
				{Name: "--b", Start: []string{"focus"}, End: []string{"blur"}},
			},
		},
		{"missing source dropped", "event-trigger: --p", nil},
		{"bad name dropped", "event-trigger: p click", nil},
		{"two slashes dropped", "event-trigger: --p a / b / c", nil},
		{"empty end dropped", "event-trigger: --p a /", nil},
		{"empty start dropped", "event-trigger: --p / a", nil},
		{"digit event dropped", "event-trigger: --p 1click", nil},
		{"initial keyword", "event-trigger: --p click; event-trigger: initial", nil},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := boxWithStyle(t, test.style).EventTriggers; !reflect.DeepEqual(got, test.want) {
				t.Fatalf("EventTriggers = %#v, want %#v", got, test.want)
			}
		})
	}
}

// TestBoxEventTriggerLonghands covers the name and source longhands, lists,
// repetition, and the wide keywords.
func TestBoxEventTriggerLonghands(t *testing.T) {
	t.Parallel()

	got := boxWithStyle(t, "event-trigger-name: --press, --key, --third;"+
		"event-trigger-source: click / keydown, focus").EventTriggers
	want := []layout.EventTrigger{
		{Name: "--press", Start: []string{"click"}, End: []string{"keydown"}},
		{Name: "--key", Start: []string{"focus"}},
		{Name: "--third", Start: []string{"click"}, End: []string{"keydown"}},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("EventTriggers = %#v, want %#v", got, want)
	}

	// A name with no source declaration has the initial none source.
	nameOnly := boxWithStyle(t, "event-trigger-name: --a").EventTriggers
	if !reflect.DeepEqual(nameOnly, []layout.EventTrigger{{Name: "--a"}}) {
		t.Fatalf("name only = %#v, want one trigger with no events", nameOnly)
	}

	// A source with no name declares nothing.
	if got := boxWithStyle(t, "event-trigger-source: click").EventTriggers; got != nil {
		t.Fatalf("source only = %#v, want nil", got)
	}

	invalid := []string{
		"event-trigger-name: --a; event-trigger-name: a",
		"event-trigger-name: --a; event-trigger-name: none, --b",
		"event-trigger-name: --a; event-trigger-name: --b,",
	}

	for _, style := range invalid {
		if got := boxWithStyle(t, style).EventTriggers; len(got) != 1 || got[0].Name != "--a" {
			t.Errorf("%q = %#v, want the earlier --a kept", style, got)
		}
	}

	for _, keyword := range []string{"initial", "unset", "revert"} {
		if got := boxWithStyle(t, "event-trigger-name: --a; event-trigger-name: "+keyword).EventTriggers; got != nil {
			t.Errorf("event-trigger-name: %s = %#v, want none", keyword, got)
		}
	}

	if got := boxWithStyle(t, "event-trigger-name: --a; event-trigger-name: none").EventTriggers; got != nil {
		t.Errorf("event-trigger-name: none = %#v, want none", got)
	}

	badSources := []string{"none click", "click, , focus", "click /", "inherit click", "(click)"}
	for _, source := range badSources {
		style := "event-trigger-name: --a; event-trigger-source: click; event-trigger-source: " + source
		got := boxWithStyle(t, style).EventTriggers
		if len(got) != 1 || !reflect.DeepEqual(got[0].Start, []string{"click"}) {
			t.Errorf("source %q = %#v, want the earlier click kept", source, got)
		}
	}
}
