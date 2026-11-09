package html

import (
	"fmt"
	"sync"
	"testing"
)

func TestFindIndexParity(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte(`<div id="dup">aaa</div><div id="dup">bbb</div>` +
		`<div id="box">Hello <span id="inner">World</span></div>` +
		`<template><div id="tpl">hidden</div></template>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	first, ok := doc.Find("dup")
	if !ok || first.Text != "aaa" || first.Tag != "div" {
		t.Fatalf("Find(dup) = %+v, %v, want first element with text aaa", first, ok)
	}

	box, ok := doc.Find("box")
	if !ok || box.Text != "Hello World" {
		t.Fatalf("Find(box) = %+v, %v, want text %q", box, ok, "Hello World")
	}

	if _, ok := doc.Find("tpl"); ok {
		t.Fatal("Find(tpl) found template contents, want miss")
	}

	if _, ok := doc.Find(""); ok {
		t.Fatal("Find(empty) hit, want miss")
	}

	if _, ok := doc.Find("missing"); ok {
		t.Fatal("Find(missing) hit, want miss")
	}

	var nilDoc *Document
	if _, ok := nilDoc.Find("dup"); ok {
		t.Fatal("nil Find hit, want miss")
	}

	if !doc.SetAttribute("dup", "id", "dup2") {
		t.Fatal("SetAttribute rename reported no element")
	}

	moved, ok := doc.Find("dup")
	if !ok || moved.Text != "bbb" {
		t.Fatalf("Find(dup) after rename = %+v, %v, want second element text bbb", moved, ok)
	}

	renamed, ok := doc.Find("dup2")
	if !ok || renamed.Text != "aaa" {
		t.Fatalf("Find(dup2) = %+v, %v, want renamed text aaa", renamed, ok)
	}

	if !doc.SetText("box", "Bye") {
		t.Fatal("SetText reported no element")
	}

	if _, ok := doc.Find("inner"); ok {
		t.Fatal("Find(inner) hit after drop, want miss")
	}

	after, ok := doc.Find("box")
	if !ok || after.Text != "Bye" {
		t.Fatalf("Find(box) after drop = %+v, %v, want text Bye", after, ok)
	}
}

func TestFindIndexConcurrentWithEdits(t *testing.T) {
	t.Parallel()

	doc := buildPatchDoc(t, 200)

	const readers = 4
	const rounds = 200

	var wg sync.WaitGroup

	for r := range readers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for i := range rounds {
				id := fmt.Sprintf("e%04d", (r*50+i)%200)
				if _, ok := doc.Find(id); !ok {
					t.Error("Find lost an element under concurrent edits")
					return
				}
			}
		}()
	}

	for i := range rounds {
		id := fmt.Sprintf("e%04d", 100+(i%50))
		if !doc.SetAttribute(id, "data-k", "v") {
			t.Fatalf("SetAttribute(%q) reported no element", id)
		}

		if i%25 == 0 && !doc.SetText("e0199", "scratch") {
			t.Fatal("SetText reported no element")
		}
	}

	wg.Wait()
}

func TestFindIndexRenameDrop(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte(`<div id="box" class="c">Hi</div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if !doc.SetAttribute("box", "id", "box2") {
		t.Fatal("SetAttribute reported no element")
	}

	if _, ok := doc.Find("box"); ok {
		t.Fatal("Find(box) hit after rename, want miss")
	}

	renamed, ok := doc.Find("box2")
	if !ok || renamed.Tag != "div" || renamed.Text != "Hi" {
		t.Fatalf("Find(box2) = %+v, %v, want div Hi", renamed, ok)
	}

	outer, err := Parse([]byte(`<div id="outer"><span id="inner">x</span></div><div id="sib">y</div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if !outer.SetText("outer", "hi") {
		t.Fatal("SetText reported no element")
	}

	if _, ok := outer.Find("inner"); ok {
		t.Fatal("Find(inner) hit after drop, want miss")
	}

	survivor, ok := outer.Find("outer")
	if !ok || survivor.Text != "hi" {
		t.Fatalf("Find(outer) = %+v, %v, want text hi", survivor, ok)
	}

	sib, ok := outer.Find("sib")
	if !ok || sib.Text != "y" {
		t.Fatalf("Find(sib) = %+v, %v, want text y", sib, ok)
	}
}
