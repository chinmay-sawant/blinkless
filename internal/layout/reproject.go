package layout

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"

	"github.com/chinmay-sawant/blinkless/internal/errs"
)

// The restamp pass re-projects one or more element subtrees without running
// the cascade or the placement again. It exists because a 3D transform and a
// perspective property have no 2D affine equivalent: the projection is computed
// across a whole subtree at stamp time, so an animation of either one has to
// re-run that stamp instead of replacing a matrix.
//
// The pass mutates the retained placement in place, exactly as RelayoutSubtree
// does, and returns the operation windows it rewrote. A caller repaints those
// windows and adopts the new display the public wrapper builds.
var (
	// ErrNoElement means a restamp override named an id no laid-out box
	// carries.
	ErrNoElement = errors.New("layout: no element with that id")

	// ErrInvalidTransform means a restamp override carried a transform list
	// the cascade parser rejects. A malformed override is reported once and
	// changes nothing; a page never becomes half restamped.
	ErrInvalidTransform = errors.New("layout: invalid transform override")

	// ErrNoRetainedState means the restamp pass was given a placement that
	// keeps no box tree. A display only keeps one when it was laid out with
	// Options.RetainState.
	ErrNoRetainedState = errors.New("layout: display carries no retained layout state")

	// ErrNotStampable means the element cannot be re-projected on its own:
	// its box is not reachable through the children chain the stamp walk
	// follows, or the static path never stamps a transform declared on it
	// (a table cell).
	ErrNotStampable = errors.New("layout: element subtree cannot be restamped")

	// errInvalidPerspective means an override carried a negative or
	// non-finite perspective distance.
	errInvalidPerspective = errors.New("layout: invalid perspective override")
)

// RestampOverride replaces resolved transform values for one element during
// one restamp pass. It is the engine-side form of the public
// layout.TransformOverride.
//
// OriginX, OriginY, Perspective, and the perspective origin are in layout
// points. The origins are page coordinates: the vanishing point or the
// transform origin the stamp walk resolves to, not an offset inside the box.
type RestampOverride struct {
	ElementID string
	// Transform is a raw CSS transform list. HasTransform=false leaves the
	// element's transform untouched; HasTransform=true with "" means none.
	Transform    string
	HasTransform bool
	// OriginX and OriginY replace transform-origin in layout points.
	OriginX   float64
	OriginY   float64
	HasOrigin bool
	// Perspective replaces the perspective property this element gives its
	// descendants, in layout points. Zero means none.
	Perspective    float64
	HasPerspective bool
	// PerspectiveOriginX and PerspectiveOriginY replace perspective-origin
	// in layout points.
	PerspectiveOriginX   float64
	PerspectiveOriginY   float64
	HasPerspectiveOrigin bool
}

// OpWindow is one inclusive operation range a restamp pass rewrote.
type OpWindow struct {
	Start int
	End   int
}

// RestampPatch is the result of a restamp pass.
type RestampPatch struct {
	// Changed reports whether any override actually replaced a resolved
	// value. When it is false nothing was stamped and Windows is empty.
	Changed bool
	// Windows are the patched operation ranges, in page order.
	Windows []OpWindow
	// OldX, OldY, OldW, and OldH bound the painted area of the patched
	// subtrees before the pass, in layout points.
	OldX, OldY, OldW, OldH float64
	// NewX, NewY, NewW, and NewH bound that area after the pass, in layout
	// points.
	NewX, NewY, NewW, NewH float64
}

// RestampTransforms re-runs the perspective pre-pass and the transform stamp
// walk for the subtrees named by overrides, inside a placement that is already
// laid out. It never re-cascades and never re-places: the boxes keep their
// geometry, and only the transform state stamped on their operations changes.
//
// The overrides are applied for this pass only. The cached ResolvedStyle of
// every element is left untouched, so the next static frame is the one the
// cascade produced.
//
// An override list with no effective change returns Changed=false and leaves
// every operation as it was. Nested overrides are supported: an inner target
// is stamped inside the outer target's subtree walk, so the inner value wins
// for its own subtree and inherits the outer camera.
func RestampTransforms(
	ctx context.Context, prev *Result, overrides []RestampOverride, opts Options,
) (*RestampPatch, error) {
	ready, err := restampReady(ctx, prev, overrides)
	if err != nil {
		return nil, err
	}

	if !ready {
		return &RestampPatch{}, nil
	}

	scratch := getRestampScratch()
	defer putRestampScratch(scratch)

	plan, err := planRestamp(prev, overrides, scratch)
	if err != nil {
		return nil, err
	}

	if !plan.changed {
		return &RestampPatch{}, nil
	}

	roots := plan.outermost(prev.boxTreeRoot(), scratch)
	if len(roots) == 0 {
		return &RestampPatch{}, nil
	}

	// The styles are swapped in before the seed walk: an ancestor of a
	// nested target has to show the effective perspective it overrides.
	plan.install()

	defer plan.restore()

	states, err := restampStates(prev, roots, scratch)
	if err != nil {
		return nil, err
	}

	if len(states) == 0 {
		return &RestampPatch{}, nil
	}

	return applyRestamp(prev, states, opts.Media, scratch)
}

