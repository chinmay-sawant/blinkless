package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

func TestCascadeShorthandRespectsSourceOrder(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body><h1>heading</h1><p>paragraph</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	sheetAfter := sheet(t, `h1, p { margin-top: 0 } h1 { margin: 25mm 0 7mm }`)
	styles := resolveStyles(root, []*css.Stylesheet{sheetAfter}, "print", testViewport, 800)
	heading := findElementByName(root, "h1")

	if got := styles[heading].MarginTop; got < 70 || got > 72 {
		t.Fatalf("later margin shorthand lost to earlier longhand: got %.2fpt", got)
	}

	root, err = html.Parse(`<html><body><h1>heading</h1></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	sheetBefore := sheet(t, `h1 { margin: 25mm 0 7mm } h1 { margin-top: 0 }`)
	styles = resolveStyles(root, []*css.Stylesheet{sheetBefore}, "print", testViewport, 800)
	heading = findElementByName(root, "h1")

	if got := styles[heading].MarginTop; got != 0 {
		t.Fatalf("later margin-top lost to earlier shorthand: got %.2fpt", got)
	}
}

func TestCascadeVendorAliasesShareSourceOrder(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		style string
		want  string
	}{
		{
			name:  "prefixed declaration comes later",
			style: "box-sizing:border-box;-webkit-box-sizing:content-box",
			want:  "content-box",
		},
		{
			name:  "standard declaration comes later",
			style: "-webkit-box-sizing:content-box;box-sizing:border-box",
			want:  borderBox,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := mustParse(t, `<html><body><div style="`+testCase.style+`">x</div></body></html>`)
			styles := resolveStyles(root, nil, "print", testViewport, 800)
			div := findElementByName(root, "div")

			if got := styles[div].BoxSizing; got != testCase.want {
				t.Fatalf("BoxSizing = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestMaskShorthandExpansionAndBorderReset(t *testing.T) {
	t.Parallel()

	wins := make(map[string]cascadeWin)
	applyCascadeDeclaration(wins, "mask-image", "url(old.png)", 0, 0, 0, 1, 0, false)
	applyCascadeDeclaration(wins, "mask-border-source", "url(border.png)", 0, 0, 0, 1, 0, false)
	applyCascadeDeclaration(
		wins, maskShorthandProperty,
		`url(mask.png) center / cover no-repeat alpha padding-box intersect`,
		0, 0, 0, 2, 0, false,
	)
	applyCascadeDeclaration(wins, "mask-mode", "luminance", 0, 0, 0, 3, 0, false)

	want := map[string]string{
		"mask-image":         "url(mask.png)",
		"mask-mode":          "luminance",
		"mask-position":      "center",
		"mask-size":          "cover",
		"mask-repeat":        "no-repeat",
		"mask-origin":        "padding-box",
		"mask-clip":          "padding-box",
		"mask-composite":     "intersect",
		"mask-border":        "initial",
		"mask-border-source": "none",
		"mask-border-mode":   "alpha",
		"mask-border-slice":  "0",
		"mask-border-width":  "auto",
		"mask-border-outset": "0",
		"mask-border-repeat": "stretch",
	}
	for prop, value := range want {
		if got := wins[prop].value; got != value {
			t.Errorf("mask cascade %s = %q, want %q", prop, got, value)
		}
	}
}

func TestMaskShorthandGradientImageAndBorderReset(t *testing.T) {
	t.Parallel()

	const gradient = "linear-gradient(red, blue)"

	wins := make(map[string]cascadeWin)
	applyCascadeDeclaration(wins, "mask-image", "url(old.png)", 0, 0, 0, 1, 0, false)
	applyCascadeDeclaration(wins, "mask-border-source", "url(border.png)", 0, 0, 0, 1, 0, false)
	applyCascadeDeclaration(wins, maskShorthandProperty, gradient, 0, 0, 0, 2, 0, false)

	// An image-only mask shorthand keeps the function text and the layer
	// defaults from newMaskShorthandState, and still resets mask-border.
	want := map[string]string{
		"mask-image":         gradient,
		"mask-mode":          "match-source",
		"mask-position":      "0% 0%",
		"mask-size":          "auto",
		"mask-repeat":        "repeat",
		"mask-origin":        "border-box",
		"mask-clip":          "border-box",
		"mask-composite":     "add",
		"mask-border":        "initial",
		"mask-border-source": "none",
		"mask-border-mode":   "alpha",
		"mask-border-slice":  "0",
		"mask-border-width":  "auto",
		"mask-border-outset": "0",
		"mask-border-repeat": "stretch",
	}
	for prop, value := range want {
		if got := wins[prop].value; got != value {
			t.Errorf("mask cascade %s = %q, want %q", prop, got, value)
		}
	}
}

func TestMaskShorthandImageSetAndBorderReset(t *testing.T) {
	t.Parallel()

	const imageSet = "image-set(url(b.png) 1x)"

	wins := make(map[string]cascadeWin)
	applyCascadeDeclaration(wins, maskShorthandProperty, imageSet, 0, 0, 0, 1, 0, false)

	if got := wins["mask-image"].value; got != imageSet {
		t.Fatalf("mask-image = %q, want %q", got, imageSet)
	}

	for prop, value := range map[string]string{
		"mask-border":        "initial",
		"mask-border-source": "none",
		"mask-border-mode":   "alpha",
		"mask-border-slice":  "0",
		"mask-border-width":  "auto",
		"mask-border-outset": "0",
		"mask-border-repeat": "stretch",
	} {
		if got := wins[prop].value; got != value {
			t.Errorf("mask cascade %s = %q, want %q", prop, got, value)
		}
	}
}

func TestMaskVendorAliasDeclarationSourceOrder(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name  string
		style string
		want  string
	}{
		{
			name:  "prefixed declaration comes later",
			style: "mask-image:url(standard.png);-webkit-mask-image:url(prefixed.png)",
			want:  "url(prefixed.png)",
		},
		{
			name:  "standard declaration comes later",
			style: "-webkit-mask-image:url(prefixed.png);mask-image:url(standard.png)",
			want:  "url(standard.png)",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := mustParse(t, `<html><body><div style="`+testCase.style+`">x</div></body></html>`)
			div := findElementByName(root, "div")
			raw := cascadeRaw(nil, div, nil)

			if got := raw[maskImageProperty]; got != testCase.want {
				t.Fatalf("mask-image winner = %q, want %q; raw=%v", got, testCase.want, raw)
			}
		})
	}
}

func TestMaskShorthandCSSWideKeywords(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div id="parent" style="mask-image:url(parent.png);`+
		`mask-mode:luminance;mask-position:center;mask-size:cover;mask-repeat:no-repeat;`+
		`mask-origin:padding-box;mask-clip:content-box;mask-composite:exclude">`+
		`<span id="reset" style="mask-image:url(old.png);mask-mode:luminance;mask:initial"></span>`+
		`<span id="inherit" style="mask:inherit"></span></div></body></html>`)
	styles := resolveStyles(root, nil, "print", testViewport, 800)
	reset := styles[findElementByID(t, root, "reset")]

	if got, want := maskStyleValuesOf(reset), initialMaskStyleValues(); got != want {
		t.Fatalf("mask:initial fields = %+v, want %+v", got, want)
	}

	parent := styles[findElementByID(t, root, "parent")]
	inherited := styles[findElementByID(t, root, "inherit")]

	if got, want := maskStyleValuesOf(inherited), maskStyleValuesOf(parent); got != want {
		t.Fatalf("mask:inherit = %+v, parent = %+v", got, want)
	}
}

