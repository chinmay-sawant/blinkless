package layout

import (
	"fmt"
	"strings"
	"testing"

	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// wiringDoc builds the few-hundred-row table fixture. Every declaration rides
// on inline style attributes and opts.Sheets stays nil, so the run never
// touches the cross-layout shared selector bundle (see style_match_cache.go):
// the golden below is deterministic no matter which suites ran before it.
// The table path exercises the pooled box list plus per-cell rows and chrome.
func wiringDoc() string {
	var doc strings.Builder
	doc.WriteString(`<html><body style="margin:0"><table style="width:480pt;border-collapse:collapse">`)
	for i := 0; i < 300; i++ {
		fmt.Fprintf(
			&doc,
			`<tr><td style="padding:2pt;border:1pt solid #999">row %d alpha beta gamma delta epsilon</td>`+
				`<td style="padding:2pt;border:1pt solid #999">cell %d price one two</td></tr>`,
			i, i,
		)
	}
	doc.WriteString(`</table></body></html>`)
	return doc.String()
}

func layoutWiringDoc(t *testing.T) *Result {
	t.Helper()

	res, err := Layout(mustParse(t, wiringDoc()), Options{
		Width: testViewport, Height: 800, Background: true,
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	return res
}

// TestWiringGoldenParity pins the pre-edit layout output for the 300-row
// fixture: the box-pool rent must not move a single operation.
//
//nolint:paralleltest // shares the process-wide box pools
func TestWiringGoldenParity(t *testing.T) {
	res := layoutWiringDoc(t)

	if got := len(res.Ops); got != 900 {
		t.Fatalf("ops = %d, want golden 900", got)
	}

	if got := res.BoxCount(); got != 604 {
		t.Fatalf("boxes = %d, want golden 604", got)
	}

	if !near(res.Height, 5850.0) {
		t.Fatalf("height = %.3f, want golden 5850.000", res.Height)
	}

	if !near(res.MaxContentX, 500.0) {
		t.Fatalf("maxX = %.3f, want golden 500.000", res.MaxContentX)
	}

	first := res.Ops[0]
	if first.Kind != OpText || !near(first.X, 3.0) || !near(first.Y, 13.91) ||
		!near(first.W, 202.793) || !near(first.H, 13.5) ||
		first.Text != "row 0 alpha beta gamma delta epsilon" {
		t.Fatalf("first op = %+v, want golden OpText row 0", first)
	}

	last := res.Ops[len(res.Ops)-1]
	if last.Kind != OpGridRun || !near(last.X, 0) || !near(last.Y, 5830.5) ||
		!near(last.W, 480.0) || !near(last.H, 19.5) {
		t.Fatalf("last op = %+v, want golden OpGridRun", last)
	}
}

// TestWiringPoolBounds builds the full fixture repeatedly and requires every
// retained pool count to stay within its documented cap.
//
//nolint:paralleltest // asserts process-wide pool counts
func TestWiringPoolBounds(t *testing.T) {
	for range 3 {
		layoutWiringDoc(t)
	}

	stats := boxAllocRetained()
	if stats.BoxPtrSlices > maxBoxAllocRetained ||
		stats.FloatSlices > maxBoxAllocRetained ||
		stats.Uint64Slices > maxBoxAllocRetained ||
		stats.BoxRows > maxBoxAllocRetained ||
		stats.OpIDs > maxBoxAllocRetained ||
		stats.Placed > maxBoxAllocRetained {
		t.Fatalf("slice pools unbounded: %+v", stats)
	}

	if stats.Boxes > maxBoxAllocBoxes {
		t.Fatalf("box structs unbounded: %+v", stats)
	}

	if stats.Clips > maxBoxAllocClips {
		t.Fatalf("clip rects unbounded: %+v", stats)
	}

	if stats.BoxClasses > maxBoxAllocBuckets {
		t.Fatalf("box slice classes unbounded: %+v", stats)
	}

	// Rent/return balance on a live check: a rent followed by its return
	// leaves the backing zeroed and reusable at the same capacity.
	const n = 48
	s := rentBoxPtrs(n)
	for i := range s {
		s[i] = &box{x: float64(i + 1)}
	}
	returnBoxPtrs(s)

	again := rentBoxPtrs(n)
	if len(again) != n || cap(again) != n {
		t.Fatalf("reused rent len %d cap %d, want %d", len(again), cap(again), n)
	}

	for i, b := range again {
		if b != nil {
			t.Fatalf("reused rent [%d] = %+v, want nil", i, b)
		}
	}
	returnBoxPtrs(again)
}

// wiringEngine mirrors the grid memo test helper: an engine over root with
// the default faces and the resolved cascade.
func wiringEngine(t *testing.T, root *html.Node, opts Options) *engine {
	t.Helper()

	styles, containers, err := resolveStylesForLayoutContext(t.Context(), root, opts)
	if err != nil {
		t.Fatalf("resolve styles: %v", err)
	}

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatalf("load faces: %v", err)
	}

	return newEngine(t.Context(), opts, faces, faces.Regular, styles, containers, nil)
}

func wiringFindElement(t *testing.T, root *html.Node, id string) *html.Node {
	t.Helper()

	var found *html.Node
	root.Walk(func(n *html.Node) {
		if found != nil || n.Type != html.ElementNode {
			return
		}

		if n.Attribute("id") == id {
			found = n
		}
	})

	if found == nil {
		t.Fatalf("missing element #%s", id)
	}

	return found
}

// TestWiringBlockMemoHit proves a repeated identical intrinsic measurement
// hits: same height, one entry, and the second call replays the same sticky
// and group counter deltas the first build consumed. The fixture uses inline
// styles and no sheets so the shared selector bundle cannot alias it.
//
//nolint:paralleltest // shares the process-wide block intrinsic memo
func TestWiringBlockMemoHit(t *testing.T) {
	const src = `<div id="b" style="width:400pt">` +
		`<div style="position:sticky;top:0">sticky header text</div>` +
		`<div style="mix-blend-mode:multiply"><p>alpha beta gamma delta epsilon zeta eta theta</p>` +
		`<p>second paragraph with enough text to wrap across lines</p></div></div>`

	// Resolve against the same root the engine was built for.
	root := mustParse(t, src)
	opts := Options{Width: testViewport, Height: 800, Background: true}
	eng := wiringEngine(t, root, opts)
	target := wiringFindElement(t, root, "b")

	stickyBefore, groupsBefore := eng.stickySeq, eng.nextGroupID
	first := eng.measureBlockIntrinsicHeight(target, 400)
	if first <= 0 {
		t.Fatalf("first height = %v, want positive", first)
	}

	stickyAfterFirst, groupsAfterFirst := eng.stickySeq, eng.nextGroupID
	if eng.blockIntrinsicMemo.Len() != 1 {
		t.Fatalf("memo holds %d entries after first measure, want 1", eng.blockIntrinsicMemo.Len())
	}

	second := eng.measureBlockIntrinsicHeight(target, 400)
	if second != first {
		t.Fatalf("second height = %v, want %v", second, first)
	}

	if eng.blockIntrinsicMemo.Len() != 1 {
		t.Fatalf("memo holds %d entries after hit, want 1", eng.blockIntrinsicMemo.Len())
	}

	firstSticky, firstGroups := stickyAfterFirst-stickyBefore, groupsAfterFirst-groupsBefore
	secondSticky, secondGroups := eng.stickySeq-stickyAfterFirst, eng.nextGroupID-groupsAfterFirst
	if secondSticky != firstSticky || secondGroups != firstGroups {
		t.Fatalf(
			"hit replayed sticky=%d groups=%d, want first-build deltas sticky=%d groups=%d",
			secondSticky, secondGroups, firstSticky, firstGroups,
		)
	}

	if firstSticky == 0 && firstGroups == 0 {
		t.Fatalf("first build consumed no counters, replay proof is vacuous")
	}

	// A fresh engine measuring uncached twice must agree on height and total
	// counter movement with the memoized run.
	plainRoot := mustParse(t, src)
	plain := wiringEngine(t, plainRoot, opts)
	plainTarget := wiringFindElement(t, plainRoot, "b")
	plainStickyBefore, plainGroupsBefore := plain.stickySeq, plain.nextGroupID
	plainFirst := plain.measureBlockIntrinsicHeightUncached(plainTarget, 400)
	plainSecond := plain.measureBlockIntrinsicHeightUncached(plainTarget, 400)
	if plainFirst != first || plainSecond != first {
		t.Fatalf("uncached heights %v/%v, want memoized %v", plainFirst, plainSecond, first)
	}

	plainSticky, plainGroups := plain.stickySeq-plainStickyBefore, plain.nextGroupID-plainGroupsBefore
	if plainSticky != 2*firstSticky || plainGroups != 2*firstGroups {
		t.Fatalf(
			"uncached consumed sticky=%d groups=%d, want twice the memoized deltas %d/%d",
			plainSticky, plainGroups, firstSticky, firstGroups,
		)
	}

	// Bypassed runs measure without storing.
	bypassOpts := opts
	bypassOpts.blockMemoBypass = true
	bypassRoot := mustParse(t, src)
	bypassed := wiringEngine(t, bypassRoot, bypassOpts)
	bypassedTarget := wiringFindElement(t, bypassRoot, "b")
	if height := bypassed.measureBlockIntrinsicHeight(bypassedTarget, 400); height != first {
		t.Fatalf("bypassed height = %v, want %v", height, first)
	}

	if bypassed.blockIntrinsicMemo.Len() != 0 {
		t.Fatalf("bypassed memo holds %d entries, want 0", bypassed.blockIntrinsicMemo.Len())
	}
}
