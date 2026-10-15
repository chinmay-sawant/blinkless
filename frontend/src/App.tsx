import React, { useState } from 'react';
import { Navbar } from './components/Navbar';
import { ProductLanding } from './components/ProductLanding';
import { Documentation } from './components/Documentation';
import { Footer } from './components/Footer';

export const App: React.FC = () => {
  const [currentView, setCurrentView] = useState<'product' | 'docs'>('product');
  const [docsSection, setDocsSection] = useState<string>('overview');

  const handleGoToDocs = (section?: string) => {
    if (section) {
      setDocsSection(section);
    }
    setCurrentView('docs');
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const handleViewChange = (view: 'product' | 'docs') => {
    setCurrentView(view);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100 flex flex-col font-sans selection:bg-zinc-800 selection:text-white antialiased">
      <Navbar currentView={currentView} onViewChange={handleViewChange} />
      <main className="flex-1">
        {currentView === 'product' ? (
          <ProductLanding onGoToDocs={handleGoToDocs} />
        ) : (
          <Documentation initialSection={docsSection} />
        )}
      </main>
      <Footer onGoToDocs={handleGoToDocs} />
    </div>
  );
};

export default App;
