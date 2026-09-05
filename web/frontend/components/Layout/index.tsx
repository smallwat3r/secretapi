import { ComponentChildren } from 'preact';
import { Icon } from '../Icon';
import { Theme } from '../../hooks/useTheme';
import styles from './Layout.module.css';

interface LayoutProps {
  children: ComponentChildren;
  theme: Theme;
  onToggleTheme: () => void;
}

export function Layout({ children, theme, onToggleTheme }: LayoutProps) {
  const next = theme === 'dark' ? 'light' : 'dark';

  return (
    <div class={styles.shell}>
      <header class={styles.header}>
        <a href="/" class={styles.wordmark}>
          secretapi
        </a>
        <nav class={styles.nav} aria-label="Main">
          <a href="/about" class={styles.navLink}>
            About
          </a>
          <button
            type="button"
            class="iconButton"
            onClick={onToggleTheme}
            aria-label={`Switch to ${next} theme`}
          >
            <Icon name={theme === 'dark' ? 'sun' : 'moon'} />
          </button>
        </nav>
      </header>
      <main class={styles.main}>{children}</main>
      <footer class={styles.footer}>
        <span>{`© ${new Date().getFullYear()} secretapi`}</span>
        <a href="https://github.com/smallwat3r/secretapi">Source</a>
      </footer>
    </div>
  );
}
