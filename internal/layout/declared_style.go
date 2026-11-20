package layout

import "strings"

// declaredProperty reports whether prop is stored in declaredStyle.
func declaredProperty(prop string) bool {
	return declaredGroupOf(prop) != ""
}

// normalizeDeclared hands a value to the normalizer of its group.
func normalizeDeclared(prop, value string) (string, bool) {
	switch declaredGroupOf(prop) {
	case declaredGroupNav:
		return normalizeNavDeclared(prop, value)
	case declaredGroupRead:
		return normalizeReadDeclared(prop, value)
	case declaredGroupShapes:
		return normalizeShapesDeclared(prop, value)
	case declaredGroupSvg:
		return normalizeSvgDeclared(prop, value)
	case declaredGroupMotion:
		return normalizeMotionDeclared(prop, value)
	case declaredGroupFrag:
		return normalizeFragDeclared(prop, value)
	default:
		return "", false
	}
}

// declaredDeclarationValueAccepted is the cascade gate. It reports (accepted,
// handled); handled is false for a property this file does not own. A wide
// keyword and a var() reference are valid at parse time.
func declaredDeclarationValueAccepted(prop, value string) (bool, bool) {
	if !declaredProperty(prop) {
		return false, false
	}

	trimmed := strings.TrimSpace(value)
	if cssWideKeyword(strings.ToLower(trimmed)) || containsVarFunc(trimmed) {
		return true, true
	}

	_, ok := normalizeDeclared(prop, trimmed)

	return ok, true
}

// declaredTakesParent reports whether a keyword value means "copy the parent
// value" for prop.
func declaredTakesParent(prop, keyword string) bool {
	return keyword == inheritKeyword || (keyword == cssKeywordUnset && declaredInheritsByDefault(prop))
}

// applyDeclaredProps stores one declared property on style. It returns false
// for a property it does not own.
func applyDeclaredProps(style *ResolvedStyle, prop, value string, parent *ResolvedStyle, hasParent bool) bool {
	field := declaredField(&style.Declared, prop)
	if field == nil {
		return false
	}

	keyword := strings.ToLower(strings.TrimSpace(value))
	takeParent := declaredTakesParent(prop, keyword)

	switch {
	case takeParent && hasParent && parent != nil:
		*field = *declaredField(&parent.Declared, prop)
	case takeParent || keyword == cssKeywordInitial || keyword == cssKeywordUnset:
		*field = ""
	default:
		if text, ok := normalizeDeclared(prop, value); ok {
			*field = text
		}
	}

	return true
}

// inheritDeclaredProps copies the inheriting properties from the parent
// unless the element declares them.
func inheritDeclaredProps(dst, parent *ResolvedStyle, raw map[string]string) {
	if parent == nil || parent.Declared == (declaredStyle{}) {
		return
	}

	for _, prop := range declaredInheritedNames() {
		if _, ok := raw[prop]; ok {
			continue
		}

		*declaredField(&dst.Declared, prop) = *declaredField(&parent.Declared, prop)
	}
}

// declaredValues returns the non-empty declared values of one style by
// property name, or nil when the element declares none.
func declaredValues(style *ResolvedStyle) map[string]string {
	if style == nil || style.Declared == (declaredStyle{}) {
		return nil
	}

	var out map[string]string

	for _, prop := range declaredPropNames() {
		if text := *declaredField(&style.Declared, prop); text != "" {
			if out == nil {
				out = map[string]string{}
			}

			out[prop] = text
		}
	}

	return out
}

// familyDeclarationValueAccepted is the cascade gate for the trigger family
// and the declared-only properties, tried in that order.
func familyDeclarationValueAccepted(prop, value string) (bool, bool) {
	if accepted, handled := triggerDeclarationValueAccepted(prop, value); handled {
		return accepted, true
	}

	return declaredDeclarationValueAccepted(prop, value)
}
