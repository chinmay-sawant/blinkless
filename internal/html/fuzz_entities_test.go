package html_test

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

func FuzzUnescapeEntities(f *testing.F) {
	f.Add("&amp;")
	f.Add("&#65;")
	f.Add("&#x1F4A9;")
	f.Add("&#00000000000000000065;")
	f.Add("&#999999999999;")

	f.Fuzz(func(_ *testing.T, src string) {
		if len(src) > 4096 {
			return
		}

		_ = html.UnescapeEntities(src)
	})
}
