package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

// dotPNG is the same self-contained 1x1 data URL the WASM fixture uses.
const dotPNG = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

// richSource exercises every paint kind plus resources, transforms, groups,
// and z-order in one document.
const richSource = `<!DOCTYPE html>
<html><head><style>
  body { margin: 0; font-family: sans-serif; }
  .box { width: 120px; height: 40px; background: #336699; }
  .round { width: 80px; height: 40px; background: rgba(255,0,0,0.5); border-radius: 8px; }
  .ring { width: 60px; height: 30px; border: 3px solid #245b8f; border-radius: 5px; box-sizing: border-box; }
  .rule { width: 100px; height: 20px; border-left: 4px solid #000; box-sizing: border-box; }
  .iso { isolation: isolate; background: #eee; }
  .blend { mix-blend-mode: multiply; background: #f0f; }
  .shift { transform: rotate(10deg); transform-origin: 0 0; }
  .hi { position: relative; z-index: 4; }
</style></head><body>
  <div class="box hi">alpha</div>
  <div class="round"></div>
  <div class="ring"></div>
  <div class="rule"></div>
  <div class="iso"><p>grouped</p></div>
  <div class="blend">blended</div>
  <div class="shift box">turned</div>
  <p><a href="https://example.com/x">link</a></p>
  <img alt="dot" src="` + dotPNG + `">
  <ul><li>item</li></ul>
  <table style="border-collapse:collapse"><tr><td style="border:1px solid #000">cell</td></tr></table>
</body></html>`

func convertSource(t *testing.T, source string, width, height int) (Result, drawingListJSON) {
	t.Helper()

	result, err := Convert(context.Background(), Request{HTML: source, Width: width, Height: height}, nil)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}

	var list drawingListJSON
	if err := json.Unmarshal(result.Bytes, &list); err != nil {
		t.Fatalf("decode drawing list: %v", err)
	}

	return result, list
}

func nativeDisplay(t *testing.T, source string, width, height int) *layout.Display {
	t.Helper()

	ctx := context.Background()

	tree, err := html.Parse([]byte(source))
	if err != nil {
		t.Fatalf("html parse: %v", err)
	}

	styled, err := css.Apply(ctx, tree, css.Options{WidthPx: width, HeightPx: height, Media: "screen"})
	if err != nil {
		t.Fatalf("css apply: %v", err)
	}

	display, err := layout.DisplayListOptions(ctx, styled, layout.Options{Images: inlineImageResolver(ctx)})
	if err != nil {
		t.Fatalf("display list: %v", err)
	}

	return display
}

func TestConvertReturnsVersionedDrawingList(t *testing.T) {
	t.Parallel()

	result, list := convertSource(t, `<p>hello</p>`, 640, 480)

	if result.Mode != displayMode {
		t.Fatalf("mode %q, want %q", result.Mode, displayMode)
	}

	if result.MIME != "application/json" {
		t.Fatalf("mime %q", result.MIME)
	}

	if list.Schema != drawingListSchema || list.Version != drawingListSchemaVersion {
		t.Fatalf("schema %q version %d", list.Schema, list.Version)
	}

	if list.Units != drawingListUnits {
		t.Fatalf("units %q", list.Units)
	}

	if result.Width != list.Width || result.Height != list.Height {
		t.Fatalf("result canvas %dx%d, payload %dx%d", result.Width, result.Height, list.Width, list.Height)
	}

	if len(list.Ops) == 0 {
		t.Fatal("no operations")
	}

	if list.Fonts == nil || list.Images == nil || list.Groups == nil || list.Boxes == nil || list.Order == nil {
		t.Fatal("resource or order arrays must serialize as arrays, not null")
	}
}

