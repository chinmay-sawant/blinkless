import React, { useEffect, useMemo, useState } from 'react';
import { BLINK_COMPARISONS, ENGINE_STATS } from '../data/properties';
import catalog from '../data/catalog.json';
import { ApiReference } from './ApiReference';
import { CodeBlock } from './CodeBlock';
import { isDocSection, type DocSection } from '../route';

interface DocumentationProps {
  initialSection?: string;
  onSectionChange: (section: DocSection) => void;
}

type CatalogStatus = 'all' | 'implemented' | 'partial';

const catalogRows = catalog.properties;
const implementedCount = catalogRows.filter((row) => row.status === 'implemented').length;
const partialCount = catalogRows.filter((row) => row.status === 'partial').length;

const NAV_GROUPS: { label: string; items: { id: string; label: string }[] }[] = [
  {
    label: 'Getting started',
    items: [
      { id: 'overview', label: 'Overview' },
      { id: 'quickstart', label: 'Quickstart' },
      { id: 'pipeline', label: 'Pipeline' },
    ],
  },
  {
    label: 'Comparison',
    items: [{ id: 'comparison', label: 'Blink vs blinkless' }],
  },
  {
    label: 'CSS',
    items: [
      { id: 'properties', label: 'CSS catalog' },
      { id: 'selectors', label: 'Selectors' },
    ],
  },
  {
    label: 'Go API',
    items: [{ id: 'api', label: 'API' }],
  },
];

const PACKAGES = [
  { path: 'github.com/chinmay-sawant/blinkless/html', detail: 'HTML parser' },
  { path: 'github.com/chinmay-sawant/blinkless/css', detail: 'cascade and selectors' },
  { path: 'github.com/chinmay-sawant/blinkless/layout', detail: 'placement and the drawing list' },
];

