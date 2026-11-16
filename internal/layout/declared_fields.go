package layout

// declaredStyle holds the cascaded text of the properties the engine stores
// and exports for a host without acting on them in layout: caret, navigation,
// reading order, draft borders, motion path, bookmarks and the SVG leftovers.
// The empty string is the initial value. Each group has a normalizer in
// declared_<group>.go that decides which text a property accepts.
type declaredStyle struct {
	// caret (nav).
	Caret string
	// caret-animation (nav).
	CaretAnimation string
	// caret-shape (nav).
	CaretShape string
	// field-sizing (nav).
	FieldSizing string
	// input-security (nav).
	InputSecurity string
	// nav-up (nav).
	NavUp string
	// nav-down (nav).
	NavDown string
	// nav-left (nav).
	NavLeft string
	// nav-right (nav).
	NavRight string
	// slider-orientation (nav).
	SliderOrientation string
	// spatial-navigation-action (nav).
	SpatialNavigationAction string
	// spatial-navigation-contain (nav).
	SpatialNavigationContain string
	// spatial-navigation-function (nav).
	SpatialNavigationFunction string
	// window-drag (nav).
	WindowDrag string
	// will-change (nav).
	WillChange string
	// -webkit-appearance (nav).
	WebkitAppearance string
	// -webkit-user-select (nav).
	WebkitUserSelect string
	// -webkit-text-size-adjust (nav).
	WebkitTextSizeAdjust string
	// text-size-adjust (nav).
	TextSizeAdjust string
	// image-animation (nav).
	ImageAnimation string
	// reading-flow (read).
	ReadingFlow string
	// reading-order (read).
	ReadingOrder string
	// overlay (read).
	Overlay string
	// background-attachment (read).
	BackgroundAttachment string
	// background-tbd (read).
	BackgroundTbd string
	// border-clip (shapes).
	BorderClip string
	// border-top-clip (shapes).
	BorderTopClip string
	// border-right-clip (shapes).
	BorderRightClip string
	// border-bottom-clip (shapes).
	BorderBottomClip string
	// border-left-clip (shapes).
	BorderLeftClip string
	// border-block-clip (shapes).
	BorderBlockClip string
	// border-block-start-clip (shapes).
	BorderBlockStartClip string
	// border-block-end-clip (shapes).
	BorderBlockEndClip string
	// border-inline-clip (shapes).
	BorderInlineClip string
	// border-inline-start-clip (shapes).
	BorderInlineStartClip string
	// border-inline-end-clip (shapes).
	BorderInlineEndClip string
	// border-boundary (shapes).
	BorderBoundary string
	// border-limit (shapes).
	BorderLimit string
	// border-shape (shapes).
	BorderShape string
	// text-fit (shapes).
	TextFit string
	// float-defer (shapes).
	FloatDefer string
	// shape-image-threshold (shapes).
	ShapeImageThreshold string
	// column-rule-visibility-items (shapes).
	ColumnRuleVisibilityItems string
	// text-rendering (svg).
	TextRendering string
	// lighting-color (svg).
	LightingColor string
	// glyph-orientation-vertical (svg).
	GlyphOrientationVertical string
	// baseline-shift (svg).
	BaselineShift string
	// marker-side (svg).
	MarkerSide string
	// offset (motion).
	Offset string
	// offset-anchor (motion).
	OffsetAnchor string
	// offset-distance (motion).
	OffsetDistance string
	// offset-path (motion).
	OffsetPath string
	// offset-position (motion).
	OffsetPosition string
	// offset-rotate (motion).
	OffsetRotate string
	// bookmark-label (frag).
	BookmarkLabel string
	// bookmark-level (frag).
	BookmarkLevel string
	// bookmark-state (frag).
	BookmarkState string
	// footnote-display (frag).
	FootnoteDisplay string
	// footnote-policy (frag).
	FootnotePolicy string
	// copy-into (frag).
	CopyInto string
	// block-ellipsis (frag).
	BlockEllipsis string
}

// Declared property groups. A group owns one normalizer file.
const (
	declaredGroupNav    = "nav"
	declaredGroupRead   = "read"
	declaredGroupShapes = "shapes"
	declaredGroupSvg    = "svg"
	declaredGroupMotion = "motion"
	declaredGroupFrag   = "frag"
)

