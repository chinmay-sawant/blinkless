package layout

import "testing"

// This file pins the hover-subtree paint fast paths from a second angle:
// nested overflow clips (overflow_clip.go), deferred chrome merge
// (layout_chrome.go), lean placed walks (placed.go), the mask-free flat stamp
// path (transform.go), and the projected-op fast path in FinalizeDisplayOps
// (display_geometry.go). Companion to clip_chrome_meta_test.go with a distinct
// generated doc and per-item fast-path plus negative asserts.

func chromeMetaDocHTML() string {
	rows := ""

	for r := 0; r < 5; r++ {
		rows += "<tr>"

		for c := 0; c < 4; c++ {
			rows += "<td>m</td>"
		}

		rows += "</tr>"
	}

	return `<html><body>
<div id="cOuter">
  <div id="cMiddle">
    <div id="cTilt">
      <table id="cGrid">` + rows + `</table>
    </div>
  </div>
</div>
<div id="cStage"><div id="cCard">hi</div></div>
<div id="cStick">s</div>
</body></html>`
}

func chromeMetaCSS() string {
	return `
html, body { margin: 0; padding: 0; }
#cOuter { width: 320px; height: 170px; overflow: hidden; background: #c33; border: 4px solid #111; }
#cMiddle { width: 270px; height: 130px; overflow: hidden; background: #3c3; margin: 8px; }
#cTilt { width: 210px; height: 70px; transform: rotate(12deg); background: #33c; }
#cGrid { border-collapse: collapse; }
#cGrid td { border: 1px solid #000; width: 40px; height: 18px; }
#cStage { perspective: 600px; width: 200px; height: 100px; }
#cCard { width: 120px; height: 60px; transform: rotateY(25deg); background: #cc3; }
#cStick { position: sticky; top: 0; width: 100px; height: 20px; background: #c3c; }
`
}

func chromeMetaResult(t *testing.T) *Result {
	t.Helper()

	return layoutHTML(t, chromeMetaDocHTML(), sheet(t, chromeMetaCSS()))
}

// TestChromeMetaGoldenParity checks the generated doc paints the same drawing
// list twice: op counts, key rects, and paint geometry must be stable.
func TestChromeMetaGoldenParity(t *testing.T) {
	t.Parallel()

	first := chromeMetaResult(t)
	second := chromeMetaResult(t)

	if len(first.Ops) != len(second.Ops) {
		t.Fatalf("op count parity: %d vs %d", len(first.Ops), len(second.Ops))
	}

	if len(first.Ops) == 0 {
		t.Fatal("no ops emitted")
	}

	if got, want := opsDigest(first.Ops), opsDigest(second.Ops); got != want {
		t.Fatalf("ops digest parity: %s vs %s", got, want)
	}

	full := PlacedElements(first)
	again := PlacedElements(second)

	if len(full) != len(again) {
		t.Fatalf("placed parity: %d vs %d", len(full), len(again))
	}

	if len(full) == 0 {
		t.Fatal("no placed elements")
	}

	// Key rects: each chrome fill keeps its distinct height.
	// 170px content (127.5pt) + 4px borders (3pt each side) = 133.5pt border box,
	// 130px = 97.5pt, 70px = 52.5pt, 60px = 45pt, 20px = 15pt.
	outer := fillByHeight(t, first.Ops, 133.5)
	if !near(outer.W, 246) {
		t.Errorf("outer fill w = %.2f, want 246 (320px content box plus borders)", outer.W)
	}

	tilt := fillByHeight(t, first.Ops, 52.5)
	if !tilt.XformSet {
		t.Error("tilt bg XformSet = false, want the flat-path bake")
	}

	if tilt.Transform().IsIdentity() {
		t.Error("tilt bg transform is identity, want the rotate bake")
	}

	card := fillByHeight(t, first.Ops, 45)
	if card.Kind != OpFillRect {
		t.Errorf("card op kind = %d, want fill", card.Kind)
	}

	stick := fillByHeight(t, first.Ops, 15)
	if stick.StickyID == 0 {
		t.Error("sticky bg StickyID = 0, want nonzero")
	}
}

