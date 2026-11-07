package layout

import (
	"context"
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// Row-granular table splicing.
//
// A tr places no box of its own, so a patch of a table row or cell used to
// escalate to the owning table and rebuild every row before splicing the
// table back. For a wide table that cost is linear in the table size even
// when a single row changed. This file adds a narrower attempt in front of
// that escalation: when the patched element is (or retargets to) a table
// row or cell whose siblings are untouched, only the patched row's cells are
// rebuilt and the row's operation window is spliced back. The table as a
// whole is rebuilt only when the row rebuild actually moves siblings.
//
// The attempt runs on the patch path only. Layout never calls it, so the
// full-layout output cannot change. Every doubt declines the attempt (ok is
// false) and the caller falls back to the existing whole-table escalation
// with byte-identical behavior to before. In particular:
//
//   - The row rebuild reuses the table pipeline's own per-row functions
//     (buildCell, measureCellHeight, emitCell, emitCollapsedRowGrid) with the
//     kept column widths, spacings, and row tops, so a spliced row carries
//     the same boxes and operations a whole-table rebuild would produce,
//     including the hidden atomic inline boxes the visibility path keeps.
//   - Column widths are recovered from the kept cells, never re-derived, so
//     the attempt only proceeds when the row's width contributions
//     (contentW/contentMin) and width hints are unchanged: any edit that
//     would move a column declines and the table path re-derives the grid.
//   - A size change (row height, spans, cell counts, spans, box shape)
//     declines and the table path escalates exactly as before.
//   - State-driven restyles (:hover and friends) decline: the kept styles of
//     the untouched rows were resolved under a previous state the retained
//     result does not record, so only the whole-table rebuild can heal them.
//
// No new globals, no new goroutines: the attempt rents the pooled subtree
// style store the existing patch path uses and builds on a fresh engine.
const (
	// tableRowEmitMinHeight is the content-height floor below which a cell
	// emits no paint, the same cutoff emitTableCells uses.
	tableRowEmitMinHeight = 0.01
	// tableRowCounterFn and tableRowCountersFn mark content that evaluates a
	// counter. Counters resolve against engine state a fresh row engine does
	// not carry, so such a row declines and the table path keeps its verdict.
	tableRowCounterFn  = "counter("
	tableRowCountersFn = "counters("
)

// tableRowPatch is one rebuilt table row waiting to be committed: the fresh
// cells with fresh-engine operation ranges, the engine that built them, the
// styles they were resolved with, and the kept operation window they replace.
type tableRowPatch struct {
	tableBox   *box
	target     *relayoutTarget
	trNode     *html.Node
	rowIdx     int
	keptCells  []*box
	freshCells []*box
	eng        *engine
	styles     map[*html.Node]*ResolvedStyle
	start      int
	oldCount   int
}

// tryRelayoutTableRowPatch attempts the row-granular splice for the single
// element path. cellTarget is the found cell target when the patched element
// owns a cell box, or nil when the patched element owns no box. It returns
// ok false when the row cannot be spliced alone; the caller then runs the
// existing table escalation unchanged.
func tryRelayoutTableRowPatch(
	ctx context.Context, prev *Result, node *html.Node, cellTarget *relayoutTarget, opts Options,
) (*Result, bool) {
	patch, ok := buildTableRowPatch(ctx, prev, node, cellTarget, opts)
	if !ok {
		return nil, false
	}

	return commitTableRowPatch(prev, patch), true
}

// tryRelayoutTableRowBatch attempts the row-granular splice for one batch
// id. It mirrors relayoutSpliceOne's bookkeeping for a committed window and
// reports done true when the id spliced at row granularity. ownersDirty marks
// the deferred owners derivation the commit leaves behind.
func tryRelayoutTableRowBatch(
	ctx context.Context, prev *Result, thread *boxThread, node *html.Node, cellTarget *relayoutTarget,
	baseOpts Options, out *SubtreeBatchResult, relevant, grouped *int, ownersDirty *bool,
) bool {
	patch, ok := buildTableRowPatch(ctx, prev, node, cellTarget, baseOpts)
	if !ok {
		return false
	}

	commitTableRowPatchBatch(prev, patch, thread, out, relevant, grouped, ownersDirty)

	return true
}

// buildTableRowPatch locates the patched row, checks that a row-only rebuild
// reproduces the table build for it, and rebuilds just that row's cells.
// Any doubt returns ok false without touching prev.
func buildTableRowPatch(
	ctx context.Context, prev *Result, node *html.Node, cellTarget *relayoutTarget, opts Options,
) (*tableRowPatch, bool) {
	// A viewport change re-derives every column width, so the kept widths
	// the row rebuild reuses would be stale. The table path re-derives them.
	if !relayoutNear(opts.Width, prev.Width) {
		return nil, false
	}

	// A stateful patch may restyle rows the patch does not touch. Their kept
	// styles were resolved under a state the retained result does not record,
	// so only the whole-table rebuild can heal them.
	if opts.State.Hover != "" || opts.State.Focus != "" || opts.State.Active != "" {
		return nil, false
	}

	target, trNode, rowIdx, keptCells, ok := locateTableRow(prev, node, cellTarget)
	if !ok {
		return nil, false
	}

	tableBox := target.box

	// The kept-side structural checks are the table path's own: floats,
	// out-of-flow content, transformed ancestors, and outside anchors refuse
	// a row splice exactly where they refuse the table splice.
	if err := checkRelayoutTarget(prev, target); err != nil {
		return nil, false
	}

	tableStyle := tableBox.style
	if !relayoutTablePlainForRow(tableStyle) {
		return nil, false
	}

	if !tableRowChildrenAreCells(tableBox, prev) {
		return nil, false
	}

	for _, cell := range keptCells {
		if cell == nil || cell.node == nil || cell.rowSpan != 1 {
			return nil, false
		}

		if cell.col < 0 || cell.span < 1 {
			return nil, false
		}
	}

	if !tableRowCountersClean(tableBox, trNode, prev.styles) {
		return nil, false
	}

	styles, ok := relayoutTableRowStyles(ctx, prev, target, trNode, opts)
	if !ok {
		return nil, false
	}

	trStyle := styles[trNode]
	if trStyle == nil {
		return nil, false
	}

	if trStyle.Display != displayTableRow || trStyle.Visibility == borderCollapseValue {
		return nil, false
	}

	eng, err := relayoutEngine(ctx, prev, opts, styles)
	if err != nil {
		return nil, false
	}

	if !tableRowCellsMatch(eng, trNode, keptCells, styles) {
		return nil, false
	}

	if !tableRowFreshClean(trNode, styles) {
		return nil, false
	}

	colW, nCols, ok := tableRowColumnWidths(tableBox)
	if !ok {
		return nil, false
	}

	fresh, rowH, ok := buildTableRowCells(
		eng, keptCells, colW, eng.tableHSpacing(*tableStyle), tableRowPadLeft(eng, tableStyle),
		tableBox.x, nCols, rowIdx, tableBox.rowTops,
	)
	if !ok {
		return nil, false
	}

	if rowIdx >= len(tableBox.rowHeights) || rowIdx >= len(tableBox.rowTops) ||
		!relayoutNear(rowH, tableBox.rowHeights[rowIdx]) {
		return nil, false
	}

	return finishTableRowPatch(
		prev, target, tableBox, tableStyle, trNode, rowIdx, keptCells, fresh,
		colW, eng, styles, opts,
	)
}

// finishTableRowPatch paints the rebuilt row, runs the table build's own
// post steps over it, and checks that the row kept its geometry. The holder
// carries the table's style and geometry (never the row's) so the chrome
// merge, the overflow clip seeding, and the transform stamp read exactly
// what the table build would read; it is scaffolding and is never committed.
func finishTableRowPatch(
	prev *Result, target *relayoutTarget, tableBox *box, tableStyle *ResolvedStyle, trNode *html.Node,
	rowIdx int, keptCells, fresh []*box, colW []float64, eng *engine,
	styles map[*html.Node]*ResolvedStyle, opts Options,
) (*tableRowPatch, bool) {
	// The paint state the table build emits under: the ancestors' stacking
	// seeding plus the table's own positioned flag. A plain table installs
	// nothing else, which relayoutTablePlainForRow guarantees.
	viewportFixed := relayoutEnterAncestorPaint(eng, target)

	if tableStyle.Position == positionAbsolute || tableStyle.Position == positionFixed {
		eng.positioned = true
	}

	for _, cell := range fresh {
		cell.y = tableBox.rowTops[rowIdx]
		cell.height = tableBox.rowHeights[rowIdx]
		cell.rowBoxH = tableBox.rowHeights[rowIdx]
	}

	if !emitTableRowCells(eng, tableBox, fresh, rowIdx, colW, tableStyle) || eng.err != nil {
		return nil, false
	}

	holder := &box{ //nolint:exhaustruct // transient finalize scaffolding, never committed
		node: tableBox.node, style: tableStyle, kind: boxKindTable,
		x: tableBox.x, y: tableBox.y, w: tableBox.w, height: tableBox.height,
		children: fresh, opStart: 0, opEnd: len(eng.ops) - 1,
	}

	eng.finalizeChrome(holder)

	if viewportFixed || tableStyle.Position == positionFixed {
		eng.markOpsFixed(0, len(eng.ops)-1)
	}

	relayoutApplyDocumentColor(eng, prev, tableBox.node)
	relayoutApplyAncestorClip(eng, target)

	if eng.needsXformStamp {
		propagatePerspective(holder)
		stampBoxTransforms(holder, IdentityMatrix(), eng.ops, opts.Media)
	}

	if eng.err != nil {
		return nil, false
	}

	// Blend groups number by creation order in a fresh engine. A group the
	// row creates would number differently than the table build's, so such a
	// row declines and the table path keeps its numbering.
	if eng.nextGroupID != 0 {
		return nil, false
	}

	for index := range fresh {
		if !sameBoxGeometry(fresh[index], keptCells[index]) {
			return nil, false
		}
	}

	start, oldCount, ok := tableRowWindow(prev, target, tableBox, rowIdx, keptCells)
	if !ok {
		return nil, false
	}

	return &tableRowPatch{
		tableBox: tableBox, target: target, trNode: trNode, rowIdx: rowIdx,
		keptCells: keptCells, freshCells: fresh, eng: eng, styles: styles,
		start: start, oldCount: oldCount,
	}, true
}

// locateTableRow resolves the patched element to its table row: the owning
// table target, the tr node, its kept row index, and the row's kept cells.
// The row must own exactly the kept cells whose cell node sits directly
// under the tr; anything else (a moved row, a structural edit above the row)
// declines so the table path places it.
func locateTableRow(
	prev *Result, node *html.Node, cellTarget *relayoutTarget,
) (*relayoutTarget, *html.Node, int, []*box, bool) {
	var trNode *html.Node

	if cellTarget != nil {
		if cellTarget.box == nil || cellTarget.box.kind != boxKindCell || cellTarget.box.node == nil {
			return nil, nil, 0, nil, false
		}

		trNode = cellTarget.box.node.Parent
	} else {
		if node == nil {
			return nil, nil, 0, nil, false
		}

		for current := node; current != nil; current = current.Parent {
			if style := prev.styles[current]; style != nil && style.Display == displayTableRow {
				trNode = current

				break
			}
		}
	}

	if trNode == nil {
		return nil, nil, 0, nil, false
	}

	target, ok := tableAncestorTarget(prev.boxTreeRoot(), trNode)
	if !ok || target.box == nil || target.box.node == nil {
		return nil, nil, 0, nil, false
	}

	tableBox := target.box
	rowIdx := -1

	for index, row := range tableBox.rows {
		for _, cell := range row {
			if cell != nil && cell.node != nil && cell.node.Parent == trNode {
				if rowIdx >= 0 && rowIdx != index {
					return nil, nil, 0, nil, false
				}

				rowIdx = index
			}
		}
	}

	if rowIdx < 0 || rowIdx >= len(tableBox.rows) {
		return nil, nil, 0, nil, false
	}

	kept := tableBox.rows[rowIdx]
	if len(kept) == 0 {
		return nil, nil, 0, nil, false
	}

	for _, cell := range kept {
		if cell == nil || cell.node == nil || cell.node.Parent != trNode {
			return nil, nil, 0, nil, false
		}
	}

	return target, trNode, rowIdx, kept, true
}

// relayoutTablePlainForRow reports whether the table's own style lets a row
// rebuild stand in for the table build. Anything pushZ would install
// (stacking, grouping, blend isolation) or the stamp pass would thread from
// the table (transform, opacity, perspective) must be absent, so the fresh
// row cells build under the same engine state and stamp the same operations
// with only the ancestor seeding both paths share. The positioned flag for
// an absolute or fixed table is replicated directly at the build site.
func relayoutTablePlainForRow(style *ResolvedStyle) bool {
	if style == nil {
		return false
	}

	if style.HasTransform || style.HasTransform3D || style.HasPerspective || style.ZIndexSet {
		return false
	}

	if style.Opacity < 1 {
		return false
	}

	if style.MixBlendMode != blendNormal || style.Isolation == unicodeBidiIsolate {
		return false
	}

	if maskGroupRequested(style) {
		return false
	}

	if nonNormalBlendMode(style.MixBlendMode) != "" || style.Isolation == "isolate" ||
		hasBackdropFilter(style.BackdropFilter) {
		return false
	}

	return true
}

// tableRowChildrenAreCells reports whether the kept table holds no caption
// or other non-cell child and no caption was added under it. Captions paint
// outside the row windows, so only a caption-free table splices by row.
func tableRowChildrenAreCells(tableBox *box, prev *Result) bool {
	if tableBox == nil {
		return false
	}

	for _, child := range tableBox.children {
		if child == nil || child.kind != boxKindCell {
			return false
		}
	}

	if tableBox.node == nil {
		return false
	}

	for _, child := range tableBox.node.Children {
		if child.Type != html.ElementNode || child.Name != htmlCaption {
			continue
		}

		if style := prev.styles[child]; style == nil || style.Display != cssDisplayNone {
			return false
		}
	}

	return true
}

// tableRowCountersClean reports whether no style the row rebuild depends on
// evaluates a counter. Counters resolve against engine state a fresh row
// engine restarts, so only the table build (which replays every row in
// order) may place counter content. The scan covers the kept styles of the
// whole table subtree because increments in sibling rows change what the
// patched row would display.
func tableRowCountersClean(tableBox *box, trNode *html.Node, styles map[*html.Node]*ResolvedStyle) bool {
	for current := trNode; current != nil; current = current.Parent {
		if style := styles[current]; style != nil && !counterStyleClean(style) {
			return false
		}

		if current == tableBox.node {
			break
		}
	}

	var walk func(boxNode *box) bool

	walk = func(boxNode *box) bool {
		if boxNode == nil {
			return true
		}

		if boxNode.node != nil {
			if style := styles[boxNode.node]; style != nil && !counterStyleClean(style) {
				return false
			}
		}

		for _, child := range boxNode.children {
			if !walk(child) {
				return false
			}
		}

		for _, row := range boxNode.rows {
			for _, cell := range row {
				if !walk(cell) {
					return false
				}
			}
		}

		return true
	}

	return walk(tableBox)
}

// counterStyleClean reports whether a style neither touches a counter nor
// displays one.
func counterStyleClean(style *ResolvedStyle) bool {
	if style.CounterReset != "" || style.CounterIncrement != "" || style.CounterSet != "" {
		return false
	}

	lower := strings.ToLower(style.Content)

	return !strings.Contains(lower, tableRowCounterFn) && !strings.Contains(lower, tableRowCountersFn)
}

// relayoutTableRowStyles resolves the styles of the patched row's subtree.
// The row inherits the kept style of its table row group, the way the table
// build inherits it, and the kept styles of the boxes above the row join the
// map for the containing-block walks, the same join the table path makes.
func relayoutTableRowStyles(
	ctx context.Context, prev *Result, target *relayoutTarget, trNode *html.Node, opts Options,
) (map[*html.Node]*ResolvedStyle, bool) {
	parentNode := trNode.Parent
	if parentNode == nil {
		return nil, false
	}

	parent := prev.styles[parentNode]
	if parent == nil {
		return nil, false
	}

	styleCtx := &styleContext{ //nolint:exhaustruct // the pass fills only the inputs the subtree cascade reads
		ctx:                ctx,
		sheets:             opts.Sheets,
		properties:         registeredProperties(opts.Sheets),
		media:              opts.Media,
		viewportW:          opts.Width,
		viewportH:          opts.Height,
		printLinkUnderline: opts.PrintLinkUnderline,
		state:              opts.State,
	}

	if remBase := relayoutRemBase(prev); remBase > 0 {
		styleCtx.remBase = remBase
	}

	tag := subtreeStyleDocTagOf(prev, opts)
	store := rentSubtreeStyleStore(tag)

	// A failed pass is dropped, never pooled: only a complete pass owns
	// records the next patch may share.
	if err := resolveSubtreeStylesPooled(ctx, styleCtx, trNode, parent, store); err != nil {
		return nil, false
	}

	styles := subtreeStyles(styleCtx, trNode, parent)
	if len(styles) == 0 {
		return nil, false
	}

	relayoutTextStyles(styles)

	tableNode := target.box.node

	for current := parentNode; current != nil; current = current.Parent {
		if _, ok := styles[current]; !ok {
			if kept, ok := prev.styles[current]; ok && kept != nil {
				styles[current] = kept
			}
		}

		if current == tableNode {
			break
		}
	}

	for _, ancestor := range target.ancestors {
		if ancestor.node == nil || ancestor.style == nil {
			continue
		}

		if _, ok := styles[ancestor.node]; !ok {
			styles[ancestor.node] = ancestor.style
		}
	}

	putSubtreeStyleStore(tag, store)

	return styles, true
}

// tableRowCellsMatch reports whether the patched row still holds exactly the
// kept cells: the same nodes in the same order with the same column spans,
// width hints, and table-cell display. A structural edit declines and the
// table path re-places the grid.
func tableRowCellsMatch(
	eng *engine, trNode *html.Node, kept []*box, styles map[*html.Node]*ResolvedStyle,
) bool {
	freshNodes := rowCellNodes(trNode, eng)
	if len(freshNodes) != len(kept) {
		return false
	}

	for index, cellNode := range freshNodes {
		if cellNode != kept[index].node {
			return false
		}

		span, rowSpan := colSpan(cellNode), cellRowSpan(cellNode)
		if span < 1 {
			span = 1
		}

		if rowSpan < 1 {
			rowSpan = 1
		}

		if span != kept[index].span || rowSpan != kept[index].rowSpan {
			return false
		}

		fresh, old := styles[cellNode], kept[index].style
		if fresh == nil || old == nil {
			return false
		}

		if fresh.Width != old.Width || fresh.WidthPercent != old.WidthPercent {
			return false
		}
	}

	return true
}

// tableRowFreshClean reports whether the row's fresh styles keep the
// placeability the kept-side check verified: no float, out-of-flow, sticky,
// anchor, or named flow the row build could not place alone, and no counter
// content the fresh engine would number from scratch. Only nodes under the
// row are read; the joined ancestor styles the map carries are skipped.
func tableRowFreshClean(trNode *html.Node, styles map[*html.Node]*ResolvedStyle) bool {
	for node, style := range styles {
		if style == nil || node == nil || node.Type != html.ElementNode {
			continue
		}

		var under bool

		for current := node; current != nil; current = current.Parent {
			if current == trNode {
				under = true

				break
			}
		}

		if !under {
			continue
		}

		if reason := relayoutSubtreeReason(style); reason != "" {
			return false
		}

		if !counterStyleClean(style) {
			return false
		}
	}

	return true
}

// tableRowColumnWidths recovers the used column widths from the kept cells:
// a single-span cell carries its column's width exactly. Every column must
// have one, or the row cannot be placed at the kept widths.
func tableRowColumnWidths(tableBox *box) ([]float64, int, bool) {
	nCols := 0

	for _, child := range tableBox.children {
		if child == nil {
			return nil, 0, false
		}

		if end := child.col + child.span; end > nCols {
			nCols = end
		}
	}

	if nCols <= 0 {
		return nil, 0, false
	}

	colW := make([]float64, nCols)
	filled := make([]bool, nCols)

	for _, child := range tableBox.children {
		if child.span != 1 {
			continue
		}

		if child.col < 0 || child.col >= nCols {
			return nil, 0, false
		}

		if !filled[child.col] {
			colW[child.col] = child.w
			filled[child.col] = true
		} else if colW[child.col] != child.w {
			return nil, 0, false
		}
	}

	for _, ok := range filled {
		if !ok {
			return nil, 0, false
		}
	}

	return colW, nCols, true
}

// tableRowPadLeft returns the table's left chrome, the origin the table
// build places its first column at.
func tableRowPadLeft(eng *engine, tableStyle *ResolvedStyle) float64 {
	return eng.scalePt(tableStyle.PaddingLeft) + eng.scalePt(tableStyle.BorderLeft.Width)
}

// buildTableRowCells rebuilds the row's cells with the kept column widths:
// each cell measures at its final column width exactly like measureRowCells
// does, and the row height follows the same single-row rule. A changed width
// contribution declines before anything moves.
func buildTableRowCells(
	eng *engine, kept []*box, colW []float64, spacingH, padL, posX float64,
	nCols, rowIdx int, rowTops []float64,
) ([]*box, float64, bool) {
	fresh := make([]*box, 0, len(kept))
	rowH := 0.0

	for _, keptCell := range kept {
		cell := eng.buildCell(keptCell.node, keptCell.col, keptCell.span)
		cell.row, cell.rowSpan = keptCell.row, keptCell.rowSpan

		if !relayoutNear(cell.contentW, keptCell.contentW) ||
			!relayoutNear(cell.contentMin, keptCell.contentMin) {
			return nil, 0, false
		}

		cellW := 0.0

		for k := 0; k < cell.span && cell.col+k < nCols; k++ {
			cellW += colW[cell.col+k]
		}

		cellW += spacingH * float64(cell.span-1)
		cell.w = cellW
		cell.x = posX + padL + spacingH

		for c := 0; c < cell.col && c < nCols; c++ {
			cell.x += colW[c] + spacingH
		}

		cell.y = rowTops[rowIdx]
		eng.measureCellHeight(cell, cellW)

		if cell.rowSpan == 1 && cell.contentH > rowH {
			rowH = cell.contentH
		}

		fresh = append(fresh, cell)
	}

	if eng.err != nil {
		return nil, 0, false
	}

	if rowH > 0 && rowCellsHaveNoInk(fresh) {
		rowH = 0
	}

	return fresh, rowH, true
}

// emitTableRowCells paints the rebuilt row exactly like emitTableCells
// paints one row: the row's cells, then the collapsed grid run folded into
// their ranges. The table chrome and the other rows keep their kept paint.
// The facade carries the fresh row inside the kept table so the border
// conflict scans read the same neighbors the table build would. It reports
// false when the kept child list does not hold the row's cells exactly once.
func emitTableRowCells(
	eng *engine, tableBox *box, fresh []*box, rowIdx int, colW []float64, tableStyle *ResolvedStyle,
) bool {
	padL := tableRowPadLeft(eng, tableStyle)

	facade := *tableBox

	swapped := make([]*box, 0, len(tableBox.children))
	keptSet := make(map[*box]struct{}, len(fresh))

	for _, cell := range tableBox.rows[rowIdx] {
		keptSet[cell] = struct{}{}
	}

	index := 0

	for _, child := range tableBox.children {
		if _, ok := keptSet[child]; ok {
			if index >= len(fresh) {
				return false
			}

			swapped = append(swapped, fresh[index])
			index++
		} else {
			swapped = append(swapped, child)
		}
	}

	if index != len(fresh) {
		return false
	}

	facade.children = swapped

	facadeRows := make([][]*box, len(tableBox.rows))
	copy(facadeRows, tableBox.rows)
	facadeRows[rowIdx] = fresh
	facade.rows = facadeRows

	if tableStyle.BorderCollapse == borderCollapseValue {
		for _, cell := range fresh {
			if cell.height > tableRowEmitMinHeight {
				eng.emitCell(cell, true)
			}
		}

		if tableBox.rowHeights[rowIdx] > tableRowEmitMinHeight {
			xList := gridColumnEdges(tableBox.x+padL, colW)
			last := rowIdx == lastNonEmptyRow(tableBox.rowHeights)
			eng.emitCollapsedRowGrid(&facade, rowIdx, last, xList, tableBox.rowTops, tableBox.rowHeights)
		}

		return true
	}

	for _, cell := range fresh {
		if cell.height > tableRowEmitMinHeight {
			eng.emitCell(cell, false)
		}
	}

	return true
}

// tableRowWindow returns the kept operation window the row owns: the span
// over its cells, widened by the grid run the collapse path folds into them.
// A row that emitted nothing splices at the insertion point the next row's
// operations start at, or at the table window end.
func tableRowWindow(
	prev *Result, target *relayoutTarget, tableBox *box, rowIdx int, kept []*box,
) (int, int, bool) {
	start, end := -1, -1

	for _, cell := range kept {
		if cell.opEnd < cell.opStart {
			continue
		}

		if start < 0 || cell.opStart < start {
			start = cell.opStart
		}

		if cell.opEnd > end {
			end = cell.opEnd
		}
	}

	if start >= 0 {
		return start, end - start + 1, true
	}

	for _, row := range tableBox.rows[rowIdx+1:] {
		for _, cell := range row {
			if cell != nil && cell.opEnd >= cell.opStart && cell.opStart >= 0 {
				if start < 0 || cell.opStart < start {
					start = cell.opStart
				}
			}
		}
	}

	if start >= 0 {
		return start, 0, true
	}

	tableStart, tableCount, err := relayoutWindow(prev, target)
	if err != nil {
		return 0, 0, false
	}

	return tableStart + tableCount, 0, true
}

// swapTableRowCells puts the fresh cells where the kept cells stood: the
// row list and, in order, the table's child list.
func swapTableRowCells(tableBox *box, rowIdx int, kept, fresh []*box) {
	tableBox.rows[rowIdx] = fresh

	keptSet := make(map[*box]struct{}, len(kept))
	for _, cell := range kept {
		keptSet[cell] = struct{}{}
	}

	index := 0

	for childIdx, child := range tableBox.children {
		if _, ok := keptSet[child]; ok && index < len(fresh) {
			tableBox.children[childIdx] = fresh[index]
			index++
		}
	}
}

// spliceTableRowPatch commits a rebuilt row into prev: the row's operation
// window is replaced in the operation list, the fresh cells are rebased and
// take the kept cells' places, and the ranges after the window shift. The
// census and the operation identity high-water mark advance from the window
// alone. inPlace selects the splice for a caller that owns prev exclusively,
// the batch on its private clone: the flattened box list rebuild is skipped
// and the batch rebuilds it once after its last splice.
func spliceTableRowPatch(prev *Result, patch *tableRowPatch, inPlace bool) {
	oldMax, oldFrag := censusOpsWindow(prev.Ops, patch.start, patch.oldCount)

	if len(patch.eng.ops) == patch.oldCount {
		relayoutKeepWindowIDs(prev.Ops[patch.start:patch.start+patch.oldCount], patch.eng.ops)
	}

	if inPlace {
		relayoutSpliceOpsInPlace(prev, patch.start, patch.oldCount, patch.eng.ops)
	} else {
		relayoutSpliceOps(prev, patch.start, patch.oldCount, patch.eng.ops)
	}

	for _, cell := range patch.freshCells {
		relayoutRebaseTree(cell, patch.start)
	}

	delta := len(patch.eng.ops) - patch.oldCount
	relayoutShiftRanges(prev.boxTreeRoot(), nil, patch.start, patch.start+patch.oldCount-1, delta)
	swapTableRowCells(patch.tableBox, patch.rowIdx, patch.keptCells, patch.freshCells)
	unionChildOpRanges(prev.boxTreeRoot())

	if !inPlace {
		relayoutKeepBoxes(prev)
	}

	relayoutKeepStyles(prev, patch.styles)
	refreshCensusSpliced(prev, oldMax, oldFrag, patch.start, len(patch.eng.ops))
	storeOpIDHighWater(prev, patch.eng.ops)
}

// commitTableRowPatch splices a rebuilt row into prev for the single element
// path and returns the row window on its own for inspection.
func commitTableRowPatch(prev *Result, patch *tableRowPatch) *Result {
	spliceTableRowPatch(prev, patch, false)

	holder := &box{ //nolint:exhaustruct // row-window result root; ranges rebased by the result below
		node: patch.trNode, style: patch.styles[patch.trNode], kind: boxKindBlock,
		x: patch.tableBox.x, y: patch.tableBox.rowTops[patch.rowIdx],
		w: patch.tableBox.w, height: patch.tableBox.rowHeights[patch.rowIdx],
		children: patch.freshCells, opStart: patch.start, opEnd: patch.start + len(patch.eng.ops) - 1,
	}

	return relayoutSubtreeResult(prev, holder, patch.start, len(patch.eng.ops), patch.styles)
}

// commitTableRowPatchBatch splices a rebuilt row into prev for one batch id.
// The batch owns its private clone, so the splice runs in place, and the
// owners fall back to the full derivation the table splice uses when the
// patched boxes sit under a table row. The derivation is deferred: it is a
// pure function of the committed placement, so consecutive row splices skip
// the page-sized derivation the next splice overwrites unread, and the batch
// derives once the owners are read or the batch ends.
func commitTableRowPatchBatch(
	prev *Result, patch *tableRowPatch, thread *boxThread,
	out *SubtreeBatchResult, relevant, grouped *int, ownersDirty *bool,
) {
	oldDamage, hasOld := spliceDamage(prev.Ops, patch.start, patch.oldCount)
	oldRel, oldGrp := countOrderingRelevant(prev.Ops[patch.start : patch.start+patch.oldCount])

	spliceTableRowPatch(prev, patch, true)

	out.PaintOnly = false

	newRel, newGrp := countOrderingRelevant(patch.eng.ops)
	*relevant += newRel - oldRel
	*grouped += newGrp - oldGrp
	*ownersDirty = true

	neutral := oldRel == 0 && newRel == 0 && *grouped == 0
	out.Order = splicePaintOrder(prev.Ops, out.Order, patch.start, patch.oldCount, len(patch.eng.ops), neutral)

	newDamage, hasNew := spliceDamage(prev.Ops, patch.start, len(patch.eng.ops))
	out.Windows = append(out.Windows, SpliceWindow{ //nolint:exhaustruct // damage flags default false
		Start:        patch.start,
		OldCount:     patch.oldCount,
		NewCount:     len(patch.eng.ops),
		OldDamage:    oldDamage,
		NewDamage:    newDamage,
		HasOldDamage: hasOld,
		HasNewDamage: hasNew,
	})

	for index := range patch.keptCells {
		thread.replace(patch.keptCells[index], patch.freshCells[index], patch.tableBox)
	}
}
