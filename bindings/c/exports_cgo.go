//go:build cgo

// cgo facade for libblinkless. This file owns every C pointer
// interaction: it validates the ABI gate, copies borrowed buffers into Go
// memory, drives the shared run hooks, and hands results back under the
// ownership rules of include/blinkless.h.
package main

/*
// The struct layouts below mirror include/blinkless.h. Including the
// committed header directly is not possible here: cgo regenerates
// prototypes for exported functions without const qualifiers, which
// conflicts with the header's const declarations. The runtime abi_version
// and struct_size gate keeps any layout drift between this preamble and
// the authoritative header detectable at call time.

#include <stdint.h>
#include <stdlib.h>

typedef struct {
    int32_t abi_version;
    int32_t struct_size;
    const char* page_size;
    const char* orientation;
    const char* title;
    const char* pdf_version;
    const char* pdf_profile;
    const char* base_url;
    const char* const* allow;
    size_t allow_len;
    double width_mm;
    double height_mm;
    double margin_top;
    double margin_right;
    double margin_bottom;
    double margin_left;
    int32_t copies;
    int32_t grayscale;
    int32_t enable_local_file_access;
    int32_t network_policy;
    int32_t timeout_ms;
} GwkPdfOptions;

typedef struct {
    int32_t abi_version;
    int32_t struct_size;
    const char* format;
    const char* base_url;
    const char* const* allow;
    size_t allow_len;
    int32_t width;
    int32_t height;
    int32_t quality;
    int32_t smart_width;
    int32_t transparent;
    int32_t crop_left;
    int32_t crop_top;
    int32_t crop_width;
    int32_t crop_height;
    double zoom;
    int32_t enable_local_file_access;
    int32_t network_policy;
    int32_t timeout_ms;
} GwkImageOptions;
*/
import "C"

import (
	"context"
	"fmt"
	"math"
	"time"
	"unsafe"
)

// networkPolicyRestricted selects blinkless.RestrictedNetworkPolicy when
// set in either options struct; 0 keeps the compatible default policy.
const networkPolicyRestricted = 1

//export blinkless_abi_version
func blinkless_abi_version() C.int32_t {
	return C.int32_t(abiVersionValue)
}

//export blinkless_version
func blinkless_version() *C.char {
	return C.CString(libVersion)
}

//export blinkless_html_to_image
func blinkless_html_to_image(
	cHTML *C.char,
	cLen C.size_t,
	cOpts *C.GwkImageOptions,
	cOutData **C.uchar,
	cOutLen *C.size_t,
	cErr **C.char,
) C.int {
	html, opts, rejected := parseImageRequest(cHTML, cLen, cOpts, cErr)
	if rejected {
		return C.int(statusInvalidArg)
	}

	ctx, cancel := requestContext(opts.timeoutMS)
	defer cancel()

	status, data, message := runImageWithContext(ctx, html, opts)

	return finishResult(status, data, message, cOutData, cOutLen, cErr)
}

//export blinkless_free
func blinkless_free(p unsafe.Pointer) {
	C.free(p)
}

//export blinkless_free_string
func blinkless_free_string(s *C.char) {
	C.free(unsafe.Pointer(s))
}

//export blinkless_last_error_length
func blinkless_last_error_length() C.int32_t {
	return C.int32_t(lastErrorLength())
}

//export blinkless_last_error
func blinkless_last_error(buf *C.char, bufLen C.int32_t) C.int32_t {
	if buf == nil || bufLen <= 0 {
		return 0
	}

	sink := unsafe.Slice((*byte)(unsafe.Pointer(buf)), int(bufLen))

	return C.int32_t(copyLastErrorInto(sink))
}

// runImageWithContext rejects image encoding. The engine returns a drawing
// list from the Go layout package. This ABI entry no longer writes PNG or JPEG.
func runImageWithContext(ctx context.Context, html []byte, opts imageOptions) (int32, []byte, string) {
	if message, ok := validateImageRange(opts); !ok {
		return statusInvalidArg, nil, message
	}

	if ctx == nil || len(html) == 0 {
		return statusInvalidArg, nil, "html is required"
	}

	return statusRenderError, nil, "image encoding was removed; the engine returns a drawing list"
}

// requestContext builds the conversion context. Positive timeout values
// install the documented millisecond deadline; every other value disables it.
func requestContext(timeoutMS int64) (context.Context, context.CancelFunc) {
	if timeoutMS <= 0 {
		return context.Background(), func() {}
	}

	return context.WithTimeout(context.Background(), time.Duration(timeoutMS)*time.Millisecond)
}

