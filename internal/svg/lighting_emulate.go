package svg

import (
	"math"
	"strconv"
	"strings"
)

// A filter that holds one feDiffuseLighting and one feDistantLight is drawn as
// flat shapes. The canvas parser has no filter graph, so the shape is replaced
// by a rect over the filter region in the lit color. A rect also gets the
// bands along its outline where the surface normal tilts.
//
// The numbers follow the SVG filter spec. The surface of the filter input is
// its alpha channel times surfaceScale, so the interior is flat with the
// normal (0, 0, 1) and the color is lighting-color * diffuseConstant *
// sin(elevation). Across an outline the alpha steps from 0 to 1, which tilts
// the normal by surfaceScale along that axis for one pixel on each side.
// Colors are blended in linear RGB unless color-interpolation-filters is sRGB.
//
// Covered: feDiffuseLighting with feDistantLight, flat interior for rect,
// circle and ellipse, outline bands for rect. Not covered: feSpecularLighting,
// point and spot lights, outline bands on round shapes and at rect corners,
// kernelUnitLength, and filters with other primitives.

// lightBand is the width of an outline band in pixels: one pixel on each side.
const lightBand = 2

// svgRectTag is the element the lit regions are drawn with.
const svgRectTag = "rect"

// emulateLighting is the lighting of one filter.
type emulateLighting struct {
	surfaceScale float64
	diffuse      float64
	azimuth      float64
	elevation    float64
	color        [3]float64 // 0..1, as written (sRGB)
	linear       bool
}

// emulateLightingFilter reads a filter whose primitives are exactly one
// feDiffuseLighting and one feDistantLight.
func emulateLightingFilter(prims []string, filterAttrs, diffuse, distant map[string]string) (emulateLighting, bool) {
	if len(prims) != 2 || prims[0] != "fediffuselighting" || prims[1] != "fedistantlight" {
		return emulateLighting{}, false
	}

	color, ok := parseLightColor(lightingColorOf(diffuse))
	if !ok {
		return emulateLighting{}, false
	}

	light := emulateLighting{
		surfaceScale: lightNumber(diffuse["surfacescale"], 1),
		diffuse:      lightNumber(diffuse["diffuseconstant"], 1),
		azimuth:      lightNumber(distant["azimuth"], 0),
		elevation:    lightNumber(distant["elevation"], 0),
		color:        color,
		linear:       !strings.EqualFold(strings.TrimSpace(interpolationOf(diffuse, filterAttrs)), "srgb"),
	}

	if light.diffuse < 0 || light.surfaceScale < 0 {
		return emulateLighting{}, false
	}

	return light, true
}

func lightingColorOf(attrs map[string]string) string {
	if value := styleDeclarations(attrs["style"])["lighting-color"]; value != "" {
		return value
	}

	if value := attrs["lighting-color"]; value != "" {
		return value
	}

	return "white"
}

func interpolationOf(primitive, filter map[string]string) string {
	if value := styleDeclarations(primitive["style"])["color-interpolation-filters"]; value != "" {
		return value
	}

	if value := primitive["color-interpolation-filters"]; value != "" {
		return value
	}

	return filter["color-interpolation-filters"]
}

func lightNumber(raw string, fallback float64) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return fallback
	}

	return value
}

// shade returns the color of a point whose alpha gradient is (gradX, gradY)
// pixels. (0, 0) is the flat interior.
func (l emulateLighting) shade(gradX, gradY float64) string {
	azimuth, elevation := l.azimuth*math.Pi/180, l.elevation*math.Pi/180
	flat := math.Cos(elevation)
	lightX, lightY, lightZ := math.Cos(azimuth)*flat, math.Sin(azimuth)*flat, math.Sin(elevation)

	normX, normY := -l.surfaceScale*gradX, -l.surfaceScale*gradY
	length := math.Sqrt(normX*normX + normY*normY + 1)
	dot := math.Max((normX*lightX+normY*lightY+lightZ)/length, 0) * l.diffuse

	var rgb [3]int

	for channel, value := range l.color {
		if l.linear {
			value = srgbToLinear(value)
		}

		value = math.Min(value*dot, 1)
		if l.linear {
			value = linearToSRGB(value)
		}

		rgb[channel] = int(math.Round(value * 255))
	}

	return "rgb(" + strconv.Itoa(rgb[0]) + "," + strconv.Itoa(rgb[1]) + "," + strconv.Itoa(rgb[2]) + ")"
}

