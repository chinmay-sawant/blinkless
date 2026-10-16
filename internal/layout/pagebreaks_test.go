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

	forced, keep := PageBreaks(res)

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

func TestPageBreaksEmpty(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div style="height:10px"></div></body></html>`)
	forced, keep := PageBreaks(res)

	if len(forced) != 0 || len(keep) != 0 {
		t.Errorf("breaks = (%v, %v), want empty", forced, keep)
	}

	if f, k := PageBreaks(nil); len(f) != 0 || len(k) != 0 {
		t.Errorf("nil result breaks = (%v, %v), want empty", f, k)
	}
}
