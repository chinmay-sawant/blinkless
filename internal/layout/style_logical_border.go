package layout

// applyLogicalRadiusLonghand maps logical corners and sides to their
// physical fields in the engine's fixed left-to-right corner order.
func applyLogicalRadiusLonghand(style *ResolvedStyle, prop, value string, fsize float64) bool {
	var first, second string
	switch prop {
	case "border-start-start-radius":
		first = cornerNameTopLeft
	case "border-start-end-radius":
		first = cornerNameTopRight
	case "border-end-start-radius":
		first = cornerNameBottomLeft
	case "border-end-end-radius":
		first = cornerNameBottomRight
	case "border-block-start-radius":
		first, second = cornerNameTopLeft, cornerNameTopRight
	case "border-block-end-radius":
		first, second = cornerNameBottomLeft, cornerNameBottomRight
	case "border-inline-start-radius":
		first, second = cornerNameTopLeft, cornerNameBottomLeft
	case "border-inline-end-radius":
		first, second = cornerNameTopRight, cornerNameBottomRight
	default:
		return false
	}

	setCornerRadii(style.editBorders(), first, second, value, fsize)

	return true
}
