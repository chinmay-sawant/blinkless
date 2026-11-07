package layout

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/errs"
	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// The subtree splice reports every case it cannot represent as an error. A
// wrong display list is worse than no display list: the host would paint
// stale pixels over a page it believes is correct.
var (
	errNoPreviousLayout    = errors.New("layout: no previous layout to patch")
	errElementNotPlaced    = errors.New("layout: element has no box")
	errNilElementNode      = errors.New("layout: nil element node")
	errSubtreeNotSplicable = errors.New("layout: element subtree cannot be spliced")
	errSubtreeResized      = errors.New("layout: element subtree changed size")
)

// documentNodeName is the name the parser gives the synthetic root node.
const documentNodeName = "#document"

// relayoutEpsilon is the tolerance of the geometry check. The subtree is laid
// out by the same code with the same inputs, so the numbers match exactly;
// the tolerance only absorbs the last bit of a float comparison.
const relayoutEpsilon = 1e-6

// RelayoutSubtree re-lays out one element's subtree inside prev and returns
// the patched result. The rest of the tree keeps its boxes and its ops.
//
// prev is patched in place. The element's operations are spliced over the
// range its box owned, the operations after that range shift by the change in
// count, the box is replaced in the tree, and the ancestor ranges are
// re-unioned over their children. Nothing else in prev is laid out again, so
// the boxes and the operations of the rest of the document are the ones the
// page already had.
//
// The returned Result carries the patched subtree on its own: its operations,
// its boxes, and the operation ranges that point into that slice. A caller
// repaints that region and leaves the rest of the page alone.
//
// The element is first laid out in the width the kept box carries, so a grid
// item is placed in its own track and a flex item in its own line. That width
// is the kept one, so a width the content drives is not seen as a change.
//
// The height is the check that catches a content change, and a size change
// widens the patch instead of refusing it: the element's parent is re-placed
// instead, repeating up the tree until an ancestor keeps its size. The one
// exception is a block size the containing block assigned: a flex container
// gives an item its used main or cross size before the item is built, and
// that size can be larger than the item's content. The kept style carries the
// assignment, so the build carries the kept height as a definite size and
// the content change check stays in force for every other element.
//
// Out-of-flow boxes need a containing block inside the subtree, or the known
// viewport for viewport-fixed boxes. Floats, anchored elements, named-flow
// regions, a document using @container rules, and a size change that reaches
// the document root are refused: no wider range can place them either.
func RelayoutSubtree(ctx context.Context, prev *Result, node *html.Node, opts Options) (*Result, error) {
	if err := relayoutGuards(ctx, prev, node, opts); err != nil {
		return nil, err
	}

	target := findRelayoutTarget(prev.boxTreeRoot(), node)
	if target == nil {
		// A box-less element inside a table row splices at row granularity
		// when the row rebuild keeps its geometry; otherwise the patch
		// widens to the owning table below.
		if res, ok := tryRelayoutTableRowPatch(ctx, prev, node, nil, opts); ok {
			return res, nil
		}

		table, ok := tableAncestorTarget(prev.boxTreeRoot(), node)
		if !ok {
			return nil, errElementNotPlaced
		}

		target, node = table, table.box.node
	} else if target.box.kind == boxKindCell {
		// A patched cell splices its row alone under the same rule.
		if res, ok := tryRelayoutTableRowPatch(ctx, prev, node, target, opts); ok {
			return res, nil
		}

		table, ok := tableOwnerTarget(target)
		if !ok {
			return nil, fmt.Errorf("%w: table cell with no owning table", errSubtreeNotSplicable)
		}

		target, node = table, table.box.node
	}

	built, err := relayoutBuildEscalating(ctx, prev, target, opts)
	if err != nil {
		return nil, err
	}

	start, oldCount, err := relayoutWindow(prev, built.target)
	if err != nil {
		return nil, err
	}

	commitSubtreeSplice(prev, built.box, built.target, built.styles, start, oldCount, built.eng.ops, false, nil)

	return relayoutSubtreeResult(prev, built.box, start, len(built.eng.ops), built.styles), nil
}

// relayoutBuiltTarget is one winning escalation attempt: the target whose
// subtree kept its size, with the styles and the engine that built it and the
// box they produced.
type relayoutBuiltTarget struct {
	target *relayoutTarget
	styles map[*html.Node]*ResolvedStyle
	eng    *engine
	box    *box
}

// relayoutBuildEscalating builds target's subtree, widening the patched range
// to the holder while the build reports a size change. Each attempt builds
// the range from the kept placement, so nothing is committed until an
// ancestor keeps its size; the rest of the page then keeps its boxes and its
// operations, and only the winning range is re-laid out.
//
// A wider range the splice cannot place ends the escalation with the size
// change the host must handle: the original errSubtreeResized is what a full
// layout resolves. Anything else (context, engine) is reported as is, and a
// first attempt that never resized keeps its own error.
func relayoutBuildEscalating(
	ctx context.Context, prev *Result, target *relayoutTarget, opts Options,
) (*relayoutBuiltTarget, error) {
	var firstResize error

	for {
		if target == nil || target.box == nil || target.box.node == nil {
			if firstResize != nil {
				return nil, firstResize
			}

			return nil, errElementNotPlaced
		}

		if err := checkRelayoutTarget(prev, target); err != nil {
			if firstResize != nil && isRelayoutStructural(err) {
				return nil, firstResize
			}

			return nil, err
		}

		styles, err := relayoutSubtreeStyles(ctx, prev, target, target.box.node, opts)
		if err != nil {
			if firstResize != nil && isRelayoutStructural(err) {
				return nil, firstResize
			}

			return nil, err
		}

		eng, err := relayoutEngine(ctx, prev, opts, styles)
		if err != nil {
			return nil, err
		}

		//nolint:contextcheck // the engine keeps the context and reads it at its recursion boundaries
		newBox, err := relayoutBuild(eng, prev, target, target.box.node, opts)
		if err == nil {
			return &relayoutBuiltTarget{target: target, styles: styles, eng: eng, box: newBox}, nil
		}

		if !errors.Is(err, errSubtreeResized) {
			return nil, err
		}

		if firstResize == nil {
			firstResize = err
		}

		// A moved grid refuses instead of widening: the table check compares
		// the whole grid, and a wider range would re-place the table without
		// that check, so the size change ends here.
		if target.box != nil && target.box.kind == boxKindTable {
			return nil, firstResize
		}

		target = relayoutWiderTarget(target)
		if target == nil {
			return nil, firstResize
		}
	}
}

// isRelayoutStructural reports whether err refuses a splice range for a
// reason a wider range inherits: the range cannot be placed, so the
// escalation ends with the size change instead of the wider refusal.
func isRelayoutStructural(err error) bool {
	return errors.Is(err, errSubtreeNotSplicable) || errors.Is(err, errElementNotPlaced)
}

// relayoutWiderTarget returns the patch target one level above target: the
// box that holds it becomes the patched box. It returns nil when target is
// already the outermost box, so the escalation ends with the size error.
func relayoutWiderTarget(target *relayoutTarget) *relayoutTarget {
	if target == nil || len(target.ancestors) == 0 {
		return nil
	}

	ancestors := target.ancestors
	grand := ancestors[:len(ancestors)-1]

	var holder *box

	if len(grand) > 0 {
		holder = grand[len(grand)-1]
	}

	return &relayoutTarget{box: ancestors[len(ancestors)-1], parent: holder, ancestors: grand}
}

// relayoutGuards rejects the inputs the splice cannot work with.
func relayoutGuards(ctx context.Context, prev *Result, node *html.Node, opts Options) error {
	if ctx == nil {
		return errs.ErrNilContext
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("layout: context: %w", err)
	}

	if prev == nil || prev.boxTreeRoot() == nil {
		return errNoPreviousLayout
	}

	if node == nil {
		return errNilElementNode
	}

	if err := opts.validate(); err != nil {
		return err
	}

	// A @container rule depends on the used size of a container, which the
	// kept state does not carry. The subtree pass would resolve the subtree
	// against the wrong container sizes, so the splice refuses the document.
	if css.HasContainerRules(opts.Sheets) {
		return fmt.Errorf("%w: the document uses @container rules", errSubtreeNotSplicable)
	}

	return nil
}

// relayoutTarget is the element a subtree patch works on: the box that owns
// the element, the box that holds it, and the boxes above it.
type relayoutTarget struct {
	box       *box
	parent    *box
	ancestors []*box
}

// findRelayoutTarget returns the box that owns node, the box that holds it,
// and the boxes above it, or nil when the element owns no box.
func findRelayoutTarget(root *box, node *html.Node) *relayoutTarget {
	if root == nil || node == nil {
		return nil
	}

	var found *relayoutTarget

	var walk func(boxNode *box, parent *box, ancestors []*box) bool

	walk = func(boxNode *box, parent *box, ancestors []*box) bool {
		if boxNode.node == node {
			found = &relayoutTarget{box: boxNode, parent: parent, ancestors: ancestors}

			return true
		}

		chain := make([]*box, len(ancestors), len(ancestors)+1)
		copy(chain, ancestors)
		chain = append(chain, boxNode)

		for _, child := range boxNode.children {
			if walk(child, boxNode, chain) {
				return true
			}
		}

		for _, row := range boxNode.rows {
			for _, cell := range row {
				if walk(cell, boxNode, chain) {
					return true
				}
			}
		}

		return false
	}

	walk(root, nil, nil)

	return found
}

// checkRelayoutTarget rejects the elements and subtrees the splice cannot
// place: the document root, a table box, a subtree that leaves the flow, and
// a subtree under a transformed or faded ancestor.
func checkRelayoutTarget(prev *Result, target *relayoutTarget) error {
	if err := checkRelayoutBox(target); err != nil {
		return err
	}

	if err := checkRelayoutAncestors(target); err != nil {
		return err
	}

	if err := checkRelayoutSubtree(target.box, target.box.node, prev.styles); err != nil {
		return err
	}

	return checkRelayoutAnchors(prev.boxTreeRoot(), target.box)
}

// checkRelayoutBox rejects the element itself: the document root carries the
// canvas fill. A table box re-runs the whole table and keeps only unmoved
// grids (see relayoutSameTable).
func checkRelayoutBox(target *relayoutTarget) error {
	if target.parent == nil {
		return fmt.Errorf("%w: the element is the root of the document", errSubtreeNotSplicable)
	}

	name := ""
	if target.box.node != nil {
		name = target.box.node.Name
	}

	if name == documentNodeName || name == htmlRootName {
		return fmt.Errorf("%w: the element is the document root", errSubtreeNotSplicable)
	}

	if target.box.kind != boxKindBlock && target.box.kind != boxKindTable {
		return fmt.Errorf("%w: the element is not a block or table box", errSubtreeNotSplicable)
	}

	return nil
}

