package layout

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	pdf "github.com/chinmay-sawant/blinkless/internal/fonts"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// inlineOpSnapshot renders the fields the single-run fast path must preserve
// exactly. IDs and opExtra addresses are excluded: a second layout pass on the
// same engine legitimately advances both.
func inlineOpSnapshot(op Op) string {
	uri, blend, transform := "", "", ""
	groupMark := uint8(0)

	if op.opExtra != nil {
		uri, blend, transform, groupMark = op.URI, op.BlendMode, op.TextTransform, op.GroupMark
	}

	return fmt.Sprintf(
		"kind=%d x=%.9f y=%.9f w=%.9f h=%.9f rgb=%.9f/%.9f/%.9f a=%.9f "+
			"width=%.9f size=%.9f ls=%.9f ink=%.9f text=%q font=%p bold=%t "+
			"oblique=%t rot=%.9f uri=%q blend=%q transform=%q group=%d",
		op.Kind, op.X, op.Y, op.W, op.H, op.R, op.G, op.B, op.Alpha,
		op.Width, op.Size, op.LetterSpacing, op.InkDescent, op.Text, op.Font,
		op.Bold, op.FakeOblique, op.RotateDeg, uri, blend, transform, groupMark,
	)
}

func inlineOpsSnapshot(ops []Op) []string {
	out := make([]string, len(ops))

	for idx := range ops {
		out[idx] = inlineOpSnapshot(ops[idx])
	}

	return out
}

func newInlineSingleRunEngine(t *testing.T, src, cssText string) (*engine, *html.Node) {
	t.Helper()

	root := mustParse(t, src)

	var sheets []*css.Stylesheet
	if cssText != "" {
		sheets = append(sheets, sheet(t, cssText))
	}

	styles := resolveStyles(root, sheets, "print", testViewport, 800)

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatalf("LoadDefaultFaces: %v", err)
	}

	opts := Options{Width: testViewport, Height: 800, Sheets: sheets}
	eng := newEngine(context.Background(), opts, faces, faces.Regular, styles, nil, nil)

	return eng, root
}

func newInlineSingleRunBox(eng *engine, block *html.Node) *box {
	return &box{node: block, style: eng.stylePtr(block), kind: boxKindBlock}
}

