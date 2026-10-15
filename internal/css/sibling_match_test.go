package css

import "testing"

func wideSiblingRow() string {
	var src []byte

	src = append(src, `<div id="row"><span id="early"></span>`...)

	for range 80 {
		src = append(src, `<b></b>`...)
	}

	src = append(src, `<zzz></zzz>`...)

	for range 80 {
		src = append(src, `<b></b>`...)
	}

	src = append(src, `<span id="mid"></span>`...)

	for range 80 {
		src = append(src, `<i></i>`...)
	}

	src = append(src, `<span id="late"></span></div>`...)

	return string(src)
}

func TestGeneralSiblingWide(t *testing.T) {
	t.Parallel()

	root := treeFor(t, wideSiblingRow())
	sel, ok := parseSelector("zzz ~ span")

	if !ok {
		t.Fatal("parse zzz ~ span")
	}

	early := byID(root, "early")
	mid := byID(root, "mid")
	late := byID(root, "late")

	if early == nil || mid == nil || late == nil {
		t.Fatal("missing spans")
	}

	if Match(sel, early) || !Match(sel, mid) || !Match(sel, late) {
		t.Error("zzz ~ span matched the wrong spans")
	}
}
