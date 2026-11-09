package layout

import (
	"errors"
	"reflect"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// relayoutSizeSliderCSS is the slider shape: a fixed-size track with a fill
// whose width a percentage class drives, and a card below the track. The fill
// going 20% to 80% resizes the patched element while the track keeps its
// size, so the splice must widen to the track instead of refusing.
const relayoutSizeSliderCSS = `
	#track { width: 400px; height: 40px; background-color: #dddddd; }
	#fill { height: 40px; background-color: #3366cc; }
	#fill.narrow { width: 20%; }
	#fill.wide { width: 80%; }
	.card { width: 200px; height: 60px; background-color: #eeeeee; }
`

// relayoutSizeSliderHTML holds the fill inside its track with a card below.
const relayoutSizeSliderHTML = `<html><body>` +
	`<div id="track"><div id="fill" class="narrow">x</div></div>` +
	`<div id="after" class="card">Below</div>` +
	`</body></html>`

// relayoutSizeChartCSS is the chart shape: a fixed-size chart with a bar
// whose height a percentage class drives. The bar going 20% to 70% resizes
// the patched element while the chart keeps its size.
const relayoutSizeChartCSS = `
	#chart { width: 300px; height: 200px; background-color: #dddddd; }
	#bar { width: 40px; background-color: #3366cc; }
	#bar.short { height: 20%; }
	#bar.tall { height: 70%; }
`

// relayoutSizeChartHTML holds the bar inside its chart.
const relayoutSizeChartHTML = `<html><body>` +
	`<div id="chart"><div id="bar" class="short">x</div></div>` +
	`</body></html>`

// relayoutSizeGrowHTML holds one content-sized block: growing its text grows
// the block, the body, and everything up to the root, so no ancestor keeps
// its size and the splice still refuses.
const relayoutSizeGrowHTML = `<html><body><div id="box">short</div></body></html>`

// relayoutSizeGrowCSS gives the block a width that makes long text wrap.
const relayoutSizeGrowCSS = `#box { width: 120px; font-size: 16px; }`

// assertSizeSpliceMatchesFull proves the patched page is the one a full
// relayout of the same document produces: the operations agree and every
// named box sits where the full layout puts it.
func assertSizeSpliceMatchesFull(t *testing.T, prev *Result, root *html.Node, opts Options, ids ...string) {
	t.Helper()

	full, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout (full relayout): %v", err)
	}

	if !reflect.DeepEqual(relayoutStripIDs(prev.Ops), relayoutStripIDs(full.Ops)) {
		t.Fatal("patched operations differ from a full relayout")
	}

	for _, id := range ids {
		got, want := relayoutBoxByID(t, prev, id), relayoutBoxByID(t, full, id)
		if !relayoutSameGeometry(got, want) {
			t.Fatalf("box %q = %+v, want the full relayout's %+v", id, *got, *want)
		}
	}
}

// TestRelayoutSizeEscalationWidthSplices proves a width-changing
// single-element relayout splices instead of refusing: the fill going
// 20% to 80% re-places the track it sits in, the card below the track stays
// where it was, and the page matches a full relayout.
func TestRelayoutSizeEscalationWidthSplices(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutSizeSliderHTML)
	opts := relayoutOptionsForCSS(t, relayoutSizeSliderCSS)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	before := relayoutBoxByID(t, prev, "fill").w

	relayoutNodeByID(t, root, "fill").Attrs["class"] = "wide"

	patch, err := RelayoutSubtree(t.Context(), prev, relayoutNodeByID(t, root, "fill"), opts)
	if err != nil {
		t.Fatalf("RelayoutSubtree: %v", err)
	}

	if patch.rootBox == nil || patch.rootBox.node == nil || patch.rootBox.node.Attribute("id") != "track" {
		t.Fatalf("patched root = %+v, want the track the resized fill sits in", patch.rootBox)
	}

	if wide := relayoutBoxByID(t, prev, "fill").w; wide <= before {
		t.Fatalf("resized fill width = %g, want more than the kept %g", wide, before)
	}

	assertSizeSpliceMatchesFull(t, prev, root, opts, "track", "fill", "after")
}

// TestRelayoutSizeEscalationHeightSplices proves a height-changing
// single-element relayout splices instead of refusing: the bar going 20% to
// 70% of the fixed chart re-places the chart, and the page matches a full
// relayout.
func TestRelayoutSizeEscalationHeightSplices(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutSizeChartHTML)
	opts := relayoutOptionsForCSS(t, relayoutSizeChartCSS)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	before := relayoutBoxByID(t, prev, "bar").height

	relayoutNodeByID(t, root, "bar").Attrs["class"] = "tall"

	patch, err := RelayoutSubtree(t.Context(), prev, relayoutNodeByID(t, root, "bar"), opts)
	if err != nil {
		t.Fatalf("RelayoutSubtree: %v", err)
	}

	if patch.rootBox == nil || patch.rootBox.node == nil || patch.rootBox.node.Attribute("id") != "chart" {
		t.Fatalf("patched root = %+v, want the chart the resized bar sits in", patch.rootBox)
	}

	if tall := relayoutBoxByID(t, prev, "bar").height; tall <= before {
		t.Fatalf("resized bar height = %g, want more than the kept %g", tall, before)
	}

	assertSizeSpliceMatchesFull(t, prev, root, opts, "chart", "bar")
}

// TestRelayoutSizeEscalationBatchSplices proves the batch path widens too: one
// resized id commits through its stable ancestor and the page matches a full
// relayout.
func TestRelayoutSizeEscalationBatchSplices(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutSizeSliderHTML)
	opts := relayoutOptionsForCSS(t, relayoutSizeSliderCSS)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	relayoutNodeByID(t, root, "fill").Attrs["class"] = "wide"

	out, err := RelayoutSubtrees(t.Context(), prev, []string{"fill"}, opts,
		OperationOwners(prev), PaintOrder(prev.Ops))
	if err != nil {
		t.Fatalf("RelayoutSubtrees: %v", err)
	}

	if len(out.Windows) != 1 {
		t.Fatalf("splice windows = %d, want 1", len(out.Windows))
	}

	assertSizeSpliceMatchesFull(t, prev, root, opts, "track", "fill", "after")
}

// TestRelayoutSizeEscalationExhaustionRefuses proves the refusal stays when no
// ancestor keeps its size: growing content-sized text to the root still
// reports the size change instead of splicing a wrong page.
func TestRelayoutSizeEscalationExhaustionRefuses(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutSizeGrowHTML)
	opts := relayoutOptionsForCSS(t, relayoutSizeGrowCSS)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	node := relayoutNodeByID(t, root, "box")
	node.Children[0].Text = "wrapping text wrapping text wrapping text wrapping text wrapping text "

	_, err = RelayoutSubtree(t.Context(), prev, node, opts)
	if !errors.Is(err, errSubtreeResized) {
		t.Fatalf("error = %v, want %v", err, errSubtreeResized)
	}
}

// TestRelayoutSizeEscalationFloatStillRefuses proves the escalation never
// papers over a structural refusal: a subtree holding a float is still
// refused even though widening past it would change the error.
func TestRelayoutSizeEscalationFloatStillRefuses(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>`+
		`<div id="host"><div id="floater" style="float:left">floated</div></div>`+
		`</body></html>`)
	opts := relayoutOptionsForCSS(t, `#host { width: 200px; }`)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	_, err = RelayoutSubtree(t.Context(), prev, relayoutNodeByID(t, root, "host"), opts)
	if !errors.Is(err, errSubtreeNotSplicable) {
		t.Fatalf("error = %v, want %v", err, errSubtreeNotSplicable)
	}
}
