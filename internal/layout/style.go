//go:generate go run ../../scripts/gen-style-intern -dir . -out style_intern_gen.go
package layout

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
	"github.com/chinmay-sawant/blinkless/internal/style"
)

// CSS keyword constants shared by the cascade (goconst).
const (
	inheritKeyword                = "inherit"
	solidKeyword                  = "solid"
	clearKeyword                  = "clear"
	visibleKeyword                = "visible"
	pageKeyword                   = "page"
	avoidKeyword                  = "avoid"
	avoidPageValue                = "avoid-page"
	avoidColumnValue              = "avoid-column"
	remUnit                       = "rem"
	divElementName                = "div"
	styleElement                  = "style"
	borderWidthKeyword            = "border-width"
	borderStyleKeyword            = "border-style"
	borderColorKeyword            = "border-color"
	gapKeyword                    = "gap"
	containerKeyword              = "container"
	flexKeyword                   = "flex"
	flexStartKeyword              = "flex-start"
	cssPropMarginInline           = "margin-inline"
	cssPropMarginBlock            = "margin-block"
	cssPropPaddingInline          = "padding-inline"
	cssPropPaddingBlock           = "padding-block"
	cssPropInsetBlock             = "inset-block"
	cssPropInsetInline            = "inset-inline"
	cssPropInsetBlockStart        = "inset-block-start"
	cssPropInsetBlockEnd          = "inset-block-end"
	cssPropInsetInlineStart       = "inset-inline-start"
	cssPropInsetInlineEnd         = "inset-inline-end"
	cssPropBorderBlock            = "border-block"
	cssPropBorderInline           = "border-inline"
	cssPropBorderBlockStart       = "border-block-start"
	cssPropBorderBlockEnd         = "border-block-end"
	cssPropBorderInlineStart      = "border-inline-start"
	cssPropBorderInlineEnd        = "border-inline-end"
	cssPropBorderBlockColor       = "border-block-color"
	cssPropBorderInlineColor      = "border-inline-color"
	cssPropBorderBlockStartColor  = "border-block-start-color"
	cssPropBorderBlockEndColor    = "border-block-end-color"
	cssPropBorderInlineStartColor = "border-inline-start-color"
	cssPropBorderInlineEndColor   = "border-inline-end-color"
	cssPropBorderBlockStyle       = "border-block-style"
	cssPropBorderInlineStyle      = "border-inline-style"
	cssPropBorderBlockStartStyle  = "border-block-start-style"
	cssPropBorderBlockEndStyle    = "border-block-end-style"
	cssPropBorderInlineStartStyle = "border-inline-start-style"
	cssPropBorderInlineEndStyle   = "border-inline-end-style"
	cssPropBorderBlockWidth       = "border-block-width"
	cssPropBorderInlineWidth      = "border-inline-width"
	cssPropBorderBlockStartWidth  = "border-block-start-width"
	cssPropBorderBlockEndWidth    = "border-block-end-width"
	cssPropBorderInlineStartWidth = "border-inline-start-width"
	cssPropBorderInlineEndWidth   = "border-inline-end-width"
	cssDirectionLTR               = "ltr"
	cssDirectionRTL               = "rtl"
)

// hashFontFamily fingerprints a CSS font-family list without allocating.
func hashFontFamily(fams []string) uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)

	hash := uint64(offset64)

	for _, fam := range fams {
		for i := range len(fam) {
			hash ^= uint64(fam[i])
			hash *= prime64
		}

		hash ^= 0xff // token separator
		hash *= prime64
	}

	return hash
}

// fontWeightStep is the bolder/lighter adjustment applied to the current
// weight (CSS Fonts 3 §3.3; clamped by the 100..900 numeric range).
const fontWeightStep = 100

// asciiFoldBit is the single-bit mask that lowercases an ASCII letter
// (s[i]|asciiFoldBit maps 'A'-'Z' to 'a'-'z').
const asciiFoldBit = 0x20

