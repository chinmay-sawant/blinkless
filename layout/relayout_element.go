package layout

import (
	"context"
	"errors"
	"fmt"
	"image"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/chinmay-sawant/blinkless/css"
	icss "github.com/chinmay-sawant/blinkless/internal/css"
	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
	ihtml "github.com/chinmay-sawant/blinkless/internal/html"
	ilayout "github.com/chinmay-sawant/blinkless/internal/layout"
	"github.com/chinmay-sawant/blinkless/internal/pubstate"
)

var (
	// ErrNilDisplay means RelayoutElement was given no previous display
	// list to patch.
	ErrNilDisplay = errors.New("layout: nil display")

	// ErrNoElement means RelayoutElement found no element with the given id.
	ErrNoElement = errors.New("layout: no element with that id")

	// ErrNoRetainedState means the display carries no retained layout
	// state, so it cannot be patched. The display was laid out without
	// Options.RetainState, so the engine kept no box tree and no resolved
	// styles to re-place the element inside. Lay the page out with
	// Options.RetainState set and patch that display.
	ErrNoRetainedState = errors.New("layout: display carries no retained layout state")

	// ErrUnknownOverrideProperty means an override named a property the
	// relayout path cannot apply to the element's used style. A custom
	// property, a paint property, a transform, an SVG paint value, and a
	// display or flow property are refused here. The page is left untouched.
	ErrUnknownOverrideProperty = errors.New("layout: unknown override property")

	// ErrInvalidOverrideValue means an override value is empty, is a
	// CSS-wide keyword, contains var(), or is not a value the property's
	// grammar accepts. The page is left untouched.
	ErrInvalidOverrideValue = errors.New("layout: invalid override value")

	// ErrOverrideReflow means the override changed the size of the box the
	// patch would re-place, so the change escapes the containing block the
	// patch covers. Fall back to a full layout for that frame. The page is
	// left untouched.
	ErrOverrideReflow = errors.New("layout: override reflow escapes its containing block")
)

// relayoutFingerprintCap bounds the no-op fingerprint cache: one entry per
// patched element of a retained result. Clearing on overflow keeps one-off
// pages from pinning every display they touched.
const relayoutFingerprintCap = 1024

//nolint:gochecknoglobals // fast-path fingerprint cache and test counters, mutex guarded and bounded
var (
	relayoutEngineCalls int64
	relayoutFastHits    int64
	relayoutFontSkips   int64

	relayoutFingerprintMu sync.Mutex
	relayoutFingerprints  = make(map[relayoutFingerprintKey]string)
)

// relayoutFingerprintKey identifies one patchable element: the retained result
// it patches (shared by an adopted patch display), the styled document holding
// the edited tree, and the element id.
type relayoutFingerprintKey struct {
	result   *ilayout.Result
	document *css.Document
	element  string
}

// ElementPatch is the result of patching one element of a laid out page.
type ElementPatch struct {
	// Ops are the operations the patched element's subtree emits. They are
	// for inspection; the whole page the host draws is Display.
	Ops []DisplayOp

	// OpOwners maps each operation index in Ops to the nearest element box
	// in Boxes that owns it, the rule Display.OpOwners uses.
	OpOwners []int

	// Boxes are the element border boxes of the patched subtree, in document
	// order, in CSS pixels.
	Boxes []Box

	// Old is the element's border box before the patch, in CSS pixels.
	Old image.Rectangle

	// New is the element's border box after the patch, in CSS pixels.
	New image.Rectangle

	// Display is the patched page: the operations, the paint order, the
	// owners and the boxes of the whole page after the patch. A host adopts
	// this display and treats the one it passed as stale, because the
	// engine splices into the retained placement and a count change moves
	// every later operation index, the paint order, and the owners.
	Display *Display
}

