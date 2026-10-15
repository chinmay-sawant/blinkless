import React, { useEffect, useRef, useState } from 'react';
import { BLINK_COMPARISONS, ENGINE_STATS } from '../data/properties';
import { CodeBlock } from './CodeBlock';
import { Logo } from './Logo';
import { Specimen } from './Specimen';

interface ProductLandingProps {
  onGoToDocs: (section?: string) => void;
}

const installCmd = `go get ${ENGINE_STATS.modulePath}`;

const pipeline = [
  {
    n: '01',
    title: 'Parse',
    text: 'html.Parse reads the document into a tree.',
  },
  {
    n: '02',
    title: 'Style',
    text: 'css.Apply cascades the stylesheets onto that tree.',
  },
  {
    n: '03',
    title: 'Layout',
    text: 'layout.DisplayList places the boxes.',
  },
  {
    n: '04',
    title: 'Drawing list',
    text: 'display.Order is paint order. display.Ops is source order.',
  },
];

const stops = [
  {
    title: 'JavaScript',
    text: 'Script in the document is not run. The tree stays as it was parsed.',
  },
  {
    title: 'PDF',
    text: 'Nothing writes a PDF file. layout.DisplayList returns the operations.',
  },
  {
    title: 'Picture of the page',
    text: 'Nothing encodes the whole page as a PNG or JPEG. Your program paints the operations. An image already in the HTML stays one image operation.',
  },
];

const limitSheets = [
  {
    title: 'Two modules',
    text: 'go-text/typesetting for shaping, tdewolff/canvas for SVG raster.',
  },
  {
    title: 'CGO',
    text: 'The default build sets CGO_ENABLED=0.',
  },
  {
    title: 'Caps',
    text: 'A stylesheet over 32 MiB is rejected. Element nesting stops at 1024.',
  },
];

const sampleGoCode = `package main

import (
	"context"
	"fmt"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

func main() {
	ctx := context.Background()
	doc, err := html.Parse([]byte(\`<h1>Hello</h1>\`))
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
}
`;

