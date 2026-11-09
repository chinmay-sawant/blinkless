package css

import (
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	pubhtml "github.com/chinmay-sawant/blinkless/html"
	ihtml "github.com/chinmay-sawant/blinkless/internal/html"
)

func hoverCacheTestStyled(t *testing.T, styleRules string, rows int) *Document {
	t.Helper()

	var out strings.Builder
	out.WriteString(`<style>`)
	out.WriteString(styleRules)
	out.WriteString(`</style><div id="panel">`)
	for i := range rows {
		cls := "inv-row"
		switch i % 3 {
		case 1:
			cls = "inv-alt"
		case 2:
			cls = "inv-sel"
		}
		out.WriteString(`<div class="`)
		out.WriteString(cls)
		out.WriteString(`" id="r`)
		out.WriteString(strconv.Itoa(i))
		out.WriteString(`"><span class="label">x</span></div>`)
	}
	out.WriteString(`</div>`)

	doc, err := pubhtml.Parse([]byte(out.String()))
	if err != nil {
		t.Fatal(err)
	}

	styled, err := Apply(t.Context(), doc, Options{WidthPx: 320, HeightPx: 200, Hover: "r20"})
	if err != nil {
		t.Fatal(err)
	}

	return styled
}

func hoverCacheReference(t *testing.T, styled *Document, previous string) ([]string, bool) {
	t.Helper()

	old := styled.state
	old.Hover = previous
	selectors := hoverSelectors(styled.sheets)
	scopes := make(map[*ihtml.Node]bool)
	complete := true

	var walk func(*ihtml.Node)
	walk = func(node *ihtml.Node) {
		if !complete {
			return
		}
		if node.Type == ihtml.ElementNode && hoverMatchChanged(node, selectors, old, styled.state) {
			scope := hoverIDScope(node)
			if scope == nil {
				complete = false

				return
			}
			scopes[scope] = true
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(styled.root)
	if !complete {
		t.Fatal("reference walk found a scope without an ID")
	}

	return outerHoverScopes(styled.root, scopes), true
}

func TestHoverCacheParityMultiHundred(t *testing.T) { //nolint:paralleltest // reads process-wide hover memo counters
	resetHoverCache()

	rules := `.inv-row:hover{color:red}.inv-alt:hover{color:blue}.inv-sel:hover{color:green}` +
		`.inv-row:hover .label{color:red}` +
		`.inv-row:hover::before{content:'x'}` +
		`div.inv-row:hover + div.inv-row{color:black}`
	styled := hoverCacheTestStyled(t, rules, 400)

	for _, previous := range []string{"r50", "", "r399"} {
		got, ok := styled.HoverChangedScopes(previous)
		if !ok {
			t.Fatalf("previous %q: HoverChangedScopes returned false", previous)
		}
		want, ok := hoverCacheReference(t, styled, previous)
		if !ok {
			t.Fatalf("previous %q: reference walk was incomplete", previous)
		}
		if !slices.Equal(got, want) {
			t.Fatalf("previous %q: scopes = %v, want %v", previous, got, want)
		}
	}
}

func TestHoverCacheSecondCallHitsMemo(t *testing.T) { //nolint:paralleltest // reads process-wide hover memo counters
	resetHoverCache()

	rules := `.inv-row:hover{color:red}.inv-alt:hover{color:blue}.inv-sel:hover{color:green}`
	styled := hoverCacheTestStyled(t, rules, 100)

	first, ok := styled.HoverChangedScopes("r10")
	if !ok {
		t.Fatal("first call returned false")
	}
	hitsAfterFirst := hoverMemoHits.Load()
	missesAfterFirst := hoverMemoMisses.Load()
	if missesAfterFirst == 0 {
		t.Fatal("first call recorded no memo misses")
	}

	second, ok := styled.HoverChangedScopes("r10")
	if !ok {
		t.Fatal("second call returned false")
	}
	if !slices.Equal(first, second) {
		t.Fatalf("second call scopes = %v, want %v", second, first)
	}

	hitsAfterSecond := hoverMemoHits.Load()
	missesAfterSecond := hoverMemoMisses.Load()
	if hitsAfterSecond <= hitsAfterFirst {
		t.Fatalf("second call added no hits: %d then %d", hitsAfterFirst, hitsAfterSecond)
	}
	if missesAfterSecond != missesAfterFirst {
		t.Fatalf("local-only second call added misses: %d then %d", missesAfterFirst, missesAfterSecond)
	}
	if hoverListHits.Load() == 0 {
		t.Fatal("selector list was not reused across calls")
	}

	third, ok := styled.HoverChangedScopes("r30")
	if !ok {
		t.Fatal("third call returned false")
	}
	if len(third) == 0 {
		t.Fatal("third call returned no scopes")
	}
	hitsAfterThird := hoverMemoHits.Load()
	missesAfterThird := hoverMemoMisses.Load()
	if hitsAfterThird <= hitsAfterSecond {
		t.Fatalf("third call added no hits for cached endpoint: %d then %d", hitsAfterSecond, hitsAfterThird)
	}
	if missesAfterThird <= missesAfterSecond {
		t.Fatalf("third call added no misses for new endpoint: %d then %d", missesAfterSecond, missesAfterThird)
	}
}

func TestHoverCacheConcurrentRace(t *testing.T) { //nolint:paralleltest // resets a global cache
	resetHoverCache()

	rules := `.inv-row:hover{color:red}.inv-alt:hover{color:blue}.inv-sel:hover{color:green}` +
		`.inv-row:hover .label{color:red}` +
		`div.inv-row:hover + div.inv-row{color:black}`
	styled := hoverCacheTestStyled(t, rules, 300)

	const workers = 8
	const perWorker = 25

	var mu sync.Mutex
	var results [][]string

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perWorker {
				got, ok := styled.HoverChangedScopes("r10")
				if !ok {
					t.Error("concurrent call returned false")

					return
				}
				mu.Lock()
				results = append(results, got)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if t.Failed() {
		t.FailNow()
	}
	if len(results) != workers*perWorker {
		t.Fatalf("got %d results, want %d", len(results), workers*perWorker)
	}
	for i := 1; i < len(results); i++ {
		if !slices.Equal(results[i], results[0]) {
			t.Fatalf("concurrent result %d = %v, want %v", i, results[i], results[0])
		}
	}
}
