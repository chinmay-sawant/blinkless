package layout

import "strings"

const (
	maskShorthandProperty       = "mask"
	maskShorthandDeferredPrefix = "\x00mask:"
	webkitMaskCompositeProperty = "-webkit-mask-composite"
	maskImageProperty           = "mask-image"
	maskModeProperty            = "mask-mode"
	maskPositionProperty        = "mask-position"
	maskSizeProperty            = "mask-size"
	maskRepeatProperty          = "mask-repeat"
	maskOriginProperty          = "mask-origin"
	maskClipProperty            = "mask-clip"
	maskCompositeProperty       = "mask-composite"
	maskTypeProperty            = "mask-type"
	maskModeMatchSource         = "match-source"
	maskModeAlpha               = "alpha"
	maskModeLuminance           = "luminance"
	maskCompositeAdd            = "add"
	maskCompositeSubtract       = "subtract"
	maskCompositeExclude        = "exclude"
	maskRepeatNoRepeat          = "no-repeat"
	maskInitialPosition         = "0% 0%"
	maskLayerComponentLimit     = 2
	cssSpaceTokenCapacity       = 4
)

type maskShorthandLayer struct {
	image     string
	mode      string
	position  string
	size      string
	repeat    string
	origin    string
	clip      string
	composite string
}

func maskShorthandValueAccepted(value string) bool {
	_, ok := parseMaskShorthand(value)

	return ok
}

func parseMaskShorthand(value string) ([]maskShorthandLayer, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.HasPrefix(trimmed, ",") || strings.HasSuffix(trimmed, ",") {
		return nil, false
	}

	parts := splitParenArgs(trimmed, ',')
	if len(parts) == 0 {
		return nil, false
	}

	layers := make([]maskShorthandLayer, 0, len(parts))

	for _, part := range parts {
		layer, ok := parseMaskShorthandLayer(part)
		if !ok {
			return nil, false
		}

		layers = append(layers, layer)
	}

	return layers, true
}

func maskShorthandPropertyValue(layers []maskShorthandLayer, prop string) string {
	var value strings.Builder

	for idx, layer := range layers {
		if idx > 0 {
			value.WriteString(", ")
		}

		value.WriteString(layer.propertyValue(prop))
	}

	return value.String()
}

func (layer maskShorthandLayer) propertyValue(prop string) string {
	switch prop {
	case maskImageProperty:
		return layer.image
	case maskModeProperty:
		return layer.mode
	case maskPositionProperty:
		return layer.position
	case maskSizeProperty:
		return layer.size
	case maskRepeatProperty:
		return layer.repeat
	case maskOriginProperty:
		return layer.origin
	case maskClipProperty:
		return layer.clip
	case maskCompositeProperty:
		return layer.composite
	default:
		return ""
	}
}

type maskShorthandTokenKind uint8

const (
	maskTokenImage maskShorthandTokenKind = iota + 1
	maskTokenMode
	maskTokenRepeat
	maskTokenComposite
	maskTokenNoClip
	maskTokenBox
	maskTokenPosition
)

type maskShorthandState struct {
	layer          maskShorthandLayer
	positionTokens []string
	boxes          []string
	sourceSeen     bool
	modeSeen       bool
	repeatSeen     bool
	compositeSeen  bool
	noClip         bool
}

func newMaskShorthandState() maskShorthandState {
	return maskShorthandState{
		positionTokens: nil,
		boxes:          nil,
		sourceSeen:     false,
		modeSeen:       false,
		repeatSeen:     false,
		compositeSeen:  false,
		noClip:         false,
		layer: maskShorthandLayer{
			image:     "none",
			mode:      maskModeMatchSource,
			position:  maskInitialPosition,
			size:      "auto",
			repeat:    "repeat",
			origin:    "border-box",
			clip:      "border-box",
			composite: maskCompositeAdd,
		},
	}
}

func parseMaskShorthandLayer(value string) (maskShorthandLayer, bool) {
	beforeSize, afterSize, hasSize, ok := splitMaskLayerSlash(value)
	if !ok {
		return emptyMaskShorthandLayer(), false
	}

	state := newMaskShorthandState()

	if !state.addTokens(cssSpaceTokenList(beforeSize), true) {
		return emptyMaskShorthandLayer(), false
	}

	if len(state.positionTokens) > 0 {
		state.layer.position = normalizeCSSValue(strings.Join(state.positionTokens, " "))
	}

	if hasSize {
		sizeTokens := cssSpaceTokenList(afterSize)
		sizeCount := maskShorthandSizeTokenCount(sizeTokens)

		if sizeCount == 0 || !maskSizeLayerValueAccepted(strings.Join(sizeTokens[:sizeCount], " ")) {
			return emptyMaskShorthandLayer(), false
		}

		state.layer.size = normalizeCSSValue(strings.Join(sizeTokens[:sizeCount], " "))

		if !state.addTokens(sizeTokens[sizeCount:], false) {
			return emptyMaskShorthandLayer(), false
		}
	}

	applyMaskShorthandBoxDefaults(&state)

	return state.layer, true
}

