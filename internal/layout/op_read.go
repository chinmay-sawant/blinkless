package layout

import (
	"bytes"
	"math"
)

// Nil-safe accessors for the opExtra payload. The fields on opExtra are
// promoted through an embedded pointer, so reading one directly on an op that
// never bound a rare payload dereferences a nil pointer. A display-list
// consumer outside this module cannot construct an opExtra to guard with, so
// every rare-payload read crosses one of these methods instead. They are the
// supported way for an external replay engine to read that payload.

// HasExtra reports whether the operation carries a non-empty rare opExtra
// payload (link URI, image, transform, opacity, blend mode, group, or
// outline). Ops with a nil extra or the shared zero emptyExtra singleton
// report false.
func (op *Op) HasExtra() bool {
	return op != nil && op.opExtra != nil && op.opExtra != emptyExtra
}

// HasBoundExtra reports whether op has any opExtra pointer bound (including
// the shared zero emptyExtra singleton).
func (op *Op) HasBoundExtra() bool {
	return op != nil && op.opExtra != nil
}

// SameExtra compares the rare payload fields of op and other by pointer without
// copying the 256-byte Op struct across value-receiver getters. sharedImage
// reports whether a non-empty image match was decided by shared backing array.
func (op *Op) SameExtra(other *Op) (bool, bool) {
	if !op.HasExtra() && !other.HasExtra() {
		return op.HasBoundExtra() == other.HasBoundExtra(), false
	}

	if op.isOutline() != other.isOutline() ||
		effectiveOpOpacity(op) != effectiveOpOpacity(other) ||
		op.GroupBoundary() != other.GroupBoundary() {
		return false, false
	}

	left := boundOpExtra(op)
	right := boundOpExtra(other)

	if !sameExtraMeta(left, right) || extraTransformOf(op) != extraTransformOf(other) {
		return false, false
	}

	return sameExtraImages(left, right)
}

func boundOpExtra(op *Op) *opExtra {
	if op == nil || op.opExtra == nil {
		return emptyExtra
	}

	return op.opExtra
}

func extraTransformOf(op *Op) Matrix2D {
	if op == nil || op.opExtra == nil {
		return IdentityMatrix()
	}

	return op.opExtra.Xform
}

func sameExtraMeta(left, right *opExtra) bool {
	return left.NoFakeBold == right.NoFakeBold &&
		left.URI == right.URI &&
		left.Alt == right.Alt &&
		left.BlendMode == right.BlendMode &&
		left.FontFeatures == right.FontFeatures &&
		left.TextLanguage == right.TextLanguage &&
		left.TextAutospace == right.TextAutospace &&
		left.TextTransform == right.TextTransform
}

func sameExtraImages(left, right *opExtra) (bool, bool) {
	if left.ImgW != right.ImgW || left.ImgH != right.ImgH || len(left.Image) != len(right.Image) {
		return false, false
	}

	if len(left.Image) == 0 {
		return true, false
	}

	if &left.Image[0] == &right.Image[0] {
		return true, true
	}

	return bytes.Equal(left.Image, right.Image), false
}

const (
	extraTagURI           uint64 = 11
	extraTagAlt           uint64 = 12
	extraTagBlendMode     uint64 = 13
	extraTagFontFeatures  uint64 = 14
	extraTagTextLanguage  uint64 = 15
	extraTagTextAutospace uint64 = 16
	extraTagTextTransform uint64 = 17
	extraTagNoFakeBold    uint64 = 18
	extraTagIsOutline     uint64 = 19
	extraTagGroupBoundary uint64 = 20
	extraTagXform         uint64 = 21
	extraTagOpacity       uint64 = 22
	extraTagImage         uint64 = 23
)

// MixExtraStamp folds the non-default rare payload fields of op into an FNV-1a
// hash without copying the 256-byte Op struct. It returns false if any float
// field is NaN.
func (op *Op) MixExtraStamp(hash, prime uint64) (uint64, bool) {
	if !op.HasExtra() {
		return hash, true
	}

	extra := op.opExtra
	hash = mixExtraTextMeta(hash, prime, op, extra)

	var ok bool
	if hash, ok = mixExtraTransformAndOpacity(hash, prime, op, extra); !ok {
		return hash, false
	}

	return mixExtraImage(hash, prime, extra), true
}

