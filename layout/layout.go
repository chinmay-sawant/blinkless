package layout

import (
	ilayout "github.com/chinmay-sawant/blinkless/internal/layout"
)

// ptToPx converts a layout point to one CSS pixel at zoom 1.
const ptToPx = 96.0 / 72.0

// Box is one element border box in CSS pixels.
// Action is the data-action attribute. Text is the descendant text.
type Box struct {
	ID     string
	Tag    string
	Action string
	Text   string
	X      float64
	Y      float64
	W      float64
	H      float64
}

// Options carries the optional inputs a placement can use beyond the styled
// document itself.
type Options struct {
	// Images returns encoded image bytes (PNG, JPEG, or SVG) for one source,
	// such as an <img src> value or a CSS background-image url(...) target.
	// Nil means no source resolves, so image paint is skipped. A transform
	// that cannot stay in those source bytes is re-encoded as a PNG on the
	// image operation. That payload is the bitmap fallback. It is not a
	// picture of the page.
	Images func(src string) ([]byte, error)
}

func boxesFrom(placed []ilayout.PlacedElement) []Box {
	boxes := make([]Box, len(placed))

	for i, item := range placed {
		boxes[i] = Box{
			ID:     item.ID,
			Tag:    item.Tag,
			Action: item.Action,
			Text:   item.Text,
			X:      item.X * ptToPx,
			Y:      item.Y * ptToPx,
			W:      item.W * ptToPx,
			H:      item.H * ptToPx,
		}
	}

	return boxes
}
