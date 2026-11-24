package svg

import (
	"bytes"
	"math"
	"strconv"
	"strings"

	"github.com/tdewolff/canvas"
)

// emulateClipScan collects <clipPath id="..."> definitions into settled
// *canvas.Path masks so shapes referencing clip-path="url(#id)" can be
// geometrically intersected before canvas.ParseSVG (which ignores <clipPath>).
type emulateClipScan struct {
	open     bool
	id       string
	clipRule canvas.FillRule
	combined *canvas.Path
}

func emulateMayCollectClips(data []byte) bool {
	return bytesContainsFold(data, []byte("clippath"))
}

func emulateCollectClips(data []byte) map[string]*canvas.Path {
	if !emulateMayCollectClips(data) {
		return nil
	}

	clips := map[string]*canvas.Path{}

	var scan emulateClipScan

	pos, tags := 0, 0

	for pos < len(data) && tags < maxEmulateTags {
		tags++

		rel := bytes.IndexByte(data[pos:], '<')
		if rel < 0 {
			break
		}

		start := pos + rel

		end, ok := emulateTagEnd(data, start)
		if !ok {
			break
		}

		pos = start + 1

		if bytes.HasPrefix(data[start:], []byte("</")) {
			if scan.open && strings.EqualFold(strings.TrimSpace(string(data[start+2:end])), "clippath") {
				if scan.id != "" && scan.combined != nil && !scan.combined.Empty() {
					clips[scan.id] = scan.combined
				}

				scan = emulateClipScan{}
			}

			continue
		}

		name, attrs, selfClose := emulateParseTag(data, start, end)
		scan.element(name, attrs, selfClose)
	}

	return clips
}

func (s *emulateClipScan) element(name string, attrs []emulateAttr, selfClose bool) {
	switch {
	case name == "clippath" && !selfClose:
		index := emulateAttrMap(attrs)
		s.open = true
		s.id = index["id"]
		s.clipRule = emulateClipRuleOf(index, canvas.NonZero)
		s.combined = nil
	case s.open && emulateIsShape(name):
		index := emulateAttrMap(attrs)
		rule := emulateClipRuleOf(index, s.clipRule)

		shape := emulateShapeToPath(name, index)
		if shape == nil || shape.Empty() {
			return
		}

		settled := emulateSafeSettle(shape, rule)
		if settled == nil || settled.Empty() {
			return
		}

		if s.combined == nil {
			s.combined = settled
		} else if merged := emulateSafeOr(s.combined, settled); merged != nil && !merged.Empty() {
			s.combined = merged
		}
	}
}

func emulateClipRuleOf(index map[string]string, fallback canvas.FillRule) canvas.FillRule {
	rule := strings.TrimSpace(index["clip-rule"])
	if rule == "" {
		rule = strings.TrimSpace(styleDeclarations(index["style"])["clip-rule"])
	}

	if strings.EqualFold(rule, "evenodd") {
		return canvas.EvenOdd
	}

	if strings.EqualFold(rule, "nonzero") {
		return canvas.NonZero
	}

	return fallback
}

func emulateClipPathRef(index map[string]string) (string, bool) {
	ref := strings.TrimSpace(index["clip-path"])
	if ref == "" {
		ref = strings.TrimSpace(styleDeclarations(index["style"])["clip-path"])
	}

	if !strings.HasPrefix(ref, "url(#") || !strings.HasSuffix(ref, ")") {
		return "", false
	}

	id := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(ref, "url(#"), ")"))
	if id == "" {
		return "", false
	}

	return id, true
}

// emulateClippedShape intersects a leaf shape with its referenced <clipPath>
// and emits a settled <path d="..."/> with clip-path removed.
func emulateClippedShape(
	name string, attrs []emulateAttr, index map[string]string, clips map[string]*canvas.Path,
) (string, bool) {
	if len(clips) == 0 {
		return "", false
	}

	id, ok := emulateClipPathRef(index)
	if !ok {
		return "", false
	}

	clipPath, found := clips[id]
	if !found || clipPath == nil || clipPath.Empty() {
		return "", false
	}

	shape := emulateShapeToPath(name, index)
	if shape == nil || shape.Empty() {
		return "", false
	}

	fillRule := canvas.NonZero
	if emulateEvenOddRequested(index) {
		fillRule = canvas.EvenOdd
	}

	settledShape := emulateSafeSettle(shape, fillRule)
	if settledShape == nil || settledShape.Empty() {
		return "", false
	}

	clipped := emulateSafeAnd(settledShape, clipPath)
	if clipped == nil || clipped.Empty() {
		return "", true
	}

	svgD := clipped.ToSVG()
	if svgD == "" {
		return "", true
	}

	return emulateEmitClippedPath(attrs, svgD), true
}

