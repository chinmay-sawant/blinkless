package fonts

import (
	"encoding/binary"
	"slices"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/fonts/assets"
)

// buildTTC packs standalone SFNT faces into one collection, the way a system
// .ttc shares a single file: member directories first, absolute table offsets.
func buildTTC(t *testing.T, faces ...[]byte) []byte {
	t.Helper()

	header := make([]byte, ttcHeaderSize+4*len(faces))
	copy(header, "ttcf")
	binary.BigEndian.PutUint32(header[4:], 0x00010000)
	binary.BigEndian.PutUint32(header[8:], testU32(len(faces)))

	out := slices.Clone(header)

	for i, face := range faces {
		base := len(out)
		binary.BigEndian.PutUint32(out[ttcHeaderSize+4*i:], testU32(base))

		member := slices.Clone(face)
		numTables := int(binary.BigEndian.Uint16(member[4:]))

		for tb := range numTables {
			rec := member[sfntHeaderSize+sfntRecordSize*tb:]
			offset := binary.BigEndian.Uint32(rec[8:12])
			binary.BigEndian.PutUint32(rec[8:12], offset+testU32(base))
		}

		out = append(out, member...)
	}

	return out
}

// testU32 converts a small test offset or length to the 32 bits a font file
// stores. The values here are bounded by the size of the test fonts.
func testU32(n int) uint32 {
	return uint32(n) //nolint:gosec // reason: test offsets and lengths are small.
}

func TestSplitTTCRoundTrip(t *testing.T) {
	regular := assets.LiberationSansRegular()
	bold := assets.LiberationSansBold()

	faces, err := SplitTTC(buildTTC(t, regular, bold))
	if err != nil {
		t.Fatalf("SplitTTC: %v", err)
	}

	if len(faces) != 2 {
		t.Fatalf("members = %d, want 2", len(faces))
	}

	for i, want := range [][]byte{regular, bold} {
		got, err := ParseTTF(faces[i])
		if err != nil {
			t.Fatalf("member %d does not parse: %v", i, err)
		}

		orig, err := ParseTTF(want)
		if err != nil {
			t.Fatal(err)
		}

		if got.GlyphID('A') == 0 || got.GlyphID('A') != orig.GlyphID('A') {
			t.Fatalf("member %d glyph for A = %d, original %d", i, got.GlyphID('A'), orig.GlyphID('A'))
		}
	}
}

func TestSplitTTCRejectsMalformed(t *testing.T) {
	good := buildTTC(t, assets.LiberationSansRegular())

	cases := map[string][]byte{
		"not a collection": assets.LiberationSansRegular(),
		"too short":        []byte("ttcf"),
		"zero members":     append([]byte("ttcf\x00\x01\x00\x00\x00\x00\x00\x00"), 0),
		"too many members": append([]byte("ttcf\x00\x01\x00\x00\x00\x00\x01\x00"), make([]byte, 1024)...),
	}

	// A member offset past the end of the file.
	badOffset := append([]byte(nil), good...)
	binary.BigEndian.PutUint32(badOffset[ttcHeaderSize:], testU32(len(good)+10))
	cases["member offset out of range"] = badOffset

	// A table that claims bytes beyond the file.
	badTable := append([]byte(nil), good...)
	member := int(binary.BigEndian.Uint32(badTable[ttcHeaderSize:]))
	binary.BigEndian.PutUint32(badTable[member+sfntHeaderSize+12:], testU32(len(good)))
	cases["table length out of range"] = badTable

	for name, data := range cases {
		if _, err := SplitTTC(data); err == nil {
			t.Errorf("%s: SplitTTC succeeded, want an error", name)
		}
	}
}

func FuzzSplitTTC(f *testing.F) {
	f.Add([]byte("ttcf\x00\x01\x00\x00\x00\x00\x00\x01\x00\x00\x00\x10"))
	f.Fuzz(func(_ *testing.T, data []byte) { _, _ = SplitTTC(data) })
}
