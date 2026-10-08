package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"

	"github.com/chinmay-sawant/blinkless/internal/fonts"
	"github.com/chinmay-sawant/blinkless/layout"
)

// The browser adapter answers with one versioned JSON drawing list. The schema
// name and number are part of every payload so a consumer can pin the contract
// it was built against. documentation/library-api.md is the written schema;
// the constants and structs in this file are its executable form.
const (
	// drawingListSchema names the contract and carries its major version.
	drawingListSchema = "blinkless.drawinglist/1"
	// drawingListSchemaVersion repeats the major version as a number.
	drawingListSchemaVersion = 1
	// drawingListUnits names the unit of every operation coordinate.
	drawingListUnits = "points"
)

// Operation kinds in the JSON document. These are stable strings, independent
// of the numeric layout.OpKind values (which stay available in kindValue for
// cross-checking a native layout.Display).
const (
	kindNoop       = "noop"
	kindUnknown    = "unknown"
	kindFillRect   = "fillRect"
	kindStrokeRect = "strokeRect"
	kindLine       = "line"
	kindText       = "text"
	kindImage      = "image"
	kindLinkURI    = "linkURI"
	kindBullet     = "bullet"
	kindGridRun    = "gridRun"
	kindGroupBegin = "groupBegin"
	kindGroupEnd   = "groupEnd"
)

// drawingListJSON is one versioned result. Ops stay in the source order of
// layout.Display.Ops; Order lists the indexes to replay, in paint order.
type drawingListJSON struct {
	Schema  string `json:"schema"`
	Version int    `json:"version"`
	// Units names the unit of operation geometry: "points".
	Units string `json:"units"`
	// Width and Height are the canvas size in CSS pixels.
	Width  int `json:"width"`
	Height int `json:"height"`
	// PxPerPt multiplies an operation coordinate to reach CSS pixels. It
	// mirrors layout.Display.PixelPerPoint (96/72). PtPerPx is the reciprocal
	// and mirrors layout.Display.PointsPerPixel (72/96).
	PxPerPt float64 `json:"pxPerPt"`
	PtPerPx float64 `json:"ptPerPx"`
	// Ops is the display list in source order. An entry is one
	// layout.DisplayOp; see the kind field for its payload.
	Ops []drawOpJSON `json:"ops"`
	// Order indexes Ops into paint order: iterate Order and replay
	// Ops[Order[i]]. It mirrors layout.Display.Order exactly.
	Order []int `json:"order"`
	// Boxes are element border boxes in CSS pixels, in document order.
	Boxes []drawBoxJSON `json:"boxes"`
	// Groups describes the blend and isolation groups Ops reference.
	Groups []drawGroupJSON `json:"groups"`
	// Fonts is the face table text and bullet entries reference. Bytes are the
	// raw SFNT payload, base64 encoded.
	Fonts []drawFontJSON `json:"fonts"`
	// Images is the payload table image ops reference. Bytes are the encoded
	// image payload (PNG, JPEG, or the source bytes), base64 encoded.
	Images []drawImageJSON `json:"images"`
}

