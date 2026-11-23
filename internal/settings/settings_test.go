package settings

import (
	"errors"
	"math"
	"testing"
)

func TestDefaultPdfGlobalSnapshot(t *testing.T) {
	t.Parallel()

	global := DefaultPdfGlobal()

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"PageSize", global.PageSize, "A4"},
		{"Orientation", global.Orientation, OrientationPortrait},
		{"PdfVersion", global.PdfVersion, ""},
		{"PdfProfile", global.PdfProfile, ""},
		{"Grayscale", global.Grayscale, false},
		{"Collate", global.Collate, true},
		{"Outline", global.Outline, true},
		{"OutlineDepth", global.OutlineDepth, 4},
		{"UseCompression", global.UseCompression, true},
		{"Copies", global.Copies, 1},
		{"SmartShrinking", global.SmartShrinking, true},
		{"Background", global.Background, true},
		{"Web.Images", global.Web.Images, true},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}

	assertDefaultMargins(t, global.Margin)
	assertDefaultHeader(t, global.Header)
	assertDefaultTOC(t, global.TOC)
}

func assertDefaultMargins(t *testing.T, m Margin) {
	t.Helper()

	if m.Top != 10 || m.Bottom != 10 || m.Left != 10 || m.Right != 10 {
		t.Errorf("default margins = %+v, want 10mm all sides", m)
	}
}

func assertDefaultHeader(t *testing.T, h HeaderFooter) {
	t.Helper()

	if h.FontSize != 12 || h.FontName != "Arial" || h.Spacing != 0 {
		t.Errorf("default header = %+v, want Arial 12 spacing 0", h)
	}
}

func assertDefaultTOC(t *testing.T, toc TableOfContent) {
	t.Helper()

	if toc.CaptionText != "Table of Contents" || toc.FontScale != 0.8 || !toc.DottedLines {
		t.Errorf("default toc = %+v", toc)
	}
}

func TestDefaultPdfObjectSnapshot(t *testing.T) {
	t.Parallel()

	obj := DefaultPdfObject()
	if !obj.ExternalLinks || !obj.LocalLinks || !obj.IncludeInOutline || !obj.UseOutline {
		t.Errorf("default object = %+v", obj)
	}
}

func TestDefaultLoadPageSnapshot(t *testing.T) {
	t.Parallel()

	obj := DefaultPdfObject()
	if !obj.Load.BlockLocalFileAccess {
		t.Error("default blockLocalFileAccess must be true")
	}

	if obj.Load.LoadErrorHandling != LoadErrorAbort {
		t.Error("default loadErrorHandling must be abort")
	}
}

func TestParseUnitReal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		unit string
		val  float64
	}{
		{"10mm", "mm", 10},
		{"1.5cm", "cm", 1.5},
		{"1in", "in", 1},
		{"12pt", "pt", 12},
		{"96px", "px", 96},
		{"2em", "em", 2},
		{"100%", "%", 100},
		{"5", "", 5}, // implied unit
		{" 7.5 mm ", "mm", 7.5},
	}
	for _, testCase := range tests {
		unit, err := ParseUnitReal(testCase.in, "")
		if err != nil {
			t.Errorf("ParseUnitReal(%q): %v", testCase.in, err)

			continue
		}

		if unit.Value != testCase.val || unit.Unit != testCase.unit {
			t.Errorf("ParseUnitReal(%q) = %+v, want val=%v unit=%q", testCase.in, unit, testCase.val, testCase.unit)
		}
	}

	if _, err := ParseUnitReal("abc", "mm"); !errors.Is(err, ErrInvalidUnitReal) {
		t.Errorf("expected ErrInvalidUnitReal for %q, got %v", "abc", err)
	}

	if _, err := ParseUnitReal("", "mm"); !errors.Is(err, ErrInvalidUnitReal) {
		t.Errorf("expected ErrInvalidUnitReal for empty")
	}
}

