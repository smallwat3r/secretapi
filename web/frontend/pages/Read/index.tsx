import { h, JSX } from 'preact';
import { useState, useEffect, useRef } from 'preact/hooks';
import styles from './Read.module.css';
import { useCancellableFetch } from '../../hooks/useCancellableFetch';
import { NewSecretButton } from '../../components/NewSecretButton';
import { CopyButton } from '../../components/CopyButton';
import { ApiErrorResponse, ReadResponse } from '../../types';

interface ReadProps {
  id: string;
}

const AUTO_CLEAR_SECONDS = 300; // 5 minutes

export function Read(props: ReadProps) {
  const [passcode, setPasscode] = useState<string>('');
  const [secret, setSecret] = useState<string | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [secondsRemaining, setSecondsRemaining] = useState<number>(AUTO_CLEAR_SECONDS);
  const id = props.id;
  const cancellableFetch = useCancellableFetch();
  const timerRef = useRef<number | null>(null);

  // Clear secret from memory on page hide/unload (back button, close, navigate away)
  useEffect(() => {
    const clearSecret = () => {
      setSecret(null);
      setPasscode('');
    };

    // Handle page hide (works with bfcache, back/forward navigation)
    const handlePageHide = () => clearSecret();

    // Handle before unload (closing tab, navigating away)
    const handleBeforeUnload = () => clearSecret();

    window.addEventListener('pagehide', handlePageHide);
    window.addEventListener('beforeunload', handleBeforeUnload);

    return () => {
      window.removeEventListener('pagehide', handlePageHide);
      window.removeEventListener('beforeunload', handleBeforeUnload);
      clearSecret(); // Also clear on component unmount
    };
  }, []);

  // Auto-clear secret from memory after timeout
  useEffect(() => {
    if (!secret) return;

    setSecondsRemaining(AUTO_CLEAR_SECONDS);

    timerRef.current = window.setInterval(() => {
      setSecondsRemaining((prev) => {
        if (prev <= 1) {
          setSecret(null);
          if (timerRef.current) {
            clearInterval(timerRef.current);
            timerRef.current = null;
          }
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current);
        timerRef.current = null;
      }
    };
  }, [secret]);

  const handleSubmit = async (e: JSX.TargetedEvent<HTMLFormElement, Event>) => {
    e.preventDefault();
    if (loading) return;

    setLoading(true);
    setError(null);

    try {
      const response = await cancellableFetch(`/read/${id}`, {
        method: 'POST',
        headers: { 'X-Passcode': passcode },
      });

      if (response.ok) {
        const data: ReadResponse = await response.json();
        setSecret(data.secret);
        setPasscode(''); // Clear passcode from memory
      } else {
        const errorData: ApiErrorResponse = await response.json();

        if (errorData.remaining_attempts !== undefined) {
          if (errorData.remaining_attempts > 0) {
            setError(`Invalid passcode. ${errorData.remaining_attempts} attempts remaining.`);
          } else {
            setError('No attempts remaining. Secret deleted.');
          }
        } else {
          setError(errorData.error || 'An unknown error occurred.');
        }
      }
    } catch (err: any) {
      if (err?.name !== 'AbortError') {
        setError('An unexpected error occurred. Please try again.');
      }
    } finally {
      setLoading(false);
    }
  };

  const formatTime = (seconds: number): string => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  if (secret) {
    return (
      <div class={`${styles.result} ${styles.pageWrapper}`}>
        <div class={styles.secret}>{secret}</div>
        <div class={styles.warning}>
          <strong>Save this secret now.</strong> For security, it will be cleared from
          this page in <strong>{formatTime(secondsRemaining)}</strong>. The secret has
          been deleted from the server and cannot be retrieved again.
        </div>
        <CopyButton textToCopy={secret} />
        <NewSecretButton />
      </div>
    );
  }

  // Show message if secret was auto-cleared
  if (secondsRemaining === 0) {
    return (
      <div class={`${styles.result} ${styles.pageWrapper}`}>
        <div class={styles.warning}>
          The secret was cleared from this page for security. If you did not save it,
          it is no longer retrievable.
        </div>
        <NewSecretButton />
      </div>
    );
  }

  return (
    <form class={styles.pageWrapper} onSubmit={handleSubmit}>
      <input
        value={passcode}
        onInput={(e: JSX.TargetedEvent<HTMLInputElement, Event>) => setPasscode(e.currentTarget.value)}
        placeholder="Enter passcode"
        required
        autocomplete="off"
      />
      <button type="submit" disabled={loading}>
        {loading ? 'Loading...' : 'Read Secret'}
      </button>
      {error && (
        <div class={`${styles.errorMessage} error`} role="alert" aria-live="polite">
          {error}
        </div>
      )}
    </form>
  );
}

