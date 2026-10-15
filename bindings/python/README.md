# blinkless (Python)

In-process Python bindings for the blinkless HTML layout engine. The
package loads `libblinkless` (a Go `-buildmode=c-shared` library) with
stdlib `ctypes`; there is no subprocess and no compiled Python extension.

The engine lays out HTML and returns a drawing list. It writes no PDF and
encodes no image, so every conversion entry point
(`convert_html_to_pdf`, `convert_file_to_pdf`, `convert_url_to_pdf`,
`convert_html_to_image`, `Document.pdf`, `ImageDocument.image`) raises
`RuntimeError` with the removal reason.

## What works today

```python
from blinkless import abi_version, library_version_string

assert abi_version() == 1
print(library_version_string())
```

The ctypes loader, the version and ABI queries, the error taxonomy, and
the document models (validation included) remain in place. The WASM
adapter is the first binding that returns a drawing-list payload; a
drawing-list entry point for the C ABI is the next step for this package.

## Build and install

Requires Go with cgo and a C toolchain:

```sh
CGO_ENABLED=1 make c-shared
pip install -e ./bindings/python
```

`make c-shared` writes `dist/libblinkless.so`, which the loader finds
automatically. Set `BLINKLESS_LIBRARY_PATH` when the library lives
elsewhere.

## Thread safety

`libblinkless` is not documented as thread-affine and keeps process-wide
state, including the last-error slot. The binding therefore serializes
access with two module-level locks defined in `blinkless._lib`:

- `_LOAD_LOCK` guards library discovery, dlopen, prototype binding, and
  the ABI check inside `load_library()`. It protects the
  `_LOADED_LIBRARY` slot only.
- `_CALL_LOCK` serializes the wrapper's foreign calls, including the
  initial ABI check, version queries, and string releases.

ctypes releases the GIL around each CDLL call, so holding `_CALL_LOCK`
does not block other Python threads from running pure Python. When the
wrapper needs both locks, it takes `_LOAD_LOCK` first. Do not hold either
private lock when calling a wrapper function. Calls made directly through
the CDLL returned by `load_library()` need caller-managed serialization.