// parseImageRequest validates the borrowed HTML buffer and GwkImageOptions.
func parseImageRequest(
	cHTML *C.char,
	cLen C.size_t,
	cOpts *C.GwkImageOptions,
	cErr **C.char,
) ([]byte, imageOptions, bool) {
	var empty imageOptions

	if cHTML == nil || cLen == 0 {
		rejectRequest("html buffer is nil or empty", cErr)

		return nil, empty, true
	}
	if int64(cLen) > int64(math.MaxInt32) {
		message := fmt.Sprintf("html length %d exceeds the %d byte limit",
			int64(cLen), int64(math.MaxInt32))
		rejectRequest(message, cErr)

		return nil, empty, true
	}

	opts, ok := convertImageOptions(cOpts, cErr)
	if !ok {
		return nil, empty, true
	}

	return C.GoBytes(unsafe.Pointer(cHTML), C.int(cLen)), opts, false
}

// convertImageOptions maps a GwkImageOptions pointer onto imageOptions and
// applies the ABI gate.
func convertImageOptions(cOpts *C.GwkImageOptions, cErr **C.char) (imageOptions, bool) {
	opts := defaultImageOptions()
	if cOpts == nil {
		return opts, true
	}

	if int32(cOpts.abi_version) != abiVersionValue {
		rejectRequest(abiGateMessage(int64(cOpts.abi_version)), cErr)

		return opts, false
	}
	if cOpts.struct_size != 0 && int64(cOpts.struct_size) != int64(C.sizeof_GwkImageOptions) {
		rejectRequest(structSizeMessage("GwkImageOptions",
			int64(cOpts.struct_size), int64(C.sizeof_GwkImageOptions)), cErr)

		return opts, false
	}

	if cOpts.format != nil {
		opts.format = C.GoString(cOpts.format)
	}
	if cOpts.base_url != nil {
		opts.baseURL = C.GoString(cOpts.base_url)
	}

	allow, ok := convertAllowList(cOpts.allow, cOpts.allow_len)
	if !ok {
		rejectRequest(allowListMessage(int64(cOpts.allow_len)), cErr)

		return opts, false
	}
	opts.allow = allow

	opts.width = int(cOpts.width)
	opts.height = int(cOpts.height)
	opts.quality = int(cOpts.quality)
	opts.smartWidth = int(cOpts.smart_width)
	opts.transparent = cOpts.transparent != 0
	opts.cropLeft = int(cOpts.crop_left)
	opts.cropTop = int(cOpts.crop_top)
	opts.cropWidth = int(cOpts.crop_width)
	opts.cropHeight = int(cOpts.crop_height)
	opts.zoom = float64(cOpts.zoom)
	opts.localFiles = cOpts.enable_local_file_access != 0
	opts.restricted = int(cOpts.network_policy) == networkPolicyRestricted
	opts.timeoutMS = int64(cOpts.timeout_ms)

	return opts, true
}

// maxAllowEntries is defined in options_image.go and caps caller-controlled
// allow array lengths before an unsafe slice is formed. The allow array is
// borrowed from the caller, so an unbounded length would read out of bounds
// and crash the process from inside an exported call.

// convertAllowList copies allow_len entries from the borrowed allow array.
// Nil entries are skipped because the header lets callers leave slots empty.
// The bool is false when allow_len exceeds maxAllowEntries; no slice is
// formed in that case.
func convertAllowList(allow **C.char, allowLen C.size_t) ([]string, bool) {
	if allow == nil || allowLen == 0 {
		return nil, true
	}
	if allowLen > maxAllowEntries {
		return nil, false
	}

	entries := unsafe.Slice(allow, int(allowLen))
	copied := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry != nil {
			copied = append(copied, C.GoString(entry))
		}
	}

	if len(copied) == 0 {
		return nil, true
	}

	return copied, true
}

// probeAllowListTooLong reports whether convertAllowList rejects a length
// above the cap without touching the array. The pointer is a single dummy
// slot, so any read past the guard would be out of bounds. Test-only helper:
// cgo is not supported in test files, so C pointer work stays here.
func probeAllowListTooLong() ([]string, bool) {
	var slot *C.char

	return convertAllowList(&slot, C.size_t(maxAllowEntries+1))
}

