package layout

import (
	"sort"
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// canBuildAutoRowGrid reports whether a multi-column grid can build each item
// once. The rows are auto-sized, each item occupies one cell, and the placed
// order is row-major fill order. Two-pass cases stay on the general path.
func canBuildAutoRowGrid(e *engine, sty ResolvedStyle, placed []gridCell, cols []float64) bool {
	if len(cols) < 2 || len(placed) == 0 {
		return false
	}
	if strings.TrimSpace(sty.GridTemplateRows) != "" ||
		strings.TrimSpace(sty.GridTemplateAreas) != "" {
		return false
	}
	if autoRows := strings.TrimSpace(sty.GridAutoRows); autoRows != "" && !strings.EqualFold(autoRows, "auto") {
		return false
	}

	row, col := -1, 0
	for _, page := range placed {
		if page.rowSpan != 1 || page.colSpan != 1 || page.col < 0 || page.col >= len(cols) {
			return false
		}
		child := e.stylePtr(page.n)
		if child == nil || child.GridRowStart != 0 || child.GridColumnStart != 0 ||
			child.GridRowEnd != 0 || child.GridColumnEnd != 0 || child.GridArea != "" {
			return false
		}
		if child.Position == "absolute" || child.Position == "fixed" || child.Position == "sticky" ||
			child.HasTransform || child.Isolation == "isolate" ||
			nonNormalBlendMode(child.MixBlendMode) != "" || maskGroupRequested(*child) ||
			child.Display == displayFlex || child.Display == displayInlineFlex ||
			child.ColumnCount > 0 {
			return false
		}
		if child.HeightPercent >= 0 || child.MinHeightPercent >= 0 || child.MaxHeightPercent >= 0 {
			return false
		}
		if justify := gridItemJustify(*child, sty.JustifyItems); justify != "" &&
			justify != fxStretch && justify != "auto" {
			return false
		}
		if child.Display == displayGrid || child.Display == displayInlineGrid {
			if strings.TrimSpace(child.GridTemplateRows) != "" {
				return false
			}
			align := strings.TrimSpace(child.AlignContent)
			if align != "" && align != "normal" && align != "stretch" {
				return false
			}
		}
		if itemHasStretchHazard(e, page.n) {
			return false
		}

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
func itemHasStretchHazard(e *engine, node *html.Node) bool {
	hazard := false
	node.Walk(func(child *html.Node) {
		if hazard || child.Type != html.ElementNode || child == node {
			return
		}
		style := e.stylePtr(child)
		if style == nil {
			return
		}
		if style.HeightPercent >= 0 || style.MinHeightPercent >= 0 || style.MaxHeightPercent >= 0 {
			hazard = true

			return
		}
		if (style.Position == "absolute" || style.Position == "fixed") && !style.BottomAuto {
			hazard = true
		}
	})

	return hazard
}

// layoutAutoRowGrid builds each item once, then stretches or aligns it after
// the row height is known. The single-column fast path stays separate.
func (e *engine) layoutAutoRowGrid(
	boxNode *box, sty ResolvedStyle, placed []gridCell,
	cols []float64, columnGap, rowGap, contentX, contentW, curY, posY, contentH float64,
	contentStart int,
) *box {
	numRows := gridRowCount(placed, gridTemplateAreasMap{})
	rowItems := make([][]int, numRows)
	for i := range placed {
		rowItems[placed[i].row] = append(rowItems[placed[i].row], i)
	}

	type builtItem struct {
		b         *box
		style     *ResolvedStyle
		cellW     float64
		chromeIdx int
		chromeAt  int
	}
	items := make([]builtItem, len(placed))
	rows := make([]float64, numRows)

	containerJustify := sty.JustifyItems
	if containerJustify == "" {
		containerJustify = fxStretch
	}
	containerAlign := sty.AlignItems
	if containerAlign == "" {
		containerAlign = fxStretch
	}

	y := curY
	for row := 0; row < numRows; row++ {
		rowTop := y
		maxH := 0.0

		for _, idx := range rowItems[row] {
			page := &placed[idx]
			cellW, cx := gridCellExtent(cols, columnGap, contentX, *page)
			child := e.stylePtr(page.n)
			justify := gridItemJustify(*child, containerJustify)
			buildW := gridItemBuildWidth(e, page.n, *child, justify, cellW)

			oldMax := e.imgMaxW
			if cellW > 0 {
				e.imgMaxW = cellW
			}
			b := e.build(page.n, buildW, cx, posY+rowTop)
			e.imgMaxW = oldMax
			if b == nil {
				continue
			}

			deltaX := cx + gridAlignOffset(justify, cellW, b.w) - b.x
			e.shiftBoxOps(b, deltaX, 0)
			b.x += deltaX

			chromeIdx, chromeAt := -1, -1
			for i := len(e.deferredChrome) - 1; i >= 0; i-- {
				if e.deferredChrome[i].b == b {
					chromeIdx, chromeAt = i, e.deferredChrome[i].at

					break
				}
			}
			items[idx] = builtItem{
				b: b, style: child, cellW: cellW, chromeIdx: chromeIdx, chromeAt: chromeAt,
			}
			if b.height > maxH {
				maxH = b.height
			}
		}

		rows[row] = maxH

		type stretchReq struct {
			idx                 int
			target              float64
			chromeIdx, chromeAt int
		}
		stretches := make([]stretchReq, 0, len(rowItems[row]))
		for _, idx := range rowItems[row] {
			item := &items[idx]
			if item.b == nil {
				continue
			}
			align := gridItemAlign(*item.style, containerAlign)
			buildH := gridStretchBuildHeight(align, maxH, *item.style)
			if buildH > 0 {
				target := buildH
				if item.style.MaxHeight >= 0 {
					if maxHgt := e.scalePt(item.style.MaxHeight); maxHgt < target {
						target = maxHgt
					}
				}
				if target > item.b.height {
					stretches = append(stretches, stretchReq{
						idx: idx, target: target, chromeIdx: item.chromeIdx, chromeAt: item.chromeAt,
					})
				}

				continue
			}

			deltaY := posY + rowTop + gridAlignOffset(align, maxH, item.b.height) - item.b.y
			e.shiftBoxOps(item.b, 0, deltaY)
			item.b.y += deltaY
		}

		sort.Slice(stretches, func(i, j int) bool { return stretches[i].chromeIdx > stretches[j].chromeIdx })
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

		y += maxH + rowGap
	}

	for i := range placed {
		if items[i].b != nil {
			boxNode.children = append(boxNode.children, items[i].b)
		}
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
	for i := range placed {
		b := items[i].b
		if b == nil {
			continue
		}
		deltaY := posY + rowYs[placed[i].row] - b.y
		e.shiftBoxOps(b, 0, deltaY)
		b.y += deltaY
	}

	e.paintGridGapRules(sty, placed, contentX, contentW, cols, columnGap, rowYs, rows, posY)

	usedH := curY
	if numRows > 0 {
		usedH = rowYs[numRows-1] + rows[numRows-1]
	}
	usedH = resolveGridUsedHeight(e, sty, usedH, contentH)
	boxNode.height = usedH
	e.prependChrome(contentStart, boxNode, sty, boxNode.x, posY, boxNode.w, boxNode.height)

	return boxNode
}

// stretchGridItemToHeight extends an item's border box and regenerates its
// deferred background and border at the new height. The subtree stays put.
func (e *engine) stretchGridItemToHeight(b *box, style ResolvedStyle, height float64, chromeAt int) {
	if height <= b.height {
		return
	}
	b.height = height
	if e.noEmit || chromeAt < 0 {
		return
	}

	scope := e.pushZ(style, b.style)
	e.prependChrome(chromeAt, b, style, b.x, b.y, b.w, height)
	e.popZ(scope, nil)
}
