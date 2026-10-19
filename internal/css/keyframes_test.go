package css

import "testing"

const selectorListValue = "middle"

func TestKeyframesPreserveOffsetsAndDeclarations(t *testing.T) {
	t.Parallel()

	sheet := mustSheet(t, `
		@keyframes slide {
			from {
				opacity: 0;
				--host-value: calc(2 * 3);
				unknown-to-layout: 4px;
				unknown-to-layout: 5px;
				color: red !important;
			}
			to {
				opacity: 1;
				animation-timing-function: ease-out;
			}
		}
		p { color: blue }
	`)

	if len(sheet.Keyframes) != 1 {
		t.Fatalf("got %d keyframes rules, want 1", len(sheet.Keyframes))
	}

	rule := sheet.Keyframes[0]
	if rule.Name != "slide" {
		t.Fatalf("keyframes name = %q, want slide", rule.Name)
	}

	if len(rule.Keyframes) != 2 {
		t.Fatalf("got %d keyframes, want 2: %+v", len(rule.Keyframes), rule.Keyframes)
	}

	assertFromKeyframeDeclarations(t, rule.Keyframes[0])
	assertToKeyframeDeclarations(t, rule.Keyframes[1])
	assertStylesheetContinuesAfterKeyframes(t, sheet)
}

func assertFromKeyframeDeclarations(t *testing.T, startKeyframe Keyframe) {
	t.Helper()

	if startKeyframe.Offset != 0 {
		t.Errorf("from offset = %v, want 0", startKeyframe.Offset)
	}

	if len(startKeyframe.Declarations) != 5 {
		t.Fatalf("from declarations = %+v, want 5", startKeyframe.Declarations)
	}

	wantDeclarations := []Declaration{
		{Prop: "opacity", Value: "0"},
		{Prop: "--host-value", Value: "calc(2 * 3)"},
		{Prop: "unknown-to-layout", Value: "4px"},
		{Prop: "unknown-to-layout", Value: "5px"},
		{Prop: "color", Value: "red", Important: true},
	}

	for idx, want := range wantDeclarations {
		if got := startKeyframe.Declarations[idx]; got != want {
			t.Errorf("from declaration %d = %+v, want %+v", idx, got, want)
		}
	}
}

func assertToKeyframeDeclarations(t *testing.T, endKeyframe Keyframe) {
	t.Helper()

	if endKeyframe.Offset != 1 {
		t.Errorf("to offset = %v, want 1", endKeyframe.Offset)
	}

	wantTimingFunction := Declaration{Prop: "animation-timing-function", Value: "ease-out"}
	if len(endKeyframe.Declarations) != 2 || endKeyframe.Declarations[1] != wantTimingFunction {
		t.Errorf("to declarations = %+v", endKeyframe.Declarations)
	}
}

func assertStylesheetContinuesAfterKeyframes(t *testing.T, sheet *Stylesheet) {
	t.Helper()

	if len(sheet.Rules) != 1 || sheet.Rules[0].Selectors[0].Parts[0].Tag != "p" {
		t.Errorf("parser did not resume after keyframes: %+v", sheet.Rules)
	}
}

func TestKeyframesSelectorListsAndDocumentOrder(t *testing.T) {
	t.Parallel()

	sheet := mustSheet(t, `
		@keyframes move {
			25%, 75% { --offset: `+selectorListValue+` }
			50% { --offset: first }
			50% { --offset: second }
		}
		@-webkit-keyframes move { to { --vendor: kept } }
		@keyframes "initial" { 0%, 12.5% { --quoted-name: kept } }
	`)

	if len(sheet.Keyframes) != 3 {
		t.Fatalf("got %d keyframes rules, want 3", len(sheet.Keyframes))
	}

	assertKeyframeRuleNames(t, sheet.Keyframes)
	assertSelectorListOffsetsAndDeclarations(t, sheet.Keyframes[0].Keyframes)
	assertVendorKeyframeOffset(t, sheet.Keyframes[1])
	assertQuotedKeyframes(t, sheet.Keyframes[2])
}

func assertKeyframeRuleNames(t *testing.T, rules []KeyframesRule) {
	t.Helper()

	wantNames := []string{"move", "move", "initial"}
	for idx, want := range wantNames {
		if got := rules[idx].Name; got != want {
			t.Errorf("rule %d name = %q, want %q", idx, got, want)
		}
	}
}

func assertSelectorListOffsetsAndDeclarations(t *testing.T, keyframes []Keyframe) {
	t.Helper()

	wantOffsets := []float64{0.25, 0.75, 0.5, 0.5}
	if len(keyframes) != len(wantOffsets) {
		t.Fatalf("first rule has %d keyframes, want %d: %+v", len(keyframes), len(wantOffsets), keyframes)
	}

	for idx, want := range wantOffsets {
		if got := keyframes[idx].Offset; got != want {
			t.Errorf("keyframe %d offset = %v, want %v", idx, got, want)
		}
	}

	if keyframes[0].Declarations[0].Value != selectorListValue ||
		keyframes[1].Declarations[0].Value != selectorListValue {
		t.Errorf("selector-list declarations = %+v, want both offsets to keep %q",
			keyframes[:2], selectorListValue)
	}

	keyframes[0].Declarations[0].Value = "changed"

	if got := keyframes[1].Declarations[0].Value; got != selectorListValue {
		t.Errorf("selector-list declaration slices alias: second value = %q", got)
	}
}

func assertVendorKeyframeOffset(t *testing.T, rule KeyframesRule) {
	t.Helper()

	if got := rule.Keyframes[0].Offset; got != 1 {
		t.Errorf("vendor to offset = %v, want 1", got)
	}
}

func assertQuotedKeyframes(t *testing.T, rule KeyframesRule) {
	t.Helper()

	quotedKeyframes := rule.Keyframes
	if len(quotedKeyframes) != 2 || quotedKeyframes[1].Offset != 0.125 {
		t.Errorf("quoted-name keyframes = %+v", quotedKeyframes)
	}
}

func TestKeyframesIgnoreInvalidSelectorBlocks(t *testing.T) {
	t.Parallel()

	sheet := mustSheet(t, `
		@keyframes bounds {
			-1% { --offset: below }
			100.01% { --offset: above }
			0 { --offset: missing-percent-unit }
			0%, 101% { --offset: mixed-invalid }
			from { --offset: valid }
		}
	`)

	if len(sheet.Keyframes) != 1 || len(sheet.Keyframes[0].Keyframes) != 1 {
		t.Fatalf("invalid selector blocks were kept: %+v", sheet.Keyframes)
	}

	keyframe := sheet.Keyframes[0].Keyframes[0]
	if keyframe.Offset != 0 || len(keyframe.Declarations) != 1 ||
		keyframe.Declarations[0] != (Declaration{Prop: "--offset", Value: "valid"}) {
		t.Errorf("valid from keyframe = %+v", keyframe)
	}
}