// tableOwnerTarget returns the owning table of a cell target: the nearest
// table in its ancestor chain, keeping the holder and the chain above it.
func tableOwnerTarget(target *relayoutTarget) (*relayoutTarget, bool) {
	if target == nil {
		return nil, false
	}

	for index := len(target.ancestors) - 1; index >= 0; index-- {
		ancestor := target.ancestors[index]
		if ancestor == nil || ancestor.kind != boxKindTable {
			continue
		}

		var parent *box

		var ancestors []*box

		if index > 0 {
			parent = target.ancestors[index-1]
			ancestors = target.ancestors[:index]
		}

		return &relayoutTarget{box: ancestor, parent: parent, ancestors: ancestors}, true
	}

	return nil, false
}

// tableAncestorTarget returns the owning table of a box-less element: it
// climbs the document parents past cells to the nearest table. Any other
// placed box stops the climb with false.
func tableAncestorTarget(root *box, node *html.Node) (*relayoutTarget, bool) {
	if root == nil || node == nil {
		return nil, false
	}

	for current := node.Parent; current != nil; current = current.Parent {
		target := findRelayoutTarget(root, current)
		if target == nil || target.box == nil {
			continue
		}

		switch target.box.kind {
		case boxKindTable:
			return target, true
		case boxKindCell:
			continue
		default:
			return nil, false
		}
	}

	return nil, false
}

// checkRelayoutAncestors rejects a subtree under an ancestor that transforms
// or fades its content: the operations of the subtree carry that transform or
// that opacity, and a subtree laid out on its own does not.
func checkRelayoutAncestors(target *relayoutTarget) error {
	for _, ancestor := range target.ancestors {
		if ancestor.style == nil {
			continue
		}

		if ancestor.style.HasTransform || ancestor.style.Opacity < 1 {
			return fmt.Errorf(
				"%w: an ancestor carries a transform or an opacity",
				errSubtreeNotSplicable,
			)
		}
	}

	return nil
}

// checkRelayoutSubtree rejects a subtree that leaves the normal flow, floats,
// sticks, anchors, or feeds a named flow. Each of those places the box from
// state the kept result does not carry. An out-of-flow box is the exception
// when its containing block is inside the subtree or is the known viewport.
func checkRelayoutSubtree(root *box, rootNode *html.Node, styles map[*html.Node]*ResolvedStyle) error {
	var walk func(boxNode *box) error

	walk = func(boxNode *box) error {
		if reason := relayoutSubtreeReason(boxNode.style); reason != "" {
			viewportFixed := relayoutViewportFixed(boxNode, styles)
			if reason != relayoutOutOfFlowReason || (!viewportFixed &&
				!relayoutContainingBlockInside(boxNode.node, rootNode, styles)) {
				return fmt.Errorf("%w: %s", errSubtreeNotSplicable, reason)
			}
		}

		for _, child := range boxNode.children {
			if err := walk(child); err != nil {
				return err
			}
		}

		for _, row := range boxNode.rows {
			for _, cell := range row {
				if err := walk(cell); err != nil {
					return err
				}
			}
		}

		return nil
	}

	return walk(root)
}

// A viewport-fixed box uses the known viewport instead of its parent's
// containing block, so a standalone subtree can place it again.
func relayoutViewportFixed(root *box, styles map[*html.Node]*ResolvedStyle) bool {
	if root.style.Position != positionFixed {
		return false
	}

	if len(root.style.AnchorNames) > 0 || root.style.PositionAnchor != "" ||
		root.style.FlowInto != "" || root.style.FlowFrom != "" {
		return false
	}

	for node := root.node.Parent; node != nil; node = node.Parent {
		if style := styles[node]; style != nil && fixedContainingBlock(*style) {
			return false
		}
	}

	return true
}

// relayoutOutOfFlowReason names an out-of-flow box in a subtree.
const relayoutOutOfFlowReason = "the subtree holds an out-of-flow box"

// relayoutContainingBlockInside reports whether the containing block of an
// out-of-flow box is inside the patched subtree. The block is the nearest
// ancestor that establishes a containing block, so a box under a relative
// stage inside the subtree is placed again by the subtree layout, while a box
// under a positioned ancestor above the subtree is not.
//
// The walk starts at the box's parent: a box never establishes its own
// containing block. A fixed box uses the viewport unless an ancestor inside
// the subtree establishes one through transform or containment.
func relayoutContainingBlockInside(node, rootNode *html.Node, styles map[*html.Node]*ResolvedStyle) bool {
	if node == nil || rootNode == nil {
		return false
	}

	fixed := false

	if style, ok := styles[node]; ok && style != nil {
		fixed = style.Position == positionFixed
	}

	for current := node.Parent; current != nil; current = current.Parent {
		if current.Type != html.ElementNode {
			continue
		}

		style, ok := styles[current]
		if ok && relayoutCBAncestor(style, fixed) {
			return true
		}

		if current == rootNode {
			return false
		}
	}

	return false
}

// relayoutCBAncestor reports whether one ancestor establishes the containing
// block of the out-of-flow box.
func relayoutCBAncestor(style *ResolvedStyle, fixed bool) bool {
	if style == nil {
		return false
	}

	if fixed {
		return fixedContainingBlock(*style)
	}

	return establishesAbsCB(*style)
}

// fixedContainingBlock reports whether one ancestor establishes a containing
// block for a fixed box. Position alone does not: a fixed box against a
// relative ancestor still uses the viewport.
func fixedContainingBlock(style ResolvedStyle) bool {
	if style.HasTransform {
		return true
	}

	if containsLayout(style) {
		return true
	}

	return containsPaint(style)
}

// relayoutSubtreeReason names the flow the style leaves, or an empty string
// when the style keeps the box in the normal flow.
func relayoutSubtreeReason(style *ResolvedStyle) string {
	if style == nil {
		return ""
	}

	switch {
	case style.Float != cssDisplayNone:
		return "the subtree holds a float"
	case style.Position == positionAbsolute || style.Position == positionFixed:
		return relayoutOutOfFlowReason
	case style.Position == positionSticky:
		return "the subtree holds a sticky box"
	case len(style.AnchorNames) > 0 || style.PositionAnchor != "":
		return "the subtree holds an anchored element"
	case style.FlowInto != "" || style.FlowFrom != "":
		return "the subtree holds a named flow"
	}

	return ""
}

// checkRelayoutAnchors rejects a document that anchors a box outside the
// patched subtree: that box sits where its anchor sits, so a patch that
// moved the anchor would leave it stale.
func checkRelayoutAnchors(root *box, patched *box) error {
	var walk func(boxNode *box) error

	walk = func(boxNode *box) error {
		if boxNode == patched {
			return nil
		}

		if boxNode.style != nil && boxNode.style.PositionAnchor != "" {
			return fmt.Errorf("%w: an anchored box sits outside the subtree", errSubtreeNotSplicable)
		}

		for _, child := range boxNode.children {
			if err := walk(child); err != nil {
				return err
			}
		}

		for _, row := range boxNode.rows {
			for _, cell := range row {
				if err := walk(cell); err != nil {
					return err
				}
			}
		}

		return nil
	}

	return walk(root)
}

// maxSubtreeStyleStores bounds the pooled subtree style stores: one entry
// per recently patched document. maxSubtreeStyleStoreChunks bounds one
// entry: four chunks hold 8+16+32+64 records, about 700 KB of 5872-byte
// styles. maxSubtreeStyleStoreInternLen bounds one entry's intern map:
// a pathological document could otherwise grow one pooled store's buckets
// without bound and the pool slot would pin them. A store past either bound
// is dropped instead of pooled, so one large subtree cannot pin memory for
// later small patches; the live result keeps its own records through its
// pointers either way, and the next patch simply rents fresh.
const (
	maxSubtreeStyleStores         = 4
	maxSubtreeStyleStoreChunks    = 4
	maxSubtreeStyleStoreInternLen = 4096
)

// subtreeStyleDocTag identifies the document a pooled style store resolved:
// the document root and the stylesheet list. The addresses are kept as
// integers so the tag never pins the document or its sheets. A reused
// address can only waste sharing, never share wrongly, because
// styleStore.append verifies every record with styleInternEqual before it
// shares one.
type subtreeStyleDocTag struct {
	root   uintptr
	sheet  uintptr
	sheets int
}

// pooledSubtreeStyleStore is one retained resolution store with the document
// tag it resolved.
type pooledSubtreeStyleStore struct {
	tag   subtreeStyleDocTag
	store *styleStore
}

// subtreeStyleStorePool is a bounded process-wide free list of subtree
// resolution stores. Entries leave the pool on rent and re-enter on put, so
// one store is never resolved into from two goroutines at once.
type subtreeStyleStorePool struct {
	mu      sync.Mutex
	entries []pooledSubtreeStyleStore
}

//nolint:gochecknoglobals,exhaustruct // process-wide bounded style-store free list, guarded by mu
var subtreeStyleStores = &subtreeStyleStorePool{}

// subtreeStyleDocTagOf tags the document a patch resolves. Hover state is
// deliberately not part of the tag: a state change only appends records for
// styles that differ, while identical styles keep sharing the pooled ones.
func subtreeStyleDocTagOf(prev *Result, opts Options) subtreeStyleDocTag {
	tag := subtreeStyleDocTag{} //nolint:exhaustruct // unknown document until proven

	if prev != nil {
		if root := prev.documentRootNode(); root != nil {
			tag.root = reflect.ValueOf(root).Pointer()
		}
	}

	if len(opts.Sheets) > 0 && opts.Sheets[0] != nil {
		tag.sheet = reflect.ValueOf(opts.Sheets[0]).Pointer()
	}

	tag.sheets = len(opts.Sheets)

	return tag
}

// rentSubtreeStyleStore returns the pooled store that resolved tag, or a
// fresh store when no entry matches. A match keeps the entry's chunks and
// its intern map, so a repeated patch shares records instead of appending
// fresh ones; a mismatch rents empty, because another document's intern map
// must never be consulted.
func rentSubtreeStyleStore(tag subtreeStyleDocTag) *styleStore {
	subtreeStyleStores.mu.Lock()
	defer subtreeStyleStores.mu.Unlock()

	for index := range subtreeStyleStores.entries {
		if subtreeStyleStores.entries[index].tag != tag {
			continue
		}

		store := subtreeStyleStores.entries[index].store
		subtreeStyleStores.entries = slices.Delete(subtreeStyleStores.entries, index, index+1)

		if store == nil {
			break
		}

		return store
	}

	return &styleStore{} //nolint:exhaustruct // intentional zero-value store
}