// restampScratch holds reusable per-pass slices and maps so steady-state
// animation frames avoid per-frame heap allocations.
type restampScratch struct {
	targets        []restampTarget
	targetStyles   []ResolvedStyle
	targetEffects  []styleEffects
	byBox          map[*box]int
	targetMap      map[*box]int
	seen           []bool
	roots          []*restampTarget
	states         []subtreeStamp
	ancestors      []*box
	actions        []ancestorStamp
	backupOps      []Op
	backupExtras   []opExtra
	backupExtraIdx []int
	backupGridSegs []GridSeg
	backupGrids    []gridBackup
	backupRanges   []opBackupRange
}

type gridBackup struct {
	opIdx    int
	segStart int
	segLen   int
}

type opBackupRange struct {
	windowStart int
	opsStart    int
	opsLen      int
}

var restampScratchPool = sync.Pool{
	New: func() any {
		return &restampScratch{
			ancestors: make([]*box, 0, 16),
		}
	},
}

func getRestampScratch() *restampScratch {
	return restampScratchPool.Get().(*restampScratch)
}

func putRestampScratch(s *restampScratch) {
	clear(s.targets)
	s.targets = s.targets[:0]
	clear(s.targetStyles)
	s.targetStyles = s.targetStyles[:0]
	clear(s.targetEffects)
	s.targetEffects = s.targetEffects[:0]
	if len(s.byBox) > 0 {
		clear(s.byBox)
	}
	if len(s.targetMap) > 0 {
		clear(s.targetMap)
	}
	s.seen = s.seen[:0]
	clear(s.roots)
	s.roots = s.roots[:0]
	clear(s.states)
	s.states = s.states[:0]
	clear(s.ancestors)
	s.ancestors = s.ancestors[:0]
	clear(s.actions)
	s.actions = s.actions[:0]
	clear(s.backupOps)
	s.backupOps = s.backupOps[:0]
	clear(s.backupExtras)
	s.backupExtras = s.backupExtras[:0]
	s.backupExtraIdx = s.backupExtraIdx[:0]
	s.backupGridSegs = s.backupGridSegs[:0]
	s.backupGrids = s.backupGrids[:0]
	s.backupRanges = s.backupRanges[:0]
	restampScratchPool.Put(s)
}

// restampReady rejects the inputs the pass cannot run on. A false answer
// without an error means there is nothing to do.
func restampReady(ctx context.Context, prev *Result, overrides []RestampOverride) (bool, error) {
	if ctx == nil {
		return false, errs.ErrNilContext
	}

	if err := ctx.Err(); err != nil {
		return false, fmt.Errorf("layout: context: %w", err)
	}

	if len(overrides) == 0 {
		return false, nil
	}

	if prev == nil || prev.boxTreeRoot() == nil {
		return false, ErrNoRetainedState
	}

	return true, nil
}

// restampStates resolves the seed state and the ancestor stamps of every
// outermost subtree, and the operation window each one owns.
func restampStates(prev *Result, roots []*restampTarget, scratch *restampScratch) ([]subtreeStamp, error) {
	states := scratch.states[:0]
	scratch.actions = scratch.actions[:0]
	root := prev.boxTreeRoot()

	for _, target := range roots {
		actionStart := len(scratch.actions)
		state, perspAncestor, hasState := stampStateAtInto(root, target.box, &scratch.ancestors, &scratch.actions)
		if !hasState {
			return nil, fmt.Errorf("%w: %s", ErrNotStampable, target.id)
		}

		actionEnd := len(scratch.actions)

		window, hasWindow := opWindowFor(target.box, len(prev.Ops))
		if !hasWindow {
			scratch.actions = scratch.actions[:actionStart]

			continue
		}

		states = append(states, subtreeStamp{
			box:           target.box,
			window:        window,
			state:         state,
			actionStart:   actionStart,
			actionEnd:     actionEnd,
			perspAncestor: perspAncestor,
		})
	}

	scratch.states = states

	return states, nil
}

