package layout

import "errors"

var (
	// ErrNilContext means DisplayList was called without a context.
	ErrNilContext = errors.New("layout: nil context")

	// ErrNilDocument means DisplayList was given no styled document.
	ErrNilDocument = errors.New("layout: nil document")
)
