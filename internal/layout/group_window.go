package layout

// widenGroupWindow grows a splice window over the group markers an element
// previously owned. A full layout emits the begin marker before the element's
// box range opens and the end marker after it closes, so the retained window
// sits strictly inside its own pair. A rebuilt subtree is a build root: when it
// still initiates a group it emits a fresh pair at its first and last
// operations, and when it no longer initiates a group it emits no pair at all.
// In both cases the old pair must be part of the replaced window so patching a
// grouped element never duplicates or strands group markers.
//
// Callers must only invoke this when initiatesBlendGroup(target.box.style) is
// true, ensuring the adjacent markers belong to target rather than an enclosing
// parent whose only child is target.
func widenGroupWindow(prevOps []Op, start, oldCount int) (int, int) {
	const groupMarkerCount = 2
	if start < 1 || start+oldCount >= len(prevOps) {
		return start, oldCount
	}

	begin, end := &prevOps[start-1], &prevOps[start+oldCount]
	if !matchingGroupMarks(begin, end) {
		return start, oldCount
	}

	return start - 1, oldCount + groupMarkerCount
}

func matchingGroupMarks(begin, end *Op) bool {
	return begin.IsGroupBegin() && end.IsGroupEnd() && begin.Group() != nil && begin.Group() == end.Group()
}

func relayoutGroupWindow(prev *Result, target *relayoutTarget) (int, int, error) {
	start, count, err := relayoutWindow(prev, target)
	if err != nil {
		return 0, 0, err
	}

	// If the target only gains a group in the new build, neighbouring
	// markers can belong to a parent whose only child is this target.
	if initiatesBlendGroup(target.box.style) {
		start, count = widenGroupWindow(prev.Ops, start, count)
	}

	return start, count, nil
}