// TestDrawingListAgreesWithNativeDisplay re-runs the native pipeline for the
// same document and compares every serialized field with the public
// layout.Display values the browser adapter consumed.
func TestDrawingListAgreesWithNativeDisplay(t *testing.T) {
	t.Parallel()

	const width, height = 800, 600

	display := nativeDisplay(t, richSource, width, height)
	result, list := convertSource(t, richSource, width, height)

	if result.Width != display.Width || result.Height != display.Height {
		t.Fatalf("canvas %dx%d, want %dx%d", list.Width, list.Height, display.Width, display.Height)
	}

	if list.PxPerPt != display.PixelPerPoint || list.PtPerPx != display.PointsPerPixel {
		t.Fatalf("unit factors %v/%v, want %v/%v",
			list.PxPerPt, list.PtPerPx, display.PixelPerPoint, display.PointsPerPixel)
	}

	if len(list.Ops) != len(display.Ops) {
		t.Fatalf("ops %d, want %d", len(list.Ops), len(display.Ops))
	}

	if len(list.Order) != len(display.Order) {
		t.Fatalf("order %d, want %d", len(list.Order), len(display.Order))
	}

	for i := range display.Order {
		if list.Order[i] != display.Order[i] {
			t.Fatalf("order[%d] = %d, want %d", i, list.Order[i], display.Order[i])
		}
	}

	for i := range display.Ops {
		assertOpMatchesNative(t, &list.Ops[i], &display.Ops[i])
	}

	if len(list.Boxes) != len(display.Boxes) {
		t.Fatalf("boxes %d, want %d", len(list.Boxes), len(display.Boxes))
	}

	for i := range display.Boxes {
		box := &display.Boxes[i]
		got := &list.Boxes[i]

		if got.ID != box.ID || got.Tag != box.Tag || got.Action != box.Action || got.Text != box.Text ||
			got.X != box.X || got.Y != box.Y || got.W != box.W || got.H != box.H {
			t.Fatalf("box[%d] = %+v, want %+v", i, *got, *box)
		}
	}

	assertReferencesResolve(t, list)
	assertOrderIsPermutation(t, list)

	if len(list.Groups) == 0 || len(list.Images) == 0 {
		t.Fatal("rich fixture must produce groups and image resources")
	}
}

