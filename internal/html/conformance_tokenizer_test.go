//nolint:all // conformance harness: corpus format parsing and diff plumbing, not product code
package html

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// --- tokenizer category ---

type tokenizerTestFile struct {
	Tests             []tokenizerCase `json:"tests"`
	XMLViolationTests []tokenizerCase `json:"xmlViolationTests"`
}

type tokenizerCase struct {
	Description   string            `json:"description"`
	Input         string            `json:"input"`
	Output        []json.RawMessage `json:"output"`
	InitialStates []string          `json:"initialStates"`
	LastStartTag  string            `json:"lastStartTag"`
	DoubleEscaped bool              `json:"doubleEscaped"`
}

func runTokenizerCategory(t *testing.T, plan conformanceCategoryPlan, report *conformanceReport) {
	t.Helper()

	for _, path := range plan.files {
		rel := relativeSlash(report.dir, path)
		fileReason := plan.fileReasons[rel]

		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("conformance harness: read %s: %v", rel, err)
		}

		var file tokenizerTestFile

		if err := json.Unmarshal(raw, &file); err != nil {
			t.Fatalf("conformance harness: parse %s: %v", rel, err)
		}

		for i := range file.XMLViolationTests {
			id := fmt.Sprintf("%s#%d/xmlViolation", rel, i+1)
			report.addCase(plan.category.ID, id, statusUnsupported, tokenizerFileReason(plan, fileReason, true), "")
		}

		for i := range file.Tests {
			id := fmt.Sprintf("%s#%d", rel, i+1)
			reason := tokenizerFileReason(plan, fileReason, false)

			if reason != "" {
				report.addCase(plan.category.ID, id, statusUnsupported, reason, "")
				continue
			}

			runTokenizerCase(t, file.Tests[i], id, plan.category.ID, report)
		}
	}
}

func tokenizerFileReason(plan conformanceCategoryPlan, fileReason string, xmlViolation bool) string {
	switch {
	case plan.wholeReason != "":
		return plan.wholeReason
	case fileReason != "":
		return fileReason
	case xmlViolation:
		return reasonXMLViolation
	default:
		return ""
	}
}

func runTokenizerCase(t *testing.T, tc tokenizerCase, id, category string, report *conformanceReport) {
	t.Helper()

	states := tc.InitialStates
	if len(states) == 0 {
		states = []string{dataStateName}
	}

	for _, state := range states {
		caseID := id + "/" + stateSlug(state)
		if state != dataStateName {
			report.addCase(category, caseID, statusUnsupported, reasonInitialState+state, "")
			continue
		}

		classifyTokenizerCase(t, tc, caseID, category, report)
	}
}

func classifyTokenizerCase(t *testing.T, tc tokenizerCase, id, category string, report *conformanceReport) {
	t.Helper()

	input := tc.Input

	if tc.DoubleEscaped {
		unescaped, ok := unescapeDoubleEscaped(input)
		if !ok {
			report.addCase(category, id, statusUnsupported, reasonLoneSurrogate, "")
			return
		}

		input = unescaped
	}

	expected, hasPI := parseExpectedTokenStream(t, tc.Output, tc.DoubleEscaped)
	if hasPI {
		report.addCase(category, id, statusUnsupported, reasonProcessingInstr, "")
		return
	}

	toks, err := tokenize(input)
	if err != nil {
		report.addCase(category, id, statusFailed, reasonEngineError+err.Error(), "")
		return
	}

	got := engineTokenStream(toks)

	if mismatch := compareTokenStreams(expected, got); mismatch != nil {
		report.addCase(category, id, statusFailed, "token-stream-mismatch", mismatchDetail(mismatch))
		return
	}

	report.addCase(category, id, statusPassed, "", "")
}

func stateSlug(state string) string {
	return strings.ReplaceAll(state, " ", "-")
}

// --- tokenizer expectations ---

func parseExpectedTokenStream(t *testing.T, raws []json.RawMessage, doubleEscaped bool) ([]cfToken, bool) {
	t.Helper()

	tokens := make([]cfToken, 0, len(raws))
	hasPI := false

	for i, raw := range raws {
		tok, pi := parseExpectedToken(t, i, raw, doubleEscaped)

		if pi {
			hasPI = true
			continue
		}

		tokens = append(tokens, tok)
	}

	return tokens, hasPI
}

