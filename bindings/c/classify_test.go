package main

import "testing"

func TestZeroImageOptionsUseUnsetMarkers(t *testing.T) {
	t.Parallel()

	opts := defaultImageOptions()
	if opts.smartWidth != smartWidthUnset {
		t.Fatalf("smartWidth = %d, want unset marker %d", opts.smartWidth, smartWidthUnset)
	}
	if anyCropSet(opts) {
		t.Fatal("zero image options unexpectedly selected a crop")
	}
}
