package layout

// styleBackground holds background CSS values.
type styleBackground struct {
	BackgroundBlendMode string // comma-separated background layer modes
	// BackgroundImage is a raw url(...) target for the first layer, empty if none.
	BackgroundImage        string
	BackgroundPosX         string
	BackgroundPosY         string
	BackgroundSize         string
	BackgroundRepeat       string
	BackgroundRepeatX      string
	BackgroundRepeatY      string
	BackgroundRepeatBlock  string
	BackgroundRepeatInline string
	BackgroundClip         string
	BackgroundOrigin       string
	BackgroundAttachment   string
}

var initialStyleBackground = styleBackground{
	BackgroundBlendMode: blendNormal,
}

// styleBorders holds borders CSS values.
type styleBorders struct {
	BorderTop                                                                                                          border
	BorderRight                                                                                                        border
	BorderBottom                                                                                                       border
	BorderLeft                                                                                                         border
	BorderRadius                                                                                                       float64
	BorderRadiusPercent                                                                                                float64
	BorderRadiusTopLeft, BorderRadiusTopRight, BorderRadiusBottomRight, BorderRadiusBottomLeft                         float64
	BorderRadiusTopLeftY, BorderRadiusTopRightY, BorderRadiusBottomRightY, BorderRadiusBottomLeftY                     float64
	CornerShapeTopLeft, CornerShapeTopRight, CornerShapeBottomRight, CornerShapeBottomLeft                             float64
	CornerShapeTopLeftKeyword, CornerShapeTopRightKeyword, CornerShapeBottomRightKeyword, CornerShapeBottomLeftKeyword string
	// Outline* is CSS outline. Empty OutlineStyle means none. Does not affect layout size.
	OutlineWidth       float64
	OutlineStyle       string
	OutlineColor       [3]float64
	OutlineColorSet    bool
	OutlineOffset      float64
	BorderImageSource  string
	BorderImageSlice   string
	BorderImageWidth   string
	BorderImageOutset  string
	BorderImageRepeat  string
	BoxDecorationBreak string
}

var initialStyleBorders = styleBorders{
	CornerShapeTopLeft:            CornerShapeRoundK,
	CornerShapeTopRight:           CornerShapeRoundK,
	CornerShapeBottomRight:        CornerShapeRoundK,
	CornerShapeBottomLeft:         CornerShapeRoundK,
	CornerShapeTopLeftKeyword:     "round",
	CornerShapeTopRightKeyword:    "round",
	CornerShapeBottomRightKeyword: "round",
	CornerShapeBottomLeftKeyword:  "round",
}

// styleColumns holds columns CSS values.
type styleColumns struct {
	ColumnCount                  int     // 0 = auto; ≥1 = used count hint
	ColumnWidth                  float64 // -1 = auto; else length in pt
	ColumnHeight                 float64 // -1 = auto; else length in pt (Multicol 2)
	ColumnWrap                   string  // "auto" | "nowrap" | "wrap"
	ColumnSpan                   string  // cssDisplayNone | "all" (multicol spanner)
	ColumnFill                   string  // "balance" | overflowAuto
	ColumnRuleWidth              float64 // pt; CSS initial medium
	ColumnRuleStyle              string  // none | solid | dashed | dotted
	ColumnRuleColor              [3]float64
	ColumnRuleColorSet           bool       // false → paint uses currentColor (Color)
	RowRuleWidth                 float64    // pt; first item; CSS initial medium. Painter uses this.
	RowRuleStyle                 string     // none | solid | dashed | dotted. Painter uses this.
	RowRuleColor                 [3]float64 // first color. Painter uses this.
	RowRuleColorSet              bool       // false means paint uses currentColor (Color)
	RowRuleWidths                []float64  // every accepted width, first is RowRuleWidth
	RowRuleStyles                []string   // every accepted style, first is RowRuleStyle
	RowRuleColors                []string   // raw color tokens, first is the one parsed into RowRuleColor
	RowRuleBreak                 string     // none | normal | intersection
	RowRuleVisibility            string     // all | around | between | normal
	RowRuleInsetCapStart         ruleInset  // start endpoint at a content edge
	RowRuleInsetCapEnd           ruleInset  // end endpoint at a content edge
	RowRuleInsetJunctionStart    ruleInset  // start endpoint at a gap junction
	RowRuleInsetJunctionEnd      ruleInset  // end endpoint at a gap junction
	ColumnRuleBreak              string     // none | normal | intersection
	ColumnRuleInsetCapStart      ruleInset
	ColumnRuleInsetCapEnd        ruleInset
	ColumnRuleInsetJunctionStart ruleInset
	ColumnRuleInsetJunctionEnd   ruleInset
	RuleOverlap                  string // row-over-column | column-over-row
}

