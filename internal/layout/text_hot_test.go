package layout

import (
	"strconv"
	"strings"
	"testing"

	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
)

// loadHotFaces loads the bundled faces for engine-level hot-path tests.
func loadHotFaces(t *testing.T) (*pdf.FaceSet, error) {
	t.Helper()

	return pdf.LoadDefaultFaces()
}

// hotStyle builds a 12pt test style with the given autospace and emoji
// variant values.
func hotStyle(size float64, autospace, emoji string) ResolvedStyle {
	style := initialStyle()
	style.FontSize = size
	style.TextAutospace = autospace
	style.FontVariantEmoji = emoji

	return style
}

// hotSpacedStyle builds a letter/word-spaced test style.
func hotSpacedStyle() ResolvedStyle {
	style := initialStyle()
	style.FontSize = 12
	style.LetterSpacing = 1.5
	style.WordSpacing = 2

	return style
}

// hotTabStyle builds a tabbed test style with a numeric tab size.
func hotTabStyle() ResolvedStyle {
	style := initialStyle()
	style.FontSize = 12
	style.TabSize = 4

	return style
}

// Hot-path tests for the inline paint/measure reuse work: paint consults
// the collect-time width, one coverage bit serves metrics and paint, the
// string and rune caches share one combined lookup each, batched advances
// hold one read lock, the post-transform paint string is stored once per
// run, and autospace/emoji/line-height invariants hoist per run. Every test
// asserts cold-cache and warm-cache layouts produce identical output, so a
// cache can only skip work, never change paint.

// hotTextGoldenOps pins the op count for hotTextDoc(40). Filled from the
// first green run; the test prints the actual on mismatch.
const hotTextGoldenOps = 1230

// hotTextGoldenFirst pins the first op of hotTextDoc(40).
const hotTextGoldenFirst = `kind=4 x=0 y=11.9755859375 w=336.0546875 h=16.5 size=14 text="Paragraph 0 alpha beta gamma delta with "`

// hotTextGoldenSample pins one mid-document sample: the first uppercase
// spaced word op, as kind/x/y/w/h/size/text.
const hotTextGoldenSample = `kind=4 x=0 y=41.2861328125 w=290.2900390625 h=11.25 size=10 text="spaced upper words 0 repeat repeat repeat"`

// hotTextDoc builds a generated text-heavy document exercising every
// hot-path change: repeated words (width/cache reuse), descender links
// (decoration segments plus skip-ink), uppercase transform (paint-string
// reuse), letter/word spacing, tabbed pre text, CJK/Latin autospace, emoji
// presentation, explicit and normal line heights, and text emphasis.
func hotTextDoc(paras int) string {
	var b strings.Builder

	b.WriteString(`<html><body style="margin:0;font-size:10pt;font-family:monospace">`)

	for i := 0; i < paras; i++ {
		n := strconv.Itoa(i)
		b.WriteString(`<p style="margin:0;font-size:14pt">Paragraph ` + n +
			` alpha beta gamma delta with <a href="https://example.com/` + n +
			`">jog pygmy ` + n + `</a> tail jets fypqy glyphs.</p>`)
		b.WriteString(`<p style="margin:0;letter-spacing:1px;word-spacing:3px;` +
			`text-transform:uppercase">spaced upper words ` + n + ` repeat repeat repeat</p>`)
		b.WriteString(`<p style="margin:0;font-size:16pt;text-autospace:ideograph-alpha">` +
			`mixed 漢A ideograph alpha boundary ` + n + `</p>`)
		b.WriteString(`<p style="margin:0;font-size:18pt;font-variant-emoji:emoji">` +
			`smile &#9786; grin &#128512; text ` + n + `</p>`)
		b.WriteString(`<p style="margin:0"><span style="text-emphasis:filled dot;` +
			`text-emphasis-color:#c00">emphasis dots here ` + n + `</span></p>`)
		b.WriteString(`<pre style="margin:0">tabbed	indent ` + n + ` end</pre>`)
	}

	b.WriteString(`</body></html>`)

	return b.String()
}

// clearHotCaches drops every cross-layout cache and zeroes every
// cross-layout counter so a test starts from a known cold state.
func clearHotCaches() {
	clearSharedMeasureCaches()
	clearCoverageCache()
	clearLineHeightCache()
	resetSharedMeasureCounters()
	resetCoverageCounters()
	resetLineHeightCounters()
}

