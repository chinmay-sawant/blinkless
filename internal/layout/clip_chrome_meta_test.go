package layout

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"
	"testing"
)

// This file pins the hover-subtree paint fast paths: nested overflow clips
// (overflow_clip.go), deferred chrome merge (layout_chrome.go), lean placed
// walks (placed.go), the mask-free flat stamp path (transform.go), and the
// projected-op fast path in FinalizeDisplayOps (display_geometry.go).
//
// The golden doc exercises all five at once: nested overflow clips around a
// transformed table, a 3D-projected card, and a sticky box. The constants in
// TestClipChromeMetaGoldenParity were captured pre-edit; every fast path
// must reproduce them exactly.

// metaGoldenDocHTML builds the parity doc. It is generated in Go so the grid
// is big enough to exercise batch paths (grid runs, chrome merge, clip
// walks) instead of a hand-written toy.
func metaGoldenDocHTML() string {
	rows := ""

	for r := 0; r < 6; r++ {
		rows += "<tr>"

		for c := 0; c < 5; c++ {
			rows += "<td>r</td>"
		}

		rows += "</tr>"
	}

	return `<html><body>
<div id="outer">
  <div id="middle">
    <div id="tilt">
      <table id="grid">` + rows + `</table>
    </div>
  </div>
</div>
<div id="stage"><div id="card">hi</div></div>
<div id="stick">s</div>
</body></html>`
}

func metaGoldenSheet(t *testing.T) string {
	t.Helper()

	return `
html, body { margin: 0; padding: 0; }
#outer { width: 300px; height: 160px; overflow: hidden; background: #c33; border: 4px solid #111; }
#middle { width: 260px; height: 120px; overflow: hidden; background: #3c3; margin: 8px; }
#tilt { width: 200px; height: 60px; transform: rotate(10deg); background: #33c; }
#grid { border-collapse: collapse; }
#grid td { border: 1px solid #000; width: 40px; height: 18px; }
#stage { perspective: 600px; width: 200px; height: 100px; }
#card { width: 120px; height: 60px; transform: rotateY(25deg); background: #cc3; }
#stick { position: sticky; top: 0; width: 100px; height: 20px; background: #c3c; }
`
}

// opsDigest folds op count, kinds, and geometry into one hex string so the
// parity test catches any drift the itemized asserts miss.
func opsDigest(ops []Op) string {
	sum := fnv.New64a()

	var buf [8]byte

	put := func(v uint64) {
		binary.LittleEndian.PutUint64(buf[:], v)
		sum.Write(buf[:])
	}

	put(uint64(len(ops)))

	for i := range ops {
		put(uint64(ops[i].Kind))
		put(math.Float64bits(ops[i].X))
		put(math.Float64bits(ops[i].Y))
		put(math.Float64bits(ops[i].W))
		put(math.Float64bits(ops[i].H))
	}

	return fmt.Sprintf("%x", sum.Sum64())
}

func metaGoldenResult(t *testing.T) *Result {
	t.Helper()

	return layoutHTML(t, metaGoldenDocHTML(), sheet(t, metaGoldenSheet(t)))
}

// TestClipChromeMetaGoldenParity pins the drawing list of the golden doc.
// The constants below were captured pre-edit; every fast path must
// reproduce them exactly.
func TestClipChromeMetaGoldenParity(t *testing.T) {
	t.Parallel()

	const (
		wantOps    = 47
		wantDigest = "76dbf33a4582df95"
		wantPlaced = 40
	)

	res := metaGoldenResult(t)
	placed := PlacedElements(res)

	if len(res.Ops) != wantOps {
		t.Fatalf("op count = %d, want %d", len(res.Ops), wantOps)
	}

	if got := opsDigest(res.Ops); got != wantDigest {
		t.Fatalf("ops digest = %s, want %s", got, wantDigest)
	}

	if len(placed) != wantPlaced {
		t.Fatalf("placed count = %d, want %d", len(placed), wantPlaced)
	}

	first, last := res.Ops[0], res.Ops[len(res.Ops)-1]

	if first.Kind != OpFillRect || !near(first.X, 0) || !near(first.Y, 0) ||
		!near(first.W, 231) || !near(first.H, 126) {
		t.Errorf("first op = kind %d %.2f,%.2f %.2fx%.2f, want fill 0,0 231x126",
			first.Kind, first.X, first.Y, first.W, first.H)
	}

	if last.Kind != OpText || !near(last.X, 0) || !near(last.Y, 211.910156) ||
		!near(last.W, 6) || !near(last.H, 13.5) {
		t.Errorf("last op = kind %d %.2f,%.2f %.2fx%.2f, want text 0,211.91 6x13.5",
			last.Kind, last.X, last.Y, last.W, last.H)
	}
}