func emptyMaskShorthandLayer() maskShorthandLayer {
	return maskShorthandLayer{
		image:     "",
		mode:      "",
		position:  "",
		size:      "",
		repeat:    "",
		origin:    "",
		clip:      "",
		composite: "",
	}
}

func applyMaskShorthandBoxDefaults(state *maskShorthandState) {
	if len(state.boxes) > 0 {
		state.layer.origin = state.boxes[0]
		state.layer.clip = state.boxes[0]
	}

	if len(state.boxes) == maskLayerComponentLimit {
		state.layer.clip = state.boxes[1]
	}

	if state.noClip {
		state.layer.clip = "no-clip"
	}
}

func (state *maskShorthandState) addTokens(tokens []string, allowPosition bool) bool {
	for _, token := range tokens {
		kind, value := classifyMaskShorthandToken(token)
		if !state.addToken(kind, value, allowPosition) {
			return false
		}
	}

	return true
}

func classifyMaskShorthandToken(token string) (maskShorthandTokenKind, string) {
	value := normalizeCSSValue(token)

	switch {
	case maskImageLayerValueAccepted(token):
		return maskTokenImage, strings.TrimSpace(token)
	case maskModeLayerValueAccepted(value):
		return maskTokenMode, value
	case maskRepeatLayerValueAccepted(value):
		return maskTokenRepeat, value
	case maskCompositeLayerValueAccepted(value):
		return maskTokenComposite, value
	case value == "no-clip":
		return maskTokenNoClip, value
	case maskOriginLayerValueAccepted(value):
		return maskTokenBox, value
	default:
		return maskTokenPosition, token
	}
}

func (state *maskShorthandState) addToken(kind maskShorthandTokenKind, value string, allowPosition bool) bool {
	switch kind {
	case maskTokenImage:
		return state.addImage(value)
	case maskTokenMode:
		return state.addMode(value)
	case maskTokenRepeat:
		return state.addRepeat(value)
	case maskTokenComposite:
		return state.addComposite(value)
	case maskTokenNoClip:
		return state.addNoClip()
	case maskTokenBox:
		return state.addBox(value)
	case maskTokenPosition:
		return state.addPosition(value, allowPosition)
	default:
		return false
	}
}

func (state *maskShorthandState) addImage(value string) bool {
	if state.sourceSeen {
		return false
	}

	state.sourceSeen = true
	state.layer.image = value

	return true
}

func (state *maskShorthandState) addMode(value string) bool {
	if state.modeSeen {
		return false
	}

	state.modeSeen = true
	state.layer.mode = value

	return true
}

func (state *maskShorthandState) addRepeat(value string) bool {
	if state.repeatSeen {
		return false
	}

	state.repeatSeen = true
	state.layer.repeat = value

	return true
}

func (state *maskShorthandState) addComposite(value string) bool {
	if state.compositeSeen {
		return false
	}

	state.compositeSeen = true
	state.layer.composite = value

	return true
}

func (state *maskShorthandState) addNoClip() bool {
	if state.noClip {
		return false
	}

	state.noClip = true

	return true
}

func (state *maskShorthandState) addBox(value string) bool {
	if len(state.boxes) == maskLayerComponentLimit {
		return false
	}

	state.boxes = append(state.boxes, value)

	return true
}

func (state *maskShorthandState) addPosition(token string, allowPosition bool) bool {
	if !allowPosition || len(state.positionTokens) == maskLayerComponentLimit {
		return false
	}

	state.positionTokens = append(state.positionTokens, token)

	return maskPositionLayerValueAccepted(strings.Join(state.positionTokens, " "))
}

//nolint:cyclop,funlen // One scanner must track nested functions, quotes, escapes, and a single top-level slash.
func splitMaskLayerSlash(value string) (string, string, bool, bool) {
	depth := 0
	quote := byte(0)
	escaped := false
	slash := -1

	for idx := range len(value) {
		char := value[idx]

		if escaped {
			escaped = false

			continue
		}

		if quote != 0 {
			switch char {
			case '\\':
				escaped = true
			case quote:
				quote = 0
			}

			continue
		}

		switch char {
		case '\'', '"':
			quote = char
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return "", "", false, false
			}
		case '/':
			if depth == 0 {
				if slash >= 0 {
					return "", "", false, false
				}

				slash = idx
			}
		}
	}

	if depth != 0 || quote != 0 {
		return "", "", false, false
	}

	if slash < 0 {
		return strings.TrimSpace(value), "", false, true
	}

	before := strings.TrimSpace(value[:slash])
	after := strings.TrimSpace(value[slash+1:])

	if before == "" || after == "" {
		return "", "", false, false
	}

	return before, after, true, true
}