// putSubtreeStyleStore returns a store to the pool after a patch resolved
// into it. The candidate scratch is cleared so the next rent starts clean;
// the chunks and the intern map stay, because the live result still points
// into them and the next patch for the same document shares them by
// appending only, never by overwriting. A store past the chunk or intern
// bound, a store whose tag carries no document address to match cheaply, or
// a pool that is full, drops the store instead of retaining it. Dropping
// never changes a resolved style: sharing still requires content equality
// through styleInternEqual, so the next patch rents fresh and resolves the
// same styles by content.
func putSubtreeStyleStore(tag subtreeStyleDocTag, store *styleStore) {
	if store == nil {
		return
	}

	// Without a document address the tag equality cannot identify the
	// document cheaply, so the store is not pooled. Rent keeps the existing
	// tag equality and never consults another document's intern map.
	if tag.root == 0 {
		return
	}

	store.candidate = ResolvedStyle{} //nolint:exhaustruct // scratch must not leak into the next rent

	if len(store.chunks) > maxSubtreeStyleStoreChunks {
		return
	}

	if len(store.intern) > maxSubtreeStyleStoreInternLen {
		return
	}

	total := 0

	for _, bucket := range store.intern {
		total += len(bucket)
		if total > maxSubtreeStyleStoreInternLen {
			return
		}
	}

	subtreeStyleStores.mu.Lock()
	defer subtreeStyleStores.mu.Unlock()

	if len(subtreeStyleStores.entries) >= maxSubtreeStyleStores {
		subtreeStyleStores.entries = slices.Delete(subtreeStyleStores.entries, 0, 1)
	}

	subtreeStyleStores.entries = append(subtreeStyleStores.entries, pooledSubtreeStyleStore{tag: tag, store: store})
}

// resolveSubtreeStylesPooled resolves the styles of node's subtree into the
// pooled store. It mirrors resolveStylesForSubtree in style_subtree.go, which
// builds a fresh styleStore per call; the walk is repeated here because the
// store must come from the pool on this file's patch path. Keep the two walks
// in sync: every element goes through subtreeNodeStyle, text nodes are
// skipped, and cancellation is polled every 64 visits.
func resolveSubtreeStylesPooled(
	ctx context.Context,
	styleCtx *styleContext,
	node *html.Node,
	parent *ResolvedStyle,
	store *styleStore,
) error {
	if styleCtx == nil || node == nil || store == nil {
		return nil
	}

	visited := 0

	var walk func(elem *html.Node, inherited *ResolvedStyle) error

	walk = func(elem *html.Node, inherited *ResolvedStyle) error {
		visited++

		if visited&63 == 0 && ctx != nil {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("layout: style subtree: %w", err)
			}
		}

		sty := subtreeNodeStyle(elem, styleCtx, inherited, store)

		for _, child := range elem.Children {
			if child.Type == html.TextNode {
				continue
			}

			if err := walk(child, sty); err != nil {
				return err
			}
		}

		return nil
	}

	return walk(node, parent)
}

// relayoutSubtreeStyles resolves the styles of the patched subtree and hands
// back the map the fresh engine reads. The subtree inherits from the style
// its parent box carries, and rem lengths resolve against the root's used
// font size, so the branch comes out with the same values a full cascade
// would give it. The pass rents a pooled styleStore for the document, so a
// repeated patch shares already interned records instead of appending fresh
// ones.
func relayoutSubtreeStyles(
	ctx context.Context, prev *Result, target *relayoutTarget, node *html.Node, opts Options,
) (map[*html.Node]*ResolvedStyle, error) {
	styleCtx := &styleContext{ //nolint:exhaustruct // intentional zero fields
		ctx:                ctx,
		sheets:             opts.Sheets,
		properties:         registeredProperties(opts.Sheets),
		media:              opts.Media,
		viewportW:          opts.Width,
		viewportH:          opts.Height,
		printLinkUnderline: opts.PrintLinkUnderline,
		state:              opts.State,
	}

	parent := relayoutParentStyle(target)
	remBase := relayoutRemBase(prev)

	// Rem lengths resolve against the root's used font size, the assignment
	// resolveStylesForSubtree makes for its own pass.
	if remBase > 0 {
		styleCtx.remBase = remBase
	}

	// The pooled store carries the chunks and the intern map of the last
	// patch for this document, so a repeated hover patch shares its records
	// through styleStore.append instead of appending fresh 5872-byte copies.
	tag := subtreeStyleDocTagOf(prev, opts)
	store := rentSubtreeStyleStore(tag)

	// A failed pass is dropped, never pooled: only a complete pass owns
	// records the next patch may share.
	if err := resolveSubtreeStylesPooled(ctx, styleCtx, node, parent, store); err != nil {
		return nil, err
	}

	styles := subtreeStyles(styleCtx, node, parent)
	if len(styles) == 0 {
		return nil, fmt.Errorf("%w: the subtree pass resolved no styles", errSubtreeNotSplicable)
	}

	relayoutTextStyles(styles)

	// The engine reads the styles of the ancestors when it walks up the tree
	// for the containing block of an out-of-flow child, so the kept styles of
	// the boxes above the subtree join the map.
	for _, ancestor := range target.ancestors {
		if ancestor.node == nil || ancestor.style == nil {
			continue
		}

		if _, ok := styles[ancestor.node]; !ok {
			styles[ancestor.node] = ancestor.style
		}
	}

	putSubtreeStyleStore(tag, store)

	return styles, nil
}

// relayoutParentStyle returns the style the patched element inherits from.
func relayoutParentStyle(target *relayoutTarget) *ResolvedStyle {
	if target.parent == nil || target.parent.node == nil || target.parent.style == nil {
		return nil
	}

	if target.parent.node.Name == documentNodeName {
		return nil
	}

	return target.parent.style
}

// relayoutRemBase returns the used font size of the root element, the base
// rem lengths resolve against. Zero means the document has no root element,
// and the cascade then uses the initial medium size.
func relayoutRemBase(prev *Result) float64 {
	for _, boxNode := range prev.boxes {
		if boxNode == nil || boxNode.node == nil || boxNode.style == nil {
			continue
		}

		if boxNode.node.Name == htmlRootName && boxNode.style.FontSize > 0 {
			return boxNode.style.FontSize
		}
	}

	return 0
}

// relayoutTextStyles gives every text node of the subtree the style of its
// parent element, the way the document cascade does.
func relayoutTextStyles(styles map[*html.Node]*ResolvedStyle) {
	for node, style := range styles {
		if node == nil || node.Type != html.ElementNode {
			continue
		}

		for _, child := range node.Children {
			if child.Type != html.TextNode {
				continue
			}

			if _, ok := styles[child]; !ok {
				styles[child] = style
			}
		}
	}
}

// relayoutEngine builds the fresh engine that lays out the subtree. The
// operation counter continues past the operations the kept result already
// carries, so the patched operations keep their own identity.
func relayoutEngine(
	ctx context.Context, prev *Result, opts Options, styles map[*html.Node]*ResolvedStyle,
) (*engine, error) {
	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		return nil, fmt.Errorf("layout: load default faces: %w", err)
	}

	if opts.Faces != nil {
		faces = opts.Faces
	}

	font := opts.Font
	if font == nil {
		font = faces.Regular
	}

	eng := newEngine(ctx, opts, faces, font, styles, nil, nil)
	eng.nextOpID = seedOpIDHighWater(prev) + 1

	return eng, nil
}

// relayoutMaxOpID returns the highest operation identity the kept result
// carries.
func relayoutMaxOpID(ops []Op) uint64 {
	var maxID uint64

	for index := range ops {
		if ops[index].ID > maxID {
			maxID = ops[index].ID
		}
	}

	return maxID
}

// relayoutBuild lays the subtree out and returns its new box. The build
// starts at the place the old box held, so the operations land where the
// operations they replace stood.
func relayoutBuild(
	eng *engine, prev *Result, target *relayoutTarget, node *html.Node, opts Options,
) (*box, error) {
	availW, posX, posY := relayoutOrigin(eng, target, opts)
	viewportFixed := relayoutEnterAncestorPaint(eng, target)

	newBox := eng.buildWithStyle(node, relayoutBlockSizeOverride(eng, target), availW, posX, posY)

	if eng.err != nil {
		return nil, eng.err
	}

	if newBox == nil {
		return nil, fmt.Errorf("%w: the element placed no box", errElementNotPlaced)
	}

	// The deferred backgrounds and borders of the subtree merge here. The
	// merge is scoped to the subtree's own operation window, and it remaps
	// the ranges of the boxes it placed.
	eng.finalizeChrome(newBox)

	if viewportFixed {
		eng.markOpsFixed(0, len(eng.ops)-1)
	}

	relayoutApplyDocumentColor(eng, prev, node)
	relayoutApplyAncestorClip(eng, target)

	if eng.needsXformStamp {
		propagatePerspective(newBox)
		stampBoxTransforms(newBox, IdentityMatrix(), eng.ops, opts.Media)
	}

	if !relayoutSameTable(newBox, target.box) {
		return nil, fmt.Errorf(
			"%w: the element is %gx%g at (%g, %g), the patch makes it %gx%g at (%g, %g)",
			errSubtreeResized,
			target.box.w, target.box.height, target.box.x, target.box.y,
			newBox.w, newBox.height, newBox.x, newBox.y,
		)
	}

	return newBox, nil
}

// The standalone build skips the ancestors' pushZ calls. Seed their stacking
// level and positioning flag before building, so a child's own scope can still
// override them exactly as it does during a full layout.
func relayoutEnterAncestorPaint(eng *engine, target *relayoutTarget) bool {
	viewportFixed, containingBlock := false, false

	for _, ancestor := range target.ancestors {
		style := ancestor.style
		if style == nil {
			continue
		}

		if style.Position == positionAbsolute || style.Position == positionFixed {
			eng.positioned = true
		}

		createsGroupContext := style.MixBlendMode != blendNormal ||
			style.Isolation == unicodeBidiIsolate || maskGroupRequested(style)
		eng.enterStackingContext(style, createsGroupContext)

		if style.Position == positionFixed && !containingBlock {
			viewportFixed = true
		}

		containingBlock = containingBlock || fixedContainingBlock(*style)
	}

	return viewportFixed
}

// relayoutBlockSizeOverride returns the style the subtree build carries when
// the containing block assigned the element's used block size, or nil when
// the element's own style decides it.
//
// A flex container sizes its items before they are built: a column item gets
// the flex-used main size and a stretched row item gets the line's cross
// size, which the container writes into the item's style. The standalone
// build has no parent flex pass to repeat that, so the retained height is
// installed as a definite size. The override is the one the flex pass writes:
// the block size and the max block size in the box's own sizing, so a taller
// content still overflows the assigned box instead of growing it.
func relayoutBlockSizeOverride(eng *engine, target *relayoutTarget) *ResolvedStyle {
	if target == nil || target.box == nil || target.box.node == nil {
		return nil
	}

	fresh := eng.styles[target.box.node]
	if fresh == nil {
		return nil
	}

	height, ok := relayoutAssignedBlockSize(eng, target, fresh)
	if !ok {
		return nil
	}

	override := *fresh
	eng.forceFlexItemCrossSize(&override, height)

	return &override
}

