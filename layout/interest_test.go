package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

func delayOf(t *testing.T, style string) layout.InterestDelay {
	t.Helper()

	return boxWithStyle(t, style).InterestDelay
}

// TestBoxInterestDelayShorthand covers interest-delay: one and two values,
// normal, units, and invalid values.
func TestBoxInterestDelayShorthand(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		style string
		want  layout.InterestDelay
	}{
		{"initial", "", layout.InterestDelay{StartNormal: true, EndNormal: true}},
		{"one value sets both", "interest-delay: 200ms", layout.InterestDelay{StartMs: 200, EndMs: 200}},
		{"two values", "interest-delay: 200ms 500ms", layout.InterestDelay{StartMs: 200, EndMs: 500}},
		{"seconds", "interest-delay: 1s 0.5s", layout.InterestDelay{StartMs: 1000, EndMs: 500}},
		{"normal then time", "interest-delay: normal 3s", layout.InterestDelay{StartNormal: true, EndMs: 3000}},
		{"time then normal", "interest-delay: 2s normal", layout.InterestDelay{StartMs: 2000, EndNormal: true}},
		{
			"normal",
			"interest-delay: 1s; interest-delay: normal",
			layout.InterestDelay{StartNormal: true, EndNormal: true},
		},
		{
			"three values dropped",
			"interest-delay: 1s; interest-delay: 1s 2s 3s",
			layout.InterestDelay{StartMs: 1000, EndMs: 1000},
		},
		{
			"negative dropped",
			"interest-delay: 1s; interest-delay: -1s",
			layout.InterestDelay{StartMs: 1000, EndMs: 1000},
		},
		{
			"unitless dropped",
			"interest-delay: 1s; interest-delay: 5",
			layout.InterestDelay{StartMs: 1000, EndMs: 1000},
		},
		{
			"initial keyword",
			"interest-delay: 1s; interest-delay: initial",
			layout.InterestDelay{StartNormal: true, EndNormal: true},
		},
		{
			"longhand after shorthand",
			"interest-delay: 1s 2s; interest-delay-end: 9s",
			layout.InterestDelay{StartMs: 1000, EndMs: 9000},
		},
		{
			"shorthand after longhand",
			"interest-delay-end: 9s; interest-delay: 1s",
			layout.InterestDelay{StartMs: 1000, EndMs: 1000},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := delayOf(t, test.style); got != test.want {
				t.Fatalf("InterestDelay = %#v, want %#v", got, test.want)
			}
		})
	}
}

// TestBoxInterestDelayLonghands checks that each longhand changes only its
// own side.
func TestBoxInterestDelayLonghands(t *testing.T) {
	t.Parallel()

	if got := delayOf(t, "interest-delay-start: 200ms"); got != (layout.InterestDelay{StartMs: 200, EndNormal: true}) {
		t.Errorf("start only = %#v", got)
	}

	if got := delayOf(t, "interest-delay-end: 500ms"); got != (layout.InterestDelay{StartNormal: true, EndMs: 500}) {
		t.Errorf("end only = %#v", got)
	}

	if got := delayOf(t, "interest-delay-start: 200ms; interest-delay-start: normal"); !got.StartNormal {
		t.Errorf("start normal = %#v", got)
	}

	if got := delayOf(t, "interest-delay-start: 200ms; interest-delay-start: 1px"); got.StartMs != 200 {
		t.Errorf("invalid start changed the value: %#v", got)
	}

	if got := delayOf(t, "interest-delay-end: 5ms; interest-delay-end: 1s 2s"); got.EndMs != 5 {
		t.Errorf("two values on a longhand changed the value: %#v", got)
	}
}

// TestBoxInterestDelayInherits checks interest-delay is inherited and that
// the wide keywords resolve.
func TestBoxInterestDelayInherits(t *testing.T) {
	t.Parallel()

	const page = `<!DOCTYPE html><html><body>
		<div id="p" style="interest-delay: 200ms 500ms">
			<div id="plain">a</div>
			<div id="own" style="interest-delay-end: 1s">b</div>
			<div id="reset" style="interest-delay: initial">c</div>
			<div id="unset" style="interest-delay: unset">d</div>
			<div id="inherit" style="interest-delay-start: inherit">e
				<div id="deep">f</div></div>
		</div></body></html>`

	display := renderDisplay(t, page, "")

	want := map[string]layout.InterestDelay{
		"plain":   {StartMs: 200, EndMs: 500},
		"own":     {StartMs: 200, EndMs: 1000},
		"reset":   {StartNormal: true, EndNormal: true},
		"unset":   {StartMs: 200, EndMs: 500},
		"inherit": {StartMs: 200, EndMs: 500},
		"deep":    {StartMs: 200, EndMs: 500},
	}

	for id, delay := range want {
		if got := boxByID(t, display, id).InterestDelay; got != delay {
			t.Errorf("#%s InterestDelay = %#v, want %#v", id, got, delay)
		}
	}
}

// TestBoxInteractivity covers interactivity: values, invalid values dropped,
// inheritance, the wide keywords, and BoxInert.
func TestBoxInteractivity(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                     "auto",
		"interactivity: auto":  "auto",
		"interactivity: inert": "inert",
		"interactivity: INERT": "inert",
		"interactivity: inert; interactivity: bogus":      "inert",
		"interactivity: inert; interactivity: none":       "inert",
		"interactivity: inert; interactivity: initial":    "auto",
		"interactivity: inert; interactivity: unset":      "auto",
		"interactivity: inert; interactivity: inert auto": "inert",
	}

	for style, want := range cases {
		if got := boxWithStyle(t, style).Interactivity; got != want {
			t.Errorf("%q Interactivity = %q, want %q", style, got, want)
		}
	}
}

// TestBoxInteractivityInheritsAndInert checks inheritance down the tree and
// the BoxInert subtree rule.
func TestBoxInteractivityInheritsAndInert(t *testing.T) {
	t.Parallel()

	const page = `<!DOCTYPE html><html><body>
		<div id="top" style="interactivity: inert">
			<div id="child">a
				<div id="grand">b</div></div>
			<div id="revived" style="interactivity: auto">c
				<div id="revivedChild">d</div></div>
			<div id="unset" style="interactivity: unset">e</div>
			<div id="initial" style="interactivity: initial">f</div>
			<div id="inherit" style="interactivity: inherit">g</div>
		</div>
		<div id="free">h</div></body></html>`

	display := renderDisplay(t, page, "")

	computed := map[string]string{
		"top": "inert", "child": "inert", "grand": "inert", "revived": "auto", "revivedChild": "auto",
		"unset": "inert", "initial": "auto", "inherit": "inert", "free": "auto",
	}

	for id, want := range computed {
		if got := boxByID(t, display, id).Interactivity; got != want {
			t.Errorf("#%s Interactivity = %q, want %q", id, got, want)
		}
	}

	indexOf := func(id string) int {
		for i, box := range display.Boxes {
			if box.ID == id {
				return i
			}
		}

		t.Fatalf("no box #%s", id)

		return -1
	}

	// A descendant that declares auto stays inert: the subtree is inert.
	inert := map[string]bool{
		"top": true, "child": true, "grand": true, "revived": true, "revivedChild": true,
		"unset": true, "initial": true, "inherit": true, "free": false,
	}

	for id, want := range inert {
		if got := display.BoxInert(indexOf(id)); got != want {
			t.Errorf("BoxInert(#%s) = %t, want %t", id, got, want)
		}
	}

	if display.BoxInert(-1) || display.BoxInert(len(display.Boxes)) {
		t.Error("BoxInert out of range = true, want false")
	}
}
