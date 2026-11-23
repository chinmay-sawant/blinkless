// Package settings implements the wkhtmltopdf-compatible typed settings
// model.
//
// Engine callers use the typed structs directly: css.Apply builds a
// DefaultPdfGlobal for sheet collection and maps its media option onto
// MediaType; load reads LoadGlobal and LoadPage; convert.prepare reads the
// page and web fields; bindings/wasm reads the load policy for inline data:
// URLs. ClonePdfGlobal, ClonePdfObject, and CloneImageGlobal return
// independent snapshots when a caller needs one.
//
// # Policy A (settings honesty)
//
// Only options with a live engine consumer (convert, load, or css) get typed
// fields. Inert wkhtml keys (dpi, javascript, plugins, log-level, js-delay,
// user-style-sheet, produce-forms, …) must not reappear as typed stubs
// without a live consumer.
//
// Grayscale is the sole color bit; page geometry is PageSize name + Size
// width/height (mm); DumpOutline / DumpDefaultTOCXSL live on PdfGlobal.
package settings