// TestChromeMetaClipSinglePass checks one active clip rect threads down the
// walk: nested fills cut to the innermost clip, the clip owner's own chrome
// stays whole, and a visible-overflow subtree plus an unclipped sibling stay
// untouched.
func TestChromeMetaClipSinglePass(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body>
<div id="hOuter"><div id="hInner">x</div></div>
<div id="vOuter"><div id="vInner">y</div></div>
<div id="hFree">z</div>
</body></html>`, sheet(t, `
html, body { margin: 0; padding: 0; }
div { margin: 0; padding: 0; }
#hOuter { width: 200px; height: 100px; overflow: hidden; background: #c33; }
#hInner { width: 300px; height: 40px; background: #33c; }
#vOuter { width: 200px; height: 100px; overflow: visible; background: #3c3; }
#vInner { width: 300px; height: 60px; background: #cc3; }
#hFree { width: 50px; height: 50px; background: #c3c; }
`))

	// 100px = 75pt, 40px = 30pt, 60px = 45pt, 50px = 37.5pt.
	outer := fillByHeight(t, res.Ops, 75)
	if !near(outer.W, 150) {
		t.Errorf("hidden outer chrome w = %.2f, want 150 (own clip must not eat own chrome)", outer.W)
	}

	inner := fillByHeight(t, res.Ops, 30)
	if !near(inner.W, 150) {
		t.Errorf("hidden inner fill w = %.2f, want 150 (cut to the outer box)", inner.W)
	}

	visible := fillByHeight(t, res.Ops, 45)
	if !near(visible.W, 225) {
		t.Errorf("visible inner fill w = %.2f, want 225 (negative: visible overflow never clips)", visible.W)
	}

	free := fillByHeight(t, res.Ops, 37.5)
	if !near(free.W, 37.5) {
		t.Errorf("free fill w = %.2f, want 37.5 (no ancestor clip, untouched)", free.W)
	}
}

// TestChromeMetaSortKeysFusedWalk checks the fused merge keeps paint order
// and re-stamps marks: ancestor chrome under descendant content, sticky IDs
// on deferred chrome, fixed marks kept, static boxes unmarked.
func TestChromeMetaSortKeysFusedWalk(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body>
<div id="sOuter"><div id="sInner">x</div></div>
<div id="sStick"><div id="sKid">s</div></div>
<div id="sFix">f</div>
</body></html>`, sheet(t, `
html, body { margin: 0; padding: 0; }
div { margin: 0; padding: 0; }
#sOuter { width: 200px; height: 100px; background: #c33; }
#sInner { width: 300px; height: 40px; background: #33c; }
#sStick { position: sticky; top: 0; width: 100px; height: 40px; background: #c3c; }
#sKid { width: 80px; height: 20px; background: #3cc; }
#sFix { position: fixed; top: 0; width: 60px; height: 24px; background: #cc3; }
`))

	// 100px = 75pt, 40px = 30pt, 24px = 18pt, 20px = 15pt.
	outer := fillByHeight(t, res.Ops, 75)
	inner := fillByHeight(t, res.Ops, 30)

	outerIdx, innerIdx := -1, -1

	for i := range res.Ops {
		if res.Ops[i] == outer {
			outerIdx = i
		}

		if res.Ops[i] == inner {
			innerIdx = i
		}
	}

	if outerIdx < 0 || innerIdx < 0 || outerIdx > innerIdx {
		t.Errorf("chrome order: outer bg at %d, inner bg at %d, want outer before inner", outerIdx, innerIdx)
	}

	if outer.StickyID != 0 {
		t.Errorf("static outer StickyID = %d, want 0 (negative)", outer.StickyID)
	}

	if outer.Fixed {
		t.Error("static outer Fixed = true, want false (negative)")
	}

	stick := fillByHeight(t, res.Ops, 30)
	_ = stick

	kid := fillByHeight(t, res.Ops, 15)
	if kid.StickyID == 0 {
		t.Error("sticky child bg StickyID = 0, want nonzero (fused walk re-stamps)")
	}

	fixed := fillByHeight(t, res.Ops, 18)
	if !fixed.Fixed {
		t.Error("fixed box bg Fixed = false, want true")
	}

	if fixed.StickyID != 0 {
		t.Errorf("fixed box StickyID = %d, want 0 (negative)", fixed.StickyID)
	}
}

// checkPlacedParity compares one lean-walk element against its full-walk twin.
// The lean walk skips text and bakes Own/World once, so geometry, operation
// ownership, transform metadata, and the empty text field must still match.
func checkPlacedParity(t *testing.T, index int, lean, full PlacedElement) {
	t.Helper()

	if lean.ID != full.ID || lean.Tag != full.Tag {
		t.Fatalf("elem %d = %q %q, want %q %q", index, lean.ID, lean.Tag, full.ID, full.Tag)
	}

	if lean.X != full.X || lean.Y != full.Y || lean.W != full.W || lean.H != full.H {
		t.Fatalf("elem %q box differs between lean and full walks", full.ID)
	}

	if len(lean.OperationIDs) != len(full.OperationIDs) {
		t.Fatalf("elem %q op IDs differ between lean and full walks", full.ID)
	}

	for j := range lean.OperationIDs {
		if lean.OperationIDs[j] != full.OperationIDs[j] {
			t.Fatalf("elem %q op ID %d differs between lean and full walks", full.ID, j)
		}
	}

	if lean.Transform != full.Transform {
		t.Fatalf("elem %q transform differs between lean and full walks", full.ID)
	}

	if lean.Text != "" {
		t.Fatalf("elem %q lean Text = %q, want empty", full.ID, lean.Text)
	}
}