const inlineCode = 'font-mono text-[0.9em] bg-paper-deep px-1 py-0.5 rounded-sm text-ink';

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

	// Parse HTML into a document tree.
	doc, err := html.Parse([]byte("<h1 style='color: black;'>Hello World</h1>"))
	if err != nil {
		panic(err)
	}

	// Apply styles. WidthPx and HeightPx must be greater than 0.
	styled, err := css.Apply(ctx, doc, css.Options{
		WidthPx:  800,
		HeightPx: 600,
	})
	if err != nil {
		panic(err)
	}

	// Place the tree and return the drawing list.
	display, err := layout.DisplayList(ctx, styled)
	if err != nil {
		panic(err)
	}

	// Walk display.Order and read each op.
	for _, idx := range display.Order {
		op := display.Ops[idx]
		fmt.Println(op.Kind, op.X, op.Y)
	}
}`;

function navButtonClass(isActive: boolean, full: boolean): string {
  const state = isActive ? 'text-ink bg-paper-deep' : 'text-muted hover:text-ink';
  const width = full ? 'w-full' : 'whitespace-nowrap';
  return `${width} text-left px-3 min-h-11 text-sm rounded-sm cursor-pointer flex items-center ${state}`;
}

function SectionHeader({ kicker, title }: { kicker: string; title: string }) {
  return (
    <header className="border-b border-line pb-4">
      <p className="font-mono text-[11px] tracking-wide text-faint">{kicker}</p>
      <h1 className="font-display text-4xl text-ink tracking-tight mt-2">{title}</h1>
    </header>
  );
}

export const Documentation: React.FC<DocumentationProps> = ({ initialSection = 'overview', onSectionChange }) => {
  const [activeSection, setActiveSection] = useState<string>(initialSection);
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [statusFilter, setStatusFilter] = useState<CatalogStatus>('all');

  useEffect(() => {
    setActiveSection(initialSection);
  }, [initialSection]);

  const chooseSection = (id: string) => {
    setActiveSection(id);
    if (isDocSection(id)) {
      onSectionChange(id);
    }
  };

  const filteredProperties = useMemo(() => {
    const query = searchQuery.toLowerCase();
    return catalogRows.filter((prop) => {
      const matchesStatus = statusFilter === 'all' || prop.status === statusFilter;
      const haystack = `${prop.name} ${prop.values ?? ''} ${prop.note ?? ''}`.toLowerCase();
      return matchesStatus && haystack.includes(query);
    });
  }, [statusFilter, searchQuery]);

  return (
    <div className="max-w-6xl mx-auto px-4 sm:px-6 py-10 font-sans">
      <div className="grid grid-cols-1 lg:grid-cols-[220px_minmax(0,1fr)] gap-10 items-start">
        <aside className="w-full lg:sticky lg:top-20 self-start">
          <nav aria-label="Documentation sections">
            <ul className="flex flex-wrap gap-2 lg:hidden">
              {NAV_GROUPS.flatMap((group) => group.items).map((item) => (
                <li key={item.id} className="shrink-0">
                  <button
                    type="button"
                    onClick={() => chooseSection(item.id)}
                    className={navButtonClass(activeSection === item.id, false)}
                  >
                    {item.label}
                  </button>
                </li>
              ))}
            </ul>
            <div className="hidden lg:block space-y-8">
              {NAV_GROUPS.map((group) => (
                <div key={group.label}>
                  <div className="font-mono text-[11px] tracking-wide text-faint mb-2">{group.label}</div>
                  <ul className="space-y-1">
                    {group.items.map((item) => (
                      <li key={item.id}>
                        <button
                          type="button"
                          onClick={() => chooseSection(item.id)}
                          className={navButtonClass(activeSection === item.id, true)}
                        >
                          {item.label}
                        </button>
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          </nav>
        </aside>

        <main className="min-w-0 max-w-none">
          {activeSection === 'overview' && (
            <div className="space-y-6">
              <SectionHeader kicker="Getting started" title="Overview" />
              <p className="text-base text-muted leading-relaxed max-w-2xl">
                Blinkless reads HTML and CSS and returns a drawing list. The list is{' '}
                <code className={inlineCode}>layout.DisplayList</code>. The host paints the pixels.
              </p>
              <p className="text-base text-muted leading-relaxed max-w-2xl">
                Script in the document is not run. Nothing writes a PDF, and nothing encodes the page as a PNG or JPEG. Your program paints the operations. The default build sets CGO_ENABLED=0.
              </p>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div className="bg-sheet border border-line rounded-md p-4">
                  <h2 className="font-sans font-medium text-ink text-lg">Module path</h2>
                  <p className="font-mono text-sm text-ink mt-2 break-all">{ENGINE_STATS.modulePath}</p>
                </div>
                <div className="bg-sheet border border-line rounded-md p-4">
                  <h2 className="font-sans font-medium text-ink text-lg">Dependencies</h2>
                  <p className="font-mono text-sm text-ink mt-2">go-text/typesetting</p>
                  <p className="font-mono text-sm text-ink">tdewolff/canvas</p>
                </div>
              </div>
              <h2 className="font-sans font-medium text-ink text-lg">Packages</h2>
              <ul>
                {PACKAGES.map((pkg) => (
                  <li key={pkg.path} className="border-b border-line py-3">
                    <div className="font-mono text-sm text-ink break-all">{pkg.path}</div>
                    <p className="text-muted text-sm mt-1">{pkg.detail}</p>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {activeSection === 'quickstart' && (
            <div className="space-y-6">
              <SectionHeader kicker="Getting started" title="Quickstart" />
              <p className="text-base text-muted leading-relaxed max-w-2xl">
                Parse, apply, then walk <code className={inlineCode}>display.Order</code>.
              </p>
              <h2 className="font-sans font-medium text-ink text-lg">Install</h2>
              <CodeBlock
                code="go get github.com/chinmay-sawant/blinkless"
                filename="terminal"
                language="bash"
              />
              <h2 className="font-sans font-medium text-ink text-lg">Sample</h2>
              <CodeBlock code={quickstartGoCode} filename="main.go" language="go" />
            </div>
          )}

          {activeSection === 'pipeline' && (
            <div className="space-y-6">
              <SectionHeader kicker="Getting started" title="Pipeline" />
              <p className="text-base text-muted leading-relaxed max-w-2xl">Four steps, in order.</p>
              <div className="space-y-4">
                <div className="bg-sheet border border-line rounded-md p-4">
                  <h2 className="font-sans font-medium text-ink text-lg">1. Load</h2>
                  <p className="text-muted text-sm mt-2 leading-relaxed">
                    Fetches the document and subresources when a document asks for them. Local file access is off unless the caller turns it on.
                  </p>
                </div>
                <div className="bg-sheet border border-line rounded-md p-4">
                  <h2 className="font-sans font-medium text-ink text-lg">2. HTML parse</h2>
                  <p className="text-muted text-sm mt-2 leading-relaxed">
                    Builds the tree. Nesting stops at 1024. Script is not executed.
                  </p>
                </div>
                <div className="bg-sheet border border-line rounded-md p-4">
                  <h2 className="font-sans font-medium text-ink text-lg">3. CSS cascade</h2>
                  <p className="text-muted text-sm mt-2 leading-relaxed">
                    Selector matching, specificity, layers, custom properties. A stylesheet over 32 MiB is rejected.
                  </p>
                </div>
                <div className="bg-sheet border border-line rounded-md p-4">
                  <h2 className="font-sans font-medium text-ink text-lg">4. Drawing list</h2>
                  <p className="text-muted text-sm mt-2 leading-relaxed">
                    Places block, inline, flex, grid, and table boxes. Text is shaped with go-text/typesetting. The result is{' '}
                    <code className={inlineCode}>layout.DisplayList</code>.
                  </p>
                </div>
              </div>
            </div>
          )}

          {activeSection === 'comparison' && (
            <div className="space-y-6">
              <SectionHeader kicker="Comparison" title="Blink vs blinkless" />
              <div className="overflow-x-auto">
                <table className="w-full min-w-[720px] text-left border-collapse">
                  <thead>
                    <tr className="border-b border-line text-faint text-xs font-sans">
                      <th className="p-3 font-medium text-left">Dimension</th>
                      <th className="p-3 font-medium text-left">Blink</th>
                      <th className="p-3 font-medium text-left">blinkless</th>
                    </tr>
                  </thead>
                  <tbody>
                    {BLINK_COMPARISONS.map((row) => (
                      <tr key={row.title} className="border-b border-line">
                        <td className="p-3 align-top">
                          <div className="text-sm text-ink">{row.title}</div>
                          <div className="text-xs text-faint mt-1">{row.advantage}</div>
                        </td>
                        <td className="p-3 align-top text-sm text-muted leading-relaxed">{row.blink}</td>
                        <td className="p-3 align-top text-sm text-ink leading-relaxed">{row.blinkless}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {activeSection === 'properties' && (
            <div className="space-y-6">
              <SectionHeader kicker="CSS" title="CSS catalog" />
              <p className="text-base text-muted leading-relaxed max-w-2xl">
                {implementedCount} implemented and {partialCount} partial. This list is those {catalogRows.length} names, from {catalog.source}, updated {catalog.updated}.
              </p>
              <div className="space-y-3">
                <div>
                  <label htmlFor="property-search" className="sr-only">
                    Search properties
                  </label>
                  <input
                    id="property-search"
                    type="search"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder="Name, value, or note"
                    className="min-h-11 w-full bg-sheet border border-line rounded px-3 text-sm text-ink placeholder:text-faint font-sans"
                  />
                </div>
                <div className="flex flex-wrap gap-2">
                  {(['all', 'implemented', 'partial'] as const).map((status) => {
                    const isActive = statusFilter === status;
                    const label = status === 'all' ? `All ${catalogRows.length}` : status === 'implemented' ? `Implemented ${implementedCount}` : `Partial ${partialCount}`;
                    return (
                      <button
                        key={status}
                        type="button"
                        onClick={() => setStatusFilter(status)}
                        className={`inline-flex items-center min-h-11 px-3 text-sm rounded-sm border border-line cursor-pointer ${
                          isActive ? 'bg-ink text-paper' : 'bg-sheet text-muted hover:text-ink'
                        }`}
                      >
                        {label}
                      </button>
                    );
                  })}
                </div>
              </div>
              {filteredProperties.length === 0 ? (
                <p className="text-muted text-sm">No properties match that search.</p>
              ) : (
                <ul className="divide-y divide-line border border-line rounded-md bg-sheet">
                  {filteredProperties.map((prop) => (
                    <li key={prop.name} className="px-4 py-3">
                      <div className="flex items-center justify-between gap-3">
                        <span className="font-mono text-sm text-ink">{prop.name}</span>
                        <span
                          className={`shrink-0 text-xs px-2 py-0.5 rounded-sm ${
                            prop.status === 'implemented' ? 'text-moss bg-moss-soft' : 'text-warn bg-warn-soft'
                          }`}
                        >
                          {prop.status}
                        </span>
                      </div>
                      {prop.values ? <p className="text-sm text-muted mt-1 break-words">{prop.values}</p> : null}
                      {prop.note ? <p className="text-sm text-muted mt-1">{prop.note}</p> : null}
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}

          {activeSection === 'selectors' && (
            <div className="space-y-6">
              <SectionHeader kicker="CSS" title="Selectors" />
              <h2 className="font-sans font-medium text-ink text-lg">Specificity (a, b, c)</h2>
              <div className="bg-sheet border border-line rounded-md p-4">
                <p className="text-base text-muted leading-relaxed max-w-2xl">
                  Specificity is the triple (a, b, c).
                </p>
                <ul className="mt-3 space-y-2 text-base text-muted leading-relaxed max-w-2xl">
                  <li>
                    <span className="text-ink">a.</span> Count of ID selectors (#id).
                  </li>
                  <li>
                    <span className="text-ink">b.</span> Count of class (.class), attribute ([attr]), and pseudo-classes.
                  </li>
                  <li>
                    <span className="text-ink">c.</span> Count of type selectors (tag) and pseudo-elements (::before, ::after).
                  </li>
                </ul>
              </div>
              <h2 className="font-sans font-medium text-ink text-lg">Pseudo-classes and pseudo-elements</h2>
              <div className="space-y-4">
                <div className="bg-sheet border border-line rounded-md p-4">
                  <h3 className="font-sans font-medium text-ink text-lg">
                    <code className="font-mono text-[0.9em]">:has(relative_selector)</code>
                  </h3>
                  <p className="text-base text-muted leading-relaxed mt-2 max-w-2xl">
                    Relational selector with depth inspection budget.
                  </p>
                </div>
                <div className="bg-sheet border border-line rounded-md p-4">
                  <h3 className="font-sans font-medium text-ink text-lg">
                    <code className="font-mono text-[0.9em]">:nth-child(an+b), :first-child, :last-child</code>
                  </h3>
                  <p className="text-base text-muted leading-relaxed mt-2 max-w-2xl">
                    Structural tree matching with formulas and keywords (odd, even).
                  </p>
                </div>
                <div className="bg-sheet border border-line rounded-md p-4">
                  <h3 className="font-sans font-medium text-ink text-lg">
                    <code className="font-mono text-[0.9em]">:is(), :where(), :not()</code>
                  </h3>
                  <p className="text-base text-muted leading-relaxed mt-2 max-w-2xl">
                    Functional logical grouping selectors.
                  </p>
                </div>
                <div className="bg-sheet border border-line rounded-md p-4">
                  <h3 className="font-sans font-medium text-ink text-lg">
                    <code className="font-mono text-[0.9em]">::before, ::after</code>
                  </h3>
                  <p className="text-base text-muted leading-relaxed mt-2 max-w-2xl">
                    Generated pseudo-elements with content support.
                  </p>
                </div>
              </div>
            </div>
          )}

          {activeSection === 'api' && (
            <div className="space-y-6">
              <SectionHeader kicker="Go API" title="API" />
              <p className="max-w-2xl text-base leading-relaxed text-muted">
                Choose a call. Its example opens under the signature.
              </p>
              <ApiReference />
            </div>
          )}
        </main>
      </div>
    </div>
  );
};
