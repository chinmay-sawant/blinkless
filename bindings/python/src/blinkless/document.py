"""Python models for the historical Document / ImageDocument API.

Field names use snake_case parity with the Go types they mirrored
(``PageSize`` -> ``page_size``, ``AllowLocalFiles`` ->
``allow_local_files``, and so on). The models and their validation stay
for API compatibility; the engine returns a drawing list, so the render
methods report that PDF writing and image encoding were removed.
"""

import ctypes
import dataclasses
import urllib.parse
from typing import List, Optional, Sequence

from .exceptions import (
    ErrEmptyContent,
    ErrInvalidContent,
    ErrInvalidOrientation,
    ErrInvalidPDFVersion,
    ErrInvalidPageSize,
    ErrNoPageObjects,
    InvalidArgumentError,
)

#: Upper bound for Document.copies, matching Go MaxDocumentCopies (api.go).
MAX_DOCUMENT_COPIES = 1000

#: Page size names accepted by the engine (internal/settings/pagesize.go).
PAGE_SIZES = frozenset(
    [
        "a0", "a1", "a2", "a3", "a4", "a5", "a6",
        "b0", "b1", "b2", "b3", "b4", "b5", "b6",
        "c5e", "comm10e", "dle", "executive", "folio",
        "ledger", "legal", "letter", "tabloid",
    ]
)

_ORIENTATIONS = ("portrait", "landscape")
_PDF_VERSIONS = ("1.4", "1.7", "2.0")


@dataclasses.dataclass
class Content:
    """One document source. Exactly one of html, file, url must be set.

    ``html`` accepts str or bytes; bytes are passed through unchanged and
    str is encoded UTF-8 when serialized. ``base`` is only valid together
    with ``html``.
    """

    html: object = None  # str or bytes or None
    base: Optional[str] = None
    file: Optional[str] = None
    url: Optional[str] = None

    def validate(self):
        # type: () -> None
        """Raise a sentinel error unless exactly one source is well-formed."""
        sources = sum(
            [
                self.html is not None,
                bool((self.file or "").strip()),
                bool((self.url or "").strip()),
            ]
        )
        if sources == 0:
            raise ErrEmptyContent(
                "content has no HTML, file, or URL source"
            )
        if sources > 1:
            raise ErrInvalidContent(
                "exactly one of HTML, File, or URL is required"
            )
        if self.html is not None:
            if not self.html_bytes:
                raise ErrEmptyContent("HTML content is empty")
            return
        if (self.base or "").strip():
            raise ErrInvalidContent("base is only valid with HTML source")
        if not (self.url or "").strip():
            return
        parsed = urllib.parse.urlparse(self.url)
        if (
            parsed.scheme not in ("http", "https")
            or not parsed.netloc
        ):
            raise ErrInvalidContent("URL must be an absolute HTTP(S) URL")

    @property
    def html_bytes(self):
        # type: () -> bytes
        """Return the HTML source as bytes (UTF-8 encoding for str)."""
        if isinstance(self.html, str):
            return self.html.encode("utf-8")
        if isinstance(self.html, (bytes, bytearray)):
            return bytes(self.html)
        return b""

    @property
    def kind(self):
        # type: () -> str
        """Return which single source is set: html, file, url, or none."""
        if self.html is not None:
            return "html"
        if (self.file or "").strip():
            return "file"
        if (self.url or "").strip():
            return "url"
        return "none"

    @classmethod
    def from_html(cls, html, base=None):
        # type: (object, Optional[str]) -> Content
        """Build an inline-HTML source, mirroring Go Content HTML()."""
        return cls(html=html, base=base)

    @classmethod
    def from_file(cls, path):
        # type: (str) -> Content
        """Build a local-file source, mirroring Go Content File()."""
        return cls(file=path)

    @classmethod
    def from_url(cls, url):
        # type: (str) -> Content
        """Build a remote-URL source, mirroring Go Content URL()."""
        return cls(url=url)