func parseExpectedToken(t *testing.T, index int, raw json.RawMessage, doubleEscaped bool) (cfToken, bool) {
	t.Helper()

	var parts []json.RawMessage
	if err := json.Unmarshal(raw, &parts); err != nil {
		t.Fatalf("conformance harness: token %d: %v", index, err)
	}

	if len(parts) == 0 {
		t.Fatalf("conformance harness: token %d is empty", index)
	}

	typ := unescapeTokenString(t, jsonString(t, parts[0]), doubleEscaped)

	switch typ {
	case tokenCharacter, tokenComment:
		requireTokenParts(t, index, typ, parts, 2)

		data := unescapeTokenString(t, jsonString(t, parts[1]), doubleEscaped)

		return cfToken{kind: typ, data: data}, false
	case tokenEndTag:
		requireTokenParts(t, index, typ, parts, 2)

		name := unescapeTokenString(t, jsonString(t, parts[1]), doubleEscaped)

		return cfToken{kind: typ, name: name}, false
	case tokenStartTag:
		if len(parts) < 3 {
			t.Fatalf("conformance harness: token %d: StartTag needs a name and attributes", index)
		}

		name := unescapeTokenString(t, jsonString(t, parts[1]), doubleEscaped)
		attrs := parseExpectedAttrs(t, index, parts[2], doubleEscaped)
		selfClosing := len(parts) >= 4 && jsonBool(t, parts[3])

		return cfToken{kind: typ, name: name, attrs: attrs, selfClosing: selfClosing}, false
	case tokenDoctype:
		requireTokenParts(t, index, typ, parts, 5)

		name := unescapeTokenString(t, jsonNullableString(t, parts[1]), doubleEscaped)
		public, publicSet := jsonNullableStringSet(t, parts[2])
		system, systemSet := jsonNullableStringSet(t, parts[3])
		public = unescapeTokenString(t, public, doubleEscaped)
		system = unescapeTokenString(t, system, doubleEscaped)
		correct := jsonBool(t, parts[4])

		return cfToken{kind: typ, doctype: cfDoctype{
			name:        name,
			public:      public,
			publicSet:   publicSet,
			system:      system,
			systemSet:   systemSet,
			forceQuirks: !correct,
		}}, false
	case "ProcessingInstruction":
		return cfToken{}, true
	default:
		t.Fatalf("conformance harness: token %d has unknown type %q", index, typ)
	}

	return cfToken{}, false
}

func requireTokenParts(t *testing.T, index int, typ string, parts []json.RawMessage, want int) {
	t.Helper()

	if len(parts) != want {
		t.Fatalf("conformance harness: token %d (%s) has %d fields, want %d", index, typ, len(parts), want)
	}
}

func parseExpectedAttrs(t *testing.T, index int, raw json.RawMessage, doubleEscaped bool) []cfAttr {
	t.Helper()

	var attrsMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &attrsMap); err != nil {
		t.Fatalf("conformance harness: token %d attributes: %v", index, err)
	}

	attrs := make([]cfAttr, 0, len(attrsMap))

	for name, valueRaw := range attrsMap {
		value := ""

		if string(valueRaw) != "null" {
			value = jsonString(t, valueRaw)
		}

		attrs = append(attrs, cfAttr{
			name:  unescapeTokenString(t, name, doubleEscaped),
			value: unescapeTokenString(t, value, doubleEscaped),
		})
	}

	sortCFAttrs(attrs)

	return attrs
}

func jsonString(t *testing.T, raw json.RawMessage) string {
	t.Helper()

	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("conformance harness: expected a JSON string, got %s: %v", raw, err)
	}

	return s
}

func jsonNullableString(t *testing.T, raw json.RawMessage) string {
	t.Helper()

	s, _ := jsonNullableStringSet(t, raw)

	return s
}

func jsonNullableStringSet(t *testing.T, raw json.RawMessage) (string, bool) {
	t.Helper()

	var s *string
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("conformance harness: expected a JSON string or null, got %s: %v", raw, err)
	}

	if s == nil {
		return "", false
	}

	return *s, true
}

func jsonBool(t *testing.T, raw json.RawMessage) bool {
	t.Helper()

	var b bool
	if err := json.Unmarshal(raw, &b); err != nil {
		t.Fatalf("conformance harness: expected a JSON bool, got %s: %v", raw, err)
	}

	return b
}

