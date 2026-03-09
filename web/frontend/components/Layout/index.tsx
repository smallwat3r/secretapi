import { ComponentChildren } from 'preact';
import styles from './Layout.module.css';

interface LayoutProps {
  children: ComponentChildren;
  onToggleTheme: () => void;
}

export function Layout({ children, onToggleTheme }: LayoutProps) {
  return (
    <div class={styles.container}>
      <div class={styles.nav}>
        <a href="/" class={styles.navLink}>
          create
        </a>
        <a href="/about" class={styles.navLink}>
          about
        </a>
        <span>{`secretapi \u00A9 ${new Date().getFullYear()}`}</span>
        <button class={styles.themeToggle} onClick={onToggleTheme} aria-label="Toggle theme">
          <svg
            width="14"
            height="14"
            viewBox="0 0 14 14"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            <circle cx="7" cy="7" r="6" stroke="currentColor" stroke-width="1.5" />
            <path d="M7 1 A6 6 0 0 0 7 13 Z" fill="currentColor" />
          </svg>
        </button>
      </div>
      {children}
    </div>
  );
}
