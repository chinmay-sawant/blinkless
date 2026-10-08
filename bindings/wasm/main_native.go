//go:build !js || !wasm

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

// main is the native entry point of the contract package. The browser build
// exports blinklessWASM; this native build produces the same versioned
// drawing-list JSON from a fixture file so the consumer check and tests can
// compare a browser result with a native one:
//
//	go run ./bindings/wasm -fixture testdata/wasm/sample.html -width 640 -height 480 -out result.json
func main() {
	fixture := flag.String("fixture", "", "path to an HTML fixture file")
	out := flag.String("out", "", "write the drawing-list JSON here instead of stdout")
	width := flag.Int("width", 0, "viewport width in CSS pixels (0 uses the fallback)")
	height := flag.Int("height", 0, "viewport height in CSS pixels (0 uses the fallback)")
	flag.Parse()

	if err := run(*fixture, *out, *width, *height); err != nil {
		fmt.Fprintln(os.Stderr, "wasm-native:", err)
		os.Exit(1)
	}
}

func run(fixture, out string, width, height int) error {
	if fixture == "" {
		return fmt.Errorf("%w: -fixture is required", errInvalidRequest)
	}

	raw, err := os.ReadFile(fixture)
	if err != nil {
		return fmt.Errorf("read fixture: %w", err)
	}

	result, err := Convert(context.Background(), Request{HTML: string(raw), Width: width, Height: height}, nil)
	if err != nil {
		return err
	}

	if out == "" {
		_, err = os.Stdout.Write(result.Bytes)

		return err
	}

	return os.WriteFile(out, result.Bytes, 0o644)
}
