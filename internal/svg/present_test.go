package svg

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

func TestEmulateShapeLength(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		shape string
		want  float64
	}{
		{"path", `<path d="M0 0 H10 V5"/>`, 15},
		{"line", `<line x1="0" y1="0" x2="3" y2="4"/>`, 5},
		{"circle", `<circle r="10"/>`, 2 * math.Pi * 10},
		{"rect", `<rect width="10" height="4"/>`, 28},
		{"polyline", `<polyline points="0,0 3,4 3,10"/>`, 11},
		{"polygon", `<polygon points="0,0 4,0 4,3"/>`, 12},
	}

	for _, tc := range cases {
		data := []byte(strings.Replace(tc.shape, "/>", ` pathLength="1" stroke-dasharray="2"/>`, 1))
		out := string(emulatePresentation(data))
		want := strconv.FormatFloat(2*tc.want, 'g', -1, 64)

		if !strings.Contains(out, `stroke-dasharray="`+want+`"`) || strings.Contains(strings.ToLower(out), "pathlength") {
			t.Errorf("%s: got %s, want dasharray %s and no pathLength", tc.name, out, want)
		}
	}
}

func TestEmulatePresentationLeavesUnrelatedTags(t *testing.T) {
	t.Parallel()

	in := []byte(`<svg><rect width="1" height="1" paint-order="stroke" fill="none" stroke="#000"/></svg>`)
	if out := emulatePresentation(in); string(out) != string(in) {
		t.Errorf("a shape with no fill was rewritten: %s", out)
	}

	in = []byte(`<svg><g paint-order="stroke"><rect width="1" height="1"/></g></svg>`)
	if out := emulatePresentation(in); string(out) != string(in) {
		t.Errorf("a group paint-order was rewritten: %s", out)
	}
}