var initialStyleColumns = styleColumns{
	ColumnWidth:       -1,
	ColumnHeight:      -1,
	ColumnWrap:        columnWrapAuto,
	ColumnSpan:        cssDisplayNone,
	ColumnFill:        "balance",
	ColumnRuleWidth:   borderWidth(mediumKeyword, 0),
	ColumnRuleStyle:   cssDisplayNone,
	ColumnRuleBreak:   contentNormal,
	RowRuleWidth:      borderWidth(mediumKeyword, 0),
	RowRuleStyle:      cssDisplayNone,
	RowRuleBreak:      contentNormal,
	RowRuleVisibility: contentNormal,
}

// styleContainment holds containment CSS values.
type styleContainment struct {
	ContainerType            string // "" | "normal" | "inline-size" | "size"
	ContainerName            string // space-separated lower-case names; empty = none
	OverflowClipMarginTop    float64
	OverflowClipMarginRight  float64
	OverflowClipMarginBottom float64
	OverflowClipMarginLeft   float64
	// Scroll-margin snap outsets (CSS Scroll Snap 1), in points. Zero on
	// every side is the CSS initial value. The static engine carries no
	// scroll offset, so these inset only the snap/visibility rect computed
	// in scroll.go; static paint still clips to the padding box.
	ScrollMarginTop    float64
	ScrollMarginRight  float64
	ScrollMarginBottom float64
	ScrollMarginLeft   float64
	// Containment and content visibility (CSS Containment).
	Contain                    string  // "none" or space-separated: size layout paint style content
	ContainIntrinsicWidth      float64 // pt; -1 = auto/unset
	ContainIntrinsicHeight     float64 // pt; -1 = auto/unset
	ContainIntrinsicBlockSize  float64 // pt; -1 = auto/unset
	ContainIntrinsicInlineSize float64 // pt; -1 = auto/unset
	ContentVisibility          string  // "visible" | "hidden" | "auto"
	// Print color adjustment family.
	ColorAdjust       string // print-color-adjust/color-adjust: "economy" | "exact"
	ForcedColorAdjust string // "auto" | "none"
	ColorScheme       string // raw value: "normal" | "light" | "dark" | "light dark" | "only light"
	DynamicRangeLimit string // raw value
}

var initialStyleContainment = styleContainment{
	Contain:                    "none",
	ContainIntrinsicWidth:      -1,
	ContainIntrinsicHeight:     -1,
	ContainIntrinsicBlockSize:  -1,
	ContainIntrinsicInlineSize: -1,
	ContentVisibility:          "visible",
	ColorAdjust:                "economy",
	ForcedColorAdjust:          "auto",
	ColorScheme:                "normal",
	DynamicRangeLimit:          "no-limit",
}

// styleContent holds content CSS values.
type styleContent struct {
	// AccentColor is CSS accent-color when authored; AccentColorSet is false
	// when the property is absent so widgets can keep their default fill.
	AccentColor    [3]float64
	AccentColorSet bool
	ListStyleType  string // "disc" | "circle" | "square" | "decimal" | cssDisplayNone | …
	Content        string
	// ListStylePosition is "inside" or "outside"; empty means outside.
	ListStylePosition string
	QuotesRaw         string
	QuotesOpen        string
	QuotesClose       string
	CounterReset      string
	CounterIncrement  string
	CounterSet        string
	ListStyleImage    string
	// LinkParameters is the canonical link-parameters list; empty means none.
	LinkParameters string
}

var initialStyleContent = styleContent{
	ListStyleType: "disc",
}