// applyRestamp rewrites every subtree window and finalizes the projected
// geometry it produced. A failure rolls every window back, because a display
// list that is half projected paints wrong pixels.
func applyRestamp(prev *Result, states []subtreeStamp, media string, scratch *restampScratch) (*RestampPatch, error) {
	windows := make([]OpWindow, 0, len(states))
	for idx := range states {
		windows = append(windows, states[idx].window)
	}

	oldBounds := unionWindowBounds(prev.Ops, windows)

	scratch.backupOps = scratch.backupOps[:0]
	scratch.backupExtras = scratch.backupExtras[:0]
	scratch.backupExtraIdx = scratch.backupExtraIdx[:0]
	scratch.backupGridSegs = scratch.backupGridSegs[:0]
	scratch.backupGrids = scratch.backupGrids[:0]
	scratch.backupRanges = scratch.backupRanges[:0]
	committed := false

	defer func() {
		if committed {
			return
		}

		restoreBackups(prev.Ops, scratch)
	}()

	for idx := range windows {
		backupWindowInto(prev.Ops, windows[idx], scratch)
	}

	for idx := range states {
		if err := restampSubtree(prev, &states[idx], scratch.actions, media); err != nil {
			return nil, err
		}
	}

	committed = true

	newBounds := unionWindowBounds(prev.Ops, windows)

	return &RestampPatch{
		Changed: true,
		Windows: windows,
		OldX:    oldBounds.x,
		OldY:    oldBounds.y,
		OldW:    oldBounds.w,
		OldH:    oldBounds.h,
		NewX:    newBounds.x,
		NewY:    newBounds.y,
		NewW:    newBounds.w,
		NewH:    newBounds.h,
	}, nil
}

// restampSubtree resets and re-stamps one subtree window.
func restampSubtree(prev *Result, item *subtreeStamp, allActions []ancestorStamp, media string) error {
	resetStampFields(item.box)
	propagatePerspectiveFrom(item.box, item.perspAncestor)
	resetWindowTransforms(prev.Ops, item.window)

	restampBoxTransforms(item.box, item.state, prev.Ops, media)

	for actionIdx := item.actionStart; actionIdx < item.actionEnd; actionIdx++ {
		applyAncestorStamp(allActions[actionIdx], prev.Ops, item.window)
	}

	if err := FinalizeDisplayOps(prev.Ops[item.window.Start : item.window.End+1]); err != nil {
		return fmt.Errorf("layout: restamp: %w", err)
	}

	return nil
}

// restampTarget is one element an override applies to, with the effective
// style this pass stamps.
type restampTarget struct {
	id       string
	box      *box
	original *ResolvedStyle
	style    *ResolvedStyle
}

// restampPlan is the resolved override list.
type restampPlan struct {
	targets []restampTarget
	changed bool
}

// planRestamp resolves every override against the retained box tree and builds
// one effective style per element. Overrides for the same element apply in
// order: the last one wins for a field it sets.
func planRestamp(prev *Result, overrides []RestampOverride, scratch *restampScratch) (restampPlan, error) {
	if cap(scratch.targetStyles) < len(overrides) {
		scratch.targetStyles = make([]ResolvedStyle, len(overrides))
	} else {
		scratch.targetStyles = scratch.targetStyles[:len(overrides)]
	}

	if cap(scratch.targetEffects) < len(overrides) {
		scratch.targetEffects = make([]styleEffects, len(overrides))
	} else {
		scratch.targetEffects = scratch.targetEffects[:len(overrides)]
	}

	plan := restampPlan{
		targets: scratch.targets[:0],
	}

	var byBox map[*box]int
	if len(overrides) > 8 {
		if scratch.byBox == nil {
			scratch.byBox = make(map[*box]int, len(overrides))
		} else {
			clear(scratch.byBox)
		}

		byBox = scratch.byBox
	}

	for idx := range overrides {
		override := &overrides[idx]

		if override.ElementID == "" {
			return restampPlan{}, fmt.Errorf("%w: empty element id", ErrNoElement)
		}

		boxNode := prev.boxForID(override.ElementID)
		if boxNode == nil {
			return restampPlan{}, fmt.Errorf("%w: %s", ErrNoElement, override.ElementID)
		}

		if boxNode.style == nil {
			return restampPlan{}, fmt.Errorf("%w: %s has no resolved style", ErrNotStampable, override.ElementID)
		}

		// A table cell is placed by the table layout, and the static path
		// does not run the stamp walk for a transform declared on the cell
		// alone. Restamping it would paint a projection the authored page
		// never shows, so the pass refuses it.
		if boxNode.kind == boxKindCell {
			return restampPlan{}, fmt.Errorf("%w: %s is a table cell", ErrNotStampable, override.ElementID)
		}

		target := plan.targetFor(boxNode, override.ElementID, byBox, scratch)
		if err := applyRestampOverride(target.style, target.box, override); err != nil {
			return restampPlan{}, err
		}
	}

	scratch.targets = plan.targets

	for idx := range plan.targets {
		if !sameTransformState(plan.targets[idx].style, plan.targets[idx].original) {
			plan.changed = true

			break
		}
	}

	return plan, nil
}

