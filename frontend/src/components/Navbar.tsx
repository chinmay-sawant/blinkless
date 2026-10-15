import React from 'react';
import { GitBranch, BookOpen, Layout, Star } from 'lucide-react';
import { Logo } from './Logo';
import { ENGINE_STATS } from '../data/properties';

interface NavbarProps {
  currentView: 'product' | 'docs';
  onViewChange: (view: 'product' | 'docs') => void;
}

export const Navbar: React.FC<NavbarProps> = ({ currentView, onViewChange }) => {
  return (
    <header className="sticky top-0 z-50 bg-zinc-950/95 backdrop-blur border-b border-zinc-800/80">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-14">
          {/* Brand with New Unique Logo */}
          <div className="flex items-center gap-6">
            <button
              onClick={() => onViewChange('product')}
              className="flex items-center gap-2.5 text-left group"
            >
              <div className="w-8 h-8 rounded bg-zinc-900 border border-zinc-700/80 flex items-center justify-center text-white group-hover:border-zinc-500 transition-colors">
                <Logo size={20} className="w-5 h-5 text-white" />
              </div>
              <div className="flex items-center gap-2">
                <span className="font-semibold text-sm tracking-tight text-white group-hover:text-zinc-200 transition-colors">
                  blinkless
                </span>
                <span className="text-[11px] px-1.5 py-0.5 rounded bg-zinc-900 text-zinc-400 font-mono border border-zinc-800">
                  {ENGINE_STATS.version}
                </span>
              </div>
            </button>

            {/* Navigation Tabs */}
            <nav className="flex items-center gap-1">
              <button
                onClick={() => onViewChange('product')}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium transition-colors ${
                  currentView === 'product'
                    ? 'bg-zinc-800 text-white'
                    : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'
                }`}
              >
                <Layout className="w-3.5 h-3.5" />
                <span>Product</span>
              </button>
              <button
                onClick={() => onViewChange('docs')}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium transition-colors ${
                  currentView === 'docs'
                    ? 'bg-zinc-800 text-white'
                    : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'
                }`}
              >
                <BookOpen className="w-3.5 h-3.5" />
                <span>Documentation</span>
              </button>
            </nav>
          </div>

          {/* Right Links: GitHub with Stars Badge */}
          <div className="flex items-center gap-3">
            <a
              href="https://github.com/chinmay-sawant/blinkless"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 text-xs font-mono py-1 px-2.5 rounded bg-zinc-900 hover:bg-zinc-850 text-zinc-300 border border-zinc-800 hover:border-zinc-700 transition-all group"
            >
              <div className="flex items-center gap-1.5">
                <GitBranch className="w-3.5 h-3.5 text-zinc-400 group-hover:text-white transition-colors" />
                <span>GitHub</span>
              </div>
              <div className="border-l border-zinc-700/80 pl-2 flex items-center gap-1 text-zinc-300 font-semibold">
                <Star className="w-3 h-3 text-amber-400 fill-amber-400" />
                <span>1</span>
              </div>
            </a>
          </div>
        </div>
      </div>
    </header>
  );
};
