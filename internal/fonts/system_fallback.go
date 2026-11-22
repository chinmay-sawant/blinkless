package fonts

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// System CJK fallback.
//
// The bundled faces (Liberation, DejaVu) carry no Han, kana, or Hangul. When
// layout meets such a character and no CSS family, bundled face, or opt-in
// registry face covers it, this file lends a face from the operating system's
// own font files instead of leaving a missing-glyph box.
//
// It is on by default and deliberately narrow:
//   - it only runs for CJK and fullwidth code points (see systemFallbackRune),
//     so Latin documents never touch the file system;
//   - it reads a short per-OS candidate list, never a whole font directory
//     tree on Windows or macOS;
//   - each file is read and parsed once, on the first lookup that needs it;
//   - SetSystemFallback(false) or BLINKLESS_SYSTEM_FONTS=off turns it off.
//
// Collections (.ttc) are split into standalone faces so downstream shapers
// can load them one face at a time. CFF/OTTO faces are skipped like anywhere
// else in this package.

const (
	systemFontsEnvVar   = "BLINKLESS_SYSTEM_FONTS"
	systemFontsOffValue = "off"
	// systemFallbackCacheMax bounds the per-code-point result cache.
	systemFallbackCacheMax = 8192
	// linuxScanDepth is how many directory levels below a font root are read.
	linuxScanDepth = 3
	// systemFallbackScanMax bounds how many files the Linux scan keeps.
	systemFallbackScanMax = 64
)

// systemFallbackOff is the process-wide switch. It starts from the
// environment so an operator can opt out without a code change.
var systemFallbackOff atomic.Bool

func init() {
	if strings.EqualFold(os.Getenv(systemFontsEnvVar), systemFontsOffValue) {
		systemFallbackOff.Store(true)
	}
}

// SetSystemFallback turns the operating-system CJK fallback on or off for
// the whole process. It is on by default.
func SetSystemFallback(enabled bool) { systemFallbackOff.Store(!enabled) }

// SystemFallbackEnabled reports whether the fallback is on.
func SystemFallbackEnabled() bool { return !systemFallbackOff.Load() }

// systemFallbackRune reports whether r is a code point the system fallback
// serves: CJK ideographs and symbols, kana, Hangul, compatibility and
// fullwidth forms. Emoji and general symbols are not served here.
func systemFallbackRune(r rune) bool {
	switch {
	case r >= 0x2E80 && r <= 0x9FFF: // radicals, CJK symbols, kana, Bopomofo, Han
		return true
	case r >= 0xAC00 && r <= 0xD7AF: // Hangul syllables
		return true
	case r >= 0xF900 && r <= 0xFAFF: // CJK compatibility ideographs
		return true
	case r >= 0xFE30 && r <= 0xFE4F: // CJK compatibility forms
		return true
	case r >= 0xFF00 && r <= 0xFFEF: // halfwidth and fullwidth forms
		return true
	case r >= 0x20000 && r <= 0x3FFFF: // Han extensions
		return true
	}

	return false
}

// systemFontCandidate is one font file the fallback may read.
type systemFontCandidate struct {
	path string
	bold bool
}

// fontPlatform is what candidate discovery needs from the host. Tests supply
// a fake so every OS branch runs anywhere.
type fontPlatform struct {
	goos    string
	getenv  func(string) string
	home    string
	exists  func(path string) bool
	readDir func(path string) ([]fs.DirEntry, error)
}

func hostFontPlatform() fontPlatform {
	home, _ := os.UserHomeDir()

	return fontPlatform{
		goos:   runtime.GOOS,
		getenv: os.Getenv,
		home:   home,
		exists: func(p string) bool {
			st, err := os.Stat(p)

			return err == nil && st.Mode().IsRegular()
		},
		readDir: os.ReadDir,
	}
}

// Windows font files, in preference order. Japanese faces come first, then
// Simplified and Traditional Chinese, then Korean.
var (
	windowsRegularFonts = []string{
		"YuGothR.ttc", "YuGothM.ttc", "meiryo.ttc", "msgothic.ttc",
		"BIZ-UDGothicR.ttc", "YuGothL.ttc", "msyh.ttc", "msjh.ttc",
		"simsun.ttc", "malgun.ttf", "mingliub.ttc", "gulim.ttc",
	}
	windowsBoldFonts = []string{
		"YuGothB.ttc", "meiryob.ttc", "BIZ-UDGothicB.ttc", "msyhbd.ttc",
		"msjhbd.ttc", "malgunbd.ttf",
	}
)