// ResolvedStyle is the used style of one element: values the layout engine
// consumes, in points (or unitless where noted). The record keeps common layout fields inline. Optional CSS groups
// share their initial values until a property needs different storage.
type ResolvedStyle struct {
	Display       string
	Position      string  // "static" | "relative" | "absolute" | "fixed" | "sticky"
	Float         string  // cssDisplayNone | floatLeft | floatRight
	Width         float64 // -1 = auto; absolute length in pt when WidthPercent < 0
	WidthPercent  float64 // >=0 means width is that % of the containing block at layout time
	Height        float64 // -1 = auto; absolute length in pt when HeightPercent < 0
	HeightPercent float64 // >=0 means height is that % of the CB; indefinite CB → auto (cyclic honesty)
	Overflow      string  // "visible" | "hidden" | "scroll" | "auto" | "clip" (non-visible = sticky scrollport)
	// Visibility is CSS visibility: "visible" | "hidden" | "collapse".
	// Empty means visible. hidden and collapse skip paint and keep layout size.
	Visibility    string
	MarginTop     float64
	MarginRight   float64
	MarginBottom  float64
	MarginLeft    float64
	PaddingTop    float64
	PaddingRight  float64
	PaddingBottom float64
	PaddingLeft   float64
	Color         [3]float64
	BGColor       [4]float64 // rgba, 0..1
	FontFamily    []string
	// famHash is the FNV-1a fingerprint of FontFamily, computed once during
	// style resolution. Text measurement reuses
	// it instead of re-hashing the family list per run.
	famHash            uint64
	FontSize           float64 // pts
	FontWeight         int
	FontItalic         bool
	LineHeight         float64 // pts; 0 = "normal"
	LineHeightUnitless float64 // multiplier when line-height was unitless; 0 otherwise
	TextAlign          string  // floatLeft | floatRight | "center" | "justify"
	WhiteSpace         string  // "normal" | "nowrap" | "pre" | "pre-wrap" | "pre-line"
	IsReplaced         bool    // img, hr
	Opacity            float64 // 0..1; initial 1; also from filter:opacity()
	// CustomProps holds resolved CSS custom properties (--*) for this element
	// (inherited). Shared with the parent map when the element declares none.
	CustomProps map[string]string
	// Triggers holds timeline-scope, the pointer timeline, the timeline and
	// event triggers, interest-delay, and interactivity as normalized
	// strings. Nil when unset; allocated lazily on first write.
	Triggers *triggerStyle
	// Declared holds the properties the engine stores and exports for a host
	// without acting on them. Nil when unset; allocated lazily on first write.
	Declared *declaredStyle

	// The group accessors distinguish a nil group (Go zero values) from the
	// shared CSS defaults. Use edit accessors for writes to preserve copies.
	backgroundGroup    *styleBackground    `stylegroup:"background"`
	bordersGroup       *styleBorders       `stylegroup:"borders"`
	columnsGroup       *styleColumns       `stylegroup:"columns"`
	containmentGroup   *styleContainment   `stylegroup:"containment"`
	contentGroup       *styleContent       `stylegroup:"content"`
	effectsGroup       *styleEffects       `stylegroup:"effects"`
	flexGroup          *styleFlex          `stylegroup:"flex"`
	fontGroup          *styleFont          `stylegroup:"font"`
	fragmentationGroup *styleFragmentation `stylegroup:"fragmentation"`
	geometryGroup      *styleGeometry      `stylegroup:"geometry"`
	gridGroup          *styleGrid          `stylegroup:"grid"`
	imagesGroup        *styleImages        `stylegroup:"images"`
	inlineTextGroup    *styleInlineText    `stylegroup:"inlineText"`
	motionGroup        *styleMotion        `stylegroup:"motion"`
	positioningGroup   *stylePositioning   `stylegroup:"positioning"`
	svgGroup           *styleSvg           `stylegroup:"svg"`
	groups             *styleGroupBuilder  `styleintern:"-"`
}

