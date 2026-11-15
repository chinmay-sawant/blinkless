package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBehaviorWordSpaceTransformSpaceWidensRun reads the fixture. none keeps
// A, a zero-width space, and B as a narrow run. space replaces U+200B with
// U+0020, so the text op is wider and the stored text contains that space.
func TestBehaviorWordSpaceTransformSpaceWidensRun(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "testdata", "css", "v012", "word-space-transform", "page.html")

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	res := layoutHTML(t, string(raw))
	noneText, noneW := wordSpaceJoined(t, res, "none")
	spaceText, spaceW := wordSpaceJoined(t, res, "space")

	if !strings.ContainsRune(noneText, wordSpaceZWSP) {
		t.Fatalf("none text = %q, want U+200B", noneText)
	}

	if strings.ContainsRune(noneText, ' ') {
		t.Fatalf("none text = %q, want no U+0020", noneText)
	}

	if !strings.ContainsRune(spaceText, ' ') || strings.ContainsRune(spaceText, wordSpaceZWSP) {
		t.Fatalf("space text = %q, want U+0020 and no U+200B", spaceText)
	}

	if spaceW <= noneW {
		t.Fatalf("space width %.4fpt is not wider than none %.4fpt", spaceW, noneW)
	}
}

// TestBehaviorWordSpaceTransformIdeographicSpace stores U+3000 in place of U+200B.
func TestBehaviorWordSpaceTransformIdeographicSpace(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, wordSpacePage(
		`<p id="ideo" style="margin:0;font-size:16px;white-space:nowrap;word-space-transform:ideographic-space">A&#x200b;B</p>`,
	))
	text, _ := wordSpaceJoined(t, res, "ideo")

	if !strings.ContainsRune(text, wordSpaceIdeographic) || strings.ContainsRune(text, wordSpaceZWSP) {
		t.Fatalf("ideographic-space text = %q, want U+3000 and no U+200B", text)
	}
}

// TestBehaviorWordSpaceTransformAutoPhraseInsertsNothing checks two limits.
// space auto-phrase does not split HelloWorld. auto-phrase alone is outside
// the grammar, so it inserts nothing and leaves the zero-width space.
func TestBehaviorWordSpaceTransformAutoPhraseInsertsNothing(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, wordSpacePage(
		`<p id="plain" style="margin:0;font-size:16px;white-space:nowrap">HelloWorld</p>`+
			`<p id="phrase" style="margin:0;font-size:16px;white-space:nowrap;word-space-transform:space auto-phrase">HelloWorld</p>`+
			`<p id="only" style="margin:0;font-size:16px;white-space:nowrap;word-space-transform:auto-phrase">A&#x200b;B</p>`+
			`<p id="still" style="margin:0;font-size:16px;white-space:nowrap;word-space-transform:auto-phrase space">A&#x200b;B</p>`,
	))

	plainText, plainW := wordSpaceJoined(t, res, "plain")
	phraseText, phraseW := wordSpaceJoined(t, res, "phrase")

	if phraseText != plainText || phraseText != "HelloWorld" {
		t.Fatalf("space auto-phrase text = %q, plain = %q, want HelloWorld", phraseText, plainText)
	}

	if !near(phraseW, plainW) {
		t.Fatalf("space auto-phrase width %.4fpt, plain %.4fpt, want the same run", phraseW, plainW)
	}

	onlyText, _ := wordSpaceJoined(t, res, "only")
	if !strings.ContainsRune(onlyText, wordSpaceZWSP) || strings.ContainsRune(onlyText, ' ') {
		t.Fatalf("auto-phrase alone text = %q, want U+200B and no inserted space", onlyText)
	}

	stillText, _ := wordSpaceJoined(t, res, "still")
	if !strings.ContainsRune(stillText, ' ') || strings.ContainsRune(stillText, wordSpaceZWSP) {
		t.Fatalf("auto-phrase space text = %q, want U+0020 (phrase detection adds nothing)", stillText)
	}
}

