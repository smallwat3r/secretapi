import { h, JSX } from 'preact';
import styles from './CopyableDiv.module.css';
import { useCopyToClipboard } from '../../hooks/useCopyToClipboard';

interface CopyableDivProps {
  value: string;
  header?: string;
}

export function CopyableDiv({ value, header }: CopyableDivProps) {
  const { copied, copyToClipboard } = useCopyToClipboard(value);

  const handleKeyDown = (e: JSX.TargetedKeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      copyToClipboard();
    }
  };

  return (
    <div>
      {header && <p>{header}</p>}
      <div
        className={styles.copyable}
        onClick={copyToClipboard}
        onKeyDown={handleKeyDown}
        role="button"
        tabIndex={0}
        aria-label={`Click to copy ${header || 'value'}`}
      >
        {value}
      </div>
      {copied && <div className={styles.copyFeedback}>Copied!</div>}
    </div>
  );
}
