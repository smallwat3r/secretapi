import { h } from 'preact';
import { useState, useRef } from 'preact/hooks';
import styles from './NewSecretButton.module.css';

export function NewSecretButton() {
  const [holding, setHolding] = useState(false);
  const timer = useRef<number | null>(null);

  const handlePressStart = (e: Event) => {
    e.preventDefault();
    setHolding(true);
    timer.current = window.setTimeout(() => {
      window.location.href = '/';
    }, 1000);
  };

  const handlePressEnd = () => {
    if (timer.current) {
      clearTimeout(timer.current);
      timer.current = null;
    }
    setHolding(false);
  };

  return (
    <button
      onMouseDown={handlePressStart}
      onTouchStart={handlePressStart}
      onMouseUp={handlePressEnd}
      onTouchEnd={handlePressEnd}
      onMouseLeave={handlePressEnd}
      class={styles.newSecretButton}
    >
      <div class={styles.progressBar} style={{ width: holding ? '100%' : '0%' }} />
      <span class={styles.buttonText}>{holding ? 'Hold...' : 'Create New Secret'}</span>
    </button>
  );
}
