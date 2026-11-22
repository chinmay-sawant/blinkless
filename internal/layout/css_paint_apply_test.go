package layout

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

func TestCurrentColor(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="outer">
			<p class="inner">x</p>
			<p class="own">y</p>
			<p class="outline">z</p>
			<p class="fold">w</p>
		</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.outer { color: red }
		.inner { border-color: currentColor }
		.own { color: currentColor }
		.outline { outline-color: currentColor }
		.fold { border-top-color: currentcolor }
	`)}, "print", testViewport, 800)

	red := [3]float64{1, 0, 0}

	inner := styleByClass(t, styles, "inner")
	if inner.Color != red {
		t.Fatalf("inner Color = %v, want red inherited", inner.Color)
	}

	if inner.borders().BorderTop.Color != red || inner.borders().BorderRight.Color != red ||
		inner.borders().BorderBottom.Color != red || inner.borders().BorderLeft.Color != red {
		t.Fatalf("inner border-color currentColor = T%v R%v B%v L%v, want red",
			inner.borders().BorderTop.Color, inner.borders().BorderRight.Color, inner.borders().BorderBottom.Color, inner.borders().BorderLeft.Color)
	}

	own := styleByClass(t, styles, "own")
	if own.Color != red {
		t.Fatalf("own color:currentColor = %v, want parent red", own.Color)
	}

	outline := styleByClass(t, styles, "outline")
	if outline.borders().OutlineColor != red || !outline.borders().OutlineColorSet {
		t.Fatalf("outline-color currentColor = %v set=%v, want red", outline.borders().OutlineColor, outline.borders().OutlineColorSet)
	}

	fold := styleByClass(t, styles, "fold")
	if fold.borders().BorderTop.Color != red {
		t.Fatalf("border-top-color currentcolor = %v, want red", fold.borders().BorderTop.Color)
	}
}

func TestOutlineParse(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="sh">x</div>
		<div class="width">x</div>
		<div class="thin">x</div>
		<div class="thick">x</div>
		<div class="dashed">x</div>
		<div class="dotted">x</div>
		<div class="none">x</div>
		<div class="color">x</div>
		<div class="offset">x</div>
		<div class="box">x</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.sh { outline: 2pt solid red }
		.width { outline-width: medium }
		.thin { outline-width: thin }
		.thick { outline-width: thick }
		.dashed { outline-style: dashed }
		.dotted { outline-style: dotted }
		.none { outline-style: none }
		.color { outline-color: blue }
		.offset { outline-offset: 4pt }
		.box { width: 100pt; height: 40pt; outline: 8pt solid red }
	`)}, "print", testViewport, 800)

	sh := styleByClass(t, styles, "sh")
	if !near(sh.borders().OutlineWidth, 2) {
		t.Fatalf("outline width = %.3f, want 2pt", sh.borders().OutlineWidth)
	}

	if sh.borders().OutlineStyle != solidKeyword {
		t.Fatalf("outline style = %q, want solid", sh.borders().OutlineStyle)
	}

	if sh.borders().OutlineColor != ([3]float64{1, 0, 0}) || !sh.borders().OutlineColorSet {
		t.Fatalf("outline color = %v set=%v, want red", sh.borders().OutlineColor, sh.borders().OutlineColorSet)
	}

	if !near(styleByClass(t, styles, "width").borders().OutlineWidth, borderWidth(mediumKeyword, 12)) {
		t.Fatalf("outline-width:medium = %.3f, want medium", styleByClass(t, styles, "width").borders().OutlineWidth)
	}

	if !near(styleByClass(t, styles, "thin").borders().OutlineWidth, borderWidth("thin", 12)) {
		t.Fatalf("outline-width:thin = %.3f", styleByClass(t, styles, "thin").borders().OutlineWidth)
	}

	if !near(styleByClass(t, styles, "thick").borders().OutlineWidth, borderWidth("thick", 12)) {
		t.Fatalf("outline-width:thick = %.3f", styleByClass(t, styles, "thick").borders().OutlineWidth)
	}

	if got := styleByClass(t, styles, "dashed").borders().OutlineStyle; got != "dashed" {
		t.Fatalf("outline-style:dashed = %q", got)
	}

	if got := styleByClass(t, styles, "dotted").borders().OutlineStyle; got != "dotted" {
		t.Fatalf("outline-style:dotted = %q", got)
	}

	if got := styleByClass(t, styles, "none").borders().OutlineStyle; got != cssDisplayNone {
		t.Fatalf("outline-style:none = %q", got)
	}

	blue := styleByClass(t, styles, "color")
	if blue.borders().OutlineColor != ([3]float64{0, 0, 1}) || !blue.borders().OutlineColorSet {
		t.Fatalf("outline-color:blue = %v set=%v", blue.borders().OutlineColor, blue.borders().OutlineColorSet)
	}

	if !near(styleByClass(t, styles, "offset").borders().OutlineOffset, 4) {
		t.Fatalf("outline-offset = %.3f, want 4pt", styleByClass(t, styles, "offset").borders().OutlineOffset)
	}

	box := styleByClass(t, styles, "box")
	if !near(box.Width, 100) || !near(box.Height, 40) {
		t.Fatalf("outline changed layout size: width=%.3f height=%.3f", box.Width, box.Height)
	}

	if !near(box.borders().OutlineWidth, 8) || box.borders().OutlineStyle != solidKeyword {
		t.Fatalf("box outline = width %.3f style %q", box.borders().OutlineWidth, box.borders().OutlineStyle)
	}
}

