import React, { useState } from 'react';
import { CodeBlock } from './CodeBlock';

interface ApiCall {
  id: string;
  signature: string;
  summary: string;
  detail: string;
  filename: string;
  code: string;
}

const calls: ApiCall[] = [
  {
    id: 'parse',
    signature: 'html.Parse(source []byte) (*html.Document, error)',
    summary: 'Read UTF-8 HTML into a tree.',
    detail: 'The tree has no boxes yet. A nil source is an empty document. Pass the tree to css.Apply.',
    filename: 'parse.go',
    code: `package main

import "github.com/chinmay-sawant/blinkless/html"

func main() {
	doc, err := html.Parse([]byte("<h1>Hello</h1>"))
	if err != nil {
		panic(err)
	}
	_ = doc
}`,
  },
  {
    id: 'apply',
    signature: 'css.Apply(ctx context.Context, doc *html.Document, opts css.Options) (*css.Document, error)',
    summary: 'Apply the stylesheets to that tree.',
    detail: 'WidthPx and HeightPx are CSS pixels, and both must be greater than 0. An empty Media means screen.',
    filename: 'apply.go',
    code: `package main

import (
	"context"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
)

func main() {
	ctx := context.Background()
	doc, err := html.Parse([]byte("<h1>Hello</h1>"))
	if err != nil {
		panic(err)
	}
	styled, err := css.Apply(ctx, doc, css.Options{
		WidthPx:  800,
		HeightPx: 600,
	})
	if err != nil {
		panic(err)
	}
	_ = styled
}`,
  },
  {
    id: 'display',
    signature: 'layout.DisplayList(ctx context.Context, doc *css.Document) (*layout.Display, error)',
    summary: 'Place the tree and return the drawing list.',
    detail: 'display.Order is paint order. display.Ops is source order. X and Y are canvas points, y down. For text, Y is the baseline.',
    filename: 'list.go',
    code: `package main

import (
	"context"
	"fmt"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

func main() {
	ctx := context.Background()
	doc, err := html.Parse([]byte("<h1>Hello</h1>"))
	if err != nil {
		panic(err)
	}
	styled, err := css.Apply(ctx, doc, css.Options{
		WidthPx:  800,
		HeightPx: 600,
	})
	if err != nil {
		panic(err)
	}
	display, err := layout.DisplayList(ctx, styled)
	if err != nil {
		panic(err)
	}
	for _, index := range display.Order {
		op := display.Ops[index]
		fmt.Printf("%v %.1f %.1f\\n", op.Kind, op.X, op.Y)
	}
}`,
  },
];

export const ApiReference: React.FC = () => {
  const [openId, setOpenId] = useState(calls[0].id);

  return (
    <div className="space-y-3">
      {calls.map((call) => {
        const open = openId === call.id;
        return (
          <div key={call.id} className="border border-line rounded-md bg-sheet">
            <button
              type="button"
              aria-expanded={open}
              onClick={() => setOpenId(call.id)}
              className={`w-full cursor-pointer px-4 py-4 text-left ${open ? 'bg-paper-deep' : 'hover:bg-paper-deep'}`}
            >
              <span className="block font-mono text-sm text-ink break-words">{call.signature}</span>
              <span className="mt-2 block text-sm text-muted">{call.summary}</span>
            </button>
            {open ? (
              <div className="space-y-4 border-t border-line px-4 py-4">
                <p className="max-w-2xl text-base leading-relaxed text-muted">{call.detail}</p>
                <CodeBlock code={call.code} filename={call.filename} language="go" />
              </div>
            ) : null}
          </div>
        );
      })}
    </div>
  );
};