func TestMaskShorthandResolvesCustomPropertiesAndKeepsLonghandOrder(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>`+
		`<div id="shorthand-wins" style="--mask:url(variable.png) 25% 50% / cover no-repeat luminance padding-box intersect;`+
		`mask-image:url(old.png);mask:var(--mask)"></div>`+
		`<div id="longhand-wins" style="--mask:url(variable.png) center / contain no-repeat alpha content-box;`+
		`mask:var(--mask);mask-image:url(last.png)"></div>`+
		`<div id="prefixed" style="--mask:url(prefixed.png) right / contain no-repeat alpha content-box;`+
		`-webkit-mask:var(--mask)"></div>`+
		`</body></html>`)
	styles := resolveStyles(root, nil, "print", testViewport, 800)
	styleByID := func(identifier string) *ResolvedStyle {
		return styles[findElementByID(t, root, identifier)]
	}

	for _, testCase := range []struct {
		id   string
		want maskStyleValues
	}{
		{
			id: "shorthand-wins",
			want: maskStyleValues{
				image: "url(variable.png)", mode: maskModeLuminance, position: "25% 50%",
				size: "cover", repeat: maskRepeatNoRepeat, origin: "padding-box",
				clip: "padding-box", composite: "intersect",
			},
		},
		{
			id: "longhand-wins",
			want: maskStyleValues{
				image: "url(last.png)", mode: maskModeAlpha, position: "center",
				size: "contain", repeat: maskRepeatNoRepeat, origin: "content-box",
				clip: "content-box", composite: maskCompositeAdd,
			},
		},
		{
			id: "prefixed",
			want: maskStyleValues{
				image: "url(prefixed.png)", mode: maskModeAlpha, position: "right",
				size: "contain", repeat: maskRepeatNoRepeat, origin: "content-box",
				clip: "content-box", composite: maskCompositeAdd,
			},
		},
	} {
		got := maskStyleValuesOf(styleByID(testCase.id))
		if got != testCase.want {
			t.Errorf("%s mask fields = %+v, want %+v", testCase.id, got, testCase.want)
		}
	}
}

