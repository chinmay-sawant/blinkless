"""blinkless: in-process Python bindings for the blinkless engine.

The engine lays out HTML and returns a drawing list. It writes no PDF and
encodes no image, so the conversion helpers raise ``RuntimeError`` with
the removal reason. What remains is the ctypes loader, the ABI and
version queries, the error taxonomy, and the document models:

    from blinkless import abi_version, library_version_string

    assert abi_version() == 1
    print(library_version_string())

The shared library is located and loaded only when an ABI query runs;
building model objects never touches it.
"""

from .exceptions import (
    ConversionError,
    ConversionTimeoutError,
    ErrEmptyContent,
    ErrInvalidContent,
    ErrInvalidOrientation,
    ErrInvalidPDFProfile,
    ErrInvalidPageSize,
    ErrInvalidPDFVersion,
    ErrMissingOutput,
    ErrNoPageObjects,
    GowkhtmltopdfError,
    InternalEngineError,
    InvalidArgumentError,
    LoadDeniedError,
    RenderError,
    ResourceLimitError,
    error_from_status,
    sniff_sentinel,
)
from .document import (
    Content,
    Crop,
    Document,
    HeaderFooter,
    ImageDocument,
    ImageOptions,
    Margin,
    NetworkPolicy,
    Page,
    PDFOptions,
    TOC,
    compatible_network_policy,
    restricted_network_policy,
)
from .api import (
    convert_file_to_pdf,
    convert_html_to_image,
    convert_html_to_pdf,
    convert_url_to_pdf,
)

__version__ = "0.2.6"

#: Historical upstream settings-surface identifier, kept for import
#: compatibility. The Go constant was removed with the PDF writer.
library_version = "0.12.7-dev"


def abi_version():
    # type: () -> int
    """Return the ABI revision of the loaded shared library (always 1 today).

    Raises ImportError when the library is missing or built for another ABI.
    """
    from ._lib import abi_version as _abi_version

    return _abi_version()


def library_version_string():
    # type: () -> str
    """Return the runtime version string reported by the shared library."""
    from ._lib import library_version_string as _lvs

    return _lvs()


__all__ = [
    "GowkhtmltopdfError",
    "ConversionError",
    "InvalidArgumentError",
    "LoadDeniedError",
    "RenderError",
    "ConversionTimeoutError",
    "ResourceLimitError",
    "InternalEngineError",
    "ErrEmptyContent",
    "ErrInvalidContent",
    "ErrNoPageObjects",
    "ErrInvalidPageSize",
    "ErrInvalidOrientation",
    "ErrInvalidPDFVersion",
    "ErrInvalidPDFProfile",
    "ErrMissingOutput",
    "error_from_status",
    "sniff_sentinel",
    "Content",
    "Page",
    "Margin",
    "HeaderFooter",
    "TOC",
    "Crop",
    "NetworkPolicy",
    "compatible_network_policy",
    "restricted_network_policy",
    "PDFOptions",
    "ImageOptions",
    "Document",
    "ImageDocument",
    "convert_html_to_pdf",
    "convert_file_to_pdf",
    "convert_url_to_pdf",
    "convert_html_to_image",
    "__version__",
    "library_version",
    "abi_version",
    "library_version_string",
]
