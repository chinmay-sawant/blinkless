package layout

import (
	"fmt"
	"strings"
	"testing"
)

func TestFlexOrderFastPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		orders []string
		want   []string
	}{
		{
			name:   "default",
			orders: []string{"", "", "", ""},
			want:   []string{"a", "b", "c", "d"},
		},
		{
			name:   "equal",
			orders: []string{"2", "2", "2", "2"},
			want:   []string{"a", "b", "c", "d"},
		},
		{
			name:   "nondecreasing_negative_positive",
			orders: []string{"-2", "-2", "0", "3"},
			want:   []string{"a", "b", "c", "d"},
		},
		{
			name:   "mixed_negative_positive_stable_ties",
			orders: []string{"3", "-2", "3", "0", "-2"},
			want:   []string{"b", "e", "d", "a", "c"},
		},
	}

	for _, direction := range []string{"row", "column"} {
		for _, tc := range cases {
			t.Run(direction+"/"+tc.name, func(t *testing.T) {
				t.Parallel()

				var source strings.Builder
				source.WriteString(`<html style="margin:0"><body style="margin:0">`)
				source.WriteString(`<div id="container" style="display:flex;width:200px;height:200px;flex-direction:`)
				source.WriteString(direction + `">`)
				for i, order := range tc.orders {
					source.WriteString(fmt.Sprintf(`<div id="%c" style="width:20px;height:20px;flex:none;`, 'a'+i))
					if order != "" {
						source.WriteString("order:" + order)
					}
					source.WriteString(`"></div>`)
				}
				source.WriteString(`</div></body></html>`)

				res := layoutHTML(t, source.String())
				for i, id := range tc.want {
					var dx, dy float64
					if direction == "row" {
						dx = float64(i) * 20
					} else {
						dy = float64(i) * 20
					}
					behaviorFlexGridCheckOffset(t, res, id, "container", dx, dy)
				}
			})
		}
	}
}
