package blinkless

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
)

var (
	// ErrNilDocument reports a method call on a nil Document receiver.
	ErrNilDocument = errors.New("blinkless: nil document")
	// ErrNilImageDocument reports a method call on a nil ImageDocument receiver.
	ErrNilImageDocument = errors.New("blinkless: nil image document")
	// ErrInvalidContent reports a Content value with an invalid source shape.
	ErrInvalidContent = errors.New("blinkless: invalid content")
	// ErrInvalidImageFormat reports an unsupported ImageDocument format.
	ErrInvalidImageFormat = errors.New("blinkless: invalid image format")
	// ErrEmptyContent is a content-oriented alias for the legacy HTML
	// sentinel. Empty HTML remains matchable through errors.Is.
	ErrEmptyContent = ErrEmptyHTML
	// ErrInvalidOrientation identifies an unsupported Document orientation.
	ErrInvalidOrientation = errors.New("blinkless: invalid orientation")
	// ErrInvalidImageQuality reports an image quality outside 0 to 100.
	ErrInvalidImageQuality = errors.New("blinkless: image quality must be between 0 and 100")
	// ErrInvalidCrop reports negative crop dimensions or offsets.
	ErrInvalidCrop = errors.New("blinkless: crop dimensions and offsets must be non-negative")
	// ErrInvalidDimensions reports incomplete, negative, or non-finite page or image dimensions.
	ErrInvalidDimensions = errors.New("blinkless: invalid dimensions")
	// ErrInvalidMargin reports non-finite margins or a negative left/right
	// margin. Negative top/bottom are the engine's auto header/footer
	// sentinel and pass validation.
	ErrInvalidMargin = errors.New("blinkless: invalid margin")
	// ErrInvalidZoom reports a negative or non-finite zoom factor.
	ErrInvalidZoom = errors.New("blinkless: invalid zoom")
)

// Validate checks that Content identifies one valid source and that Base is
// used only with in-memory HTML.
func (c Content) Validate() error {
	return c.validate()
}

func (d *ImageDocument) Validate() error {
	if d == nil {
		return ErrNilImageDocument
	}

	if err := d.Source.validate(); err != nil {
		return fmt.Errorf("source: %w", err)
	}

	if err := validateImageDimensions(d.Width, d.Height); err != nil {
		return err
	}

	if d.Padding < 0 {
		return fmt.Errorf("%w: image padding must be non-negative", ErrInvalidDimensions)
	}

	if err := validateZoom(d.Zoom); err != nil {
		return err
	}

	switch format := strings.ToLower(strings.TrimSpace(d.Format)); format {
	case "", "png", "jpg", "jpeg":
	default:
		return fmt.Errorf("%w: %q", ErrInvalidImageFormat, d.Format)
	}

	if d.Quality < 0 || d.Quality > 100 {
		return fmt.Errorf("%w: got %d", ErrInvalidImageQuality, d.Quality)
	}

	return validateImageCrop(d.Crop)
}

func validateImageCrop(crop *Crop) error {
	if crop == nil {
		return nil
	}

	if crop.Width < 0 || crop.Height < 0 || crop.Left < 0 || crop.Top < 0 {
		return ErrInvalidCrop
	}

	return nil
}

func validateImageDimensions(width, height int) error {
	if width < 0 || height < 0 {
		return fmt.Errorf("%w: image width and height must be non-negative", ErrInvalidDimensions)
	}

	return nil
}

func validateZoom(zoom float64) error {
	if zoom == 0 {
		return nil
	}

	if !finitePositive(zoom) {
		return fmt.Errorf("%w: zoom must be finite and greater than zero", ErrInvalidZoom)
	}

	return nil
}

func finitePositive(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value > 0
}

//nolint:cyclop,wsl // exact-one-source validation has one branch per source kind.
func (c Content) validate() error {
	sources := 0
	if c.HTML != nil {
		sources++
	}
	if strings.TrimSpace(c.File) != "" {
		sources++
	}
	if strings.TrimSpace(c.URL) != "" {
		sources++
	}

	switch {
	case sources == 0:
		return fmt.Errorf("%w: %w", ErrInvalidContent, ErrEmptyHTML)
	case sources > 1:
		return fmt.Errorf("%w: exactly one of HTML, File, or URL is required", ErrInvalidContent)
	}

	if c.HTML != nil {
		if len(c.HTML) == 0 {
			return fmt.Errorf("%w: %w", ErrInvalidContent, ErrEmptyHTML)
		}

		return nil
	}

	if c.Base != "" {
		return fmt.Errorf("%w: Base is only valid with HTML", ErrInvalidContent)
	}
	if c.URL == "" {
		return nil
	}

	parsed, err := url.Parse(c.URL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%w: URL must be an absolute HTTP(S) URL", ErrInvalidContent)
	}

	return nil
}
