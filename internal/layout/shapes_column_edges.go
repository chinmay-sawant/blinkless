package layout

// columnRulePaintsEdges reports whether column-rule-visibility-items asks for
// the rule on the two content edges too. `around` and `all` do; `between` and
// `normal` (stored as "") paint the gaps only, as row-rule-visibility-items does.
func columnRulePaintsEdges(style *ResolvedStyle) bool {
	switch style.Declared.ColumnRuleVisibilityItems {
	case rowRuleVisibilityAround, rowRuleVisibilityAll:
		return true
	default:
		return false
	}
}

// emitColumnRuleEdges paints the column rule at the start and the end of the
// content box, from top to bottom. left and right are the two edge positions.
func (e *engine) emitColumnRuleEdges(style *ResolvedStyle, left, right, top, bottom, ruleW float64) {
	if !columnRulePaintsEdges(style) || bottom <= top {
		return
	}

	col := style.Color
	if style.ColumnRuleColorSet {
		col = style.ColumnRuleColor
	}

	for _, x := range [2]float64{left, right} {
		e.emitBorderLine(x, top, 0, bottom-top, ruleW, style.ColumnRuleStyle, col[0], col[1], col[2])
	}
}