func unescapeTokenString(t *testing.T, s string, doubleEscaped bool) string {
	t.Helper()

	if !doubleEscaped {
		return s
	}

	out, ok := unescapeDoubleEscaped(s)
	if !ok {
		t.Fatalf("conformance harness: lone surrogate in expected token string %q", s)
	}

	return out
}

// unescapeDoubleEscaped applies the html5lib-tests doubleEscaped rule: every
// literal \uHHHH sequence becomes the code point it names. Surrogate pairs
// combine; a lone surrogate reports ok=false because a Go string cannot carry
// it faithfully (those cases are counted unsupported, never silently passed).
func unescapeDoubleEscaped(s string) (string, bool) {
	if !strings.Contains(s, `\u`) {
		return s, true
	}

	var b strings.Builder

	b.Grow(len(s))

	for i := 0; i < len(s); {
		if s[i] == '\\' && i+1 < len(s) && s[i+1] == 'u' && i+6 <= len(s) {
			if r, ok := parseUnicodeEscape(s[i+2 : i+6]); ok {
				i += 6

				if isHighSurrogate(r) && i+6 <= len(s) && s[i] == '\\' && s[i+1] == 'u' {
					if low, lowOK := parseUnicodeEscape(s[i+2 : i+6]); lowOK && isLowSurrogate(low) {
						r = 0x10000 + ((r - 0xD800) << 10) + (low - 0xDC00)
						i += 6
					}
				}

				if isSurrogate(r) {
					return "", false
				}

				b.WriteRune(r)

				continue
			}
		}

		b.WriteByte(s[i])
		i++
	}

	return b.String(), true
}

func parseUnicodeEscape(hex string) (rune, bool) {
	value, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return 0, false
	}

	return rune(value), true
}

func isHighSurrogate(r rune) bool {
	return r >= 0xD800 && r <= 0xDBFF
}

func isLowSurrogate(r rune) bool {
	return r >= 0xDC00 && r <= 0xDFFF
}

func isSurrogate(r rune) bool {
	return r >= 0xD800 && r <= 0xDFFF
}

// --- engine token stream ---

func engineTokenStream(toks []token) []cfToken {
	out := make([]cfToken, 0, len(toks))

	for _, tok := range toks {
		switch tok.kind {
		case tokDoctype:
			out = append(out, cfToken{kind: tokenDoctype, doctype: cfDoctype{
				name:        tok.doctype.Name,
				public:      tok.doctype.PublicID,
				publicSet:   tok.doctype.HasPublicID,
				system:      tok.doctype.SystemID,
				systemSet:   tok.doctype.HasSystemID,
				forceQuirks: tok.doctype.ForceQuirks,
			}})
		case tokStart:
			out = append(out, cfToken{kind: tokenStartTag, name: tok.data, attrs: pairAttrs(tok.attrs), selfClosing: tok.selfClosing})
		case tokEnd:
			out = append(out, cfToken{kind: tokenEndTag, name: tok.data})
		case tokText:
			out = append(out, cfToken{kind: tokenCharacter, data: tok.data})
		case tokComment:
			out = append(out, cfToken{kind: tokenComment, data: tok.data})
		}
	}

	return out
}

func pairAttrs(pairs []string) []cfAttr {
	seen := make(map[string]string, len(pairs)/cfAttrPairSize)
	order := make([]string, 0, len(pairs)/cfAttrPairSize)

	for i := 0; i+1 < len(pairs); i += cfAttrPairSize {
		name := pairs[i]
		if _, dup := seen[name]; dup {
			continue
		}

		seen[name] = pairs[i+1]
		order = append(order, name)
	}

	attrs := make([]cfAttr, 0, len(order))
	for _, name := range order {
		attrs = append(attrs, cfAttr{name: name, value: seen[name]})
	}

	sortCFAttrs(attrs)

	return attrs
}

// --- token stream comparison ---

func compareTokenStreams(want, got []cfToken) *cfMismatch {
	want = coalesceCharacterTokens(want)
	got = coalesceCharacterTokens(got)

	if len(want) != len(got) {
		return &cfMismatch{
			path: "token count",
			want: renderTokenStream(want),
			got:  renderTokenStream(got),
		}
	}

	for i := range want {
		if mismatch := compareToken(want[i], got[i], i); mismatch != nil {
			return mismatch
		}
	}

	return nil
}