func emulateEmitClippedPath(attrs []emulateAttr, svgD string) string {
	drop := []string{"clip-path", "d", "points", "x", "y", "width", "height", "rx", "ry", "cx", "cy", "r", "fill-rule"}
	cleaned := make([]emulateAttr, 0, len(attrs))

	for _, attr := range attrs {
		low := strings.ToLower(attr.name)
		if emulateContains(drop, low) {
			continue
		}

		if low == "style" {
			attr.value = emulateStripStyleProp(attr.value, "clip-path")
			if attr.value == "" {
				continue
			}
		}

		cleaned = append(cleaned, attr)
	}

	return emulateTagText(emulatePathTag, cleaned, map[string]string{"d": svgD})
}

func emulateStripStyleProp(styleAttr, dropProp string) string {
	var kept []string

	for _, decl := range strings.Split(styleAttr, ";") {
		name, _, ok := strings.Cut(decl, ":")
		if !ok {
			continue
		}

		if strings.EqualFold(strings.TrimSpace(name), dropProp) {
			continue
		}

		if trimmed := strings.TrimSpace(decl); trimmed != "" {
			kept = append(kept, trimmed)
		}
	}

	return strings.Join(kept, ";")
}

// emulateEvenOddPolygon rewrites a <polygon fill-rule="evenodd" points="..."/>
// into a settled <path d="..."/> so nonzero rasterization paints its evenodd holes.
func emulateEvenOddPolygon(name string, attrs []emulateAttr, index map[string]string) (string, bool) {
	if name != "polygon" || !emulateEvenOddRequested(index) {
		return "", false
	}

	shape := emulateShapeToPath(name, index)
	if shape == nil || shape.Empty() {
		return "", false
	}

	settled := emulateSafeSettle(shape, canvas.EvenOdd)
	if settled == nil || settled.Empty() {
		return "", false
	}

	svgD := settled.ToSVG()
	if svgD == "" {
		return "", false
	}

	return emulateTagText(emulatePathTag, attrs, map[string]string{"d": svgD}, "points", "fill-rule"), true
}

func emulateShapeToPath(name string, index map[string]string) *canvas.Path {
	switch name {
	case emulatePathTag:
		path, err := canvas.ParseSVGPath(index["d"])
		if err != nil {
			return nil
		}

		return path
	case "polygon":
		return emulatePolygonToPath(index["points"])
	case svgRectTag:
		return emulateRectToPath(index)
	case emulateCircleTag:
		r := emulateNum(index, "r")
		if r <= 0 {
			return nil
		}

		return canvas.Circle(r).Translate(emulateNum(index, "cx"), emulateNum(index, "cy"))
	case emulateEllipseTag:
		rx, ry := emulateNum(index, "rx"), emulateNum(index, "ry")
		if rx <= 0 || ry <= 0 {
			return nil
		}

		return canvas.Ellipse(rx, ry).Translate(emulateNum(index, "cx"), emulateNum(index, "cy"))
	default:
		return nil
	}
}

func emulateRectToPath(index map[string]string) *canvas.Path {
	w, h := emulateNum(index, "width"), emulateNum(index, "height")
	if w <= 0 || h <= 0 {
		return nil
	}

	x, y := emulateNum(index, "x"), emulateNum(index, "y")
	rx, ry := emulateNum(index, "rx"), emulateNum(index, "ry")

	if rx <= 0 {
		rx = ry
	}

	if rx > 0 {
		r := math.Min(rx, math.Min(w, h)/emulateTwo)
		return canvas.RoundedRectangle(w, h, r).Translate(x, y)
	}

	return canvas.Rectangle(w, h).Translate(x, y)
}

func emulatePolygonToPath(points string) *canvas.Path {
	fields := strings.FieldsFunc(points, func(r rune) bool {
		return r == ' ' || r == ',' || r == '\t' || r == '\n' || r == '\r'
	})
	if len(fields) < 6 || len(fields)%2 != 0 {
		return nil
	}

	path := &canvas.Path{}

	for idx := 0; idx < len(fields); idx += 2 {
		x, errX := strconv.ParseFloat(fields[idx], 64)
		y, errY := strconv.ParseFloat(fields[idx+1], 64)

		if errX != nil || errY != nil {
			return nil
		}

		if idx == 0 {
			path.MoveTo(x, y)
		} else {
			path.LineTo(x, y)
		}
	}

	path.Close()

	return path
}

func emulateSafeSettle(path *canvas.Path, rule canvas.FillRule) (out *canvas.Path) {
	defer func() {
		if recover() != nil {
			out = nil
		}
	}()

	return path.Settle(rule)
}

func emulateSafeOr(a, b *canvas.Path) (out *canvas.Path) {
	defer func() {
		if recover() != nil {
			out = nil
		}
	}()

	return a.Or(b)
}

func emulateSafeAnd(a, b *canvas.Path) (out *canvas.Path) {
	defer func() {
		if recover() != nil {
			out = nil
		}
	}()

	return a.And(b)
}