export const ProductLanding: React.FC<ProductLandingProps> = ({ onGoToDocs }) => {
  const [copied, setCopied] = useState(false);
  const copyTimer = useRef<number | null>(null);

  useEffect(() => {
    return () => {
      if (copyTimer.current !== null) {
        window.clearTimeout(copyTimer.current);
      }
    };
  }, []);

  const copyInstall = () => {
    if (!navigator.clipboard) {
      return;
    }
    void navigator.clipboard.writeText(installCmd).then(
      () => {
        setCopied(true);
        if (copyTimer.current !== null) {
          window.clearTimeout(copyTimer.current);
        }
        copyTimer.current = window.setTimeout(() => {
          setCopied(false);
          copyTimer.current = null;
        }, 2000);
      },
      () => {
        setCopied(false);
      },
    );
  };

  return (
    <div className="font-sans text-ink">
      <section aria-labelledby="landing-title" className="mx-auto max-w-6xl px-4 py-16 sm:px-6 sm:py-24">
        <div className="grid grid-cols-1 items-start gap-10 lg:grid-cols-12 lg:gap-12">
          <div className="lg:col-span-7">
            <p className="flex items-center gap-2 font-mono text-xs tracking-wide text-faint">
              <Logo size={16} className="h-4 w-4 shrink-0 text-ink" />
              <span>Go layout engine</span>
            </p>
            <h1
              id="landing-title"
              className="mt-4 font-display text-5xl leading-[1.05] tracking-tight text-ink sm:text-6xl"
            >
              <span className="block">HTML and CSS in.</span>
              <span className="block">
                A <span className="font-display italic text-ink">drawing list</span> out.
              </span>
            </h1>
            <p className="mt-6 max-w-xl font-sans text-lg leading-relaxed text-muted">
              Parse the HTML. Apply the stylesheets. Place the tree. The result is a drawing list your program paints.
            </p>
            <div className="mt-8 flex flex-wrap items-center gap-3">
              <button
                type="button"
                onClick={() => onGoToDocs()}
                className="inline-flex h-11 cursor-pointer items-center justify-center rounded bg-ink px-4 text-sm font-medium text-paper transition-colors duration-200 hover:bg-ink/90 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ink"
              >
                Read the docs
              </button>
              <div className="inline-flex h-11 max-w-full items-center rounded border border-line bg-sheet">
                <code className="truncate px-3 font-mono text-sm text-ink">{installCmd}</code>
                <button
                  type="button"
                  onClick={copyInstall}
                  aria-label={copied ? 'Copied install command' : 'Copy install command'}
                  className="inline-flex h-11 shrink-0 cursor-pointer items-center justify-center border-l border-line px-3 text-sm font-medium text-ink transition-colors duration-200 hover:bg-paper-deep focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ink"
                >
                  {copied ? 'Copied' : 'Copy'}
                </button>
              </div>
            </div>
          </div>

          <div className="lg:col-span-5">
            <Specimen />
          </div>
        </div>
      </section>

      <section aria-labelledby="pipeline-title" className="border-t border-line">
        <div className="mx-auto max-w-6xl px-4 py-16 sm:px-6 sm:py-20">
          <h2 id="pipeline-title" className="sr-only">
            Pipeline
          </h2>
          <ol className="grid grid-cols-1 gap-10 lg:grid-cols-4 lg:gap-8">
            {pipeline.map((step) => (
              <li key={step.n}>
                <p className="font-display text-3xl text-ink">{step.n}</p>
                <h3 className="mt-3 text-base font-medium text-ink">{step.title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-muted">{step.text}</p>
              </li>
            ))}
          </ol>
        </div>
      </section>

      <section aria-labelledby="limits-title" className="border-t border-line">
        <div className="mx-auto grid max-w-6xl grid-cols-1 gap-12 px-4 py-16 sm:px-6 sm:py-20 lg:grid-cols-2 lg:gap-16">
          <div>
            <h2 id="limits-title" className="font-display text-4xl text-ink">
              Where it stops
            </h2>
            <p className="mt-4 max-w-xl leading-relaxed text-muted">
              Layout ends when the drawing list comes back.
            </p>
            <ul className="mt-6 space-y-5">
              {stops.map((item) => (
                <li key={item.title}>
                  <h3 className="text-base font-medium text-ink">{item.title}</h3>
                  <p className="mt-1 text-sm leading-relaxed text-muted">{item.text}</p>
                </li>
              ))}
            </ul>
          </div>
          <div className="space-y-4">
            {limitSheets.map((sheet) => (
              <article key={sheet.title} className="rounded-md border border-line bg-sheet p-5">
                <h3 className="text-base font-medium text-ink">{sheet.title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-muted">{sheet.text}</p>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section aria-labelledby="compare-title" className="border-t border-line">
        <div className="mx-auto max-w-6xl px-4 py-16 sm:px-6 sm:py-20">
          <h2 id="compare-title" className="font-display text-4xl text-ink">
            Next to Blink
          </h2>
          <p className="mt-4 max-w-xl leading-relaxed text-muted">
            Blink is a browser engine. This is a library that stops at the drawing list.
          </p>
          <div className="mt-8 overflow-x-auto">
            <table className="w-full min-w-[40rem] text-left text-sm">
              <thead>
                <tr className="border-b border-line font-sans text-xs text-faint">
                  <th scope="col" className="py-3 pr-4 text-left font-medium">
                    Dimension
                  </th>
                  <th scope="col" className="py-3 pr-4 text-left font-medium">
                    Blink
                  </th>
                  <th scope="col" className="py-3 text-left font-medium">
                    blinkless
                  </th>
                </tr>
              </thead>
              <tbody>
                {BLINK_COMPARISONS.map((row) => (
                  <tr key={row.title} className="border-b border-line">
                    <th scope="row" className="py-4 pr-4 align-top font-medium text-ink">
                      <div>{row.title}</div>
                      <div className="mt-1 text-xs font-normal text-faint">{row.advantage}</div>
                    </th>
                    <td className="py-4 pr-4 align-top text-muted">{row.blink}</td>
                    <td className="py-4 align-top text-ink">{row.blinkless}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <section aria-labelledby="calls-title" className="border-t border-line">
        <div className="mx-auto max-w-6xl px-4 py-16 sm:px-6 sm:py-20">
          <h2 id="calls-title" className="font-display text-4xl text-ink">
            Three calls
          </h2>
          <p className="mt-4 text-muted">Parse, apply, then read the drawing list.</p>
          <div className="mt-8 max-w-3xl">
            <CodeBlock code={sampleGoCode} filename="main.go" language="go" />
          </div>
        </div>
      </section>

      <section className="border-t border-line">
        <div className="mx-auto flex max-w-6xl flex-col justify-between gap-4 px-4 pb-16 pt-8 sm:flex-row sm:px-6">
          <p className="text-muted">The CSS page lists every implemented and partial property.</p>
          <button
            type="button"
            onClick={() => onGoToDocs('properties')}
            className="inline-flex min-h-11 cursor-pointer items-center justify-center self-start rounded border border-ink px-4 text-sm font-medium text-ink transition-colors duration-200 hover:bg-ink hover:text-paper focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ink sm:self-auto"
          >
            Open the CSS catalog
          </button>
        </div>
      </section>
    </div>
  );
};