func compareToken(want, got cfToken, index int) *cfMismatch {
	path := fmt.Sprintf("token[%d]", index)

	if want.kind != got.kind {
		return &cfMismatch{path: path, want: []string{want.kind}, got: []string{got.kind}}
	}

	switch want.kind {
	case tokenCharacter, tokenComment:
		if want.data != got.data {
			return &cfMismatch{path: path, want: []string{quoteDump(want.data)}, got: []string{quoteDump(got.data)}}
		}
	case tokenEndTag:
		if want.name != got.name {
			return &cfMismatch{path: path, want: []string{want.name}, got: []string{got.name}}
		}
	case tokenStartTag:
		if want.name != got.name {
			return &cfMismatch{path: path, want: []string{want.name}, got: []string{got.name}}
		}

		if want.selfClosing != got.selfClosing {
			return &cfMismatch{path: path + " self-closing", want: []string{strconv.FormatBool(want.selfClosing)}, got: []string{strconv.FormatBool(got.selfClosing)}}
		}

		if mismatch := compareTokenAttrs(want.attrs, got.attrs, path); mismatch != nil {
			return mismatch
		}
	case tokenDoctype:
		return compareTokenDoctype(want, got, path)
	}

	return nil
}

func compareTokenAttrs(want, got []cfAttr, path string) *cfMismatch {
	if len(want) != len(got) {
		return &cfMismatch{path: path + " attributes", want: renderAttrs(want), got: renderAttrs(got)}
	}

	for _, wantAttr := range want {
		gotAttr, ok := findCFAttr(got, wantAttr.ns, wantAttr.name)
		if !ok || gotAttr.value != wantAttr.value {
			return &cfMismatch{
				path: path + " attribute " + wantAttr.name,
				want: []string{quoteDump(wantAttr.value)},
				got:  []string{quoteDump(attrValueOrMissing(gotAttr, ok))},
			}
		}
	}

	return nil
}

func compareTokenDoctype(want, got cfToken, path string) *cfMismatch {
	if want.doctype.forceQuirks != got.doctype.forceQuirks {
		return &cfMismatch{path: path + " force-quirks", want: []string{renderDoctype(want.doctype)}, got: []string{renderDoctype(got.doctype)}}
	}

	if !sameDoctype(want.doctype, got.doctype) {
		return &cfMismatch{path: path, want: []string{renderDoctype(want.doctype)}, got: []string{renderDoctype(got.doctype)}}
	}

	return nil
}

func sameDoctype(want, got cfDoctype) bool {
	return strings.EqualFold(want.name, got.name) &&
		want.publicSet == got.publicSet && want.public == got.public &&
		want.systemSet == got.systemSet && want.system == got.system
}

func coalesceCharacterTokens(tokens []cfToken) []cfToken {
	out := make([]cfToken, 0, len(tokens))

	for _, tok := range tokens {
		if tok.kind == tokenCharacter && len(out) > 0 && out[len(out)-1].kind == tokenCharacter {
			out[len(out)-1].data += tok.data

			continue
		}

		out = append(out, tok)
	}

	return out
}

func renderTokenStream(tokens []cfToken) []string {
	lines := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		lines = append(lines, renderToken(tok))
	}

	return lines
}

func renderToken(tok cfToken) string {
	switch tok.kind {
	case tokenCharacter:
		return tokenCharacter + " " + quoteDump(tok.data)
	case tokenComment:
		return tokenComment + " " + quoteDump(tok.data)
	case tokenEndTag:
		return tokenEndTag + " " + tok.name
	case tokenStartTag:
		var b strings.Builder

		b.WriteString(tokenStartTag + " " + tok.name)

		for _, attr := range tok.attrs {
			b.WriteString(" " + attr.name + "=" + quoteDump(attr.value))
		}

		if tok.selfClosing {
			b.WriteString(" self-closing")
		}

		return b.String()
	case tokenDoctype:
		if tok.doctype.forceQuirks {
			return renderDoctype(tok.doctype) + " force-quirks"
		}

		return renderDoctype(tok.doctype)
	default:
		return tok.kind
	}
}