// targetFor returns the plan entry for one box, cloning its cached style into
// scratch on first use without heap-allocating per frame.
func (p *restampPlan) targetFor(
	boxNode *box, elementID string, byBox map[*box]int, scratch *restampScratch,
) *restampTarget {
	if byBox != nil {
		if index, ok := byBox[boxNode]; ok {
			return &p.targets[index]
		}
	} else {
		for idx := range p.targets {
			if p.targets[idx].box == boxNode {
				return &p.targets[idx]
			}
		}
	}

	slot := len(p.targets)
	style := &scratch.targetStyles[slot]
	*style = *boxNode.style
	style.groups = nil
	effects := &scratch.targetEffects[slot]
	*effects = *boxNode.style.effects()
	style.effectsGroup = effects

	if byBox != nil {
		byBox[boxNode] = slot
	}

	p.targets = append(p.targets, restampTarget{
		id:       elementID,
		box:      boxNode,
		original: boxNode.style,
		style:    style,
	})

	return &p.targets[slot]
}

// install points every target box at its effective style for the pass.
func (p *restampPlan) install() {
	for idx := range p.targets {
		p.targets[idx].box.style = p.targets[idx].style
	}
}

// restore puts the cached styles back.
func (p *restampPlan) restore() {
	for idx := len(p.targets) - 1; idx >= 0; idx-- {
		p.targets[idx].box.style = p.targets[idx].original
	}
}

// outermost returns the targets that no other target contains, in document
// order. A nested target is stamped inside its outer target's subtree walk.
func (p *restampPlan) outermost(root *box, scratch *restampScratch) []*restampTarget {
	if len(p.targets) == 1 {
		scratch.roots = append(scratch.roots[:0], &p.targets[0])

		return scratch.roots
	}

	if scratch.targetMap == nil {
		scratch.targetMap = make(map[*box]int, len(p.targets))
	} else {
		clear(scratch.targetMap)
	}

	targets := scratch.targetMap
	for idx := range p.targets {
		targets[p.targets[idx].box] = idx
	}

	if cap(scratch.seen) < len(p.targets) {
		scratch.seen = make([]bool, len(p.targets))
	} else {
		scratch.seen = scratch.seen[:len(p.targets)]
		clear(scratch.seen)
	}

	seen := scratch.seen
	out := scratch.roots[:0]

	var walk func(boxNode *box, inside bool)
	walk = func(boxNode *box, inside bool) {
		index, isTarget := targets[boxNode]

		if isTarget {
			seen[index] = true

			if !inside {
				out = append(out, &p.targets[index])
			}
		}

		next := inside || isTarget

		for _, child := range boxNode.children {
			walk(child, next)
		}
	}

	walk(root, false)

	// A box that hangs off the children chain (a table cell lives in its row
	// list) is handed back too, so the caller reports it as not stampable
	// instead of silently ignoring the override.
	for idx := range p.targets {
		if !seen[idx] {
			out = append(out, &p.targets[idx])
		}
	}

	scratch.roots = out

	return out
}