// TestCascadeListStyleShorthandBeatsUADisc: an author list-style shorthand
// must beat the UA `ul { list-style-type: disc }` longhand on the same
// element (fixture-61 rows 108-111). The shorthand expands at cascade time
// with its own origin and specificity; without that, both keys coexisted in
// raw and the later-applied UA longhand won regardless of origin, so
// `ul { list-style: none }` still painted discs.
func TestCascadeListStyleShorthandBeatsUADisc(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><ul><li>a</li><li>b</li></ul></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `ul { list-style: none }`)},
		"print", testViewport, 800)
	ul := findElementByName(root, "ul")
	listItem := findElementByName(root, "li")

	if got := styles[ul].ListStyleType; got != "none" {
		t.Fatalf("ul list-style:none type = %q, want none", got)
	}

	if got := styles[listItem].ListStyleType; got != "none" {
		t.Fatalf("li inherited type = %q, want none", got)
	}

	res := layoutHTML(t, `<html><body><ul><li>a</li><li>b</li></ul></body></html>`,
		sheet(t, `ul { list-style: none }`))
	if n := len(opsOfKind(res, OpBullet)); n != 0 {
		t.Fatalf("list-style:none bullets = %d, want 0", n)
	}

	res = layoutHTML(t, `<html><body><ul style="list-style:none;"><li>a</li></ul></body></html>`)
	if n := len(opsOfKind(res, OpBullet)); n != 0 {
		t.Fatalf("inline list-style:none bullets = %d, want 0", n)
	}

	// Same-origin source order still holds between shorthand and longhand.
	typeCase := layoutHTML(t, `<html><body><ul><li>a</li></ul></body></html>`,
		sheet(t, `ul { list-style: square; list-style-type: decimal }`))
	if bullets := opsOfKind(typeCase, OpBullet); len(bullets) != 1 || bullets[0].Text != "1." {
		t.Fatalf("later longhand decimal lost to shorthand: %+v", bullets)
	}

	typeCase = layoutHTML(t, `<html><body><ul><li>a</li></ul></body></html>`,
		sheet(t, `ul { list-style-type: decimal; list-style: square }`))
	if bullets := opsOfKind(typeCase, OpBullet); len(bullets) != 1 || bullets[0].Text != "\u25AA" {
		t.Fatalf("later shorthand square lost to longhand: %+v", bullets)
	}
}

// TestCascadeBorderShorthandOverridesEarlierBorderTop: a later border
// shorthand must beat an earlier border-top longhand (fixture-56 domain-03
// accent top vs .domains > section frame).
func TestCascadeBorderShorthandOverridesEarlierBorderTop(t *testing.T) {
	t.Parallel()

	// Equal specificity (type+class): later border shorthand must win.
	root, err := html.Parse(`<html><body><div class="domains"><section class="d03">x</section></div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	cssSheet := sheet(t, `
section.d03 { border-top: 3px solid #2563eb }
.domains > section { border: 1px solid #cbd5e1; border-left: 4px solid #0f766e }
`)
	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", testViewport, 800)
	sec := findElementByName(root, "section")
	sty := styles[sec]

	if sty.BorderTop.Width > 1.5 {
		t.Fatalf("border-top width = %.2f, want ≤1.5 after later border shorthand", sty.BorderTop.Width)
	}

	// #2563eb accent must not survive; #cbd5e1 frame is ~ (0.80, 0.84, 0.88).
	if sty.BorderTop.Color[0] < 0.5 && sty.BorderTop.Color[2] > 0.85 {
		t.Fatalf("border-top still accent blue (%.2f,%.2f,%.2f), want neutral frame",
			sty.BorderTop.Color[0], sty.BorderTop.Color[1], sty.BorderTop.Color[2])
	}

	if sty.BorderLeft.Width < 2.5 {
		t.Fatalf("border-left width = %.2f, want accent rail ≥2.5", sty.BorderLeft.Width)
	}
}

// TestCascadeClampWinsAndResolves: clamp() is now evaluated by the math
// parser, so it wins the cascade and resolves to 70% of the 500pt viewport,
// between 24rem (288pt) and 46rem (552pt).
func TestCascadeClampWinsAndResolves(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body><table class="d01-exit"><tr><td>x</td></tr></table></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	cssSheet := sheet(t, `
table.d01-exit {
  width: 100%;
  width: clamp(24rem, 70%, 46rem);
}
`)
	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", testViewport, 800)
	tbl := findElementByName(root, "table")
	sty := styles[tbl]

	if !near(sty.Width, 350) {
		t.Fatalf("Width = %.2fpt, want 350 (70%% of 500pt viewport)", sty.Width)
	}

	if sty.WidthPercent >= 0 {
		t.Fatalf("WidthPercent = %.2f, want unset when clamp resolves absolutely", sty.WidthPercent)
	}
}

func TestWritingModeInherits(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body><div style="writing-mode: vertical-rl"><p>child text</p></div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	styles := resolveStyles(root, nil, "print", testViewport, 800)
	pElem := findElementByName(root, "p")

	if styles[pElem].WritingMode != "vertical-rl" {
		t.Fatalf("p WritingMode = %q, want %q", styles[pElem].WritingMode, "vertical-rl")
	}

	doc, err := html.Parse(`<html><body><div style="writing-mode: vertical-rl"><p>vertical</p></div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(doc, Options{Width: 800, Height: 600})
	if err != nil {
		t.Fatal(err)
	}

	var textOp *Op

	for i := range res.Ops {
		if res.Ops[i].Kind == OpText && res.Ops[i].Text == "vertical" {
			textOp = &res.Ops[i]

			break
		}
	}

	if textOp == nil {
		t.Fatal("no OpText found for 'vertical'")

		return
	}

	if textOp.RotateDeg != -90 {
		t.Fatalf("textOp.RotateDeg = %v, want -90", textOp.RotateDeg)
	}
}

//nolint:cyclop // test assertion helper
func TestTextIndentInheritsAndShiftsFirstLine(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body><div style="text-indent: 2em"><p><span>indented</span></p></div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	styles := resolveStyles(root, nil, "print", testViewport, 800)
	pNode := findElementByName(root, "p")
	spanNode := findElementByName(root, "span")

	if styles[pNode].TextIndent <= 0 {
		t.Fatalf("p TextIndent = %v, want > 0", styles[pNode].TextIndent)
	}

	if styles[spanNode].TextIndent != styles[pNode].TextIndent {
		t.Fatalf("span TextIndent = %v, want %v (inherited)", styles[spanNode].TextIndent, styles[pNode].TextIndent)
	}

	docPlain, err := html.Parse(`<html><body><p style="margin:0;font-size:16px">hello</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	resPlain, err := Layout(docPlain, Options{Width: 800, Height: 600})
	if err != nil {
		t.Fatal(err)
	}

	docIndent, err := html.Parse(`<html><body><p style="margin:0;font-size:16px;text-indent:2em">hello</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	resIndent, err := Layout(docIndent, Options{Width: 800, Height: 600})
	if err != nil {
		t.Fatal(err)
	}

	var xPlain, xIndent float64

	for _, op := range resPlain.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "hello") {
			xPlain = op.X

			break
		}
	}

	for _, op := range resIndent.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "hello") {
			xIndent = op.X

			break
		}
	}

	if xIndent <= xPlain {
		t.Fatalf("xIndent (%v) must be greater than xPlain (%v)", xIndent, xPlain)
	}
}

