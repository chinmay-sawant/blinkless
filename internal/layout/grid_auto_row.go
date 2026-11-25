package layout

import (
	"slices"
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// autoRowBuiltItem is one item the auto-row fast path built once.
type autoRowBuiltItem struct {
	b         *box
	style     *ResolvedStyle
	cellW     float64
	chromeIdx int
	chromeAt  int
}

// autoRowStretch is one deferred stretch request for an item shorter than its
// row.
type autoRowStretch struct {
	idx                 int
	target              float64
	chromeIdx, chromeAt int
}

// canBuildAutoRowGrid reports whether a multi-column grid can build each item
// once. The rows are auto-sized, each item occupies one cell, and the placed
// order is row-major fill order. Two-pass cases stay on the general path.
func canBuildAutoRowGrid(eng *engine, sty ResolvedStyle, placed []gridCell, cols []float64) bool {
	if len(cols) < 2 || len(placed) == 0 {
		return false
	}

	if strings.TrimSpace(sty.grid().GridTemplateRows) != "" ||
		strings.TrimSpace(sty.grid().GridTemplateAreas) != "" {
		return false
	}

	if autoRows := strings.TrimSpace(sty.grid().GridAutoRows); autoRows != "" && !strings.EqualFold(autoRows, "auto") {
		return false
	}

	for _, page := range placed {
		if !autoRowCellEligible(eng, page, cols, sty.flex().JustifyItems) {
			return false
		}
	}

	return autoRowPlacementSequential(placed)
}

// autoRowCellEligible reports whether one cell and its item keep the one-pass
// shape. Percentage heights, explicit placement, and stretch hazards fall
// back to the two-pass path.
func autoRowCellEligible(eng *engine, page gridCell, cols []float64, containerJustify string) bool {
	if !autoRowCellShape(page, cols) {
		return false
	}

	child := eng.stylePtr(page.n)
	if child == nil || autoRowExplicitPlacement(child) {
		return false
	}

	if autoRowPaintHazard(child) || autoRowHeightHazard(child) ||
		!gridJustifyEligible(child, containerJustify) || !autoRowInnerGridEligible(child) {
		return false
	}

	return !itemHasStretchHazard(eng, page.n)
}

// autoRowCellShape reports whether one placed cell occupies exactly one track
// inside the column list.
func autoRowCellShape(page gridCell, cols []float64) bool {
	return page.rowSpan == 1 && page.colSpan == 1 && page.col >= 0 && page.col < len(cols)
}

// autoRowExplicitPlacement reports whether the item names a grid line or area.
func autoRowExplicitPlacement(child *ResolvedStyle) bool {
	return child.grid().GridRowStart != 0 || child.grid().GridColumnStart != 0 ||
		child.grid().GridRowEnd != 0 || child.grid().GridColumnEnd != 0 || child.grid().GridArea != ""
}

// autoRowPaintHazard reports styles that carry their own compositing or
// out-of-flow placement and cannot be stretched by chrome alone.
func autoRowPaintHazard(child *ResolvedStyle) bool {
	return autoRowOutOfFlowHazard(child) || autoRowCompositingHazard(child) ||
		autoRowLayoutHazard(child) || autoRowClipHazard(child)
}

// autoRowOutOfFlowHazard reports boxes placed outside the normal flow.
func autoRowOutOfFlowHazard(child *ResolvedStyle) bool {
	return child.Position == positionAbsolute || child.Position == positionFixed || child.Position == "sticky"
}

// autoRowCompositingHazard reports boxes that create their own paint group.
func autoRowCompositingHazard(child *ResolvedStyle) bool {
	return child.effects().HasTransform || child.effects().Isolation == unicodeBidiIsolate ||
		nonNormalBlendMode(child.effects().MixBlendMode) != "" || maskGroupRequested(child)
}

// autoRowLayoutHazard reports boxes whose used height depends on a second
// layout at the stretched size.
func autoRowLayoutHazard(child *ResolvedStyle) bool {
	return child.Display == displayFlex || child.Display == displayInlineFlex ||
		isMulticol(child) || child.Display == displayTable || child.Display == displaySubgrid ||
		isVerticalWritingMode(child.positioning().WritingMode)
}

// autoRowClipHazard reports boxes that clip their own paint. A chrome-only
// stretch cannot re-clip the item's descendants safely.
func autoRowClipHazard(child *ResolvedStyle) bool {
	return clipsPaintAxis(child, child.geometry().OverflowX) || clipsPaintAxis(child, child.geometry().OverflowY)
}

// autoRowHeightHazard reports percentage heights that depend on a definite
// row height the one-pass path has not resolved yet.
func autoRowHeightHazard(child *ResolvedStyle) bool {
	return child.HeightPercent >= 0 || child.geometry().MinHeightPercent >= 0 || child.geometry().MaxHeightPercent >= 0
}

// gridJustifyEligible reports whether the item's inline justify leaves the
// item at the cell width.
func gridJustifyEligible(child *ResolvedStyle, containerJustify string) bool {
	justify := gridItemJustifyPtr(child, containerJustify)

	return justify == "" || justify == fxStretch || justify == "auto"
}

// autoRowInnerGridEligible reports whether a nested grid can be built once
// inside the row. A nested explicit row template or align-content needs the
// measure pass.
func autoRowInnerGridEligible(child *ResolvedStyle) bool {
	if child.Display != displayGrid && child.Display != displayInlineGrid {
		return true
	}

	if strings.TrimSpace(child.grid().GridTemplateRows) != "" {
		return false
	}

	align := strings.TrimSpace(child.flex().AlignContent)

	return align == "" || align == "normal" || align == fxStretch
}

// autoRowPlacementSequential reports whether placed items fill rows in order,
// with no gaps and no repeated column.
func autoRowPlacementSequential(placed []gridCell) bool {
	row, col := -1, 0

	for _, page := range placed {
		if page.row != row {
			if row >= 0 && page.row != row+1 {
				return false
			}

			row, col = page.row, 0
		}

		if page.col != col {
			return false
		}

		col++
	}

	return true
}

// itemHasStretchHazard reports descendant styles whose used size depends on a
// definite item height. The chrome-only stretch cannot resolve those without
// laying the subtree out again.
func itemHasStretchHazard(eng *engine, node *html.Node) bool {
	return subtreeHasStretchHazard(eng, node)
}

func subtreeHasStretchHazard(eng *engine, parent *html.Node) bool {
	for _, child := range parent.Children {
		if child.Type == html.ElementNode && styleHasStretchHazard(eng.stylePtr(child)) {
			return true
		}

		if len(child.Children) > 0 && subtreeHasStretchHazard(eng, child) {
			return true
		}
	}

	return false
}

// styleHasStretchHazard reports whether one style needs a real layout at the
// stretched height instead of a chrome-only stretch.
func styleHasStretchHazard(style *ResolvedStyle) bool {
	if style == nil {
		return false
	}

	if style.HeightPercent >= 0 || style.geometry().MinHeightPercent >= 0 || style.geometry().MaxHeightPercent >= 0 {
		return true
	}

	return (style.Position == positionAbsolute || style.Position == positionFixed) && !style.positioning().BottomAuto
}

// layoutAutoRowGrid builds each item once, then stretches or aligns it after
// the row height is known. The single-column fast path stays separate.
func (e *engine) layoutAutoRowGrid(
	boxNode *box, sty ResolvedStyle, placed []gridCell,
	cols []float64, columnGap, rowGap, contentX, contentW, curY, posY, contentH float64,
	contentStart int,
) *box {
	var noAreas gridTemplateAreasMap

	numRows := gridRowCount(placed, noAreas)
	items := make([]autoRowBuiltItem, len(placed))

	rows := rentFloats(numRows)
	defer returnFloats(rows)

	containerJustify := sty.flex().JustifyItems
	if containerJustify == "" {
		containerJustify = fxStretch
	}

	containerAlign := sty.flex().AlignItems
	if containerAlign == "" {
		containerAlign = fxStretch
	}

	var stretches []autoRowStretch

	rowTop := curY
	placedIdx := 0

	for row := range numRows {
		rowStart := placedIdx
		for placedIdx < len(placed) && placed[placedIdx].row == row {
			placedIdx++
		}

		rowEnd := placedIdx
		maxH := 0.0

		for idx := rowStart; idx < rowEnd; idx++ {
			item := e.buildAutoRowItem(&placed[idx], cols, columnGap, contentX, posY, rowTop, containerJustify)
			if item.b == nil {
				continue
			}

			items[idx] = item

			if item.b.height > maxH {
				maxH = item.b.height
			}
		}

		rows[row] = maxH
		stretches = e.applyAutoRowStretch(rowStart, rowEnd, items, stretches[:0], containerAlign, maxH, posY, rowTop)
		rowTop += maxH + rowGap
	}

	e.appendAutoRowBoxes(boxNode, items)

	rowYs := rentFloats(numRows)
	defer returnFloats(rowYs)
	gridRowTopsInto(rowYs, curY, rows, rowGap, contentH, sty.flex().AlignContent)

	e.shiftAutoRowBoxes(placed, items, rowYs, posY)
	e.paintGridGapRules(sty, placed, contentX, contentW, cols, columnGap, rowYs, rows, posY)

	usedH := autoRowUsedHeight(curY, numRows, rowYs, rows)
	usedH = resolveGridUsedHeight(e, sty, usedH, contentH)
	boxNode.height = usedH
	e.prependChrome(contentStart, boxNode, &sty, boxNode.x, posY, boxNode.w, boxNode.height)

	return boxNode
}

// appendAutoRowBoxes appends the built items to the grid box in placement
// order, skipping items that produced no box.
func (e *engine) appendAutoRowBoxes(boxNode *box, items []autoRowBuiltItem) {
	if len(boxNode.children) == 0 && len(items) > 0 {
		boxNode.children = make([]*box, 0, len(items))
	}

	for index := range items {
		if items[index].b != nil {
			boxNode.children = append(boxNode.children, items[index].b)
		}
	}
}

// shiftAutoRowBoxes moves every built item to its final row position.
func (e *engine) shiftAutoRowBoxes(placed []gridCell, items []autoRowBuiltItem, rowYs []float64, posY float64) {
	for index := range placed {
		itemBox := items[index].b
		if itemBox == nil {
			continue
		}

		deltaY := posY + rowYs[placed[index].row] - itemBox.y
		e.shiftBoxOps(itemBox, 0, deltaY)
		itemBox.y += deltaY
	}
}

// autoRowUsedHeight returns the content height the rows occupy, or the start
// position when the grid holds no rows.
func autoRowUsedHeight(curY float64, numRows int, rowYs, rows []float64) float64 {
	if numRows == 0 {
		return curY
	}

	return rowYs[numRows-1] + rows[numRows-1]
}

// buildAutoRowItem builds one item at its cell start and records the deferred
// chrome that a later stretch must replace. It returns a zero item when the
// build produced no box.
func (e *engine) buildAutoRowItem(
	page *gridCell, cols []float64, columnGap, contentX, posY, rowTop float64, containerJustify string,
) autoRowBuiltItem {
	cellW, cellX := gridCellExtent(cols, columnGap, contentX, *page)
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
		var empty autoRowBuiltItem

		return empty
	}

	deltaX := cellX + gridAlignOffset(justify, cellW, built.w) - built.x
	e.shiftBoxOps(built, deltaX, 0)
	built.x += deltaX

	chromeIdx, chromeAt := -1, -1

	for i := len(e.deferredChrome) - 1; i >= 0; i-- {
		if e.deferredChrome[i].b == built {
			chromeIdx, chromeAt = i, e.deferredChrome[i].at

			break
		}
	}

	return autoRowBuiltItem{
		b: built, style: child, cellW: cellW, chromeIdx: chromeIdx, chromeAt: chromeAt,
	}
}