// applyRestampOverride writes one override onto an effective style.
func applyRestampOverride(style *ResolvedStyle, boxNode *box, override *RestampOverride) error {
	if override.HasTransform {
		if err := applyTransformOverride(style, override.Transform); err != nil {
			return err
		}
	}

	if override.HasOrigin {
		style.applyOriginOverride(boxNode, override.OriginX, override.OriginY)
	}

	if override.HasPerspective {
		if err := style.applyPerspectiveOverride(override.Perspective); err != nil {
			return err
		}
	}

	if override.HasPerspectiveOrigin {
		style.applyPerspectiveOriginOverride(boxNode, override.PerspectiveOriginX, override.PerspectiveOriginY)
	}

	return nil
}

// restampEffects returns a mutable styleEffects for a cloned restamp style.
// targetFor pre-binds style.effectsGroup to a scratch slot so steady-state
// restamps never call editEffects (which heap-allocates when groups is nil).
func (style *ResolvedStyle) restampEffects() *styleEffects {
	if style.effectsGroup != nil {
		return style.effectsGroup
	}

	return style.editEffects()
}

// applyTransformOverride replaces the element's used transform with one parsed
// CSS list. The cascade parser is reused, so a 2D and a 3D function take the
// same path they take in a stylesheet.
func applyTransformOverride(style *ResolvedStyle, value string) error {
	transform := strings.TrimSpace(value)

	matrix, xPercent, yPercent, has, ok, accum3D, has3D := parseTransformList3D(transform, style.FontSize)
	if !ok && transform != "" {
		return fmt.Errorf("%w: %q", ErrInvalidTransform, value)
	}

	if transform == "" {
		matrix, has = IdentityMatrix(), false
	}

	effects := style.restampEffects()
	effects.Transform = matrix
	effects.HasTransform = has
	effects.Transform3D = accum3D
	effects.HasTransform3D = has3D

	effects.TranslateXPercent, effects.TranslateXPercentSet = 0, false
	if !math.IsNaN(xPercent) {
		effects.TranslateXPercent, effects.TranslateXPercentSet = xPercent, true
	}

	effects.TranslateYPercent, effects.TranslateYPercentSet = 0, false
	if !math.IsNaN(yPercent) {
		effects.TranslateYPercent, effects.TranslateYPercentSet = yPercent, true
	}

	return nil
}

// applyOriginOverride replaces transform-origin with the resolved page point
// the caller hands back. The offset from the border box lands on that point, so
// the override bypasses the transform-box reference box.
func (style *ResolvedStyle) applyOriginOverride(boxNode *box, originX, originY float64) {
	effects := style.restampEffects()
	effects.TransformOrigin = transformOriginSpec{
		X:        originX - boxNode.x,
		Y:        originY - boxNode.y,
		XPercent: false,
		YPercent: false,
	}
	effects.TransformBox = "border-box"
}

// applyPerspectiveOverride replaces the perspective distance in layout points.
// Zero means none.
func (style *ResolvedStyle) applyPerspectiveOverride(distance float64) error {
	if math.IsNaN(distance) || math.IsInf(distance, 0) || distance < 0 {
		return fmt.Errorf("%w: %g", errInvalidPerspective, distance)
	}

	effects := style.restampEffects()
	effects.PerspectiveDist = distance
	effects.HasPerspective = distance > 0

	return nil
}

// applyPerspectiveOriginOverride replaces perspective-origin with the resolved
// page point the caller hands back.
func (style *ResolvedStyle) applyPerspectiveOriginOverride(boxNode *box, originX, originY float64) {
	effects := style.restampEffects()
	effects.PerspectiveOriginX = originX - boxNode.x
	effects.PerspectiveOriginXPct = false
	effects.PerspectiveOriginY = originY - boxNode.y
	effects.PerspectiveOriginYPct = false
	effects.PerspectiveOriginSet = true
}

// restampStyleValues is the transform slice of a ResolvedStyle. It is a
// comparable snapshot so a no-op override can be detected with one compare.
type restampStyleValues struct {
	transform             Matrix2D
	origin                transformOriginSpec
	transform3D           [16]float64
	translateXPercent     float64
	translateYPercent     float64
	perspective           float64
	perspectiveOriginX    float64
	perspectiveOriginY    float64
	transformBox          string
	hasTransform          bool
	hasTransform3D        bool
	translateXPercentSet  bool
	translateYPercentSet  bool
	hasPerspective        bool
	perspectiveOriginXPct bool
	perspectiveOriginYPct bool
	perspectiveOriginSet  bool
}

