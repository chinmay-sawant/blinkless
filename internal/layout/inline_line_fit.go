package layout

const (
	lineFitEdgeLeading = "leading"
	lineFitEdgeText    = "text"
)

// applyLineFitEdgeProps owns line-fit-edge. leading is the initial. text is
// the only other stored keyword. A token outside that pair leaves the
// previous value. The property inherits through inheritGapBProps because the
// inherit mask is full.
func applyLineFitEdgeProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext,
	parent *ResolvedStyle, hasParent bool,
) bool {
	if prop != "line-fit-edge" {
		return false
	}

	parentValue := lineFitEdgeLeading
	if parent != nil && parent.inlineText().LineFitEdge != "" {
		parentValue = parent.inlineText().LineFitEdge
	}

	next, ok := inheritedToken(
		value, parentValue, lineFitEdgeLeading, hasParent, normalizeLineFitEdge,
	)
	if ok {
		style.editInlineText().LineFitEdge = next
	}

	return true
}

func normalizeLineFitEdge(value string) (string, bool) {
	switch normalizeCSSValue(value) {
	case lineFitEdgeLeading:
		return lineFitEdgeLeading, true
	case lineFitEdgeText:
		return lineFitEdgeText, true
	default:
		return "", false
	}
}

// lineFitEdgeContribution is the span's padding and border added to the line
// box. leading (and any non-text value) keeps that chrome on top of the
// half-leading already in ascent and descent. text leaves the chrome out.
func (e *engine) lineFitEdgeContribution(item *inlineItem, ascent, descent float64) (float64, float64) {
	if lineFitEdgeDropsChrome(item) {
		return ascent, descent
	}

	top := e.inlineChromeTop(item.style)
	bottom := e.inlineChromeBottom(item.style)

	return ascent + top, descent + bottom
}

// lineFitEdgeDropsChrome reports that this item must not grow the line box
// by its padding or border. Missing chrome keeps the old zero add. text
// drops chrome that is present.
func lineFitEdgeDropsChrome(item *inlineItem) bool {
	if item == nil || item.style == nil || !item.chrome {
		return true
	}

	return item.style.inlineText().LineFitEdge == lineFitEdgeText
}
