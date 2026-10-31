package layout

import (
	"sync"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// flexChildren returns the element flex items plus anonymous flex items for
// direct text runs. CSS turns non-whitespace text directly under a flex
// container into anonymous block-level flex items; dropping those nodes makes
// prose after an inline marker disappear from the display list.
const (
	defaultFlexKidsCapacity   = 32
	maxPooledFlexKidsCapacity = 2048
)

// flexKidPool provides reusable slices of html.Node pointers for flex child layout.
//
//nolint:gochecknoglobals // worker pool for child slices
var flexKidPool = sync.Pool{
	New: func() any {
		s := make([]*html.Node, 0, defaultFlexKidsCapacity)

		return &s
	},
}

//nolint:forcetypeassert // pool stores *[]*html.Node
func getFlexKids(capacity int) *[]*html.Node {
	kidsPtr := flexKidPool.Get().(*[]*html.Node)
	*kidsPtr = (*kidsPtr)[:0]

	if cap(*kidsPtr) < capacity {
		*kidsPtr = make([]*html.Node, 0, capacity)
	}

	return kidsPtr
}

func putFlexKids(kidsPtr *[]*html.Node) {
	if kidsPtr == nil || cap(*kidsPtr) > maxPooledFlexKidsCapacity {
		return
	}

	clear(*kidsPtr)
	*kidsPtr = (*kidsPtr)[:0]
	flexKidPool.Put(kidsPtr)
}

func (e *engine) flexChildren(node *html.Node, parentStyle *ResolvedStyle) []*html.Node {
	kids, _ := e.flexChildrenPooled(node, parentStyle)

	return kids
}

//nolint:cyclop // inspects DOM children for non-whitespace text runs and anonymous flex wrappers
func (e *engine) flexChildrenPooled(node *html.Node, parentStyle *ResolvedStyle) ([]*html.Node, *[]*html.Node) {
	allElements := true

	for _, child := range node.Children {
		if child.Type != html.ElementNode {
			allElements = false

			break
		}

		if cs := e.stylePtr(child); cs.Display == cssDisplayNone {
			allElements = false

			break
		}
	}

	if allElements {
		return node.Children, nil
	}

	kidPoolBuf := getFlexKids(len(node.Children))
	kids := *kidPoolBuf

	for idx := 0; idx < len(node.Children); idx++ {
		child := node.Children[idx]
		if child.Type == html.ElementNode {
			if cs := e.stylePtr(child); cs.Display != cssDisplayNone {
				kids = append(kids, child)
			}

			continue
		}

		if child.Type != html.TextNode || isAllWhitespace(child.Text) {
			continue
		}

		textNodes := []*html.Node{child}

		for idx+1 < len(node.Children) && node.Children[idx+1].Type == html.TextNode {
			idx++
			if !isAllWhitespace(node.Children[idx].Text) {
				textNodes = append(textNodes, node.Children[idx])
			}
		}

		anonymous := &html.Node{ //nolint:exhaustruct // synthetic anonymous flex item
			Type: html.ElementNode, Name: "span", Parent: node, Children: textNodes,
		}
		anonymousStyle := e.allocAnonymousFlexItemStyle(parentStyle)
		e.setSyntheticStyle(anonymous, anonymousStyle)

		kids = append(kids, anonymous)
	}

	*kidPoolBuf = kids

	return kids, kidPoolBuf
}

// anonymousFlexItemStyle preserves inherited text properties while removing
// the flex container's own box and layout properties from the anonymous item.
//
//nolint:funlen // resets the complete anonymous box model
func applyAnonymousFlexItemStyle(style *ResolvedStyle) {
	style.Display = displayBlock
	style.Position = positionStatic
	style.Float = cssDisplayNone
	style.Clear = cssDisplayNone
	style.FlexDirection = fxRow
	style.FlexWrap = cssWhiteSpaceNowrap
	style.FlexGrow = 0
	style.FlexShrink = 1
	style.FlexBasis = -1
	style.FlexBasisPercent = -1
	style.FlexOrder = 0
	style.Width = -1
	style.WidthPercent = -1
	style.Height = -1
	style.HeightPercent = -1
	style.MinWidth = 0
	style.MinWidthPercent = -1
	style.MinWidthSet = false
	style.MaxWidth = -1
	style.MaxWidthPercent = -1
	style.MinHeight = 0
	style.MinHeightPercent = -1
	style.MaxHeight = -1
	style.MarginTop = 0
	style.MarginRight = 0
	style.MarginBottom = 0
	style.MarginLeft = 0
	style.MarginTopAuto = false
	style.MarginRightAuto = false
	style.MarginBottomAuto = false
	style.MarginLeftAuto = false
	style.PaddingTop = 0
	style.PaddingRight = 0
	style.PaddingBottom = 0
	style.PaddingLeft = 0
	style.BorderTop = border{}    //nolint:exhaustruct // zero border
	style.BorderRight = border{}  //nolint:exhaustruct // zero border
	style.BorderBottom = border{} //nolint:exhaustruct // zero border
	style.BorderLeft = border{}   //nolint:exhaustruct // zero border
	style.BorderRadius = 0
	style.BorderRadiusPercent = 0
	style.BorderRadiusTopLeft = 0
	style.BorderRadiusTopRight = 0
	style.BorderRadiusBottomRight = 0
	style.BorderRadiusBottomLeft = 0
	style.BGColor = [4]float64{}
	style.ListStyleType = cssDisplayNone
	style.PageBreakBefore = ""
	style.PageBreakAfter = ""
	style.PageBreakInside = ""
	style.HasTransform = false
}

func (e *engine) allocAnonymousFlexItemStyle(parent *ResolvedStyle) *ResolvedStyle {
	style := e.allocStyle(parent)
	applyAnonymousFlexItemStyle(style)

	return style
}

func anonymousFlexItemStyle(parent *ResolvedStyle) ResolvedStyle {
	var style ResolvedStyle

	if parent != nil {
		style = *parent
	}

	applyAnonymousFlexItemStyle(&style)

	return style
}
