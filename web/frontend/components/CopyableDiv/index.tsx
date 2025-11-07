import { h } from 'preact';
import { useState } from 'preact/hooks';
import styles from './CopyableDiv.module.css';

interface CopyableDivProps {
  value: string;
  header?: string;
}

function CopyableDiv({ value, header }: CopyableDivProps) {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = () => {
    navigator.clipboard.writeText(value).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }, err => console.error('Could not copy text: ', err));
  };

  return (
    <div>
      {header && <p>{header}</p>}
      <div className={styles.copyable} onClick={copyToClipboard}>
        {value}
      </div>
      {copied && <div className={styles.copyFeedback}>Copied!</div>}
    </div>
  );
}

export default CopyableDiv;
