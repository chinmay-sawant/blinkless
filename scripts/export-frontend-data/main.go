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
		return err
	}
	styled, err := css.Apply(ctx, doc, css.Options{
		WidthPx:  specimenWidthPx,
		HeightPx: specimenHeightPx,
	})
	if err != nil {
		return err
	}
	display, err := layout.DisplayList(ctx, styled)
	if err != nil {
		return err
	}

	pxPerPt := display.PixelPerPoint
	ops := make([]specimenOp, 0, len(display.Order))
	for _, index := range display.Order {
		op := display.Ops[index]
		name, paint := kindName(op.Kind)
		if !paint {
			continue
		}
		r := channel(op.R)
		g := channel(op.G)
		b := channel(op.B)
		text := strings.TrimSpace(op.Text)
		ops = append(ops, specimenOp{
			Kind:  name,
			Label: paintLabel(name, text, r, g, b),
			X:     round1(op.X * pxPerPt),
			Y:     round1(op.Y * pxPerPt),
			W:     round1(op.W * pxPerPt),
			H:     round1(op.H * pxPerPt),
			R:     r,
			G:     g,
			B:     b,
			Size:  round1(op.Size * pxPerPt),
			Text:  text,
		})
	}

	payload := specimenFile{
		HTML:          specimenHTML,
		WidthPx:       specimenWidthPx,
		HeightPx:      specimenHeightPx,
		CanvasWidth:   display.Width,
		CanvasHeight:  display.Height,
		PixelPerPoint: pxPerPt,
		Ops:           ops,
	}
	return writeJSON(specimenOut, payload)
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

func paintLabel(kind, text string, r, g, b int) string {
	if kind == "Text" && text != "" {
		return text
	}
	switch {
	case near(r, 156) && near(g, 59) && near(b, 46):
		return "red"
	case near(r, 28) && near(g, 25) && near(b, 21):
		return "black"
	case near(r, 246) && near(g, 244) && near(b, 239):
		return "outer"
	default:
		return kind
	}
}

func near(got, want int) bool {
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	return diff <= 2
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
	n := int(math.Round(unit * 255))
	if n < 0 {
		return 0
	}
	if n > 255 {
		return 255
	}
	return n
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

type catalogFile struct {
	Source     string         `json:"source"`
	Updated    string         `json:"updated"`
	Properties []catalogEntry `json:"properties"`
}

type catalogEntry struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Values string `json:"values,omitempty"`
	Note   string `json:"note,omitempty"`
}

type rawCatalog struct {
	Updated    string `json:"updated"`
	Properties []struct {
		Name        string   `json:"name"`
		Status      string   `json:"status"`
		Values      []string `json:"values"`
		Limitations []string `json:"limitations"`
	} `json:"properties"`
}

func writeCatalog() error {
	rawBytes, err := os.ReadFile(catalogPath)
	if err != nil {
		return err
	}
	var raw rawCatalog
	if err := json.Unmarshal(rawBytes, &raw); err != nil {
		return err
	}

	entries := make([]catalogEntry, 0, len(raw.Properties))
	for _, row := range raw.Properties {
		if row.Status != "implemented" && row.Status != "partial" {
			continue
		}
		entries = append(entries, catalogEntry{
			Name:   row.Name,
			Status: row.Status,
			Values: strings.Join(row.Values, " "),
			Note:   shortNote(row.Limitations),
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
	if i := strings.Index(note, " Pinned upstream"); i >= 0 {
		note = strings.TrimSpace(note[:i])
	}
	note = strings.TrimSpace(note)
	if len(note) > 180 {
		note = note[:177] + "..."
	}
	return note
}

func writeJSON(path string, payload any) error {
	buf, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	buf = append(buf, '\n')
	return os.WriteFile(path, buf, 0o644)
}
