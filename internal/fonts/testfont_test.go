package fonts

import "testing"

func testFont(t *testing.T) *Font {
	t.Helper()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	fnt.ensureParsed()

	return fnt
}