// clipNestDoc is a dedicated nested-clip fixture: a 300pt box inside a
// 200pt hidden box, plus an unclipped sibling.
func clipNestDoc() (string, string) {
	return `<html><body>
<div id="outer"><div id="inner">x</div></div>
<div id="free">y</div>
</body></html>`, `
html, body { margin: 0; padding: 0; }
div { margin: 0; padding: 0; }
#outer { width: 200px; height: 100px; overflow: hidden; background: #c33; }
#inner { width: 300px; height: 40px; background: #33c; }
#free { width: 50px; height: 50px; background: #3c3; }
`
}

// fillByHeight returns the first fill rect of the given height.
func fillByHeight(t *testing.T, ops []Op, h float64) Op {
	t.Helper()

	for _, op := range ops {
		if op.Kind == OpFillRect && near(op.H, h) {
			return op
		}
	}

	t.Fatalf("no fill rect with height %.2f", h)

	return Op{} //nolint:exhaustruct // unreachable
}

// TestClipNestedPaintGeometry checks the single-pass clip paints what the
// per-depth rescan painted: descendant fills cut to the innermost clip, the
// clip owner's own chrome intact, and unclipped siblings untouched.
func TestClipNestedPaintGeometry(t *testing.T) {
	t.Parallel()

	doc, css := clipNestDoc()
	res := layoutHTML(t, doc, sheet(t, css))

	// 200px = 150pt, 100px = 75pt, 300px = 225pt, 40px = 30pt, 50px = 37.5pt.
	outer := fillByHeight(t, res.Ops, 75)
	if !near(outer.X, 0) || !near(outer.Y, 0) || !near(outer.W, 150) {
		t.Errorf("outer chrome = %.2f,%.2f %.2fx%.2f, want 0,0 150x75 (own clip must not eat own chrome)",
			outer.X, outer.Y, outer.W, outer.H)
	}

	inner := fillByHeight(t, res.Ops, 30)
	if !near(inner.X, 0) || !near(inner.W, 150) {
		t.Errorf("inner fill = %.2f,%.2f %.2fx%.2f, want x=0 w=150 (cut to the outer box)",
			inner.X, inner.Y, inner.W, inner.H)
	}

	free := fillByHeight(t, res.Ops, 37.5)
	if !near(free.W, 37.5) {
		t.Errorf("free fill w = %.2f, want 37.5 (no ancestor clip, untouched)", free.W)
	}
}

// chromeOrderDoc carries a fixed box so the immediate-splice path is pinned
// alongside the deferred merge.
func chromeOrderDoc() (string, string) {
	doc, css := clipNestDoc()

	return doc[:len(doc)-len("</body></html>")] + "<div id=\"fix\">f</div></body></html>", css + `
#fix { position: fixed; top: 0; width: 60px; height: 20px; background: #cc3; }
`
}

