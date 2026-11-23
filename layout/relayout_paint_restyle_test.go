package layout_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"runtime"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	ilayout "github.com/chinmay-sawant/blinkless/internal/layout"
	"github.com/chinmay-sawant/blinkless/layout"
)

// restyleCase is one class edit on one element. wantHit is true when the
// edit only moves a background color the patch can prove, and false when it
// must keep the full build.
type restyleCase struct {
	name    string
	source  string
	id      string
	base    string
	wantHit bool
	// revertHits is the expectation for the edit that removes the class again,
	// when it differs from wantHit.
	revertHits *bool
}

func boolPtr(value bool) *bool { return &value }

const restyleGlass = `display:flex;flex-direction:column;gap:4px;border-radius:12px;` +
	`background:rgba(255,255,255,0.14);border:1px solid rgba(255,255,255,0.25);` +
	`backdrop-filter:blur(12px);box-shadow:0 8px 24px rgba(0,0,0,0.35);width:84px;height:22px;` +
	`padding:2px 4px;box-sizing:border-box;font-size:11px`

func restyleCases() []restyleCase {
	return []restyleCase{
		{
			name: "plain background color", id: "t", wantHit: true,
			source: `<style>.box{width:120px;height:40px;background:#eeeeee}.box.on{background:#3366cc}</style>
<div id="t" class="box">Alpha</div><div id="u" class="box">Beta</div>`,
		},
		{
			name: "glass alpha in a flex grid", id: "t", base: "card", wantHit: true,
			source: `<style>.grid{display:flex;flex-wrap:wrap;gap:4px;padding:4px}.card{` + restyleGlass +
				`}.card.on{background:rgba(255,255,255,0.22)}</style><div class="grid">` +
				`<div id="a" class="card">one</div><div id="t" class="card">two</div><div id="c" class="card">three</div></div>`,
		},
		{
			name: "nested flex container", id: "t", base: "row", wantHit: true,
			source: `<style>.row{display:flex;flex-direction:row;gap:6px;width:240px;height:60px;background:#ddd}
.row.on{background:#bbccee}.cell{flex:1;height:30px;background:#fff}</style>
<div id="t" class="row"><div class="cell">a</div><div class="cell">b</div></div>`,
		},
		{
			name: "clipped overflow ancestor", id: "t", wantHit: true,
			source: `<style>.clip{overflow:hidden;width:200px;height:80px}.box{width:100px;height:40px;background:#eee}
.box.on{background:#cc6633}</style><div class="clip"><div id="t" class="box">Clip</div></div>`,
		},
		{
			name: "backdrop filter ancestor", id: "t", wantHit: true,
			source: `<style>.frost{backdrop-filter:blur(4px);width:200px;height:80px;background:rgba(255,255,255,.2)}
.box{width:100px;height:40px;background:#eee}.box.on{background:#99cc33}</style>
<div class="frost"><div id="t" class="box">Frost</div></div>`,
		},
		{
			name: "text child", id: "t", wantHit: true,
			source: `<style>.box{width:160px;height:50px;background:#eef;color:#123;font-size:14px}
.box.on{background:#fee}</style><div id="t" class="box">Some label text</div>`,
		},
		{
			name: "transition declared", id: "t", wantHit: true,
			source: `<style>.box{width:120px;height:40px;background:#eee;transition:background-color 1s ease}
.box.on{background:#abcdef}</style><div id="t" class="box">T</div>`,
		},
		{
			name: "siblings that both group", id: "t", base: "card", wantHit: true,
			source: `<style>.card{width:90px;height:30px;backdrop-filter:blur(6px);background:rgba(0,0,0,.2)}
.card.on{background:rgba(0,0,0,.4)}</style><div id="a" class="card">a</div><div id="t" class="card">t</div>`,
		},
		{
			name: "element opacity group keeps its group", id: "t", wantHit: true,
			source: `<style>.box{width:120px;height:40px;opacity:.6;background:#eee}.box.on{background:#222}</style>
<div id="t" class="box">O</div>`,
		},
		{
			name: "width change", id: "t",
			source: `<style>.box{width:120px;height:40px;background:#eee}.box.on{width:160px;background:#222}</style>
<div id="t" class="box">W</div>`,
		},
		{
			name: "padding change", id: "t",
			source: `<style>.box{width:120px;height:40px;background:#eee}.box.on{padding:10px;background:#222}</style>
<div id="t" class="box">P</div>`,
		},
		{
			name: "margin change", id: "t",
			source: `<style>.box{width:120px;height:40px;background:#eee}.box.on{margin-top:12px;background:#222}</style>
<div id="t" class="box">M</div><div id="n">next</div>`,
		},
		{
			name: "transform change", id: "t",
			source: `<style>.box{width:120px;height:40px;background:#eee}
.box.on{transform:translateY(-3px);background:#222}</style>
<div id="t" class="box">X</div>`,
		},
		{
			name: "opacity group appears", id: "t",
			source: `<style>.box{width:120px;height:40px;background:#eee}.box.on{opacity:.5;background:#222}</style>
<div id="t" class="box">G</div>`,
		},
		{
			name: "text color moves too", id: "t",
			source: `<style>.box{width:120px;height:40px;background:#eee;color:#000}.box.on{color:#f00;background:#222}</style>
<div id="t" class="box">C</div>`,
		},
		{
			name: "border color moves too", id: "t",
			source: `<style>.box{width:120px;height:40px;background:#eee;border:2px solid #000}
.box.on{border-color:#f00;background:#222}</style>
<div id="t" class="box">B</div>`,
		},
		{
			name: "background goes transparent", id: "t",
			source: `<style>.box{width:120px;height:40px;background:#eee}.box.on{background:transparent}</style>
<div id="t" class="box">N</div>`,
		},
		{
			name: "background image appears", id: "t",
			source: `<style>.box{width:120px;height:40px;background:#eee}
.box.on{background-color:#222;background-image:linear-gradient(#fff,#000)}</style>
<div id="t" class="box">I</div>`,
		},
		{
			name: "display none", id: "t",
			source: `<style>.box{width:120px;height:40px;background:#eee}.box.on{display:none}</style>
<div id="t" class="box">D</div><div id="n">next</div>`,
		},
		{
			name: "child paints the same color", id: "t", revertHits: boolPtr(true),
			source: `<style>.box{width:160px;height:80px;background:#eee}.box.on{background:#222}
.kid{width:160px;height:80px;background:#eee}</style><div id="t" class="box"><div class="kid">k</div></div>`,
		},
	}
}