// styleValues snapshots the transform state one style declares.
func styleValues(style *ResolvedStyle) restampStyleValues {
	return restampStyleValues{
		transform:             style.effects().Transform,
		origin:                style.effects().TransformOrigin,
		transform3D:           style.effects().Transform3D,
		translateXPercent:     style.effects().TranslateXPercent,
		translateYPercent:     style.effects().TranslateYPercent,
		perspective:           style.effects().PerspectiveDist,
		perspectiveOriginX:    style.effects().PerspectiveOriginX,
		perspectiveOriginY:    style.effects().PerspectiveOriginY,
		transformBox:          style.effects().TransformBox,
		hasTransform:          style.effects().HasTransform,
		hasTransform3D:        style.effects().HasTransform3D,
		translateXPercentSet:  style.effects().TranslateXPercentSet,
		translateYPercentSet:  style.effects().TranslateYPercentSet,
		hasPerspective:        style.effects().HasPerspective,
		perspectiveOriginXPct: style.effects().PerspectiveOriginXPct,
		perspectiveOriginYPct: style.effects().PerspectiveOriginYPct,
		perspectiveOriginSet:  style.effects().PerspectiveOriginSet,
	}
}

// sameTransformState reports whether the effective style resolves to the same
// transform values as the cached style. A no-op override then costs nothing.
func sameTransformState(effective, cached *ResolvedStyle) bool {
	return styleValues(effective) == styleValues(cached)
}

// subtreeStamp is the work of one outermost override subtree.
type subtreeStamp struct {
	box           *box
	window        OpWindow
	state         stampState
	actionStart   int
	actionEnd     int
	perspAncestor *box
}

// ancestorStamp is one ancestor stamp the original walk applied after the
// subtree's own stamp. The restamp replays it so an enclosing 3D projection or
// flat-boundary composition keeps wrapping the new op matrices.
type ancestorStamp struct {
	box  *box
	step stampBoxStep
}

// findBoxAncestors appends the ancestor chain from cur down to (but not
// including) target into path and reports whether target was found.
func findBoxAncestors(cur, target *box, path *[]*box) bool {
	if cur == target {
		return true
	}

	*path = append(*path, cur)

	for _, child := range cur.children {
		if findBoxAncestors(child, target, path) {
			return true
		}
	}

	*path = (*path)[:len(*path)-1]

	return false
}

// stampStateAt returns the state the full walk had when it reached target, and
// the ancestor stamps it applied after target's subtree.
func stampStateAt(root, target *box) (stampState, []ancestorStamp, bool) {
	scratch := make([]*box, 0, 16)
	state, actions, _, found := stampStateAtWithScratch(root, target, &scratch)

	return state, actions, found
}

// stampStateAtWithScratch resolves target's seed stamp state, reversed ancestor
// stamps, and nearest perspective ancestor in two passes: a lightweight DFS
// finds the ancestor chain without advancing stamp state on unrelated subtrees,
// then advanceStamp runs only along that chain.
func stampStateAtWithScratch(root, target *box, scratch *[]*box) (stampState, []ancestorStamp, *box, bool) {
	var actions []ancestorStamp
	current, perspAncestor, found := stampStateAtInto(root, target, scratch, &actions)
	if !found {
		return stampState{}, nil, nil, false
	}

	return current, actions, perspAncestor, true
}

// stampStateAtInto appends reversed ancestor stamps into *actions instead of
// allocating a fresh slice per target.
func stampStateAtInto(
	root, target *box, scratch *[]*box, actions *[]ancestorStamp,
) (stampState, *box, bool) {
	if root == nil || target == nil {
		return stampState{}, nil, false
	}

	*scratch = (*scratch)[:0]
	if !findBoxAncestors(root, target, scratch) {
		return stampState{}, nil, false
	}

	ancestors := *scratch
	current := initialStampState(IdentityMatrix())

	start := len(*actions)
	if len(ancestors) > 0 {
		*actions = slices.Grow(*actions, len(ancestors))
		*actions = (*actions)[:start+len(ancestors)]
	}

	var perspAncestor *box

	for idx, boxNode := range ancestors {
		if boxNode.style != nil && boxNode.style.effects().HasPerspective {
			perspAncestor = boxNode
		}

		step := advanceStamp(boxNode, current)
		(*actions)[start+len(ancestors)-1-idx] = ancestorStamp{box: boxNode, step: step}
		current = step.child
	}

	return current, perspAncestor, true
}

