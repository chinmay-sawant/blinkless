package layout

import (
	"testing"
)

// This file holds the string-set behavior tests: every observable below is
// asserted through a USED value (the decoded concatenated string the table
// stores), never through a stored declaration string. string-set feeds GCPM
// running headers and footers, but this engine has no page margin pipeline,
// so layout and paint cannot observe it: the no-paint test pins that with an
// identical-geometry comparison and the rest prove set-then-read table
// semantics. Reference: Chrome 143.0.7499.40 has no string-set support,
// stated explicitly, so parsing follows the css-content-3 draft and the
// catalog upstream syntax none | [ <custom-ident> <string>+ ]#
// (testdata/css/catalog/properties.json). Full margin-box rendering is out of
// scope; the seam is CollectNamedStrings, which the future margin pipeline
// calls once per layout to resolve string() lookups.

// TestBehaviorStringSetFirstAssignmentWins proves set-then-read on used
// values: the first element assigning chapter wins, and adjacent strings
// concatenate ('Guide ' + 'One' = 'Guide One'), so the test reads an
// evaluated value, not the declaration text.
func TestBehaviorStringSetFirstAssignmentWins(t *testing.T) {
	t.Parallel()

	src := `<html><body>` +
		`<h2 style="string-set: chapter 'Guide ' 'One'">Guide One</h2>` +
		`<h2 style="string-set: chapter 'Beta'">Beta</h2>` +
		`</body></html>`

	root := mustParse(t, src)

	if _, err := Layout(root, Options{Width: testViewport, Height: 800, Background: true}); err != nil {
		t.Fatalf("Layout: %v", err)
	}

	got, ok := LookupNamedString(root, "chapter")
	if !ok {
		t.Fatalf("Lookup(chapter) missed, want used value %q", "Guide One")
	}

	if got != "Guide One" {
		t.Errorf("Lookup(chapter) = %q, want used value %q (concatenated strings)", got, "Guide One")
	}
}

// TestBehaviorStringSetMultipleNames proves one declaration can assign two
// names at once via the comma list.
func TestBehaviorStringSetMultipleNames(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>`+
		`<div style="string-set: header 'Top', footer 'Bottom'">body</div>`+
		`</body></html>`)

	table := CollectNamedStrings(root)

	for name, want := range map[string]string{"header": "Top", "footer": "Bottom"} {
		got, ok := table.Lookup(name)
		if !ok {
			t.Errorf("Lookup(%s) missed, want used value %q", name, want)

			continue
		}

		if got != want {
			t.Errorf("Lookup(%s) = %q, want used value %q", name, got, want)
		}
	}
}

// TestBehaviorStringSetNoneAndInvalidIgnored proves none, a bare identifier
// with no string, a function value, and a non-identifier are all validly
// ignored (no table entry), while a later valid assignment still lands.
func TestBehaviorStringSetNoneAndInvalidIgnored(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>`+
		`<p style="string-set: none">a</p>`+
		`<p style="string-set: chapter">b</p>`+
		`<p style="string-set: chapter counter(page)">c</p>`+
		`<p style="string-set: 12px">d</p>`+
		`<p style="string-set: chapter 'Kept'">e</p>`+
		`</body></html>`)

	table := CollectNamedStrings(root)

	if table.Len() != 1 {
		t.Errorf("table length = %d, want 1 (only the valid assignment lands)", table.Len())
	}

	got, ok := table.Lookup("chapter")
	if !ok {
		t.Fatalf("Lookup(chapter) missed, want used value %q", "Kept")
	}

	if got != "Kept" {
		t.Errorf("Lookup(chapter) = %q, want used value %q", got, "Kept")
	}

	if _, ok := table.Lookup("missing"); ok {
		t.Errorf("Lookup(missing) hit, want miss")
	}
}

// TestBehaviorStringSetEscapesDecode proves the used value decodes escapes:
// 'a\'b' stores a'b, not the raw five characters.
func TestBehaviorStringSetEscapesDecode(t *testing.T) {
	t.Parallel()

	assignments, ok := ParseStringSet(`chapter 'a\'b'`)
	if !ok {
		t.Fatalf("ParseStringSet missed, want one assignment")
	}

	if len(assignments) != 1 {
		t.Fatalf("assignment count = %d, want 1", len(assignments))
	}

	if assignments[0].Value != "a'b" {
		t.Errorf("used value = %q, want %q (decoded escape)", assignments[0].Value, "a'b")
	}
}

// TestBehaviorStringSetNoPaintEffect pins the current boundary: string-set
// changes nothing in the drawing list because no margin pipeline consumes the
// table yet. Unobservable in paint: reported as a no-op, never failing.
func TestBehaviorStringSetNoPaintEffect(t *testing.T) {
	t.Parallel()

	plain := `<html style="margin:0"><body style="margin:0"><h2>Guide One</h2></body></html>`
	styled := `<html style="margin:0"><body style="margin:0">` +
		`<h2 style="string-set: chapter 'Guide One'">Guide One</h2></body></html>`

	behaviorColumnsAssertSameOps(t, layoutHTML(t, plain), layoutHTML(t, styled))
}
