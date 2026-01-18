import { h, ComponentChildren } from 'preact';
import styles from './Layout.module.css';

interface LayoutProps {
  children: ComponentChildren;
}

export function Layout({ children }: LayoutProps) {
  return (
    <div class={styles.container}>
      <div class={styles.nav}>
        <a href="/" class={styles.navLink}>
          create
        </a>
        {' | '}
        <a href="/about" class={styles.navLink}>
          about
        </a>
        {' | secretapi \u00A9 2025'}
      </div>
      {children}
    </div>
  );
}
