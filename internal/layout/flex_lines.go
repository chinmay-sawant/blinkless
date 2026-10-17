// Flex line formation: wrapping measured items into lines, the line-fit
// trial, and the per-line order helpers. Extracted from flex.go (module-wise
// split, no behavior change) so the file-size gate stays satisfied; see
// flex.go for the flex layout overview.
package layout

import (
	"slices"
)

// flexWrapLines packs measured items into flex lines, honoring wrap mode and
// reverse direction (item order within each line, then line order). The trial
// line is checked after flex grow/shrink and min/max clamping, because a
// content-based minimum can make a line overflow even when raw flex bases fit.
func (e *engine) flexWrapLines(
	items []flexMeas, wrap, reverse, wrapReverse bool, colGap, contentW float64,
) [][]flexMeas {
	if !wrap {
		if reverse {
			reverseFlexMeas(items)
		}

		return [][]flexMeas{items}
	}

	var lines [][]flexMeas

	line := make([]flexMeas, 0, len(items))
	used := 0.0

	poll := newCtxPoll(e.ctx)
	for _, item := range items {
		if poll.poll() {
			e.err = poll.err

			return nil
		}

		// wrap-before: flex forces a new flex line even when the item fits
		// (wrap_break.go; nowrap already returned above).
		if len(line) > 0 && e.flexWrapBreakBefore(item) {
			lines = append(lines, finalizeFlexLine(line, reverse))
			line = nil
			used = 0
		}

		need := item.hypotheticalW
		if len(line) > 0 {
			need += colGap
		}

		trial := append(append([]flexMeas(nil), line...), item)
		if len(line) > 0 && !e.flexLineFits(trial, contentW, colGap) {
			lines = append(lines, finalizeFlexLine(line, reverse))
			line = nil
			used = 0
			need = item.hypotheticalW
		}

		line = append(line, item)
		used += need

		// wrap-after: flex ends the line after the item.
		if e.flexWrapBreakAfter(item) {
			lines = append(lines, finalizeFlexLine(line, reverse))
			line = nil
			used = 0
		}
	}

	if len(line) > 0 {
		lines = append(lines, finalizeFlexLine(line, reverse))
	}

	if wrapReverse {
		slices.Reverse(lines)
	}

	return lines
}

func (e *engine) flexLineFits(items []flexMeas, contentW, gap float64) bool {
	widths := e.flexLineWidths(items, contentW, gap)
	total := gap * float64(len(widths)-1)

	for idx, width := range widths {
		total += width + flexRowNormalMargins(e, items[idx])
	}

	if total > contentW+layoutEpsilon {
		return false
	}

	// A wrapped line may be a few points over its raw hypothetical sizes when
	// flex-shrink resolves font-metric rounding. Do not use a large shrink to
	// defeat wrapping: that collapses later items into unreadable slivers and
	// also pulls full-width captions into the row. A candidate is therefore
	// accepted only when every positive-base item remains within ten percent of
	// its hypothetical size.
	const maxLineShrink = 0.10

	for idx, item := range items {
		if item.baseW <= layoutEpsilon || item.shrink <= 0 {
			continue
		}

		if widths[idx] < item.baseW*(1-maxLineShrink) {
			return false
		}
	}

	return true
}

func finalizeFlexLine(line []flexMeas, reverse bool) []flexMeas {
	if reverse {
		reverseFlexMeas(line)
	}

	return line
}

func reverseFlexMeas(line []flexMeas) {
	slices.Reverse(line)
}