const (
	widthMaxContent = -2.0
	widthMinContent = -3.0
)

func isIntrinsicWidth(width float64) bool {
	return width == widthMaxContent || width == widthMinContent
}

type border struct {
	Width      float64 // layout width in CSS points, retained for pagination geometry
	PaintWidth float64 // device paint width; zero means use Width
	Style      string  // cssDisplayNone | "solid" | "dashed" | "dotted"
	Color      [3]float64
	// Transparent marks a fully transparent color: layout keeps its width,
	// paint emits nothing. Zero value false keeps every other border opaque.
	Transparent bool
}

// initialStyle returns the CSS initial values.
func initialStyle() ResolvedStyle {
	return ResolvedStyle{
		Display:            "inline",
		Position:           "static",
		Float:              cssDisplayNone,
		Width:              -1,
		WidthPercent:       -1,
		Height:             -1,
		HeightPercent:      -1,
		Overflow:           "visible",
		Visibility:         visibleKeyword,
		Color:              [3]float64{0, 0, 0},
		BGColor:            [4]float64{0, 0, 0, 0},
		FontFamily:         nil,
		famHash:            hashFontFamily(nil),
		FontSize:           12,
		FontWeight:         400,
		WhiteSpace:         "normal",
		Opacity:            1,
		backgroundGroup:    &initialStyleBackground,
		bordersGroup:       &initialStyleBorders,
		columnsGroup:       &initialStyleColumns,
		containmentGroup:   &initialStyleContainment,
		contentGroup:       &initialStyleContent,
		effectsGroup:       &initialStyleEffects,
		flexGroup:          &initialStyleFlex,
		fontGroup:          &initialStyleFont,
		fragmentationGroup: &initialStyleFragmentation,
		geometryGroup:      &initialStyleGeometry,
		gridGroup:          &initialStyleGrid,
		imagesGroup:        &initialStyleImages,
		inlineTextGroup:    &initialStyleInlineText,
		motionGroup:        &initialStyleMotion,
		positioningGroup:   &initialStylePositioning,
		svgGroup:           &initialStyleSvg,
	}
}

// defaultTextStyle covers a text node without an element parent. Ordinary
// text nodes reuse their parent's style because text has no declarations of
// its own and the inline layout path only consumes inherited text properties.
var defaultTextStyle = initialStyle()

// styleContext carries per-element resolution inputs.
type styleContext struct {
	ctx       context.Context
	err       error
	work      uint32
	sheets    []*css.Stylesheet
	media     string
	viewportW float64 // containing-block width for % of margins/padding/width
	viewportH float64 // for % of height
	// properties holds @property registrations from sheets for custom-property
	// initial values and inheritance.
	properties map[string]css.PropertyRule
	// state carries the focused, hovered, and pressed ids for stateful
	// pseudo-classes.
	state css.MatchState
	// remBase is the used font-size of the root element for rem units (pt).
	// 0 means the CSS initial medium size (16px → 12pt).
	remBase float64
	// printLinkUnderline is the opt-in --print-link-underline operator policy.
	printLinkUnderline bool
	// containers maps size-query containers (inline-size|size) to their used
	// content-box inline size. nil means first pass: skip @container rules.
	containers map[*html.Node]sizeContainer
	// ruleHits is reused between sequential cascade lookups. A lookup consumes
	// the returned slice before the next element is resolved.
	ruleHits []ruleHit
	// ruleCands is the candidate selector buffer that feeds ruleHits. It is
	// reused between sequential lookups like ruleHits.
	ruleCands []style.RuleSel
	// index narrows selector matching to rules that can match the node. It is
	// built once per pass from sheets; a pass with no sheets leaves it nil.
	index *style.RuleIndex
	// Cascade maps are reused between sequential element lookups. Their values
	// are consumed before the next element is resolved.
	cascadeWins  map[string]cascadeWin
	cascadeProps map[string]string
	// memo caches element resolutions for repeated declaration shapes. It is
	// pass-local: every resolution pass builds a fresh styleContext, and
	// container re-cascade passes do not use it.
	memo styleResolutionMemo
	// matchCache memoizes element-local selector outcomes per element
	// storeChunks collects the style chunk backings allocated during this
	// pass so they can be transferred to the engine and Result for pooling.
	storeChunks [][]ResolvedStyle
	// signature for this pass (see matchedRules). A container re-cascade
	// pass reuses it: entries only cover selectors whose outcome is a pure
	// function of the signature, and container-gated rules evaluate fresh.
	matchCache map[matchCacheKey]*matchCacheEntry
	// matchTags interns the namespaced tag strings matchSignature builds, so
	// repeated probes of the same tag share one copy for this pass.
	matchTags map[string]string
	// classKeys interns the canonical single-class attribute key for this
	// pass. Tables repeat a handful of classes across thousands of elements,
	// so the pass-local map avoids taking the shared cache lock per element.
	classKeys map[string]string
	// gateCache memoizes the node-independent rule gates (media query and
	// @supports) per rule for this pass, so repeated elements do not
	// re-parse the same query strings.
	gateCache map[*css.Rule]bool
}