func TestRadiusLonghand(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="tl">x</div>
		<div class="tr">x</div>
		<div class="br">x</div>
		<div class="bl">x</div>
		<div class="pct">x</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.tl { border-top-left-radius: 8pt }
		.tr { border-top-right-radius: 6pt }
		.br { border-bottom-right-radius: 4pt }
		.bl { border-bottom-left-radius: 2pt }
		.pct { border-top-left-radius: 50% }
	`)}, "print", testViewport, 800)

	tl := styleByClass(t, styles, "tl")
	if !near(tl.borders().BorderRadiusTopLeft, 8) {
		t.Fatalf("border-top-left-radius = %.3f, want 8pt", tl.borders().BorderRadiusTopLeft)
	}

	if !near(tl.borders().BorderRadiusPercent, -1) {
		t.Fatalf("absolute longhand should clear BorderRadiusPercent, got %.3f", tl.borders().BorderRadiusPercent)
	}

	if !near(styleByClass(t, styles, "tr").borders().BorderRadiusTopRight, 6) {
		t.Fatalf("border-top-right-radius = %.3f, want 6pt", styleByClass(t, styles, "tr").borders().BorderRadiusTopRight)
	}

	if !near(styleByClass(t, styles, "br").borders().BorderRadiusBottomRight, 4) {
		t.Fatalf("border-bottom-right-radius = %.3f, want 4pt", styleByClass(t, styles, "br").borders().BorderRadiusBottomRight)
	}

	if !near(styleByClass(t, styles, "bl").borders().BorderRadiusBottomLeft, 2) {
		t.Fatalf("border-bottom-left-radius = %.3f, want 2pt", styleByClass(t, styles, "bl").borders().BorderRadiusBottomLeft)
	}

	pct := styleByClass(t, styles, "pct")
	if !near(pct.borders().BorderRadiusPercent, 50) {
		t.Fatalf("border-top-left-radius:50%% BorderRadiusPercent = %.3f, want 50", pct.borders().BorderRadiusPercent)
	}

	if !near(pct.borders().BorderRadiusTopLeft, 50) {
		t.Fatalf("border-top-left-radius:50%% corner = %.3f, want 50", pct.borders().BorderRadiusTopLeft)
	}
}

func TestBackgroundImageParse(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="dq">x</div>
		<div class="sq">x</div>
		<div class="bare">x</div>
		<div class="none">x</div>
		<div class="grad">x</div>
		<div class="sh">x</div>
		<div class="clear">x</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.dq { background-image: url("logo.png") }
		.sq { background-image: url('mark.jpg') }
		.bare { background-image: url(icon.svg) }
		.none { background-image: url("x.png"); }
		.none { background-image: none }
		.grad { background-image: linear-gradient(red, blue) }
		.sh { background: url("banner.png") red }
		.clear { background: none }
	`)}, "print", testViewport, 800)

	if got := styleByClass(t, styles, "dq").background().BackgroundImage; got != "logo.png" {
		t.Fatalf("double-quoted url = %q, want logo.png", got)
	}

	if got := styleByClass(t, styles, "sq").background().BackgroundImage; got != "mark.jpg" {
		t.Fatalf("single-quoted url = %q, want mark.jpg", got)
	}

	if got := styleByClass(t, styles, "bare").background().BackgroundImage; got != "icon.svg" {
		t.Fatalf("bare url = %q, want icon.svg", got)
	}

	if got := styleByClass(t, styles, "none").background().BackgroundImage; got != "" {
		t.Fatalf("background-image:none = %q, want empty", got)
	}

	if got := styleByClass(t, styles, "grad").background().BackgroundImage; got != "linear-gradient(red, blue)" {
		t.Fatalf("gradient background = %q, want linear-gradient(red, blue)", got)
	}

	sh := styleByClass(t, styles, "sh")
	if sh.background().BackgroundImage != "banner.png" {
		t.Fatalf("background shorthand url = %q, want banner.png", sh.background().BackgroundImage)
	}

	if sh.BGColor != ([4]float64{1, 0, 0, 1}) {
		t.Fatalf("background shorthand color = %v, want red", sh.BGColor)
	}

	if got := styleByClass(t, styles, "clear").background().BackgroundImage; got != "" {
		t.Fatalf("background:none image = %q, want empty", got)
	}
}