// relayoutAssignedBlockSize returns the block size the containing block
// assigned the element, when the element's own cascade leaves the size auto.
//
// The kept style carries the fingerprint of that assignment: the flex pass
// writes the used border-box height into both Height and MaxHeight and
// clears both percentages, a pair the cascade does not produce on its own.
// The parent must be the formatting context that assigns it: a column flex
// container with a definite content height, or a row flex container whose
// align stretch applies to the item.
//
// A content-sized element keeps no such fingerprint, and a flex item in an
// indefinite column keeps the size its content chose, so both still fail the
// size check when their content changes.
func relayoutAssignedBlockSize(
	eng *engine, target *relayoutTarget, fresh *ResolvedStyle,
) (float64, bool) {
	if target.parent == nil || target.parent.style == nil || fresh == nil {
		return 0, false
	}

	// The element's own cascade decides the size when it names one. The
	// guard keeps its power over a class that changes that size.
	if fresh.Height >= 0 || fresh.HeightPercent >= 0 {
		return 0, false
	}

	if !relayoutFlexAssignedHeight(target.box.style) {
		return 0, false
	}

	if !relayoutFlexAssignsBlockSize(eng, target, fresh) {
		return 0, false
	}

	return target.box.height, true
}

// relayoutFlexAssignedHeight reports whether style carries the block size a
// flex container assigned an item. forceFlexItemCrossSize writes the used
// border-box height into Height and MaxHeight and clears both percentages.
func relayoutFlexAssignedHeight(style *ResolvedStyle) bool {
	if style == nil || style.Height < 0 || style.MaxHeight < 0 {
		return false
	}

	if style.HeightPercent >= 0 || style.MaxHeightPercent >= 0 {
		return false
	}

	return relayoutNear(style.Height, style.MaxHeight)
}

// relayoutFlexAssignsBlockSize reports whether the parent flex container
// assigned the item's used block size. The assignment is expressed on the
// physical block axis, so a vertical writing mode keeps the full layout.
func relayoutFlexAssignsBlockSize(
	eng *engine, target *relayoutTarget, fresh *ResolvedStyle,
) bool {
	parentStyle := target.parent.style

	if isVerticalWritingMode(parentStyle.WritingMode) || isVerticalWritingMode(fresh.WritingMode) {
		return false
	}

	if parentStyle.Display != displayFlex && parentStyle.Display != displayInlineFlex {
		return false
	}

	switch relayoutFlexDirection(parentStyle) {
	case fxRow, fxRowRev:
		// The line's cross size is the item's used block size, whatever the
		// item's content measures.
		return flexItemCrossStretchNode(target.box.node, parentStyle, fresh)
	case fxCol, fxColRev:
		// The flex main size is definite only when the container's content
		// height is and the retained line consumed it. A content-sized
		// column, or one that leaves free space, keeps every item at its
		// content height, so the check must not claim an assignment there.
		return relayoutFlexColumnFilled(eng, target.parent)
	}

	return false
}

// relayoutFlexColumnFilled reports whether the retained column consumed its
// definite content height with flex sizing. When the items leave free space,
// no grow or shrink ran and every used main size is the item's content-based
// base, so the kept height is not an assignment and the size check keeps its
// power. Wrapped lines have per-line free space the sum does not expose, so
// they keep the full layout.
func relayoutFlexColumnFilled(eng *engine, parent *box) bool {
	if parent.style.FlexWrap == fxWrap || parent.style.FlexWrap == fxWrapRev {
		return false
	}

	contentH := resolveFlexColumnContentHeight(*parent.style, eng)
	if contentH < 0 {
		return false
	}

	used := 0.0
	items := 0

	for _, child := range parent.children {
		if child == nil || child.style == nil {
			continue
		}

		if child.style.Position == positionAbsolute || child.style.Position == positionFixed {
			continue
		}

		used += child.height + eng.scalePt(child.style.MarginTop) + eng.scalePt(child.style.MarginBottom)
		items++
	}

	if items == 0 {
		return false
	}

	rowGap, _ := eng.styleGaps(parent.style)
	used += rowGap * float64(items-1)

	return relayoutNear(used, contentH)
}

// relayoutFlexDirection returns the container's main axis, applying the
// initial row direction when the declaration is absent.
func relayoutFlexDirection(style *ResolvedStyle) string {
	if style.FlexDirection == "" {
		return fxRow
	}

	return style.FlexDirection
}

// relayoutOrigin returns the width the element is laid out in, the x it is
// placed at, and the y it is placed at.
//
// The element keeps the box it had, so the kept box gives the place the
// parent gave it: a grid item is placed in its own track and a flex item in
// its own line, not in the whole holding box.
//
// The engine puts the border box at the origin plus the element's left
// margin, and it takes both margins off the width it is given. The origin is
// therefore the kept origin minus the left margin, and the width is the
// kept width plus both margins.
func relayoutOrigin(eng *engine, target *relayoutTarget, opts Options) (float64, float64, float64) {
	posY := target.box.y

	kept := target.box
	if kept.style == nil {
		return opts.Width, kept.x, posY
	}

	marginL := eng.scalePt(kept.style.MarginLeft)
	marginR := eng.scalePt(kept.style.MarginRight)

	return relayoutAvailableWidth(eng, kept, marginL, marginR), kept.x - marginL, posY
}

// relayoutAvailableWidth returns the width the element is laid out in.
//
// The element keeps the width it had, so the width the parent gave it comes
// from the kept box. An auto width takes the kept border-box width plus the
// margins the engine takes off it, and a percentage width takes the width
// the kept content width was a percentage of.
func relayoutAvailableWidth(eng *engine, kept *box, marginL, marginR float64) float64 {
	pct := kept.style.WidthPercent
	if pct <= 0 {
		return kept.w + marginL + marginR
	}

	contentW := kept.w
	if kept.style.BoxSizing != borderBox {
		contentW -= kept.style.horizontalChrome(eng)
	}

	if contentW < 0 {
		contentW = 0
	}

	return contentW * oneHundred / pct
}

// relayoutSameBox reports whether two boxes sit at the same place with the
// same size.
func relayoutSameBox(left, right *box) bool {
	if left == nil || right == nil {
		return false
	}

	return relayoutNear(left.x, right.x) && relayoutNear(left.y, right.y) &&
		relayoutNear(left.w, right.w) && relayoutNear(left.height, right.height)
}

// relayoutSameTable reports whether a rebuilt subtree keeps its place: a
// non-table delegates to relayoutSameBox, a table compares the whole grid
// with sameBoxGeometry so only an unmoved grid splices.
func relayoutSameTable(newBox, kept *box) bool {
	if newBox == nil || kept == nil {
		return false
	}

	if kept.kind != boxKindTable && newBox.kind != boxKindTable {
		return relayoutSameBox(newBox, kept)
	}

	if kept.kind != boxKindTable || newBox.kind != boxKindTable {
		return false
	}

	return sameBoxGeometry(newBox, kept)
}

// relayoutNear reports whether two lengths are the same length.
func relayoutNear(left, right float64) bool {
	diff := left - right
	if diff < 0 {
		diff = -diff
	}

	return diff <= relayoutEpsilon
}

// relayoutPaintOnlyWindow reports whether a freshly built subtree can replace
// the old one by overwriting the operation window alone. The check is
// deliberately conservative: it requires the operation count, every box's
// geometry and child shape, the pagination inputs PageBreaks reads, and every
// paint-order key to stay the same. A false result only sends the splice down
// the allocating path, which is always correct.
func relayoutPaintOnlyWindow(
	prev *Result, target *relayoutTarget, newBox *box, start, oldCount int, newOps []Op,
) bool {
	if len(newOps) != oldCount {
		return false
	}

	if !sameBoxGeometry(target.box, newBox) {
		return false
	}

	if !sameBoxPaintInputs(target.box, newBox, start) {
		return false
	}

	return samePaintOnlyOpWindow(prev.Ops[start:start+oldCount], newOps)
}

// sameBoxGeometry compares the placed geometry of two box subtrees: the border
// box, the child count, and the row and cell counts. Styles and text are not
// compared; sameBoxPaintInputs carries the inputs that can change paint
// order, ownership, or pagination without moving a box.
func sameBoxGeometry(left, right *box) bool {
	if left == nil || right == nil {
		return left == right
	}

	if !relayoutNear(left.x, right.x) || !relayoutNear(left.y, right.y) ||
		!relayoutNear(left.w, right.w) || !relayoutNear(left.height, right.height) {
		return false
	}

	if len(left.children) != len(right.children) || len(left.rows) != len(right.rows) {
		return false
	}

	for index := range left.rows {
		if len(left.rows[index]) != len(right.rows[index]) {
			return false
		}
	}

	for index := range left.children {
		if !sameBoxGeometry(left.children[index], right.children[index]) {
			return false
		}
	}

	return true
}

// sameBoxPaintInputs compares the subtree inputs that can change owners, paint
// order, or page breaks without changing a box's geometry: the box kind, the
// operation range of every box, the break-relevant style fields, the heading
// tag, and the table row metadata. Operation ranges are compared relative to
// shift because the fresh engine numbers its operations from zero.
func sameBoxPaintInputs(left, right *box, shift int) bool {
	if left == nil || right == nil {
		return left == right
	}

	if left.kind != right.kind ||
		!sameBoxOpWindow(left, right, shift) ||
		!sameBreakStyle(left.style, right.style) ||
		!sameBreakNode(left.node, right.node) ||
		!sameRowBreakMetadata(left, right, shift) {
		return false
	}

	for index := range left.children {
		if !sameBoxPaintInputs(left.children[index], right.children[index], shift) {
			return false
		}
	}

	return true
}

// sameBoxOpWindow compares one box's operation range after rebasing the old
// range from the committed window to the fresh subtree's local numbering. An
// empty range on one side must be empty on the other.
func sameBoxOpWindow(left, right *box, shift int) bool {
	leftEmpty := left.opEnd < left.opStart
	rightEmpty := right.opEnd < right.opStart

	if leftEmpty || rightEmpty {
		return leftEmpty == rightEmpty
	}

	return left.opStart-shift == right.opStart && left.opEnd-shift == right.opEnd
}

// sameRowBreakMetadata compares the table row tops and heights PageBreaks
// reads, plus the operation ranges of the row's cells.
func sameRowBreakMetadata(left, right *box, shift int) bool {
	if len(left.rowTops) != len(right.rowTops) || len(left.rowHeights) != len(right.rowHeights) {
		return false
	}

	for index := range left.rowTops {
		if left.rowTops[index] != right.rowTops[index] ||
			left.rowHeights[index] != right.rowHeights[index] {
			return false
		}
	}

	if len(left.rows) != len(right.rows) {
		return false
	}

	for rowIndex := range left.rows {
		if len(left.rows[rowIndex]) != len(right.rows[rowIndex]) {
			return false
		}

		for cellIndex := range left.rows[rowIndex] {
			leftCell, rightCell := left.rows[rowIndex][cellIndex], right.rows[rowIndex][cellIndex]

			if leftCell == nil || rightCell == nil {
				if leftCell != rightCell {
					return false
				}

				continue
			}

			if !sameBoxOpWindow(leftCell, rightCell, shift) {
				return false
			}
		}
	}

	return true
}

