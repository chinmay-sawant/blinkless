package fonts

import (
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"github.com/chinmay-sawant/blinkless/internal/fonts/assets"
)

func TestLazyFontAdvanceInPoints(t *testing.T) {
	t.Parallel()

	fnt := newLazyFont(assets.UnicodeFallbackRegular)
	got := fnt.AdvanceInPoints('a', 12)

	if want := fnt.GlyphAdvancePoints('a', 12); got <= 0 || got != want {
		t.Fatalf("first lazy AdvanceInPoints = %v, want %v", got, want)
	}
}

func TestCmap12RejectsNonUnicodeRange(t *testing.T) {
	t.Parallel()

	state := make([]byte, 28)
	binary.BigEndian.PutUint32(state[12:], 1)

	for _, codepoint := range []uint32{0xfffffffe, 0xffffffff} {
		binary.BigEndian.PutUint32(state[16:], codepoint)
		binary.BigEndian.PutUint32(state[20:], codepoint)

		fnt := &Font{cmap: make(map[uint32]uint16)}
		if err := fnt.parseCmap12(state); err == nil {
			t.Fatal("accepted codepoints outside Unicode")
		}
	}
}

func TestTableDirectoryUnsignedOffsets(t *testing.T) {
	t.Parallel()

	for _, offset := range []uint32{0x7fffffff, 0x80000000, 0xffffffff} {
		data := make([]byte, sfntOffsetTableSize+sfntTableRecordSize)
		binary.BigEndian.PutUint16(data[4:], 1)
		binary.BigEndian.PutUint32(data[sfntOffsetTableSize+8:], offset)
		binary.BigEndian.PutUint32(data[sfntOffsetTableSize+12:], 16)

		if _, err := parseTableDirectory(data); err == nil {
			t.Fatalf("accepted out-of-range offset %x", offset)
		}
	}
}

func TestCmapSubtableOffsetOverflow(t *testing.T) {
	t.Parallel()

	fnt := &Font{cmap: make(map[uint32]uint16)}
	mergeCmapSubtables(fnt, make([]byte, 4), []cmapSubtable{{platform: 3, encoding: 1, offset: 0x7fffffff}})

	if len(fnt.cmap) != 0 {
		t.Fatal("invalid offset populated cmap")
	}
}

func TestCmap12RepeatedRangesHaveWorkBudget(t *testing.T) {
	t.Parallel()

	state := make([]byte, 16+12*16)
	binary.BigEndian.PutUint32(state[12:], 16)

	for i := range 16 {
		binary.BigEndian.PutUint32(state[16+i*12+4:], maxCmapGroupSpan)
	}

	fnt := &Font{cmap: make(map[uint32]uint16)}
	if err := fnt.parseCmap12(state); err == nil {
		t.Fatal("accepted repeated expansions past the work budget")
	}
}

func TestCmap12GlyphArithmeticDoesNotWrap(t *testing.T) {
	t.Parallel()

	state := make([]byte, 28)
	binary.BigEndian.PutUint32(state[12:], 1)
	binary.BigEndian.PutUint32(state[16:], 65)
	binary.BigEndian.PutUint32(state[20:], 66)
	binary.BigEndian.PutUint32(state[24:], 0xffffffff)

	fnt := &Font{cmap: make(map[uint32]uint16), numGlyphs: 2}
	if err := fnt.parseCmap12(state); err != nil {
		t.Fatal(err)
	}

	if len(fnt.cmap) != 0 {
		t.Fatal("overflowed glyph arithmetic created a bogus mapping")
	}
}

// TestZeroFontAdvances proves a failed or empty font yields zero advances
// instead of panicking or producing Inf/NaN.
func TestZeroFontAdvances(t *testing.T) {
	t.Parallel()

	empty := &Font{}

	if got := empty.Advance('a'); got != 0 {
		t.Fatalf("Advance = %v, want 0", got)
	}

	if got := empty.AdvanceInPoints('a', 12); got != 0 {
		t.Fatalf("AdvanceInPoints = %v, want 0", got)
	}

	if got := empty.GlyphAdvancePoints('a', 12); got != 0 {
		t.Fatalf("GlyphAdvancePoints = %v, want 0", got)
	}

	if empty.glyphOutline(0) != nil {
		t.Fatal("glyphOutline on empty font must be nil")
	}
}

// TestCmap12GroupSpanCap proves a group spanning billions of codepoints is
// rejected instead of expanding into the cmap.
func TestCmap12GroupSpanCap(t *testing.T) {
	t.Parallel()

	fnt := &Font{
		cmap: make(map[uint32]uint16),
	}

	state := make([]byte, 28)
	state[12] = 0
	state[13] = 0
	state[14] = 0
	state[15] = 1

	// start=0, end=0xFFFFFFFF, startGlyph=0.
	state[20] = 0xFF
	state[21] = 0xFF
	state[22] = 0xFF
	state[23] = 0xFF

	if err := fnt.parseCmap12(state); err == nil {
		t.Fatal("parseCmap12 accepted a group spanning 2^32 codepoints")
	}

	if len(fnt.cmap) != 0 {
		t.Fatalf("cmap has %d entries after rejected group", len(fnt.cmap))
	}
}

