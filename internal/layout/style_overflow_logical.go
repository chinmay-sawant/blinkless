package layout

// applyOverflowLogicalProps maps overflow-block / overflow-inline onto the
// physical OverflowX / OverflowY axes using writing-mode. Horizontal-tb maps
// block→Y and inline→X; vertical-* writing modes swap the axes. No new
// ResolvedStyle fields: clip already reads OverflowX/OverflowY.
func applyOverflowLogicalProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	switch prop {
	case "overflow-block", "overflow-inline":
		setOverflowLogical(style, prop, value)
	default:
		return false
	}

	return true
}

func setOverflowLogical(style *ResolvedStyle, prop, value string) {
	overflow, ok := parseOverflowKeyword(value)
	if !ok {
		return
	}

	blockIsY := !isVerticalWritingMode(style.positioning().WritingMode)
	toY := (prop == "overflow-block" && blockIsY) || (prop == "overflow-inline" && !blockIsY)

	if toY {
		style.editGeometry().OverflowY = overflow
	} else {
		style.editGeometry().OverflowX = overflow
	}

	if overflow != visibleKeyword {
		style.Overflow = overflow
	}
}
