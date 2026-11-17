package layout

import "strings"

// baseline-source (CSS Inline 3) picks which baseline set of an inline-level
// box lines up with the parent line. It does not inherit. The initial auto
// uses the last baseline on an inline-block and the first baseline on every
// other box. A token outside auto, first, and last leaves the previous value.

const (
	baselineSourceProp  = "baseline-source"
	baselineSourceAuto  = "auto"
	baselineSourceFirst = "first"
	baselineSourceLast  = "last"
)

// applyBaselineSourceProps owns baseline-source. Registered in styleGroups
// (style_cascade.go). A bad token returns true without writing, so the
// previous value stays.
func applyBaselineSourceProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	if prop != baselineSourceProp {
		return false
	}

	switch normalized := normalizeCSSValue(value); normalized {
	case baselineSourceAuto, baselineSourceFirst, baselineSourceLast:
		style.BaselineSource = normalized
	}

	return true
}

// noteInlineBaselineSource records the text baseline an inline-block shares
// with its parent line. A box with no line, or one that already uses the
// bottom margin edge, keeps that edge.
func (e *engine) noteInlineBaselineSource(item *inlineItem) {
	choice, ok := baselineSourceChoice(item)
	if !ok {
		return
	}

	first, last, found := e.inlineBlockTextBaselines(item.blockBox)
	if !found {
		return
	}

	chosen := last
	if choice == baselineSourceFirst {
		chosen = first
	}

	item.sourceBaseline = clampBaselineOffset(chosen-item.blockBox.y, item.h)
	item.sourceBaselineSet = true
}

// baselineSourceChoice reports the baseline set to use. auto on an
// inline-block is last. auto on any other box in this collector keeps the
// historical border edge, because this phase only moves inline-block text.
func baselineSourceChoice(item *inlineItem) (string, bool) {
	if item == nil || item.blockBox == nil || item.marginBaseline || item.style == nil {
		return "", false
	}

	switch item.style.BaselineSource {
	case baselineSourceFirst, baselineSourceLast:
		return item.style.BaselineSource, true
	case baselineSourceAuto, "":
		if item.style.Display != cssDisplayInlineBlock {
			return "", false
		}

		return baselineSourceLast, true
	default:
		return "", false
	}
}

// inlineBlockTextBaselines returns the first and last text-op baselines
// inside block. Y grows downward, so the first line is the smaller Y.
func (e *engine) inlineBlockTextBaselines(block *box) (float64, float64, bool) {
	end, ok := e.inlineBlockOpEnd(block)
	if !ok {
		return 0, 0, false
	}

	var first, last float64

	found := false

	for index := block.opStart; index <= end; index++ {
		if e.ops[index].Kind != OpText || strings.TrimSpace(e.ops[index].Text) == "" {
			continue
		}

		lineY := e.ops[index].Y
		if !found || lineY < first {
			first = lineY
		}

		if !found || lineY > last {
			last = lineY
		}

		found = true
	}

	return first, last, found
}

// inlineBlockOpEnd returns the last op index inside block to scan. ok is false
// when block has no usable op range.
func (e *engine) inlineBlockOpEnd(block *box) (int, bool) {
	if block == nil || block.opEnd < block.opStart || block.opStart < 0 {
		return 0, false
	}

	end := block.opEnd
	if end >= len(e.ops) {
		end = len(e.ops) - 1
	}

	return end, true
}

func clampBaselineOffset(offset, height float64) float64 {
	if offset < 0 {
		return 0
	}

	if height > 0 && offset > height {
		return height
	}

	return offset
}

// applyBaselineSourceEdges moves an atomic's alignment point from the bottom
// border edge up to the stored text baseline. The distance below that point
// becomes descent so the line box still covers the rest of the border box.
func applyBaselineSourceEdges(item *inlineItem, ascent, descent float64) (float64, float64) {
	delta := baselineSourceDelta(item)
	ascent -= delta
	descent += delta

	return ascent, descent
}

// applyBaselineSourceAscent is the border-box part of alignedInlineTop after
// the same shift applyBaselineSourceEdges applies to the line ascent.
func applyBaselineSourceAscent(item *inlineItem, ascent float64) float64 {
	return ascent - baselineSourceDelta(item)
}

func baselineSourceDelta(item *inlineItem) float64 {
	if item == nil || !item.sourceBaselineSet || item.marginBaseline {
		return 0
	}

	return item.h - item.sourceBaseline
}
