// Command css-review-dump lays one HTML fixture out through the public
// layout.DisplayList API and joins the resulting element border boxes to the
// fixture's element tree.
//
// Join method: element identity is the key (tag, id, data-action, normalized
// descendant text). Browser and Blinkless element order can differ inside
// flex/grid containers (paint order vs document order), so matching is
// order-independent: every box consumes the first unconsumed matching element
// in document order. When several elements share a key the match quality is
// "duplicate", and the joiner compares those groups as coordinate sets.
//
// Usage:
//
//	go run ./test/chrome/harness -fixture testdata/golden/fixture-32-flex-grid-full.html \
//	    -width 1024 -height 768 -out test/chrome/harness/raw/fixture-32.blinkless.json
//
// This is a throwaway comparison harness for CSS-REVIEW-01. It is not product
// code and lives under test/chrome/harness.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chinmay-sawant/blinkless/css"
	pubhtml "github.com/chinmay-sawant/blinkless/html"
	ihtml "github.com/chinmay-sawant/blinkless/internal/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

const textCap = 200

// domElem is one element node in document order with its stable path.
type domElem struct {
	Pos   int    `json:"pos"`
	Path  string `json:"path"`
	Tag   string `json:"tag"`
	ID    string `json:"id"`
	Act   string `json:"action"`
	Class string `json:"class"`
	Text  string `json:"text"`
}

