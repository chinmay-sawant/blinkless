import React from 'react';
import { Logo } from './Logo';
import { ENGINE_STATS } from '../data/properties';
import { useTheme } from '../theme';
import { GitHubLink } from './GitHubLink';

interface NavbarProps {
  currentView: 'product' | 'docs';
  onViewChange: (view: 'product' | 'docs') => void;
}

export const Navbar: React.FC<NavbarProps> = ({ currentView, onViewChange }) => {
  const { theme, toggle } = useTheme();
  const itemClass = (active: boolean) =>
    `cursor-pointer min-h-11 px-3 text-sm border-b-2 ${
      active
        ? 'text-ink border-ink'
        : 'text-muted border-transparent hover:text-ink'
    }`;

  return (
    <header className="sticky top-0 z-50 bg-paper/90 backdrop-blur border-b border-line">
      <div className="max-w-6xl mx-auto px-4 sm:px-6">
        <div className="flex flex-wrap items-center gap-x-6 gap-y-1 min-h-14">
          <button
            type="button"
            onClick={() => onViewChange('product')}
            className="flex items-center gap-2.5 text-left cursor-pointer min-h-11"
          >
            <span className="inline-flex h-8 w-8 items-center justify-center text-ink">
              <Logo size={32} className="h-8 w-8" />
            </span>
            <span className="font-sans font-medium tracking-tight text-ink">blinkless</span>
            <span className="text-xs font-mono text-faint">{ENGINE_STATS.version}</span>
          </button>

          <nav className="flex items-center">
            <button
              type="button"
              onClick={() => onViewChange('product')}
              className={itemClass(currentView === 'product')}
            >
              Product
            </button>
            <button
              type="button"
              onClick={() => onViewChange('docs')}
              className={itemClass(currentView === 'docs')}
            >
              Documentation
            </button>
          </nav>

          <div className="ml-auto flex items-center gap-1">
            <button
              type="button"
              onClick={toggle}
              aria-pressed={theme === 'dark'}
              aria-label={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
              className="inline-flex min-h-11 cursor-pointer items-center border border-line px-3 text-sm text-muted hover:text-ink"
            >
              {theme === 'dark' ? 'Dark' : 'Light'}
            </button>
            <GitHubLink className="px-3" />
          </div>
        </div>
      </div>
    </header>
  );
};
