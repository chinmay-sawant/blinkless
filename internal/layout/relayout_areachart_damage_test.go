package layout

import (
	"testing"
)

// relayoutAreachartCSS mirrors the ownframe AreaChart geometry: a fixed-height
// flex plot, flex columns, column-direction tracks packing a fixed dot above
// a percentage spacer, plus a band track with a painted band over a low
// spacer. Spacer and band heights are percentages of the shared plot scale,
// so a value change that moves the scale extremes rewrites every percentage.
const relayoutAreachartCSS = `
	#chart { width: 600px; background-color: #ffffff; }
	#plot { display: flex; height: 180px; background-color: #f0f0f0; }
	.col { display: flex; flex: 1; align-items: stretch; }
	.track { display: flex; flex-direction: column; justify-content: flex-end; align-items: center; flex: 1; }
	.dot { width: 7px; height: 7px; background-color: #3366cc; }
	.sp { width: 100%; }
	.band { width: 100%; background-color: #9999cc; }
`

// relayoutAreachartHTML holds three columns of two dot tracks plus one band
// track each. Scale is 0..80, so moving sp-0-1 from 60% to 50% of the kept
// scale touches one spacer, while moving the 80% maximum rescales the plot.
const relayoutAreachartHTML = `<html><body>` +
	`<div id="chart"><div id="plot">` +
	`<div id="col-0" class="col">` +
	`<div id="track-0-0" class="track"><div id="dot-0-0" class="dot"></div><div id="sp-0-0" class="sp" style="height:50%"></div></div>` +
	`<div id="track-0-1" class="track"><div id="dot-0-1" class="dot"></div><div id="sp-0-1" class="sp" style="height:75%"></div></div>` +
	`<div id="bandtrack-0" class="track"><div id="band-0" class="band" style="height:25%"></div><div id="bandlow-0" class="sp" style="height:50%"></div></div>` +
	`</div>` +
	`<div id="col-1" class="col">` +
	`<div id="track-1-0" class="track"><div id="dot-1-0" class="dot"></div><div id="sp-1-0" class="sp" style="height:37.5%"></div></div>` +
	`<div id="track-1-1" class="track"><div id="dot-1-1" class="dot"></div><div id="sp-1-1" class="sp" style="height:100%"></div></div>` +
	`<div id="bandtrack-1" class="track"><div id="band-1" class="band" style="height:62.5%"></div><div id="bandlow-1" class="sp" style="height:37.5%"></div></div>` +
	`</div>` +
	`<div id="col-2" class="col">` +
	`<div id="track-2-0" class="track"><div id="dot-2-0" class="dot"></div><div id="sp-2-0" class="sp" style="height:62.5%"></div></div>` +
	`<div id="track-2-1" class="track"><div id="dot-2-1" class="dot"></div><div id="sp-2-1" class="sp" style="height:87.5%"></div></div>` +
	`<div id="bandtrack-2" class="track"><div id="band-2" class="band" style="height:25%"></div><div id="bandlow-2" class="sp" style="height:62.5%"></div></div>` +
	`</div>` +
	`</div></div>` +
	`</body></html>`

// relayoutAreachartPatch runs the batch splice for ids and returns the windows.
func relayoutAreachartPatch(t *testing.T, prev *Result, opts Options, ids []string) []SpliceWindow {
	t.Helper()

	out, err := RelayoutSubtrees(t.Context(), prev, ids, opts,
		OperationOwners(prev), PaintOrder(prev.Ops))
	if err != nil {
		t.Fatalf("RelayoutSubtrees: %v", err)
	}

	return out.Windows
}

// relayoutDamageUnion unions the old and new damage of every window.
func relayoutDamageUnion(windows []SpliceWindow) ([4]float64, float64) {
	var union [4]float64

	unionSet := false

	sum := 0.0

	for _, w := range windows {
		for index, has := range []bool{w.HasOldDamage, w.HasNewDamage} {
			if !has {
				continue
			}

			rect := w.OldDamage
			if index == 1 {
				rect = w.NewDamage
			}

			if rect[2] <= 0 || rect[3] <= 0 {
				continue
			}

			sum += rect[2] * rect[3]

			if !unionSet {
				union = [4]float64{rect[0], rect[1], rect[2], rect[3]}
				unionSet = true

				continue
			}

			x0 := min(union[0], rect[0])
			y0 := min(union[1], rect[1])
			x1 := max(union[0]+union[2], rect[0]+rect[2])
			y1 := max(union[1]+union[3], rect[1]+rect[3])
			union = [4]float64{x0, y0, x1 - x0, y1 - y0}
		}
	}

	return union, sum
}

