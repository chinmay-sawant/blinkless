package fonts

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSystemFallbackRune(t *testing.T) {
	for _, r := range []rune{'か', 'ナ', '漢', '한', '，', 'Ａ', '〒', '𠀀'} {
		if !systemFallbackRune(r) {
			t.Errorf("U+%04X should use the system fallback", r)
		}
	}

	for _, r := range []rune{'a', 'é', 'ж', '→', '😀', ' '} {
		if systemFallbackRune(r) {
			t.Errorf("U+%04X must not touch the system fallback", r)
		}
	}
}

func fakePlatform(goos string, files map[string]bool, env map[string]string) fontPlatform {
	return fontPlatform{
		goos:   goos,
		getenv: func(k string) string { return env[k] },
		exists: func(p string) bool { return files[p] },
		readDir: func(string) ([]fs.DirEntry, error) {
			return nil, fs.ErrNotExist
		},
	}
}

func TestSystemFontCandidatesWindows(t *testing.T) {
	dir := filepath.Join("D:", "Win", "Fonts")
	files := map[string]bool{
		filepath.Join(dir, "YuGothR.ttc"): true,
		filepath.Join(dir, "meiryo.ttc"):  true,
		filepath.Join(dir, "YuGothB.ttc"): true,
	}

	got := systemFontCandidates(fakePlatform("windows", files, map[string]string{"WINDIR": filepath.Join("D:", "Win")}))

	want := []string{"YuGothR.ttc", "meiryo.ttc", "YuGothB.ttc"}
	if len(got) != len(want) {
		t.Fatalf("candidates = %+v, want %v", got, want)
	}

	for i := range want {
		if !strings.HasSuffix(got[i].path, want[i]) {
			t.Fatalf("candidate %d = %q, want %q", i, got[i].path, want[i])
		}
	}

	if got[0].bold || got[1].bold || !got[2].bold {
		t.Fatalf("bold flags = %v %v %v, want regular, regular, bold", got[0].bold, got[1].bold, got[2].bold)
	}
}

func TestSystemFontCandidatesLinuxScan(t *testing.T) {
	root := fstest.MapFS{
		"usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf": {},
		"usr/share/fonts/truetype/takao-gothic/TakaoGothic.ttf":    {},
		"usr/share/fonts/truetype/dejavu/DejaVuSans.ttf":           {},
		"usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc":    {},
	}

	plat := fontPlatform{
		goos:   "linux",
		getenv: func(string) string { return "" },
		exists: func(string) bool { return false },
		readDir: func(dir string) ([]fs.DirEntry, error) {
			return fs.ReadDir(root, strings.TrimPrefix(filepath.ToSlash(dir), "/"))
		},
	}

	got := systemFontCandidates(plat)
	if len(got) != 3 {
		t.Fatalf("candidates = %+v, want the three CJK files only", got)
	}

	if filepath.Base(got[0].path) != "DroidSansFallbackFull.ttf" ||
		filepath.Base(got[1].path) != "TakaoGothic.ttf" ||
		filepath.Base(got[2].path) != "NotoSansCJK-Regular.ttc" {
		t.Fatalf("candidates out of priority order: %+v", got)
	}
}

func TestSystemFallbackDisabled(t *testing.T) {
	SetSystemFallback(false)
	t.Cleanup(func() { SetSystemFallback(true) })

	if FindSystemGlyph('か', 400, false) != nil {
		t.Fatal("fallback returned a face while disabled")
	}

	if SystemFallbackEnabled() {
		t.Fatal("SystemFallbackEnabled reports true after SetSystemFallback(false)")
	}
}

func TestSystemFallbackCachesMisses(t *testing.T) {
	fb := newSystemFallback(fakePlatform("windows", nil, nil))

	// The second lookup is answered from the cache, not by a second scan.
	for range 2 {
		if fb.find('か', false) != nil {
			t.Fatal("no candidates must produce no face")
		}
	}

	if !fb.discovered || len(fb.results) != 1 {
		t.Fatalf("discovered=%t results=%d, want one cached miss", fb.discovered, len(fb.results))
	}
}
