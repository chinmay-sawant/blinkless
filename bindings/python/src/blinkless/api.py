"""High-level one-call helpers mirroring the historical issue contract.

The engine lays out HTML and returns a drawing list. It has no PDF writer
and no image encoder, so every helper here raises ``RuntimeError`` with
the removal reason. The names and signatures stay so existing imports
keep working and fail with a clear message instead of an
``AttributeError`` from a missing library symbol.
"""


def convert_html_to_pdf(html, options=None, **overrides):
    # type: (object, object, object) -> bytes
    """PDF writing was removed. The engine returns a drawing list."""
    raise RuntimeError(
        "PDF writing was removed; the engine returns a drawing list"
    )


def convert_file_to_pdf(source, out_path=None, **options):
    # type: (str, object, object) -> bytes
    """PDF writing was removed. The engine returns a drawing list."""
    raise RuntimeError(
        "PDF writing was removed; the engine returns a drawing list"
    )


def convert_url_to_pdf(url, **options):
    # type: (str, object) -> bytes
    """PDF writing was removed. The engine returns a drawing list."""
    raise RuntimeError(
        "PDF writing was removed; the engine returns a drawing list"
    )


def convert_html_to_image(html, options=None, **overrides):
    # type: (object, object, object) -> bytes
    """Image encoding was removed. The engine returns a drawing list."""
    raise RuntimeError(
        "image encoding was removed; the engine returns a drawing list"
    )