// TestRelayoutAreachartSingleValueStaysNarrow proves a value change inside the
// kept scale splices one single-operation window: the spacer moves its dot,
// the track keeps its size, and the damage is the dot's old and new rects.
func TestRelayoutAreachartSingleValueStaysNarrow(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutAreachartHTML)
	opts := relayoutOptionsForCSS(t, relayoutAreachartCSS)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	relayoutNodeByID(t, root, "sp-0-1").Attrs["style"] = "height:62.5%"

	windows := relayoutAreachartPatch(t, prev, opts, []string{"sp-0-1"})
	if len(windows) != 1 {
		t.Fatalf("splice windows = %d, want 1", len(windows))
	}

	window := windows[0]
	if window.OldCount != 1 || window.NewCount != 1 {
		t.Fatalf("window = %d->%d ops, want a single-operation splice", window.OldCount, window.NewCount)
	}

	if !window.HasOldDamage || !window.HasNewDamage {
		t.Fatalf("window damage = old %v new %v, want both", window.HasOldDamage, window.HasNewDamage)
	}

	for index, rect := range [][4]float64{window.OldDamage, window.NewDamage} {
		if area := rect[2] * rect[3]; area > 64 {
			t.Fatalf("damage rect %d = %v (area %g), want a dot-sized rect", index, rect, area)
		}
	}

	assertSizeSpliceMatchesFull(t, prev, root, opts, "plot", "col-0", "track-0-1", "sp-0-1")
}

// TestRelayoutAreachartScaleMoveCoversPlot proves the width of a scale-moving
// value change is inherent: one value moves the shared scale max 80 -> 95, so
// every spacer and band percentage changes. Each splice still wins the narrow
// track-level single-operation window, and the page matches a full relayout,
// but the union of those minimal windows necessarily spans the plot because
// every column's band fill and every dot moved. No tighter winner selection
// or finer per-window damage can narrow it: the changed paint itself is
// plot-wide.
func TestRelayoutAreachartScaleMoveCoversPlot(t *testing.T) {
	t.Parallel()

	root := mustParse(t, relayoutAreachartHTML)
	opts := relayoutOptionsForCSS(t, relayoutAreachartCSS)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	// New scale 0..95 after the 80% maximum moves to 95. Every percentage is
	// value/95*100, so every spacer, band, and low spacer changes.
	set := map[string]string{
		"sp-0-0": "height:42.105263%", "sp-0-1": "height:63.157895%",
		"band-0": "height:21.052632%", "bandlow-0": "height:42.105263%",
		"sp-1-0": "height:31.578947%", "sp-1-1": "height:100%",
		"band-1": "height:52.631579%", "bandlow-1": "height:31.578947%",
		"sp-2-0": "height:52.631579%", "sp-2-1": "height:73.684211%",
		"band-2": "height:21.052632%", "bandlow-2": "height:52.631579%",
	}
	ids := []string{
		"sp-0-0", "sp-0-1", "band-0", "bandlow-0",
		"sp-1-0", "sp-1-1", "band-1", "bandlow-1",
		"sp-2-0", "sp-2-1", "band-2", "bandlow-2",
	}
	for id, style := range set {
		relayoutNodeByID(t, root, id).Attrs["style"] = style
	}

	windows := relayoutAreachartPatch(t, prev, opts, ids)
	if len(windows) != len(ids) {
		t.Fatalf("splice windows = %d, want %d", len(windows), len(ids))
	}

	// Every splice holds the narrowest range: a single-operation window at
	// the track or band-track level. The escalation never splices a column,
	// the plot, or anything wider.
	for index, window := range windows {
		if window.OldCount != 1 || window.NewCount != 1 {
			t.Fatalf("window %d = %d->%d ops, want single-operation windows",
				index, window.OldCount, window.NewCount)
		}
	}

	union, _ := relayoutDamageUnion(windows)
	column := relayoutBoxByID(t, prev, "col-0")
	plot := relayoutBoxByID(t, prev, "plot")

	if union[2] <= column.w {
		t.Fatalf("damage width = %g, want wider than one column (%g)", union[2], column.w)
	}

	if union[3] <= plot.height/2 {
		t.Fatalf("damage height = %g, want more than half the plot (%g)", union[3], plot.height)
	}

	assertSizeSpliceMatchesFull(t, prev, root, opts, "plot", "col-0", "col-1", "col-2")
}