// Property names the declared store owns. The five bookmark and footnote
// names reuse the constants their own files define.
const (
	declPropCaret                     = "caret"
	declPropCaretAnimation            = "caret-animation"
	declPropCaretShape                = "caret-shape"
	declPropFieldSizing               = "field-sizing"
	declPropInputSecurity             = "input-security"
	declPropNavUp                     = "nav-up"
	declPropNavDown                   = "nav-down"
	declPropNavLeft                   = "nav-left"
	declPropNavRight                  = "nav-right"
	declPropSliderOrientation         = "slider-orientation"
	declPropSpatialNavigationAction   = "spatial-navigation-action"
	declPropSpatialNavigationContain  = "spatial-navigation-contain"
	declPropSpatialNavigationFunction = "spatial-navigation-function"
	declPropWindowDrag                = "window-drag"
	declPropWillChange                = "will-change"
	declPropWebkitAppearance          = "-webkit-appearance"       //nolint:gosec // CSS property name, not a credential
	declPropWebkitUserSelect          = "-webkit-user-select"      //nolint:gosec // CSS property name, not a credential
	declPropWebkitTextSizeAdjust      = "-webkit-text-size-adjust" //nolint:gosec // CSS property name, not a credential
	declPropTextSizeAdjust            = "text-size-adjust"
	declPropImageAnimation            = "image-animation"
	declPropReadingFlow               = "reading-flow"
	declPropReadingOrder              = "reading-order"
	declPropOverlay                   = "overlay"
	declPropBackgroundAttachment      = "background-attachment"
	declPropBackgroundTbd             = "background-tbd"
	declPropBorderClip                = "border-clip"
	declPropBorderTopClip             = "border-top-clip"
	declPropBorderRightClip           = "border-right-clip"
	declPropBorderBottomClip          = "border-bottom-clip"
	declPropBorderLeftClip            = "border-left-clip"
	declPropBorderBlockClip           = "border-block-clip"
	declPropBorderBlockStartClip      = "border-block-start-clip"
	declPropBorderBlockEndClip        = "border-block-end-clip"
	declPropBorderInlineClip          = "border-inline-clip"
	declPropBorderInlineStartClip     = "border-inline-start-clip"
	declPropBorderInlineEndClip       = "border-inline-end-clip"
	declPropBorderBoundary            = "border-boundary"
	declPropBorderLimit               = "border-limit"
	declPropBorderShape               = "border-shape"
	declPropTextFit                   = "text-fit"
	declPropFloatDefer                = "float-defer"
	declPropShapeImageThreshold       = "shape-image-threshold"
	declPropColumnRuleVisibilityItems = "column-rule-visibility-items"
	declPropTextRendering             = "text-rendering"
	declPropLightingColor             = "lighting-color"
	declPropGlyphOrientationVertical  = "glyph-orientation-vertical"
	declPropBaselineShift             = "baseline-shift"
	declPropMarkerSide                = "marker-side"
	declPropOffset                    = "offset"
	declPropOffsetAnchor              = "offset-anchor"
	declPropOffsetDistance            = "offset-distance"
	declPropOffsetPath                = "offset-path"
	declPropOffsetPosition            = "offset-position"
	declPropOffsetRotate              = "offset-rotate"
	declPropCopyInto                  = "copy-into"
	declPropBlockEllipsis             = "block-ellipsis"
)

// declaredPropCount is the number of stored properties.
const declaredPropCount = 61

// declaredPropNames lists every stored property.
func declaredPropNames() []string {
	out := make([]string, 0, declaredPropCount)
	out = append(out, declaredNavNames()...)
	out = append(out, declaredReadNames()...)
	out = append(out, declaredShapesNames()...)
	out = append(out, declaredSvgNames()...)
	out = append(out, declaredMotionNames()...)
	out = append(out, declaredFragNames()...)

	return out
}

