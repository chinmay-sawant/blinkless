package layout

import (
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// applySizeShorthandProps owns the element property size. One accepted value
// is copied onto width and height. Two accepted values set width, then height.
// A token that either axis would reject writes neither axis, so an earlier
// width and height stay. This is not the @page size descriptor.
func applySizeShorthandProps(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext,
	_ *ResolvedStyle, _ bool,
) bool {
	if prop != "size" {
		return false
	}

	widthTok, heightTok, ok := sizeShorthandTokens(value)
	if !ok || !widthValueAccepted(widthTok) || !heightValueAccepted(heightTok) {
		return true
	}

	setWidthValue(style, widthTok, fsize, ctx)
	setHeightValue(style, heightTok, fsize, ctx)

	return true
}

// sizeValueAccepted reports whether value is a legal element size. The width
// token must pass widthValueAccepted and the height token must pass
// heightValueAccepted. One token is checked for both axes, so size:min-content
// fails because height does not accept that keyword.
func sizeValueAccepted(value string) bool {
	widthTok, heightTok, ok := sizeShorthandTokens(value)
	if !ok {
		return false
	}

	return widthValueAccepted(widthTok) && heightValueAccepted(heightTok)
}

// sizeShorthandTokens splits a size value into the width token and the height
// token. One token is copied to both. Two tokens are width, then height.
func sizeShorthandTokens(value string) (string, string, bool) {
	var tokens [2]string

	count := splitSpaceTokens(value, tokens[:])
	if count < 1 || count > len(tokens) {
		return "", "", false
	}

	if count == 1 {
		return tokens[0], tokens[0], true
	}

	return tokens[0], tokens[1], true
}

// applyMinSizeProps owns the min-size shorthand. One value is copied onto
// min-width and min-height. Two values set min-width, then min-height.
// Both tokens are checked before either setter runs, so a bad token leaves
// an earlier floor in place. Registered on styleGroups.
func applyMinSizeProps(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext,
	_ *ResolvedStyle, _ bool,
) bool {
	if prop != "min-size" {
		return false
	}

	widthValue, heightValue, ok := minSizeLonghands(value)
	if !ok {
		return true
	}

	setMinWidthValue(style, widthValue, fsize, ctx.viewportW)
	setMinHeightValue(style, heightValue, fsize, ctx.viewportH)

	return true
}

// minSizeValueAccepted is the cascade gate for min-size. An invalid value
// must not outrank an earlier valid floor.
func minSizeValueAccepted(value string) bool {
	_, _, ok := minSizeLonghands(value)

	return ok
}

// minSizeLonghands returns the min-width and min-height values to store.
// A single token is copied to both axes. Zero or more than two tokens fail.
func minSizeLonghands(raw string) (string, string, bool) {
	fields := strings.Fields(strings.ToLower(raw))
	if len(fields) == 0 || len(fields) > 2 {
		return "", "", false
	}

	for _, field := range fields {
		if !minSizeTokenOK(field) {
			return "", "", false
		}
	}

	heightValue := fields[0]
	if len(fields) == 2 {
		heightValue = fields[1]
	}

	return fields[0], heightValue, true
}

// minSizeTokenOK reuses the min-width and min-height acceptance check, then
// rejects a missing unit and a negative length. Those two still parse as
// lengths in the longhand setter, but a shorthand must not write them.
func minSizeTokenOK(value string) bool {
	if !minExtentValueAccepted(value) || minSizeMissingUnit(value) {
		return false
	}

	return !minSizeValueNegative(value)
}

// minSizeMissingUnit reports a bare nonzero number. ParseLength invents px
// for that form. Zero without a unit is a real length and is allowed.
func minSizeMissingUnit(value string) bool {
	number, unit, ok := css.ParseLength(value)
	if !ok || strings.HasSuffix(value, unit) {
		return false
	}

	return number != 0
}

// minSizeValueNegative reports a length or percentage below zero. auto, none,
// inherit, and initial are not lengths. lengthBox maps those keywords to -1,
// so they are excluded before that probe.
func minSizeValueNegative(value string) bool {
	switch value {
	case overflowAuto, cssDisplayNone, inheritKeyword, cssKeywordInitial:
		return false
	}

	if number, _, ok := css.ParseLength(value); ok {
		return number < 0
	}

	point, ok := lengthBox(value, 0, 0, cssDisplayNone)

	return ok && point < 0
}

// applyMaxSizeProps owns the max-size shorthand. One accepted value is copied
// onto max-width and max-height. Two values set those axes in that order.
// A rejected token writes neither axis. none goes through the longhand
// setters, which store the unset ceiling as -1.
func applyMaxSizeProps(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext,
	_ *ResolvedStyle, _ bool,
) bool {
	if prop != "max-size" {
		return false
	}

	widthValue, heightValue, ok := maxSizeLonghands(value)
	if !ok {
		return true
	}

	setMaxWidthValue(style, widthValue, fsize, ctx.viewportW)
	setMaxHeightValue(style, heightValue, fsize, ctx.viewportH)

	return true
}

// maxSizeLonghands splits max-size into the max-width and max-height values.
// Both tokens must pass the max-width length check before either is returned.
func maxSizeLonghands(value string) (string, string, bool) {
	var tokens [2]string

	count := splitSpaceTokens(strings.ToLower(strings.TrimSpace(value)), tokens[:])
	if count < 1 || count > len(tokens) {
		return "", "", false
	}

	widthValue := tokens[0]
	heightValue := widthValue

	if count == 2 {
		heightValue = tokens[1]
	}

	widthOK := boxLengthValueAccepted(widthValue, cssDisplayNone)
	heightOK := boxLengthValueAccepted(heightValue, cssDisplayNone)
	if !widthOK || !heightOK {
		return "", "", false
	}

	return widthValue, heightValue, true
}
