package layout

import "strings"

// applyRowRuleGroup stores row-rule, row-rule-width, row-rule-style, and
// row-rule-color. Other row-rule longhands and every rule* shorthand are
// left untouched. A value that does not parse returns true and keeps the
// fields already stored.
//
// ctx, parent, and hasParent match styleGroupFn. This group does not read
// them.
//
//nolint:cyclop,goconst // one switch owns the row-rule longhands
func applyRowRuleGroup(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext,
	parent *ResolvedStyle, hasParent bool,
) bool {
	if expanded, ok := expandGapRuleDeclaration(prop, value); ok {
		for _, item := range expanded {
			if !applyRowRuleGroup(style, item.prop, item.val, fsize, ctx, parent, hasParent) {
				applyColumnRuleProps(style, item.prop, item.val, fsize)
			}
		}

		return true
	}

	switch prop {
	case "row-rule":
		applyRowRuleShorthand(style, value, fsize)
	case "row-rule-width":
		applyRowRuleWidths(style, value, fsize)
	case "row-rule-style":
		applyRowRuleStyles(style, value)
	case "row-rule-color":
		applyRowRuleColors(style, value)
	case "row-rule-break":
		applyRowRuleKeyword(style, value, &style.RowRuleBreak, cssDisplayNone, contentNormal, "intersection")
	case "row-rule-visibility-items":
		applyRowRuleKeyword(style, value, &style.RowRuleVisibility, "all", "around", "between", contentNormal)
	case "row-rule-inset-cap-start":
		applyRowRuleInsetSide(style, value, fsize, &style.RowRuleInsetCapStart)
	case "row-rule-inset-cap-end":
		applyRowRuleInsetSide(style, value, fsize, &style.RowRuleInsetCapEnd)
	case "row-rule-inset-junction-start":
		applyRowRuleInsetSide(style, value, fsize, &style.RowRuleInsetJunctionStart)
	case "row-rule-inset-junction-end":
		applyRowRuleInsetSide(style, value, fsize, &style.RowRuleInsetJunctionEnd)
	case "rule-overlap":
		applyRowRuleKeyword(style, value, &style.RuleOverlap, "row-over-column", "column-over-row")
	default:
		return false
	}

	return true
}

// expandRowRuleDeclaration turns row-rule into the three longhands so a later
// shorthand competes with an earlier longhand by source order. Omitted pieces
// are the same initials parseRuleShorthand writes: medium, none, and currentColor.
func expandRowRuleDeclaration(prop, value string) ([]logicalPropDecl, bool) {
	if prop != "row-rule" || !outlineShorthandValueAccepted(value) {
		return nil, false
	}

	width := mediumKeyword
	ruleStyle := cssDisplayNone
	color := "currentColor"

	for start := 0; ; {
		token, next, ok := nextSpaceToken(value, start)
		if !ok {
			break
		}

		if _, parsed := parseOutlineStyle(token); parsed {
			ruleStyle = token
		} else if _, parsed := parseOutlineWidth(token, 0); parsed {
			width = token
		} else if _, parsed := parseUsedColor(token, [3]float64{}); parsed {
			color = token
		}

		start = next
	}

	return []logicalPropDecl{
		{prop: "row-rule-width", val: width},
		{prop: "row-rule-style", val: ruleStyle},
		{prop: "row-rule-color", val: color},
	}, true
}

func expandRowAndGapRules(prop, value string) ([]logicalPropDecl, bool) {
	if expanded, ok := expandRowRuleDeclaration(prop, value); ok {
		return expanded, true
	}

	if expanded, ok := expandGapRuleDeclaration(prop, value); ok {
		return expanded, true
	}

	return expandLogicalBoxDeclaration(prop, value)
}

// expandGapRuleDeclaration expands the row-rule inset shorthands and the
// rule* names that set both axes or only the row half. A value that does
// not parse returns false so the cascade keeps the earlier declaration.
//
//nolint:cyclop // each shorthand is one case
func expandGapRuleDeclaration(prop, value string) ([]logicalPropDecl, bool) {
	switch prop {
	case "rule":
		if !outlineShorthandValueAccepted(value) {
			return nil, false
		}

		return []logicalPropDecl{
			{prop: "column-rule", val: value},
			{prop: "row-rule", val: value},
		}, true
	case "rule-width":
		return duplicateGapLonghand(value, rowRuleWidthValueAccepted, "column-rule-width", "row-rule-width")
	case "rule-style":
		return duplicateGapLonghand(value, rowRuleStyleValueAccepted, "column-rule-style", "row-rule-style")
	case "rule-color":
		return duplicateGapLonghand(value, rowRuleColorValueAccepted, "column-rule-color", "row-rule-color")
	case "rule-break":
		return aliasGapLonghand(value, rowRuleBreakAccepted, "row-rule-break")
	case "rule-visibility-items":
		return aliasGapLonghand(value, rowRuleVisibilityAccepted, "row-rule-visibility-items")
	case "row-rule-inset-start", "rule-inset-start":
		return aliasGapLonghand(value, rowRuleInsetTokenAccepted, "row-rule-inset-cap-start")
	case "row-rule-inset-end", "rule-inset-end":
		return aliasGapLonghand(value, rowRuleInsetTokenAccepted, "row-rule-inset-cap-end")
	case "row-rule-inset-cap", "rule-inset-cap":
		return expandInsetPair(value, "row-rule-inset-cap-start", "row-rule-inset-cap-end")
	case "row-rule-inset-junction", "rule-inset-junction":
		return expandInsetPair(value, "row-rule-inset-junction-start", "row-rule-inset-junction-end")
	case "row-rule-inset", "rule-inset":
		return expandRowRuleInset(value)
	default:
		return nil, false
	}
}