func linearToSRGB(value float64) float64 {
	if value <= 0.0031308 {
		return value * 12.92
	}

	return 1.055*math.Pow(value, 1/2.4) - 0.055
}

// emulateLitShape draws the lit region of one shape: a rect over the filter
// region and, for a rect, the outline bands. unit is one device pixel in user
// units.
func emulateLitShape(
	name string, index map[string]string, light emulateLighting, regionX, regionY, regionW, regionH,
	boxX, boxY, boxW, boxH, unit float64,
) string {
	transform := index["transform"]

	tag := func(x, y, w, h float64, fill string) string {
		attrs := []emulateAttr{
			{"x", num(x)}, {"y", num(y)}, {"width", num(w)}, {"height", num(h)}, {"fill", fill},
		}
		if transform != "" {
			attrs = append(attrs, emulateAttr{"transform", transform})
		}

		return emulateTagText(svgRectTag, attrs, nil)
	}

	out := tag(regionX, regionY, regionW, regionH, light.shade(0, 0))
	if name != svgRectTag || unit <= 0 {
		return out
	}

	// One pixel on each side of the outline. Corners keep the interior color.
	left, right := boxX-unit, boxX+boxW-unit
	top, bottom := boxY-unit, boxY+boxH-unit
	spanX, spanY := boxW-unit, boxH-unit

	if spanX <= 0 || spanY <= 0 {
		return out
	}

	band := lightBand * unit
	out += tag(left, top+band, band, spanY-unit, light.shade(1, 0))
	out += tag(right, top+band, band, spanY-unit, light.shade(-1, 0))
	out += tag(left+band, top, spanX-unit, band, light.shade(0, 1))
	out += tag(left+band, bottom, spanX-unit, band, light.shade(0, -1))

	return out
}

// parseLightColor reads #rgb, #rrggbb, rgb(r,g,b) and a few color names.
func parseLightColor(raw string) ([3]float64, bool) {
	raw = strings.ToLower(strings.TrimSpace(raw))

	if hexColor, ok := strings.CutPrefix(raw, "#"); ok {
		return parseHexColor(hexColor)
	}

	if body, ok := strings.CutPrefix(raw, "rgb("); ok && strings.HasSuffix(body, ")") {
		parts := strings.Split(strings.TrimSuffix(body, ")"), ",")
		if len(parts) != 3 {
			return [3]float64{}, false
		}

		var out [3]float64

		for index, part := range parts {
			value, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
			if err != nil {
				return [3]float64{}, false
			}

			out[index] = math.Min(math.Max(value, 0), 255) / 255
		}

		return out, true
	}

	named := map[string][3]float64{
		"white": {1, 1, 1}, "black": {0, 0, 0}, "red": {1, 0, 0}, "lime": {0, 1, 0}, "blue": {0, 0, 1},
		"yellow": {1, 1, 0}, "cyan": {0, 1, 1}, "magenta": {1, 0, 1}, "gray": {0.5, 0.5, 0.5}, "grey": {0.5, 0.5, 0.5},
	}

	value, ok := named[raw]

	return value, ok
}

func parseHexColor(hexColor string) ([3]float64, bool) {
	if len(hexColor) == 3 {
		hexColor = string([]byte{hexColor[0], hexColor[0], hexColor[1], hexColor[1], hexColor[2], hexColor[2]})
	}

	if len(hexColor) != 6 {
		return [3]float64{}, false
	}

	var out [3]float64

	for index := range out {
		value, err := strconv.ParseUint(hexColor[index*2:index*2+2], 16, 8)
		if err != nil {
			return [3]float64{}, false
		}

		out[index] = float64(value) / 255
	}

	return out, true
}
