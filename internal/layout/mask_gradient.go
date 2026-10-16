package layout

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// normalizeMaskLinearGradient converts CSS length stop positions in linear
// (and repeating-linear) gradient masks into percentage positions along the
// gradient line. renderGradientPNG stores stop positions verbatim, which
// means `black 6px, transparent 12px` would be interpreted as t=6..12
// instead of 6..12 CSS pixels measured along the gradient line.
//
// The returned string keeps the original angle and colors; only positions
// expressed as lengths are rewritten as percentages of the rasterized box's
// gradient-line length.
func normalizeMaskLinearGradient(raw string, width, height float64) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	low := strings.ToLower(trimmed)

	repeating := false
	prefix := "linear-gradient("
	if strings.HasPrefix(low, "repeating-linear-gradient(") {
		repeating = true
		prefix = "repeating-linear-gradient("
	} else if !strings.HasPrefix(low, "linear-gradient(") {
		return "", false
	}

	body := trimmed[len(prefix) : len(trimmed)-1]
	parts := splitFunctionArgs(body)
	if len(parts) == 0 {
		return "", false
	}

	angle := 180.0
	isDirection := false
	stopStart := 0
	firstPart := strings.TrimSpace(parts[0])
	firstLow := strings.ToLower(firstPart)

	if strings.HasPrefix(firstLow, "to ") {
		isDirection = true
		stopStart = 1
		dir := strings.TrimSpace(firstLow[3:])
		switch dir {
		case "top":
			angle = 0
		case "right":
			angle = 90
		case "bottom":
			angle = 180
		case "left":
			angle = 270
		case "top right", "right top":
			angle = 45
		case "bottom right", "right bottom":
			angle = 135
		case "bottom left", "left bottom":
			angle = 225
		case "top left", "left top":
			angle = 315
		}
	} else if deg, ok := parseAngleDeg(firstPart); ok {
		isDirection = true
		stopStart = 1
		angle = deg
	}

	if stopStart >= len(parts) {
		return "", false
	}

	rad := angle * math.Pi / degToRadFactor
	sinA := math.Sin(rad)
	cosA := math.Cos(rad)
	length := math.Abs(width*sinA) + math.Abs(height*cosA)
	if length <= 0 {
		length = 1
	}

	// Rebuild parts with normalized stops.
	newParts := make([]string, len(parts))
	copy(newParts, parts)
	if isDirection {
		newParts[0] = firstPart
	}

	for i := stopStart; i < len(parts); i++ {
		normalized, ok := normalizeMaskGradientStop(strings.TrimSpace(parts[i]), length)
		if !ok {
			return "", false
		}

		newParts[i] = normalized
	}

	args := strings.Join(newParts, ", ")
	open := prefix
	if repeating {
		open = "repeating-linear-gradient("
	} else {
		open = "linear-gradient("
	}

	return open + args + ")", true
}

func normalizeMaskGradientStop(stop string, lineLength float64) (string, bool) {
	tokens := strings.Fields(stop)
	if len(tokens) == 0 {
		return "", false
	}

	out := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		if strings.HasSuffix(tok, "%") {
			out = append(out, tok)
			continue
		}
		if _, err := strconv.ParseFloat(strings.TrimSuffix(tok, "%"), 64); err == nil && strings.Contains(tok, "%") {
			out = append(out, tok)
			continue
		}
		if v, ok := plainLength(tok, 12, 0); ok {
			percent := 0.0
			if lineLength > 0 {
				percent = v / lineLength * 100
			}

			out = append(out, fmt.Sprintf("%.6g%%", percent))
			continue
		}

		out = append(out, tok)
	}

	return strings.Join(out, " "), true
}