// macOS font files. Arial Unicode MS is a single large TrueType face with
// broad CJK coverage on systems that still ship it.
var (
	darwinRegularFonts = []string{
		"/System/Library/Fonts/ヒラギノ角ゴシック W3.ttc",
		"/System/Library/Fonts/Hiragino Sans GB.ttc",
		"/System/Library/Fonts/PingFang.ttc",
		"/System/Library/Fonts/AppleSDGothicNeo.ttc",
		"/System/Library/Fonts/STHeiti Light.ttc",
		"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
		"/Library/Fonts/Arial Unicode.ttf",
	}
	darwinBoldFonts = []string{
		"/System/Library/Fonts/ヒラギノ角ゴシック W6.ttc",
		"/System/Library/Fonts/STHeiti Medium.ttc",
	}
)

// Linux and Android keep fonts in distribution-specific places, so the scan
// is by file name. Earlier keywords win. Noto Sans CJK ships as CFF on most
// distributions and is skipped at parse time; the TrueType families listed
// before it are what actually load there.
var (
	linuxFontRoots = []string{
		"/usr/share/fonts", "/usr/local/share/fonts", "/system/fonts",
	}
	linuxFontHomeRoots = []string{".fonts", ".local/share/fonts"}
	linuxFontKeywords  = []string{
		"droidsansfallback", "takao", "vlgothic", "ipag", "ipaex",
		"wqy", "wenquanyi", "sazanami", "notosanscjk", "notosansjp",
		"unifont",
	}
)

// wslWindowsFonts is where WSL exposes the Windows font folder.
const wslWindowsFonts = "/mnt/c/Windows/Fonts"

// systemFontCandidates lists the font files worth trying on this platform,
// regular faces first and bold faces after. Files that do not exist are
// dropped here so the lookup never pays for them.
func systemFontCandidates(plat fontPlatform) []systemFontCandidate {
	var out []systemFontCandidate

	add := func(path string, bold bool) {
		if plat.exists(path) {
			out = append(out, systemFontCandidate{path: path, bold: bold})
		}
	}

	addNamed := func(dir string, regular, bold []string) {
		for _, name := range regular {
			add(filepath.Join(dir, name), false)
		}

		for _, name := range bold {
			add(filepath.Join(dir, name), true)
		}
	}

	switch plat.goos {
	case "windows":
		root := plat.getenv("WINDIR")
		if root == "" {
			root = plat.getenv("SystemRoot")
		}

		if root == "" {
			root = `C:\Windows`
		}

		addNamed(filepath.Join(root, "Fonts"), windowsRegularFonts, windowsBoldFonts)
	case "darwin", "ios":
		for _, p := range darwinRegularFonts {
			add(p, false)
		}

		for _, p := range darwinBoldFonts {
			add(p, true)
		}
	case "linux", "android", "freebsd", "openbsd", "netbsd":
		out = append(out, scanLinuxFonts(plat)...)

		// WSL sees the Windows fonts, so a Linux build running there can
		// still render Japanese.
		addNamed(wslWindowsFonts, windowsRegularFonts, windowsBoldFonts)
	}

	return out
}

// fontHit is one scanned Linux font file and its keyword priority.
type fontHit struct {
	path     string
	priority int
	bold     bool
}