func assertOpMatchesNative(t *testing.T, got *drawOpJSON, op *layout.DisplayOp) {
	t.Helper()

	if got.ID != op.ID || got.KindValue != uint8(op.Kind) {
		t.Fatalf("op %d identity %d/%d, want %d/%d", op.ID, got.ID, got.KindValue, op.ID, op.Kind)
	}

	if got.Kind != expectedKindName(op) {
		t.Fatalf("op %d kind %q, want %q", op.ID, got.Kind, expectedKindName(op))
	}

	if got.X != op.X || got.Y != op.Y || got.W != op.W || got.H != op.H {
		t.Fatalf("op %d geometry (%v,%v,%v,%v), want (%v,%v,%v,%v)",
			op.ID, got.X, got.Y, got.W, got.H, op.X, op.Y, op.W, op.H)
	}

	if got.R != op.R || got.G != op.G || got.B != op.B || got.Alpha != op.Alpha {
		t.Fatalf("op %d color (%v,%v,%v,%v), want (%v,%v,%v,%v)",
			op.ID, got.R, got.G, got.B, got.Alpha, op.R, op.G, op.B, op.Alpha)
	}

	if got.Opacity != op.Opacity() {
		t.Fatalf("op %d opacity %v, want %v", op.ID, got.Opacity, op.Opacity())
	}

	if got.Width != op.Width || got.Size != op.Size || got.LetterSpacing != op.LetterSpacing ||
		got.InkDescent != op.InkDescent || got.RotateDeg != float64(op.RotateDeg) {
		t.Fatalf("op %d text metrics mismatch: %+v", op.ID, *got)
	}

	if got.Text != op.Text || got.URI != op.LinkURI() || got.BlendMode != op.BlendModeName() {
		t.Fatalf("op %d payload text=%q uri=%q blend=%q", op.ID, got.Text, got.URI, got.BlendMode)
	}

	if got.TextTransform != op.TextTransformValue() || got.TextLanguage != op.TextLanguage() ||
		got.TextAutospace != op.TextAutospace() || got.FontFeatures != op.FontFeatures() {
		t.Fatalf("op %d shaping metadata mismatch", op.ID)
	}

	if got.StrokeMask != op.StrokeMask || got.LineInset != op.LineInset {
		t.Fatalf("op %d stroke mask %d inset %d, want %d/%d",
			op.ID, got.StrokeMask, got.LineInset, op.StrokeMask, op.LineInset)
	}

	if got.Bold != op.Bold || got.FakeOblique != op.FakeOblique ||
		got.NoFakeBold != op.NoFakeBoldValue() || got.Outline != op.Outline() {
		t.Fatalf("op %d flag mismatch: %+v", op.ID, *got)
	}

	if got.FakeBold != layout.DisplayFakeBold(op) {
		t.Fatalf("op %d fakeBold %v, want %v", op.ID, got.FakeBold, layout.DisplayFakeBold(op))
	}

	if got.IsJPEG != op.IsJPEG || got.IsBackground != op.IsBackground || got.Fixed != op.Fixed ||
		got.Pinned != op.Pinned || got.Positioned != op.Positioned || got.StickyID != op.StickyID ||
		got.ZIndex != op.ZIndex || got.ZIndexSet != op.ZIndexSet {
		t.Fatalf("op %d paint layer mismatch: %+v", op.ID, *got)
	}

	if op.XformSet {
		matrix := op.Transform()

		if got.Transform == nil || got.Transform.A != matrix.A || got.Transform.B != matrix.B ||
			got.Transform.C != matrix.C || got.Transform.D != matrix.D ||
			got.Transform.E != matrix.E || got.Transform.F != matrix.F {
			t.Fatalf("op %d transform %+v, want %+v", op.ID, got.Transform, matrix)
		}
	} else if got.Transform != nil {
		t.Fatalf("op %d has a transform but xformSet is false", op.ID)
	}

	if op.Font != nil && got.FontID == "" {
		t.Fatalf("op %d carries a face but no font reference", op.ID)
	}

	if op.Font == nil && got.FontID != "" {
		t.Fatalf("op %d has font reference %q with no native face", op.ID, got.FontID)
	}

	if data, _, _ := op.ImageBytes(); data != nil && got.ImageID == "" {
		t.Fatalf("op %d carries image bytes but no image reference", op.ID)
	}

	if data, _, _ := op.ImageBytes(); data == nil && got.ImageID != "" {
		t.Fatalf("op %d has image reference %q with no native payload", op.ID, got.ImageID)
	}

	if group := op.Group(); group != nil {
		if got.GroupID == nil || *got.GroupID != group.ID {
			t.Fatalf("op %d group %v, want %d", op.ID, got.GroupID, group.ID)
		}
	} else if got.GroupID != nil {
		t.Fatalf("op %d references group %d but native group is nil", op.ID, *got.GroupID)
	}

	switch {
	case op.IsGroupBegin():
		if got.GroupMark != "begin" {
			t.Fatalf("op %d group mark %q, want begin", op.ID, got.GroupMark)
		}
	case op.IsGroupEnd():
		if got.GroupMark != "end" {
			t.Fatalf("op %d group mark %q, want end", op.ID, got.GroupMark)
		}
	case got.GroupMark != "":
		t.Fatalf("op %d has group mark %q without a native boundary", op.ID, got.GroupMark)
	}

	assertRadiiMatch(t, got, op)
	assertSegmentsMatch(t, got, op)
}

func assertRadiiMatch(t *testing.T, got *drawOpJSON, op *layout.DisplayOp) {
	t.Helper()

	hasRadii := op.Radius != 0 || op.RadiusY != 0 || op.RadiusTopLeft != 0 || op.RadiusTopRight != 0 ||
		op.RadiusBottomRight != 0 || op.RadiusBottomLeft != 0 || op.RadiusTopLeftY != 0 ||
		op.RadiusTopRightY != 0 || op.RadiusBottomRightY != 0 || op.RadiusBottomLeftY != 0

	if !hasRadii {
		if got.Radii != nil {
			t.Fatalf("op %d has radii but native has none", op.ID)
		}

		return
	}

	want := drawRadiiJSON{
		Radius:       op.Radius,
		RadiusY:      op.RadiusY,
		TopLeft:      op.RadiusTopLeft,
		TopRight:     op.RadiusTopRight,
		BottomRight:  op.RadiusBottomRight,
		BottomLeft:   op.RadiusBottomLeft,
		TopLeftY:     op.RadiusTopLeftY,
		TopRightY:    op.RadiusTopRightY,
		BottomRightY: op.RadiusBottomRightY,
		BottomLeftY:  op.RadiusBottomLeftY,
	}

	if got.Radii == nil || *got.Radii != want {
		t.Fatalf("op %d radii %+v, want %+v", op.ID, got.Radii, want)
	}
}