//nolint:gocognit,cyclop,funlen,gocyclo,lll,wsl // table-driven webkit aliases assertion
func TestWebkitPrefixAliases(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body>
		<div class="box-sizing" style="-webkit-box-sizing: border-box">x</div>
		<div class="radius" style="-webkit-border-radius: 12pt">x</div>
		<div class="flex" style="-webkit-flex-direction: column; -webkit-flex-wrap: wrap; ` +
		`-webkit-justify-content: center; -webkit-align-items: center; ` +
		`-webkit-align-content: space-between; -webkit-order: 3">x</div>
		<div class="transform" style="-webkit-transform: rotate(45deg)">x</div>
		<div class="shadow" style="-webkit-box-shadow: 2pt 2pt 4pt #000">x</div>
		<div class="box-align-start" style="-webkit-box-align: start">x</div>
		<div class="box-align-baseline" style="-webkit-box-align: baseline">x</div>
		<div class="box-flex" style="-webkit-box-flex: 2">x</div>
		<div class="box-ordinal" style="-webkit-box-ordinal-group: 3">x</div>
		<div class="box-orient-h" style="-webkit-box-orient: horizontal">x</div>
		<div class="box-orient-v" style="-webkit-box-orient: vertical">x</div>
		<div class="box-pack" style="-webkit-box-pack: justify">x</div>
		<div class="text-fill" style="-webkit-text-fill-color: #ff0000">x</div>
		<div class="webkit-box" style="display: -webkit-box">x</div>
		<div class="webkit-inline-box" style="display: -webkit-inline-box">x</div>
	</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	styles := resolveStyles(root, nil, "print", testViewport, 800)

	body := findElementByName(root, "body")
	if body == nil {
		t.Fatal("body element not found")
	}

	for _, child := range body.Children {
		if child.Type != html.ElementNode {
			continue
		}

		sty := styles[child]
		switch child.Attribute("class") {
		case "box-sizing":
			if sty.BoxSizing != "border-box" {
				t.Fatalf("-webkit-box-sizing: got %q, want border-box", sty.BoxSizing)
			}
		case "radius":
			if !near(sty.BorderRadius, 12) {
				t.Fatalf("-webkit-border-radius: got %.1f, want 12", sty.BorderRadius)
			}
		case "flex":
			if sty.FlexDirection != "column" {
				t.Fatalf("-webkit-flex-direction: got %q, want column", sty.FlexDirection)
			}
			if sty.FlexWrap != "wrap" {
				t.Fatalf("-webkit-flex-wrap: got %q, want wrap", sty.FlexWrap)
			}
			if sty.JustifyContent != fxCenter {
				t.Fatalf("-webkit-justify-content: got %q, want center", sty.JustifyContent)
			}
			if sty.AlignItems != fxCenter {
				t.Fatalf("-webkit-align-items: got %q, want center", sty.AlignItems)
			}
			if sty.AlignContent != "space-between" {
				t.Fatalf("-webkit-align-content: got %q, want space-between", sty.AlignContent)
			}
			if sty.FlexOrder != 3 {
				t.Fatalf("-webkit-order: got %d, want 3", sty.FlexOrder)
			}
		case "transform":
			if !sty.HasTransform {
				t.Fatal("-webkit-transform: HasTransform false, want true")
			}
		case "shadow":
			if !sty.BoxShadowSet {
				t.Fatal("-webkit-box-shadow: BoxShadowSet false, want true")
			}
		case "box-align-start":
			if sty.AlignItems != flexStartKeyword {
				t.Fatalf("-webkit-box-align: start got %q, want %q", sty.AlignItems, flexStartKeyword)
			}
		case "box-align-baseline":
			if sty.AlignItems != "baseline" {
				t.Fatalf("-webkit-box-align: baseline got %q, want baseline", sty.AlignItems)
			}
		case "box-flex":
			if sty.FlexGrow != 2 {
				t.Fatalf("-webkit-box-flex: got %.1f, want 2", sty.FlexGrow)
			}
		case "box-ordinal":
			if sty.FlexOrder != 2 {
				t.Fatalf("-webkit-box-ordinal-group: 3 got %d, want 2 (order = group-1)", sty.FlexOrder)
			}
		case "box-orient-h":
			if sty.FlexDirection != fxRow {
				t.Fatalf("-webkit-box-orient: horizontal got %q, want %q", sty.FlexDirection, fxRow)
			}
		case "box-orient-v":
			if sty.FlexDirection != fxCol {
				t.Fatalf("-webkit-box-orient: vertical got %q, want %q", sty.FlexDirection, fxCol)
			}
		case "box-pack":
			if sty.JustifyContent != fxBetween {
				t.Fatalf("-webkit-box-pack: justify got %q, want %q", sty.JustifyContent, fxBetween)
			}
		case "text-fill":
			if sty.Color != [3]float64{1, 0, 0} {
				t.Fatalf("-webkit-text-fill-color: got %v, want red", sty.Color)
			}
		case "webkit-box":
			if sty.Display != displayFlex {
				t.Fatalf("display: -webkit-box got %q, want %q", sty.Display, displayFlex)
			}
			if !sty.IsWebkitBox {
				t.Fatal("display: -webkit-box IsWebkitBox=false, want true")
			}
		case "webkit-inline-box":
			if sty.Display != displayInlineFlex {
				t.Fatalf("display: -webkit-inline-box got %q, want %q", sty.Display, displayInlineFlex)
			}
		}
	}

	// Prefixed equals unprefixed equivalents for all 6 plus display.
	prefixed := map[string]string{
		"-webkit-box-align":         "center",
		"-webkit-box-flex":          "2.5",
		"-webkit-box-ordinal-group": "2",
		"-webkit-box-orient":        "vertical",
		"-webkit-box-pack":          "center",
		"-webkit-text-fill-color":   "blue",
	}
	unprefixed := map[string]string{
		"-webkit-box-align":         "align-items: center",
		"-webkit-box-flex":          "flex-grow: 2.5",
		"-webkit-box-ordinal-group": "order: 1",
		"-webkit-box-orient":        "flex-direction: column",
		"-webkit-box-pack":          "justify-content: center",
		"-webkit-text-fill-color":   "color: blue",
	}
	for prop, val := range prefixed {
		stylePrefixed := styleForDecl(t, prop+": "+val)
		decl := unprefixed[prop]
		parts := strings.SplitN(decl, ":", 2)
		styleCanon := styleForDecl(t, strings.TrimSpace(parts[0])+": "+strings.TrimSpace(parts[1]))
		if !stylesEqualForProp(prop, stylePrefixed, styleCanon) {
			t.Fatalf("%s: %q style not equal to canonical %q", prop, val, decl)
		}
	}

	// Legacy -webkit-box maps to flex so box-* longhands can lay out.
	if s1, s2 := styleForDecl(t, "display: -webkit-box"), styleForDecl(t, "display: flex"); s1.Display != s2.Display {
		t.Fatalf("display -webkit-box %q != flex %q", s1.Display, s2.Display)
	}
	if s1, s2 := styleForDecl(t, "display: -webkit-inline-box"), styleForDecl(t, "display: inline-flex"); s1.Display != s2.Display {
		t.Fatalf("display -webkit-inline-box %q != inline-flex %q", s1.Display, s2.Display)
	}
}

//nolint:wsl // test helper
func styleForDecl(t *testing.T, decl string) *ResolvedStyle {
	t.Helper()
	root, err := html.Parse(`<html><body><div style="` + decl + `">x</div></body></html>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	styles := resolveStyles(root, nil, "print", testViewport, 800)

	body := findElementByName(root, "body")
	for _, child := range body.Children {
		if child.Type == html.ElementNode {
			return styles[child]
		}
	}

	t.Fatal("div not found")

	return nil
}