// applyAncestorStamp replays one ancestor stamp over a restamped window.
func applyAncestorStamp(action ancestorStamp, ops []Op, window OpWindow) {
	if action.step.preservePath {
		if action.step.chainActive && !action.step.isPreserve {
			stampPreserveRangeWindow(
				action.box, action.step.worldChain, action.step.parentAccum, ops,
				window.Start, window.End, action.step.viewport,
			)
		}

		return
	}

	projective, ok := projectiveStampForBox(action.box, action.step.parentAccum, action.step.viewport)
	if !ok {
		return
	}

	for idx := window.Start; idx <= window.End; idx++ {
		if ops[idx].Kind == OpKindNoop {
			continue
		}

		ops[idx].setProjectiveTransform(projective)
	}
}

// resetStampFields drops the computed state a previous stamp left on one
// subtree. preserveComposed is derived from the chain above the box and the
// perspective fields from the nearest perspective ancestor, so both are
// recomputed rather than adjusted.
func resetStampFields(root *box) {
	if root == nil {
		return
	}

	root.preserveComposed = false

	for _, child := range root.children {
		resetStampFields(child)
	}
}

// opWindowFor returns the operation range a box's subtree emitted.
func opWindowFor(boxNode *box, opCount int) (OpWindow, bool) {
	if boxNode == nil || !boxOwnsOps(boxNode) {
		return OpWindow{}, false
	}

	start := boxNode.opStart
	end := min(boxNode.opEnd, opCount-1)

	if start < 0 || start > end {
		return OpWindow{}, false
	}

	return OpWindow{Start: start, End: end}, true
}

// resetWindowTransforms restores every operation in a window to the geometry
// it carried before finalize baked a projection into it, then drops the
// transform the previous stamp applied. The restamp walk rebuilds both.
func resetWindowTransforms(ops []Op, window OpWindow) {
	for idx := window.Start; idx <= window.End; idx++ {
		restorePristineGeometry(&ops[idx])
		ops[idx].XformSet = false
		ops[idx].clearProjectiveTransform()

		// setXform always detaches before writing, so every op that ever
		// carried a transform owns its extra. An op still pointing at the
		// shared zero singleton never had one, and the singleton is the
		// no-transform value by definition. Skip it: writing the shared
		// Xform would race with any other display reading Transform(), and
		// detaching would allocate for every op the restamp never touched.
		if extra := ops[idx].opExtra; extra != nil && extra != emptyExtra {
			extra.Xform = Matrix2D{}
		}
	}
}

// savePristineGeometry records the operation before FinalizeDisplayOps rewrites
// it into final geometry, so a restamp can project it again from its flat form.
// The copy detaches the rare payload and clones the grid segments, which the
// finalize pass rewrites in place. When the operation already carries a saved
// pristine snapshot from an earlier pass, it is reused without reallocating.
func (op *Op) savePristineGeometry() {
	if op == nil {
		return
	}

	if op.opExtra != nil && op.opExtra != emptyExtra && op.opExtra.Pristine != nil {
		return
	}

	extra := op.detachExtra()
	saved := op.Clone()
	saved.Grid = cloneGridRun(op.Grid)

	if saved.opExtra != nil {
		saved.opExtra.Pristine = nil
	}

	extra.Pristine = &saved
}

// restorePristineGeometry puts the pre-finalize form of one operation back
// while keeping the op's own opExtra and its Pristine pointer intact, so the
// next FinalizeDisplayOps reuses the saved snapshot with zero allocations.
func restorePristineGeometry(op *Op) bool {
	if op == nil || op.opExtra == nil || op.opExtra.Pristine == nil {
		return false
	}

	extra := op.opExtra
	pristine := extra.Pristine
	grid := op.Grid

	*op = *pristine
	if pristine.opExtra != nil {
		*extra = *pristine.opExtra
	} else {
		*extra = opExtra{}
	}

	extra.Pristine = pristine
	op.opExtra = extra

	if pristine.Grid != nil {
		if grid != nil && grid != pristine.Grid && cap(grid.Segs) >= len(pristine.Grid.Segs) {
			grid.Segs = grid.Segs[:len(pristine.Grid.Segs)]
			copy(grid.Segs, pristine.Grid.Segs)
			op.Grid = grid
		} else {
			op.Grid = cloneGridRun(pristine.Grid)
		}
	}

	return true
}

