import React from 'react';
import { GitBranch, ArrowUp, Star } from 'lucide-react';
import { Logo } from './Logo';
import { ENGINE_STATS } from '../data/properties';

interface FooterProps {
  onGoToDocs: (section?: string) => void;
}

export const Footer: React.FC<FooterProps> = ({ onGoToDocs }) => {
  const scrollToTop = () => {
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  return (
    <footer className="bg-zinc-950 border-t border-zinc-900 py-12 text-zinc-400 text-xs font-mono">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-6 pb-8 border-b border-zinc-900">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded bg-zinc-900 border border-zinc-800 flex items-center justify-center text-white">
              <Logo size={18} className="w-4 h-4 text-white" />
            </div>
            <div>
              <div className="font-semibold text-white text-xs">blinkless {ENGINE_STATS.version}</div>
              <div className="text-[11px] text-zinc-500 font-sans">Pure-Go HTML & CSS Layout Engine</div>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-6 text-xs text-zinc-400">
            <button
              onClick={() => onGoToDocs('overview')}
              className="hover:text-white transition-colors"
            >
              Documentation
            </button>
            <button
              onClick={() => onGoToDocs('properties')}
              className="hover:text-white transition-colors"
            >
              CSS Reference
            </button>
            <button
              onClick={() => onGoToDocs('comparison')}
              className="hover:text-white transition-colors"
            >
              Blink Comparison
            </button>
            <a
              href="https://github.com/chinmay-sawant/blinkless"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-white transition-colors flex items-center gap-1.5"
            >
              <GitBranch className="w-3.5 h-3.5" />
              <span>GitHub</span>
              <span className="flex items-center gap-0.5 text-zinc-300 ml-1">
                <Star className="w-3 h-3 text-amber-400 fill-amber-400" />
                <span>1</span>
              </span>
            </a>
            <button
              onClick={scrollToTop}
              className="hover:text-white transition-colors flex items-center gap-1 ml-auto md:ml-0"
            >
              <span>Back to Top</span>
              <ArrowUp className="w-3.5 h-3.5" />
            </button>
          </div>
        </div>

        <div className="pt-6 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 text-zinc-500 text-[11px]">
          <div>
            Module: <code className="text-zinc-400">{ENGINE_STATS.modulePath}</code>
          </div>
          <div>
            MIT Licensed &bull; Pure Go &bull; 100% CGO-free
          </div>
        </div>
      </div>
    </footer>
  );
};
