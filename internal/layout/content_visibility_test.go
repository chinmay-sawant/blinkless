package layout

import (
	"strings"
	"testing"
)

// TestContentVisibilityHiddenSkipsDescendantLayout: content-visibility:hidden
// skips descendant layout and paint and keeps the box's own definite size;
// an auto content size falls back to contain-intrinsic-size
// (internal/layout/layout_flow.go:217-227, intrinsic measure at
// internal/layout/layout_measure.go:95-115). Browser intent:
// testdata/golden/fixture-61-implemented-props-b.html row 43.
func TestContentVisibilityHiddenSkipsDescendantLayout(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="definite" style="content-visibility:hidden;height:40pt">hidden words one</div>`+
		`<div id="auto" style="content-visibility:hidden;contain-intrinsic-size:30pt">hidden words two</div>`+
		`<span id="inline" style="display:inline-block;content-visibility:hidden;`+
		`contain-intrinsic-size:30pt">hidden words three</span>`+
		`<p id="after" style="margin:0">visible</p>`+
		`</body></html>`)

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "hidden words") {
			t.Errorf("content-visibility:hidden painted descendant text %q", op.Text)
		}
	}

	definite := boxByID(t, res, "definite")
	if !near(definite.height, 40) {
		t.Errorf("definite hidden box height = %.2f, want 40 (keeps its own size)", definite.height)
	}

	auto := boxByID(t, res, "auto")
	if !near(auto.height, 30) {
		t.Errorf("hidden box with contain-intrinsic-size:30pt height = %.2f, want 30", auto.height)
	}

	inline := boxByID(t, res, "inline")
	if !near(inline.w, 30) {
		t.Errorf("hidden inline-block intrinsic width = %.2f, want 30", inline.w)
	}

	after := boxByID(t, res, "after")
	if after.y < auto.y+auto.height-0.01 {
		t.Errorf("after.y = %.2f, want at or below %.2f (no descendant height)",
			after.y, auto.y+auto.height)
	}
}
