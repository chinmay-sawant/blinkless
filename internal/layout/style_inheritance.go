package layout

// inheritCopy is one inheritable property group: the CSS property names
// (declaring any of them on the element suppresses the copy) and the closure
// that copies the parent's resolved value into the child.
type inheritCopy struct {
	names []string
	copy  func(dst, src *ResolvedStyle)
}

// inheritableProps is the immutable inherit table used by inheritProps.
// Package-level so inheritProps does not allocate a new slice, name slices,
// and closures on every styled node (was ~40% of alloc_objects on a 500-page document).
var inheritableProps = []inheritCopy{ //nolint:gochecknoglobals // static inherit table
	{[]string{"color"}, func(dst, src *ResolvedStyle) { dst.Color = src.Color }},
	{[]string{"accent-color"}, func(dst, src *ResolvedStyle) {
		dst.editContent().AccentColor = src.content().AccentColor
		dst.editContent().AccentColorSet = src.content().AccentColorSet
	}},
	{[]string{"font-family"}, func(dst, src *ResolvedStyle) { dst.FontFamily = src.FontFamily }},
	{[]string{"font-size"}, func(dst, src *ResolvedStyle) { dst.FontSize = src.FontSize }},
	{[]string{"font-weight"}, func(dst, src *ResolvedStyle) { dst.FontWeight = src.FontWeight }},
	{[]string{"font-style"}, func(dst, src *ResolvedStyle) { dst.FontItalic = src.FontItalic }},
	{[]string{"line-height"}, func(dst, src *ResolvedStyle) {
		dst.LineHeight = src.LineHeight
		dst.LineHeightUnitless = src.LineHeightUnitless
	}},
	{[]string{"text-align"}, func(dst, src *ResolvedStyle) { dst.TextAlign = src.TextAlign }},
	{[]string{"text-align-last"}, func(dst, src *ResolvedStyle) { dst.editInlineText().TextAlignLast = src.inlineText().TextAlignLast }},
	{[]string{"text-transform"}, func(dst, src *ResolvedStyle) { dst.editInlineText().TextTransform = src.inlineText().TextTransform }},
	{[]string{"white-space"}, func(dst, src *ResolvedStyle) { dst.WhiteSpace = src.WhiteSpace }},
	{[]string{"white-space-collapse"}, func(dst, src *ResolvedStyle) {
		dst.editInlineText().WhiteSpaceCollapse = src.inlineText().WhiteSpaceCollapse
	}},
	{[]string{"white-space-trim"}, func(dst, src *ResolvedStyle) { dst.editInlineText().WhiteSpaceTrim = src.inlineText().WhiteSpaceTrim }},
	{[]string{propTextWrap}, func(dst, src *ResolvedStyle) { dst.editInlineText().TextWrap = src.inlineText().TextWrap }},
	{[]string{"text-wrap-mode"}, func(dst, src *ResolvedStyle) { dst.editInlineText().TextWrapMode = src.inlineText().TextWrapMode }},
	{[]string{propTextWrapStyle}, func(dst, src *ResolvedStyle) { dst.editInlineText().TextWrapStyle = src.inlineText().TextWrapStyle }},
	{[]string{tabSizeProperty}, func(dst, src *ResolvedStyle) { dst.editInlineText().TabSize = src.inlineText().TabSize }},
	{[]string{propHyphens}, func(dst, src *ResolvedStyle) { dst.editInlineText().Hyphens = src.inlineText().Hyphens }},
	{[]string{"hyphenate-character"}, func(dst, src *ResolvedStyle) {
		dst.editInlineText().HyphenateCharacter = src.inlineText().HyphenateCharacter
	}},
	{[]string{"text-justify"}, func(dst, src *ResolvedStyle) { dst.editInlineText().TextJustify = src.inlineText().TextJustify }},
	{[]string{"line-break"}, func(dst, src *ResolvedStyle) { dst.editInlineText().LineBreak = src.inlineText().LineBreak }},
	// overflow-wrap / word-wrap and word-break are inherited (CSS Text).
	{
		[]string{"overflow-wrap", "word-wrap"},
		func(dst, src *ResolvedStyle) { dst.editInlineText().OverflowWrap = src.inlineText().OverflowWrap },
	},
	{[]string{"word-break"}, func(dst, src *ResolvedStyle) { dst.editInlineText().WordBreak = src.inlineText().WordBreak }},
	{
		[]string{"vertical-align"},
		func(dst, src *ResolvedStyle) {
			dst.editInlineText().VerticalAlign = src.inlineText().VerticalAlign
			dst.editInlineText().VerticalAlignShift = src.inlineText().VerticalAlignShift
		},
	},
	{
		[]string{"text-decoration"},
		func(dst, src *ResolvedStyle) { dst.editInlineText().TextDecoration = src.inlineText().TextDecoration },
	},
	{
		[]string{"letter-spacing"},
		func(dst, src *ResolvedStyle) { dst.editInlineText().LetterSpacing = src.inlineText().LetterSpacing },
	},
	{
		[]string{"word-spacing"},
		func(dst, src *ResolvedStyle) { dst.editInlineText().WordSpacing = src.inlineText().WordSpacing },
	},
	{[]string{"visibility"}, func(dst, src *ResolvedStyle) { dst.Visibility = src.Visibility }},
	{[]string{"caption-side"}, func(dst, src *ResolvedStyle) { dst.editFragmentation().CaptionSide = src.fragmentation().CaptionSide }},
	{
		[]string{"list-style-type", "list-style"},
		func(dst, src *ResolvedStyle) { dst.editContent().ListStyleType = src.content().ListStyleType },
	},
	{
		[]string{"list-style-position", "list-style"},
		func(dst, src *ResolvedStyle) { dst.editContent().ListStylePosition = src.content().ListStylePosition },
	},
	{
		[]string{"list-style-image", "list-style"},
		func(dst, src *ResolvedStyle) { dst.editContent().ListStyleImage = src.content().ListStyleImage },
	},
	{
		[]string{"quotes"},
		func(dst, src *ResolvedStyle) {
			dst.editContent().QuotesRaw = src.content().QuotesRaw
			dst.editContent().QuotesOpen = src.content().QuotesOpen
			dst.editContent().QuotesClose = src.content().QuotesClose
		},
	},
	{
		[]string{"border-collapse"},
		func(dst, src *ResolvedStyle) {
			dst.editFragmentation().BorderCollapse = src.fragmentation().BorderCollapse
		},
	},
	{
		[]string{propBorderSpacing},
		func(dst, src *ResolvedStyle) {
			dst.editFragmentation().BorderSpacing = src.fragmentation().BorderSpacing
			dst.editFragmentation().BorderSpacingV = src.fragmentation().BorderSpacingV
		},
	},
	{[]string{"orphans"}, func(dst, src *ResolvedStyle) { dst.editFragmentation().Orphans = src.fragmentation().Orphans }},
	{[]string{"widows"}, func(dst, src *ResolvedStyle) { dst.editFragmentation().Widows = src.fragmentation().Widows }},
	// CSS page is not inherited at computed-value time, but auto uses the
	// parent used value. Copying when the property is unspecified matches that.
	{[]string{pageKeyword}, func(dst, src *ResolvedStyle) { dst.editFragmentation().PageName = src.fragmentation().PageName }},
	{[]string{"writing-mode"}, func(dst, src *ResolvedStyle) { dst.editPositioning().WritingMode = src.positioning().WritingMode }},
	{[]string{"direction"}, func(dst, src *ResolvedStyle) { dst.editPositioning().Direction = src.positioning().Direction }},
	{[]string{"text-indent"}, func(dst, src *ResolvedStyle) { dst.editInlineText().TextIndent = src.inlineText().TextIndent }},
	{[]string{"fill"}, func(dst, src *ResolvedStyle) {
		dst.editSvg().Fill = src.svg().Fill
		dst.editSvg().FillSet = src.svg().FillSet
	}},
	{[]string{"stroke"}, func(dst, src *ResolvedStyle) {
		dst.editSvg().Stroke = src.svg().Stroke
		dst.editSvg().StrokeSet = src.svg().StrokeSet
	}},
	{[]string{"stroke-width"}, func(dst, src *ResolvedStyle) {
		dst.editSvg().StrokeWidth = src.svg().StrokeWidth
		dst.editSvg().StrokeWidthSet = src.svg().StrokeWidthSet
	}},
	{[]string{"stroke-linecap"}, func(dst, src *ResolvedStyle) { dst.editSvg().StrokeLineCap = src.svg().StrokeLineCap }},
	{[]string{"stroke-linejoin"}, func(dst, src *ResolvedStyle) { dst.editSvg().StrokeLineJoin = src.svg().StrokeLineJoin }},
	{[]string{"stroke-dasharray"}, func(dst, src *ResolvedStyle) { dst.editSvg().StrokeDashArray = src.svg().StrokeDashArray }},
	{[]string{"stroke-dashoffset"}, func(dst, src *ResolvedStyle) { dst.editSvg().StrokeDashOffset = src.svg().StrokeDashOffset }},
	{[]string{"stroke-miterlimit"}, func(dst, src *ResolvedStyle) { dst.editSvg().StrokeMiterLimit = src.svg().StrokeMiterLimit }},
	{[]string{"text-decoration-skip-ink"}, func(dst, src *ResolvedStyle) {
		dst.editInlineText().TextDecorationSkipInk = src.inlineText().TextDecorationSkipInk
	}},
	{[]string{"empty-cells"}, func(dst, src *ResolvedStyle) { dst.editFragmentation().EmptyCells = src.fragmentation().EmptyCells }},
	// Re-added support properties (2026-09-12 demotions): inherited per
	// plans/0.2.6/catalog/mapping.json. The text-decoration-skip entry also
	// copies its inherited longhands because the uint64 declared mask caps the
	// table at 64 entries.
	{[]string{"color-adjust", "print-color-adjust"}, func(dst, src *ResolvedStyle) {
		dst.editContainment().ColorAdjust = src.containment().ColorAdjust
	}},
	{[]string{"forced-color-adjust"}, func(dst, src *ResolvedStyle) {
		dst.editContainment().ForcedColorAdjust = src.containment().ForcedColorAdjust
	}},
	{[]string{"color-scheme"}, func(dst, src *ResolvedStyle) { dst.editContainment().ColorScheme = src.containment().ColorScheme }},
	{[]string{"dynamic-range-limit"}, func(dst, src *ResolvedStyle) {
		dst.editContainment().DynamicRangeLimit = src.containment().DynamicRangeLimit
	}},
	// One inherit entry for the CSS Fonts shaping cluster so the uint64
	// declared mask stays within 64 slots while new feature fields inherit.
	{[]string{
		"font-language-override", "font-optical-sizing", "font-palette", "font-variation-settings",
		"font-feature-settings", "font-kerning", "font-size-adjust", "font-stretch", "font-width",
		"font-synthesis", "font-synthesis-weight", "font-synthesis-style",
		"font-synthesis-small-caps", "font-synthesis-position",
		"font-variant", "font-variant-caps", "font-variant-ligatures", "font-variant-numeric",
		"font-variant-position", "font-variant-east-asian", "font-variant-alternates",
		"font-variant-emoji",
	}, copyInheritedFontShapingProps},
	{[]string{"image-orientation"}, func(dst, src *ResolvedStyle) {
		dst.editImages().ImageOrientation = src.images().ImageOrientation
		dst.editImages().ImageOrientationAngle = src.images().ImageOrientationAngle
	}},
	{[]string{"image-resolution"}, func(dst, src *ResolvedStyle) {
		dst.editImages().ImageResolution = src.images().ImageResolution
		dst.editImages().ImageResolutionDPI = src.images().ImageResolutionDPI
	}},
	{[]string{"text-combine-upright"}, func(dst, src *ResolvedStyle) {
		dst.editInlineText().TextCombineUpright = src.inlineText().TextCombineUpright
	}},
	{[]string{"text-decoration-skip"}, func(dst, src *ResolvedStyle) {
		dst.editInlineText().TextDecorationSkip = src.inlineText().TextDecorationSkip
		dst.editInlineText().TextDecorationSkipBox = src.inlineText().TextDecorationSkipBox
		dst.editInlineText().TextDecorationSkipSpaces = src.inlineText().TextDecorationSkipSpaces
	}},
	// Phase 87.4: one slot left in the uint64 inherit mask, so the new
	// inherited text-box-edge / spacing / hyphenate-limit / hanging fields
	// share a single entry (same coalesce pattern as text-decoration-skip).
	{[]string{
		"text-autospace", "text-box-edge", "text-spacing", "text-spacing-trim",
		"hanging-punctuation",
		"hyphenate-limit-chars", "hyphenate-limit-last", "hyphenate-limit-lines", "hyphenate-limit-zone",
	}, func(dst, src *ResolvedStyle) {
		dst.editInlineText().TextAutospace = src.inlineText().TextAutospace
		dst.editInlineText().TextBoxEdgeOver = src.inlineText().TextBoxEdgeOver
		dst.editInlineText().TextBoxEdgeUnder = src.inlineText().TextBoxEdgeUnder
		dst.editInlineText().TextSpacing = src.inlineText().TextSpacing
		dst.editInlineText().TextSpacingTrim = src.inlineText().TextSpacingTrim
		dst.editInlineText().HangingPunctuation = src.inlineText().HangingPunctuation
		dst.editInlineText().HyphenateLimitMinWord = src.inlineText().HyphenateLimitMinWord
		dst.editInlineText().HyphenateLimitMinBefore = src.inlineText().HyphenateLimitMinBefore
		dst.editInlineText().HyphenateLimitMinAfter = src.inlineText().HyphenateLimitMinAfter
		dst.editInlineText().HyphenateLimitLast = src.inlineText().HyphenateLimitLast
		dst.editInlineText().HyphenateLimitLines = src.inlineText().HyphenateLimitLines
		dst.editInlineText().HyphenateLimitZonePt = src.inlineText().HyphenateLimitZonePt
		dst.editInlineText().HyphenateLimitZonePct = src.inlineText().HyphenateLimitZonePct
	}},
	// initial-letter is not inherited; align/wrap are (CSS Inline 3).
	{[]string{"initial-letter-align", "initial-letter-wrap"}, func(dst, src *ResolvedStyle) {
		dst.editInlineText().InitialLetterAlign = src.inlineText().InitialLetterAlign
		dst.editInlineText().InitialLetterWrap = src.inlineText().InitialLetterWrap
	}},
	{[]string{"text-orientation"}, func(dst, src *ResolvedStyle) { dst.editInlineText().TextOrientation = src.inlineText().TextOrientation }},
	// clip-rule is inherited (SVG 1.1 §14.5): an undeclared element uses its
	// parent's rule at the clip-path mask sites.
	{[]string{"clip-rule"}, func(dst, src *ResolvedStyle) { dst.editSvg().ClipRule = src.svg().ClipRule }},
}

