package layout

import (
	"testing"
)

func TestRelayoutSubtreesAbsTranslate(t *testing.T) {
	t.Parallel()

	const fixtureCSS = `
html,body{margin:0;padding:0}
#shell{position:relative;width:800px;height:600px;overflow:hidden}
#layer{position:fixed;left:40px;top:100px;width:0;height:0;z-index:25}
.card{display:flex;flex-direction:column;box-sizing:border-box;overflow:hidden;background:#ffffff;border:1px solid #d4d4d4}
.head{display:flex;height:24px;padding:0 8px;background:#f3f2f1}
.body{flex:1;padding:8px}
`
	const fixture = `<!DOCTYPE html><html><body><div id="shell">
<div id="layer">
<div id="c1" class="card" style="position:absolute;left:100px;top:20px;width:200px;height:150px">
  <div id="h1" class="head"><span>Chart 1</span></div>
  <div class="body"><div>Bar A</div><div>Bar B</div></div>
</div>
<div id="c2" class="card" style="position:absolute;left:350px;top:20px;width:200px;height:150px">
  <div id="h2" class="head"><span>Chart 2</span></div>
  <div class="body"><div>Bar C</div></div>
</div>
</div>
</div></body></html>`

	root := mustParse(t, fixture)
	opts := relayoutOptionsForCSS(t, fixtureCSS)
	opts.Width = 800
	opts.Height = 600

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	relayoutNodeByID(t, root, "c1").Attrs["style"] = "position:absolute;left:140px;top:50px;width:200px;height:150px"

	clone := CloneResultForBatch(prev)
	owners := OperationOwners(clone)
	order := PaintOrder(clone.Ops)

	out, err := RelayoutSubtrees(t.Context(), clone, []string{"c1"}, opts, owners, order)
	if err != nil {
		t.Fatalf("RelayoutSubtrees: %v", err)
	}

	if out.Restyled != 1 {
		t.Fatalf("Restyled = %d, want 1 (abs translate fast path did not trigger)", out.Restyled)
	}

	full, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout (full): %v", err)
	}

	for _, id := range []string{"c1", "h1", "c2", "h2"} {
		got := relayoutBoxByID(t, clone, id)
		want := relayoutBoxByID(t, full, id)
		if !relayoutSameGeometry(got, want) {
			t.Fatalf("box %q = %+v, want %+v", id, *got, *want)
		}
	}
}
