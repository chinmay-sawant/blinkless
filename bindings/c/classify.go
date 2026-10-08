package main

import (
	"context"
	"errors"

	"github.com/chinmay-sawant/blinkless/internal/load"
)

// classifyError maps an engine failure onto the ABI status table documented
// in include/blinkless.h. It stays free of C types so tests can drive it
// under both the cgo and pure-Go build modes. A done context always wins:
// when cancellation raced with another failure, callers observe TIMEOUT
// instead of a generic render error.
func classifyError(err error, ctx context.Context) int32 {
	if err == nil {
		return statusOK
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded), ctxDone(ctx):
		return statusTimeout
	case errors.Is(err, load.ErrAccessDenied),
		errors.Is(err, load.ErrNetworkPolicy),
		errors.Is(err, load.ErrInvalidProxy):
		return statusLoadDenied
	default:
		return statusRenderError
	}
}

// ctxDone reports whether ctx already carries a deadline or cancellation.
func ctxDone(ctx context.Context) bool {
	return ctx != nil && ctx.Err() != nil
}