// inheritablePropBits maps an inheritable property name to the bit set of its
// inheritableProps entries. Names can be shared between entries (list-style
// appears in the type and position entries), so bits are ORed. Built once from
// the table so inheritProps can fold the element's declarations into one word
// instead of testing every entry against the raw map (about 65 lookups per
// element before this).
//
//nolint:gochecknoglobals // derived from the static inherit table
var inheritablePropBits = func() map[string]uint64 {
	bits := make(map[string]uint64, len(inheritableProps))

	for i, entry := range inheritableProps {
		for _, name := range entry.names {
			bits[name] |= uint64(1) << i
		}
	}

	return bits
}()

// declaredInheritableMask folds the raw declarations into one bit per
// inheritableProps entry. Iterating the raw keys (about 10 per element)
// replaces the per-entry raw map lookups.
func declaredInheritableMask(raw map[string]string) uint64 {
	var mask uint64

	for prop := range raw {
		if bit, ok := inheritablePropBits[prop]; ok {
			mask |= bit
		}
	}

	return mask
}

// inheritProps copies inheritable properties from the parent, unless the
// element declares its own value (present in raw). The declared-set mask is a
// local word; it deliberately does not grow ResolvedStyle, whose byte size is
// pinned by the interning and storage tests.
func inheritProps(dst *ResolvedStyle, parent *ResolvedStyle, raw map[string]string) {
	if parent == nil {
		return
	}

	declared := declaredInheritableMask(raw)

	for i := range inheritableProps {
		if declared&(uint64(1)<<i) != 0 {
			continue
		}

		inheritableProps[i].copy(dst, parent)
	}

	inheritGapBProps(dst, parent, raw)
	inheritExtraProps(dst, parent, raw)
}