// styleEffects holds effects CSS values.
type styleEffects struct {
	Filter         string
	BackdropFilter string
	MixBlendMode   string // normal | multiply | screen | ...
	Isolation      string // auto | isolate
	// Static 2D CSS transforms (paint-time CTM; sibling flow unchanged).
	Transform            Matrix2D
	HasTransform         bool
	TransformOrigin      transformOriginSpec
	TranslateXPercent    float64 // % of border-box width for `translate` longhand
	TranslateXPercentSet bool
	TranslateYPercent    float64 // % of border-box height
	TranslateYPercentSet bool
	MaskImage            string
	MaskMode             string
	MaskComposite        string
	MaskPosition         string
	MaskRepeat           string
	MaskSize             string
	MaskOrigin           string
	MaskClip             string
	MaskType             string // specified alpha or luminance; empty means the author did not declare it
	MaskBorderSource     string
	MaskBorderSlice      string
	MaskBorderWidth      string
	MaskBorderOutset     string
	MaskBorderRepeat     string
	MaskBorderMode       string
	// ClipPath is the canonical clip-path value: inset() | circle() | ellipse()
	// | polygon(). Empty means no clip (none / unsupported / invalid).
	ClipPath        string
	BoxShadowX      float64
	BoxShadowY      float64
	BoxShadowBlur   float64
	BoxShadowSpread float64
	BoxShadowColor  [3]float64
	BoxShadowSet    bool
	BoxShadowInset  bool
	BoxShadowRaw    string
	// Gap-B transform and SVG text used values. TransformBox selects the
	// transform-origin reference box ("view-box" initial per CSS Transforms 2;
	// content-box resolves against the content box, all other boxes use the
	// border box for CSS layout boxes). TransformStyle is flat or preserve-3d
	// (2D engine flattens both). BackfaceVisibility is visible or hidden (2D
	// engine has no 3D back face to hide). FillRule is nonzero or evenodd
	// (external canvas rasterizer ignores it; clip masks keep using ClipRule).
	// ShapeRendering is the SVG hint (external rasterizer ignores it).
	// DominantBaseline is the SVG/CSS baseline selector; offsets come from
	// font metrics in inline_vertical_align.go.
	// AlignmentBaseline is the inline alignment selector: auto and baseline
	// defer to DominantBaseline, other selectors shift the run from font
	// metrics in inline_vertical_align.go. Not inherited per CSS Inline 3.
	// BaselineSource picks the first or last baseline set of an inline-level
	// box (CSS Inline 3). auto means last on an inline-block and first on
	// every other box. It does not inherit. A token outside auto, first, and
	// last leaves the previous value.
	// Clip is the deprecated CSS2 clip rect on absolutely positioned boxes:
	// "" means auto (no clip), otherwise the canonical rect() declaration
	// resolved against the border box in overflow_clip.go. Reference for all
	// three: Chrome 143.0.7499.40.
	// ColorInterpolation selects the blending space for gradient sampling:
	// "srgb" interpolates gamma-encoded channels directly (the historic path)
	// while "linearrgb" converts stops to linear light, interpolates there,
	// and encodes back to sRGB. ColorInterpolationFilters does the same for
	// filter primitives (blur, grayscale, invert). Both are inherited per SVG
	// 2; "auto" reads as "srgb" in this engine. Reference: Chrome 143.0.7499.40.
	TransformBox       string
	TransformStyle     string
	BackfaceVisibility string
	// Gap-L 3D flattening state (CSS Transforms 2). Transform3D accumulates
	// the 3D functions of the transform list (rotateX/Y/Z, rotate3d,
	// translateZ, scaleZ, perspective(), matrix3d) in post-multiply order;
	// HasTransform3D reports whether any parsed. The 2D bake in Transform
	// carries identity for those functions. boxTransformAccum flattens the
	// side matrix onto the paint op. PerspectiveDist is the
	// perspective property distance in pt (HasPerspective set unless none);
	// PerspectiveOrigin parts position the vanishing point inside the
	// perspective box (PerspectiveOriginSet when declared). Reference for
	// all: Chrome 143.0.7499.40.
	Transform3D           [16]float64
	HasTransform3D        bool
	PerspectiveDist       float64
	HasPerspective        bool
	PerspectiveOriginX    float64
	PerspectiveOriginXPct bool
	PerspectiveOriginY    float64
	PerspectiveOriginYPct bool
	PerspectiveOriginSet  bool
	Clip                  string
}

var initialStyleEffects = styleEffects{
	MixBlendMode:       blendNormal,
	Isolation:          "auto",
	MaskImage:          "none",
	MaskMode:           "match-source",
	MaskComposite:      "add",
	MaskPosition:       "0% 0%",
	MaskRepeat:         "repeat",
	MaskSize:           "auto",
	MaskOrigin:         "border-box",
	MaskClip:           "border-box",
	MaskType:           "",
	MaskBorderSource:   "none",
	MaskBorderSlice:    "0",
	MaskBorderWidth:    "auto",
	MaskBorderOutset:   "0",
	MaskBorderRepeat:   "stretch",
	MaskBorderMode:     "alpha",
	Transform:          IdentityMatrix(),
	TransformOrigin:    defaultTransformOrigin(),
	TransformBox:       "view-box",
	TransformStyle:     "flat",
	BackfaceVisibility: "visible",
}

// styleFlex holds flex CSS values.
type styleFlex struct {
	IsWebkitBox      bool    // true when display was -webkit-box / -webkit-inline-box (legacy)
	FlexDirection    string  // "row" | fxCol | "row-reverse" | "column-reverse"
	FlexWrap         string  // "nowrap" | "wrap" | "wrap-reverse"
	JustifyContent   string  // flex-start | flex-end | center | space-between | space-around | space-evenly
	AlignItems       string  // stretch | flex-start | center | flex-end
	AlignContent     string  // flex-start | flex-end | center | space-between | space-around | space-evenly | stretch
	AlignSelf        string  // auto | stretch | flex-start | flex-end | center | start | end
	JustifyItems     string  // grid: stretch | start | end | center
	JustifySelf      string  // grid item: auto | stretch | start | end | center
	Gap              float64 // flex/grid gap shorthand (pt); kept for backward compat
	RowGap           float64 // pt; 0 with ColumnGap 0 → layout falls back to Gap
	ColumnGap        float64
	ColumnGapNormal  bool // true when column-gap is normal/initial (multicol → 1em; flex/grid → 0)
	FlexGrow         float64
	FlexShrink       float64 // default 1; 0 disables shrink
	FlexBasis        float64 // -1 = auto
	FlexBasisPercent float64 // >=0 means % of flex container content main size (width/height)
	FlexOrder        int
}