// patchChain patches id on display after an edit and returns the result.
func patchChain(t *testing.T, styled *css.Document, display *layout.Display, id string) *layout.Display {
	t.Helper()

	return patchChainCtx(t.Context(), t, styled, display, id)
}

// patchChainCtx is patchChain under a caller context, which can carry a test
// config.
func patchChainCtx(
	ctx context.Context, t *testing.T, styled *css.Document, display *layout.Display, id string,
) *layout.Display {
	t.Helper()

	next, err := layout.RelayoutElements(ctx, styled, display, []string{id}, layout.Options{RetainState: true})
	if err != nil {
		t.Fatalf("RelayoutElements(%s): %v", id, err)
	}

	return next
}

// slowPatch is patchChain with the fast path switched off for this call only.
func slowPatch(styled *css.Document, display *layout.Display, id string) (*layout.Display, error) {
	ctx := ilayout.WithTestConfig(context.Background(), ilayout.TestConfig{PaintRestyleDisabled: true})

	next, err := layout.RelayoutElements(
		ctx, styled, display, []string{id}, layout.Options{RetainState: true})
	if err != nil {
		return nil, fmt.Errorf("slow relayout of %s: %w", id, err)
	}

	return next, nil
}

// compareOpLists requires two displays to carry the same operations.
func compareOpLists(t *testing.T, got, want *layout.Display) {
	t.Helper()

	if len(got.Ops) != len(want.Ops) {
		t.Fatalf("ops %d, want %d", len(got.Ops), len(want.Ops))
	}

	for i := range want.Ops {
		compareOp(t, i, &got.Ops[i], &want.Ops[i])
	}
}

