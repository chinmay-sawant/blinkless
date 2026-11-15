package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBehaviorWebkitBackgroundClipClipsImage is -webkit-background-clip on
// the same padded image box as TestBehaviorBackgroundClipClipsImage. The
// prefixed property must clip to the same two image boxes.
func TestBehaviorWebkitBackgroundClipClipsImage(t *testing.T) {
	t.Parallel()

	png := tinyPNG(10, 10)
	res := layoutHTMLWithImages(t, readV012Fixture(t, "webkit-background-clip"), png, "")

	borderBox := boxByID(t, res, "border")
	contentBox := boxByID(t, res, "content")
	borderImg := webkitBackgroundImageInBox(res.Ops, borderBox)
	contentImg := webkitBackgroundImageInBox(res.Ops, contentBox)

	if borderImg == nil || contentImg == nil {
		t.Fatalf("background images = border %v content %v, want one each",
			borderImg != nil, contentImg != nil)
	}

	if !near(borderImg.W, pxToPt(100)) || !near(borderImg.H, pxToPt(60)) {
		t.Errorf("border-box clip size = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want 100px x 60px",
			borderImg.W, borderImg.H, borderImg.W/ptPerCSSPx, borderImg.H/ptPerCSSPx)
	}

	if !near(contentImg.X, contentBox.x+pxToPt(10)) || !near(contentImg.Y, contentBox.y+pxToPt(10)) {
		t.Errorf("content-box clip origin = (%.4fpt, %.4fpt), want 10px inside the border box",
			contentImg.X, contentImg.Y)
	}

	if !near(contentImg.W, pxToPt(80)) || !near(contentImg.H, pxToPt(40)) {
		t.Errorf("content-box clip size = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want 80px x 40px",
			contentImg.W, contentImg.H, contentImg.W/ptPerCSSPx, contentImg.H/ptPerCSSPx)
	}
}

// TestBehaviorWebkitBackgroundOriginInsetsImage is -webkit-background-origin.
// border-box starts the 10px image on the border-box origin. padding-box
// starts it 10px inside that origin.
func TestBehaviorWebkitBackgroundOriginInsetsImage(t *testing.T) {
	t.Parallel()

	paddingSrc := readV012Fixture(t, "webkit-background-origin")
	borderSrc := strings.ReplaceAll(
		paddingSrc,
		"-webkit-background-origin:padding-box",
		"-webkit-background-origin:border-box",
	)
	png := tinyPNG(10, 10)
	borderRes := layoutHTMLWithImages(t, borderSrc, png, "")
	paddingRes := layoutHTMLWithImages(t, paddingSrc, png, "")

	borderBox := boxByID(t, borderRes, "item")
	borderImg := behaviorBgFirstImage(borderRes.Ops)
	paddingImg := behaviorBgFirstImage(paddingRes.Ops)

	if borderImg == nil || paddingImg == nil {
		t.Fatalf("background images = border %v padding %v, want one each",
			borderImg != nil, paddingImg != nil)
	}

	if !near(borderImg.X, borderBox.x) || !near(borderImg.Y, borderBox.y) {
		t.Errorf("border-box origin at (%.4fpt, %.4fpt), want box origin (%.4fpt, %.4fpt)",
			borderImg.X, borderImg.Y, borderBox.x, borderBox.y)
	}

	if !near(paddingImg.X, borderBox.x+pxToPt(10)) || !near(paddingImg.Y, borderBox.y+pxToPt(10)) {
		t.Errorf("padding-box origin at (%.4fpt, %.4fpt), want 10px inside (%.4fpt, %.4fpt)",
			paddingImg.X, paddingImg.Y, borderBox.x+pxToPt(10), borderBox.y+pxToPt(10))
	}
}

// TestBehaviorWebkitBackgroundSizeScalesImage is -webkit-background-size.
// The prefixed declaration paints the same 40px by 20px image as
// background-size, and the encoded payload stays 10px by 10px.
func TestBehaviorWebkitBackgroundSizeScalesImage(t *testing.T) {
	t.Parallel()

	png := tinyPNG(10, 10)
	res := layoutHTMLWithImages(t, readV012Fixture(t, "webkit-background-size"), png, "")
	img := behaviorBgFirstImage(res.Ops)

	if img == nil {
		t.Fatal("no painted background image op")
	}

	if !near(img.W, pxToPt(40)) || !near(img.H, pxToPt(20)) {
		t.Errorf("painted size = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want 40px x 20px",
			img.W, img.H, img.W/ptPerCSSPx, img.H/ptPerCSSPx)
	}

	if img.ImgW != 10 || img.ImgH != 10 {
		t.Errorf("intrinsic payload = %dx%d, want 10x10", img.ImgW, img.ImgH)
	}
}

func readV012Fixture(t *testing.T, slug string) string {
	t.Helper()

	rel := filepath.Join("testdata", "css", "v012", slug, "page.html")
	candidates := []string{
		rel,
		filepath.Join("..", rel),
		filepath.Join("..", "..", rel),
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		return string(data)
	}

	t.Fatalf("fixture %s not found in %v", rel, candidates)

	return ""
}

func webkitBackgroundImageInBox(ops []Op, item *box) *Op {
	for _, paintOp := range ops {
		if paintOp.Kind != OpImage || !paintOp.IsBackground || paintOp.W <= 0 || paintOp.H <= 0 {
			continue
		}

		centerX := paintOp.X + paintOp.W/2
		centerY := paintOp.Y + paintOp.H/2
		insideX := centerX >= item.x && centerX <= item.x+item.w
		insideY := centerY >= item.y && centerY <= item.y+item.height

		if insideX && insideY {
			candidate := paintOp

			return &candidate
		}
	}

	return nil
}