// pollContext checks cancellation at bounded work intervals. Style matching is
// intentionally hot, so the interval avoids a context lookup for every CSS
// declaration while keeping cancellation latency proportional to a small
// amount of cascade work rather than the complete document.
func (ctx *styleContext) pollContext() bool {
	if ctx == nil || ctx.err != nil || ctx.ctx == nil {
		return ctx != nil && ctx.err != nil
	}

	ctx.work++
	if ctx.work&63 != 0 {
		return false
	}

	if err := ctx.ctx.Err(); err != nil {
		ctx.err = err

		return true
	}

	return false
}

// sizeContainer is one element that establishes a size query container.
type sizeContainer struct {
	inlineSize float64 // content-box inline size in pt
	fontSize   float64 // used font-size (em base for query lengths)
	names      string  // space-separated container-name values
}

// sameSizeContainerState reports whether a second container measurement is
// equivalent to the previous one. Container queries can use em lengths, so a
// changed used font-size is just as significant as a changed inline size or
// name. Keeping this comparison here gives the convergence loop one policy
// for deciding whether a second style pass is required.
func sameSizeContainerState(a, b sizeContainer) bool {
	return nearlyEqual(a.inlineSize, b.inlineSize) &&
		nearlyEqual(a.fontSize, b.fontSize) &&
		a.names == b.names
}

func nearlyEqual(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9
}

// resolveStylesWith is the single cascade entry: Options + optional size
// containers for @container rules (nil = first pass, skip container queries).
// Values are heap-backed once; the engine reuses these pointers (no second copy).
func resolveStylesWith(
	root *html.Node, opts Options, containers map[*html.Node]sizeContainer,
) map[*html.Node]*ResolvedStyle {
	styles, _ := resolveStylesWithContext(context.Background(), root, opts, containers)

	return styles
}

func resolveStylesWithContext(
	ctx context.Context, root *html.Node, opts Options, containers map[*html.Node]sizeContainer,
) (map[*html.Node]*ResolvedStyle, error) {
	styles, _, _, err := resolveStylesWithContextChunks(ctx, root, opts, containers)

	return styles, err
}

func resolveStylesWithContextChunks(
	ctx context.Context, root *html.Node, opts Options, containers map[*html.Node]sizeContainer,
) (map[*html.Node]*ResolvedStyle, [][]ResolvedStyle, int, error) {
	sc := &styleContext{
		ctx:                ctx,
		sheets:             opts.Sheets,
		properties:         registeredProperties(opts.Sheets),
		media:              opts.Media,
		viewportW:          opts.Width,
		viewportH:          opts.Height,
		printLinkUnderline: opts.PrintLinkUnderline,
		containers:         containers,
		state:              opts.State,
	}

	styles, elemCount, err := resolveStylesCtxCount(root, sc)
	sc.releaseMatchCache()
	if err != nil {
		return nil, nil, 0, err
	}

	return styles, sc.storeChunks, elemCount, nil
}