// RelayoutElement lays out one element's subtree inside prev and returns the
// operations, boxes and old and new bounds for that element only.
//
// doc is the styled document that holds the element and prev is the display
// list of that document before the change. The element is found by id in
// doc's tree, so a host that edited the tree first (html.Document.SetText,
// html.Document.SetAttribute) gets a patch of the edited element.
//
// prev must carry the retained layout state. The host asks for it when it
// lays the page out: layout.DisplayListOptions with Options.RetainState set.
// A display laid out without that state cannot be patched, and
// RelayoutElement returns ErrNoRetainedState for it.
//
// On success the element keeps its place and its size. A change that would
// move or resize it is refused, so the boxes after it stay valid. The patch
// carries the element's own operations and boxes for inspection (Ops, Boxes,
// and OpOwners), its old and new bounds (Old and New), and Display, the whole
// patched page. A host adopts Display and treats prev as stale.
//
// The engine does not lock the retained state, so one display must not be
// patched from two goroutines at once.
func RelayoutElement(
	ctx context.Context, doc *css.Document, prev *Display, elementID string, options Options,
) (*ElementPatch, error) {
	if err := checkRelayoutInput(ctx, doc, prev, elementID); err != nil {
		return nil, err
	}

	styled, ok := pubstate.StyledOf(doc)
	if !ok {
		return nil, ErrNilDocument
	}

	node := elementByID(styled.Root, elementID)
	if node == nil {
		return nil, ErrNoElement
	}

	if prev.result == nil {
		return nil, ErrNoRetainedState
	}

	// A repeated call with identical relevant inputs hands prev back with no
	// engine work: the edit was an identical value or touched only attributes
	// the cascade ignores, so the display already shows it. Image subtrees
	// skip the check because the image resolver lives outside the fingerprint.
	fingerprint, cacheable := "", false

	if !subtreeHasImage(node) {
		fingerprint = relayoutSubtreeFingerprint(styled, node)

		if relayoutFingerprintHit(prev, doc, elementID, fingerprint) {
			atomic.AddInt64(&relayoutFastHits, 1)

			return &ElementPatch{Display: prev}, nil //nolint:exhaustruct // a no-op patch carries only the display
		}

		cacheable = true
	}

	res, err := subtreeResult(ctx, styled, prev, node, options)
	if err != nil {
		return nil, err
	}

	if cacheable {
		relayoutStoreFingerprint(prev, doc, elementID, fingerprint)
	}

	boxes := boxesFrom(ilayout.PlacedElements(res))

	return &ElementPatch{
		Ops:      res.Ops,
		OpOwners: ilayout.OperationOwners(res),
		Boxes:    boxes,
		Old:      elementBounds(prev.Boxes, elementID),
		New:      elementBounds(boxes, elementID),
		Display:  patchedDisplay(styled, prev),
	}, nil
}

