package layout

import (
	"strconv"
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// textFitIterations bounds the re-measure loop. Glyph advances are close to
// linear in font size, so the second pass only trims rounding.
const textFitIterations = 3

// textFitSpec is a parsed text-fit value.
type textFitSpec struct {
	grow, shrink bool
	// limit is the scale cap as a ratio (1.5 for 150%), or 0 for none.
	limit float64
}

// parseTextFit reads the normalized declared value. The mode words
// consistent, per-line and per-line-all pick how several lines share one
// scale. A block with one text line has one scale, so they read the same.
func parseTextFit(value string) (textFitSpec, bool) {
	if value == "" || value == "none" {
		return textFitSpec{grow: false, shrink: false, limit: 0}, false
	}

	fields := strings.Fields(value)
	if len(fields) == 0 {
		return textFitSpec{grow: false, shrink: false, limit: 0}, false
	}

	spec := textFitSpec{grow: fields[0] == "grow", shrink: fields[0] == "shrink", limit: 0}
	if !spec.grow && !spec.shrink {
		return textFitSpec{grow: false, shrink: false, limit: 0}, false
	}

	last := fields[len(fields)-1]
	if strings.HasSuffix(last, "%") {
		if number, err := strconv.ParseFloat(strings.TrimSuffix(last, "%"), 64); err == nil && number > 0 {
			spec.limit = number / 100 //nolint:mnd // percent
		}
	}

	return spec, true
}

// textFitSize searches for the font size at which line fills contentW. It
// returns base when the declaration asks for no change.
//
//nolint:cyclop // the search stops on four separate conditions
func (e *engine) textFitSize(spec textFitSpec, line string, first *ResolvedStyle, contentW float64) float64 {
	base := first.FontSize
	size := base

	for range textFitIterations {
		probe := *first
		probe.FontSize = size

		width := e.measureTextFace(line, &probe)
		if width <= 0 {
			return base
		}

		ratio := contentW / width
		if (ratio < 1 && !spec.shrink) || (ratio > 1 && !spec.grow) || (ratio > 0.9999 && ratio < 1.0001) {
			break
		}

		next := size * ratio
		if ratio < 1 {
			next *= 0.9999 // stay on the fitting side of rounding
		}

		if spec.limit > 0 && ratio < 1 {
			next = max(next, base*spec.limit)
		}

		if spec.limit > 0 && ratio > 1 {
			next = min(next, base*spec.limit)
		}

		size = next
	}

	return size
}

// textFitLine joins the text nodes of a block into one collapsed line, or
// reports false when the block holds anything but text.
func textFitLine(nodes []*html.Node) (string, bool) {
	var text strings.Builder

	for _, node := range nodes {
		if node == nil || node.Type != html.TextNode {
			return "", false
		}

		text.WriteString(node.Text)
	}

	line := strings.TrimSpace(collapseWS(text.String()))

	return line, line != ""
}

// applyTextFit scales the font of a block whose inline content is plain text
// so the whole text fits one line of the content box. For text-fit: shrink a
// line wider than the box gets a smaller font. For grow a narrower line gets a
// larger one. A percentage caps the scale: shrink stops at that fraction of the
// font, grow at that multiple. The text nodes keep the new size for the
// duration of the layout call and the returned func puts the old size back.
// A block holding elements, images or line breaks keeps its declared font size.
//
//nolint:cyclop // one guard per reason a block keeps its declared size
func (e *engine) applyTextFit(boxNode *box, nodes []*html.Node, contentW float64) func() {
	if boxNode == nil || boxNode.style == nil || contentW <= 0 || len(nodes) == 0 {
		return nil
	}

	spec, ok := parseTextFit(boxNode.style.declared().TextFit)
	if !ok {
		return nil
	}

	line, plain := textFitLine(nodes)
	first := e.stylePtr(nodes[0])

	if !plain || first == nil {
		return nil
	}

	base := first.FontSize

	size := e.textFitSize(spec, line, first, contentW)
	if size == base {
		return nil
	}

	styles := make([]*ResolvedStyle, 0, len(nodes))

	for _, node := range nodes {
		if style := e.stylePtr(node); style != nil {
			style.FontSize = size

			styles = append(styles, style)
		}
	}

	return func() {
		for _, style := range styles {
			style.FontSize = base
		}
	}
}