// resolveStyles walks the tree top-down (test helper; no operator policies).
// @container rules are ignored on this first pass (no used sizes yet).
func resolveStyles(
	root *html.Node, sheets []*css.Stylesheet, media string, viewportW, viewportH float64,
) map[*html.Node]*ResolvedStyle {
	return resolveStylesWith(root, Options{
		Sheets: sheets, Media: media, Width: viewportW, Height: viewportH,
	}, nil)
}

// resolveStylesWithContainers is the second style pass: @container rules are
// applied when their query matches the nearest eligible ancestor in containers.
// Test helper; media/viewport always come from the caller's fixture.
func resolveStylesWithContainers(
	root *html.Node,
	sheets []*css.Stylesheet,
	media string, //nolint:unparam // test helper: media fixed per call site
	viewportW, viewportH float64, //nolint:unparam // test helper: viewport fixed per call site
	containers map[*html.Node]sizeContainer,
) map[*html.Node]*ResolvedStyle {
	return resolveStylesWith(root, Options{
		Sheets: sheets, Media: media, Width: viewportW, Height: viewportH,
	}, containers)
}

func resolveStylesCtx(root *html.Node, ctx *styleContext) (map[*html.Node]*ResolvedStyle, error) {
	out, _, err := resolveStylesCtxCount(root, ctx)

	return out, err
}

func resolveStylesCtxCount(root *html.Node, ctx *styleContext) (map[*html.Node]*ResolvedStyle, int, error) {
	nodeCount, elemCount, err := countStyleNodesContext(ctx.ctx, root)
	if err != nil {
		return nil, 0, err
	}

	out := make(map[*html.Node]*ResolvedStyle, nodeCount)
	store := styleStore{}

	var walk func(n *html.Node, parent *ResolvedStyle)
	walk = func(node *html.Node, parent *ResolvedStyle) {
		if ctx.pollContext() {
			return
		}

		var sty *ResolvedStyle

		switch node.Type {
		case html.ElementNode:
			sty = resolveElementStyleMemo(node, ctx, parent, &store)
		case html.TextNode:
			sty = parent
			if sty == nil {
				sty = &defaultTextStyle
			}
		case html.CommentNode, html.DoctypeNode:
			// No style resolution; store a shared zero so map lookups stay non-nil.
			sty = &zeroResolvedStyle
		}

		out[node] = sty

		// Text nodes have no declarations of their own. Pointing them at the
		// already-resolved parent avoids allocating another full style copy per
		// text-bearing element while preserving inherited text properties.
		for _, child := range node.Children {
			if child.Type == html.TextNode {
				out[child] = sty

				continue
			}

			walk(child, sty)
		}
	}
	walk(root, nil)

	if ctx.err != nil {
		return nil, 0, ctx.err
	}

	if ctx != nil {
		ctx.storeChunks = append(ctx.storeChunks, store.chunks...)
	}

	return out, elemCount, nil
}

// countStyleNodes counts the nodes needed for the result map. ResolvedStyle
// values are allocated lazily by styleStore, so an element count is no longer
// needed to reserve one full style record per element.
func countStyleNodesContext(ctx context.Context, root *html.Node) (int, int, error) {
	nodeCount := 0
	elemCount := 0
	visited := 0

	var walk func(*html.Node) error
	walk = func(node *html.Node) error {
		visited++
		if visited&63 == 0 && ctx != nil {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("layout: style node count: %w", err)
			}
		}

		nodeCount++
		if node.Type == html.ElementNode {
			elemCount++
		}

		for _, child := range node.Children {
			if child.Type == html.TextNode {
				nodeCount++

				continue
			}

			if err := walk(child); err != nil {
				return err
			}
		}

		return nil
	}

	if root == nil {
		return 0, 0, nil
	}
	if err := walk(root); err != nil {
		return 0, 0, err
	}

	return nodeCount, elemCount, nil
}