// scanLinuxFonts walks the font roots for files whose name matches a CJK
// keyword, ordered by keyword priority.
func scanLinuxFonts(plat fontPlatform) []systemFontCandidate {
	roots := append([]string(nil), linuxFontRoots...)

	if plat.home != "" {
		for _, rel := range linuxFontHomeRoots {
			roots = append(roots, filepath.Join(plat.home, rel))
		}
	}

	var hits []fontHit

	var walk func(dir string, depth int)

	walk = func(dir string, depth int) {
		entries, err := plat.readDir(dir)
		if err != nil {
			return
		}

		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())

			if entry.IsDir() {
				if depth > 0 {
					walk(path, depth-1)
				}

				continue
			}

			low := strings.ToLower(entry.Name())
			if !strings.HasSuffix(low, ".ttf") && !strings.HasSuffix(low, ".ttc") &&
				!strings.HasSuffix(low, ".otf") {
				continue
			}

			for priority, keyword := range linuxFontKeywords {
				if strings.Contains(strings.ReplaceAll(low, "-", ""), keyword) {
					hits = append(hits, fontHit{
						path:     path,
						priority: priority,
						bold:     strings.Contains(low, "bold"),
					})

					break
				}
			}
		}
	}

	for _, root := range roots {
		walk(root, linuxScanDepth)
	}

	// Stable order: keyword priority, then path, so two runs agree.
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].priority != hits[j].priority {
			return hits[i].priority < hits[j].priority
		}

		return hits[i].path < hits[j].path
	})

	if len(hits) > systemFallbackScanMax {
		hits = hits[:systemFallbackScanMax]
	}

	out := make([]systemFontCandidate, 0, len(hits))
	for _, h := range hits {
		out = append(out, systemFontCandidate{path: h.path, bold: h.bold})
	}

	return out
}

// loadedSystemFont is one candidate file after its first read.
type loadedSystemFont struct {
	loaded bool
	faces  []*Font
}

// systemFallback lazily loads candidate files and answers glyph lookups.
type systemFallback struct {
	mu         sync.Mutex
	plat       fontPlatform
	discovered bool
	candidates []systemFontCandidate
	files      []loadedSystemFont
	// results caches the answer per code point and boldness, including
	// misses, so a document full of uncovered characters scans once.
	results map[systemFallbackKey]*Font
}

type systemFallbackKey struct {
	r    rune
	bold bool
}

func newSystemFallback(plat fontPlatform) *systemFallback {
	return &systemFallback{plat: plat} //nolint:exhaustruct // lazy fields start zero
}

var hostSystemFallback = newSystemFallback(hostFontPlatform())

// FindSystemGlyph returns an operating-system face with a glyph for r, or nil
// when the fallback is off, r is outside the CJK set, or no installed face
// covers it. The first call that needs a file reads it; later calls are
// memory lookups.
func FindSystemGlyph(r rune, weight int, _ bool) *Font {
	if systemFallbackOff.Load() || !systemFallbackRune(r) {
		return nil
	}

	return hostSystemFallback.find(r, weight >= fontWeightBoldMin)
}

func (s *systemFallback) find(r rune, bold bool) *Font {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := systemFallbackKey{r: r, bold: bold}
	if face, ok := s.results[key]; ok {
		return face
	}

	if !s.discovered {
		s.candidates = systemFontCandidates(s.plat)
		s.files = make([]loadedSystemFont, len(s.candidates))
		s.discovered = true
	}

	var found *Font

	// A bold request tries bold files first; a regular request tries regular
	// files first. Either falls back to the other kind rather than a box.
	for _, wantBold := range []bool{bold, !bold} {
		if found = s.scan(r, wantBold); found != nil {
			break
		}
	}

	if s.results == nil || len(s.results) >= systemFallbackCacheMax {
		s.results = make(map[systemFallbackKey]*Font)
	}

	s.results[key] = found

	return found
}

// scan returns the first loaded face of the requested boldness with a glyph
// for r, loading candidate files as needed.
func (s *systemFallback) scan(r rune, bold bool) *Font {
	for i, cand := range s.candidates {
		if cand.bold != bold {
			continue
		}

		file := &s.files[i]
		if !file.loaded {
			file.faces = loadSystemFontFile(cand.path)
			file.loaded = true
		}

		for _, face := range file.faces {
			if face.GlyphID(r) != 0 {
				return face
			}
		}
	}

	return nil
}

// loadSystemFontFile reads one file and returns its usable TrueType faces.
// Collections yield one face per member; unreadable, oversized, and CFF
// files yield none.
func loadSystemFontFile(path string) []*Font {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxFontBytes {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	blobs := [][]byte{data}

	if IsTTC(data) {
		blobs, err = SplitTTC(data)
		if err != nil {
			return nil
		}
	}

	var faces []*Font

	for _, blob := range blobs {
		face, err := ParseTTF(blob)
		if err != nil {
			continue
		}

		if face.PostScriptName == "" {
			face.PostScriptName = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}

		faces = append(faces, face)
	}

	return faces
}