func mixExtraTextMeta(hash, prime uint64, paintOp *Op, extra *opExtra) uint64 {
	if extra.URI != "" {
		hash = mixExtraTaggedStr(hash, prime, extraTagURI, extra.URI)
	}

	if extra.Alt != "" {
		hash = mixExtraTaggedStr(hash, prime, extraTagAlt, extra.Alt)
	}

	if extra.BlendMode != "" {
		hash = mixExtraTaggedStr(hash, prime, extraTagBlendMode, extra.BlendMode)
	}

	if extra.FontFeatures != "" {
		hash = mixExtraTaggedStr(hash, prime, extraTagFontFeatures, extra.FontFeatures)
	}

	if extra.TextLanguage != "" {
		hash = mixExtraTaggedStr(hash, prime, extraTagTextLanguage, extra.TextLanguage)
	}

	if extra.TextAutospace != "" {
		hash = mixExtraTaggedStr(hash, prime, extraTagTextAutospace, extra.TextAutospace)
	}

	if extra.TextTransform != "" {
		hash = mixExtraTaggedStr(hash, prime, extraTagTextTransform, extra.TextTransform)
	}

	return mixExtraFlags(hash, prime, paintOp, extra)
}

func mixExtraFlags(hash, prime uint64, paintOp *Op, extra *opExtra) uint64 {
	if extra.NoFakeBold {
		hash = (hash ^ extraTagNoFakeBold) * prime
	}

	if extra.IsOutline {
		hash = (hash ^ extraTagIsOutline) * prime
	}

	if groupBound := paintOp.GroupBoundary(); groupBound != 0 {
		hash = ((hash^extraTagGroupBoundary)*prime ^ uint64(groupBound)) * prime
	}

	return hash
}

func mixExtraTransformAndOpacity(hash, prime uint64, paintOp *Op, extra *opExtra) (uint64, bool) {
	zeroMatrix := Matrix2D{A: 0, B: 0, C: 0, D: 0, E: 0, F: 0}
	if mat := extra.Xform; mat != zeroMatrix {
		hash = (hash ^ extraTagXform) * prime

		for _, val := range [6]float64{mat.A, mat.B, mat.C, mat.D, mat.E, mat.F} {
			if math.IsNaN(val) {
				return hash, false
			}

			if val == 0 {
				hash *= prime
			} else {
				hash = (hash ^ math.Float64bits(val)) * prime
			}
		}
	}

	if math.IsNaN(extra.PaintOpacity) {
		return hash, false
	}

	if eff := effectiveOpOpacity(paintOp); eff != 1 && eff != paintOp.Alpha {
		if math.IsNaN(eff) {
			return hash, false
		}

		hash = (hash ^ extraTagOpacity) * prime
		if eff == 0 {
			hash *= prime
		} else {
			hash = (hash ^ math.Float64bits(eff)) * prime
		}
	}

	return hash, true
}

func effectiveOpOpacity(paintOp *Op) float64 {
	opacity := 1.0
	if paintOp == nil {
		return opacity
	}

	if paintOp.opExtra != nil && paintOp.opExtra.PaintOpacity > 0 && paintOp.opExtra.PaintOpacity < 1 {
		opacity = paintOp.opExtra.PaintOpacity
	}

	if paintOp.Alpha > 0 && paintOp.Alpha < 1 {
		opacity *= paintOp.Alpha
	}

	return opacity
}

func mixExtraImage(hash, prime uint64, extra *opExtra) uint64 {
	if extra.ImgW == 0 && extra.ImgH == 0 && len(extra.Image) == 0 {
		return hash
	}

	hash = (hash ^ extraTagImage) * prime
	//nolint:gosec // deterministic hash mixing of image dimensions; wrap is harmless
	hash = (hash ^ uint64(extra.ImgW)) * prime
	//nolint:gosec // deterministic hash mixing of image dimensions; wrap is harmless
	hash = (hash ^ uint64(extra.ImgH)) * prime
	hash = (hash ^ uint64(len(extra.Image))) * prime

	for i := range len(extra.Image) {
		hash = (hash ^ uint64(extra.Image[i])) * prime
	}

	return hash
}

func mixExtraTaggedStr(hash, prime, tag uint64, str string) uint64 {
	hash = (hash ^ tag) * prime
	hash = (hash ^ uint64(len(str))) * prime

	for i := range len(str) {
		hash = (hash ^ uint64(str[i])) * prime
	}

	return hash
}

// LinkURI returns the op's link target, or "" when it carries none.
func (op Op) LinkURI() string {
	if op.opExtra == nil {
		return ""
	}

	return op.opExtra.URI
}