// probeAllowListCopies drives convertAllowList over a C array holding the
// given strings, leaving nil entries where nilAt marks the index. The
// borrowed C memory is released before returning.
func probeAllowListCopies(values []string, nilAt map[int]bool) ([]string, bool) {
	cValues := make([]*C.char, len(values))
	defer func() {
		for _, p := range cValues {
			if p != nil {
				C.free(unsafe.Pointer(p))
			}
		}
	}()

	for i, v := range values {
		if nilAt[i] {
			continue
		}
		cValues[i] = C.CString(v)
	}

	return convertAllowList(&cValues[0], C.size_t(len(cValues)))
}

// probeFinishResult drives finishResult with Go-side flags selecting which
// out parameters are nil, releasing every allocation before returning. The
// returned message is the diagnostic that would land in cErr (empty on
// success).
func probeFinishResult(
	status int32,
	data []byte,
	message string,
	nilOutData, nilOutLen, nilErr bool,
) (int32, int, string) {
	var out *C.uchar
	var outLen C.size_t
	var cErr *C.char

	var outDataP **C.uchar
	var outLenP *C.size_t
	var errP **C.char
	if !nilOutData {
		outDataP = &out
	}
	if !nilOutLen {
		outLenP = &outLen
	}
	if !nilErr {
		errP = &cErr
	}

	got := finishResult(status, data, message, outDataP, outLenP, errP)

	if out != nil {
		blinkless_free(unsafe.Pointer(out))
	}
	if cErr == nil {
		return int32(got), int(outLen), ""
	}

	msg := C.GoString(cErr)
	blinkless_free_string(cErr)

	return int32(got), int(outLen), msg
}

// finishResult writes the success payload or the failure diagnostic into the
// caller-owned out parameters and records diagnostics in the last-error
// slot. Memory conventions follow include/blinkless.h: success stores an
// allocation for blinkless_free with NULL out_err; failure stores NULL
// data, zero length, and an allocation for blinkless_free_string.
func finishResult(
	status int32,
	data []byte,
	message string,
	cOutData **C.uchar,
	cOutLen *C.size_t,
	cErr **C.char,
) C.int {
	if cOutData == nil || cOutLen == nil || cErr == nil {
		rejectRequest("nil out parameter", cErr)

		return C.int(statusInvalidArg)
	}

	if status == statusOK && len(data) > 0 {
		*cOutData = (*C.uchar)(C.CBytes(data))
		*cOutLen = C.size_t(len(data))
		*cErr = nil

		return C.int(statusOK)
	}

	if status == statusOK {
		status = statusRenderError
		message = "conversion produced no output"
	}
	if message == "" {
		message = "conversion failed"
	}

	*cOutData = nil
	*cOutLen = 0
	*cErr = C.CString(message)
	setLastError(message)

	return C.int(status)
}

// rejectRequest records a validation failure in both the out_err parameter
// and the process-wide last-error slot.
func rejectRequest(message string, cErr **C.char) {
	setLastError(message)
	if cErr != nil {
		*cErr = C.CString(message)
	}
}

// abiGateMessage formats the rejection text for an unsupported abi_version.
func abiGateMessage(got int64) string {
	return fmt.Sprintf("unsupported abi_version %d; expected %d", got, int64(abiVersionValue))
}

// structSizeMessage formats the rejection text for a mismatched struct_size.
func structSizeMessage(kind string, got, want int64) string {
	return fmt.Sprintf("%s struct_size %d does not match expected %d", kind, got, want)
}

// exportedABI returns the result of blinkless_abi_version. It and the
// helpers below let Go-side tests drive the real export signatures; go vet
// rejects cgo inside test files, so every C pointer interaction lives here.
func exportedABI() int32 {
	return int32(blinkless_abi_version())
}

// exportedVersion returns the version string allocated by
// blinkless_version, releasing it through blinkless_free_string.
func exportedVersion() string {
	version := blinkless_version()
	defer blinkless_free_string(version)

	return C.GoString(version)
}

// readLastErrorViaExport copies the process-wide diagnostic through
// blinkless_last_error into C memory and back as a Go string.
func readLastErrorViaExport() string {
	length := blinkless_last_error_length()
	if length <= 0 {
		return ""
	}

	sink := (*C.char)(C.malloc(C.size_t(length) + 1))
	if sink == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(sink))

	written := blinkless_last_error(sink, length+1)
	if written <= 0 {
		return ""
	}

	return C.GoStringN(sink, C.int(written))
}