// TestChromeMergeOrderAndImmediateSplice checks the fused merge keeps paint
// order (ancestor chrome under descendant content) and keeps the
// immediate-splice marks for sticky and fixed boxes.
func TestChromeMergeOrderAndImmediateSplice(t *testing.T) {
	t.Parallel()

	doc, css := chromeOrderDoc()
	res := layoutHTML(t, doc, sheet(t, css))

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
		t.Errorf("chrome order: outer bg at %d, inner bg at %d, want outer before inner",
			outerIdx, innerIdx)
	}

	if outer.StickyID != 0 {
		t.Errorf("static outer bg StickyID = %d, want 0 (negative: no sticky stamp)", outer.StickyID)
	}

	if outer.Fixed {
		t.Error("static outer bg Fixed = true, want false (negative: not viewport-fixed)")
	}

	fixed := fillByHeight(t, res.Ops, 15)

	if !fixed.Fixed {
		t.Error("fixed box bg Fixed = false, want true (immediate splice keeps the mark)")
	}

	if fixed.StickyID != 0 {
		t.Errorf("fixed box bg StickyID = %d, want 0", fixed.StickyID)
	}
}

// TestChromeStickyRestamp checks deferred chrome under a sticky ancestor
// still gets the StickyID through the fused walk.
func TestChromeStickyRestamp(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div id="stick"><div id="kid">s</div></div></body></html>`, sheet(t, `
html, body { margin: 0; padding: 0; }
div { margin: 0; padding: 0; }
#stick { position: sticky; top: 0; width: 100px; height: 40px; background: #c3c; }
#kid { width: 80px; height: 20px; background: #3cc; }
`))

	// 40px = 30pt, 20px = 15pt.
	stick := fillByHeight(t, res.Ops, 30)
	if stick.StickyID == 0 {
		t.Error("sticky box bg StickyID = 0, want nonzero")
	}

	kid := fillByHeight(t, res.Ops, 15)
	if kid.StickyID == 0 {
		t.Error("sticky child bg StickyID = 0, want nonzero (fused walk re-stamps the range)")
	}
}

// TestPlacedLeanWalkParity checks the lean walk (no text) reports the same
// boxes, transforms, and operation ownership as the full walk.
func TestPlacedLeanWalkParity(t *testing.T) {
	t.Parallel()

	res := metaGoldenResult(t)
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
		if lean[i].ID != full[i].ID || lean[i].Tag != full[i].Tag {
			t.Fatalf("elem %d = %q %q, want %q %q", i, lean[i].ID, lean[i].Tag, full[i].ID, full[i].Tag)
		}

		if lean[i].X != full[i].X || lean[i].Y != full[i].Y ||
			lean[i].W != full[i].W || lean[i].H != full[i].H {
			t.Fatalf("elem %q box differs between lean and full walks", full[i].ID)
		}

		if len(lean[i].OperationIDs) != len(full[i].OperationIDs) {
			t.Fatalf("elem %q op IDs differ between lean and full walks", full[i].ID)
		}

		for j := range lean[i].OperationIDs {
			if lean[i].OperationIDs[j] != full[i].OperationIDs[j] {
				t.Fatalf("elem %q op ID %d differs between lean and full walks", full[i].ID, j)
			}
		}

		if lean[i].Transform != full[i].Transform {
			t.Fatalf("elem %q transform differs between lean and full walks", full[i].ID)
		}

		if lean[i].Text != "" {
			t.Fatalf("elem %q lean Text = %q, want empty (text not requested)", full[i].ID, lean[i].Text)
		}

		if full[i].Text != "" {
			sawText = true
		}
	}

	if !sawText {
		t.Fatal("full walk reported no text at all; fixture must contain some")
	}

	byID := map[string]PlacedElement{}

	for _, elem := range full {
		if elem.ID != "" {
			byID[elem.ID] = elem
		}
	}

	outer := byID["outer"]
	if outer.Transform.HasMatrix {
		t.Error("outer HasMatrix = true, want false (no static transform)")
	}

	if outer.Transform.Own != IdentityMatrix() {
		t.Errorf("outer Own = %+v, want identity (fast path skips the bake)", outer.Transform.Own)
	}

	if outer.Transform.World != outer.Transform.Parent {
		t.Error("outer World != Parent, want the parent matrix passed through")
	}

	tilt := byID["tilt"]
	if !tilt.Transform.HasMatrix {
		t.Fatal("tilt HasMatrix = false, want true")
	}

	if tilt.Transform.Own.IsIdentity() {
		t.Error("tilt Own is identity, want the rotate bake")
	}

	if want := tilt.Transform.Parent.Mul(tilt.Transform.Own); want != tilt.Transform.World {
		t.Error("tilt World != Parent*Own, want the single-bake product")
	}
}

// TestPlacedNilResult checks the walks invoke nothing on a nil result.
func TestPlacedNilResult(t *testing.T) {
	t.Parallel()

	called := false

	WalkPlaced(nil, func(_ *PlacedElement) { called = true })
	WalkPlacedWithOptions(nil, WalkPlacedOptions{}, func(_ *PlacedElement) { called = true })
	WalkPlacedWithOptions(&Result{}, WalkPlacedOptions{IncludeText: true}, func(_ *PlacedElement) { called = true })

	if called {
		t.Error("walk invoked fn on nil result, want no calls")
	}
}

// flatStampDoc has a 2D transform and no 3D, so the stamp walk takes the
// mask-free flat path.
func flatStampDoc() (string, string) {
	doc, css := clipNestDoc()

	return doc[:len(doc)-len("</body></html>")] +
			"<div id=\"tilt\">t</div></body></html>", css + `
#tilt { width: 100px; height: 30px; transform: rotate(10deg); background: #33c; }
`
}

// TestFlatStampPath checks the 2D-only tree stamps without the coverage mask
// and paints identical matrices: the transformed box carries its bake,
// untransformed boxes carry nothing.
func TestFlatStampPath(t *testing.T) {
	t.Parallel()

	doc, css := flatStampDoc()
	res := layoutHTML(t, doc, sheet(t, css))

	if subtreeNeedsCoveredMask(res.boxTreeRoot()) {
		t.Error("flat doc needs the coverage mask, want the mask-free path")
	}

	// 30px tilt bg = 22.5pt high; outer bg = 75pt.
	tilt := fillByHeight(t, res.Ops, 22.5)
	if !tilt.XformSet {
		t.Fatal("tilt bg XformSet = false, want the flat-path bake")
	}

	if tilt.Transform().IsIdentity() {
		t.Error("tilt bg transform is identity, want the rotate bake")
	}

	outer := fillByHeight(t, res.Ops, 75)
	if outer.XformSet {
		t.Error("outer bg XformSet = true, want false (negative: untransformed boxes untouched)")
	}

	if resWith3D := metaGoldenResult(t); !subtreeNeedsCoveredMask(resWith3D.boxTreeRoot()) {
		t.Error("golden doc skips the coverage mask, want the mask path (it carries rotateY)")
	}
}

// TestProjectFinalizePaths checks FinalizeDisplayOps finalizes projected ops
// and is a no-op (early return) when nothing is projected.
func TestProjectFinalizePaths(t *testing.T) {
	t.Parallel()

	flatDoc, flatCSS := flatStampDoc()
	flat := layoutHTML(t, flatDoc, sheet(t, flatCSS))

	if got := countProjectedOps(flat.Ops); got != 0 {
		t.Fatalf("flat doc projected count = %d, want 0", got)
	}

	before := opsDigest(flat.Ops)
	if err := FinalizeDisplayOps(flat.Ops); err != nil {
		t.Fatalf("FinalizeDisplayOps(flat): %v", err)
	}

	if got := opsDigest(flat.Ops); got != before {
		t.Error("FinalizeDisplayOps changed a doc with nothing projected (negative: must be a no-op)")
	}

	res := metaGoldenResult(t)

	if got := countProjectedOps(res.Ops); got != 2 {
		t.Fatalf("golden doc projected count = %d, want 2", got)
	}

	if err := FinalizeDisplayOps(res.Ops); err != nil {
		t.Fatalf("FinalizeDisplayOps: %v", err)
	}

	if got := countProjectedOps(res.Ops); got != 0 {
		t.Errorf("post-finalize projected count = %d, want 0 (flags cleared)", got)
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
}
