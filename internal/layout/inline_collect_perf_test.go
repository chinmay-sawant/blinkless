package layout

import (
	"strconv"
	"strings"
	"testing"

	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// wideTableDoc builds a multi-hundred-row table that exercises the inline
// collect hot path: plain multi-word cells, nested spans, links, and
// trailing/leading whitespace across pretty-printed rows.
func wideTableDoc(rows int) string {
	var b strings.Builder

	b.WriteString(`<html><head><style>` +
		`table{border-collapse:collapse}` +
		`td{font:10pt/12pt monospace;padding:0}` +
		`</style></head><body style="margin:0"><table>`)

	for i := 0; i < rows; i++ {
		b.WriteString("<tr><td>Row ")
		b.WriteString(strconv.Itoa(i))
		b.WriteString(" alpha beta gamma </td><td>value <span>inner ")
		b.WriteString(strconv.Itoa(i))
		b.WriteString("</span> tail</td><td><a href=\"https://example.com/")
		b.WriteString(strconv.Itoa(i))
		b.WriteString("\">link ")
		b.WriteString(strconv.Itoa(i))
		b.WriteString("</a> end</td></tr>")
	}

	b.WriteString(`</table></body></html>`)

	return b.String()
}

func fmtOpFields(op Op) string {
	return "kind=" + strconv.Itoa(int(op.Kind)) +
		" x=" + strconv.FormatFloat(op.X, 'g', -1, 64) +
		" y=" + strconv.FormatFloat(op.Y, 'g', -1, 64) +
		" w=" + strconv.FormatFloat(op.W, 'g', -1, 64) +
		" h=" + strconv.FormatFloat(op.H, 'g', -1, 64) +
		" size=" + strconv.FormatFloat(op.Size, 'g', -1, 64) +
		" text=" + strconv.Quote(op.Text)
}

// TestInlineCollectParityWideTable is the golden parity gate for the inline
// collect fast paths: op count plus first/last op fields must not move.
func TestInlineCollectParityWideTable(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, wideTableDoc(300))

	const wantOps = 2600

	if len(res.Ops) != wantOps {
		t.Fatalf("op count = %d, want %d", len(res.Ops), wantOps)
	}

	first := res.Ops[0]
	if got := fmtOpFields(first); got != `kind=4 x=2.25 y=13.16015625 w=136.74609375 h=13.5 size=12 text="Row 0 alpha beta gamma"` {
		t.Fatalf("first op = %s, want golden", got)
	}

	last := res.Ops[len(res.Ops)-1]
	if got := fmtOpFields(last); got != `kind=3 x=255.744140625 y=7638.343125 w=41.361328125 h=0 size=0 text=""` {
		t.Fatalf("last op = %s, want golden", got)
	}
}

// TestWrapBreakSkipKeepsAvoidSemantics pins that the wrap-flag fast path
// (skipping applyWrapBreakFlags when an element carries no wrap properties)
// keeps avoid behavior intact: avoid still glues a fitting boundary word,
// while the same markup without the property still wraps.
func TestWrapBreakSkipKeepsAvoidSemantics(t *testing.T) {
	t.Parallel()

	doc := func(span string) string {
		return `<html style="margin:0"><body style="margin:0;font:10pt/12pt monospace;width:200pt">` +
			`<p id="p" style="margin:0">aaaa aaaa aaaa aaaa aaaa aaaa aa ` +
			`<span id="s" style="` + span + `">x</span></p>` +
			`</body></html>`
	}

	plain := layoutHTML(t, doc(""))
	avoid := layoutHTML(t, doc("wrap-before:avoid"))

	plainFirst := textOpContaining(t, plain, "aaaa")
	plainX := textOpContaining(t, plain, "x")

	if plainX.Y <= plainFirst.Y {
		t.Fatalf("control kept the boundary word on line 1 (%.2f), want it pushed to line 2", plainX.Y)
	}

	avoidFirst := textOpContaining(t, avoid, "aaaa")
	avoidX := textOpContaining(t, avoid, "x")

	if !near(avoidX.Y, avoidFirst.Y) {
		t.Fatalf("wrap-before avoid boundary Y = %.2f, want glued on line 1 (%.2f)",
			avoidX.Y, avoidFirst.Y)
	}

	if lines := distinctTextYs(avoid); lines != 1 {
		t.Fatalf("fitting avoid fixture line count = %d, want 1", lines)
	}
}