// declaredNavNames lists the stored nav properties.
func declaredNavNames() []string {
	return []string{
		declPropCaret,
		declPropCaretAnimation,
		declPropCaretShape,
		declPropFieldSizing,
		declPropInputSecurity,
		declPropNavUp,
		declPropNavDown,
		declPropNavLeft,
		declPropNavRight,
		declPropSliderOrientation,
		declPropSpatialNavigationAction,
		declPropSpatialNavigationContain,
		declPropSpatialNavigationFunction,
		declPropWindowDrag,
		declPropWillChange,
		declPropWebkitAppearance,
		declPropWebkitUserSelect,
		declPropWebkitTextSizeAdjust,
		declPropTextSizeAdjust,
		declPropImageAnimation,
	}
}

// declaredReadNames lists the stored read properties.
func declaredReadNames() []string {
	return []string{
		declPropReadingFlow,
		declPropReadingOrder,
		declPropOverlay,
		declPropBackgroundAttachment,
		declPropBackgroundTbd,
	}
}

// declaredShapesNames lists the stored shapes properties.
func declaredShapesNames() []string {
	return []string{
		declPropBorderClip,
		declPropBorderTopClip,
		declPropBorderRightClip,
		declPropBorderBottomClip,
		declPropBorderLeftClip,
		declPropBorderBlockClip,
		declPropBorderBlockStartClip,
		declPropBorderBlockEndClip,
		declPropBorderInlineClip,
		declPropBorderInlineStartClip,
		declPropBorderInlineEndClip,
		declPropBorderBoundary,
		declPropBorderLimit,
		declPropBorderShape,
		declPropTextFit,
		declPropFloatDefer,
		declPropShapeImageThreshold,
		declPropColumnRuleVisibilityItems,
	}
}

// declaredSvgNames lists the stored svg properties.
func declaredSvgNames() []string {
	return []string{
		declPropTextRendering,
		declPropLightingColor,
		declPropGlyphOrientationVertical,
		declPropBaselineShift,
		declPropMarkerSide,
	}
}

// declaredMotionNames lists the stored motion properties.
func declaredMotionNames() []string {
	return []string{
		declPropOffset,
		declPropOffsetAnchor,
		declPropOffsetDistance,
		declPropOffsetPath,
		declPropOffsetPosition,
		declPropOffsetRotate,
	}
}

// declaredFragNames lists the stored frag properties.
func declaredFragNames() []string {
	return []string{
		bookmarkLabelProp,
		bookmarkLevelProp,
		bookmarkStateProp,
		footnotePropDisplay,
		footnotePropPolicy,
		declPropCopyInto,
		declPropBlockEllipsis,
	}
}

// declaredInheritedNames lists the properties that inherit by default.
func declaredInheritedNames() []string {
	return []string{
		declPropCaretAnimation,
		declPropCaretShape,
		declPropWebkitTextSizeAdjust,
		declPropTextSizeAdjust,
		declPropImageAnimation,
		declPropTextRendering,
		declPropGlyphOrientationVertical,
		declPropBlockEllipsis,
	}
}

// declaredInheritsByDefault reports whether prop inherits when unset.
func declaredInheritsByDefault(prop string) bool {
	switch prop {
	case declPropCaretAnimation, declPropCaretShape, declPropWebkitTextSizeAdjust, declPropTextSizeAdjust,
		declPropImageAnimation, declPropTextRendering, declPropGlyphOrientationVertical, declPropBlockEllipsis:
		return true
	}

	return false
}

// declaredGroupOf returns the group of a stored property, or "".
func declaredGroupOf(prop string) string {
	switch prop {
	case declPropCaret, declPropCaretAnimation, declPropCaretShape, declPropFieldSizing,
		declPropInputSecurity, declPropNavUp, declPropNavDown, declPropNavLeft, declPropNavRight,
		declPropSliderOrientation, declPropSpatialNavigationAction, declPropSpatialNavigationContain,
		declPropSpatialNavigationFunction, declPropWindowDrag, declPropWillChange,
		declPropWebkitAppearance, declPropWebkitUserSelect, declPropWebkitTextSizeAdjust,
		declPropTextSizeAdjust, declPropImageAnimation:
		return declaredGroupNav
	case declPropReadingFlow, declPropReadingOrder, declPropOverlay, declPropBackgroundAttachment,
		declPropBackgroundTbd:
		return declaredGroupRead
	case declPropBorderClip, declPropBorderTopClip, declPropBorderRightClip, declPropBorderBottomClip,
		declPropBorderLeftClip, declPropBorderBlockClip, declPropBorderBlockStartClip,
		declPropBorderBlockEndClip, declPropBorderInlineClip, declPropBorderInlineStartClip,
		declPropBorderInlineEndClip, declPropBorderBoundary, declPropBorderLimit, declPropBorderShape,
		declPropTextFit, declPropFloatDefer, declPropShapeImageThreshold,
		declPropColumnRuleVisibilityItems:
		return declaredGroupShapes
	case declPropTextRendering, declPropLightingColor, declPropGlyphOrientationVertical,
		declPropBaselineShift, declPropMarkerSide:
		return declaredGroupSvg
	case declPropOffset, declPropOffsetAnchor, declPropOffsetDistance, declPropOffsetPath,
		declPropOffsetPosition, declPropOffsetRotate:
		return declaredGroupMotion
	case bookmarkLabelProp, bookmarkLevelProp, bookmarkStateProp, footnotePropDisplay,
		footnotePropPolicy, declPropCopyInto, declPropBlockEllipsis:
		return declaredGroupFrag
	}

	return ""
}

