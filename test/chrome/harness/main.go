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

const (
	defaultViewportWidth  = 1024
	defaultViewportHeight = 768
	outputDirPerm         = 0o755
	outputFilePerm        = 0o600
	docNodeName           = "#document"
	exitUsage             = 2
	exitFailure           = 1
)

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
	DOMText string  `json:"domText,omitempty"`
}

type report struct {
	Fixture   string         `json:"fixture"`
	WidthPx   int            `json:"widthPx"`
	HeightPx  int            `json:"heightPx"`
	CanvasW   int            `json:"canvasW"`
	CanvasH   int            `json:"canvasH"`
	ElemCount int            `json:"elementCount"`
	BoxCount  int            `json:"boxCount"`
	Matches   map[string]int `json:"matchCounts"`
	Boxes     []boxOut       `json:"boxes"`
}

// parseFlags reads the CLI flags for one dump run.
func parseFlags() (string, int, int, string) {
	fixtureFlag := flag.String("fixture", "", "HTML fixture to lay out (required)")
	widthFlag := flag.Int("width", defaultViewportWidth, "viewport width in CSS px")
	heightFlag := flag.Int("height", defaultViewportHeight, "viewport height in CSS px")
	outFlag := flag.String("out", "", "output JSON path (default stdout)")
	flag.Parse()

	return *fixtureFlag, *widthFlag, *heightFlag, *outFlag
}

func main() {
	fixture, width, height, out := parseFlags()

	if fixture == "" {
		fmt.Fprintln(os.Stderr, "css-review-dump: -fixture is required")
		os.Exit(exitUsage)
	}

	source, err := os.ReadFile(fixture)
	if err != nil {
		fmt.Fprintf(os.Stderr, "css-review-dump: read: %v\n", err)
		os.Exit(exitFailure)
	}

	domRoot, err := ihtml.ParseDocument(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "css-review-dump: parse: %v\n", err)
		os.Exit(exitFailure)
	}

	pubDoc, err := pubhtml.Parse(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "css-review-dump: public parse: %v\n", err)
		os.Exit(exitFailure)
	}

	ctx := context.Background()

	styled, err := css.Apply(ctx, pubDoc, css.Options{
		WidthPx:  width,
		HeightPx: height,
		Media:    "screen",
		Extra:    nil,
		Focus:    "",
		Hover:    "",
		Active:   "",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "css-review-dump: css apply: %v\n", err)
		os.Exit(exitFailure)
	}

	display, err := layout.DisplayList(ctx, styled)
	if err != nil {
		fmt.Fprintf(os.Stderr, "css-review-dump: display list: %v\n", err)
		os.Exit(exitFailure)
	}

	elems := walkElements(domRoot)

	rep := newReport(fixture, width, height, display.Width, display.Height, len(elems), len(display.Boxes))

	joinBoxes(&rep, elems, display.Boxes)

	if err := emitReport(&rep, out, fixture, len(elems), len(display.Boxes)); err != nil {
		fmt.Fprintf(os.Stderr, "css-review-dump: %v\n", err)
		os.Exit(exitFailure)
	}
}

// emitReport marshals the join report and writes it to outPath, or stdout
// when outPath is empty. It prints a one-line summary when writing to a file.
func emitReport(rep *report, outPath, fixture string, elemCount, boxCount int) error {
	payload, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	payload = append(payload, '\n')

	if outPath == "" {
		_, _ = os.Stdout.Write(payload)

		return nil
	}

	if err := os.MkdirAll(filepath.Dir(outPath), outputDirPerm); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	if err := os.WriteFile(outPath, payload, outputFilePerm); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	fmt.Fprintln(os.Stdout,
		filepath.Base(fixture)+":", elemCount, "elements,",
		boxCount, "boxes, matches:", rep.Matches)

	return nil
}

// newReport builds an empty join report for one fixture run.
func newReport(fixture string, width, height, canvasW, canvasH, elemCount, boxCount int) report {
	return report{
		Fixture:   fixture,
		WidthPx:   width,
		HeightPx:  height,
		CanvasW:   canvasW,
		CanvasH:   canvasH,
		ElemCount: elemCount,
		BoxCount:  boxCount,
		Matches:   map[string]int{},
		Boxes:     make([]boxOut, 0, boxCount),
	}
}

