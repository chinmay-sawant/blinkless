package layout

// remapVendorDeclaration maps a vendor-prefixed property to its standard name
// and remaps the value under the prefixed name. Other declarations pass through.
func remapVendorDeclaration(prop, value string) (string, string) {
	if effectiveProp := normalizeVendorPrefix(prop); effectiveProp != prop {
		return effectiveProp, remapWebkitValue(prop, value)
	}

	return prop, value
}

// inheritGapBPropsWide is the second half of inheritGapBProps: the properties
// that sit outside the 64-entry inheritable mask.
func inheritGapBPropsWide(dst *ResolvedStyle, parent *ResolvedStyle, raw map[string]string) {
	// line-fit-edge inherits. The 64-entry mask is full, so it cannot join
	// inheritableProps; a declared value is applied later and wins.
	if _, ok := raw["line-fit-edge"]; !ok {
		dst.editInlineText().LineFitEdge = parent.inlineText().LineFitEdge
	}

	// line-padding inherits, but the uint64 inherit mask is full.
	if _, ok := raw["line-padding"]; !ok {
		dst.editInlineText().LinePadding = parent.inlineText().LinePadding
	}

	if _, ok := raw[inlineSizingProp]; !ok {
		dst.editInlineText().InlineSizing = parent.inlineText().InlineSizing
	}

	// word-space-transform inherits, but the uint64 inherit mask is full.
	if _, ok := raw["word-space-transform"]; !ok {
		dst.editInlineText().WordSpaceTransform = parent.inlineText().WordSpaceTransform
	}

	// paint-order inherits per SVG 2 §13.6; the uint64 inherit mask is full.
	if _, ok := raw["paint-order"]; !ok {
		dst.editSvg().SVGPaintOrder = parent.svg().SVGPaintOrder
	}
}
