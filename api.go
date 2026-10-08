// Package blinkless exposes the Go-native Document and ImageDocument
// models over the pure-Go conversion engines.
package blinkless

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/errs"
	"github.com/chinmay-sawant/blinkless/internal/imageout"
	"github.com/chinmay-sawant/blinkless/internal/line"
	"github.com/chinmay-sawant/blinkless/internal/load"
)

// NetworkPolicy controls HTTP(S) document and subresource loading.
type NetworkPolicy = load.NetworkPolicy

// CompatibleNetworkPolicy preserves historical permissive URL behavior.
func CompatibleNetworkPolicy() NetworkPolicy {
	return load.CompatibleNetworkPolicy()
}

// RestrictedNetworkPolicy blocks private destinations and cross-host
// redirects unless an explicit policy exception permits them.
func RestrictedNetworkPolicy() NetworkPolicy {
	return load.RestrictedNetworkPolicy()
}

// Static errors are stable errors.Is targets for the image API.
var (
	ErrEmptyHTML          = errors.New("blinkless: empty HTML")
	ErrMissingImageOutput = imageout.ErrMissingOutput
	ErrNilContext         = errs.ErrNilContext
	errNilLogWriter       = errors.New("blinkless: nil log writer")
)

// convertHooks translates engine log/progress streams into native Document
// callbacks without exposing internal request or settings types.
type convertHooks struct {
	OnInfo, OnWarn, OnError func(string)
	OnPhase                 func(string)
	OnProgress              func(int)
}

func (h convertHooks) lineLog() *lineLog {
	return &lineLog{
		buf:     bytes.Buffer{},
		onInfo:  h.OnInfo,
		onWarn:  h.OnWarn,
		onError: h.OnError,
	}
}

func (h convertHooks) progress() func(string, int) {
	if h.OnPhase == nil && h.OnProgress == nil {
		return nil
	}

	return func(phase string, percent int) {
		if h.OnPhase != nil {
			h.OnPhase(phase)
		}

		if h.OnProgress != nil {
			h.OnProgress(percent)
		}
	}
}

func (h convertHooks) executeImageTo(ctx context.Context, req *imageout.Request) error {
	if ctx == nil {
		return reportPreflight(h.OnError, ErrNilContext)
	}

	if err := imageout.RunRequest(ctx, req, h.lineLog()); err != nil {
		return reportPreflight(h.OnError, fmt.Errorf("image convert: %w", err))
	}

	return nil
}

func reportPreflight(onError func(string), err error) error {
	if err != nil && onError != nil {
		onError(err.Error())
	}

	return err
}

type lineLog struct {
	buf     bytes.Buffer
	onInfo  func(string)
	onWarn  func(string)
	onError func(string)
}

// Compile-time check: lineLog satisfies io.Writer.
var _ io.Writer = (*lineLog)(nil)

// Write splits engine log lines and routes them to the corresponding public
// callback. The loop is deliberately kept here so partial writes are buffered
// across engine calls.
//
//nolint:cyclop,wsl // line classification has one branch per public severity.
func (w *lineLog) Write(payload []byte) (int, error) {
	if w == nil {
		return 0, errNilLogWriter
	}

	w.buf.Write(payload)

	for {
		raw := w.buf.Bytes()
		index := bytes.IndexByte(raw, '\n')

		if index < 0 {
			break
		}

		message := strings.TrimSpace(string(raw[:index]))
		w.buf.Next(index + 1)

		if message == "" {
			continue
		}

		switch line.SeverityOf(message) {
		case line.Warn:

			if w.onWarn != nil {
				w.onWarn(message)
			}
		case line.Error:

			if w.onError != nil {
				w.onError(message)
			}
		case line.Info:

			if w.onInfo != nil {
				w.onInfo(message)
			}
		case line.Unknown:
			// SeverityOf never returns Unknown; listed for exhaustiveness.
		}
	}

	return len(payload), nil
}
