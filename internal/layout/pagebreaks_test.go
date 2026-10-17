package layout

import "testing"

func TestPageBreaksExport(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body>`+
		`<div id="first" style="height:100px"></div>`+
		`<div id="brk" style="page-break-before: always; height:50px"></div>`+
		`<div id="aft" style="page-break-after: always; height:50px"></div>`+
		`<div id="keep" style="break-inside: avoid; height:60px"></div>`+
		`</body></html>`)

	forced, keep := PageBreaks(res, 800)

	if len(forced) != 2 {
		t.Fatalf("forced breaks = %v, want 2 entries", forced)
	}

	brk := boxByID(t, res, "brk")
	if forced[0] != brk.y {
		t.Errorf("forced[0] = %v, want break-before box top %v", forced[0], brk.y)
	}

	aft := boxByID(t, res, "aft")
	if forced[1] != aft.y+aft.height {
		t.Errorf("forced[1] = %v, want break-after box bottom %v", forced[1], aft.y+aft.height)
	}

	if len(keep) != 1 {
		t.Fatalf("keep ranges = %v, want 1 entry", keep)
	}

	kb := boxByID(t, res, "keep")
	if keep[0][0] != kb.y || keep[0][1] != kb.y+kb.height {
		t.Errorf("keep[0] = %v, want [%v %v]", keep[0], kb.y, kb.y+kb.height)
	}
}

// TestPageBreaksSingleLineKeep: a block whose only line cannot fit the page
// containing its top is exported as a keep range covering its border box, so
// a paginator moves the line with its decoration instead of splitting them.
func TestPageBreaksSingleLineKeep(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html style="margin:0"><body style="margin:0">`+
		`<div style="height:95pt"></div>`+
		`<div id="line">one line near the bottom</div>`+
		`</body></html>`)

	res, err := Layout(root, Options{
		Width: testViewport, Height: 100, Background: true,
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	line := boxByID(t, res, "line")
	if line.y >= 100 || line.y+line.height <= 100 {
		t.Fatalf("line box [%.2f %.2f] must straddle the page boundary (100)",
			line.y, line.y+line.height)
	}

	_, keep := PageBreaks(res, 100)
	if len(keep) != 1 || keep[0] != [2]float64{line.y, line.y + line.height} {
		t.Fatalf("keep = %v, want [%v %v]", keep, line.y, line.y+line.height)
	}

	// The same viewport height with the line comfortably inside the page
	// exports no keep: the line fits, so its block is not moved.
	resTall, err := Layout(mustParse(t, `<html style="margin:0"><body style="margin:0">`+
		`<div style="height:10pt"></div>`+
		`<div id="line">one line near the top</div>`+
		`</body></html>`), Options{
		Width: testViewport, Height: 100, Background: true,
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	if _, tallKeep := PageBreaks(resTall, 100); len(tallKeep) != 0 {
		t.Fatalf("keep = %v, want none for a fitting line", tallKeep)
	}
}

// TestPageBreaksDropsKeepsCutByForcedBreak: a forced break wins over
// keep-together, and the paginator shifts each keep endpoint at or below a
// forced break independently. A range that starts above the break and ends
// on it therefore comes back stretched (fixture-60: the last audit row's
// 44.93pt keep inflated to 492.50pt and then shifted the continuation table
// and footer 390.53pt down, adding a page). The exporter must drop such
// ranges; a range wholly below the break keeps its extent.
func TestPageBreaksDropsKeepsCutByForcedBreak(t *testing.T) { //nolint:cyclop // forced-break keep assertions
	t.Parallel()

	res := layoutHTML(t, `<html><body>`+
		`<div style="height:200px"></div>`+
		`<div id="cut" style="break-inside: avoid; height:40px"></div>`+
		`<div id="brk" style="page-break-before: always; height:40px"></div>`+
		`<div id="below" style="break-inside: avoid; height:40px"></div>`+
		`</body></html>`)

	forced, keep := PageBreaks(res, 0)

	if len(forced) != 1 {
		t.Fatalf("forced breaks = %v, want 1 entry", forced)
	}

	brk := boxByID(t, res, "brk")
	if forced[0] != brk.y {
		t.Fatalf("forced[0] = %v, want break-before box top %v", forced[0], brk.y)
	}

	cut := boxByID(t, res, "cut")
	if cut.y+cut.height != brk.y {
		t.Fatalf("cut box [%v..%v] does not end on the forced break %v",
			cut.y, cut.y+cut.height, brk.y)
	}

	for _, k := range keep {
		if k[0] == cut.y && k[1] == cut.y+cut.height {
			t.Fatalf("keep range %v for a box ending on the forced break survived", k)
		}
	}

	below := boxByID(t, res, "below")
	found := false

	for _, k := range keep {
		if k[0] == below.y && k[1] == below.y+below.height {
			found = true
		}
	}

	if !found {
		t.Fatalf("no keep range for the box below the forced break; keeps = %v", keep)
	}
}

func TestPageBreaksEmpty(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div style="height:10px"></div></body></html>`)
	forced, keep := PageBreaks(res, 800)

	if len(forced) != 0 || len(keep) != 0 {
		t.Errorf("breaks = (%v, %v), want empty", forced, keep)
	}

	if f, k := PageBreaks(nil, 800); len(f) != 0 || len(k) != 0 {
		t.Errorf("nil result breaks = (%v, %v), want empty", f, k)
	}

	// No page height means no page-relative single-line keeps.
	resLine := layoutHTML(t, `<html><body><div>one line</div></body></html>`)
	if _, k := PageBreaks(resLine, 0); len(k) != 0 {
		t.Errorf("keep with pageH 0 = %v, want empty", k)
	}
}
