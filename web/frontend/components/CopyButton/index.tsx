import { h } from 'preact';
import styles from './CopyButton.module.css';
import { useCopyToClipboard } from '../../hooks/useCopyToClipboard';

interface CopyButtonProps {
  textToCopy: string;
}

export function CopyButton({ textToCopy }: CopyButtonProps) {
  const { copied, copyToClipboard } = useCopyToClipboard(textToCopy);

  return (
    <button onClick={copyToClipboard} class={styles.copyButton}>
      {copied ? 'Copied!' : 'Copy to Clipboard'}
    </button>
  );
}