// samePaintOnlyOpWindow compares the operation window fields that decide
// pagination and paint order. A same-length window whose break fields and
// ordering keys match at every index leaves the maintained owners, order, and
// page breaks untouched. A blend group falls back because every build creates
// fresh group pointers.
func samePaintOnlyOpWindow(oldOps, newOps []Op) bool {
	if len(oldOps) != len(newOps) {
		return false
	}

	for index := range oldOps {
		if !sameBreakOp(&oldOps[index], &newOps[index]) {
			return false
		}

		if !samePaintOrderKey(&oldOps[index], &newOps[index]) {
			return false
		}
	}

	return true
}

// samePaintOrderKey compares the keys comparePaintOrder reads. The operations
// keep their indices, so equal keys at every index keep the previous order.
func samePaintOrderKey(left, right *Op) bool {
	if left.Group() != nil || right.Group() != nil {
		return false
	}

	if left.ZIndexSet != right.ZIndexSet {
		return false
	}

	if left.ZIndexSet && left.ZIndex != right.ZIndex {
		return false
	}

	if left.Positioned != right.Positioned {
		return false
	}

	return paintLayer(left) == paintLayer(right)
}

// relayoutApplyDocumentColor applies the document's color-adjust paint
// policy to the new operations. The subtree pass runs the policy with the
// subtree root, which is not the document root, so the policy is applied
// here with the root the document laid out.
func relayoutApplyDocumentColor(eng *engine, prev *Result, node *html.Node) {
	if node != nil && node.Name == htmlRootName {
		return
	}

	eng.applyColorAdjustPaintPolicy(prev.boxTreeRoot())
}

// relayoutApplyAncestorClip clips the new operations to the padding box of
// the ancestors that clip paint. The fresh engine knows the clips inside the
// subtree; the clips above it live in the kept result.
func relayoutApplyAncestorClip(eng *engine, target *relayoutTarget) {
	var clip *clipRect

	for _, ancestor := range target.ancestors {
		next := eng.computeBoxOverflowClip(ancestor, clip)
		clip = next
	}

	if clip == nil || clip.empty() || len(eng.ops) == 0 {
		return
	}

	clipOpsRange(eng.ops, 0, len(eng.ops)-1, *clip)
}

// relayoutWindow returns the operation range the element's box owned and the
// number of operations in it. An element that emitted no operation has no
// range to replace, so the place its operations would have taken is derived
// from its neighbours.
func relayoutWindow(prev *Result, target *relayoutTarget) (int, int, error) {
	boxNode := target.box

	if boxNode.opStart >= 0 && boxNode.opEnd >= boxNode.opStart && boxNode.opEnd < len(prev.Ops) {
		return boxNode.opStart, boxNode.opEnd - boxNode.opStart + 1, nil
	}

	start := relayoutInsertionIndex(prev, target)
	if start < 0 || start > len(prev.Ops) {
		return 0, 0, fmt.Errorf("%w: the element owns no operations", errSubtreeNotSplicable)
	}

	return start, 0, nil
}

// relayoutInsertionIndex returns the place the operations of an element that
// emitted none belong: in front of the next sibling that has operations, or
// behind the last operation of the box that holds it.
func relayoutInsertionIndex(prev *Result, target *relayoutTarget) int {
	parent := target.parent

	if parent != nil {
		for index, child := range parent.children {
			if child != target.box {
				continue
			}

			for _, next := range parent.children[index+1:] {
				if next.opEnd >= next.opStart && next.opStart >= 0 {
					return next.opStart
				}
			}

			break
		}
	}

	if parent != nil && parent.opEnd >= parent.opStart && parent.opEnd < len(prev.Ops) {
		return parent.opEnd + 1
	}

	return len(prev.Ops)
}

// relayoutSpliceOps replaces the operations of the old range with the new
// ones and shifts the tail. The result keeps its own operation list, so a
// display that holds the old one still reads the operations it had until the
// host copies the patch over.
func relayoutSpliceOps(prev *Result, start, oldCount int, newOps []Op) {
	ops := make([]Op, 0, len(prev.Ops)-oldCount+len(newOps))
	ops = append(ops, prev.Ops[:start]...)
	ops = append(ops, newOps...)
	ops = append(ops, prev.Ops[start+oldCount:]...)

	prev.Ops = ops
}

// relayoutSpliceOpsInPlace replaces the operations of the old range inside the
// result's own list: a splice that keeps the operation count overwrites the
// window, and one that changes the count slides the tail inside the spare
// capacity, so a batch does not copy the page-wide list once per patched
// element. A list that must grow past its capacity falls back to
// relayoutSpliceOps.
//
// Only a caller that owns prev exclusively may use it, the batch on its private
// clone. The single-element path keeps the allocating splice because the
// display the host still holds shares the result's operation backing.
func relayoutSpliceOpsInPlace(prev *Result, start, oldCount int, newOps []Op) {
	if ops, ok := spliceOpsInPlace(prev.Ops, start, oldCount, newOps); ok {
		prev.Ops = ops

		return
	}

	relayoutSpliceOps(prev, start, oldCount, newOps)
}

// spliceOpsInPlace returns ops with the window replaced inside its own backing
// and true, or nil and false when the backing would have to grow.
func spliceOpsInPlace(ops []Op, start, oldCount int, newOps []Op) ([]Op, bool) {
	if oldCount == len(newOps) {
		copy(ops[start:start+oldCount], newOps)

		return ops, true
	}

	oldLen := len(ops)
	newLen := oldLen - oldCount + len(newOps)
	if newLen > cap(ops) {
		return nil, false
	}

	ops = slices.Replace(ops, start, start+oldCount, newOps...)

	// A shrink leaves the dropped operations past the new end. Clear them so
	// their rare payloads do not stay reachable through the retained backing.
	if newLen < oldLen {
		clear(ops[newLen:oldLen])
	}

	return ops, true
}

// relayoutRebaseTree moves the operation ranges of a new box tree from the
// window the fresh engine used to the window the patched result uses.
func relayoutRebaseTree(boxNode *box, shift int) {
	if boxNode == nil {
		return
	}

	boxNode.opStart += shift
	boxNode.opEnd += shift

	for _, child := range boxNode.children {
		relayoutRebaseTree(child, shift)
	}

	for _, row := range boxNode.rows {
		for _, cell := range row {
			relayoutRebaseTree(cell, shift)
		}
	}
}

// relayoutSwapBox puts the new box where the old one stood.
func relayoutSwapBox(target *relayoutTarget, newBox *box) {
	if target.parent == nil {
		return
	}

	for index, child := range target.parent.children {
		if child == target.box {
			target.parent.children[index] = newBox

			return
		}
	}
}

// relayoutShiftRanges moves the operation ranges of the boxes outside the
// patched subtree. The ranges after the window shift by delta, and the ranges
// that cover the window grow or shrink with it.
//
// A table cell is reachable twice — through its parent's children and through
// the table's row grid — so the walk visits each box once: a second visit
// would shift the same range twice and corrupt every window located after a
// count-changing splice in the same batch.
func relayoutShiftRanges(root *box, patched *box, start, end, delta int) {
	seen := make(map[*box]struct{})

	var walk func(boxNode *box)

	walk = func(boxNode *box) {
		if boxNode == patched {
			return
		}

		if _, ok := seen[boxNode]; ok {
			return
		}

		seen[boxNode] = struct{}{}

		shiftBoxRange(boxNode, start, end, delta)

		for _, child := range boxNode.children {
			walk(child)
		}

		for _, row := range boxNode.rows {
			for _, cell := range row {
				walk(cell)
			}
		}
	}

	walk(root)
}

// shiftBoxRange moves one box's operation range over a splice: ranges after
// the replaced window shift by delta, and a range covering the window grows
// or shrinks with it. A box that starts inside the window takes the window
// start, the rule for an empty box sitting at the insertion point.
func shiftBoxRange(boxNode *box, start, end, delta int) {
	if boxNode.opEnd < boxNode.opStart || boxNode.opEnd < start {
		return
	}

	if boxNode.opStart > end {
		boxNode.opStart += delta
		boxNode.opEnd += delta

		return
	}

	if boxNode.opStart >= start {
		boxNode.opStart = start
	}

	boxNode.opEnd += delta
}

// unionChildRange widens a parent range over one child's range, the step the
// ancestor re-widening applies bottom-up.
func unionChildRange(boxNode, child *box) {
	if child.opEnd < child.opStart {
		return
	}

	if boxNode.opEnd < boxNode.opStart {
		boxNode.opStart, boxNode.opEnd = child.opStart, child.opEnd

		return
	}

	if child.opStart < boxNode.opStart {
		boxNode.opStart = child.opStart
	}

	if child.opEnd > boxNode.opEnd {
		boxNode.opEnd = child.opEnd
	}
}

// relayoutKeepBoxes rebuilds the flattened box list of the patched result.
func relayoutKeepBoxes(prev *Result) {
	boxes := make([]*box, 0)
	flattenBoxes(prev.boxTreeRoot(), &boxes)

	prev.boxes = boxes
}

// relayoutBoxListCapacity returns the length the flattened box list will have
// after one splice: the kept list, minus the boxes the old subtree contributed,
// plus the boxes the new subtree contributes.
func relayoutBoxListCapacity(prev *Result, oldBox, newBox *box) int {
	capacity := len(prev.boxes) - countBoxNodes(oldBox) + countBoxNodes(newBox)
	if capacity < 0 {
		return 0
	}

	return capacity
}

// countBoxNodes counts the boxes flattenBoxes appends for one subtree. Table
// cells hang off rows, which the flattened list does not follow, so the count
// walks children only.
func countBoxNodes(node *box) int {
	if node == nil {
		return 0
	}

	count := 1
	for _, child := range node.children {
		count += countBoxNodes(child)
	}

	return count
}

// relayoutKeepStyles writes the styles of the patched subtree into the kept
// style map, so the map and the box tree agree after the patch.
func relayoutKeepStyles(prev *Result, styles map[*html.Node]*ResolvedStyle) {
	if prev.styles == nil {
		prev.styles = make(map[*html.Node]*ResolvedStyle, len(styles))
	}

	for node, style := range styles {
		prev.styles[node] = style
	}
}

// relayoutSubtreeResult returns the patched subtree on its own: its
// operations and its boxes, with the box ranges pointing into that slice.
func relayoutSubtreeResult(
	prev *Result, newBox *box, start, count int, styles map[*html.Node]*ResolvedStyle,
) *Result {
	subRoot := rebaseBox(newBox, start)

	boxes := make([]*box, 0)
	flattenBoxes(subRoot, &boxes)

	return &Result{ //nolint:exhaustruct // a subtree result keeps the zero fields
		Ops:     prev.Ops[start : start+count],
		Width:   prev.Width,
		Height:  prev.Height,
		root:    subRoot,
		rootBox: subRoot,
		boxes:   boxes,
		styles:  styles,
	}
}

