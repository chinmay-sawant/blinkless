package css

import "testing"

func TestIndexASCIIFoldKeepsOffsetsAfterWideLower(t *testing.T) {
	t.Parallel()

	// U+0130 lowercases to a 3-byte sequence, shifting ToLower offsets.
	src := `local("İstanbul"), URL(Font.woff2)`

	got := FontFaceURLs(src)
	if len(got) != 1 || got[0] != "Font.woff2" {
		t.Fatalf("FontFaceURLs = %q, want [Font.woff2]", got)
	}

	if i := IndexASCIIFold("İİİ REPEAT(", "repeat("); i != len("İİİ ") {
		t.Fatalf("IndexASCIIFold = %d", i)
	}

	if IndexASCIIFold("abc", "abcd") != -1 || IndexASCIIFold("abc", "") != 0 {
		t.Fatal("edge cases")
	}
}