func stylesEqualForProp(prop string, first, second *ResolvedStyle) bool {
	switch prop {
	case "-webkit-box-align":
		return first.AlignItems == second.AlignItems
	case "-webkit-box-flex":
		return first.FlexGrow == second.FlexGrow
	case "-webkit-box-ordinal-group":
		return first.FlexOrder == second.FlexOrder
	case "-webkit-box-orient":
		return first.FlexDirection == second.FlexDirection
	case "-webkit-box-pack":
		return first.JustifyContent == second.JustifyContent
	case "-webkit-text-fill-color":
		return first.Color == second.Color
	default:
		return false
	}
}

//nolint:cyclop // SVG presentation test assertions
func TestSVGPresentationProps(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body>
		<svg style="fill: red; stroke: blue; stroke-width: 2pt; fill-opacity: 0.8; stroke-opacity: 0.5"></svg>
	</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	styles := resolveStyles(root, nil, "print", testViewport, 800)
	svgElem := findElementByName(root, "svg")

	if svgElem == nil {
		t.Fatal("svg element not found")
	}

	sty := styles[svgElem]
	if !sty.FillSet || sty.Fill != [3]float64{1, 0, 0} {
		t.Fatalf("Fill = %v, FillSet = %v, want red (1,0,0) and true", sty.Fill, sty.FillSet)
	}

	if !sty.StrokeSet || sty.Stroke != [3]float64{0, 0, 1} {
		t.Fatalf("Stroke = %v, StrokeSet = %v, want blue (0,0,1) and true", sty.Stroke, sty.StrokeSet)
	}

	if !sty.StrokeWidthSet || !near(sty.StrokeWidth, 2) {
		t.Fatalf("StrokeWidth = %.1f, StrokeWidthSet = %v, want 2 and true", sty.StrokeWidth, sty.StrokeWidthSet)
	}

	if !near(sty.FillOpacity, 0.8) {
		t.Fatalf("FillOpacity = %.2f, want 0.8", sty.FillOpacity)
	}

	if !near(sty.StrokeOpacity, 0.5) {
		t.Fatalf("StrokeOpacity = %.2f, want 0.5", sty.StrokeOpacity)
	}
}

