"""Integration tests against libblinkless.

Every test skips when the shared library is absent so the suite stays
green on machines that have not built bindings/c yet. Build it with:

    CGO_ENABLED=1 go build -buildmode=c-shared \
        -o dist/libblinkless.so ./bindings/c

The engine returns a drawing list. It writes no PDF and encodes no image,
so the binding loads, reports the ABI and version, and raises a removal
error from every conversion entry point. These tests pin that contract.
"""

import ctypes
import unittest

import blinkless
from blinkless import (
    Content,
    Document,
    ImageDocument,
    Page,
    convert_file_to_pdf,
    convert_html_to_image,
    convert_html_to_pdf,
)
from blinkless import _lib


def _find_library():
    try:
        return _lib.find_library_path()
    except Exception:
        return None


_LIB_PATH = _find_library()

_REASON = (
    "libblinkless not found; build with"
    " 'CGO_ENABLED=1 go build -buildmode=c-shared"
    " -o dist/libblinkless.so ./bindings/c' or set"
    " BLINKLESS_LIBRARY_PATH"
)

_INLINE_HTML = (
    b"<html><body><h1>Invoice #42</h1><p>Total: $19.00</p></body></html>"
)

# Status 3 from the header table: layout or encoding failed. The image
# entry point reports every call this way because encoding was removed.
_STATUS_RENDER_ERROR = 3


@unittest.skipUnless(_LIB_PATH is not None, _REASON)
class BindingTest(unittest.TestCase):
    def test_library_reports_abi_and_version(self):
        lib = _lib.load_library()
        self.assertIsNotNone(lib)
        self.assertEqual(blinkless.abi_version(), 1)
        reported = blinkless.library_version_string()
        self.assertIsInstance(reported, str)
        self.assertGreater(len(reported), 0)

    def test_pdf_entry_points_report_removed(self):
        with self.assertRaisesRegex(RuntimeError, "removed"):
            convert_html_to_pdf(_INLINE_HTML)
        with self.assertRaisesRegex(RuntimeError, "removed"):
            convert_file_to_pdf("invoice.html")
        with self.assertRaisesRegex(RuntimeError, "removed"):
            Document(
                pages=[Page(source=Content(html=_INLINE_HTML))],
                page_size="A4",
            ).pdf()

    def test_image_entry_points_report_removed(self):
        with self.assertRaisesRegex(RuntimeError, "removed"):
            convert_html_to_image(_INLINE_HTML)
        with self.assertRaisesRegex(RuntimeError, "removed"):
            ImageDocument(source=Content(html=_INLINE_HTML)).image()

    def test_image_abi_call_reports_removed(self):
        lib = _lib.load_library()
        opts = _lib.GwkImageOptions.create()
        out_data = ctypes.POINTER(ctypes.c_ubyte)()
        out_len = ctypes.c_size_t(0)
        out_err = ctypes.c_char_p()
        status = lib.blinkless_html_to_image(
            _INLINE_HTML,
            len(_INLINE_HTML),
            ctypes.byref(opts),
            ctypes.byref(out_data),
            ctypes.byref(out_len),
            ctypes.byref(out_err),
        )
        try:
            self.assertEqual(status, _STATUS_RENDER_ERROR)
            self.assertIn(b"removed", out_err.value)
            self.assertEqual(out_len.value, 0)
            self.assertFalse(out_data)
        finally:
            lib.blinkless_free_string(out_err)


if __name__ == "__main__":
    unittest.main()