// styleStoreChunkSize keeps resolved styles in small stable backing arrays. A
// chunk's capacity is never exceeded, so pointers returned from append stay
// valid even when later chunks are appended.
const styleStoreChunkSize = 64

// styleStoreFirstChunk is the capacity of the first chunk. The chunks grow
// from there to styleStoreChunkSize, so a page with a few distinct styles
// keeps a small backing array instead of a full chunk of unused records.
const styleStoreFirstChunk = 8

// styleStore chunk backings are pooled per capacity so an 8-cap backing can
// never serve a 16-cap request and vice versa. The 16-cap step keeps using
// the shared engine pool (its producer is full-layout spare chunks); the
// 8/32/64-cap steps use these segregated lists because the shared pool only
// retains defaultStyleChunkSize backings. Each list only holds backings with
// its exact capacity, and both get and put verify the capacity before reuse.
const (
	maxStyleStoreChunksFree8  = 8
	maxStyleStoreChunksFree32 = 4
	maxStyleStoreChunksFree64 = 2
)

var (
	styleStoreChunkFree8  = make(chan *[]ResolvedStyle, maxStyleStoreChunksFree8)
	styleStoreChunkFree32 = make(chan *[]ResolvedStyle, maxStyleStoreChunksFree32)
	styleStoreChunkFree64 = make(chan *[]ResolvedStyle, maxStyleStoreChunksFree64)
)

// drainStyleStoreChunkPools drains all segregated style store chunk channels
// and clears their buffers so no lingering pointers remain.
func drainStyleStoreChunkPools() {
	drain := func(free chan *[]ResolvedStyle) {
		for {
			select {
			case chunk := <-free:
				if chunk != nil {
					clear((*chunk)[:cap(*chunk)])
					*chunk = (*chunk)[:0]
				}
			default:
				return
			}
		}
	}

	drain(styleStoreChunkFree8)
	drain(styleStoreChunkFree32)
	drain(styleStoreChunkFree64)
}

// getStyleStoreChunk returns a pooled backing with exactly cap want, or nil
// when the per-size list is empty. A backing with any other capacity is never
// served: it is routed back to its own list and the caller allocates fresh.
func getStyleStoreChunk(want int) *[]ResolvedStyle {
	var free chan *[]ResolvedStyle

	switch want {
	case styleStoreFirstChunk:
		free = styleStoreChunkFree8
	case defaultStyleChunkSize:
		spare := getStyleChunk()
		if spare == nil || cap(*spare) != want {
			if spare != nil {
				putStyleChunk(spare)
			}

			return nil
		}

		*spare = (*spare)[:0]

		return spare
	case styleStoreFirstChunk * 4:
		free = styleStoreChunkFree32
	case styleStoreChunkSize:
		free = styleStoreChunkFree64
	default:
		return nil
	}

	select {
	case chunk := <-free:
		if chunk == nil || cap(*chunk) != want {
			if chunk != nil {
				putStyleStoreChunk(chunk)
			}

			return nil
		}

		*chunk = (*chunk)[:0]

		return chunk
	default:
		return nil
	}
}

// putStyleStoreChunk retains chunk in its capacity's list. Backings with a
// capacity outside the growth schedule are dropped, and a full list drops
// the return, so one large patch cannot pin memory for later small ones.
// A 16-cap backing belongs to the shared engine pool and is returned there.
func putStyleStoreChunk(chunk *[]ResolvedStyle) {
	if chunk == nil {
		return
	}

	var free chan *[]ResolvedStyle

	switch cap(*chunk) {
	case styleStoreFirstChunk:
		free = styleStoreChunkFree8
	case styleStoreFirstChunk * 4:
		free = styleStoreChunkFree32
	case styleStoreChunkSize:
		free = styleStoreChunkFree64
	case defaultStyleChunkSize:
		putStyleChunk(chunk)

		return
	default:
		return
	}

	clear((*chunk)[:cap(*chunk)])
	*chunk = (*chunk)[:0]

	select {
	case free <- chunk:
	default:
	}
}