@dataclasses.dataclass
class Margin:
    """Page margins in millimeters."""

    top: float = 0.0
    right: float = 0.0
    bottom: float = 0.0
    left: float = 0.0


@dataclasses.dataclass
class HeaderFooter:
    """Header/footer template. Unset fields keep engine defaults.

    Not yet transmitted by the v1 one-shot ABI; accepted for parity.
    """

    left: str = ""
    center: str = ""
    right: str = ""
    font_size: float = 0.0
    font_name: str = ""
    line: bool = False
    spacing: float = 0.0
    html_url: str = ""
    replace: Optional[bool] = None


@dataclasses.dataclass
class TOC:
    """Table-of-contents settings. Not yet transmitted by the v1 ABI."""

    caption: str = ""
    dotted_lines: Optional[bool] = None
    font_scale: float = 0.0
    indentation: str = ""
    forward_links: Optional[bool] = None
    back_links: Optional[bool] = None


@dataclasses.dataclass
class Crop:
    """Image crop rectangle in pixels.

    The ABI uses -1 for "no crop on this axis", so fields default to -1.
    """

    left: int = -1
    top: int = -1
    width: int = -1
    height: int = -1


@dataclasses.dataclass
class NetworkPolicy:
    """HTTP(S) loading policy for document and subresource fetches.

    ``restricted`` marks policies built by ``restricted_network_policy``;
    the serializer maps any restricted marker or block flag to ABI
    network_policy=1.
    """

    allowed_schemes: Optional[Sequence[str]] = None
    allowed_hosts: Optional[Sequence[str]] = None
    block_private_networks: bool = False
    block_cross_host_redirects: bool = False
    restricted: bool = dataclasses.field(default=False, repr=False)


def compatible_network_policy():
    # type: () -> NetworkPolicy
    """Permissive historical policy: HTTP(S) anywhere, redirects allowed."""
    return NetworkPolicy()


def restricted_network_policy():
    # type: () -> NetworkPolicy
    """Policy blocking private destinations and cross-host redirects."""
    return NetworkPolicy(
        block_private_networks=True,
        block_cross_host_redirects=True,
        restricted=True,
    )


@dataclasses.dataclass
class Page:
    """One page of a Document. ``header``/``footer`` None inherits the
    document-level value."""

    source: Content = None  # type: ignore
    header: Optional[HeaderFooter] = None
    footer: Optional[HeaderFooter] = None
    include_in_outline: Optional[bool] = None
    external_links: Optional[bool] = None
    local_links: Optional[bool] = None
    zoom: float = 0.0


@dataclasses.dataclass
class PDFOptions:
    """Flat option bag for the historical convert_html_to_pdf helper."""

    page_size: str = "A4"
    orientation: str = "portrait"
    width_mm: float = 0.0
    height_mm: float = 0.0
    margin: Optional[Margin] = None
    title: str = ""
    pdf_version: str = ""
    pdf_profile: str = ""
    copies: int = 0
    grayscale: bool = False
    allow: Optional[List[str]] = None
    allow_local_files: bool = False
    network: Optional[NetworkPolicy] = None
    base_url: Optional[str] = None
    timeout_ms: int = 0

    def update(self, **overrides):
        # type: (object) -> PDFOptions
        """Return a copy with the given fields replaced."""
        return dataclasses.replace(self, **overrides)

    def to_kwargs(self):
        # type: () -> dict
        """Return Document constructor kwargs for these options.

        ``timeout_ms`` is dropped because it belongs to the serializer,
        not the Document tree.
        """
        return {
            field.name: getattr(self, field.name)
            for field in dataclasses.fields(self)
            if field.name != "timeout_ms"
        }


@dataclasses.dataclass
class ImageOptions:
    """Flat option bag behind convert_html_to_image."""

    format: str = "png"
    width: int = 0
    height: int = 0
    quality: int = 94
    smart_width: Optional[bool] = None
    transparent: bool = False
    crop: Optional[Crop] = None
    zoom: float = 0.0
    allow: Optional[List[str]] = None
    allow_local_files: bool = False
    base_url: Optional[str] = None
    network: Optional[NetworkPolicy] = None
    timeout_ms: int = 0

    def update(self, **overrides):
        # type: (object) -> ImageOptions
        """Return a copy with the given fields replaced."""
        return dataclasses.replace(self, **overrides)


