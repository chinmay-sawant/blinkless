package layout

import (
	"strings"
	"sync"
)

// allSentinelInitial and allSentinelInherit are private cascade values.
// Appliers that see the keyword "initial" leave an inherited color in place,
// so the shorthand records a sentinel, blocks or allows inheritance, then
// drops the sentinel before those appliers run.
const (
	allSentinelInitial = "\x00all-initial"
	allSentinelInherit = "\x00all-inherit"
)

// Property names the all expander compares against.
const (
	propBackgroundColor = "background-color"
	propUnicodeBidi     = "unicode-bidi"
	propDirection       = "direction"
)

// expandAllDeclaration expands the all shorthand into one longhand per
// property it resets. direction, unicode-bidi, and custom properties are
// skipped. revert, revert-layer, and revert-rule are not expanded: this
// cascade does not walk origins. A value that is not a single wide keyword
// returns false so the declaration is ignored and an earlier longhand stays.
func expandAllDeclaration(prop, value string) ([]logicalPropDecl, bool) {
	if prop != "all" {
		return nil, false
	}

	keyword, ok := allWideKeyword(value)
	if !ok {
		return nil, false
	}

	names := allResetPropertyNames()
	out := make([]logicalPropDecl, 0, len(names))

	for _, name := range names {
		out = append(out, logicalPropDecl{prop: name, val: allSentinelFor(keyword, name)})
	}

	return out, true
}

// allWideKeyword accepts initial, inherit, and unset only. Extra tokens and
// the revert family fail so they do not pretend to reset longhands.
func allWideKeyword(value string) (string, bool) {
	fields := strings.Fields(value)
	if len(fields) != 1 {
		return "", false
	}

	switch strings.ToLower(fields[0]) {
	case cssKeywordInitial, inheritKeyword, cssKeywordUnset:
		return strings.ToLower(fields[0]), true
	default:
		return "", false
	}
}

// allSentinelFor maps one reset property to the sentinel inheritProps and
// the background copy understand. unset follows the inherit table: inherited
// properties copy the parent, others stay at the initial value.
func allSentinelFor(keyword, name string) string {
	switch keyword {
	case inheritKeyword:
		return allSentinelInherit
	case cssKeywordUnset:
		if allPropertyInherited(name) {
			return allSentinelInherit
		}

		return allSentinelInitial
	default:
		return allSentinelInitial
	}
}

// applyAllProps consumes a leftover all declaration. Expansion already
// rewrote initial, inherit, and unset. revert, revert-layer, revert-rule,
// and a bad token reach here and do not walk cascade origins.
func applyAllProps(
	_ *ResolvedStyle, prop, _ string, _ float64, _ *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	return prop == "all"
}

// takeAllInheritSentinels removes inherit sentinels so inheritProps copies
// those properties. It reports whether background-color asked for the parent
// fill. background-color is not in the inherit table, so the caller copies
// that field itself.
func takeAllInheritSentinels(raw map[string]string) bool {
	bgInherit := false

	for prop, value := range raw {
		if value != allSentinelInherit {
			continue
		}

		if prop == propBackgroundColor {
			bgInherit = true
		}

		delete(raw, prop)
	}

	return bgInherit
}

// dropAllInitialSentinels removes initial sentinels after inheritProps has
// seen them. Leaving the key in place through that copy is what stops the
// parent color from sticking. Appliers must not parse the sentinel.
func dropAllInitialSentinels(raw map[string]string) {
	for prop, value := range raw {
		if value == allSentinelInitial {
			delete(raw, prop)
		}
	}
}

// The three values below are a lazy cache built once by allResetOnce.
var (
	allResetOnce  sync.Once
	allResetNames []string
	allInherited  map[string]struct{}
)

func allResetPropertyNames() []string {
	allResetOnce.Do(buildAllResetProperties)

	return allResetNames
}

func allPropertyInherited(name string) bool {
	allResetOnce.Do(buildAllResetProperties)

	_, ok := allInherited[name]

	return ok
}

// buildAllResetProperties lists every property all resets. direction and
// unicode-bidi stay out, and so does any custom property. The inherit set is
// the same names inheritProps already copies, plus background-color, which
// all: inherit assigns even though it is not inherited.
func buildAllResetProperties() {
	allInherited = make(map[string]struct{})

	add := func(name string, inherited bool) {
		if name == propDirection || name == propUnicodeBidi || strings.HasPrefix(name, "--") {
			return
		}

		if inherited {
			allInherited[name] = struct{}{}
		}

		for _, existing := range allResetNames {
			if existing == name {
				return
			}
		}

		allResetNames = append(allResetNames, name)
	}

	for _, entry := range inheritableProps {
		for _, name := range entry.names {
			add(name, true)
		}
	}

	for _, name := range []string{
		"fill-rule", "shape-rendering", "dominant-baseline",
		"color-interpolation", "color-interpolation-filters",
		"line-fit-edge", "line-padding", inlineSizingProp,
	} {
		add(name, true)
	}

	add("background-color", false)
}
