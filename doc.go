// Package blinkless is the layout engine. HTML and CSS are the inputs.
// html.Parse reads the document, css.Apply applies the stylesheets, and
// layout.DisplayList returns the drawing list. The list is the output.
// The engine does not run script and it does not encode a page bitmap.
// An image operation keeps its own encoded bytes, and re-encodes those
// bytes as a PNG only when orientation or a clip requires it.
//
// The module path is github.com/chinmay-sawant/blinkless.
package blinkless
