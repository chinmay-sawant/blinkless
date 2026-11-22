package layout

import (
	"reflect"
	"testing"
)

func resolveContainmentProp(prop, value string) (ResolvedStyle, bool) {
	st := initialStyle()
	ok := applyContainmentProps(&st, prop, value, 12, nil, nil, false)

	return st, ok
}

func TestApplyContainmentPropsParsing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		prop   string
		value  string
		wantOK bool
		check  func(t *testing.T, st ResolvedStyle)
	}{
		{
			name: "contain-none", prop: "contain", value: "none", wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().Contain != "none" {
					t.Errorf("Contain = %q, want none", st.containment().Contain)
				}
			},
		},
		{
			name: "contain-strict", prop: "contain", value: "strict", wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().Contain != "strict" {
					t.Errorf("Contain = %q, want strict", st.containment().Contain)
				}
			},
		},
		{
			name: "contain-content", prop: "contain", value: "content", wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().Contain != "content" {
					t.Errorf("Contain = %q, want content", st.containment().Contain)
				}
			},
		},
		{
			name: "contain-keyword-list-normalized", prop: "contain", value: "SIZE  layout paint style",
			wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().Contain != "size layout paint style" {
					t.Errorf("Contain = %q, want normalized keyword list", st.containment().Contain)
				}
			},
		},
		{
			name: "contain-subset-order-kept", prop: "contain", value: "paint size", wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().Contain != "paint size" {
					t.Errorf("Contain = %q, want paint size", st.containment().Contain)
				}
			},
		},
		{name: "contain-duplicate-rejected", prop: "contain", value: "size size", wantOK: false},
		{name: "contain-shorthand-mixed-rejected", prop: "contain", value: "strict size", wantOK: false},
		{name: "contain-unknown-rejected", prop: "contain", value: "banana", wantOK: false},
		{name: "contain-empty-rejected", prop: "contain", value: "", wantOK: false},
		{
			name: "intrinsic-size-one-length", prop: "contain-intrinsic-size", value: "300px", wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().ContainIntrinsicWidth != 225 || st.containment().ContainIntrinsicHeight != 225 {
					t.Errorf("size = %v x %v, want 225 x 225",
						st.containment().ContainIntrinsicWidth, st.containment().ContainIntrinsicHeight)
				}
			},
		},
		{
			name: "intrinsic-size-two-lengths", prop: "contain-intrinsic-size", value: "300px 200pt",
			wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().ContainIntrinsicWidth != 225 || st.containment().ContainIntrinsicHeight != 200 {
					t.Errorf("size = %v x %v, want 225 x 200",
						st.containment().ContainIntrinsicWidth, st.containment().ContainIntrinsicHeight)
				}
			},
		},
		{
			name: "intrinsic-size-auto-prefix-uses-length", prop: "contain-intrinsic-size", value: "auto 300px",
			wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().ContainIntrinsicWidth != 225 || st.containment().ContainIntrinsicHeight != 225 {
					t.Errorf("size = %v x %v, want 225 x 225",
						st.containment().ContainIntrinsicWidth, st.containment().ContainIntrinsicHeight)
				}
			},
		},
		{
			name: "intrinsic-size-auto-two-entries", prop: "contain-intrinsic-size", value: "auto 300px 200pt",
			wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().ContainIntrinsicWidth != 225 || st.containment().ContainIntrinsicHeight != 200 {
					t.Errorf("size = %v x %v, want 225 x 200",
						st.containment().ContainIntrinsicWidth, st.containment().ContainIntrinsicHeight)
				}
			},
		},
		{
			name: "intrinsic-size-em-uses-fsize", prop: "contain-intrinsic-size", value: "2em", wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().ContainIntrinsicWidth != 24 || st.containment().ContainIntrinsicHeight != 24 {
					t.Errorf("size = %v x %v, want 24 x 24",
						st.containment().ContainIntrinsicWidth, st.containment().ContainIntrinsicHeight)
				}
			},
		},
		{
			name: "intrinsic-size-none", prop: "contain-intrinsic-size", value: "none", wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().ContainIntrinsicWidth != -1 || st.containment().ContainIntrinsicHeight != -1 {
					t.Errorf("size = %v x %v, want -1 x -1",
						st.containment().ContainIntrinsicWidth, st.containment().ContainIntrinsicHeight)
				}
			},
		},
		{name: "intrinsic-size-three-rejected", prop: "contain-intrinsic-size", value: "1pt 2pt 3pt", wantOK: false},
		{name: "intrinsic-size-garbage-rejected", prop: "contain-intrinsic-size", value: "auto junk", wantOK: false},
		{name: "intrinsic-size-negative-rejected", prop: "contain-intrinsic-size", value: "-5pt", wantOK: false},
		{
			name: "intrinsic-width-auto-length", prop: "contain-intrinsic-width", value: "auto 40pt", wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().ContainIntrinsicWidth != 40 {
					t.Errorf("ContainIntrinsicWidth = %v, want 40", st.containment().ContainIntrinsicWidth)
				}
			},
		},
		{
			name: "intrinsic-width-auto-alone", prop: "contain-intrinsic-width", value: "auto", wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().ContainIntrinsicWidth != -1 {
					t.Errorf("ContainIntrinsicWidth = %v, want -1 (unset)", st.containment().ContainIntrinsicWidth)
				}
			},
		},
		{
			name: "intrinsic-width-px", prop: "contain-intrinsic-width", value: "10px", wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().ContainIntrinsicWidth != 7.5 {
					t.Errorf("ContainIntrinsicWidth = %v, want 7.5", st.containment().ContainIntrinsicWidth)
				}
			},
		},
		{
			name: "intrinsic-height", prop: "contain-intrinsic-height", value: "15pt", wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().ContainIntrinsicHeight != 15 {
					t.Errorf("ContainIntrinsicHeight = %v, want 15", st.containment().ContainIntrinsicHeight)
				}
			},
		},
		{
			name: "intrinsic-block-size", prop: "contain-intrinsic-block-size", value: "25pt", wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().ContainIntrinsicBlockSize != 25 {
					t.Errorf("ContainIntrinsicBlockSize = %v, want 25", st.containment().ContainIntrinsicBlockSize)
				}
			},
		},
		{
			name: "intrinsic-inline-size", prop: "contain-intrinsic-inline-size", value: "12pt", wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().ContainIntrinsicInlineSize != 12 {
					t.Errorf("ContainIntrinsicInlineSize = %v, want 12", st.containment().ContainIntrinsicInlineSize)
				}
			},
		},
		{
			name: "intrinsic-longhand-two-values-rejected", prop: "contain-intrinsic-width",
			value: "10pt 20pt", wantOK: false,
		},
		{
			name: "content-visibility-visible", prop: "content-visibility", value: "visible", wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().ContentVisibility != "visible" {
					t.Errorf("ContentVisibility = %q, want visible", st.containment().ContentVisibility)
				}
			},
		},
		{
			name: "content-visibility-auto", prop: "content-visibility", value: "auto", wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().ContentVisibility != "auto" {
					t.Errorf("ContentVisibility = %q, want auto", st.containment().ContentVisibility)
				}
			},
		},
		{
			name: "content-visibility-hidden", prop: "content-visibility", value: "hidden", wantOK: true,
			check: func(t *testing.T, st ResolvedStyle) {
				t.Helper()

				if st.containment().ContentVisibility != "hidden" {
					t.Errorf("ContentVisibility = %q, want hidden", st.containment().ContentVisibility)
				}
			},
		},
		{name: "content-visibility-invalid-rejected", prop: "content-visibility", value: "collapse", wantOK: false},
		{name: "unrelated-prop-rejected", prop: "contain-foo", value: "size", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			st, ok := resolveContainmentProp(tt.prop, tt.value)
			if ok != tt.wantOK {
				t.Fatalf("applyContainmentProps(%q, %q) ok = %v, want %v", tt.prop, tt.value, ok, tt.wantOK)
			}

			if !tt.wantOK {
				if !reflect.DeepEqual(st, initialStyle()) {
					t.Fatalf("rejected %q changed the style", tt.value)
				}

				return
			}

			tt.check(t, st)
		})
	}
}