func assertSegmentsMatch(t *testing.T, got *drawOpJSON, op *layout.DisplayOp) {
	t.Helper()

	if op.Grid == nil {
		if got.Segments != nil {
			t.Fatalf("op %d has grid segments but native grid is nil", op.ID)
		}

		return
	}

	if len(got.Segments) != len(op.Grid.Segs) {
		t.Fatalf("op %d segments %d, want %d", op.ID, len(got.Segments), len(op.Grid.Segs))
	}

	for i := range op.Grid.Segs {
		seg := &op.Grid.Segs[i]
		gotSeg := &got.Segments[i]

		if gotSeg.X != seg.X || gotSeg.Y != seg.Y || gotSeg.W != seg.W || gotSeg.H != seg.H ||
			gotSeg.Width != seg.Width || gotSeg.R != seg.R || gotSeg.G != seg.G ||
			gotSeg.B != seg.B || gotSeg.LineInset != seg.LineInset {
			t.Fatalf("op %d segment[%d] = %+v, want %+v", op.ID, i, *gotSeg, *seg)
		}
	}
}

// expectedKindName is the test's independent copy of the kind contract. A bug
// in the serializer's own mapping must not hide behind a shared helper.
func expectedKindName(op *layout.DisplayOp) string {
	switch {
	case op.IsGroupBegin():
		return kindGroupBegin
	case op.IsGroupEnd():
		return kindGroupEnd
	}

	switch op.Kind {
	case layout.DisplayOpNoop:
		return kindNoop
	case layout.DisplayOpFillRect:
		return kindFillRect
	case layout.DisplayOpStrokeRect:
		return kindStrokeRect
	case layout.DisplayOpLine:
		return kindLine
	case layout.DisplayOpText:
		return kindText
	case layout.DisplayOpImage:
		return kindImage
	case layout.DisplayOpLinkURI:
		return kindLinkURI
	case layout.DisplayOpBullet:
		return kindBullet
	case layout.DisplayOpGridRun:
		return kindGridRun
	default:
		return kindUnknown
	}
}

func assertReferencesResolve(t *testing.T, list drawingListJSON) {
	t.Helper()

	fontIDs := make(map[string]struct{}, len(list.Fonts))
	for _, face := range list.Fonts {
		fontIDs[face.ID] = struct{}{}
	}

	imageIDs := make(map[string]struct{}, len(list.Images))
	for _, image := range list.Images {
		imageIDs[image.ID] = struct{}{}
	}

	groupIDs := make(map[int]struct{}, len(list.Groups))
	for _, group := range list.Groups {
		groupIDs[group.ID] = struct{}{}
	}

	for _, group := range list.Groups {
		if group.Parent == nil {
			continue
		}

		if _, ok := groupIDs[*group.Parent]; !ok {
			t.Fatalf("group %d references missing parent %d", group.ID, *group.Parent)
		}
	}

	for i := range list.Ops {
		op := &list.Ops[i]

		if op.FontID != "" {
			if _, ok := fontIDs[op.FontID]; !ok {
				t.Fatalf("op %d references missing font %q", i, op.FontID)
			}
		}

		if op.ImageID != "" {
			if _, ok := imageIDs[op.ImageID]; !ok {
				t.Fatalf("op %d references missing image %q", i, op.ImageID)
			}
		}

		if op.GroupID != nil {
			if _, ok := groupIDs[*op.GroupID]; !ok {
				t.Fatalf("op %d references missing group %d", i, *op.GroupID)
			}
		}
	}
}