// TestPaintRestyleMatchesSlowAndFull edits one class on one element, in both
// directions, and requires the patch to equal a full layout (operations,
// boxes, owners, order; operation ids differ between any patch and a fresh
// layout) and the forced slow path (operations). It also pins which edits skip
// the build. A patch the engine refuses (a resize) is the host's cue to lay
// the page out again, so a refused edit only has to stay off the fast path.
func TestPaintRestyleMatchesSlowAndFull(t *testing.T) {
	for _, tc := range restyleCases() {
		t.Run(tc.name, func(t *testing.T) {
			tree, err := html.Parse([]byte(tc.source))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			styled := mustApply(t, tree, 400, 300, "", "", "")
			fast := layRetained(t, styled)
			slow := layRetained(t, styled)

			base := tc.base
			if base == "" {
				base = "box"
			}

			for edit, class := range []string{base + " on", base} {
				if !tree.SetAttribute(tc.id, "class", class) {
					t.Fatalf("no element with id %s", tc.id)
				}

				want := tc.wantHit
				if edit == 1 && tc.revertHits != nil {
					want = *tc.revertHits
				}

				before := ilayout.PaintRestyleStatsSnapshot()
				next, fastErr := layout.RelayoutElements(
					t.Context(), styled, fast, []string{tc.id}, layout.Options{RetainState: true})
				hit := ilayout.PaintRestyleStatsSnapshot().Hits > before.Hits

				if hit != want {
					t.Fatalf("class %q: fast path hit = %v, want %v (err %v)", class, hit, want, fastErr)
				}

				slowNext, slowErr := slowPatch(styled, slow, tc.id)
				if (fastErr == nil) != (slowErr == nil) {
					t.Fatalf("class %q: fast error %v, slow error %v", class, fastErr, slowErr)
				}

				full := layRetained(t, styled)
				if fastErr != nil {
					// The host lays out again; continue from a fresh retained layout.
					fast, slow = full, layRetained(t, styled)

					continue
				}

				if hit {
					// A taken fast path is held to a full layout, the reference.
					compareDisplaysIgnoringOpIDs(t, next, full)

					// The slow path is held to it only where it is itself exact. It adds
					// two group-mark operations per patch on a grouped element.
					if len(slowNext.Ops) == len(full.Ops) {
						compareOpLists(t, next, slowNext)
					} else {
						t.Logf("slow path carries %d operations, a full layout %d", len(slowNext.Ops), len(full.Ops))
					}
				} else {
					// A declined attempt must leave the ordinary path untouched.
					compareOpLists(t, next, slowNext)
				}

				fast, slow = next, slowNext
			}
		})
	}
}

// TestPaintRestyleHoverMatchesFull drives the glass hover the way a host does:
// the hover state changes through css.Relayout, and every enter and leave
// patches the previous result. Each step must equal a full layout.
func TestPaintRestyleHoverMatchesFull(t *testing.T) {
	source := restyleHoverSource(12, "h3")
	base := applyAt(t, source, 400, 300, "", "", "")
	hovered, err := css.Relayout(t.Context(), base, 400, 300, "", "h3", "")
	if err != nil {
		t.Fatalf("relayout: %v", err)
	}

	current := layRetained(t, base)
	before := ilayout.PaintRestyleStatsSnapshot()

	for step := range 6 {
		doc, state := hovered, "hover"
		if step%2 == 1 {
			doc, state = base, "leave"
		}

		current = patchChain(t, doc, current, "h3")
		compareDisplays(t, current, layRetained(t, doc))

		if !hasFill(current, 255, 255, 255) {
			t.Fatalf("step %d (%s): no card fill", step, state)
		}
	}

	if got := ilayout.PaintRestyleStatsSnapshot().Hits - before.Hits; got != 6 {
		t.Fatalf("restyle hits = %d, want 6", got)
	}
}

func restyleHoverSource(count int, hovered string) string {
	var b strings.Builder

	b.WriteString(`<style>.grid{display:flex;flex-wrap:wrap;gap:4px;padding:4px}.card{` + restyleGlass +
		`}#` + hovered + `:hover{background:rgba(255,255,255,0.22)}</style><div class="grid">`)

	for i := range count {
		fmt.Fprintf(&b, `<div id="h%d" class="card">item %d</div>`, i, i)
	}

	b.WriteString(`</div>`)

	return b.String()
}

// TestPaintRestyleRandomChain edits random cards through random background
// classes, patching the previous result each time, and checks every step
// against a full layout. Every edit that moves the color must take the fast
// path.
func TestPaintRestyleRandomChain(t *testing.T) {
	const cards, classes, steps = 10, 6, 60

	var b strings.Builder

	b.WriteString(`<style>.grid{display:flex;flex-wrap:wrap;gap:4px;padding:4px}.card{` + restyleGlass + `}`)

	for k := range classes {
		fmt.Fprintf(&b, `.k%d{background:rgba(%d,%d,%d,0.%d)}`, k, 20+k*30, 200-k*25, 40+k*35, 10+k*13)
	}

	b.WriteString(`</style><div class="grid">`)

	for i := range cards {
		fmt.Fprintf(&b, `<div id="c%d" class="card k0">card %d</div>`, i, i)
	}

	b.WriteString(`</div>`)

	tree, err := html.Parse([]byte(b.String()))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")
	current := layRetained(t, styled)
	classOf := make([]int, cards)
	rng := rand.New(rand.NewPCG(7, 11)) //nolint:gosec // a fixed seed keeps the edit sequence reproducible
	before := ilayout.PaintRestyleStatsSnapshot()
	edits := uint64(0)

	for range steps {
		card, class := rng.IntN(cards), rng.IntN(classes)
		if class == classOf[card] {
			continue
		}

		classOf[card] = class
		id := fmt.Sprintf("c%d", card)

		if !tree.SetAttribute(id, "class", fmt.Sprintf("card k%d", class)) {
			t.Fatalf("no element %s", id)
		}

		current = patchChain(t, styled, current, id)
		compareDisplays(t, current, layRetained(t, styled))

		edits++
	}

	if got := ilayout.PaintRestyleStatsSnapshot().Hits - before.Hits; got != edits {
		t.Fatalf("restyle hits = %d for %d color edits", got, edits)
	}
}