// ImageBytes returns the op's encoded image payload with its pixel bounds.
// The slice aliases the op's buffer; read it, never write it. It returns nil
// for every op that is not an image.
func (op Op) ImageBytes() ([]byte, int, int) {
	if op.opExtra == nil {
		return nil, 0, 0
	}

	return op.opExtra.Image, op.opExtra.ImgW, op.opExtra.ImgH
}

// ImageAlt returns the op's alt text, or "" when it carries none.
func (op Op) ImageAlt() string {
	if op.opExtra == nil {
		return ""
	}

	return op.opExtra.Alt
}

// Transform returns the op's baked 2D transform. It returns the identity when
// the op has none; check XformSet to tell "identity" from "unset".
func (op Op) Transform() Matrix2D {
	if op.opExtra == nil {
		return IdentityMatrix()
	}

	return op.opExtra.Xform
}

// XformValue returns the stored 2D transform without copying the operation.
// A nil payload returns the zero matrix, which callers treat as no transform.
// Unlike Transform, it does not allocate or bind the shared empty payload.
func (op *Op) XformValue() Matrix2D {
	if op == nil || op.opExtra == nil {
		return Matrix2D{}
	}

	return op.opExtra.Xform
}

// PathValue returns the straight-edged path attached to an operation, if any.
// The returned points belong to the display list and must not be modified.
func (op Op) PathValue() (DisplayPath, bool) {
	if op.opExtra == nil || !op.opExtra.HasPath {
		return DisplayPath{}, false
	}

	return op.opExtra.Path, true
}

// GlyphBasisValue returns the final local basis for a projected text run.
func (op Op) GlyphBasisValue() (GlyphBasis, bool) {
	if op.opExtra == nil || !op.opExtra.HasGlyphBasis {
		return GlyphBasis{}, false
	}

	return op.opExtra.GlyphBasis, true
}

// ProjectiveTransformValue returns the projection for a payload the display
// list could not turn into final vector geometry, such as a textured image.
func (op Op) ProjectiveTransformValue() (ProjectiveTransform, bool) {
	if op.opExtra == nil || !op.opExtra.HasProjective {
		return ProjectiveTransform{}, false
	}

	return op.opExtra.Projective, true
}

// BlendModeName returns the op's CSS mix-blend-mode value, or "" for the
// default normal blending.
func (op Op) BlendModeName() string {
	if op.opExtra == nil {
		return ""
	}

	return op.opExtra.BlendMode
}

// Opacity returns the element opacity the op was painted with, folded with the
// op's own alpha exactly as the engine folds it.
//
// The engine stores 0 for "no opacity override" and treats only values
// strictly between 0 and 1 as a real opacity, so this reports the same
// effective number the painters used: 1 when nothing was set, and the product
// of the element opacity and the op alpha when both were.
func (op Op) Opacity() float64 {
	return pdfPaintOpacity(&op, true)
}

// Outline reports whether the op is a CSS outline operation. Outlines paint
// above descendant content, unlike ordinary backgrounds and borders.
func (op Op) Outline() bool {
	if op.opExtra == nil {
		return false
	}

	return op.opExtra.IsOutline
}

// IsCanvasBackground reports whether the op is the root canvas default fill
// (the dark color-scheme canvas rect). It is the canvas background, not an
// element background: a consumer that paginates the display list paints it
// across every page, while a single-canvas consumer paints it once.
func (op Op) IsCanvasBackground() bool {
	if op.opExtra == nil {
		return false
	}

	return op.opExtra.CanvasBackground
}

// TextTransformValue returns the CSS text-transform value the op carries, or
// "" when it carries none. It is spelled with a Value suffix rather than
// TextTransform because a method of that name would shadow the promoted
// opExtra field of the same name.
func (op Op) TextTransformValue() string {
	if op.opExtra == nil {
		return ""
	}

	return op.opExtra.TextTransform
}

// NoFakeBoldValue reports whether the op forbids synthesizing bold weight.
// The suffix keeps the accessor name distinct from the opExtra field of the
// same name, so the two read as one concept without one shadowing the other.
func (op Op) NoFakeBoldValue() bool {
	if op.opExtra == nil {
		return false
	}

	return op.opExtra.NoFakeBold
}

// Group and GroupBoundary already exist on *Op and are nil-safe, so a blend
// group is read through those rather than through a duplicate here.
