import { JSX } from 'preact';
import { Icon } from '../Icon';
import { useCopyToClipboard } from '../../hooks/useCopyToClipboard';
import styles from './CopyableDiv.module.css';

interface CopyableDivProps {
  value: string;
  header: string;
  mono?: boolean;
}

export function CopyableDiv({ value, header, mono }: CopyableDivProps) {
  const { copied, copyToClipboard } = useCopyToClipboard(value);

  const handleKeyDown = (e: JSX.TargetedKeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      copyToClipboard();
    }
  };

  return (
    <div class="field">
      <div class="fieldHead">
        <span class="label">{header}</span>
        <span class="hint" aria-live="polite">
          {copied ? 'Copied' : ''}
        </span>
      </div>
      <div
        class={`panel ${styles.copyable} ${mono ? 'mono' : ''}`}
        onClick={copyToClipboard}
        onKeyDown={handleKeyDown}
        role="button"
        tabIndex={0}
        aria-label={`Copy ${header.toLowerCase()}`}
      >
        <span class={styles.value}>{value}</span>
        <span class={styles.icon}>
          <Icon name={copied ? 'check' : 'copy'} />
        </span>
      </div>
    </div>
  );
}
