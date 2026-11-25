package layout

import "strings"

// canBuildSingleColumnGrid reports whether a one-column grid can skip the
// measure pass. A row with one item sizes to that item, so the item can be
// built once at its final position. Explicit rows, areas, placement, spans,
// and percentage heights still take the two-pass path.
func canBuildSingleColumnGrid(eng *engine, sty ResolvedStyle, placed []gridCell, cols []float64) bool {
	if len(placed) == 0 || len(cols) != 1 || strings.TrimSpace(sty.grid().GridTemplateRows) != "" ||
		strings.TrimSpace(sty.grid().GridTemplateAreas) != "" {
		return false
	}

	if autoRows := strings.TrimSpace(sty.grid().GridAutoRows); autoRows != "" && !strings.EqualFold(autoRows, "auto") {
		return false
	}

	maxRow := -1
	for _, page := range placed {
		if page.row < 0 {
			return false
		}

		if page.row > maxRow {
			maxRow = page.row
		}
	}

	if maxRow < len(placed)*2 {
		seen := getFlexBools(maxRow + 1)
		defer putFlexBools(seen)

		for _, page := range placed {
			if !singleColumnCellEligibleSlice(eng, page, sty.flex().JustifyItems, seen) {
				return false
			}
		}

		return true
	}

	seen := make(map[int]bool, len(placed))

	for _, page := range placed {
		if !singleColumnCellEligible(eng, page, sty.flex().JustifyItems, seen) {
			return false
		}
	}

	return true
}

// singleColumnCellEligibleSlice is the slice-backed eligibility check used
// when row indices are compact, avoiding a per-grid map allocation.
func singleColumnCellEligibleSlice(
	eng *engine, page gridCell, containerJustify string, seen []bool,
) bool {
	if page.col != 0 || page.colSpan > 1 || page.rowSpan > 1 || seen[page.row] {
		return false
	}

	child := eng.stylePtr(page.n)
	if child == nil || autoRowExplicitPlacement(child) || autoRowHeightHazard(child) {
		return false
	}

	if !gridJustifyEligible(child, containerJustify) {
		return false
	}

	seen[page.row] = true

	return true
}

// singleColumnCellEligible reports whether one placed cell keeps the
// one-item-per-row shape the fast path needs.
func singleColumnCellEligible(
	eng *engine, page gridCell, containerJustify string, seen map[int]bool,
) bool {
	if page.col != 0 || page.colSpan > 1 || page.rowSpan > 1 || seen[page.row] {
		return false
	}

	child := eng.stylePtr(page.n)
	if child == nil || autoRowExplicitPlacement(child) || autoRowHeightHazard(child) {
		return false
	}

	if !gridJustifyEligible(child, containerJustify) {
		return false
	}

	seen[page.row] = true

	return true
}

// layoutSingleColumnGrid builds each item once at its natural row top, then
// shifts the rows for align-content. It is the one-column fast path for
// layoutStandardGrid.
func (e *engine) layoutSingleColumnGrid(
	boxNode *box, sty ResolvedStyle, placed []gridCell, cols []float64, columnGap float64,
	contentX, contentW, curY, posY, contentH float64, contentStart int,
) *box {
	var noAreas gridTemplateAreasMap

	numRows := gridRowCount(placed, noAreas)
	byRow := singleColumnByRow(placed, numRows)
	cellW := cols[0]

	containerJustify := sty.flex().JustifyItems
	if containerJustify == "" {
		containerJustify = fxStretch
	}

	rowGap, _ := e.styleGaps(&sty)
	rows := rentFloats(numRows)
	defer returnFloats(rows)

	children := make([]*box, numRows)
	if len(boxNode.children) == 0 && len(placed) > 0 {
		boxNode.children = make([]*box, 0, len(placed))
	}

	rowTop := curY

	for row := range numRows {
		idx := byRow[row]
		if idx < 0 {
			rowTop += rowGap

			continue
		}

		cblock := e.buildSingleColumnItem(&placed[idx], containerJustify, contentX, cellW, posY, rowTop)
		if cblock == nil {
			rowTop += rowGap

			continue
		}

		boxNode.children = append(boxNode.children, cblock)
		children[row] = cblock
		rows[row] = cblock.height
		rowTop += cblock.height + rowGap
	}

	rowYs := rentFloats(numRows)
	defer returnFloats(rowYs)
	gridRowTopsInto(rowYs, curY, rows, rowGap, contentH, sty.flex().AlignContent)

	e.shiftSingleColumnRows(children, numRows, rowYs, posY)
	e.paintGridGapRules(sty, placed, contentX, contentW, cols, columnGap, rowYs, rows, posY)

	usedH := curY
	if numRows > 0 {
		usedH = rowTop - rowGap
	}

	usedH = resolveGridUsedHeight(e, sty, usedH, contentH)
	boxNode.height = usedH
	e.prependChrome(contentStart, boxNode, &sty, boxNode.x, posY, boxNode.w, boxNode.height)

	return boxNode
}

// singleColumnByRow maps each row to its placed item index, or -1 for a row
// with no item.
func singleColumnByRow(placed []gridCell, numRows int) []int {
	byRow := make([]int, numRows)

	for row := range byRow {
		byRow[row] = -1
	}

	for index := range placed {
		byRow[placed[index].row] = index
	}

	return byRow
}

// shiftSingleColumnRows moves every built item to its final row position.
func (e *engine) shiftSingleColumnRows(children []*box, numRows int, rowYs []float64, posY float64) {
	for row := range numRows {
		cblock := children[row]
		if cblock == nil {
			continue
		}

		deltaY := posY + rowYs[row] - cblock.y
		e.shiftBoxOps(cblock, 0, deltaY)
		cblock.y += deltaY
	}
}

// buildSingleColumnItem builds one item at its cell start and applies the
// inline justify offset. It returns nil when the item produced no box.
func (e *engine) buildSingleColumnItem(
	page *gridCell, containerJustify string, cellX, cellW, posY, rowTop float64,
) *box {
	child := e.stylePtr(page.n)
	justify := gridItemJustifyPtr(child, containerJustify)
	buildW := gridItemBuildWidthPtr(e, page.n, child, justify, cellW)

	oldMax := e.imgMaxW

	if cellW > 0 {
		e.imgMaxW = cellW
	}

	built := e.build(page.n, buildW, cellX, posY+rowTop)
	e.imgMaxW = oldMax

	if built == nil {
		return nil
	}

	deltaX := cellX + gridAlignOffset(justify, cellW, built.w) - built.x
	e.shiftBoxOps(built, deltaX, 0)
	built.x += deltaX

	return built
}