func assertOrderIsPermutation(t *testing.T, list drawingListJSON) {
	t.Helper()

	seen := make([]bool, len(list.Ops))

	for _, index := range list.Order {
		if index < 0 || index >= len(list.Ops) {
			t.Fatalf("order index %d out of range for %d ops", index, len(list.Ops))
		}

		if seen[index] {
			t.Fatalf("order repeats index %d", index)
		}

		seen[index] = true
	}

	if len(list.Order) != len(list.Ops) {
		t.Fatalf("order covers %d of %d ops", len(list.Order), len(list.Ops))
	}
}

func TestUnitsAndBaselineSemantics(t *testing.T) {
	t.Parallel()

	const source = `<div style="width:100px;height:50px;background:#000"></div><p>baseline</p>`

	display := nativeDisplay(t, source, 640, 480)
	_, list := convertSource(t, source, 640, 480)

	if list.PxPerPt <= 1 || list.PtPerPx >= 1 || list.PxPerPt*list.PtPerPx != 1 {
		t.Fatalf("unit factors pxPerPt=%v ptPerPx=%v", list.PxPerPt, list.PtPerPx)
	}

	if list.Width != display.Width || list.Height != display.Height {
		t.Fatalf("canvas %dx%d, want %dx%d", list.Width, list.Height, display.Width, display.Height)
	}

	var fill *drawOpJSON

	for i := range list.Ops {
		if list.Ops[i].Kind == kindFillRect && list.Ops[i].W > 0 {
			fill = &list.Ops[i]

			break
		}
	}

	if fill == nil {
		t.Fatal("no fillRect for the 100px box")
	}

	if fill.W != 75 {
		t.Fatalf("fillRect width %v points, want 75 (100 CSS px)", fill.W)
	}

	boxFound := false

	for _, box := range list.Boxes {
		if box.Tag == "div" && box.W == 100 {
			boxFound = true
		}
	}

	if !boxFound {
		t.Fatal("no div border box at 100 CSS pixels")
	}

	baselineFound := false

	for i := range display.Ops {
		if display.Ops[i].Kind != layout.DisplayOpText {
			continue
		}

		baselineFound = list.Ops[i].Y == display.Ops[i].Y && list.Ops[i].H == display.Ops[i].H

		break
	}

	if !baselineFound {
		t.Fatal("text baseline/line-box metrics drifted from the native op")
	}
}

