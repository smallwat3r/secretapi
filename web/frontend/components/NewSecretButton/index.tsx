import { h } from 'preact';
import { useState, useRef, useEffect } from 'preact/hooks';
import styles from './NewSecretButton.module.css';

const HOLD_DURATION_MS = 1000;

export function NewSecretButton() {
  const [holding, setHolding] = useState(false);
  const timerRef = useRef<number | null>(null);

  // Clear timeout on unmount
  useEffect(() => {
    return () => {
      if (timerRef.current !== null) {
        clearTimeout(timerRef.current);
      }
    };
  }, []);

  const handlePressStart = (e: Event) => {
    e.preventDefault();
    setHolding(true);
    timerRef.current = window.setTimeout(() => {
      window.location.href = '/';
    }, HOLD_DURATION_MS);
  };

  const handlePressEnd = () => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
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
      aria-label="Hold to create new secret"
    >
      <div class={styles.progressBar} style={{ width: holding ? '100%' : '0%' }} />
      <span class={styles.buttonText}>{holding ? 'Hold...' : 'Create New Secret'}</span>
    </button>
  );
}