// TestFontLonghandAfterShorthandWins: the font shorthand must compete with
// its longhands per property in cascade order. It used to be applied after
// every longhand, so an earlier `font` beat a later `font-size` regardless of
// order or specificity.
func TestFontLonghandAfterShorthandWins(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		css  string
		want float64
	}{
		{"later font-size wins", `p { font: 12pt serif } p { font-size: 20pt }`, 20},
		{"later font shorthand wins", `p { font-size: 20pt } p { font: 12pt serif }`, 12},
		{"specificity beats earlier shorthand", `p { font: 12pt serif } p.a { font-size: 20pt }`, 20},
		{"specificity beats earlier longhand", `p { font-size: 20pt } p.a { font: 12pt serif }`, 12},
		{"shorthand keeps its family", `p { font: italic 12pt serif } p { font-size: 14pt }`, 14},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html><body><p class="a">x</p></body></html>`, sheet(t, testCase.css))
			op := firstText(res)

			if !near(op.Size, testCase.want) {
				t.Fatalf("text size = %.3f, want %.3f", op.Size, testCase.want)
			}
		})
	}
}

// TestPseudoElementResolvesCustomProperties: generated content inherits the
// host's custom properties and var() references resolve before parsing, so
// p::after { color: var(--accent) } paints the host's accent color. The
// pseudo path used to skip mergeCustomProps and resolveRawVars, leaving the
// literal var() string for the color parser to drop.
func TestPseudoElementResolvesCustomProperties(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><p class="x">base</p></body></html>`, sheet(t, `
.x { --accent: #ff0000 }
.x::after { content: "tail"; color: var(--accent) }
`))

	for _, oper := range res.Ops {
		if oper.Kind != OpText || oper.Text != "tail" {
			continue
		}

		if !(oper.R > 0.9 && oper.G < 0.1 && oper.B < 0.1) {
			t.Fatalf("::after color = %.3f,%.3f,%.3f, want red from var(--accent)", oper.R, oper.G, oper.B)
		}

		return
	}

	t.Fatal("::after content op not found")
}