// drawOpJSON is one serialized display-list operation. Coordinates are canvas
// points, y down. For text and bullet entries y is the baseline; h stays the
// line-box height and inkDescent is the glyph descent below the baseline.
type drawOpJSON struct {
	ID        uint64  `json:"id"`
	Kind      string  `json:"kind"`
	KindValue uint8   `json:"kindValue"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	W         float64 `json:"w"`
	H         float64 `json:"h"`
	R         float64 `json:"r"`
	G         float64 `json:"g"`
	B         float64 `json:"b"`
	// Alpha is the operation's own color alpha.
	Alpha float64 `json:"alpha"`
	// Opacity is the effective opacity a replay should paint with. It folds
	// the element opacity with Alpha, the same number layout.DisplayOp.Opacity
	// returns. Use Opacity, not Alpha * element opacity.
	Opacity       float64 `json:"opacity"`
	Width         float64 `json:"width"`
	Size          float64 `json:"size,omitempty"`
	LetterSpacing float64 `json:"letterSpacing,omitempty"`
	InkDescent    float64 `json:"inkDescent,omitempty"`
	RotateDeg     float64 `json:"rotateDeg,omitempty"`
	// Text is the raw shaped-run text. Apply textTransform before drawing.
	Text string `json:"text,omitempty"`
	// FontID references Fonts. Set on text and bullet entries that carry a
	// face.
	FontID string `json:"font,omitempty"`
	// ImageID references Images. Alt is the image's alt text.
	ImageID string `json:"image,omitempty"`
	Alt     string `json:"alt,omitempty"`
	// URI is the link target of a linkURI entry.
	URI string `json:"uri,omitempty"`
	// BlendMode is the CSS mix-blend-mode value; empty means normal.
	BlendMode string `json:"blendMode,omitempty"`
	// GroupID references Groups; absent outside any group.
	GroupID *int `json:"group,omitempty"`
	// GroupMark is "begin" or "end" on a group boundary entry. Boundary
	// entries paint nothing.
	GroupMark string `json:"groupMark,omitempty"`
	// Transform is the baked affine transform when xformSet is true. Bounds
	// x/y/w/h are already in the canvas space; the transform applies at paint.
	Transform *drawMatrixJSON `json:"transform,omitempty"`
	// Radii describes rounded corners for fillRect and strokeRect entries.
	Radii *drawRadiiJSON `json:"radii,omitempty"`
	// Segments are the ordered line segments of a gridRun entry.
	Segments      []drawSegJSON `json:"segments,omitempty"`
	TextTransform string        `json:"textTransform,omitempty"`
	TextLanguage  string        `json:"textLanguage,omitempty"`
	TextAutospace string        `json:"textAutospace,omitempty"`
	FontFeatures  string        `json:"fontFeatures,omitempty"`
	// StrokeMask selects sides for a rounded strokeRect; zero means all sides.
	StrokeMask uint8 `json:"strokeMask,omitempty"`
	// LineInset selects the inward side for a mixed-width border line.
	LineInset uint8 `json:"lineInset,omitempty"`
	// Bold is the requested weight. FakeBold means synthesize it by double
	// striking; FakeOblique means synthesize italic by skewing.
	Bold        bool `json:"bold,omitempty"`
	FakeBold    bool `json:"fakeBold,omitempty"`
	FakeOblique bool `json:"fakeOblique,omitempty"`
	NoFakeBold  bool `json:"noFakeBold,omitempty"`
	// IsJPEG marks an image payload that is JPEG data.
	IsJPEG       bool `json:"isJPEG,omitempty"`
	IsBackground bool `json:"isBackground,omitempty"`
	Fixed        bool `json:"fixed,omitempty"`
	Pinned       bool `json:"pinned,omitempty"`
	Positioned   bool `json:"positioned,omitempty"`
	StickyID     int  `json:"stickyID,omitempty"`
	ZIndex       int  `json:"zIndex,omitempty"`
	ZIndexSet    bool `json:"zIndexSet,omitempty"`
	Outline      bool `json:"outline,omitempty"`
}

// drawMatrixJSON is one 2D affine transform: x' = A*x + C*y + E,
// y' = B*x + D*y + F.
type drawMatrixJSON struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
	C float64 `json:"c"`
	D float64 `json:"d"`
	E float64 `json:"e"`
	F float64 `json:"f"`
}

// drawRadiiJSON is the corner geometry of a rounded rectangle. Radius is the
// uniform value; the per-corner fields override it.
type drawRadiiJSON struct {
	Radius       float64 `json:"radius"`
	RadiusY      float64 `json:"radiusY"`
	TopLeft      float64 `json:"topLeft"`
	TopRight     float64 `json:"topRight"`
	BottomRight  float64 `json:"bottomRight"`
	BottomLeft   float64 `json:"bottomLeft"`
	TopLeftY     float64 `json:"topLeftY"`
	TopRightY    float64 `json:"topRightY"`
	BottomRightY float64 `json:"bottomRightY"`
	BottomLeftY  float64 `json:"bottomLeftY"`
}

// drawSegJSON is one line segment of a gridRun entry, in canvas points.
type drawSegJSON struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	W         float64 `json:"w"`
	H         float64 `json:"h"`
	Width     float64 `json:"width"`
	R         float64 `json:"r"`
	G         float64 `json:"g"`
	B         float64 `json:"b"`
	LineInset uint8   `json:"lineInset,omitempty"`
}

// drawBoxJSON is one element border box in CSS pixels.
type drawBoxJSON struct {
	ID     string  `json:"id"`
	Tag    string  `json:"tag"`
	Action string  `json:"action,omitempty"`
	Text   string  `json:"text,omitempty"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	W      float64 `json:"w"`
	H      float64 `json:"h"`
}

