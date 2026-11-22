package fonts

import (
	"unicode/utf8"

	gtfont "github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/shaping"
)

// KerningAdjustment returns the change in advance due to kern alone, at size.
// Shaping twice keeps ligatures and other features from changing the legacy
// layout advances. The caller supplies one face and one unbroken paint span.
func (f *Font) KerningAdjustment(text string, size float64, settings, lang string) float64 {
	if len(text) > maxShapeRunBytes || utf8.RuneCountInString(text) < 2 {
		return 0
	}

	parsed, ok := f.gotextFace()
	if !ok {
		return 0
	}

	features := ParseFontFeatureSettings(settings)
	kern := ot.MustNewTag("kern")

	for index := range features {
		if features[index].Tag == kern && features[index].Value != 0 {
			// Each call owns the face's mutable glyph caches.
			face := gtfont.NewFace(parsed.Font)
			input := shapingInput([]rune(text), face, features, shapingLanguageOverride(lang))

			return shapeKerningAdjustment(input, index) * size
		}
	}

	return 0
}

// Shape at 64 points in 26.6 fixed point to preserve subpixel adjustments.
// Work in ems so the caller can scale the adjustment to its actual size.
const kerningMeasureSize = 64 * 64

func shapeKerningAdjustment(input shaping.Input, featureIndex int) float64 {
	shaper, hasShaper := shaperPool.Get().(*shaping.HarfbuzzShaper)
	if !hasShaper {
		return 0
	}

	defer shaperPool.Put(shaper)

	segmenter, ok := segmenterPool.Get().(*shaping.Segmenter)
	if !ok {
		return 0
	}

	defer segmenterPool.Put(segmenter)

	input.Size = kerningMeasureSize
	feature := &input.FontFeatures[featureIndex]

	var delta int64

	for _, run := range segmenter.Split(input, singleFaceMap{input.Face}) {
		run.FontFeatures = input.FontFeatures
		on := shapedAdvance(shaper.Shape(run))
		value := feature.Value
		feature.Value = 0
		off := shapedAdvance(shaper.Shape(run))
		feature.Value = value
		delta += on - off
	}

	return float64(delta) / kerningMeasureSize
}

func shapedAdvance(run shaping.Output) int64 {
	var advance int64

	for i := range run.Glyphs {
		advance += int64(run.Glyphs[i].Advance)
	}

	return advance
}