// TestInlineCollectWrappedWordItemParity pins the per-fragment hoist:
// wrappedWordItem with caller-hoisted invariants must equal textItem for
// every style variant, including the vertical column-width path.
func TestInlineCollectWrappedWordItemParity(t *testing.T) {
	t.Parallel()

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatalf("LoadDefaultFaces: %v", err)
	}

	eng := &engine{font: faces.Regular, faces: faces, scale: 1}

	base := initialStyle()
	base.FontSize = 12

	normal := base
	normal.LineHeight = 0

	leading := base
	leading.LineHeight = 15

	spaced := base
	spaced.LetterSpacing = 1.5
	spaced.WordSpacing = 2

	upper := base
	upper.TextTransform = textTransformUppercase

	vertical := base
	vertical.WritingMode = writingModeVerticalRL

	words := []string{"hello ", "world", "x", "with-hyphen "}

	for name, style := range map[string]ResolvedStyle{
		"base":     base,
		"normal":   normal,
		"leading":  leading,
		"spaced":   spaced,
		"upper":    upper,
		"vertical": vertical,
	} {
		style := style
		lineH := eng.lineHeightOf(&style) * eng.scale
		isVert := isVerticalWritingMode(style.WritingMode)

		var vertW float64

		if isVert {
			vertW = verticalWritingColumnWidth(eng, &style)
		}

		for _, word := range words {
			want := eng.textItem(word, &style)
			got := eng.wrappedWordItem(word, &style, lineH, vertW, isVert)

			if got.text != want.text || got.style != want.style ||
				got.w != want.w || got.h != want.h {
				t.Fatalf("%s %q: wrappedWordItem = %+v, want textItem %+v",
					name, word, got, want)
			}
		}
	}
}

// TestInlineCollectPreserveTrailingGap pins the gap fast path: a trailing
// space stays attached with a width identical to the full remeasure, a node
// without trailing whitespace gains nothing, and autospace text still takes
// the full remeasure fallback with the same result.
func TestInlineCollectPreserveTrailingGap(t *testing.T) {
	t.Parallel()

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatalf("LoadDefaultFaces: %v", err)
	}

	collect := func(text, autospace string) []inlineItem {
		node := &html.Node{Type: html.TextNode, Text: text}
		style := initialStyle()
		style.FontSize = 12
		style.TextAutospace = autospace
		eng := &engine{
			font:   faces.Regular,
			faces:  faces,
			scale:  1,
			styles: map[*html.Node]*ResolvedStyle{node: &style},
		}

		var out []inlineItem

		eng.collectWrappedText(node, style, &out)

		return out
	}

	got := collect("alpha beta ", "")
	if last := got[len(got)-1]; last.text != "beta " {
		t.Fatalf("trailing gap text = %q, want %q", last.text, "beta ")
	}

	// Positive: fast-path width equals the full remeasure.
	plain := initialStyle()
	plain.FontSize = 12
	eng := &engine{font: faces.Regular, faces: faces, scale: 1}
	wantW := eng.inlineTextWidth("beta ", &plain, false)

	if got[len(got)-1].w != wantW {
		t.Fatalf("trailing gap w = %v, want full remeasure %v", got[len(got)-1].w, wantW)
	}

	// Negative: no trailing whitespace means no gap is added.
	bare := collect("alpha beta", "")
	if last := bare[len(bare)-1]; last.text != "beta" {
		t.Fatalf("no-gap text = %q, want %q", last.text, "beta")
	}

	// Autospace fallback: same visible result as the full remeasure.
	mixed := collect("alpha \u4e2d ", textAutospaceIdeographAlpha)

	if last := mixed[len(mixed)-1]; !strings.HasSuffix(last.text, " ") {
		t.Fatalf("autospace gap text = %q, want trailing space kept", last.text)
	}
}

// TestInlineCollectHasWrapBreakProps pins the hook-skip gate: empty styles
// skip applyWrapBreakFlags, any wrap property keeps it.
func TestInlineCollectHasWrapBreakProps(t *testing.T) {
	t.Parallel()

	if hasWrapBreakProps(ResolvedStyle{}) {
		t.Fatal("empty style reports wrap props, want the skip path")
	}

	withBefore := ResolvedStyle{WrapBefore: wrapBreakAvoid}
	if !hasWrapBreakProps(withBefore) {
		t.Fatal("wrap-before:avoid reports no props, want the hook path")
	}

	withInside := ResolvedStyle{WrapInside: wrapBreakAvoid}
	if !hasWrapBreakProps(withInside) {
		t.Fatal("wrap-inside:avoid reports no props, want the hook path")
	}
}

// trailing-gap rule through the paint ops: a trailing space stays attached
// to the last word, and a node without one gains no phantom space.
func TestInlineCollectTrailingGapKeepsWordItems(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0;font:10pt/12pt monospace;width:400pt">`+
		`<p id="p" style="margin:0">foo <b>bar</b> baz qux</p></body></html>`)

	foo := textOpContaining(t, res, "foo")
	if !strings.HasSuffix(foo.Text, " ") {
		t.Fatalf("trailing gap op text = %q, want trailing space kept", foo.Text)
	}

	qux := textOpContaining(t, res, "qux")
	if strings.HasSuffix(qux.Text, " ") {
		t.Fatalf("final word op text = %q, want no trailing space added", qux.Text)
	}
}
