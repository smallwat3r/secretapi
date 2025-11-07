import { h } from 'preact';
import { useState } from 'preact/hooks';
import styles from './CopyButton.module.css';

interface CopyButtonProps {
  textToCopy: string;
}

export function CopyButton({ textToCopy }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = () => {
    navigator.clipboard.writeText(textToCopy).then(
      () => {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      },
      err => console.error('Could not copy text: ', err)
    );
  };

  return (
    <button onClick={copyToClipboard} class={styles.copyButton}>
      {copied ? 'Copied!' : 'Copy to Clipboard'}
    </button>
  );
}
