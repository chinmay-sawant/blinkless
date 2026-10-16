package layout

import "sort"

func PageBreaks(res *Result) ([]float64, [][2]float64) {
	if res == nil || res.root == nil {
		return nil, nil
	}

	collector := &breakCollector{} //nolint:exhaustruct // collectors start empty by design

	var walk func(boxNode *box)
	walk = func(boxNode *box) {
		if boxNode == nil {
			return
		}

		collector.add(boxNode, res.Height)

		for _, child := range boxNode.children {
			walk(child)
		}
	}
	walk(res.root)

	return collector.forcedOut(), collector.keep
}

type breakCollector struct {
	forced []float64
	keep   [][2]float64
}

func (c *breakCollector) add(boxNode *box, canvasH float64) {
	if c == nil || boxNode == nil || boxNode.style == nil || boxNode.height <= 0 {
		return
	}

	if boxNode.style.PageBreakBefore == pageBreakAlways && boxNode.y > 0 {
		c.forced = append(c.forced, boxNode.y)
	}

	if boxNode.style.PageBreakAfter == pageBreakAlways && boxNode.y+boxNode.height < canvasH {
		c.forced = append(c.forced, boxNode.y+boxNode.height)
	}

	if boxNode.style.PageBreakInside == avoidKeyword {
		c.keep = append(c.keep, [2]float64{boxNode.y, boxNode.y + boxNode.height})
	}
}

func (c *breakCollector) forcedOut() []float64 {
	if c == nil {
		return nil
	}

	sort.Float64s(c.forced)

	out := c.forced[:0]
	for _, y := range c.forced {
		if len(out) == 0 || y-out[len(out)-1] > 0.01 {
			out = append(out, y)
		}
	}

	return out
}
