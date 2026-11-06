package layout

import (
	"reflect"
	"testing"
)

// TestPageBreaksEqualIgnoresPaintOnlyDifferences pins that the comparison
// ignores what pagination cannot read: two placements that differ only in
// operation color carry the same pagination inputs.
func TestPageBreaksEqualIgnoresPaintOnlyDifferences(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><p id="a">first line</p><p id="b">second line</p></body></html>`)

	if !PageBreaksEqual(res, res) {
		t.Fatal("a placement differs from itself")
	}

	painted := CloneResult(res)

	for index := range painted.Ops {
		if painted.Ops[index].Kind == OpFillRect {
			painted.Ops[index].R, painted.Ops[index].G, painted.Ops[index].B = 0.2, 0.4, 0.8
		}
	}

	if !PageBreaksEqual(res, painted) {
		t.Fatal("a paint-only change reports different pagination inputs")
	}
}

// TestPageBreaksEqualDetectsEachPaginationInput pins that every input
// PageBreaks reads is compared: box geometry, the content height, a measured
// operation field, the operation count, the box tree shape, and a break
// style. A missed input would let a patch reuse stale breaks.
func TestPageBreaksEqualDetectsEachPaginationInput(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><p id="a">first line</p><p id="b">second line</p></body></html>`)

	moved := CloneResult(res)
	boxByID(t, moved, "b").y += 40

	if PageBreaksEqual(res, moved) {
		t.Fatal("a moved box reports equal pagination inputs")
	}

	taller := CloneResult(res)
	taller.Height += 40

	if PageBreaksEqual(res, taller) {
		t.Fatal("a changed content height reports equal pagination inputs")
	}

	baseline := CloneResult(res)

	movedLine := false

	for index := range baseline.Ops {
		if baseline.Ops[index].Kind == OpText {
			baseline.Ops[index].Y += 12
			movedLine = true

			break
		}
	}

	if !movedLine {
		t.Fatal("the fixture holds no text operation")
	}

	if PageBreaksEqual(res, baseline) {
		t.Fatal("a moved text line reports equal pagination inputs")
	}

	shortened := CloneResult(res)
	shortened.Ops = shortened.Ops[:len(shortened.Ops)-1]

	if PageBreaksEqual(res, shortened) {
		t.Fatal("a different operation count reports equal pagination inputs")
	}

	reshaped := CloneResult(res)
	reshaped.root.children = reshaped.root.children[:0]

	if PageBreaksEqual(res, reshaped) {
		t.Fatal("a different box tree reports equal pagination inputs")
	}

	broken := layoutHTML(t, `<html><body><p id="a">first line</p>`+
		`<p id="b" style="page-break-before:always">second line</p></body></html>`)

	unbrokenForced, _ := PageBreaks(res, 0)
	brokenForced, _ := PageBreaks(broken, 0)

	if reflect.DeepEqual(unbrokenForced, brokenForced) {
		t.Fatalf("the fixture does not change the forced breaks: %v", unbrokenForced)
	}

	if PageBreaksEqual(res, broken) {
		t.Fatal("a page-break-before change reports equal pagination inputs")
	}
}

// TestPageBreaksEqualReadsTableRowMetadata pins that the row heights a keep
// range is built from are compared. Table cells are not on the children track
// PageBreaks walks, so a row change is only visible in the table's own row
// metadata.
func TestPageBreaksEqualReadsTableRowMetadata(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><table>`+
		`<tr><td>cell one</td></tr><tr><td>cell two</td></tr>`+
		`</table></body></html>`)

	if !PageBreaksEqual(res, CloneResult(res)) {
		t.Fatal("a clone reports different pagination inputs")
	}

	rows := CloneResult(res)
	tables := tablesOf(rows)

	if len(tables) != 1 || len(tables[0].rowHeights) < 2 {
		t.Fatalf("the fixture laid out %d tables, want one with two rows", len(tables))
	}

	// The clone shares the row metadata slices, so the changed fixture needs
	// its own slice before the row height moves.
	heights := append([]float64(nil), tables[0].rowHeights...)
	heights[1] += 10
	tables[0].rowHeights = heights

	if PageBreaksEqual(res, rows) {
		t.Fatal("a changed row height reports equal pagination inputs")
	}
}