// RelayoutElementWithOverrides lays out one element with a set of property
// values applied on top of the cascade, and returns the patch for that element.
//
// It is the geometry-animation path: a host samples an animated width, a font
// size, or another layout property, hands the values here, and adopts the
// returned display. The values are applied to the element's used style after
// the cascade and before placement, are not written to the parsed tree, and
// are gone when the call returns. The next patch starts from the base page.
//
// An inherited property reaches the element's descendants: the engine resolves
// the element's subtree against the overridden style, so a font-size override
// sizes the child text runs the way a stylesheet declaration would.
//
// The patch target is the element's containing block, not the element alone,
// because a width or font-size change can move the element's siblings. Ops and
// Boxes describe the element's own subtree, the shape RelayoutElement returns;
// Display is the whole page with the reflowed siblings. Old and New are the
// element's border box before and after the call.
//
// prev must carry the retained layout state, the same rule RelayoutElement
// has. An empty override list is RelayoutElement exactly.
//
// An unknown property is ErrUnknownOverrideProperty, a malformed value is
// ErrInvalidOverrideValue, and a change that would resize the containing block
// is ErrOverrideReflow (fall back to a full layout for that frame). Every error
// leaves prev untouched.
//
// The engine does not lock the retained state, so one display must not be
// patched from two goroutines at once.
func RelayoutElementWithOverrides(
	ctx context.Context, doc *css.Document, prev *Display,
	elementID string, overrides []Override, options Options,
) (*ElementPatch, error) {
	if len(overrides) == 0 {
		return RelayoutElement(ctx, doc, prev, elementID, options)
	}

	if err := checkRelayoutInput(ctx, doc, prev, elementID); err != nil {
		return nil, err
	}

	styled, ok := pubstate.StyledOf(doc)
	if !ok {
		return nil, ErrNilDocument
	}

	node := elementByID(styled.Root, elementID)
	if node == nil {
		return nil, ErrNoElement
	}

	if prev.result == nil {
		return nil, ErrNoRetainedState
	}

	// An override that cannot change geometry or paint hands prev back with
	// no engine work, even for a subtree the engine would refuse: there is
	// nothing to re-place, so there is nothing the refusal protects.
	if patch, ok := relayoutInertOverridePatch(prev, styled, node, overrides); ok {
		return patch, nil
	}

	engineOverrides, err := engineStyleOverrides(elementID, overrides)
	if err != nil {
		return nil, err
	}

	layoutOpts, err := relayoutOptions(styled, prev, options, nil)
	if err != nil {
		return nil, err
	}

	res, err := ilayout.RelayoutSubtreeWithOverrides(ctx, prev.result, node, engineOverrides, layoutOpts)
	if err != nil {
		return nil, overrideRelayoutError(err)
	}

	boxes := boxesFrom(ilayout.PlacedElements(res))

	return &ElementPatch{
		Ops:      res.Ops,
		OpOwners: ilayout.OperationOwners(res),
		Boxes:    boxes,
		Old:      elementBounds(prev.Boxes, elementID),
		New:      elementBounds(boxes, elementID),
		Display:  patchedDisplay(styled, prev),
	}, nil
}

// engineStyleOverrides converts the public override list for the engine. An
// override may name the relayout target or leave the id empty; any other id is
// a caller error, not a silently ignored entry.
func engineStyleOverrides(elementID string, overrides []Override) ([]ilayout.StyleOverride, error) {
	converted := make([]ilayout.StyleOverride, len(overrides))

	for index := range overrides {
		override := &overrides[index]

		if override.ElementID != "" && override.ElementID != elementID {
			return nil, fmt.Errorf(
				"%w: override names %q, the relayout target is %q",
				ErrNoElement, override.ElementID, elementID,
			)
		}

		converted[index] = ilayout.StyleOverride{
			Property: override.Property,
			Value:    override.Value,
		}
	}

	return converted, nil
}

// overrideRelayoutError maps the engine's override failures onto the public
// sentinels.
func overrideRelayoutError(err error) error {
	switch {
	case errors.Is(err, ilayout.ErrUnknownOverrideProperty):
		return fmt.Errorf("%w: %w", ErrUnknownOverrideProperty, err)
	case errors.Is(err, ilayout.ErrInvalidOverrideValue):
		return fmt.Errorf("%w: %w", ErrInvalidOverrideValue, err)
	case errors.Is(err, ilayout.ErrOverrideReflow):
		return fmt.Errorf("%w: %w", ErrOverrideReflow, err)
	default:
		return err
	}
}

// patchedDisplay builds the display list of the patched page. The engine
// patched the placement the previous display was built from, so the display
// carries that placement and the operations, the paint order, the owners and
// the boxes that go with them. Page breaks are recomputed because a style
// patch can change them.
func patchedDisplay(styled pubstate.Styled, prev *Display) *Display {
	canvasPt := float64(prev.Height) / ptToPx
	forced, keep := ilayout.PageBreaks(prev.result, canvasPt)

	return newDisplay(prev.result, styled.Sheets, canvasPt,
		forced, keep, prev.zoom, true)
}

// checkRelayoutInput rejects the inputs RelayoutElement cannot work with.
func checkRelayoutInput(ctx context.Context, doc *css.Document, prev *Display, elementID string) error {
	if ctx == nil {
		return ErrNilContext
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("layout: context: %w", err)
	}

	if doc == nil {
		return ErrNilDocument
	}

	if prev == nil {
		return ErrNilDisplay
	}

	if elementID == "" {
		return ErrNoElement
	}

	return nil
}

