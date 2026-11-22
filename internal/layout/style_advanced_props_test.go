package layout

import (
	"testing"
)

func TestWaveBTextTruncationAndClamping(t *testing.T) {
	t.Parallel()

	var s ResolvedStyle
	applyAdvancedProps(&s, "text-overflow", "ellipsis", 12)
	applyAdvancedProps(&s, "line-clamp", "3", 12)
	applyAdvancedProps(&s, "max-lines", "5", 12)
	applyAdvancedProps(&s, "margin-trim", "block", 12)

	if s.inlineText().TextOverflow != "ellipsis" {
		t.Errorf("TextOverflow = %q, want 'ellipsis'", s.inlineText().TextOverflow)
	}
	if s.inlineText().LineClamp != 3 {
		t.Errorf("LineClamp = %d, want 3", s.inlineText().LineClamp)
	}
	if s.inlineText().MaxLines != 5 {
		t.Errorf("MaxLines = %d, want 5", s.inlineText().MaxLines)
	}
	if s.geometry().MarginTrim != "block" {
		t.Errorf("MarginTrim = %q, want 'block'", s.geometry().MarginTrim)
	}
}

func TestLineClampClearsWebkitBoxNowrap(t *testing.T) {
	t.Parallel()

	var s ResolvedStyle
	setDisplayKeyword(&s, "-webkit-box")
	if s.WhiteSpace != cssWhiteSpaceNowrap {
		t.Fatalf("webkit-box WhiteSpace=%q, want nowrap", s.WhiteSpace)
	}
	applyAdvancedProps(&s, "-webkit-line-clamp", "2", 12)
	if s.inlineText().LineClamp != 2 {
		t.Fatalf("LineClamp=%d, want 2", s.inlineText().LineClamp)
	}
	if s.WhiteSpace == cssWhiteSpaceNowrap {
		t.Fatal("line-clamp should clear webkit-box nowrap so multiple lines can form")
	}
}

func TestWordBreakOfLineBreakAnywhere(t *testing.T) {
	t.Parallel()

	st := ResolvedStyle{
		WhiteSpace: "normal",
		inlineTextGroup: &styleInlineText{
			LineBreak: "anywhere",
		},
	}
	if got := wordBreakOf(&st); got != breakAll {
		t.Fatalf("wordBreakOf(line-break:anywhere)=%v, want breakAll", got)
	}
}

func TestWaveCBoxDecorationBreak(t *testing.T) {
	t.Parallel()

	var s ResolvedStyle
	applyAdvancedProps(&s, "box-decoration-break", "clone", 12)

	if s.borders().BoxDecorationBreak != "clone" {
		t.Errorf("BoxDecorationBreak = %q, want 'clone'", s.borders().BoxDecorationBreak)
	}
}

func TestWaveDTextDecorationSkipInkAndOverflowClipMargin(t *testing.T) {
	t.Parallel()

	var s ResolvedStyle
	applyAdvancedProps(&s, "text-decoration-skip-ink", "auto", 12)
	applyAdvancedProps(&s, "overflow-clip-margin-top", "10px", 12)
	applyAdvancedProps(&s, "overflow-clip-margin-right", "20pt", 12)

	if s.inlineText().TextDecorationSkipInk != "auto" {
		t.Errorf("TextDecorationSkipInk = %q, want 'auto'", s.inlineText().TextDecorationSkipInk)
	}
	if s.containment().OverflowClipMarginTop != 7.5 {
		t.Errorf("OverflowClipMarginTop = %v, want 7.5", s.containment().OverflowClipMarginTop)
	}
	if s.containment().OverflowClipMarginRight != 20 {
		t.Errorf("OverflowClipMarginRight = %v, want 20", s.containment().OverflowClipMarginRight)
	}
}
