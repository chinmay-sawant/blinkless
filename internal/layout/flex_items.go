package layout

import (
	"sort"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

type flexMeas struct {
	n             *html.Node
	baseW         float64
	hypotheticalW float64
	grow          float64
	shrink        float64
	order         int
}

type flexColMeas struct {
	n      *html.Node
	baseH  float64
	grow   float64
	shrink float64
	order  int
}

func (e *engine) flexRowItems(kids []*html.Node, contentW float64, crossSize ...float64) []flexMeas {
	items := getFlexMeas(len(kids))
	ordered := true

	for _, kid := range kids {
		cstate := e.stylePtr(kid)

		grow := cstate.FlexGrow
		if grow < 0 {
			grow = 0
		}

		shrink := cstate.FlexShrink
		if shrink < 0 {
			shrink = 1
		}

		baseW := e.flexItemBaseWidth(kid, cstate, contentW)
		if len(crossSize) > 0 {
			if aspectW := e.flexAspectContentWidth(kid, cstate, crossSize[0]); aspectW > baseW {
				baseW = aspectW
			}
		}

		item := flexMeas{
			n: kid, baseW: baseW,
			hypotheticalW: 0, grow: grow, shrink: shrink, order: cstate.FlexOrder,
		}
		item.hypotheticalW = e.flexHypotheticalMainSize(item, contentW)

		if len(items) > 0 && items[len(items)-1].order > item.order {
			ordered = false
		}

		items = append(items, item)
	}

	if !ordered {
		sort.SliceStable(items, func(i, j int) bool { return items[i].order < items[j].order })
	}

	return items
}

func (e *engine) flexColumnItems(kids []*html.Node, contentW, contentH float64) []flexColMeas {
	items := getFlexColMeas(len(kids))
	ordered := true

	for _, kid := range kids {
		cstate := e.stylePtr(kid)

		grow := cstate.FlexGrow
		if grow < 0 {
			grow = 0
		}

		shrink := cstate.FlexShrink
		if shrink < 0 {
			shrink = 1
		}

		item := flexColMeas{
			n: kid, baseH: e.flexItemBaseHeight(kid, cstate, contentW, contentH),
			grow: grow, shrink: shrink, order: cstate.FlexOrder,
		}

		if len(items) > 0 && items[len(items)-1].order > item.order {
			ordered = false
		}

		items = append(items, item)
	}

	if !ordered {
		sort.SliceStable(items, func(i, j int) bool { return items[i].order < items[j].order })
	}

	return items
}
