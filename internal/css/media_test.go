package css_test

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

func TestMediaMatchesTypes(t *testing.T) {
	t.Parallel()

	const width, height = 538.0, 785.0 // ~A4 content

	cases := []struct {
		q, media string
		want     bool
	}{
		{"", "print", true},
		{"all", "print", true},
		{"print", "print", true},
		{"screen", "print", false},
		{"screen", "screen", true},
		{"only print", "print", true},
		{"not print", "print", false},
		{"not screen", "print", true},
		{"print, screen", "print", true},
		{"screen, print", "screen", true},
	}
	for _, c := range cases {
		if got := css.MediaMatches(c.q, c.media, width, height); got != c.want {
			t.Errorf("MediaMatches(%q,%q)=%v want %v", c.q, c.media, got, c.want)
		}
	}
}

func TestMediaMatchesSizeFeatures(t *testing.T) {
	t.Parallel()
	// 640px = 480pt; 500px = 375pt
	wide := 538.0
	narrow := 300.0
	height := 785.0

	if !css.MediaMatches("(min-width: 500px)", "print", wide, height) {
		t.Error("wide viewport should match min-width:500px")
	}

	if css.MediaMatches("(min-width: 500px)", "print", narrow, height) {
		t.Error("narrow viewport should not match min-width:500px")
	}

	if !css.MediaMatches("(min-width:640px)", "print", wide, height) {
		t.Error("538pt should match min-width:640px (480pt)")
	}

	if css.MediaMatches("screen and (min-width: 500px)", "print", wide, height) {
		t.Error("screen feature query must not match print media")
	}

	if !css.MediaMatches("print and (min-width: 400px)", "print", wide, height) {
		t.Error("print + matching min-width should apply")
	}

	if css.MediaMatches("print and (min-width: 800px)", "print", wide, height) {
		t.Error("print + failing min-width should not apply")
	}

	if css.MediaMatches("(prefers-color-scheme: dark)", "print", wide, height) {
		t.Error("unknown features must evaluate false")
	}

	if !css.MediaMatches("(orientation: portrait)", "print", wide, height) {
		t.Error("portrait A4 should match")
	}

	if css.MediaMatches("(orientation: landscape)", "print", wide, height) {
		t.Error("portrait A4 should not match landscape")
	}
}

func TestMediaMatchesEmptyMediaType(t *testing.T) {
	t.Parallel()
	// Legacy Options.Media == "": apply everything.
	if !css.MediaMatches("screen", "", 100, 100) {
		t.Error("empty mediaType should match all queries")
	}
}

func TestMediaMatchesHeightFeatureBoundaries(t *testing.T) {
	t.Parallel()

	// Each viewport is below, at, or above 500 CSS px. Width stays above
	// that threshold so evaluating the wrong axis cannot pass these cases.
	heights := []float64{374.25, 375, 375.75}
	cases := []struct {
		query string
		want  [3]bool
	}{
		{query: "(min-height: 500px)", want: [3]bool{false, true, true}},
		{query: "(max-height: 500px)", want: [3]bool{true, true, false}},
		{query: "(height: 500px)", want: [3]bool{false, true, false}},
		{query: "(MIN-HEIGHT: 31.25em)", want: [3]bool{false, true, true}},
		{query: "(max-height: 31.25rem)", want: [3]bool{true, true, false}},
		{query: "(height: 375pt)", want: [3]bool{false, true, false}},
		{query: "(min-block-size: 500px)", want: [3]bool{false, true, true}},
		{query: "(max-block-size: 500px)", want: [3]bool{true, true, false}},
		{query: "(block-size: 500px)", want: [3]bool{false, true, false}},
		{query: "(height > 500px)", want: [3]bool{false, false, true}},
		{query: "(height >= 500px)", want: [3]bool{false, true, true}},
		{query: "(height < 500px)", want: [3]bool{true, false, false}},
		{query: "(height <= 500px)", want: [3]bool{true, true, false}},
		{query: "(height = 500px)", want: [3]bool{false, true, false}},
		{query: "(500px < height)", want: [3]bool{false, false, true}},
		{query: "(500px <= height)", want: [3]bool{false, true, true}},
		{query: "(500px > height)", want: [3]bool{true, false, false}},
		{query: "(500px >= height)", want: [3]bool{true, true, false}},
		{query: "(500px = height)", want: [3]bool{false, true, false}},
		{query: "(BLOCK-SIZE >= 31.25em)", want: [3]bool{false, true, true}},
		{query: "(31.25em >= block-size)", want: [3]bool{true, true, false}},
		{query: "screen and (min-width: 700px) and (min-height: 500px)", want: [3]bool{false, true, true}},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			t.Parallel()
			for index, height := range heights {
				got := css.MediaMatches(
					tc.query,
					"screen",
					600,
					height,
				)
				if got != tc.want[index] {
					t.Errorf(
						"height %gpt: got %v, want %v",
						height,
						got,
						tc.want[index],
					)
				}
			}
		})
	}
}

func TestMediaHeightRejectsUnsupportedFeaturesAndRanges(t *testing.T) {
	t.Parallel()

	for _, query := range []string{
		"(depth: 500px)",
		"(min-depth: 500px)",
		"(depth > 500px)",
		"(500px < depth)",
		"(height: bananas)",
		"(height: 50%)",
		"(height > 10vh)",
		"(400px < height < 600px)",
	} {
		if css.MediaMatches(
			query,
			"screen",
			600,
			375,
		) {
			t.Errorf("unsupported media feature matched: %q", query)
		}
	}
}