// TestChromeMetaPlacedLazy checks the lean walk skips text, bakes Own/World
// once, and reports the same operation ownership as the full walk.
func TestChromeMetaPlacedLazy(t *testing.T) {
	t.Parallel()

	res := chromeMetaResult(t)
	full := PlacedElements(res)

	var lean []PlacedElement

	WalkPlacedWithOptions(res, WalkPlacedOptions{}, func(elem *PlacedElement) {
		lean = append(lean, *elem)
	})

	if len(lean) != len(full) {
		t.Fatalf("lean walk count = %d, full walk count = %d", len(lean), len(full))
	}

	sawText := false

	for i := range full {
		checkPlacedParity(t, i, lean[i], full[i])

		if full[i].Text != "" {
			sawText = true
		}
	}

	if !sawText {
		t.Fatal("full walk reported no text; fixture must contain some")
	}

	byID := map[string]PlacedElement{}

	for _, elem := range full {
		if elem.ID != "" {
			byID[elem.ID] = elem
		}
	}

	outer := byID["cOuter"]
	if outer.Transform.HasMatrix {
		t.Error("cOuter HasMatrix = true, want false")
	}

	if outer.Transform.Own != IdentityMatrix() {
		t.Errorf("cOuter Own = %+v, want identity (fast path skips the bake)", outer.Transform.Own)
	}

	tilt := byID["cTilt"]
	if !tilt.Transform.HasMatrix {
		t.Fatal("cTilt HasMatrix = false, want true")
	}

	if want := tilt.Transform.Parent.Mul(tilt.Transform.Own); want != tilt.Transform.World {
		t.Error("cTilt World != Parent*Own, want the single-bake product")
	}

	// Negative: nil results invoke nothing, and out-of-range ID ranges are empty.
	called := false

	WalkPlaced(nil, func(_ *PlacedElement) { called = true })
	WalkPlacedWithOptions(nil, WalkPlacedOptions{}, func(_ *PlacedElement) { called = true })

	if called {
		t.Error("walk invoked fn on nil result, want no calls")
	}

	if got := appendOperationIDs(res.Ops, -1, -2, &[]uint64{}); len(got) != 0 {
		t.Errorf("appendOperationIDs(-1,-2) len = %d, want 0", len(got))
	}

	if got := appendOperationIDs(res.Ops, len(res.Ops)+10, len(res.Ops)+20, &[]uint64{}); len(got) != 0 {
		t.Errorf("appendOperationIDs past end len = %d, want 0", len(got))
	}
}

