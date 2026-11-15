package layout

import (
	"strconv"
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// setBackdropFilterValue stores a valid author string unchanged.
// none and a blur/saturate list are kept as written. A bad token is ignored
// so an earlier valid list stays in place. Pixel blur and saturation are not
// painted here; the host reads the stored string.
func setBackdropFilterValue(style *ResolvedStyle, value string) {
	trimmed := strings.TrimSpace(value)
	if !backdropFilterValueAccepted(trimmed) {
		return
	}

	style.BackdropFilter = trimmed
}

// backdropFilterValueAccepted reports whether value is none or a list of
// blur(<length>) and saturate(<number> | <percentage>) functions, in order.
// The caller stores the author text when this is true. It does not rewrite
// units or drop a second function.
func backdropFilterValueAccepted(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}

	if strings.EqualFold(value, cssDisplayNone) {
		return true
	}

	rest := value

	for rest != "" {
		name, args, next, ok := splitTransformFunc(rest)
		if !ok || !backdropFilterFuncAccepted(name, args) {
			return false
		}

		rest = next
	}

	return true
}

func backdropFilterFuncAccepted(name, args string) bool {
	switch name {
	case "blur":
		return backdropBlurAccepted(args)
	case "saturate":
		return backdropSaturateAccepted(args)
	default:
		return false
	}
}

func backdropBlurAccepted(args string) bool {
	amount, _, ok := css.ParseLength(args)
	if !ok {
		return false
	}

	return amount >= 0
}

func backdropSaturateAccepted(args string) bool {
	token := strings.TrimSpace(args)
	if token == "" {
		return false
	}

	if before, found := strings.CutSuffix(token, "%"); found {
		number, err := strconv.ParseFloat(strings.TrimSpace(before), 64)

		return err == nil && number >= 0
	}

	number, ok := css.ParseNumber(token)

	return ok && number >= 0
}