// styleStore owns resolved styles for one resolution pass. It deliberately
// does not cross Layout calls or @container re-cascade passes.
//
// Stored styles are immutable after insertion. append interns an exact
// duplicate (styleInternFingerprint bucket plus styleInternEqual verification)
// instead of storing another copy, so repeated table cells and inherited text
// styles share one 6104-byte record. The fingerprint is only a bucket key; the
// equality check is the sharing decision.
type styleStore struct {
	candidate ResolvedStyle
	chunks    [][]ResolvedStyle
	// intern maps a candidate fingerprint to stored records. Buckets stay
	// small because most documents repeat a few dozen distinct styles.
	intern map[uint64][]*ResolvedStyle
	// memo caches (parent, nodeName, inline, remBase, linkUnderline, hits) ->
	// *ResolvedStyle across pooled subtree resolution passes for the same
	// document tag.
	memo styleResolutionMemo
}

func (s *styleStore) append(style ResolvedStyle) *ResolvedStyle {
	fingerprint := styleInternFingerprint(&style)

	for _, stored := range s.intern[fingerprint] {
		if styleInternEqual(stored, &style) {
			return stored
		}
	}

	if len(s.chunks) == 0 || len(s.chunks[len(s.chunks)-1]) == cap(s.chunks[len(s.chunks)-1]) {
		s.chunks = append(s.chunks, s.newChunk())
	}

	chunk := len(s.chunks) - 1
	s.chunks[chunk] = append(s.chunks[chunk], style)
	stored := &s.chunks[chunk][len(s.chunks[chunk])-1]

	if s.intern == nil {
		s.intern = make(map[uint64][]*ResolvedStyle)
	}

	s.intern[fingerprint] = append(s.intern[fingerprint], stored)

	return stored
}

// nextChunkSize returns the capacity of the next chunk. The chunks grow from
// styleStoreFirstChunk to styleStoreChunkSize, so the backing arrays stay
// close to the number of distinct styles the page holds.
func (s *styleStore) nextChunkSize() int {
	size := styleStoreFirstChunk
	for range s.chunks {
		size *= 2
	}

	if size > styleStoreChunkSize {
		size = styleStoreChunkSize
	}

	return size
}

// newChunk returns the backing for the next style chunk. The 16-cap step
// reuses the shared engine pool the mark/reset scopes recycle through, so a
// patch resolution reuses a backing a full layout already released. The
// 8/32/64-cap steps reuse their segregated per-size lists, so a patch reuses
// a backing a dropped store returned instead of allocating one. Every list
// only serves its exact capacity with a cap check on both get and put, and
// the first-chunk-doubling schedule is the pointer-stability contract the
// chunk test pins.
func (s *styleStore) newChunk() []ResolvedStyle {
	want := s.nextChunkSize()
	if spare := getStyleStoreChunk(want); spare != nil {
		return (*spare)[:0]
	}

	return make([]ResolvedStyle, 0, want)
}

// zeroResolvedStyle is the empty style for comment/doctype nodes (shared).
var zeroResolvedStyle ResolvedStyle