// canonicalOp renders the stable paint fields of one op for identity
// comparison across layouts.
func canonicalOp(op Op) string {
	return "kind=" + strconv.Itoa(int(op.Kind)) +
		" x=" + strconv.FormatFloat(op.X, 'g', -1, 64) +
		" y=" + strconv.FormatFloat(op.Y, 'g', -1, 64) +
		" w=" + strconv.FormatFloat(op.W, 'g', -1, 64) +
		" h=" + strconv.FormatFloat(op.H, 'g', -1, 64) +
		" size=" + strconv.FormatFloat(op.Size, 'g', -1, 64) +
		" text=" + strconv.Quote(op.Text)
}

// assertOpsIdentical requires two layouts to paint exactly the same
// operations in the same order.
func assertOpsIdentical(t *testing.T, a, b *Result) {
	t.Helper()

	if len(a.Ops) != len(b.Ops) {
		t.Fatalf("warm op count = %d, want cold %d", len(b.Ops), len(a.Ops))
	}

	for i := range a.Ops {
		if canonicalOp(a.Ops[i]) != canonicalOp(b.Ops[i]) {
			t.Fatalf("op %d differs:\n cold %s\n warm %s", i, canonicalOp(a.Ops[i]), canonicalOp(b.Ops[i]))
		}
	}
}

// TestTextHotWidthsPaintIdentical pins widths and paint on a generated
// text-heavy doc: golden op count plus sample advances, and byte-identical
// ops between a cold-cache and a warm-cache layout.
func TestTextHotWidthsPaintIdentical(t *testing.T) {
	doc := hotTextDoc(40)
	clearHotCaches()

	cold := layoutHTML(t, doc)

	if hotTextGoldenOps > 0 && len(cold.Ops) != hotTextGoldenOps {
		t.Fatalf("op count = %d, want %d", len(cold.Ops), hotTextGoldenOps)
	}

	if hotTextGoldenFirst != "" {
		if got := canonicalOp(cold.Ops[0]); got != hotTextGoldenFirst {
			t.Fatalf("first op = %s, want %s", got, hotTextGoldenFirst)
		}
	}

	if hotTextGoldenSample != "" {
		sample := textOpContaining(t, cold, "repeat")
		if got := canonicalOp(sample); got != hotTextGoldenSample {
			t.Fatalf("sample op = %s, want %s", got, hotTextGoldenSample)
		}
	}

	warm := layoutHTML(t, doc)
	assertOpsIdentical(t, cold, warm)

	// Behavioral pins: the transform, autospace, and emoji paths ran.
	foundTransform := false
	foundAutospace := false

	for _, op := range cold.Ops {
		if op.Kind == OpText && op.TextTransformValue() == "uppercase" {
			foundTransform = true
		}

		if op.Kind == OpText && op.TextAutospace() == "ideograph-alpha" {
			foundAutospace = true
		}
	}

	if !foundTransform {
		t.Fatalf("no uppercase text op painted, want the transform path covered")
	}

	if !foundAutospace {
		t.Fatalf("no ideograph-alpha text op painted, want the autospace path covered")
	}

	if op := textOpContaining(t, cold, "jog pygmy 0"); op.Text == "" {
		t.Fatalf("descender link op missing, want the skip-ink path covered")
	}

	link := textOpContaining(t, cold, "jog pygmy 0")
	if link.W != textOpContaining(t, warm, "jog pygmy 0").W {
		t.Fatalf("link width moved between layouts, want identical advances")
	}
}

// TestTextHotCacheHits proves the reuse paths actually hit: a second layout
// of the same doc must serve string widths, rune widths, coverage bits, and
// line heights from the shared caches.
func TestTextHotCacheHits(t *testing.T) {
	doc := hotTextDoc(40)
	clearHotCaches()

	layoutHTML(t, doc)

	if sharedTextMisses.Load() == 0 {
		t.Fatalf("cold layout recorded no shared string misses, want measure activity")
	}

	if sharedCoverageMisses.Load() == 0 {
		t.Fatalf("cold layout recorded no coverage misses, want coverage activity")
	}

	layoutHTML(t, doc)

	if got := sharedTextHits.Load(); got == 0 {
		t.Fatalf("warm layout shared string hits = 0, want cache reuse")
	}

	if got := sharedRuneHits.Load(); got == 0 {
		t.Fatalf("warm layout shared rune hits = 0, want batched rune reuse")
	}

	if got := sharedCoverageHits.Load(); got == 0 {
		t.Fatalf("warm layout coverage hits = 0, want one bit shared by metrics and paint")
	}

	if got := sharedLineHeightHits.Load(); got == 0 {
		t.Fatalf("warm layout line-height hits = 0, want memoized normal heights")
	}
}