func TestUnitRealPoints(t *testing.T) {
	t.Parallel()

	unit, _ := ParseUnitReal("10mm", "mm")

	pt, found := unit.Points()
	if !found || math.Abs(pt-28.346) > 0.01 {
		t.Errorf("10mm = %v pt (ok=%v)", pt, found)
	}

	unit, _ = ParseUnitReal("72pt", "pt")
	if pt, _ := unit.Points(); pt != 72 {
		t.Errorf("72pt = %v", pt)
	}

	unit, _ = ParseUnitReal("96px", "px")
	if pt, _ := unit.Points(); pt != 72 {
		t.Errorf("96px = %v pt, want 72", pt)
	}

	unit, _ = ParseUnitReal("2em", "em")
	if _, found := unit.Points(); found {
		t.Error("em must not convert without font context")
	}
}

func TestParsePageSize(t *testing.T) {
	t.Parallel()

	width, height, err := ParsePageSize("A4")
	if err != nil || math.Abs(width-595.28) > 0.1 || math.Abs(height-841.89) > 0.1 {
		t.Errorf("A4 = %v x %v, err %v", width, height, err)
	}

	width, height, err = ParsePageSize("letter")
	if err != nil || width != 612 || height != 792 {
		t.Errorf("letter = %v x %v, err %v", width, height, err)
	}

	if _, _, err := ParsePageSize("Bogus"); err == nil {
		t.Error("expected error for unknown size")
	}
}

func TestEnumStringReportsUnknownForInvalid(t *testing.T) {
	t.Parallel()

	if got := Orientation(99).String(); got != sUnknown {
		t.Errorf("Orientation(99).String() = %q, want unknown", got)
	}
	if got := LoadErrorHandling(42).String(); got != sUnknown {
		t.Errorf("LoadErrorHandling(42).String() = %q, want unknown", got)
	}
	if got := MediaType(7).String(); got != sUnknown {
		t.Errorf("MediaType(7).String() = %q, want unknown", got)
	}
}

func TestMediaTypeZeroIsUnset(t *testing.T) {
	t.Parallel()

	if MediaType(0) != MediaUnset {
		t.Fatalf("MediaType zero value = %d, want MediaUnset", MediaType(0))
	}
}

func TestHttpErrorCode(t *testing.T) {
	t.Parallel()

	if HttpErrorCode(401) != 3 {
		t.Error("401 must map to exit 3")
	}

	if HttpErrorCode(500) != 1 {
		t.Error("500 must map to exit 1")
	}
}

func TestHeaderForFooterForInherit(t *testing.T) {
	t.Parallel()

	global := DefaultPdfGlobal()
	global.Header.Left = "global header"
	obj := DefaultPdfObject()

	if obj.HeaderFor(global).Left != "global header" {
		t.Error("object must inherit global header")
	}

	obj.HeaderSet = true
	obj.Header.Left = "own header"

	if obj.HeaderFor(global).Left != "own header" {
		t.Error("object header override must win")
	}
}

func TestDefaultImageGlobalNoQuietLogLevel(t *testing.T) {
	t.Parallel()

	img := DefaultImageGlobal()
	// Quiet lives on PdfGlobal only; fonts.LogFontRegistryScan reads it there.
	if img.Width != 1024 || !img.SmartWidth || img.Quality != 94 {
		t.Errorf("default image = %+v", img)
	}
}

func TestParsePDFVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    string
		wantErr error
	}{
		{"", "1.4", nil},
		{"1.4", "1.4", nil},
		{"1.7", "1.7", nil},
		{" 1.7 ", "1.7", nil},
		{"2.0", "2.0", nil},
		{" 2.0 ", "2.0", nil},
		{"9.9", "", ErrInvalidPDFVersion},
		{"invalid", "", ErrInvalidPDFVersion},
		{"1.5", "", ErrInvalidPDFVersion},
	}

	for _, testCase := range tests {
		got, err := ParsePDFVersion(testCase.input)
		if testCase.wantErr != nil {
			if !errors.Is(err, testCase.wantErr) {
				t.Errorf("ParsePDFVersion(%q) error = %v, wantErr %v", testCase.input, err, testCase.wantErr)
			}

			continue
		}

		if err != nil {
			t.Errorf("ParsePDFVersion(%q) unexpected error: %v", testCase.input, err)
		}

		if got != testCase.want {
			t.Errorf("ParsePDFVersion(%q) = %q, want %q", testCase.input, got, testCase.want)
		}
	}
}
