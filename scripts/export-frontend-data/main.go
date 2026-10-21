// Command export-frontend-data writes the two JSON files the site reads.
//
//	go run ./scripts/export-frontend-data
//
// specimen.json is one layout.DisplayList for a fixed document.
// catalog.json is the implemented and partial rows of the CSS catalog.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"unicode/utf16"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

const (
	specimenWidthPx  = 420
	specimenHeightPx = 1
	catalogPath      = "testdata/css/catalog/properties.json"
	specimenOut      = "frontend/src/data/specimen.json"
	catalogOut       = "frontend/src/data/catalog.json"

	// siteDataPerm is world-readable because the JSON files are committed
	// site data, not secrets.
	siteDataPerm = 0o644

	// channelMax is the top of an 8-bit color channel.
	channelMax = 255
	// channelSlack is how far a painted channel may sit from the CSS color.
	channelSlack = 2
	// roundTenths keeps one decimal place in pixel coordinates.
	roundTenths = 10
	// maxNoteLen is the catalog note cutoff, in bytes.
	maxNoteLen = 180
	// asciiMax is the last ASCII code point.
	asciiMax = 0x7F
	// utf16Max is the last code point a single \u escape can carry.
	utf16Max = 0xFFFF

	// Specimen CSS colors as 8-bit channels: #9c3b2e, #1c1915, #f6f4ef.
	swatchRedR   = 156
	swatchRedG   = 59
	swatchRedB   = 46
	swatchBlackR = 28
	swatchBlackG = 25
	swatchBlackB = 21
	swatchPaperR = 246
	swatchPaperG = 244
	swatchPaperB = 239

	pinnedUpstreamMark = " Pinned upstream"
	noteEllipsis       = "..."
)

// Padding lives on the page, so the white fill covers the canvas and the
// black and red bars sit inside that margin. A child margin would collapse
// out of the background and leave a dark gap around the bars.
const specimenHTML = `<!DOCTYPE html>
<html>
<head>
<style>
  html, body { margin: 0; }
  body { padding: 20px; background: #f6f4ef; font-family: sans-serif; }
  .black {
    margin: 0;
    background: #1c1915;
    color: #f6f4ef;
    padding: 10px 14px;
    font-size: 16px;
  }
  .red {
    margin: 16px 0 0;
    background: #9c3b2e;
    color: #f6f4ef;
    padding: 10px 14px;
    font-size: 16px;
  }
</style>
</head>
<body>
  <div class="black">black</div>
  <div class="red">red</div>
</body>
</html>`

func main() {
	if err := writeSpecimen(); err != nil {
		fmt.Fprintf(os.Stderr, "specimen: %v\n", err)
		os.Exit(1)
	}

	if err := writeCatalog(); err != nil {
		fmt.Fprintf(os.Stderr, "catalog: %v\n", err)
		os.Exit(1)
	}
}

func writeSpecimen() error {
	ctx := context.Background()

	doc, err := html.Parse([]byte(specimenHTML))
	if err != nil {
		return fmt.Errorf("parse specimen: %w", err)
	}

	styled, err := css.Apply(ctx, doc, css.Options{
		WidthPx:  specimenWidthPx,
		HeightPx: specimenHeightPx,
		Media:    "screen",
		Extra:    nil,
		Focus:    "",
		Hover:    "",
		Active:   "",
	})
	if err != nil {
		return fmt.Errorf("apply specimen css: %w", err)
	}

	display, err := layout.DisplayList(ctx, styled)
	if err != nil {
		return fmt.Errorf("layout specimen: %w", err)
	}

	payload := specimenFile{
		HTML:          specimenHTML,
		WidthPx:       specimenWidthPx,
		HeightPx:      specimenHeightPx,
		CanvasWidth:   display.Width,
		CanvasHeight:  display.Height,
		PixelPerPoint: display.PixelPerPoint,
		Ops:           specimenOps(display),
	}

	return writeJSON(specimenOut, payload)
}

func specimenOps(display *layout.Display) []specimenOp {
	pxPerPt := display.PixelPerPoint
	ops := make([]specimenOp, 0, len(display.Order))

	for _, index := range display.Order {
		operation := display.Ops[index]

		name, paint := kindName(operation.Kind)
		if !paint {
			continue
		}

		red := channel(operation.R)
		green := channel(operation.G)
		blue := channel(operation.B)
		text := strings.TrimSpace(operation.Text)

		ops = append(ops, specimenOp{
			Kind:  name,
			Label: paintLabel(name, text, red, green, blue),
			X:     round1(operation.X * pxPerPt),
			Y:     round1(operation.Y * pxPerPt),
			W:     round1(operation.W * pxPerPt),
			H:     round1(operation.H * pxPerPt),
			R:     red,
			G:     green,
			B:     blue,
			Size:  round1(operation.Size * pxPerPt),
			Text:  text,
		})
	}

	return ops
}

type specimenFile struct {
	HTML          string       `json:"html"`
	WidthPx       int          `json:"widthPx"`
	HeightPx      int          `json:"heightPx"`
	CanvasWidth   int          `json:"canvasWidth"`
	CanvasHeight  int          `json:"canvasHeight"`
	PixelPerPoint float64      `json:"pixelPerPoint"`
	Ops           []specimenOp `json:"ops"`
}

