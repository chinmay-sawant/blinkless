package html

import (
	"bytes"
	"testing"
)

func TestParseDefaultCapAndExplicitLimit(t *testing.T) {
	t.Parallel()

	if _, err := ParseWithLimit([]byte("<p>hello</p>"), 5); err == nil {
		t.Fatal("ParseWithLimit accepted a source over its limit")
	}

	if _, err := ParseWithLimit([]byte("<p>hello</p>"), 0); err != nil {
		t.Fatalf("default limit rejected a tiny source: %v", err)
	}

	big := bytes.Repeat([]byte("x"), MaxParseBytes+1)
	if _, err := Parse(big); err == nil {
		t.Fatal("Parse accepted a source over MaxParseBytes")
	}
}

func TestSetAttributeKeepsForeignCase(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte(`<svg id="s"></svg><div id="d"></div>`))
	if err != nil {
		t.Fatal(err)
	}

	if !doc.SetAttribute("s", "viewBox", "0 0 10 10") || !doc.SetAttribute("d", "DataX", "1") {
		t.Fatal("SetAttribute failed")
	}

	svg := doc.elementByID("s")
	if v, ok := svg.AttrValue("viewBox"); !ok || v != "0 0 10 10" {
		t.Fatalf("svg viewBox = %q, %v", v, ok)
	}

	div := doc.elementByID("d")
	if v, ok := div.AttrValue("datax"); !ok || v != "1" {
		t.Fatalf("html attribute should stay lowercased: %q, %v", v, ok)
	}
}
