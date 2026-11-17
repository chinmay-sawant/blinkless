package layout

import "strings"

// expandBackgroundDeclaration expands the background shorthand into the color
// and image longhands the painter reads, so the shorthand and a longhand
// compete per property with each declaration's own origin, specificity, and
// source order. Without this, the UA background-color on a button, select, or
// textarea coexists with an author background shorthand as a separate raw key,
// and the later-applied longhand wins regardless of origin.
func expandBackgroundDeclaration(prop, value string) ([]logicalPropDecl, bool) {
	if prop == declPropBackgroundTbd {
		return expandBackgroundTbd(value)
	}

	if prop != "background" {
		return nil, false
	}

	var out []logicalPropDecl

	if tok, ok := firstBackgroundColorToken(value); ok {
		out = append(out, logicalPropDecl{prop: "background-color", val: tok})
	}

	if hasBackgroundImage(value) {
		out = append(out, logicalPropDecl{prop: "background-image", val: value})
	}

	if len(out) == 0 {
		return nil, false
	}

	return out, true
}

// backgroundTbdHasColor reports whether a background-tbd value names a color.
// The none keyword is an image value and never counts as one.
func backgroundTbdHasColor(value string) bool {
	if strings.EqualFold(strings.TrimSpace(value), cssDisplayNone) {
		return false
	}

	_, found := firstBackgroundColorToken(value)

	return found
}

// expandBackgroundTbd expands the background-tbd layer list of Backgrounds 4
// into the image, attachment, and repeat longhands. It is the background
// shorthand without a color, so a color token anywhere drops the whole
// declaration and an earlier background keeps both its image and its color.
func expandBackgroundTbd(value string) ([]logicalPropDecl, bool) {
	if backgroundTbdHasColor(value) || !hasBackgroundImage(value) {
		return nil, false
	}

	attach := make([]string, 0, len(splitCommaLayers(value)))
	repeat := ""

	for _, layer := range splitCommaLayers(value) {
		kind := "scroll"

		for _, token := range strings.Fields(strings.ToLower(layer)) {
			switch token {
			case backgroundAttachFixed, "local", "scroll":
				kind = token
			case maskRepeatKeywordValue, "no-repeat", "repeat-x", "repeat-y", "space", "round":
				if repeat == "" {
					repeat = token
				}
			}
		}

		attach = append(attach, kind)
	}

	out := []logicalPropDecl{
		{prop: "background-image", val: value},
		{prop: "background-attachment", val: strings.Join(attach, ", ")},
	}

	if repeat != "" {
		out = append(out, logicalPropDecl{prop: "background-repeat", val: repeat})
	}

	return out, true
}

// hasBackgroundImage reports whether a background shorthand carries an image
// the painter can resolve: a url(), a gradient, or none.
func hasBackgroundImage(value string) bool {
	if _, ok := firstCSSUrl(value); ok {
		return true
	}

	trimmed := strings.TrimSpace(value)

	return isGradientFunc(trimmed) || strings.EqualFold(trimmed, "none")
}