type specimenOp struct {
	Kind  string  `json:"kind"`
	Label string  `json:"label"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	W     float64 `json:"w"`
	H     float64 `json:"h"`
	R     int     `json:"r"`
	G     int     `json:"g"`
	B     int     `json:"b"`
	Size  float64 `json:"size"`
	Text  string  `json:"text,omitempty"`
}

type colorSwatch struct {
	red   int
	green int
	blue  int
	label string
}

func paintLabel(kind, text string, red, green, blue int) string {
	if kind == "Text" && text != "" {
		return text
	}

	for _, swatch := range specimenSwatches() {
		if near(red, swatch.red) && near(green, swatch.green) && near(blue, swatch.blue) {
			return swatch.label
		}
	}

	return kind
}

func specimenSwatches() []colorSwatch {
	return []colorSwatch{
		{red: swatchRedR, green: swatchRedG, blue: swatchRedB, label: "red"},
		{red: swatchBlackR, green: swatchBlackG, blue: swatchBlackB, label: "black"},
		{red: swatchPaperR, green: swatchPaperG, blue: swatchPaperB, label: "outer"},
	}
}

func near(got, want int) bool {
	diff := got - want
	if diff < 0 {
		diff = -diff
	}

	return diff <= channelSlack
}

func kindName(kind layout.DisplayKind) (string, bool) {
	switch kind {
	case layout.DisplayOpFillRect:
		return "FillRect", true
	case layout.DisplayOpStrokeRect:
		return "StrokeRect", true
	case layout.DisplayOpLine:
		return "Line", true
	case layout.DisplayOpText:
		return "Text", true
	case layout.DisplayOpImage:
		return "Image", true
	case layout.DisplayOpBullet:
		return "Bullet", true
	case layout.DisplayOpGridRun:
		return "GridRun", true
	case layout.DisplayOpLinkURI, layout.DisplayOpNoop, layout.DisplayOpUnknown:
		return "", false
	default:
		return "", false
	}
}

func channel(unit float64) int {
	scaled := int(math.Round(unit * channelMax))
	if scaled < 0 {
		return 0
	}

	if scaled > channelMax {
		return channelMax
	}

	return scaled
}

func round1(value float64) float64 {
	return math.Round(value*roundTenths) / roundTenths
}

type catalogFile struct {
	Source     string         `json:"source"`
	Updated    string         `json:"updated"`
	Properties []catalogEntry `json:"properties"`
}

type catalogEntry struct {
	Name                string `json:"name"`
	Status              string `json:"status"`
	Values              string `json:"values,omitempty"`
	Note                string `json:"note,omitempty"`
	ImplementationScope string `json:"implementation_scope,omitempty"` //nolint:tagliatelle // catalog schema is snake_case
}

type rawCatalog struct {
	Updated    string `json:"updated"`
	Properties []struct {
		Name                string   `json:"name"`
		Status              string   `json:"status"`
		Values              []string `json:"values"`
		Limitations         []string `json:"limitations"`
		ImplementationScope string   `json:"implementation_scope"` //nolint:tagliatelle // catalog schema is snake_case
	} `json:"properties"`
}

func writeCatalog() error {
	rawBytes, err := os.ReadFile(catalogPath)
	if err != nil {
		return fmt.Errorf("read catalog: %w", err)
	}

	var raw rawCatalog

	if err := json.Unmarshal(rawBytes, &raw); err != nil {
		return fmt.Errorf("decode catalog: %w", err)
	}

	entries := make([]catalogEntry, 0, len(raw.Properties))

	for _, row := range raw.Properties {
		if row.Status != "implemented" && row.Status != "partial" {
			continue
		}

		entries = append(entries, catalogEntry{
			Name:                row.Name,
			Status:              row.Status,
			Values:              strings.Join(row.Values, " "),
			Note:                shortNote(row.Limitations),
			ImplementationScope: row.ImplementationScope,
		})
	}

	payload := catalogFile{
		Source:     catalogPath,
		Updated:    raw.Updated,
		Properties: entries,
	}

	return writeJSON(catalogOut, payload)
}

func shortNote(limits []string) string {
	if len(limits) == 0 {
		return ""
	}

	note := limits[0]

	if cut := strings.Index(note, pinnedUpstreamMark); cut >= 0 {
		note = strings.TrimSpace(note[:cut])
	}

	note = strings.TrimSpace(note)

	if len(note) > maxNoteLen {
		note = note[:maxNoteLen-len(noteEllipsis)] + noteEllipsis
	}

	return note
}

func writeJSON(path string, payload any) error {
	buf, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}

	buf = append(escapeNonASCII(buf), '\n')

	if err := os.WriteFile(path, buf, siteDataPerm); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

// escapeNonASCII rewrites every rune above ASCII as a \uXXXX escape. The
// committed site data is ASCII-only, and encoding/json writes UTF-8, so the
// exporter has to do this to keep regeneration diff-free.
func escapeNonASCII(buf []byte) []byte {
	var out strings.Builder

	out.Grow(len(buf))

	for _, codePoint := range string(buf) {
		switch {
		case codePoint <= asciiMax:
			out.WriteRune(codePoint)
		case codePoint > utf16Max:
			high, low := utf16.EncodeRune(codePoint)
			fmt.Fprintf(&out, `\u%04x\u%04x`, high, low)
		default:
			fmt.Fprintf(&out, `\u%04x`, codePoint)
		}
	}

	return []byte(out.String())
}
