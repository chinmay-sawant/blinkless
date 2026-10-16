package layout

// expandShorthandDeclaration decodes the winning side of one shorthand into
// the longhand declarations that carry its origin and specificity. Each
// expansion returns false for a prop it does not own or a value it cannot
// read, so the raw key keeps the post-cascade fallback path.
func expandShorthandDeclaration(prop, value string) ([]logicalPropDecl, bool) {
	if expanded, ok := expandPositionTryDeclaration(prop, value); ok {
		return expanded, true
	}

	if expanded, ok := expandFontDeclaration(prop, value); ok {
		return expanded, true
	}

	if expanded, ok := expandMaskShorthandDeclaration(prop, value); ok {
		return expanded, true
	}

	if expanded, ok := expandListStyleDeclaration(prop, value); ok {
		return expanded, true
	}

	if expanded, ok := expandBackgroundDeclaration(prop, value); ok {
		return expanded, true
	}

	return expandRowAndGapRules(prop, value)
}