// TestCascadeRejectsInvalidDeclarationBeforeWin (CSS-01a): a declaration the
// engine cannot apply must not win the cascade. Before the acceptance gate,
// `width: bogus` outranked `width: 80px` and left the box auto-sized instead
// of keeping the valid declaration. 80px converts to 60pt at 96dpi.
func TestCascadeRejectsInvalidDeclarationBeforeWin(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		css  string
		want float64
	}{
		{"same rule later value invalid", `.box { width: 80px; width: bogus }`, 60},
		{"invalid earlier does not block later valid", `.box { width: bogus; width: 80px }`, 60},
		{"higher specificity invalid loses", `.box { width: 80px } .box#x { width: bogus }`, 60},
		{"invalid important loses to valid normal", `.box { width: 80px } .box#x { width: bogus !important }`, 60},
		{"valid later still wins", `.box { width: 80px; width: 120px }`, 90},
		{"display block accepted", `.box { display: block }`, 0},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := mustParse(t, `<html><body><div id="x" class="box">x</div></body></html>`)
			styles := resolveStyles(root, []*css.Stylesheet{sheet(t, testCase.css)}, "print", testViewport, 800)
			div := findElementByName(root, "div")

			if testCase.name == "display block accepted" {
				if got := styles[div].Display; got != displayBlock {
					t.Fatalf("Display = %q, want %q", got, displayBlock)
				}

				return
			}

			if got := styles[div].Width; !near(got, testCase.want) {
				t.Fatalf("Width = %.2fpt, want %.2fpt", got, testCase.want)
			}
		})
	}

	// display: block stays accepted; a later invalid display must not erase it.
	root := mustParse(t, `<html><body><div class="box">x</div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `.box { display: flex; display: bogus }`)},
		"print", testViewport, 800)
	div := findElementByName(root, "div")

	if got := styles[div].Display; got != displayFlex {
		t.Fatalf("Display = %q, want %q: invalid display outranked valid display", got, displayFlex)
	}
}

// TestCascadeVarAcceptanceRules (CSS-01c): var() declarations are valid at
// parse time. A missing variable with a fallback substitutes and applies; a
// substitution that cannot be applied leaves the property unset (like no
// declaration) and must not revive an earlier valid declaration.
func TestCascadeVarAcceptanceRules(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		css      string
		want     float64
		wantAuto bool
	}{
		{"valid substitution", `:root { --w: 80px } .box { width: var(--w) }`, 60, false},
		{"fallback when missing", `.box { width: var(--missing, 80px) }`, 60, false},
		{"fallback after valid declaration", `.box { width: 120px; width: var(--missing, 80px) }`, 60, false},
		{"missing without fallback unsets, no revival", `.box { width: 80px; width: var(--missing) }`, 0, true},
		{"invalid fallback unsets, no revival", `.box { width: 80px; width: var(--missing, bogus) }`, 0, true},
		{"invalid custom property unsets, no revival", `.box { --w: bogus; width: 80px; width: var(--w) }`, 0, true},
		{"css-wide initial resets to auto", `.box { width: 80px; width: initial }`, 0, true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := mustParse(t, `<html><body><div class="box">x</div></body></html>`)
			styles := resolveStyles(root, []*css.Stylesheet{sheet(t, testCase.css)}, "print", testViewport, 800)
			div := findElementByName(root, "div")
			got := styles[div].Width

			if testCase.wantAuto {
				// Unset must match an element with no width declaration at all.
				controlRoot := mustParse(t, `<html><body><div class="box">x</div></body></html>`)
				controlStyles := resolveStyles(controlRoot, nil, "print", testViewport, 800)
				control := findElementByName(controlRoot, "div")

				if !near(got, controlStyles[control].Width) {
					t.Fatalf("Width = %.2fpt, want unset matching control %.2fpt (no revival)",
						got, controlStyles[control].Width)
				}

				return
			}

			if !near(got, testCase.want) {
				t.Fatalf("Width = %.2fpt, want %.2fpt", got, testCase.want)
			}
		})
	}
}

// TestCascadeKeepsOrderingAfterAcceptanceGate (CSS-01d): shorthand resets,
// longhand order, specificity, inheritance, @layer, and vendor aliases still
// select the same values now that the gate validates declarations.
//
//nolint:funlen // cascade ordering regression; splitting would hide the sequence under test
func TestCascadeKeepsOrderingAfterAcceptanceGate(t *testing.T) {
	t.Parallel()

	longhandOrder := []struct {
		name string
		css  string
		want float64
	}{
		{"shorthand resets earlier longhand", `.box { margin-top: 12px; margin: 0 }`, 0},
		{"longhand after shorthand wins", `.box { margin: 0; margin-top: 12px }`, 9},
		{"higher specificity wins among valid values", `.box { width: 80px } .box#x { width: 120px }`, 90},
	}

	for _, testCase := range longhandOrder {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := mustParse(t, `<html><body><div id="x" class="box">x</div></body></html>`)
			styles := resolveStyles(root, []*css.Stylesheet{sheet(t, testCase.css)}, "print", testViewport, 800)
			div := findElementByName(root, "div")

			var got float64
			if strings.HasPrefix(testCase.name, "shorthand resets") ||
				strings.HasPrefix(testCase.name, "longhand after") {
				got = styles[div].MarginTop
			} else {
				got = styles[div].Width
			}

			if !near(got, testCase.want) {
				t.Fatalf("value = %.2fpt, want %.2fpt", got, testCase.want)
			}
		})
	}

	layerCases := []struct {
		name string
		css  string
		want float64
	}{
		{"unlayered beats layered", `@layer base { .box { width: 80px } } .box { width: 120px }`, 90},
		{"later layer rank wins", `@layer a, b; @layer a { .box { width: 80px } } @layer b { .box { width: 120px } }`, 90},
	}

	for _, testCase := range layerCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := mustParse(t, `<html><body><div class="box">x</div></body></html>`)
			styles := resolveStyles(root, []*css.Stylesheet{sheet(t, testCase.css)}, "print", testViewport, 800)
			div := findElementByName(root, "div")

			if got := styles[div].Width; !near(got, testCase.want) {
				t.Fatalf("Width = %.2fpt, want %.2fpt", got, testCase.want)
			}
		})
	}

	t.Run("inheritance still copies color", func(t *testing.T) {
		t.Parallel()

		root := mustParse(t, `<html><body><div style="color: #ff0000"><p>child</p></div></body></html>`)
		styles := resolveStyles(root, nil, "print", testViewport, 800)
		p := findElementByName(root, "p")

		if got := styles[p].Color; got != [3]float64{1, 0, 0} {
			t.Fatalf("p Color = %v, want red inherited", got)
		}
	})

	t.Run("vendor alias still maps", func(t *testing.T) {
		t.Parallel()

		root := mustParse(t, `<html><body><div style="-webkit-box-sizing: border-box">x</div></body></html>`)
		styles := resolveStyles(root, nil, "print", testViewport, 800)
		div := findElementByName(root, "div")

		if got := styles[div].BoxSizing; got != borderBox {
			t.Fatalf("BoxSizing = %q, want %q from -webkit-box-sizing", got, borderBox)
		}
	})
}

