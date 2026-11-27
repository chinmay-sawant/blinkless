package layout

// flowCBState holds the active in-flow containing-block height without heap
// boxing a float64 pointer on every definite-height element.
type flowCBState struct {
	height float64
	set    bool
}

// seedInitialContainingBlock installs a positive Options.Height as the
// in-flow containing-block height. The root element resolves percentage
// heights against that viewport. A non-positive height leaves the block
// indefinite, so a percentage stays auto.
func (e *engine) seedInitialContainingBlock() {
	if e.opts.Height <= 0 {
		return
	}

	e.initialCBHeight = e.opts.Height
	e.flowCB = flowCBState{height: e.initialCBHeight, set: true}
}

// containingBlockHeight returns the current in-flow containing-block height.
// Percent heights are intentionally unresolved when flowCB.set is false.
func (e *engine) containingBlockHeight() float64 {
	if !e.flowCB.set {
		return -1
	}

	return e.flowCB.height
}

// definiteContentHeight returns the content-box height established by a
// definite height on style. The caller's containing block is used for a
// percentage height; auto height remains indefinite.
func (e *engine) definiteContentHeight(style *ResolvedStyle) (float64, bool) {
	if style == nil {
		return 0, false
	}

	used, ok := resolveUsedHeight(style, e.containingBlockHeight(), e)
	if !ok {
		return 0, false
	}

	content := used - style.verticalChrome(e)
	if content < 0 {
		content = 0
	}

	return content, true
}

func (e *engine) setFlowCB(style *ResolvedStyle) (flowCBState, float64) {
	previous := e.flowCB
	parentCB := e.containingBlockHeight()
	if contentH, ok := e.definiteContentHeight(style); ok {
		e.flowCB = flowCBState{height: contentH, set: true}
	} else {
		e.flowCB = flowCBState{}
	}

	return previous, parentCB
}
