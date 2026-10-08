package main

import (
	"testing"

	blinkless "github.com/chinmay-sawant/blinkless"
)

func TestImageValidationErrorsAreInvalidArguments(t *testing.T) {
	t.Parallel()

	for _, err := range []error{
		blinkless.ErrInvalidImageQuality,
		blinkless.ErrInvalidCrop,
	} {
		if got := classifyError(err, t.Context()); got != statusInvalidArg {
			t.Errorf("classifyError(%v) = %d, want %d", err, got, statusInvalidArg)
		}
	}
}

func TestZeroImageOptionsUseUnsetMarkers(t *testing.T) {
	t.Parallel()

	opts := defaultImageOptions()
	if opts.smartWidth != smartWidthUnset {
		t.Fatalf("smartWidth = %d, want unset marker %d", opts.smartWidth, smartWidthUnset)
	}
	if anyCropSet(opts) {
		t.Fatal("zero image options unexpectedly selected a crop")
	}
	if doc := buildImageDocument([]byte("<p>image</p>"), opts); doc.SmartWidth != nil || doc.Crop != nil {
		t.Fatalf("default image document overrides engine defaults: SmartWidth=%v Crop=%v", doc.SmartWidth, doc.Crop)
	}
}
