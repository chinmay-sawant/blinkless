package layout

// inline-sizing (CSS Inline 3) sets how tall an inline box's background is.
// normal keeps the font and line-height fill. stretch, after the line is
// placed, grows that fill so the margin edges meet the line box. The text
// baseline does not move. A token outside the grammar leaves the previous
// value. The property inherits. It does not apply to a ruby annotation.

const (
	inlineSizingProp    = "inline-sizing"
	inlineSizingStretch = "stretch"
)

// applyInlineSizingProps owns inline-sizing. Registered in styleGroups.
// A bad token leaves the stored value alone.
func applyInlineSizingProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext,
	parent *ResolvedStyle, hasParent bool,
) bool {
	if prop != inlineSizingProp {
		return false
	}

	if used, ok := resolveInlineSizing(value, parent, hasParent); ok {
		style.InlineSizing = used
	}

	return true
}

// inlineSizingValueAccepted is the cascade gate for inline-sizing. CSS-wide
// keywords are accepted before this runs. Anything else must be normal or stretch.
func inlineSizingValueAccepted(value string) bool {
	switch normalizeCSSValue(value) {
	case contentNormal, inlineSizingStretch:
		return true
	default:
		return false
	}
}

// resolveInlineSizing returns the used keyword. ok is false when value is
// outside the grammar, so the caller keeps the previous value.
func resolveInlineSizing(value string, parent *ResolvedStyle, hasParent bool) (string, bool) {
	parentValue := contentNormal
	if hasParent && parent != nil && parent.InlineSizing != "" {
		parentValue = parent.InlineSizing
	}

	if used, ok := cssWideValue(value, parentValue, contentNormal, hasParent); ok {
		return used, true
	}

	switch normalized := normalizeCSSValue(value); normalized {
	case contentNormal, inlineSizingStretch:
		return normalized, true
	case stringSetRevertLayer:
		return contentNormal, true
	default:
		return "", false
	}
}

// inlineSizingFill is the span background rectangle. normal is the font
// ascent, descent, half the line-height gap, and padding. stretch moves the
// top and bottom to the line box, inset by vertical margins. Ruby
// annotations keep the normal rectangle.
func (e *engine) inlineSizingFill(
	item *inlineItem, baseline, ascent, descent, top, bottom float64,
) (float64, float64) {
	style := item.style
	spanLH := e.lineHeightOf(style) * e.scale
	extra := (spanLH - ascent - descent) / two
	boxY := baseline - ascent - extra - top
	boxH := ascent + extra + top + descent + extra + bottom

	if !inlineSizingStretches(item) {
		return boxY, boxH
	}

	marginT := e.scalePt(style.MarginTop)
	marginB := e.scalePt(style.MarginBottom)
	stretchedH := item.lineBoxH - marginT - marginB

	if stretchedH <= 0 {
		return boxY, boxH
	}

	return item.lineTop + marginT, stretchedH
}

// inlineSizingStretches reports whether this item's fill grows to the line box.
func inlineSizingStretches(item *inlineItem) bool {
	if item == nil || item.style == nil || item.rubyAnnot || item.lineBoxH <= 0 {
		return false
	}

	return item.style.InlineSizing == inlineSizingStretch
}