// rebaseBox copies one box subtree and shifts every operation range so the
// indices point into the sliced operation list of the subtree.
func rebaseBox(src *box, shift int) *box {
	if src == nil {
		return nil
	}

	clone := *src
	clone.children = nil
	clone.rows = nil
	clone.stickyPort = nil

	if src.opEnd >= src.opStart {
		clone.opStart = src.opStart - shift
		clone.opEnd = src.opEnd - shift
	}

	for _, child := range src.children {
		clone.children = append(clone.children, rebaseBox(child, shift))
	}

	for _, row := range src.rows {
		clonedRow := make([]*box, 0, len(row))

		for _, cell := range row {
			clonedRow = append(clonedRow, rebaseBox(cell, shift))
		}

		clone.rows = append(clone.rows, clonedRow)
	}

	clone.stickyPort = rebaseBox(src.stickyPort, shift)

	return &clone
}

// commitSubtreeSplice splices a freshly built subtree into prev: the new
// operations replace the old window, the new box takes the old box's place,
// and one fused walk shifts the later ranges, re-widens the ancestors, and
// rebuilds the flattened box list. The census and the operation identity
// high-water mark advance from the window alone, rescanning the page only
// when the removed window could have carried the maximum extent or the only
// fragment link. privateOps selects the in-place splice for a caller that owns
// prev exclusively, the batch on its clone. A batch splice skips the
// page-sized flat-list rebuild: the batch rebuilds the list once after its
// last splice, and the shifting, re-widening, and position recording run
// either way. pos collects the placed position
// of the splice for the incremental owners update, or nil when the caller
// keeps no owners.
func commitSubtreeSplice(
	prev *Result, newBox *box, target *relayoutTarget,
	styles map[*html.Node]*ResolvedStyle, start, oldCount int, newOps []Op,
	privateOps bool, pos *splicePosition,
) {
	delta := len(newOps) - oldCount
	oldMax, oldFrag := censusOpsWindow(prev.Ops, start, oldCount)
	// The rebuilt list holds every kept box plus the new subtree in place of
	// the old one. The old box is still in the tree here, so both counts are
	// taken before the swap.
	boxCapacity := relayoutBoxListCapacity(prev, target.box, newBox)

	// A same-count splice keeps the identities of the operations it replaces:
	// the boxes that keep their ranges keep pointing at the same operation
	// identities, and a host matching the old display to the new one sees the
	// same operation at the same index.
	if len(newOps) == oldCount {
		relayoutKeepWindowIDs(prev.Ops[start:start+oldCount], newOps)
	}

	if privateOps {
		relayoutSpliceOpsInPlace(prev, start, oldCount, newOps)
	} else {
		relayoutSpliceOps(prev, start, oldCount, newOps)
	}

	relayoutRebaseTree(newBox, start)
	relayoutSwapBox(target, newBox)
	relayoutFuseSplice(prev, newBox, start, start+oldCount-1, delta, boxCapacity, pos, !privateOps)
	relayoutKeepStyles(prev, styles)
	refreshCensusSpliced(prev, oldMax, oldFrag, start, len(newOps))
	storeOpIDHighWater(prev, newOps)
}

// commitPaintOnlySplice commits a geometry-neutral batch splice: the new
// operations overwrite the old window, the new box takes the old box's place,
// and everything else the page keeps stays as it was. The flat box list, the
// operation ranges outside the window, the owners, the paint order, and the
// page breaks are untouched, so the caller can adopt a display built from the
// previous one. The census and the identity high-water mark still advance,
// because a paint-only change can alter a fill extent or a fragment link.
func commitPaintOnlySplice(
	prev *Result, newBox *box, target *relayoutTarget,
	styles map[*html.Node]*ResolvedStyle, start, count int, newOps []Op,
) {
	oldMax, oldFrag := censusOpsWindow(prev.Ops, start, count)

	relayoutKeepWindowIDs(prev.Ops[start:start+count], newOps)
	copy(prev.Ops[start:start+count], newOps)

	relayoutRebaseTree(newBox, start)
	relayoutSwapBox(target, newBox)
	relayoutKeepStyles(prev, styles)
	refreshCensusSpliced(prev, oldMax, oldFrag, start, len(newOps))
	storeOpIDHighWater(prev, newOps)
}

// relayoutKeepWindowIDs copies the operation identities of a same-count window
// onto its replacement. A change that keeps the operation count repaints the
// window in place semantically; the boxes that keep their operation ranges keep
// pointing at the same identities, so Display.Box.OperationIDs stays accurate
// without a box rebuild, and the previous display's operations and the patched
// ones match by identity for a host's transition capture.
func relayoutKeepWindowIDs(oldOps, newOps []Op) {
	for index := range newOps {
		newOps[index].ID = oldOps[index].ID
	}
}

// splicePosition collects the placed position of one splice while the fused
// walk runs: base is the placed index the new box holds, seed is the placed
// index of the deepest element ancestor the owners walk can see, and
// foundBase is false when the new box sits under a table row, where the
// owners walk never visits. The walk counts element boxes on the children
// track only, the same order WalkPlaced and OperationOwners use.
type splicePosition struct {
	base, seed int
	foundBase  bool
	count      int
	chain      []*box
}

// noteSplicePosition records one visited box for the splice position: an
// element on the target's ancestor chain refreshes the seed, and the element
// count is the placed index the next element box would hold.
func noteSplicePosition(pos *splicePosition, node *box) {
	if pos == nil {
		return
	}

	if node.node == nil || node.node.Type != html.ElementNode {
		return
	}

	for _, ancestor := range pos.chain {
		if ancestor == node {
			pos.seed = pos.count

			break
		}
	}

	pos.count++
}

// relayoutFuseSplice commits one splice in a single walk over the box tree:
// it shifts the ranges after the replaced window, re-widens every ancestor
// over the window, rebuilds the flattened box list, and records the placed
// position of the splice for the incremental owners update. boxCapacity is the
// length the rebuilt list will have, so the walk does not grow it from zero.
// collectBoxes selects whether the walk rebuilds the flattened box list. A
// batch splice passes false: the batch owns prev exclusively and rebuilds the
// list once after its last splice, so intermediate splices skip the
// page-sized rebuild while the shifting, re-widening, and position recording
// run either way. No mid-batch reader needs the list: the rem-base scan only
// reads the document root, which no splice replaces.
// The rows track only shifts: table cells keep their ranges but never join the
// owner, placed, or flattened walks, the rule the three passes it replaces
// followed.
func relayoutFuseSplice(
	prev *Result, patched *box, start, end, delta, boxCapacity int, pos *splicePosition, collectBoxes bool,
) {
	var boxes []*box

	if collectBoxes {
		boxes = make([]*box, 0, boxCapacity)
	}

	var fuse func(node *box)
	var shiftRow func(node *box)

	fuse = func(node *box) {
		if node == nil {
			return
		}

		if node == patched {
			if pos != nil {
				pos.base, pos.foundBase = pos.count, true
			}

			unionPatchedSubtree(node)

			if collectBoxes {
				flattenBoxes(node, &boxes)
			}

			return
		}

		shiftBoxRange(node, start, end, delta)
		noteSplicePosition(pos, node)

		if collectBoxes {
			boxes = append(boxes, node)
		}

		for _, child := range node.children {
			fuse(child)
			unionChildRange(node, child)
		}

		for _, row := range node.rows {
			for _, cell := range row {
				shiftRow(cell)
			}
		}
	}

	shiftRow = func(node *box) {
		if node == nil || node == patched {
			return
		}

		shiftBoxRange(node, start, end, delta)

		for _, child := range node.children {
			shiftRow(child)
		}

		for _, row := range node.rows {
			for _, cell := range row {
				shiftRow(cell)
			}
		}
	}

	fuse(prev.boxTreeRoot())

	if collectBoxes {
		prev.boxes = boxes
	}
}

// unionPatchedSubtree re-widens the committed subtree over its own children,
// the part of the ancestor re-widening the fused walk skips with the patched
// box itself.
func unionPatchedSubtree(node *box) {
	for _, child := range node.children {
		unionPatchedSubtree(child)
		unionChildRange(node, child)
	}
}

// censusOpsWindow summarizes one operation window the way censusOps does:
// the right edge of fill, stroke, image, and path operations, and whether
// any link operation targets a same-document fragment.
func censusOpsWindow(ops []Op, start, count int) (maxExtent float64, hasFrag bool) {
	if start < 0 {
		count += start
		start = 0
	}

	end := start + count
	if end > len(ops) {
		end = len(ops)
	}

	for index := start; index < end; index++ {
		switch ops[index].Kind {
		case OpFillRect, OpStrokeRect, OpImage, OpPath:
			if extent := ops[index].X + ops[index].W; extent > maxExtent {
				maxExtent = extent
			}
		case OpLinkURI:
			if !hasFrag && strings.HasPrefix(ops[index].URI, "#") {
				hasFrag = true
			}
		case OpLine, OpGridRun, OpText, OpBullet, OpUnknown, opKindNoop:
			continue
		}
	}

	return maxExtent, hasFrag
}

// refreshCensusSpliced advances the page census over one committed splice.
// The new window maximum wins outright; otherwise the old maximum stands
// unless the removed window held it, which a rescan settles. Fragment links
// stand unless the removed window held the only one, which a rescan settles.
// A non-positive maximum means unknown, and the page is rescanned.
func refreshCensusSpliced(prev *Result, oldMax float64, oldFrag bool, start, newCount int) {
	oldPageMax, oldPageFrag := prev.MaxContentX, prev.HasFragmentLinks

	if oldPageMax <= 0 {
		prev.MaxContentX, prev.HasFragmentLinks = censusOps(prev.Ops, prev.Width)

		return
	}

	newMax, newFrag := censusOpsWindow(prev.Ops, start, newCount)

	maxX := oldPageMax

	switch {
	case newMax >= oldPageMax:
		maxX = newMax
	case oldMax >= oldPageMax:
		maxX, _ = censusOps(prev.Ops, prev.Width)
	}

	frag := oldPageFrag

	switch {
	case !oldPageFrag:
		frag = newFrag
	case oldFrag && !newFrag:
		_, frag = censusOps(prev.Ops, prev.Width)
	}

	if maxX < prev.Width {
		maxX = prev.Width
	}

	prev.MaxContentX, prev.HasFragmentLinks = maxX, frag
}

// seedOpIDHighWater returns the operation identity the next splice builds
// from: the stored high-water mark, or a full scan for a placement built
// before the mark existed.
func seedOpIDHighWater(prev *Result) uint64 {
	if prev.nextOpID != 0 {
		return prev.nextOpID
	}

	return relayoutMaxOpID(prev.Ops)
}

