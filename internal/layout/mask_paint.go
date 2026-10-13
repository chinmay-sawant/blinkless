package layout

import (
	"net/url"
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/svg"
)

func maskImageEnabled(value string) bool {
	layers, ok := splitMaskLayerList(value)
	if !ok {
		return false
	}

	for _, layer := range layers {
		if !strings.EqualFold(strings.TrimSpace(layer), "none") {
			return true
		}
	}

	return false
}

func (e *engine) maskDescriptor(style ResolvedStyle) *MaskDescriptor {
	images, ok := splitMaskLayerList(style.MaskImage)
	if !ok || len(images) == 0 {
		return nil
	}

	descriptor := &MaskDescriptor{Layers: make([]MaskLayer, 0, len(images))}

	for index, source := range images {
		layer := MaskLayer{
			Source:       strings.TrimSpace(source),
			Mode:         maskLayerValue(style.MaskMode, index, maskModeMatchSource),
			Composite:    maskLayerValue(style.MaskComposite, index, maskCompositeAdd),
			Position:     maskLayerValue(style.MaskPosition, index, maskInitialPosition),
			Repeat:       maskLayerValue(style.MaskRepeat, index, "repeat"),
			Size:         maskLayerValue(style.MaskSize, index, "auto"),
			Origin:       maskLayerValue(style.MaskOrigin, index, "border-box"),
			Clip:         maskLayerValue(style.MaskClip, index, "border-box"),
			ImageBytes:   nil,
			PixelWidth:   0,
			PixelHeight:  0,
			Preprocessed: false,
		}

		if !strings.EqualFold(layer.Source, "none") {
			layer = e.resolveMaskLayer(layer)
		}

		descriptor.Layers = append(descriptor.Layers, layer)
	}

	return descriptor
}

func maskLayerValue(value string, index int, initial string) string {
	values, ok := splitMaskLayerList(value)
	if !ok || len(values) == 0 {
		return initial
	}

	return strings.TrimSpace(values[index%len(values)])
}

func (e *engine) resolveMaskLayer(layer MaskLayer) MaskLayer {
	target := backgroundImageSrc(layer.Source)
	if target == "" {
		return layer
	}

	resourceURL, fragment, ok := splitMaskSourceURL(target)
	if !ok {
		return layer
	}

	if fragment != "" {
		return e.resolveSVGMaskLayer(layer, resourceURL, fragment)
	}

	return e.resolveRasterMaskLayer(layer, resourceURL)
}

func (e *engine) resolveSVGMaskLayer(layer MaskLayer, resourceURL, fragment string) MaskLayer {
	if resourceURL == "" {
		resourceURL = "#" + fragment
	}

	ref := e.resolveImage(resourceURL)
	if ref == nil || len(ref.sourceData) == 0 {
		return layer
	}

	maskBytes, width, height, err := svg.RasterizeMask(ref.sourceData, fragment, 0, 0, layer.Mode)
	if err != nil {
		return layer
	}

	layer.ImageBytes = maskBytes
	layer.PixelWidth = width
	layer.PixelHeight = height
	layer.Preprocessed = true

	return layer
}

func (e *engine) resolveRasterMaskLayer(layer MaskLayer, resourceURL string) MaskLayer {
	ref := e.resolveImage(resourceURL)
	if ref == nil || len(ref.data) == 0 {
		return layer
	}

	layer.ImageBytes = ref.data
	layer.PixelWidth = ref.w
	layer.PixelHeight = ref.h

	if layer.Mode == maskModeMatchSource {
		// CSS image sources use their alpha channel for match-source.
		layer.Mode = maskModeAlpha
	}

	return layer
}

func splitMaskSourceURL(target string) (string, string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(target))
	if err != nil {
		return "", "", false
	}

	fragment := parsed.Fragment
	parsed.Fragment = ""

	return parsed.String(), fragment, true
}
