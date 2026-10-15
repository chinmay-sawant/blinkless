import React, { useState, useMemo } from 'react';
import { IMPLEMENTED_PROPERTIES, CSS_CATEGORIES, BLINK_COMPARISONS, ENGINE_STATS } from '../data/properties';
import { Search } from 'lucide-react';
import { CodeBlock } from './CodeBlock';

interface DocumentationProps {
  initialSection?: string;
}

export const Documentation: React.FC<DocumentationProps> = ({ initialSection = 'overview' }) => {
  const [activeSection, setActiveSection] = useState<string>(initialSection);
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [selectedCategory, setSelectedCategory] = useState<string>('All');

  const filteredProperties = useMemo(() => {
    return IMPLEMENTED_PROPERTIES.filter((prop) => {
      const matchesCategory = selectedCategory === 'All' || prop.category === selectedCategory;
      const matchesSearch =
        prop.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        prop.description.toLowerCase().includes(searchQuery.toLowerCase());
      return matchesCategory && matchesSearch;
    });
  }, [selectedCategory, searchQuery]);

  const quickstartGoCode = `package main

import (
	"context"
	"fmt"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

func main() {
	ctx := context.Background()

	// 1. Parse WHATWG HTML
	doc, err := html.Parse([]byte(\`<h1 style="color: black;">Hello World</h1>\`))
	if err != nil {
		panic(err)
	}

	// 2. Cascade Styles with Canvas Bounds
	styled, err := css.Apply(ctx, doc, css.Options{
		WidthPx:  800,
		HeightPx: 600,
	})
	if err != nil {
		panic(err)
	}

	// 3. Emit Vector Display List
	display, err := layout.DisplayList(ctx, styled)
	if err != nil {
		panic(err)
	}

	// 4. Iterate over display.Order to paint in stacking context order
	for _, idx := range display.Order {
		op := display.Ops[idx]
		fmt.Printf("Draw op: kind=%v at (%.1f, %.1f)\\n", op.Kind, op.X, op.Y)
	}
}`;

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="flex flex-col lg:flex-row gap-8 items-start">
        {/* Left Sidebar */}
        <aside className="w-full lg:w-64 shrink-0 bg-zinc-950 lg:sticky lg:top-20 border border-zinc-800/80 rounded p-4 text-xs font-medium">
          <div className="space-y-6">
            <div>
              <div className="text-[11px] font-mono uppercase tracking-wider text-zinc-500 mb-2 font-semibold">
                Getting Started
              </div>
              <ul className="space-y-1">
                <li>
                  <button
                    onClick={() => setActiveSection('overview')}
                    className={`w-full text-left px-2.5 py-1.5 rounded transition-colors ${
                      activeSection === 'overview'
                        ? 'bg-zinc-800 text-white font-semibold'
                        : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'
                    }`}
                  >
                    Overview
                  </button>
                </li>
                <li>
                  <button
                    onClick={() => setActiveSection('quickstart')}
                    className={`w-full text-left px-2.5 py-1.5 rounded transition-colors ${
                      activeSection === 'quickstart'
                        ? 'bg-zinc-800 text-white font-semibold'
                        : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'
                    }`}
                  >
                    Quickstart
                  </button>
                </li>
                <li>
                  <button
                    onClick={() => setActiveSection('pipeline')}
                    className={`w-full text-left px-2.5 py-1.5 rounded transition-colors ${
                      activeSection === 'pipeline'
                        ? 'bg-zinc-800 text-white font-semibold'
                        : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'
                    }`}
                  >
                    Pipeline & Architecture
                  </button>
                </li>
              </ul>
            </div>

            <div>
              <div className="text-[11px] font-mono uppercase tracking-wider text-zinc-500 mb-2 font-semibold">
                Comparison
              </div>
              <ul className="space-y-1">
                <li>
                  <button
                    onClick={() => setActiveSection('comparison')}
                    className={`w-full text-left px-2.5 py-1.5 rounded transition-colors ${
                      activeSection === 'comparison'
                        ? 'bg-zinc-800 text-white font-semibold'
                        : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'
                    }`}
                  >
                    Blink vs Blinkless
                  </button>
                </li>
              </ul>
            </div>

            <div>
              <div className="text-[11px] font-mono uppercase tracking-wider text-zinc-500 mb-2 font-semibold">
                CSS Reference
              </div>
              <ul className="space-y-1">
                <li>
                  <button
                    onClick={() => setActiveSection('properties')}
                    className={`w-full text-left px-2.5 py-1.5 rounded transition-colors flex items-center justify-between ${
                      activeSection === 'properties'
                        ? 'bg-zinc-800 text-white font-semibold'
                        : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'
                    }`}
                  >
                    <span>Supported Properties</span>
                    <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-zinc-900 text-zinc-400 border border-zinc-800">
                      {ENGINE_STATS.propertiesCount}
                    </span>
                  </button>
                </li>
                <li>
                  <button
                    onClick={() => setActiveSection('selectors')}
                    className={`w-full text-left px-2.5 py-1.5 rounded transition-colors ${
                      activeSection === 'selectors'
                        ? 'bg-zinc-800 text-white font-semibold'
                        : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'
                    }`}
                  >
                    Selectors & Specificity
                  </button>
                </li>
              </ul>
            </div>

            <div>
              <div className="text-[11px] font-mono uppercase tracking-wider text-zinc-500 mb-2 font-semibold">
                Go API Reference
              </div>
              <ul className="space-y-1">
                <li>
                  <button
                    onClick={() => setActiveSection('api')}
                    className={`w-full text-left px-2.5 py-1.5 rounded transition-colors ${
                      activeSection === 'api'
                        ? 'bg-zinc-800 text-white font-semibold'
                        : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'
                    }`}
                  >
                    Public API Surface
                  </button>
                </li>
              </ul>
            </div>
          </div>
        </aside>

        {/* Right Main Content Pane */}
        <main className="flex-1 min-w-0 bg-zinc-950 border border-zinc-800/80 rounded p-6 sm:p-8">
          {/* Section: Overview */}
          {activeSection === 'overview' && (
            <div className="space-y-6">
              <div className="border-b border-zinc-800 pb-4">
                <span className="text-xs font-mono text-zinc-400 uppercase">Getting Started</span>
                <h1 className="text-2xl sm:text-3xl font-bold text-white tracking-tight mt-1">Overview</h1>
              </div>

              <p className="text-sm text-zinc-300 leading-relaxed">
                <strong className="text-white">Blinkless</strong> is an embeddable, pure-Go HTML and CSS layout engine. HTML and CSS are its inputs; its output is a structured <code className="text-zinc-200 font-mono bg-zinc-900 px-1.5 py-0.5 rounded border border-zinc-800">layout.DisplayList</code> vector drawing list.
              </p>

              <p className="text-sm text-zinc-400 leading-relaxed">
                Unlike full browser engines such as Google Chrome's Blink, Blinkless does not execute scripts, does not run an event loop, does not talk to a GPU, and does not require CGO. It is designed specifically for backend document generation, print layout, and headless vector drawing operations.
              </p>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 my-6">
                <div className="p-4 rounded bg-zinc-900/60 border border-zinc-800">
                  <div className="font-semibold text-sm text-white mb-1">Module Path</div>
                  <div className="text-xs font-mono text-zinc-400">{ENGINE_STATS.modulePath}</div>
                </div>
                <div className="p-4 rounded bg-zinc-900/60 border border-zinc-800">
                  <div className="font-semibold text-sm text-white mb-1">Dependencies</div>
                  <div className="text-xs font-mono text-zinc-400">Strictly 2 direct modules (go-text, canvas)</div>
                </div>
              </div>

              <h3 className="text-base font-semibold text-white mt-8 mb-2">Package Structure</h3>
              <p className="text-xs text-zinc-400 mb-4">The repository is divided into clean public facades and internal implementations:</p>
              <ul className="space-y-2 text-xs text-zinc-300 font-mono">
                <li className="p-2.5 rounded bg-zinc-900/40 border border-zinc-800 flex items-center justify-between">
                  <span>github.com/chinmay-sawant/blinkless/html</span>
                  <span className="text-zinc-500 font-sans">WHATWG HTML parser</span>
                </li>
                <li className="p-2.5 rounded bg-zinc-900/40 border border-zinc-800 flex items-center justify-between">
                  <span>github.com/chinmay-sawant/blinkless/css</span>
                  <span className="text-zinc-500 font-sans">CSS parser, cascade, and selector matching</span>
                </li>
                <li className="p-2.5 rounded bg-zinc-900/40 border border-zinc-800 flex items-center justify-between">
                  <span>github.com/chinmay-sawant/blinkless/layout</span>
                  <span className="text-zinc-500 font-sans">Box placement and layout.DisplayList</span>
                </li>
              </ul>
            </div>
          )}

          {/* Section: Quickstart with Highlighted Code */}
          {activeSection === 'quickstart' && (
            <div className="space-y-6">
              <div className="border-b border-zinc-800 pb-4">
                <span className="text-xs font-mono text-zinc-400 uppercase">Getting Started</span>
                <h1 className="text-2xl sm:text-3xl font-bold text-white tracking-tight mt-1">Quickstart</h1>
              </div>

              <h3 className="text-sm font-semibold text-white">1. Installation</h3>
              <CodeBlock code="$ go get github.com/chinmay-sawant/blinkless" filename="terminal" language="bash" />

              <h3 className="text-sm font-semibold text-white mt-6">2. Basic Usage</h3>
              <p className="text-xs text-zinc-400">Parse HTML, cascade stylesheets, and produce a vector display list:</p>
              <CodeBlock code={quickstartGoCode} filename="main.go" language="go" />
            </div>
          )}

          {/* Section: Pipeline */}
          {activeSection === 'pipeline' && (
            <div className="space-y-6">
              <div className="border-b border-zinc-800 pb-4">
                <span className="text-xs font-mono text-zinc-400 uppercase">Architecture</span>
                <h1 className="text-2xl sm:text-3xl font-bold text-white tracking-tight mt-1">Pipeline & Architecture</h1>
              </div>

              <p className="text-sm text-zinc-300 leading-relaxed">
                The execution pipeline follows four clean, sequential stages:
              </p>

              <div className="space-y-4">
                <div className="p-4 rounded bg-zinc-900/60 border border-zinc-800">
                  <div className="font-semibold text-sm text-white">1. Load (internal/load, internal/convert/prepare)</div>
                  <p className="text-xs text-zinc-400 mt-1">
                    Fetches document bytes and subresources. Enforces security policies: default-deny local file ACLs, SSRF-restricted network policies, and a rigid 100 MiB body size cap.
                  </p>
                </div>

                <div className="p-4 rounded bg-zinc-900/60 border border-zinc-800">
                  <div className="font-semibold text-sm text-white">2. HTML Parse (html, internal/html)</div>
                  <p className="text-xs text-zinc-400 mt-1">
                    WHATWG compliant tokenizer and tree construction. Element nesting depth is capped at 1024 levels. Script elements are retained as inert text without script execution.
                  </p>
                </div>

                <div className="p-4 rounded bg-zinc-900/60 border border-zinc-800">
                  <div className="font-semibold text-sm text-white">3. CSS Cascade (css, internal/css)</div>
                  <p className="text-xs text-zinc-400 mt-1">
                    Right-to-left selector matching, specificity sorting (a, b, c), cascade layers (@layer), container queries (@container), and custom properties (var() and @property).
                  </p>
                </div>

                <div className="p-4 rounded bg-zinc-900/60 border border-zinc-800">
                  <div className="font-semibold text-sm text-white">4. Layout Drawing List (layout, internal/layout)</div>
                  <p className="text-xs text-zinc-400 mt-1">
                    Builds box tree for block, inline, flexbox, grid, and tables. Performs OpenType text shaping with HarfBuzz via go-text/typesetting, Chrome margin collapsing, and emits layout.DisplayList.
                  </p>
                </div>
              </div>
            </div>
          )}

          {/* Section: Comparison */}
          {activeSection === 'comparison' && (
            <div className="space-y-6">
              <div className="border-b border-zinc-800 pb-4">
                <span className="text-xs font-mono text-zinc-400 uppercase">Engine Contrast</span>
                <h1 className="text-2xl sm:text-3xl font-bold text-white tracking-tight mt-1">Blink vs Blinkless</h1>
              </div>

              <div className="rounded border border-zinc-800 overflow-hidden">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr className="bg-zinc-900 border-b border-zinc-800 text-zinc-400 font-mono">
                      <th className="p-3 w-1/4">Dimension</th>
                      <th className="p-3 w-3/8 text-zinc-400">Google Chrome (Blink)</th>
                      <th className="p-3 w-3/8 text-white">Blinkless (Pure Go)</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800/60">
                    {BLINK_COMPARISONS.map((row, idx) => (
                      <tr key={idx} className="hover:bg-zinc-900/30">
                        <td className="p-3 font-medium text-white align-top">
                          <div>{row.title}</div>
                          <div className="text-[10px] text-zinc-500 font-mono mt-0.5">{row.advantage}</div>
                        </td>
                        <td className="p-3 text-zinc-400 align-top leading-relaxed">{row.blink}</td>
                        <td className="p-3 text-zinc-200 align-top font-medium leading-relaxed bg-zinc-900/10">
                          {row.blinkless}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Section: Supported CSS Properties */}
          {activeSection === 'properties' && (
            <div className="space-y-6">
              <div className="border-b border-zinc-800 pb-4">
                <span className="text-xs font-mono text-zinc-400 uppercase">CSS Reference</span>
                <h1 className="text-2xl sm:text-3xl font-bold text-white tracking-tight mt-1">Supported Properties</h1>
                <p className="text-xs text-zinc-400 mt-1">
                  Blinkless implements {ENGINE_STATS.propertiesCount} CSS properties covering layout, typography, borders, and colors.
                </p>
              </div>

              {/* Search and Category Filters */}
              <div className="space-y-3">
                <div className="relative">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500" />
                  <input
                    type="text"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder="Search property (e.g. width, flex, grid, margin)..."
                    className="w-full pl-9 pr-4 py-2 rounded bg-zinc-900 border border-zinc-800 focus:border-zinc-600 focus:outline-none text-xs text-zinc-200 placeholder-zinc-500 font-mono"
                  />
                </div>

                <div className="flex items-center gap-1.5 overflow-x-auto pb-2 no-scrollbar">
                  {CSS_CATEGORIES.map((category) => (
                    <button
                      key={category}
                      onClick={() => setSelectedCategory(category)}
                      className={`whitespace-nowrap px-2.5 py-1 rounded text-xs transition-colors ${
                        selectedCategory === category
                          ? 'bg-zinc-700 text-white font-medium'
                          : 'bg-zinc-900 hover:bg-zinc-800 text-zinc-400 border border-zinc-800'
                      }`}
                    >
                      {category}
                    </button>
                  ))}
                </div>
              </div>

              {/* Properties Grid */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2">
                {filteredProperties.map((prop, idx) => (
                  <div key={idx} className="p-3 rounded bg-zinc-900/50 border border-zinc-800">
                    <div className="flex items-center justify-between gap-2 mb-1.5">
                      <span className="font-mono text-xs font-semibold text-white bg-zinc-900 px-2 py-0.5 rounded border border-zinc-800">
                        {prop.name}
                      </span>
                      <span className="text-[10px] font-mono text-zinc-400">
                        {prop.status}
                      </span>
                    </div>
                    <p className="text-xs text-zinc-400 leading-relaxed mt-1">
                      {prop.description}
                    </p>
                    <div className="text-[10px] text-zinc-500 font-mono mt-2 pt-1.5 border-t border-zinc-800/60">
                      {prop.category}
                    </div>
                  </div>
                ))}
              </div>

              {filteredProperties.length === 0 && (
                <div className="text-center py-10 text-xs text-zinc-500 font-mono">
                  No matching properties found.
                </div>
              )}
            </div>
          )}

          {/* Section: Selectors & Specificity */}
          {activeSection === 'selectors' && (
            <div className="space-y-6">
              <div className="border-b border-zinc-800 pb-4">
                <span className="text-xs font-mono text-zinc-400 uppercase">CSS Reference</span>
                <h1 className="text-2xl sm:text-3xl font-bold text-white tracking-tight mt-1">Selectors & Specificity</h1>
              </div>

              <h3 className="text-sm font-semibold text-white">Specificity Tuple (a, b, c)</h3>
              <p className="text-xs text-zinc-400 leading-relaxed">
                Selector specificity is computed as a three-component tuple matching the WHATWG and W3C CSS Selectors Level 4 specification:
              </p>
              <ul className="space-y-1 text-xs text-zinc-300 font-mono list-disc list-inside">
                <li><strong>a:</strong> Count of ID selectors (#id)</li>
                <li><strong>b:</strong> Count of class (.class), attribute ([attr]), and pseudo-classes</li>
                <li><strong>c:</strong> Count of type selectors (tag) and pseudo-elements (::before, ::after)</li>
              </ul>

              <h3 className="text-sm font-semibold text-white mt-6">Supported Pseudo-classes & Pseudos</h3>
              <div className="space-y-2 text-xs text-zinc-300 font-mono">
                <div className="p-2.5 rounded bg-zinc-900/60 border border-zinc-800">
                  <div className="text-white font-semibold mb-0.5">:has(relative_selector)</div>
                  <div className="text-zinc-400 font-sans text-[11px]">Relational selector with depth inspection budget.</div>
                </div>
                <div className="p-2.5 rounded bg-zinc-900/60 border border-zinc-800">
                  <div className="text-white font-semibold mb-0.5">:nth-child(an+b), :first-child, :last-child</div>
                  <div className="text-zinc-400 font-sans text-[11px]">Structural tree matching with formulas and keywords (odd, even).</div>
                </div>
                <div className="p-2.5 rounded bg-zinc-900/60 border border-zinc-800">
                  <div className="text-white font-semibold mb-0.5">:is(), :where(), :not()</div>
                  <div className="text-zinc-400 font-sans text-[11px]">Functional logical grouping selectors.</div>
                </div>
                <div className="p-2.5 rounded bg-zinc-900/60 border border-zinc-800">
                  <div className="text-white font-semibold mb-0.5">::before, ::after</div>
                  <div className="text-zinc-400 font-sans text-[11px]">Generated pseudo-elements with content support.</div>
                </div>
              </div>
            </div>
          )}

          {/* Section: Public API */}
          {activeSection === 'api' && (
            <div className="space-y-6">
              <div className="border-b border-zinc-800 pb-4">
                <span className="text-xs font-mono text-zinc-400 uppercase">Go API</span>
                <h1 className="text-2xl sm:text-3xl font-bold text-white tracking-tight mt-1">Public API Surface</h1>
              </div>

              <div className="space-y-4">
                <div className="p-4 rounded bg-zinc-900/60 border border-zinc-800">
                  <div className="font-mono text-xs text-white font-semibold mb-1">
                    html.Parse(source []byte) (*html.Document, error)
                  </div>
                  <p className="text-xs text-zinc-400">
                    Parses UTF-8 HTML byte slice into an in-memory document tree.
                  </p>
                </div>

                <div className="p-4 rounded bg-zinc-900/60 border border-zinc-800">
                  <div className="font-mono text-xs text-white font-semibold mb-1">
                    css.Apply(ctx context.Context, doc *html.Document, opts css.Options) (*css.Document, error)
                  </div>
                  <p className="text-xs text-zinc-400">
                    Resolves stylesheets, media queries, and property cascade across the DOM tree. Requires WidthPx and HeightPx &gt; 0.
                  </p>
                </div>

                <div className="p-4 rounded bg-zinc-900/60 border border-zinc-800">
                  <div className="font-mono text-xs text-white font-semibold mb-1">
                    layout.DisplayList(ctx context.Context, doc *css.Document) (*layout.Display, error)
                  </div>
                  <p className="text-xs text-zinc-400">
                    Computes box tree placement and returns a *layout.Display containing Ops []DisplayOp and Order []int.
                  </p>
                </div>
              </div>
            </div>
          )}
        </main>
      </div>
    </div>
  );
};