func duplicateGapLonghand(
	value string, accept func(string) bool, columnProp, rowProp string,
) ([]logicalPropDecl, bool) {
	if !accept(value) {
		return nil, false
	}

	return []logicalPropDecl{
		{prop: columnProp, val: value},
		{prop: rowProp, val: value},
	}, true
}

func aliasGapLonghand(value string, accept func(string) bool, prop string) ([]logicalPropDecl, bool) {
	if !accept(value) {
		return nil, false
	}

	return []logicalPropDecl{{prop: prop, val: strings.TrimSpace(value)}}, true
}

func expandInsetPair(value, startProp, endProp string) ([]logicalPropDecl, bool) {
	fields := strings.Fields(value)
	if len(fields) == 0 || len(fields) > 2 {
		return nil, false
	}

	if !rowRuleInsetTokenAccepted(fields[0]) {
		return nil, false
	}

	end := fields[0]

	if len(fields) == maxTwoValueArgs {
		if !rowRuleInsetTokenAccepted(fields[1]) {
			return nil, false
		}

		end = fields[1]
	}

	return []logicalPropDecl{
		{prop: startProp, val: fields[0]},
		{prop: endProp, val: end},
	}, true
}

func expandRowRuleInset(value string) ([]logicalPropDecl, bool) {
	capPart, junctionPart, hasJunction := strings.Cut(value, "/")
	insetCap, parsed := expandInsetPair(capPart, "row-rule-inset-cap-start", "row-rule-inset-cap-end")

	if !parsed {
		return nil, false
	}

	junctionValue := "0"
	if hasJunction {
		junctionValue = junctionPart
	}

	junction, junctionOK := expandInsetPair(
		junctionValue, "row-rule-inset-junction-start", "row-rule-inset-junction-end",
	)

	if !junctionOK {
		return nil, false
	}

	return append(insetCap, junction...), true
}

func rowRuleWidthValueAccepted(value string) bool {
	return rowRuleListValueAccepted(value, rowRuleWidthTokenAccepted)
}

func rowRuleStyleValueAccepted(value string) bool {
	return rowRuleListValueAccepted(value, rowRuleStyleTokenAccepted)
}

func rowRuleColorValueAccepted(value string) bool {
	return rowRuleListValueAccepted(value, rowRuleColorTokenAccepted)
}

func rowRuleBreakAccepted(value string) bool {
	return rowRuleKeywordList(value, cssDisplayNone, contentNormal, "intersection")
}

func rowRuleVisibilityAccepted(value string) bool {
	return rowRuleKeywordList(value, "all", "around", "between", contentNormal)
}

func rowRuleInsetTokenAccepted(token string) bool {
	_, ok := parseRowRuleInset(token, 0)

	return ok
}

func rowRuleKeywordList(value string, keywords ...string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return false
	}

	for _, keyword := range keywords {
		if trimmed == keyword {
			return true
		}
	}

	return false
}

func applyRowRuleKeyword(style *ResolvedStyle, value string, dest *string, keywords ...string) {
	if !rowRuleKeywordList(value, keywords...) {
		return
	}

	*dest = strings.ToLower(strings.TrimSpace(value))
	_ = style
}

func applyRowRuleInsetSide(style *ResolvedStyle, value string, fsize float64, dest *float64) {
	length, ok := parseRowRuleInset(value, fsize)
	if !ok {
		return
	}

	*dest = length
	_ = style
}

func parseRowRuleInset(value string, fsize float64) (float64, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "0" {
		return 0, true
	}

	length, ok := lengthBox(trimmed, fsize, 0, overflowAuto)
	if !ok || length < 0 {
		return 0, false
	}

	return length, true
}