@dataclasses.dataclass
class Document:
    """Multi-field document model kept for API compatibility.

    The engine returns a drawing list; ``pdf`` reports that PDF writing
    was removed.
    """

    pages: List[Page] = dataclasses.field(default_factory=list)
    cover: Optional[Page] = None
    toc: Optional[TOC] = None
    page_size: str = "A4"
    width_mm: float = 0.0
    height_mm: float = 0.0
    orientation: str = "portrait"
    margin: Optional[Margin] = None
    title: str = ""
    pdf_version: str = ""
    pdf_profile: str = ""
    copies: int = 0
    collate: Optional[bool] = None
    outline: Optional[bool] = None
    outline_depth: int = 0
    background: Optional[bool] = None
    smart_shrinking: Optional[bool] = None
    compression: Optional[bool] = None
    resolve_relative_links: Optional[bool] = None
    grayscale: bool = False
    page_offset: int = 0
    exclude_from_outline: Optional[bool] = None
    header: Optional[HeaderFooter] = None
    footer: Optional[HeaderFooter] = None
    allow: Optional[List[str]] = None
    allow_local_files: bool = False
    font_paths: Optional[List[str]] = None
    use_system_fonts: bool = False
    network: Optional[NetworkPolicy] = None
    on_info: Optional[object] = None
    on_warn: Optional[object] = None
    on_error: Optional[object] = None

    def validate(self):
        # type: () -> None
        """Check the document tree without touching the library."""
        if self.cover is None and not self.pages:
            raise ErrNoPageObjects(
                "document needs at least one page or a cover"
            )
        size_key = (self.page_size or "").strip().lower()
        if size_key and size_key not in PAGE_SIZES:
            raise ErrInvalidPageSize(
                'unknown page size "{0}"'.format(self.page_size)
            )
        orientation_key = (self.orientation or "").strip().lower()
        if orientation_key and orientation_key not in _ORIENTATIONS:
            raise ErrInvalidOrientation(
                'unknown orientation "{0}"'.format(self.orientation)
            )
        version_key = (self.pdf_version or "").strip()
        if version_key and version_key not in _PDF_VERSIONS:
            raise ErrInvalidPDFVersion(
                'unsupported PDF version "{0}"'.format(self.pdf_version)
            )
        if (
            isinstance(self.copies, bool)
            or not isinstance(self.copies, int)
            or self.copies < 0
            or self.copies > MAX_DOCUMENT_COPIES
        ):
            raise InvalidArgumentError(
                1,
                "copies must be an integer between 0 and {0}".format(
                    MAX_DOCUMENT_COPIES
                ),
            )
        if self.cover is not None:
            self.cover.source.validate()
        for page in self.pages:
            page.source.validate()

    def pdf(self, timeout=None):
        # type: (Optional[float]) -> bytes
        """PDF writing was removed. The engine returns a drawing list."""
        raise RuntimeError(
            "PDF writing was removed; the engine returns a drawing list"
        )

    def write_pdf(self, fileobj, timeout=None):
        # type: (object, Optional[float]) -> None
        """PDF writing was removed. The engine returns a drawing list."""
        fileobj.write(self.pdf(timeout=timeout))