// boxOut is one Display.Box joined (or not) to a DOM element.
type boxOut struct {
	Tag     string  `json:"tag"`
	ID      string  `json:"id"`
	Action  string  `json:"action"`
	Text    string  `json:"text"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	W       float64 `json:"w"`
	H       float64 `json:"h"`
	Path    string  `json:"path"`
	Match   string  `json:"match"`
	DOMText string  `json:"dom_text,omitempty"`
}

type report struct {
	Fixture   string         `json:"fixture"`
	WidthPx   int            `json:"width_px"`
	HeightPx  int            `json:"height_px"`
	CanvasW   int            `json:"canvas_w"`
	CanvasH   int            `json:"canvas_h"`
	ElemCount int            `json:"element_count"`
	BoxCount  int            `json:"box_count"`
	Matches   map[string]int `json:"match_counts"`
	Boxes     []boxOut       `json:"boxes"`
}

func main() {
	fixture := flag.String("fixture", "", "HTML fixture to lay out (required)")
	width := flag.Int("width", 1024, "viewport width in CSS px")
	height := flag.Int("height", 768, "viewport height in CSS px")
	out := flag.String("out", "", "output JSON path (default stdout)")
	flag.Parse()

	if *fixture == "" {
		fmt.Fprintln(os.Stderr, "css-review-dump: -fixture is required")
		os.Exit(2)
	}

	source, err := os.ReadFile(*fixture)
	if err != nil {
		fmt.Fprintf(os.Stderr, "css-review-dump: read: %v\n", err)
		os.Exit(1)
	}

	domRoot, err := ihtml.ParseDocument(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "css-review-dump: parse: %v\n", err)
		os.Exit(1)
	}

	pubDoc, err := pubhtml.Parse(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "css-review-dump: public parse: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	styled, err := css.Apply(ctx, pubDoc, css.Options{
		WidthPx:  *width,
		HeightPx: *height,
		Media:    "screen",
		Extra:    nil,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "css-review-dump: css apply: %v\n", err)
		os.Exit(1)
	}

	display, err := layout.DisplayList(ctx, styled)
	if err != nil {
		fmt.Fprintf(os.Stderr, "css-review-dump: display list: %v\n", err)
		os.Exit(1)
	}

	elems := walkElements(domRoot)

	rep := report{
		Fixture:   *fixture,
		WidthPx:   *width,
		HeightPx:  *height,
		CanvasW:   display.Width,
		CanvasH:   display.Height,
		ElemCount: len(elems),
		BoxCount:  len(display.Boxes),
		Matches:   map[string]int{},
		Boxes:     make([]boxOut, 0, len(display.Boxes)),
	}

	exact := map[string][]int{}
	noText := map[string][]int{}

	for _, e := range elems {
		exact[e.key()] = append(exact[e.key()], e.Pos)
		noText[e.tagIDAct()] = append(noText[e.tagIDAct()], e.Pos)
	}

	used := make([]bool, len(elems))
	exactCursor := map[string]int{}
	noTextCursor := map[string]int{}

	consume := func(queue []int, cursor *int) int {
		for *cursor < len(queue) && used[queue[*cursor]] {
			*cursor = *cursor + 1
		}

		if *cursor >= len(queue) {
			return -1
		}

		pos := queue[*cursor]
		used[pos] = true
		*cursor = *cursor + 1

		return pos
	}

	for _, b := range display.Boxes {
		if b.Tag == "#document" || strings.HasPrefix(b.Tag, "#") {
			rep.Matches["structural-skip"]++

			continue
		}

		keyExact := elemKey(b.Tag, b.ID, b.Action, b.Text)
		keyNoText := elemTagKey(b.Tag, b.ID, b.Action)

		var pos int

		quality := "exact"

		cursor := exactCursor[keyExact]

		if len(exact[keyExact]) > 1 {
			quality = "duplicate"
		}

		pos = consume(exact[keyExact], &cursor)
		exactCursor[keyExact] = cursor

		if pos < 0 {
			quality = "text-fallback"

			cursor = noTextCursor[keyNoText]
			pos = consume(noText[keyNoText], &cursor)
			noTextCursor[keyNoText] = cursor
		}

		if pos < 0 {
			// Last resort: first unconsumed element with the same tag.
			for i, e := range elems {
				if used[i] || e.Tag != strings.ToLower(b.Tag) {
					continue
				}

				if b.ID != "" && e.ID != b.ID {
					continue
				}

				if b.Action != "" && e.Act != b.Action {
					continue
				}

				pos = i
				used[i] = true

				break
			}

			quality = "order-fallback"
		}

		if pos < 0 {
			rep.Matches["unmatched"]++

			rep.Boxes = append(rep.Boxes, boxOut{
				Tag: b.Tag, ID: b.ID, Action: b.Action, Text: truncate(b.Text, textCap),
				X: b.X, Y: b.Y, W: b.W, H: b.H, Path: "", Match: "unmatched",
			})

			continue
		}

		rep.Matches[quality]++

		rep.Boxes = append(rep.Boxes, boxOut{
			Tag: b.Tag, ID: b.ID, Action: b.Action, Text: truncate(b.Text, textCap),
			X: b.X, Y: b.Y, W: b.W, H: b.H,
			Path: elems[pos].Path, Match: quality, DOMText: truncate(elems[pos].Text, textCap),
		})
	}

	payload, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "css-review-dump: marshal: %v\n", err)
		os.Exit(1)
	}

	payload = append(payload, '\n')

	if *out == "" {
		_, _ = os.Stdout.Write(payload)

		return
	}

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "css-review-dump: mkdir: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*out, payload, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "css-review-dump: write: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s: %d elements, %d boxes, matches=%v\n", filepath.Base(*fixture), len(elems), len(display.Boxes), rep.Matches)
}

func walkElements(root *ihtml.Node) []domElem {
	var out []domElem

	var visit func(n *ihtml.Node)

	visit = func(n *ihtml.Node) {
		if n == nil {
			return
		}

		if n.Type == ihtml.ElementNode && n.Name != "#document" {
			out = append(out, domElem{
				Pos:   len(out),
				Path:  pathOf(n),
				Tag:   strings.ToLower(n.Name),
				ID:    n.Attribute("id"),
				Act:   n.Attribute("data-action"),
				Class: n.Attribute("class"),
				Text:  normalizeText(n.TextContent()),
			})
		}

		for _, c := range n.Children {
			visit(c)
		}
	}

	visit(root)

	return out
}

func pathOf(n *ihtml.Node) string {
	var parts []string

	for cur := n; cur != nil && cur.Type == ihtml.ElementNode && cur.Name != "#document"; cur = cur.Parent {
		parts = append([]string{fmt.Sprintf("%s:nth-of-type(%d)", strings.ToLower(cur.Name), nthOfType(cur))}, parts...)
	}

	return strings.Join(parts, "/")
}

func nthOfType(n *ihtml.Node) int {
	idx := 1

	if n.Parent == nil {
		return idx
	}

	for _, sib := range n.Parent.Children {
		if sib == n {
			break
		}

		if sib.Type == ihtml.ElementNode && strings.EqualFold(sib.Name, n.Name) {
			idx++
		}
	}

	return idx
}

func normalizeText(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func (e domElem) tagIDAct() string { return elemTagKey(e.Tag, e.ID, e.Act) }

func elemTagKey(tag, id, act string) string {
	return strings.ToLower(tag) + "\x00" + id + "\x00" + act
}

func (e domElem) key() string {
	return e.tagIDAct() + "\x00" + e.Text
}

func elemKey(tag, id, act, text string) string {
	return elemTagKey(tag, id, act) + "\x00" + normalizeText(text)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}

	return s[:n]
}