// drawGroupJSON is one blend or isolation group. Parent is absent at the page
// root.
type drawGroupJSON struct {
	ID      int    `json:"id"`
	Mode    string `json:"mode,omitempty"`
	Isolate bool   `json:"isolate,omitempty"`
	Parent  *int   `json:"parent,omitempty"`
}

// drawFontJSON is one font face referenced by text and bullet entries.
type drawFontJSON struct {
	// ID is content-derived ("f-" plus a truncated SHA-256 of Bytes), so the
	// same face keeps the same ID across runs and runtimes.
	ID             string `json:"id"`
	PostScriptName string `json:"postScriptName,omitempty"`
	UnitsPerEm     int    `json:"unitsPerEm"`
	Ascent         int    `json:"ascent"`
	Descent        int    `json:"descent"`
	CapHeight      int    `json:"capHeight,omitempty"`
	XHeight        int    `json:"xHeight,omitempty"`
	ByteLength     int    `json:"byteLength"`
	Bytes          string `json:"bytes"`
}

// drawImageJSON is one encoded image payload referenced by image entries.
type drawImageJSON struct {
	// ID is content-derived ("i-" plus a truncated SHA-256 of Bytes).
	ID          string `json:"id"`
	Format      string `json:"format"`
	PixelWidth  int    `json:"pixelWidth"`
	PixelHeight int    `json:"pixelHeight"`
	ByteLength  int    `json:"byteLength"`
	Bytes       string `json:"bytes"`
}

// errNilDisplay is returned when serialization is asked for no display list.
var errNilDisplay = errors.New("nil drawing list")

// serializeDisplay marshals display into the versioned drawing-list document
// and enforces the browser output cap after serialization.
func serializeDisplay(display *layout.Display) ([]byte, error) {
	if display == nil {
		return nil, errNilDisplay
	}

	payload, err := json.Marshal(newDrawingList(display))
	if err != nil {
		return nil, err
	}

	if maxOutputBytes >= 0 && len(payload) > maxOutputBytes {
		return nil, errOutputTooLarge
	}

	return payload, nil
}

// newDrawingList projects the public layout.Display to its JSON document.
// Every value crosses an exported field or an accessor; no internal pointer is
// serialized.
func newDrawingList(display *layout.Display) drawingListJSON {
	resources := newDrawingResources()

	ops := make([]drawOpJSON, len(display.Ops))
	for i := range display.Ops {
		ops[i] = resources.convertOp(&display.Ops[i])
	}

	order := make([]int, len(display.Order))
	copy(order, display.Order)

	boxes := make([]drawBoxJSON, len(display.Boxes))
	for i := range display.Boxes {
		box := &display.Boxes[i]
		boxes[i] = drawBoxJSON{
			ID:     box.ID,
			Tag:    box.Tag,
			Action: box.Action,
			Text:   box.Text,
			X:      box.X,
			Y:      box.Y,
			W:      box.W,
			H:      box.H,
		}
	}

	return drawingListJSON{
		Schema:  drawingListSchema,
		Version: drawingListSchemaVersion,
		Units:   drawingListUnits,
		Width:   display.Width,
		Height:  display.Height,
		PxPerPt: display.PixelPerPoint,
		PtPerPx: display.PointsPerPixel,
		Ops:     ops,
		Order:   order,
		Boxes:   boxes,
		Groups:  resources.sortedGroups(),
		Fonts:   resources.sortedFonts(),
		Images:  resources.sortedImages(),
	}
}

// drawingResources collects the per-result font, image, and group tables
// while operations are converted. Fonts and images get content-derived IDs;
// groups keep their native creation-order ID.
type drawingResources struct {
	fontByFace map[*fonts.Font]string
	fonts      []drawFontJSON
	imageSeen  map[string]struct{}
	images     []drawImageJSON
	groups     map[int]drawGroupJSON
}

func newDrawingResources() *drawingResources {
	return &drawingResources{
		fontByFace: map[*fonts.Font]string{},
		imageSeen:  map[string]struct{}{},
		groups:     map[int]drawGroupJSON{},
	}
}