// TestMeasureHotRuneBatchParity requires the batched advance walk to return
// exactly the sequential measureRuneFace values for every style and text
// shape, including tabs, autospace, emoji, soft hyphens, and variation
// selectors.
func TestMeasureHotRuneBatchParity(t *testing.T) {
	faces, err := loadHotFaces(t)
	if err != nil {
		t.Fatalf("faces: %v", err)
	}

	styles := map[string]ResolvedStyle{
		"plain":   hotStyle(12, "", ""),
		"spaced":  hotSpacedStyle(),
		"tabbed":  hotTabStyle(),
		"auto":    hotStyle(12, "ideograph-alpha", ""),
		"emoji":   hotStyle(18, "", "emoji"),
		"emojtxt": hotStyle(18, "", "text"),
	}

	texts := []string{
		"hello world repeat repeat",
		"tab\there and\there",
		"mixed 漢A ideograph alpha",
		"smile ☺ grin 😀 end",
		"soft\u00adhyphen",
		"text\ufe0eselector",
		"jog pygmy fypqy",
	}

	for name, style := range styles {
		for _, text := range texts {
			eng := &engine{font: faces.Regular, faces: faces, scale: 1}
			st := style

			var want []float64

			for _, r := range text {
				want = append(want, eng.measureRuneFace(r, &st))
			}

			eng2 := &engine{font: faces.Regular, faces: faces, scale: 1}

			var got []float64

			eng2.eachRuneAdvance(text, &st, func(_ rune, w float64) bool {
				got = append(got, w)

				return true
			})

			if len(got) != len(want) {
				t.Fatalf("%s %q: batch runes = %d, want %d", name, text, len(got), len(want))
			}

			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("%s %q rune %d: batch = %v, want %v", name, text, i, got[i], want[i])
				}
			}
		}
	}
}

// TestInlineHotPaintWidthReuse requires the primary-face paint path to reuse
// the collect-time width exactly, including the stored post-transform paint
// string, on both the cold loop fallback and the warm cache hit.
func TestInlineHotPaintWidthReuse(t *testing.T) {
	faces, err := loadHotFaces(t)
	if err != nil {
		t.Fatalf("faces: %v", err)
	}

	plain := hotStyle(12, "", "")
	upper := hotStyle(12, "", "")
	upper.TextTransform = "uppercase"

	clearHotCaches()

	eng := &engine{font: faces.Regular, faces: faces, scale: 1}

	const text = "alpha beta gamma repeat"

	// Simulate collect: measure the transformed string first.
	wantPlain := eng.measureTextFace(text, &plain)
	run, ok := eng.primaryFaceRun(text, &plain)

	if !ok {
		t.Fatalf("primaryFaceRun(%q) = miss, want hit", text)
	}

	if run.w != wantPlain {
		t.Fatalf("paint width = %v, want collect width %v", run.w, wantPlain)
	}

	if run.paint != "" {
		t.Fatalf("plain run paint = %q, want empty (reuse derives it)", run.paint)
	}

	// Warm path: the same run must return the identical width.
	runWarm, ok := eng.primaryFaceRun(text, &plain)
	if !ok || runWarm.w != wantPlain {
		t.Fatalf("warm paint width = %v, want collect width %v", runWarm.w, wantPlain)
	}

	// Transform path: the stored paint string is the uppercase source.
	const src = "hello world"
	engT := &engine{font: faces.Regular, faces: faces, scale: 1}
	wantUpper := engT.measureTextFace("HELLO WORLD", &upper)
	runT, ok := engT.primaryFaceRun(src, &upper)

	if !ok {
		t.Fatalf("primaryFaceRun(%q) = miss, want hit", src)
	}

	if runT.w != wantUpper {
		t.Fatalf("uppercase paint width = %v, want collect width %v", runT.w, wantUpper)
	}

	if runT.paint != "HELLO WORLD" {
		t.Fatalf("uppercase run paint = %q, want %q", runT.paint, "HELLO WORLD")
	}

	// Coverage bit: one scan serves metrics and paint, proven by counters.
	clearCoverageCache()
	resetCoverageCounters()

	if !faceRunAllPrimary("alpha beta", faces.Regular) {
		t.Fatalf("coverage of ascii text = false, want true")
	}

	if got := sharedCoverageMisses.Load(); got != 1 {
		t.Fatalf("coverage misses = %d, want 1", got)
	}

	if !faceRunAllPrimary("alpha beta", faces.Regular) {
		t.Fatalf("warm coverage of ascii text = false, want true")
	}

	if got := sharedCoverageHits.Load(); got != 1 {
		t.Fatalf("coverage hits = %d, want 1", got)
	}
}

