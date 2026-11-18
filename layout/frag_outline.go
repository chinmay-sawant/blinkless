package layout

import (
	"context"
	"fmt"

	"github.com/chinmay-sawant/blinkless/css"
	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
	ihtml "github.com/chinmay-sawant/blinkless/internal/html"
	ilayout "github.com/chinmay-sawant/blinkless/internal/layout"
	"github.com/chinmay-sawant/blinkless/internal/pubstate"
)

// OutlineEntry is one document outline entry made by the bookmark-label,
// bookmark-level and bookmark-state properties. The engine paints none of it:
// a host reads the list and builds its own outline or table of contents.
type OutlineEntry struct {
	// Label is the used bookmark-label: quoted strings, content, and attr()
	// joined into one title.
	Label string
	// Level is the declared bookmark-level, 1 or more.
	Level int
	// Open is false when bookmark-state is closed.
	Open bool
	// Parent is the index in the returned list of the nearest earlier entry
	// with a smaller level, or -1 for a top entry.
	Parent int
}

// Outline returns the bookmark entries of doc in document order. An element
// is listed when its bookmark-level is an integer of 1 or more and its label
// is not blank; bookmark-level: none and bookmark-label: none leave it out.
func Outline(ctx context.Context, doc *css.Document) ([]OutlineEntry, error) {
	root, opts, err := fragRootOptions(ctx, doc)
	if err != nil {
		return nil, err
	}

	var out []OutlineEntry

	var walk func(nodes []*ilayout.BookmarkNode, parent int)

	walk = func(nodes []*ilayout.BookmarkNode, parent int) {
		for _, n := range nodes {
			out = append(out, OutlineEntry{Label: n.Label, Level: n.Level, Open: n.Open, Parent: parent})
			walk(n.Children, len(out)-1)
		}
	}

	walk(ilayout.BuildBookmarkOutline(root, opts), -1)

	return out, nil
}

// CopiedFragment is one named copy made by copy-into: the text of an element
// (or one of its attributes) under a name a host can show elsewhere, such as
// a running header. The element still paints where it is.
type CopiedFragment = ilayout.CopiedFragment

// CopiedFragments returns every copy-into entry of doc in document order.
func CopiedFragments(ctx context.Context, doc *css.Document) ([]CopiedFragment, error) {
	root, opts, err := fragRootOptions(ctx, doc)
	if err != nil {
		return nil, err
	}

	return ilayout.CopiedFragments(root, opts), nil
}

func fragRootOptions(ctx context.Context, doc *css.Document) (*ihtml.Node, ilayout.Options, error) {
	if ctx == nil {
		return nil, ilayout.Options{}, ErrNilContext
	}

	styled, ok := pubstate.StyledOf(doc)
	if !ok {
		return nil, ilayout.Options{}, ErrNilDocument
	}

	font, err := pdf.DefaultFont()
	if err != nil {
		return nil, ilayout.Options{}, fmt.Errorf("layout: default font: %w", err)
	}

	widthPt, heightPt := displayViewportPt(styled)

	return styled.Root, displayLayoutOptions(styled, font, nil, widthPt, heightPt), nil
}
