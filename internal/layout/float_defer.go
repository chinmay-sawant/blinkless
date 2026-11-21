package layout

import (
	"math"
	"strconv"
)

// deferFloatsToColumns moves each floated child of a multicol container that
// declares float-defer to a later column. An integer N moves it N columns to
// the right and `last` moves it to the last column. A float already in the last
// column stays there, which is also where a float lands when fewer than N
// columns follow it. The float keeps its own y. Outside a multicol container
// there is no later column, so the float stays where it is.
func (e *engine) deferFloatsToColumns(boxNode *box, contentX, colW, gap float64, nCols int) {
	step := colW + gap
	if step <= 0 || nCols < 2 {
		return
	}

	for _, child := range boxNode.children {
		if child == nil || child.style == nil || child.style.Float == cssDisplayNone {
			continue
		}

		target, ok := floatDeferTarget(child.style.declared().FloatDefer, columnOf(child.x, contentX, step), nCols)
		if !ok {
			continue
		}

		delta := float64(target)*step + contentX - child.x
		if math.Abs(delta) < borderEps {
			continue
		}

		e.shiftBoxOps(child, delta, 0)
		child.x += delta
	}
}

// columnOf is the index of the column whose left edge is nearest to x.
func columnOf(x, contentX, step float64) int {
	return int(math.Round((x - contentX) / step))
}

// floatDeferTarget returns the column a float with the given float-defer value
// ends up in, clamped to the last column.
func floatDeferTarget(value string, column, nCols int) (int, bool) {
	last := nCols - 1

	switch value {
	case "":
		return 0, false
	case "last":
		return last, true
	}

	count, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}

	return min(max(column+count, 0), last), true
}