// applyRowRuleShorthand stores one width, one style, and the shorthand color.
// RowRuleColors holds that color token when one parsed. When the shorthand
// omits a color, RowRuleColors stays nil, RowRuleColorSet is still true, and
// RowRuleColor is the current color (parseRuleShorthand's omitted-color result).
func applyRowRuleShorthand(style *ResolvedStyle, value string, fsize float64) {
	width, ruleStyle, color, ok := parseRuleShorthand(value, fsize, style.Color)
	if !ok {
		return
	}

	style.RowRuleWidth = width
	style.RowRuleStyle = ruleStyle
	style.RowRuleColor = color
	style.RowRuleColorSet = true
	style.RowRuleWidths = []float64{width}
	style.RowRuleStyles = []string{ruleStyle}
	style.RowRuleColors = rowRuleShorthandColors(value, fsize, style.Color)
}

func applyRowRuleWidths(style *ResolvedStyle, value string, fsize float64) {
	var widths []float64

	tokensOK := rowRuleTokensAccepted(value, func(token string) bool {
		width, parsed := parseRowRuleWidthToken(token, fsize)
		if !parsed {
			return false
		}

		widths = append(widths, width)

		return true
	})
	if !tokensOK {
		return
	}

	style.RowRuleWidths = widths
	style.RowRuleWidth = widths[0]
}

func applyRowRuleStyles(style *ResolvedStyle, value string) {
	var ruleStyles []string

	tokensOK := rowRuleTokensAccepted(value, func(token string) bool {
		ruleStyle, parsed := parseRowRuleStyleToken(token)
		if !parsed {
			return false
		}

		ruleStyles = append(ruleStyles, ruleStyle)

		return true
	})
	if !tokensOK {
		return
	}

	style.RowRuleStyles = ruleStyles
	style.RowRuleStyle = ruleStyles[0]
}

func applyRowRuleColors(style *ResolvedStyle, value string) {
	var colors []string

	var first [3]float64

	tokensOK := rowRuleTokensAccepted(value, func(token string) bool {
		color, raw, parsed := parseRowRuleColorToken(token, style.Color)
		if !parsed {
			return false
		}

		if len(colors) == 0 {
			first = color
		}

		colors = append(colors, raw)

		return true
	})
	if !tokensOK {
		return
	}

	style.RowRuleColor = first
	style.RowRuleColorSet = true
	style.RowRuleColors = colors
}

// rowRuleTokensAccepted reports whether every space-separated token is
// accepted. Parentheses stay inside one token. An empty value is rejected
// so the caller does not replace an earlier list.
func rowRuleTokensAccepted(value string, accept func(token string) bool) bool {
	accepted := false

	for start := 0; ; {
		token, next, ok := nextSpaceToken(value, start)
		if !ok {
			break
		}

		if !accept(token) {
			return false
		}

		accepted = true
		start = next
	}

	return accepted
}

// parseRowRuleWidthToken accepts the outline widths plus auto. auto is the
// initial medium width; parseOutlineWidth does not accept it.
func parseRowRuleWidthToken(token string, fsize float64) (float64, bool) {
	if strings.EqualFold(token, overflowAuto) {
		return borderWidth(mediumKeyword, fsize), true
	}

	return parseOutlineWidth(token, fsize)
}

// parseRowRuleStyleToken accepts outline line styles plus auto. auto is the
// initial line style, stored as none.
func parseRowRuleStyleToken(token string) (string, bool) {
	if strings.EqualFold(token, overflowAuto) {
		return cssDisplayNone, true
	}

	return parseOutlineStyle(token)
}

// parseRowRuleColorToken accepts a used color plus auto. auto means
// currentColor and is stored as the token "auto", not the used color.
func parseRowRuleColorToken(token string, current [3]float64) ([3]float64, string, bool) {
	if strings.EqualFold(token, overflowAuto) {
		return current, overflowAuto, true
	}

	color, parsed := parseUsedColor(token, current)
	if !parsed {
		return [3]float64{}, "", false
	}

	return color, token, true
}

// rowRuleShorthandColors is the color token parseRuleShorthand kept, or nil
// when no token parsed as a color. Style and width tokens are not colors.
func rowRuleShorthandColors(value string, fsize float64, current [3]float64) []string {
	var found string

	for start := 0; ; {
		token, next, ok := nextSpaceToken(value, start)
		if !ok {
			break
		}

		found = rowRuleKeptColorToken(found, token, fsize, current)
		start = next
	}

	if found == "" {
		return nil
	}

	return []string{found}
}

func rowRuleKeptColorToken(previous, token string, fsize float64, current [3]float64) string {
	if _, parsed := parseOutlineStyle(token); parsed {
		return previous
	}

	if _, parsed := parseOutlineWidth(token, fsize); parsed {
		return previous
	}

	if _, parsed := parseUsedColor(token, current); parsed {
		return token
	}

	return previous
}
