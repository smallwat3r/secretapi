import { useState, useEffect } from 'preact/hooks';

export type Theme = 'light' | 'dark';

const STORAGE_KEY = 'theme';

function applyTheme(theme: Theme) {
  document.documentElement.setAttribute('data-theme', theme);
}

function getInitialTheme(serverDefault?: Theme): Theme {
  const stored = localStorage.getItem(STORAGE_KEY) as Theme | null;
  if (stored === 'light' || stored === 'dark') {
    return stored;
  }
  if (serverDefault) {
    return serverDefault;
  }
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

export function useTheme(serverDefault?: Theme): { theme: Theme; toggleTheme: () => void } {
  const [theme, setTheme] = useState<Theme>(() => {
    const initial = getInitialTheme(serverDefault);
    applyTheme(initial);
    return initial;
  });

  useEffect(() => {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored && serverDefault) {
      applyTheme(serverDefault);
      setTheme(serverDefault);
    }
  }, [serverDefault]);

  function toggleTheme() {
    const next: Theme = theme === 'light' ? 'dark' : 'light';
    applyTheme(next);
    localStorage.setItem(STORAGE_KEY, next);
    setTheme(next);
  }

  return { theme, toggleTheme };
}