// declaredField returns the stored field a property owns, or nil. Each group
// has its own switch, split into parts small enough to read.
func declaredField(store *declaredStyle, prop string) *string {
	switch declaredGroupOf(prop) {
	case declaredGroupNav:
		return declaredNavField(store, prop)
	case declaredGroupRead:
		return declaredReadField(store, prop)
	case declaredGroupShapes:
		return declaredShapesField(store, prop)
	case declaredGroupSvg:
		return declaredSvgField(store, prop)
	case declaredGroupMotion:
		return declaredMotionField(store, prop)
	case declaredGroupFrag:
		return declaredFragField(store, prop)
	}

	return nil
}

// declaredNavField returns the stored field of a nav property, or nil.
func declaredNavField(store *declaredStyle, prop string) *string {
	if field := declaredNavField1(store, prop); field != nil {
		return field
	}

	if field := declaredNavField2(store, prop); field != nil {
		return field
	}

	if field := declaredNavField3(store, prop); field != nil {
		return field
	}

	return nil
}

// declaredNavField1 returns the stored field of one slice of the nav properties, or nil.
func declaredNavField1(store *declaredStyle, prop string) *string {
	switch prop {
	case declPropCaret:
		return &store.Caret
	case declPropCaretAnimation:
		return &store.CaretAnimation
	case declPropCaretShape:
		return &store.CaretShape
	case declPropFieldSizing:
		return &store.FieldSizing
	case declPropInputSecurity:
		return &store.InputSecurity
	case declPropNavUp:
		return &store.NavUp
	case declPropNavDown:
		return &store.NavDown
	case declPropNavLeft:
		return &store.NavLeft
	}

	return nil
}

// declaredNavField2 returns the stored field of one slice of the nav properties, or nil.
func declaredNavField2(store *declaredStyle, prop string) *string {
	switch prop {
	case declPropNavRight:
		return &store.NavRight
	case declPropSliderOrientation:
		return &store.SliderOrientation
	case declPropSpatialNavigationAction:
		return &store.SpatialNavigationAction
	case declPropSpatialNavigationContain:
		return &store.SpatialNavigationContain
	case declPropSpatialNavigationFunction:
		return &store.SpatialNavigationFunction
	case declPropWindowDrag:
		return &store.WindowDrag
	case declPropWillChange:
		return &store.WillChange
	case declPropWebkitAppearance:
		return &store.WebkitAppearance
	}

	return nil
}

// declaredNavField3 returns the stored field of one slice of the nav properties, or nil.
func declaredNavField3(store *declaredStyle, prop string) *string {
	switch prop {
	case declPropWebkitUserSelect:
		return &store.WebkitUserSelect
	case declPropWebkitTextSizeAdjust:
		return &store.WebkitTextSizeAdjust
	case declPropTextSizeAdjust:
		return &store.TextSizeAdjust
	case declPropImageAnimation:
		return &store.ImageAnimation
	}

	return nil
}