// TestPaintRestyleBatchMixesFastAndSlowIDs patches one color-only element and
// one resizing element in one call. The batch must still equal a full layout.
func TestPaintRestyleBatchMixesFastAndSlowIDs(t *testing.T) {
	source := `<style>.box{width:120px;height:40px;background:#eee}.box.on{background:#369}
.wide{width:100px;height:30px;background:#ddd}.wide.on{width:180px}</style>
<div id="a" class="box">A</div><div id="b" class="wide">B</div>`

	tree, err := html.Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")
	display := layRetained(t, styled)

	if !tree.SetAttribute("a", "class", "box on") || !tree.SetAttribute("b", "class", "wide on") {
		t.Fatal("missing element")
	}

	before := ilayout.PaintRestyleStatsSnapshot()

	got, err := layout.RelayoutElements(
		t.Context(), styled, display, []string{"a", "b"}, layout.Options{RetainState: true})
	if err != nil {
		t.Fatalf("RelayoutElements: %v", err)
	}

	after := ilayout.PaintRestyleStatsSnapshot()
	compareDisplays(t, got, layRetained(t, styled))

	if after.Hits-before.Hits != 1 {
		t.Fatalf("restyle hits = %d, want the color-only id to hit once", after.Hits-before.Hits)
	}
}

// TestPaintRestyleTextEditKeepsFullBuild changes the text and the color
// together. The fast path must not claim an edit it cannot see.
func TestPaintRestyleTextEditKeepsFullBuild(t *testing.T) {
	source := `<style>.box{width:200px;height:40px;background:#eee}.box.on{background:#369}</style>
<div id="t" class="box"><span id="s">Alpha</span></div>`

	tree, err := html.Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")
	display := layRetained(t, styled)

	if !tree.SetAttribute("t", "class", "box on") || !tree.SetText("s", "Omega") {
		t.Fatal("missing element")
	}

	before := ilayout.PaintRestyleStatsSnapshot()
	got := patchChain(t, styled, display, "t")

	if ilayout.PaintRestyleStatsSnapshot().Hits != before.Hits {
		t.Fatal("a text edit took the paint restyle path")
	}

	compareDisplays(t, got, layRetained(t, styled))
}

// TestPaintRestyleAllocations pins the point of the fast path on the glass
// grid: a chained hover enter and leave allocates a small fraction of what
// the build and commit allocate (10x fewer objects, at least 2x fewer bytes).
func TestPaintRestyleAllocations(t *testing.T) {
	source := restyleHoverSource(64, "h0")
	base := applyAt(t, source, 760, 520, "", "", "")
	hovered, err := css.Relayout(t.Context(), base, 760, 520, "", "h0", "")
	if err != nil {
		t.Fatalf("relayout: %v", err)
	}

	measure := func(ctx context.Context) (float64, uint64) {
		current, err := layout.DisplayListOptions(ctx, base, layout.Options{RetainState: true})
		if err != nil {
			t.Fatalf("display: %v", err)
		}

		flip := false
		step := func() {
			doc := hovered
			if flip {
				doc = base
			}

			flip = !flip
			current = patchChainCtx(ctx, t, doc, current, "h0")
		}

		allocs := testing.AllocsPerRun(40, step)

		var m0, m1 runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&m0)

		const rounds = 40
		for range rounds {
			step()
		}

		runtime.ReadMemStats(&m1)

		return allocs, (m1.TotalAlloc - m0.TotalAlloc) / rounds
	}

	fastAllocs, fastBytes := measure(t.Context())

	slowCtx := ilayout.WithTestConfig(t.Context(), ilayout.TestConfig{PaintRestyleDisabled: true})
	slowAllocs, slowBytes := measure(slowCtx)

	t.Logf("glass grid, 64 cards: fast path %.0f allocs %d B per patch; slow path %.0f allocs %d B per patch",
		fastAllocs, fastBytes, slowAllocs, slowBytes)

	// The bytes that remain are the private copy of the operation list and its
	// payloads that every patch makes so the previous display stays intact.
	if fastAllocs*10 > slowAllocs || fastBytes*2 > slowBytes {
		t.Fatalf("fast path did not cut the cost enough: %.0f vs %.0f allocs, %d vs %d bytes",
			fastAllocs, slowAllocs, fastBytes, slowBytes)
	}
}