// subtreeResult asks the engine to place the element's subtree inside the
// layout the previous display list was built from.
func subtreeResult(
	ctx context.Context,
	styled pubstate.Styled,
	prev *Display,
	node *ihtml.Node,
	options Options,
) (*ilayout.Result, error) {
	layoutOpts, err := relayoutOptions(styled, prev, options, node)
	if err != nil {
		return nil, err
	}

	res, err := ilayout.RelayoutSubtree(ctx, prev.result, node, layoutOpts)
	if err != nil {
		return nil, fmt.Errorf("layout: relayout element: %w", err)
	}

	return res, nil
}

// relayoutOptions builds the engine options a patch pass runs with. The
// viewport and the media come from the styled document, and the zoom is the
// one the display was laid out at. A text-free element subtree shapes nothing,
// so the default font load is skipped and the engine falls back to its regular
// face; a nil node means an unknown scope (the override containing block), so
// the font is always loaded there.
func relayoutOptions(styled pubstate.Styled, prev *Display, options Options, node *ihtml.Node) (ilayout.Options, error) {
	atomic.AddInt64(&relayoutEngineCalls, 1)

	var font *pdf.Font

	if node != nil && !subtreeHasText(node) {
		atomic.AddInt64(&relayoutFontSkips, 1)
	} else {
		loaded, err := pdf.DefaultFont()
		if err != nil {
			var empty ilayout.Options

			return empty, fmt.Errorf("layout: default font: %w", err)
		}

		font = loaded
	}

	widthPt, heightPt := displayViewportPt(styled)
	layoutOpts := displayLayoutOptions(styled, font, options.Images, widthPt, heightPt)
	layoutOpts.Zoom = prev.zoom

	return layoutOpts, nil
}

// elementByID returns the element node with the given id in the tree, or nil.
func elementByID(root *ihtml.Node, elementID string) *ihtml.Node {
	if root == nil || elementID == "" {
		return nil
	}

	return root.FindFirst(func(candidate *ihtml.Node) bool {
		return candidate.Type == ihtml.ElementNode && candidate.Attribute("id") == elementID
	})
}

// elementBounds returns the border box of the element with the given id, in
// CSS pixels. A missing element gives the empty rectangle.
func elementBounds(boxes []Box, elementID string) image.Rectangle {
	for i := range boxes {
		if boxes[i].ID == elementID {
			return boxBounds(boxes[i])
		}
	}

	return image.Rectangle{} //nolint:exhaustruct // the zero rectangle marks a missing element
}

// boxBounds converts one element box to a rectangle in CSS pixels.
func boxBounds(box Box) image.Rectangle {
	left := int(math.Round(box.X))
	top := int(math.Round(box.Y))

	return image.Rect(left, top, left+int(math.Round(box.W)), top+int(math.Round(box.H)))
}

// relayoutSubtreeFingerprint hashes the inputs a plain element patch reads:
// the viewport, the media, and the pseudo-class state, plus the subtree text
// and the attributes that can reach the cascade or the paint. An attribute the
// cascade ignores for this subtree (title, aria, or data storage no selector
// names and no attr() call reads) is left out, so editing it keeps the
// fingerprint and the patch returns prev unchanged.
func relayoutSubtreeFingerprint(styled pubstate.Styled, node *ihtml.Node) string {
	var fingerprint strings.Builder

	fingerprint.WriteString(strconv.Itoa(styled.WidthPx))
	fingerprint.WriteByte('|')
	fingerprint.WriteString(strconv.Itoa(styled.HeightPx))
	fingerprint.WriteByte('|')
	fingerprint.WriteString(styled.Media)
	fingerprint.WriteByte('|')
	fingerprint.WriteString(styled.State.Focus)
	fingerprint.WriteByte('|')
	fingerprint.WriteString(styled.State.Hover)
	fingerprint.WriteByte('|')
	fingerprint.WriteString(styled.State.Active)
	fingerprint.WriteByte('|')

	useAttrs := relayoutSheetsUseAttrs(styled.Sheets)
	writeRelayoutSubtree(&fingerprint, node, useAttrs, false)

	return fingerprint.String()
}

