# Fixture test template

Use this reference after dumping the exact fixture's drawing list. Replace
every placeholder with values from that dump. Do not reuse a coordinate or
count from another test.

```go
package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

func TestDisplayFixtureNNSlug(t *testing.T) {
	t.Parallel()

	// displayOf parses HTML and applies CSS; see layout/displaylist_test.go.
	// Read the fixture with os.ReadFile when it is large.
	display := displayOf(t, `<fixture markup or file contents>`)

	if display.Width != measuredWidth || display.Height != measuredHeight {
		t.Errorf("canvas = %dx%d, want %dx%d",
			display.Width, display.Height, measuredWidth, measuredHeight)
	}

	// Values below came from
	// go run ./bindings/wasm -fixture testdata/golden/fixture-NN-slug.html -out /tmp/dump.json
	op := findTextOp(t, display, "Unique title")
	assertOpGeometry(t, op, measuredX, measuredY, measuredW, measuredH)
	assertOpColor(t, op, measuredR, measuredG, measuredB)

	if got := countKind(display, layout.DisplayOpFillRect); got != measuredFills {
		t.Errorf("fill ops = %d, want %d", got, measuredFills)
	}

	if got := countKind(display, layout.DisplayOpLine); got != measuredLines {
		t.Errorf("line ops = %d, want %d", got, measuredLines)
	}

	if got := countKind(display, layout.DisplayOpImage); got != measuredImages {
		t.Errorf("image ops = %d, want %d", got, measuredImages)
	}
}
```

The helper names above are illustrative. Use the existing helpers in the
package you add the test to (`displayOf`, `styledOf` in
`layout/displaylist_test.go`), or write the smallest local ones. If you need
the rare payload of an op, use its nil-safe accessor (`LinkURI`,
`ImageBytes`, `Opacity`, `Transform`, ...) instead of the promoted fields.

## Choosing assertions

Use only assertions that match the HTML and the measured drawing list.

- Text: choose unique strings. Include origin, size, and fill color.
- Rule or border: choose a measured line op. Include endpoints, width, and
  stroke color.
- Fill: choose a measured fillRect. Include the box and fill color.
- Image: choose the measured placement box. Confirm the HTML really authors
  the image.
- Canvas: pin the measured width and height.
- Counts: pin them when the fixture is simple enough that an unexpected op
  should be a failure. Counts are supporting evidence, not a replacement for
  authored anchors.
- Paint order: assert the relative order of anchors when layering matters.

## Measurement record

Before writing the test, keep a short working record in the response or local
notes allowed by the user's scope:

```text
HTML: testdata/golden/fixture-NN-slug.html
Canvas: WxH
Ops: text=N lines=N fills=N images=N
Anchors:
  text | "..." | x=... y=... w=... h=... size=... color=#......
  line | x1=... y1=... x2=... y2=... width=... color=#......
```

The working record is evidence for the test values. It is not a substitute for
the test or for the engine dump.
