//go:build js && wasm

package main

// wasmVersion is stamped into the browser response by the Makefile target
// (-ldflags -X main.wasmVersion=...). Only the js/wasm entry point reads it.
var wasmVersion = "dev"
