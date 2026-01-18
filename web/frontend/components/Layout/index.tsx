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
        <a href="/about" class={styles.footerLink}>
          about
        </a>
        {' | '}
        <a href="https://github.com/smallwat3r/secretapi" class={styles.footerLink}>
          github
        </a>
      </div>
    </div>
  );
}