// TestSingleTextRunFastPathMatchesGeneric is the differential gate for the
// one-text-run fast path: every representative cell must emit the same ops,
// height, and first baseline as the generic collector, and cases that cannot
// prove one-line behavior must decline the fast path.
func TestSingleTextRunFastPathMatchesGeneric(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		body     string
		css      string
		contentW float64
		wantFast bool
	}{
		{
			name:     "plain-cell-with-padding-border-and-background",
			body:     `<div class="cell"><span class="label">R0C0</span></div>`,
			css:      `.cell{font-size:10pt;color:#223344;padding:1px 4px;border:1px solid #ccc;background-color:#f8f8f8}.label{padding:1px 2px}`,
			contentW: 180,
			wantFast: true,
		},
		{
			name:     "text-align-right",
			body:     `<div class="cell"><span class="label">R0C0</span></div>`,
			css:      `.cell{text-align:right}.label{font-size:10pt}`,
			contentW: 180,
			wantFast: true,
		},
		{
			name:     "text-align-center",
			body:     `<div class="cell"><span class="label">R0C0</span></div>`,
			css:      `.cell{text-align:center}.label{font-size:10pt}`,
			contentW: 180,
			wantFast: true,
		},
		{
			name:     "font-size-18pt",
			body:     `<div class="cell"><span class="label">R0C0</span></div>`,
			css:      `.cell{font-size:18pt;color:#445566}`,
			contentW: 240,
			wantFast: true,
		},
		{
			name:     "color",
			body:     `<div class="cell"><span class="label">R0C0</span></div>`,
			css:      `.cell{color:#336699}`,
			contentW: 180,
			wantFast: true,
		},
		{
			name:     "wide-content-width",
			body:     `<div class="cell"><span class="label">R0C0</span></div>`,
			css:      `.cell{font-size:10pt}`,
			contentW: 400,
			wantFast: true,
		},
		{
			name:     "narrow-content-width-still-fits",
			body:     `<div class="cell"><span class="label">R0C0</span></div>`,
			css:      `.cell{font-size:10pt}`,
			contentW: 60,
			wantFast: true,
		},
		{
			name:     "direct-text-node",
			body:     `<div class="cell">R0C0</div>`,
			css:      `.cell{font-size:10pt;color:#223344}`,
			contentW: 180,
			wantFast: true,
		},
		{
			name:     "anchor-href",
			body:     `<div class="cell"><a class="label" href="https://example.com/inv">INV-1</a></div>`,
			css:      `.cell{font-size:10pt}.label{color:#0055aa;text-decoration:none}`,
			contentW: 180,
			wantFast: true,
		},
		{
			name:     "text-that-wraps",
			body:     `<div class="cell"><span class="label">Long value that wraps</span></div>`,
			css:      `.cell{font-size:10pt}`,
			contentW: 50,
			wantFast: false,
		},
		{
			name:     "single-token-wider-than-line",
			body:     `<div class="cell"><span class="label">R0C0R0C0R0C0R0C0</span></div>`,
			css:      `.cell{font-size:10pt}`,
			contentW: 40,
			wantFast: false,
		},
		{
			name:     "underline-decoration",
			body:     `<div class="cell"><span class="label">R0C0</span></div>`,
			css:      `.cell{font-size:10pt}.label{text-decoration:underline}`,
			contentW: 180,
			wantFast: false,
		},
		{
			name:     "rtl-direction",
			body:     `<div class="cell"><span class="label">R0C0</span></div>`,
			css:      `.cell{font-size:10pt;direction:rtl}`,
			contentW: 180,
			wantFast: false,
		},
		{
			name:     "white-space-nowrap",
			body:     `<div class="cell"><span class="label">R0C0</span></div>`,
			css:      `.cell{font-size:10pt;white-space:nowrap}`,
			contentW: 180,
			wantFast: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			eng, root := newInlineSingleRunEngine(t, tc.body, tc.css)
			block := firstElementNamed(root, "div")
			if block == nil || len(block.Children) == 0 {
				t.Fatal("fixture block has no inline child")
			}

			nodes := []*html.Node{block.Children[0]}
			run := func(fn func(*engine, *box) float64) ([]string, float64, float64) {
				boxNode := newInlineSingleRunBox(eng, block)
				eng.ops = nil

				h := fn(eng, boxNode)

				return inlineOpsSnapshot(eng.ops), h, boxNode.firstBaseline
			}

			generic, genericH, genericBaseline := run(func(e *engine, b *box) float64 {
				return e.layoutInlineFloatsGeneric(b, nodes, tc.contentW, 12.5, 7.25, nil)
			})

			fast, fastH, fastBaseline := run(func(e *engine, b *box) float64 {
				h, _ := e.layoutSingleTextRun(b, nodes, tc.contentW, 12.5, 7.25, nil)

				return h
			})

			// The fast path must decline before reachable uncertainty, but
			// when it accepts it must match the generic output exactly.
			boxNode := newInlineSingleRunBox(eng, block)

			if _, ok := eng.layoutSingleTextRun(boxNode, nodes, tc.contentW, 12.5, 7.25, nil); ok != tc.wantFast {
				t.Fatalf("layoutSingleTextRun accepted = %t, want %t", ok, tc.wantFast)
			}

			dispatcher, dispatcherH, dispatcherBaseline := run(func(e *engine, b *box) float64 {
				return e.layoutInlineFloats(b, nodes, tc.contentW, 12.5, 7.25, nil)
			})

			assertInlineSnapshotEqual(t, "generic vs dispatcher", generic, dispatcher)

			if !near(dispatcherH, genericH) {
				t.Fatalf("dispatcher height = %v, generic height = %v", dispatcherH, genericH)
			}

			if !near(dispatcherBaseline, genericBaseline) {
				t.Fatalf("dispatcher firstBaseline = %v, generic firstBaseline = %v",
					dispatcherBaseline, genericBaseline)
			}

			if tc.wantFast {
				assertInlineSnapshotEqual(t, "generic vs fast", generic, fast)

				if !near(fastH, genericH) {
					t.Fatalf("fast height = %v, generic height = %v", fastH, genericH)
				}

				if !near(fastBaseline, genericBaseline) {
					t.Fatalf("fast firstBaseline = %v, generic firstBaseline = %v", fastBaseline, genericBaseline)
				}

				if !hasInlineTextOp(generic) {
					t.Fatal("eligible fast path emitted no text op")
				}

				return
			}

			if len(fast) != 0 {
				t.Fatalf("declined fast path emitted %d ops, want none", len(fast))
			}
		})
	}
}

func assertInlineSnapshotEqual(t *testing.T, label string, want, got []string) {
	t.Helper()

	if len(want) != len(got) {
		t.Fatalf("%s: op count = %d, want %d\nwant: %s\ngot:  %s",
			label, len(got), len(want), strings.Join(want, "\n"), strings.Join(got, "\n"))
	}

	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("%s: op %d = %s, want %s", label, idx, got[idx], want[idx])
		}
	}
}

func hasInlineTextOp(ops []string) bool {
	for _, op := range ops {
		if strings.Contains(op, "kind=4 ") { // OpText
			return true
		}
	}

	return false
}