// TestBehaviorWordSpaceTransformWbrWidensRun treats wbr like U+200B under space.
func TestBehaviorWordSpaceTransformWbrWidensRun(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, wordSpacePage(
		`<p id="zwsp" style="margin:0;font-size:16px;white-space:nowrap;word-space-transform:space">A&#x200b;B</p>`+
			`<p id="wbr" style="margin:0;font-size:16px;white-space:nowrap;word-space-transform:space">A<wbr>B</p>`+
			`<p id="bare" style="margin:0;font-size:16px;white-space:nowrap;word-space-transform:none">A<wbr>B</p>`,
	))

	zwspText, zwspW := wordSpaceJoined(t, res, "zwsp")
	wbrText, wbrW := wordSpaceJoined(t, res, "wbr")
	bareText, bareW := wordSpaceJoined(t, res, "bare")

	if !strings.ContainsRune(wbrText, ' ') {
		t.Fatalf("wbr under space text = %q, want U+0020", wbrText)
	}

	if !near(wbrW, zwspW) {
		t.Fatalf("wbr width %.4fpt, zero-width-space width %.4fpt (%q), want the same gap", wbrW, zwspW, zwspText)
	}

	if strings.ContainsRune(bareText, ' ') || wbrW <= bareW {
		t.Fatalf("none wbr text = %q width %.4fpt, space wbr width %.4fpt, want no space and a wider space run",
			bareText, bareW, wbrW)
	}
}

// TestBehaviorWordSpaceTransformBesideBreakStays leaves a separator that
// touches a br. The separator on the next line still expands.
func TestBehaviorWordSpaceTransformBesideBreakStays(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, wordSpacePage(
		`<p id="kept" style="margin:0;font-size:16px;white-space:nowrap;word-space-transform:space">A&#x200b;<br>C&#x200b;D</p>`+
			`<p id="plain" style="margin:0;font-size:16px;white-space:nowrap;word-space-transform:none">A&#x200b;<br>C&#x200b;D</p>`+
			`<p id="wbr" style="margin:0;font-size:16px;white-space:nowrap;word-space-transform:space">A<wbr><br>B</p>`+
			`<p id="nobr" style="margin:0;font-size:16px;white-space:nowrap;word-space-transform:space">A<br>B</p>`,
	))

	kept := wordSpaceLines(t, res, "kept")
	plain := wordSpaceLines(t, res, "plain")

	if len(kept) < 2 || len(plain) < 2 {
		t.Fatalf("kept lines %d, plain lines %d, want two", len(kept), len(plain))
	}

	if !strings.ContainsRune(kept[0].text, wordSpaceZWSP) || strings.ContainsRune(kept[0].text, ' ') {
		t.Fatalf("separator before br = %q, want U+200B kept", kept[0].text)
	}

	if !near(kept[0].w, plain[0].w) {
		t.Fatalf("line beside br width %.4fpt, none %.4fpt, want no expansion", kept[0].w, plain[0].w)
	}

	if !strings.ContainsRune(kept[1].text, ' ') || strings.ContainsRune(kept[1].text, wordSpaceZWSP) {
		t.Fatalf("separator away from br = %q, want U+0020", kept[1].text)
	}

	wbrLine := wordSpaceLines(t, res, "wbr")
	plainBreak := wordSpaceLines(t, res, "nobr")

	if len(wbrLine) == 0 || len(plainBreak) == 0 {
		t.Fatal("missing wbr or control lines")
	}

	if strings.ContainsRune(wbrLine[0].text, ' ') || !near(wbrLine[0].w, plainBreak[0].w) {
		t.Fatalf("wbr beside br text = %q width %.4fpt, control %.4fpt, want the wbr to stay empty",
			wbrLine[0].text, wbrLine[0].w, plainBreak[0].w)
	}
}