// inheritGapBProps copies inherited properties that cannot
// join inheritableProps: the table already holds 64 entries and the uint64
// declared mask is full, so a 65th entry would overflow. Each copies only when
// the element does not declare it; a declared value is applied later by
// applyRestProps and wins. TransformBox, TransformStyle, and
// BackfaceVisibility are not inherited per CSS Transforms 2 and SVG 2, so
// they keep their initials here.
func inheritGapBProps(dst *ResolvedStyle, parent *ResolvedStyle, raw map[string]string) {
	if parent == nil {
		return
	}

	if _, ok := raw["fill-rule"]; !ok {
		dst.editSvg().FillRule = parent.svg().FillRule
	}

	if _, ok := raw["shape-rendering"]; !ok {
		dst.editSvg().ShapeRendering = parent.svg().ShapeRendering
	}

	if _, ok := raw["dominant-baseline"]; !ok {
		dst.editInlineText().DominantBaseline = parent.inlineText().DominantBaseline
	}

	if _, ok := raw["color-interpolation"]; !ok {
		dst.editSvg().ColorInterpolation = parent.svg().ColorInterpolation
	}

	if _, ok := raw["color-interpolation-filters"]; !ok {
		dst.editSvg().ColorInterpolationFilters = parent.svg().ColorInterpolationFilters
	}

	inheritGapBPropsWide(dst, parent, raw)
}
