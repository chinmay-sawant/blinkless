package layout

import (
	"context"
	"testing"
)

// subtreeFixtureCSS sets a root font size (so rem has a base other than the
// default), a colour and a custom property on the host, a rem font size on the
// branch, and an em font size on the leaf.
const subtreeFixtureCSS = `
	html { font-size: 32px; }
	#host { color: rgb(17, 34, 51); --brand: teal; }
	#branch { font-size: 2rem; }
	.lead { font-size: 1.5em; }
`

// subtreeFixtureHTML holds the branch to re-style, a sibling that shares the
// branch's parent, and one element further outside the branch's parent.
const subtreeFixtureHTML = `<html><body>` +
	`<div id="host">` +
	`<nav id="sibling">menu</nav>` +
	`<section id="branch"><p class="lead">text</p></section>` +
	`</div>` +
	`<aside id="outside">other</aside>` +
	`</body></html>`

// subtreeStyleContext builds the style context a subtree pass runs in. It
// mirrors resolveStylesWithContext so the cascade sees the same sheets, media,
// viewport, and registered properties.
func subtreeStyleContext(ctx context.Context, opts Options) *styleContext {
	return &styleContext{
		ctx:                ctx,
		sheets:             opts.Sheets,
		properties:         registeredProperties(opts.Sheets),
		media:              opts.Media,
		viewportW:          opts.Width,
		viewportH:          opts.Height,
		printLinkUnderline: opts.PrintLinkUnderline,
		state:              opts.State,
	}
}

// memoEntryCount counts the records the memo holds.
func memoEntryCount(memo *styleResolutionMemo) int {
	total := 0

	for _, entries := range memo.entries {
		total += len(entries)
	}

	return total
}

// assertSubtreeStyle compares one subtree style with the full-document style
// of the same element.
func assertSubtreeStyle(t *testing.T, got *ResolvedStyle, want *ResolvedStyle, name string) {
	t.Helper()

	if got == nil {
		t.Fatalf("subtree pass stored no style for %s", name)
	}

	if got.FontSize != want.FontSize {
		t.Errorf("%s font size = %v, want %v", name, got.FontSize, want.FontSize)
	}

	if got.Color != want.Color {
		t.Errorf("%s color = %v, want %v", name, got.Color, want.Color)
	}

	if got.CustomProps["--brand"] != want.CustomProps["--brand"] {
		t.Errorf(
			"%s --brand = %q, want %q",
			name, got.CustomProps["--brand"], want.CustomProps["--brand"],
		)
	}
}

// TestResolveStylesForSubtreeMatchesFullDocument proves a subtree pass
// inherits from the given parent style exactly like the full-document walk:
// the used font size, the colour, and a custom property reach the same values.
//
// The branch sets its font size in rem and the leaf in em, so the values also
// prove the pass used the root's used font size for rem lengths.
func TestResolveStylesForSubtreeMatchesFullDocument(t *testing.T) {
	t.Parallel()

	root := mustParse(t, subtreeFixtureHTML)
	opts := Options{
		Sheets: sheetSlice(sheet(t, subtreeFixtureCSS)),
		Media:  "print",
		Width:  testViewport,
		Height: 800,
	}

	full, err := resolveStylesWithContext(t.Context(), root, opts, nil)
	if err != nil {
		t.Fatalf("resolveStylesWithContext: %v", err)
	}

	host := findElementByID(t, root, "host")
	branch := findElementByID(t, root, "branch")
	lead := findElementByName(root, "p")
	sibling := findElementByID(t, root, "sibling")
	outside := findElementByID(t, root, "outside")
	body := findElementByName(root, "body")
	rootEl := findElementByName(root, "html")

	styleCtx := subtreeStyleContext(t.Context(), opts)
	remBase := full[rootEl].FontSize

	if err := resolveStylesForSubtree(t.Context(), styleCtx, branch, full[host], remBase); err != nil {
		t.Fatalf("resolveStylesForSubtree: %v", err)
	}

	got := subtreeStyles(styleCtx, branch, full[host])

	assertSubtreeStyle(t, got[branch], full[branch], "section")
	assertSubtreeStyle(t, got[lead], full[lead], "p")

	// 32px root font size is 24pt, so 2rem is 48pt and 1.5em of that is 72pt.
	if got[branch].FontSize != 48 {
		t.Errorf("branch font size = %v, want 48", got[branch].FontSize)
	}

	if got[lead].FontSize != 72 {
		t.Errorf("lead font size = %v, want 72", got[lead].FontSize)
	}

	if got[lead].Color != [3]float64{17.0 / 255, 34.0 / 255, 51.0 / 255} {
		t.Errorf("lead color = %v, want rgb(17, 34, 51)", got[lead].Color)
	}

	if got[lead].CustomProps["--brand"] != "teal" {
		t.Errorf("lead --brand = %q, want %q", got[lead].CustomProps["--brand"], "teal")
	}

	// The walk stops at the subtree: the sibling that shares the branch's
	// parent style, and the element outside that parent, were never cascaded,
	// so the memo holds no record for either.
	if memoStyleFor(styleCtx, sibling, full[host]) != nil {
		t.Error("subtree pass resolved the sibling of the subtree root")
	}

	if memoStyleFor(styleCtx, outside, full[body]) != nil {
		t.Error("subtree pass resolved an element outside the subtree")
	}
}

// TestResolveStylesForSubtreeKeepsMemoWorking proves the subtree pass fills
// the memo and reuses it: a second pass over the same subtree adds no record
// and hands back the same styles.
func TestResolveStylesForSubtreeKeepsMemoWorking(t *testing.T) {
	t.Parallel()

	root := mustParse(t, subtreeFixtureHTML)
	opts := Options{
		Sheets: sheetSlice(sheet(t, subtreeFixtureCSS)),
		Media:  "print",
		Width:  testViewport,
		Height: 800,
	}

	full, err := resolveStylesWithContext(t.Context(), root, opts, nil)
	if err != nil {
		t.Fatalf("resolveStylesWithContext: %v", err)
	}

	host := findElementByID(t, root, "host")
	branch := findElementByID(t, root, "branch")
	rootEl := findElementByName(root, "html")

	styleCtx := subtreeStyleContext(t.Context(), opts)
	remBase := full[rootEl].FontSize

	if err := resolveStylesForSubtree(t.Context(), styleCtx, branch, full[host], remBase); err != nil {
		t.Fatalf("resolveStylesForSubtree: %v", err)
	}

	first := subtreeStyles(styleCtx, branch, full[host])

	if memoEntryCount(&styleCtx.memo) == 0 {
		t.Fatal("subtree pass stored no memo record")
	}

	entries := memoEntryCount(&styleCtx.memo)

	if err := resolveStylesForSubtree(t.Context(), styleCtx, branch, full[host], remBase); err != nil {
		t.Fatalf("resolveStylesForSubtree (second pass): %v", err)
	}

	if got := memoEntryCount(&styleCtx.memo); got != entries {
		t.Errorf("memo records = %d after the second pass, want %d", got, entries)
	}

	second := subtreeStyles(styleCtx, branch, full[host])

	for node, style := range first {
		if second[node] != style {
			t.Errorf("%s style was not reused from the memo", node.Name)
		}
	}
}