// storeOpIDHighWater advances the high-water mark over the operations one
// splice committed. Identities only grow, so the window maximum suffices and
// no later splice rescans the page. A splice that keeps operation identities
// can commit a window maximum below the page maximum, and a placement built
// before the mark existed carries no stored value, so the first commit over
// such a placement seeds the mark from the page once.
func storeOpIDHighWater(prev *Result, newOps []Op) {
	if prev.nextOpID == 0 {
		prev.nextOpID = relayoutMaxOpID(prev.Ops)
	}

	if maxID := maxOpIDInWindow(newOps); maxID > prev.nextOpID {
		prev.nextOpID = maxID
	}
}

// maxOpIDInWindow returns the highest operation identity in a committed
// window.
func maxOpIDInWindow(newOps []Op) uint64 {
	var maxID uint64

	for index := range newOps {
		if newOps[index].ID > maxID {
			maxID = newOps[index].ID
		}
	}

	return maxID
}

// spliceDamage unions the bounds of the operations in one window as
// x, y, w, h. An empty window or one with no painted extent reports false.
// Hosts repaint the union of the old and the new damage of a splice.
func spliceDamage(ops []Op, start, count int) ([4]float64, bool) {
	if start < 0 {
		count += start
		start = 0
	}

	end := start + count
	if end > len(ops) {
		end = len(ops)
	}

	var bounds [4]float64

	found := false

	for index := start; index < end; index++ {
		op := &ops[index]

		if op.W <= 0 && op.H <= 0 {
			continue
		}

		x0, y0 := op.X, op.Y
		x1, y1 := op.X+op.W, op.Y+op.H

		if !found {
			bounds = [4]float64{x0, y0, x1 - x0, y1 - y0}
			found = true

			continue
		}

		if x0 < bounds[0] {
			bounds[2] += bounds[0] - x0
			bounds[0] = x0
		}

		if y0 < bounds[1] {
			bounds[3] += bounds[1] - y0
			bounds[1] = y0
		}

		if x1 > bounds[0]+bounds[2] {
			bounds[2] = x1 - bounds[0]
		}

		if y1 > bounds[1]+bounds[3] {
			bounds[3] = y1 - bounds[1]
		}
	}

	return bounds, found
}

// boxThread threads element ids to their boxes and each box to its parent,
// so a batch resolves every id in place instead of rewalking the tree per
// id. Refreshing the committed subtree keeps every other pointer valid.
type boxThread struct {
	byID   map[string]*box
	parent map[*box]*box
}

// newBoxThread indexes the box tree in document order over children and
// rows, the coverage findBoxByID uses. The first box wins a repeated id,
// the rule elementByID uses.
func newBoxThread(root *box) *boxThread {
	thread := &boxThread{
		byID:   make(map[string]*box),
		parent: make(map[*box]*box),
	}

	thread.index(root, nil)

	return thread
}

// index hangs a subtree off its holder, recording every element id and
// parent pointer under it.
func (thread *boxThread) index(node, holder *box) {
	if node == nil {
		return
	}

	if holder != nil {
		thread.parent[node] = holder
	}

	if node.node != nil && node.node.Type == html.ElementNode {
		if id := node.node.Attribute("id"); id != "" {
			if _, taken := thread.byID[id]; !taken {
				thread.byID[id] = node
			}
		}
	}

	for _, child := range node.children {
		thread.index(child, node)
	}

	for _, row := range node.rows {
		for _, cell := range row {
			thread.index(cell, node)
		}
	}
}

// unindex drops every pointer into a detached subtree. An id entry moves
// only when it points into the subtree, so a repeated id owned elsewhere
// keeps its box.
func (thread *boxThread) unindex(node *box) {
	if node == nil {
		return
	}

	if node.node != nil && node.node.Type == html.ElementNode {
		if id := node.node.Attribute("id"); id != "" {
			if thread.byID[id] == node {
				delete(thread.byID, id)
			}
		}
	}

	delete(thread.parent, node)

	for _, child := range node.children {
		thread.unindex(child)
	}

	for _, row := range node.rows {
		for _, cell := range row {
			thread.unindex(cell)
		}
	}
}

// replace swaps the indexed subtree: the old box leaves the thread and the
// new box takes its place under the same holder.
func (thread *boxThread) replace(oldBox, newBox, holder *box) {
	thread.unindex(oldBox)
	thread.index(newBox, holder)
}

// lookup returns the relayout target for one id: the box, its holder, and
// the boxes above it from the root down. It reports false when the id owns
// no box.
func (thread *boxThread) lookup(elementID string) (*relayoutTarget, bool) {
	boxNode, ok := thread.byID[elementID]
	if !ok || boxNode == nil {
		return nil, false
	}

	var ancestors []*box

	for current, ok := thread.parent[boxNode]; ok; current, ok = thread.parent[current] {
		ancestors = append(ancestors, current)
	}

	for i, j := 0, len(ancestors)-1; i < j; i, j = i+1, j-1 {
		ancestors[i], ancestors[j] = ancestors[j], ancestors[i]
	}

	var holder *box

	if len(ancestors) > 0 {
		holder = ancestors[len(ancestors)-1]
	}

	return &relayoutTarget{box: boxNode, parent: holder, ancestors: ancestors}, true
}

// batchNodeForID resolves one batch id to its element node: the threaded box
// node when the element owns a box, else the document tree node. It returns
// nil when no element carries the id.
func batchNodeForID(thread *boxThread, prev *Result, elementID string) *html.Node {
	if target, ok := thread.lookup(elementID); ok && target.box != nil && target.box.node != nil {
		return target.box.node
	}

	return findNodeByID(prev.documentRootNode(), elementID)
}

// subtreeHasTextNode reports whether the subtree holds any text node. The
// font fast path needs it: without text no shaping runs.
func subtreeHasTextNode(node *html.Node) bool {
	if node == nil {
		return false
	}

	for _, child := range node.Children {
		if child.Type == html.TextNode && child.Text != "" {
			return true
		}

		if subtreeHasTextNode(child) {
			return true
		}
	}

	return false
}

// countElementBoxes counts the element boxes in a subtree over children
// only, the coverage the placed and owners walks use.
func countElementBoxes(root *box) int {
	if root == nil {
		return 0
	}

	count := 0

	if root.node != nil && root.node.Type == html.ElementNode {
		count = 1
	}

	for _, child := range root.children {
		count += countElementBoxes(child)
	}

	return count
}

// SpliceWindow describes one committed result-only splice: the operation
// window it replaced and the damage it paints. Damage is op-window-only:
// the union of the removed operations' bounds and of the committed ones,
// never a box or page scan. A host repaints the union of the two.
type SpliceWindow struct {
	Start, OldCount, NewCount  int
	OldDamage, NewDamage       [4]float64
	HasOldDamage, HasNewDamage bool
	// PaintOnly reports that the splice kept its operation count and every
	// pagination, owner, and paint-order input, so it overwrote the window in
	// place instead of rebuilding the box list, the ranges, the owners, and
	// the order.
	PaintOnly bool
}

// PaintOnlyText carries the refreshed box text of one paint-only id: the
// element's own subtree text first, then each element ancestor's text from the
// nearest upward. The host applies it to Display.Boxes by walking ParentIndex,
// so a text edit reaches the boxes of the element and of its ancestors
// without rebuilding the whole box list.
type PaintOnlyText struct {
	ID    string
	Texts []string
}

// SubtreeBatchResult is the outcome of RelayoutSubtrees: the owners and the
// paint order maintained incrementally across every splice, one window
// descriptor per committed id, and the count of patched subtrees that shape
// no text.
type SubtreeBatchResult struct {
	Owners, Order []int
	Windows       []SpliceWindow
	Textless      int
	// PaintOnly reports that every window was committed in place, so the
	// result's box list, operation ranges, owners, order, and page breaks are
	// the previous ones. The host can build the patched display by reusing
	// prev's boxes, breaks, keyframes, and size.
	PaintOnly bool
	// Texts carries the refreshed box text of each paint-only id, in batch
	// order. It is empty when the batch fell back to a full splice.
	Texts []PaintOnlyText
}

// ErrBatchUnknownID means a batch id names no element of the document. The
// public batch maps it onto its own no-element error.
var ErrBatchUnknownID = errors.New("layout: batch element id is unknown")

// RelayoutSubtrees re-lays out several element subtrees inside prev in one
// call, threading one placement through every id. Each splice is result-only:
// it mutates the placement and maintains the working owners and order, and
// no display is built until the caller builds one from the patched result.
//
// owners and order are the working slices the batch maintains in place; the
// caller passes disposable copies (usually of a display's slices) and adopts
// the returned ones. baseOpts carries the viewport, media, and zoom every
// splice shares; the font is loaded once and installed per element only when
// its subtree shapes text.
//
// A missing or refused id aborts the whole batch with the same error a
// single RelayoutSubtree call would return for it, and prev keeps the state
// its last committed splice left: the caller works on a private clone and
// drops it, so the input display is untouched. The ids run sequentially:
// each splice mutates the placement the next splice reads.
func RelayoutSubtrees(
	ctx context.Context, prev *Result, elementIDs []string,
	baseOpts Options, owners, order []int,
) (*SubtreeBatchResult, error) {
	if len(elementIDs) == 0 {
		return &SubtreeBatchResult{Owners: owners, Order: order}, nil //nolint:exhaustruct // an empty batch carries no windows
	}

	if prev == nil || prev.boxTreeRoot() == nil {
		return nil, errNoPreviousLayout
	}

	thread := newBoxThread(prev.boxTreeRoot())
	relevant, grouped := countOrderingRelevant(prev.Ops)

	out := &SubtreeBatchResult{
		Owners:    owners,
		Order:     order,
		Windows:   make([]SpliceWindow, 0, len(elementIDs)),
		PaintOnly: true,
	}

	// ownersDirty marks a deferred owners derivation: a row-granular splice
	// replaces out.Owners with a full derivation, so consecutive row splices
	// would each pay a page-sized derivation the next splice overwrites
	// unread. The derivation is a pure function of the committed placement,
	// so the batch defers it until the owners are read or the batch ends. A
	// full splice flushes the deferred derivation before it maintains the
	// owners incrementally.
	var ownersDirty bool

	// paintTextChanged marks whether any committed paint-only window changed
	// text payloads. A background-only toggle keeps every OpText identical, so
	// the box texts the previous display already carries stay correct and the
	// refreshed chain can skip the whole-table ancestor copies.
	var paintTextChanged bool

	for _, elementID := range elementIDs {
		if err := relayoutSpliceOne(ctx, prev, thread, elementID, baseOpts, out, &relevant, &grouped, &ownersDirty, &paintTextChanged); err != nil {
			return nil, err
		}
	}

	if ownersDirty {
		out.Owners = fullOwnersFor(prev)
	}

	// Intermediate batch splices skip the page-sized flat-list rebuild; one
	// rebuild here covers every full splice. A paint-only batch never
	// rebuilds, the rule commitPaintOnlySplice follows.
	if !out.PaintOnly {
		relayoutKeepBoxes(prev)
	}

	if out.PaintOnly {
		out.Texts = paintOnlyTexts(thread, prev, elementIDs, paintTextChanged)
	}

	return out, nil
}