func maskShorthandSizeTokenCount(tokens []string) int {
	if len(tokens) == 0 {
		return 0
	}

	if strings.EqualFold(tokens[0], "cover") || strings.EqualFold(tokens[0], "contain") {
		return 1
	}

	if !maskSizeDimensionValueAccepted(tokens[0]) {
		return 0
	}

	if len(tokens) > 1 && maskSizeDimensionValueAccepted(tokens[1]) {
		return maskLayerComponentLimit
	}

	return 1
}

func cssSpaceTokenList(value string) []string {
	tokens := make([]string, 0, cssSpaceTokenCapacity)

	for start := 0; ; {
		token, next, ok := nextSpaceToken(value, start)
		if !ok {
			return tokens
		}

		tokens = append(tokens, token)
		start = next
	}
}

func splitMaskLayerList(value string) ([]string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.HasPrefix(trimmed, ",") || strings.HasSuffix(trimmed, ",") {
		return nil, false
	}

	layers := splitParenArgs(trimmed, ',')
	if len(layers) == 0 {
		return nil, false
	}

	for _, layer := range layers {
		if strings.TrimSpace(layer) == "" {
			return nil, false
		}
	}

	return layers, true
}

func maskLayerListValueAccepted(value string, accepted func(string) bool) bool {
	layers, ok := splitMaskLayerList(value)
	if !ok {
		return false
	}

	for _, layer := range layers {
		if !accepted(strings.TrimSpace(layer)) {
			return false
		}
	}

	return true
}

func maskImageLayerValueAccepted(value string) bool {
	trimmed := strings.TrimSpace(value)
	if strings.EqualFold(trimmed, "none") {
		return true
	}

	if maskGradientAccepted(trimmed) {
		return true
	}

	if _, ok := maskImageSetChoice(trimmed); ok {
		return true
	}

	if !cssURLFunctionValue(trimmed) {
		return false
	}

	_, ok := firstCSSUrl(trimmed)

	return ok
}

//nolint:cyclop // URL validation tracks nested parentheses, quoted text, and escapes.
func cssURLFunctionValue(value string) bool {
	if len(value) < len("url()") || !strings.EqualFold(value[:4], "url(") {
		return false
	}

	depth := 1
	quote := byte(0)
	escaped := false

	for idx := 4; idx < len(value); idx++ {
		char := value[idx]

		if escaped {
			escaped = false

			continue
		}

		if quote != 0 {
			switch char {
			case '\\':
				escaped = true
			case quote:
				quote = 0
			}

			continue
		}

		switch char {
		case '\'', '"':
			quote = char
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return strings.TrimSpace(value[idx+1:]) == ""
			}
		}
	}

	return false
}

func maskModeLayerValueAccepted(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case maskModeMatchSource, maskModeAlpha, maskModeLuminance:
		return true
	default:
		return false
	}
}

func maskTypeValueAccepted(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case maskModeAlpha, maskModeLuminance:
		return true
	default:
		return false
	}
}

func maskPositionLayerValueAccepted(value string) bool {
	_, _, ok := parseObjectPosition(value)

	return ok
}

func maskSizeLayerValueAccepted(value string) bool {
	tokens := cssSpaceTokenList(value)
	if len(tokens) == 1 {
		switch strings.ToLower(tokens[0]) {
		case "auto", "cover", "contain":
			return true
		}
	}

	if len(tokens) < 1 || len(tokens) > 2 {
		return false
	}

	for _, token := range tokens {
		if !maskSizeDimensionValueAccepted(token) {
			return false
		}
	}

	return true
}

func maskSizeDimensionValueAccepted(value string) bool {
	if strings.EqualFold(strings.TrimSpace(value), "auto") {
		return true
	}

	length, ok := lengthBox(value, 0, 0, "")

	return ok && length >= 0
}

func maskRepeatLayerValueAccepted(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "repeat", "repeat-x", "repeat-y", maskRepeatNoRepeat:
		return true
	default:
		return false
	}
}

func maskOriginLayerValueAccepted(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "border-box", "padding-box", "content-box":
		return true
	default:
		return false
	}
}

func maskClipLayerValueAccepted(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "no-clip") || maskOriginLayerValueAccepted(value)
}

func maskCompositeLayerValueAccepted(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case maskCompositeAdd, maskCompositeSubtract, "intersect", maskCompositeExclude:
		return true
	default:
		return false
	}
}

func webkitMaskCompositeValueAccepted(value string) bool {
	return maskLayerListValueAccepted(value, func(layer string) bool {
		switch strings.ToLower(strings.TrimSpace(layer)) {
		case "source-over", "source-out", "source-in", "xor":
			return true
		default:
			return false
		}
	})
}

func remapWebkitMaskCompositeValue(value string) string {
	layers, ok := splitMaskLayerList(value)
	if !ok {
		return value
	}

	for idx, layer := range layers {
		switch strings.ToLower(strings.TrimSpace(layer)) {
		case "source-over":
			layers[idx] = "add"
		case "source-out":
			layers[idx] = "subtract"
		case "source-in":
			layers[idx] = "intersect"
		case "xor":
			layers[idx] = "exclude"
		default:
			return value
		}
	}

	return strings.Join(layers, ", ")
}