// TestBehaviorWordSpaceTransformBadTokenKeepsSpace rejects a later bad token
// so an earlier space still replaces U+200B.
func TestBehaviorWordSpaceTransformBadTokenKeepsSpace(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, wordSpacePage(
		`<p id="bad" style="margin:0;font-size:16px;white-space:nowrap;word-space-transform:space;word-space-transform:nope">A&#x200b;B</p>`,
	))
	text, width := wordSpaceJoined(t, res, "bad")
	none := layoutHTML(t, wordSpacePage(
		`<p id="none" style="margin:0;font-size:16px;white-space:nowrap;word-space-transform:none">A&#x200b;B</p>`,
	))
	_, noneW := wordSpaceJoined(t, none, "none")

	if !strings.ContainsRune(text, ' ') || strings.ContainsRune(text, wordSpaceZWSP) {
		t.Fatalf("bad token text = %q, want the earlier space to stay", text)
	}

	if width <= noneW {
		t.Fatalf("bad token width %.4fpt, none %.4fpt, want the earlier space to widen the run", width, noneW)
	}
}

// TestBehaviorWordSpaceTransformInherits copies the value onto a child that
// does not declare it. The inherit mask is full, so the side copy does this.
func TestBehaviorWordSpaceTransformInherits(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, wordSpacePage(
		`<div style="word-space-transform:space">`+
			`<p id="kid" style="margin:0;font-size:16px;white-space:nowrap">A&#x200b;B</p>`+
			`</div>`+
			`<p id="own" style="margin:0;font-size:16px;white-space:nowrap;word-space-transform:none">A&#x200b;B</p>`,
	))

	kidText, kidW := wordSpaceJoined(t, res, "kid")
	ownText, ownW := wordSpaceJoined(t, res, "own")

	if !strings.ContainsRune(kidText, ' ') || strings.ContainsRune(kidText, wordSpaceZWSP) {
		t.Fatalf("inherited text = %q, want U+0020", kidText)
	}

	if !strings.ContainsRune(ownText, wordSpaceZWSP) || kidW <= ownW {
		t.Fatalf("child width %.4fpt text %q, none width %.4fpt text %q, want the child to inherit space",
			kidW, kidText, ownW, ownText)
	}
}

func wordSpacePage(body string) string {
	return `<html><body style="margin:0">` + body + `</body></html>`
}

func wordSpaceJoined(t *testing.T, res *Result, id string) (string, float64) {
	t.Helper()

	runs := wordSpaceRuns(t, res, id)
	var b strings.Builder
	var width float64

	for _, run := range runs {
		b.WriteString(run.Text)
		width += run.W
	}

	return b.String(), width
}

type wordSpaceLine struct {
	text string
	w    float64
	y    float64
}

// wordSpaceLines joins text ops that share a baseline. A zero-width space can
// be its own op when the font falls back, and it still belongs to that line.
func wordSpaceLines(t *testing.T, res *Result, id string) []wordSpaceLine {
	t.Helper()

	var lines []wordSpaceLine

	for _, run := range wordSpaceRuns(t, res, id) {
		if len(lines) == 0 || !near(run.Y, lines[len(lines)-1].y) {
			lines = append(lines, wordSpaceLine{y: run.Y})
		}

		last := &lines[len(lines)-1]
		last.text += run.Text
		last.w += run.W
	}

	return lines
}

func wordSpaceRuns(t *testing.T, res *Result, id string) []Op {
	t.Helper()

	boxNode := boxByID(t, res, id)
	if boxNode.opEnd < boxNode.opStart {
		t.Fatalf("#%s emitted no ops", id)
	}

	var runs []Op

	for i := boxNode.opStart; i <= boxNode.opEnd && i < len(res.Ops); i++ {
		if res.Ops[i].Kind == OpText {
			runs = append(runs, res.Ops[i])
		}
	}

	if len(runs) == 0 {
		t.Fatalf("#%s has no text runs", id)
	}

	return runs
}
