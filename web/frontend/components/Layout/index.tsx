import { h, ComponentChildren } from 'preact';
import styles from './Layout.module.css';

interface LayoutProps {
  children: ComponentChildren;
}

export function Layout({ children }: LayoutProps) {
  return (
    <div class={styles.container}>
      {children}
      <div class={styles.footer}>
        <a href="https://github.com/smallwat3r/secretapi" class={styles.footerLink}>secretapi</a> © 2025
      </div>
    </div>
  );
}