// relayoutSpliceOne re-lays out one batch id inside prev and commits the
// splice, maintaining the working owners and order and recording the window
// it replaced. Errors match the single-splice call: an unknown id is
// ErrBatchUnknownID, and every other failure is wrapped the way the public
// element patch wraps the engine error. ownersDirty marks a deferred owners
// derivation a row-granular splice left behind; a full splice flushes it
// before it maintains the owners. paintTextChanged reports whether any
// committed paint-only window changed text payloads, so the final text refresh
// can skip the ancestor copies when every window kept its text.
func relayoutSpliceOne(
	ctx context.Context, prev *Result, thread *boxThread, elementID string,
	baseOpts Options, out *SubtreeBatchResult, relevant, grouped *int, ownersDirty *bool, paintTextChanged *bool,
) error {
	if elementID == "" {
		return ErrBatchUnknownID
	}

	node := batchNodeForID(thread, prev, elementID)
	if node == nil {
		return ErrBatchUnknownID
	}

	opts := baseOpts

	if subtreeHasTextNode(node) {
		if opts.Font == nil {
			loaded, err := pdf.DefaultFont()
			if err != nil {
				return fmt.Errorf("layout: default font: %w", err)
			}

			opts.Font = loaded
		}
	} else {
		opts.Font = nil
		out.Textless++
	}

	if err := relayoutGuards(ctx, prev, node, opts); err != nil {
		return relayoutSpliceError(err)
	}

	target, ok := thread.lookup(elementID)
	if !ok {
		if tryRelayoutTableRowBatch(ctx, prev, thread, node, nil, opts, out, relevant, grouped, ownersDirty) {
			return nil
		}

		table, ok := tableAncestorTarget(prev.boxTreeRoot(), node)
		if !ok {
			return relayoutSpliceError(errElementNotPlaced)
		}

		target, node = table, table.box.node
	} else if target.box == nil || target.box.node != node {
		return relayoutSpliceError(errElementNotPlaced)
	} else if target.box.kind == boxKindCell {
		if tryRelayoutTableRowBatch(ctx, prev, thread, node, target, opts, out, relevant, grouped, ownersDirty) {
			return nil
		}

		table, ok := tableOwnerTarget(target)
		if !ok {
			return relayoutSpliceError(fmt.Errorf("%w: table cell with no owning table", errSubtreeNotSplicable))
		}

		target, node = table, table.box.node
	}

	// A size-changing id widens to the nearest ancestor that keeps its size,
	// the same escalation the single-element call runs, so a batch carrying
	// one resized element still commits instead of aborting.
	built, err := relayoutBuildEscalating(ctx, prev, target, opts)
	if err != nil {
		return relayoutSpliceError(err)
	}

	target, styles, eng, newBox := built.target, built.styles, built.eng, built.box

	start, oldCount, err := relayoutWindow(prev, target)
	if err != nil {
		return relayoutSpliceError(err)
	}

	oldDamage, hasOld := spliceDamage(prev.Ops, start, oldCount)

	// A geometry- and pagination-neutral patch overwrites its operation
	// window in place: the box list, the ranges, the owners, and the order all
	// stay put, so the host can build the display by reusing the previous one.
	if relayoutPaintOnlyWindow(prev, target, newBox, start, oldCount, eng.ops) {
		if paintTextChanged != nil && !samePaintOnlyText(prev.Ops[start:start+oldCount], eng.ops) {
			*paintTextChanged = true
		}

		commitPaintOnlySplice(prev, newBox, target, styles, start, oldCount, eng.ops)
		thread.replace(target.box, newBox, target.parent)

		newDamage, hasNew := spliceDamage(prev.Ops, start, oldCount)
		out.Windows = append(out.Windows, SpliceWindow{ //nolint:exhaustruct // damage flags default false
			Start:        start,
			OldCount:     oldCount,
			NewCount:     oldCount,
			OldDamage:    oldDamage,
			NewDamage:    newDamage,
			HasOldDamage: hasOld,
			HasNewDamage: hasNew,
			PaintOnly:    true,
		})

		return nil
	}

	out.PaintOnly = false

	oldRel, oldGrp := countOrderingRelevant(prev.Ops[start : start+oldCount])
	oldElem := countElementBoxes(target.box)
	pos := &splicePosition{seed: -1, chain: target.ancestors}

	// The batch owns its private clone, so it splices into the clone's own
	// operation list instead of copying the page-wide slice once per id.
	commitSubtreeSplice(prev, newBox, target, styles, start, oldCount, eng.ops, true, pos)
	thread.replace(target.box, newBox, target.parent)

	newRel, newGrp := countOrderingRelevant(eng.ops)
	*relevant += newRel - oldRel
	*grouped += newGrp - oldGrp

	if *ownersDirty {
		// A row-granular splice left the owners derivation deferred, so the
		// working slice is stale. Derive from the committed placement: the
		// derivation is the same fullOwnersFor the deferred splice would
		// have run, and the incremental maintenance below would reproduce
		// it from a fresh slice.
		out.Owners = fullOwnersFor(prev)
		*ownersDirty = false
	} else if pos.foundBase {
		out.Owners = spliceOwnerWindow(out.Owners, start, oldCount, len(eng.ops), pos.seed, pos.base, oldElem, newBox)
	} else {
		out.Owners = fullOwnersFor(prev)
	}

	neutral := oldRel == 0 && newRel == 0 && *grouped == 0
	out.Order = splicePaintOrder(prev.Ops, out.Order, start, oldCount, len(eng.ops), neutral)

	newDamage, hasNew := spliceDamage(prev.Ops, start, len(eng.ops))
	out.Windows = append(out.Windows, SpliceWindow{ //nolint:exhaustruct // damage flags default false
		Start:        start,
		OldCount:     oldCount,
		NewCount:     len(eng.ops),
		OldDamage:    oldDamage,
		NewDamage:    newDamage,
		HasOldDamage: hasOld,
		HasNewDamage: hasNew,
	})

	return nil
}

// relayoutSpliceError wraps one batch splice failure the way the public
// element patch wraps the engine error, so a refused batch id reports the
// same error the single call would.
func relayoutSpliceError(err error) error {
	return fmt.Errorf("layout: relayout element: %w", err)
}

// fullOwnersFor derives the owners from scratch, the fallback when the
// spliced subtree sits under a table row where the owners walk never visits.
// The lean walk skips text: only parent links feed the derivation.
func fullOwnersFor(prev *Result) []int {
	var placed []PlacedElement

	WalkPlacedWithOptions(prev, WalkPlacedOptions{IncludeText: false}, func(elem *PlacedElement) { //nolint:exhaustruct // the lean walk computes no text
		placed = append(placed, *elem)
	})

	return OperationOwnersPlaced(prev, placed)
}

// paintOnlyTexts collects the refreshed subtree text of every paint-only id
// from the committed box tree: the element's own text first, then each element
// ancestor's text from the nearest upward. The display applies it to the boxes
// it reuses by walking ParentIndex, so a text edit refreshes the element and
// its ancestors without a full placed walk. Ancestor texts are cached by node,
// because a batch of rows shares the same table, body, and document nodes.
// When no committed paint-only window changed text payloads, every box text
// the previous display carries is already correct: Box.Text is a pure function
// of the subtree and a class-only toggle leaves every text node alone, so the
// refresh returns one entry per resolved id with no texts and the caller keeps
// the previous boxes in place instead of copying whole-table strings.
func paintOnlyTexts(thread *boxThread, prev *Result, elementIDs []string, textChanged bool) []PaintOnlyText {
	if !textChanged {
		texts := make([]PaintOnlyText, 0, len(elementIDs))

		for _, elementID := range elementIDs {
			target, ok := thread.lookup(elementID)
			if !ok {
				ancestor, ok := paintOnlyAncestorTarget(thread, prev, elementID)
				if !ok {
					continue
				}

				if ancestor.box == nil || ancestor.box.node == nil {
					continue
				}

				texts = append(texts, PaintOnlyText{ID: elementID, Texts: nil})
			} else {
				if target.box == nil || target.box.node == nil {
					continue
				}

				texts = append(texts, PaintOnlyText{ID: elementID, Texts: nil})
			}
		}

		return texts
	}

	cache := make(map[*html.Node]string)
	texts := make([]PaintOnlyText, 0, len(elementIDs))

	for _, elementID := range elementIDs {
		target, ok := thread.lookup(elementID)
		if !ok {
			ancestor, ok := paintOnlyAncestorTarget(thread, prev, elementID)
			if !ok {
				continue
			}

			target = ancestor
		}

		if target.box == nil || target.box.node == nil {
			continue
		}

		chain := make([]string, 0, len(target.ancestors)+1)
		chain = append(chain, cachedNodeText(cache, target.box.node))

		for index := len(target.ancestors) - 1; index >= 0; index-- {
			ancestor := target.ancestors[index]
			if ancestor == nil || ancestor.node == nil || ancestor.node.Type != html.ElementNode {
				continue
			}

			chain = append(chain, cachedNodeText(cache, ancestor.node))
		}

		texts = append(texts, PaintOnlyText{ID: elementID, Texts: chain})
	}

	return texts
}

// paintOnlyAncestorTarget returns the nearest ancestor with an addressed box
// for a box-less id: it resolves the id to its document node and climbs to
// the nearest ancestor whose id addresses a box in the thread.
func paintOnlyAncestorTarget(thread *boxThread, prev *Result, elementID string) (*relayoutTarget, bool) {
	if thread == nil || prev == nil || elementID == "" {
		return nil, false
	}

	node := findNodeByID(prev.documentRootNode(), elementID)
	if node == nil {
		return nil, false
	}

	for current := node.Parent; current != nil; current = current.Parent {
		if current.Type != html.ElementNode {
			continue
		}

		id := current.Attribute("id")
		if id == "" {
			continue
		}

		if target, ok := thread.lookup(id); ok && target.box != nil && target.box.node != nil {
			return target, true
		}
	}

	return nil, false
}

// cachedNodeText returns one node's subtree text, computing it once per node.
func cachedNodeText(cache map[*html.Node]string, node *html.Node) string {
	if text, ok := cache[node]; ok {
		return text
	}

	text := node.TextContent()
	cache[node] = text

	return text
}

// samePaintOnlyText reports whether a committed paint-only window kept every
// text payload: the same number of operations with identical Text at each
// index. A background-only toggle repaints fills while leaving every text run
// alone, so equality proves no text node the window covers changed. An empty
// window on either side reports false: an element that emits no operation can
// still carry descendant text (the hidden-box path), so only the full text
// chain the caller builds today stays correct there.
func samePaintOnlyText(oldOps, newOps []Op) bool {
	if len(oldOps) == 0 || len(newOps) == 0 {
		return false
	}

	if len(oldOps) != len(newOps) {
		return false
	}

	for index := range oldOps {
		if oldOps[index].Text != newOps[index].Text {
			return false
		}
	}

	return true
}
