package layout

import (
	"strconv"
	"strings"
)

// parseOpacityValue parses CSS opacity (number 0..1 or percentage).
func parseOpacityValue(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 1, false
	}

	if strings.HasSuffix(value, "%") {
		v, err := strconv.ParseFloat(strings.TrimSpace(value[:len(value)-1]), 64)
		if err != nil {
			return 1, false
		}

		return clamp01(v / oneHundred), true
	}

	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 1, false
	}

	return clamp01(v), true
}

// parseFilterOpacity extracts opacity() from a filter list; other functions
// are ignored (blur/drop-shadow are permanent print-engine non-goals).
func parseFilterOpacity(value string) (float64, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" || value == cssDisplayNone {
		return 1, false
	}

	found := false
	acc := 1.0
	rest := value

	for {
		rest = strings.TrimSpace(rest)
		if rest == "" {
			break
		}

		name, args, next, ok := splitTransformFunc(rest)
		if !ok {
			break
		}

		if name == "opacity" {
			if v, pok := parseOpacityValue(args); pok {
				acc *= v
				found = true
			}
		}
		// blur, brightness, etc. ignored
		rest = next
	}

	return acc, found
}

func clamp01(val float64) float64 {
	if val < 0 {
		return 0
	}

	if val > 1 {
		return 1
	}

	return val
}
