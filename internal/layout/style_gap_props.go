package layout

// applyGapProps owns gap, row-gap, column-gap, and the legacy grid-* aliases.
func applyGapProps(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	switch prop {
	case gapKeyword, "grid-gap":
		return applyGapShorthand(style, value, fsize, ctx.viewportW)
	case propRowGap, "grid-row-gap":
		return applyRowGap(style, value, fsize, ctx.viewportW)
	case propColumnGap, "grid-column-gap":
		return applyColumnGap(style, value, fsize, ctx.viewportW)
	default:
		return false
	}
}

func applyGapShorthand(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	if value == contentNormal {
		style.editFlex().Gap = 0
		style.editFlex().RowGap = 0
		style.editFlex().ColumnGap = 0
		style.editFlex().ColumnGapNormal = true
	} else if v, ok := lengthBox(value, fsize, viewportW, cssDisplayNone); ok && v >= 0 {
		style.editFlex().Gap = v
		style.editFlex().RowGap = v
		style.editFlex().ColumnGap = v
		style.editFlex().ColumnGapNormal = false
	}

	return true
}

func applyRowGap(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	if v, ok := lengthBox(value, fsize, viewportW, cssDisplayNone); ok && v >= 0 {
		style.editFlex().RowGap = v
		style.editFlex().Gap = v
	}

	return true
}

func applyColumnGap(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	if value == contentNormal {
		style.editFlex().ColumnGap = 0
		style.editFlex().ColumnGapNormal = true
	} else if v, ok := lengthBox(value, fsize, viewportW, cssDisplayNone); ok && v >= 0 {
		style.editFlex().ColumnGap = v
		style.editFlex().ColumnGapNormal = false
	}

	return true
}