var initialStyleFlex = styleFlex{
	FlexGrow:         0,
	FlexShrink:       1,
	FlexBasis:        -1,
	FlexBasisPercent: -1,
	FlexDirection:    "row",
	FlexWrap:         "nowrap",
	JustifyContent:   "flex-start",
	AlignItems:       "stretch",
	AlignContent:     "stretch",
	AlignSelf:        overflowAuto,
	JustifyItems:     "stretch",
	JustifySelf:      overflowAuto,
	ColumnGapNormal:  true,
}

// styleFont holds font CSS values.
type styleFont struct {
	// Font variation / shaping overrides.
	FontLanguageOverride  string // "normal" or quoted string
	FontOpticalSizing     string // "auto" | "none"
	FontPalette           string // raw value
	FontVariationSettings string // raw value
	// OpenType feature / variant / synthesis / width / size-adjust (CSS Fonts).
	FontFeatureSettings    string  // "normal" or canonical `"tag" N, ...`
	FontKerning            string  // "auto" | "normal" | "none"
	FontSizeAdjust         float64 // aspect number; used when FontSizeAdjustSet
	FontSizeAdjustSet      bool
	FontWidth              float64 // percent; 100 = normal
	FontSynthesisWeight    bool    // true = auto (allow fake bold)
	FontSynthesisStyle     bool    // true = auto; no consumer yet
	FontSynthesisSmallCaps bool    // true = auto; no consumer yet
	FontSynthesisPosition  bool    // true = auto; no consumer yet
	FontVariantCaps        string  // "normal" | small-caps | ...
	FontVariantLigatures   string  // "normal" | "none" | keyword list
	FontVariantNumeric     string  // "normal" | keyword list
	FontVariantPosition    string  // "normal" | "sub" | "super"
	FontVariantEastAsian   string  // "normal" | keyword list
	FontVariantAlternates  string  // historical-forms / stylistic() / styleset() / swash(); OT tags
	FontVariantEmoji       string  // normal|text|emoji|unicode; presentation consumer
}

var initialStyleFont = styleFont{
	FontLanguageOverride:   "normal",
	FontOpticalSizing:      "auto",
	FontPalette:            "normal",
	FontVariationSettings:  "normal",
	FontFeatureSettings:    "normal",
	FontKerning:            "auto",
	FontWidth:              100,
	FontSynthesisWeight:    true,
	FontSynthesisStyle:     true,
	FontSynthesisSmallCaps: true,
	FontSynthesisPosition:  true,
	FontVariantCaps:        "normal",
	FontVariantLigatures:   "normal",
	FontVariantNumeric:     "normal",
	FontVariantPosition:    "normal",
	FontVariantEastAsian:   "normal",
	FontVariantAlternates:  "normal",
	FontVariantEmoji:       "normal",
}

// styleFragmentation holds fragmentation CSS values.
type styleFragmentation struct {
	BorderCollapse  string // "separate" | "collapse"
	BorderSpacing   float64
	BorderSpacingV  float64
	TableLayout     string // overflowAuto | "fixed"
	CaptionSide     string // "top" | "bottom" | "left" | "right"; empty means top
	PageBreakBefore string // "" | "always" | "avoid"
	PageBreakAfter  string // "" | "always" | "avoid"
	PageBreakInside string // "" | "always" | "avoid" | "avoid-column"
	// MarginBreak is CSS margin-break ("auto" | "keep" | "discard") for page breaks.
	MarginBreak string
	// PageName is the CSS page used value: empty is auto, else a lower-case
	// named page ident. Specified auto keeps the parent used value.
	PageName   string
	Orphans    int    // CSS orphans; inherited; initial 2; integer ≥ 1
	Widows     int    // CSS widows; inherited; initial 2; integer ≥ 1
	EmptyCells string // "show" | "hide"; inherited per CSS Tables 3; "" = show (initial)
	// Named flows (css-regions-1 lite, one canvas). FlowInto is "" (none) or
	// the flow custom ident; FlowIntoElement selects the element keyword
	// (the source box moves with its content) over the initial content
	// keyword. FlowFrom is the region's flow ident ("" = none).
	// RegionFragment is "" (auto) or "break" (move a straddling block child
	// whole into the next region instead of slicing it).
	FlowInto        string
	FlowIntoElement bool
	FlowFrom        string
	RegionFragment  string
}

