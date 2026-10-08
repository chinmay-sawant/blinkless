package imageout

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/settings"
)

// TestGoldenFixturesRenderPNG renders each body fixture under testdata/golden
// to a PNG. Header and footer companion files are not documents.
func TestGoldenFixturesRenderPNG(t *testing.T) {
	t.Parallel()

	matches, err := filepath.Glob(filepath.Join("..", "..", "testdata", "golden", "fixture-*.html"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	if len(matches) == 0 {
		t.Fatal("no golden fixtures")
	}

	fontDir := filepath.Join("..", "..", "testdata", "fonts")
	cjkDir := filepath.Join("..", "..", "testdata", "fixture-27-fonts")

	for _, path := range matches {
		base := filepath.Base(path)
		if strings.HasSuffix(base, "-header.html") || strings.HasSuffix(base, "-footer.html") {
			continue
		}

		t.Run(base, func(t *testing.T) {
			t.Parallel()

			abs, err := filepath.Abs(path)
			if err != nil {
				t.Fatalf("abs: %v", err)
			}

			global := settings.DefaultPdfGlobal()
			global.Load.EnableLocalFileAccess = true
			object := settings.DefaultPdfObject()
			object.Load.BlockLocalFileAccess = false
			// A few poster fixtures are taller than the raster cap at zoom 1.
			object.Load.ZoomFactor = 0.4
			if st, err := os.Stat(fontDir); err == nil && st.IsDir() {
				global.FontPaths = append(global.FontPaths, fontDir)
			}
			if st, err := os.Stat(cjkDir); err == nil && st.IsDir() {
				global.FontPaths = append(global.FontPaths, cjkDir)
			}

			image := settings.DefaultImageGlobal()
			image.Format = "png"
			object.Page = abs

			var buf bytes.Buffer
			req := NewRequest(global, image, []settings.PdfObject{object}, &buf)
			if err := RunRequest(t.Context(), req, io.Discard); err != nil {
				t.Fatalf("render %s: %v", base, err)
			}

			if !bytes.HasPrefix(buf.Bytes(), []byte("\x89PNG\r\n\x1a\n")) {
				t.Fatalf("%s: output is not a PNG (%d bytes)", base, buf.Len())
			}
			if buf.Len() < 32 {
				t.Fatalf("%s: PNG is too small (%d bytes)", base, buf.Len())
			}
		})
	}
}