func TestKindCoverage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		kind   string
		source string
		check  func(t *testing.T, op *drawOpJSON)
	}{
		{
			name:   "fillRect",
			source: `<div style="width:40px;height:20px;background:#123456"></div>`,
			check: func(t *testing.T, op *drawOpJSON) {
				t.Helper()

				if op.W <= 0 || op.H <= 0 || op.R+op.G+op.B == 0 {
					t.Fatalf("fill payload %+v", *op)
				}
			},
		},
		{
			name:   "strokeRect",
			source: `<div style="width:40px;height:20px;border:3px solid #f00;border-radius:4px;box-sizing:border-box"></div>`,
			check: func(t *testing.T, op *drawOpJSON) {
				t.Helper()

				if op.Width <= 0 || op.Radii == nil || op.Radii.Radius <= 0 {
					t.Fatalf("stroke payload %+v", *op)
				}
			},
		},
		{
			name:   "line",
			source: `<div style="width:40px;height:20px;border-left:3px solid #00f;box-sizing:border-box"></div>`,
			check: func(t *testing.T, op *drawOpJSON) {
				t.Helper()

				if op.Width <= 0 || op.LineInset == 0 {
					t.Fatalf("line payload %+v", *op)
				}
			},
		},
		{
			name:   "text",
			source: `<p style="font-weight:bold">hello</p>`,
			check: func(t *testing.T, op *drawOpJSON) {
				t.Helper()

				if op.Text == "" || op.FontID == "" || op.Size <= 0 {
					t.Fatalf("text payload %+v", *op)
				}
			},
		},
		{
			name:   "image",
			source: `<img alt="dot" src="` + dotPNG + `">`,
			check: func(t *testing.T, op *drawOpJSON) {
				t.Helper()

				if op.ImageID == "" || op.Alt != "dot" || op.W <= 0 || op.H <= 0 {
					t.Fatalf("image payload %+v", *op)
				}
			},
		},
		{
			name:   "linkURI",
			source: `<a href="https://example.com/page">go</a>`,
			check: func(t *testing.T, op *drawOpJSON) {
				t.Helper()

				if op.URI != "https://example.com/page" {
					t.Fatalf("link payload %+v", *op)
				}
			},
		},
		{
			name:   "bullet",
			source: `<ul><li>item</li></ul>`,
			check: func(t *testing.T, op *drawOpJSON) {
				t.Helper()

				if op.Text == "" || op.FontID == "" {
					t.Fatalf("bullet payload %+v", *op)
				}
			},
		},
		{
			name:   "gridRun",
			source: `<table style="border-collapse:collapse"><tr><td style="border:1px solid #000">cell</td></tr></table>`,
			check: func(t *testing.T, op *drawOpJSON) {
				t.Helper()

				if len(op.Segments) == 0 || op.Width <= 0 {
					t.Fatalf("grid payload %+v", *op)
				}
			},
		},
		{
			name:   "groupBegin",
			source: `<div style="isolation:isolate"><p>in</p></div>`,
			check: func(t *testing.T, op *drawOpJSON) {
				t.Helper()

				if op.GroupMark != "begin" || op.GroupID == nil {
					t.Fatalf("group payload %+v", *op)
				}
			},
		},
		{
			name:   "groupEnd",
			source: `<div style="mix-blend-mode:multiply"><p>in</p></div>`,
			check: func(t *testing.T, op *drawOpJSON) {
				t.Helper()

				if op.GroupMark != "end" || op.GroupID == nil {
					t.Fatalf("group payload %+v", *op)
				}
			},
		},
		{
			name:   "transform",
			kind:   "fillRect",
			source: `<div style="transform:rotate(30deg);transform-origin:0 0;width:40px;height:20px;background:#0aa"></div>`,
			check: func(t *testing.T, op *drawOpJSON) {
				t.Helper()

				if op.Transform == nil || op.Transform.A == 1 {
					t.Fatalf("transform payload %+v", *op)
				}
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, list := convertSource(t, testCase.source, 640, 480)

			wantKind := testCase.kind
			if wantKind == "" {
				wantKind = testCase.name
			}

			for i := range list.Ops {
				if list.Ops[i].Kind != wantKind {
					continue
				}

				testCase.check(t, &list.Ops[i])

				return
			}

			t.Fatalf("no %s operation in %+v", wantKind, list.Ops)
		})
	}
}

func TestResourcesAreSharedAndStable(t *testing.T) {
	t.Parallel()

	source := `<p>one</p><p>two</p><img src="` + dotPNG + `"><img src="` + dotPNG + `">`
	_, first := convertSource(t, source, 640, 480)
	_, second := convertSource(t, source, 640, 480)

	if len(first.Fonts) != 1 {
		t.Fatalf("fonts %d, want 1 shared face", len(first.Fonts))
	}

	if len(first.Images) != 1 {
		t.Fatalf("images %d, want 1 shared payload", len(first.Images))
	}

	if len(second.Fonts) != 1 || second.Fonts[0].ID != first.Fonts[0].ID {
		t.Fatalf("font ID not stable across runs: %v vs %v", second.Fonts, first.Fonts)
	}

	if len(second.Images) != 1 || second.Images[0].ID != first.Images[0].ID {
		t.Fatalf("image ID not stable across runs: %v vs %v", second.Images, first.Images)
	}

	face := first.Fonts[0]
	if face.UnitsPerEm <= 0 || face.ByteLength <= 0 {
		t.Fatalf("font metrics %+v", face)
	}

	raw, err := base64.StdEncoding.DecodeString(face.Bytes)
	if err != nil || len(raw) != face.ByteLength {
		t.Fatalf("font bytes do not decode: %v", err)
	}

	image := first.Images[0]
	if image.Format != "png" || image.PixelWidth != 1 || image.PixelHeight != 1 {
		t.Fatalf("image metadata %+v", image)
	}

	raw, err = base64.StdEncoding.DecodeString(image.Bytes)
	if err != nil || len(raw) != image.ByteLength {
		t.Fatalf("image bytes do not decode: %v", err)
	}

	imageOps := 0

	for i := range first.Ops {
		if first.Ops[i].Kind == kindImage {
			imageOps++

			if first.Ops[i].ImageID != image.ID {
				t.Fatalf("image op references %q, want %q", first.Ops[i].ImageID, image.ID)
			}
		}
	}

	if imageOps != 2 {
		t.Fatalf("image ops %d, want 2", imageOps)
	}

	assertReferencesResolve(t, first)
}