func (res *drawingResources) convertOp(op *layout.DisplayOp) drawOpJSON {
	kind, kindValue := drawKind(op)

	out := drawOpJSON{
		ID:            op.ID,
		Kind:          kind,
		KindValue:     kindValue,
		X:             op.X,
		Y:             op.Y,
		W:             op.W,
		H:             op.H,
		R:             op.R,
		G:             op.G,
		B:             op.B,
		Alpha:         op.Alpha,
		Opacity:       op.Opacity(),
		Width:         op.Width,
		Size:          op.Size,
		LetterSpacing: op.LetterSpacing,
		InkDescent:    op.InkDescent,
		RotateDeg:     float64(op.RotateDeg),
		Text:          op.Text,
		BlendMode:     op.BlendModeName(),
		TextTransform: op.TextTransformValue(),
		TextLanguage:  op.TextLanguage(),
		TextAutospace: op.TextAutospace(),
		FontFeatures:  op.FontFeatures(),
		StrokeMask:    op.StrokeMask,
		LineInset:     op.LineInset,
		Bold:          op.Bold,
		FakeBold:      layout.DisplayFakeBold(op),
		FakeOblique:   op.FakeOblique,
		NoFakeBold:    op.NoFakeBoldValue(),
		IsJPEG:        op.IsJPEG,
		IsBackground:  op.IsBackground,
		Fixed:         op.Fixed,
		Pinned:        op.Pinned,
		Positioned:    op.Positioned,
		StickyID:      op.StickyID,
		ZIndex:        op.ZIndex,
		ZIndexSet:     op.ZIndexSet,
		Outline:       op.Outline(),
	}

	if op.Font != nil {
		out.FontID = res.addFont(op.Font)
	}

	if data, width, height := op.ImageBytes(); data != nil {
		out.ImageID = res.addImage(data, width, height)
		out.Alt = op.ImageAlt()
	}

	out.URI = op.LinkURI()

	if op.XformSet {
		matrix := op.Transform()
		out.Transform = &drawMatrixJSON{
			A: matrix.A,
			B: matrix.B,
			C: matrix.C,
			D: matrix.D,
			E: matrix.E,
			F: matrix.F,
		}
	}

	if op.IsGroupBegin() {
		out.GroupMark = "begin"
	} else if op.IsGroupEnd() {
		out.GroupMark = "end"
	}

	if group := op.Group(); group != nil {
		res.addGroup(group)
		id := group.ID
		out.GroupID = &id
	}

	if radii := drawRadii(op); radii != nil {
		out.Radii = radii
	}

	if op.Grid != nil {
		out.Segments = make([]drawSegJSON, len(op.Grid.Segs))
		for i := range op.Grid.Segs {
			seg := &op.Grid.Segs[i]
			out.Segments[i] = drawSegJSON{
				X:         seg.X,
				Y:         seg.Y,
				W:         seg.W,
				H:         seg.H,
				Width:     seg.Width,
				R:         seg.R,
				G:         seg.G,
				B:         seg.B,
				LineInset: seg.LineInset,
			}
		}
	}

	return out
}

// drawKind maps a native operation to its stable JSON kind. A group boundary
// entry is recognized by its group mark, not its kind: a deactivating pass may
// rewrite Kind to noop while the mark stays, and the boundary semantics must
// survive that.
func drawKind(op *layout.DisplayOp) (string, uint8) {
	value := uint8(op.Kind)

	if op.IsGroupBegin() {
		return kindGroupBegin, value
	}

	if op.IsGroupEnd() {
		return kindGroupEnd, value
	}

	switch op.Kind {
	case layout.DisplayOpNoop:
		return kindNoop, value
	case layout.DisplayOpFillRect:
		return kindFillRect, value
	case layout.DisplayOpStrokeRect:
		return kindStrokeRect, value
	case layout.DisplayOpLine:
		return kindLine, value
	case layout.DisplayOpText:
		return kindText, value
	case layout.DisplayOpImage:
		return kindImage, value
	case layout.DisplayOpLinkURI:
		return kindLinkURI, value
	case layout.DisplayOpBullet:
		return kindBullet, value
	case layout.DisplayOpGridRun:
		return kindGridRun, value
	default:
		return kindUnknown, value
	}
}

