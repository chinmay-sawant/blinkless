"""ctypes loader for libblinkless and the C ABI structs.

Search order for the shared library:

1. ``BLINKLESS_LIBRARY_PATH`` environment variable (exact path).
2. ``libblinkless.{so,dylib,dll}`` next to this package (wheel layout).
3. ``<repo root>/dist/libblinkless.<ext>`` for in-tree builds.

The first existing candidate wins; when none exists ``find_library_path``
raises ``FileNotFoundError`` listing every path tried.

The engine returns a drawing list and no longer writes PDF or encodes
images. ``convert_html_to_pdf`` and ``convert_html_to_image`` raise
``RuntimeError`` with the removal reason before any foreign call.

Memory ownership follows the committed header
``bindings/c/include/blinkless.h``: strings returned by the library are
allocated by the library and must be released through
``blinkless_free_string`` after Python copies them. Input buffers are
borrowed for the duration of a call only.

Engine calls are not documented as thread-affine, so every foreign call is
serialized through a module-wide lock. ctypes releases the GIL around each
CDLL call.
"""

import ctypes
import os
import sys
import threading
from pathlib import Path

#: ABI revision this binding is compiled against. Must match the header.
ABI_VERSION = 1

_LOAD_LOCK = threading.Lock()
_CALL_LOCK = threading.Lock()

_LOADED_LIBRARY = None  # type: ctypes.CDLL


class GwkImageOptions(ctypes.Structure):
    """Mirror of GwkImageOptions from include/blinkless.h."""

    _fields_ = [
        ("abi_version", ctypes.c_int32),
        ("struct_size", ctypes.c_int32),
        ("format", ctypes.c_char_p),
        ("base_url", ctypes.c_char_p),
        ("allow", ctypes.POINTER(ctypes.c_char_p)),
        ("allow_len", ctypes.c_size_t),
        ("width", ctypes.c_int32),
        ("height", ctypes.c_int32),
        ("quality", ctypes.c_int32),
        ("smart_width", ctypes.c_int32),
        ("transparent", ctypes.c_int32),
        ("crop_left", ctypes.c_int32),
        ("crop_top", ctypes.c_int32),
        ("crop_width", ctypes.c_int32),
        ("crop_height", ctypes.c_int32),
        ("zoom", ctypes.c_double),
        ("enable_local_file_access", ctypes.c_int32),
        ("network_policy", ctypes.c_int32),
        ("timeout_ms", ctypes.c_int32),
    ]

    @classmethod
    def create(cls):
        # type: () -> GwkImageOptions
        """Return a zeroed struct with the size gate fields filled."""
        instance = cls()
        instance.abi_version = ABI_VERSION
        instance.struct_size = ctypes.sizeof(cls)
        return instance


def _library_filename():
    # type: () -> str
    if sys.platform == "darwin":
        return "libblinkless.dylib"
    if sys.platform == "win32":
        return "libblinkless.dll"
    return "libblinkless.so"


def candidate_paths():
    # type: () -> list
    """Return the loader's search candidates, highest priority first."""
    paths = []
    env_path = os.environ.get("BLINKLESS_LIBRARY_PATH")
    if env_path:
        paths.append(Path(env_path))
    filename = _library_filename()
    package_dir = Path(__file__).resolve().parent
    paths.append(package_dir / filename)
    try:
        # _lib.py sits at <root>/bindings/python/src/blinkless/, so
        # parents[4] is the repository root.
        repo_root = Path(__file__).resolve().parents[4]
        paths.append(repo_root / "dist" / filename)
    except IndexError:  # installed outside any repo-like tree
        pass
    return paths


