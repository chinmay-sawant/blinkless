//nolint:cyclop // color-adjust, forced-colors, and dynamic-range property dispatch
package layout

import "strings"

// Print color adjustment keywords. The apply arm stores only these normalized
// spellings; paint consumers compare against the same constants.
const (
	colorAdjustEconomy = "economy"
	colorAdjustExact   = "exact"

	forcedColorAdjustAuto = "auto"
	forcedColorAdjustNone = "none"
	// forcedColorAdjustPreserveParentColor is spec syntax (css-color-adjust-1).
	// Print has no forced-colors mode, so the paint path treats it like auto.
	forcedColorAdjustPreserveParentColor = "preserve-parent-color"

	colorSchemeNormal = "normal"
	colorSchemeLight  = "light"
	colorSchemeDark   = "dark"

	dynamicRangeLimitNoLimit         = "no-limit"
	dynamicRangeLimitStandard        = "standard"
	dynamicRangeLimitHigh            = "high"
	dynamicRangeLimitConstrainedHigh = "constrained-high"
)

// applyColorAdjustProps owns color-adjust / print-color-adjust,
// forced-color-adjust, color-scheme, and dynamic-range-limit. Values are
// normalized to the constants above; anything else is dropped. The CSS-wide
// keywords are resolved here because inheritProps suppresses its parent copy
// for any declared property (style_cascade.go), so an explicit
// "inherit"/"initial"/"unset"/"revert" must land on this arm.
func applyColorAdjustProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext, parent *ResolvedStyle, hasParent bool,
) bool {
	switch prop {
	case "color-adjust", "print-color-adjust":
		parentValue := colorAdjustEconomy
		if parent != nil {
			parentValue = parent.ColorAdjust
		}

		if v, ok := inheritedToken(value, parentValue, colorAdjustEconomy, hasParent, normalizeColorAdjust); ok {
			style.ColorAdjust = v
		}
	case "forced-color-adjust":
		parentValue := forcedColorAdjustAuto
		if parent != nil {
			parentValue = parent.ForcedColorAdjust
		}

		if v, ok := inheritedToken(value, parentValue, forcedColorAdjustAuto, hasParent, normalizeForcedColorAdjust); ok {
			style.ForcedColorAdjust = v
		}
	case "color-scheme":
		parentValue := colorSchemeNormal
		if parent != nil {
			parentValue = parent.ColorScheme
		}

		if v, ok := inheritedToken(value, parentValue, colorSchemeNormal, hasParent, normalizeColorScheme); ok {
			style.ColorScheme = v
		}
	case "dynamic-range-limit":
		parentValue := dynamicRangeLimitNoLimit
		if parent != nil {
			parentValue = parent.DynamicRangeLimit
		}

		if v, ok := inheritedToken(value, parentValue, dynamicRangeLimitNoLimit, hasParent, normalizeDynamicRangeLimit); ok {
			style.DynamicRangeLimit = v
		}
	default:
		return false
	}

	return true
}

// inheritedToken resolves one declared token for an inherited property.
// CSS-wide keywords map to the parent's value (inherit/unset) or the property
// initial (initial/revert); ordinary tokens must pass normalize, and an
// invalid token leaves the field untouched.
func inheritedToken(
	value, parentValue, initial string, hasParent bool,
	normalize func(string) (string, bool),
) (string, bool) {
	if v, ok := cssWideValue(value, parentValue, initial, hasParent); ok {
		return v, true
	}

	return normalize(value)
}

// cssWideValue resolves the CSS-wide keywords shared by every property in
// this group: parent's value for inherit/unset (initial when there is no
// parent), property initial for initial/revert. Ordinary values return
// ("", false) so the caller can validate them.
func cssWideValue(value, parentValue, initial string, hasParent bool) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case inheritKeyword, cssKeywordUnset:
		if hasParent {
			return parentValue, true
		}

		return initial, true
	case cssKeywordInitial, cssKeywordRevert:
		return initial, true
	default:
		return "", false
	}
}

func normalizeColorAdjust(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case colorAdjustEconomy:
		return colorAdjustEconomy, true
	case colorAdjustExact:
		return colorAdjustExact, true
	default:
		return "", false
	}
}

func normalizeForcedColorAdjust(value string) (string, bool) {
	switch low := strings.ToLower(strings.TrimSpace(value)); low {
	case forcedColorAdjustAuto, forcedColorAdjustNone, forcedColorAdjustPreserveParentColor:
		return low, true
	default:
		return "", false
	}
}

