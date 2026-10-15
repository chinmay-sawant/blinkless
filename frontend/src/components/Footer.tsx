import React from 'react';
import { Logo } from './Logo';
import { ENGINE_STATS } from '../data/properties';
import { GitHubLink } from './GitHubLink';

interface FooterProps {
  onGoToDocs: (section?: string) => void;
}

export const Footer: React.FC<FooterProps> = ({ onGoToDocs }) => {
  const scrollToTop = () => {
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  return (
    <footer className="border-t border-line bg-paper py-10 text-sm text-muted">
      <div className="max-w-6xl mx-auto px-4 sm:px-6">
        <div className="flex flex-col gap-6 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <div className="flex items-center gap-2.5 text-ink">
              <Logo size={20} className="h-5 w-5" />
              <span className="font-medium tracking-tight">blinkless</span>
              <span className="font-mono text-xs text-faint">{ENGINE_STATS.version}</span>
            </div>
            <p className="mt-1 font-mono text-xs">{ENGINE_STATS.modulePath}</p>
          </div>

          <nav className="flex flex-wrap items-center gap-x-5 gap-y-1">
            <button
              type="button"
              onClick={() => onGoToDocs('overview')}
              className="cursor-pointer min-h-11 hover:text-ink"
            >
              Documentation
            </button>
            <button
              type="button"
              onClick={() => onGoToDocs('properties')}
              className="cursor-pointer min-h-11 hover:text-ink"
            >
              CSS catalog
            </button>
            <GitHubLink />
            <button
              type="button"
              onClick={scrollToTop}
              className="cursor-pointer min-h-11 hover:text-ink"
            >
              Back to top
            </button>
          </nav>
        </div>

        <p className="mt-6 flex flex-wrap gap-x-4 gap-y-1">
          <span>MIT</span>
          <span>Default build leaves CGO off.</span>
        </p>
      </div>
    </footer>
  );
};
