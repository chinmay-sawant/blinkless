package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

// TestDrawingCornerShapeScoopPath is the WASM half of the corner-shape Phase 3
// gate: the versioned JSON drawing list must include the path kind and the
// path commands for one scooped box (checklist Phase 3, bindings/wasm/
// drawing_list.go row). It skips while the painter ignores corner-shape, and
// fails once native paints a path that the JSON drops or ships without
// commands.
//
// Reference is the CSS Borders 4 draft corner-shape section (fetched
// 2026-10-15, https://drafts.csswg.org/css-borders-4/#typedef-corner-shape-value).
// There is no Chrome parity claim; the catalog records no browser result.
//
// File name note: this must stay corner_shape_test.go, not
// corner_shape_wasm_test.go. The go tool reads a trailing _wasm before
// _test.go as a GOARCH suffix (wasm is a GOARCH), so that name is silently
// excluded from native test binaries via IgnoredGoFiles.

// cornerWasmKnownOpKeys is the current drawOpJSON key set in drawing_list.go.
// Any extra key on a path-kind op is path payload (commands). Kept in sync
// with the struct tags by hand; a new renderer field that is not path payload
// would need an entry here.
var cornerWasmKnownOpKeys = map[string]bool{ //nolint:gochecknoglobals // test key table
	"id": true, "kind": true, "kindValue": true,
	"x": true, "y": true, "w": true, "h": true,
	"r": true, "g": true, "b": true,
	"alpha": true, "opacity": true, "width": true,
	"size": true, "letterSpacing": true, "inkDescent": true, "rotateDeg": true,
	"text": true, "font": true, "image": true, "alt": true, "uri": true,
	"blendMode": true, "group": true, "groupMark": true, "transform": true,
	"radii": true, "segments": true,
	"textTransform": true, "textLanguage": true, "textAutospace": true,
	"fontFeatures": true, "strokeMask": true, "lineInset": true,
	"bold": true, "fakeBold": true, "fakeOblique": true, "noFakeBold": true,
	"isJPEG": true, "isBackground": true, "fixed": true, "pinned": true,
	"positioned": true, "stickyID": true, "zIndex": true, "zIndexSet": true,
	"outline": true,
}

// cornerWasmPathOps selects the native path-kind ops (future OpPath, explicit
// value 9 per the checklist) plus any op carrying PathValue points. Numeric
// on purpose: it compiles before corner_path.go exists.
func cornerWasmPathOps(display *layout.Display) []layout.DisplayOp {
	var out []layout.DisplayOp

	for _, op := range display.Ops {
		if uint8(op.Kind) == 9 {
			out = append(out, op)

			continue
		}

		if path, ok := op.PathValue(); ok && len(path.Points) > 0 {
			out = append(out, op)
		}
	}

	return out
}

// cornerWasmHasCommands reports whether a JSON path-kind op carries path
// payload: a non-empty value under a key outside the current drawOpJSON set,
// or (fallback) non-empty geometry under a reused key. The kind alone is not
// enough; the checklist requires the commands too.
func cornerWasmHasCommands(op map[string]any) bool {
	for key, value := range op {
		if cornerWasmKnownOpKeys[key] {
			continue
		}

		switch payload := value.(type) {
		case []any:
			if len(payload) > 0 {
				return true
			}
		case map[string]any:
			if len(payload) > 0 {
				return true
			}
		}
	}

	// Fallback: a future change may reuse an existing geometry key for the
	// corner path instead of adding a new one.
	if segs, ok := op["segments"].([]any); ok && len(segs) > 0 {
		return true
	}

	if radii, ok := op["radii"].(map[string]any); ok && len(radii) > 0 {
		return true
	}

	if xf, ok := op["transform"].(map[string]any); ok && len(xf) > 0 {
		return true
	}

	return false
}

func TestDrawingCornerShapeScoopPath(t *testing.T) {
	t.Parallel()

	const source = `<html><head><style>body{margin:0}</style></head><body>` +
		`<div style="width:120px;height:60px;background:#f00;border-radius:16px;corner-top-left-shape:scoop"></div>` +
		`</body></html>`

	const width, height = 640, 480

	display := nativeDisplay(t, source, width, height)
	nativePaths := cornerWasmPathOps(display)

	if len(nativePaths) == 0 {
		t.Skipf("corner-shape not yet wired: scooped box paints no path-kind op (kind 9) and no PathValue points")
	}

	result, err := Convert(context.Background(), Request{HTML: source, Width: width, Height: height}, nil)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}

	var doc struct {
		Schema string           `json:"schema"`
		Ops    []map[string]any `json:"ops"`
	}

	if err := json.Unmarshal(result.Bytes, &doc); err != nil {
		t.Fatalf("decode drawing list: %v", err)
	}

	var jsonPaths []map[string]any

	for _, op := range doc.Ops {
		kind, _ := op["kind"].(string)

		kindValue := -1
		if value, ok := op["kindValue"].(float64); ok {
			kindValue = int(value)
		}

		if kind == "path" || kindValue == 9 {
			jsonPaths = append(jsonPaths, op)
		}
	}

	if len(jsonPaths) == 0 {
		t.Fatalf("native paints %d path ops but WASM JSON has no path kind "+
			"(want kind \"path\" with kindValue 9); teach drawKind in bindings/wasm/drawing_list.go",
			len(nativePaths))
	}

	if len(jsonPaths) != len(nativePaths) {
		t.Errorf("WASM path ops = %d, native = %d, want equal (ops convert 1:1)", len(jsonPaths), len(nativePaths))
	}

	for _, op := range jsonPaths {
		if !cornerWasmHasCommands(op) {
			raw, _ := json.Marshal(op)
			t.Errorf("path op carries the kind but no commands "+
				"(want a non-empty path/commands payload): %s", raw)
		}
	}
}
