package blinkless

import (
	"bytes"
	"context"
	"io"
	"slices"
	"time"

	"github.com/chinmay-sawant/blinkless/internal/imageout"
	"github.com/chinmay-sawant/blinkless/internal/load"
	"github.com/chinmay-sawant/blinkless/internal/settings"
)

// Content identifies exactly one document source. HTML is an in-memory
// document and Base resolves its relative resources; File and URL are loaded
// by the engine without public inline/data prefixes.
type Content struct {
	HTML []byte
	Base string
	File string
	URL  string
}

// HTML returns an owned in-memory HTML source. An optional base URL resolves
// relative resources referenced by the document.
func HTML(html []byte, base ...string) Content {
	owned := make([]byte, len(html))
	copy(owned, html)

	var baseURL string
	if len(base) > 0 {
		baseURL = base[0]
	}

	return Content{HTML: owned, Base: baseURL, File: "", URL: ""}
}

// File returns a local filesystem document source.
func File(path string) Content {
	return Content{HTML: nil, Base: "", File: path, URL: ""}
}

// URL returns an HTTP(S) document source.
func URL(rawURL string) Content {
	return Content{HTML: nil, Base: "", File: "", URL: rawURL}
}

// Crop identifies an image crop rectangle in pixels.
type Crop struct {
	Left   int
	Top    int
	Width  int
	Height int
}

// ImageDocument is the preferred HTML-to-image API.
type ImageDocument struct {
	Source Content

	Width       int
	Height      int
	Padding     int
	Format      string
	Quality     int
	SmartWidth  *bool
	Transparent bool
	Crop        *Crop
	Zoom        float64

	Allow           []string
	AllowLocalFiles bool
	Background      *bool
	FontPaths       []string
	UseSystemFonts  bool
	Network         *NetworkPolicy

	Now        func() time.Time
	OnInfo     func(string)
	OnWarn     func(string)
	OnError    func(string)
	OnPhase    func(string)
	OnProgress func(int)
}

// WriteImage validates d, maps it to the image engine request, and writes
// encoded image bytes to w.
//
//nolint:wsl,mnd // the image lifecycle reports its documented 0-to-100 range.
func (d *ImageDocument) WriteImage(ctx context.Context, writer io.Writer) error {
	if d == nil {
		return ErrNilImageDocument
	}

	if err := d.Validate(); err != nil {
		return reportPreflight(d.OnError, err)
	}

	if writer == nil {
		return reportPreflight(d.OnError, ErrMissingImageOutput)
	}

	req := d.toImageRequest(writer)
	hooks := convertHooks{
		OnInfo:     d.OnInfo,
		OnWarn:     d.OnWarn,
		OnError:    d.OnError,
		OnPhase:    d.OnPhase,
		OnProgress: d.OnProgress,
	}

	if d.OnPhase != nil {
		d.OnPhase("Rendering image")
	}
	if d.OnProgress != nil {
		d.OnProgress(0)
	}

	if err := hooks.executeImageTo(ctx, req); err != nil {
		return err
	}

	if d.OnProgress != nil {
		d.OnProgress(100)
	}
	if d.OnPhase != nil {
		d.OnPhase("Done")
	}

	return nil
}

// Image returns encoded PNG or JPEG bytes produced by the image document.
//
// It buffers the entire image in memory and then returns an owned copy, so
// peak memory is about twice the image size (the staging buffer plus the
// returned slice). For large renders prefer WriteImage, which streams directly
// to the supplied io.Writer without retaining a second copy. The returned
// slice is owned by the caller and the staging buffer is not retained after
// return.
func (d *ImageDocument) Image(ctx context.Context) ([]byte, error) {
	if d == nil {
		return nil, ErrNilImageDocument
	}

	var output bytes.Buffer
	if err := d.WriteImage(ctx, &output); err != nil {
		return nil, err
	}

	return append([]byte(nil), output.Bytes()...), nil
}

func (d *ImageDocument) toImageRequest(output io.Writer) *imageout.Request {
	global := settings.DefaultPdfGlobal()
	image := settings.DefaultImageGlobal()

	if d.Background != nil {
		global.Background = *d.Background
	}
	if d.AllowLocalFiles {
		global.Load.EnableLocalFileAccess = true
	}
	global.Load.Allow = slices.Clone(d.Allow)
	global.FontPaths = slices.Clone(d.FontPaths)
	global.UseSystemFonts = d.UseSystemFonts
	if d.Network != nil {
		load.ApplyNetworkPolicy(&global.Load, *d.Network)
	}

	if d.Width != 0 {
		image.Width = d.Width
	}
	if d.Height != 0 {
		image.Height = d.Height
	}
	if d.Padding != 0 {
		image.Padding = d.Padding
	}
	if d.Quality != 0 {
		image.Quality = d.Quality
	}
	if d.Format != "" {
		image.Format = d.Format
	}
	if d.SmartWidth != nil {
		image.SmartWidth = *d.SmartWidth
	}
	image.Transparent = d.Transparent
	if d.Crop != nil {
		image.Crop = settings.CropSettings{
			Left:   d.Crop.Left,
			Top:    d.Crop.Top,
			Width:  d.Crop.Width,
			Height: d.Crop.Height,
		}
	}

	object := settings.DefaultPdfObject()
	mapContent(&object, d.Source)
	if d.Zoom != 0 {
		object.Load.ZoomFactor = d.Zoom
	}
	if d.AllowLocalFiles {
		object.Load.BlockLocalFileAccess = false
	}

	req := imageout.NewRequest(global, image, []settings.PdfObject{object}, output)
	req.Now = d.Now

	return req
}

func mapContent(object *settings.PdfObject, content Content) {
	switch {
	case content.HTML != nil:
		object.Page = ""
		object.Load.InlineHTML = slices.Clone(content.HTML)
		object.Load.InlineBase = content.Base
	case content.File != "":
		object.Page = content.File
	case content.URL != "":
		object.Page = content.URL
	}
}
