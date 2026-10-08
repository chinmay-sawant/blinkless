// Package blinkless is the blinkless renderer. It turns HTML into a
// drawing list and can encode that list as a PNG or JPEG. It does not write
// PDF files, and it does not start a browser.
//
// ImageDocument is the root image API. html.Parse, css.Apply, and
// layout.DisplayList are the pieces underneath. screen.Render parses, applies
// CSS, lays out, and returns a PNG plus the boxes.
//
// The module path is github.com/chinmay-sawant/blinkless.
package blinkless