// applyAutoRowStretch stretches the items shorter than the row or aligns them
// when their align value does not stretch. Deferred chrome entries are removed
// in reverse index order before the replacement chrome is prepended.
func (e *engine) applyAutoRowStretch(
	rowStart, rowEnd int, items []autoRowBuiltItem, stretches []autoRowStretch,
	containerAlign string, maxH, posY, rowTop float64,
) []autoRowStretch {
	for idx := rowStart; idx < rowEnd; idx++ {
		item := &items[idx]
		if item.b == nil {
			continue
		}

		align := gridItemAlignPtr(item.style, containerAlign)
		buildH := gridStretchBuildHeightPtr(align, maxH, item.style)

		if buildH > 0 {
			target := buildH
			if item.style.geometry().MaxHeight >= 0 {
				if maxHeight := e.scalePt(item.style.geometry().MaxHeight); maxHeight < target {
					target = maxHeight
				}
			}

			if target > item.b.height {
				stretches = append(stretches, autoRowStretch{
					idx: idx, target: target, chromeIdx: item.chromeIdx, chromeAt: item.chromeAt,
				})
			}

			continue
		}

		deltaY := posY + rowTop + gridAlignOffset(align, maxH, item.b.height) - item.b.y
		e.shiftBoxOps(item.b, 0, deltaY)
		item.b.y += deltaY
	}

	if len(stretches) == 0 {
		return stretches
	}

	if len(stretches) > 1 {
		slices.SortFunc(stretches, func(a, b autoRowStretch) int { return b.chromeIdx - a.chromeIdx })
	}

	for _, stretch := range stretches {
		if stretch.chromeIdx < 0 {
			continue
		}

		idx := stretch.chromeIdx
		e.deferredChrome = append(e.deferredChrome[:idx], e.deferredChrome[idx+1:]...)
	}

	for _, stretch := range stretches {
		item := &items[stretch.idx]
		e.stretchGridItemToHeight(item.b, *item.style, stretch.target, stretch.chromeAt)
	}

	return stretches
}

// stretchGridItemToHeight extends an item's border box and regenerates its
// deferred background and border at the new height. The subtree stays put.
func (e *engine) stretchGridItemToHeight(itemBox *box, style ResolvedStyle, height float64, chromeAt int) {
	if height <= itemBox.height {
		return
	}

	itemBox.height = height

	if e.noEmit || chromeAt < 0 {
		return
	}

	scope := e.pushZ(&style, itemBox.style)
	e.prependChrome(chromeAt, itemBox, &style, itemBox.x, itemBox.y, itemBox.w, height)
	e.popZ(scope, nil)
}