var initialStyleFragmentation = styleFragmentation{
	BorderCollapse: "separate",
	BorderSpacing:  0,
	TableLayout:    overflowAuto,
	Orphans:        2,
	Widows:         2,
	EmptyCells:     "",
}

// styleGeometry holds geometry CSS values.
type styleGeometry struct {
	BoxSizing        string  // "content-box" | "border-box"
	MinWidth         float64 // absolute pt when MinWidthPercent < 0; 0 = auto (content min for flex)
	MinWidthPercent  float64 // >=0 means % of containing block (deferred like WidthPercent)
	MinWidthSet      bool    // true when min-width was explicitly declared, including 0
	MaxWidth         float64
	MaxWidthPercent  float64 // >=0 means % of containing block / img clamp context
	MinHeight        float64
	MinHeightPercent float64 // >=0 means % of CB height; indefinite → ignore
	MinHeightSet     bool    // true when min-height was explicitly declared (non-auto), including 0
	MaxHeight        float64
	MaxHeightPercent float64 // >=0 means % of CB height; -1 = none/auto (mirrors MaxWidthPercent)
	OverflowX        string
	OverflowY        string
	MarginTopAuto    bool // margin-top: auto in a flex column
	MarginBottomAuto bool // margin-bottom: auto in a flex column
	MarginLeftAuto   bool // margin-left: auto (horizontal centering with right auto)
	MarginRightAuto  bool // margin-right: auto
	MarginTrim       string
	// AspectRatio is width/height; 0 means auto/unset.
	AspectRatio float64
	// Zoom is this element's CSS zoom factor. The initial value is 1.
	// A zero value is read as 1 so a zero ResolvedStyle does not scale.
	Zoom float64
}

var initialStyleGeometry = styleGeometry{
	BoxSizing:        "content-box",
	MinWidth:         0,
	MinWidthPercent:  -1,
	MaxWidth:         -1,
	MaxWidthPercent:  -1,
	MinHeight:        0,
	MinHeightPercent: -1,
	MaxHeight:        -1,
	MaxHeightPercent: -1,
	Zoom:             1,
	OverflowX:        "visible",
	OverflowY:        "visible",
}

// styleGrid holds grid CSS values.
type styleGrid struct {
	GridTemplateColumns string // raw grid-template-columns value
	GridTemplateRows    string
	GridTemplateAreas   string // raw grid-template-areas value
	GridArea            string // named area (custom-ident); empty = line-based placement
	GridAutoFlow        string // "row" | fxCol | "dense" | "row dense" | "column dense"
	GridAutoColumns     string // raw grid-auto-columns value (auto | length | minmax(...))
	GridAutoRows        string // raw grid-auto-rows value
	GridColumnSpan      int    // from grid-column: span N (default 1)
	GridColumnStart     int    // 1-based; 0 = auto
	GridRowSpan         int    // from grid-row: span N (default 1)
	GridRowStart        int    // 1-based; 0 = auto
	GridColumnEnd       int
	GridRowEnd          int
	// FlowTolerancePt is the css-grid-3 masonry placement slack in pt;
	// < 0 means normal (resolved to one em when the packer runs).
	// FlowToleranceInfinite keeps the shortest-column choice.
	FlowTolerancePt       float64
	FlowToleranceInfinite bool
}

var initialStyleGrid = styleGrid{
	FlowTolerancePt: -1,
	GridColumnSpan:  1,
	GridRowSpan:     1,
}

// styleImages holds images CSS values.
type styleImages struct {
	// Image adjustment.
	ImageOrientation      string  // "from-image" | "none" | raw angle
	ImageOrientationAngle float64 // degrees; 0 = unset
	ImageResolution       string  // "from-image" | raw resolution token
	ImageResolutionDPI    float64 // 0 = unset/from-image
	ObjectViewBox         string  // raw value
	ObjectFit             string  // fill | contain | cover | none | scale-down
	ObjectPositionX       string  // background-position-x grammar subset
	ObjectPositionY       string
	// ImageRendering is the image-rendering keyword; empty means auto.
	ImageRendering string
}

var initialStyleImages = styleImages{
	ImageOrientation: imageAdjustFromImage,
	ImageResolution:  imageAdjustFromImage,
	ObjectViewBox:    "none",
	ObjectFit:        "fill",
	ObjectPositionX:  "50%",
	ObjectPositionY:  "50%",
}

