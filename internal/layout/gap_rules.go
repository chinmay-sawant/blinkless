package layout

// emitFlexRowGapRules paints one horizontal rule between each pair of
// wrapped flex lines when row-gap is positive. Tops and bottoms come from
// the children's final border boxes, read after align-content. That shift
// moves the boxes and does not update placed[i].y0. One line, a
// non-positive row gap, or a nil parent paints nothing. An empty child
// range skips that gap. The rule does not move children or add gap.
func (e *engine) emitFlexRowGapRules(
	parent *box, style ResolvedStyle, placed []flexLinePlace,
	rowGap, contentX, contentW float64,
) {
	if len(placed) <= 1 || rowGap <= 0 || parent == nil {
		return
	}

	for index := range len(placed) - 1 {
		_, bottom, okBottom := flexLineChildSpan(parent, placed[index])
		top, _, okTop := flexLineChildSpan(parent, placed[index+1])

		if !okBottom || !okTop {
			continue
		}

		e.emitRowGapRule(style, contentX, contentW, bottom, top-bottom)
	}

	if !rowRulePaintsAround(style) {
		return
	}

	_, _, okFirst := flexLineChildSpan(parent, placed[0])
	_, _, okLast := flexLineChildSpan(parent, placed[len(placed)-1])

	if !okFirst || !okLast {
		return
	}

	firstTop, _, _ := flexLineChildSpan(parent, placed[0])
	_, lastBottom, _ := flexLineChildSpan(parent, placed[len(placed)-1])

	e.emitRowRuleLine(style, contentX, contentW, firstTop)
	e.emitRowRuleLine(style, contentX, contentW, lastBottom)
}

// flexLineChildSpan reports one flex line's top and bottom from the
// children's final border boxes in [startChild, endChild). Top is the
// minimum y. Bottom is the maximum y+height. ok is false when that range
// is empty.
func flexLineChildSpan(parent *box, line flexLinePlace) (float64, float64, bool) {
	if parent == nil || line.startChild >= line.endChild || line.startChild >= len(parent.children) {
		return 0, 0, false
	}

	end := line.endChild
	if end > len(parent.children) {
		end = len(parent.children)
	}

	boxes := parent.children[line.startChild:end]
	if len(boxes) == 0 {
		return 0, 0, false
	}

	top := boxes[0].y
	bottom := boxes[0].y + boxes[0].height

	for _, child := range boxes[1:] {
		top = min(top, child.y)
		bottom = max(bottom, child.y+child.height)
	}

	return top, bottom, true
}

// emitRowGapRule paints one horizontal rule centered in a row gap.
// gapTop is the y of the top of the gap in paint space. gap is the gap size.
// contentX and contentW are the content box. A zero gap, a none style,
// or a zero width paints nothing. The stroke is the first stored item
// (RowRuleWidth / RowRuleStyle / RowRuleColor). Extra list items stay
// on the style and are not painted here.
func (e *engine) emitRowGapRule(style ResolvedStyle, contentX, contentW, gapTop, gap float64) {
	if e == nil || e.noEmit || gap <= 0 || contentW <= 0 {
		return
	}

	if style.RowRuleStyle == "" || style.RowRuleStyle == cssDisplayNone {
		return
	}

	e.emitRowRuleLine(style, contentX, contentW, gapTop+gap/2)
}

// emitRowRuleLine paints one horizontal rule at y. Cap insets shorten the
// ends. Junction insets are stored and not painted. none, intersection, and
// normal all draw the same line until a column rule crosses it.
func (e *engine) emitRowRuleLine(style ResolvedStyle, contentX, contentW, lineY float64) {
	if e == nil || e.noEmit || contentW <= 0 {
		return
	}

	if style.RowRuleStyle == "" || style.RowRuleStyle == cssDisplayNone {
		return
	}

	ruleW := e.scalePt(style.RowRuleWidth)
	if ruleW <= 0 {
		return
	}

	start := e.scalePt(style.RowRuleInsetCapStart)
	end := e.scalePt(style.RowRuleInsetCapEnd)
	span := contentW - start - end

	if span <= 0 {
		return
	}

	var col [3]float64
	if style.RowRuleColorSet {
		col = style.RowRuleColor
	} else {
		col = style.Color
	}

	e.emitBorderLine(contentX+start, lineY, span, 0, ruleW, style.RowRuleStyle, col[0], col[1], col[2])
}

const rowRuleVisibilityAround = "around"

const rowRuleVisibilityAll = "all"

const gapCenter = 0.5

// rowRulePaintsAround reports whether a rule is also drawn on the content edges.
func rowRulePaintsAround(style ResolvedStyle) bool {
	switch style.RowRuleVisibility {
	case rowRuleVisibilityAround, rowRuleVisibilityAll:
		return true
	default:
		return false
	}
}

// paintGridGapRules paints column lines from the column-rule fields and row
// lines from the row-rule fields. rule-overlap chooses which line is later
// in the display list. column-over-row paints the column line last.
func (e *engine) paintGridGapRules(
	style ResolvedStyle, contentX, contentW float64,
	cols []float64, columnGap float64,
	rowYs, rows []float64, posY float64,
) {
	if e == nil || len(rowYs) == 0 || len(rowYs) != len(rows) {
		return
	}

	paintColumnsFirst := style.RuleOverlap != "column-over-row"
	if paintColumnsFirst {
		e.paintGridColumnRules(style, contentX, cols, columnGap, rowYs, rows, posY)
	}

	e.paintGridRowRules(style, contentX, contentW, rowYs, rows, posY)

	if !paintColumnsFirst {
		e.paintGridColumnRules(style, contentX, cols, columnGap, rowYs, rows, posY)
	}
}

func (e *engine) paintGridRowRules(
	style ResolvedStyle, contentX, contentW float64, rowYs, rows []float64, posY float64,
) {
	if rowRulePaintsAround(style) {
		top := posY + rowYs[0]
		bottom := posY + rowYs[len(rows)-1] + rows[len(rows)-1]

		e.emitRowRuleLine(style, contentX, contentW, top)
		e.emitRowRuleLine(style, contentX, contentW, bottom)
	}

	for index := range len(rows) - 1 {
		gapTop := posY + rowYs[index] + rows[index]
		gap := posY + rowYs[index+1] - gapTop
		e.emitRowGapRule(style, contentX, contentW, gapTop, gap)
	}
}

func (e *engine) paintGridColumnRules(
	style ResolvedStyle, contentX float64, cols []float64, columnGap float64,
	rowYs, rows []float64, posY float64,
) {
	if style.ColumnRuleStyle == "" || style.ColumnRuleStyle == cssDisplayNone || columnGap <= 0 || len(cols) < 2 {
		return
	}

	ruleW := e.scalePt(style.ColumnRuleWidth)
	if ruleW <= 0 {
		return
	}

	var col [3]float64
	if style.ColumnRuleColorSet {
		col = style.ColumnRuleColor
	} else {
		col = style.Color
	}

	top := posY + rowYs[0]
	height := posY + rowYs[len(rows)-1] + rows[len(rows)-1] - top

	if height <= 0 {
		return
	}

	cursor := contentX

	for index := range len(cols) - 1 {
		cursor += cols[index]
		center := cursor + columnGap*gapCenter

		e.emitBorderLine(center, top, 0, height, ruleW, style.ColumnRuleStyle, col[0], col[1], col[2])

		cursor += columnGap
	}
}