// joinBoxes matches every display box to its source element and appends one
// boxOut per box to rep. Boxes with no element match are kept with an empty
// path and Match "unmatched" so the join output stays complete.
func joinBoxes(rep *report, elems []domElem, boxes []layout.Box) {
	exact, noText, used, exactCursor, noTextCursor, consume := newMatcher(elems)

	for _, box := range boxes {
		if box.Tag == docNodeName || strings.HasPrefix(box.Tag, "#") {
			rep.Matches["structural-skip"]++

			continue
		}

		keyExact := elemKey(box.Tag, box.ID, box.Action, box.Text)
		keyNoText := elemTagKey(box.Tag, box.ID, box.Action)

		pos, quality := resolveBoxPos(
			elems, used, exact, noText, exactCursor, noTextCursor,
			consume, keyExact, keyNoText, box,
		)

		if pos < 0 {
			rep.Matches["unmatched"]++

			rep.Boxes = append(rep.Boxes, boxOut{
				Tag: box.Tag, ID: box.ID, Action: box.Action, Text: truncate(box.Text, textCap),
				X: box.X, Y: box.Y, W: box.W, H: box.H, Path: "", Match: "unmatched", DOMText: "",
			})

			continue
		}

		rep.Matches[quality]++

		rep.Boxes = append(rep.Boxes, boxOut{
			Tag: box.Tag, ID: box.ID, Action: box.Action, Text: truncate(box.Text, textCap),
			X: box.X, Y: box.Y, W: box.W, H: box.H,
			Path: elems[pos].Path, Match: quality, DOMText: truncate(elems[pos].Text, textCap),
		})
	}
}

// newMatcher builds the order-independent match indexes over the document
// elements: exact and text-insensitive queues, consumption flags, per-key
// cursors, and the queue consumer used by the join.
func newMatcher(elems []domElem) (
	map[string][]int, map[string][]int, []bool, map[string]int, map[string]int, func([]int, *int) int,
) {
	exact := map[string][]int{}
	noText := map[string][]int{}

	for _, elem := range elems {
		exact[elem.key()] = append(exact[elem.key()], elem.Pos)
		noText[elem.tagIDAct()] = append(noText[elem.tagIDAct()], elem.Pos)
	}

	used := make([]bool, len(elems))
	exactCursor := map[string]int{}
	noTextCursor := map[string]int{}

	consume := func(queue []int, cursor *int) int {
		for *cursor < len(queue) && used[queue[*cursor]] {
			*cursor++
		}

		if *cursor >= len(queue) {
			return -1
		}

		pos := queue[*cursor]
		used[pos] = true
		*cursor++

		return pos
	}

	return exact, noText, used, exactCursor, noTextCursor, consume
}

// resolveBoxPos matches one display box to its source element through the
// exact, text-fallback, and tag-scan strategies in order. It returns the
// element position and the match quality, or -1 with "unmatched".
func resolveBoxPos(
	elems []domElem,
	used []bool,
	exact, noText map[string][]int,
	exactCursor, noTextCursor map[string]int,
	consume func([]int, *int) int,
	keyExact, keyNoText string,
	box layout.Box,
) (int, string) {
	quality := "exact"

	cursor := exactCursor[keyExact]

	if len(exact[keyExact]) > 1 {
		quality = "duplicate"
	}

	pos := consume(exact[keyExact], &cursor)
	exactCursor[keyExact] = cursor

	if pos < 0 {
		quality = "text-fallback"

		cursor = noTextCursor[keyNoText]
		pos = consume(noText[keyNoText], &cursor)
		noTextCursor[keyNoText] = cursor
	}

	if pos < 0 {
		quality = "order-fallback"
		pos = scanByTag(elems, used, box)
	}

	if pos < 0 {
		return -1, "unmatched"
	}

	return pos, quality
}

// scanByTag is the last-resort match: the first unconsumed element with the
// same tag, id, and action. Returns -1 when nothing matches.
func scanByTag(elems []domElem, used []bool, box layout.Box) int {
	for idx, elem := range elems {
		if used[idx] || elem.Tag != strings.ToLower(box.Tag) {
			continue
		}

		if box.ID != "" && elem.ID != box.ID {
			continue
		}

		if box.Action != "" && elem.Act != box.Action {
			continue
		}

		used[idx] = true

		return idx
	}

	return -1
}

func walkElements(root *ihtml.Node) []domElem {
	var out []domElem

	var visit func(n *ihtml.Node)

	visit = func(node *ihtml.Node) {
		if node == nil {
			return
		}

		if node.Type == ihtml.ElementNode && node.Name != docNodeName {
			out = append(out, domElem{
				Pos:   len(out),
				Path:  pathOf(node),
				Tag:   strings.ToLower(node.Name),
				ID:    node.Attribute("id"),
				Act:   node.Attribute("data-action"),
				Class: node.Attribute("class"),
				Text:  normalizeText(node.TextContent()),
			})
		}

		for _, c := range node.Children {
			visit(c)
		}
	}

	visit(root)

	return out
}

func pathOf(node *ihtml.Node) string {
	var parts []string

	for cur := node; cur != nil && cur.Type == ihtml.ElementNode && cur.Name != docNodeName; cur = cur.Parent {
		parts = append([]string{fmt.Sprintf("%s:nth-of-type(%d)", strings.ToLower(cur.Name), nthOfType(cur))}, parts...)
	}

	return strings.Join(parts, "/")
}

func nthOfType(node *ihtml.Node) int {
	idx := 1

	if node.Parent == nil {
		return idx
	}

	for _, sib := range node.Parent.Children {
		if sib == node {
			break
		}

		if sib.Type == ihtml.ElementNode && strings.EqualFold(sib.Name, node.Name) {
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