// styleInlineText holds inline text CSS values.
type styleInlineText struct {
	// Initial-letter drop/raised caps (CSS Inline 3). Size 0 = normal.
	// Authoring: apply on a real leading element (e.g. <span>); :first-letter
	// is rejected by the CSS selector parser.
	InitialLetterSize  float64 // size in lines; 0 = normal
	InitialLetterSink  int     // sink lines; 0 with size>0 means size (drop)
	InitialLetterAlign string  // alphabetic | ideographic | hanging | leading
	InitialLetterWrap  string  // none | first | all | grid | <length>
	TextAlignLast      string  // "auto" | "left" | "right" | "center" | "justify"
	TextTransform      string  // "none" | "uppercase" | "lowercase" | "capitalize"
	VerticalAlign      string  // "baseline" | "top" | "middle" | cssVerticalAlignBottom
	VerticalAlignShift float64 // pt; CSS length vertical-align (positive raises)
	WhiteSpaceCollapse string  // "collapse" | "preserve" | "preserve-breaks" | "preserve-spaces" | "break-spaces"
	WhiteSpaceTrim     string  // "none" | "discard-before" | "discard-after" | "discard-inner"
	TextWrap           string
	TextWrapMode       string
	TextWrapStyle      string
	TabSize            float64
	Hyphens            string
	HyphenateCharacter string
	// HyphenateLimitMinWord/Before/After are hyphenate-limit-chars components.
	// Zero means CSS auto (5 / 2 / 2) at the SHY consumer.
	HyphenateLimitMinWord   int
	HyphenateLimitMinBefore int
	HyphenateLimitMinAfter  int
	HyphenateLimitLast      string  // "none" | "always" | "column" | "page" | "spread"
	HyphenateLimitLines     int     // -1 = no-limit (initial)
	HyphenateLimitZonePt    float64 // absolute zone; ignored when ZonePercent >= 0
	HyphenateLimitZonePct   float64 // 0..100 when set; -1 means use ZonePt
	HangingPunctuation      string  // "none" | "first" | "last" | "allow-end" | …
	TextJustify             string
	LineBreak               string
	// OverflowWrap is CSS overflow-wrap / word-wrap: "normal" | "break-word" | "anywhere".
	OverflowWrap string
	// WordBreak is CSS word-break: "normal" | "break-all" | "keep-all".
	WordBreak               string
	TextDecoration          string // cssDisplayNone | "underline" | "line-through"
	TextDecorationLine      string
	TextDecorationColor     [3]float64
	TextDecorationColorSet  bool
	TextDecorationStyle     string
	TextDecorationThickness float64
	TextUnderlineOffset     float64
	TextUnderlinePosition   string
	TextShadowX             float64
	TextShadowY             float64
	TextShadowBlur          float64
	TextShadowColor         [3]float64
	TextShadowSet           bool
	LetterSpacing           float64
	// WordSpacing is CSS word-spacing in points; 0 is normal.
	WordSpacing float64
	// WordSpaceTransform is CSS word-space-transform. "none" (initial)
	// leaves U+200B and wbr alone. "space" replaces them with U+0020.
	// "ideographic-space" replaces them with U+3000. Either keyword may
	// be paired with auto-phrase; this engine inserts no phrase breaks.
	// Inherited. A token outside the grammar leaves the previous value.
	WordSpaceTransform string
	TextIndent         float64
	// LinePadding is CSS line-padding in points. It insets both ends of
	// every line, including lines after the first. Zero is the initial.
	LinePadding           float64
	TextOverflow          string
	LineClamp             int
	MaxLines              int
	TextDecorationSkipInk string
	// Advanced text support.
	TextCombineUpright       string  // "none" | "all" | "digits N"
	TextDecorationInset      float64 // pt
	TextDecorationSkip       string  // raw shorthand value
	TextDecorationSkipBox    string
	TextDecorationSkipSelf   string
	TextDecorationSkipSpaces string
	TextOrientation          string // "mixed" | "upright" | "sideways"
	UnicodeBidi              string // "normal" | "embed" | "isolate" | "bidi-override" | "isolate-override" | "plaintext"
	// CSS Inline 3 text-box family (trim is not inherited; edge is).
	TextBoxTrim      string // "none" | "trim-start" | "trim-end" | "trim-both"
	TextBoxEdgeOver  string // "auto" | "text" | "cap" | "ex" | …
	TextBoxEdgeUnder string // "auto" | "text" | "alphabetic" | …
	// LineFitEdge is line-fit-edge. "leading" (initial) keeps the span's
	// padding and border in the line box. "text" leaves them out. Inherited.
	LineFitEdge string
	// CSS Text 4/5 spacing + group align + fit.
	TextAutospace     string // "no-autospace" | "ideograph-alpha" | …
	TextSpacing       string // "normal" | "trim-start" | …
	TextSpacingTrim   string // "space-all" | "trim-start" | "trim-both" | …
	TextGroupAlign    string // "none" | "start" | "end" | "left" | "right" | "center"
	TextFit           string // "none" | "auto" | "scale" (apply-only; no scale consumer yet)
	DominantBaseline  string
	AlignmentBaseline string
	BaselineSource    string
	// Continue is the CSS Overflow 4 continue keyword; empty means normal.
	Continue string
	// Text stroke: width in points (0 means none), and the color when set.
	TextStrokeWidth    float64
	TextStrokeColor    [3]float64
	TextStrokeColorSet bool
	// ContinueLines is the line limit an anonymous flex item inherits from a
	// continue: -webkit-legacy parent. 0 means the item computes its own.
	ContinueLines int
	// InlineSizing is inline-sizing. "normal" keeps the font line-height
	// background. "stretch" grows that fill to the line box. Inherited.
	InlineSizing string
	// Line and flex wrap opportunities (css-text-4 lite): "" (auto) or line
	// / avoid / avoid-line / flex / avoid-flex.
	WrapBefore string
	WrapAfter  string
	WrapInside string
}

