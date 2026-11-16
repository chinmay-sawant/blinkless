package layout

import ilayout "github.com/chinmay-sawant/blinkless/internal/layout"

// InterestDelay is the computed interest-delay of a box: how long a host
// waits after a pointer arrives before it shows interest, and how long after
// the pointer leaves before it clears interest. The engine has no pointer and
// no clock, so it exports the two delays and a host applies them.
//
// interest-delay inherits, so a box that declares nothing carries its
// parent's delays. normal is the initial value and means the host uses its
// own default; in that case the matching Normal flag is true and the Ms
// value is zero. One value in an interest-delay declaration sets both
// delays.
type InterestDelay struct {
	// StartNormal is true when the start delay is the normal keyword.
	StartNormal bool
	// EndNormal is true when the end delay is the normal keyword.
	EndNormal bool
	// StartMs is the start delay in milliseconds. It is valid when StartNormal
	// is false.
	StartMs float64
	// EndMs is the end delay in milliseconds. It is valid when EndNormal is
	// false.
	EndMs float64
}

// interestDelayFrom maps the engine-side delays to the public ones.
func interestDelayFrom(delay ilayout.InterestDelayDef) InterestDelay {
	return InterestDelay{
		StartNormal: delay.StartNormal,
		EndNormal:   delay.EndNormal,
		StartMs:     delay.StartMs,
		EndMs:       delay.EndMs,
	}
}

// BoxInert reports whether the box at index in Boxes is inert. A box is inert
// when its computed Interactivity is "inert" or when any ancestor box is
// inert. interactivity inherits, so a descendant of an inert box normally
// already reads "inert"; the ancestor walk also covers a descendant that
// declares interactivity: auto, because inertness of a subtree is not undone
// from inside it. A host skips pointer hits and focus for an inert box. An
// index outside Boxes reports false.
func (d *Display) BoxInert(index int) bool {
	for index >= 0 && index < len(d.Boxes) {
		if d.Boxes[index].Interactivity == "inert" {
			return true
		}

		index = d.Boxes[index].ParentIndex
	}

	return false
}
