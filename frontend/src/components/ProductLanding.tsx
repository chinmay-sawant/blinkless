import React, { useState } from 'react';
import { BLINK_COMPARISONS, ENGINE_STATS } from '../data/properties';
import { ArrowRight, Check, Copy, Shield, Cpu, Zap, Layers, Star, GitBranch } from 'lucide-react';
import { Logo } from './Logo';
import { CodeBlock } from './CodeBlock';

interface ProductLandingProps {
  onGoToDocs: (section?: string) => void;
}

export const ProductLanding: React.FC<ProductLandingProps> = ({ onGoToDocs }) => {
  const [copiedInstall, setCopiedInstall] = useState(false);
  const installCmd = 'go get github.com/chinmay-sawant/blinkless';

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

	// 1. Parse HTML into memory tree
	doc, err := html.Parse([]byte(\`<div style="font-size: 16px; padding: 20px;">Pure Go Layout</div>\`))
	if err != nil {
		panic(err)
	}

	// 2. Cascade CSS rules and dimensions
	styled, err := css.Apply(ctx, doc, css.Options{
		WidthPx:  1024,
		HeightPx: 768,
	})
	if err != nil {
		panic(err)
	}

	// 3. Emit deterministic vector DisplayList
	display, err := layout.DisplayList(ctx, styled)
	if err != nil {
		panic(err)
	}

	// 4. Render drawing operations in stacking order
	for _, idx := range display.Order {
		op := display.Ops[idx]
		fmt.Printf("DisplayOp: kind=%v at (%.1f, %.1f)\\n", op.Kind, op.X, op.Y)
	}
}`;

  const copyInstall = () => {
    navigator.clipboard.writeText(installCmd);
    setCopiedInstall(true);
    setTimeout(() => setCopiedInstall(false), 2000);
  };

  return (
    <div className="bg-zinc-950 text-zinc-100">
      {/* Hero Section */}
      <section className="py-20 sm:py-28 px-4 sm:px-6 lg:px-8 border-b border-zinc-800/80">
        <div className="max-w-4xl mx-auto text-center">
          {/* Distinctive Logo Emblem */}
          <div className="flex justify-center mb-6">
            <div className="w-14 h-14 rounded-xl bg-zinc-900 border border-zinc-700/80 flex items-center justify-center text-white shadow-inner">
              <Logo size={36} className="w-9 h-9 text-white" />
            </div>
          </div>

          <div className="inline-flex items-center gap-2 px-3 py-1 rounded bg-zinc-900 border border-zinc-800 text-xs font-mono text-zinc-300 mb-6">
            <span className="w-1.5 h-1.5 rounded-full bg-zinc-400"></span>
            <span>Blink Alternative in Pure Go &bull; {ENGINE_STATS.version}</span>
          </div>

          <h1 className="text-4xl sm:text-5xl lg:text-6xl font-bold tracking-tight text-white mb-6 leading-tight">
            A script-free alternative to Chromium's Blink.
          </h1>

          <p className="text-base sm:text-lg text-zinc-400 max-w-2xl mx-auto mb-10 leading-relaxed font-normal">
            Blinkless is an embeddable, memory-safe HTML and CSS layout engine written in pure Go. It cascades styles and emits vector drawing lists without JavaScript, without CGO, and without browser bloat.
          </p>

          {/* Quick Install Bar & GitHub Star */}
          <div className="flex flex-col sm:flex-row items-center justify-center gap-3 max-w-lg mx-auto mb-10">
            <div className="flex items-center justify-between w-full px-3.5 py-2.5 rounded bg-zinc-900 border border-zinc-800 text-xs font-mono text-zinc-300">
              <span className="flex items-center gap-2 truncate">
                <span className="text-zinc-500">$</span>
                <span className="truncate">{installCmd}</span>
              </span>
              <button
                onClick={copyInstall}
                className="ml-2 text-zinc-400 hover:text-white transition-colors"
                title="Copy install command"
              >
                {copiedInstall ? <Check className="w-4 h-4 text-zinc-200" /> : <Copy className="w-4 h-4" />}
              </button>
            </div>
            <div className="flex items-center gap-2 w-full sm:w-auto">
              <button
                onClick={() => onGoToDocs()}
                className="flex-1 sm:flex-none whitespace-nowrap inline-flex items-center justify-center gap-1.5 px-4 py-2.5 rounded bg-white hover:bg-zinc-200 text-black text-xs font-semibold transition-colors"
              >
                <span>Read Docs</span>
                <ArrowRight className="w-3.5 h-3.5" />
              </button>
              <a
                href="https://github.com/chinmay-sawant/blinkless"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1.5 px-3 py-2.5 rounded bg-zinc-900 hover:bg-zinc-850 text-zinc-300 border border-zinc-800 text-xs font-mono transition-colors"
                title="Star repository on GitHub"
              >
                <GitBranch className="w-3.5 h-3.5 text-zinc-400" />
                <Star className="w-3 h-3 text-amber-400 fill-amber-400" />
                <span>1</span>
              </a>
            </div>
          </div>

          {/* Key Specs Row */}
          <div className="flex flex-wrap items-center justify-center gap-6 sm:gap-10 text-xs font-mono text-zinc-400 pt-6 border-t border-zinc-900">
            <div>
              <span className="text-white font-semibold">100%</span> Pure Go
            </div>
            <div>
              <span className="text-white font-semibold">0 MB</span> JavaScript
            </div>
            <div>
              <span className="text-white font-semibold">{ENGINE_STATS.propertiesCount}+</span> CSS Properties
            </div>
            <div>
              <span className="text-white font-semibold">2</span> Direct Dependencies
            </div>
          </div>
        </div>
      </section>

      {/* 4 Feature Pillars */}
      <section className="py-20 px-4 sm:px-6 lg:px-8 border-b border-zinc-800/80 max-w-7xl mx-auto">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="p-6 rounded bg-zinc-900/60 border border-zinc-800/80">
            <div className="w-8 h-8 rounded bg-zinc-800 border border-zinc-700/80 flex items-center justify-center text-zinc-200 mb-4">
              <Zap className="w-4 h-4" />
            </div>
            <h3 className="text-sm font-semibold text-white mb-2">Zero JavaScript Engine</h3>
            <p className="text-xs text-zinc-400 leading-relaxed">
              No V8 runtime, no JIT compiler, and no DOM microtask loops. Inert script tags are discarded cleanly during HTML parsing.
            </p>
          </div>

          <div className="p-6 rounded bg-zinc-900/60 border border-zinc-800/80">
            <div className="w-8 h-8 rounded bg-zinc-800 border border-zinc-700/80 flex items-center justify-center text-zinc-200 mb-4">
              <Cpu className="w-4 h-4" />
            </div>
            <h3 className="text-sm font-semibold text-white mb-2">100% Pure Go & CGO-Free</h3>
            <p className="text-xs text-zinc-400 leading-relaxed">
              Builds with standard go build ./... in seconds. Embeds directly into any Go backend service, microservice, or CLI without C++ toolchains.
            </p>
          </div>

          <div className="p-6 rounded bg-zinc-900/60 border border-zinc-800/80">
            <div className="w-8 h-8 rounded bg-zinc-800 border border-zinc-700/80 flex items-center justify-center text-zinc-200 mb-4">
              <Shield className="w-4 h-4" />
            </div>
            <h3 className="text-sm font-semibold text-white mb-2">Bounded Memory Safety</h3>
            <p className="text-xs text-zinc-400 leading-relaxed">
              Go memory-safe runtime with rigid depth limits (1024 DOM depth, 32 MiB max CSS) protecting against denial-of-service memory spikes.
            </p>
          </div>

          <div className="p-6 rounded bg-zinc-900/60 border border-zinc-800/80">
            <div className="w-8 h-8 rounded bg-zinc-800 border border-zinc-700/80 flex items-center justify-center text-zinc-200 mb-4">
              <Layers className="w-4 h-4" />
            </div>
            <h3 className="text-sm font-semibold text-white mb-2">layout.DisplayList Output</h3>
            <p className="text-xs text-zinc-400 leading-relaxed">
              Emits high-level vector drawing primitives (rectangles, stroked lines, text runs, images) in point coordinates with device-pixel snapping.
            </p>
          </div>
        </div>
      </section>

      {/* Chromium Blink Comparison Section */}
      <section className="py-20 px-4 sm:px-6 lg:px-8 border-b border-zinc-800/80 max-w-7xl mx-auto">
        <div className="max-w-3xl mb-12">
          <h2 className="text-2xl sm:text-3xl font-bold text-white tracking-tight mb-3">
            Chromium Blink vs Blinkless
          </h2>
          <p className="text-sm text-zinc-400 leading-relaxed">
            Chromium's Blink is designed for full interactive web browsers. Blinkless is focused on server-side document layout, headless rendering, and programmatic drawing lists.
          </p>
        </div>

        <div className="rounded border border-zinc-800 bg-zinc-950 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse text-xs sm:text-sm">
              <thead>
                <tr className="bg-zinc-900/80 border-b border-zinc-800 text-xs font-mono uppercase text-zinc-400">
                  <th className="py-3 px-5 font-semibold w-1/4">Dimension</th>
                  <th className="py-3 px-5 font-semibold w-3/8 text-zinc-400">Google Chrome (Blink)</th>
                  <th className="py-3 px-5 font-semibold w-3/8 text-white">Blinkless (Pure Go)</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-800/60">
                {BLINK_COMPARISONS.map((row, idx) => (
                  <tr key={idx} className="hover:bg-zinc-900/40 transition-colors">
                    <td className="py-3.5 px-5 font-medium text-white align-top">
                      <div className="font-semibold text-zinc-200">{row.title}</div>
                      <div className="text-[11px] text-zinc-500 font-mono mt-0.5">{row.advantage}</div>
                    </td>
                    <td className="py-3.5 px-5 text-zinc-400 align-top text-xs leading-relaxed">
                      {row.blink}
                    </td>
                    <td className="py-3.5 px-5 text-zinc-200 align-top text-xs font-medium leading-relaxed bg-zinc-900/20">
                      {row.blinkless}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </section>

      {/* Code Snippet Preview with Syntax Highlighting */}
      <section className="py-20 px-4 sm:px-6 lg:px-8 max-w-7xl mx-auto">
        <div className="max-w-3xl mb-8">
          <h2 className="text-2xl sm:text-3xl font-bold text-white tracking-tight mb-3">
            Minimal Go Integration
          </h2>
          <p className="text-sm text-zinc-400 leading-relaxed">
            Three function calls to parse HTML, cascade stylesheets, and generate a layout drawing list.
          </p>
        </div>

        <div className="max-w-4xl">
          <CodeBlock code={sampleGoCode} filename="example_test.go" language="go" />
        </div>

        {/* Bottom Documentation CTA */}
        <div className="mt-12 pt-8 border-t border-zinc-800/80 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <div>
            <h4 className="text-sm font-semibold text-white">Looking for the full CSS reference?</h4>
            <p className="text-xs text-zinc-400 mt-0.5">
              Browse all {ENGINE_STATS.propertiesCount} implemented CSS properties, selectors, and API details.
            </p>
          </div>
          <button
            onClick={() => onGoToDocs('properties')}
            className="inline-flex items-center gap-1.5 px-4 py-2 rounded bg-zinc-800 hover:bg-zinc-700 text-white text-xs font-medium transition-colors"
          >
            <span>Open CSS Reference</span>
            <ArrowRight className="w-3.5 h-3.5" />
          </button>
        </div>
      </section>
    </div>
  );
};
