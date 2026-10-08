package layout

import (
	"strconv"
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// Property-name and keyword constants used by the acceptance table.
const (
	propDisplay   = "display"
	propPosition  = "position"
	propOverflow  = "overflow"
	propOverflowX = "overflow-x"
	propOverflowY = "overflow-y"
	propHeight    = "height"
	propWidth     = "width"
	propMinWidth  = "min-width"
	propMinHeight = "min-height"
	propMaxWidth  = "max-width"
	propMaxHeight = "max-height"

	propMarginTop    = "margin-top"
	propMarginRight  = "margin-right"
	propMarginBottom = "margin-bottom"
	propMarginLeft   = "margin-left"

	propPaddingTop    = "padding-top"
	propPaddingRight  = "padding-right"
	propPaddingBottom = "padding-bottom"
	propPaddingLeft   = "padding-left"

	sizeMaxContent = "max-content"
	sizeMinContent = "min-content"

	fontWeightBolder  = "bolder"
	fontWeightLighter = "lighter"
)

// supportedDeclaration reports whether the engine accepts one declaration as
// applicable. It is the single value-acceptance rule shared by the cascade
// gate in cascadeRaw and the @supports probe in engineSupportsProperty, so an
// invalid declaration neither outranks a valid earlier one nor satisfies a
// feature query.
//
// Three value classes pass with no property check:
//
//   - custom properties (--*), whose values resolve in mergeCustomProps;
//   - CSS-wide keywords (inherit, initial, unset, revert, revert-layer), which
//     every property accepts;
//   - values containing var(), which are valid at parse time and checked
//     again after substitution by the applier. A computed-value failure makes
//     the property unset; it must not revive an earlier declaration.
func supportedDeclaration(prop, value string) bool {
	if strings.HasPrefix(prop, "--") {
		return true
	}

	trimmed := strings.TrimSpace(value)
	if cssWideKeyword(trimmed) || containsVarFunc(trimmed) {
		return true
	}

	effectiveProp := normalizeVendorPrefix(prop)
	if effectiveProp != prop {
		trimmed = strings.TrimSpace(remapWebkitValue(prop, value))
	}

	return declarationValueAccepted(effectiveProp, trimmed)
}

// cssWideKeyword reports whether value is a CSS-wide keyword. The style
// pipeline decides what each resolves to; acceptance is universal.
func cssWideKeyword(value string) bool {
	switch value {
	case inheritKeyword, cssKeywordInitial, cssKeywordUnset, cssKeywordRevert, "revert-layer":
		return true
	default:
		return false
	}
}

// declarationValueAccepted mirrors the value parser behind each property's
// apply arm. Handler ownership alone is not support: display:bogus reaches
// setDisplayKeyword, which ignores it, so the declaration must not win the
// cascade or satisfy @supports.
//
// A property without an entry is accepted. Its applier either takes the value
// as written (transforms, images, strings) or has a grammar this table does
// not model yet; rejecting a valid value would break layout, while the
// applier already drops an invalid one.
//
//nolint:cyclop,funlen // flat per-property acceptance table reads clearer than a split dispatch
func declarationValueAccepted(prop, value string) bool {
	switch prop {
	case propDisplay:
		return displayValueAccepted(value)
	case propPosition:
		return keywordIn(value, positionStatic, positionRelative, positionAbsolute, positionFixed, positionSticky)
	case "float":
		return keywordIn(value, floatLeft, floatRight, cssDisplayNone)
	case clearKeyword:
		return keywordIn(value, floatLeft, floatRight, clearBoth, cssDisplayNone)
	case "box-sizing":
		return keywordIn(value, "content-box", borderBox)
	case "writing-mode":
		return keywordIn(value, writingModeHorizontalTB, writingModeVerticalRL, writingModeVerticalLR)
	case "direction":
		return keywordIn(value, cssDirectionLTR, cssDirectionRTL)
	case propOverflow, propOverflowX, propOverflowY:
		_, ok := parseOverflowKeyword(value)

		return ok
	case "visibility":
		return keywordIn(value, visibleKeyword, overflowHidden, borderCollapseValue)
	case propWidth:
		return widthValueAccepted(value)
	case propHeight:
		return heightValueAccepted(value)
	case propMinWidth, propMinHeight:
		return minExtentValueAccepted(value)
	case propMaxWidth, propMaxHeight:
		return boxLengthValueAccepted(value, cssDisplayNone)
	case marginProperty:
		return boxShorthandValueAccepted(value, marginLengthValueAccepted)
	case propMarginTop, propMarginRight, propMarginBottom, propMarginLeft,
		cssVerticalAlignTop, floatRight, cssVerticalAlignBottom, floatLeft:
		// The inset offsets share the margin value grammar.
		return marginLengthValueAccepted(value)
	case paddingProperty:
		return boxShorthandValueAccepted(value, paddingLengthValueAccepted)
	case propPaddingTop, propPaddingRight, propPaddingBottom, propPaddingLeft:
		return paddingLengthValueAccepted(value)
	case "font-style":
		return keywordIn(value, contentNormal, cssFontStyleItalic, cssFontStyleOblique)
	case "font-weight":
		return fontWeightValueAccepted(value)
	case "text-transform":
		return keywordIn(value, textTransformNone, textTransformUppercase, textTransformLowercase, textTransformCapitalize)
	case "text-align":
		return keywordIn(value, floatLeft, fxCenter, cssTextAlignJustify, floatRight, fxEnd, fxStart)
	case "white-space":
		return keywordIn(value, contentNormal, cssWhiteSpaceNowrap, cssWhiteSpacePre,
			cssWhiteSpacePreWrap, cssWhiteSpacePreLine)
	case "vertical-align":
		return verticalAlignValueAccepted(value)
	case "overflow-wrap", "word-wrap":
		return keywordIn(value, contentNormal, overflowWrapBreakWord, overflowWrapAnywhere, "break-spaces")
	case "word-break":
		return keywordIn(value, contentNormal, "break-all", "keep-all", overflowWrapBreakWord)
	case "opacity":
		_, ok := parseOpacityValue(value)

		return ok
	case "z-index":
		return zIndexValueAccepted(value)
	default:
		return true
	}
}

// displayValueAccepted is setDisplayKeyword's keyword set, shared so the
// setter and the acceptance predicate cannot drift.
func displayValueAccepted(value string) bool {
	switch value {
	case displayBlock, "inline", cssDisplayNone, displayListItem, displayTable, displayTableRow, displayTableCell,
		displayRowGroup, displayHeaderGroup, displayFooterGroup,
		cssDisplayInlineBlock, displayTableCaption, "table-column", "table-column-group",
		displayFlex, displayInlineFlex, displayGrid, displayInlineGrid, displaySubgrid, displayFlowRoot,
		"-webkit-box", "-webkit-inline-box":
		return true
	default:
		return false
	}
}

// widthValueAccepted mirrors setWidthValue, which also takes max-content,
// min-content, and the two-token calc form.
func widthValueAccepted(value string) bool {
	switch value {
	case sizeMaxContent, sizeMinContent, overflowAuto:
		return true
	}

	if _, _, ok := calcFlexWidth(value, 0); ok {
		return true
	}

	return boxLengthValueAccepted(value, overflowAuto)
}

// heightValueAccepted mirrors setHeightValue, which takes neither the
// intrinsic keywords nor the two-token calc form.
func heightValueAccepted(value string) bool {
	if value == overflowAuto {
		return true
	}

	return boxLengthValueAccepted(value, overflowAuto)
}

// minExtentValueAccepted mirrors setMinWidthValue and setMinHeightValue,
// which take auto plus the max-extent length grammar (none is valid there).
func minExtentValueAccepted(value string) bool {
	if value == overflowAuto {
		return true
	}

	return boxLengthValueAccepted(value, cssDisplayNone)
}

// boxLengthValueAccepted accepts the length forms lengthBox parses for box
// sizing: vmin/vmax, dvh/svh/lvh, math functions, and CSS lengths. The probe
// context is all zeroes; acceptance must not depend on the element's resolved
// font size or containing block.
func boxLengthValueAccepted(value, autoValue string) bool {
	if value == autoValue {
		return true
	}

	_, ok := lengthBox(value, 0, 0, autoValue)

	return ok
}

// marginLengthValueAccepted mirrors marginLen for margins and inset offsets:
// auto plus every length form marginLen parses.
func marginLengthValueAccepted(value string) bool {
	if value == overflowAuto {
		return true
	}

	return lengthValueAccepted(value)
}

// paddingLengthValueAccepted mirrors the padding appliers, which take lengths
// only. marginLen maps auto to 0, but auto is not a padding value, so the
// declaration is rejected instead of resetting the side.
func paddingLengthValueAccepted(value string) bool {
	if value == overflowAuto {
		return false
	}

	return lengthValueAccepted(value)
}

// lengthValueAccepted accepts the forms marginLen parses: vmin/vmax, math
// functions, and CSS lengths (including percentages).
func lengthValueAccepted(value string) bool {
	if _, ok := vminVmaxPt(value, 0, 0); ok {
		return true
	}

	if _, ok := calcLength(value, 0, 0); ok {
		return true
	}

	_, _, ok := css.ParseLength(value)

	return ok
}

// boxShorthandValueAccepted checks 1-4 space-separated tokens of a margin or
// padding shorthand against the longhand predicate.
func boxShorthandValueAccepted(value string, tokenAccepted func(string) bool) bool {
	var tokens [4]string

	count := splitSpaceTokens(value, tokens[:])
	if count < 1 || count > len(tokens) {
		return false
	}

	for idx := range count {
		if !tokenAccepted(tokens[idx]) {
			return false
		}
	}

	return true
}

// fontWeightValueAccepted mirrors resolveFontWeight.
func fontWeightValueAccepted(value string) bool {
	switch value {
	case contentNormal, cssFontWeightBold, fontWeightBolder, fontWeightLighter:
		return true
	}

	weight, ok := css.ParseNumber(value)

	return ok && weight >= 100 && weight <= 900
}

// verticalAlignValueAccepted mirrors setVerticalAlignValue: keywords,
// percentages, and lengths.
func verticalAlignValueAccepted(value string) bool {
	trimmed := strings.TrimSpace(value)
	low := strings.ToLower(trimmed)

	switch low {
	case "baseline", cssVerticalAlignTop, "middle", cssVerticalAlignBottom, verticalAlignSub, verticalAlignSuper:
		return true
	}

	if strings.HasSuffix(low, "%") {
		_, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(trimmed, "%")), 64)

		return err == nil
	}

	_, ok := plainLength(value, 0, 0)

	return ok
}

// zIndexValueAccepted mirrors setZIndexValue.
func zIndexValueAccepted(value string) bool {
	if value == overflowAuto {
		return true
	}

	_, err := strconv.Atoi(strings.TrimSpace(value))

	return err == nil
}

// keywordIn reports whether value equals one of the accepted keywords.
func keywordIn(value string, accepted ...string) bool {
	for _, keyword := range accepted {
		if value == keyword {
			return true
		}
	}

	return false
}
