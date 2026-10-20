package layout

import "strings"

// canBuildSingleColumnGrid reports whether a one-column grid can skip the
// measure pass. A row with one item sizes to that item, so the item can be
// built once at its final position. Explicit rows, spans, and percentage
// heights still take the two-pass path.
func canBuildSingleColumnGrid(e *engine, sty ResolvedStyle, placed []gridCell) bool {
	if len(placed) == 0 || strings.TrimSpace(sty.GridTemplateRows) != "" {
		return false
	}
	if autoRows := strings.TrimSpace(sty.GridAutoRows); autoRows != "" && !strings.EqualFold(autoRows, "auto") {
		return false
	}

	seen := make(map[int]bool, len(placed))
	for _, page := range placed {
		if page.col != 0 || page.colSpan > 1 || page.rowSpan > 1 || seen[page.row] {
			return false
		}
		child := e.stylePtr(page.n)
		if child == nil || child.HeightPercent >= 0 || child.MinHeightPercent >= 0 {
			return false
		}
		seen[page.row] = true
	}

	return true
}

// layoutSingleColumnGrid builds each item once at its natural row top, then
// shifts the rows for align-content. It is the one-column fast path for
// layoutStandardGrid.
func (e *engine) layoutSingleColumnGrid(
	boxNode *box, sty ResolvedStyle, placed []gridCell,
	contentX, contentW, curY, posY, contentH float64, contentStart int,
) *box {
	numRows := 0
	for _, page := range placed {
		if page.row+1 > numRows {
			numRows = page.row + 1
		}
	}

	byRow := make([]int, numRows)
	for i := range byRow {
		byRow[i] = -1
	}
	for i := range placed {
		byRow[placed[i].row] = i
	}

	containerJustify := sty.JustifyItems
	if containerJustify == "" {
		containerJustify = fxStretch
	}
	containerAlign := sty.AlignItems
	if containerAlign == "" {
		containerAlign = fxStretch
	}

	_, rowGap := e.styleGaps(sty)
	rows := make([]float64, numRows)
	children := make([]*box, numRows)
	y := curY

	for row := 0; row < numRows; row++ {
		idx := byRow[row]
		if idx < 0 {
			y += rowGap

			continue
		}
		page := &placed[idx]
		child := e.stylePtr(page.n)
		justify := gridItemJustify(*child, containerJustify)
		buildW := gridItemBuildWidth(e, page.n, *child, justify, contentW)

		oldMax := e.imgMaxW
		if contentW > 0 {
			e.imgMaxW = contentW
		}
		cblock := e.build(page.n, buildW, contentX, posY+y)
		e.imgMaxW = oldMax

		if cblock == nil {
			y += rowGap

			continue
		}

		deltaX := contentX + gridAlignOffset(justify, contentW, cblock.w) - cblock.x
		e.shiftBoxOps(cblock, deltaX, 0)
		cblock.x += deltaX
		boxNode.children = append(boxNode.children, cblock)
		children[row] = cblock
		rows[row] = cblock.height
		y += cblock.height + rowGap
	}

	rowYs := make([]float64, numRows)
	if numRows > 0 {
		rowYs[0] = curY
	}
	for row := 1; row < numRows; row++ {
		rowYs[row] = rowYs[row-1] + rows[row-1] + rowGap
	}
	if contentH >= 0 {
		applyGridAlignContentOffsets(rowYs, rows, rowGap, contentH, sty.AlignContent)
	}

	for row := 0; row < numRows; row++ {
		cblock := children[row]
		if cblock == nil {
			continue
		}
		deltaY := posY + rowYs[row] - cblock.y
		e.shiftBoxOps(cblock, 0, deltaY)
		cblock.y += deltaY
	}

	usedH := curY
	if numRows > 0 {
		usedH = y - rowGap
	}
	usedH = resolveGridUsedHeight(e, sty, usedH, contentH)
	boxNode.height = usedH
	e.prependChrome(contentStart, boxNode, sty, boxNode.x, posY, boxNode.w, boxNode.height)

	return boxNode
}