// TestCmapSubtablesTruncated proves a short cmap directory parses without
// indexing out of range.
func TestCmapSubtablesTruncated(t *testing.T) {
	t.Parallel()

	if subs := parseCmapSubtables([]byte{0, 0}); subs != nil {
		t.Fatalf("parseCmapSubtables(short) = %v, want nil", subs)
	}

	// Declares two records but carries one.
	tbl := []byte{0, 0, 0, 2, 0, 3, 0, 1, 0, 0, 0, 0}

	if subs := parseCmapSubtables(tbl); len(subs) != 1 {
		t.Fatalf("parseCmapSubtables(truncated) = %d records, want 1", len(subs))
	}
}

// TestTableDirectoryOverlap proves aliased tables are rejected.
func TestTableDirectoryOverlap(t *testing.T) {
	t.Parallel()

	data := buildTwoTableDirectory(t, 64, 32, 64, 32)

	if _, err := parseTableDirectory(data); err == nil {
		t.Fatal("parseTableDirectory accepted overlapping tables")
	}
}

// TestTableDirectoryTooMany proves an absurd table count is rejected.
func TestTableDirectoryTooMany(t *testing.T) {
	t.Parallel()

	data := make([]byte, sfntOffsetTableSize)
	data[0], data[1], data[2], data[3] = 0, 1, 0, 0
	data[4] = byte(maxFontTables + 1)

	if _, err := parseTableDirectory(data); err == nil {
		t.Fatal("parseTableDirectory accepted too many tables")
	}
}

// oversizeShapeSlop pushes one run just past maxShapeRunBytes with a single
// strings.Repeat, keeping the regression test bounded and fast.
const oversizeShapeSlop = 16

// TestOversizeShapeRunFallsBack proves the maxShapeRunBytes cap in
// tryShapeOpenType is safe: a >1MB run against a real GSUB face with an
// effectful feature returns the documented fallback instead of hanging,
// exhausting memory, or panicking. The salt control maps small "a" to U+0251
// on this face, so an unchanged result on the big run must come from the cap.
// Latin needs no shaping, so the fallback is the input unchanged.
func TestOversizeShapeRunFallsBack(t *testing.T) {
	t.Parallel()

	fnt := bundledDejaVu(t)
	feats := ParseFontFeatureSettings(`"salt" 1`)

	if len(feats) != 1 {
		t.Fatalf("salt features = %d, want 1", len(feats))
	}

	if ShapeTextFontWithFeatures("a", fnt, feats) == "a" {
		t.Fatal("control: salt must reshape small a on this face")
	}

	big := strings.Repeat("a", maxShapeRunBytes+oversizeShapeSlop)

	if _, ok := tryShapeOpenType(big, fnt, feats, ""); ok {
		t.Fatal("tryShapeOpenType accepted an oversized run")
	}

	start := time.Now()
	got := ShapeTextFontWithFeatures(big, fnt, feats)
	elapsed := time.Since(start)

	if got != big {
		t.Fatalf("oversized Latin run: len %d, want %d unchanged", len(got), len(big))
	}

	if elapsed > 5*time.Second {
		t.Fatalf("oversized fallback took %v, want under 5s", elapsed)
	}

	t.Logf("oversized run (%d bytes) fell back unchanged in %v", len(big), elapsed)
}

// buildTwoTableDirectory builds a minimal SFNT header with two tables at the
// given offset/length pairs over a zero body.
func buildTwoTableDirectory(t *testing.T, off1, len1, off2, len2 int) []byte {
	t.Helper()

	bodyLen := off2 + len2 + 16
	data := make([]byte, sfntOffsetTableSize+2*sfntTableRecordSize+bodyLen)

	data[0], data[1], data[2], data[3] = 0, 1, 0, 0
	data[4], data[5] = 0, 2

	putTable := func(rec int, tag string, off, length int) {
		base := sfntOffsetTableSize + rec*sfntTableRecordSize
		copy(data[base:], tag)
		data[base+8] = byte(off >> 24)
		data[base+9] = byte(off >> 16)
		data[base+10] = byte(off >> 8)
		data[base+11] = byte(off)
		data[base+12] = byte(length >> 24)
		data[base+13] = byte(length >> 16)
		data[base+14] = byte(length >> 8)
		data[base+15] = byte(length)
	}

	putTable(0, "head", off1, len1)
	putTable(1, "maxp", off2, len2)

	return data
}
