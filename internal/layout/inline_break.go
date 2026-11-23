package layout

import "unicode/utf8"

func isASCIIText(text string) bool {
	for idx := range len(text) {
		if text[idx] >= utf8.RuneSelf {
			return false
		}
	}

	return true
}

func (e *engine) splitTextToWidthASCII(
	text string, style *ResolvedStyle, firstMax, restMax float64, mode softBreakMode,
) []string {
	chunks := make([]string, 0, 4)

	for len(text) > 0 {
		limit := restMax
		if len(chunks) == 0 && firstMax > 0 {
			limit = firstMax
		}

		if limit < 1 {
			limit = restMax
		}

		count := e.fittingPrefixASCII(text, limit, style)
		if mode != softBreakNone && count < len(text) && count > 1 {
			if soft := lastSoftBreakASCII(text[:count], mode); soft > 0 {
				count = soft
			}
		}

		chunks = append(chunks, text[:count])
		text = text[count:]
	}

	return chunks
}

func (e *engine) fittingPrefixASCII(text string, limit float64, style *ResolvedStyle) int {
	if style == nil {
		return 1
	}

	const fitEps = 0.01

	count := 0
	width := 0.0

	for count < len(text) {
		glyphW := e.measureRuneFace(rune(text[count]), style)
		if count > 0 && width+glyphW > limit+fitEps {
			break
		}

		if count == 0 && glyphW > limit+fitEps {
			return 1
		}

		width += glyphW
		count++
	}

	if count <= 0 {
		return 1
	}

	return count
}

func lastSoftBreakASCII(text string, mode softBreakMode) int {
	for idx := len(text) - 1; idx >= 1; idx-- {
		if isSoftWrapRune(rune(text[idx-1]), mode) {
			return idx
		}
	}

	return 0
}