var initialStyleInlineText = styleInlineText{
	InitialLetterAlign:       "alphabetic",
	InitialLetterWrap:        "none",
	TextTransform:            textTransformNone,
	VerticalAlign:            baselineName,
	TabSize:                  defaultTabSize,
	HyphenateCharacter:       "-",
	HyphenateLimitLast:       cssDisplayNone,
	HyphenateLimitLines:      -1,
	HyphenateLimitZonePct:    -1,
	HangingPunctuation:       cssDisplayNone,
	OverflowWrap:             "normal",
	WordBreak:                "normal",
	TextDecoration:           cssDisplayNone,
	TextCombineUpright:       "none",
	TextDecorationSkip:       "auto",
	TextDecorationSkipBox:    "none",
	TextDecorationSkipSelf:   "auto",
	TextDecorationSkipSpaces: "start end",
	TextOrientation:          "mixed",
	UnicodeBidi:              "normal",
	TextBoxTrim:              cssDisplayNone,
	TextBoxEdgeOver:          "auto",
	TextBoxEdgeUnder:         "auto",
	LineFitEdge:              lineFitEdgeLeading,
	TextAutospace:            "no-autospace",
	TextSpacing:              contentNormal,
	TextSpacingTrim:          "space-all",
	TextGroupAlign:           cssDisplayNone,
	TextFit:                  cssDisplayNone,
	DominantBaseline:         "auto",
	AlignmentBaseline:        "auto",
	BaselineSource:           "auto",
	InlineSizing:             contentNormal,
	WordSpaceTransform:       wordSpaceTransformNone,
}

// styleMotion holds motion CSS values.
type styleMotion struct {
	// Animation and Transition keep declared values until the public payload
	// aligns their comma-separated lists.
	Animation  animationStyle
	Transition transitionStyle
	// Named timeline declarations (CSS Scroll-driven Animations). The names
	// are dashed idents; "" is the initial none. Neither name is inherited.
	// Each axis keeps the declared keyword (block | inline | x | y) and
	// resolves to a physical axis at export, where the writing mode is used.
	// The insets are layout points; a Set flag is false for the auto initial.
	// One element may declare both a scroll and a view timeline.
	ScrollTimelineName        string
	ScrollTimelineAxis        string
	ViewTimelineName          string
	ViewTimelineAxis          string
	ViewTimelineInsetStart    float64
	ViewTimelineInsetEnd      float64
	ViewTimelineInsetStartSet bool
	ViewTimelineInsetEndSet   bool
	// ViewTimelineInsets is the resolved inset of every timeline in the
	// view-timeline-inset list, encoded by encodeViewInsets. The four fields
	// above hold its first item.
	ViewTimelineInsets string
}

var initialStyleMotion = styleMotion{
	ScrollTimelineAxis: timelineAxisBlock,
	ViewTimelineAxis:   timelineAxisBlock,
}

