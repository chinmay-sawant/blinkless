package layout

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// Wire tests for the standalone clip-rule property. Reference browser for
// every case: Chrome 143.0.7499.40. The star polygon comes from
// behaviorClipStarPolygon (css_behavior_clip_rule_test.go): its center sits
// inside the twice-wound inner pentagon, so nonzero keeps it painted and
// evenodd cuts it out. Every test below asserts used paint (decoded image
// alpha), never stored style strings.

// clipRuleWireCenterAlpha lays out one clipped img and returns the decoded
// center alpha of its single image op.
func clipRuleWireCenterAlpha(t *testing.T, styleAttr string) uint8 {
	t.Helper()

	res := layoutHTMLWithImages(t,
		`<html><body><img src="x.png" style="`+styleAttr+`"></body></html>`,
		tinyPNG(40, 40), "x.png")

	imgs := opsOfKind(res, OpImage)
	if len(imgs) != 1 {
		t.Fatalf("img ops = %d, want 1", len(imgs))
	}

	return decodeNRGBA(t, imgs[0].Image).NRGBAAt(20, 20).A
}

func TestBehaviorClipRuleWireEvenOddMasksStarCenter(t *testing.T) {
	t.Parallel()

	if got := clipRuleWireCenterAlpha(t, "clip-path:"+behaviorClipStarPolygon); got != 255 {
		t.Errorf("default center alpha = %d, want 255 (nonzero initial keeps twice-wound center)", got)
	}

	if got := clipRuleWireCenterAlpha(t,
		"clip-path:"+behaviorClipStarPolygon+";clip-rule:evenodd"); got != 0 {
		t.Errorf("evenodd center alpha = %d, want 0 (twice-wound center cut out)", got)
	}
}

func TestBehaviorClipRuleWireNonzeroKeepsStarCenter(t *testing.T) {
	t.Parallel()

	if got := clipRuleWireCenterAlpha(t,
		"clip-path:"+behaviorClipStarPolygon+";clip-rule:nonzero"); got != 255 {
		t.Errorf("nonzero center alpha = %d, want 255 (twice-wound center stays)", got)
	}
}

func TestBehaviorClipRuleWireInlineFillRuleWins(t *testing.T) {
	t.Parallel()

	// An inline polygon() fill rule beats the standalone property.
	if got := clipRuleWireCenterAlpha(t,
		"clip-path:polygon(evenodd, 50% 10%, 73.5% 82.4%, 12% 37.6%, 88% 37.6%, 26.5% 82.4%);"+
			"clip-rule:nonzero"); got != 0 {
		t.Errorf("inline evenodd center alpha = %d, want 0 (inline rule wins over clip-rule:nonzero)", got)
	}
}

func TestBehaviorClipRuleWireInheritsFromParent(t *testing.T) {
	t.Parallel()

	res := layoutHTMLWithImages(t,
		`<html><body><div style="clip-rule:evenodd"><img src="x.png" style="clip-path:`+
			behaviorClipStarPolygon+`"></div></body></html>`,
		tinyPNG(40, 40), "x.png")

	imgs := opsOfKind(res, OpImage)
	if len(imgs) != 1 {
		t.Fatalf("img ops = %d, want 1", len(imgs))
	}

	if got := decodeNRGBA(t, imgs[0].Image).NRGBAAt(20, 20).A; got != 0 {
		t.Errorf("inherited evenodd center alpha = %d, want 0 (clip-rule inherits)", got)
	}
}

func TestBehaviorClipRuleWireInvalidKeepsDefault(t *testing.T) {
	t.Parallel()

	if got := clipRuleWireCenterAlpha(t,
		"clip-path:"+behaviorClipStarPolygon+";clip-rule:bogus"); got != 255 {
		t.Errorf("invalid clip-rule center alpha = %d, want 255 (bad value keeps nonzero)", got)
	}
}

// clipRuleWireBackgroundCenterAlpha lays out one gradient box clipped by the
// star polygon and returns the decoded center alpha of its background image
// op. The 100pt box rasterizes to 100x100, so pixel (50, 50) is the star
// center.
func clipRuleWireBackgroundCenterAlpha(t *testing.T, clipRule string) uint8 {
	t.Helper()

	sheet, err := css.Parse(`.box {
  width: 100pt;
  height: 100pt;
  background-image: conic-gradient(from 0deg, #ff0000, #0000ff);
  clip-path: ` + behaviorClipStarPolygon + `;
  clip-rule: ` + clipRule + `;
}`)
	if err != nil {
		t.Fatal(err)
	}

	root := mustParse(t, `<html><body><div class="box"></div></body></html>`)

	res, err := Layout(root, Options{
		Width: testViewport, Height: 400, Background: true, Media: "print",
		Sheets: []*css.Stylesheet{sheet},
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	for i := range res.Ops {
		if res.Ops[i].Kind == OpImage && res.Ops[i].IsBackground {
			return decodeNRGBA(t, res.Ops[i].Image).NRGBAAt(50, 50).A
		}
	}

	t.Fatal("no background image op emitted")

	return 0
}

func TestBehaviorClipRuleWireBackgroundMasksStarCenter(t *testing.T) {
	t.Parallel()

	if got := clipRuleWireBackgroundCenterAlpha(t, "nonzero"); got != 255 {
		t.Errorf("nonzero background center alpha = %d, want 255", got)
	}

	if got := clipRuleWireBackgroundCenterAlpha(t, "evenodd"); got != 0 {
		t.Errorf("evenodd background center alpha = %d, want 0 (twice-wound center cut out)", got)
	}
}