// cloneGridRun copies a collapsed border run so the saved operation cannot see
// the projected segments finalize writes in place.
func cloneGridRun(run *GridRun) *GridRun {
	if run == nil {
		return nil
	}

	return &GridRun{Segs: slices.Clone(run.Segs)}
}

// backupWindowInto records a window's operations into scratch so a failed pass
// can roll them back without heap-allocating per op on the happy path.
// FinalizeDisplayOps only mutates opExtra fields and Grid.Segs in place
// (Path.Points, CornerPath, and Image slices are never mutated in place), so
// saving value copies of Op, *opExtra, and Grid.Segs is sufficient for exact
// rollback.
func backupWindowInto(ops []Op, window OpWindow, scratch *restampScratch) {
	count := window.End - window.Start + 1
	opsStart := len(scratch.backupOps)

	scratch.backupOps = slices.Grow(scratch.backupOps, count)
	scratch.backupExtraIdx = slices.Grow(scratch.backupExtraIdx, count)

	for idx := window.Start; idx <= window.End; idx++ {
		op := ops[idx]
		scratch.backupOps = append(scratch.backupOps, op)

		if op.opExtra != nil && op.opExtra != emptyExtra {
			extraIdx := len(scratch.backupExtras)
			scratch.backupExtras = append(scratch.backupExtras, *op.opExtra)
			scratch.backupExtraIdx = append(scratch.backupExtraIdx, extraIdx)
		} else {
			scratch.backupExtraIdx = append(scratch.backupExtraIdx, -1)
		}

		if op.Grid != nil {
			segStart := len(scratch.backupGridSegs)
			scratch.backupGridSegs = append(scratch.backupGridSegs, op.Grid.Segs...)
			scratch.backupGrids = append(scratch.backupGrids, gridBackup{
				opIdx:    idx,
				segStart: segStart,
				segLen:   len(op.Grid.Segs),
			})
		}
	}

	scratch.backupRanges = append(scratch.backupRanges, opBackupRange{
		windowStart: window.Start,
		opsStart:    opsStart,
		opsLen:      count,
	})
}

// restoreBackups rolls back every backed-up window in reverse order.
func restoreBackups(ops []Op, scratch *restampScratch) {
	for idx := len(scratch.backupRanges) - 1; idx >= 0; idx-- {
		rng := scratch.backupRanges[idx]
		for offset := range rng.opsLen {
			k := rng.opsStart + offset
			opIdx := rng.windowStart + offset
			ops[opIdx] = scratch.backupOps[k]
			if extraIdx := scratch.backupExtraIdx[k]; extraIdx >= 0 && ops[opIdx].opExtra != nil {
				*ops[opIdx].opExtra = scratch.backupExtras[extraIdx]
			}
		}
	}

	for idx := len(scratch.backupGrids) - 1; idx >= 0; idx-- {
		gb := scratch.backupGrids[idx]
		if grid := ops[gb.opIdx].Grid; grid != nil {
			if cap(grid.Segs) >= gb.segLen {
				grid.Segs = grid.Segs[:gb.segLen]
			} else {
				grid.Segs = make([]GridSeg, gb.segLen)
			}

			copy(grid.Segs, scratch.backupGridSegs[gb.segStart:gb.segStart+gb.segLen])
		}
	}
}

// bounds is an axis-aligned box in layout points.
type bounds struct {
	x, y, w, h float64
	set        bool
}

// unionWindowBounds returns the painted area of every patched window.
func unionWindowBounds(ops []Op, windows []OpWindow) bounds {
	var out bounds

	for idx := range windows {
		window := windows[idx]

		for opIdx := window.Start; opIdx <= window.End; opIdx++ {
			op := &ops[opIdx]
			if op.Kind == OpKindNoop {
				continue
			}

			out.union(op.X, op.Y, op.W, op.H)
		}
	}

	return out
}

// union grows b to cover one rectangle.
func (b *bounds) union(left, top, width, height float64) {
	if width < 0 {
		left += width
		width = -width
	}

	if height < 0 {
		top += height
		height = -height
	}

	if !b.set {
		b.x, b.y, b.w, b.h, b.set = left, top, width, height, true

		return
	}

	right := math.Max(b.x+b.w, left+width)
	bottom := math.Max(b.y+b.h, top+height)
	b.x = math.Min(b.x, left)
	b.y = math.Min(b.y, top)
	b.w = right - b.x
	b.h = bottom - b.y
}