// TestSpacingHotAutospaceParity requires autospace layouts to paint
// identically cold and warm, and wider than no-autospace.
func TestSpacingHotAutospaceParity(t *testing.T) {
	plainDoc := `<html><body style="margin:0">` +
		`<p style="margin:0;font-size:16pt;text-autospace:no-autospace">漢A mixed</p>` +
		`</body></html>`
	spacedDoc := `<html><body style="margin:0">` +
		`<p style="margin:0;font-size:16pt;text-autospace:ideograph-alpha">漢A mixed</p>` +
		`</body></html>`

	clearHotCaches()

	plain := layoutHTML(t, plainDoc)
	cold := layoutHTML(t, spacedDoc)
	warm := layoutHTML(t, spacedDoc)
	assertOpsIdentical(t, cold, warm)

	wp := textOpContaining(t, plain, "漢A").W
	ws := textOpContaining(t, cold, "漢A").W

	if ws <= wp+0.1 {
		t.Fatalf("ideograph-alpha width %v should exceed no-autospace width %v", ws, wp)
	}
}

// TestEmojiHotParity requires emoji layouts to paint identically cold and
// warm, with the all-emoji run taking the lite presentation fill.
func TestEmojiHotParity(t *testing.T) {
	doc := `<html><body style="margin:0">` +
		`<p style="margin:0;font-size:18pt;font-variant-emoji:emoji">` +
		`&#9786;&#128512;</p>` +
		`<p style="margin:0;font-size:18pt;font-variant-emoji:text">` +
		`&#9786;&#128512;</p>` +
		`</body></html>`

	clearHotCaches()

	cold := layoutHTML(t, doc)
	warm := layoutHTML(t, doc)
	assertOpsIdentical(t, cold, warm)

	op := textOpContaining(t, cold, "☺")
	if op.Text == "" {
		t.Fatalf("no emoji text op painted")
	}

	if op.R != liteEmojiFillR || op.G != liteEmojiFillG || op.B != liteEmojiFillB {
		t.Fatalf("emoji fill = (%v,%v,%v), want lite (%v,%v,%v)",
			op.R, op.G, op.B, liteEmojiFillR, liteEmojiFillG, liteEmojiFillB)
	}
}

// TestLineHeightHotMemo requires the memoized normal line height to match
// the direct computation and to hit the shared cache on repeats, while an
// explicit line height bypasses the cache untouched.
func TestLineHeightHotMemo(t *testing.T) {
	faces, err := loadHotFaces(t)
	if err != nil {
		t.Fatalf("faces: %v", err)
	}

	clearLineHeightCache()
	resetLineHeightCounters()

	eng := &engine{font: faces.Regular, faces: faces, scale: 1}
	style := hotStyle(16, "", "")

	first := eng.lineHeightOf(&style)
	if first <= 0 {
		t.Fatalf("normal line height = %v, want positive", first)
	}

	if got := sharedLineHeightMisses.Load(); got != 1 {
		t.Fatalf("line-height misses = %d, want 1", got)
	}

	eng2 := &engine{font: faces.Regular, faces: faces, scale: 1}

	if second := eng2.lineHeightOf(&style); second != first {
		t.Fatalf("memoized line height = %v, want %v", second, first)
	}

	if got := sharedLineHeightHits.Load(); got != 1 {
		t.Fatalf("line-height hits = %d, want 1", got)
	}

	explicit := hotStyle(16, "", "")
	explicit.LineHeight = 20
	beforeHits := sharedLineHeightHits.Load()
	beforeMisses := sharedLineHeightMisses.Load()

	if got := eng.lineHeightOf(&explicit); got != 20 {
		t.Fatalf("explicit line height = %v, want 20", got)
	}

	if sharedLineHeightHits.Load() != beforeHits || sharedLineHeightMisses.Load() != beforeMisses {
		t.Fatalf("explicit line height touched the cache, want bypass")
	}
}
