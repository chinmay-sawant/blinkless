package layout

// margin-break (CSS Fragmentation 4) says what happens to a margin that
// touches a fragmentation break. auto and discard drop it. keep leaves it.
//
// Multicol lays children into column stacks without their margins, so a
// margin at the top of a column after a break is already gone, which is what
// auto and discard want. keep is the one value that has to add something: the
// child's margin-top is put back at the top of a column that starts after a
// break. A margin in the middle of a column is not touched, and a page break
// is not handled because this engine has no page index.

// keptBreakMargin returns the margin-top, in laid-out points, that a child
// keeps at the top of a column it starts after a break. It is 0 for the first
// column, for a column that already holds content, and for auto or discard.
func (e *engine) keptBreakMargin(style *ResolvedStyle, col int, colHeight float64) float64 {
	if style == nil || style.MarginBreak != marginBreakKeep || col <= 0 || colHeight > 0 {
		return 0
	}

	if style.MarginTop <= 0 {
		return 0
	}

	return e.scalePt(style.MarginTop)
}
