package layout

import (
	"strings"

	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
)

const (
	synthSmallCapsScale      = 0.8
	synthSubSuperScale       = 0.65
	synthSubShiftRatio       = 0.2
	synthSuperShiftRatio     = 0.35
	synthObliqueSkew         = 0.22
	fontWidthReference       = 100
	fontVariantCapsSmall     = "small-caps"
	fontVariantCapsAllSmall  = "all-small-caps"
	fontVariantCapsPetite    = "petite-caps"
	fontVariantCapsAllPetite = "all-petite-caps"
)

func fontWidthScale(sty *ResolvedStyle) float64 {
	if sty == nil || sty.font().FontWidth <= 0 || sty.font().FontWidth == 100 {
		return 1
	}

	return sty.font().FontWidth / fontWidthReference
}

func synthesizeSmallCapsText(sty *ResolvedStyle, text string) (string, float64) {
	if sty == nil || text == "" || !sty.font().FontSynthesisSmallCaps {
		return text, 1
	}

	switch sty.font().FontVariantCaps {
	case fontVariantCapsSmall, fontVariantCapsAllSmall, fontVariantCapsPetite, fontVariantCapsAllPetite:
		return strings.ToUpper(text), synthSmallCapsScale
	default:
		return text, 1
	}
}

func synthesizePositionAdjust(sty *ResolvedStyle, size float64) (float64, float64) {
	if sty == nil || !sty.font().FontSynthesisPosition {
		return 1, 0
	}

	switch sty.font().FontVariantPosition {
	case verticalAlignSub:
		return synthSubSuperScale, size * synthSubShiftRatio
	case verticalAlignSuper:
		return synthSubSuperScale, -size * synthSuperShiftRatio
	default:
		return 1, 0
	}
}

func needsFakeOblique(sty *ResolvedStyle, face *pdf.Font) bool {
	if sty == nil || !sty.FontItalic || !sty.font().FontSynthesisStyle {
		return false
	}

	if face == nil {
		return true
	}

	return !face.Italic()
}
