package prepare

import (
	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// inlineSheet returns the sheet parsed for one <style> node, or nil.
func (c *SheetCache) inlineSheet(node *html.Node) *css.Stylesheet {
	if c == nil {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.inline[node]
}

// noteInline stores one parsed <style> sheet.
func (c *SheetCache) noteInline(node *html.Node, sheet *css.Stylesheet) {
	if c == nil || sheet == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.inline[node] = sheet
}

// linkSheet returns the sheet fetched for one <link> node, or the zero value.
func (c *SheetCache) linkSheet(node *html.Node) cachedSheet {
	if c == nil {
		return cachedSheet{}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.links[node]
}

// noteLink stores one fetched <link> sheet.
func (c *SheetCache) noteLink(node *html.Node, sheet cachedSheet) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.links[node] = sheet
}

// importSheet returns the sheet fetched for one @import rule, or the zero value.
func (c *SheetCache) importSheet(key importKey) cachedSheet {
	if c == nil {
		return cachedSheet{}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.imports[key]
}

// noteImport stores one fetched @import sheet.
func (c *SheetCache) noteImport(key importKey, sheet cachedSheet) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.imports[key] = sheet
}

// noteViewportGating records that stylesheet collection encountered a <link
// media> or @import media query whose inclusion can vary with viewport size.
func (c *SheetCache) noteViewportGating() {
	if c == nil {
		return
	}

	c.viewportGating.Store(true)
}

// HasViewportGating reports whether any collected <link> or @import rule in
// this tree carried a media query. When false, a viewport-only relayout over
// an untouched tree produces the same sheet list and font registry without
// walking the DOM again.
func (c *SheetCache) HasViewportGating() bool {
	if c == nil {
		return true
	}

	return c.viewportGating.Load()
}