func TestListStylePositionParse(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<ul class="outer">
			<li class="in">a</li>
			<li class="out">b</li>
			<li class="sh">c</li>
			<li class="type">d</li>
		</ul>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.outer { list-style-position: inside }
		.in { list-style-position: inside }
		.out { list-style-position: outside }
		.sh { list-style: square inside }
		.type { list-style-type: decimal }
	`)}, "print", testViewport, 800)

	if got := styleByClass(t, styles, "in").content().ListStylePosition; got != listPosInside {
		t.Fatalf("list-style-position:inside = %q", got)
	}

	if got := styleByClass(t, styles, "out").content().ListStylePosition; got != listPosOutside {
		t.Fatalf("list-style-position:outside = %q", got)
	}

	sh := styleByClass(t, styles, "sh")
	if sh.content().ListStylePosition != listPosInside {
		t.Fatalf("list-style shorthand position = %q, want inside", sh.content().ListStylePosition)
	}

	if sh.content().ListStyleType != listStyleSquare {
		t.Fatalf("list-style shorthand type = %q, want square", sh.content().ListStyleType)
	}

	inherited := styleByClass(t, styles, "type")
	if inherited.content().ListStylePosition != listPosInside {
		t.Fatalf("list-style-position did not inherit (type-only child) = %q", inherited.content().ListStylePosition)
	}

	if inherited.content().ListStyleType != listStyleDecimal {
		t.Fatalf("list-style-type override = %q, want decimal", inherited.content().ListStyleType)
	}
}

func TestCounterResetIncrement(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="outer">
			<p class="child">x</p>
		</div>
		<div class="inc">y</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.outer { counter-reset: section 0 }
		.inc { counter-increment: section }
	`)}, "print", testViewport, 800)

	outer := styleByClass(t, styles, "outer")
	if outer.content().CounterReset != "section 0" {
		t.Fatalf("counter-reset = %q, want %q", outer.content().CounterReset, "section 0")
	}

	if outer.content().CounterIncrement != "" {
		t.Fatalf("outer CounterIncrement = %q, want empty", outer.content().CounterIncrement)
	}

	child := styleByClass(t, styles, "child")
	if child.content().CounterReset != "" {
		t.Fatalf("counter-reset inherited = %q, want empty", child.content().CounterReset)
	}

	if child.content().CounterIncrement != "" {
		t.Fatalf("counter-increment inherited = %q, want empty", child.content().CounterIncrement)
	}

	inc := styleByClass(t, styles, "inc")
	if inc.content().CounterIncrement != "section" {
		t.Fatalf("counter-increment = %q, want section", inc.content().CounterIncrement)
	}

	if inc.content().CounterReset != "" {
		t.Fatalf("inc CounterReset = %q, want empty", inc.content().CounterReset)
	}
}

func TestQuotesParse(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="q">
			<p class="child">x</p>
		</div>
		<div class="none">y</div>
		<div class="own">z</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.q { quotes: "«" "»" }
		.none { quotes: none }
		.own { quotes: '‹' '›' }
	`)}, "print", testViewport, 800)

	q := styleByClass(t, styles, "q")
	if q.content().QuotesOpen != "«" || q.content().QuotesClose != "»" {
		t.Fatalf("quotes = %q %q, want « »", q.content().QuotesOpen, q.content().QuotesClose)
	}

	child := styleByClass(t, styles, "child")
	if child.content().QuotesOpen != "«" || child.content().QuotesClose != "»" {
		t.Fatalf("quotes did not inherit: %q %q", child.content().QuotesOpen, child.content().QuotesClose)
	}

	none := styleByClass(t, styles, "none")
	if none.content().QuotesOpen != "" || none.content().QuotesClose != "" {
		t.Fatalf("quotes:none = %q %q, want empty", none.content().QuotesOpen, none.content().QuotesClose)
	}

	own := styleByClass(t, styles, "own")
	if own.content().QuotesOpen != "‹" || own.content().QuotesClose != "›" {
		t.Fatalf("single-quoted quotes = %q %q, want ‹ ›", own.content().QuotesOpen, own.content().QuotesClose)
	}
}

func TestOutlineDoesNotInherit(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="outer"><p class="child">x</p></div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.outer { outline: 3pt dashed blue }
	`)}, "print", testViewport, 800)

	child := styleByClass(t, styles, "child")
	if child.borders().OutlineStyle != "" || child.borders().OutlineWidth != 0 {
		t.Fatalf("outline inherited: style=%q width=%.3f", child.borders().OutlineStyle, child.borders().OutlineWidth)
	}
}
