package main

import "fmt"

const (
	// smartWidthUnset leaves SmartWidth at the engine default (enabled).
	smartWidthUnset = -1
	// cropUnset marks "no crop on this axis" per the header contract.
	cropUnset = -1
	// maxAllowEntries caps caller-controlled allow array lengths before an
	// unsafe slice is formed.
	maxAllowEntries = 1024
)

// imageOptions mirrors GwkImageOptions with Go-native types. The C adapter
// converts zero-initialized structs to the explicit unset markers below before
// this value is checked before the call is rejected. Image encoding is gone.
type imageOptions struct {
	format      string
	baseURL     string
	allow       []string
	width       int
	height      int
	quality     int
	smartWidth  int
	transparent bool
	cropLeft    int
	cropTop     int
	cropWidth   int
	cropHeight  int
	zoom        float64
	localFiles  bool
	restricted  bool
	timeoutMS   int64
}

func defaultImageOptions() imageOptions {
	return imageOptions{
		smartWidth: smartWidthUnset,
		cropLeft:   cropUnset,
		cropTop:    cropUnset,
		cropWidth:  cropUnset,
		cropHeight: cropUnset,
	}
}

// anyCropSet reports whether at least one crop axis overrides the unset
// marker; untouched axes keep their marker value.
func anyCropSet(opts imageOptions) bool {
	return opts.cropLeft != cropUnset || opts.cropTop != cropUnset ||
		opts.cropWidth != cropUnset || opts.cropHeight != cropUnset
}

// validateImageRange enforces the option ranges the header classifies as
// INVALID_ARG before any engine work starts. Quality 0 selects the engine
// default and 1 through 100 is accepted. Crop axes use -1 (cropUnset) for
// "no crop on this axis"; any other negative value is rejected. The string
// result is the diagnostic message; false means the request was rejected.
func validateImageRange(opts imageOptions) (string, bool) {
	if opts.quality < 0 || opts.quality > 100 {
		return fmt.Sprintf("quality %d outside supported range 0 through 100", opts.quality), false
	}
	if opts.width < 0 || opts.height < 0 {
		return fmt.Sprintf("image dimensions %dx%d must be non-negative", opts.width, opts.height), false
	}
	if opts.cropLeft != cropUnset && opts.cropLeft < 0 {
		return fmt.Sprintf("crop left %d must be non-negative", opts.cropLeft), false
	}
	if opts.cropTop != cropUnset && opts.cropTop < 0 {
		return fmt.Sprintf("crop top %d must be non-negative", opts.cropTop), false
	}
	if opts.cropWidth != cropUnset && opts.cropWidth < 0 {
		return fmt.Sprintf("crop width %d must be non-negative", opts.cropWidth), false
	}
	if opts.cropHeight != cropUnset && opts.cropHeight < 0 {
		return fmt.Sprintf("crop height %d must be non-negative", opts.cropHeight), false
	}
	if len(opts.allow) > maxAllowEntries {
		return allowListMessage(int64(len(opts.allow))), false
	}
	return "", true
}

// allowListMessage formats the rejection text for an oversized allow array.
func allowListMessage(got int64) string {
	return fmt.Sprintf("allow list length %d exceeds the %d entry limit",
		got, int64(maxAllowEntries))
}