// declaredReadField returns the stored field of a read property, or nil.
func declaredReadField(store *declaredStyle, prop string) *string {
	switch prop {
	case declPropReadingFlow:
		return &store.ReadingFlow
	case declPropReadingOrder:
		return &store.ReadingOrder
	case declPropOverlay:
		return &store.Overlay
	case declPropBackgroundAttachment:
		return &store.BackgroundAttachment
	case declPropBackgroundTbd:
		return &store.BackgroundTbd
	}

	return nil
}

// declaredShapesField returns the stored field of a shapes property, or nil.
func declaredShapesField(store *declaredStyle, prop string) *string {
	if field := declaredShapesField1(store, prop); field != nil {
		return field
	}

	if field := declaredShapesField2(store, prop); field != nil {
		return field
	}

	if field := declaredShapesField3(store, prop); field != nil {
		return field
	}

	return nil
}

// declaredShapesField1 returns the stored field of one slice of the shapes properties, or nil.
func declaredShapesField1(store *declaredStyle, prop string) *string {
	switch prop {
	case declPropBorderClip:
		return &store.BorderClip
	case declPropBorderTopClip:
		return &store.BorderTopClip
	case declPropBorderRightClip:
		return &store.BorderRightClip
	case declPropBorderBottomClip:
		return &store.BorderBottomClip
	case declPropBorderLeftClip:
		return &store.BorderLeftClip
	case declPropBorderBlockClip:
		return &store.BorderBlockClip
	case declPropBorderBlockStartClip:
		return &store.BorderBlockStartClip
	case declPropBorderBlockEndClip:
		return &store.BorderBlockEndClip
	}

	return nil
}

// declaredShapesField2 returns the stored field of one slice of the shapes properties, or nil.
func declaredShapesField2(store *declaredStyle, prop string) *string {
	switch prop {
	case declPropBorderInlineClip:
		return &store.BorderInlineClip
	case declPropBorderInlineStartClip:
		return &store.BorderInlineStartClip
	case declPropBorderInlineEndClip:
		return &store.BorderInlineEndClip
	case declPropBorderBoundary:
		return &store.BorderBoundary
	case declPropBorderLimit:
		return &store.BorderLimit
	case declPropBorderShape:
		return &store.BorderShape
	case declPropTextFit:
		return &store.TextFit
	case declPropFloatDefer:
		return &store.FloatDefer
	}

	return nil
}

// declaredShapesField3 returns the stored field of one slice of the shapes properties, or nil.
func declaredShapesField3(store *declaredStyle, prop string) *string {
	switch prop {
	case declPropShapeImageThreshold:
		return &store.ShapeImageThreshold
	case declPropColumnRuleVisibilityItems:
		return &store.ColumnRuleVisibilityItems
	}

	return nil
}

// declaredSvgField returns the stored field of a svg property, or nil.
func declaredSvgField(store *declaredStyle, prop string) *string {
	switch prop {
	case declPropTextRendering:
		return &store.TextRendering
	case declPropLightingColor:
		return &store.LightingColor
	case declPropGlyphOrientationVertical:
		return &store.GlyphOrientationVertical
	case declPropBaselineShift:
		return &store.BaselineShift
	case declPropMarkerSide:
		return &store.MarkerSide
	}

	return nil
}

// declaredMotionField returns the stored field of a motion property, or nil.
func declaredMotionField(store *declaredStyle, prop string) *string {
	switch prop {
	case declPropOffset:
		return &store.Offset
	case declPropOffsetAnchor:
		return &store.OffsetAnchor
	case declPropOffsetDistance:
		return &store.OffsetDistance
	case declPropOffsetPath:
		return &store.OffsetPath
	case declPropOffsetPosition:
		return &store.OffsetPosition
	case declPropOffsetRotate:
		return &store.OffsetRotate
	}

	return nil
}

// declaredFragField returns the stored field of a frag property, or nil.
func declaredFragField(store *declaredStyle, prop string) *string {
	switch prop {
	case bookmarkLabelProp:
		return &store.BookmarkLabel
	case bookmarkLevelProp:
		return &store.BookmarkLevel
	case bookmarkStateProp:
		return &store.BookmarkState
	case footnotePropDisplay:
		return &store.FootnoteDisplay
	case footnotePropPolicy:
		return &store.FootnotePolicy
	case declPropCopyInto:
		return &store.CopyInto
	case declPropBlockEllipsis:
		return &store.BlockEllipsis
	}

	return nil
}