def find_library_path():
    # type: () -> Path
    """Return the first existing shared-library candidate.

    Raises:
        FileNotFoundError: When no candidate exists on disk.
    """
    tried = []
    for path in candidate_paths():
        tried.append(str(path))
        if path.is_file():
            return path
    raise FileNotFoundError(
        "libblinkless not found; build it with"
        " 'CGO_ENABLED=1 go build -buildmode=c-shared -o dist/{0} ./bindings/c'"
        " or set BLINKLESS_LIBRARY_PATH. Tried: {1}".format(
            _library_filename(), ", ".join(tried)
        )
    )


def _bind_prototypes(lib):
    # type: (ctypes.CDLL) -> None
    ubyte_pp = ctypes.POINTER(ctypes.POINTER(ctypes.c_ubyte))
    size_p = ctypes.POINTER(ctypes.c_size_t)
    char_pp = ctypes.POINTER(ctypes.c_char_p)

    fn = lib.blinkless_html_to_image
    fn.restype = ctypes.c_int
    fn.argtypes = [
        ctypes.c_char_p,
        ctypes.c_size_t,
        ctypes.POINTER(GwkImageOptions),
        ubyte_pp,
        size_p,
        char_pp,
    ]

    fn = lib.blinkless_free
    fn.restype = None
    fn.argtypes = [ctypes.c_void_p]

    fn = lib.blinkless_free_string
    fn.restype = None
    fn.argtypes = [ctypes.c_char_p]

    fn = lib.blinkless_abi_version
    fn.restype = ctypes.c_int32
    fn.argtypes = []

    # Declared c_void_p instead of c_char_p so the raw pointer survives;
    # the header requires releasing it with blinkless_free_string.
    fn = lib.blinkless_version
    fn.restype = ctypes.c_void_p
    fn.argtypes = []

    fn = lib.blinkless_last_error_length
    fn.restype = ctypes.c_int32
    fn.argtypes = []

    fn = lib.blinkless_last_error
    fn.restype = ctypes.c_int32
    fn.argtypes = [ctypes.c_char_p, ctypes.c_int32]


def load_library():
    # type: () -> ctypes.CDLL
    """Load, prototype-bind, and ABI-check the shared library once.

    Raises:
        ImportError: When the library is missing or reports a foreign ABI.
    """
    global _LOADED_LIBRARY
    with _LOAD_LOCK:
        if _LOADED_LIBRARY is not None:
            return _LOADED_LIBRARY
        path = find_library_path()
        # CDLL uses RTLD_LOCAL by default, keeping Go runtime symbols out
        # of the global namespace.
        lib = ctypes.CDLL(str(path))
        _bind_prototypes(lib)
        reported = int(lib.blinkless_abi_version())
        if reported != ABI_VERSION:
            raise ImportError(
                "ABI mismatch: library {0}, binding expects {1}".format(
                    reported, ABI_VERSION
                )
            )
        _LOADED_LIBRARY = lib
        return _LOADED_LIBRARY


def abi_version():
    # type: () -> int
    """Return the ABI revision reported by the loaded library."""
    return int(load_library().blinkless_abi_version())


def library_version_string():
    # type: () -> str
    """Return the runtime version string, freeing the library allocation."""
    lib = load_library()
    ptr = 0
    try:
        with _CALL_LOCK:
            ptr = int(lib.blinkless_version() or 0)
        if not ptr:
            return ""
        text = ctypes.cast(ptr, ctypes.c_char_p).value
        return (text or b"").decode("utf-8", "replace")
    finally:
        if ptr:
            lib.blinkless_free_string(ctypes.cast(ptr, ctypes.c_char_p))


def convert_html_to_pdf(html, opts=None):
    # type: (bytes, object) -> bytes
    """PDF writing was removed. The engine returns a drawing list."""
    raise RuntimeError(
        "PDF writing was removed; the engine returns a drawing list"
    )


def convert_html_to_image(html, opts=None):
    # type: (bytes, GwkImageOptions) -> bytes
    """Image encoding was removed. The engine returns a drawing list."""
    raise RuntimeError(
        "image encoding was removed; the engine returns a drawing list"
    )