// normalizeColorScheme accepts the keyword combinations this engine can use:
// normal, light, dark, light dark, dark light, only light, only dark. Custom
// idents are legal CSS but carry no print meaning here, so they are dropped.
func normalizeColorScheme(value string) (string, bool) {
	normalized := strings.Join(strings.Fields(strings.ToLower(value)), " ")

	switch normalized {
	case colorSchemeNormal, colorSchemeLight, colorSchemeDark,
		"light dark", "dark light", "only light", "only dark":
		return normalized, true
	default:
		return "", false
	}
}

// normalizeDynamicRangeLimit accepts the spec keywords plus the "constrained"
// spelling the catalog records, canonicalizing it to constrained-high.
func normalizeDynamicRangeLimit(value string) (string, bool) {
	switch low := strings.ToLower(strings.TrimSpace(value)); low {
	case dynamicRangeLimitNoLimit, dynamicRangeLimitStandard, dynamicRangeLimitHigh:
		return low, true
	case dynamicRangeLimitConstrainedHigh, "constrained":
		return dynamicRangeLimitConstrainedHigh, true
	default:
		return "", false
	}
}

// darkSchemeCanvasByte and darkSchemeTextByte are the used paint bytes for a
// dark color-scheme root: #121212 canvas with #e8e8e8 default text, matching
// the fixture-63 prose and Chrome 143.0.7499.40 dark defaults.
const (
	darkSchemeCanvasByte = 0x12
	darkSchemeTextByte   = 0xe8
	// srgbChannelMax scales a paint byte into the 0..1 channel range.
	srgbChannelMax = 255
)

// colorSchemeHasDark reports whether a normalized color-scheme value lets the
// element use a dark scheme: "dark", "light dark", "dark light", "only dark".
// "normal", "light", and "only light" return false. Custom idents never reach
// here because normalizeColorScheme drops them.
func colorSchemeHasDark(scheme string) bool {
	for _, token := range strings.Fields(strings.ToLower(scheme)) {
		if token == colorSchemeDark {
			return true
		}
	}

	return false
}

// defaultCanvasForScheme returns the used canvas fill for a color-scheme
// value. A dark scheme paints an opaque #121212 fill; any other scheme keeps
// the paper itself (transparent, no fill op). Only the html root's value is
// consumed by the paint path; nested values stay parsed and inherited.
func defaultCanvasForScheme(scheme string) [4]float64 {
	if !colorSchemeHasDark(scheme) {
		return [4]float64{}
	}

	channel := float64(darkSchemeCanvasByte) / srgbChannelMax

	return [4]float64{channel, channel, channel, 1}
}

// defaultTextForScheme returns the used default text color for a color-scheme
// value: #e8e8e8 under a dark scheme, black otherwise. Elements with an
// author color keep it; this only replaces the initial black.
func defaultTextForScheme(scheme string) [3]float64 {
	if !colorSchemeHasDark(scheme) {
		return [3]float64{}
	}

	channel := float64(darkSchemeTextByte) / srgbChannelMax

	return [3]float64{channel, channel, channel}
}

// clampDynamicRangeColor clamps one used sRGB color to the output range of a
// dynamic-range-limit value. "standard" and "constrained-high" target sRGB
// output, so channels fold into [0, 1]; "no-limit" and "high" preserve
// headroom and return the input unchanged. The color parser only produces
// 0..1 channels, so every parseable author color passes through untouched;
// the clamp only bites on out-of-range computed values.
func clampDynamicRangeColor(color [3]float64, limit string) [3]float64 {
	switch limit {
	case dynamicRangeLimitStandard, dynamicRangeLimitConstrainedHigh:
		return [3]float64{
			clampDynamicRangeChannel(color[0]),
			clampDynamicRangeChannel(color[1]),
			clampDynamicRangeChannel(color[2]),
		}
	default:
		return color
	}
}

// clampDynamicRangeChannel folds one color channel into [0, 1].
func clampDynamicRangeChannel(channel float64) float64 {
	if channel < 0 {
		return 0
	}

	if channel > 1 {
		return 1
	}

	return channel
}

// forcedColorsPaintColor resolves one used paint color for forced-colors
// mode. When forced mode is off (print has none), the author color always
// wins. When forced mode is on, "auto" and "preserve-parent-color" map to the
// system color while "none" preserves the author color per css-color-adjust-1.
func forcedColorsPaintColor(forcedActive bool, adjust string, author, system [3]float64) [3]float64 {
	if !forcedActive {
		return author
	}

	if adjust == forcedColorAdjustNone {
		return author
	}

	return system
}
