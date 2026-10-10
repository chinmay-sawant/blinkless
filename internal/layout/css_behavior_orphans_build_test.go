package layout

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// TestOrphansKeepsLinesInFirstColumn is orphans in multicol: a lone
// anonymous text strip fragments across columns by height bands, and the
// first band boundary now snaps later so at least orphans line rows stay in
// the first column. The test lays out plain text, counts its used line rows
// per column, then re-lays out with orphans set two rows past the plain
// first-column count and asserts the used distribution moved there with no
// row lost. It asserts used geometry (per-column line counts), never a
// stored string. Reference: Chrome 143.0.7499.40 keeps orphans lines
// together before a column break.
func TestOrphansKeepsLinesInFirstColumn(t *testing.T) {
	t.Parallel()

	text := strings.Repeat("Multi-column sample text repeated. ", 12)
	doc := func(extra string) string {
		mcStyle := "width:300px;column-count:2;column-gap:20px;font-size:10pt"
		if extra != "" {
			mcStyle += ";" + extra
		}

		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="mc" style="` + mcStyle + `">` + text + `</div></body></html>`
	}

	plain := layoutHTML(t, doc(""))
	plainCols := orphanBuildColumnLineCounts(plain.Ops)
	total := plainCols[0] + plainCols[1]

	if total < 8 {
		t.Fatalf("plain strip rows = %d, want at least 8 to test orphans", total)
	}

	if plainCols[1] < 3 {
		t.Fatalf("plain second-column rows = %d, want at least 3", plainCols[1])
	}

	want := plainCols[0] + 2

	frag := layoutHTML(t, doc(fmt.Sprintf("orphans:%d", want)))

	if got := boxByID(t, frag, "mc").style.Orphans; got != want {
		t.Fatalf("orphans used = %d, want %d", got, want)
	}

	got := orphanBuildColumnLineCounts(frag.Ops)

	if got[0] != want {
		t.Errorf("orphans:%d first-column rows = %d, want %d", want, got[0], want)
	}

	if got[0]+got[1] != total {
		t.Errorf("orphans:%d total rows = %d, want %d (no row lost or clipped)", want, got[0]+got[1], total)
	}

	if got[0] == plainCols[0] {
		t.Errorf("orphans:%d first-column rows = %d, want a move from plain %d", want, got[0], plainCols[0])
	}
}

// TestSnapAnonBandForOrphans pins the band-snapping helper contract: no snap
// when the even split already keeps enough rows, a snap to the mid-gap after
// the Nth baseline when short, a blocked snap past a finite column cap, and
// a whole-strip keep when orphans outnumbers the rows.
func TestSnapAnonBandForOrphans(t *testing.T) {
	t.Parallel()

	top := 100.0
	rows := []float64{110, 122, 134, 146, 158, 170}
	totalH := 74.0
	bandH := totalH / 2
	midGap := (rows[4]+rows[5])/2 - top
	cases := []struct {
		name    string
		lines   []float64
		orphans int
		maxColH float64
		want    float64
		snapped bool
	}{
		{"kept", rows, 2, 0, bandH, false},
		{"short", rows, 5, 0, midGap, true},
		{"capped", rows, 5, 40, bandH, false},
		{"whole", rows, 9, 0, totalH, true},
		{"empty", nil, 5, 0, bandH, false},
	}

	for _, tc := range cases {
		got, ok := snapAnonBandForOrphans(top, totalH, bandH, tc.maxColH, tc.lines, tc.orphans)

		if ok != tc.snapped || !near(got, tc.want) {
			t.Errorf("%s: band = %.4f snap = %v, want %.4f snap %v",
				tc.name, got, ok, tc.want, tc.snapped)
		}
	}
}

// orphanBuildColumnLineCounts counts used text rows per column from display
// ops. Column 2 ops sit one column-plus-gap step (160px = 120pt here) right
// of column 1, so a 60pt threshold splits them; rows cluster baselines in a
// 1pt window, the same epsilon stripLineBaselines uses.
func orphanBuildColumnLineCounts(ops []Op) [2]int {
	const colSplitPt = 60.0

	texts := make([]Op, 0, len(ops))

	for _, paintOp := range ops {
		if paintOp.Kind != OpText || strings.TrimSpace(paintOp.Text) == "" {
			continue
		}

		texts = append(texts, paintOp)
	}

	var out [2]int

	if len(texts) == 0 {
		return out
	}

	minX := texts[0].X

	for _, paintOp := range texts[1:] {
		if paintOp.X < minX {
			minX = paintOp.X
		}
	}

	rows := [2][]float64{}

	for _, paintOp := range texts {
		col := 0

		if paintOp.X-minX > colSplitPt {
			col = 1
		}

		rows[col] = append(rows[col], paintOp.Y)
	}

	for col := range 2 {
		sort.Float64s(rows[col])
		out[col] = countOrphanRows(rows[col])
	}

	return out
}

// countOrphanRows counts distinct rows in sorted baselines.
func countOrphanRows(sorted []float64) int {
	count := 0

	for i := range sorted {
		if i > 0 && sorted[i]-sorted[i-1] < orphanLineEpsilonPt {
			continue
		}

		count++
	}

	return count
}
