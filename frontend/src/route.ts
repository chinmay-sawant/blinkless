export const DOC_SECTIONS = [
  'overview',
  'quickstart',
  'pipeline',
  'comparison',
  'properties',
  'selectors',
  'api',
] as const;

export type DocSection = (typeof DOC_SECTIONS)[number];
export type SiteView = 'product' | 'docs';

export function isDocSection(value: string): value is DocSection {
  return (DOC_SECTIONS as readonly string[]).includes(value);
}

export function parseHash(hash: string): { view: SiteView; section: DocSection } {
  const raw = hash.replace(/^#/, '').replace(/^\//, '');
  if (raw === '' || raw === 'product') {
    return { view: 'product', section: 'overview' };
  }
  if (raw === 'docs') {
    return { view: 'docs', section: 'overview' };
  }
  if (raw.startsWith('docs/')) {
    const section = raw.slice('docs/'.length);
    if (isDocSection(section)) {
      return { view: 'docs', section };
    }
  }
  return { view: 'product', section: 'overview' };
}

export function hashFor(view: SiteView, section: DocSection = 'overview'): string {
  if (view === 'product') {
    return '#product';
  }
  return `#docs/${section}`;
}