@dataclasses.dataclass
class ImageDocument:
    """Single-source image model kept for API compatibility."""

    source: Content = None  # type: ignore
    width: int = 0
    height: int = 0
    format: str = "png"
    quality: int = 94
    smart_width: Optional[bool] = None
    transparent: bool = False
    crop: Optional[Crop] = None
    zoom: float = 0.0
    allow: Optional[List[str]] = None
    allow_local_files: bool = False
    base_url: Optional[str] = None
    network: Optional[NetworkPolicy] = None

    def validate(self):
        # type: () -> None
        """Check the image source and format without touching the library."""
        self.source.validate()
        format_key = (self.format or "").strip().lower()
        if format_key and format_key not in ("png", "jpg", "jpeg"):
            raise InvalidArgumentError(
                1,
                'unsupported image format "{0}"; expected png, jpg, or'
                " jpeg".format(self.format),
            )
        if (
            isinstance(self.quality, bool)
            or not isinstance(self.quality, int)
            or self.quality < 0
            or self.quality > 100
        ):
            raise InvalidArgumentError(
                1,
                "quality must be an integer between 0 and 100: got {0}".format(
                    self.quality
                ),
            )
        if self.crop is not None:
            for field in ("left", "top", "width", "height"):
                value = getattr(self.crop, field)
                if isinstance(value, bool) or not isinstance(value, int) or value < 0:
                    raise InvalidArgumentError(
                        1,
                        "crop {0} {1} must be non-negative".format(field, value),
                    )

    def image(self, timeout=None):
        # type: (Optional[float]) -> bytes
        """Image encoding was removed. The engine returns a drawing list."""
        raise RuntimeError(
            "image encoding was removed; the engine returns a drawing list"
        )

    def write_image(self, fileobj, timeout=None):
        # type: (object, Optional[float]) -> None
        """Image encoding was removed. The engine returns a drawing list."""
        fileobj.write(self.image(timeout=timeout))


def _encode_optional(text, keepalive):
    # type: (Optional[str], list) -> Optional[bytes]
    """Encode a string for a c_char_p field; empty becomes NULL."""
    if text is None:
        return None
    encoded = text.encode("utf-8")
    keepalive.append(encoded)
    return encoded or None


def _network_policy_flag(policy):
    # type: (Optional[NetworkPolicy]) -> int
    if policy is None:
        return 0
    if (
        policy.restricted
        or policy.block_private_networks
        or policy.block_cross_host_redirects
    ):
        return 1
    return 0


def _allow_array(entries, keepalive):
    # type: (Optional[Sequence[str]], list)
    """Build the c_char_p array for the allow list, keeping refs alive."""
    encoded = [_encode_optional(item, keepalive) for item in entries or []]
    array = (ctypes.c_char_p * len(encoded))(*encoded)
    keepalive.append(array)
    return array


def _deadline_ms(timeout):
    # type: (Optional[float]) -> int
    if not timeout or timeout <= 0:
        return 0
    return max(1, int(round(float(timeout) * 1000)))


def _serialize_image_options(image_doc, timeout):
    # type: (ImageDocument, Optional[float]) -> tuple
    """Fill every pinned GwkImageOptions field from an ImageDocument."""
    from . import _lib

    keepalive = []  # type: list
    opts = _lib.GwkImageOptions.create()
    opts.format = _encode_optional(image_doc.format or "", keepalive)
    base_url = image_doc.base_url
    if not (base_url or "").strip() and image_doc.source.kind == "html":
        base_url = image_doc.source.base
    opts.base_url = _encode_optional(base_url or "", keepalive)
    opts.allow = _allow_array(image_doc.allow, keepalive)
    opts.allow_len = len(image_doc.allow or [])
    opts.width = image_doc.width
    opts.height = image_doc.height
    opts.quality = image_doc.quality
    if image_doc.smart_width is None:
        opts.smart_width = -1
    else:
        opts.smart_width = 1 if image_doc.smart_width else 0
    opts.transparent = 1 if image_doc.transparent else 0
    crop = image_doc.crop or Crop()
    opts.crop_left = crop.left
    opts.crop_top = crop.top
    opts.crop_width = crop.width
    opts.crop_height = crop.height
    opts.zoom = image_doc.zoom
    opts.enable_local_file_access = 1 if image_doc.allow_local_files else 0
    opts.network_policy = _network_policy_flag(image_doc.network)
    opts.timeout_ms = _deadline_ms(timeout)
    return opts, keepalive
