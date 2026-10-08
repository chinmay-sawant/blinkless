package layout

import (
	"math"
	"testing"

)

func TestOptionsValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		opts Options
	}{
		{name: "zero width", opts: Options{Width: 0, Height: 800}},
		{name: "negative width", opts: Options{Width: -10, Height: 800}},
		{name: "negative height", opts: Options{Width: 500, Height: -10}},
		{name: "nan width", opts: Options{Width: math.NaN(), Height: 800}},
		{name: "negative zoom", opts: Options{Width: 500, Height: 800, Zoom: -1}},
		{name: "nan zoom", opts: Options{Width: 500, Height: 800, Zoom: math.NaN()}},
		{name: "unknown media", opts: Options{Width: 500, Height: 800, Media: "tv"}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if err := testCase.opts.validate(); err == nil {
				t.Fatal("validate accepted invalid options")
			}
		})
	}
}

func TestOptionsValidateAcceptsDefaults(t *testing.T) {
	t.Parallel()

	opts := []Options{
		{Width: 500, Height: 800},
		{Width: 500, Height: 0, Zoom: 0.95, Media: "print"},
		{Width: 400, Height: 600, Media: "screen"},
	}

	for _, option := range opts {
		if err := option.validate(); err != nil {
			t.Fatalf("validate(%+v) = %v, want nil", option, err)
		}
	}
}

func TestLayoutContextRejectsNegativeZoom(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><p>x</p></body></html>`)

	_, err := Layout(root, Options{
		Width: 500, Height: 800, Zoom: -1,
	})
	if err == nil {
		t.Fatal("Layout accepted a negative zoom")
	}
}

