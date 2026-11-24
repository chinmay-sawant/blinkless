package layout

import "strings"

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

func parseGapComponent(tok string, fsize, viewportW float64) (float64, bool, bool) {
	if tok == contentNormal {
		return 0, true, true
	}

	if v, ok := lengthBox(tok, fsize, viewportW, cssDisplayNone); ok && v >= 0 {
		return v, false, true
	}

	return 0, false, false
}

func applyGapShorthand(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	parts := strings.Fields(strings.TrimSpace(value))
	switch len(parts) {
	case 1:
		v, isNormal, ok := parseGapComponent(parts[0], fsize, viewportW)
		if !ok {
			return true
		}

		style.editFlex().Gap = v
		style.editFlex().RowGap = v
		style.editFlex().ColumnGap = v
		style.editFlex().ColumnGapNormal = isNormal
	case maxTwoValueArgs:
		rowV, _, rowOK := parseGapComponent(parts[0], fsize, viewportW)
		colV, colNormal, colOK := parseGapComponent(parts[1], fsize, viewportW)
		if !rowOK || !colOK {
			return true
		}

		style.editFlex().Gap = rowV
		style.editFlex().RowGap = rowV
		style.editFlex().ColumnGap = colV
		style.editFlex().ColumnGapNormal = colNormal
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