func TestInlineImageResolverRejectsNonDataSources(t *testing.T) {
	t.Parallel()

	resolver := inlineImageResolver(context.Background()) //nolint:contextcheck // resolver captures the test context

	for _, source := range []string{"https://example.com/a.png", "logo.png", "/tmp/a.png", ""} {
		if _, err := resolver(source); !errors.Is(err, errNonInlineImage) {
			t.Fatalf("resolver(%q) error %v, want errNonInlineImage", source, err)
		}
	}

	if _, err := resolver("data:image/png;base64,!!!!"); err == nil {
		t.Fatal("malformed data URL must fail")
	}

	body, err := resolver(dotPNG)
	if err != nil || len(body) == 0 {
		t.Fatalf("data URL did not resolve: %v", err)
	}
}

func TestPaintOrderMatchesNative(t *testing.T) {
	t.Parallel()

	const source = `<div style="position:relative;z-index:3;height:8px;background:#ccc">hi</div><div style="height:8px;background:#999">lo</div>`

	display := nativeDisplay(t, source, 640, 480)
	_, list := convertSource(t, source, 640, 480)

	assertOrderIsPermutation(t, list)

	nativeOrder := layout.DisplayOrder(display.Ops)
	if len(nativeOrder) != len(list.Order) {
		t.Fatalf("order length %d, want %d", len(list.Order), len(nativeOrder))
	}

	for i := range nativeOrder {
		if nativeOrder[i] != list.Order[i] {
			t.Fatalf("order[%d] = %d, want %d", i, list.Order[i], nativeOrder[i])
		}
	}

	highIndex, lowIndex := -1, -1

	for i := range list.Ops {
		if list.Ops[i].Text == "hi" {
			highIndex = i
		}

		if list.Ops[i].Text == "lo" {
			lowIndex = i
		}
	}

	if highIndex < 0 || lowIndex < 0 {
		t.Fatal("z-index fixture text ops missing")
	}

	positionOf := func(index int) int {
		for position, candidate := range list.Order {
			if candidate == index {
				return position
			}
		}

		return -1
	}

	if positionOf(highIndex) <= positionOf(lowIndex) {
		t.Fatal("z-index:3 op must paint after the in-flow op")
	}
}

func TestDecodeRequestRejectsMalformedInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		raw     string
		wantErr error
	}{
		{"empty", "", errInvalidRequest},
		{"not json", "not json", errInvalidRequest},
		{"empty html", `{"html":""}`, errInvalidRequest},
		{"unknown writer field", `{"html":"<p>x</p>","pageSize":"A4"}`, errInvalidRequest},
		{"quality field", `{"html":"<p>x</p>","quality":80}`, errInvalidRequest},
		{"padding field", `{"html":"<p>x</p>","padding":10}`, errInvalidRequest},
		{"multiple values", `{"html":"<p>x</p>"}{}`, errInvalidRequest},
		{"png mode", `{"html":"<p>x</p>","mode":"png"}`, errUnsupportedMode},
		{"jpeg mode", `{"html":"<p>x</p>","mode":"jpeg"}`, errUnsupportedMode},
		{"bogus mode", `{"html":"<p>x</p>","mode":"svg"}`, errUnsupportedMode},
		{"negative width", `{"html":"<p>x</p>","width":-1}`, errViewportTooLarge},
		{"oversized viewport", `{"html":"<p>x</p>","width":8192}`, errViewportTooLarge},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := DecodeRequest(testCase.raw)
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("error %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestDecodeRequestDefaultsToDisplay(t *testing.T) {
	t.Parallel()

	request, err := DecodeRequest(`{"html":"<p>x</p>"}`)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if request.Mode != displayMode {
		t.Fatalf("mode %q, want %q", request.Mode, displayMode)
	}
}

func TestDecodeRequestRejectsOversizedHTML(t *testing.T) {
	t.Parallel()

	raw := `{"html":"` + strings.Repeat("a", maxHTMLBytes+1) + `"}`

	if _, err := DecodeRequest(raw); !errors.Is(err, errInputTooLarge) {
		t.Fatalf("error %v, want errInputTooLarge", err)
	}
}

// TestConvertEnforcesOutputSize lowers the cap instead of building a document
// large enough to cross 32 MiB. It does not run in parallel: the cap is
// package state and every parallel test must observe the production value.
func TestConvertEnforcesOutputSize(t *testing.T) {
	original := maxOutputBytes
	maxOutputBytes = 128

	defer func() { maxOutputBytes = original }()

	_, err := Convert(context.Background(), Request{HTML: `<p>hello</p>`}, nil)
	if !errors.Is(err, errOutputTooLarge) {
		t.Fatalf("error %v, want errOutputTooLarge", err)
	}
}

func TestErrorResponseCodesAreStable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want string
	}{
		{"invalid request", errInvalidRequest, "invalid_request"},
		{"unsupported mode", errUnsupportedMode, "invalid_request"},
		{"input too large", errInputTooLarge, "resource_limit"},
		{"output too large", errOutputTooLarge, "resource_limit"},
		{"viewport too large", errViewportTooLarge, "resource_limit"},
		{"timeout", context.DeadlineExceeded, "timeout"},
		{"render", errors.New("boom"), "render_error"},
	}

	for _, testCase := range cases {
		if got := errorResponse(testCase.err).Code; got != testCase.want {
			t.Fatalf("%s code %q, want %q", testCase.name, got, testCase.want)
		}
	}
}

// wasmManifest mirrors testdata/wasm/manifest.json, the contract the browser
// consumer check reads.
type wasmManifest struct {
	Schema  string `json:"schema"`
	Fixture string `json:"fixture"`
	Request struct {
		Mode   string `json:"mode"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"request"`
	Expected struct {
		Kinds     []string `json:"kinds"`
		MinOps    int      `json:"minOps"`
		MinBoxes  int      `json:"minBoxes"`
		MinFonts  int      `json:"minFonts"`
		MinImages int      `json:"minImages"`
	} `json:"expected"`
}

func TestSampleFixtureMatchesManifest(t *testing.T) {
	t.Parallel()

	manifestRaw, err := os.ReadFile("../../testdata/wasm/manifest.json")
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}

	var manifest wasmManifest
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatalf("manifest decode: %v", err)
	}

	if manifest.Schema != drawingListSchema {
		t.Fatalf("manifest schema %q, want %q", manifest.Schema, drawingListSchema)
	}

	if manifest.Request.Mode != displayMode {
		t.Fatalf("manifest mode %q, want %q", manifest.Request.Mode, displayMode)
	}

	raw, err := os.ReadFile("../../testdata/wasm/" + manifest.Fixture)
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}

	result, list := convertSource(t, string(raw), manifest.Request.Width, manifest.Request.Height)

	if result.Mode != displayMode {
		t.Fatalf("result mode %q", result.Mode)
	}

	if list.Schema != manifest.Schema {
		t.Fatalf("payload schema %q, want %q", list.Schema, manifest.Schema)
	}

	if len(list.Ops) < manifest.Expected.MinOps || len(list.Boxes) < manifest.Expected.MinBoxes ||
		len(list.Fonts) < manifest.Expected.MinFonts || len(list.Images) < manifest.Expected.MinImages {
		t.Fatalf("sample payload below manifest floors: ops=%d boxes=%d fonts=%d images=%d",
			len(list.Ops), len(list.Boxes), len(list.Fonts), len(list.Images))
	}

	kinds := map[string]bool{}
	for i := range list.Ops {
		kinds[list.Ops[i].Kind] = true
	}

	for _, kind := range manifest.Expected.Kinds {
		if !kinds[kind] {
			t.Fatalf("sample is missing expected kind %q", kind)
		}
	}

	assertReferencesResolve(t, list)
	assertOrderIsPermutation(t, list)
}