// writeRelayoutSubtree hashes one subtree in document order. Text nodes hash
// their content; other nodes hash their tag and their relevant attributes.
// Comments and doctypes hash nothing: they never reach paint.
func writeRelayoutSubtree(b *strings.Builder, node *ihtml.Node, useAttrs, inSVG bool) {
	if node == nil {
		return
	}

	if node.Type == ihtml.TextNode {
		b.WriteString("t:")
		b.WriteString(node.Text)
		b.WriteByte(';')

		return
	}

	if node.Type != ihtml.ElementNode {
		return
	}

	svg := inSVG || node.Name == "svg"
	b.WriteString("e:")
	b.WriteString(node.Name)
	b.WriteByte(';')

	for _, name := range slices.Sorted(maps.Keys(node.Attrs)) {
		if !relayoutAttrRelevant(name, useAttrs, svg) {
			continue
		}

		b.WriteString(name)
		b.WriteByte('=')
		b.WriteString(node.Attrs[name])
		b.WriteByte(';')
	}

	for _, child := range node.Children {
		writeRelayoutSubtree(b, child, useAttrs, svg)
	}
}

// relayoutAttrRelevant reports whether an attribute can reach the cascade or
// the paint of the subtree. Attribute selectors and attr() calls force every
// attribute to count, and inline SVG paint reads any presentation attribute.
// Otherwise only the layout-read attributes count: id, class, inline style,
// image and link targets, form and table geometry, and the two data attributes
// the engine reads. Title, aria, and other data storage stay out, so editing
// them keeps the fingerprint.
func relayoutAttrRelevant(name string, useAttrs, inSVG bool) bool {
	if useAttrs || inSVG {
		return true
	}

	switch strings.ToLower(name) {
	case "id", "class", "style",
		"src", "srcset", "sizes", "href", "alt",
		"width", "height", "value", "type", "min", "max",
		"checked", "selected", "rows", "placeholder",
		"colspan", "rowspan", "start",
		"hidden", "open", "disabled", "readonly", "multiple",
		"data-action", "data-gowk-anon":
		return true
	default:
		return false
	}
}

// relayoutSheetsUseAttrs reports whether any sheet can match on attributes:
// an attribute selector anywhere (including inside :has, :is, and :not), or
// an attr() call in a declaration value. When true the fingerprint keeps
// every attribute; otherwise the ignored ones drop out.
func relayoutSheetsUseAttrs(sheets []*icss.Stylesheet) bool {
	for _, sheet := range sheets {
		if sheet == nil {
			continue
		}

		for _, rule := range sheet.Rules {
			for _, selector := range rule.Selectors {
				if selectorUsesAttrs(selector) {
					return true
				}
			}

			for _, decl := range rule.Decls {
				if strings.Contains(strings.ToLower(decl.Value), "attr(") {
					return true
				}
			}
		}

		for _, keyframes := range sheet.Keyframes {
			for _, frame := range keyframes.Keyframes {
				for _, decl := range frame.Declarations {
					if strings.Contains(strings.ToLower(decl.Value), "attr(") {
						return true
					}
				}
			}
		}
	}

	return false
}

// selectorUsesAttrs reports whether a selector names an attribute, including
// inside its pseudo-class arguments.
func selectorUsesAttrs(selector icss.Selector) bool {
	for _, part := range selector.Parts {
		if len(part.Attrs) > 0 {
			return true
		}

		for _, pseudo := range part.Pseudos {
			if pseudoUsesAttrs(pseudo) {
				return true
			}
		}
	}

	return false
}

// pseudoUsesAttrs reports whether a pseudo-class argument names an attribute.
func pseudoUsesAttrs(pseudo icss.PseudoClass) bool {
	for _, has := range pseudo.Has {
		for _, part := range has.Parts {
			if len(part.Attrs) > 0 {
				return true
			}

			for _, inner := range part.Pseudos {
				if pseudoUsesAttrs(inner) {
					return true
				}
			}
		}
	}

	for _, inner := range pseudo.Is {
		if selectorUsesAttrs(inner) {
			return true
		}
	}

	for _, inner := range pseudo.Not {
		if selectorUsesAttrs(inner) {
			return true
		}
	}

	return false
}