// TestCascadeEngineSupportsPropertyValues (CSS-02b): @supports needs handler
// ownership AND a value the handlers accept. Ownership alone reported true
// for display:bogus.
//
//nolint:funlen // value table plus gating sequence; splitting would hide the pair under test
func TestCascadeEngineSupportsPropertyValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		prop  string
		value string
		want  bool
	}{
		{"display", "block", true},
		{"display", "grid", true},
		{"display", "bogus", false},
		{"width", "80px", true},
		{"width", "auto", true},
		{"width", "bogus", false},
		{"min-width", "none", true},
		{"min-width", "bogus", false},
		{"margin-top", "auto", true},
		{"margin-top", "bogus", false},
		{"padding-top", "auto", false},
		{"padding-top", "4px", true},
		{"position", "static", true},
		{"position", "bogus", false},
		{"color", "#ff0000", true},
		{"unknown-property", "1", false},
		{"--custom", "bogus", true},
		{"width", "var(--w)", true},
		{"display", "inherit", true},
		{"-webkit-box-orient", "horizontal", true},
		{"mask", `url(mask.png) center / cover no-repeat`, true},
		{"mask", "url(mask.png) bogus", false},
		{"-webkit-mask", `url(mask.png) center / cover no-repeat`, true},
		{"-webkit-mask-composite", "source-over, xor", true},
		{"-webkit-mask-composite", "destination-out", false},
		{"opacity", "bogus", false},
		{"z-index", "3", true},
		{"z-index", "bogus", false},
	}

	for _, testCase := range cases {
		if got := engineSupportsProperty(testCase.prop, testCase.value); got != testCase.want {
			t.Errorf("engineSupportsProperty(%q, %q) = %v, want %v",
				testCase.prop, testCase.value, got, testCase.want)
		}
	}

	// The gate uses the same rule: an invalid feature query must not apply.
	root := mustParse(t, `<html><body><div class="box">x</div></body></html>`)
	styles := resolveStyles(root,
		[]*css.Stylesheet{sheet(t, `
			.box { width: 80px }
			@supports (display: bogus) { .box { width: 120px } }
		`)}, "print", testViewport, 800)
	div := findElementByName(root, "div")

	if got := styles[div].Width; !near(got, 60) {
		t.Fatalf("invalid @supports applied: Width = %.2fpt, want 60", got)
	}

	root = mustParse(t, `<html><body><div class="box">x</div></body></html>`)
	styles = resolveStyles(root,
		[]*css.Stylesheet{sheet(t, `
			.box { width: 80px }
			@supports (display: grid) { .box { width: 120px } }
		`)}, "print", testViewport, 800)
	div = findElementByName(root, "div")

	if got := styles[div].Width; !near(got, 90) {
		t.Fatalf("valid @supports did not apply: Width = %.2fpt, want 90", got)
	}

	root = mustParse(t, `<html><body><div class="box">x</div></body></html>`)
	styles = resolveStyles(root,
		[]*css.Stylesheet{sheet(t, `
			.box { width: 80px }
			@supports (mask: url(mask.png) center / cover no-repeat) { .box { width: 120px } }
		`)}, "print", testViewport, 800)
	div = findElementByName(root, "div")

	if got := styles[div].Width; !near(got, 90) {
		t.Fatalf("valid mask @supports did not apply: Width = %.2fpt, want 90", got)
	}
}

type maskStyleValues struct {
	image     string
	mode      string
	position  string
	size      string
	repeat    string
	origin    string
	clip      string
	composite string
}

func maskStyleValuesOf(style *ResolvedStyle) maskStyleValues {
	return maskStyleValues{
		image:     style.MaskImage,
		mode:      style.MaskMode,
		position:  style.MaskPosition,
		size:      style.MaskSize,
		repeat:    style.MaskRepeat,
		origin:    style.MaskOrigin,
		clip:      style.MaskClip,
		composite: style.MaskComposite,
	}
}

func initialMaskStyleValues() maskStyleValues {
	return maskStyleValues{
		image:     "none",
		mode:      maskModeMatchSource,
		position:  maskInitialPosition,
		size:      "auto",
		repeat:    "repeat",
		origin:    "border-box",
		clip:      "border-box",
		composite: maskCompositeAdd,
	}
}

func findElementByID(t *testing.T, root *html.Node, identifier string) *html.Node {
	t.Helper()

	if found := findElementByIDNode(root, identifier); found != nil {
		return found
	}

	t.Fatalf("element id=%q not found", identifier)

	return nil
}

func findElementByIDNode(root *html.Node, identifier string) *html.Node {
	if root == nil {
		return nil
	}

	if root.Type == html.ElementNode && root.Attribute("id") == identifier {
		return root
	}

	for _, child := range root.Children {
		if found := findElementByIDNode(child, identifier); found != nil {
			return found
		}
	}

	return nil
}

func findElementByName(root *html.Node, name string) *html.Node {
	if root.Type == html.ElementNode && root.Name == name {
		return root
	}

	for _, child := range root.Children {
		if found := findElementByName(child, name); found != nil {
			return found
		}
	}

	return nil
}