// drawRadii returns the rounded-corner geometry, or nil for a square
// rectangle.
func drawRadii(op *layout.DisplayOp) *drawRadiiJSON {
	if op.Radius == 0 && op.RadiusY == 0 &&
		op.RadiusTopLeft == 0 && op.RadiusTopRight == 0 &&
		op.RadiusBottomRight == 0 && op.RadiusBottomLeft == 0 &&
		op.RadiusTopLeftY == 0 && op.RadiusTopRightY == 0 &&
		op.RadiusBottomRightY == 0 && op.RadiusBottomLeftY == 0 {
		return nil
	}

	return &drawRadiiJSON{
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
}

// addFont records one face and returns its content-derived ID. Faces are
// deduplicated by pointer, so several operations that use one face share one
// table entry.
func (res *drawingResources) addFont(face *fonts.Font) string {
	if id, ok := res.fontByFace[face]; ok {
		return id
	}

	data := face.Bytes()
	id := resourceID("f-", data)

	res.fontByFace[face] = id
	res.fonts = append(res.fonts, drawFontJSON{
		ID:             id,
		PostScriptName: face.PostScriptName,
		UnitsPerEm:     int(face.UnitsPerEm()),
		Ascent:         int(face.Ascent()),
		Descent:        int(face.Descent()),
		CapHeight:      int(face.CapHeight()),
		XHeight:        int(face.XHeight()),
		ByteLength:     len(data),
		Bytes:          base64.StdEncoding.EncodeToString(data),
	})

	return id
}

// addImage records one encoded payload and returns its content-derived ID.
// Identical payloads share one table entry.
func (res *drawingResources) addImage(data []byte, pixelWidth, pixelHeight int) string {
	id := resourceID("i-", data)
	if _, ok := res.imageSeen[id]; ok {
		return id
	}

	res.imageSeen[id] = struct{}{}
	res.images = append(res.images, drawImageJSON{
		ID:          id,
		Format:      imageFormat(data),
		PixelWidth:  pixelWidth,
		PixelHeight: pixelHeight,
		ByteLength:  len(data),
		Bytes:       base64.StdEncoding.EncodeToString(data),
	})

	return id
}

// addGroup records a group and every ancestor, keyed by the native group ID.
func (res *drawingResources) addGroup(group *layout.DisplayGroup) {
	if group == nil {
		return
	}

	if _, ok := res.groups[group.ID]; ok {
		return
	}

	entry := drawGroupJSON{
		ID:      group.ID,
		Mode:    group.Mode,
		Isolate: group.Isolate,
	}

	if group.Parent != nil {
		res.addGroup(group.Parent)
		parent := group.Parent.ID
		entry.Parent = &parent
	}

	res.groups[group.ID] = entry
}

func (res *drawingResources) sortedFonts() []drawFontJSON {
	fonts := make([]drawFontJSON, len(res.fonts))
	copy(fonts, res.fonts)
	sort.Slice(fonts, func(i, j int) bool { return fonts[i].ID < fonts[j].ID })

	return fonts
}

func (res *drawingResources) sortedImages() []drawImageJSON {
	images := make([]drawImageJSON, len(res.images))
	copy(images, res.images)
	sort.Slice(images, func(i, j int) bool { return images[i].ID < images[j].ID })

	return images
}

func (res *drawingResources) sortedGroups() []drawGroupJSON {
	groups := make([]drawGroupJSON, 0, len(res.groups))
	for id := range res.groups {
		groups = append(groups, res.groups[id])
	}

	sort.Slice(groups, func(i, j int) bool { return groups[i].ID < groups[j].ID })

	return groups
}

// resourceID builds a stable per-result resource ID from the payload bytes:
// prefix plus the first 8 bytes of SHA-256, hex encoded (64 bits). The same
// bytes produce the same ID in every run and every Go runtime.
func resourceID(prefix string, data []byte) string {
	sum := sha256.Sum256(data)

	return prefix + hex.EncodeToString(sum[:8])
}

// imageFormat names the payload encoding. The display list keeps source bytes;
// layout rewrites SVG sources to PNG before they reach an operation.
func imageFormat(data []byte) string {
	switch {
	case len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n":
		return "png"
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return "jpeg"
	default:
		return "binary"
	}
}
