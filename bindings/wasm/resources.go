package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/chinmay-sawant/blinkless/internal/load"
	"github.com/chinmay-sawant/blinkless/internal/settings"
)

// errNonInlineImage marks an image source the browser adapter cannot resolve.
// Browser input is inline HTML with no base URL, so only data: URLs are
// readable; file paths and network URLs are refused before any IO.
var errNonInlineImage = errors.New("browser image source must be a data: URL")

// inlineImageResolver returns the layout image resolver for the browser
// adapter. It accepts data: URLs only and decodes them through the shared
// loader, so body caps, base64 handling, and percent escapes stay identical to
// the native pipeline. The loader is built on first use so a document without
// images pays nothing.
func inlineImageResolver(ctx context.Context) func(string) ([]byte, error) {
	var (
		once    sync.Once
		loader  *load.Loader
		initErr error
	)

	return func(src string) ([]byte, error) {
		trimmed := strings.TrimSpace(src)
		if !strings.HasPrefix(strings.ToLower(trimmed), "data:") {
			return nil, fmt.Errorf("%w: %q", errNonInlineImage, clipSource(src))
		}

		once.Do(func() {
			loader, initErr = load.NewLoaderWithError(settings.LoadGlobal{})
		})

		if initErr != nil {
			return nil, initErr
		}

		res, err := loader.FetchSub(ctx, "", trimmed, settings.LoadPage{})
		if err != nil {
			return nil, fmt.Errorf("inline image: %w", err)
		}

		return res.Body, nil
	}
}

// clipSource shortens a source string for an error message; data URLs can be
// megabytes long and must not be echoed whole.
func clipSource(src string) string {
	const limit = 64

	if len(src) <= limit {
		return src
	}

	return src[:limit] + "..."
}
