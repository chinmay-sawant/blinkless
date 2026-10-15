import React, { useEffect, useState } from 'react';
import { Navbar } from './components/Navbar';
import { ProductLanding } from './components/ProductLanding';
import { Documentation } from './components/Documentation';
import { Footer } from './components/Footer';
import { hashFor, isDocSection, parseHash, type DocSection, type SiteView } from './route';

export const App: React.FC = () => {
  const initial = parseHash(window.location.hash);
  const [currentView, setCurrentView] = useState<SiteView>(initial.view);
  const [docsSection, setDocsSection] = useState<DocSection>(initial.section);

  useEffect(() => {
    if (window.location.hash === '') {
      history.replaceState(null, '', '#product');
    }

    const apply = () => {
      const next = parseHash(window.location.hash);
      setCurrentView(next.view);
      setDocsSection(next.section);
      window.scrollTo({ top: 0, behavior: 'smooth' });
    };

    window.addEventListener('hashchange', apply);
    return () => window.removeEventListener('hashchange', apply);
  }, []);

  const go = (view: SiteView, section: DocSection = 'overview') => {
    const next = hashFor(view, section);
    if (window.location.hash === next) {
      window.scrollTo({ top: 0, behavior: 'smooth' });
      return;
    }
    window.location.hash = next;
  };

  const handleGoToDocs = (section?: string) => {
    const next = section && isDocSection(section) ? section : 'overview';
    go('docs', next);
  };

  const handleViewChange = (view: SiteView) => {
    if (view === 'product') {
      go('product');
      return;
    }
    go('docs', docsSection);
  };

  return (
    <div className="min-h-screen bg-paper text-ink font-sans antialiased flex flex-col">
      <Navbar currentView={currentView} onViewChange={handleViewChange} />
      <main className="flex-1">
        {currentView === 'product' ? (
          <ProductLanding onGoToDocs={handleGoToDocs} />
        ) : (
          <Documentation initialSection={docsSection} onSectionChange={(section) => go('docs', section)} />
        )}
      </main>
      <Footer onGoToDocs={handleGoToDocs} />
    </div>
  );
};

export default App;