// applyRawToUsed turns a cascade raw map into a used ResolvedStyle. Element
// and pseudo-element resolution share this sequence so neither path can skip
// custom-property inheritance or var() substitution: inheritProps copies
// inherited properties, mergeCustomProps folds in the node's custom
// properties, resolveRawVars substitutes var() references, then the font,
// remaining, and unitless line-height passes run. node is nil for generated
// content (no html rem-base update).
func applyRawToUsed(
	node *html.Node, ctx *styleContext, parent *ResolvedStyle, sty *ResolvedStyle, raw map[string]string,
) {
	*sty = initialStyle()
	sty.beginStyleGroups()

	// Inherit sentinels from all are removed first so those properties copy.
	// Initial sentinels stay through the copy, which is what blocks the
	// parent color, and are dropped before any applier can read them.
	bgInherit := takeAllInheritSentinels(raw)

	var parentProps map[string]string

	if parent != nil {
		inheritProps(sty, parent, raw)
		parentProps = parent.CustomProps

		if bgInherit {
			sty.BGColor = parent.BGColor
		}
	}

	dropAllInitialSentinels(raw)

	var registered map[string]css.PropertyRule
	if ctx != nil {
		registered = ctx.properties
	}

	sty.CustomProps = mergeCustomProps(parentProps, raw, registered)
	raw = resolveRawVars(raw, sty.CustomProps)

	parentSize := sty.FontSize
	if parent != nil {
		parentSize = parent.FontSize
	}

	applyFontProps(sty, raw, parentSize, ctx)

	if node != nil && node.Name == "html" && sty.FontSize > 0 && ctx != nil {
		ctx.remBase = sty.FontSize
	}

	applyRestProps(sty, raw, ctx, parent)

	if _, declared := raw[stringSetPropName]; !declared {
		forgetInheritedStringSet(sty)
	}

	inheritUnitlessLineHeight(sty, parent, raw)
	// FontFamily is final here (inherited, or parsed by applyFontProps /
	// parseFontShorthand); fingerprint it once so inline text measurement
	// does not re-hash the family list per run.
	sty.famHash = hashFontFamily(sty.FontFamily)
	sty.finishStyleGroups()
}

// resolveElementStyle cascades one element: the shared raw-to-used sequence
// plus the operator and blockify policies.
func resolveElementStyle(
	node *html.Node, ctx *styleContext, parent, sty *ResolvedStyle,
) {
	resolveElementStyleWithHits(node, ctx, parent, sty, matchedElementHits(ctx, node))
}

// matchedElementHits returns the author-rule matches for node. Split out so
// the memo path can build its key from the same list the cascade consumed.
func matchedElementHits(ctx *styleContext, node *html.Node) []ruleHit {
	if ctx == nil {
		return nil
	}

	return ctx.matchedRules(node, "")
}

// resolveElementStyleWithHits is resolveElementStyle with the matched rules
// already computed.
func resolveElementStyleWithHits(
	node *html.Node, ctx *styleContext, parent, sty *ResolvedStyle, hits []ruleHit,
) {
	raw := cascadeRaw(ctx, node, hits)
	applyRawToUsed(node, ctx, parent, sty, raw)
	applyDarkFormControlDefaults(node, sty, raw)

	// Opt-in operator policy (--print-link-underline): underline
	// anchors with href after the cascade. Default off so author CSS
	// (including text-decoration: inherit from the parent) wins otherwise.
	if ctx != nil && ctx.printLinkUnderline && node.Name == "a" && strings.TrimSpace(node.Attribute("href")) != "" {
		sty.editInlineText().TextDecoration = cssTextDecorationUnderline
	}
	// CSS2.1 §9.7: float ≠ none blockifies table-internal / inline
	// displays before layout (table/flex/grid stay). Floated <table>
	// keeps display:table so fixture-29 wrapper packing still works.
	if sty.Float != cssDisplayNone {
		sty.Display = blockifyDisplayForFloat(sty.Display)
	}
}

// hasExplicitLineHeight reports whether a declaration sets line-height either
// directly or through a font shorthand containing a slash value.
func hasExplicitLineHeight(raw map[string]string) bool {
	if _, ok := raw["line-height"]; ok {
		return true
	}

	font, ok := raw["font"]

	return ok && strings.Contains(font, "/")
}

func inheritUnitlessLineHeight(sty, parent *ResolvedStyle, raw map[string]string) {
	if parent == nil || hasExplicitLineHeight(raw) || parent.LineHeightUnitless <= 0 {
		return
	}

	sty.LineHeightUnitless = parent.LineHeightUnitless
	sty.LineHeight = sty.FontSize * sty.LineHeightUnitless
}
