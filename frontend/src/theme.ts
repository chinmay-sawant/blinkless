import { useState } from 'react';

export type ThemeName = 'dark' | 'light';

const storageKey = 'blinkless-theme';

export function currentTheme(): ThemeName {
  if (typeof document === 'undefined') {
    return 'dark';
  }
  return document.documentElement.getAttribute('data-theme') === 'light' ? 'light' : 'dark';
}

export function applyTheme(theme: ThemeName): void {
  document.documentElement.setAttribute('data-theme', theme);
  try {
    localStorage.setItem(storageKey, theme);
  } catch {
    // Private mode can refuse storage. The attribute still switches the page.
  }
}

export function useTheme(): { theme: ThemeName; toggle: () => void } {
  const [theme, setTheme] = useState<ThemeName>(currentTheme);

  const toggle = () => {
    const next: ThemeName = theme === 'dark' ? 'light' : 'dark';
    applyTheme(next);
    setTheme(next);
  };

  return { theme, toggle };
}