// stylePositioning holds positioning CSS values.
type stylePositioning struct {
	Clear      string  // cssDisplayNone | floatLeft | floatRight | "both"
	Top        float64 // position offsets (pt); 0 = unset for absolute uses Auto flags
	Right      float64
	Bottom     float64
	Left       float64
	TopAuto    bool
	RightAuto  bool
	BottomAuto bool
	LeftAuto   bool
	ZIndex     int
	ZIndexSet  bool
	// WritingMode is CSS writing-mode ("horizontal-tb" | "vertical-rl" | "vertical-lr").
	WritingMode string
	// Direction is CSS direction ("ltr" | "rtl").
	Direction string
	// AnchorNames holds anchor-name dashed-idents; empty means none.
	AnchorNames []string
	// AnchorScope is anchor-scope: "none" | "all" | space-joined names.
	// Empty is the initial (hides nothing, like none).
	AnchorScope string
	// PositionAnchor is position-anchor: "auto" or one dashed-ident.
	// Empty behaves as auto (anchor only when exactly one in-scope name).
	PositionAnchor string
	// PositionArea is position-area: "none" or a 1-2 keyword area.
	// Empty behaves as none (keeps left/top offsets).
	PositionArea string
	// PositionTryFallbacks holds one entry per position-try-fallbacks item
	// (area, dashed-ident with optional try-tactics, or tactics alone).
	// Empty means none.
	PositionTryFallbacks []string
	// PositionTryOrder is position-try-order: "normal" or a try-size
	// keyword. Empty behaves as normal (fallback document order).
	PositionTryOrder string
	// PositionVisibility is position-visibility: "always" or a combo of
	// anchor-valid, anchor-visible, no-overflow. Empty behaves like
	// anchor-valid (anchor-visible is identical until a scroll offset exists).
	PositionVisibility string
	// CSS Shapes (outside wrap) + page-float extras (87.6).
	ShapeOutside       string  // "none" | circle()/ellipse()/inset() canonical
	ShapeMargin        float64 // pt when ShapeMarginPercent < 0
	ShapeMarginPercent float64 // >=0 means % of reference diagonal/√2; -1 = length
	FloatOffset        float64 // pt nudge on placeFloat when FloatOffsetPercent < 0
	FloatOffsetPercent float64 // >=0 means % of float border-box height; -1 = length
	FloatReference     string  // "inline" | "column" | "region" | "page"
	// CSS Shapes inside (v0.0.3 shapes ledger): shape-inside storage plus
	// shape-padding. shape-inside is not inherited, so children keep the
	// initial auto while the element's own inline content carries the contour.
	ShapeInside         string  // "auto" | circle()/ellipse()/inset() canonical
	ShapeInsideBox      string  // "" = content-box reference; margin/border/padding/content-box
	ShapeInsideOutside  bool    // shape-inside: outside-shape copies ShapeOutside
	ShapePadding        float64 // pt when ShapePaddingPercent < 0
	ShapePaddingPercent float64 // >=0 means % of reference diagonal/√2; -1 = length
	// Exclusions (css-exclusions-1 lite). WrapFlow is "" (auto) or both /
	// start / end / minimum / maximum / clear. WrapThrough is "" (wrap) or
	// none (ignore exclusions from outside the element).
	WrapFlow    string
	WrapThrough string
}

var initialStylePositioning = stylePositioning{
	Clear:               cssDisplayNone,
	TopAuto:             true,
	RightAuto:           true,
	BottomAuto:          true,
	LeftAuto:            true,
	WritingMode:         writingModeHorizontalTB,
	Direction:           cssDirectionLTR,
	ShapeOutside:        shapeOutsideNone,
	ShapeMarginPercent:  -1,
	ShapeInside:         shapeInsideAuto,
	ShapePaddingPercent: -1,
	FloatOffsetPercent:  -1,
	FloatReference:      floatRefInline,
}

// styleSvg holds svg CSS values.
type styleSvg struct {
	// ClipRule is the standalone clip-rule fill rule ("nonzero" | "evenodd").
	// Empty means nonzero (the CSS initial). A polygon() with its own inline
	// fill rule keeps that rule; the mask sites use this value otherwise.
	ClipRule                  string
	Fill                      [3]float64
	FillSet                   bool
	FillOpacity               float64
	Stroke                    [3]float64
	StrokeSet                 bool
	StrokeWidth               float64
	StrokeWidthSet            bool
	StrokeOpacity             float64
	StrokeDashArray           []float64
	StrokeDashOffset          float64
	StrokeLineCap             string
	StrokeLineJoin            string
	StrokeMiterLimit          float64
	FillRule                  string
	ShapeRendering            string
	ColorInterpolation        string
	ColorInterpolationFilters string
	// SVG geometry from CSS. Empty means unset. A stylesheet value is baked
	// onto the attribute and replaces the presentation attribute.
	SVGCx string
	SVGCy string
	SVGR  string
	SVGRx string
	SVGRy string
	SVGX  string
	SVGY  string
	SVGD  string
	// SVG paint presentation from CSS. Empty means unset. The marker
	// shorthand is expanded onto the three longhands when stored, so
	// cascade order between the shorthand and a longhand holds. "none" is
	// kept (not empty) so it can clear a presentation attribute.
	SVGMarkerStart  string
	SVGMarkerMid    string
	SVGMarkerEnd    string
	SVGStopColor    string
	SVGStopOpacity  string
	SVGPaintOrder   string
	SVGPathLength   string
	SVGVectorEffect string
	SVGFloodColor   string
	SVGFloodOpacity string
}

var initialStyleSvg = styleSvg{
	FillOpacity:               1,
	StrokeOpacity:             1,
	FillRule:                  "nonzero",
	ShapeRendering:            "auto",
	ColorInterpolation:        "srgb",
	ColorInterpolationFilters: "srgb",
}