// relayoutFingerprintHit reports whether the fingerprint matches the last
// successful patch of the same retained result, document, and element.
func relayoutFingerprintHit(prev *Display, doc *css.Document, elementID, fingerprint string) bool {
	key := relayoutFingerprintKey{result: prev.result, document: doc, element: elementID}

	relayoutFingerprintMu.Lock()
	defer relayoutFingerprintMu.Unlock()

	previous, ok := relayoutFingerprints[key]

	return ok && previous == fingerprint
}

// relayoutStoreFingerprint remembers the fingerprint of a successful patch.
func relayoutStoreFingerprint(prev *Display, doc *css.Document, elementID, fingerprint string) {
	key := relayoutFingerprintKey{result: prev.result, document: doc, element: elementID}

	relayoutFingerprintMu.Lock()
	defer relayoutFingerprintMu.Unlock()

	if len(relayoutFingerprints) >= relayoutFingerprintCap {
		clear(relayoutFingerprints)
	}

	relayoutFingerprints[key] = fingerprint
}

// subtreeHasText reports whether the subtree holds any text node. The font
// fast path needs it: without text no shaping runs.
func subtreeHasText(node *ihtml.Node) bool {
	if node == nil {
		return false
	}

	for _, child := range node.Children {
		if child.Type == ihtml.TextNode && child.Text != "" {
			return true
		}

		if subtreeHasText(child) {
			return true
		}
	}

	return false
}

// subtreeHasImage reports whether the subtree holds a replaced image element.
// The no-op fingerprint skips those subtrees: the image resolver lives
// outside the fingerprint and could serve new bytes for the same tree.
func subtreeHasImage(node *ihtml.Node) bool {
	if node == nil {
		return false
	}

	if node.Type == ihtml.ElementNode {
		switch node.Name {
		case "img", "video", "canvas", "picture":
			return true
		}
	}

	for _, child := range node.Children {
		if subtreeHasImage(child) {
			return true
		}
	}

	return false
}

// relayoutInertOverridePatch returns prev unchanged when every override is
// provably inert for the element: the property cannot change its geometry or
// paint, so the engine pass would reproduce the display it already has. Any
// other override falls through, and the engine keeps the existing validation
// and refusal errors.
func relayoutInertOverridePatch(
	prev *Display, styled pubstate.Styled, node *ihtml.Node, overrides []Override,
) (*ElementPatch, bool) {
	for index := range overrides {
		if !overrideInertForElement(styled, node, &overrides[index]) {
			return nil, false
		}
	}

	atomic.AddInt64(&relayoutFastHits, 1)

	return &ElementPatch{Display: prev}, true //nolint:exhaustruct // an inert patch carries only the display
}

// overrideInertForElement reports whether one override cannot change the
// geometry or the paint of the element: object-position on a non-replaced
// element, text spacing on a subtree without text, or font size on a subtree
// without text and without relative lengths. A value outside the small valid
// shapes below falls through so the engine keeps its exact validation errors.
func overrideInertForElement(styled pubstate.Styled, node *ihtml.Node, override *Override) bool {
	prop := strings.ToLower(strings.TrimSpace(override.Property))
	value := strings.TrimSpace(override.Value)

	switch prop {
	case "object-position":
		return !isReplacedElement(node) && validObjectPosition(value)
	case "letter-spacing", "word-spacing":
		return !subtreeHasText(node) && validSpacingValue(value)
	case "font-size":
		return !subtreeHasText(node) && !subtreeUsesRelativeUnits(styled, node) && validFontSize(value)
	default:
		return false
	}
}

// isReplacedElement reports whether object-position can paint the element
// itself. Only replaced elements use it; a plain div keeps no replaced
// content, so the override cannot change paint.
func isReplacedElement(node *ihtml.Node) bool {
	if node == nil || node.Type != ihtml.ElementNode {
		return false
	}

	switch node.Name {
	case "img", "video", "canvas", "iframe", "embed", "object", "picture":
		return true
	default:
		return false
	}
}

