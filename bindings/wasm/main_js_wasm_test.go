//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"
	"testing"
)

// Run with GOOS=js GOARCH=wasm go test -exec=<GOROOT>/lib/wasm/go_js_wasm_exec
// ./bindings/wasm -run TestConvertJSRequestBounds.
func TestConvertJSRequestBounds(t *testing.T) {
	for _, testCase := range []struct {
		name string
		raw  string
		code string
	}{
		{name: "valid primitive string", raw: `{"html":"<p>hello</p>"}`},
		{
			name: "HTML at byte cap with JSON envelope",
			raw:  `{"html":"<!--` + strings.Repeat("x", maxHTMLBytes-7) + `-->"}`,
		},
		{
			name: "request above unit cap",
			raw:  strings.Repeat(" ", maxRequestBytes+1),
			code: "resource_limit",
		},
		{
			name: "multibyte request above byte cap",
			raw:  `{"html":"` + strings.Repeat("é", maxRequestBytes/2+1) + `"}`,
			code: "resource_limit",
		},
		{
			name: "malformed JSON",
			raw:  `{"html":`,
			code: "invalid_request",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			value := convertJS(js.Undefined(), []js.Value{js.ValueOf(testCase.raw)})
			response, ok := value.(js.Value)
			if !ok {
				t.Fatalf("convertJS returned %T, want js.Value", value)
			}
			if testCase.code == "" {
				if !response.Get("ok").Bool() {
					t.Fatalf("convertJS failed: %s", response.Get("error").Get("message").String())
				}

				return
			}
			if response.Get("ok").Bool() {
				t.Fatal("convertJS accepted invalid input")
			}
			if got := response.Get("error").Get("code").String(); got != testCase.code {
				t.Fatalf("error code = %q, want %q", got, testCase.code)
			}
		})
	}
}
