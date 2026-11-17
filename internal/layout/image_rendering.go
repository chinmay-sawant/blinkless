package layout

import "strings"

// image-rendering (CSS Images, SVG 1.1 keywords) hints how an image is
// resampled. The catalog syntax is auto | optimizeQuality | optimizeSpeed.
// pixelated and crisp-edges are not in that syntax and are refused.
//
// The engine does not resample: the page is a drawing list. The keyword is
// stored on the image's resolved style and carried on the OpImage extra, so a
// host that replays the list can pick its own filter. The property inherits.

const (
	imageRenderingProp    = "image-rendering"
	imageRenderingAuto    = "auto"
	imageRenderingQuality = "optimizeQuality"
	imageRenderingSpeed   = "optimizeSpeed"
	// imageRenderingSpeedKeyword is the lowercase token the switch matches.
	imageRenderingSpeedKeyword = "optimizespeed"
)

// applyImageRenderingProps owns image-rendering. Registered in styleGroups
// (style_cascade.go). A token outside the three keywords is refused, so the
// previous value stays.
func applyImageRenderingProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	if prop != imageRenderingProp {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "auto":
		style.ImageRendering = imageRenderingAuto
	case "optimizequality":
		style.ImageRendering = imageRenderingQuality
	case imageRenderingSpeedKeyword:
		style.ImageRendering = imageRenderingSpeed
	}

	return true
}

// inheritExtraProps copies inherited properties the full inherit mask cannot
// hold: image-rendering and the text-stroke trio.
func inheritExtraProps(dst, parent *ResolvedStyle, raw map[string]string) {
	if _, ok := raw[imageRenderingProp]; !ok {
		dst.ImageRendering = parent.ImageRendering
	}

	inheritTextStroke(dst, parent, raw)
	inheritTriggerProps(dst, parent, raw)
	inheritDeclaredProps(dst, parent, raw)
}

// withImageRendering records the image-rendering keyword on an image op.
// auto is the default and is not stored.
func (op Op) withImageRendering(keyword string) Op {
	if keyword == "" || keyword == imageRenderingAuto {
		return op
	}

	extra := op.detachedExtraCopy()
	extra.ImageRendering = keyword
	op.opExtra = extra

	return op
}

// ImageRendering returns the image-rendering keyword of an image op:
// "auto", "optimizeQuality", or "optimizeSpeed".
func (op Op) ImageRendering() string {
	if op.opExtra == nil || op.opExtra.ImageRendering == "" {
		return imageRenderingAuto
	}

	return op.opExtra.ImageRendering
}
