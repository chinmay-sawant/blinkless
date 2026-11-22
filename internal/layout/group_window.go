package layout

// widenGroupWindow grows a splice window over the group markers an element
// owns. A full layout emits the begin marker before the element's box range
// opens and the end marker after it closes, so the retained window sits
// strictly inside its own pair. A rebuilt subtree is a build root, and it
// emits that pair itself, at the first and last operation it returns.
// Splicing those operations into the unwidened window would leave the old
// pair in place around the new one, and every patch of a grouped element
// (backdrop-filter, blend, isolation, mask) would add a pair.
//
// The window widens by one operation on each side only when the new
// operations carry their own begin and end marker of one group and the
// retained neighbours are the begin and end marker of one group, so the old
// pair is replaced by the new one. Anything else keeps the window as is.
func widenGroupWindow(prevOps []Op, start, oldCount int, newOps []Op) (int, int) {
	const groupMarkerCount = 2
	if len(newOps) < groupMarkerCount || start < 1 || start+oldCount >= len(prevOps) {
		return start, oldCount
	}

	first, last := &newOps[0], &newOps[len(newOps)-1]
	if !matchingGroupMarks(first, last) {
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

func relayoutGroupWindow(prev *Result, target *relayoutTarget, newOps []Op) (int, int, error) {
	start, count, err := relayoutWindow(prev, target)
	if err != nil {
		return 0, 0, err
	}

	// If the target only gains a group in the new build, neighbouring
	// markers can belong to a parent whose only child is this target.
	if initiatesBlendGroup(target.box.style) {
		start, count = widenGroupWindow(prev.Ops, start, count, newOps)
	}

	return start, count, nil
}
