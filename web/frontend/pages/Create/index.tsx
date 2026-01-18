import { h, JSX } from 'preact';
import { useState, useMemo, useEffect } from 'preact/hooks';
import { CopyableDiv } from '../../components/CopyableDiv';
import styles from './Create.module.css';
import { useCancellableFetch } from '../../hooks/useCancellableFetch';
import { useConfig } from '../../hooks/useConfig';
import { ApiErrorResponse, CreateResponse, Expiry } from '../../types';

export function Create() {
  const config = useConfig();
  const [secret, setSecret] = useState<string>('');
  const [expiry, setExpiry] = useState<Expiry>('1d');
  const [result, setResult] = useState<CreateResponse | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const cancellableFetch = useCancellableFetch();

  // Clear sensitive data from memory on page hide/unload
  useEffect(() => {
    const clearSensitiveData = () => {
      setSecret('');
      setResult(null);
    };

    const handlePageHide = () => clearSensitiveData();
    const handleBeforeUnload = () => clearSensitiveData();

    window.addEventListener('pagehide', handlePageHide);
    window.addEventListener('beforeunload', handleBeforeUnload);

    return () => {
      window.removeEventListener('pagehide', handlePageHide);
      window.removeEventListener('beforeunload', handleBeforeUnload);
      clearSensitiveData();
    };
  }, []);

  const secretByteLength = useMemo(() => new Blob([secret]).size, [secret]);
  const isSecretTooLong = secretByteLength > config.max_secret_size;
  const isSecretEmpty = secret.trim() === '';

  const handleSubmit = async (e: JSX.TargetedEvent<HTMLFormElement, Event>) => {
    e.preventDefault();
    if (loading) return;

    // Client-side validation
    if (isSecretEmpty) {
      setError('Secret cannot be empty.');
      return;
    }
    if (isSecretTooLong) {
      setError(`Secret exceeds ${formatBytes(config.max_secret_size)} limit.`);
      return;
    }

    setLoading(true);
    setResult(null);
    setError(null);

    try {
      const response = await cancellableFetch('/create', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ secret, expiry }),
      });

      if (response.ok) {
        const data: CreateResponse = await response.json();
        setSecret(''); // Clear secret from memory after successful submission
        setResult(data);
      } else {
        const errorData: ApiErrorResponse = await response.json();
        setError(errorData.error || 'An unknown error occurred.');
      }
    } catch (err: unknown) {
      if (err instanceof Error && err.name !== 'AbortError') {
        setError('An unexpected error occurred. Please try again.');
      }
    } finally {
      setLoading(false);
    }
  };

  const formatBytes = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`;
    return `${(bytes / 1024).toFixed(1)} KB`;
  };

  const formatExpiryLabel = (expiry: string): string => {
    const match = expiry.match(/^(\d+)([hd])$/);
    if (!match) return expiry;
    const [, num, unit] = match;
    const n = parseInt(num, 10);
    if (unit === 'h') return n === 1 ? '1 hour' : `${n} hours`;
    if (unit === 'd') return n === 1 ? '1 day' : `${n} days`;
    return expiry;
  };

  if (result) {
    const expiresAt = new Date(result.expires_at).toUTCString();
    const messageTemplate = `I've shared a secret with you.

URL: ${result.read_url}
Passcode: ${result.passcode}

Expires: ${expiresAt}
You have 3 attempts to enter the correct passcode. The secret will be deleted after reading.`;

    return (
      <div class={`${styles.result} ${styles.pageWrapper}`}>
        <p class={styles.resultInfo}>
          Share the Read URL and passcode with the recipient, or use the message template below.
        </p>
        <CopyableDiv value={result.read_url} header="Read URL" />
        <CopyableDiv value={result.passcode} header="Passcode" />
        <p>Expires At</p>
        <div>{expiresAt}</div>
        <CopyableDiv value={messageTemplate} header="Message Template" />
      </div>
    );
  }

  return (
    <form class={`${styles.form} ${styles.pageWrapper}`} onSubmit={handleSubmit}>
      <label class={styles.fieldLabel} for="secret-input">
        Secret
      </label>
      <textarea
        id="secret-input"
        value={secret}
        onInput={(e: JSX.TargetedEvent<HTMLTextAreaElement, Event>) =>
          setSecret(e.currentTarget.value)
        }
        placeholder="Enter your secret"
        class={isSecretTooLong ? styles.textareaError : ''}
      />
      <div class={`${styles.charCount} ${isSecretTooLong ? styles.charCountError : ''}`}>
        {formatBytes(secretByteLength)} / {formatBytes(config.max_secret_size)}
      </div>
      <p class={styles.expiryLabel}>Expiry</p>
      <select
        value={expiry}
        onChange={(e: JSX.TargetedEvent<HTMLSelectElement, Event>) =>
          setExpiry(e.currentTarget.value as Expiry)
        }
      >
        {config.expiry_options.map((opt) => (
          <option key={opt} value={opt}>
            {formatExpiryLabel(opt)}
          </option>
        ))}
      </select>
      <button type="submit" disabled={loading || isSecretTooLong}>
        {loading ? 'Loading...' : 'Create Secret'}
      </button>
      {error && (
        <div class={`${styles.errorMessage} error`} role="alert" aria-live="polite">
          {error}
        </div>
      )}
    </form>
  );
}
