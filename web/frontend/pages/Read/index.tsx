import { JSX } from 'preact';
import { useState, useEffect, useRef } from 'preact/hooks';
import { CopyButton } from '../../components/CopyButton';
import { Icon } from '../../components/Icon';
import { useCancellableFetch } from '../../hooks/useCancellableFetch';
import { ApiErrorResponse, ReadResponse } from '../../types';
import styles from './Read.module.css';

// id is injected by preact-router from the :id segment, path is the route.
interface ReadProps {
  id?: string;
  path?: string;
}

const AUTO_CLEAR_SECONDS = 300; // 5 minutes

const formatTime = (seconds: number): string => {
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins}:${secs.toString().padStart(2, '0')}`;
};

export function Read(props: ReadProps) {
  const [passcode, setPasscode] = useState<string>('');
  const [secret, setSecret] = useState<string | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [secondsRemaining, setSecondsRemaining] = useState<number>(AUTO_CLEAR_SECONDS);
  const id = props.id ?? '';
  const cancellableFetch = useCancellableFetch();
  const timerRef = useRef<number | null>(null);

  // Clear secret from memory on page hide/unload (back button, close, navigate away)
  useEffect(() => {
    const clearSecret = () => {
      setSecret(null);
      setPasscode('');
    };

    window.addEventListener('pagehide', clearSecret);
    window.addEventListener('beforeunload', clearSecret);

    return () => {
      window.removeEventListener('pagehide', clearSecret);
      window.removeEventListener('beforeunload', clearSecret);
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
            const n = errorData.remaining_attempts;
            setError(`Wrong passcode. ${n} ${n === 1 ? 'attempt' : 'attempts'} left.`);
          } else {
            setError('Wrong passcode. No attempts left, the secret has been deleted.');
          }
        } else {
          setError(errorData.error || 'Something went wrong.');
        }
      }
    } catch (err: unknown) {
      if (err instanceof Error && err.name !== 'AbortError') {
        setError('Could not reach the server. Try again.');
      }
    } finally {
      setLoading(false);
    }
  };

  if (secret) {
    return (
      <div>
        <h1 class="title">Your secret</h1>
        <p class="lead">It has been deleted from the server and cannot be opened again.</p>
        <div class={`panel mono ${styles.secret}`}>{secret}</div>
        <div class={styles.actions}>
          <CopyButton textToCopy={secret} label="Copy secret" />
        </div>
        <div class="notice">
          <Icon name="clock" />
          <span>
            Save it now. This page clears itself in <strong>{formatTime(secondsRemaining)}</strong>.
          </span>
        </div>
      </div>
    );
  }

  // Show message if secret was auto-cleared
  if (secondsRemaining === 0) {
    return (
      <div>
        <h1 class="title">Secret cleared</h1>
        <p class="lead">
          This page cleared the secret after five minutes. If you did not save it, it cannot be
          recovered.
        </p>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} novalidate>
      <h1 class="title">Read a secret</h1>
      <p class="lead">Enter the passcode you were given. The secret can only be opened once.</p>

      <div class="fieldRow">
        <div class="field">
          <label class="label" for="passcode-input">
            Passcode
          </label>
          <input
            id="passcode-input"
            class="mono"
            value={passcode}
            onInput={(e: JSX.TargetedEvent<HTMLInputElement, Event>) =>
              setPasscode(e.currentTarget.value)
            }
            placeholder="word-word-word"
            required
            autocomplete="off"
            spellcheck={false}
          />
        </div>
        <button type="submit" class="btn" disabled={loading || passcode.trim() === ''}>
          {loading ? 'Opening' : 'Open secret'}
        </button>
      </div>

      {error && (
        <p class="error" role="alert">
          <Icon name="alert" />
          <span>{error}</span>
        </p>
      )}
    </form>
  );
}
