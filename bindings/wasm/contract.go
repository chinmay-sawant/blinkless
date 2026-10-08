package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

const (
	defaultMode       = "png"
	maxHTMLBytes      = 4 << 20
	maxOutputBytes    = 32 << 20
	maxImageDimension = 4096
)

var (
	errInvalidRequest  = errors.New("invalid request")
	errUnsupportedMode = errors.New("unsupported output mode")
	errInputTooLarge   = errors.New("HTML input exceeds browser limit")
	errOutputTooLarge  = errors.New("output exceeds browser limit")
	errImageTooLarge   = errors.New("image dimensions exceed browser limit")
)

// Request is the JSON boundary used by the browser adapter. It deliberately
// contains no native file or URL fields: browser input is inline HTML only.
type Request struct {
	HTML        string `json:"html"`
	Mode        string `json:"mode"`
	PageSize    string `json:"pageSize"`
	Orientation string `json:"orientation"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Padding     int    `json:"padding"`
	Quality     int    `json:"quality"`
}

// Result is the owned output returned by one browser conversion.
type Result struct {
	Mode   string
	MIME   string
	Bytes  []byte
	Width  int
	Height int
}

// ErrorResponse is the stable error shape exposed to JavaScript.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// DecodeRequest validates JSON before any renderer or loader work starts.
func DecodeRequest(raw string) (Request, error) {
	decoder := json.NewDecoder(io.LimitReader(strings.NewReader(raw), maxHTMLBytes+1<<20))
	decoder.DisallowUnknownFields()

	var request Request
	if err := decoder.Decode(&request); err != nil {
		return Request{}, fmt.Errorf("%w: %v", errInvalidRequest, err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Request{}, fmt.Errorf("%w: multiple JSON values", errInvalidRequest)
		}
		return Request{}, fmt.Errorf("%w: %v", errInvalidRequest, err)
	}

	if err := request.validate(); err != nil {
		return Request{}, err
	}

	return request, nil
}

func (r *Request) validate() error {
	if r == nil {
		return fmt.Errorf("%w: request is nil", errInvalidRequest)
	}

	if len([]byte(r.HTML)) == 0 {
		return fmt.Errorf("%w: HTML is empty", errInvalidRequest)
	}
	if len([]byte(r.HTML)) > maxHTMLBytes {
		return fmt.Errorf("%w: %w", errInvalidRequest, errInputTooLarge)
	}

	r.Mode = strings.ToLower(strings.TrimSpace(r.Mode))
	if r.Mode == "" {
		r.Mode = defaultMode
	}

	switch r.Mode {
	case "png", "jpeg":
		if r.Width < 0 || r.Height < 0 || r.Padding < 0 || r.Width > maxImageDimension || r.Height > maxImageDimension {
			return fmt.Errorf("%w: %w", errInvalidRequest, errImageTooLarge)
		}
		if r.Quality < 0 || r.Quality > 100 {
			return fmt.Errorf("%w: image quality must be between 0 and 100", errInvalidRequest)
		}
	default:
		return fmt.Errorf("%w: %q", errUnsupportedMode, r.Mode)
	}

	return nil
}

// Convert lays the HTML out and returns the drawing list as JSON. It does
// not encode a PNG or JPEG. Width and height use the bitmap viewport
// fallback: an unset width is 1024, and an unset height uses that width.
func Convert(ctx context.Context, request Request, onProgress func(string, int)) (Result, error) {
	if err := request.validate(); err != nil {
		return Result{}, err
	}
	if ctx == nil {
		return Result{}, layout.ErrNilContext
	}

	if onProgress != nil {
		onProgress("layout", 0)
	}

	width := request.Width
	if width <= 0 {
		width = 1024
	}

	height := request.Height
	if height <= 0 {
		height = width
	}

	tree, err := html.Parse([]byte(request.HTML))
	if err != nil {
		return Result{}, err
	}

	styled, err := css.Apply(ctx, tree, css.Options{
		WidthPx:  width,
		HeightPx: height,
		Media:    "screen",
	})
	if err != nil {
		return Result{}, err
	}

	display, err := layout.DisplayList(ctx, styled)
	if err != nil {
		return Result{}, err
	}

	payload, err := json.Marshal(map[string]int{
		"ops":    len(display.Ops),
		"width":  display.Width,
		"height": display.Height,
	})
	if err != nil {
		return Result{}, err
	}
	if len(payload) > maxOutputBytes {
		return Result{}, errOutputTooLarge
	}

	if onProgress != nil {
		onProgress("layout", 100)
	}

	return Result{
		Mode:   "display",
		MIME:   "application/json",
		Bytes:  payload,
		Width:  display.Width,
		Height: display.Height,
	}, nil
}

func errorResponse(err error) ErrorResponse {
	if err == nil {
		return ErrorResponse{}
	}

	code := "render_error"
	switch {
	case errors.Is(err, errInputTooLarge), errors.Is(err, errOutputTooLarge),
		errors.Is(err, errImageTooLarge):
		code = "resource_limit"
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		code = "timeout"
	case errors.Is(err, errInvalidRequest), errors.Is(err, errUnsupportedMode),
		errors.Is(err, css.ErrBadSize), errors.Is(err, layout.ErrNilDocument):
		code = "invalid_request"
	}

	return ErrorResponse{Code: code, Message: err.Error()}
}