// subtreeUsesRelativeUnits reports whether the subtree could resolve a
// percentage or a font- or viewport-relative length: an inline style holding
// a relative token, or any sheet declaration holding one. The sheet scan is
// deliberately coarse (any declaration, not just matching ones): a false
// positive only skips the font-size fast path, never claims an unsafe one.
func subtreeUsesRelativeUnits(styled pubstate.Styled, node *ihtml.Node) bool {
	var walk func(current *ihtml.Node) bool

	walk = func(current *ihtml.Node) bool {
		if current == nil {
			return false
		}

		if current.Type == ihtml.ElementNode && hasRelativeToken(current.Attribute("style")) {
			return true
		}

		for _, child := range current.Children {
			if walk(child) {
				return true
			}
		}

		return false
	}

	if walk(node) {
		return true
	}

	for _, sheet := range styled.Sheets {
		if sheet == nil {
			continue
		}

		for _, rule := range sheet.Rules {
			for _, decl := range rule.Decls {
				if hasRelativeToken(decl.Value) {
					return true
				}
			}
		}

		for _, keyframes := range sheet.Keyframes {
			for _, frame := range keyframes.Keyframes {
				for _, decl := range frame.Declarations {
					if hasRelativeToken(decl.Value) {
						return true
					}
				}
			}
		}
	}

	return false
}

// hasRelativeToken reports whether text holds a percentage or a font- or
// viewport-relative length unit.
func hasRelativeToken(text string) bool {
	lower := strings.ToLower(text)

	for _, token := range []string{
		"%",
		"em", "rem", "ex", "rex", "ch", "rch", "cap", "rcap", "ic", "ric", "lh", "rlh",
		"vw", "svw", "lvw", "dvw", "vh", "svh", "lvh", "dvh",
		"vi", "svi", "lvi", "dvi", "vb", "svb", "lvb", "dvb",
		"vmin", "svmin", "lvmin", "dvmin", "vmax", "svmax", "lvmax", "dvmax",
		"cqw", "cqh", "cqi", "cqb", "cqmin", "cqmax",
	} {
		if strings.Contains(lower, token) {
			return true
		}
	}

	return false
}

// validSpacingValue reports whether value is a letter- or word-spacing the
// inert path understands: normal or a pixel length. Anything else falls
// through to the engine.
func validSpacingValue(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))

	if lower == "normal" {
		return true
	}

	return validPixelLength(lower)
}

// validObjectPosition reports whether value is an object-position the inert
// path understands: a single keyword or one or two pixel or percentage
// positions. Anything else falls through to the engine.
func validObjectPosition(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))

	switch lower {
	case "center", "left", "right", "top", "bottom":
		return true
	}

	fields := strings.Fields(lower)

	if len(fields) == 0 || len(fields) > 2 {
		return false
	}

	for _, field := range fields {
		switch field {
		case "center", "left", "right", "top", "bottom":
			continue
		default:
			if !validPixelLength(field) && !validPercentLength(field) {
				return false
			}
		}
	}

	return true
}

// validFontSize reports whether value is a font size the inert path
// understands: an absolute keyword or a pixel length. Anything else falls
// through to the engine.
func validFontSize(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "xx-small", "x-small", "small", "medium", "large", "x-large", "xx-large", "xxx-large":
		return true
	default:
		return validPixelLength(strings.ToLower(strings.TrimSpace(value)))
	}
}

// validPixelLength reports whether value is a plain pixel length.
func validPixelLength(value string) bool {
	number, ok := strings.CutSuffix(value, "px")
	if !ok {
		return false
	}

	number = strings.TrimSpace(number)

	if number == "" {
		return false
	}

	parsed, err := strconv.ParseFloat(number, 64)

	return err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0)
}

// validPercentLength reports whether value is a plain percentage.
func validPercentLength(value string) bool {
	number, ok := strings.CutSuffix(value, "%")
	if !ok {
		return false
	}

	number = strings.TrimSpace(number)

	if number == "" {
		return false
	}

	parsed, err := strconv.ParseFloat(number, 64)

	return err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0)
}