// TestChromeMetaFlatStampNoMask checks the 2D-only tree stamps without the
// coverage mask, the identity fast path writes nothing, and a 3D tree still
// needs the mask.
func TestChromeMetaFlatStampNoMask(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body>
<div id="fOuter"><div id="fTilt">t</div></div>
</body></html>`, sheet(t, `
html, body { margin: 0; padding: 0; }
div { margin: 0; padding: 0; }
#fOuter { width: 200px; height: 100px; background: #c33; }
#fTilt { width: 100px; height: 30px; transform: rotate(10deg); background: #33c; }
`))

	if subtreeNeedsCoveredMask(res.boxTreeRoot()) {
		t.Error("flat doc needs the coverage mask, want the mask-free path")
	}

	tilt := fillByHeight(t, res.Ops, 22.5)
	if !tilt.XformSet {
		t.Fatal("tilt bg XformSet = false, want the flat-path bake")
	}

	outer := fillByHeight(t, res.Ops, 75)
	if outer.XformSet {
		t.Error("outer bg XformSet = true, want false (negative: untransformed untouched)")
	}

	// Identity fast path writes nothing.
	plain := make([]Op, 2)
	stampFlatExclusiveTransform(IdentityMatrix(), plain, nil, 0, 1)

	for i := range plain {
		if plain[i].XformSet {
			t.Errorf("identity stamp set XformSet on op %d, want untouched", i)
		}
	}

	// Negative: the projected doc carries rotateY and needs the mask.
	if projected := chromeMetaResult(t); !subtreeNeedsCoveredMask(projected.boxTreeRoot()) {
		t.Error("projected doc skips the coverage mask, want the mask path")
	}

	if subtreeNeedsCoveredMask(nil) {
		t.Error("subtreeNeedsCoveredMask(nil) = true, want false")
	}
}

// TestChromeMetaProjectEarlyReturn checks FinalizeDisplayOps returns early
// with no projected ops and finalizes the card's fill and text with presized
// bounds when projection exists.
func TestChromeMetaProjectEarlyReturn(t *testing.T) {
	t.Parallel()

	flat := layoutHTML(t, `<html><body><div id="pFlat">x</div></body></html>`, sheet(t, `
html, body { margin: 0; padding: 0; }
#pFlat { width: 100px; height: 30px; background: #c33; }
`))

	if got := countProjectedOps(flat.Ops); got != 0 {
		t.Fatalf("flat doc projected count = %d, want 0", got)
	}

	before := opsDigest(flat.Ops)
	if err := FinalizeDisplayOps(flat.Ops); err != nil {
		t.Fatalf("FinalizeDisplayOps(flat): %v", err)
	}

	if got := opsDigest(flat.Ops); got != before {
		t.Error("FinalizeDisplayOps changed a doc with nothing projected (must be a no-op)")
	}

	res := chromeMetaResult(t)

	if got := countProjectedOps(res.Ops); got != 2 {
		t.Fatalf("projected doc count = %d, want 2", got)
	}

	if err := FinalizeDisplayOps(res.Ops); err != nil {
		t.Fatalf("FinalizeDisplayOps: %v", err)
	}

	if got := countProjectedOps(res.Ops); got != 0 {
		t.Errorf("post-finalize projected count = %d, want 0", got)
	}

	fills, texts := 0, 0

	for i := range res.Ops {
		extra := res.Ops[i].opExtra
		if extra == nil {
			continue
		}

		if extra.HasProjective {
			t.Errorf("op %d still projected after finalize", i)
		}

		if res.Ops[i].Kind == OpFillRect && extra.HasPath {
			fills++
		}

		if res.Ops[i].Kind == OpText && extra.HasGlyphBasis {
			texts++
		}
	}

	if fills != 1 || texts != 1 {
		t.Errorf("finalized card = %d fills with paths, %d texts with basis; want 1 and 1", fills, texts)
	}

	if err := FinalizeDisplayOps(nil); err != nil {
		t.Errorf("FinalizeDisplayOps(nil) = %v, want nil", err)
	}

	if err := FinalizeDisplayOps([]Op{}); err != nil {
		t.Errorf("FinalizeDisplayOps(empty) = %v, want nil", err)
	}

	if got := countProjectedOps(nil); got != 0 {
		t.Errorf("countProjectedOps(nil) = %d, want 0 (negative)", got)
	}
}

// TestChromeMetaCoverIntervalsFastPaths pins the child-cover sweep: sorted
// input stays linear, inversions sort, overlaps merge, and bad ranges drop.
func TestChromeMetaCoverIntervalsFastPaths(t *testing.T) {
	t.Parallel()

	if got := childCoverIntervals(nil, 0, 10); got != nil {
		t.Errorf("childCoverIntervals(nil) = %v, want nil", got)
	}

	leaf := &box{opStart: 0, opEnd: -1}
	if got := childCoverIntervals(leaf, 0, 10); got != nil {
		t.Errorf("childCoverIntervals(no children) = %v, want nil", got)
	}

	sorted := &box{
		opStart: 0,
		opEnd:   10,
		children: []*box{
			{opStart: 0, opEnd: 2},
			{opStart: 4, opEnd: 6},
		},
	}

	if got := childCoverIntervals(sorted, 0, 10); len(got) != 2 {
		t.Fatalf("sorted cover intervals = %v, want 2 ranges", got)
	}

	unsorted := &box{
		opStart: 0,
		opEnd:   10,
		children: []*box{
			{opStart: 6, opEnd: 8},
			{opStart: 0, opEnd: 2},
		},
	}

	got := childCoverIntervals(unsorted, 0, 10)
	if len(got) != 2 || got[0].start != 0 || got[1].start != 6 {
		t.Fatalf("unsorted cover intervals = %v, want [{0 2} {6 8}]", got)
	}

	overlap := &box{
		opStart: 0,
		opEnd:   10,
		children: []*box{
			{opStart: 0, opEnd: 5},
			{opStart: 3, opEnd: 7},
		},
	}

	merged := childCoverIntervals(overlap, 0, 10)
	if len(merged) != 1 || merged[0].start != 0 || merged[0].end != 7 {
		t.Fatalf("overlapping cover intervals = %v, want [{0 7}]", merged)
	}

	bad := &box{
		opStart: 0,
		opEnd:   10,
		children: []*box{
			{opStart: 5, opEnd: 3},
		},
	}

	if got := childCoverIntervals(bad, 0, 10); got != nil {
		t.Errorf("inverted child range cover = %v, want nil (negative)", got)
	}
}