func TestContainmentKeywordHelpers(t *testing.T) {
	t.Parallel()

	strict := ResolvedStyle{
		containmentGroup: &styleContainment{
			Contain: "strict",
		},
	}
	if !containsSize(&strict) || !containsLayout(&strict) || !containsPaint(&strict) {
		t.Fatalf("strict must expand to size layout paint style: %+v", strict)
	}

	content := ResolvedStyle{
		containmentGroup: &styleContainment{
			Contain: "content",
		},
	}
	if containsSize(&content) || !containsLayout(&content) || !containsPaint(&content) {
		t.Fatalf("content must expand to layout paint style: %+v", content)
	}

	subset := ResolvedStyle{
		containmentGroup: &styleContainment{
			Contain: "size layout",
		},
	}
	if !containsSize(&subset) || !containsLayout(&subset) || containsPaint(&subset) {
		t.Fatalf("size layout must not imply paint containment: %+v", subset)
	}

	none := ResolvedStyle{
		containmentGroup: &styleContainment{
			Contain: "none",
		},
	}
	if containsSize(&none) || containsLayout(&none) || containsPaint(&none) {
		t.Fatalf("none must not imply containment: %+v", none)
	}
}

func TestContainmentIntrinsicAxisMapping(t *testing.T) {
	t.Parallel()

	horizontal := ResolvedStyle{
		containmentGroup: &styleContainment{
			ContainIntrinsicWidth:      -1,
			ContainIntrinsicHeight:     -1,
			ContainIntrinsicBlockSize:  30,
			ContainIntrinsicInlineSize: 60,
		},
	}

	if got := containmentIntrinsicWidth(&horizontal); got != 60 {
		t.Fatalf("horizontal width = %v, want inline size 60", got)
	}

	if got := containmentIntrinsicHeight(&horizontal); got != 30 {
		t.Fatalf("horizontal height = %v, want block size 30", got)
	}

	vertical := horizontal
	vertical.editPositioning().WritingMode = writingModeVerticalRL

	if got := containmentIntrinsicWidth(&vertical); got != 30 {
		t.Fatalf("vertical width = %v, want block size 30", got)
	}

	if got := containmentIntrinsicHeight(&vertical); got != 60 {
		t.Fatalf("vertical height = %v, want inline size 60", got)
	}

	physical := horizontal
	physical.editContainment().ContainIntrinsicWidth = 11
	physical.editContainment().ContainIntrinsicHeight = 22

	if got := containmentIntrinsicWidth(&physical); got != 11 {
		t.Fatalf("physical width = %v, want 11", got)
	}

	if got := containmentIntrinsicHeight(&physical); got != 22 {
		t.Fatalf("physical height = %v, want 22", got)
	}
}
